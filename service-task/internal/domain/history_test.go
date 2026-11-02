package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func s(v string) *string { return &v }

func mustDate(t *testing.T, raw string) *Date {
	t.Helper()
	d, err := ParseDate(raw)
	require.NoError(t, err)
	return &d
}

func TestDiff(t *testing.T) {
	base := Task{Title: "t", Description: "d", Status: "todo", Priority: PriorityMedium}
	nov15 := mustDate(t, "2026-11-15")
	nov20 := mustDate(t, "2026-11-20")
	sameNov15 := mustDate(t, "2026-11-15")

	tests := []struct {
		name   string
		before Task
		after  func(Task) Task
		want   []FieldChange
	}{
		{name: "no changes", before: base, after: func(t Task) Task { return t }, want: nil},
		{
			name:   "title",
			before: base,
			after:  func(t Task) Task { t.Title = "novo"; return t },
			want:   []FieldChange{{Field: FieldTitle, From: s("t"), To: s("novo")}},
		},
		{
			name:   "description records only the field",
			before: base,
			after:  func(t Task) Task { t.Description = "outra"; return t },
			want:   []FieldChange{{Field: FieldDescription}},
		},
		{
			name:   "status and priority",
			before: base,
			after:  func(t Task) Task { t.Status = "doing"; t.Priority = PriorityHigh; return t },
			want: []FieldChange{
				{Field: FieldStatus, From: s("todo"), To: s("doing")},
				{Field: FieldPriority, From: s(PriorityMedium), To: s(PriorityHigh)},
			},
		},
		{
			name:   "due date added",
			before: base,
			after:  func(t Task) Task { t.DueDate = nov15; return t },
			want:   []FieldChange{{Field: FieldDueDate, To: s("2026-11-15")}},
		},
		{
			name:   "due date changed",
			before: Task{DueDate: nov15},
			after:  func(t Task) Task { t.DueDate = nov20; return t },
			want:   []FieldChange{{Field: FieldDueDate, From: s("2026-11-15"), To: s("2026-11-20")}},
		},
		{
			name:   "due date removed",
			before: Task{DueDate: nov15},
			after:  func(t Task) Task { t.DueDate = nil; return t },
			want:   []FieldChange{{Field: FieldDueDate, From: s("2026-11-15")}},
		},
		{
			name:   "same due date in another pointer",
			before: Task{DueDate: nov15},
			after:  func(t Task) Task { t.DueDate = sameNov15; return t },
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Diff(tt.before, tt.after(tt.before)))
		})
	}
}

func TestInitialChanges(t *testing.T) {
	tests := []struct {
		name string
		task Task
		want []FieldChange
	}{
		{
			name: "without due date",
			task: Task{Status: "todo", Priority: PriorityLow},
			want: []FieldChange{
				{Field: FieldStatus, To: s("todo")},
				{Field: FieldPriority, To: s(PriorityLow)},
			},
		},
		{
			name: "with due date",
			task: Task{Status: "doing", Priority: PriorityUrgent, DueDate: mustDate(t, "2026-12-01")},
			want: []FieldChange{
				{Field: FieldStatus, To: s("doing")},
				{Field: FieldPriority, To: s(PriorityUrgent)},
				{Field: FieldDueDate, To: s("2026-12-01")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, InitialChanges(tt.task))
		})
	}
}

func TestStatusChange(t *testing.T) {
	require.Equal(t, FieldChange{Field: FieldStatus, From: s("todo"), To: s("done")}, StatusChange("todo", "done"))
}

func TestDateJSON(t *testing.T) {
	var payload struct {
		DueDate *Date `json:"due_date"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"due_date":"2026-11-15"}`), &payload))
	require.Equal(t, "2026-11-15", payload.DueDate.String())

	out, err := json.Marshal(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"due_date":"2026-11-15"}`, string(out))

	require.NoError(t, json.Unmarshal([]byte(`{"due_date":null}`), &payload))
	require.Nil(t, payload.DueDate)
}

func TestDateJSONRejectsInvalidValues(t *testing.T) {
	for _, raw := range []string{`{"due_date":"15/11/2026"}`, `{"due_date":"2026-11-15T10:00:00Z"}`, `{"due_date":20261115}`} {
		var payload struct {
			DueDate *Date `json:"due_date"`
		}
		require.Error(t, json.Unmarshal([]byte(raw), &payload), raw)
	}
}

func TestNewDateTruncatesToUTCDay(t *testing.T) {
	brt := time.FixedZone("BRT", -3*60*60)
	got := NewDate(time.Date(2026, 10, 30, 22, 30, 0, 0, brt))
	require.Equal(t, "2026-10-31", got.String())
	require.Equal(t, time.UTC, got.Location())
}

func TestSameDate(t *testing.T) {
	d := mustDate(t, "2026-11-15")
	require.True(t, SameDate(nil, nil))
	require.False(t, SameDate(d, nil))
	require.False(t, SameDate(nil, d))
	require.True(t, SameDate(d, mustDate(t, "2026-11-15")))
}
