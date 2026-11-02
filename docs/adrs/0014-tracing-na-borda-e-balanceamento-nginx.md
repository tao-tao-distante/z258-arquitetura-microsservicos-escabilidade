# 0014 - Tracing na borda e balanceamento NGINX

## Status

Aceito.

## Data

2026-10-30

## Contexto e Problema

O navegador criava spans `CLIENT`, `traceparent` e `x-correlation-id`. Isso transferia ao cliente a responsabilidade pela
origem do trace e aumentava o bundle. O consumo de `task.created` também precisa permanecer na mesma árvore após o limite
assíncrono do RabbitMQ.

## Fatores de Decisão

- O primeiro componente confiável deve criar o trace.
- A borda deve encaminhar tráfego para réplicas sem alterar URLs do frontend.
- O outbox deve preservar o contexto depois que a requisição HTTP terminar.
- O consumidor deve usar o span de publicação como pai.

## Resultado da Decisão

Instalar `ngx_otel_module` no container NGINX. Para rotas `/api/*`, `otel_trace on` cria o span de borda e
`otel_trace_context inject` substitui qualquer contexto recebido por W3C Trace Context confiável. Grupos `upstream` usam
`least_conn`, `zone`, DNS do Docker e `resolve` para acompanhar réplicas.

Angular não cria spans nem IDs. Na ausência de `x-correlation-id`, o middleware Go gera UUID v4 e o devolve na resposta.
Esse identificador continua separado do trace e segue nos logs, outbox e headers AMQP.

`service-task` persiste `traceparent`/`tracestate` no outbox. O dispatcher extrai esse carrier, cria span `INTERNAL`, o
publisher cria span `PRODUCER` e injeta o novo contexto nos headers AMQP. `service-notification` extrai os headers e cria
span `CONSUMER`, preservando a mesma `TraceID` e usando o `SpanID` do publisher como pai.

## Consequências

- Bom: trace nasce na borda confiável e inclui NGINX, API, outbox, publisher e consumer.
- Bom: bundle Angular perde dependências OpenTelemetry e exportação OTLP/HTTP.
- Bom: NGINX suporta balanceamento entre endereços retornados pelo DNS do Compose.
- Ruim: imagem NGINX instala `nginx-module-otel` e suas dependências gRPC, aumentando o tamanho.
- Neutro: `x-correlation-id` continua independente da `TraceID`.

## Confirmação

Executar `docker compose --env-file .env.example build frontend` e `nginx -t` no container. Criar tarefa pela porta
`:8081` e confirmar no Tempo uma árvore `frontend-edge` -> `service-task` -> `outbox dispatch` -> `rabbitmq publish` ->
`task.created consume`, com uma única `TraceID`.
