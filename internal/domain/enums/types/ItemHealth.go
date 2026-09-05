package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Health string

const (
	ACTIVE   Health = "ATIVO"
	INACTIVE Health = "INATIVO"
	GHOST    Health = "FANTASMA"
)

func (s Health) String() string {
	return string(s)
}

func (s Health) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (s *Health) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	tmp := Health(str)

	if !tmp.IsValid() {
		return NewInvalidValue("Situação do componente", str, ValidHealths()...)
	}

	*s = tmp
	return nil
}

func (s Health) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("situação do componente inválida: %s", s)
	}
	return string(s), nil
}

func (s *Health) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("a situação do componente não pode ficar vazia")
	}

	var str string

	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("não foi possível interpretar %T como situação do componente", value)
	}

	tmp := Health(str)

	if !tmp.IsValid() {
		return fmt.Errorf("situação do componente inválida no banco: %s", str)
	}

	*s = tmp
	return nil
}

func (t Health) IsValid() bool {
	switch t {
	case ACTIVE, INACTIVE, GHOST:
		return true
	default:
		return false
	}
}

// ValidHealths lista as situações aceitas para o componente.
func ValidHealths() []string {
	return []string{string(ACTIVE), string(INACTIVE), string(GHOST)}
}
