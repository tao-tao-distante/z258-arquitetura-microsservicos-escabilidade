# AGENTS.md

## Skills Relevantes Ativadas

Projeto usa as skills abaixo:

### Arquitetura, Design e Planejamento
- `documentation-and-adrs` - Manter `docs/ARCHITECTURE.md`, alinhar ADRs em `docs/adrs/`
- `domain-modeling` - Estruturar domínios Identity, Task e Notification
- `api-and-interface-design` - Design endpoints REST HTTP e contratos AMQP
- `planning-and-task-breakdown` - Organização roteiro de execução

### Desenvolvimento Backend em Go
- `golang-project-layout` - Organização idiomática de pastas Go
- `golang-code-style`, `golang-naming`, `golang-structs-interfaces` - Padronização Go
- `golang-context` - Repassar `context.Context` em middlewares, DBs e RabbitMQ
- `golang-error-handling` - Tratamento explícito de erros com `if err != nil`
- `golang-samber-slog` & `golang-observability` - Logging JSON estruturado com `log/slog`
- `golang-database` - Conexão isolada com PostgreSQL e migrations
- `golang-concurrency` - Consumo assíncrono e eventos AMQP
- `golang-testing` & `test-driven-development` - Ciclo Red-Green-Refactor, cobertura agregada >= 95% por módulo
- `golang-lint` - Código sem warnings

### Desenvolvimento Frontend
- `frontend-design` & `frontend-ui-engineering` - Componentização SPA Angular, rotas e interceptor HTTP

### Observabilidade, Qualidade e Git
- `observability-and-instrumentation` - Métricas Prometheus, traces OpenTelemetry/Tempo, Health Checks
- `code-review-and-quality` & `code-simplification` - Refatoração e limpeza pós-TDD
- `conventional-git` & `git-workflow-and-versioning` - Convenções Git

## Estrutura e Arquitetura

### Microsserviços Go
- **service-identity** (8082:8080): Autenticação, registro, login, emissão/validação JWT. PostgreSQL isolado (postgres-identity).
- **service-task** (8083:8080): CRUD de tarefas protegido por JWT. PostgreSQL isolado (postgres-task). Grava a tarefa e o evento `task.created` em `task_outbox` na mesma transação; um dispatcher publica no RabbitMQ com publisher confirms, `mandatory`, retry/backoff e `x-correlation-id` nos headers AMQP.
- **service-notification** (8084:8080): Consumidor AMQP assíncrono. Extrai `x-correlation-id` dos headers AMQP. Suporte a Dead Letter Queue (DLQ): evento inválido recebe `Nack(false,false)` e segue para `task.events.dlq` via `tasks.dlx`.

### Frontend
- **frontend** (8081:80): SPA Angular 20 servida via NGINX. Interceptor HTTP global injeta `x-correlation-id` (UUID v4) em todas as requisições.
- Estrutura feature-first (ADR-0010): `src/app/core/{guards,http,interceptors,models,services}`, `src/app/shared/components`, `src/app/features/{auth,dashboard,tasks,notifications}`.
- URLs relativas em `src/environments/environment.ts` (`/api/identity`, `/api/tasks`, `/api/notifications`), resolvidas pelo proxy do `frontend/nginx.conf` para evitar CORS.
- Build de produção em `frontend/dist/frontend/browser`; Dockerfile copia esse diretório e o `nginx.conf` para o container.

### Mensageria e Persistência
- **RabbitMQ**: Broker AMQP interno na porta `5672` (sem publicação no host) e Management em `127.0.0.1:15672`. Healthcheck `rabbitmq-diagnostics -q ping`.
- **Topologia AMQP**: exchange `tasks` (topic), fila `task.events`, routing key `task.created`, exchange `tasks.dlx` (fanout) e DLQ `task.events.dlq`.
- **service-task** publica via outbox transacional e **service-notification** consome. Ambos declaram topologia via `shared/broker` de forma idempotente no startup; consumidor não depende da ordem de subida dos serviços. Publisher usa confirms com `mandatory`; entrega é at-least-once e consumidores devem deduplicar por `MessageId`/`event_id`.
- **PostgreSQL 15**: Database-per-service (ADR-0001). `postgres-identity` e `postgres-task` com healthcheck `pg_isready`.

### Endpoints Implementados
- **service-identity**: `POST /register` (201), `POST /login` (200), `GET /health` (200/503).
- **service-task** (JWT obrigatório): `GET /tasks`, `POST /tasks`, `GET /tasks/{id}`, `PUT /tasks/{id}`, `DELETE /tasks/{id}` (204), `GET /health`.
- **service-notification**: `GET /health`; consumo assíncrono de `task.events` com Ack/Nack e DLQ.

