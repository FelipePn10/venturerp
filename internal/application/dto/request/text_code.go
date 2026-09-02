package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
)

var integerJSONCode = regexp.MustCompile(`^[0-9]+$`)
var legacyNumericTextCodeCount atomic.Uint64

// TextCode is the canonical JSON contract for business codes. During the
// compatibility window it also accepts a positive JSON integer and normalizes
// it to text. Deprecation telemetry is emitted by the HTTP boundary, which has
// the operation context required to produce a useful signal.
type TextCode string

func (c *TextCode) UnmarshalJSON(data []byte) error {
	raw := bytes.TrimSpace(data)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return fmt.Errorf("o código do item deve ser informado como texto")
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("o código do item deve ser um texto válido")
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("o código do item é obrigatório")
		}
		*c = TextCode(value)
		return nil
	}
	if !integerJSONCode.Match(raw) || bytes.Equal(raw, []byte("0")) {
		return fmt.Errorf("o código do item deve ser texto ou número inteiro positivo")
	}
	legacyNumericTextCodeCount.Add(1)
	*c = TextCode(string(raw))
	return nil
}

func (c TextCode) String() string { return string(c) }

// LegacyNumericTextCodeCount exposes deprecation telemetry for the temporary
// numeric JSON compatibility path. Monitoring can sample this monotonic value
// until clients have migrated entirely to textual business codes.
func LegacyNumericTextCodeCount() uint64 { return legacyNumericTextCodeCount.Load() }
