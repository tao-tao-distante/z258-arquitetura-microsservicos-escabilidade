# provisioning

Configurações da stack local de observabilidade e do RabbitMQ. Nada aqui é código
de aplicação: são arquivos montados como volume somente-leitura nos containers do
`compose.yml`.

- Nenhum destes serviços publica porta no host, exceto Grafana (`127.0.0.1:3000`)
  e RabbitMQ Management (`127.0.0.1:15672`).
- Os serviços Go enviam traces para `otel-collector:4317`; o Prometheus coleta
  métricas diretamente de cada `/metrics`.
- Os serviços Go enviam traces para `otel-collector:4317`; o Alloy lê os arquivos
  `*-json.log` do diretório de containers do Docker e os envia ao Loki. Em Docker
  rootless, defina `DOCKER_ROOT_DIR` no `.env` (por exemplo `$HOME/.local/share/docker`).
- Os dados persistem nos volumes `tempo-data`, `prometheus-data`, `grafana-data`,
  `loki-data`, `alloy-data` e `rabbitmq-data`.

## Estrutura

```text
provisioning
├── otel-collector/config.yaml          recebe OTLP e exporta para Tempo/Prometheus
├── tempo/tempo.yaml                    backend de traces (local + WAL)
├── loki/loki.yaml                      backend de logs (TSDB v13 + filesystem, 7 dias)
├── alloy/config.alloy                  coleta Docker e envia logs ao Loki
├── prometheus/prometheus.yml           scrape dos serviços, collector e RabbitMQ
├── grafana
│   ├── datasources/datasources.yml     Prometheus (default), Tempo e Loki com correlação
│   ├── dashboards/providers.yml        provider de dashboards por arquivo
│   ├── dashboard-definitions/*.json    quatro dashboards versionados
│   ├── alerting/default.yml            provisionamento de alertas (vazio)
│   └── plugins/default.yml             provisionamento de plugins (vazio)
└── rabbitmq
    ├── enabled_plugins                 management + prometheus
    └── rabbitmq.conf                   métricas por objeto
```

## Fluxo de telemetria

```text
service-* --OTLP gRPC :4317--> otel-collector --OTLP gRPC--> tempo :4317
                                     |
                                     +-- exporter prometheus :8889 <-- scrape

service-* /metrics <---------------- prometheus scrape
rabbitmq :15692 <-------------------- prometheus scrape
prometheus :9090 <------------------- grafana datasource
tempo :3200 <------------------------ grafana datasource
loki :3100 <------------------------- grafana datasource
docker stdout/stderr --Alloy-------> loki :3100
```

- Métricas de aplicação são expostas em `/metrics` e coletadas a cada 15 s.
- Traces são exportados por OTLP/gRPC; o collector aplica limite de memória e
  batching antes de gravar no Tempo.
- O Alloy descobre os arquivos `*-json.log` no data root do Docker, desembrulha o
  JSON do driver com `stage.docker` e envia as linhas ao Loki.
- Loki extrai `trace_id` como campo derivado que abre o Tempo; o datasource Tempo
  consulta o Loki por `trace_id` e `service_name`. Ambos são consultados em Explore.

## otel-collector

`otel-collector/config.yaml`:

| Bloco | Configuração |
|---|---|
| Receivers | OTLP gRPC em `0.0.0.0:4317` e HTTP em `0.0.0.0:4318`. |
| Processors | `memory_limiter` (256 MiB, spike de 64 MiB, checagem a cada 1 s) e `batch` (5 s ou 1024 itens). |
| Exporters | `prometheus` em `0.0.0.0:8889`; `otlp/tempo` para `tempo:4317` com TLS inseguro (rede interna). |
| Extensions | `health_check` em `0.0.0.0:13133`. |
| Pipelines | `metrics` → Prometheus; `traces` → Tempo. |

O healthcheck do container executa `validate --config`, ou seja, valida a
configuração em vez de consultar o endpoint de saúde.

## tempo

`tempo/tempo.yaml`:

- Modo monolítico (`-target=all`), HTTP em `:3200` e OTLP em `:4317`/`:4318`.
- `stream_over_http_enabled: true` para consultas via Grafana.
- Armazenamento local em `/var/tempo/blocks` com WAL em `/var/tempo/wal`.
- Retenção de blocos de `24h` na compactação.
- `usage_report` desabilitado.

## loki

`loki/loki.yaml`:

- Modo monolítico, HTTP em `:3100` e gRPC em `:9096`, sem porta publicada.
- `auth_enabled: false` restrito à rede interna `telemetry`.
- Schema TSDB `v13`, store `tsdb`, object store `filesystem` em `/loki`.
- Retenção de `168h` (7 dias) com compactor habilitado e `delete_request_store: filesystem`.
- `allow_structured_metadata` e `volume_enabled` habilitados.

## alloy

`alloy/config.alloy`:

- `local.file_match` descobre `/var/lib/docker/containers/*/*-json.log` e
  `discovery.relabel` deriva o label `container` do caminho.
- `loki.source.file` lê os arquivos descobertos; `loki.process` usa
  `stage.docker` para desembrulhar o driver, `stage.json` para extrair
  `service_name` e `level`, e `stage.labels` para publicá-los como labels.
  Linhas não JSON recebem `level="unknown"` e `service_name="unknown"`.
