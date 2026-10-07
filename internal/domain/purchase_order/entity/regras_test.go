package entity

import (
	"testing"
	"time"
)

func ptr(s string) *string { return &s }

func linha(qtd, preco, desc, ipi float64) *PurchaseOrderItem {
	return &PurchaseOrderItem{RequestedQty: qtd, UnitPrice: preco, DiscountPct: desc, IPIPct: ipi, IsActive: true, Status: PurchaseOrderItemStatusOPEN}
}

func TestCalcularTotais(t *testing.T) {
	cancelada := linha(5, 100, 0, 0)
	cancelada.Status = PurchaseOrderItemStatusCANCELLED
	removida := linha(5, 100, 0, 0)
	removida.IsActive = false
	comResiduo := linha(10, 2, 0, 0)
	comResiduo.CancelledQty = 4 // conta 6 × 2 = 12

	itens := []*PurchaseOrderItem{linha(10, 5, 10, 5), cancelada, removida, comResiduo}
	casos := []struct {
		nome            string
		capa            PurchaseOrder
		bruto, liquido  string
		frete, desconto string
	}{
		{"CIF não soma frete", PurchaseOrder{FreightType: "CIF", FreightValue: 30}, "62", "59.25", "0", "5"},
		{"FOB valor fechado", PurchaseOrder{FreightType: "FOB", FreightValue: 30}, "62", "89.25", "30", "5"},
		{"FOB por unidade", PurchaseOrder{FreightType: "fob", FreightValue: 0.5, FreightValueMode: ptr("UNITARIO")}, "62", "67.25", "8", "5"},
		{"FOB percentual da mercadoria", PurchaseOrder{FreightType: "FOB", FreightValue: 10, FreightValueType: ptr("PERCENTUAL")}, "62", "64.95", "5.7", "5"},
	}
	for _, c := range casos {
		tot := CalcularTotais(&c.capa, itens)
		if tot.Bruto.String() != c.bruto || tot.Liquido.String() != c.liquido || tot.Frete.String() != c.frete || tot.Desconto.String() != c.desconto {
			t.Errorf("%s: bruto=%s desc=%s ipi=%s frete=%s liquido=%s", c.nome, tot.Bruto, tot.Desconto, tot.IPI, tot.Frete, tot.Liquido)
		}
		c.capa.AplicarTotais(tot)
		if c.capa.TotalGross != 62 {
			t.Errorf("%s: total_gross=%v", c.nome, c.capa.TotalGross)
		}
	}
}

func TestEditavelEVoltaParaRascunho(t *testing.T) {
	for _, s := range []PurchaseOrderStatus{PurchaseOrderStatusDRAFT, PurchaseOrderStatusREQUESTED} {
		if err := (&PurchaseOrder{Status: s}).Editavel(); err != nil {
			t.Errorf("%s deveria ser editável: %v", s, err)
		}
	}
	for _, s := range []PurchaseOrderStatus{PurchaseOrderStatusAPPROVED, PurchaseOrderStatusPARTIAL, PurchaseOrderStatusRECEIVED, PurchaseOrderStatusCANCELLED} {
		if (&PurchaseOrder{Status: s}).Editavel() == nil {
			t.Errorf("%s não deveria ser editável", s)
		}
	}
	o := &PurchaseOrder{Status: PurchaseOrderStatusREQUESTED, AlcadaStatus: "B"}
	o.VoltarParaRascunho()
	if o.Status != PurchaseOrderStatusDRAFT || o.AlcadaStatus != "N" {
		t.Fatalf("depois de alterar um pedido bloqueado: %s/%s", o.Status, o.AlcadaStatus)
	}
}

func TestSituacaoPelasLinhas(t *testing.T) {
	aberta := linha(10, 1, 0, 0)
	recebidaEmParte := linha(10, 1, 0, 0)
	recebidaEmParte.ReceivedQty = 4
	fechadaComResiduo := linha(10, 1, 0, 0)
	fechadaComResiduo.ReceivedQty, fechadaComResiduo.CancelledQty, fechadaComResiduo.Status = 4, 6, PurchaseOrderItemStatusRECEIVED
	cancelada := linha(10, 1, 0, 0)
	cancelada.CancelledQty, cancelada.Status = 10, PurchaseOrderItemStatusCANCELLED

	casos := []struct {
		nome  string
		atual PurchaseOrderStatus
		itens []*PurchaseOrderItem
		quer  PurchaseOrderStatus
	}{
		{"rascunho não muda", PurchaseOrderStatusDRAFT, []*PurchaseOrderItem{cancelada}, PurchaseOrderStatusDRAFT},
		{"ainda há saldo e algo chegou", PurchaseOrderStatusAPPROVED, []*PurchaseOrderItem{recebidaEmParte, cancelada}, PurchaseOrderStatusPARTIAL},
		{"saldo aberto, nada chegou", PurchaseOrderStatusAPPROVED, []*PurchaseOrderItem{aberta, cancelada}, PurchaseOrderStatusAPPROVED},
		{"tudo fechado com recebimento", PurchaseOrderStatusPARTIAL, []*PurchaseOrderItem{fechadaComResiduo, cancelada}, PurchaseOrderStatusRECEIVED},
		{"tudo cancelado sem recebimento", PurchaseOrderStatusAPPROVED, []*PurchaseOrderItem{cancelada}, PurchaseOrderStatusCANCELLED},
	}
	for _, c := range casos {
		if got := SituacaoPelasLinhas(c.atual, c.itens); got != c.quer {
			t.Errorf("%s: %s, quer %s", c.nome, got, c.quer)
		}
	}
	if s := fechadaComResiduo.Saldo(); s != 0 {
		t.Errorf("saldo da linha fechada = %v", s)
	}
	if TotalDaLinha(3, 0.1, 0) != 0.3 || TotalDaLinha(10, 5, 10) != 45 {
		t.Errorf("total da linha sem arredondamento binário")
	}
}

