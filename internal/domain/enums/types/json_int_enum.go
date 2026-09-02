package types

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func unmarshalStringOrIntEnum(data []byte, enumName string, values map[string]int) (int, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return 0, fmt.Errorf("valor vazio para %s", enumName)
	}

	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return 0, err
		}
		parsed, ok := values[value]
		if !ok {
			return 0, fmt.Errorf("valor inválido para %s: %s", enumName, value)
		}
		return parsed, nil
	}

	var value int
	if err := json.Unmarshal(data, &value); err != nil {
		return 0, fmt.Errorf("valor inválido para %s: %w", enumName, err)
	}
	for _, candidate := range values {
		if candidate == value {
			return value, nil
		}
	}
	return 0, fmt.Errorf("valor inválido para %s: %d", enumName, value)
}
