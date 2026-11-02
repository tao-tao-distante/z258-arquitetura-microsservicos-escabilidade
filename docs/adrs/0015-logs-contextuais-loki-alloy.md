# ADR-0015: Logs contextuais com Loki, Alloy e correlação por trace

## Status

Aceito

## Contexto

Os serviços já produzem logs JSON com `log/slog` e propagam W3C Trace Context de ponta a ponta, mas os logs ficam no
`stdout` dos containers. Isso impede busca centralizada, correlação entre trace e log no Grafana e investigação de
incidentes sem acesso ao host Docker. A stack também precisa continuar leve e executável em uma única máquina.

Além disso, logar corpos HTTP integrais vazaria credenciais (`password`, `token`), dados pessoais (`email`, `name`) e
conteúdo de negócio (`title`, `description`). Um pipeline de logs precisa capturar apenas metadados operacionais.

## Decisão

1. **Formato**: todos os serviços continuam emitindo JSON no `stdout` via `log/slog`. Cada logger carrega
   `service_name`; quando o contexto tem span válido, acrescenta `trace_id` e `span_id`; quando há correlação de
   aplicação, acrescenta `correlation_id`. Os identificadores são campos JSON puros, não prefixos de mensagem.
2. **Coleta**: o Alloy lê os arquivos `*-json.log` do diretório de containers do Docker com `local.file_match` +
   `loki.source.file`, desembrulha o JSON do driver com `stage.docker` e envia para o Loki via `loki.write`. A leitura
   por arquivo evita depender do socket Docker, que não é acessível em instalações rootless.
3. **Labels**: somente `service_name`, `level` e `container`. `trace_id`, `span_id`, `correlation_id`, `task_id`,
   `event_id` e `path` permanecem no corpo JSON e são consultados com `| json`. IDs e URLs nunca viram labels, para não
   explodir a cardinalidade do índice.
4. **Correlação no Grafana**: o datasource Loki extrai `"trace_id":"..."` em um campo derivado que abre o Tempo. O
   datasource Tempo consulta o Loki por `trace_id` no intervalo do span. A navegação funciona nos dois sentidos.
5. **Retenção**: Loki 3.6.7 em modo monolítico, schema TSDB v13, armazenamento `filesystem`, retenção de 7 dias e
   `auth_enabled: false` restrito à rede interna `telemetry`. Sem porta publicada no host.
6. **Bodies HTTP**: o middleware lê no máximo 64 KiB do corpo sem consumi-lo nem alterar o limite de payload do handler.
   Apenas requisições `application/json` (ou `+json`) com corpo completo e JSON válido geram `request_body`. Campos
   operacionais permitidos (`status`, `priority`, `due_date`, `before_id`, `after_id`) mantêm o valor escalar; todos os
   demais campos — inclusive desconhecidos — viram `[REDACTED]`. Corpo inválido, truncado ou de outro tipo gera apenas
   `request_body_bytes`, `request_body_truncated` e, quando aplicável, `request_body_valid_json`. Nenhum corpo de
   resposta é registrado.
7. **NGINX**: o access log é JSON com `service_name`, método, URI sem query, status, duração e identificadores; não
   registra `$request`, query string, headers de autenticação nem body. As respostas de `/api/*` incluem
   `X-Trace-ID: $otel_trace_id`. `/api/*/metrics` é bloqueado na borda.

## Consequências

- Logs de NGINX, Identity, Task, dispatcher do outbox e Notification compartilham a mesma `trace_id`, permitindo pular
  de um trace para todos os logs correspondentes no Grafana.
- Segredos, dados pessoais e conteúdo livre não aparecem nos logs HTTP, mesmo em payload inválido ou acima do limite.
- O Alloy monta somente o subdiretório `containers` do Docker como leitura para acessar os logs. Isso não concede
  acesso ao socket, a volumes nomeados nem capacidade de alterar containers; ainda assim, expõe logs de terceiros e o
  risco residual deve ser aceito em ambiente local. Em Docker rootless, `DOCKER_ROOT_DIR` deve apontar para
  `${HOME}/.local/share/docker`.
- Loki com `filesystem` não é HA. Em produção, substituir por object storage e compactor dedicado; o Alloy também passa
  a exigir coleta compatível com o orquestrador.
- `level` vira label de baixa cardinalidade; logs de bibliotecas que não sejam JSON recebem `level="unknown"`.

## Estágios de evolução

| Estágio | Escopo |
|---|---|
| 0 | `stdout` + Prometheus + Tempo, sem logs centralizados. |
| 1 (atual) | Loki + Alloy, logs JSON contextuais, bodies sanitizados e links Loki↔Tempo. |
| 2 | Réplicas HTTP exigem rate limiter compartilhado, IP confiável atrás do proxy e scrape Prometheus por DNS. |
| 3 | Separar API e worker do outbox em processos distintos; réplicas do dispatcher usam `SKIP LOCKED`. |
| 4 | HA: object storage para Loki/Tempo, PostgreSQL e RabbitMQ gerenciados, borda redundante e TLS. |

## Alternativas consideradas

- **Ler logs apenas com `docker compose logs`**: simples, mas não permite busca, retenção nem correlação.
- **Elasticsearch/OpenSearch**: indexação por conteúdo e recursos muito acima do necessário para uma stack local.
- **Logar o corpo integral**: descartado por risco de vazamento de credenciais e dados pessoais.
- **Usar `trace_id` como label**: descartado pela cardinalidade; o corpo JSON resolve a busca com `| json`.
