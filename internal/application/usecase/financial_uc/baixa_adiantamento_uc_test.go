package financial_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── fakes (embed interfaces; override only what's exercised) ────────────────

type fakeFinAuth struct {
	ports.AuthService
	canPagar   bool
	canReceber bool
	uid        uuid.UUID
	uidErr     error
}

func (f fakeFinAuth) CanBaixarContaPagar(context.Context) bool   { return f.canPagar }
func (f fakeFinAuth) CanBaixarContaReceber(context.Context) bool { return f.canReceber }
func (f fakeFinAuth) UserID(context.Context) (uuid.UUID, error)  { return f.uid, f.uidErr }

type advCall struct {
	advID     int64
	contaTipo string
	contaID   int64
	valor     decimal.Decimal
	data      time.Time
}

type fakeFinRepo struct {
	repository.FinancialRepository
	cp    *entity.ContaPagar
	cpErr error

	baixaParams        *repository.BaixaParams
	baixaFluxo         *entity.FluxoCaixa
	baixaValorOriginal decimal.Decimal

	adv *advCall
}

func (f *fakeFinRepo) GetContaPagar(context.Context, int64) (*entity.ContaPagar, error) {
	return f.cp, f.cpErr
}

func (f *fakeFinRepo) BaixarContaPagarAtomico(_ context.Context, _ int64, params repository.BaixaParams, fc entity.FluxoCaixa, valorOriginal decimal.Decimal, _ int64) error {
	f.baixaParams = &params
	f.baixaFluxo = &fc
	f.baixaValorOriginal = valorOriginal
	return nil
}

func (f *fakeFinRepo) AplicarAdiantamentoAtomico(_ context.Context, advID int64, contaTipo string, contaID int64, valor decimal.Decimal, _ uuid.UUID, data time.Time) (*entity.AdiantamentoAplicacao, error) {
	f.adv = &advCall{advID: advID, contaTipo: contaTipo, contaID: contaID, valor: valor, data: data}
	return &entity.AdiantamentoAplicacao{ID: 1, AdiantamentoID: advID, ContaID: contaID, ValorAplicado: valor}, nil
}

type fakeFiscalRepo struct {
	fiscalrepo.FiscalRepository
	cfg *fiscalentity.FiscalConfig
	err error
}

func (f fakeFiscalRepo) GetFiscalConfig(context.Context) (*fiscalentity.FiscalConfig, error) {
	return f.cfg, f.err
}

func contaPagar(status entity.ContaPagarStatus, bruto float64, venc time.Time) *entity.ContaPagar {
	return &entity.ContaPagar{
		ID:             1,
		Status:         status,
		ValorBruto:     decimal.NewFromFloat(bruto),
		ValorPago:      decimal.Zero,
		DataVencimento: venc,
	}
}

// ─── BaixarContaPagar ────────────────────────────────────────────────────────

func TestBaixarContaPagar_Unauthorized(t *testing.T) {
	uc := BaixarContaPagarUseCase{Repo: &fakeFinRepo{}, FiscalRepo: fakeFiscalRepo{}, Auth: fakeFinAuth{canPagar: false}}
	err := uc.Execute(context.Background(), 1, request.BaixarContaPagarDTO{})
	if !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestBaixarContaPagar_RejectsNonPendingStatus(t *testing.T) {
	repo := &fakeFinRepo{cp: contaPagar(entity.ContaPagarStatusPago, 100, time.Now())}
	uc := BaixarContaPagarUseCase{Repo: repo, FiscalRepo: fakeFiscalRepo{}, Auth: fakeFinAuth{canPagar: true}}

	err := uc.Execute(context.Background(), 1, request.BaixarContaPagarDTO{DataPagamento: "2026-01-10", ValorPago: 100})
	if err == nil {
		t.Fatal("expected error for already-paid title")
	}
	if repo.baixaParams != nil {
		t.Fatal("must not write down a title in an invalid status")
	}
}

func TestBaixarContaPagar_InvalidDate(t *testing.T) {
	repo := &fakeFinRepo{cp: contaPagar(entity.ContaPagarStatusPendente, 100, time.Now())}
	uc := BaixarContaPagarUseCase{Repo: repo, FiscalRepo: fakeFiscalRepo{}, Auth: fakeFinAuth{canPagar: true}}

	err := uc.Execute(context.Background(), 1, request.BaixarContaPagarDTO{DataPagamento: "10/01/2026", ValorPago: 100})
	if err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestBaixarContaPagar_OnTimeHasNoInterestOrFine(t *testing.T) {
	venc := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeFinRepo{cp: contaPagar(entity.ContaPagarStatusAprovado, 1000, venc)}
	uc := BaixarContaPagarUseCase{Repo: repo, FiscalRepo: fakeFiscalRepo{}, Auth: fakeFinAuth{canPagar: true, uid: uuid.New()}}

	// Paid exactly on the due date → no juros/multa.
	err := uc.Execute(context.Background(), 1, request.BaixarContaPagarDTO{
		DataPagamento: "2026-02-01", ValorPago: 1000, ContaBancariaID: 9,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.baixaParams.Juros != 0 || repo.baixaParams.Multa != 0 {
		t.Fatalf("on-time payment should have no juros/multa, got juros=%v multa=%v",
			repo.baixaParams.Juros, repo.baixaParams.Multa)
	}
	// Cash-flow entry is an outflow equal to the amount paid.
	if repo.baixaFluxo.Tipo != entity.FluxoCaixaTipoSaida {
		t.Fatalf("fluxo tipo = %v, want SAIDA", repo.baixaFluxo.Tipo)
	}
	if !repo.baixaFluxo.Valor.Equal(decimal.NewFromFloat(1000)) {
		t.Fatalf("fluxo valor = %v, want 1000", repo.baixaFluxo.Valor)
	}
}

func TestBaixarContaPagar_LateAccruesInterestAndFineWithDefaults(t *testing.T) {
	// Defaults: juros 1%/mês, multa 2%. Empty fiscal config → defaults apply.
	venc := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeFinRepo{cp: contaPagar(entity.ContaPagarStatusPendente, 1000, venc)}
	uc := BaixarContaPagarUseCase{Repo: repo, FiscalRepo: fakeFiscalRepo{cfg: &fiscalentity.FiscalConfig{}}, Auth: fakeFinAuth{canPagar: true, uid: uuid.New()}}

	// 30 days late → monthsLate = 1.0 → juros = 1000*0.01*1 = 10; multa = 1000*0.02 = 20.
	err := uc.Execute(context.Background(), 1, request.BaixarContaPagarDTO{
		DataPagamento: "2026-01-31", ValorPago: 1000, ContaBancariaID: 9,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := repo.baixaParams.Juros; !almostEqual(got, 10) {
		t.Fatalf("juros = %v, want 10", got)
	}
	if got := repo.baixaParams.Multa; !almostEqual(got, 20) {
		t.Fatalf("multa = %v, want 20", got)
	}
	// Total cash outflow = principal + juros + multa = 1030.
	if !repo.baixaFluxo.Valor.Equal(decimal.NewFromFloat(1030)) {
		t.Fatalf("fluxo valor = %v, want 1030", repo.baixaFluxo.Valor)
	}
	if !repo.baixaValorOriginal.Equal(decimal.NewFromFloat(1000)) {
		t.Fatalf("valorOriginal = %v, want 1000", repo.baixaValorOriginal)
	}
}

func TestBaixarContaPagar_LateUsesConfiguredRates(t *testing.T) {
	venc := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeFinRepo{cp: contaPagar(entity.ContaPagarStatusPendente, 1000, venc)}
	cfg := &fiscalentity.FiscalConfig{JurosMes: 0.02, MultaAtraso: 0.05}
	uc := BaixarContaPagarUseCase{Repo: repo, FiscalRepo: fakeFiscalRepo{cfg: cfg}, Auth: fakeFinAuth{canPagar: true, uid: uuid.New()}}

	// 30 days late → juros = 1000*0.02*1 = 20; multa = 1000*0.05 = 50.
	if err := uc.Execute(context.Background(), 1, request.BaixarContaPagarDTO{
		DataPagamento: "2026-01-31", ValorPago: 1000, ContaBancariaID: 9,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !almostEqual(repo.baixaParams.Juros, 20) || !almostEqual(repo.baixaParams.Multa, 50) {
		t.Fatalf("configured rates not applied: juros=%v multa=%v", repo.baixaParams.Juros, repo.baixaParams.Multa)
	}
}

// ─── AplicarAdiantamento ─────────────────────────────────────────────────────

func TestAplicarAdiantamento_InvalidContaTipo(t *testing.T) {
	uc := AplicarAdiantamentoUseCase{Repo: &fakeFinRepo{}, Auth: fakeFinAuth{}}
	_, err := uc.Execute(context.Background(), 1, request.AplicarAdiantamentoDTO{ContaTipo: "XPTO", Valor: 10})
	if err == nil {
		t.Fatal("expected error for invalid conta_tipo")
	}
}

func TestAplicarAdiantamento_UnauthorizedForReceber(t *testing.T) {
	uc := AplicarAdiantamentoUseCase{Repo: &fakeFinRepo{}, Auth: fakeFinAuth{canReceber: false}}
	_, err := uc.Execute(context.Background(), 1, request.AplicarAdiantamentoDTO{
		ContaTipo: string(entity.AdiantamentoTipoReceber), Valor: 10,
	})
	if !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestAplicarAdiantamento_RejectsNonPositiveValue(t *testing.T) {
	uc := AplicarAdiantamentoUseCase{Repo: &fakeFinRepo{}, Auth: fakeFinAuth{canPagar: true}}
	_, err := uc.Execute(context.Background(), 1, request.AplicarAdiantamentoDTO{
		ContaTipo: string(entity.AdiantamentoTipoPagar), Valor: 0,
	})
	if err == nil {
		t.Fatal("expected error for non-positive value")
	}
}

func TestAplicarAdiantamento_AppliesWithParsedDate(t *testing.T) {
	repo := &fakeFinRepo{}
	uc := AplicarAdiantamentoUseCase{Repo: repo, Auth: fakeFinAuth{canPagar: true, uid: uuid.New()}}
	d := "2026-03-15"
	out, err := uc.Execute(context.Background(), 77, request.AplicarAdiantamentoDTO{
		ContaTipo: string(entity.AdiantamentoTipoPagar), ContaID: 555, Valor: 250.50, DataAplicacao: &d,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || repo.adv == nil {
		t.Fatal("application should have been delegated to the repository")
	}
	if repo.adv.advID != 77 || repo.adv.contaID != 555 || repo.adv.contaTipo != "PAGAR" {
		t.Fatalf("delegated args wrong: %+v", repo.adv)
	}
	if !repo.adv.valor.Equal(decimal.NewFromFloat(250.50)) {
		t.Fatalf("valor = %v, want 250.50", repo.adv.valor)
	}
	if repo.adv.data.Year() != 2026 || repo.adv.data.Month() != 3 || repo.adv.data.Day() != 15 {
		t.Fatalf("date parsed wrong: %v", repo.adv.data)
	}
}

func almostEqual(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-6
}
