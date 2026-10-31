# Arquitetura do Task Manager System

## Contexto

O sistema contém uma SPA Angular servida por NGINX e três serviços Go:

- `service-identity`: cadastro, login, perfil do usuário (nome e troca de senha), JWT e PostgreSQL próprio.
- `service-task`: CRUD de tarefas com prioridade, prazo e histórico de alterações, PostgreSQL próprio e publicação de
  `task.created`.
- `service-notification`: consumo AMQP e registro do evento recebido.

RabbitMQ desacopla criação de tarefas do consumidor. OpenTelemetry Collector, Tempo, Prometheus e Grafana formam a stack
local de telemetria. O módulo OpenTelemetry do NGINX inicia o span de entrada e injeta W3C Trace Context nas requisições
encaminhadas. O NGINX exporta para `otel-collector:4317`; os três serviços usam `http://otel-collector:4317`, definido no
Compose via `OTEL_EXPORTER_OTLP_ENDPOINT`. Sem essa variável, traces dos serviços são desativados e o serviço inicia
normalmente. Métricas
Prometheus continuam disponíveis diretamente em `/metrics`.

## Fluxos

### HTTP

O frontend é a entrada da aplicação em `:8081`. O NGINX cria o trace na borda e balanceia `/api/identity`, `/api/tasks` e
`/api/notifications` com `least_conn` e resolução DNS dinâmica na rede `edge`. APIs não publicam portas no host; isso
permite escalar réplicas pelo Compose sem conflito de bind.

### Eventos

`service-task` grava a tarefa e o evento em `task_outbox` na mesma transação. Um dispatcher reivindica eventos pendentes
com lease de 30s, publica com publisher confirms e `mandatory`, e agenda retry com backoff exponencial em caso de falha.
A exchange `tasks` é `topic` e a fila durável `task.events` usa `tasks.dlx` como dead-letter exchange.
`service-notification` confirma eventos válidos; JSON inválido recebe `Nack(requeue=false)` e vai para `task.events.dlq`.
O consumidor não faz retry próprio: falhas de Ack/Nack encerram o consumo para o supervisor reiniciar a conexão. A
entrega é at-least-once; consumidores devem deduplicar por `MessageId`/`event_id`.

### Organização e histórico de tarefas

Cada tarefa tem `priority` (`low`, `medium`, `high`, `urgent`; padrão `medium`) e `due_date` opcional (data de
calendário `YYYY-MM-DD`). `GET /tasks` filtra por status, prioridade e atrasadas e ordena por posição, prioridade, criação
ou prazo.

Toda alteração relevante grava uma linha em `task_history` **na mesma transação** da escrita:

- `created` na criação;
- `updated` com o diff dos campos em `PUT`;
- `moved` quando `PATCH /move` troca a coluna.

A descrição registra apenas que mudou. Reordenar dentro da mesma coluna não gera histórico.
`GET /tasks/{id}/history` expõe a linha do tempo exibida no detalhe da tarefa. Decisão em
[ADR-0013](adrs/0013-prioridade-prazo-e-historico-de-tarefas.md).

### Correlação

`x-correlation-id` é um ID de aplicação:

- middlewares Go aceitam apenas UUIDs v4 válidos, geram um novo na entrada da API quando ausente ou inválido e o incluem na resposta e nos logs JSON;
- o publicador o inclui no header AMQP e o consumidor o extrai.

`x-correlation-id` é um identificador de aplicação e continua separado do W3C Trace Context. O NGINX gera `traceparent`
na borda. OpenTelemetry o propaga em HTTP, persiste o carrier no outbox e o injeta nos headers
AMQP, criando spans para NGINX, requisições HTTP, dispatch, publicação e consumo. Os logs JSON incluem `correlation_id`,
`trace_id` e `span_id` quando disponíveis, além de `service_name`; os identificadores são campos puros, não prefixos.

## Observabilidade

O Collector aceita OTLP/gRPC em `0.0.0.0:4317` e OTLP/HTTP em `0.0.0.0:4318` dentro da rede `telemetry`. Pipelines usam
`memory_limiter` antes de `batch`:

- traces: Collector -> exporter `otlp/tempo` -> Tempo;
- métricas OTLP: Collector -> exporter Prometheus em `:8889` -> scrape pelo Prometheus;
- métricas Go e de negócio: serviços `/metrics` -> scrape direto pelo Prometheus;
- métricas do broker: plugin `rabbitmq_prometheus` em `rabbitmq:15692/metrics` -> Prometheus.

