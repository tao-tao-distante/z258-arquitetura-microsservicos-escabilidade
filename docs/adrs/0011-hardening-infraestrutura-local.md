# 0011 - Hardening da infraestrutura local

## Status

Substituído parcialmente pelo ADR-0014 para a exposição das APIs e o papel do NGINX.

## Data

2026-10-28

## Nota de atualização

Somente o NGINX publica uma porta em todas as interfaces, `0.0.0.0:8081`. As APIs não publicam portas no host. Grafana
em `127.0.0.1:3000` e RabbitMQ Management em `127.0.0.1:15672` são as únicas portas administrativas publicadas.

## Contexto

A stack local usava tags flutuantes, credenciais conhecidas, rede implícita única, portas desnecessariamente públicas e
dados não persistentes para mensageria/observabilidade. Grafana também exigia configuração manual de datasources.

## Decisão

- Fixar versões completas das imagens declaradas no Compose.
- Exigir segredos por interpolação `${VAR:?mensagem}` e fornecer apenas `.env.example` sem credenciais utilizáveis.
- Substituir RabbitMQ `guest` por usuário configurável.
- Publicar o frontend em todas as interfaces, não publicar as APIs e vincular consoles administrativos a `127.0.0.1`.
- Não publicar bancos, AMQP ou endpoints de telemetria.
- Separar redes `edge`, `identity-data`, `task-data`, `messaging` e `telemetry`; marcar redes de dependências como
  internas.
- Ligar o Grafana também à rede bridge `admin`, pois Docker não publica portas de containers conectados apenas a redes
  `internal`; a porta `3000` continua restrita a `127.0.0.1`.
- Persistir dados de PostgreSQL, RabbitMQ, Tempo, Prometheus e Grafana em volumes nomeados.
- Aplicar `restart: unless-stopped`, healthchecks e dependência por saúde.
- Provisionar Prometheus e Tempo como datasources Grafana.
- Usar Tempo monolítico 3.1.1 com backend local e retenção de 24 horas para desenvolvimento.

## Consequências

- Bom: builds locais ficam reproduzíveis no nível das imagens de infraestrutura do Compose.
- Bom: falhas de configuração e ausência de segredos aparecem antes da inicialização.
- Bom: menor superfície de rede entre serviços e no host.
- Bom: reinícios não apagam estado operacional por padrão.
- Ruim: troca de senha de banco/RabbitMQ após criação de volumes exige rotação coordenada ou recriação dos volumes.
- Ruim: backend local do Tempo é adequado apenas para uma instância e desenvolvimento.
- Ruim: versões e correções de segurança precisam de atualização explícita.

## Confirmação

Executar as validações do README, inspecionar `docker compose config` e confirmar saúde com `docker compose ps`. Verificar
que somente `8081` aceita conexões externas, que as APIs não possuem bind no host e que `3000` e `15672` aceitam
conexões somente por `127.0.0.1`.
