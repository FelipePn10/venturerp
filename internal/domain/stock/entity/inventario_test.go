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
	if (&PhysicalInventory{ApprovalMode: AprovacaoNunca}).ExigeAprovacao(r, nil) {
		t.Error("NUNCA não exige")
	}
	if !(&PhysicalInventory{ApprovalMode: AprovacaoSempre}).ExigeAprovacao(r, nil) {
		t.Error("SEMPRE exige quando há diferença")
	}
	if (&PhysicalInventory{ApprovalMode: AprovacaoSempre}).ExigeAprovacao(ResumoFechamento{}, nil) {
		t.Error("sem diferença não há o que aprovar")
	}
	if (&PhysicalInventory{ApprovalMode: AprovacaoForaTolerancia}).ExigeAprovacao(r, nil) {
		t.Error("dentro da tolerância não exige")
	}
}

func TestLimiteDoPerfilValeSobreARegraDoInventario(t *testing.T) {
	cem := decimal.NewFromInt(100)
	r := ResumoFechamento{ComDiferenca: 1, ForaTolerancia: 1, ValorAbsoluto: decimal.NewFromInt(80), MaiorLinha: decimal.NewFromInt(80)}
	dentro := &LimiteDeAjuste{ValorDocumento: &cem}
	if (&PhysicalInventory{ApprovalMode: AprovacaoForaTolerancia}).ExigeAprovacao(r, dentro) {
		t.Fatal("dentro do limite dispensa a tolerância do inventário")
	}
	if !(&PhysicalInventory{ApprovalMode: AprovacaoSempre}).ExigeAprovacao(r, dentro) {
		t.Fatal("SEMPRE não abre exceção")
	}
	acima := ResumoFechamento{ComDiferenca: 1, ValorAbsoluto: decimal.NewFromInt(150), MaiorLinha: decimal.NewFromInt(20)}
	if !(&PhysicalInventory{ApprovalMode: AprovacaoNunca}).ExigeAprovacao(acima, dentro) {
		t.Fatal("acima do limite vai à alçada mesmo no modo NUNCA")
	}
	linha := decimal.NewFromInt(10)
	if !(&PhysicalInventory{ApprovalMode: AprovacaoNunca}).ExigeAprovacao(acima, &LimiteDeAjuste{ValorLinha: &linha}) {
		t.Fatal("o limite por linha também barra")
	}
	if (&PhysicalInventory{ApprovalMode: AprovacaoNunca}).ExigeAprovacao(ResumoFechamento{}, dentro) {
		t.Fatal("sem diferença não há o que aprovar")
	}
}

func TestDividirEmCadernos(t *testing.T) {
	got := DividirEmCadernos([]int64{1, 1, 1, 2, 2}, 2)
	want := []int{1, 1, 2, 3, 3}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%v, quero %v: limite e troca de almoxarifado abrem caderno", got, want)
		}
	}
}

func TestSituacaoDoCaderno(t *testing.T) {
	for _, c := range []struct {
		caderno CadernoInventario
		quero   string
	}{
		{CadernoInventario{Linhas: 3}, "GERADO"},
		{CadernoInventario{Linhas: 3, Impressoes: 1}, "IMPRESSO"},
		{CadernoInventario{Linhas: 3, Impressoes: 1, Contadas: 2}, "EM_DIGITACAO"},
		{CadernoInventario{Linhas: 3, Impressoes: 2, Contadas: 3}, "DIGITADO"},
	} {
		if got := c.caderno.Situacao(); got != c.quero {
			t.Errorf("%+v: %s, quero %s", c.caderno, got, c.quero)
		}
	}
}
