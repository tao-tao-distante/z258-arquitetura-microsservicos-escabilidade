# 0008 - Containerização com Docker Compose e Multi-Stage Builds em Go

## Context and Problem Statement

O projeto é composto por múltiplos serviços e dependências de infraestrutura (Frontend Angular, 3 Microsserviços em Go,
PostgreSQL, RabbitMQ e OTel Collector). É necessário garantir que qualquer avaliador consiga subir o ambiente funcional
com um único comando durante a apresentação.

## Decision Drivers

* Portabilidade e reprodução exata do ambiente de execução sem exigir a instalação local do Go ou Node.js na máquina do
  avaliador.
* Uso de Docker Multi-Stage Build para compilar o código Go e gerar imagens finais mínimas (baseadas em `alpine` ou
  `scratch`).
* Requisito de apresentação ao vivo: execução via `docker compose up --build -d`.

## Considered Options

* **Opção 1:** Execução nativa executando `go run main.go` em cada pasta.
* **Opção 2:** Containerização com Dockerfiles Multi-Stage Build e orquestração unificada via `compose.yml`.

## Decision Outcome

Chosen option: "**Opção 2: Containerização via Dockerfiles Multi-Stage e Docker Compose**", isolando o ambiente de build
da imagem final de execução.

### Consequences

* Good, porque permite compilar e subir toda a solução com apenas `docker compose up --build -d`.
* Good, porque as imagens Docker finais dos microsserviços em Go ficam extremamente leves (~15MB a 25MB cada).
* Bad, porque a primeira compilação no Docker pode demorar alguns segundos a mais para baixar os módulos de dependência
  do Go.

### Confirmation

Validação através da execução do comando `docker compose up --build -d` e verificação da saúde dos microsserviços via
rotas de `/health`.

## Pros and Cons of the Options

### Opção 1: Execução Nativa Host

* Good, porque não exige Docker instalado.
* Bad, porque causa divergências de versão do Go ou variáveis de ambiente entre diferentes sistemas operacionais.

### Opção 2: Docker Compose + Multi-Stage Build em Go

* Good, porque garante um binário compilado limpo dentro do container de build sem poluir a imagem final.
* Good, porque atende estritamente às exigências de entrega da disciplina.
* Neutral, porque necessita de um arquivo `compose.yml` bem estruturado com políticas de dependência (`depends_on`).