func dia(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestAceitaRecebimento(t *testing.T) {
	for _, s := range []PurchaseOrderStatus{PurchaseOrderStatusAPPROVED, PurchaseOrderStatusPARTIAL, PurchaseOrderStatusRECEIVED} {
		if err := SituacaoAceitaRecebimento(1, s); err != nil {
			t.Errorf("%s deveria receber: %v", s, err)
		}
	}
	for _, s := range []PurchaseOrderStatus{PurchaseOrderStatusDRAFT, PurchaseOrderStatusREQUESTED, PurchaseOrderStatusCANCELLED, ""} {
		if (&PurchaseOrder{Code: 7, Status: s}).AceitaRecebimento() == nil {
			t.Errorf("%q não deveria receber material", s)
		}
	}
}

func TestDiasDeAtrasoEDataPrevista(t *testing.T) {
	if d := DiasDeAtraso(dia("2026-10-01"), dia("2026-10-04").Add(23*time.Hour)); d != 3 {
		t.Fatalf("atraso = %d, quer 3 (hora do dia não conta)", d)
	}
	if d := DiasDeAtraso(dia("2026-10-04"), dia("2026-10-04")); d != 0 {
		t.Fatalf("vence hoje não está atrasado: %d", d)
	}
	capa := &PurchaseOrder{EmissionDate: dia("2026-09-01")}
	it := &PurchaseOrderItem{}
	if d, est := it.DataPrevista(capa); !est || !d.Equal(dia("2026-09-01")) {
		t.Fatalf("sem datas usa a emissão, estimada: %v %v", d, est)
	}
	entrega := dia("2026-10-10")
	capa.DeliveryDate = &entrega
	if d, est := it.DataPrevista(capa); est || !d.Equal(entrega) {
		t.Fatalf("entrega da capa: %v %v", d, est)
	}
	pedida, prometida := dia("2026-10-12"), dia("2026-10-20")
	it.DeliveryDate = &pedida
	if d, _ := it.DataPrevista(capa); !d.Equal(pedida) {
		t.Fatalf("entrega da linha vale mais que a da capa: %v", d)
	}
	it.PromisedDate = &prometida
	if d, _ := it.DataPrevista(capa); !d.Equal(prometida) {
		t.Fatalf("a promessa do fornecedor vale mais que a data pedida: %v", d)
	}
}

func TestValoresAFaturar(t *testing.T) {
	e1, e2 := dia("2026-10-10"), dia("2026-10-20")
	capa := &PurchaseOrder{EmissionDate: dia("2026-10-01"), FreightType: "FOB", FreightValue: 20}
	a := linha(10, 5, 0, 0) // 50, metade faturada
	a.InvoicedQty = 5
	a.DeliveryDate = &e1
	b := linha(2, 40, 10, 5) // 80 − 8 + 3,60 = 75,60
	b.DeliveryDate = &e2
	cancelada := linha(1, 1000, 0, 0)
	cancelada.Status = PurchaseOrderItemStatusCANCELLED
	got := ValoresAFaturar(capa, []*PurchaseOrderItem{a, b, cancelada})
	if len(got) != 2 {
		t.Fatalf("grupos = %+v", got)
	}
	// frete 20 rateado pelo bruto (50 + 80 = 130): a tem 25 em aberto → 3,85; b → 12,31.
	if !got[0].Entrega.Equal(e1) || got[0].Valor.String() != "28.85" {
		t.Errorf("grupo 1: %s %s", got[0].Entrega.Format("2006-01-02"), got[0].Valor)
	}
	if !got[1].Entrega.Equal(e2) || got[1].Valor.String() != "87.91" {
		t.Errorf("grupo 2: %s %s", got[1].Entrega.Format("2006-01-02"), got[1].Valor)
	}
	a.InvoicedQty, b.InvoicedQty = 10, 2
	if len(ValoresAFaturar(capa, []*PurchaseOrderItem{a, b})) != 0 {
		t.Error("tudo faturado não deixa previsão")
	}
	if (&PurchaseOrderItem{RequestedQty: 5, InvoicedQty: 7, IsActive: true}).SaldoAFaturar() != 0 {
		t.Error("faturado a mais não vira saldo negativo")
	}
}
