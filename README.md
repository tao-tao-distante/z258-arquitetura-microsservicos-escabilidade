# Task Manager System

Plataforma de gerenciamento de tarefas com três microsserviços Go, SPA Angular, PostgreSQL isolado por domínio,
RabbitMQ e uma stack local de observabilidade. Consulte [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) e os
[ADRs](docs/adrs/).

## Arquitetura

```text
cliente -> frontend/NGINX :8081 -> service-identity :8080 -> postgres-identity
                              |-> service-task     :8080 -> postgres-task
                                      |                     |
                                      +---- RabbitMQ <------ service-notification

OTLP -> OpenTelemetry Collector -> Tempo (traces)
                              \-> Prometheus (metrics) -> Grafana
```

O NGINX é a entrada pública da aplicação em `0.0.0.0:8081`. APIs diretas, Grafana e RabbitMQ Management são portas
de desenvolvimento/administração vinculadas a `127.0.0.1`. Bancos, AMQP, OTLP, Tempo e Prometheus não publicam portas
no host.

## Setup

1. Crie o arquivo local de segredos:

   ```sh
   cp .env.example .env
   ```

2. Substitua todos os valores de exemplo. Use senhas aleatórias e distintas. Como as senhas dos bancos e do RabbitMQ
   são inseridas em URLs, use caracteres alfanuméricos ou aplique percent-encoding.

3. Valide e suba a stack:

   ```sh
   docker compose config -q
   docker compose up --build -d
   docker compose ps
   ```

O Compose falha imediatamente se uma variável obrigatória estiver ausente. O usuário RabbitMQ vem de `.env`; o usuário
`guest` não é utilizado pelos containers da aplicação.

| Componente | URL local | Exposição |
|---|---|---|
| Frontend | http://localhost:8081 | todas as interfaces |
| service-identity | http://localhost:8082 | somente loopback |
| service-task | http://localhost:8083 | somente loopback |
| service-notification | http://localhost:8084 | somente loopback |
| Grafana | http://localhost:3000 | somente loopback |
| RabbitMQ Management | http://localhost:15672 | somente loopback |

Grafana recebe as credenciais de `.env`. Os datasources Prometheus e Tempo são provisionados automaticamente em
`provisioning/grafana/datasources/datasources.yml`.

```sh
docker compose logs -f service-notification
docker compose down       # preserva volumes nomeados
docker compose down -v    # remove dados locais persistidos
```

## API

### Identity

| Método | Rota | Descrição | Sucesso |
|---|---|---|---|
| POST | `/register` | Cria usuário e devolve JWT | `201` |
| POST | `/login` | Autentica e devolve JWT | `200` |
| GET | `/health` | Verifica serviço e PostgreSQL | `200` ou `503` |

Cadastro/login: `{"email":"user@example.com","password":"secret123"}`.

### Tasks

Rotas de tarefas exigem `Authorization: Bearer <jwt>`.

| Método | Rota | Descrição | Sucesso |
|---|---|---|---|
| GET | `/tasks` | Lista tarefas do usuário | `200` |
| POST | `/tasks` | Cria tarefa | `201` |
| GET | `/tasks/{id}` | Busca tarefa | `200` |
| PUT | `/tasks/{id}` | Atualiza tarefa | `200` |
| DELETE | `/tasks/{id}` | Remove tarefa | `204` |
| GET | `/health` | Verifica serviço e dependências | `200` ou `503` |

Corpo de tarefa: `{"title":"...","description":"...","status":"todo|doing|done"}`.

### Notifications

`GET /health` verifica o processo e RabbitMQ. O consumidor recebe `task.created` na fila durável `task.events`.
Evento JSON inválido recebe `Nack` sem requeue e segue para `task.events.dlq` pela exchange `tasks.dlx`. O consumidor não
faz retry próprio; o publisher usa outbox transacional com retry/backoff e entrega at-least-once. Não há entrega de
notificação externa implementada.

## Correlação e telemetria

