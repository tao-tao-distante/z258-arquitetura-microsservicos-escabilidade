# 0006 - Escolha da Linguagem Go (Golang) para os Microsserviços Backend

## Context and Problem Statement

Os microsserviços backend (`service-identity`, `service-task` e `service-notification`) precisam de concorrência para I/O,
binários simples de distribuir e consumo de recursos mensurável no ambiente Docker. Metas de latência e vazão dependem
de benchmark e teste de carga; a escolha da linguagem, isoladamente, não as comprova.

## Decision Drivers

* Potencial de baixo consumo de CPU e memória por container, sujeito a medição.
* Compilação estática gerando um único binário executável sem dependência de runtime (permitindo imagens Docker
  minimalistas baseadas em `alpine` ou `scratch`).
* Concorrência nativa eficiente via Goroutines e Channels.
* Facilidade na escrita de testes de performance e benchmarks integrados.

## Considered Options

* **Opção 1:** Node.js com NestJS / Express.
* **Opção 2:** Go (Golang) utilizando biblioteca padrão / Gin / Fiber.
* **Opção 3:** Java com Spring Boot.

## Decision Outcome

Chosen option: "**Opção 2: Go (Golang)**", pois oferece linguagem compilada com sintaxe simples, suporte à concorrência
e geração de binário único.

### Consequences

* Good, porque possibilita criar imagens Docker extremamente pequenas (binário estático distribuído).
* Neutral, porque latência, vazão, memória e inicialização precisam ser medidas; não há benchmark registrado.
* Bad, porque exige tratamento de erros explícito (`if err != nil`) em todas as camadas de I/O.

### Confirmation

Verificação via checagem dos arquivos `go.mod` e `go.sum` em cada diretório de serviço, além de aferição de uso de
memória no `docker stats`.

## Pros and Cons of the Options

### Opção 1: Node.js com NestJS

* Good, porque compartilha a linguagem TypeScript com o frontend Angular.
* Neutral, porque diferenças de memória e inicialização dependeriam de implementação e benchmark comparável.

### Opção 2: Go (Golang)

* Good, porque Goroutines e biblioteca padrão oferecem primitivas adequadas para I/O concorrente.
* Bad, porque possui tratamento de erros explícito que aumenta a verbosidade do código.

### Opção 3: Java com Spring Boot

* Good, porque possui o ecossistema corporativo mais maduro da indústria.
* Neutral, porque consumo de memória e cold start dependeriam de implementação e medição comparável.
