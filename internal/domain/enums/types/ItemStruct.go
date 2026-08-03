package types

import "encoding/json"

type TypeStructItem int

const (
	INDUSTRIAL TypeStructItem = iota // Itens do qual o MRP gera ordem e controla estoque
	COMERCIAL                        // Item pronto para a venda
)

func (t TypeStructItem) String() string {
	switch t {
	case INDUSTRIAL:
		return "INDUSTRIAL"
	case COMERCIAL:
		return "COMERCIAL"

	default:
		return "Desconhecido"
	}
}

func (t TypeStructItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TypeStructItem) UnmarshalJSON(data []byte) error {
	value, err := unmarshalStringOrIntEnum(data, "TypeStructItem", map[string]int{"INDUSTRIAL": int(INDUSTRIAL), "COMERCIAL": int(COMERCIAL)})
	if err != nil {
		return err
	}
	*t = TypeStructItem(value)
	return nil
}

func (t TypeStructItem) IsValid() bool { return t == INDUSTRIAL || t == COMERCIAL }
