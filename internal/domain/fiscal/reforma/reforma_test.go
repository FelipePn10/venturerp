package reforma

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestCalcular2026PelaLei(t *testing.T) {
	a, err := ResolverAliquotas(2026, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Fonte != FonteLei {
		t.Fatalf("fonte %s", a.Fonte)
	}
	g := Calcular(a, ClassificacaoPadrao(), d("1000"), d("180"), d("16.50"), d("76"))
	// base 1000 − 180 − 16,50 − 76 = 727,50; CBS 0,9% = 6,55 (6,5475); IBS UF 0,1% = 0,73 (0,7275).
	if !g.ComValores || !g.Base.Equal(d("727.5")) || !g.CBS.Valor.Equal(d("6.55")) || !g.IBSUF.Valor.Equal(d("0.73")) ||
		!g.IBSMun.Valor.IsZero() || !g.IBS.Equal(d("0.73")) {
		t.Fatalf("grupo = %+v", g)
	}
	if g.CST != "000" || g.ClassTrib != "000001" || g.ComReducao {
		t.Fatalf("CST/cClassTrib = %s/%s redução %v", g.CST, g.ClassTrib, g.ComReducao)
	}
	if g := Calcular(a, ClassificacaoPadrao(), d("10"), d("20"), d("0"), d("0")); !g.Base.IsZero() {
		t.Fatalf("base negativa deveria zerar: %s", g.Base)
	}
}

func Test2027IBSDaLeiECBSObrigatoria(t *testing.T) {
	_, err := ResolverAliquotas(2027, nil)
	if err == nil || !strings.Contains(err.Error(), "art. 347") {
		t.Fatalf("2027 sem CBS cadastrada deveria recusar citando o art. 347: %v", err)
	}
	ibsUF, ibsMun, cbs, ok := AliquotasDaLei(2028)
	if !ok || !ibsUF.Equal(d("0.05")) || !ibsMun.Equal(d("0.05")) || cbs != nil {
		t.Fatalf("2028 pela lei: %s %s %v %v", ibsUF, ibsMun, cbs, ok)
	}
	if _, err := ResolverAliquotas(2029, nil); err == nil {
		t.Fatal("2029 sem cadastro deveria recusar")
	}
	cad := &Aliquotas{IBSUF: d("0.05"), IBSMun: d("0.05"), CBS: d("8.7")}
	a, err := ResolverAliquotas(2027, cad)
	if err != nil || a.Fonte != FonteEmpresa || !a.CBS.Equal(d("8.7")) {
		t.Fatalf("cadastro da empresa deveria prevalecer: %+v %v", a, err)
	}
}

func TestReducaoDeAliquota(t *testing.T) {
	a := Aliquotas{IBSUF: d("0.05"), IBSMun: d("0.05"), CBS: d("8.7"), Fonte: FonteEmpresa}
	c := Classificacao{ClassTrib: "200030", CST: "200", ExigeTributacao: true, PercReducaoIBS: d("60"), PercReducaoCBS: d("60")}
	g := Calcular(a, c, d("1000"), d("0"), d("0"), d("0"))
	// CBS 8,7% com 60% de redução = 3,48%; 1000 × 3,48% = 34,80.
	if !g.ComReducao || !g.CBS.AliquotaEfetiva.Equal(d("3.48")) || !g.CBS.Valor.Equal(d("34.8")) || !g.CBS.Aliquota.Equal(d("8.7")) {
		t.Fatalf("CBS reduzida = %+v", g.CBS)
	}
	if !g.IBSUF.AliquotaEfetiva.Equal(d("0.02")) || !g.IBSUF.Valor.Equal(d("0.2")) {
		t.Fatalf("IBS UF reduzido = %+v", g.IBSUF)
	}
}

func TestCSTSemTributacaoNaoLevaValores(t *testing.T) {
	a, _ := ResolverAliquotas(2026, nil)
	g := Calcular(a, Classificacao{ClassTrib: "410004", CST: "410"}, d("1000"), d("0"), d("0"), d("0"))
	if g.ComValores || !g.Base.IsZero() || !g.IBS.IsZero() || g.CST != "410" || g.ClassTrib != "410004" {
		t.Fatalf("imunidade/não incidência não leva valores: %+v", g)
	}
}

func TestPrecedenciaDoClassTrib(t *testing.T) {
	casos := []struct{ item, cfop, classif, esperado, origem string }{
		{"200030", "410001", "000001", "200030", OrigemItem},
		{"", "410001", "200030", "410001", OrigemCFOP},
		{"", "", "200030", "200030", OrigemClassificacao},
		{"", "", "", ClassTribPadrao, OrigemPadrao},
	}
	for _, c := range casos {
		got, origem := EscolherClassTrib(c.item, c.cfop, c.classif)
		if got != c.esperado || origem != c.origem {
			t.Fatalf("%+v: veio %s/%s", c, got, origem)
		}
	}
}
