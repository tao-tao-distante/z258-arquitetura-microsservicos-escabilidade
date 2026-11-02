# AGENTS.md

## Skills relevantes ativadas

Projeto usa as skills abaixo:

### Arquitetura, design e planejamento
- `graphify` - Fonte padrão para navegar, consultar e compreender o repositório
- `documentation-and-adrs` - Manter `docs/ARCHITECTURE.md`, alinhar ADRs em `docs/adrs/`
- `domain-modeling` - Estruturar domínios Identity, Task e Notification
- `api-and-interface-design` - Design endpoints REST HTTP e contratos AMQP
- `planning-and-task-breakdown` - Organização roteiro de execução
- `deslop-writing` - Escrita concreta em português, com voz ativa, títulos em sentence case e sem fórmulas genéricas
- `debloat` - Corte de repetição e palavras sem função, sem remover fatos, limites ou decisões

### Desenvolvimento backend em Go
- `golang-project-layout` - Organização idiomática de pastas Go
- `golang-code-style`, `golang-naming`, `golang-structs-interfaces` - Padronização Go
- `golang-context` - Repassar `context.Context` em middlewares, DBs e RabbitMQ
- `golang-error-handling` - Tratamento explícito de erros com `if err != nil`
- `golang-samber-slog` & `golang-observability` - Logging JSON estruturado com `log/slog`
- `golang-database` - Conexão isolada com PostgreSQL e migrations
- `golang-concurrency` - Consumo assíncrono e eventos AMQP
- `golang-testing` & `test-driven-development` - Ciclo Red-Green-Refactor, cobertura agregada >= 95% por módulo
- `golang-lint` - Código sem warnings

### Desenvolvimento frontend
- `frontend-design` & `frontend-ui-engineering` - Componentização SPA Angular, rotas e interceptor HTTP

### Observabilidade, qualidade e Git
- `observability-and-instrumentation` - Métricas Prometheus, traces OpenTelemetry/Tempo, Health Checks
- `code-review-and-quality` & `code-simplification` - Refatoração e limpeza pós-TDD
- `conventional-git` & `git-workflow-and-versioning` - Convenções Git

## Escrita no repositório

- Carregar `deslop-writing` antes de criar ou alterar README, ADR, documentação, texto de interface ou comentário extenso.
- Escrever em português claro e concreto, com voz ativa e títulos em sentence case.
- Não usar travessão, aspas curvas, conclusão decorativa, rótulo em negrito no início de bullet nem introdução genérica.
- Preservar comandos, números, URLs, nomes técnicos, fatos arquiteturais, limites e incertezas.
- Aplicar `debloat` em toda prosa do repositório. Cortar repetição e enchimento sem remover conteúdo necessário.
- Não inventar nomes de integrantes nem fatos ausentes. Manter placeholder ou registrar a informação faltante.

## Graphify por padrão

- Sempre carregar a skill `graphify` antes de explorar ou responder perguntas sobre arquitetura, fluxos, dependências, símbolos ou conteúdo do repositório.
- Tratar `graphify-out/graph.json` como o mapa inicial do projeto. Quando existir, executar `graphify query "<pergunta>"` antes de buscas amplas com Glob, Grep ou leitura manual.
- Usar o resultado do grafo para localizar os arquivos relevantes; confirmar detalhes no código-fonte antes de editar ou afirmar comportamento.
- Se o grafo não existir, executar `/graphify .` para construí-lo. Não pedir caminho quando a raiz do projeto for o alvo.
- Depois de alterar arquivos rastreados pelo corpus, executar `/graphify . --update` para manter o grafo sincronizado.
- Nunca inventar relações ausentes no grafo. Sinalizar lacunas e citar `source_location` ao sustentar fatos específicos.

## Estrutura e arquitetura

### Microsserviços Go
- **service-identity** (interno `:8080`): Autenticação, registro, login, perfil (nome e troca de senha), emissão/validação JWT. PostgreSQL isolado (postgres-identity).
- `service-task` (interno `:8080`): CRUD de tarefas protegido por JWT, com prioridade (`low|medium|high|urgent`), prazo (`due_date`) e histórico de alterações (`task_history`, ADR-0013). PostgreSQL isolado (postgres-task). Grava a tarefa, o histórico e o evento `task.created` em `task_outbox` na mesma transação; `trace_context` preserva o carrier W3C em JSONB. Um dispatcher publica no RabbitMQ com publisher confirms, `mandatory`, retry/backoff e `x-correlation-id` nos headers AMQP.
- **service-notification** (interno `:8080`): Consumidor AMQP assíncrono. Extrai `x-correlation-id` dos headers AMQP. Suporte a Dead Letter Queue (DLQ): evento inválido recebe `Nack(false,false)` e segue para `task.events.dlq` via `tasks.dlx`.

