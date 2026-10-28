# 0011 - Hardening da infraestrutura local

## Status

Aceito.

## Data

2026-10-28

## Contexto

A stack local usava tags flutuantes, credenciais conhecidas, rede implícita única, portas desnecessariamente públicas e
dados não persistentes para mensageria/observabilidade. Grafana também exigia configuração manual de datasources.

## Decisão

- Fixar versões completas das imagens declaradas no Compose.
- Exigir segredos por interpolação `${VAR:?mensagem}` e fornecer apenas `.env.example` sem credenciais utilizáveis.
- Substituir RabbitMQ `guest` por usuário configurável.
- Publicar o frontend em todas as interfaces e vincular APIs/consoles de desenvolvimento a `127.0.0.1`.
- Não publicar bancos, AMQP ou endpoints de telemetria.
- Separar redes `edge`, `identity-data`, `task-data`, `messaging` e `telemetry`; marcar redes de dependências como
  internas.
- Persistir dados de PostgreSQL, RabbitMQ, Tempo, Prometheus e Grafana em volumes nomeados.
- Aplicar `restart: unless-stopped`, healthchecks e dependência por saúde.
- Provisionar Prometheus e Tempo como datasources Grafana.
- Usar Tempo monolítico 3.1.1 com backend local e retenção de 24 horas para desenvolvimento.

## Consequências

- Good: builds locais ficam reproduzíveis no nível das imagens de infraestrutura do Compose.
- Good: falhas de configuração e ausência de segredos aparecem antes da inicialização.
- Good: menor superfície de rede entre serviços e no host.
- Good: reinícios não apagam estado operacional por padrão.
- Bad: troca de senha de banco/RabbitMQ após criação de volumes exige rotação coordenada ou recriação dos volumes.
- Bad: backend local do Tempo é adequado apenas para uma instância e desenvolvimento.
- Bad: versões e correções de segurança precisam de atualização explícita.

## Confirmação

Executar as validações do README, inspecionar `docker compose config` e confirmar saúde com `docker compose ps`. Verificar
que somente `8081` aceita conexões externas e que `8082`-`8084`, `3000` e `15672` estão em `127.0.0.1`.
