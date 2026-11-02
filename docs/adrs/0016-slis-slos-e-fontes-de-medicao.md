# 0016 - SLIs, SLOs e fontes de medição

## Status

Aceito.

## Data

2026-11-02

## Contexto e problema

As metas anteriores misturavam medições feitas na borda com métricas dos handlers Go. Também calculavam a entrega do
outbox somente sobre eventos enviados, o que retirava eventos pendentes do denominador. Uma meta verificável precisa
fixar população, janela, numerador, denominador e fonte.

## Decisão

O projeto mantém dois tipos de janela:

- a janela de aceitação é uma execução completa do k6, com `DURATION=2m` por padrão;
- a janela operacional é móvel de 7 dias, limitada pela retenção atual do Prometheus e do Loki.

Uma execução do k6 comprova somente o cenário, as taxas e a duração registrados no resumo daquela execução. Ela não comprova um SLO operacional de 7 dias.

### SLIs HTTP na borda

O k6 chama `http://localhost:8081`, portanto mede navegador excluído, NGINX incluído e resposta completa recebida pelo cliente de teste.

| SLI                  | População e cálculo                                                                                                                                                | SLO na janela de aceitação  | Fonte                                        |
|----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------|----------------------------------------------|
| Latência de listagem | Todas as respostas concluídas de `GET /api/tasks/tasks` com a tag `endpoint:list_tasks`. O percentil usa `http_req_duration`, inclusive respostas HTTP com erro.   | p95 < 200 ms e p99 < 500 ms | `loadtest/tasks.js` e `results-summary.json` |
| Latência de criação  | Todas as respostas concluídas de `POST /api/tasks/tasks` com a tag `endpoint:create_task`. O percentil usa `http_req_duration`, inclusive respostas HTTP com erro. | p95 < 300 ms                | `loadtest/tasks.js` e `results-summary.json` |
| Falhas de listagem   | Numerador: requisições do cenário `list_tasks` que o k6 marca como falha HTTP ou de rede. Denominador: todas as requisições do mesmo cenário.                      | taxa < 1%                   | `http_req_failed{scenario:list_tasks}`       |
| Falhas de criação    | Numerador: requisições do cenário `create_tasks` que o k6 marca como falha HTTP ou de rede. Denominador: todas as requisições do mesmo cenário.                    | taxa < 1%                   | `http_req_failed{scenario:create_tasks}`     |

Os checks de status `200` e `201` aparecem no relatório, mas não alteram `http_req_failed`. Sem `setResponseCallback`, o
k6 considera esperados os status de `200` a `399`. Por isso, o threshold atual falha para erros de rede e status fora
dessa faixa, mas aceita um `3xx` mesmo quando o check exato falha. Esse limite permanece explícito até o teste definir os
status esperados por endpoint.

### SLIs HTTP nos handlers

`task_manager_http_request_duration_seconds` e `task_manager_http_requests_total` começam e terminam dentro de cada
serviço Go. Essas séries excluem DNS, conexão do cliente, fila e processamento no NGINX. Elas servem para diagnóstico
dos handlers e não substituem os SLIs de borda.

Na janela móvel de 7 dias:

- taxa de erro do handler: numerador de respostas `5xx`; denominador de todas as respostas do mesmo serviço, método e padrão de rota;
- latência do handler: percentis calculados sobre todos os buckets do mesmo serviço, método e padrão de rota, separados por `status` quando a investigação exigir;
- disponibilidade de scrape: amostras com `up == 1` divididas por todas as amostras esperadas do alvo.

O objetivo de taxa de erro do handler é menor que 1%. O objetivo de disponibilidade de scrape é 99,5%. `up` mede a
capacidade do Prometheus de coletar o alvo. Não mede disponibilidade do caminho do usuário pela borda. O projeto ainda
não possui probe sintético contínuo no NGINX, então não há SLI operacional de disponibilidade ponta a ponta.

### SLI do outbox

O SLO de publicação usa uma coorte por data de criação e inclui pendências vencidas:

- janela: eventos com `created_at` nos últimos 7 dias;
- população elegível: eventos criados até 5 segundos antes do fim da consulta;
- denominador: todos os eventos elegíveis, enviados ou pendentes;
- numerador: eventos elegíveis com `sent_at IS NOT NULL` e `sent_at - created_at <= 5 seconds`;
- objetivo: pelo menos 99,9% no numerador.

Eventos pendentes com mais de 5 segundos permanecem no denominador e contam como falha. Eventos com menos de 5 segundos
ainda não venceram o prazo e ficam fora da população elegível. `task_manager_task_outbox_events_total` mostra resultados
do dispatcher, mas não contém idade nem quantidade de pendentes. A fonte normativa deste SLI é o PostgreSQL:

```sql
WITH eligible AS (
  SELECT sent_at, created_at
  FROM task_outbox
  WHERE event_type = 'task.created'
    AND created_at >= NOW() - INTERVAL '7 days'
    AND created_at <= NOW() - INTERVAL '5 seconds'
)
SELECT count(*) AS total_eligible,
       count(*) FILTER (
         WHERE sent_at IS NOT NULL
           AND sent_at - created_at <= INTERVAL '5 seconds'
       ) AS within_slo,
       count(*) FILTER (WHERE sent_at IS NULL) AS pending_overdue,
       coalesce(avg((
         sent_at IS NOT NULL
         AND sent_at - created_at <= INTERVAL '5 seconds'
       )::int), 1) AS ratio_within_slo
FROM eligible;
```

Quando `total_eligible` é zero, a consulta não fornece evidência de conformidade. O valor `1` evita divisão indefinida
no resultado, mas deve ser lido junto com a contagem.

### Contexto persistido

`task_outbox.trace_context` guarda o carrier W3C como JSONB. Ele preserva `traceparent` e `tracestate` entre a requisição
e o dispatcher. Esse campo não participa do cálculo do SLI de publicação.

## Consequências

- As metas de borda e as métricas de handler deixam de representar a mesma população.
- Pendências vencidas reduzem o resultado do SLO do outbox.
- A retenção atual limita a janela operacional a 7 dias.
- O projeto precisa de probe sintético contínuo para definir disponibilidade ponta a ponta na janela operacional.
- Alertas de burn rate continuam fora do escopo enquanto não houver Alertmanager.

## Confirmação

Executar o k6 com os parâmetros registrados no resumo. Consultar as séries Prometheus por serviço, método, rota e status.
Executar a consulta SQL do outbox durante e depois da drenagem. Confirmar que uma linha pendente com mais de 5 segundos
aumenta `pending_overdue` e permanece em `total_eligible`.
