# service-task

Microsserviço de tarefas: expõe o CRUD autenticado por JWT, com prioridade, prazo e
histórico de alterações, e publica o evento `task.created` no RabbitMQ por meio de um
outbox transacional. É o único serviço que escreve no banco de tarefas.

- Porta interna: `8080` (publicada como `127.0.0.1:8083`).
- Banco exclusivo: `postgres-task` (database `task`).
- Dependências: PostgreSQL e RabbitMQ.
- Valida o JWT localmente; não chama o `service-identity`.

## Estrutura

| Caminho | Responsabilidade |
|---|---|
| `cmd/server` | Bootstrap, sinais, telemetria e adaptação do cliente AMQP. |
| `internal/domain` | Entidade `Task`, `Date`, histórico (`Diff`, `HistoryEntry`), consulta de listagem e erros de domínio. |
| `internal/service` | Validação de título/status/prioridade/prazo, normalização de filtros e regras de CRUD. |
| `internal/repository` | SQL de `tasks`, `task_history` e `task_outbox`, filtros/ordenação, claim/retry e migrações. |
| `internal/handler` | Rotas HTTP, escopo por usuário, filtros, paginação e métricas. |
| `internal/platform/messaging` | Publisher com confirms/returns e dispatcher do outbox. |
| `internal/platform/app` | Pool, migração, topologia, dispatcher, servidor HTTP e shutdown. |
| `internal/platform/config` | Leitura das variáveis de ambiente. |

## Endpoints

Todas as rotas de tarefas exigem `Authorization: Bearer <jwt>` válido. O `user_id`
vem das claims; o cliente nunca o informa.

| Método | Rota | Sucesso | Erros |
|---|---|---|---|
| `GET` | `/tasks?limit=&offset=&status=&priority=&overdue=&sort=&order=` | `200` com lista (vazia como `[]`) | `400` paginação ou filtro inválido, `401`, `500` |
| `POST` | `/tasks` | `201` com a tarefa | `400` título/status/prioridade/prazo inválido, `401`, `413` payload > 1 MiB, `500` |
| `GET` | `/tasks/{id}` | `200` com a tarefa | `400` id inválido, `401`, `404` quando não pertence ao usuário, `500` |
| `PUT` | `/tasks/{id}` | `200` com a tarefa atualizada | `400`, `401`, `404`, `413`, `500` |
| `DELETE` | `/tasks/{id}` | `204` sem corpo | `400`, `401`, `404`, `500` |
| `PATCH` | `/tasks/{id}/move` | `200` com a tarefa movida (`status` e `position`) | `400` status/vizinhos inválidos, `401`, `404`, `409` quadro desatualizado, `413`, `500` |
| `GET` | `/tasks/{id}/history?limit=&offset=` | `200` com as alterações, mais recentes primeiro | `400` id/paginação inválidos, `401`, `404`, `500` |

`PATCH /tasks/{id}/move` recebe `{"status":"doing","before_id":12,"after_id":null}`: `before_id` é a tarefa
imediatamente acima e `after_id` a imediatamente abaixo na coluna de destino. O servidor calcula `position` no
ponto médio entre os vizinhos (ou ±1024 nas pontas; sem vizinhos, vai para o topo) e renumera a coluna quando o
intervalo se esgota. Vizinho inexistente, de outra coluna ou fora de ordem devolve `409`. `GET /tasks` ordena por
`position ASC, id DESC`; tarefas novas e mudanças de status via `PUT` entram no topo da coluna.

| Método | Rota | Sucesso | Erros |
|---|---|---|---|
| `GET` | `/health` | `200` com `status: ok` | `503` quando PostgreSQL ou RabbitMQ falham |
| `GET` | `/metrics` | `200` no formato Prometheus | — |

Paginação: `limit` padrão `50`, máximo `100`; `offset` padrão `0`, mínimo `0`.

Payload de criação/atualização:

```json
{ "title": "Comprar café", "description": "Opcional", "status": "todo", "priority": "high", "due_date": "2026-11-15" }
```

