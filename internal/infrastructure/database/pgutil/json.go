package pgutil

import "encoding/json"

func ToJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func FromJSON(data []byte, dest any) error {
	return json.Unmarshal(data, dest)
}
