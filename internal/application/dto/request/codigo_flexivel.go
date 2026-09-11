package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// CodigoFlexivel é um código de cadastro que o banco guarda como texto, mas que
// clientes antigos enviavam como número JSON.
//
// Os almoxarifados de transferência e de assistência técnica do item têm chave
// estrangeira para warehouse(code), e em produção esses códigos são
// alfanuméricos (ALM-PA, ALM-EXP). O DTO era *int64: o código alfanumérico não
// cabia, a tela o convertia para número e ele sumia sem erro. Aceitar os dois
// formatos mantém funcionando o cliente mais antigo ainda suportado.
//
// Vazio ou null significam "não informado" — ao contrário de TextCode, que é
// para código de item e recusa vazio.
type CodigoFlexivel string

func (c *CodigoFlexivel) UnmarshalJSON(data []byte) error {
	raw := bytes.TrimSpace(data)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		*c = ""
		return nil
	}
	if raw[0] == '"' {
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return errors.New("o código informado não é um texto válido")
		}
		*c = CodigoFlexivel(strings.TrimSpace(v))
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err != nil {
		return errors.New("o código deve ser informado como texto")
	}
	*c = CodigoFlexivel(n.String())
	return nil
}

// Ptr devolve o código como *string, ou nil quando não foi informado.
func (c *CodigoFlexivel) Ptr() *string {
	if c == nil || *c == "" {
		return nil
	}
	v := string(*c)
	return &v
}
