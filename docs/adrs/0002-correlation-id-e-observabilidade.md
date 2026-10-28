# 0002 - Correlação de aplicação e base para OpenTelemetry

## Status

Aceito, com instrumentação OpenTelemetry de traces implementada.

## Context and Problem Statement

Requisições HTTP e eventos AMQP precisam de um identificador pesquisável nos logs. Rastreamento distribuído também é
desejável, mas correlação de logs e W3C Trace Context são mecanismos diferentes.

## Decision Drivers

- Logs JSON devem conter `correlation_id`.
- O identificador deve atravessar frontend, APIs e RabbitMQ.
- A infraestrutura deve aceitar métricas e traces OTLP quando as aplicações forem instrumentadas.

## Decision Outcome

Usar `x-correlation-id` como identificador de aplicação e `log/slog` para logs estruturados. O frontend gera UUID v4;
middlewares Go capturam ou geram o valor; `service-task` e `service-notification` o propagam em header AMQP.

`x-correlation-id` não é header W3C. O SDK OpenTelemetry propaga `traceparent`/`tracestate` em HTTP e nos headers AMQP.
Os serviços criam spans no servidor HTTP (`otelhttp`), na publicação/consumo RabbitMQ e no dispatch do outbox. Os logs
permanecem indexados por `correlation_id`; `trace_id`/`span_id` ainda não são adicionados aos logs.

Collector, Tempo, Prometheus e Grafana são provisionados. Os serviços exportam traces OTLP quando
`OTEL_EXPORTER_OTLP_ENDPOINT` está definido (o Compose aponta para `http://otel-collector:4317`). Métricas de runtime
e de negócio ainda não são exportadas pelas aplicações; o Prometheus coleta apenas métricas internas do Collector.

## Consequences

- Good: uma requisição e seu evento podem ser localizados nos logs por `correlation_id`.
- Good: traces HTTP, AMQP e de dispatch do outbox são exportados para o Tempo e permitem visualizar a árvore de spans.
- Bad: `correlation_id` e `trace_id` são independentes; não há link direto entre log e trace.
- Bad: métricas de aplicação não são exportadas e logs ficam apenas no `stdout`; Loki não faz parte da solução.

## Confirmation

Enviar `x-correlation-id: test-123`, verificar o mesmo header na resposta e buscar `test-123` nos logs do serviço e do
consumidor. Subir a stack com `OTEL_EXPORTER_OTLP_ENDPOINT` definido e confirmar no Tempo/Grafana os spans de
`service-identity`, `service-task` e `service-notification`, incluindo o span de consumo AMQP.
