# 0007 - Estratégia de Autenticação Stateless via JSON Web Tokens (JWT) em Go

## Status

Aceito.

## Data

2026-10-28

## Contexto e Problema

O sistema necessita autenticar usuários no `service-identity` e permitir que essas credenciais e permissões sejam
validadas nos demais microsserviços (como o `service-task`) sem a necessidade de consultar o serviço de identidade a
cada requisição HTTP.

## Fatores de Decisão

* Validação de credenciais sem estado (stateless), visando escalabilidade horizontal.
* Validação de tokens de altíssima velocidade em Go via bibliotecas como `golang-jwt/jwt`.
* Padronização do transporte de claims (ID do usuário, e-mail) no cabeçalho `Authorization: Bearer <token>`.

## Opções Consideradas

* **Opção 1:** Sessões centralizadas baseadas em Redis.
* **Opção 2:** Tokens JWT assinados e validados de forma descentralizada.

## Resultado da Decisão

Opção escolhida: "**Opção 2: Tokens JWT assinados com validação descentralizada**", permitindo que qualquer microsserviço
Go valide a assinatura do token localmente em microssegundos.

### Consequências

* Bom, porque elimina a latência de chamadas HTTP internas para checagem de sessão a cada chamada de API.
* Bom, porque a validação de assinatura de token em Go consome recursos de CPU insignificantes.
* Ruim, porque inviabiliza a revogação instantânea de tokens sem a implementação de uma lista de bloqueio (blacklisting).
* Ruim, porque exige atenção rigorosa ao tempo de expiração do token (`exp`).

### Confirmação

Validação por testes automatizados em Go onde o `service-task` decodifica e aceita um token válido gerado pelo
`service-identity` sem conectividade direta entre os dois containers.

### Adendo (2026-10-30): rotas de perfil no Identity

As rotas `GET/PATCH /me` e `PUT /me/password` do `service-identity` validam o mesmo JWT localmente com
`shared/auth.Middleware`, tomando o `user_id` das claims. A troca de senha não revoga tokens emitidos antes dela; o
risco fica limitado ao TTL de 1 hora, coerente com a consequência "inviabiliza a revogação instantânea" acima.

## Prós e Contras das Opções

### Opção 1: Sessões Centralizadas em Memória (Redis)

* Bom, porque permite invalidação e logout instantâneo de usuários.
* Ruim, porque adiciona dependência de consulta em rede síncrona a cada requisição recebida pelos microsserviços.

### Opção 2: Tokens JWT Stateless em Go

* Bom, porque preserva a autonomia e independência de performance entre os microsserviços.
* Bom, porque simplifica a passagem de dados de contexto do usuário logado através dos middlewares de HTTP em Go.
* Ruim, porque tokens mantêm validade até a expiração natural.
