# service-identity

Microsserviço de identidade: registra usuários, autentica credenciais, mantém o perfil (nome e senha) e emite JWTs
consumidos pelo `service-task`. É o único serviço que conhece senhas e o segredo de
assinatura.

- Porta interna: `8080` (publicada como `127.0.0.1:8082`).
- Banco exclusivo: `postgres-identity` (database `identity`).
- Não acessa tarefas, RabbitMQ ou o banco de outro serviço.

## Estrutura

| Caminho | Responsabilidade |
|---|---|
| `cmd/server` | Bootstrap: logger, configuração, telemetria, sinais e execução do app. |
| `internal/domain` | Entidade `User` e erros de domínio. |
| `internal/service` | Regras de registro/login, perfil, troca de senha, normalização de credenciais e emissão de token. |
| `internal/repository` | SQL do usuário e migração idempotente da tabela `users`. |
| `internal/handler` | Rotas HTTP, validação de payload, JWT nas rotas `/me`, rate limit, métricas e health. |
| `internal/platform/app` | Pool PostgreSQL, migração, mux, middlewares e shutdown gracioso. |
| `internal/platform/config` | Leitura das variáveis de ambiente. |

## Endpoints

| Método | Rota | Sucesso | Erros |
|---|---|---|---|
| `POST` | `/register` | `201` com `{id, email, name, token}` | `400` payload/credenciais inválidas, `409` e-mail já cadastrado, `413` payload > 16 KiB, `429` rate limit, `500` interno |
| `POST` | `/login` | `200` com `{id, email, name, token}` | `400`, `401` credenciais inválidas, `413`, `429`, `500` |
| `GET` | `/me` | `200` com `{id, email, name}` | `401` JWT ausente/inválido, `404` usuário inexistente, `500` |
| `PATCH` | `/me` | `200` com `{id, email, name}` | `400` payload ou nome inválido, `401`, `404`, `413`, `500` |
| `PUT` | `/me/password` | `204` sem corpo | `400` payload ou nova senha inválida, `401`, `403` senha atual incorreta, `404`, `413`, `429`, `500` |
| `GET` | `/health` | `200` com `status: ok` | `503` com `status: degraded` quando o PostgreSQL falha |
| `GET` | `/metrics` | `200` no formato Prometheus | — |

`POST /register`, `POST /login` e `PUT /me/password` compartilham um rate limit de `10 req/min/IP` por
processo. Excedentes recebem `429` e `Retry-After: 60`. O IP vem de `RemoteAddr`;
o NGINX repassa `X-Real-IP` e `X-Forwarded-For`, mas o limitador não confia nesses
headers.

## Fluxo exclusivo do serviço

### Registro

1. O handler decodifica o JSON com limite de `16 KiB`, rejeita campos desconhecidos
   e exige `email` e `password` não vazios.
2. `AuthService.Register` normaliza o e-mail (`trim` + minúsculas) e valida:
   e-mail sintaticamente válido com até 254 caracteres; senha entre 8 e 72 bytes.
3. Consulta `users` por e-mail. Se existir, devolve `409`; qualquer erro diferente
   de `ErrNotFound` é tratado como falha interna.
4. Gera o hash bcrypt (`bcrypt.DefaultCost`) e insere o usuário.
5. A constraint `UNIQUE(email)` também protege contra corrida: violação `23505`
   vira `409`.
6. Emite um JWT HS256 com `user_id`, `email`, `iss`, `aud`, `iat` e `exp` (TTL de
   1 hora) e responde `201`.
7. Em falha de validação, o serviço incrementa a métrica de negócio e responde sem
   vazar detalhes internos.

### Login

1. Mesma validação de payload e normalização do registro.
2. Busca o usuário por e-mail. Ausência e senha incorreta produzem a mesma resposta
   `401` para não revelar se o e-mail existe.
3. Compara a senha com bcrypt e emite um novo JWT com TTL de 1 hora.

### Perfil

1. As rotas `/me` passam por `auth.Middleware` (`shared/auth`) com o mesmo `JWT_SECRET`; o `user_id` vem das
   claims do token, nunca do corpo. Sem segredo configurado, as rotas respondem `401` (fail-closed).
2. `GET /me` busca o usuário por `id`; o e-mail não é editável.
3. `PATCH /me` aceita somente `{"name"}`. `AuthService.UpdateName` aplica `trim` e exige de 1 a 100 caracteres
   (runas), sem caracteres de controle.
4. `PUT /me/password` exige `current_password` e `new_password`. A nova senha segue a regra do registro (8 a 72
   bytes) e precisa ser diferente da atual. A senha atual é conferida com bcrypt; se não confere, a resposta é `403`
   (e não `401`, para o frontend não encerrar a sessão). O novo hash bcrypt substitui o anterior.
5. Como o JWT é stateless, tokens emitidos antes da troca de senha continuam válidos até expirar (TTL 1 h).

### Token emitido

- Algoritmo: HS256 com `JWT_SECRET` compartilhado com o `service-task`.
- Emissor: `service-identity`; audiência: `service-task`.
- Validação do token é stateless: o `service-task` não consulta este serviço. O próprio Identity valida o token
  nas rotas `/me`.

## Modelo de dados

```sql
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE users ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
```

A migração é idempotente (`CREATE TABLE IF NOT EXISTS` e `ADD COLUMN IF NOT EXISTS`) e roda no startup, após o
pool responder ao `PingContext`. Bancos existentes recebem a coluna `name` vazia.

## Configuração

| Variável | Obrigatória | Padrão | Uso |
|---|---|---|---|
| `PORT` | não | `8080` | Porta HTTP interna. |
| `DATABASE_URL` | não | `postgres://identity:identity@localhost:5432/identity?sslmode=disable` | DSN do PostgreSQL. |
| `JWT_SECRET` | sim | — | Assinatura HS256; o processo não inicia sem ela. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | não | vazio | Endpoint OTLP/gRPC; vazio desabilita traces. |

Pool: máximo de 10 conexões abertas, 5 ociosas, vida máxima de 30 min e ociosidade
máxima de 5 min.

## Observabilidade

- Logs `slog` JSON com `correlation_id` e evento `http_request` por requisição.
- Métrica de negócio: `task_manager_identity_auth_attempts_total`, com labels
  `operation` (`register`, `login`, `get_profile`, `update_profile` ou `change_password`) e `outcome`
  (`success`, `invalid_payload`, `invalid_input`, `invalid_credentials`, `email_taken`, `unauthorized`,
  `wrong_password`, `not_found`, `rate_limited` ou `error`).
- Métricas HTTP RED comuns em `/metrics`.
- Traces via `otelhttp` e OTLP.

## Execução e testes

```sh
# a partir da raiz do repositório
docker compose up --build service-identity

# testes do módulo
go test -race ./...
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
```

O `Dockerfile` usa contexto de build na raiz para copiar `shared/` e compila com
`CGO_ENABLED=0`; o container roda como usuário não-root.
