package service

import (
	"fmt"
	"github.com/shopspring/decimal"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
)

// exceptionBasis guarda o que as mensagens de exceção precisam e que o netting
// destrói: o estoque disponível e o suprimento firme ANTES do cálculo, e cada
// necessidade bruta do item na ordem em que aparece.
//
// Antes as mensagens eram geradas sobre o mapa de suprimento já consumido pelo
// netting e comparadas com a necessidade LÍQUIDA (que já descontava esse
// suprimento) e com a primeira data de necessidade do item. Na prática: ordem
// atrasada só gerava mensagem se chegasse depois da última necessidade do
// horizonte, excesso quase nunca aparecia, e uma ordem que cobria a necessidade
// de novembro seria "atrasada" em relação à de outubro.
type exceptionBasis struct {
	stock        map[int64]decimal.Decimal
	supply       map[int64][]ports.SupplyEntry
	requirements map[int64][]requirement
	// suggestions são as sugestões gravadas pelo cálculo, por item — o
	// suprimento novo que o pegging distribui depois do estoque e das firmes.
	suggestions map[int64][]suggestedSupply
	// skip: itens fantasma não gravados (a necessidade passa direto aos filhos).
	skip map[int64]bool
	// labels: código de negócio do item, para o texto das mensagens. Item sem
	// rótulo aparece pelo código interno.
	labels map[int64]string
}

// rotulo é como o item aparece na mensagem: o código de negócio que o
// planejador conhece (RODA-200), não o código interno (9003).
func (b *exceptionBasis) rotulo(item int64) string {
	if l := b.labels[item]; l != "" {
		return l
	}
	return fmt.Sprintf("Item %d", item)
}

// qtd formata a quantidade como o usuário lê: 80 e 12,5 (não 80.00 e 12.50).
func qtd(v decimal.Decimal) string {
	return strings.Replace(v.Round(4).String(), ".", ",", 1)
}

type requirement struct {
	date     time.Time
	quantity decimal.Decimal
	// Origem da necessidade (pegging).
	kind string
	code *int64
}

type suggestedSupply struct {
	code     int64
	needDate time.Time
	quantity decimal.Decimal
}

func newExceptionBasis(snapshots map[int64]*entity.StockSnapshot, supply map[int64][]ports.SupplyEntry) *exceptionBasis {
	b := &exceptionBasis{
		stock:        make(map[int64]decimal.Decimal, len(snapshots)),
		supply:       make(map[int64][]ports.SupplyEntry, len(supply)),
		requirements: make(map[int64][]requirement),
		suggestions:  make(map[int64][]suggestedSupply),
		skip:         make(map[int64]bool),
	}
	for item, snap := range snapshots {
		if snap == nil {
			continue
		}
		// Mesma conta do netting (calcNetReqFast): o estoque de segurança não
		// cobre necessidade.
		if avail := snap.Quantity.Sub(snap.ReservedQty).Sub(snap.SafetyStock); avail.IsPositive() {
			b.stock[item] = avail
		}
	}
	for item, entries := range supply {
		b.supply[item] = append([]ports.SupplyEntry(nil), entries...)
	}
	return b
}

func (b *exceptionBasis) addRequirement(item int64, date time.Time, quantity decimal.Decimal, kind string, code *int64) {
	if b == nil || !quantity.IsPositive() {
		return
	}
	b.requirements[item] = append(b.requirements[item], requirement{date: date, quantity: quantity, kind: kind, code: code})
}

// addSuggestion registra a sugestão gravada que traz o item ao estoque.
func (b *exceptionBasis) addSuggestion(s *entity.PlannedOrderSuggestion) {
	if b == nil || s == nil || s.Code <= 0 || !s.Quantity.IsPositive() || !suppliesItem(s) {
		return
	}
	b.suggestions[s.ItemCode] = append(b.suggestions[s.ItemCode], suggestedSupply{code: s.Code, needDate: s.NeedDate, quantity: s.Quantity})
}

