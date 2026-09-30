package cost_uc

import (
	"context"
	"fmt"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"sort"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// routingReader is the slice of the routing repository the cost roll-up needs to
// charge each operation at its real work-center rate using the rich time model.
type routingReader interface {
	GetRouteForItem(ctx context.Context, itemCode int64, mask string) (*routingentity.ManufacturingRoute, error)
	GetRouteOperations(ctx context.Context, routeID int64) ([]*routingentity.RouteOperation, error)
}

type StandardCostUseCase struct {
	repo        domainrepo.StandardCostRepository
	routing     routingReader // optional; when nil, falls back to the legacy average-rate estimate
	thirdParty  thirdPartyCostReader
	workCenters workCenterReader
}
type workCenterReader interface {
	ListActiveWorkCenterTypes(context.Context, string, int, int) ([]*machineentity.MachineType, int64, error)
}
type thirdPartyCostReader interface {
	StandardCostPerUnit(context.Context, int64, string, int64, time.Time) (decimal.Decimal, error)
}

func New(repo domainrepo.StandardCostRepository) *StandardCostUseCase {
	return &StandardCostUseCase{repo: repo}
}

func (uc *StandardCostUseCase) WithWorkCenters(reader workCenterReader) *StandardCostUseCase {
	uc.workCenters = reader
	return uc
}

func (uc *StandardCostUseCase) ListWorkCenters(ctx context.Context, search string, limit, offset int) (*response.WorkCenterOptionPageResponse, error) {
	if uc.workCenters == nil {
		return nil, fmt.Errorf("consulta de centros de trabalho não configurada")
	}
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 500 || offset < 0 {
		return nil, errorsuc.NewValidationError("limit deve estar entre 1 e 500 e offset não pode ser negativo")
	}
	rows, total, err := uc.workCenters.ListActiveWorkCenterTypes(ctx, search, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]response.WorkCenterOptionResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, response.WorkCenterOptionResponse{ID: row.ID, Code: row.Code, Name: row.Name, Description: row.Description, IsActive: row.IsActive})
	}
	return &response.WorkCenterOptionPageResponse{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

// WithRouting enables per-operation, per-work-center, quantity-aware labor costing.
func (uc *StandardCostUseCase) WithRouting(r routingReader) *StandardCostUseCase {
	uc.routing = r
	return uc
}
func (uc *StandardCostUseCase) WithThirdPartyPrices(v thirdPartyCostReader) *StandardCostUseCase {
	uc.thirdParty = v
	return uc
}

// ─── price catalog management ─────────────────────────────────────────────────

func (uc *StandardCostUseCase) UpsertWorkCenterCost(ctx context.Context, dto request.UpsertWorkCenterCostDTO) (*response.WorkCenterCostResponse, error) {
	uid, err := uuid.Parse(dto.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid updated_by UUID: %w", err)
	}
	// When the machine × labor split is not provided, the machine rate defaults to the
	// blended cost_per_hour so the stored/displayed value matches the effective rate.
	machineRate := dto.MachineCostPerHour
	if machineRate <= 0 {
		machineRate = dto.CostPerHour
	}
	wcc := &entity.WorkCenterCost{
		WorkCenterID:       dto.WorkCenterID,
		CostPerHour:        dto.CostPerHour,
		MachineCostPerHour: machineRate,
		LaborCostPerHour:   dto.LaborCostPerHour,
		Currency:           dto.Currency,
		UpdatedBy:          uid,
	}
	saved, err := uc.repo.UpsertWorkCenterCost(ctx, wcc)
	if err != nil {
		return nil, err
	}
	return toWCCResponse(saved), nil
}

func (uc *StandardCostUseCase) ListWorkCenterCosts(ctx context.Context) ([]*response.WorkCenterCostResponse, error) {
	wccs, err := uc.repo.ListWorkCenterCosts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.WorkCenterCostResponse, 0, len(wccs))
	for _, w := range wccs {
		out = append(out, toWCCResponse(w))
	}
	return out, nil
}

func (uc *StandardCostUseCase) UpsertItemPurchaseCost(ctx context.Context, dto request.UpsertItemPurchaseCostDTO) (*response.ItemPurchaseCostResponse, error) {
	uid, err := uuid.Parse(dto.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid updated_by UUID: %w", err)
	}
	ipc := &entity.ItemPurchaseCost{
		ItemCode:  dto.ItemCode,
		UnitCost:  dto.UnitCost,
		Currency:  dto.Currency,
		UpdatedBy: uid,
	}
	saved, err := uc.repo.UpsertItemPurchaseCost(ctx, ipc)
	if err != nil {
		return nil, err
	}
	return toIPCResponse(saved), nil
}