### Observabilidade
- **otel-collector**: Recebe métricas e traces via OTLP gRPC (4317). Config em `provisioning/otel-collector/config.yaml`.
- **prometheus**: Coleta métricas. Config em `provisioning/prometheus/prometheus.yml`.
- **tempo**: Armazenamento de traces distribuídos. Config em `provisioning/tempo/tempo.yaml`.
- **grafana**: Visualização em `127.0.0.1:3000`. Datasources em `provisioning/grafana/datasources/datasources.yml`.
- Os três serviços enviam telemetria para `http://otel-collector:4317` via `OTEL_EXPORTER_OTLP_ENDPOINT`.
- Validar configs com `docker compose --env-file .env.example config -q` e comandos de validação do `README.md`.

### Rede e Segurança
- Entrada pública apenas pelo NGINX em `0.0.0.0:8081`.
- APIs diretas (`8082`, `8083`, `8084`), Grafana (`3000`) e RabbitMQ Management (`15672`) vinculadas a `127.0.0.1`.
- PostgreSQL, AMQP (`5672`), OTLP (`4317`), Tempo e Prometheus não publicam portas no host.
- `.env` contém segredos aleatórios e distintos, é gitignored e nunca deve ser commitado; use `.env.example` apenas para interpolação.
- JWT no `localStorage` é decisão consciente com risco residual de XSS; mitigações em `frontend/SECURITY.md` (CSP restritiva, escape Angular, JWT só em `/api/tasks`, expiração validada, limpeza em `401`).
- Rate limit de `10 req/min/IP` em `/register` e `/login`, com `429` e `Retry-After: 60`.
- Limites de payload: `16 KiB` em Identity e `1 MiB` em Tasks; excedentes recebem `413`.
- Contratos de erro: `400` validação, `401` autenticação, `404` recurso, `409` conflito, `413` payload, `429` rate limit, `500` interno, `503` dependência.

## Mapeamento dos ADRs

| ADR                                                                  | Decisão aplicada no código                                                                                              |
|----------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------|
| [0001](docs/adrs/0001-database-per-service.md)                       | PostgreSQL 15 isolado por serviço; nenhum acesso cruzado entre `postgres-identity` e `postgres-task`.                   |
| [0002](docs/adrs/0002-correlation-id-e-observabilidade.md)           | `x-correlation-id` UUID v4 em HTTP/AMQP, `context.Context`, logs `slog`, `otelhttp` e OTLP com fallback no-op.          |
| [0003](docs/adrs/0003-comunicacao-assincrona-rabbitmq.md)            | Exchange `tasks`, fila `task.events`, DLX/DLQ, publisher confirms, `mandatory`, retry/backoff e consumidor idempotente. |
| [0004](docs/adrs/0004-adocao-arquitetura-microsservicos.md)          | Domínios Identity, Task e Notification com bancos e processos independentes.                                            |
| [0005](docs/adrs/0005-framework-frontend-angular.md)                 | Angular 20 standalone, Signals, reactive forms, lazy routes e build estrito.                                            |
| [0006](docs/adrs/0006-backend-go-golang.md)                          | Go 1.23, `net/http`, `log/slog`, `context.Context` e tratamento explícito de erros.                                     |
| [0007](docs/adrs/0007-autenticacao-stateless-jwt.md)                 | JWT HS256 emitido pelo Identity, validado pelo Task, sem sessão no servidor.                                            |
| [0008](docs/adrs/0008-containerizacao-docker-compose.md)             | Multi-stage builds, `CGO_ENABLED=0`, Compose com healthchecks, redes e volumes nomeados.                                |
| [0009](docs/adrs/0009-estrutura-e-layout-de-projeto-go.md)           | `cmd/`, `internal/` e `shared/` com dependências via `replace ../shared`.                                               |
| [0010](docs/adrs/0010-estrutura-e-arquitetura-de-projeto-angular.md) | Feature-first em `core/{guards,http,interceptors,models,services}`, `shared/components` e `features/*`.                      |
| [0011](docs/adrs/0011-hardening-infraestrutura-local.md)             | Portas administrativas em loopback, segredos via `.env`, CSP/headers NGINX e containers non-root quando aplicável.      |

## Padrões de Código Go

