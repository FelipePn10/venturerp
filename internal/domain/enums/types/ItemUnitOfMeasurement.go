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
		return NewInvalidValue("Unidade de medida do item", s, ValidUnitsOfMeasurement()...)
	}

	*t = tmp
	return nil
}

func (t TypeUnitOfMeasurementItem) Value() (driver.Value, error) {
	if !t.IsValid() {
		return nil, fmt.Errorf("unidade de medida do item inválida: %s", t)
	}
	return string(t), nil
}

func (t *TypeUnitOfMeasurementItem) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("a unidade de medida do item não pode ficar vazia")
	}

	var str string

	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("não foi possível interpretar %T como unidade de medida do item", value)
	}

	tmp := TypeUnitOfMeasurementItem(str)

	if !tmp.IsValid() {
		return fmt.Errorf("unidade de medida do item inválida no banco: %s", str)
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

// ValidUnitsOfMeasurement lista as unidades aceitas, para a mensagem de erro
// dizer ao usuário o que ele pode informar.
func ValidUnitsOfMeasurement() []string {
	return []string{
		string(MM), string(CM), string(M), string(IN), string(KG), string(M2), string(M3),
		string(UN), string(MICROMETRO), string(TONELADA), string(L), string(CX), string(PC),
		string(GL), string(PAR),
	}
}