func (uc *StandardCostUseCase) GetItemPurchaseCost(ctx context.Context, itemCode int64) (*response.ItemPurchaseCostResponse, error) {
	ipc, err := uc.repo.GetItemPurchaseCost(ctx, itemCode)
	if err != nil {
		return nil, errorsuc.NewNotFoundError("custo de compra não cadastrado para o item")
	}
	return toIPCResponse(ipc), nil
}

// ─── cost rollup ──────────────────────────────────────────────────────────────

// RollUp apura e grava o custo-padrão de itemCode + máscara, descendo a estrutura
// de baixo para cima:
//
//	material        = Σ (custo_total_do_filho × qtd × (1 + perda%))   → NivelInferior
//	setup           = Σ_ops horas_de_setup × taxa                      ÷ lote
//	máquina         = Σ_ops (horas_máquina − setup) × taxa_máquina     ÷ lote
//	mão de obra     = Σ_ops (horas_homem − setup)  × taxa_homem        ÷ lote
//	subcontratação  = Σ_ops preço do terceiro
//	indiretos       = esquema de rateio sobre os componentes desta etapa
//
// O que mudou em relação à versão anterior: `overhead` era gravado SEMPRE ZERO — a
// coluna existia e não havia como configurar —, e a conversão saía num único
// número chamado "labor". Agora os indiretos vêm do esquema de rateio (migração
// 000373) e cada componente é gravado e devolvido separado.
func (uc *StandardCostUseCase) RollUp(ctx context.Context, dto request.CostRollupDTO) (*response.CostRollupResponse, error) {
	calculatedBy, err := uuid.Parse(dto.CalculatedBy)
	if err != nil {
		return nil, errorsuc.NewValidationError("não foi possível identificar o usuário que apurou o custo")
	}

	lotSize := dto.LotSize
	if lotSize <= 0 {
		lotSize = 1
	}

	// As regras de rateio são lidas UMA vez para a apuração inteira: elas valem
	// para toda a estrutura, e reler por nó faria uma consulta por item da árvore.
	quando := time.Now()
	regras := uc.regrasDeRateio(ctx)

	unitCache := make(map[int64]float64)
	result, err := uc.rollupItem(ctx, dto.ItemCode, dto.Mask, 0, lotSize, unitCache, regras, quando)
	if err != nil {
		return nil, err
	}
	c := result.Componentes

	cost := &entity.ItemStandardCost{
		ItemCode: dto.ItemCode,
		Mask:     dto.Mask,
		// MaterialCost guarda o material CHEIO (o que veio de baixo mais o material
		// próprio da folha): é o número que o resto do sistema já lia como
		// "material", e mudar o sentido dele quebraria precificação e relatórios.
		MaterialCost: c.Material + c.NivelInferior,
		LaborCost:    c.MaoDeObra,
		OverheadCost: c.Overhead,
		Currency:     "BRL",
		CalculatedBy: calculatedBy,
	}
	saved, err := uc.repo.UpsertItemStandardCost(ctx, cost)
	if err != nil {
		return nil, fmt.Errorf("gravando o custo-padrão: %w", err)
	}

	// O log da árvore é o que a tela mostra como composição por nível. Falha aqui
	// não invalida a apuração, mas não é engolida: sem o log a tela não explica o
	// número, e a pessoa precisa saber disso.
	var avisos []string
	for i := range result.Detail {
		if err := uc.repo.InsertRollupLog(ctx, &result.Detail[i]); err != nil {
			avisos = append(avisos, "a composição por nível não pôde ser registrada: "+err.Error())
			break
		}
	}

	// Histórico da apuração, para comparar componente a componente ao longo do
	// tempo. Só grava se o repositório suportar (migração 000373).
	total := c.Total() + c.NivelInferior
	if h, ok := uc.repo.(interface {
		GravarHistoricoDeCusto(context.Context, *entity.HistoricoDeCusto) error
	}); ok {
		erroHist := h.GravarHistoricoDeCusto(ctx, &entity.HistoricoDeCusto{
			ItemCode: dto.ItemCode, Mask: dto.Mask, LotSize: lotSize,
			Componentes: c, TotalCost: total, Currency: "BRL",
			Rateios: result.Rateios, CalculatedBy: calculatedBy,
		})
		if erroHist != nil {
			avisos = append(avisos, "o histórico desta apuração não pôde ser gravado: "+erroHist.Error())
		}
	}

	return &response.CostRollupResponse{
		ItemCode:        saved.ItemCode,
		Mask:            saved.Mask,
		MaterialCost:    saved.MaterialCost,
		LaborCost:       c.MaoDeObra,
		OverheadCost:    c.Overhead,
		SetupCost:       c.Setup,
		MachineCost:     c.Maquina,
		SubcontractCost: c.Subcontratacao,
		OwnLevelCost:    c.NivelProprio,
		LowerLevelCost:  c.NivelInferior,
		LotSize:         lotSize,
		TotalCost:       total,
		Currency:        "BRL",
		CalculatedAt:    saved.CalculatedAt,
		Overheads:       paraRateiosResponse(result.Rateios),
		Tree:            montarArvoreDeCusto(result.Detail),
		Avisos:          avisos,
	}, nil
}

