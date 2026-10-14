package mrp_report_uc

import (
	"context"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// Situação de cobertura de um item — o farol do monitor de cobertura.
const (
	// CoverageCritical: estoque e ordens firmes não cobrem uma necessidade que
	// já venceu, ou a ordem planejada que a cobriria já devia ter começado.
	CoverageCritical = "CRITICO"
	// CoverageWarning: falta coberta só por ordem planejada (ainda não firmada)
	// ou estoque projetado abaixo do estoque de segurança.
	CoverageWarning = "ATENCAO"
	// CoverageOK: estoque e ordens firmes cobrem todo o horizonte do plano.
	CoverageOK = "OK"
)

// CoverageBucket é o que o MRP calculou para o item numa data de necessidade.
type CoverageBucket struct {
	Date    time.Time
	Demand  decimal.Decimal
	Firm    decimal.Decimal
	Planned decimal.Decimal
}

// CoverageInput é o retrato de um item lido do último cálculo do plano.
type CoverageInput struct {
	ItemCode        int64
	BusinessCode    string
	Description     string
	UOM             string
	EngineeringType int
	PlannerCode     *int64
	PlannerName     string
	SafetyStock     decimal.Decimal
	Stock           decimal.Decimal
	Buckets         []CoverageBucket
	// FirstPlannedStart é o início mais cedo das ordens planejadas pendentes do
	// item (sugestões ainda não firmadas).
	FirstPlannedStart *time.Time
	PlannedOrders     int
}

// CoverageRow é uma linha do monitor de cobertura.
type CoverageRow struct {
	ItemCode      int64           `json:"item_code"`
	BusinessCode  string          `json:"business_code,omitempty"`
	Description   string          `json:"description"`
	UOM           string          `json:"uom,omitempty"`
	ItemType      string          `json:"item_type"`
	PlannerCode   *int64          `json:"planner_code,omitempty"`
	PlannerName   string          `json:"planner_name,omitempty"`
	Status        string          `json:"status"`
	Reason        string          `json:"reason"`
	Stock         decimal.Decimal `json:"stock"`
	SafetyStock   decimal.Decimal `json:"safety_stock"`
	Demand        decimal.Decimal `json:"demand"`
	FirmSupply    decimal.Decimal `json:"firm_supply"`
	PlannedSupply decimal.Decimal `json:"planned_supply"`
	// FirstNeedDate é a primeira data com necessidade no horizonte.
	FirstNeedDate *time.Time `json:"first_need_date,omitempty"`
	// StockShortageDate é quando só o estoque deixa de cobrir (sem nada a
	// receber); DaysOfSupply é a distância de hoje até ela. Vazio = o estoque
	// cobre todo o horizonte.
	StockShortageDate *time.Time `json:"stock_shortage_date,omitempty"`
	DaysOfSupply      *int       `json:"days_of_supply,omitempty"`
	// FirmShortageDate é quando estoque + ordens firmes deixam de cobrir;
	// ReceiptDaysOfSupply é a distância de hoje até ela (o "receipt days'
	// supply" do SAP).
	FirmShortageDate    *time.Time      `json:"firm_shortage_date,omitempty"`
	ReceiptDaysOfSupply *int            `json:"receipt_days_of_supply,omitempty"`
	FirmShortageQty     decimal.Decimal `json:"firm_shortage_qty"`
	// MinProjected é o menor saldo projetado com tudo (estoque, firmes e
	// planejadas) no horizonte.
	MinProjected      decimal.Decimal `json:"min_projected"`
	PlannedOrders     int             `json:"planned_orders"`
	FirstPlannedStart *time.Time      `json:"first_planned_start,omitempty"`
}

// Coverage é o monitor de cobertura de materiais do plano: para cada item, até
// quando o estoque cobre, até quando estoque + ordens firmes cobrem e o que
// depende de ordem planejada ainda não firmada. É o que o SAP chama de Monitor
// Material Coverage (dias de cobertura e farol) e o Oracle de horizon plan.
// Ordenado do mais urgente para o mais tranquilo.
func (uc *UseCase) Coverage(ctx context.Context, filter Filter, today time.Time) ([]CoverageRow, error) {
	if err := uc.authorize(ctx); err != nil {
		return nil, err
	}
	if filter.PlanCode == nil || *filter.PlanCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o plano para consultar a cobertura")
	}
	if err := validateFilter(&filter, "coverage"); err != nil {
		return nil, err
	}
	inputs, err := uc.Reader.Coverage(ctx, filter)
	if err != nil {
		return nil, err
	}
	hoje := dateOnly(today)
	rows := make([]CoverageRow, 0, len(inputs))
	for _, in := range inputs {
		row := computeCoverage(in, hoje)
		if filter.OnlyWithMessage && row.Status == CoverageOK {
			continue
		}
		rows = append(rows, row)
	}
	peso := map[string]int{CoverageCritical: 0, CoverageWarning: 1, CoverageOK: 2}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if peso[a.Status] != peso[b.Status] {
			return peso[a.Status] < peso[b.Status]
		}
		da, db := coverageSortDate(a), coverageSortDate(b)
		if !da.Equal(db) {
			return da.Before(db)
		}
		return a.ItemCode < b.ItemCode
	})
	return rows, nil
}

