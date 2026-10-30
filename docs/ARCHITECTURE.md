# Arquitetura do Task Manager System

## Contexto

O sistema contém uma SPA Angular servida por NGINX e três serviços Go:

- `service-identity`: cadastro, login, perfil do usuário (nome e troca de senha), JWT e PostgreSQL próprio.
- `service-task`: CRUD de tarefas, PostgreSQL próprio e publicação de `task.created`.
- `service-notification`: consumo AMQP e registro do evento recebido.

RabbitMQ desacopla criação de tarefas do consumidor. OpenTelemetry Collector, Tempo, Prometheus e Grafana formam a stack
local de telemetria. Os três serviços exportam traces OTLP/gRPC para `http://otel-collector:4317` (definido no Compose
via `OTEL_EXPORTER_OTLP_ENDPOINT`); sem essa variável, traces são desativados e o serviço inicia normalmente. Métricas
Prometheus continuam disponíveis diretamente em `/metrics`.

## Fluxos

### HTTP

O frontend é a entrada da aplicação em `:8081`. O NGINX encaminha `/api/identity`, `/api/tasks` e
`/api/notifications` aos serviços pela rede `edge`. As portas diretas `8082`, `8083` e `8084` existem para diagnóstico
local e aceitam conexões somente em `127.0.0.1`.

### Eventos

`service-task` grava a tarefa e o evento em `task_outbox` na mesma transação. Um dispatcher reivindica eventos pendentes
com lease de 30s, publica com publisher confirms e `mandatory`, e agenda retry com backoff exponencial em caso de falha.
A exchange `tasks` é `topic` e a fila durável `task.events` usa `tasks.dlx` como dead-letter exchange.
`service-notification` confirma eventos válidos; JSON inválido recebe `Nack(requeue=false)` e vai para `task.events.dlq`.
O consumidor não faz retry próprio: falhas de Ack/Nack encerram o consumo para o supervisor reiniciar a conexão. A
entrega é at-least-once; consumidores devem deduplicar por `MessageId`/`event_id`.

### Correlação

`x-correlation-id` é um ID de aplicação:

- o frontend gera UUID v4;
- middlewares Go aceitam apenas UUIDs v4 válidos, geram um novo quando ausente ou inválido e o incluem na resposta e nos logs JSON;
- o publicador o inclui no header AMQP e o consumidor o extrai.

`x-correlation-id` é um identificador de aplicação e continua separado do W3C Trace Context. A instrumentação
OpenTelemetry propaga `traceparent`/`tracestate` em HTTP e nos headers AMQP, criando spans para requisições HTTP,
publicação, dispatch do outbox e consumo. Os logs continuam indexados por `correlation_id`, não por `trace_id`.

## Observabilidade

O Collector aceita OTLP/gRPC em `0.0.0.0:4317` e OTLP/HTTP em `0.0.0.0:4318` dentro da rede `telemetry`. Pipelines usam
`memory_limiter` antes de `batch`:

- traces: Collector -> exporter `otlp/tempo` -> Tempo;
- métricas OTLP: Collector -> exporter Prometheus em `:8889` -> scrape pelo Prometheus;
- métricas Go e de negócio: serviços `/metrics` -> scrape direto pelo Prometheus;
- métricas do broker: plugin `rabbitmq_prometheus` em `rabbitmq:15692/metrics` -> Prometheus.

Tempo 3.1.1 roda em modo monolítico (`target=all`), recebe OTLP em binds explícitos, grava WAL/blocos em `/var/tempo`,
usa backend local e retenção de 24 horas. Prometheus mantém TSDB por 7 dias. Grafana recebe datasources provisionados
para ambos. Loki não está implantado; logs são lidos com `docker compose logs`.

Os serviços usam `otelhttp` no servidor HTTP e spans manuais na publicação/consumo AMQP e no dispatch do outbox, portanto
traces aparecem no Tempo/Grafana. `prometheus/client_golang` expõe runtime Go, processo, taxa/status/latência HTTP e os
contadores de negócio de autenticação, CRUD de tarefas, outbox e consumo de notificações. Labels usam somente serviço,
método, padrão da rota, status, operação e resultado; IDs e URLs brutas são proibidos.

O plugin RabbitMQ retorna métricas por objeto para separar `task.events` e `task.events.dlq`. Isso é adequado à stack
local pequena, mas deve ser desabilitado ou filtrado em ambientes com muitas filas. Prometheus acessa `:15692` pela rede
interna `messaging`; a porta não é publicada no host.

Grafana provisiona três dashboards versionados em `provisioning/grafana/dashboard-definitions`: visão de negócio,
backends Go e RabbitMQ. O provider em `provisioning/grafana/dashboards/providers.yml` recarrega arquivos a cada 10s.

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

Volumes nomeados preservam PostgreSQL, RabbitMQ, Tempo, Prometheus e Grafana. Todos os serviços usam
`restart: unless-stopped`. Dependências críticas usam healthchecks e `depends_on.condition: service_healthy`. `down`
preserva os volumes; `down -v` os apaga.

## Limites conhecidos

- SLOs de latência, vazão e disponibilidade não são garantidos: não há teste de carga nem alertas que os comprovem.
- O consumidor não envia e-mail, push ou webhook; registra `notification_received`.
- Não há retry no consumidor; o publisher usa outbox com retry/backoff e entrega at-least-once.
- Logs não são centralizados e não incluem `trace_id`/`span_id`.
- Não há Alertmanager nem SLOs formais; dashboards não substituem alertas operacionais.
- Os Dockerfiles usam usuário não-root e builds sem fallback silencioso; o pin por digest das imagens-base ainda não é
  aplicado (tags de versão são usadas).

## ADRs

Decisões ficam em [`docs/adrs`](adrs/). ADR-0002 separa correlação de aplicação de W3C Trace Context; ADR-0003 descreve
semântica real da DLQ; ADR-0011 registra hardening e segmentação; ADR-0012 define métricas e dashboards.
