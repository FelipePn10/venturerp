package mrp_calculation_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

type peggingStore interface {
	PeggingRows(ctx context.Context, planCode int64) ([]entity.PeggingRecord, error)
	PeggingLabels(ctx context.Context, refs entity.PeggingRefs) (*entity.PeggingLabels, error)
}

// PeggingUseCase consulta o pegging do último cálculo de um plano: de um
// pedido/demanda até as compras dos componentes (para baixo) e de uma ordem até
// os pedidos que dependem dela (para cima) — o "pegged requirements" do SAP
// MD04/MD09 e o pegging do Oracle.
type PeggingUseCase struct {
	Store peggingStore
	Auth  ports.AuthService
	// Now permite fixar o relógio nos testes (decide o que "já devia ter
	// começado").
	Now func() time.Time
}

func (uc *PeggingUseCase) hoje() time.Time {
	t := time.Now()
	if uc.Now != nil {
		t = uc.Now()
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// PeggingEnd é uma ponta do vínculo (necessidade ou suprimento).
type PeggingEnd struct {
	Kind        string     `json:"kind"`
	Code        *int64     `json:"code,omitempty"`
	Label       string     `json:"label"`
	OrderNumber *int64     `json:"order_number,omitempty"`
	OrderType   string     `json:"order_type,omitempty"`
	Date        *time.Time `json:"date,omitempty"`
}

// PeggingNode é um vínculo da cadeia. Para baixo, Children são os componentes
// que a sugestão do nó precisou; para cima, são as necessidades (do pai) que
// a ordem do nó atende.
type PeggingNode struct {
	ItemCode        int64      `json:"item_code"`
	ItemDescription string     `json:"item_description"`
	UOM             string     `json:"uom,omitempty"`
	NeedDate        time.Time  `json:"need_date"`
	Quantity        float64    `json:"quantity"`
	Demand          PeggingEnd `json:"demand"`
	Supply          PeggingEnd `json:"supply"`
	// Late: este elo atrasa a cadeia — a ordem termina depois da necessidade
	// (capacidade) ou já devia ter começado. LateReason diz qual.
	Late       bool          `json:"late"`
	LateReason string        `json:"late_reason,omitempty"`
	Children   []PeggingNode `json:"children"`
}

// PeggingRoot é uma necessidade de origem (pedido, demanda, previsão…) com o
// resumo da cadeia: quanto saiu do estoque, de firmes, de sugestões e se falta
// algo em qualquer nível.
type PeggingRoot struct {
	ItemCode        int64      `json:"item_code"`
	ItemDescription string     `json:"item_description"`
	UOM             string     `json:"uom,omitempty"`
	NeedDate        time.Time  `json:"need_date"`
	Quantity        float64    `json:"quantity"`
	Demand          PeggingEnd `json:"demand"`
	FromStock       float64    `json:"from_stock"`
	FromFirm        float64    `json:"from_firm"`
	FromSuggestions float64    `json:"from_suggestions"`
	Shortage        float64    `json:"shortage"`
	// ChainShortage: algum componente da cadeia ficou sem cobertura.
	ChainShortage bool `json:"chain_shortage"`
	// ChainLate: algum elo da cadeia atrasa (ordem que termina depois da
	// necessidade ou que já devia ter começado); LateReason é o primeiro.
	ChainLate  bool   `json:"chain_late"`
	LateReason string `json:"late_reason,omitempty"`
	// LatestSupply é a data mais tarde de suprimento da cadeia — até lá o
	// pedido não fica completo.
	LatestSupply *time.Time `json:"latest_supply,omitempty"`
}

type peggingIndex struct {
	rows     []entity.PeggingRecord
	byDemand map[string][]int
	bySupply map[string][]int
	labels   *entity.PeggingLabels
	hoje     time.Time
}

func peggingKey(kind string, code *int64) string {
	if code == nil {
		return kind + ":"
	}
	return fmt.Sprintf("%s:%d", kind, *code)
}

func (uc *PeggingUseCase) load(ctx context.Context, planCode int64) (*peggingIndex, error) {
	if !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if planCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o plano")
	}
	rows, err := uc.Store.PeggingRows(ctx, planCode)
	if err != nil {
		return nil, err
	}
	idx := &peggingIndex{rows: rows, byDemand: map[string][]int{}, bySupply: map[string][]int{}, hoje: uc.hoje()}
	refs := entity.PeggingRefs{}
	vistos := map[string]bool{}
	marca := func(chave string) bool {
		if vistos[chave] {
			return false
		}
		vistos[chave] = true
		return true
	}
	for i, r := range rows {
		idx.byDemand[peggingKey(r.DemandKind, r.DemandCode)] = append(idx.byDemand[peggingKey(r.DemandKind, r.DemandCode)], i)
		idx.bySupply[peggingKey(r.SupplyKind, r.SupplyCode)] = append(idx.bySupply[peggingKey(r.SupplyKind, r.SupplyCode)], i)
		if marca(fmt.Sprintf("i%d", r.ItemCode)) {
			refs.Items = append(refs.Items, r.ItemCode)
		}
		if r.DemandCode != nil {
			switch r.DemandKind {
			case entity.PegDemandSuggestion:
				if marca(fmt.Sprintf("s%d", *r.DemandCode)) {
					refs.Suggestions = append(refs.Suggestions, *r.DemandCode)
				}
			case entity.PegDemandSalesOrder:
				if marca(fmt.Sprintf("p%d", *r.DemandCode)) {
					refs.SalesOrderItems = append(refs.SalesOrderItems, *r.DemandCode)
				}
			}
		}
		if r.SupplyCode != nil {
			switch r.SupplyKind {
			case entity.PegSupplySuggestion:
				if marca(fmt.Sprintf("s%d", *r.SupplyCode)) {
					refs.Suggestions = append(refs.Suggestions, *r.SupplyCode)
				}
			case entity.PegSupplyFirmOrder:
				if marca(fmt.Sprintf("f%d", *r.SupplyCode)) {
					refs.FirmOrders = append(refs.FirmOrders, *r.SupplyCode)
				}
			}
		}
	}
	if idx.labels, err = uc.Store.PeggingLabels(ctx, refs); err != nil {
		return nil, err
	}
	return idx, nil
}

var tipoOrdem = map[string]string{
	"FABRICACAO": "fabricação", "COMPRA": "compra", "SERVICO": "serviço",
	"PRODUCTION": "fabricação", "PURCHASE": "compra", "TECHNICAL_ASSISTANCE": "assistência técnica",
}

func ordemLabel(prefixo string, l entity.PeggingOrderLabel, code int64) string {
	tipo := tipoOrdem[l.OrderType]
	if tipo == "" {
		tipo = strings.ToLower(l.OrderType)
	}
	if l.OrderNumber != nil {
		return fmt.Sprintf("%s %d · %s", prefixo, *l.OrderNumber, tipo)
	}
	return fmt.Sprintf("%s %d · %s", prefixo, code, tipo)
}

func (idx *peggingIndex) demandEnd(kind string, code *int64) PeggingEnd {
	e := PeggingEnd{Kind: kind, Code: code}
	switch kind {
	case entity.PegDemandSalesOrder:
		e.Label = "Pedido de venda"
		if code != nil {
			if l, ok := idx.labels.SalesOrderItems[*code]; ok && l.OrderNumber != nil {
				e.OrderNumber = l.OrderNumber
				e.Label = fmt.Sprintf("Pedido %d", *l.OrderNumber)
				if l.Customer != "" {
					e.Label += " · " + l.Customer
				}
			}
		}
	case entity.PegDemandIndependent:
		e.Label = "Demanda independente"
		if code != nil {
			e.Label = fmt.Sprintf("Demanda independente %d", *code)
		}
	case entity.PegDemandSuggestion:
		e.Label = "Ordem do item pai"
		if code != nil {
			if l, ok := idx.labels.Suggestions[*code]; ok {
				e.Label = ordemLabel("Sugestão", l, *code)
				e.OrderNumber, e.OrderType, e.Date = l.OrderNumber, l.OrderType, l.StartDate
			}
		}
	case entity.PegDemandForecast:
		e.Label = "Previsão de vendas"
	case entity.PegDemandMPS:
		e.Label = "Plano mestre (MPS)"
	case entity.PegDemandSafetyStock:
		e.Label = "Estoque de segurança"
	case entity.PegDemandSimulated:
		e.Label = "Demanda simulada"
	default:
		e.Label = "Outra necessidade"
	}
	return e
}

func (idx *peggingIndex) supplyEnd(r entity.PeggingRecord) PeggingEnd {
	e := PeggingEnd{Kind: r.SupplyKind, Code: r.SupplyCode, Date: r.SupplyDate}
	switch r.SupplyKind {
	case entity.PegSupplyStock:
		e.Label = "Estoque"
	case entity.PegSupplyShortage:
		e.Label = "Sem cobertura"
	case entity.PegSupplySuggestion:
		e.Label = "Sugestão"
		if r.SupplyCode != nil {
			if l, ok := idx.labels.Suggestions[*r.SupplyCode]; ok {
				e.Label = ordemLabel("Sugestão", l, *r.SupplyCode)
				e.OrderNumber, e.OrderType = l.OrderNumber, l.OrderType
				// A data do suprimento é quando ele fica pronto: se a máquina
				// só termina depois da necessidade, vale o fim programado.
				if l.EstimatedEnd != nil {
					if fim := diaDoFim(l); e.Date == nil || fim.After(*e.Date) {
						e.Date = &fim
					}
				}
			}
		}
	case entity.PegSupplyFirmOrder:
		e.Label = "Ordem firme"
		if r.SupplyCode != nil {
			if l, ok := idx.labels.FirmOrders[*r.SupplyCode]; ok {
				e.Label = ordemLabel("Ordem firme", l, *r.SupplyCode)
				e.OrderNumber, e.OrderType = l.OrderNumber, l.OrderType
			}
		}
	}
	return e
}

func (idx *peggingIndex) node(r entity.PeggingRecord, qtd float64) PeggingNode {
	item := idx.labels.Items[r.ItemCode]
	n := PeggingNode{
		ItemCode: r.ItemCode, ItemDescription: item.Description, UOM: item.UOM,
		NeedDate: r.NeedDate, Quantity: qtd,
		Demand: idx.demandEnd(r.DemandKind, r.DemandCode), Supply: idx.supplyEnd(r),
		Children: []PeggingNode{},
	}
	n.Late, n.LateReason = idx.atraso(r)
	return n
}

// atraso: a sugestão que cobre esta necessidade termina depois dela na
// programação da máquina (falta de capacidade) ou já devia ter começado.
func (idx *peggingIndex) atraso(r entity.PeggingRecord) (bool, string) {
	if r.SupplyKind != entity.PegSupplySuggestion || r.SupplyCode == nil {
		return false, ""
	}
	l, ok := idx.labels.Suggestions[*r.SupplyCode]
	if !ok {
		return false, ""
	}
	// Mesmo critério de "termina atrasada" das sugestões (mrp_enhanced_repo).
	if l.EstimatedEnd != nil && l.EstimatedEnd.After(r.NeedDate.AddDate(0, 0, 1)) {
		return true, fmt.Sprintf("a fabricação termina em %s, depois da necessidade (falta de capacidade)", diaDoFim(l).Format("02/01/2006"))
	}
	if l.StartDate != nil && l.StartDate.Before(idx.hoje) {
		return true, fmt.Sprintf("a ordem devia ter começado em %s", l.StartDate.Format("02/01/2006"))
	}
	return false, ""
}

// diaDoFim é o dia em que a sugestão fica pronta na programação da máquina.
func diaDoFim(l entity.PeggingOrderLabel) time.Time {
	if l.EstimatedEndDay != nil {
		return *l.EstimatedEndDay
	}
	return *l.EstimatedEnd
}

// suggestionTotal é a quantidade da sugestão — base para repartir os
// componentes quando ela atende mais de uma necessidade (sobra de lote).
func (idx *peggingIndex) suggestionTotal(code int64) float64 {
	if l, ok := idx.labels.Suggestions[code]; ok && l.Quantity > 0 {
		return l.Quantity
	}
	total := 0.0
	for _, i := range idx.bySupply[peggingKey(entity.PegSupplySuggestion, &code)] {
		total += idx.rows[i].Quantity
	}
	return total
}

const peggingMaxDepth = 40

// down monta a cadeia abaixo das linhas: cada sugestão abre as necessidades
// dos componentes que ela gerou, na proporção do que coube a esta origem.
func (idx *peggingIndex) down(linhas []int, fator float64, depth int, visit map[int64]bool) []PeggingNode {
	out := []PeggingNode{}
	for _, i := range linhas {
		r := idx.rows[i]
		n := idx.node(r, r.Quantity*fator)
		if r.SupplyKind == entity.PegSupplySuggestion && r.SupplyCode != nil && depth < peggingMaxDepth && !visit[*r.SupplyCode] {
			total := idx.suggestionTotal(*r.SupplyCode)
			if total > 0 {
				visit[*r.SupplyCode] = true
				filhos := idx.byDemand[peggingKey(entity.PegDemandSuggestion, r.SupplyCode)]
				n.Children = idx.down(filhos, fator*r.Quantity/total, depth+1, visit)
				delete(visit, *r.SupplyCode)
			}
		}
		out = append(out, n)
	}
	return out
}

// up monta, a partir das linhas, quem precisa delas: a necessidade de cada
// linha, e se ela vem de uma sugestão do pai, as linhas que essa sugestão
// atende — até o pedido/demanda de origem.
func (idx *peggingIndex) up(linhas []int, depth int, visit map[int64]bool) []PeggingNode {
	out := []PeggingNode{}
	for _, i := range linhas {
		r := idx.rows[i]
		n := idx.node(r, r.Quantity)
		if r.DemandKind == entity.PegDemandSuggestion && r.DemandCode != nil && depth < peggingMaxDepth && !visit[*r.DemandCode] {
			visit[*r.DemandCode] = true
			n.Children = idx.up(idx.bySupply[peggingKey(entity.PegSupplySuggestion, r.DemandCode)], depth+1, visit)
			delete(visit, *r.DemandCode)
		}
		out = append(out, n)
	}
	return out
}

// Down: cadeia de uma necessidade de origem (pedido, demanda…) até as compras.
// item e data distinguem origens sem código (previsão, estoque de segurança).
func (uc *PeggingUseCase) Down(ctx context.Context, planCode int64, kind string, code *int64, item *int64, need *time.Time) ([]PeggingNode, error) {
	idx, err := uc.load(ctx, planCode)
	if err != nil {
		return nil, err
	}
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if kind == "" {
		return nil, errorsuc.NewValidationError("informe o tipo da necessidade")
	}
	return idx.downFor(kind, code, item, need), nil
}

func (idx *peggingIndex) downFor(kind string, code *int64, item *int64, need *time.Time) []PeggingNode {
	linhas := []int{}
	for _, i := range idx.byDemand[peggingKey(kind, code)] {
		r := idx.rows[i]
		if item != nil && r.ItemCode != *item {
			continue
		}
		if need != nil && !sameDay(r.NeedDate, *need) {
			continue
		}
		linhas = append(linhas, i)
	}
	return idx.down(linhas, 1, 0, map[int64]bool{})
}

// Up: de uma ordem (sugestão ou ordem firme) até os pedidos/demandas que
// dependem dela.
func (uc *PeggingUseCase) Up(ctx context.Context, planCode int64, supplyKind string, supplyCode int64) ([]PeggingNode, error) {
	idx, err := uc.load(ctx, planCode)
	if err != nil {
		return nil, err
	}
	supplyKind = strings.ToUpper(strings.TrimSpace(supplyKind))
	if supplyKind != entity.PegSupplySuggestion && supplyKind != entity.PegSupplyFirmOrder {
		return nil, errorsuc.NewValidationError("informe uma sugestão ou ordem firme")
	}
	return idx.up(idx.bySupply[peggingKey(supplyKind, &supplyCode)], 0, map[int64]bool{}), nil
}

// Roots lista as necessidades de origem do plano com o resumo da cadeia.
func (uc *PeggingUseCase) Roots(ctx context.Context, planCode int64) ([]PeggingRoot, error) {
	idx, err := uc.load(ctx, planCode)
	if err != nil {
		return nil, err
	}
	return idx.roots(), nil
}

func (idx *peggingIndex) roots() []PeggingRoot {
	type chave struct {
		kind, code string
		item       int64
		dia        string
	}
	porRaiz := map[chave]*PeggingRoot{}
	linhasDa := map[chave][]int{}
	ordem := []chave{}
	for i, r := range idx.rows {
		if r.DemandKind == entity.PegDemandSuggestion {
			continue
		}
		k := chave{r.DemandKind, peggingKey("", r.DemandCode), r.ItemCode, r.NeedDate.Format("2006-01-02")}
		root := porRaiz[k]
		if root == nil {
			item := idx.labels.Items[r.ItemCode]
			root = &PeggingRoot{ItemCode: r.ItemCode, ItemDescription: item.Description, UOM: item.UOM,
				NeedDate: r.NeedDate, Demand: idx.demandEnd(r.DemandKind, r.DemandCode)}
			porRaiz[k] = root
			ordem = append(ordem, k)
		}
		root.Quantity += r.Quantity
		switch r.SupplyKind {
		case entity.PegSupplyStock:
			root.FromStock += r.Quantity
		case entity.PegSupplyFirmOrder:
			root.FromFirm += r.Quantity
		case entity.PegSupplySuggestion:
			root.FromSuggestions += r.Quantity
		case entity.PegSupplyShortage:
			root.Shortage += r.Quantity
		}
		linhasDa[k] = append(linhasDa[k], i)
	}
	out := make([]PeggingRoot, 0, len(ordem))
	for _, k := range ordem {
		root := porRaiz[k]
		arvore := idx.down(linhasDa[k], 1, 0, map[int64]bool{})
		root.ChainShortage, root.LatestSupply = resumoCadeia(arvore, root.Shortage > 0, nil)
		root.ChainLate, root.LateReason = primeiroAtraso(arvore)
		out = append(out, *root)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].NeedDate.Equal(out[j].NeedDate) {
			return out[i].ItemCode < out[j].ItemCode
		}
		return out[i].NeedDate.Before(out[j].NeedDate)
	})
	return out
}

