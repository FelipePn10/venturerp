package entity

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Regras do pedido de compra que não dependem de banco: quando ele pode ser
// alterado, como os totais se formam e o que sobra de saldo em cada linha.

// Editavel diz se capa e linhas ainda podem mudar. Depois de aprovado, o pedido
// é o compromisso enviado ao fornecedor: muda-se o saldo (eliminar resíduo),
// não o que foi combinado. Um pedido parado na alçada (REQUESTED) volta a ser
// rascunho ao ser alterado — ver VoltarParaRascunho.
func (o *PurchaseOrder) Editavel() error {
	switch o.Status {
	case PurchaseOrderStatusDRAFT, PurchaseOrderStatusREQUESTED, "":
		return nil
	}
	return fmt.Errorf("o pedido %d está %s e não pode mais ser alterado; para deixar de receber o que falta, elimine o saldo da linha", o.Code, RotuloSituacao(o.Status))
}

// AceitaRecebimento: material só entra contra pedido aprovado. Receber um
// rascunho (ou um pedido parado na alçada) seria comprar sem a aprovação que
// a alçada exige. RECEIVED continua aceitando o que a tolerância permite.
func (o *PurchaseOrder) AceitaRecebimento() error {
	return SituacaoAceitaRecebimento(o.Code, o.Status)
}

// SituacaoAceitaRecebimento é a mesma regra para quem só tem a situação.
func SituacaoAceitaRecebimento(code int64, s PurchaseOrderStatus) error {
	switch s {
	case PurchaseOrderStatusAPPROVED, PurchaseOrderStatusPARTIAL, PurchaseOrderStatusRECEIVED:
		return nil
	}
	return fmt.Errorf("o pedido de compra %d está %s: aprove o pedido antes de receber o material", code, RotuloSituacao(s))
}

// VoltarParaRascunho desfaz a avaliação de alçada: o valor mudou, então a
// decisão anterior (bloqueado ou acima do teto) já não vale e o pedido precisa
// ser aprovado de novo.
func (o *PurchaseOrder) VoltarParaRascunho() {
	if o.Status == PurchaseOrderStatusREQUESTED {
		o.Status = PurchaseOrderStatusDRAFT
	}
	o.AlcadaStatus = "N"
}

// RotuloSituacao é a situação como o comprador lê.
func RotuloSituacao(s PurchaseOrderStatus) string {
	switch s {
	case PurchaseOrderStatusDRAFT:
		return "em rascunho"
	case PurchaseOrderStatusREQUESTED:
		return "aguardando alçada"
	case PurchaseOrderStatusAPPROVED:
		return "aprovado"
	case PurchaseOrderStatusPARTIAL:
		return "recebido em parte"
	case PurchaseOrderStatusRECEIVED:
		return "recebido"
	case PurchaseOrderStatusCANCELLED:
		return "cancelado"
	}
	return strings.ToLower(string(s))
}

// Saldo é o que a linha ainda espera receber.
func (it *PurchaseOrderItem) Saldo() float64 {
	s := decimal.NewFromFloat(it.RequestedQty).Sub(decimal.NewFromFloat(it.ReceivedQty)).Sub(decimal.NewFromFloat(it.CancelledQty))
	if s.IsNegative() {
		return 0
	}
	f, _ := s.Round(6).Float64()
	return f
}

// ConsideradaNosTotais: linha removida ou cancelada por inteiro não conta.
func (it *PurchaseOrderItem) ConsideradaNosTotais() bool {
	return it.IsActive && it.Status != PurchaseOrderItemStatusCANCELLED
}

// TotalDaLinha é quantidade × preço menos o desconto da linha, em centavos.
func TotalDaLinha(qtd, preco, descontoPct float64) float64 {
	bruto := decimal.NewFromFloat(qtd).Mul(decimal.NewFromFloat(preco))
	desc := bruto.Mul(decimal.NewFromFloat(descontoPct)).Div(decimal.NewFromInt(100))
	f, _ := bruto.Sub(desc).Round(2).Float64()
	return f
}

