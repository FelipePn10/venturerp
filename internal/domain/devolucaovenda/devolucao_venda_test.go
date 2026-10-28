package devolucaovenda

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/devolucao"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestCFOP(t *testing.T) {
	for venda, quer := range map[string]string{"5101": "1201", "5102": "1202", "6101": "2201", "6102": "2202", "5405": "1411", "6403": "2411", "5949": "1949", "7101": "3201"} {
		if got := CFOP(venda); got != quer {
			t.Errorf("%s → %s, queria %s", venda, got, quer)
		}
	}
}

func linha(id int64, qtd string, serie bool, series ...int64) LinhaDaNota {
	l := LinhaDaNota{FiscalExitItemID: id, Sequencia: int(id), Descricao: "BALANCA", CFOP: "5101", ControlaSerie: serie}
	for _, s := range series {
		l.Series = append(l.Series, SerieDaNota{ID: s})
	}
	l.DefinirOrigem(devolucao.Origem{ItemID: id, Descricao: "BALANCA", Quantidade: d(qtd), ValorUnitario: d("100"), Total: d("1000"),
		BaseICMS: d("1000"), ICMS: d("180"), IPI: d("50"), PIS: d("16.5"), COFINS: d("76")})
	return l
}

func TestCalcular(t *testing.T) {
	ls := []LinhaDaNota{linha(1, "10", false), linha(2, "3", true, 70, 71, 72)}
	itens, total, err := Calcular(ls, []Pedido{{FiscalExitItemID: 1, Quantidade: d("4")}, {FiscalExitItemID: 2, Quantidade: d("1"), Series: []int64{71}}})
	if err != nil {
		t.Fatal(err)
	}
	// linha 1: 40% de 1000 = 400, IPI 20 → 420; linha 2: 1/3 de 1000 = 333.33, IPI 16.67 → 350.00
	if !itens[0].Total.Equal(d("420")) || !itens[0].ICMS.Equal(d("72")) || itens[0].CFOP != "1201" {
		t.Fatalf("linha 1: %+v", itens[0])
	}
	if !total.Equal(d("770")) {
		t.Fatalf("total %s", total)
	}
	casos := map[string][]Pedido{
		"mais que o vendido":  {{FiscalExitItemID: 1, Quantidade: d("11")}},
		"série faltando":      {{FiscalExitItemID: 2, Quantidade: d("2"), Series: []int64{70}}},
		"série de outra nota": {{FiscalExitItemID: 2, Quantidade: d("1"), Series: []int64{99}}},
		"linha de outra nota": {{FiscalExitItemID: 9, Quantidade: d("1")}},
		"linha repetida":      {{FiscalExitItemID: 1, Quantidade: d("1")}, {FiscalExitItemID: 1, Quantidade: d("1")}},
		"série fracionada":    {{FiscalExitItemID: 2, Quantidade: d("1.5"), Series: []int64{70}}},
		"nenhum item":         {},
	}
	for nome, p := range casos {
		if _, _, err := Calcular(ls, p); err == nil {
			t.Errorf("%s: devia recusar", nome)
		}
	}
}

func TestAberturaEEfetivacao(t *testing.T) {
	ch := "35261012345678000190550010000012341000012345"
	if ConferirAbertura(NFCliente, "DEFEITO", &ch) != nil || ConferirAbertura(EmissaoPropria, "DESISTENCIA", nil) != nil {
		t.Fatal("válidos")
	}
	curta := "123"
	if ConferirAbertura(NFCliente, "DEFEITO", &curta) == nil || ConferirAbertura(EmissaoPropria, "XPTO", nil) == nil {
		t.Fatal("inválidos")
	}
	if PodeEfetivar(&Devolucao{Modo: EmissaoPropria, Situacao: Rascunho}) == nil {
		t.Fatal("emissão própria só efetiva com a NF autorizada")
	}
	if PodeEfetivar(&Devolucao{Modo: NFCliente, Situacao: Rascunho}) != nil || PodeEfetivar(&Devolucao{Situacao: Efetivada}) != nil {
		t.Fatal("NF do cliente efetiva do rascunho; efetivada é idempotente")
	}
}
