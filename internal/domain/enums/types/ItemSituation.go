package types

import "encoding/json"

type TypeSituationItem int

const (
	LINHA TypeSituationItem = iota
	PROMOCAO
)

func (t TypeSituationItem) String() string {
	switch t {
	case LINHA:
		return "LINHA"
	case PROMOCAO:
		return "PROMOCAO"

	default:
		return "Desconhecido"
	}
}

func (t TypeSituationItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TypeSituationItem) UnmarshalJSON(data []byte) error {
	value, err := unmarshalStringOrIntEnum(data, "TypeSituationItem", map[string]int{"LINHA": int(LINHA), "PROMOCAO": int(PROMOCAO)})
	if err != nil {
		return err
	}
	*t = TypeSituationItem(value)
	return nil
}

func (t TypeSituationItem) IsValid() bool { return t == LINHA || t == PROMOCAO }
