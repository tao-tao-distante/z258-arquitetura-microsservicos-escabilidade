# 0009 - Padronização de Arquitetura de Pastas e Layout de Projeto em Go (Golang)

## Context and Problem Statement

Com a adoção do Go no backend dos microsserviços (`service-identity`, `service-task` e `service-notification`), é
necessário definir um padrão idiomático para a organização física de pastas e pacotes nos repositórios. Sem uma
estrutura de diretórios bem definida, projetos Go costumam sofrer com acoplamento indevido, pacotes genéricos vazios
(ex: `utils` ou `helpers`), vazamento de código de domínio e o erro de compilação de dependências circulares
(`import cycle not allowed`).

## Decision Drivers

* **Idiomatismo Go:** Seguir as convenções amplamente aceitas pela comunidade Go e pela Standard Go Project Layout.
* **Encapsulamento Rígido:** Utilizar o mecanismo nativo do Go de diretórios `internal/` para impedir que pacotes
  privados sejam importados externamente.
* **Prevenção de Dependências Circulares:** Organizar os pacotes promovendo fluxo unidirecional de dependências.
* **Desacoplamento via Interfaces:** Facilitar a injeção de dependências e a criação de Mocks em testes unitários (TDD).
* **Consistência Multisserviço:** Garantir que todos os microsserviços do repositório mantenham a mesma estrutura de
  diretórios, reduzindo a carga cognitiva da equipe.

## Considered Options

* **Opção 1:** Layout Plano (Flat Layout) — todos os arquivos `.go` na raiz do serviço.
* **Opção 2:** Layout por Camadas estilo MVC (`controllers/`, `models/`, `services/`).
* **Opção 3:** Standard Go Project Layout centrado em Domínio e Clean Architecture (`cmd/`, `internal/`).

## Decision Outcome

Chosen option: "**Opção 3: Standard Go Project Layout centrado em Domínio e Clean Architecture**", pois combina a
convenção oficial da linguagem com a separação clara de responsabilidades (Entidades, Casos de Uso, Portas e
Adaptadores).

### Estrutura de Pastas Padronizada por Microsserviço

```text
service-name/
├── cmd/
│   └── server/
│       └── main.go           # Ponto de entrada (subida da app e DI)
├── internal/                 # Código privado do serviço (bloqueado pelo compilador Go)
│   ├── domain/               # Entidades de negócio, Value Objects e Erros de Domínio
│   ├── service/              # Regras de negócio e Casos de Uso (Usecases)
│   ├── handler/              # Adaptadores de entrada (HTTP Handlers e Consumers AMQP)
│   ├── repository/           # Adaptadores de saída (Acesso ao PostgreSQL e Queries)
│   └── platform/             # Configuração, middleware, banco, mensageria e logging
├── pkg/                      # (Opcional) Código utilitário reutilizável fora do serviço
├── Dockerfile                # Multi-stage build para Go
├── go.mod
└── go.sum
```

### Módulo compartilhado

Código idêntico entre serviços vive em `shared/`, um módulo Go separado consumido via `replace ../shared`:

```text
shared/
├── auth/          # Claims, JWT (HS256), bcrypt e middleware Bearer
├── correlation/   # Contexto e geração de correlation ID
├── httpx/         # Decode/Write JSON com limites de payload
├── logger/        # slog JSON + correlation_id
├── middleware/    # Correlação HTTP, logging e rate limit
├── telemetry/     # Provider OTel, propagador W3C e exporter OTLP/gRPC
├── go.mod
└── go.sum
```

O módulo não contém regra de negócio; domínio, handlers, repositórios e configuração continuam privados em cada serviço.
Como o context Docker passa a ser a raiz do repositório, os Dockerfiles copiam `shared/` e apenas o serviço alvo.
