package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func dp(s string) *decimal.Decimal { v := decimal.RequireFromString(s); return &v }

func TestForaDaTolerancia(t *testing.T) {
	casos := []struct {
		nome                    string
		sistema, diferenca, cst string
		pct, valor              *decimal.Decimal
		quer                    bool
	}{
		{"sem diferença nunca passa", "100", "0", "5", dp("0"), dp("0"), false},
		{"sem tolerância não barra", "100", "50", "5", nil, nil, false},
		{"exemplo do Oracle: 111 contra 100 com 10%", "100", "11", "1", dp("10"), nil, true},
		{"exemplo do Oracle: 89 contra 100 com 10%", "100", "-11", "1", dp("10"), nil, true},
		{"dentro do percentual", "100", "10", "1", dp("10"), nil, false},
		{"passa do valor", "100", "1", "500", nil, dp("100"), true},
		{"sem saldo na foto passa do percentual", "0", "1", "1", dp("10"), nil, true},
	}
	for _, c := range casos {
		got := ForaDaTolerancia(decimal.RequireFromString(c.sistema), decimal.RequireFromString(c.diferenca), decimal.RequireFromString(c.cst), c.pct, c.valor)
		if got != c.quer {
			t.Errorf("%s: veio %v", c.nome, got)
		}
	}
}

func TestExigeAprovacao(t *testing.T) {
	r := ResumoFechamento{ComDiferenca: 2, ForaTolerancia: 0}
	if (&PhysicalInventory{ApprovalMode: AprovacaoNunca}).ExigeAprovacao(r) {
		t.Error("NUNCA não exige")
	}
	if !(&PhysicalInventory{ApprovalMode: AprovacaoSempre}).ExigeAprovacao(r) {
		t.Error("SEMPRE exige quando há diferença")
	}
	if (&PhysicalInventory{ApprovalMode: AprovacaoSempre}).ExigeAprovacao(ResumoFechamento{}) {
		t.Error("sem diferença não há o que aprovar")
	}
	if (&PhysicalInventory{ApprovalMode: AprovacaoForaTolerancia}).ExigeAprovacao(r) {
		t.Error("dentro da tolerância não exige")
	}
}
