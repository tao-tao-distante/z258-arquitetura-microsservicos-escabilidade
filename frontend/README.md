# frontend

SPA Angular 20 do Task Manager. Consome as três APIs por URLs relativas
(`/api/identity`, `/api/tasks`, `/api/notifications`), injeta `x-correlation-id` em
todas as requisições e serve a aplicação via NGINX.

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
│   ├── interceptors        correlation-id e jwt
│   ├── models              AuthResponse, Credentials
│   └── services            AuthService
├── shared/components       AuthLayout, PageHeader, FormField, ConfirmDialog
└── features
    ├── auth                login e register
    ├── dashboard           visão geral
    ├── tasks               board, lista, detalhe e formulário
    └── notifications       tela de atividades/health do consumidor
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
| `/notifications` | redireciona para `/activities` | — |
| `**` | redireciona para `/dashboard` | — |

`authGuard` só libera a rota quando existe token válido e usuário persistido;
caso contrário, redireciona para `/login` com `returnUrl`.

## Fluxo HTTP exclusivo do frontend

A ordem dos interceptors é `correlationIdInterceptor` e depois `jwtInterceptor`.

1. **Correlação**: toda requisição recebe um `x-correlation-id` UUID v4. Usa
   `crypto.randomUUID()` quando disponível e um fallback com
   `crypto.getRandomValues` definindo versão 4 e variante RFC 4122.
2. **JWT**: o token é anexado como `Authorization: Bearer ...` somente quando a
   URL pertence a `environment.taskUrl` (`/api/tasks`). Login, registro e health
   de notificações nunca recebem o token.
3. **401**: em resposta `401` de endpoint protegido, a sessão é limpa e o usuário
   é levado a `/login` com `returnUrl` interno. Respostas `401` de
   `/api/identity/login` e `/api/identity/register` não disparam logout.
4. Erros viram mensagens em pt-BR por `apiErrorMessage`.

### Sessão

- `AuthService` persiste `tm_token` e `tm_user` em `localStorage`.
- Na inicialização, token malformado, sem `exp` numérico ou expirado é descartado
  junto com o usuário.
- `sessionToken()` revalida a expiração a cada leitura e limpa a sessão inválida.
- O risco residual de XSS e as mitigações estão em [`SECURITY.md`](SECURITY.md).

### Dados de tarefas

`TaskDataService` centraliza o contrato com `/api/tasks`:

- `list()` busca páginas de 100 itens com `expand` até receber uma página menor;
- `get`, `create`, `update` e `remove` validam o status esperado (`200`, `201`,
  `200`, `204`) e transformam status inesperado em `HttpErrorResponse`;
- corpo vazio em resposta que deveria ter corpo também vira erro.

## Ambientes e proxy

| Ambiente | Arquivo | URLs |
|---|---|---|
| Desenvolvimento | `src/environments/environment.ts` | relativas (`/api/...`) |
| Produção | `src/environments/environment.production.ts` | relativas (`/api/...`) |

- `npm start` usa `proxy.conf.json` para encaminhar `/api/identity` a
  `localhost:8082`, `/api/tasks` a `localhost:8083` e `/api/notifications` a
  `localhost:8084`, removendo o prefixo.
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
2. `nginx:1.31.6-alpine` copia o build e o `nginx.conf`.

```sh
# a partir da raiz do repositório
docker compose up --build frontend
```

O container depende dos três serviços saudáveis e é a única entrada pública da
aplicação.
