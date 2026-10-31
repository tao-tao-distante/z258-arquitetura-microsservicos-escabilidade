# frontend

SPA Angular 20 do Task Manager. Consome as três APIs por URLs relativas
(`/api/identity`, `/api/tasks`, `/api/notifications`) e serve a aplicação via NGINX.

- Porta pública: `8081` (NGINX na porta `80` do container).
- Não fala diretamente com bancos, RabbitMQ ou Prometheus.
- Build estático em `dist/frontend/browser`.

## Stack

- Angular 20 standalone components, Signals, reactive forms e lazy routes.
- RxJS 7.8 e `provideZoneChangeDetection({ eventCoalescing: true })`.
- TypeScript 5.8 em modo `strict` e `strictTemplates`.
- Testes com Vitest + `@vitest/coverage-v8` em jsdom.

## Estrutura (feature-first, ADR-0010)

```text
src/app
├── app.component.ts        shell autenticada (rail, topo, navegação)
├── app.config.ts           providers globais e interceptors HTTP
├── app.routes.ts           rotas raiz com lazy loading e authGuard
├── core
│   ├── guards              authGuard
│   ├── http                apiErrorMessage
│   ├── interceptors        jwt e tratamento de 401
│   ├── models              AuthResponse, AuthUser, Credentials, ProfileUpdate, PasswordChange
│   └── services            AuthService
├── shared/components       AuthLayout, PageHeader, FormField, ConfirmDialog
└── features
    ├── auth                login e register
    ├── dashboard           visão geral
    ├── tasks               board, lista, detalhe e formulário
    ├── notifications       tela de atividades/health do consumidor
    └── profile             perfil: editar nome e alterar senha
```

## Rotas

| Rota | Componente | Proteção |
|---|---|---|
| `/` | redireciona para `/dashboard` | — |
| `/login` | `LoginComponent` | pública |
| `/register` | `RegisterComponent` | pública |
| `/dashboard` | `DashboardPageComponent` | `authGuard` |
| `/tasks` | `TaskListPageComponent` | `authGuard` |
| `/tasks/board` | `TaskBoardPageComponent` | `authGuard` |
| `/tasks/new` | `TaskFormPageComponent` | `authGuard` |
| `/tasks/:id/edit` | `TaskFormPageComponent` | `authGuard` |
| `/tasks/:id` | `TaskDetailPageComponent` | `authGuard` |
| `/activities` | `NotificationsComponent` | `authGuard` |
| `/profile` | `ProfilePageComponent` | `authGuard` |
| `/notifications` | redireciona para `/activities` | — |
| `**` | redireciona para `/dashboard` | — |

`authGuard` só libera a rota quando existe token válido e usuário persistido;
caso contrário, redireciona para `/login` com `returnUrl`.

## Fluxo HTTP exclusivo do frontend

A aplicação usa `jwtInterceptor`; trace e correlação pertencem à borda e às APIs.

1. **Trace**: o NGINX cria o span de entrada, injeta `traceparent` antes do proxy e devolve `X-Trace-ID` nas respostas
   de `/api/*`.
2. **Correlação**: a API gera `x-correlation-id` UUID v4 quando o header está ausente.
3. **Logs**: o access log do NGINX é JSON (`service_name=frontend-edge`, método, URI sem query, status, duração,
   `trace_id` e `correlation_id`); query string, headers de autenticação e bodies não são registrados.
4. **JWT**: o token é anexado como `Authorization: Bearer ...` somente quando a
   URL pertence a `environment.taskUrl` (`/api/tasks`) ou a `/api/identity/me`
   (e subrotas, como `/api/identity/me/password`). Login, registro, health e
   demais rotas do Identity nunca recebem o token.
5. **401**: em resposta `401` de endpoint protegido, a sessão é limpa e o usuário
   é levado a `/login` com `returnUrl` interno. Respostas `401` de
   `/api/identity/login` e `/api/identity/register` não disparam logout.
6. Erros viram mensagens em pt-BR por `apiErrorMessage`.

### Sessão

- `AuthService` persiste `tm_token` e `tm_user` em `localStorage`.
- Na inicialização, token malformado, sem `exp` numérico ou expirado é descartado
  junto com o usuário.
- `sessionToken()` revalida a expiração a cada leitura e limpa a sessão inválida.
- `tm_user` guarda `{id, email, name}`; sessões antigas sem `name` são aceitas com nome vazio.
- O risco residual de XSS e as mitigações estão em [`SECURITY.md`](SECURITY.md).

### Perfil

- `Perfil` é o primeiro item (`01`) do menu lateral e da navegação mobile; o bloco "minha conta" do rail também
  leva a `/profile`. As iniciais e o rótulo da conta usam o nome quando definido (senão, o e-mail).
