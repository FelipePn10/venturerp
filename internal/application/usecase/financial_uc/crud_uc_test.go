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
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// crudAuth grants/denies every financial capability with a single switch.
type crudAuth struct {
	ports.AuthService
	ok  bool
	uid uuid.UUID
}

func (f crudAuth) CanBaixarContaReceber(context.Context) bool { return f.ok }
func (f crudAuth) CanApproveContaPagar(context.Context) bool  { return f.ok }
func (f crudAuth) CanCancelContaPagar(context.Context) bool   { return f.ok }
func (f crudAuth) CanCancelContaReceber(context.Context) bool { return f.ok }
func (f crudAuth) CanCreateContaPagar(context.Context) bool   { return f.ok }
func (f crudAuth) CanCreateContaReceber(context.Context) bool { return f.ok }
func (f crudAuth) UserID(context.Context) (uuid.UUID, error)  { return f.uid, nil }

type crudRepo struct {
	repository.FinancialRepository
	cr *entity.ContaReceber

	approvedBy uuid.UUID
	approved   bool
	cancelledP bool
	cancelledR bool
	createdCP  *entity.ContaPagar
	createdCR  *entity.ContaReceber
	brParams   *repository.BaixaParams
	brFluxo    *entity.FluxoCaixa
}

func (f *crudRepo) GetContaReceber(context.Context, int64) (*entity.ContaReceber, error) {
	return f.cr, nil
}
func (f *crudRepo) BaixarContaReceberAtomico(_ context.Context, _ int64, p repository.BaixaParams, fc entity.FluxoCaixa, _ decimal.Decimal, _ int64) error {
	f.brParams = &p
	f.brFluxo = &fc
	return nil
}
func (f *crudRepo) ApproveContaPagar(_ context.Context, _ int64, by uuid.UUID) error {
	f.approved = true
	f.approvedBy = by
	return nil
}
func (f *crudRepo) CancelContaPagar(context.Context, int64) error   { f.cancelledP = true; return nil }
func (f *crudRepo) CancelContaReceber(context.Context, int64) error { f.cancelledR = true; return nil }
func (f *crudRepo) CreateContaPagar(_ context.Context, c *entity.ContaPagar) (*entity.ContaPagar, error) {
	f.createdCP = c
	return c, nil
}
func (f *crudRepo) CreateContaReceber(_ context.Context, c *entity.ContaReceber) (*entity.ContaReceber, error) {
	f.createdCR = c
	return c, nil
}

// ─── BaixarContaReceber (symmetric to pagar; receivable inflow) ──────────────

