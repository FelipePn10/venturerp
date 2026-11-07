package entity

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func texto(s string) *string { return &s }
func dias(n int) *int        { return &n }
func dec(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}

func itemBase() *Item {
	i := &Item{}
	i.AplicarPadroes()
	return i
}

func TestPadroesSaoOsDaColuna(t *testing.T) {
	i := itemBase()
	if i.Supplies.ReceiptRouting != RecebimentoPadrao || i.Planning.MaterialIssueTiming != BaixaNaProducao || i.Planning.ProductionReportingType != ApontaPorOperacao {
		t.Fatalf("padrões inesperados: %+v", i)
	}
	if err := i.validarCadastroAmpliado(); err != nil {
		t.Fatalf("item sem os campos novos deve continuar válido: %v", err)
	}
}

func TestValidacaoDoCadastroAmpliado(t *testing.T) {
	casos := []struct {
		nome  string
		monta func(*Item)
		erro  string
	}{
		{"GTIN com dígito errado", func(i *Item) { i.Accounting.GTIN = texto("7891000100104") }, "GTIN"},
		{"tipo EFD fora da tabela", func(i *Item) { i.Accounting.EFDItemType = texto("11") }, "EFD"},
		{"fator sem unidade", func(i *Item) { i.Accounting.TaxUnitFactor = dec("2") }, "unidade tributável"},
		{"unidade sem fator", func(i *Item) { i.Accounting.TaxUnit = texto("KG") }, "fator"},
		{"fator zero", func(i *Item) { i.Accounting.TaxUnit = texto("KG"); i.Accounting.TaxUnitFactor = dec("0") }, "maior que zero"},
		{"escala não relevante sem fabricante", func(i *Item) { nao := false; i.Accounting.RelevantScale = &nao }, "CNPJ do fabricante"},
		{"validade zero", func(i *Item) { i.Warehouse.ShelfLifeDays = dias(0) }, "validade"},
		{"vida mínima maior que a validade", func(i *Item) { i.Warehouse.ShelfLifeDays = dias(10); i.Warehouse.MinRemainingLifeDays = dias(11) }, "vida útil"},
		{"perigoso sem ONU", func(i *Item) { i.Warehouse.Hazardous = true }, "ONU"},
		{"ONU sem perigoso", func(i *Item) { i.Warehouse.UNNumber = texto("1203") }, "perigoso"},
		{"ONU inválido", func(i *Item) { i.Warehouse.Hazardous = true; i.Warehouse.UNNumber = texto("12A3") }, "4 dígitos"},
		{"prazo negativo", func(i *Item) { i.Planning.PurchaseLeadTimeDays = dias(-1) }, "negativo"},
		{"lote máximo abaixo do mínimo", func(i *Item) { i.Planning.MinimumLot = decimal.NewFromInt(10); i.Planning.Extras.MaximumLot = dec("5") }, "menor que o lote mínimo"},
		{"falta acima de 100%", func(i *Item) { i.Supplies.UnderDeliveryTolerancePct = dec("101") }, "100%"},
		{"roteiro inválido", func(i *Item) { i.Supplies.ReceiptRouting = "X" }, "roteiro"},
		{"baixa inválida", func(i *Item) { i.Planning.MaterialIssueTiming = "X" }, "baixa"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			i := itemBase()
			c.monta(i)
			err := i.validarCadastroAmpliado()
			if err == nil || !strings.Contains(err.Error(), c.erro) {
				t.Fatalf("erro = %v, esperava conter %q", err, c.erro)
			}
		})
	}
}

func TestCadastroAmpliadoCompletoPassa(t *testing.T) {
	i := itemBase()
	i.Accounting.GTIN, i.Accounting.GTINTax = texto("7891000100103"), texto("17891000100100")
	i.Accounting.EFDItemType = texto("03")
	i.Accounting.TaxUnit, i.Accounting.TaxUnitFactor = texto("KG"), dec("2.5")
	i.Warehouse.ShelfLifeDays, i.Warehouse.MinRemainingLifeDays = dias(180), dias(90)
	i.Warehouse.Hazardous, i.Warehouse.UNNumber, i.Warehouse.HazardClass = true, texto("1203"), texto("3")
	i.Supplies.UnderDeliveryTolerancePct, i.Supplies.OverDeliveryTolerancePct = dec("2"), dec("5")
	if err := i.validarCadastroAmpliado(); err != nil {
		t.Fatal(err)
	}
}

func TestConferirVendaAplicaAsRegrasDoCadastro(t *testing.T) {
	cliente, outro := int64(10), int64(20)
	base := func() *Item {
		i := itemBase()
		i.BusinessCode = "PAR-M8"
		i.Warehouse.AcceptsFractionalQuantity = true
		return i
	}
	casos := []struct {
		nome    string
		monta   func(*Item)
		qtd     string
		cliente *int64
		erro    string
	}{
		{"sem regra passa", func(*Item) {}, "1.5", nil, ""},
		{"abaixo do mínimo", func(i *Item) { i.Commercial.MinimumSaleQuantity = dec("100") }, "99", nil, "venda mínima"},
		{"no mínimo", func(i *Item) { i.Commercial.MinimumSaleQuantity = dec("100") }, "100", nil, ""},
		{"fora do múltiplo", func(i *Item) { i.Commercial.SaleMultiple = dec("50") }, "120", nil, "múltiplos"},
		{"no múltiplo", func(i *Item) { i.Commercial.SaleMultiple = dec("50") }, "150", nil, ""},
		{"múltiplo decimal", func(i *Item) { i.Commercial.SaleMultiple = dec("0.5") }, "2.5", nil, ""},
		{"fração proibida", func(i *Item) { i.Warehouse.AcceptsFractionalQuantity = false }, "2.5", nil, "fracionada"},
		{"inteiro com decimais zerados", func(i *Item) { i.Warehouse.AcceptsFractionalQuantity = false }, "2.000", nil, ""},
		{"exclusivo para o dono", func(i *Item) { i.Commercial.ExclusiveCustomerCode = &cliente }, "1", &cliente, ""},
		{"exclusivo para outro", func(i *Item) { i.Commercial.ExclusiveCustomerCode = &cliente }, "1", &outro, "exclusivo"},
		{"exclusivo sem cliente", func(i *Item) { i.Commercial.ExclusiveCustomerCode = &cliente }, "1", nil, "exclusivo"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			i := base()
			c.monta(i)
			err := i.ConferirVenda(decimal.RequireFromString(c.qtd), c.cliente)
			if c.erro == "" && err != nil {
				t.Fatal(err)
			}
			if c.erro != "" && (err == nil || !strings.Contains(err.Error(), c.erro)) {
				t.Fatalf("erro = %v, esperava %q", err, c.erro)
			}
		})
	}
}

func TestForaDoMRP(t *testing.T) {
	ativo := &Item{Planning: Planning{Active: true}}
	if ativo.ForaDoMRP() {
		t.Fatal("item ativo entra no MRP")
	}
	for nome, i := range map[string]*Item{
		"desligado": {Planning: Planning{Active: false}},
		"protótipo": {Planning: Planning{Active: true}, IsPrototype: true},
		"sucata":    {Planning: Planning{Active: true}, IsScrap: true},
	} {
		if !i.ForaDoMRP() {
			t.Fatalf("%s fica fora do MRP", nome)
		}
	}
}