- `ProfilePageComponent` carrega `GET /api/identity/me` ao abrir. O e-mail é somente leitura.
- **Nome**: `PATCH /api/identity/me` com o nome aparado (1 a 100 caracteres); o `AuthService` atualiza o signal
  `user` e o `tm_user`.
- **Senha**: senha atual, nova senha e confirmação. A validação local exige a senha atual, nova senha de 8 a 72
  caracteres, confirmação igual e nova senha diferente da atual. `PUT /api/identity/me/password`; `403` vira
  "Senha atual incorreta." e `429` mostra o tempo de espera. Após sucesso, os campos são limpos.

### Dados de tarefas

`TaskDataService` centraliza o contrato com `/api/tasks`:

- `list()` busca páginas de 100 itens com `expand` até receber uma página menor;
- `get`, `create`, `update` e `remove` validam o status esperado (`200`, `201`,
  `200`, `204`) e transformam status inesperado em `HttpErrorResponse`;
- `history(id, offset, limit)` busca páginas de 20 entradas em `/tasks/{id}/history`;
- corpo vazio em resposta que deveria ter corpo também vira erro.

### Prioridade, prazo e histórico

- **Formulário**:
  - campo de prioridade (`URGENTE`, `ALTA`, `MÉDIA`, `BAIXA`; padrão `MÉDIA`);
  - prazo em `<input type="date">`, com botão "remover prazo".
  - Um prazo anterior à data local é bloqueado. Na edição, um prazo já vencido pode ser mantido, mas não trocado por outra
    data passada.
  - O `PUT` sempre envia `priority` e `due_date`.
- **Badges**: `PriorityBadgeComponent` e `DueDateComponent` ("Vence hoje", "Vence em 3 dias", "Atrasada há 2 dias")
  aparecem na lista, no quadro e no detalhe. Tarefas concluídas nunca aparecem como atrasadas.
- **Lista** (`/tasks`):
  - colunas de prioridade, prazo e data de criação;
  - filtro por prioridade, botão "Atrasadas" e ordenação por quadro, prioridade, mais recentes ou prazo mais próximo.
  - Filtros e ordenação rodam no cliente sobre a lista completa e ficam na URL
    (`?status=&priority=&overdue=true&sort=`).
- **Detalhe** (`/tasks/:id`):
  - mostra prioridade, prazo e data de criação;
  - mostra a linha do tempo `TaskHistoryComponent` ("Criada", "Editada", "Movida", com cada mudança em texto, como
    "Prioridade: Média → Alta");
  - o histórico tem carregamento, erro e paginação ("Carregar mais") próprios, e uma falha nele não afeta a página.
- **Datas**: a formatação e as regras de prazo ficam em `features/tasks/task.utils.ts`, como funções puras.
  - Datas de calendário são formatadas sem `Date`, para não deslocar o dia pelo fuso.

## Ambientes e proxy

| Ambiente | Arquivo | URLs |
|---|---|---|
| Desenvolvimento | `src/environments/environment.ts` | relativas (`/api/...`) |
| Produção | `src/environments/environment.production.ts` | relativas (`/api/...`) |

- `npm start` usa `proxy.conf.json` para encaminhar as três rotas ao NGINX em
  `localhost:8081`, que remove o prefixo e seleciona o upstream.
- Em produção, o `nginx.conf` faz o mesmo roteamento para os nomes dos containers
  e adiciona headers de segurança (CSP, COOP, Referrer-Policy, nosniff,
  X-Frame-Options) e cache imutável para assets com hash.

## Design

- Direção visual e tokens em [`DESIGN.md`](DESIGN.md).
- Protótipo: [Task Manager — Product UI — Angular](https://www.figma.com/design/30NGvSADIVJnbFmHP3rhx0/Task-Manager-%E2%80%94-Product-UI---Angular?node-id=7-2&m=dev).
- Fontes locais (Archivo e DM Sans) em `src/assets/fonts`, sem dependência de CDN.
- Layout desktop e mobile são composições distintas, não uma escala do desktop.

## Comandos

```sh
npm install
npm start                  # dev server em http://localhost:4200
npm run build              # build de produção em dist/frontend/browser
npm test -- --no-watch --code-coverage
```

## Docker

O `Dockerfile` tem dois estágios:

1. `node:20.19-alpine` executa `npm ci` e `npm run build -- --configuration production`;
2. `nginx:1.31.8-alpine` instala `nginx-module-otel`, copia o build e configura tracing e balanceamento.

```sh
# a partir da raiz do repositório
docker compose up --build frontend
```

O container depende dos três serviços saudáveis e é a única entrada pública da
aplicação.
