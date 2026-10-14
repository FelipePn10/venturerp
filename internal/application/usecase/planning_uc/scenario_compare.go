package planning_uc

import (
	"context"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
)

// PlanSummary resume o resultado de um plano para comparar.
type PlanSummary struct {
	PlanCode         int64      `json:"plan_code"`
	CalculatedAt     *time.Time `json:"calculated_at,omitempty"`
	Suggestions      int        `json:"suggestions"`
	ProductionQty    float64    `json:"production_qty"`
	PurchaseQty      float64    `json:"purchase_qty"`
	CoverageCritical int        `json:"coverage_critical"`
	CoverageWarning  int        `json:"coverage_warning"`
	CoverageOK       int        `json:"coverage_ok"`
	Exceptions       int        `json:"exceptions"`
	OverloadDays     int        `json:"overload_days"`
	MaxLoadPct       float64    `json:"max_load_pct"`
	Roots            int        `json:"roots"`
	RootsShort       int        `json:"roots_with_shortage"`
	// RootsLate: pedidos/demandas cuja cadeia tem elo atrasado (ordem que
	// termina depois da necessidade ou já devia ter começado).
	RootsLate int `json:"roots_late"`
}

// ItemDiff é a diferença de um item entre o plano e o cenário.
type ItemDiff struct {
	ItemCode          int64      `json:"item_code"`
	Description       string     `json:"description"`
	BasePlanned       float64    `json:"base_planned"`
	ScenarioPlanned   float64    `json:"scenario_planned"`
	BaseStatus        string     `json:"base_status"`
	ScenarioStatus    string     `json:"scenario_status"`
	BaseShortage      float64    `json:"base_shortage"`
	ScenarioShortage  float64    `json:"scenario_shortage"`
	BaseShortageDate  *time.Time `json:"base_shortage_date,omitempty"`
	ScenarioShortDate *time.Time `json:"scenario_shortage_date,omitempty"`
}

// WorkCenterDiff é a diferença de carga de um centro de trabalho.
type WorkCenterDiff struct {
	WorkCenterID     int64   `json:"work_center_id"`
	Name             string  `json:"name"`
	BaseHours        float64 `json:"base_hours"`
	ScenarioHours    float64 `json:"scenario_hours"`
	BaseOverloadDays int     `json:"base_overload_days"`
	ScenarioOverDays int     `json:"scenario_overload_days"`
	BaseMaxLoadPct   float64 `json:"base_max_load_pct"`
	ScenarioMaxLoad  float64 `json:"scenario_max_load_pct"`
}

// ScenarioComparison é o lado a lado do plano real com o cenário.
type ScenarioComparison struct {
	Base        PlanSummary      `json:"base"`
	Scenario    PlanSummary      `json:"scenario"`
	Items       []ItemDiff       `json:"items"`
	WorkCenters []WorkCenterDiff `json:"work_centers"`
	// BaseStale: o plano base foi calculado antes da última alteração do
	// cenário — recalcule o plano real para comparar com a fábrica de hoje.
	BaseStale bool     `json:"base_stale"`
	Notes     []string `json:"notes"`
}

type planResult struct {
	summary  PlanSummary
	coverage map[int64]mrp_report_uc.CoverageRow
	planned  map[int64]float64
	wc       map[int64]*WorkCenterDiff // só as horas/dias/máx do lado
}

func (uc *ScenarioUseCase) planResult(ctx context.Context, plan int64) (*planResult, error) {
	out := &planResult{summary: PlanSummary{PlanCode: plan}, coverage: map[int64]mrp_report_uc.CoverageRow{}, planned: map[int64]float64{}, wc: map[int64]*WorkCenterDiff{}}
	if uc.Plans != nil {
		at, err := uc.Plans.LastCalculatedAt(ctx, plan)
		if err != nil {
			return nil, err
		}
		out.summary.CalculatedAt = at
	}
	sugs, err := uc.Results.ListSuggestionsByPlan(ctx, plan)
	if err != nil {
		return nil, err
	}
	for _, s := range sugs {
		switch s.OrderType {
		case "FABRICACAO":
			out.summary.ProductionQty += s.Quantity
		case "COMPRA":
			out.summary.PurchaseQty += s.Quantity
		default:
			continue
		}
		out.summary.Suggestions++
		out.planned[s.ItemCode] += s.Quantity
	}
	excs, err := uc.Results.ListExceptionsByPlan(ctx, plan)
	if err != nil {
		return nil, err
	}
	out.summary.Exceptions = len(excs)
	cov, err := uc.Coverage.Coverage(ctx, mrp_report_uc.Filter{PlanCode: &plan}, uc.now())
	if err != nil {
		return nil, err
	}
	for _, c := range cov {
		out.coverage[c.ItemCode] = c
		switch c.Status {
		case mrp_report_uc.CoverageCritical:
			out.summary.CoverageCritical++
		case mrp_report_uc.CoverageWarning:
			out.summary.CoverageWarning++
		default:
			out.summary.CoverageOK++
		}
	}
	crp, err := uc.CRP.ListByPlan(ctx, plan)
	if err != nil {
		return nil, err
	}
	for _, e := range crp {
		w := out.wc[e.WorkCenterID]
		if w == nil {
			w = &WorkCenterDiff{WorkCenterID: e.WorkCenterID}
			out.wc[e.WorkCenterID] = w
		}
		w.BaseHours += e.RequiredHours
		if e.RequiredHours > e.AvailableHours {
			w.BaseOverloadDays++
			out.summary.OverloadDays++
		}
		if e.LoadPct > w.BaseMaxLoadPct {
			w.BaseMaxLoadPct = e.LoadPct
		}
		if e.LoadPct > out.summary.MaxLoadPct {
			out.summary.MaxLoadPct = e.LoadPct
		}
	}
	if uc.Pegging != nil {
		roots, err := uc.Pegging.Roots(ctx, plan)
		if err != nil {
			return nil, err
		}
		out.summary.Roots = len(roots)
		for _, r := range roots {
			if r.ChainShortage {
				out.summary.RootsShort++
			}
			if r.ChainLate {
				out.summary.RootsLate++
			}
		}
	}
	return out, nil
}

