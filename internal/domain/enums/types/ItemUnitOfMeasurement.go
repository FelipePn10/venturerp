package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type TypeUnitOfMeasurementItem string

const (
	MM         TypeUnitOfMeasurementItem = "MM"
	CM         TypeUnitOfMeasurementItem = "CM"
	M          TypeUnitOfMeasurementItem = "M"
	IN         TypeUnitOfMeasurementItem = "IN"
	KG         TypeUnitOfMeasurementItem = "KG"
	M2         TypeUnitOfMeasurementItem = "M2"
	M3         TypeUnitOfMeasurementItem = "M3"
	UN         TypeUnitOfMeasurementItem = "UN"
	MICROMETRO TypeUnitOfMeasurementItem = "MICROMETRO"
	TONELADA   TypeUnitOfMeasurementItem = "TONELADA"
	L          TypeUnitOfMeasurementItem = "L"
	CX         TypeUnitOfMeasurementItem = "CX"
	PC         TypeUnitOfMeasurementItem = "PC"
	GL         TypeUnitOfMeasurementItem = "GL"
	PAR        TypeUnitOfMeasurementItem = "PAR"
)

func (t TypeUnitOfMeasurementItem) String() string {
	return string(t)
}

func (t TypeUnitOfMeasurementItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}

func (t *TypeUnitOfMeasurementItem) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	tmp := TypeUnitOfMeasurementItem(s)

	if !tmp.IsValid() {
		return fmt.Errorf("invalid TypeUnitOfMeasurementItem: %s", s)
	}

	*t = tmp
	return nil
}

func (t TypeUnitOfMeasurementItem) Value() (driver.Value, error) {
	if !t.IsValid() {
		return nil, fmt.Errorf("invalid TypeUnitOfMeasurementItem: %s", t)
	}
	return string(t), nil
}

func (t *TypeUnitOfMeasurementItem) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("null value for TypeUnitOfMeasurementItem")
	}

	var str string

	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into TypeUnitOfMeasurementItem", value)
	}

	tmp := TypeUnitOfMeasurementItem(str)

	if !tmp.IsValid() {
		return fmt.Errorf("invalid TypeUnitOfMeasurementItem from DB: %s", str)
	}

	*t = tmp
	return nil
}

func (t TypeUnitOfMeasurementItem) IsValid() bool {
	switch t {
	case MM, CM, M, IN, KG, M2, M3, UN, MICROMETRO, TONELADA, L, CX, PC, GL, PAR:
		return true
	default:
		return false
	}
}