// Totais do pedido. Liquido é o que a empresa vai pagar: mercadoria com
// desconto, mais IPI, mais o frete quando é ela quem paga (FOB).
type Totais struct {
	Bruto    decimal.Decimal
	Desconto decimal.Decimal
	IPI      decimal.Decimal
	Frete    decimal.Decimal
	Liquido  decimal.Decimal
}

// FretePagoPeloComprador: só no FOB o frete sai do caixa de quem compra. No
// CIF o fornecedor paga (e o valor informado é só referência); retira, cortesia
// e sem frete não custam nada ao pedido.
func FretePagoPeloComprador(tipo string) bool {
	return strings.EqualFold(strings.TrimSpace(tipo), "FOB")
}

// CalcularTotais soma as linhas válidas. A quantidade considerada é a pedida
// menos a cancelada: o saldo eliminado deixa de ser compromisso.
func CalcularTotais(o *PurchaseOrder, itens []*PurchaseOrderItem) Totais {
	cem := decimal.NewFromInt(100)
	var t Totais
	qtdTotal := decimal.Zero
	for _, it := range itens {
		if !it.ConsideradaNosTotais() {
			continue
		}
		qtd := decimal.NewFromFloat(it.RequestedQty).Sub(decimal.NewFromFloat(it.CancelledQty))
		if qtd.IsNegative() {
			qtd = decimal.Zero
		}
		qtdTotal = qtdTotal.Add(qtd)
		bruto := qtd.Mul(decimal.NewFromFloat(it.UnitPrice)).Round(2)
		desc := bruto.Mul(decimal.NewFromFloat(it.DiscountPct)).Div(cem).Round(2)
		ipi := bruto.Sub(desc).Mul(decimal.NewFromFloat(it.IPIPct)).Div(cem).Round(2)
		t.Bruto = t.Bruto.Add(bruto)
		t.Desconto = t.Desconto.Add(desc)
		t.IPI = t.IPI.Add(ipi)
	}
	if o != nil && FretePagoPeloComprador(o.FreightType) && o.FreightValue > 0 {
		v := decimal.NewFromFloat(o.FreightValue)
		switch {
		case o.FreightValueType != nil && strings.EqualFold(*o.FreightValueType, "PERCENTUAL"):
			t.Frete = t.Bruto.Sub(t.Desconto).Mul(v).Div(cem).Round(2)
		case o.FreightValueMode != nil && strings.EqualFold(*o.FreightValueMode, "UNITARIO"):
			t.Frete = qtdTotal.Mul(v).Round(2)
		default:
			t.Frete = v.Round(2)
		}
	}
	t.Liquido = t.Bruto.Sub(t.Desconto).Add(t.IPI).Add(t.Frete)
	return t
}

// AplicarTotais grava os totais na capa.
func (o *PurchaseOrder) AplicarTotais(t Totais) {
	o.TotalGross, _ = t.Bruto.Float64()
	o.TotalDiscount, _ = t.Desconto.Float64()
	o.TotalNet, _ = t.Liquido.Float64()
}

// SituacaoPelasLinhas recalcula a situação da capa depois que linhas mudam de
// saldo (eliminação de resíduo): tudo atendido vira RECEIVED; nada recebido e
// tudo cancelado vira CANCELLED; recebido em parte vira PARTIAL. Pedido ainda
// sem aprovação mantém a situação que tinha.
func SituacaoPelasLinhas(atual PurchaseOrderStatus, itens []*PurchaseOrderItem) PurchaseOrderStatus {
	switch atual {
	case PurchaseOrderStatusAPPROVED, PurchaseOrderStatusPARTIAL, PurchaseOrderStatusRECEIVED:
	default:
		return atual
	}
	ativas, abertas, recebidas := 0, 0, 0
	for _, it := range itens {
		if !it.IsActive {
			continue
		}
		ativas++
		if it.ReceivedQty > 0 {
			recebidas++
		}
		if it.Status != PurchaseOrderItemStatusCANCELLED && it.Saldo() > 0.0001 {
			abertas++
		}
	}
	switch {
	case ativas == 0:
		return atual
	case abertas > 0 && recebidas > 0:
		return PurchaseOrderStatusPARTIAL
	case abertas > 0:
		return PurchaseOrderStatusAPPROVED
	case recebidas > 0:
		return PurchaseOrderStatusRECEIVED
	}
	return PurchaseOrderStatusCANCELLED
}

