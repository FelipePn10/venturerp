package accounting

import (
	"context"
	"fmt"
	"time"

	accountingEntity "github.com/FelipePn10/panossoerp/internal/domain/accounting/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/accounting/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountingRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

var _ domainrepo.AccountingRepository = (*AccountingRepositorySQLC)(nil)

func New(q *sqlc.Queries, pool *pgxpool.Pool) *AccountingRepositorySQLC {
	return &AccountingRepositorySQLC{q: q, pool: pool}
}

// ─── Accounting Plans ─────────────────────────────────────────────────────────

func (r *AccountingRepositorySQLC) CreateAccountingPlan(ctx context.Context, p *accountingEntity.AccountingPlan) (*accountingEntity.AccountingPlan, error) {
	row, err := r.q.CreateAccountingPlan(ctx, sqlc.CreateAccountingPlanParams{
		PlanNumber:  int32(p.PlanNumber),
		Description: p.Description,
		ValidFrom:   pgutil.ToPgDate(p.ValidFrom),
		ValidTo:     pgutil.ToPgDateFromPtr(p.ValidTo),
		Status:      string(p.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("creating accounting plan: %w", err)
	}
	return planRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) GetAccountingPlan(ctx context.Context, id int64) (*accountingEntity.AccountingPlan, error) {
	row, err := r.q.GetAccountingPlan(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting accounting plan %d: %w", id, err)
	}
	return planRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) ListAccountingPlans(ctx context.Context) ([]*accountingEntity.AccountingPlan, error) {
	rows, err := r.q.ListAccountingPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing accounting plans: %w", err)
	}
	out := make([]*accountingEntity.AccountingPlan, len(rows))
	for i, row := range rows {
		out[i] = planRowToEntity(row)
	}
	return out, nil
}

func (r *AccountingRepositorySQLC) UpdateAccountingPlan(ctx context.Context, p *accountingEntity.AccountingPlan) (*accountingEntity.AccountingPlan, error) {
	row, err := r.q.UpdateAccountingPlan(ctx, sqlc.UpdateAccountingPlanParams{
		ID:          p.ID,
		PlanNumber:  int32(p.PlanNumber),
		Description: p.Description,
		ValidFrom:   pgutil.ToPgDate(p.ValidFrom),
		ValidTo:     pgutil.ToPgDateFromPtr(p.ValidTo),
		Status:      string(p.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("updating accounting plan %d: %w", p.ID, err)
	}
	return planRowToEntity(row), nil
}

func planRowToEntity(row sqlc.AccountingPlanRow) *accountingEntity.AccountingPlan {
	return &accountingEntity.AccountingPlan{
		ID:          row.ID,
		PlanNumber:  int(row.PlanNumber),
		Description: row.Description,
		ValidFrom:   pgutil.FromPgDate(row.ValidFrom),
		ValidTo:     pgutil.FromPgDateToPtr(row.ValidTo),
		Status:      accountingEntity.PlanStatus(row.Status),
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Accounting Accounts ──────────────────────────────────────────────────────

func (r *AccountingRepositorySQLC) CreateAccountingAccount(ctx context.Context, a *accountingEntity.AccountingAccount) (*accountingEntity.AccountingAccount, error) {
	row, err := r.q.CreateAccountingAccount(ctx, sqlc.CreateAccountingAccountParams{
		PlanID:             a.PlanID,
		ParentID:           a.ParentID,
		AccountNumber:      a.AccountNumber,
		Description:        a.Description,
		NatureCode:         a.NatureCode,
		ReducedCode:        pgutil.ToPgTextFromPtr(a.ReducedCode),
		RequiresCostCenter: a.RequiresCostCenter,
		ValidFrom:          pgutil.ToPgDate(a.ValidFrom),
		ValidTo:            pgutil.ToPgDateFromPtr(a.ValidTo),
		IsAnalytic:         a.IsAnalytic,
	})
	if err != nil {
		return nil, fmt.Errorf("creating accounting account: %w", err)
	}
	return accountRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) GetAccountingAccount(ctx context.Context, id int64) (*accountingEntity.AccountingAccount, error) {
	row, err := r.q.GetAccountingAccount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting accounting account %d: %w", id, err)
	}
	return accountRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) ListAccountingAccountsByPlan(ctx context.Context, planID int64) ([]*accountingEntity.AccountingAccount, error) {
	rows, err := r.q.ListAccountingAccountsByPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("listing accounting accounts for plan %d: %w", planID, err)
	}
	out := make([]*accountingEntity.AccountingAccount, len(rows))
	for i, row := range rows {
		out[i] = accountRowToEntity(row)
	}
	return out, nil
}

func (r *AccountingRepositorySQLC) UpdateAccountingAccount(ctx context.Context, a *accountingEntity.AccountingAccount) (*accountingEntity.AccountingAccount, error) {
	row, err := r.q.UpdateAccountingAccount(ctx, sqlc.UpdateAccountingAccountParams{
		ID:                 a.ID,
		PlanID:             a.PlanID,
		ParentID:           a.ParentID,
		AccountNumber:      a.AccountNumber,
		Description:        a.Description,
		NatureCode:         a.NatureCode,
		ReducedCode:        pgutil.ToPgTextFromPtr(a.ReducedCode),
		RequiresCostCenter: a.RequiresCostCenter,
		ValidFrom:          pgutil.ToPgDate(a.ValidFrom),
		ValidTo:            pgutil.ToPgDateFromPtr(a.ValidTo),
		IsAnalytic:         a.IsAnalytic,
	})
	if err != nil {
		return nil, fmt.Errorf("updating accounting account %d: %w", a.ID, err)
	}
	return accountRowToEntity(row), nil
}

func accountRowToEntity(row sqlc.AccountingAccountRow) *accountingEntity.AccountingAccount {
	return &accountingEntity.AccountingAccount{
		ID:                 row.ID,
		PlanID:             row.PlanID,
		ParentID:           row.ParentID,
		AccountNumber:      row.AccountNumber,
		Description:        row.Description,
		NatureCode:         row.NatureCode,
		ReducedCode:        pgutil.FromPgTextPtr(row.ReducedCode),
		RequiresCostCenter: row.RequiresCostCenter,
		ValidFrom:          pgutil.FromPgDate(row.ValidFrom),
		ValidTo:            pgutil.FromPgDateToPtr(row.ValidTo),
		IsAnalytic:         row.IsAnalytic,
		CreatedAt:          pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Reference Accounts ───────────────────────────────────────────────────────

func (r *AccountingRepositorySQLC) CreateReferenceAccount(ctx context.Context, ref *accountingEntity.AccountingReferenceAccount) (*accountingEntity.AccountingReferenceAccount, error) {
	row, err := r.q.CreateReferenceAccount(ctx, sqlc.CreateReferenceAccountParams{
		InstitutionCode: int32(ref.InstitutionCode),
		ParentRefID:     ref.ParentRefID,
		AccountNumber:   ref.AccountNumber,
		Description:     ref.Description,
		AccountType:     ref.AccountType,
	})
	if err != nil {
		return nil, fmt.Errorf("creating reference account: %w", err)
	}
	return refAccountRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) GetReferenceAccount(ctx context.Context, id int64) (*accountingEntity.AccountingReferenceAccount, error) {
	row, err := r.q.GetReferenceAccount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting reference account %d: %w", id, err)
	}
	return refAccountRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) ListReferenceAccounts(ctx context.Context, institutionCode int) ([]*accountingEntity.AccountingReferenceAccount, error) {
	rows, err := r.q.ListReferenceAccounts(ctx, int32(institutionCode))
	if err != nil {
		return nil, fmt.Errorf("listing reference accounts: %w", err)
	}
	out := make([]*accountingEntity.AccountingReferenceAccount, len(rows))
	for i, row := range rows {
		out[i] = refAccountRowToEntity(row)
	}
	return out, nil
}

func refAccountRowToEntity(row sqlc.ReferenceAccountRow) *accountingEntity.AccountingReferenceAccount {
	return &accountingEntity.AccountingReferenceAccount{
		ID:              row.ID,
		InstitutionCode: int(row.InstitutionCode),
		ParentRefID:     row.ParentRefID,
		AccountNumber:   row.AccountNumber,
		Description:     row.Description,
		AccountType:     row.AccountType,
		CreatedAt:       pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Account Refs ─────────────────────────────────────────────────────────────

func (r *AccountingRepositorySQLC) CreateAccountRef(ctx context.Context, ref *accountingEntity.AccountingAccountRef) (*accountingEntity.AccountingAccountRef, error) {
	row, err := r.q.CreateAccountRef(ctx, sqlc.CreateAccountRefParams{
		AccountID:    ref.AccountID,
		RefAccountID: ref.RefAccountID,
		EmpresaID:    int32(ref.EmpresaID),
		CostCenterID: ref.CostCenterID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating account ref: %w", err)
	}
	return accountRefRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) ListAccountRefs(ctx context.Context, empresaID int) ([]*accountingEntity.AccountingAccountRef, error) {
	rows, err := r.q.ListAccountRefs(ctx, int32(empresaID))
	if err != nil {
		return nil, fmt.Errorf("listing account refs: %w", err)
	}
	out := make([]*accountingEntity.AccountingAccountRef, len(rows))
	for i, row := range rows {
		out[i] = accountRefRowToEntity(row)
	}
	return out, nil
}

func accountRefRowToEntity(row sqlc.AccountRefRow) *accountingEntity.AccountingAccountRef {
	return &accountingEntity.AccountingAccountRef{
		ID:           row.ID,
		AccountID:    row.AccountID,
		RefAccountID: row.RefAccountID,
		EmpresaID:    int(row.EmpresaID),
		CostCenterID: row.CostCenterID,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Journal Entries ──────────────────────────────────────────────────────────

func (r *AccountingRepositorySQLC) CreateJournalEntry(ctx context.Context, e *accountingEntity.AccountingJournalEntry) (*accountingEntity.AccountingJournalEntry, error) {
	row, err := r.q.CreateJournalEntry(ctx, sqlc.CreateJournalEntryParams{
		PlanID:          e.PlanID,
		EmpresaID:       int32(e.EmpresaID),
		EntryDate:       pgutil.ToPgDate(e.EntryDate),
		EntryNumber:     e.EntryNumber,
		BatchNumber:     e.BatchNumber,
		DebitAccountID:  e.DebitAccountID,
		CreditAccountID: e.CreditAccountID,
		DebitCCID:       e.DebitCCID,
		CreditCCID:      e.CreditCCID,
		Value:           pgutil.ToPgNumericFromFloat64(e.Value),
		HistoryCode:     e.HistoryCode,
		Description:     e.Description,
		EntryType:       e.EntryType,
	})
	if err != nil {
		return nil, fmt.Errorf("creating journal entry: %w", err)
	}
	return journalEntryRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) ListJournalEntries(ctx context.Context, planID int64, empresaID int, from, to time.Time) ([]*accountingEntity.AccountingJournalEntry, error) {
	rows, err := r.q.ListJournalEntries(ctx, sqlc.ListJournalEntriesParams{
		PlanID:    planID,
		EmpresaID: int32(empresaID),
		From:      from,
		To:        to,
	})
	if err != nil {
		return nil, fmt.Errorf("listing journal entries: %w", err)
	}
	out := make([]*accountingEntity.AccountingJournalEntry, len(rows))
	for i, row := range rows {
		out[i] = journalEntryRowToEntity(row)
	}
	return out, nil
}

func journalEntryRowToEntity(row sqlc.JournalEntryRow) *accountingEntity.AccountingJournalEntry {
	return &accountingEntity.AccountingJournalEntry{
		ID:              row.ID,
		PlanID:          row.PlanID,
		EmpresaID:       int(row.EmpresaID),
		EntryDate:       pgutil.FromPgDate(row.EntryDate),
		EntryNumber:     row.EntryNumber,
		BatchNumber:     row.BatchNumber,
		DebitAccountID:  row.DebitAccountID,
		CreditAccountID: row.CreditAccountID,
		DebitCCID:       row.DebitCCID,
		CreditCCID:      row.CreditCCID,
		Value:           pgutil.FromPgNumericToFloat64(row.Value),
		HistoryCode:     row.HistoryCode,
		Description:     row.Description,
		EntryType:       row.EntryType,
		CreatedAt:       pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Demonstratives ───────────────────────────────────────────────────────────

func (r *AccountingRepositorySQLC) CreateDemonstrative(ctx context.Context, d *accountingEntity.AccountingDemonstrative) (*accountingEntity.AccountingDemonstrative, error) {
	row, err := r.q.CreateDemonstrative(ctx, sqlc.CreateDemonstrativeParams{
		Code:        d.Code,
		Description: d.Description,
		TermText:    d.TermText,
	})
	if err != nil {
		return nil, fmt.Errorf("creating demonstrative: %w", err)
	}
	return demonstrativeRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) GetDemonstrative(ctx context.Context, id int64) (*accountingEntity.AccountingDemonstrative, error) {
	row, err := r.q.GetDemonstrative(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting demonstrative %d: %w", id, err)
	}
	return demonstrativeRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) ListDemonstratives(ctx context.Context) ([]*accountingEntity.AccountingDemonstrative, error) {
	rows, err := r.q.ListDemonstratives(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing demonstratives: %w", err)
	}
	out := make([]*accountingEntity.AccountingDemonstrative, len(rows))
	for i, row := range rows {
		out[i] = demonstrativeRowToEntity(row)
	}
	return out, nil
}

func demonstrativeRowToEntity(row sqlc.DemonstrativeRow) *accountingEntity.AccountingDemonstrative {
	return &accountingEntity.AccountingDemonstrative{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		TermText:    row.TermText,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

// ─── Demonstrative Items ──────────────────────────────────────────────────────

func (r *AccountingRepositorySQLC) CreateDemonstrativeItem(ctx context.Context, item *accountingEntity.AccountingDemonstrativeItem) (*accountingEntity.AccountingDemonstrativeItem, error) {
	row, err := r.q.CreateDemonstrativeItem(ctx, sqlc.CreateDemonstrativeItemParams{
		DemonstrativeID: item.DemonstrativeID,
		ItemCode:        int32(item.ItemCode),
		Description:     item.Description,
		Formula:         item.Formula,
		IndicatorGroup:  item.IndicatorGroup,
		ShowInReport:    item.ShowInReport,
		ShowBold:        item.ShowBold,
		IsResult:        item.IsResult,
		Is100Pct:        item.Is100Pct,
		SpedEcfDigit:    item.SpedEcfDigit,
		SpedEcfType:     item.SpedEcfType,
	})
	if err != nil {
		return nil, fmt.Errorf("creating demonstrative item: %w", err)
	}
	return demonstrativeItemRowToEntity(row), nil
}

func (r *AccountingRepositorySQLC) ListDemonstrativeItems(ctx context.Context, demonstrativeID int64) ([]*accountingEntity.AccountingDemonstrativeItem, error) {
	rows, err := r.q.ListDemonstrativeItems(ctx, demonstrativeID)
	if err != nil {
		return nil, fmt.Errorf("listing demonstrative items: %w", err)
	}
	out := make([]*accountingEntity.AccountingDemonstrativeItem, len(rows))
	for i, row := range rows {
		out[i] = demonstrativeItemRowToEntity(row)
	}
	return out, nil
}

func demonstrativeItemRowToEntity(row sqlc.DemonstrativeItemRow) *accountingEntity.AccountingDemonstrativeItem {
	return &accountingEntity.AccountingDemonstrativeItem{
		ID:              row.ID,
		DemonstrativeID: row.DemonstrativeID,
		ItemCode:        int(row.ItemCode),
		Description:     row.Description,
		Formula:         row.Formula,
		IndicatorGroup:  row.IndicatorGroup,
		ShowInReport:    row.ShowInReport,
		ShowBold:        row.ShowBold,
		IsResult:        row.IsResult,
		Is100Pct:        row.Is100Pct,
		SpedEcfDigit:    row.SpedEcfDigit,
		SpedEcfType:     row.SpedEcfType,
		CreatedAt:       pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}
