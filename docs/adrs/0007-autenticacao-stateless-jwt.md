# 0007 - Estratégia de Autenticação Stateless via JSON Web Tokens (JWT) em Go

## Context and Problem Statement

O sistema necessita autenticar usuários no `service-identity` e permitir que essas credenciais e permissões sejam
validadas nos demais microsserviços (como o `service-task`) sem a necessidade de consultar o serviço de identidade a
cada requisição HTTP.

## Decision Drivers

* Validação de credenciais sem estado (stateless), visando escalabilidade horizontal.
* Validação de tokens de altíssima velocidade em Go via bibliotecas como `golang-jwt/jwt`.
* Padronização do transporte de claims (ID do usuário, e-mail) no cabeçalho `Authorization: Bearer <token>`.

## Considered Options

* **Opção 1:** Sessões centralizadas baseadas em Redis.
* **Opção 2:** Tokens JWT assinados e validados de forma descentralizada.

## Decision Outcome

Chosen option: "**Opção 2: Tokens JWT assinados com validação descentralizada**", permitindo que qualquer microsserviço
Go valide a assinatura do token localmente em microssegundos.

### Consequences

* Good, porque elimina a latência de chamadas HTTP internas para checagem de sessão a cada chamada de API.
* Good, porque a validação de assinatura de token em Go consome recursos de CPU insignificantes.
* Bad, porque inviabiliza a revogação instantânea de tokens sem a implementação de uma lista de bloqueio (blacklisting).
* Bad, porque exige atenção rigorosa ao tempo de expiração do token (`exp`).

### Confirmation

Validação por testes automatizados em Go onde o `service-task` decodifica e aceita um token válido gerado pelo
`service-identity` sem conectividade direta entre os dois containers.

## Pros and Cons of the Options

### Opção 1: Sessões Centralizadas em Memória (Redis)

* Good, porque permite invalidação e logout instantâneo de usuários.
* Bad, porque adiciona dependência de consulta em rede síncrona a cada requisição recebida pelos microsserviços.

### Opção 2: Tokens JWT Stateless em Go

* Good, porque preserva a autonomia e independência de performance entre os microsserviços.
* Good, porque simplifica a passagem de dados de contexto do usuário logado através dos middlewares de HTTP em Go.
* Bad, porque tokens mantêm validade até a expiração natural.