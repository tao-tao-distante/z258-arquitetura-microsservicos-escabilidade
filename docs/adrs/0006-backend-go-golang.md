# 0006 - Escolha da Linguagem Go (Golang) para os Microsserviços Backend

## Context and Problem Statement

Para cumprir os SLOs de latência rigorosos (p95 <= 150ms e p99 <= 300ms) e manter um consumo reduzido de recursos no
ambiente Docker, os microsserviços backend (`service-identity`, `service-task` e `service-notification`) precisam de
alta performance de I/O, tempos de inicialização quase instantâneos e pegada de memória mínima.

## Decision Drivers

* Baixíssimo consumo de CPU e memória RAM por container.
* Compilação estática gerando um único binário executável sem dependência de runtime (permitindo imagens Docker
  minimalistas baseadas em `alpine` ou `scratch`).
* Concorrência nativa eficiente via Goroutines e Channels.
* Facilidade na escrita de testes de performance e benchmarks integrados.

## Considered Options

* **Opção 1:** Node.js com NestJS / Express.
* **Opção 2:** Go (Golang) utilizando biblioteca padrão / Gin / Fiber.
* **Opção 3:** Java com Spring Boot.

## Decision Outcome

Chosen option: "**Opção 2: Go (Golang)**", pois oferece performance de linguagem compilada com sintaxe simples,
excelente suporte à concorrência e inicialização imediata, ideal para arquiteturas de microsserviços de alta vazão.

### Consequences

* Good, porque os containers sobem em milissegundos e consomem pouquíssima memória (< 30MB RAM por serviço).
* Good, porque possibilita criar imagens Docker extremamente pequenas (binário estático distribuído).
* Good, porque atende com margem folgada aos SLOs de latência e throughput do projeto.
* Bad, porque exige tratamento de erros explícito (`if err != nil`) em todas as camadas de I/O.

### Confirmation

Verificação via checagem dos arquivos `go.mod` e `go.sum` em cada diretório de serviço, além de aferição de uso de
memória no `docker stats`.

## Pros and Cons of the Options

### Opção 1: Node.js com NestJS

* Good, porque compartilha a linguagem TypeScript com o frontend Angular.
* Bad, porque possui maior consumo de memória RAM por container e inicialização mais lenta.

### Opção 2: Go (Golang)

* Good, porque entrega performance e vazão de requisições por segundo (RPS) superiores.
* Good, porque a concorrência com Goroutines trata I/O de banco e mensagens de forma ultraleve.
* Bad, porque possui tratamento de erros explícito que aumenta a verbosidade do código.

### Opção 3: Java com Spring Boot

* Good, porque possui o ecossistema corporativo mais maduro da indústria.
* Bad, porque o consumo de memória e o tempo de cold start inviabilizam execuções ultraleves no Docker Compose.