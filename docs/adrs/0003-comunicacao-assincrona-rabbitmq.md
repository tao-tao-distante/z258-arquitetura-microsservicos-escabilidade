# 0003 - Comunicação Assíncrona via RabbitMQ com Tratamento de DLQ

## Context and Problem Statement

Quando uma nova tarefa é criada ou atribuída, notificações devem ser geradas. Processar o envio de notificações de forma
síncrona dentro da requisição HTTP do `service-task` degradaria os tempos de resposta e desrespeitaria os SLOs de
latência definidos ($p_{95} \le 150\text{ms}$).

## Decision Drivers

* Cumprimento dos SLOs de latência no endpoint de criação de tarefas.
* Desacoplamento do serviço de notificações (`service-notification`).

[//]: #TODO (* **[Para aprovacao em: Peer Review]:** Tratamento de mensagens que falham no processamento via Dead Letter Queues &#40;DLQ&#41;.)

## Considered Options

* **Opção 1:** Chamada HTTP REST síncrona entre `service-task` e `service-notification`.
* **Opção 2:** Mensageria assíncrona orientada a eventos utilizando RabbitMQ (driver Go `amqp091-go`) com DLQ (Dead
  Letter Queue).

## Decision Outcome

Chosen option: "**Opção 2: Mensageria assíncrona via RabbitMQ com driver Go e DLQ**", pois permite que o `service-task`
responda ao usuário imediatamente após salvar no banco, publicando o evento na fila para consumo assíncrono.

### Consequences

* Good, porque garante baixíssima latência na resposta HTTP de criação de tarefas em Go.
* Good, porque o envio de notificações pode falhar ou ficar off-line temporariamente sem perda de dados.

[//]: # (* Good **[Para aprovacao em: Peer Review]**, porque mensagens com erro são movidas para DLQ sem travar o processamento da)

[//]: # (  fila principal.)

* Bad, porque adiciona um componente infraestrutural (broker AMQP) para manter e monitorar.

### Confirmation

Será validado simulando a queda do contêiner `service-notification`, criando tarefas via API, e confirmando que as
mensagens permanecem retidas na fila do RabbitMQ e são reprocessadas após o serviço subir novamente.

## Pros and Cons of the Options

### Opção 1: Chamada HTTP REST Síncrona

* Good, porque é simples de implementar sem necessidade de brokers.
* Bad, porque qualquer lentidão no envio de notificações impacta diretamente o usuário final.
* Bad, porque se o serviço de notificações cair, o disparo da tarefa resulta em erro HTTP 500.

### Opção 2: Mensageria Assíncrona via RabbitMQ (Com DLQ)

* Good, because desacopla o tempo de resposta do endpoint da execução do efeito colateral (notificação).
* Good, porque oferece resiliência com mecanismos de retry e Dead Letter Queue.
* Neutral, porque exige serialização e propagação do `correlation_id` nos cabeçalhos AMQP (`amqp.Table`).
* Bad, porque requer gerenciamento de concorrência e idempotência nos consumidores em Go.
