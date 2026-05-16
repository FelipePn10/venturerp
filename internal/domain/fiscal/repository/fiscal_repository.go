package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

type FiscalRepository interface {
	// Fiscal Entries
	CreateEntry(ctx context.Context, e *entity.FiscalEntry) (*entity.FiscalEntry, error)
	CreateEntryItem(ctx context.Context, item *entity.FiscalEntryItem) (*entity.FiscalEntryItem, error)
	GetEntryByID(ctx context.Context, id int64) (*entity.FiscalEntry, error)
	GetEntryItems(ctx context.Context, fiscalEntryID int64) ([]*entity.FiscalEntryItem, error)
	ListEntries(ctx context.Context) ([]*entity.FiscalEntry, error)
	ListEntriesByStatus(ctx context.Context, status entity.FiscalEntryStatus) ([]*entity.FiscalEntry, error)
	UpdateEntryStatus(ctx context.Context, id int64, status entity.FiscalEntryStatus) (*entity.FiscalEntry, error)
	GetNextNFNumber(ctx context.Context) (int64, error)

	// Fiscal Exits
	CreateExit(ctx context.Context, e *entity.FiscalExit) (*entity.FiscalExit, error)
	CreateExitItem(ctx context.Context, item *entity.FiscalExitItem) (*entity.FiscalExitItem, error)
	GetExitByID(ctx context.Context, id int64) (*entity.FiscalExit, error)
	GetExitItems(ctx context.Context, fiscalExitID int64) ([]*entity.FiscalExitItem, error)
	ListExits(ctx context.Context) ([]*entity.FiscalExit, error)
	ListExitsByStatus(ctx context.Context, status entity.FiscalExitStatus) ([]*entity.FiscalExit, error)
	UpdateExitStatus(ctx context.Context, id int64, status entity.FiscalExitStatus) (*entity.FiscalExit, error)
	UpdateExitAuthorization(ctx context.Context, id int64, chaveAcesso, protocolo, focusRef string) (*entity.FiscalExit, error)

	// Fiscal Config
	GetFiscalConfig(ctx context.Context) (*entity.FiscalConfig, error)
	UpdateFiscalConfig(ctx context.Context, cfg *entity.FiscalConfig) (*entity.FiscalConfig, error)

	// NCM Tax Table
	GetNcmTax(ctx context.Context, ncm string) (*entity.NcmTaxTable, error)
	ListNcmTaxes(ctx context.Context) ([]*entity.NcmTaxTable, error)

	// Tax Scenarios
	ListTaxScenarios(ctx context.Context) ([]*entity.TaxScenario, error)

	// ICMS Tables
	GetICMSInterstate(ctx context.Context, originUF, destUF string) (*float64, error)
	GetICMSInternal(ctx context.Context, uf string) (*float64, *float64, error)
	ListICMSInterstate(ctx context.Context) (map[string]float64, error)
	ListICMSInternal(ctx context.Context) (map[string]struct{ ICMS, FCP float64 }, error)
}
