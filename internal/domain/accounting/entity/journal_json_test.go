package entity

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// O lançamento vai direto para a resposta; o valor continua número.
func TestLancamentoSaiComValorNumerico(t *testing.T) {
	b, err := json.Marshal(&AccountingJournalEntry{ID: 7, Value: decimal.RequireFromString("1234.56")})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"value":1234.56`) || strings.Count(s, `"value"`) != 1 || !strings.Contains(s, `"id":7`) {
		t.Fatalf("json: %s", s)
	}
}