func resumoCadeia(nos []PeggingNode, falta bool, ultima *time.Time) (bool, *time.Time) {
	for _, n := range nos {
		if n.Supply.Kind == entity.PegSupplyShortage {
			falta = true
		}
		if n.Supply.Date != nil && (ultima == nil || n.Supply.Date.After(*ultima)) {
			d := *n.Supply.Date
			ultima = &d
		}
		falta, ultima = resumoCadeia(n.Children, falta, ultima)
	}
	return falta, ultima
}

// primeiroAtraso devolve o primeiro elo atrasado da cadeia (em profundidade).
func primeiroAtraso(nos []PeggingNode) (bool, string) {
	for _, n := range nos {
		if n.Late {
			return true, n.rotulo() + ": " + n.LateReason
		}
		if ok, motivo := primeiroAtraso(n.Children); ok {
			return true, motivo
		}
	}
	return false, ""
}

// rotulo identifica o item do elo na mensagem de atraso.
func (n PeggingNode) rotulo() string {
	if n.ItemDescription != "" {
		return n.ItemDescription
	}
	return fmt.Sprintf("item %d", n.ItemCode)
}

func sameDay(a, b time.Time) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

// origens percorre a árvore para cima até as folhas — as necessidades de
// origem (pedido, demanda…) — e devolve os rótulos sem repetir.
func origens(nos []PeggingNode, vistos map[string]bool, out []string) []string {
	for _, n := range nos {
		if len(n.Children) == 0 && n.Demand.Kind != entity.PegDemandSuggestion {
			if !vistos[n.Demand.Label] {
				vistos[n.Demand.Label] = true
				out = append(out, n.Demand.Label)
			}
			continue
		}
		out = origens(n.Children, vistos, out)
	}
	return out
}

