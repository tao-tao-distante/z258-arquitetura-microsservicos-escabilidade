# 0012 - Métricas Prometheus e dashboards Grafana provisionados

## Status

Aceito. O ADR-0016 define os SLIs, SLOs, janelas, populações e denominadores que usam estas fontes.

## Data

2026-10-29

## Contexto e Problema

A stack já recebia traces pelo OpenTelemetry Collector, mas não permitia responder quantos cadastros, logins, operações
de tarefa, publicações do outbox e eventos de notificação ocorriam. Também faltava visibilidade sobre filas, consumidores
e alarmes do RabbitMQ. Dashboards criados manualmente não seriam reproduzíveis após remover o volume do Grafana.

## Fatores de Decisão

- Métricas devem cobrir sinais RED HTTP, runtime Go e resultados de negócio.
- Labels devem ter conjuntos limitados; IDs, e-mails, correlation IDs, URLs brutas e mensagens de erro não podem ser labels.
- A fila principal e a DLQ precisam aparecer separadamente.
- Dashboards devem ser versionados e provisionados automaticamente.
- A solução local deve continuar sem publicar Prometheus ou a porta de métricas do RabbitMQ no host.

## Resultado da Decisão

Cada serviço expõe `GET /metrics` com `prometheus/client_golang`. O registro é isolado por processo e contém métricas Go,
processo, taxa/status/latência HTTP e contadores de negócio:

- `task_manager_identity_auth_attempts_total{operation,outcome}`;
- `task_manager_task_operations_total{operation,outcome}`;
- `task_manager_task_outbox_events_total{outcome}`;
- `task_manager_notification_events_total{outcome}`.

Prometheus coleta os três serviços diretamente pela rede `telemetry`. Traces continuam usando OTLP e o Collector; a
pipeline de métricas OTLP permanece disponível para futura instrumentação.

O plugin oficial `rabbitmq_prometheus` expõe `/metrics` em `rabbitmq:15692`. Métricas por objeto são habilitadas para
distinguir `task.events` e `task.events.dlq`; esta escolha é aceitável apenas porque a stack local possui poucas filas.
Prometheus participa também da rede interna `messaging`, sem publicação adicional de portas.

Grafana provisiona quatro dashboards JSON imutáveis no diretório `Task Manager`:

- `Task Manager - Observabilidade Unificada`;
- `Task Manager - Visão de Negócio`;
- `Task Manager - Backends Go`;
- `Task Manager - RabbitMQ`.

## Alternativas Consideradas

### Exportar todas as métricas por OTLP

Manteria um único caminho de telemetria, mas adicionaria exporter e reader periódicos aos serviços sem necessidade para o
scrape local. Rejeitada neste estágio; o Collector continua responsável por traces e disponível para métricas futuras.

### Consultar a HTTP API de Management do RabbitMQ

Exigiria credenciais no coletor e polling com maior custo. Rejeitada em favor do plugin Prometheus oficial.

### Criar dashboards somente pela interface do Grafana

Seria rápido, mas perderia reprodutibilidade e revisão por Git. Rejeitada em favor de provisioning por arquivos.

## Consequências

- Bom: métricas de negócio permitem acompanhar o fluxo de autenticação até o consumo da notificação.
- Bom: métricas HTTP e de runtime permitem correlacionar falhas de negócio com latência, erros e saturação.
- Bom: RabbitMQ, fila principal e DLQ ficam visíveis sem expor `:15692` no host.
- Bom: dashboards reaparecem após `docker compose down -v`.
- Ruim: scrape direto e OTLP usam caminhos distintos; operação deve conhecer ambos.
- Ruim: métricas por objeto do RabbitMQ não escalam para milhares de filas; produção deve reavaliar agregação e cardinalidade.
- Ruim: não há Alertmanager nem alertas de burn rate; os dashboards são diagnóstico, não garantia de nível de serviço.

## Confirmação

Subir a stack e confirmar `up == 1` para jobs `service-identity`, `service-task`, `service-notification` e `rabbitmq`.
Executar cadastro, login e criação de tarefa; verificar incremento dos quatro contadores de negócio e das séries HTTP.
No Grafana, abrir os quatro dashboards provisionados, pesquisar um `trace_id` no painel unificado e confirmar
`task.events` e `task.events.dlq` no dashboard RabbitMQ.
