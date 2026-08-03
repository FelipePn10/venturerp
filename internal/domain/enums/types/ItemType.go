package types

import "encoding/json"

type TypeItem int

const (
	FABRICADO   TypeItem = iota // Gera ordem de fabricação se tiver roteiro de fabricação e estrutura interna com alguma máteria prima
	COMPRADO                    // Gera ordem de compra
	DE_TERCEIRO                 // Item de terceiro em poder da empresa, nada de ordens
	SERVICO                     // Serviço comercial/fiscal; não gera ordem de material
)

func (s TypeItem) String() string {
	switch s {
	case FABRICADO:
		return "FABRICADO"
	case COMPRADO:
		return "COMPRADO"
	case DE_TERCEIRO:
		return "DE_TERCEIRO"
	case SERVICO:
		return "SERVICO"
	default:
		return "UNKNOWN"
	}
}

func (t TypeItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TypeItem) UnmarshalJSON(data []byte) error {
	value, err := unmarshalStringOrIntEnum(data, "TypeItem", map[string]int{"FABRICADO": int(FABRICADO), "COMPRADO": int(COMPRADO), "DE_TERCEIRO": int(DE_TERCEIRO), "SERVICO": int(SERVICO)})
	if err != nil {
		return err
	}
	*t = TypeItem(value)
	return nil
}

func (t TypeItem) IsValid() bool { return t >= FABRICADO && t <= SERVICO }
