package types

import "encoding/json"

type TypeOfUseItem int

const (
	INDUSTRIALIZACAO TypeOfUseItem = iota
	CONSUMO
	IMOBILIZADO
)

func (t TypeOfUseItem) String() string {
	switch t {
	case INDUSTRIALIZACAO:
		return "INDUSTRIALIZACAO"
	case CONSUMO:
		return "CONSUMO"
	case IMOBILIZADO:
		return "IMOBILIZADO"

	default:
		return "Desconhecido"
	}
}

func (t TypeOfUseItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TypeOfUseItem) UnmarshalJSON(data []byte) error {
	value, err := unmarshalStringOrIntEnum(data, "TypeOfUseItem", map[string]int{"INDUSTRIALIZACAO": int(INDUSTRIALIZACAO), "CONSUMO": int(CONSUMO), "IMOBILIZADO": int(IMOBILIZADO)})
	if err != nil {
		return err
	}
	*t = TypeOfUseItem(value)
	return nil
}

func (t TypeOfUseItem) IsValid() bool { return t >= INDUSTRIALIZACAO && t <= IMOBILIZADO }
