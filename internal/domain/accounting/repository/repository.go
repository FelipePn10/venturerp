package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/entity"
)

type AccountingRepository interface {
	CreateAccountingPlan(ctx context.Context, p *entity.AccountingPlan) (*entity.AccountingPlan, error)
	GetAccountingPlan(ctx context.Context, id int64) (*entity.AccountingPlan, error)
	ListAccountingPlans(ctx context.Context) ([]*entity.AccountingPlan, error)
	UpdateAccountingPlan(ctx context.Context, p *entity.AccountingPlan) (*entity.AccountingPlan, error)

	CreateAccountingAccount(ctx context.Context, a *entity.AccountingAccount) (*entity.AccountingAccount, error)
	GetAccountingAccount(ctx context.Context, id int64) (*entity.AccountingAccount, error)
	ListAccountingAccountsByPlan(ctx context.Context, planID int64) ([]*entity.AccountingAccount, error)
	UpdateAccountingAccount(ctx context.Context, a *entity.AccountingAccount) (*entity.AccountingAccount, error)

	CreateReferenceAccount(ctx context.Context, r *entity.AccountingReferenceAccount) (*entity.AccountingReferenceAccount, error)
	GetReferenceAccount(ctx context.Context, id int64) (*entity.AccountingReferenceAccount, error)
	ListReferenceAccounts(ctx context.Context, institutionCode int) ([]*entity.AccountingReferenceAccount, error)

	CreateAccountRef(ctx context.Context, r *entity.AccountingAccountRef) (*entity.AccountingAccountRef, error)
	ListAccountRefs(ctx context.Context, empresaID int) ([]*entity.AccountingAccountRef, error)

	CreateJournalEntry(ctx context.Context, e *entity.AccountingJournalEntry) (*entity.AccountingJournalEntry, error)
	ListJournalEntries(ctx context.Context, planID int64, empresaID int, from, to time.Time) ([]*entity.AccountingJournalEntry, error)

	CreateDemonstrative(ctx context.Context, d *entity.AccountingDemonstrative) (*entity.AccountingDemonstrative, error)
	GetDemonstrative(ctx context.Context, id int64) (*entity.AccountingDemonstrative, error)
	ListDemonstratives(ctx context.Context) ([]*entity.AccountingDemonstrative, error)

	CreateDemonstrativeItem(ctx context.Context, i *entity.AccountingDemonstrativeItem) (*entity.AccountingDemonstrativeItem, error)
	ListDemonstrativeItems(ctx context.Context, demonstrativeID int64) ([]*entity.AccountingDemonstrativeItem, error)
}
