# 0001 - Padrão Database-per-Service com PostgreSQL em Go

## Context and Problem Statement

O sistema Task Manager precisa escalar seus módulos de Usuários (`service-identity`) e Tarefas (`service-task`) de
maneira independente. O compartilhamento de uma única base de dados monolítica criaria forte acoplamento e um ponto
único de falha, impedindo a evolução autônoma dos microsserviços desenvolvidos em Go.

## Decision Drivers

* Necessidade de escalabilidade independente entre o domínio de tarefas e o de identidade.
* Isolamento de falhas para evitar que travamentos no banco de tarefas afetem a autenticação.
* Autonomia de deploy e migrations independentes para cada microsserviço.

## Considered Options

* **Opção 1:** Banco de dados compartilhado (Shared Database).
* **Opção 2:** Instâncias/esquemas lógicos isolados no PostgreSQL por serviço (Database-per-Service).

## Decision Outcome

Chosen option: "**Opção 2: Database-per-Service**", pois garante o desacoplamento total do modelo de dados e permite que
cada serviço Go gerencie de forma isolada suas conexões e tabelas no PostgreSQL.

### Consequences

* Good, porque garante isolamento de falhas e independência de deploy.
* Good, porque limita o raio de impacto em caso de problemas de performance na base de dados.
* Bad, porque impede queries com `JOIN` direto entre entidades de serviços diferentes.
* Bad, porque exige gerenciamento de consistência eventual entre microsserviços.

### Confirmation

Será validado via code review e checagem das variáveis de ambiente, garantindo que o `service-task` não possua
credenciais de acesso nem conexões ativas com a base do `service-identity`.

## Pros and Cons of the Options

### Opção 1: Banco de Dados Compartilhado

* Good, porque facilita a escrita de consultas complexas através de `JOIN`s diretos.
* Neutral, porque reduz o custo inicial de infraestrutura.
* Bad, porque introduz acoplamento forte entre os esquemas de banco de dados.
* Bad, porque torna o banco de dados um ponto único de falha para todo o sistema.

### Opção 2: Database-per-Service

* Good, porque assegura que mudanças na estrutura de dados de um serviço não quebrem outros.
* Good, porque melhora a resiliência e a performance sob carga.
* Neutral, porque demanda configuração de múltiplos esquemas/instâncias no Docker Compose.
* Bad, porque adiciona complexidade na manutenção da integridade referencial.