# 0004 - Adoção da Arquitetura de Microsserviços para o Task Manager

## Status

Aceito.

## Data

2026-10-28

## Contexto e Problema

O sistema Task Manager precisa suportar múltiplos domínios de negócio (Autenticação, Gestão de Tarefas e Notificações)
com taxas de utilização e requisitos de escalabilidade distintos. Um monolito tradicional resultaria em alto acoplamento
e imobilidade no processo de implantação contínua.

## Fatores de Decisão

* Escalabilidade independente para módulos de alta demanda (ex: consulta e criação de tarefas).
* Isolamento de falhas entre domínios operacionais.
* Autonomia das equipes de desenvolvimento na evolução e deploy de código.

## Opções Consideradas

* **Opção 1:** Arquitetura Monolítica Modular.
* **Opção 2:** Arquitetura de Microsserviços Orientada a Domínios.

## Resultado da Decisão

Opção escolhida: "**Opção 2: Arquitetura de Microsserviços Orientada a Domínios**", pois viabiliza o desacoplamento das
regras de negócio e a implantação autônoma dos serviços `service-identity`, `service-task` e `service-notification`.

### Consequências

* Bom, porque permite escalar horizontalmente apenas os serviços sob alta carga.
* Bom, porque falhas isoladas em um serviço (ex: envio de e-mails) não interrompem a criação de tarefas.
* Ruim, porque introduz complexidade operacional de rede, monitoramento distribuído e orquestração.
* Ruim, porque exige estratégias específicas para manter a consistência de dados distribuídos.

### Confirmação

Será validada mediante a existência de diretórios independentes com builds e pipelines de deploy totalmente isolados
para cada microsserviço.

## Prós e Contras das Opções

### Opção 1: Arquitetura Monolítica Modular

* Bom, porque reduz a complexidade inicial de infraestrutura e rede.
* Bom, porque simplifica transações entre entidades do banco de dados.
* Ruim, porque limita a escalabilidade granular por componente.
* Ruim, porque acopla os ciclos de implantação de todos os módulos em uma única release.

### Opção 2: Arquitetura de Microsserviços Orientada a Domínios

* Bom, porque favorece a alta disponibilidade e resiliência ponta a ponta.
* Bom, porque abre espaço para adoção de stacks heterogêneas no futuro se necessário.
* Ruim, porque eleva a latência na comunicação inter-serviços via rede.
