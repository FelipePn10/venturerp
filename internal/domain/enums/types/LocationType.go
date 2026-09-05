package types

import "encoding/json"

type TypeLocation int

const (
	INTERNO TypeLocation = iota
	EXTERNO
	ASSISTENCIA
	REJEICAO
	INSPECAO
	RESERVA
	TRANSITO
	ESPECIAL
	EXPEDICAO
	ASSISTENCIA_TECNICA
)

func (t TypeLocation) String() string {
	switch t {
	case INTERNO:
		return "INTERNO"
	case EXTERNO:
		return "EXTERNO"
	case ASSISTENCIA:
		return "ASSISTÊNCIA"
	case REJEICAO:
		return "REJEIÇÃO"
	case INSPECAO:
		return "INSPEÇÃO"
	case RESERVA:
		return "RESERVA"
	case TRANSITO:
		return "TRÂNSITO"
	case ESPECIAL:
		return "ESPECIAL"
	case EXPEDICAO:
		return "EXPEDICAO"
	case ASSISTENCIA_TECNICA:
		return "ASSISTENCIA_TECNICA"
	default:
		return "NENHUM"
	}
}

func (t TypeLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TypeLocation) UnmarshalJSON(data []byte) error {
	value, err := unmarshalStringOrIntEnum(data, "Tipo de localização", map[string]int{
		"INTERNO": int(INTERNO), "EXTERNO": int(EXTERNO), "ASSISTÊNCIA": int(ASSISTENCIA),
		"REJEIÇÃO": int(REJEICAO), "INSPEÇÃO": int(INSPECAO), "RESERVA": int(RESERVA),
		"TRÂNSITO": int(TRANSITO), "TRANSITO": int(TRANSITO), "ESPECIAL": int(ESPECIAL),
		"ASSISTENCIA": int(ASSISTENCIA), "INSPECAO": int(INSPECAO), "REJEICAO": int(REJEICAO),
		"EXPEDICAO": int(EXPEDICAO), "ASSISTENCIA_TECNICA": int(ASSISTENCIA_TECNICA),
	})
	if err != nil {
		return err
	}
	*t = TypeLocation(value)
	return nil
}

func (t TypeLocation) IsValid() bool { return t >= INTERNO && t <= ASSISTENCIA_TECNICA }