- `x-correlation-id` é um identificador de aplicação para logs e eventos, não um header W3C Trace Context.
- O frontend gera UUID v4; os serviços devolvem o header, armazenam o valor no `context.Context` e registram
  `correlation_id` nos logs JSON.
- O `service-task` copia `x-correlation-id` para o header AMQP e o consumidor o extrai.
- W3C Trace Context (`traceparent`/`tracestate`) é propagado via `otelhttp`; spans são exportados por OTLP/gRPC quando
  `OTEL_EXPORTER_OTLP_ENDPOINT` está definido. Sem o endpoint, a exportação fica desabilitada e a aplicação segue com
  propagação e provider no-op.
- Collector, Tempo, Prometheus e Grafana recebem traces. Métricas de negócio ainda não são emitidas. Loki não faz parte
  desta stack; logs permanecem no `stdout` dos containers.

## Persistência e redes

PostgreSQL, RabbitMQ, Tempo, Prometheus e Grafana usam volumes nomeados. Redes internas separam dados de Identity, dados
de Task, mensageria e telemetria. A rede `edge` contém apenas frontend e APIs. Cada serviço acessa somente dependências
necessárias ao seu papel.

## Desenvolvimento

```sh
cd shared && go test ./...
cd service-identity && go test ./...
cd service-task && go test ./...
cd service-notification && go test ./...

cd frontend
npm install
npm run start
npm run build
```

Os três serviços importam o módulo local `shared/` (via `replace`) com correlação, logging, middleware, auth, HTTP e
telemetria comuns. Por isso o context de build é a raiz do repositório; o `.dockerignore` da raiz limita o contexto aos
diretórios `shared/` e `service-*/`.

Contexts Docker ignoram dependências, builds, cobertura e arquivos `.env`. Arquivos `.env.*` são ignorados pelo Git,
exceto exemplos e certificados versionados explicitamente.

## Validação de infraestrutura

Com `.env` configurado:

```sh
docker compose config -q
docker run --rm -v "$PWD/tempo.yaml:/etc/tempo.yaml:ro" \
  grafana/tempo:3.1.1 -config.file=/etc/tempo.yaml -config.verify=true
docker run --rm -v "$PWD/otel-collector-config.yaml:/etc/otelcol-contrib/config.yaml:ro" \
  otel/opentelemetry-collector-contrib:0.164.0 \
  validate --config=/etc/otelcol-contrib/config.yaml
docker run --rm -v "$PWD/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
  --entrypoint /bin/promtool prom/prometheus:v3.15.1 \
  check config /etc/prometheus/prometheus.yml
```

Para validar interpolação sem criar `.env`, use `docker compose --env-file .env.example config -q`. Não use o arquivo de
exemplo como credencial de um ambiente compartilhado.

## Decisões arquiteturais

| ADR | Tema |
|---|---|
| [0001](docs/adrs/0001-database-per-service.md) | Database-per-Service com PostgreSQL |
| [0002](docs/adrs/0002-correlation-id-e-observabilidade.md) | Correlação de aplicação e telemetria |
| [0003](docs/adrs/0003-comunicacao-assincrona-rabbitmq.md) | RabbitMQ e DLQ |
| [0004](docs/adrs/0004-adocao-arquitetura-microsservicos.md) | Microsserviços |
| [0005](docs/adrs/0005-framework-frontend-angular.md) | Angular |
| [0006](docs/adrs/0006-backend-go-golang.md) | Go |
| [0007](docs/adrs/0007-autenticacao-stateless-jwt.md) | JWT |
| [0008](docs/adrs/0008-containerizacao-docker-compose.md) | Docker Compose |
| [0009](docs/adrs/0009-estrutura-e-layout-de-projeto-go.md) | Layout Go |
| [0010](docs/adrs/0010-estrutura-e-arquitetura-de-projeto-angular.md) | Layout Angular |
| [0011](docs/adrs/0011-hardening-infraestrutura-local.md) | Hardening da infraestrutura local |