### Obrigatórios
- **Logs estruturados**: Usar `log/slog` exclusivamente. Todos os logs em JSON com chave `correlation_id`. Extrair do `context.Context`.
- **Correlation ID**: Middleware HTTP aceita `x-correlation-id` apenas quando é UUID v4 válido; ausente ou inválido, gera um novo. Propagar no `context.Context` e no header de resposta. Em eventos AMQP, injetar/extrair dos headers.
- **Tratamento de erros**: Sempre explícito com `if err != nil`. Sem `_` ignorando erros sem justificativa.
- **Context**: Repassar `context.Context` em middlewares, handlers, repositórios, DB e clientes RabbitMQ.
- **Idiomas Go**: Seguir `golang-project-layout`, `golang-code-style`, `golang-naming`, `golang-structs-interfaces`.
- **Health Check**: Endpoint `GET /health` retornando 200 OK com status de dependências (DB/RabbitMQ).

### Docker
- **Multi-stage builds**: Go (golang:1.23-alpine -> alpine:3.20, `CGO_ENABLED=0`), Angular (node:20.19-alpine -> nginx:1.31.6-alpine).
- Os quatro `go.mod` (três serviços + `shared/`) declaram `go 1.23`. Manter dependências compatíveis com Go 1.23: `golang.org/x/crypto v0.41.0` é o mínimo exigido pelo `grpc v1.75` e declara `go 1.23`; não subir para versões que exijam Go mais novo.
- Código comum entre serviços vive em `shared/` (`auth`, `broker`, `correlation`, `httpx`, `logger`, `middleware`, `telemetry`) e é consumido via `replace ../shared`. Context de build Docker é a raiz do repositório (ver `.dockerignore`).
- Validar com `docker compose build` e `docker compose config -q`.

## Fluxo de Desenvolvimento em TDD

**REGRA RIGOROSA**: Ciclo Red-Green-Refactor.
1. **RED**: Escrever teste falhando primeiro (*_test.go) com table-driven tests
2. **GREEN**: Implementar código mínimo para fazer o teste passar
3. **REFACTOR**: Melhorar código mantendo testes verdes

**Nenhum código de produção sem teste automatizado falhando previamente.**

## Metas de Qualidade
- **Cobertura mínima agregada**: >= 95% por módulo Go (unitários e integração). Pacotes `cmd/` e `internal/domain` podem ficar abaixo isoladamente; a meta é o total do módulo.
- Cobertura agregada atual por módulo (sem `-race`): `shared` 97.6%, `service-identity` 98.6%, `service-task` 98.3%, `service-notification` 95.3%.
- Validar com: `go test -v -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
- Frontend: `npm run build` com `strict` e `strictTemplates`; testes com `npm test -- --no-watch --code-coverage` (requer `@vitest/coverage-v8`); não commitar `node_modules/`, `dist/`, `.angular/`, `coverage/` ou `coverage.out`.
- Smoke test manual: subir `docker compose up -d`, registrar usuário, criar/listar tarefa com JWT e confirmar log `notification_received` no `service-notification` com o `correlation_id` do evento.
- Encerrar ambiente de teste com `docker compose down -v` para remover volumes criados.

## Convenções Git
Seguir **Conventional Commits**:
- `feat:` - Nova funcionalidade
- `fix:` - Correção de bug
- `test:` - Adição/correção de testes
- `docs:` - Documentação
- `refactor:` - Refatoração sem mudança de comportamento
- `chore:` - Manutenção/infraestrutura
- `style:` - Formatação

## Comandos de Desenvolvimento
```sh
# Backend (a partir da raiz)
(cd shared && go test -race ./...)
(cd service-identity && go test -race ./...)
(cd service-task && go test -race ./...)
(cd service-notification && go test -race ./...)

# Cobertura (executar dentro de cada módulo)
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

# Infraestrutura
docker compose --env-file .env.example config -q
docker run --rm -v "$PWD/provisioning/tempo/tempo.yaml:/etc/tempo.yaml:ro" \
  grafana/tempo:3.1.1 -config.file=/etc/tempo.yaml -config.verify=true
docker run --rm -v "$PWD/provisioning/otel-collector/config.yaml:/etc/otelcol-contrib/config.yaml:ro" \
  otel/opentelemetry-collector-contrib:0.164.0 \
  validate --config=/etc/otelcol-contrib/config.yaml
docker run --rm -v "$PWD/provisioning/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
  --entrypoint /bin/promtool prom/prometheus:v3.15.1 \
  check config /etc/prometheus/prometheus.yml

# Frontend
cd frontend && npm install && npm run build
npm test -- --no-watch --code-coverage
npm run start   # dev server em http://localhost:4200

# Stack completa
docker compose up --build -d
docker compose logs -f service-notification
docker compose down -v
```

## Referências
- [Arquitetura](docs/ARCHITECTURE.md)
- [ADRs](docs/adrs/)
- [README](README.md)
- [Segurança do frontend](frontend/SECURITY.md)
- [Compose](compose.yml)