- `status` aceita `todo`, `doing` e `done`; vazio assume `todo`.
- `priority` aceita `low`, `medium`, `high` e `urgent`; vazio assume `medium`.
- `due_date` é opcional, no formato `YYYY-MM-DD`, ou `null` para remover o prazo. Outro formato devolve `400`.
  - Na criação, um prazo no passado devolve `400`.
  - Na edição, só devolve `400` se o prazo mudou. Assim dá para editar uma tarefa já atrasada sem mexer no prazo.
  - O piso é o dia anterior em UTC, para não rejeitar "hoje" de clientes atrás de UTC.
- `PUT` substitui a tarefa inteira: omitir `priority` ou `due_date` volta a prioridade para `medium` e apaga o prazo.

Filtros e ordenação de `GET /tasks`:

| Parâmetro | Valores | Padrão |
|---|---|---|
| `status` | `todo`, `doing`, `done` | todos |
| `priority` | `low`, `medium`, `high`, `urgent` | todas |
| `overdue` | `true`/`false`; `true` traz `due_date < CURRENT_DATE` e status diferente de `done` | `false` |
| `sort` | `position`, `priority`, `created_at`, `due_date` | `position` |
| `order` | `asc`, `desc` | `asc` para `position`/`due_date`; `desc` para `priority`/`created_at` |

Ordenar por `due_date` deixa as tarefas sem prazo no fim (`NULLS LAST`). Valor desconhecido devolve
`400 invalid filter`.

Resposta de `GET /tasks/{id}/history`:

```json
[
  { "id": 3, "task_id": 7, "action": "updated", "changed_at": "2026-10-30T14:02:00Z",
    "changes": [{ "field": "priority", "from": "medium", "to": "high" }, { "field": "description", "from": null, "to": null }] },
  { "id": 2, "task_id": 7, "action": "moved", "changed_at": "2026-10-29T09:10:00Z",
    "changes": [{ "field": "status", "from": "todo", "to": "doing" }] }
]
```

## Fluxo exclusivo do serviço

### CRUD

1. `auth.Middleware` valida assinatura, emissor, audiência e expiração do JWT e
   injeta as claims no contexto.
2. O handler lê `user_id` das claims e aplica o escopo em todas as queries
   (`WHERE user_id = $n`), impedindo acesso a tarefas de outro usuário.
3. `TaskService` normaliza o título e o status antes de tocar no banco.
4. `PostgresTaskRepository` executa o SQL e traduz `sql.ErrNoRows` em
   `ErrNotFound`; o handler responde `404` sem distinguir tarefa inexistente de
   tarefa de outro usuário.

### Histórico de alterações

Cada escrita grava em `task_history`, na mesma transação, uma entrada com `action` e a lista `changes`
(`field`, `from`, `to`):

- `created`: status, prioridade e prazo iniciais.
- `updated`: o `PUT` lê a linha com `SELECT ... FOR UPDATE`, aplica o `UPDATE` e grava apenas os campos que mudaram,
  calculados por `domain.Diff`. Um `PUT` sem mudanças não gera entrada.
- `moved`: o `PATCH /move` grava só quando a coluna muda; reordenar na mesma coluna não gera entrada.

A descrição registra apenas `{"field":"description"}`, sem os valores, para não duplicar textos de até 1 MiB. O
`correlation_id` da requisição é gravado para diagnóstico, mas não sai na API.

### Criação com outbox transacional

1. Abre uma transação.
2. Insere em `tasks` e obtém o registro criado.
3. Grava a entrada `created` em `task_history`.
4. Gera um `event_id` UUID v4 e o payload `{"task_id": ..., "title": ...}`.
5. Insere em `task_outbox` o tipo `task.created`, o payload e o
   `correlation_id` vindo do contexto HTTP.
6. Faz commit. Tarefa, histórico e evento são atômicos: se qualquer etapa falhar, nada é
   persistido.

### Dispatcher do outbox

Uma goroutine roda a cada `500 ms` até o contexto ser cancelado:

1. `ClaimOutbox` seleciona um evento pendente com
   `FOR UPDATE SKIP LOCKED LIMIT 1`, incrementa `attempts` e aplica um lease de
   `30 s` (`available_at = NOW() + 30 seconds`). Sem evento pendente, o ciclo é
   vazio.