### Frontend
- **frontend** (8081:80): SPA Angular 20 servida via NGINX. `ngx_otel_module` cria o trace na borda, injeta `traceparent`, exporta OTLP/gRPC e balanceia upstreams com `least_conn`; Angular não cria traces ou correlation IDs.
- Estrutura feature-first (ADR-0010): `src/app/core/{guards,http,interceptors,models,services}`, `src/app/shared/components`, `src/app/features/{auth,dashboard,tasks,notifications,profile}`.
- Tela `/profile` (`features/profile`): edita o nome e troca a senha via `/api/identity/me`.
- Tarefas (`features/tasks`): formulário com prioridade e prazo; lista com filtros de prioridade/atrasadas e ordenação no cliente (estado na URL); detalhe com linha do tempo do histórico (`TaskHistoryComponent`). Regras de data em `features/tasks/task.utils.ts`.
- URLs relativas em `src/environments/environment.ts` (`/api/identity`, `/api/tasks`, `/api/notifications`), resolvidas pelo proxy do `frontend/nginx.conf` para evitar CORS.
- Build de produção em `frontend/dist/frontend/browser`; Dockerfile copia esse diretório e o `nginx.conf` para o container.

### Mensageria e persistência
- **RabbitMQ**: Broker AMQP interno na porta `5672` (sem publicação no host) e Management em `127.0.0.1:15672`. Healthcheck `rabbitmq-diagnostics -q ping`.
- **Topologia AMQP**: exchange `tasks` (topic), fila `task.events`, routing key `task.created`, exchange `tasks.dlx` (fanout) e DLQ `task.events.dlq`.
- **service-task** publica via outbox transacional e **service-notification** consome. Ambos declaram topologia via `shared/broker` de forma idempotente no startup; consumidor não depende da ordem de subida dos serviços. Publisher usa confirms com `mandatory`; entrega é at-least-once e consumidores devem deduplicar por `MessageId`/`event_id`.
- **PostgreSQL 15**: Database-per-service (ADR-0001). `postgres-identity` e `postgres-task` com healthcheck `pg_isready`.

### Endpoints implementados
- **service-identity**: `POST /register` (201), `POST /login` (200), `GET /health` (200/503). JWT obrigatório: `GET /me` (200), `PATCH /me` (200; altera `name`), `PUT /me/password` (204; 403 se a senha atual estiver incorreta).
- `service-task` (JWT obrigatório): `GET /tasks` (filtros `status`, `priority`, `overdue`; `sort` = `position|priority|created_at|due_date`; `order`), `POST /tasks` (201 com `{"id": number}`), `GET /tasks/{id}`, `PUT /tasks/{id}` (substitui tudo e devolve `{"id": number}`), `DELETE /tasks/{id}` (204), `PATCH /tasks/{id}/move` (200/409; reposiciona no quadro), `GET /tasks/{id}/history` (200/404; mais recentes primeiro), `GET /health`. Respostas públicas não incluem `user_id`.
- **service-notification**: `GET /health`; consumo assíncrono de `task.events` com Ack/Nack e DLQ.

### Observabilidade
- **otel-collector**: Recebe métricas e traces via OTLP gRPC (4317). Config em `provisioning/otel-collector/config.yaml`.
- **prometheus**: Coleta `/metrics` dos três serviços, Collector e RabbitMQ. Config em `provisioning/prometheus/prometheus.yml`.
- **tempo**: Armazenamento de traces distribuídos. Config em `provisioning/tempo/tempo.yaml`.
- **loki**: Logs centralizados em modo monolítico, TSDB v13, filesystem e retenção de 7 dias. Sem porta publicada. Config em `provisioning/loki/loki.yaml`.
- **alloy**: Coleta os arquivos `*-json.log` do diretório de containers do Docker (`DOCKER_ROOT_DIR`; rootless usa `$HOME/.local/share/docker`), desembrulha com `stage.docker` e envia ao Loki. Labels: `service_name`, `level`, `container`. Config em `provisioning/alloy/config.alloy`.
- **grafana**: Visualização em `127.0.0.1:3000`. Datasources Prometheus, Tempo e Loki com links Loki↔Tempo e quatro dashboards provisionados em `provisioning/grafana/`, incluindo observabilidade unificada com busca por `trace_id`.
- **rabbitmq**: Plugin `rabbitmq_prometheus` em `:15692` somente na rede interna; métricas por objeto separam fila e DLQ.
- Os serviços exportam traces para `http://otel-collector:4317` e métricas HTTP/runtime/negócio em `/metrics`.
- Logs JSON carregam `service_name`; `trace_id`/`span_id` vêm do span ativo e `correlation_id` do contexto. IDs são campos JSON, nunca labels ou prefixos de mensagem.
- Métricas de negócio: autenticação, CRUD de tarefas, dispatch do outbox e consumo Ack/Nack. Nunca usar IDs ou URLs brutas como labels.
- Validar configs com `docker compose --env-file .env.example config -q` e comandos de validação do `README.md`/`provisioning/README.md`.

