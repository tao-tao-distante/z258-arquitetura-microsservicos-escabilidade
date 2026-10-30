package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// DateLayout é o formato de data de calendário usado na API e no banco.
const DateLayout = "2006-01-02"

// Date representa uma data de calendário sem horário, normalizada para meia-noite UTC.
type Date struct {
	time.Time
}

// NewDate trunca o instante informado para a data de calendário em UTC.
func NewDate(t time.Time) Date {
	y, m, d := t.UTC().Date()
	return Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

// ParseDate interpreta uma data no formato YYYY-MM-DD.
func ParseDate(raw string) (Date, error) {
	t, err := time.Parse(DateLayout, raw)
	if err != nil {
		return Date{}, fmt.Errorf("parse date: %w", err)
	}
	return Date{Time: t}, nil
}

// String devolve a data no formato YYYY-MM-DD.
func (d Date) String() string {
	return d.Format(DateLayout)
}

// MarshalJSON serializa a data como "YYYY-MM-DD".
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON aceita somente strings no formato YYYY-MM-DD.
func (d *Date) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode date: %w", err)
	}
	parsed, err := ParseDate(raw)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// SameDate compara duas datas opcionais.
func SameDate(a, b *Date) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(b.Time)
}
