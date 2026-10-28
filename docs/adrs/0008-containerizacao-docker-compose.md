# 0008 - Containerização com Docker Compose e Multi-Stage Builds em Go

## Status

Aceito.

## Data

2026-10-28

## Contexto e Problema

O projeto é composto por múltiplos serviços e dependências de infraestrutura (Frontend Angular, 3 Microsserviços em Go,
PostgreSQL, RabbitMQ e OTel Collector). É necessário garantir que qualquer avaliador consiga subir o ambiente funcional
com um único comando durante a apresentação.

## Fatores de Decisão

* Portabilidade e reprodução exata do ambiente de execução sem exigir a instalação local do Go ou Node.js na máquina do
  avaliador.
* Uso de Docker Multi-Stage Build para compilar o código Go e gerar imagens finais mínimas (baseadas em `alpine` ou
  `scratch`).
* Requisito de apresentação ao vivo: execução via `docker compose up --build -d`.

## Opções Consideradas

* **Opção 1:** Execução nativa executando `go run main.go` em cada pasta.
* **Opção 2:** Containerização com Dockerfiles Multi-Stage Build e orquestração unificada via `compose.yml`.

## Resultado da Decisão

Opção escolhida: "**Opção 2: Containerização via Dockerfiles Multi-Stage e Docker Compose**", isolando o ambiente de build
da imagem final de execução.

### Consequências

* Bom, porque permite compilar e subir toda a solução com apenas `docker compose up --build -d`.
* Bom, porque as imagens Docker finais dos microsserviços em Go ficam extremamente leves (~15MB a 25MB cada).
* Ruim, porque a primeira compilação no Docker pode demorar alguns segundos a mais para baixar os módulos de dependência
  do Go.

### Confirmação

Validação através da execução do comando `docker compose up --build -d` e verificação da saúde dos microsserviços via
rotas de `/health`.

## Prós e Contras das Opções

### Opção 1: Execução Nativa Host

* Bom, porque não exige Docker instalado.
* Ruim, porque causa divergências de versão do Go ou variáveis de ambiente entre diferentes sistemas operacionais.

### Opção 2: Docker Compose + Multi-Stage Build em Go

* Bom, porque garante um binário compilado limpo dentro do container de build sem poluir a imagem final.
* Bom, porque atende estritamente às exigências de entrega da disciplina.
* Neutro, porque necessita de `compose.yml`, `.env` local, healthchecks, volumes e redes segmentadas.
* Bom, porque imagens de infraestrutura usam versões completas e o Compose exige segredos sem versioná-los.
* Bom, porque portas administrativas usam loopback e dependências internas não são publicadas no host.
* Neutro, porque o módulo `shared/` exige que o context de build dos serviços Go seja a raiz do repositório, com um
  `.dockerignore` global limitando o contexto enviado ao daemon.