// SaldoAFaturar é o que ainda vai chegar com nota fiscal — e virar título a
// pagar. Recebido sem nota (recebimento físico) ainda será faturado.
func (it *PurchaseOrderItem) SaldoAFaturar() float64 {
	if !it.ConsideradaNosTotais() {
		return 0
	}
	s := decimal.NewFromFloat(it.RequestedQty).Sub(decimal.NewFromFloat(it.InvoicedQty)).Sub(decimal.NewFromFloat(it.CancelledQty))
	if s.IsNegative() {
		return 0
	}
	f, _ := s.Round(6).Float64()
	return f
}

// DataPrevista é quando a linha deve chegar: a data que o fornecedor prometeu,
// senão a de entrega pedida na linha, senão a da capa. Estimada indica que
// nada disso existe e a emissão foi usada.
func (it *PurchaseOrderItem) DataPrevista(o *PurchaseOrder) (time.Time, bool) {
	switch {
	case it.PromisedDate != nil:
		return *it.PromisedDate, false
	case it.DeliveryDate != nil:
		return *it.DeliveryDate, false
	case o != nil && o.DeliveryDate != nil:
		return *o.DeliveryDate, false
	case o != nil:
		return o.EmissionDate, true
	}
	return time.Time{}, true
}

// DiasDeAtraso conta os dias corridos entre a data prevista e hoje (datas,
// sem hora). Zero quando ainda não venceu.
func DiasDeAtraso(prevista, hoje time.Time) int {
	p := time.Date(prevista.Year(), prevista.Month(), prevista.Day(), 0, 0, 0, 0, time.UTC)
	h := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC)
	if !h.After(p) {
		return 0
	}
	return int(h.Sub(p).Hours() / 24)
}

// ValorPrevisto é o dinheiro que um pedido ainda vai custar, agrupado pela
// data em que o material deve chegar (a nota vem com ele).
type ValorPrevisto struct {
	Entrega  time.Time
	Estimada bool
	Valor    decimal.Decimal
}

// ValoresAFaturar reparte o que falta faturar por data prevista de chegada,
// com a mesma conta dos totais (desconto, IPI e frete FOB proporcional).
func ValoresAFaturar(o *PurchaseOrder, itens []*PurchaseOrderItem) []ValorPrevisto {
	cem := decimal.NewFromInt(100)
	total := CalcularTotais(o, itens)
	grupos := map[string]*ValorPrevisto{}
	var ordem []string
	for _, it := range itens {
		qtd := decimal.NewFromFloat(it.SaldoAFaturar())
		if !qtd.IsPositive() {
			continue
		}
		bruto := qtd.Mul(decimal.NewFromFloat(it.UnitPrice))
		desc := bruto.Mul(decimal.NewFromFloat(it.DiscountPct)).Div(cem)
		ipi := bruto.Sub(desc).Mul(decimal.NewFromFloat(it.IPIPct)).Div(cem)
		valor := bruto.Sub(desc).Add(ipi)
		if total.Frete.IsPositive() && total.Bruto.IsPositive() {
			valor = valor.Add(total.Frete.Mul(bruto).Div(total.Bruto))
		}
		data, estimada := it.DataPrevista(o)
		chave := data.Format("2006-01-02")
		g, ok := grupos[chave]
		if !ok {
			g = &ValorPrevisto{Entrega: data, Estimada: estimada}
			grupos[chave] = g
			ordem = append(ordem, chave)
		}
		g.Valor = g.Valor.Add(valor)
		g.Estimada = g.Estimada && estimada
	}
	sort.Strings(ordem)
	out := make([]ValorPrevisto, 0, len(ordem))
	for _, k := range ordem {
		g := grupos[k]
		g.Valor = g.Valor.Round(2)
		if g.Valor.IsPositive() {
			out = append(out, *g)
		}
	}
	return out
}