// regrasDeRateio lê o esquema de indiretos da empresa. Devolve vazio quando o
// repositório não expõe o esquema: a apuração continua, sem indiretos — e é por
// isso que a tela mostra o rastro dos rateios, para ficar visível quando não houve.
func (uc *StandardCostUseCase) regrasDeRateio(ctx context.Context) []*entity.RegraDeRateio {
	leitor, ok := uc.repo.(interface {
		ListarRegrasDeRateio(context.Context) ([]*entity.RegraDeRateio, error)
	})
	if !ok {
		return nil
	}
	regras, err := leitor.ListarRegrasDeRateio(ctx)
	if err != nil {
		return nil
	}
	return regras
}

// paraRateiosResponse converte o rastro do rateio para o contrato HTTP.
func paraRateiosResponse(rateios []entity.RateioAplicado) []response.CostOverheadAppliedResponse {
	if len(rateios) == 0 {
		return nil
	}
	out := make([]response.CostOverheadAppliedResponse, 0, len(rateios))
	for _, r := range rateios {
		out = append(out, response.CostOverheadAppliedResponse{
			RuleID: r.RuleID, Code: r.Code, Description: r.Description,
			Base: string(r.Base), Method: string(r.Method), Rate: r.Rate,
			BaseValue: r.BaseValue, Applied: r.Applied,
		})
	}
	return out
}

// montarArvoreDeCusto transforma o log plano da apuração na composição por nível
// que a tela mostra. O log já vem em ordem de travessia (pai antes dos filhos).
func montarArvoreDeCusto(detalhe []entity.CostRollupLogEntry) []response.CostRollupNodeResponse {
	out := make([]response.CostRollupNodeResponse, 0, len(detalhe))
	for _, d := range detalhe {
		out = append(out, response.CostRollupNodeResponse{
			ItemCode:        d.ItemCode,
			Mask:            d.Mask,
			Level:           d.BOMLevel,
			MaterialCost:    d.MaterialCost,
			SetupCost:       d.SetupCost,
			MachineCost:     d.MachineCost,
			LaborCost:       d.LaborCost,
			SubcontractCost: d.SubcontractCost,
			OverheadCost:    d.OverheadCost,
			LowerLevelCost:  d.LowerLevelCost,
			TotalCost:       d.Total(),
		})
	}
	return out
}

func (uc *StandardCostUseCase) GetStandardCost(ctx context.Context, itemCode int64, mask string) (*response.CostRollupResponse, error) {
	cost, err := uc.repo.GetItemStandardCost(ctx, itemCode, mask)
	if err != nil {
		return nil, errorsuc.NewNotFoundError("custo-padrão não apurado para o item")
	}
	return &response.CostRollupResponse{
		ItemCode:     cost.ItemCode,
		Mask:         cost.Mask,
		MaterialCost: cost.MaterialCost,
		LaborCost:    cost.LaborCost,
		OverheadCost: cost.OverheadCost,
		TotalCost:    cost.TotalCost,
		Currency:     cost.Currency,
		CalculatedAt: cost.CalculatedAt,
	}, nil
}

// ─── rollup algorithm ─────────────────────────────────────────────────────────

// costNode é o custo de um item na travessia da estrutura, aberto por componente.
// Antes carregava três números (material, "labor", overhead sempre zero); agora
// carrega o que a análise de custo precisa separar, e as horas que alimentam as
// regras de rateio por hora.
type costNode struct {
	Componentes entity.ComponentesDeCusto
	Horas       entity.HorasDoRoteiro
	Rateios     []entity.RateioAplicado
	Detail      []entity.CostRollupLogEntry
}

