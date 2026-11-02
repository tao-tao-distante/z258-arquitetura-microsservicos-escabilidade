# 0006 - Escolha da Linguagem Go (Golang) para os Microsserviços Backend

## Status

Aceito. A implementação atual fixa Go 1.23 e usa `net/http`.

## Data

2026-10-28

## Nota de atualização

Os quatro módulos Go declaram `go 1.23`. Os serviços usam `net/http`, `context.Context`, `log/slog` e tratamento explícito
de erros. Os builds definem `CGO_ENABLED=0` e geram binários para imagens Alpine. Essas escolhas concretizam a decisão;
latência e consumo continuam sujeitos a medição.

## Contexto e Problema

Os microsserviços backend (`service-identity`, `service-task` e `service-notification`) precisam de concorrência para I/O,
binários simples de distribuir e consumo de recursos mensurável no ambiente Docker. Metas de latência e vazão dependem
de benchmark e teste de carga; a escolha da linguagem, isoladamente, não as comprova.

## Fatores de Decisão

* Potencial de baixo consumo de CPU e memória por container, sujeito a medição.
* Compilação estática gerando um único binário executável sem dependência de runtime (permitindo imagens Docker
  minimalistas baseadas em `alpine` ou `scratch`).
* Concorrência nativa eficiente via Goroutines e Channels.
* Facilidade na escrita de testes de performance e benchmarks integrados.

## Opções Consideradas

* **Opção 1:** Node.js com NestJS / Express.
* **Opção 2:** Go (Golang) utilizando biblioteca padrão / Gin / Fiber.
* **Opção 3:** Java com Spring Boot.

## Resultado da Decisão

Opção escolhida: "**Opção 2: Go (Golang)**", pois oferece linguagem compilada com sintaxe simples, suporte à concorrência
e geração de binário único.

### Consequências

* Bom, porque possibilita criar imagens Docker extremamente pequenas (binário estático distribuído).
* Neutro, porque latência, vazão, memória e inicialização precisam ser medidas; não há benchmark registrado.
* Ruim, porque exige tratamento de erros explícito (`if err != nil`) em todas as camadas de I/O.

### Confirmação

Verificação via checagem dos arquivos `go.mod` e `go.sum` em cada diretório de serviço, além de aferição de uso de
memória no `docker stats`.

## Prós e Contras das Opções

### Opção 1: Node.js com NestJS

* Bom, porque compartilha a linguagem TypeScript com o frontend Angular.
* Neutro, porque diferenças de memória e inicialização dependeriam de implementação e benchmark comparável.

### Opção 2: Go (Golang)

* Bom, porque Goroutines e biblioteca padrão oferecem primitivas adequadas para I/O concorrente.
* Ruim, porque possui tratamento de erros explícito que aumenta a verbosidade do código.

### Opção 3: Java com Spring Boot

* Bom, porque possui o ecossistema corporativo mais maduro da indústria.
* Neutro, porque consumo de memória e cold start dependeriam de implementação e medição comparável.