func coverageSortDate(r CoverageRow) time.Time {
	if r.FirmShortageDate != nil {
		return *r.FirmShortageDate
	}
	if r.StockShortageDate != nil {
		return *r.StockShortageDate
	}
	return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func daysBetween(from, to time.Time) int {
	return int(dateOnly(to).Sub(dateOnly(from)).Hours() / 24)
}

func computeCoverage(in CoverageInput, hoje time.Time) CoverageRow {
	row := CoverageRow{
		ItemCode: in.ItemCode, BusinessCode: in.BusinessCode, Description: in.Description, UOM: in.UOM,
		ItemType: itemTypeLabel(in.EngineeringType), PlannerCode: in.PlannerCode, PlannerName: in.PlannerName,
		Stock: in.Stock, SafetyStock: in.SafetyStock, PlannedOrders: in.PlannedOrders, FirstPlannedStart: in.FirstPlannedStart,
	}
	buckets := append([]CoverageBucket(nil), in.Buckets...)
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].Date.Before(buckets[j].Date) })

	soEstoque := in.Stock
	comFirmes := in.Stock
	comTudo := in.Stock
	row.MinProjected = in.Stock
	abaixoSeguranca := in.Stock.LessThan(in.SafetyStock)
	for _, b := range buckets {
		row.Demand = row.Demand.Add(b.Demand)
		row.FirmSupply = row.FirmSupply.Add(b.Firm)
		row.PlannedSupply = row.PlannedSupply.Add(b.Planned)
		if b.Demand.IsPositive() && row.FirstNeedDate == nil {
			d := b.Date
			row.FirstNeedDate = &d
		}
		soEstoque = soEstoque.Sub(b.Demand)
		comFirmes = comFirmes.Add(b.Firm).Sub(b.Demand)
		comTudo = comTudo.Add(b.Firm).Add(b.Planned).Sub(b.Demand)
		if soEstoque.IsNegative() && row.StockShortageDate == nil {
			d := b.Date
			row.StockShortageDate = &d
		}
		if comFirmes.IsNegative() {
			if row.FirmShortageDate == nil {
				d := b.Date
				row.FirmShortageDate = &d
			}
			if falta := comFirmes.Neg(); falta.GreaterThan(row.FirmShortageQty) {
				row.FirmShortageQty = falta
			}
		}
		if comTudo.LessThan(row.MinProjected) {
			row.MinProjected = comTudo
		}
		if in.SafetyStock.IsPositive() && comTudo.LessThan(in.SafetyStock) {
			abaixoSeguranca = true
		}
	}
	if row.StockShortageDate != nil {
		n := daysBetween(hoje, *row.StockShortageDate)
		row.DaysOfSupply = &n
	}
	if row.FirmShortageDate != nil {
		n := daysBetween(hoje, *row.FirmShortageDate)
		row.ReceiptDaysOfSupply = &n
	}

	planejadaAtrasada := in.FirstPlannedStart != nil && dateOnly(*in.FirstPlannedStart).Before(hoje)
	switch {
	case row.FirmShortageDate != nil && !dateOnly(*row.FirmShortageDate).After(hoje):
		row.Status, row.Reason = CoverageCritical, "falta vencida: estoque e ordens firmes não cobrem uma necessidade de hoje ou anterior"
	case row.FirmShortageDate != nil && planejadaAtrasada:
		row.Status, row.Reason = CoverageCritical, "a ordem planejada que cobre a falta já devia ter começado"
	case row.MinProjected.IsNegative():
		row.Status, row.Reason = CoverageCritical, "nem as ordens planejadas cobrem toda a necessidade"
	case row.FirmShortageDate != nil:
		row.Status, row.Reason = CoverageWarning, "a falta só é coberta por ordem planejada ainda não firmada"
	case abaixoSeguranca:
		row.Status, row.Reason = CoverageWarning, "o estoque projetado fica abaixo do estoque de segurança"
	default:
		row.Status, row.Reason = CoverageOK, "estoque e ordens firmes cobrem todo o horizonte"
	}
	return row
}

func itemTypeLabel(engineeringType int) string {
	switch engineeringType {
	case 0:
		return "FABRICADO"
	case 1:
		return "COMPRADO"
	case 2:
		return "TERCEIRIZADO"
	}
	return "OUTRO"
}
