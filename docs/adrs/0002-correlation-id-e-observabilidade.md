# 0002 - Correlação de aplicação e base para OpenTelemetry

## Status

Substituído parcialmente pelo ADR-0014 para a origem do trace e pelo ADR-0015 para logs contextuais e centralização no
Loki. O `x-correlation-id` continua sendo gerado pelos serviços Go, não pelo Angular.

## Data

2026-10-28

## Nota de atualização

O resultado abaixo registra o estado original. No estado atual, Angular não cria spans nem IDs. O NGINX cria o trace na
borda e injeta W3C Trace Context. Os serviços aceitam `x-correlation-id` somente quando o valor é UUID v4; caso contrário,
geram outro. `task_outbox.trace_context` persiste o carrier W3C em JSONB. ADR-0014 e ADR-0015 prevalecem onde houver
conflito.

## Contexto e Problema

Requisições HTTP e eventos AMQP precisam de um identificador pesquisável nos logs. Rastreamento distribuído também é
desejável, mas correlação de logs e W3C Trace Context são mecanismos diferentes.

## Fatores de Decisão

- Logs JSON devem conter `correlation_id`.
- O identificador deve atravessar frontend, APIs e RabbitMQ.
- A infraestrutura deve aceitar métricas e traces OTLP quando as aplicações forem instrumentadas.

## Resultado da Decisão

Usar `x-correlation-id` como identificador de aplicação e `log/slog` para logs estruturados. O frontend gera UUID v4 e
inicia um span OpenTelemetry `CLIENT` para cada requisição API; middlewares Go aceitam apenas UUIDs v4 válidos e geram um
novo quando o header está ausente ou inválido. `service-task` e `service-notification` propagam o mesmo valor em header
AMQP.

`x-correlation-id` não é header W3C. O SDK OpenTelemetry propaga `traceparent`/`tracestate` em HTTP e nos headers AMQP.
Os serviços criam spans no servidor HTTP (`otelhttp`), na publicação/consumo RabbitMQ e no dispatch do outbox. O
`service-task` persiste o carrier W3C no outbox junto do evento e o restaura no dispatcher, preservando a mesma árvore de
spans mesmo após o término da requisição HTTP. Os logs permanecem indexados por `correlation_id`; `trace_id`/`span_id`
ainda não são adicionados aos logs.

Collector, Tempo, Prometheus e Grafana são provisionados. O frontend exporta OTLP/HTTP pelo proxy same-origin
`/otlp/v1/traces`; os serviços Go exportam OTLP/gRPC quando `OTEL_EXPORTER_OTLP_ENDPOINT` está definido (o Compose aponta
para `http://otel-collector:4317`). Os serviços também expõem métricas HTTP, runtime e negócio em `/metrics`, coletadas
diretamente pelo Prometheus conforme ADR-0012.

## Consequências

- Bom: uma requisição e seu evento podem ser localizados nos logs por `correlation_id`.
- Bom: spans do frontend, HTTP, outbox e AMQP formam uma única árvore no Tempo.
- Ruim: `correlation_id` e `trace_id` são independentes; não há link direto entre log e trace.
- Bom: métricas de aplicação e RabbitMQ alimentam dashboards versionados conforme ADR-0012.
- Ruim: logs ficam apenas no `stdout`; Loki não faz parte da solução.

## Emenda (ADR-0015)

As afirmações de que logs são indexados apenas por `correlation_id` e de que Loki não faz parte da solução descrevem o
estado original. Desde o ADR-0015, os logs JSON incluem `service_name`, `correlation_id`, `trace_id` e `span_id`; o Alloy
coleta o `stdout`/`stderr` e o Loki os armazena com retenção de 7 dias. `correlation_id` continua sendo o identificador
de aplicação e não substitui o W3C Trace Context.

## Confirmação

Enviar `x-correlation-id: 646a4ca0-94d8-4f4a-b0f0-31fb0ac7b31d`, verificar o mesmo header na resposta e buscar o
valor nos logs do serviço e do consumidor. Enviar valor inválido e confirmar que a resposta traz um UUID v4 novo, usado
também no evento AMQP. Subir a stack com `OTEL_EXPORTER_OTLP_ENDPOINT` definido e confirmar no Tempo/Grafana os spans
de `service-identity`, `service-task` e `service-notification`, incluindo o span de consumo AMQP. Confirmar também os
quatro jobs de scrape e os dashboards descritos no ADR-0012.