Tempo 3.1.1 roda em modo monolítico (`target=all`), recebe OTLP em binds explícitos, grava WAL/blocos em `/var/tempo`,
usa backend local e retenção de 24 horas. Prometheus mantém TSDB por 7 dias. Loki 3.6.7 roda em modo monolítico com
schema TSDB v13, backend `filesystem` e retenção de 7 dias, sem porta publicada. O Alloy lê os arquivos `*-json.log`
do diretório `containers` do Docker (`DOCKER_ROOT_DIR`), desembrulha o JSON do driver com `stage.docker` e envia ao Loki. Os labels são apenas
`service_name`, `level` e `container`; IDs permanecem no corpo e são consultados com `| json`. Grafana recebe datasources
provisionados para Prometheus, Tempo e Loki, com links Loki→Tempo (`derivedFields`) e Tempo→Loki (`tracesToLogsV2`).

Bodies HTTP entram nos logs de forma sanitizada: até 64 KiB, somente JSON completo e válido, com campos operacionais
permitidos (`status`, `priority`, `due_date`, `before_id`, `after_id`) e todo o restante como `[REDACTED]`. Corpo
inválido, truncado ou não JSON gera apenas tamanho e flags. O NGINX registra access log JSON sem query string, headers
de autenticação ou body e devolve `X-Trace-ID` nas respostas de `/api/*`. Decisão em
[ADR-0015](adrs/0015-logs-contextuais-loki-alloy.md).

Os serviços usam `otelhttp` no servidor HTTP e spans manuais na publicação/consumo AMQP e no dispatch do outbox, portanto
traces aparecem no Tempo/Grafana. `prometheus/client_golang` expõe runtime Go, processo, taxa/status/latência HTTP e os
contadores de negócio de autenticação, CRUD de tarefas, outbox e consumo de notificações. Labels usam somente serviço,
método, padrão da rota, status, operação e resultado; IDs e URLs brutas são proibidos.

O label `route` vem do mux mais interno de cada serviço. Como `otelhttp` e o middleware de correlação copiam o request,
o padrão casado (`GET /tasks/{id}`) não volta sozinho para o middleware de métricas; `metrics.RoutePattern` envolve o
mux das rotas e devolve o padrão por um ponteiro no `context.Context`. Sem ele, todas as requisições caíam em
`route="unmatched"` (Task) ou `route="/"` (Identity).

O plugin RabbitMQ retorna métricas por objeto para separar `task.events` e `task.events.dlq`. Isso é adequado à stack
local pequena, mas deve ser desabilitado ou filtrado em ambientes com muitas filas. Prometheus acessa `:15692` pela rede
interna `messaging`; a porta não é publicada no host.

Grafana provisiona três dashboards versionados em `provisioning/grafana/dashboard-definitions`: visão de negócio,
backends Go e RabbitMQ. O provider em `provisioning/grafana/dashboards/providers.yml` recarrega arquivos a cada 10s.

## SLOs

Os objetivos abaixo valem para a stack Compose com uma réplica por serviço e são medidos de ponta a ponta pelo NGINX
(`:8081`), que é o caminho real do usuário. Cada SLI usa uma janela de 30 dias; no ambiente local, o teste de carga é a
evidência de aderência.

| SLI                              | SLO                                     | Fonte de medição                                                                  |
|----------------------------------|-----------------------------------------|-----------------------------------------------------------------------------------|
| Latência de `GET /tasks`         | p95 < 200 ms e p99 < 500 ms             | k6 (`loadtest/tasks.js`) e `task_manager_http_request_duration_seconds`           |
| Latência de `POST /tasks`        | p95 < 300 ms                            | k6 (`loadtest/tasks.js`) e `task_manager_http_request_duration_seconds`           |
| Taxa de erro HTTP                | < 1% de respostas 5xx ou falhas de rede | k6 (`http_req_failed`) e `task_manager_http_requests_total` por `status`          |
| Entrega do evento `task.created` | 99,9% publicados em até 5 s             | `sent_at - created_at` em `task_outbox` e `task_manager_task_outbox_events_total` |
| Disponibilidade das APIs         | 99,5%                                   | `GET /health` dos serviços e alvo `up` no Prometheus                              |

A latência de criação é mais alta que a de leitura porque a escrita grava tarefa, histórico e outbox na mesma transação.
O SLO de entrega mede a consistência eventual do outbox: o dispatcher varre pendentes a cada 500 ms e, em falha, aplica
backoff exponencial de até 64 s. Por isso, uma indisponibilidade do RabbitMQ maior que alguns segundos consome o
orçamento de erro desse SLO sem afetar os SLOs HTTP.

### Medição

`loadtest/tasks.js` registra um usuário de teste, cria tarefas iniciais e roda dois cenários de taxa constante em
paralelo: leitura (`GET /tasks`, 50 req/s) e criação (`POST /tasks`, 10 req/s) por 2 minutos. Os thresholds do k6
codificam os SLOs HTTP, e o processo termina com código diferente de zero quando algum é violado. O resumo com p50, p95,
p99 e throughput por endpoint fica em `results-summary.json`, no diretório de execução.

