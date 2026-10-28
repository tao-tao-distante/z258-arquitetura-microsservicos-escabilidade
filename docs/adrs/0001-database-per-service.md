# 0001 - Padrão Database-per-Service com PostgreSQL em Go

## Status

Aceito.

## Data

2026-10-28

## Contexto e Problema

O sistema Task Manager precisa escalar seus módulos de Usuários (`service-identity`) e Tarefas (`service-task`) de
maneira independente. O compartilhamento de uma única base de dados monolítica criaria forte acoplamento e um ponto
único de falha, impedindo a evolução autônoma dos microsserviços desenvolvidos em Go.

## Fatores de Decisão

* Necessidade de escalabilidade independente entre o domínio de tarefas e o de identidade.
* Isolamento de falhas para evitar que travamentos no banco de tarefas afetem a autenticação.
* Autonomia de deploy e migrations independentes para cada microsserviço.

## Opções Consideradas

* **Opção 1:** Banco de dados compartilhado (Shared Database).
* **Opção 2:** Instâncias/esquemas lógicos isolados no PostgreSQL por serviço (Database-per-Service).

## Resultado da Decisão

Opção escolhida: "**Opção 2: Database-per-Service**", pois garante o desacoplamento total do modelo de dados e permite que
cada serviço Go gerencie de forma isolada suas conexões e tabelas no PostgreSQL.

### Consequências

* Bom, porque garante isolamento de falhas e independência de deploy.
* Bom, porque limita o raio de impacto em caso de problemas de performance na base de dados.
* Ruim, porque impede queries com `JOIN` direto entre entidades de serviços diferentes.
* Ruim, porque exige gerenciamento de consistência eventual entre microsserviços.

### Confirmação

Será validado via code review e checagem das variáveis de ambiente, garantindo que o `service-task` não possua
credenciais de acesso nem conexões ativas com a base do `service-identity`.

## Prós e Contras das Opções

### Opção 1: Banco de Dados Compartilhado

* Bom, porque facilita a escrita de consultas complexas através de `JOIN`s diretos.
* Neutro, porque reduz o custo inicial de infraestrutura.
* Ruim, porque introduz acoplamento forte entre os esquemas de banco de dados.
* Ruim, porque torna o banco de dados um ponto único de falha para todo o sistema.

### Opção 2: Database-per-Service

* Bom, porque assegura que mudanças na estrutura de dados de um serviço não quebrem outros.
* Bom, porque melhora a resiliência e a performance sob carga.
* Neutro, porque demanda configuração de múltiplos esquemas/instâncias no Docker Compose.
* Ruim, porque adiciona complexidade na manutenção da integridade referencial.