### Rede e segurança
- Entrada pública apenas pelo NGINX em `0.0.0.0:8081`.
- APIs não publicam portas no host; Grafana (`3000`) e RabbitMQ Management (`15672`) ficam vinculados a `127.0.0.1`.
- PostgreSQL, AMQP (`5672`), OTLP (`4317`), Tempo, Prometheus e Loki não publicam portas no host.
- `/api/*/metrics` é bloqueado no NGINX; as respostas de `/api/*` incluem `X-Trace-ID`.
- `.env` contém segredos aleatórios e distintos, é gitignored e nunca deve ser commitado; use `.env.example` apenas para interpolação.
- JWT no `localStorage` é decisão consciente com risco residual de XSS; mitigações em `frontend/SECURITY.md` (CSP restritiva, escape Angular, JWT só em `/api/tasks` e `/api/identity/me*`, expiração validada, limpeza em `401`).
- Rate limit de `10 req/min/IP` em `/register`, `/login` e `/me/password`, com `429` e `Retry-After: 60`.
- Limites de payload: `16 KiB` em Identity e `1 MiB` em Tasks; excedentes recebem `413`.
- Contratos de erro: `400` validação, `401` autenticação, `403` senha atual incorreta na troca de senha, `404` recurso, `409` conflito, `413` payload, `429` rate limit, `500` interno, `503` dependência.

## Mapeamento dos ADRs

| ADR                                                                  | Decisão aplicada no código                                                                                                 |
|----------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------|
| [0001](docs/adrs/0001-database-per-service.md)                       | PostgreSQL 15 isolado por serviço; nenhum acesso cruzado entre `postgres-identity` e `postgres-task`.                      |
| [0002](docs/adrs/0002-correlation-id-e-observabilidade.md)           | `x-correlation-id` UUID v4 em HTTP/AMQP, `context.Context`, logs `slog`, `otelhttp` e OTLP com fallback no-op.             |
| [0003](docs/adrs/0003-comunicacao-assincrona-rabbitmq.md)            | Exchange `tasks`, fila `task.events`, DLX/DLQ, publisher confirms, `mandatory`, retry/backoff e consumidor idempotente.    |
| [0004](docs/adrs/0004-adocao-arquitetura-microsservicos.md)          | Domínios Identity, Task e Notification com bancos e processos independentes.                                               |
| [0005](docs/adrs/0005-framework-frontend-angular.md)                 | Angular 20 standalone, Signals, reactive forms, lazy routes e build estrito.                                               |
| [0006](docs/adrs/0006-backend-go-golang.md)                          | Go 1.23, `net/http`, `log/slog`, `context.Context` e tratamento explícito de erros.                                        |
| [0007](docs/adrs/0007-autenticacao-stateless-jwt.md)                 | JWT HS256 emitido pelo Identity, validado pelo Task, sem sessão no servidor.                                               |
| [0008](docs/adrs/0008-containerizacao-docker-compose.md)             | Multi-stage builds, `CGO_ENABLED=0`, Compose com healthchecks, redes e volumes nomeados.                                   |
| [0009](docs/adrs/0009-estrutura-e-layout-de-projeto-go.md)           | `cmd/`, `internal/` e `shared/` com dependências via `replace ../shared`.                                                  |
| [0010](docs/adrs/0010-estrutura-e-arquitetura-de-projeto-angular.md) | Feature-first em `core/{guards,http,interceptors,models,services}`, `shared/components` e `features/*`.                    |
| [0011](docs/adrs/0011-hardening-infraestrutura-local.md)             | Portas administrativas em loopback, segredos via `.env`, CSP/headers NGINX e containers non-root quando aplicável.         |
| [0012](docs/adrs/0012-metricas-prometheus-e-dashboards-grafana.md)   | Scrape Prometheus, métricas de negócio, plugin RabbitMQ e quatro dashboards provisionados.                                 |
| [0013](docs/adrs/0013-prioridade-prazo-e-historico-de-tarefas.md)    | Prioridade e prazo nas tarefas; `task_history` gravado na mesma transação (`created`/`updated`/`moved`), sem reordenações. |
| [0014](docs/adrs/0014-tracing-na-borda-e-balanceamento-nginx.md)     | NGINX cria W3C trace, exporta OTLP/gRPC e balanceia upstreams; outbox/AMQP preservam a árvore até o consumer.              |
| [0015](docs/adrs/0015-logs-contextuais-loki-alloy.md)                | Logs JSON com `service_name`/`trace_id`/`span_id`/`correlation_id`, bodies sanitizados, Loki+Alloy e links Loki↔Tempo.     |
| [0016](docs/adrs/0016-slis-slos-e-fontes-de-medicao.md)              | SLIs e SLOs com janelas, populações, denominadores, fontes e tratamento explícito de eventos pendentes no outbox.          |