- `loki.write` envia para `http://loki:3100/loki/api/v1/push`.
- O diretório `containers` do Docker é montado como somente-leitura e o Alloy fica
  apenas na rede interna `telemetry`. O volume `alloy-data` guarda as posições de
  leitura.

## prometheus

`prometheus/prometheus.yml` usa intervalo de scrape e avaliação de `15 s` e coleta:

| Job | Alvo | Observação |
|---|---|---|
| `otel-collector` | `otel-collector:8889` | Métricas exportadas pelo collector. |
| `service-identity` | `service-identity:8080/metrics` | HTTP RED, runtime e negócio. |
| `service-task` | `service-task:8080/metrics` | Inclui métricas do outbox. |
| `service-notification` | `service-notification:8080/metrics` | Inclui Ack/Nack. |
| `rabbitmq` | `rabbitmq:15692/metrics` | Plugin `rabbitmq_prometheus`. |

O container roda com `--storage.tsdb.retention.time=7d` e grava em
`/prometheus` (volume `prometheus-data`).

## grafana

### Datasources

- `Prometheus` (`uid: prometheus`): `http://prometheus:9090`, método `POST`,
  padrão e não editável.
- `Tempo` (`uid: tempo`): `http://tempo:3200`, não editável. `tracesToLogsV2`
  consulta o Loki por `trace_id` e mapeia `service.name` para `service_name`.
- `Loki` (`uid: loki`): `http://loki:3100`, não editável. O campo derivado
  `TraceID` casa `"trace_id":"(\w+)"` e abre o trace no Tempo.

### Dashboards

Provider `task-manager` lê JSON de `/var/lib/grafana/dashboards`, pasta
`Task Manager`, atualiza a cada `10 s` e não permite edição pela UI
(`allowUiUpdates: false`). Os arquivos ficam em `dashboard-definitions/`:

| Dashboard | Conteúdo |
|---|---|
| `trace-explorer.json` — *Task Manager - Observabilidade Unificada* | Resumo operacional, métricas HTTP e de mensageria, traces recentes, busca exata por `trace_id` e logs correlacionados. |
| `backend-services.json` — *Task Manager - Backends Go* | Disponibilidade, taxa HTTP, erros 5xx, latência p95/p99, goroutines e memória. |
| `business-overview.json` — *Task Manager - Visão de Negócio* | Cadastros, logins, tarefas criadas, eventos processados, resultados de auth/tarefas, outbox e consumo. |
| `rabbitmq.json` — *Task Manager - RabbitMQ* | Conexões, canais, consumidores, profundidade de filas (incluindo DLQ), fluxo, uso do nó e alarmes. |

Para adicionar um dashboard, inclua o JSON em `dashboard-definitions/` com
`uid` próprio; o provider carrega em até 10 s, sem reiniciar o Grafana.

### Alertas e plugins

`alerting/default.yml` e `plugins/default.yml` são placeholders vazios: não há
regras de alerta nem plugins provisionados. Alertas, se necessários, devem ser
adicionados nesses arquivos e não pela UI, porque o provisionamento é
somente-leitura.

### Credenciais

Usuário e senha do admin vêm de `GRAFANA_ADMIN_USER` e
`GRAFANA_ADMIN_PASSWORD` no `.env`; cadastro público está desabilitado
(`GF_USERS_ALLOW_SIGN_UP=false`).

## rabbitmq

- `enabled_plugins` habilita `rabbitmq_management` e `rabbitmq_prometheus`.
- `rabbitmq.conf` define `prometheus.return_per_object_metrics = true` para
  separar métricas da fila principal e da DLQ.
- Management em `127.0.0.1:15672`; AMQP (`5672`) e métricas (`15692`) ficam apenas
  nas redes internas.

## Validação

```sh
# compose completo (a partir da raiz)
docker compose --env-file .env.example config -q

# Tempo
docker run --rm -v "$PWD/provisioning/tempo/tempo.yaml:/etc/tempo.yaml:ro" \
  grafana/tempo:3.1.1 -config.file=/etc/tempo.yaml -config.verify=true

# OpenTelemetry Collector
docker run --rm -v "$PWD/provisioning/otel-collector/config.yaml:/etc/otelcol-contrib/config.yaml:ro" \
  otel/opentelemetry-collector-contrib:0.164.0 \
  validate --config=/etc/otelcol-contrib/config.yaml

# Prometheus
docker run --rm -v "$PWD/provisioning/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
  --entrypoint /bin/promtool prom/prometheus:v3.15.1 \
  check config /etc/prometheus/prometheus.yml

# Loki
docker run --rm -v "$PWD/provisioning/loki/loki.yaml:/etc/loki/config.yaml:ro" \
  grafana/loki:3.6.7 -config.file=/etc/loki/config.yaml -verify-config

# Alloy
docker run --rm -v "$PWD/provisioning/alloy/config.alloy:/etc/alloy/config.alloy:ro" \
  grafana/alloy:v1.21.3 validate /etc/alloy/config.alloy
```

Depois de subir a stack, confirme em `http://127.0.0.1:3000` que os três
dashboards aparecem na pasta **Task Manager** e que Prometheus, Tempo e Loki
respondem como datasources saudáveis. Crie uma tarefa via `http://127.0.0.1:8081`
e valide que o mesmo `trace_id` aparece no access log do NGINX, nos logs de
`service-task`/`service-notification` e no trace do Tempo.