2. O publisher declara a topologia de forma idempotente, habilita publisher
   confirms e escuta returns.
3. Publica com `mandatory=true`, mensagem persistente, `MessageId = event_id`,
   `Type = routing key` e headers de correlação/tracing.
4. Aguarda confirmação:
   - `nack` ou retorno `unroutable` contam como falha;
   - após um `ack`, aguarda até `10 ms` por um return, porque o `amqp091-go` não
     garante ordem entre os canais de return e confirmação.
5. Em falha, agenda retry com backoff exponencial
   `1s << min(attempts-1, 6)` (1, 2, 4, 8, 16, 32 e 64 s) e incrementa
   `retry`; erro ao agendar vira `retry_error`.
6. Em sucesso, marca `sent_at` e incrementa `sent`.

A entrega é **at-least-once**: o consumidor deve deduplicar por `MessageId`
(`event_id`).

### Topologia AMQP

Declarada no startup pelo publisher e novamente pelo consumidor, de forma
idempotente:

- exchange `tasks` (topic);
- fila `task.events` com `x-dead-letter-exchange: tasks.dlx`;
- binding `task.created`;
- DLX `tasks.dlx` (fanout) e DLQ `task.events.dlq`.

## Modelo de dados

```sql
CREATE TABLE tasks (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'todo',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  position DOUBLE PRECISION NOT NULL DEFAULT 0,
  priority TEXT NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
  due_date DATE
);

CREATE TABLE task_history (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL,
  action TEXT NOT NULL CHECK (action IN ('created', 'updated', 'moved')),
  changes JSONB NOT NULL DEFAULT '[]',
  correlation_id TEXT NOT NULL DEFAULT '',
  changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE task_outbox (
  id BIGSERIAL PRIMARY KEY,
  event_id UUID NOT NULL UNIQUE,
  task_id BIGINT NOT NULL,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  correlation_id TEXT NOT NULL DEFAULT '',
  attempts INTEGER NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  sent_at TIMESTAMPTZ
);
```

Índices:

- `tasks`: `(user_id, id DESC)`, `(user_id, status, position)`, `(user_id, priority)` e `(user_id, due_date)`;
- `task_history`: `(task_id, changed_at DESC, id DESC)`;
- `task_outbox`: `(available_at, id) WHERE sent_at IS NULL`.

Em bancos existentes, a migração:

- adiciona `position` preenchendo `-id * 1024`, o que preserva a ordem anterior (mais novas primeiro);
- adiciona `priority` (`medium`) e `due_date` (nulo);
- cria uma entrada `created` em `task_history` para cada tarefa que ainda não tem histórico.

## Configuração

| Variável | Obrigatória | Padrão | Uso |
|---|---|---|---|
| `PORT` | não | `8080` | Porta HTTP interna. |
| `DATABASE_URL` | não | `postgres://task:task@localhost:5432/task?sslmode=disable` | DSN do PostgreSQL. |
| `RABBITMQ_URL` | não | `amqp://guest:guest@localhost:5672/` | Conexão AMQP. |
| `JWT_SECRET` | sim | — | Validação HS256; o processo não inicia sem ela. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | não | vazio | Endpoint OTLP/gRPC; vazio desabilita traces. |

Pool: 25 conexões abertas, 10 ociosas, vida máxima de 30 min e ociosidade máxima de
5 min.

## Observabilidade

- Métrica de negócio: `task_manager_task_operations_total{operation,outcome}`; `operation` inclui `history`, e `list`
  registra `invalid_filter`.
- Métrica do outbox: `task_manager_task_outbox_events_total{outcome}`, com
  `sent`, `retry`, `retry_error`, `claim_error` e `mark_error`.
- Métricas HTTP RED em `/metrics`.
- Logs `slog` com `correlation_id`; falhas de dispatch registram `event_id`.
- Spans para `outbox dispatch` e `rabbitmq publish`, com propagação W3C nos headers
  AMQP.

## Execução e testes

```sh
# a partir da raiz do repositório
docker compose up --build service-task

# testes do módulo
go test -race ./...
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
```

O `Dockerfile` usa contexto de build na raiz, copia `shared/` e roda como usuário
não-root.
