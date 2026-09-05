package types

import (
	"bytes"
	"encoding/json"
	"sort"
)

// unmarshalStringOrIntEnum lê um enum que chega como texto ou número.
//
// `field` é o nome do campo como o usuário o vê na tela — é ele que aparece na
// mensagem de erro, junto da lista de valores aceitos. Devolver
// InvalidValueError permite à camada HTTP responder 422 dizendo qual campo está
// errado, em vez do antigo "invalid request body" genérico.
func unmarshalStringOrIntEnum(data []byte, field string, values map[string]int) (int, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return 0, NewInvalidValue(field, "", aceitos(values)...)
	}

	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return 0, err
		}
		parsed, ok := values[value]
		if !ok {
			return 0, NewInvalidValue(field, value, aceitos(values)...)
		}
		return parsed, nil
	}

	var value int
	if err := json.Unmarshal(data, &value); err != nil {
		return 0, NewInvalidValue(field, string(data), aceitos(values)...)
	}
	for _, candidate := range values {
		if candidate == value {
			return value, nil
		}
	}
	return 0, NewInvalidValue(field, string(data), aceitos(values)...)
}

// aceitos devolve os rótulos aceitos em ordem estável, para a mensagem não
// mudar de uma requisição para a outra.
func aceitos(values map[string]int) []string {
	out := make([]string, 0, len(values))
	for k := range values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