// total é o custo unitário do item: o que esta etapa agrega MAIS o que veio dos
// níveis abaixo. `ComponentesDeCusto.Total()` soma apenas os componentes próprios,
// de propósito — quem quer o custo cheio soma o nível inferior aqui.
func (n *costNode) total() float64 { return n.Componentes.Total() + n.Componentes.NivelInferior }

// rollupItem desce a estrutura e devolve o custo unitário do item, aberto por
// componente.
//
// O que o nível de cima recebe de um componente é o TOTAL dele (material +
// conversão + indiretos), porque para o pai aquele componente é material
// comprado ou fabricado — não faz sentido somar a hora-máquina do filho na
// hora-máquina do pai. É por isso que o pai guarda o custo dos filhos em
// `NivelInferior` e não espalhado pelos próprios componentes: o corte "o que esta
// etapa agrega × o que veio de baixo" é o que diz onde o custo subiu.
func (uc *StandardCostUseCase) rollupItem(
	ctx context.Context, itemCode int64, mask string, level int, lotSize float64,
	unitCache map[int64]float64, regras []*entity.RegraDeRateio, quando time.Time,
) (*costNode, error) {
	children, err := uc.repo.GetDirectChildren(ctx, itemCode, mask)
	if err != nil {
		return nil, fmt.Errorf("lendo a estrutura do item %d: %w", itemCode, err)
	}

	var comp entity.ComponentesDeCusto
	var childNodes []*costNode

	if len(children) == 0 {
		// Folha: o custo é o de compra. Vai em Material — é o que o item É para
		// quem o consome.
		if cached, ok := unitCache[itemCode]; ok {
			comp.Material = cached
		} else {
			ipc, err2 := uc.repo.GetItemPurchaseCost(ctx, itemCode)
			if err2 == nil {
				comp.Material = ipc.UnitCost
			}
			unitCache[itemCode] = comp.Material
		}
	} else {
		for _, child := range selectPrimaryCostSubstitutes(children) {
			childNode, err2 := uc.rollupItem(ctx, child.ChildCode, mask, level+1, lotSize, unitCache, regras, quando)
			if err2 != nil {
				return nil, err2
			}
			if child.IsCoproduct {
				// Co-produto / sucata retornável: CREDITA o pai pelo valor do
				// co-produto.
				comp.NivelInferior -= childNode.total() * child.QuantidadeNaUnidadeDeEstoque()
				continue
			}
			netQty := structentity.QuantidadeComPerda(child.QuantidadeNaUnidadeDeEstoque(), child.LossPercentage, structentity.FormulaPerdaPadrao)
			if child.IsFixedQty && lotSize > 0 {
				netQty /= lotSize // componente por lote, amortizado pelo lote de referência
			}
			comp.NivelInferior += childNode.total() * netQty
			childNodes = append(childNodes, childNode)
		}
		if comp.NivelInferior < 0 {
			comp.NivelInferior = 0 // crédito de co-produto nunca deixa o custo negativo
		}
	}

	// Conversão desta etapa: setup, máquina, mão de obra e serviço de terceiro.
	conversao, horas, err := uc.conversaoDoRoteiro(ctx, itemCode, mask, lotSize)
	if err != nil {
		return nil, err
	}
	comp.Setup = conversao.Setup
	comp.Maquina = conversao.Maquina
	comp.MaoDeObra = conversao.MaoDeObra
	comp.Subcontratacao = conversao.Subcontratacao

	// Indiretos: o esquema de rateio incide sobre os componentes DESTA etapa. Não
	// incide sobre `NivelInferior` — os indiretos do componente fabricado já foram
	// aplicados na apuração dele, e aplicá-los de novo aqui os cobraria em cascata.
	var rateios []entity.RateioAplicado
	if len(regras) > 0 {
		centros, _ := uc.centrosDoRoteiro(ctx, itemCode, mask)
		aplicaveis := entity.RegrasVigentes(regras, itemCode, centros, quando)
		baseDoRateio := comp
		baseDoRateio.Material = comp.Material // material próprio (folha), não o de baixo
		comp.Overhead, rateios = entity.AplicarRateios(baseDoRateio, horas, aplicaveis)
	}

	comp.NivelProprio = comp.Material + comp.Conversao() + comp.Subcontratacao + comp.Overhead

	node := &costNode{
		Componentes: comp,
		Horas:       horas,
		Rateios:     rateios,
		Detail: []entity.CostRollupLogEntry{{
			ItemCode:        itemCode,
			Mask:            mask,
			BOMLevel:        level,
			MaterialCost:    comp.Material,
			LaborCost:       comp.MaoDeObra,
			OverheadCost:    comp.Overhead,
			SetupCost:       comp.Setup,
			MachineCost:     comp.Maquina,
			SubcontractCost: comp.Subcontratacao,
			LowerLevelCost:  comp.NivelInferior,
		}},
	}
	for _, cn := range childNodes {
		node.Detail = append(node.Detail, cn.Detail...)
	}
	return node, nil
}