// AffectedBySupply devolve, por ordem (sugestão ou ordem firme), os pedidos e
// demandas de origem que dependem dela — o "quem atrasa se esta ordem
// atrasar" que a mensagem de exceção mostra. Uma leitura do pegging serve a
// todas as ordens pedidas.
func (uc *PeggingUseCase) AffectedBySupply(ctx context.Context, planCode int64, supplyKind string, codes []int64) (map[int64][]string, error) {
	out := map[int64][]string{}
	if len(codes) == 0 {
		return out, nil
	}
	idx, err := uc.load(ctx, planCode)
	if err != nil {
		return nil, err
	}
	for _, c := range codes {
		code := c
		if r := origens(idx.up(idx.bySupply[peggingKey(supplyKind, &code)], 0, map[int64]bool{}), map[string]bool{}, nil); len(r) > 0 {
			out[c] = r
		}
	}
	return out, nil
}

type salesOrderPegging interface {
	SalesOrderPeggingPlans(ctx context.Context, orderCode int64) ([]SalesOrderPlan, error)
}

// SalesOrderPlan é um plano cujo último cálculo considerou itens do pedido.
type SalesOrderPlan = entity.SalesOrderPlan

// SalesOrderChain é a cadeia de atendimento de um pedido num plano.
type SalesOrderChain struct {
	SalesOrderPlan
	Roots []PeggingRoot   `json:"roots"`
	Trees [][]PeggingNode `json:"trees"`
}

