# 0004 - Adoção da Arquitetura de Microsserviços para o Task Manager

## Context and Problem Statement

O sistema Task Manager precisa suportar múltiplos domínios de negócio (Autenticação, Gestão de Tarefas e Notificações)
com taxas de utilização e requisitos de escalabilidade distintos. Um monolito tradicional resultaria em alto acoplamento
e imobilidade no processo de implantação contínua.

## Decision Drivers

* Escalabilidade independente para módulos de alta demanda (ex: consulta e criação de tarefas).
* Isolamento de falhas entre domínios operacionais.
* Autonomia das equipes de desenvolvimento na evolução e deploy de código.

## Considered Options

* **Opção 1:** Arquitetura Monolítica Modular.
* **Opção 2:** Arquitetura de Microsserviços Orientada a Domínios.

## Decision Outcome

Chosen option: "**Opção 2: Arquitetura de Microsserviços Orientada a Domínios**", pois viabiliza o desacoplamento das
regras de negócio e a implantação autônoma dos serviços `service-identity`, `service-task` e `service-notification`.

### Consequences

* Good, porque permite escalar horizontalmente apenas os serviços sob alta carga.
* Good, porque falhas isoladas em um serviço (ex: envio de e-mails) não interrompem a criação de tarefas.
* Bad, porque introduz complexidade operacional de rede, monitoramento distribuído e orquestração.
* Bad, porque exige estratégias específicas para manter a consistência de dados distribuídos.

### Confirmation

Será validada mediante a existência de diretórios independentes com builds e pipelines de deploy totalmente isolados
para cada microsserviço.

## Pros and Cons of the Options

### Opção 1: Arquitetura Monolítica Modular

* Good, porque reduz a complexidade inicial de infraestrutura e rede.
* Good, porque simplifica transações entre entidades do banco de dados.
* Bad, porque limita a escalabilidade granular por componente.
* Bad, porque acopla os ciclos de implantação de todos os módulos em uma única release.

### Opção 2: Arquitetura de Microsserviços Orientada a Domínios

* Good, porque favorece a alta disponibilidade e resiliência ponta a ponta.
* Good, porque abre espaço para adoção de stacks heterogêneas no futuro se necessário.
* Bad, porque eleva a latência na comunicação inter-serviços via rede.