```sh
docker compose up --build -d
docker run --rm -v "$PWD/loadtest:/scripts" -w /scripts \
  -e BASE_URL=http://host.docker.internal:8081 \
  grafana/k6:0.54.0 run tasks.js
```

`DURATION`, `LIST_RATE`, `CREATE_RATE` e `SEED_TASKS` ajustam o cenário. O registro no `setup` usa uma única chamada
para não atingir o rate limit de `/register`.

O SLO de entrega é verificado após o teste:

```sh
docker compose exec postgres-task psql -U task -d task -c "
  SELECT count(*) AS eventos,
         percentile_cont(0.95) WITHIN GROUP (ORDER BY extract(epoch FROM sent_at - created_at)) AS p95_s,
         avg((sent_at - created_at <= interval '5 seconds')::int) AS dentro_do_slo
  FROM task_outbox WHERE sent_at IS NOT NULL"
```

## Segurança e isolamento

- Variáveis obrigatórias em `.env` fornecem senhas dos bancos, RabbitMQ, JWT e Grafana; o Compose rejeita ausência.
- RabbitMQ usa usuário próprio, não `guest`.
- `identity-data`, `task-data`, `messaging` e `telemetry` são redes internas separadas; cada serviço entra apenas nas
  redes de que precisa, e todos entram em `telemetry` para exportar OTLP. Como redes `internal` não publicam portas,
  o Grafana também entra na rede bridge `admin`, usada apenas para expor `127.0.0.1:3000`.
- Somente o frontend publica porta em todas as interfaces. APIs e consoles administrativos usam loopback.
- Bancos e serviços de telemetria não publicam portas no host.
- Contexts Docker excluem segredos e artefatos locais.
- `/register` e `/login` limitam 10 requisições por minuto por IP e devolvem `429` com `Retry-After: 60`.
- Payloads são limitados a 16 KiB em Identity e 1 MiB em Tasks; excesso recebe `413`.

O frontend mantém o JWT em `localStorage` enquanto o backend não oferecer cookie `HttpOnly`; riscos e mitigações estão em
[`frontend/SECURITY.md`](../frontend/SECURITY.md).

O transporte interno usa HTTP/AMQP sem TLS porque esta é uma stack local de desenvolvimento. Produção exige secrets
manager, TLS, backend de objetos para Tempo, autenticação dos endpoints de telemetria e política de rede do orquestrador.

## Persistência e ciclo de vida

Volumes nomeados preservam PostgreSQL, RabbitMQ, Tempo, Prometheus, Grafana, Loki e Alloy. Todos os serviços usam
`restart: unless-stopped`. Dependências críticas usam healthchecks e `depends_on.condition: service_healthy`. `down`
preserva os volumes; `down -v` os apaga.

## Limites conhecidos

- Os SLOs são verificados por teste de carga local com uma réplica por serviço; não há medição contínua em produção.
- O consumidor não envia e-mail, push ou webhook; registra `notification_received`.
- Não há retry no consumidor; o publisher usa outbox com retry/backoff e entrega at-least-once.
- O dispatcher do outbox publica um evento por tick de 500 ms (cerca de 2 eventos/s por réplica). No teste de carga
  com 10 criações/s, a fila acumulou 1.642 eventos pendentes e o p95 de entrega chegou a 264,7 s, violando o SLO de
  5 s. Nenhum evento se perde, mas a consistência eventual passa a ser de minutos; drenar o outbox em lote a cada tick
  ou rodar mais réplicas do dispatcher (o `SKIP LOCKED` permite) resolve o gargalo.
- Loki com `filesystem` não é HA e o Alloy exige leitura do diretório de containers do Docker. Em produção, trocar por
  object storage e coleta compatível com o orquestrador; o bind concede leitura de logs de todos os containers do host.
- `level` é label de baixa cardinalidade; linhas que não sejam JSON da aplicação recebem `level="unknown"`.
- Não há Alertmanager nem alertas de burn rate dos SLOs; dashboards não substituem alertas operacionais.
- Os Dockerfiles usam usuário não-root e builds sem fallback silencioso; o pin por digest das imagens-base ainda não é
  aplicado (tags de versão são usadas).

## ADRs

Decisões ficam em [`docs/adrs`](adrs/). ADR-0002 separa correlação de aplicação de W3C Trace Context; ADR-0003 descreve
semântica real da DLQ; ADR-0011 registra hardening e segmentação; ADR-0012 define métricas e dashboards; ADR-0014 cobre
tracing na borda; ADR-0015 define logs contextuais, sanitização e a stack Loki/Alloy.
