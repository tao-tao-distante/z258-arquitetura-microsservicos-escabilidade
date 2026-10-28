# Documento de Arquitetura de Software: Task Manager System

## 1. Visão Geral e Contexto do Sistema

O **Task Manager System** é uma plataforma distribuída de gerenciamento de tarefas baseada em uma **Arquitetura de
Microsserviços em Go (Golang)** e uma interface **Single Page Application (SPA) em Angular**. O sistema foi projetado
para suportar alta concorrência de acessos, com isolamento de domínios, processamento assíncrono de eventos e
observabilidade ponta a ponta.

### Ecossistema do Sistema

* **Frontend:** Single Page Application desenvolvida em **Angular** (servida via NGINX em produção).
* **Service Identity (`service-identity`):** Microsserviço em **Go** responsável por autenticação, emissão e validação
  de tokens JWT.
* **Service Task (`service-task`):** Microsserviço em **Go** para gestão de projetos, tarefas, prazos e atribuições.
* **Service Notification (`service-notification`):** Microsserviço em **Go** para consumo assíncrono de eventos de
  notificação.
* **Infraestrutura de Apoio:** PostgreSQL (1 banco isolado por serviço), RabbitMQ (Mensageria AMQP com DLQ) e
  OpenTelemetry Collector (Coleta de métricas e traces para Prometheus, Tempo e Grafana).

---

## 2. Rastreabilidade das Decisões Arquiteturais (Mapeamento de ADRs)

Toda a estrutura do sistema é fundamentada nos Architecture Decision Records (ADRs) documentados no repositório:

| ID ADR       | Título / Decisão Arquitetural                              | Impacto na Arquitetura                                                                             |
|--------------|------------------------------------------------------------|----------------------------------------------------------------------------------------------------|
| **ADR-0001** | Padrão *Database-per-Service* com PostgreSQL               | Desacoplamento de dados; isolamento de falhas entre `service-identity` e `service-task`.           |
| **ADR-0002** | Rastreamento Distribuído (`x-correlation-id` e `log/slog`) | Rastreabilidade fim-a-fim do Frontend até filas AMQP usando logs JSON nativos do Go.               |
| **ADR-0003** | Comunicação Assíncrona via RabbitMQ com DLQ                | Cumprimento de SLOs de latência na criação de tarefas e resiliência via Dead Letter Queue.         |
| **ADR-0004** | Adoção de Arquitetura de Microsserviços                    | Autonomia de deploy, escalabilidade horizontal seletiva e isolamento de domínios.                  |
| **ADR-0005** | Framework Frontend Angular                                 | Padronização arquitetural com TypeScript, interceptors HTTP globais e injeção de dependência.      |
| **ADR-0006** | Linguagem Go (Golang) para o Backend                       | Altíssimo throughput, baixíssimo consumo de memória RAM (< 30MB) e inicialização em milissegundos. |
| **ADR-0007** | Autenticação Stateless via JSON Web Tokens (JWT)           | Validação descentralizada de credenciais em Go sem dependência de chamadas síncronas de rede.      |
| **ADR-0008** | Containerização via Docker Compose e Multi-Stage Build     | Reprodutibilidade total do ambiente com binários estáticos pequenos em containers Linux leves.     |

---

## 3. Objetivos de Nível de Serviço (SLOs)

Com a adoção do Go no backend, os SLOs de latência e throughput definidos para a aplicação são:

* **SLO-01 (Latência GET Tarefas):** 95% das requisições de consulta de tarefas (`GET /tasks`) devem responder em menos
  de **150ms** ($p_{95} \le 150\text{ms}$).
* **SLO-02 (Latência POST Criar Tarefa):** 99% das requisições de criação de tarefas (`POST /tasks`) devem responder em
  menos de **300ms** ($p_{99} \le 300\text{ms}$).
* **SLO-03 (Disponibilidade):** O sistema deve manter **99.5% de disponibilidade** geral mensurada via Health Checks.
* **SLO-04 (Vazão / Throughput):** O endpoint central de tarefas deve suportar no mínimo **150 RPS** (operações por
  segundo) sem degradação do $p_{95}$.

---

## 4. Estratégia de Observabilidade Distribuída

1. **Health Check Standard (`GET /health`):** Todos os microsserviços Go expõem uma rota `/health` checando
   conectividade ativa com recursos dependentes (PostgreSQL e RabbitMQ).
2. **Correlation ID Propagado:**

* O frontend Angular gera um UUID v4 no header `x-correlation-id` em todas as chamadas `HttpClient`.
* Os middlewares HTTP em Go capturam e registram o ID usando o logger estruturado nativo do Go (`log/slog`).
* Quando o `service-task` publica eventos no RabbitMQ, o `x-correlation-id` é injetado nos cabeçalhos AMQP
  (`amqp.Table`) para ser extraído pelo consumidor `service-notification`.


3. **Telemetria Centralizada:**

* **Metrics:** Exportação Prometheus para métricas de runtime do Go (Goroutines, uso de memória, GC).
* **Traces:** Spans OpenTelemetry enviados via OTLP/gRPC ao Grafana Tempo.

---

## 5. Estratégia de Segurança

* **Autenticação Stateless:** Tokens JWT com HMAC-SHA256 gerados pelo `service-identity` contendo `claims` de
  identificação do usuário.
* **Isolamento de Dados (*Least Privilege*):** Cada container Go conecta-se exclusivamente ao seu próprio banco no
  PostgreSQL através de credenciais distintas.
* **Containers Seguros:** Imagens Docker compiladas via *Multi-Stage Build* gerando executáveis estáticos executados sob
  usuários não-root em containers minimalistas.