// centrosDoRoteiro pergunta ao repositório por quais centros de trabalho o roteiro
// do item passa. Devolve vazio quando o repositório não expõe a consulta: nesse
// caso só as regras de escopo geral aplicam, que é melhor que aplicar regra de
// centro errado.
func (uc *StandardCostUseCase) centrosDoRoteiro(ctx context.Context, itemCode int64, mask string) (map[int64]bool, error) {
	leitor, ok := uc.repo.(interface {
		CentrosDoRoteiro(context.Context, int64, string) (map[int64]bool, error)
	})
	if !ok {
		return map[int64]bool{}, nil
	}
	return leitor.CentrosDoRoteiro(ctx, itemCode, mask)
}

func selectPrimaryCostSubstitutes(children []domainrepo.BOMChild) []domainrepo.BOMChild {
	out := make([]domainrepo.BOMChild, 0, len(children))
	groupBest := make(map[int16]domainrepo.BOMChild)

	for _, child := range children {
		if child.SubstituteGroup <= 0 {
			out = append(out, child)
			continue
		}
		if best, ok := groupBest[child.SubstituteGroup]; !ok || costSubstitutePrecedes(child, best) {
			groupBest[child.SubstituteGroup] = child
		}
	}

	groups := make([]int, 0, len(groupBest))
	for group := range groupBest {
		groups = append(groups, int(group))
	}
	sort.Ints(groups)
	for _, group := range groups {
		out = append(out, groupBest[int16(group)])
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ChildCode < out[j].ChildCode
	})
	return out
}

func costSubstitutePrecedes(a, b domainrepo.BOMChild) bool {
	ap := a.SubstitutePriority
	if ap < 1 {
		ap = 1
	}
	bp := b.SubstitutePriority
	if bp < 1 {
		bp = 1
	}
	if ap != bp {
		return ap < bp
	}
	return a.ChildCode < b.ChildCode
}

