package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
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
		return nil, domainerr.NewValidation(fmt.Sprintf("unidade de medida do item inválida: %s", t))
	}
	return string(t), nil
}

func (t *TypeUnitOfMeasurementItem) Scan(value interface{}) error {
	if value == nil {
		return domainerr.NewValidation("a unidade de medida do item não pode ficar vazia")
	}

	var str string

	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return domainerr.NewValidation(fmt.Sprintf("não foi possível interpretar %T como unidade de medida do item", value))
	}

	tmp := TypeUnitOfMeasurementItem(str)

	if !tmp.IsValid() {
		return domainerr.NewValidation(fmt.Sprintf("unidade de medida do item inválida no banco: %s", str))
	}

	*t = tmp
	return nil
}

// IsValid confere só o FORMATO do código (1 a 10 letras maiúsculas ou dígitos).
//
// A unidade deixou de ser uma lista fechada de 15 valores: cada empresa mantém
// o seu catálogo (units_of_measure, migração 474) e pode cadastrar BARRA,
// CHAPA, ROLO… Se a unidade EXISTE no catálogo da empresa é uma pergunta que
// precisa do banco — quem grava (cadastro de item, estrutura, conversão)
// confere pelo catálogo, e o banco garante com FK na unidade de estoque.
func (t TypeUnitOfMeasurementItem) IsValid() bool {
	return CodigoDeUnidadeValido(string(t))
}

// CodigoDeUnidadeValido: 1 a 10 caracteres, só A-Z e 0-9.
func CodigoDeUnidadeValido(s string) bool {
	if len(s) == 0 || len(s) > 10 {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// ValidUnitsOfMeasurement lista as unidades que todo catálogo traz de fábrica,
// para a mensagem de erro dar exemplos do que se pode informar.
func ValidUnitsOfMeasurement() []string {
	return []string{
		string(MM), string(CM), string(M), string(IN), string(KG), string(M2), string(M3),
		string(UN), string(MICROMETRO), string(TONELADA), string(L), string(CX), string(PC),
		string(GL), string(PAR),
	}
}
