# 0005 - Adoção do Framework Angular para a Interface SPA

## Context and Problem Statement

O frontend da aplicação precisa entregar um painel interativo de gerenciamento de tarefas (estilo Kanban e visualizações
por lista) com estado local complexo, formulários com validações avançadas e tipagem estrita no consumo das APIs dos
microsserviços.

## Decision Drivers

* Padronização arquitetural nativa (injeção de dependência, módulos, roteamento).
* Suporte nativo ao TypeScript para garantia de contratos de API.
* Interceptors HTTP para autenticação e correlação de aplicação.

## Considered Options

* **Opção 1:** React com ecossistema de bibliotecas externas (Redux, Axios).
* **Opção 2:** Angular Framework (SPA opinativa).
* **Opção 3:** Vue.js.

## Decision Outcome

Chosen option: "**Opção 2: Angular Framework**", pois oferece uma estrutura arquitetural completa e opinativa
"out-of-the-box", reduzindo divergências de padrões no projeto.

### Consequences

* Good, porque impõe um padrão de projeto claro (Modules/Standalone Components, Services, Interceptors).
* Good, porque o `HttpClient` nativo facilita a injeção de headers globais como `x-correlation-id`.
* Bad, porque apresenta uma curva de aprendizado inicial mais acentuada.
* Bad, porque resulta em um bundle inicial ligeiramente maior em comparação com soluções minimalistas.

### Confirmation

A conformidade é verificada por testes existentes e pelo build Angular com TypeScript `strict` e `strictTemplates`.
Não há script ESLint nem SDK OpenTelemetry instalados atualmente.

## Pros and Cons of the Options

### Opção 1: React

* Good, porque possui vasta comunidade e ecossistema abundante de bibliotecas.
* Neutral, porque dá liberdade total na escolha de roteamento e gerenciamento de estado.
* Bad, porque exige decisões frequentes para estruturar o ecossistema básico.

### Opção 2: Angular Framework

* Good, porque possui tudo integrado nativamente no core (Router, Forms, HttpClient, DI).
* Good, porque força o uso estrito de TypeScript por padrão.
* Bad, porque possui maior boilerplate de código inicial.
