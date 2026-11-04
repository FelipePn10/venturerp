package fiscal_uc

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func TestCfopDaVendaSegueAOperacao(t *testing.T) {
	casos := []struct {
		nome, empresa, destino, ie, pessoa, cfop string
	}{
		{"mesmo estado", "SP", "SP", "123", "J", "5101"},
		{"mesmo estado sem diferenciar caixa", "SP", "sp", "", "F", "5101"},
		{"sem UF de destino é tratada como interna", "SP", "", "123", "J", "5101"},
		{"empresa sem configuração é tratada como interna", "", "PR", "123", "J", "5101"},
		{"outro estado com contribuinte", "SP", "PR", "123", "J", "6101"},
		{"outro estado sem IE", "SP", "PR", "", "J", "6107"},
		{"outro estado isento", "SP", "PR", "ISENTO", "J", "6107"},
		{"outro estado pessoa física", "SP", "PR", "", "F", "6107"},
	}
	for _, c := range casos {
		if got := cfopDaVenda(c.empresa, c.destino, c.ie, c.pessoa).Cfop; got != c.cfop {
			t.Errorf("%s: CFOP %s, queria %s", c.nome, got, c.cfop)
		}
	}
}

func TestFaltandoNoEnderecoDizOQueASefazExige(t *testing.T) {
	completo := EnderecoDoDestinatario{Logradouro: "Rua A", Numero: "1", Bairro: "Centro", Municipio: "Assis", UF: "SP", CEP: "19800000"}
	if f := faltandoNoEndereco(completo); len(f) != 0 {
		t.Fatalf("endereço completo acusou falta de %v", f)
	}
	semNumero := completo
	semNumero.Numero, semNumero.CEP = "", ""
	if f := strings.Join(faltandoNoEndereco(semNumero), ","); f != "número,CEP" {
		t.Fatalf("faltando = %q", f)
	}
}

func TestLinhaSemQuantidadeERecusada(t *testing.T) {
	err := conferirLinhasDaNota([]request.CreateFiscalExitItemDTO{{Quantity: decimal.NewFromFloat(2)}, {UnitPrice: decimal.NewFromFloat(50), TotalPrice: decimal.NewFromFloat(100)}})
	if err == nil || !strings.Contains(err.Error(), "linha 2") {
		t.Fatalf("a linha sem quantidade passou ou não foi indicada: %v", err)
	}
	if err := conferirLinhasDaNota(nil); err == nil {
		t.Fatal("nota sem itens passou")
	}
	if err := conferirLinhasDaNota([]request.CreateFiscalExitItemDTO{{Quantity: decimal.NewFromFloat(1), UnitPrice: decimal.NewFromFloat(-1)}}); err == nil {
		t.Fatal("preço negativo passou")
	}
}

func TestQuantidadeDoDesktopAntigoContinuaValendo(t *testing.T) {
	itens := []request.CreateFiscalExitItemDTO{{Quantidade: decimal.NewFromFloat(3), UnitPrice: decimal.NewFromFloat(10)}}
	if err := conferirLinhasDaNota(itens); err != nil {
		t.Fatal(err)
	}
	if !itens[0].Quantity.Equal(decimal.NewFromFloat(3)) {
		t.Fatalf("quantidade = %v, queria 3", itens[0].Quantity)
	}
}
