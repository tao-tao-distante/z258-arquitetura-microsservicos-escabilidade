# 0005 - Adoção do Framework Angular para a Interface SPA

## Status

Aceito.

## Data

2026-10-28

## Contexto e Problema

O frontend da aplicação precisa entregar um painel interativo de gerenciamento de tarefas (estilo Kanban e visualizações
por lista) com estado local complexo, formulários com validações avançadas e tipagem estrita no consumo das APIs dos
microsserviços.

## Fatores de Decisão

* Padronização arquitetural nativa (injeção de dependência, módulos, roteamento).
* Suporte nativo ao TypeScript para garantia de contratos de API.
* Interceptors HTTP para autenticação e correlação de aplicação.

## Opções Consideradas

* **Opção 1:** React com ecossistema de bibliotecas externas (Redux, Axios).
* **Opção 2:** Angular Framework (SPA opinativa).
* **Opção 3:** Vue.js.

## Resultado da Decisão

Opção escolhida: "**Opção 2: Angular Framework**", pois oferece uma estrutura arquitetural completa e opinativa
nativas, reduzindo divergências de padrões no projeto.

### Consequências

* Bom, porque impõe um padrão de projeto claro (Modules/Standalone Components, Services, Interceptors).
* Bom, porque o `HttpClient` nativo facilita a injeção de headers globais como `x-correlation-id`.
* Ruim, porque apresenta uma curva de aprendizado inicial mais acentuada.
* Ruim, porque resulta em um bundle inicial ligeiramente maior em comparação com soluções minimalistas.

### Confirmação

A conformidade é verificada por testes existentes e pelo build Angular com TypeScript `strict` e `strictTemplates`.
Não há script ESLint nem SDK OpenTelemetry instalados atualmente.

## Prós e Contras das Opções

### Opção 1: React

* Bom, porque possui vasta comunidade e ecossistema abundante de bibliotecas.
* Neutro, porque dá liberdade total na escolha de roteamento e gerenciamento de estado.
* Ruim, porque exige decisões frequentes para estruturar o ecossistema básico.

### Opção 2: Angular Framework

* Bom, porque possui tudo integrado nativamente no core (Router, Forms, HttpClient, DI).
* Bom, porque força o uso estrito de TypeScript por padrão.
* Ruim, porque possui maior boilerplate de código inicial.
