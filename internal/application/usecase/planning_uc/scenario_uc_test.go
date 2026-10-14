package planning_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

type cenarioRepoFalso struct {
	ScenarioRepository
	s       Scenario
	changes []ScenarioChange
	synced  int
	marked  string
}

func (r *cenarioRepoFalso) Get(context.Context, int64) (*Scenario, error) { c := r.s; return &c, nil }
func (r *cenarioRepoFalso) ItemExists(_ context.Context, item int64) (bool, error) {
	return item == 9, nil
}
func (r *cenarioRepoFalso) MachineExists(_ context.Context, m int64) (bool, error) {
	return m == 3, nil
}
func (r *cenarioRepoFalso) DemandRefExists(_ context.Context, _ string, code int64) (bool, error) {
	return code == 70, nil
}
func (r *cenarioRepoFalso) AddChange(_ context.Context, _ int64, c ScenarioChange, _ uuid.UUID) (*ScenarioChange, error) {
	r.changes = append(r.changes, c)
	return &c, nil
}
func (r *cenarioRepoFalso) SyncPlan(context.Context, int64) error { r.synced++; return nil }
func (r *cenarioRepoFalso) MarkCalculated(_ context.Context, _ int64, notes string) error {
	r.marked = notes
	return nil
}

type mrpFalso struct{ plano int64 }

func (m *mrpFalso) Calculate(_ context.Context, plan, _ int64, llc bool) (*mrpentity.MRPCalculationLog, error) {
	m.plano = plan
	if llc {
		return nil, errors.New("cenário não pode gravar LLC")
	}
	return &mrpentity.MRPCalculationLog{Status: "COMPLETED", TotalOrders: 4}, nil
}

type crpFalso struct{ plano int64 }

func (c *crpFalso) CalculateCRP(_ context.Context, dto request.CalculateCRPDTO) (*response.CRPSummaryResponse, error) {
	c.plano = dto.PlanCode
	return &response.CRPSummaryResponse{OverloadCount: 2}, nil
}
func (c *crpFalso) ListByPlan(context.Context, int64) ([]*response.CRPEntryResponse, error) {
	return nil, nil
}

type lockFalso struct{ ocupado bool }

func (l *lockFalso) TryLock(context.Context, int64) (bool, error) { return !l.ocupado, nil }
func (l *lockFalso) Unlock(context.Context, int64) error          { return nil }

func dec(v int64) *decimal.Decimal { d := decimal.NewFromInt(v); return &d }
func str(v string) *string         { return &v }
func i64p(v int64) *int64          { return &v }

// Cada tipo de ajuste exige o que faz sentido e só aceita referência da
// empresa; cenário já promovido não recebe ajuste.
func TestAjustesDoCenarioValidados(t *testing.T) {
	repo := &cenarioRepoFalso{s: Scenario{ID: 1, BasePlanCode: 1, ScenarioPlanCode: 50}}
	uc := &ScenarioUseCase{Repo: repo, Auth: &authFalsa{empresa: 7, ator: uuid.New(), pode: true}}
	ctx := context.Background()
	invalidos := []request.ScenarioChangeDTO{
		{Kind: "DEMANDA", DemandRefKind: "NOVA", ItemCode: i64p(9), Quantity: dec(5)},                              // sem data
		{Kind: "DEMANDA", DemandRefKind: "NOVA", ItemCode: i64p(8), Quantity: dec(5), NeedDate: str("2026-11-01")}, // item de outra empresa
		{Kind: "DEMANDA", DemandRefKind: "PEDIDO_VENDA", DemandRefCode: i64p(70)},                                  // sem quantidade nem data
		{Kind: "DEMANDA", DemandRefKind: "PEDIDO_VENDA", DemandRefCode: i64p(71), Quantity: dec(1)},                // pedido de outra empresa
		{Kind: "ESTOQUE", ItemCode: i64p(9), Quantity: dec(0)},
		{Kind: "LEAD_TIME", ItemCode: i64p(9)},
		{Kind: "CAPACIDADE", MachineID: i64p(3)},
		{Kind: "CAPACIDADE", MachineID: i64p(3), HoursPerDay: dec(25)},
		{Kind: "CAPACIDADE", MachineID: i64p(3), StopFrom: str("2026-10-20"), StopTo: str("2026-10-19")},
		{Kind: "CAPACIDADE", MachineID: i64p(4), HoursPerDay: dec(4)},
		{Kind: "OUTRO"},
	}
	for i, dto := range invalidos {
		if _, err := uc.AddChange(ctx, 1, dto); err == nil {
			t.Fatalf("ajuste inválido %d aceito: %+v", i, dto)
		}
	}
	if len(repo.changes) != 0 {
		t.Fatalf("gravou ajuste inválido: %+v", repo.changes)
	}
	validos := []request.ScenarioChangeDTO{
		{Kind: "demanda", DemandRefKind: "nova", ItemCode: i64p(9), Quantity: dec(5), NeedDate: str("2026-11-01")},
		{Kind: "DEMANDA", DemandRefKind: "PEDIDO_VENDA", DemandRefCode: i64p(70), Quantity: dec(0)},
		{Kind: "ESTOQUE", ItemCode: i64p(9), Quantity: dec(-30)},
		{Kind: "CAPACIDADE", MachineID: i64p(3), StopFrom: str("2026-10-20")},
	}
	for _, dto := range validos {
		if _, err := uc.AddChange(ctx, 1, dto); err != nil {
			t.Fatalf("ajuste válido recusado: %+v (%v)", dto, err)
		}
	}
	if repo.changes[0].Kind != ChangeDemand || repo.changes[0].NeedDate == nil {
		t.Fatalf("ajuste gravado: %+v", repo.changes[0])
	}

	agora := time.Now()
	repo.s.PromotedAt = &agora
	if _, err := uc.AddChange(ctx, 1, validos[2]); err == nil {
		t.Fatal("ajustou cenário já promovido")
	}
}

// O cálculo usa o plano oculto, não grava LLC, recopia o plano base antes e
// respeita o cadeado do planejamento da empresa.
func TestCalculoDoCenario(t *testing.T) {
	repo := &cenarioRepoFalso{s: Scenario{ID: 1, BasePlanCode: 1, ScenarioPlanCode: 50}}
	mrp, crp, lock := &mrpFalso{}, &crpFalso{}, &lockFalso{}
	uc := &ScenarioUseCase{Repo: repo, MRP: mrp, CRP: crp, Lock: lock, Auth: &authFalsa{empresa: 7, ator: uuid.New(), pode: true}}
	out, err := uc.Calculate(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if mrp.plano != 50 || crp.plano != 50 || repo.synced != 1 || out.CRPOverload != 2 || repo.marked == "" {
		t.Fatalf("cálculo: mrp %d crp %d sync %d out %+v nota %q", mrp.plano, crp.plano, repo.synced, out, repo.marked)
	}
	lock.ocupado = true
	var conflito *errorsuc.ConflictError
	if _, err := uc.Calculate(context.Background(), 1); !errors.As(err, &conflito) {
		t.Fatalf("calculou com outro cálculo em andamento: %v", err)
	}
}

var _ = mrp_report_uc.CoverageOK