// conversaoDoRoteiro devolve o custo de conversão por unidade, ABERTO por
// componente, e as horas por unidade que alimentam as regras de rateio por hora.
//
// Cada operação é cobrada na taxa do SEU centro de trabalho pelo modelo de tempo
// completo, com o setup amortizado pelo lote de referência:
//
//	( Σ [ HorasMáquina(lote) × taxaMáquina(CT) + HorasHomem(lote) × taxaHomem(CT) ] ) ÷ lote
//
// Lote 1 cobra o setup inteiro por peça (conservador). Sem o repositório de
// roteiro ligado, cai na estimativa antiga por taxa média — e nela não há como
// separar máquina de mão de obra, então o valor vai inteiro para máquina, que é a
// leitura menos enganosa das duas.
//
// O setup sai separado porque é a informação que muda a decisão: setup alto pede
// lote maior, hora-máquina alta pede outro recurso. Lançar os dois no mesmo número
// esconde qual é o problema.
func (uc *StandardCostUseCase) conversaoDoRoteiro(ctx context.Context, itemCode int64, mask string, lot float64) (entity.ComponentesDeCusto, entity.HorasDoRoteiro, error) {
	var comp entity.ComponentesDeCusto
	var horas entity.HorasDoRoteiro
	if lot <= 0 {
		lot = 1
	}
	if uc.routing == nil {
		routeHours, _ := uc.repo.GetRouteHoursByItem(ctx, itemCode, mask)
		comp.Maquina = routeHours * uc.averageRate(ctx)
		horas.Maquina = routeHours
		return comp, horas, nil
	}

	route, err := uc.routing.GetRouteForItem(ctx, itemCode, mask)
	if err != nil || route == nil {
		return comp, horas, nil // sem roteiro, não há conversão
	}
	ops, err := uc.routing.GetRouteOperations(ctx, route.ID)
	if err != nil || len(ops) == 0 {
		return comp, horas, nil
	}

	rates := uc.workCenterRates(ctx)

	for _, op := range ops {
		if op.Situation == routingentity.RouteOpGhost {
			continue // operação fantasma não custa
		}
		if (op.OperationOrigin == routingentity.OriginExternal || op.OperationOrigin == routingentity.OriginThirdPart) && uc.thirdParty != nil {
			serviceCost, priceErr := uc.thirdParty.StandardCostPerUnit(ctx, itemCode, mask, op.OperationID, time.Now())
			if priceErr != nil {
				// Mensagem chega ao usuário: o prefixo em inglês vazava na tela de
				// apuração de custo.
				return comp, horas, fmt.Errorf("custo de terceiro do item %d, operação %d: %w", itemCode, op.OperationID, priceErr)
			}
			// O preço do terceiro já é por unidade; multiplicar pelo lote e dividir
			// depois mantém a mesma unidade do resto da soma.
			comp.Subcontratacao += serviceCost.InexactFloat64() * lot
			continue
		}
		machineRate, laborRate := 0.0, 0.0
		if op.EffectiveWorkCenterID != nil {
			if wc, ok := rates[*op.EffectiveWorkCenterID]; ok {
				machineRate = wc.MachineRate()
				laborRate = wc.LaborRate()
			}
		}
		// O setup está DENTRO de MachineHours e de LaborHours (ambos somam t.Setup).
		// Subtraí-lo para formar o componente próprio é o que impede cobrar a
		// preparação duas vezes — em máquina e em setup.
		setupMaquina := op.EffTime.SetupMachineHours()
		setupHomem := op.EffTime.SetupLaborHours()
		maquinaSemSetup := op.EffTime.MachineHours(lot) - setupMaquina
		homemSemSetup := op.EffTime.LaborHours(lot) - setupHomem
		if maquinaSemSetup < 0 {
			maquinaSemSetup = 0
		}
		if homemSemSetup < 0 {
			homemSemSetup = 0
		}

		comp.Setup += setupMaquina*machineRate + setupHomem*laborRate
		comp.Maquina += maquinaSemSetup * machineRate
		comp.MaoDeObra += homemSemSetup * laborRate
		horas.Setup += setupMaquina + setupHomem
		horas.Maquina += maquinaSemSetup
		horas.MaoDeObra += homemSemSetup
	}

	// Tudo acumulado para o LOTE; o custo-padrão é unitário.
	comp.Setup /= lot
	comp.Maquina /= lot
	comp.MaoDeObra /= lot
	comp.Subcontratacao /= lot
	horas.Setup /= lot
	horas.Maquina /= lot
	horas.MaoDeObra /= lot
	return comp, horas, nil
}

// workCenterRates indexes the configured work-center costs by work-center id.
func (uc *StandardCostUseCase) workCenterRates(ctx context.Context) map[int64]*entity.WorkCenterCost {
	wccs, err := uc.repo.ListWorkCenterCosts(ctx)
	m := make(map[int64]*entity.WorkCenterCost, len(wccs))
	if err != nil {
		return m
	}
	for _, w := range wccs {
		m[w.WorkCenterID] = w
	}
	return m
}

// averageRate is the legacy fallback used only when the routing repo is not wired.
func (uc *StandardCostUseCase) averageRate(ctx context.Context) float64 {
	wccs, err := uc.repo.ListWorkCenterCosts(ctx)
	if err != nil || len(wccs) == 0 {
		return 0
	}
	var total float64
	for _, w := range wccs {
		total += w.CostPerHour
	}
	return total / float64(len(wccs))
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func toWCCResponse(w *entity.WorkCenterCost) *response.WorkCenterCostResponse {
	return &response.WorkCenterCostResponse{
		ID:                 w.ID,
		WorkCenterID:       w.WorkCenterID,
		CostPerHour:        w.CostPerHour,
		MachineCostPerHour: w.MachineCostPerHour,
		LaborCostPerHour:   w.LaborCostPerHour,
		Currency:           w.Currency,
		UpdatedAt:          w.UpdatedAt,
	}
}

func toIPCResponse(i *entity.ItemPurchaseCost) *response.ItemPurchaseCostResponse {
	return &response.ItemPurchaseCostResponse{
		ID:        i.ID,
		ItemCode:  i.ItemCode,
		UnitCost:  i.UnitCost,
		Currency:  i.Currency,
		UpdatedAt: i.UpdatedAt,
	}
}
