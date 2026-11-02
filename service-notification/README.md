# service-notification

Consumidor assíncrono de eventos `task.created`. Não expõe API de negócio, não tem
banco e não valida JWT: sua única entrada de trabalho é a fila RabbitMQ.

- Porta interna: `8080`, acessível pelo host via NGINX em `/api/notifications`, usada para
  `/health` e `/metrics`.
- Dependência: RabbitMQ.
- Fila: `task.events`, routing key `task.created`.

## Estrutura

| Caminho | Responsabilidade |
|---|---|
| `cmd/server` | Bootstrap, sinais e adaptação do cliente AMQP. |
| `internal/domain` | Estrutura `TaskCreatedEvent` (`task_id`, `title`). |
| `internal/handler` | Consumidor AMQP, extração de correlação/tracing, Ack/Nack, DLQ e health. |
| `internal/platform/app` | Conexão, consumidor, servidor HTTP de saúde/métricas e shutdown. |
| `internal/platform/config` | Leitura das variáveis de ambiente. |

## Endpoints

| Método | Rota | Sucesso | Erros |
|---|---|---|---|
| `GET` | `/health` | `200` com `status: ok` | `503` com `status: degraded` |
| `GET` | `/metrics` | `200` no formato Prometheus | — |

O health verifica três condições do RabbitMQ: conexão aberta, canal aberto e
consumidor ativo. Qualquer uma falsa resulta em `503`.

## Fluxo exclusivo do serviço

### Inicialização

1. Lê a configuração e conecta ao RabbitMQ com retry de 1 s até o contexto ser
   cancelado.
2. Abre um canal e declara a topologia de forma idempotente (`tasks`,
   `task.events`, `tasks.dlx`, `task.events.dlq`).
3. Aplica `Qos(prefetch=10)` e registra o consumidor `service-notification` na
   fila `task.events` com `autoAck=false`.
4. Sobe o servidor HTTP de saúde/métricas e aguarda `ctx.Done()`, falha do
   consumidor ou falha do servidor.
5. No encerramento, cancela o consumo, faz shutdown do HTTP e drena as duas
   goroutines, combinando os erros.

### Consumo de uma entrega

1. Extrai o contexto de tracing dos headers AMQP (W3C).
2. Lê `x-correlation-id` dos headers. Se não for UUID v4 válido, gera um novo.
   O ID entra no `context.Context` e em todos os logs.
3. Decodifica o corpo como `TaskCreatedEvent`.
4. Valida o evento: `task_id > 0` e `title` não vazio após `trim`.
5. **Evento válido**: registra o log `notification_received` com `task_id`,
   executa `Ack(false)` e incrementa `acked`.
6. **Evento inválido** (JSON malformado ou campos ausentes): registra
   `invalid event` com o motivo, executa `Nack(false, false)` — sem requeue — e
   incrementa `rejected`. A mensagem segue para `tasks.dlx` e depois para
   `task.events.dlq`.
7. Falha em `Ack` ou `Nack` retorna erro e encerra o consumo; o processo principal
   falha e o container é reiniciado pela política de restart.

A entrega é **at-least-once**. O serviço atual apenas registra o evento; se no
futuro passar a produzir efeitos externos, deve deduplicar por `MessageId`
(`event_id`) ou por `task_id` + tipo de evento.

### Por que DLQ

`Nack(false, false)` descarta a mensagem da fila principal sem reenfileirar. Como
`task.events` declara `x-dead-letter-exchange: tasks.dlx`, o RabbitMQ encaminha a
mensagem para a DLQ. Isso evita loop infinito de eventos inválidos e preserva o
conteúdo para inspeção.

## Configuração

| Variável | Obrigatória | Padrão | Uso |
|---|---|---|---|
| `PORT` | não | `8080` | Porta HTTP de saúde/métricas. |
| `RABBITMQ_URL` | não | `amqp://guest:guest@localhost:5672/` | Conexão AMQP. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | não | vazio | Endpoint OTLP/gRPC; vazio desabilita traces. |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | não | vazio | Alternativa usada quando a variável acima está vazia. |

Não há `DATABASE_URL` nem `JWT_SECRET`: o serviço não persiste dados e não
autentica usuários.

## Observabilidade

- Log `notification_received` com `task_id`, `service_name`, `correlation_id`, `trace_id` e `span_id`.
- Log `invalid event` com o motivo da rejeição.
- Métrica de negócio: `task_manager_notification_events_total`, com label
  `outcome` (`acked`, `rejected`, `ack_error` ou `nack_error`).
- Métricas HTTP RED em `/metrics` para as rotas de saúde.
- Span de consumo `task.created consume` com `SpanKindConsumer`, continuando o
  trace publicado pelo `service-task`.

## Execução e testes

```sh
# a partir da raiz do repositório
docker compose up --build service-notification
docker compose logs -f service-notification

# testes do módulo
go test -race ./...
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
```

O `Dockerfile` usa contexto de build na raiz, copia `shared/` e roda como usuário
não-root.