func TestBaixarContaReceber_Unauthorized(t *testing.T) {
	uc := BaixarContaReceberUseCase{Repo: &crudRepo{}, FiscalRepo: fakeFiscalRepo{}, Auth: crudAuth{ok: false}}
	if err := uc.Execute(context.Background(), 1, request.BaixarContaReceberDTO{}); !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestBaixarContaReceber_LateAccruesInterestAndFine(t *testing.T) {
	venc := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &crudRepo{cr: &entity.ContaReceber{
		Status: entity.ContaReceberStatusPendente, ValorBruto: decimal.NewFromFloat(2000), ValorRecebido: decimal.Zero, DataVencimento: venc,
	}}
	uc := BaixarContaReceberUseCase{Repo: repo, FiscalRepo: fakeFiscalRepo{cfg: &fiscalentity.FiscalConfig{}}, Auth: crudAuth{ok: true, uid: uuid.New()}}

	// 30 days late, defaults → juros = 2000*0.01 = 20; multa = 2000*0.02 = 40.
	if err := uc.Execute(context.Background(), 1, request.BaixarContaReceberDTO{
		DataRecebimento: "2026-01-31", ValorRecebido: 2000, ContaBancariaID: 5,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !almostEqual(repo.brParams.Juros, 20) || !almostEqual(repo.brParams.Multa, 40) {
		t.Fatalf("juros/multa wrong: %v / %v", repo.brParams.Juros, repo.brParams.Multa)
	}
	// Receivable settlement is a cash INFLOW.
	if repo.brFluxo.Tipo != entity.FluxoCaixaTipoEntrada {
		t.Fatalf("fluxo tipo = %v, want ENTRADA", repo.brFluxo.Tipo)
	}
}

// ─── Approve / Cancel ────────────────────────────────────────────────────────

func TestApproveContaPagar(t *testing.T) {
	repo := &crudRepo{}
	uid := uuid.New()
	uc := ApproveContaPagarUseCase{Repo: repo, Auth: crudAuth{ok: true, uid: uid}}
	if err := uc.Execute(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.approved || repo.approvedBy != uid {
		t.Fatalf("approval not delegated with the actor")
	}

	deny := ApproveContaPagarUseCase{Repo: &crudRepo{}, Auth: crudAuth{ok: false}}
	if err := deny.Execute(context.Background(), 1); !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("unauthorized approve should fail")
	}
}

func TestCancelContaPagarAndReceber(t *testing.T) {
	rp := &crudRepo{}
	if err := (&CancelContaPagarUseCase{Repo: rp, Auth: crudAuth{ok: true}}).Execute(context.Background(), 1); err != nil || !rp.cancelledP {
		t.Fatalf("cancel pagar failed: err=%v cancelled=%v", err, rp.cancelledP)
	}
	rr := &crudRepo{}
	if err := (&CancelContaReceberUseCase{Repo: rr, Auth: crudAuth{ok: true}}).Execute(context.Background(), 1); err != nil || !rr.cancelledR {
		t.Fatalf("cancel receber failed: err=%v cancelled=%v", err, rr.cancelledR)
	}
	// Unauthorized paths.
	if err := (&CancelContaPagarUseCase{Repo: &crudRepo{}, Auth: crudAuth{ok: false}}).Execute(context.Background(), 1); !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatal("unauthorized cancel pagar should fail")
	}
	if err := (&CancelContaReceberUseCase{Repo: &crudRepo{}, Auth: crudAuth{ok: false}}).Execute(context.Background(), 1); !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatal("unauthorized cancel receber should fail")
	}
}

// ─── Create (mapping + defaults) ─────────────────────────────────────────────

func TestCreateContaPagar_MapsDefaults(t *testing.T) {
	repo := &crudRepo{}
	uid := uuid.New()
	uc := CreateContaPagarUseCase{Repo: repo, Auth: crudAuth{ok: true, uid: uid}}

	out, err := uc.Execute(context.Background(), request.CreateContaPagarDTO{
		NumeroDocumento: "NF-1", DataEmissao: "2026-01-01", DataVencimento: "2026-02-01",
		ValorBruto: 500, ParcelaNumero: 1, ParcelaTotal: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || repo.createdCP == nil {
		t.Fatal("conta a pagar should have been created")
	}
	c := repo.createdCP
	if c.Status != entity.ContaPagarStatusPendente || c.StatusAprovacao != entity.AprovacaoPendente {
		t.Fatalf("new payable should start PENDENTE/aprovacao PENDENTE, got %s/%s", c.Status, c.StatusAprovacao)
	}
	if !c.ValorBruto.Equal(decimal.NewFromFloat(500)) || !c.ValorPago.Equal(decimal.Zero) {
		t.Fatalf("amounts wrong: bruto=%v pago=%v", c.ValorBruto, c.ValorPago)
	}
	if c.CriadoPor != uid || !c.IsActive {
		t.Fatalf("actor/active flags wrong: %+v", c)
	}
	if c.DataVencimento.Month() != time.February {
		t.Fatalf("due date parsed wrong: %v", c.DataVencimento)
	}
}

func TestCreateContaReceber_MapsDefaults(t *testing.T) {
	repo := &crudRepo{}
	uc := CreateContaReceberUseCase{Repo: repo, Auth: crudAuth{ok: true, uid: uuid.New()}}

	if _, err := uc.Execute(context.Background(), request.CreateContaReceberDTO{
		DataEmissao: "2026-01-01", DataVencimento: "2026-02-01", ValorBruto: 750, ParcelaNumero: 1, ParcelaTotal: 1,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := repo.createdCR
	if c == nil || c.Status != entity.ContaReceberStatusPendente {
		t.Fatalf("new receivable should start PENDENTE")
	}
	if !c.ValorRecebido.Equal(decimal.Zero) || !c.ValorBruto.Equal(decimal.NewFromFloat(750)) {
		t.Fatalf("amounts wrong: bruto=%v recebido=%v", c.ValorBruto, c.ValorRecebido)
	}
}

func TestCreateContaPagar_Unauthorized(t *testing.T) {
	uc := CreateContaPagarUseCase{Repo: &crudRepo{}, Auth: crudAuth{ok: false}}
	if _, err := uc.Execute(context.Background(), request.CreateContaPagarDTO{}); !errors.Is(err, errorsuc.ErrUnauthorized) {
		t.Fatalf("unauthorized create should fail")
	}
}