// messages faz o pegging de cada item: o estoque inicial cobre as primeiras
// necessidades; depois cada ordem firme, na ordem de chegada, cobre a próxima
// necessidade em aberto. A data da necessidade que a ordem cobre é a data em
// que ela precisa chegar — é com ela que se decide acelerar, antecipar ou
// postergar, e é ela a data sugerida para reprogramar. Ordem que não cobre
// necessidade nenhuma é "cancelar"; sobra parcial acima da tolerância é
// "excesso" (uma mensagem por item). É o que o SAP faz no MD04/MD06.
func (b *exceptionBasis) messages() []*entity.MRPExceptionMessage {
	items := make([]int64, 0, len(b.supply))
	for item, entries := range b.supply {
		if len(entries) > 0 {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })

	out := []*entity.MRPExceptionMessage{}
	for _, item := range items {
		reqs := append([]requirement(nil), b.requirements[item]...)
		sort.SliceStable(reqs, func(i, j int) bool { return reqs[i].date.Before(reqs[j].date) })

		// 1. Estoque inicial cobre as primeiras necessidades.
		stock := b.stock[item]
		for i := range reqs {
			if !stock.IsPositive() {
				break
			}
			usado := decimal.Min(stock, reqs[i].quantity)
			reqs[i].quantity = reqs[i].quantity.Sub(usado)
			stock = stock.Sub(usado)
		}
		abertas := reqs[:0]
		for _, r := range reqs {
			if r.quantity.IsPositive() {
				abertas = append(abertas, r)
			}
		}
		totalAberto := decimal.Zero
		for _, r := range abertas {
			totalAberto = totalAberto.Add(r.quantity)
		}

		// 2. Ordens firmes em ordem de chegada cobrem as necessidades em aberto.
		entries := append([]ports.SupplyEntry(nil), b.supply[item]...)
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].ArrivalDate.Before(entries[j].ArrivalDate) })
		usadoFirme, sobra := decimal.Zero, decimal.Zero
		k := 0
		for _, e := range entries {
			for k < len(abertas) && !abertas[k].quantity.IsPositive() {
				k++
			}
			if k >= len(abertas) {
				// Ordem inteira sobrando já vira "cancelar"; não conta no excesso.
				out = append(out, cancelMessage(item, b.rotulo(item), e))
				continue
			}
			precisaEm := abertas[k].date
			restante := e.Quantity
			for restante.IsPositive() && k < len(abertas) {
				usado := decimal.Min(restante, abertas[k].quantity)
				abertas[k].quantity = abertas[k].quantity.Sub(usado)
				restante = restante.Sub(usado)
				if !abertas[k].quantity.IsPositive() {
					k++
				}
			}
			sobra = sobra.Add(restante)
			usadoFirme = usadoFirme.Add(e.Quantity)
			if m := timingMessage(item, b.rotulo(item), e, precisaEm); m != nil {
				out = append(out, m)
			}
		}

		// 3. Excesso: as ordens que cobrem necessidade trazem mais do que ela
		// além da tolerância. A ordem inteira sobrando já virou "cancelar".
		if totalAberto.IsPositive() && sobra.GreaterThan(totalAberto.Mul(decimal.NewFromFloat(excessThreshold))) {
			out = append(out, &entity.MRPExceptionMessage{
				ItemCode:    item,
				MessageType: entity.ExceptionExcess,
				Description: fmt.Sprintf(
					"%s: as ordens firmes (%s un.) passam da necessidade em aberto (%s un.) e deixam %s unidades de sobra. Estoque excedente projetado.",
					b.rotulo(item), qtd(usadoFirme), qtd(totalAberto), qtd(sobra)),
			})
		}
	}
	return out
}

func cancelMessage(item int64, rotulo string, e ports.SupplyEntry) *entity.MRPExceptionMessage {
	code, tipo := e.SourceCode, string(e.SourceType)
	return &entity.MRPExceptionMessage{
		ItemCode:    item,
		MessageType: entity.ExceptionCancel,
		SourceCode:  &code,
		SourceType:  &tipo,
		Description: fmt.Sprintf(
			"%s: ordem firme de %s un. não cobre nenhuma necessidade deste plano. Considere cancelar.",
			rotulo, qtd(e.Quantity)),
	}
}

// timingMessage compara a chegada da ordem com a data da necessidade que ela
// cobre: atrasada até expediteThreshold dias é acelerar, mais que isso é
// antecipar; adiantada além de tooEarlyDays é postergar.
func timingMessage(item int64, rotulo string, e ports.SupplyEntry, precisaEm time.Time) *entity.MRPExceptionMessage {
	code, tipo := e.SourceCode, string(e.SourceType)
	sugerida := precisaEm
	m := &entity.MRPExceptionMessage{ItemCode: item, SourceCode: &code, SourceType: &tipo, SuggestedDate: &sugerida}
	switch {
	case e.ArrivalDate.After(precisaEm):
		dias := int(math.Ceil(e.ArrivalDate.Sub(precisaEm).Hours() / 24))
		if dias <= expediteThreshold {
			m.MessageType = entity.ExceptionExpedite
			m.Description = fmt.Sprintf(
				"%s: ordem firme de %s un. chega em %s, %d dia(s) após a necessidade de %s. ACELERAR urgentemente.",
				rotulo, qtd(e.Quantity), e.ArrivalDate.Format("02/01/2006"), dias, precisaEm.Format("02/01/2006"))
		} else {
			m.MessageType = entity.ExceptionRescheduleIn
			m.Description = fmt.Sprintf(
				"%s: ordem firme de %s un. chega em %s, mas a necessidade é %s (%d dia(s) de atraso). Antecipar.",
				rotulo, qtd(e.Quantity), e.ArrivalDate.Format("02/01/2006"), precisaEm.Format("02/01/2006"), dias)
		}
		return m
	case precisaEm.Sub(e.ArrivalDate).Hours()/24 > tooEarlyDays:
		dias := int(math.Ceil(precisaEm.Sub(e.ArrivalDate).Hours() / 24))
		m.MessageType = entity.ExceptionRescheduleOut
		m.Description = fmt.Sprintf(
			"%s: ordem firme de %s un. chega em %s, %d dia(s) antes da necessidade (%s). Atrasar para liberar capital.",
			rotulo, qtd(e.Quantity), e.ArrivalDate.Format("02/01/2006"), dias, precisaEm.Format("02/01/2006"))
		return m
	}
	return nil
}
