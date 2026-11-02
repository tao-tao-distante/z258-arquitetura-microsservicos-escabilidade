# shared

Código Go comum aos três microsserviços. Não é um serviço executável: é um módulo
independente (`go 1.23`) consumido por `service-identity`, `service-task` e
`service-notification` por meio de `replace` no `go.mod` de cada serviço.

```go
require github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared v0.0.0

replace github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared => ../shared
```

## Pacotes

| Pacote        | Responsabilidade                                                                                                |
|---------------|-----------------------------------------------------------------------------------------------------------------|
| `auth`        | Hash bcrypt, emissão e validação de JWT HS256, extração do header `Authorization` e middleware de autenticação. |
| `broker`      | Nomes da topologia AMQP e declaração idempotente de exchange, fila, DLX e DLQ.                                  |
| `correlation` | Propagação do `x-correlation-id` via `context.Context` e geração de UUID v4.                                    |
| `httpx`       | Escrita de JSON/erro, decodificação estrita com limite de payload e handler de health.                          |
| `logger`      | Logger `log/slog` JSON em stdout e enriquecimento com `correlation_id`.                                         |
| `metrics`     | Registro Prometheus isolado por serviço, métricas HTTP RED e contadores de negócio.                             |
| `middleware`  | Middleware de correlação, log de requisições e rate limiter em memória.                                         |
| `telemetry`   | Provider OTLP/gRPC para traces, com fallback no-op quando o endpoint está vazio.                                |

## Contratos compartilhados

### Autenticação

- Algoritmo: HS256, emissor `service-identity`, audiência `service-task`.
- Claims obrigatórias: `user_id` (> 0) e `email` (não vazio), além de expiração.
- `auth.Middleware` aceita apenas `Authorization: Bearer <token>`, responde
  `401 {"error":"unauthorized"}` e injeta as claims no contexto.
- `auth.FromContext` devolve `*Claims` ou `nil`.

### Correlação

- Header HTTP e AMQP: `x-correlation-id`.
- `middleware.Correlation` aceita o valor apenas se for UUID v4 válido; caso contrário,
  reutiliza o ID já presente no contexto ou gera um novo.
- O ID é devolvido no header da resposta e usado por logs e spans.

### Topologia AMQP

Definida em `broker` e declarada de forma idempotente por publicador e consumidor:

| Elemento             | Nome                     |
|----------------------|--------------------------|
| Exchange             | `tasks` (topic, durável) |
| Fila                 | `task.events`            |
| Routing key          | `task.created`           |
| Dead letter exchange | `tasks.dlx` (fanout)     |
| Dead letter queue    | `task.events.dlq`        |

A fila `task.events` declara `x-dead-letter-exchange: tasks.dlx`.

### HTTP

- `httpx.DecodeJSON` limita o corpo, rejeita campos desconhecidos e exige exatamente
  um objeto JSON.
- `httpx.WriteError` sempre responde `{"error":"..."}`.
- `httpx.Health` executa todas as checagens e responde `200` com `status: ok` ou
  `503` com `status: degraded` e o estado por dependência.

### Métricas

- `metrics.New(service)` cria um registro isolado com métricas de runtime/processo.
- Métricas HTTP: `task_manager_http_requests_total` e
  `task_manager_http_request_duration_seconds`, com labels `service`, `method`,
  `route` (padrão da rota, nunca URL bruta) e `status`.
- Contadores de negócio são criados por `Set.Counter` e incrementados por
  `Counter.Inc(labels...)`. Os serviços usam apenas labels de baixa cardinalidade.

### Tracing

- `telemetry.Setup(ctx, endpoint, serviceName)` configura propagação
  TraceContext/Baggage e exportação OTLP/gRPC.
- Endpoint vazio desabilita a exportação sem falhar o serviço.
- Endpoint inválido retorna erro de configuração.

## Testes

```sh
go test -race ./...
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
```

## Regras de evolução

- `shared` não pode importar nenhum módulo de serviço.
- Mudanças em contratos (claims, headers, nomes AMQP, labels de métricas) exigem
  atualização conjunta dos serviços e dos dashboards.
- O módulo declara `go 1.23`; novas dependências precisam ser compatíveis com essa
  versão.
