# Glossário de domínio

Este glossário fixa os termos dos domínios de identidade, tarefas e notificações. Ele não descreve tecnologia, transporte ou persistência.

## Identidade

### Usuário

Pessoa que possui uma identidade no sistema e gerencia as próprias tarefas.

### Perfil

Dados pelos quais um usuário se apresenta no sistema. O perfil não inclui suas tarefas nem seu histórico de alterações.

## Tarefas

### Tarefa

Unidade de trabalho pertencente a um usuário. Uma tarefa reúne título, descrição, estado, prioridade e prazo opcional.

### Estado da tarefa

Etapa atual da tarefa no fluxo de trabalho: a fazer, em andamento ou concluída.

Evitar: status da aplicação, situação genérica.

### Prioridade

Importância relativa da tarefa: baixa, média, alta ou urgente. A prioridade não altera o estado nem o prazo.

### Prazo

Data de calendário até a qual a tarefa deve ser concluída. O prazo não representa um horário nem um instante em um fuso específico.

### Tarefa atrasada

Tarefa não concluída cujo prazo é anterior à data corrente.

### Posição

Ordem relativa da tarefa entre outras tarefas do mesmo estado.

### Histórico da tarefa

Sequência cronológica das criações, edições e mudanças de estado relevantes de uma tarefa.

### Alteração da tarefa

Mudança registrada em um ou mais dados da tarefa. Uma simples mudança de posição dentro do mesmo estado não constitui alteração de histórico.

## Eventos

### Evento de criação de tarefa

Fato que informa que uma tarefa foi criada. O evento não garante o envio de e-mail, push ou webhook.
