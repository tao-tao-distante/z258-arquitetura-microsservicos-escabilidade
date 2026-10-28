# 0002 - Rastreamento Distribuído com Correlation ID, log/slog e OpenTelemetry Go

## Context and Problem Statement

Em uma arquitetura distribuída com chamadas síncronas e assíncronas (RabbitMQ), torna-se extremamente difícil depurar
erros e analisar a latência entre a interface Angular e os microsserviços em Go sem rastreamento ponta a ponta.

## Decision Drivers

* Requisito obrigatório de projeto: rastreabilidade e logs estruturados em JSON contendo Correlation ID.
* Diagnóstico rápido de gargalos de performance e validação dos SLOs ($p_{95} \le 150\text{ms}$).
* Uso das bibliotecas nativas de Go (`log/slog` e `go.opentelemetry.io/otel`).

## Considered Options

* **Opção 1:** OpenTelemetry SDK com propagação do header W3C `x-correlation-id` e logs estruturados em JSON via
  `log/slog`.
* **Opção 2:** Logs puramente locais via `fmt.Println` sem identificador de correlação.

## Decision Outcome

Chosen option: "**Opção 1: OpenTelemetry SDK com propagação de x-correlation-id e log/slog**", pois atende aos
requisitos de observabilidade distribuída e integra-se diretamente ao Prometheus/Tempo/Grafana.

### Consequences

* Good, porque permite rastrear uma requisição individual desde o clique na interface Angular até o processamento em Go
  e a fila do RabbitMQ.
* Good, porque padroniza todos os logs no formato JSON com a chave `"correlation_id"`.
* Bad, porque exige repassar o `context.Context` em todas as chamadas de funções e requisições HTTP em Go.

### Confirmation

Será confirmado emitindo uma requisição HTTP contendo o cabeçalho `x-correlation-id: test-123` e validando a presença
desse mesmo ID nos logs JSON gerados no `stdout` dos microsserviços.

## Pros and Cons of the Options

### Opção 1: OpenTelemetry + Correlation ID + log/slog

* Good, porque oferece visibilidade fim-a-fim da jornada da requisição.
* Good, porque simplifica a busca por falhas em ferramentas centrais como Grafana/Loki/Tempo.
* Bad, porque exige a criação de HTTP Middlewares em Go e interceptores no Angular.

### Opção 2: Logs Locais Não Estruturados

* Good, porque é trivial de implementar sem dependências externas.
* Bad, porque torna praticamente impossível correlacionar eventos em requisições concorrentes.
* Bad, porque viola os requisitos arquiteturais estipulados para a avaliação.