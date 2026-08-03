package types

import "encoding/json"

type TypeWarehouse int

const (
	LINHA_DE_PRODUCAO TypeWarehouse = iota
	NORMAL
)

func (t TypeWarehouse) String() string {
	switch t {
	case LINHA_DE_PRODUCAO:
		return "LINHA DE PRODUÇÃO"
	case NORMAL:
		return "NORMAL"

	default:
		return "Desconhecido"
	}
}

func (t TypeWarehouse) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TypeWarehouse) UnmarshalJSON(data []byte) error {
	value, err := unmarshalStringOrIntEnum(data, "TypeWarehouse", map[string]int{"LINHA DE PRODUÇÃO": int(LINHA_DE_PRODUCAO), "NORMAL": int(NORMAL)})
	if err != nil {
		return err
	}
	*t = TypeWarehouse(value)
	return nil
}

func (t TypeWarehouse) IsValid() bool { return t == LINHA_DE_PRODUCAO || t == NORMAL }
