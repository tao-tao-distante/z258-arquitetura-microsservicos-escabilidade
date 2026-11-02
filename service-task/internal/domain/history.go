package domain

import "time"

// Ações registradas no histórico de uma tarefa.
const (
	ActionCreated = "created"
	ActionUpdated = "updated"
	ActionMoved   = "moved"
)

// Campos acompanhados pelo histórico.
const (
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldStatus      = "status"
	FieldPriority    = "priority"
	FieldDueDate     = "due_date"
)

// FieldChange descreve a mudança de um campo. A descrição registra apenas que mudou,
// sem valores, para não duplicar textos longos no histórico.
type FieldChange struct {
	Field string  `json:"field"`
	From  *string `json:"from"`
	To    *string `json:"to"`
}

// HistoryEntry representa uma alteração registrada de uma tarefa.
type HistoryEntry struct {
	ID            int64         `json:"id"`
	TaskID        int64         `json:"task_id"`
	Action        string        `json:"action"`
	Changes       []FieldChange `json:"changes"`
	CorrelationID string        `json:"-"`
	ChangedAt     time.Time     `json:"changed_at"`
}

// InitialChanges descreve os valores iniciais relevantes de uma tarefa recém-criada.
func InitialChanges(t Task) []FieldChange {
	changes := []FieldChange{
		{Field: FieldStatus, To: strPtr(t.Status)},
		{Field: FieldPriority, To: strPtr(t.Priority)},
	}
	if t.DueDate != nil {
		changes = append(changes, FieldChange{Field: FieldDueDate, To: strPtr(t.DueDate.String())})
	}
	return changes
}

// Diff lista os campos que mudaram entre o estado anterior e o atual da tarefa.
func Diff(before, after Task) []FieldChange {
	var changes []FieldChange
	if before.Title != after.Title {
		changes = append(changes, FieldChange{Field: FieldTitle, From: strPtr(before.Title), To: strPtr(after.Title)})
	}
	if before.Description != after.Description {
		changes = append(changes, FieldChange{Field: FieldDescription})
	}
	if before.Status != after.Status {
		changes = append(changes, FieldChange{Field: FieldStatus, From: strPtr(before.Status), To: strPtr(after.Status)})
	}
	if before.Priority != after.Priority {
		changes = append(changes, FieldChange{Field: FieldPriority, From: strPtr(before.Priority), To: strPtr(after.Priority)})
	}
	if !SameDate(before.DueDate, after.DueDate) {
		changes = append(changes, FieldChange{Field: FieldDueDate, From: dateStr(before.DueDate), To: dateStr(after.DueDate)})
	}
	return changes
}

// StatusChange descreve a troca de coluna de uma tarefa no quadro.
func StatusChange(from, to string) FieldChange {
	return FieldChange{Field: FieldStatus, From: strPtr(from), To: strPtr(to)}
}

func strPtr(s string) *string { return &s }

func dateStr(d *Date) *string {
	if d == nil {
		return nil
	}
	return strPtr(d.String())
}
