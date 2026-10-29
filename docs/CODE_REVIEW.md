# Revisão de simplificação do código

Revisão feita após a entrega das métricas Prometheus/Grafana, com o objetivo de reduzir
duplicação e ruído sem alterar comportamento observável, contratos HTTP/AMQP, dependências,
configuração de build ou testes.

## Método

1. Leitura de todos os módulos (`shared`, `service-identity`, `service-task`,
   `service-notification`, `frontend`) e das configurações de infraestrutura.
2. Cada alteração foi classificada como *behavior-preserving* antes de ser aplicada.
3. Testes de cada módulo executados após as mudanças (`go test -race ./...`,
   `npm test -- --no-watch`, `npm run build`, `nginx -t`).
4. Nenhum teste, contrato ou dependência foi modificado.

## Alterações aplicadas

| # | Arquivo | Alteração | Garantia de comportamento |
|---|---------|-----------|---------------------------|
| 1 | `shared/auth/auth.go` | `Middleware` passa a ter um único caminho de rejeição após tentar extrair e validar o token. | `notifyUnauthorized` e `writeUnauthorized` continuam executados exatamente uma vez, na mesma ordem, apenas em falha; sucesso segue direto para o próximo handler. |
| 2 | `shared/auth/auth.go` | `FromContext` usa type assertion com `ok` em uma linha. | Retorna `nil` para contexto ausente ou tipo diferente, como antes. |
| 3 | `shared/correlation/correlation.go` | `FromContext` usa type assertion com `ok` em uma linha. | Retorna `""` para contexto ausente ou tipo diferente, como antes. |
| 4 | `service-identity/internal/platform/app/app.go` | `Run` classifica o resultado de `serve` uma única vez, após o `select`. | O ramo de cancelamento continua chamando `Shutdown` antes de drenar o canal; erros de `Shutdown` continuam tendo precedência; `http.ErrServerClosed` continua sendo tratado como encerramento normal. |
| 5 | `service-identity/internal/handler/handler.go` | `Register` move a checagem de `ErrInvalidInput` para fora do bloco `err != nil`. | `errors.Is(nil, ...)` é falso, então o fluxo para sucesso e para erro interno permanece idêntico. |
| 6 | `service-identity/internal/handler/handler.go` | `Login` recebe a mesma simplificação do item 5. | Idem. |
| 7 | `service-task/internal/platform/app/app.go` | A interface `Channel` passa a embutir `messaging.Channel` e declarar apenas `Close()`. | A interface resultante expõe exatamente os mesmos métodos; os fakes de teste continuam satisfazendo o contrato. |
| 8 | `service-task/internal/platform/messaging/dispatcher.go` | `DispatchOne` reutiliza `recordSpanError` em vez de repetir `RecordError`/`SetStatus`. | O helper executa as mesmas duas chamadas de tracing. |
| 9 | `service-notification/internal/handler/consumer.go` | `handle` unifica a rejeição de evento inválido (JSON inválido ou campos obrigatórios ausentes) em um único bloco. | Mensagem de log, `Nack(false, false)`, métrica `rejected` e retorno permanecem iguais para ambos os casos; o motivo registrado continua sendo o erro de decode ou o texto de validação. |
| 10 | `service-notification/internal/platform/app/app.go` | `connect` retorna o erro corrente do `dial` ao detectar cancelamento, sem variável `lastErr`. | A variável anterior era atribuída imediatamente antes do `select`, logo o valor retornado é o mesmo. |
| 11 | `frontend/nginx.conf` | Os quatro `proxy_set_header` foram movidos para o escopo `server`. | `proxy_set_header` só tem efeito em locations com `proxy_pass`; as três rotas de API herdam os mesmos valores e as rotas estáticas não fazem proxy. |
| 12 | `frontend/src/app/features/tasks/pages/task-detail-page.component.ts` | Carregamento inicial usa `finalize` para encerrar o estado de loading. | `finalize` roda em `next`, `error` e cancelamento, cobrindo os dois pontos que antes chamavam `loading.set(false)`. |
| 13 | `frontend/src/app/features/tasks/pages/task-list-page.component.ts` | Carregamento inicial usa `finalize`; a lista de status reutiliza `STATUSES` do modelo. | Mesmos valores, mesma ordem e mesmo encerramento de loading. |

## Deliberadamente não alterado

- Contratos de erro HTTP, topologia AMQP, nomes de métricas e labels.
- Estrutura de pacotes e camadas (ADRs 0009 e 0010).
- Mensagens de log, campos e níveis.
- Tratamento de `errors.Join` no encerramento de `service-identity` e o fluxo de shutdown
  dos demais serviços, por envolverem semântica de erro com risco maior que o ganho.
- Testes, dependências e arquivos de build.

## Validação

```sh
(cd shared && go test -race ./...)
(cd service-identity && go test -race ./...)
(cd service-task && go test -race ./...)
(cd service-notification && go test -race ./...)
(cd frontend && npm run build)
(cd frontend && npm test -- --no-watch --code-coverage)
docker run --rm --add-host service-identity:127.0.0.1 \
  --add-host service-task:127.0.0.1 \
  --add-host service-notification:127.0.0.1 \
  -v "$PWD/frontend/nginx.conf:/etc/nginx/conf.d/default.conf:ro" \
  nginx:1.31.6-alpine nginx -t
gofmt -l shared service-identity service-task service-notification
git diff --check
```

Resultado: todos os módulos Go passam com `-race`, frontend compila em modo estrito com
32 testes passando, configuração do NGINX válida, `gofmt` e `git diff --check` limpos.
