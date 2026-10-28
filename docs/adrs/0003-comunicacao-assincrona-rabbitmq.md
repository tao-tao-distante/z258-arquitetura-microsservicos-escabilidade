# 0003 - Comunicação Assíncrona via RabbitMQ com Tratamento de DLQ

## Context and Problem Statement

Quando uma nova tarefa é criada ou atribuída, notificações devem ser geradas. Processar o envio de notificações de forma
síncrona dentro da requisição HTTP do `service-task` degradaria os tempos de resposta e desrespeitaria os SLOs de
latência definidos ($p_{95} \le 150\text{ms}$).

## Decision Drivers

* Evitar que processamento do consumidor aumente a latência do endpoint de criação.
* Desacoplamento do serviço de notificações (`service-notification`).

## Considered Options

* **Opção 1:** Chamada HTTP REST síncrona entre `service-task` e `service-notification`.
* **Opção 2:** Mensageria assíncrona orientada a eventos utilizando RabbitMQ (driver Go `amqp091-go`) com DLQ (Dead
  Letter Queue).

## Decision Outcome

Chosen option: "**Opção 2: Mensageria assíncrona via RabbitMQ com driver Go e DLQ**", pois permite que o `service-task`
responda ao usuário imediatamente após salvar no banco, publicando o evento na fila para consumo assíncrono.

### Consequences

* Good, porque o processamento do consumidor não ocorre no caminho síncrono da resposta HTTP.
* Good, porque a fila durável retém mensagens enquanto o consumidor estiver indisponível.
* Good, porque o evento é gravado no outbox na mesma transação da tarefa; um dispatcher publica com confirms e
  `mandatory`, com retry/backoff em caso de falha.
* Good, porque mensagens inválidas são movidas para a DLQ sem travar o processamento da fila principal.
* Bad, porque adiciona um componente infraestrutural (broker AMQP) para manter e monitorar.

### Confirmation

Será validado simulando a queda do contêiner `service-notification`, criando tarefas via API, e confirmando que as
mensagens permanecem retidas na fila do RabbitMQ e são reprocessadas após o serviço subir novamente. Também deve ser
verificado que uma falha de publicação mantém o evento pendente no outbox até o retry bem-sucedido.

## Pros and Cons of the Options

### Opção 1: Chamada HTTP REST Síncrona

* Good, porque é simples de implementar sem necessidade de brokers.
* Bad, porque qualquer lentidão no envio de notificações impacta diretamente o usuário final.
* Bad, porque se o serviço de notificações cair, o disparo da tarefa resulta em erro HTTP 500.

### Opção 2: Mensageria Assíncrona via RabbitMQ (Com DLQ)

* Good, because desacopla o tempo de resposta do endpoint da execução do efeito colateral (notificação).
* Good, porque eventos rejeitados com `Nack(requeue=false)` são encaminhados para a Dead Letter Queue.
* Good, porque o outbox transacional evita perda de eventos entre o commit do banco e a publicação.
* Neutral, porque a entrega é at-least-once; consumidores devem deduplicar por `MessageId`/`event_id`.
* Bad, porque o consumidor não faz retry próprio: falhas de Ack/Nack encerram o consumo e dependem do supervisor.
* Neutral, porque exige serialização e propagação do `correlation_id` nos cabeçalhos AMQP (`amqp.Table`).
* Bad, porque requer gerenciamento de concorrência e idempotência nos consumidores em Go.