## Padrões de código Go

### Obrigatórios
- **Logs estruturados**: Usar `log/slog` exclusivamente. Logger criado com `logger.New("<service-name>")`; dentro de requests/eventos usar `logger.WithContext(ctx, log)` para acrescentar `correlation_id`, `trace_id` e `span_id`. Nunca logar `title`, `description`, `email`, `name`, `password`, `token` nem bodies integrais.
- **Correlation ID**: Middleware HTTP aceita `x-correlation-id` apenas quando é UUID v4 válido; ausente ou inválido, gera um novo. Propagar no `context.Context` e no header de resposta. Em eventos AMQP, injetar/extrair dos headers.
- **Tratamento de erros**: Sempre explícito com `if err != nil`. Sem `_` ignorando erros sem justificativa.
- **Context**: Repassar `context.Context` em middlewares, handlers, repositórios, DB e clientes RabbitMQ.
- **Idiomas Go**: Seguir `golang-project-layout`, `golang-code-style`, `golang-naming`, `golang-structs-interfaces`.
- **Health Check**: Endpoint `GET /health` retornando 200 OK com status de dependências (DB/RabbitMQ).

### Docker
- **Multi-stage builds**: Go (golang:1.23-alpine -> alpine:3.20, `CGO_ENABLED=0`), Angular (node:20.19-alpine -> nginx:1.31.8-alpine com `nginx-module-otel`). O NGINX copia `nginx-main.conf`, `nginx.conf` e `security-headers.conf`.
- Os quatro `go.mod` (três serviços + `shared/`) declaram `go 1.23`. Manter dependências compatíveis com Go 1.23: `golang.org/x/crypto v0.41.0` é o mínimo exigido pelo `grpc v1.75` e declara `go 1.23`; não subir para versões que exijam Go mais novo.
- Código comum entre serviços vive em `shared/` (`auth`, `broker`, `correlation`, `httpx`, `logger`, `middleware`, `telemetry`) e é consumido via `replace ../shared`. Context de build Docker é a raiz do repositório (ver `.dockerignore`).
- Validar com `docker compose build` e `docker compose config -q`.

## Fluxo de desenvolvimento em TDD

**REGRA RIGOROSA**: Ciclo Red-Green-Refactor.
1. **RED**: Escrever teste falhando primeiro (*_test.go) com table-driven tests
2. **GREEN**: Implementar código mínimo para fazer o teste passar
3. **REFACTOR**: Melhorar código mantendo testes verdes

**Nenhum código de produção sem teste automatizado falhando previamente.**

## Metas de qualidade
- **Cobertura mínima agregada**: >= 95% por módulo Go (unitários e integração). Pacotes `cmd/` e `internal/domain` podem ficar abaixo isoladamente; a meta é o total do módulo.
- Cobertura agregada atual por módulo (sem `-race`): `shared` 97.6%, `service-identity` 99.1%, `service-task` 98.3%, `service-notification` 95.3%.
- Validar com: `go test -v -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
- Frontend: `npm run build` com `strict` e `strictTemplates`; testes com `npm test -- --no-watch --code-coverage` (requer `@vitest/coverage-v8`); não commitar `node_modules/`, `dist/`, `.angular/`, `coverage/` ou `coverage.out`.
- Smoke test manual: subir `docker compose up -d`, registrar usuário, criar/listar tarefa com JWT e confirmar log `notification_received` no `service-notification` com o `correlation_id` do evento. Confirmar a mesma `trace_id` no NGINX, em `service-task`, no dispatcher e no `service-notification`, além dos links Loki↔Tempo no Grafana.
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

## Comandos de desenvolvimento
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
- [Glossário](GLOSSARY.md)
- [Segurança do frontend](frontend/SECURITY.md)
- [Compose](compose.yml)