// SalesOrder devolve, para cada plano que considerou o pedido, a cadeia de
// cada item do pedido até as compras — o "Ver cadeia" a partir do pedido.
func (uc *PeggingUseCase) SalesOrder(ctx context.Context, orderCode int64) ([]SalesOrderChain, error) {
	if !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	repo, ok := uc.Store.(salesOrderPegging)
	if !ok {
		return nil, errorsuc.NewValidationError("o rastreamento por pedido não está disponível")
	}
	planos, err := repo.SalesOrderPeggingPlans(ctx, orderCode)
	if err != nil {
		return nil, err
	}
	out := []SalesOrderChain{}
	for _, p := range planos {
		idx, err := uc.load(ctx, p.PlanCode)
		if err != nil {
			return nil, err
		}
		raizes := idx.roots()
		doPedido := map[int64]bool{}
		for _, c := range p.ItemCodes {
			doPedido[c] = true
		}
		ch := SalesOrderChain{SalesOrderPlan: p, Roots: []PeggingRoot{}, Trees: [][]PeggingNode{}}
		for _, r := range raizes {
			if r.Demand.Kind != entity.PegDemandSalesOrder || r.Demand.Code == nil || !doPedido[*r.Demand.Code] {
				continue
			}
			item := r.ItemCode
			need := r.NeedDate
			arvore := idx.downFor(r.Demand.Kind, r.Demand.Code, &item, &need)
			ch.Roots = append(ch.Roots, r)
			ch.Trees = append(ch.Trees, arvore)
		}
		if len(ch.Roots) > 0 {
			out = append(out, ch)
		}
	}
	return out, nil
}
