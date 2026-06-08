package accounting_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/ecd"
	accountingEntity "github.com/FelipePn10/panossoerp/internal/domain/accounting/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/accounting/repository"
)

// ─── Plan ─────────────────────────────────────────────────────────────────────

type AccountingPlanUseCase struct {
	Repo domainrepo.AccountingRepository
}

func (uc *AccountingPlanUseCase) Create(ctx context.Context, p *accountingEntity.AccountingPlan) (*accountingEntity.AccountingPlan, error) {
	if p.Status == "" {
		p.Status = accountingEntity.PlanStatusIncluido
	}
	return uc.Repo.CreateAccountingPlan(ctx, p)
}

func (uc *AccountingPlanUseCase) GetActive(ctx context.Context) ([]*accountingEntity.AccountingPlan, error) {
	all, err := uc.Repo.ListAccountingPlans(ctx)
	if err != nil {
		return nil, err
	}
	var active []*accountingEntity.AccountingPlan
	for _, p := range all {
		if p.Status == accountingEntity.PlanStatusAtivo {
			active = append(active, p)
		}
	}
	return active, nil
}

func (uc *AccountingPlanUseCase) List(ctx context.Context) ([]*accountingEntity.AccountingPlan, error) {
	return uc.Repo.ListAccountingPlans(ctx)
}

// ─── Account ──────────────────────────────────────────────────────────────────

type AccountingAccountUseCase struct {
	Repo domainrepo.AccountingRepository
}

func (uc *AccountingAccountUseCase) Create(ctx context.Context, a *accountingEntity.AccountingAccount) (*accountingEntity.AccountingAccount, error) {
	return uc.Repo.CreateAccountingAccount(ctx, a)
}

func (uc *AccountingAccountUseCase) Update(ctx context.Context, a *accountingEntity.AccountingAccount) (*accountingEntity.AccountingAccount, error) {
	return uc.Repo.UpdateAccountingAccount(ctx, a)
}

func (uc *AccountingAccountUseCase) ListByPlan(ctx context.Context, planID int64) ([]*accountingEntity.AccountingAccount, error) {
	return uc.Repo.ListAccountingAccountsByPlan(ctx, planID)
}

// ─── Journal Entry ────────────────────────────────────────────────────────────

type JournalEntryUseCase struct {
	Repo domainrepo.AccountingRepository
}

func (uc *JournalEntryUseCase) Create(ctx context.Context, e *accountingEntity.AccountingJournalEntry) (*accountingEntity.AccountingJournalEntry, error) {
	if e.Value <= 0 {
		return nil, fmt.Errorf("entry value must be positive")
	}
	return uc.Repo.CreateJournalEntry(ctx, e)
}

func (uc *JournalEntryUseCase) ListByPeriod(ctx context.Context, planID int64, empresaID int, from, to time.Time) ([]*accountingEntity.AccountingJournalEntry, error) {
	return uc.Repo.ListJournalEntries(ctx, planID, empresaID, from, to)
}

// ─── Demonstrative ───────────────────────────────────────────────────────────

type DemonstrativeUseCase struct {
	Repo domainrepo.AccountingRepository
}

func (uc *DemonstrativeUseCase) Create(ctx context.Context, d *accountingEntity.AccountingDemonstrative) (*accountingEntity.AccountingDemonstrative, error) {
	return uc.Repo.CreateDemonstrative(ctx, d)
}

func (uc *DemonstrativeUseCase) List(ctx context.Context) ([]*accountingEntity.AccountingDemonstrative, error) {
	return uc.Repo.ListDemonstratives(ctx)
}

func (uc *DemonstrativeUseCase) AddItem(ctx context.Context, item *accountingEntity.AccountingDemonstrativeItem) (*accountingEntity.AccountingDemonstrativeItem, error) {
	return uc.Repo.CreateDemonstrativeItem(ctx, item)
}

// ─── ECD Use Case ─────────────────────────────────────────────────────────────

type ECDUseCase struct {
	Repo domainrepo.AccountingRepository
}

type ECDRequest struct {
	PlanID    int64
	EmpresaID int
	From      time.Time
	To        time.Time
	Empresa   ecd.ECDEmpresa
	Livros    []ecd.ECDLivro
}

func (uc *ECDUseCase) GenerateECD(ctx context.Context, req ECDRequest) (string, error) {
	plan, err := uc.Repo.GetAccountingPlan(ctx, req.PlanID)
	if err != nil {
		return "", fmt.Errorf("loading plan: %w", err)
	}

	accounts, err := uc.Repo.ListAccountingAccountsByPlan(ctx, req.PlanID)
	if err != nil {
		return "", fmt.Errorf("loading accounts: %w", err)
	}

	entries, err := uc.Repo.ListJournalEntries(ctx, req.PlanID, req.EmpresaID, req.From, req.To)
	if err != nil {
		return "", fmt.Errorf("loading entries: %w", err)
	}

	contas := make([]ecd.ECDConta, 0, len(accounts))
	for _, a := range accounts {
		tipoCta := "A"
		if !a.IsAnalytic {
			tipoCta = "S"
		}
		rc := ""
		if a.ReducedCode != nil {
			rc = *a.ReducedCode
		}
		contas = append(contas, ecd.ECDConta{
			CodCta:  a.AccountNumber,
			CodECD:  a.AccountNumber,
			TipoCta: tipoCta,
			DescCta: a.Description,
			CtaRef:  rc,
		})
	}

	lancamentos := make([]ecd.ECDLancamento, 0, len(entries))
	for _, e := range entries {
		lancamentos = append(lancamentos, ecd.ECDLancamento{
			NumLcto:  e.EntryNumber,
			DtLcto:   e.EntryDate,
			CodHist:  e.HistoryCode,
			DescHist: e.Description,
			Partidas: []ecd.ECDPartida{
				{CodCta: fmt.Sprint(e.DebitAccountID), VlLcto: e.Value, IndDC: "D"},
				{CodCta: fmt.Sprint(e.CreditAccountID), VlLcto: e.Value, IndDC: "C"},
			},
		})
	}

	params := ecd.ECDParams{
		Empresa: req.Empresa,
		Periodo: ecd.ECDPeriodo{
			DataInicial: req.From,
			DataFinal:   req.To,
		},
		Plano: ecd.ECDPlano{
			Numero:    plan.PlanNumber,
			Descricao: plan.Description,
		},
		Contas:      contas,
		Livros:      req.Livros,
		Lancamentos: lancamentos,
	}

	return ecd.Generate(params), nil
}
