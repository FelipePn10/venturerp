package purchase_order_uc

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
)

// A rejeição da sugestão responde a ordem planejada inteira. Com a quantidade em
// decimal na entidade, ela sairia como texto ("12.5"); o cliente lê número.
func TestOrdemPlanejadaRejeitadaSaiComQuantidadeNumerica(t *testing.T) {
	o := &plannedentity.PlannedOrder{Code: 7, Quantity: decimal.RequireFromString("12.5"),
		QuantityLoss: decimal.RequireFromString("0.25"), QuantityCorrected: decimal.RequireFromString("12.75")}
	b, err := json.Marshal(ParaOrdemPlanejada(o))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"Quantity":12.5`, `"QuantityLoss":0.25`, `"QuantityCorrected":12.75`, `"Code":7`} {
		if !strings.Contains(s, want) {
			t.Fatalf("faltou %s em %s", want, s)
		}
	}
	if strings.Count(s, `"Quantity"`) != 1 {
		t.Fatalf("chave Quantity repetida: %s", s)
	}
}
