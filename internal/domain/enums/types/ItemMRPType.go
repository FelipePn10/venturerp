package types

import "encoding/json"

type TypeMRPItem int

const (
	NORMAL_MRP TypeMRPItem = iota
	PROJETO
)

func (t TypeMRPItem) String() string {
	switch t {
	case NORMAL_MRP:
		return "NORMAL_MRP"
	case PROJETO:
		return "PROJETO"

	default:
		return "Desconhecido"
	}
}

func (t TypeMRPItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TypeMRPItem) UnmarshalJSON(data []byte) error {
	value, err := unmarshalStringOrIntEnum(data, "TypeMRPItem", map[string]int{"NORMAL_MRP": int(NORMAL_MRP), "PROJETO": int(PROJETO)})
	if err != nil {
		return err
	}
	*t = TypeMRPItem(value)
	return nil
}

func (t TypeMRPItem) IsValid() bool { return t == NORMAL_MRP || t == PROJETO }
