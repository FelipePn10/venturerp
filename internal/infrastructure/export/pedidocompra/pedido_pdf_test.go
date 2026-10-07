package pedidocompra

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
)

func TestPedidoCompraPDF(t *testing.T) {
	entrega := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	dados := &purchase_order_uc.DadosDocumento{
		Empresa:    purchase_order_uc.Parte{Nome: "TECNOFER LTDA", CNPJCPF: "52.454.668/0001-02", Cidade: "Curitiba", UF: "PR"},
		Fornecedor: purchase_order_uc.Parte{Nome: "AÇO E SEGURANÇA LTDA", Fantasia: "AÇO SEG", Email: "vendas@aco.test"},
		Condicao:   "30/60 DDL",
		CorMarca:   "#1F6B3A",
	}
	for i := 1; i <= 70; i++ {
		dados.Linhas = append(dados.Linhas, purchase_order_uc.LinhaDocumento{Sequence: i, ItemCode: fmt.Sprintf("RN-%03d", i),
			Descricao: "CHAPA DE AÇO SAE 1020 LAMINADA A QUENTE COM DESCRIÇÃO LONGA QUE PRECISA QUEBRAR EM MAIS DE UMA LINHA",
			Unidade:   "KG", Quantidade: 10.5, PrecoUnit: 1234.5678, DescontoPct: 5, IPIPct: 3.25, Total: 12312.34, Entrega: &entrega,
			Observacao: "entregar paletizado"})
	}
	d := purchase_order_uc.DocumentoPedido{Dados: dados, Numero: 42, Emissao: entrega, Situacao: "aprovado", Moeda: "BRL", Frete: "FOB",
		Bruto: decimal.RequireFromString("1000"), Desconto: decimal.RequireFromString("10"), IPI: decimal.RequireFromString("5"),
		FreteFOB: decimal.RequireFromString("20"), Liquido: decimal.RequireFromString("1015"), Observacao: "Mencionar o pedido na NF",
		Parcelas: []purchase_order_uc.ParcelaPrevista{{Numero: 1, Vencimento: entrega.AddDate(0, 0, 30), Valor: 507.5}, {Numero: 2, Vencimento: entrega.AddDate(0, 0, 60), Valor: 507.5, Estimada: true}},
		Gerado:   entrega}
	b, err := Gerador{}.PedidoCompra(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatal("não é PDF")
	}
	paginas := len(regexp.MustCompile(`/Type\s*/Page[^s]`).FindAll(b, -1))
	if paginas < 3 {
		t.Fatalf("70 linhas longas deveriam paginar; páginas = %d", paginas)
	}
	d.Rascunho = true
	d.Dados.Linhas = nil
	d.Parcelas = nil
	if _, err := (Gerador{}).PedidoCompra(d); err != nil {
		t.Fatalf("pedido sem linhas: %v", err)
	}
	if _, err := (Gerador{}).PedidoCompra(purchase_order_uc.DocumentoPedido{}); err == nil {
		t.Fatal("sem dados deveria falhar")
	}
	if dinheiro(1234567.891, 2) != "1.234.567,89" || qtd(10.5) != "10,5" || qtd(3) != "3" || dinheiro(-5, 2) != "-5,00" {
		t.Fatalf("formatação: %s %s %s %s", dinheiro(1234567.891, 2), qtd(10.5), qtd(3), dinheiro(-5, 2))
	}
}