// Compare põe o plano real e o cenário lado a lado: ordens, cobertura,
// exceções, carga dos centros e pedidos com falta na cadeia. Itens e centros
// só entram quando algo muda.
func (uc *ScenarioUseCase) Compare(ctx context.Context, id int64) (*ScenarioComparison, error) {
	if err := uc.canRead(ctx); err != nil {
		return nil, err
	}
	s, err := uc.Repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.CalculatedAt == nil {
		return nil, errorsuc.NewValidationError("calcule o cenário antes de comparar")
	}
	base, err := uc.planResult(ctx, s.BasePlanCode)
	if err != nil {
		return nil, err
	}
	cen, err := uc.planResult(ctx, s.ScenarioPlanCode)
	if err != nil {
		return nil, err
	}
	out := &ScenarioComparison{Base: base.summary, Scenario: cen.summary, Items: []ItemDiff{}, WorkCenters: []WorkCenterDiff{}, Notes: []string{}}
	if base.summary.CalculatedAt == nil {
		out.BaseStale = true
		out.Notes = append(out.Notes, "O plano base nunca foi calculado: calcule o plano real para comparar.")
	} else if s.CalculatedAt.Sub(*base.summary.CalculatedAt) > 24*time.Hour {
		out.BaseStale = true
		out.Notes = append(out.Notes, "O plano base foi calculado mais de um dia antes do cenário: recalcule o plano real para comparar com a situação de hoje.")
	}
	if len(base.wc) == 0 && len(cen.wc) > 0 {
		out.Notes = append(out.Notes, "O plano base não tem carga de capacidade calculada: rode o pipeline (MRP → CRP → APS) do plano real para comparar a carga dos centros.")
	}
	if s.Status == ScenarioDraft {
		out.Notes = append(out.Notes, "Os ajustes mudaram depois do último cálculo do cenário: calcule de novo para ver o efeito.")
	}

	itens := map[int64]bool{}
	for k := range base.coverage {
		itens[k] = true
	}
	for k := range cen.coverage {
		itens[k] = true
	}
	for k := range base.planned {
		itens[k] = true
	}
	for k := range cen.planned {
		itens[k] = true
	}
	for item := range itens {
		b, c := base.coverage[item], cen.coverage[item]
		d := ItemDiff{ItemCode: item, Description: c.Description, BasePlanned: base.planned[item], ScenarioPlanned: cen.planned[item],
			BaseStatus: b.Status, ScenarioStatus: c.Status, BaseShortageDate: b.FirmShortageDate, ScenarioShortDate: c.FirmShortageDate}
		if d.Description == "" {
			d.Description = b.Description
		}
		d.BaseShortage, _ = b.FirmShortageQty.Float64()
		d.ScenarioShortage, _ = c.FirmShortageQty.Float64()
		mudou := !decimal.NewFromFloat(d.BasePlanned).Equal(decimal.NewFromFloat(d.ScenarioPlanned)) ||
			d.BaseStatus != d.ScenarioStatus || !b.FirmShortageQty.Equal(c.FirmShortageQty) || !sameDate(d.BaseShortageDate, d.ScenarioShortDate)
		if mudou {
			out.Items = append(out.Items, d)
		}
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].ItemCode < out.Items[j].ItemCode })

	centros := map[int64]*WorkCenterDiff{}
	for id, w := range base.wc {
		centros[id] = &WorkCenterDiff{WorkCenterID: id, BaseHours: w.BaseHours, BaseOverloadDays: w.BaseOverloadDays, BaseMaxLoadPct: w.BaseMaxLoadPct}
	}
	for id, w := range cen.wc {
		d := centros[id]
		if d == nil {
			d = &WorkCenterDiff{WorkCenterID: id}
			centros[id] = d
		}
		d.ScenarioHours, d.ScenarioOverDays, d.ScenarioMaxLoad = w.BaseHours, w.BaseOverloadDays, w.BaseMaxLoadPct
	}
	ids := make([]int64, 0, len(centros))
	for id := range centros {
		ids = append(ids, id)
	}
	nomes, err := uc.Repo.WorkCenterNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		d := centros[id]
		d.Name = nomes[id]
		if d.BaseHours != d.ScenarioHours || d.BaseOverloadDays != d.ScenarioOverDays || d.BaseMaxLoadPct != d.ScenarioMaxLoad {
			out.WorkCenters = append(out.WorkCenters, *d)
		}
	}
	sort.Slice(out.WorkCenters, func(i, j int) bool { return out.WorkCenters[i].WorkCenterID < out.WorkCenters[j].WorkCenterID })
	return out, nil
}

func sameDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}
