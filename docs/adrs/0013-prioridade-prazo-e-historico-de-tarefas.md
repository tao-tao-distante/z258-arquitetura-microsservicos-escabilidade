# 0013 - Prioridade, prazo e histórico de alterações de tarefas

## Status

Aceito.

## Data

2026-10-30

## Contexto e Problema

As tarefas tinham apenas título, descrição, status e posição no quadro. Não era possível indicar o que é mais
importante, registrar até quando algo precisa ser feito nem saber quem mudou o quê e quando: o status anterior se perdia
em cada `PUT` ou `PATCH /move`. Usuários precisavam de mais organização para priorizar o trabalho e de rastreabilidade
das alterações dentro da própria tarefa.

## Fatores de Decisão

- Prioridade e prazo devem ser opcionais para não quebrar clientes existentes.
- O histórico deve ser consistente com a tarefa: não pode existir alteração sem registro nem registro sem alteração.
- O histórico não deve duplicar textos longos (descrição até 1 MiB) nem crescer com ruído de reordenação.
- O serviço deve continuar com banco próprio (ADR-0001) e sem nova dependência de infraestrutura.
- Prazo é uma data de calendário; não pode sofrer deslocamento de fuso entre navegador, API e banco.

## Resultado da Decisão

**Prioridade**: coluna `priority TEXT NOT NULL DEFAULT 'medium'` com `CHECK` em `low`, `medium`, `high` e `urgent`.
Vazio na API assume `medium`.

**Prazo**: coluna `due_date DATE` opcional, trafegada como `YYYY-MM-DD` pelo tipo `domain.Date`.

- Na criação, um prazo no passado é rejeitado (`400`).
- Na edição, o prazo só é validado quando muda, para permitir editar tarefas já atrasadas.
- A API não conhece o fuso do cliente. Por isso aceita como piso o dia anterior em UTC, e o frontend valida contra a
  data local exata.
- `GET /tasks?overdue=true` considera atrasadas as tarefas com `due_date < CURRENT_DATE` e status diferente de `done`.

**Histórico**: tabela `task_history` (`task_id` com `ON DELETE CASCADE`, `action`, `changes JSONB`, `correlation_id`,
`changed_at`), gravada **na mesma transação** da alteração, no mesmo padrão do outbox:

| Operação | Ação | Conteúdo |
|---|---|---|
| `POST /tasks` | `created` | status, prioridade e prazo iniciais |
| `PUT /tasks/{id}` | `updated` | somente campos alterados, calculados por `domain.Diff` sobre a linha travada com `FOR UPDATE` |
| `PATCH /tasks/{id}/move` | `moved` | somente quando a coluna muda; reordenar na mesma coluna não gera registro |

- Uma alteração de descrição registra apenas que o campo mudou, sem guardar os valores antigo e novo.
- `GET /tasks/{id}/history` devolve as entradas da mais recente para a mais antiga, com paginação.
- O `correlation_id` é guardado para diagnóstico, mas não aparece na resposta.
- A migração cria uma entrada `created` para as tarefas que já existiam.

**Listagem**: `GET /tasks` aceita `status`, `priority`, `overdue`, `sort` (`position`, `priority`, `created_at`,
`due_date`) e `order`. A ordenação é escolhida em uma lista fechada no código; a entrada do usuário nunca é interpolada
no SQL.

**Frontend**:

- O formulário ganha os campos de prioridade e prazo.
- Lista, quadro e detalhe mostram um badge de prioridade e o indicador de prazo ("Vence hoje", "Atrasada há N dias").
- A listagem filtra por prioridade e atrasadas e ordena no cliente, com o estado guardado na URL. Ela já carrega todas
  as páginas, e as contagens por status dependem do conjunto completo.
- O detalhe da tarefa mostra a linha do tempo do histórico, carregada de forma independente da tarefa.

## Alternativas Consideradas

### Event sourcing da tarefa

Reconstruiria o estado a partir de eventos e daria auditoria completa, mas mudaria todo o modelo de leitura e escrita do
serviço. Rejeitada por custo desproporcional ao problema.

### Publicar eventos de alteração e montar o histórico em outro serviço

Reaproveitaria o RabbitMQ, mas tornaria o histórico eventualmente consistente e exigiria novo consumidor e novo banco.
Além disso, o `service-notification` atual mandaria tipos de evento desconhecidos para a DLQ. Rejeitada nesta etapa; os
eventos `task.updated` e `task.moved` podem ser adicionados depois via outbox.

### Triggers no PostgreSQL

Garantiria o registro mesmo para escritas fora da aplicação, mas esconderia regra de negócio no banco, dificultaria
testar o diff com TDD e não teria acesso ao `correlation_id` do contexto. Rejeitada.

### Guardar a descrição completa no histórico

Daria o diff textual, mas poderia multiplicar até 1 MiB por edição. Rejeitada; o histórico registra apenas que a
descrição mudou.

## Consequências

- Bom: tarefa e histórico são atômicos; não há divergência entre estado e auditoria.
- Bom: filtros e ordenações cobrem prioridade, criação e prazo, com índices `(user_id, priority)` e `(user_id, due_date)`.
- Bom: clientes antigos continuam funcionando em `POST`, porque os campos novos são opcionais.
- Ruim: `PUT` substitui a tarefa inteira; um cliente que omitir `priority` ou `due_date` volta a prioridade para `medium`
  e apaga o prazo.
- Ruim: `Update` passou a abrir transação e travar a linha, com uma consulta a mais por edição.
- Ruim: a folga de um dia na validação do prazo permite, via API direta, gravar "ontem" para clientes à frente de UTC.
- Ruim: o histórico cresce sem limite enquanto a tarefa existir; não há política de retenção.

## Confirmação

1. Rodar `go test -race ./...` em `service-task` e `npm test -- --no-watch` em `frontend`.
2. Subir a stack e criar uma tarefa com prioridade `high` e prazo futuro.
3. Editar a prioridade e a descrição e mover a tarefa para outra coluna.
4. Confirmar em `GET /tasks/{id}/history`, e na tela de detalhe, as entradas `created`, `updated` e `moved`.
5. Confirmar que um prazo no passado devolve `400`.
