package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

type FiscalParamsRepository interface {
	// ── Legal Devices ──────────────────────────────────────────────────────
	CreateLegalDevice(ctx context.Context, d *entity.LegalDevice) (*entity.LegalDevice, error)
	UpdateLegalDevice(ctx context.Context, d *entity.LegalDevice) (*entity.LegalDevice, error)
	GetLegalDeviceByCode(ctx context.Context, code int64) (*entity.LegalDevice, error)
	ListLegalDevices(ctx context.Context, onlyActive bool) ([]*entity.LegalDevice, error)
	ListLegalDevicesByType(ctx context.Context, devType entity.LegalDeviceType, onlyActive bool) ([]*entity.LegalDevice, error)
	NextLegalDeviceCode(ctx context.Context) (int64, error)

	// ── CFOP ───────────────────────────────────────────────────────────────
	CreateCFOP(ctx context.Context, c *entity.CFOP) (*entity.CFOP, error)
	UpdateCFOP(ctx context.Context, c *entity.CFOP) (*entity.CFOP, error)
	GetCFOPByCode(ctx context.Context, code int32) (*entity.CFOP, error)
	ListCFOPs(ctx context.Context, onlyActive bool) ([]*entity.CFOP, error)
	ListCFOPsByDirection(ctx context.Context, direction string, onlyActive bool) ([]*entity.CFOP, error)

	// ── ICMS/IPI Tax Params ────────────────────────────────────────────────
	CreateTaxParam(ctx context.Context, p *entity.ICMSIPITaxParam) (*entity.ICMSIPITaxParam, error)
	UpdateTaxParam(ctx context.Context, p *entity.ICMSIPITaxParam) (*entity.ICMSIPITaxParam, error)
	GetTaxParamByID(ctx context.Context, id int64) (*entity.ICMSIPITaxParam, error)
	ListTaxParams(ctx context.Context, onlyActive bool) ([]*entity.ICMSIPITaxParam, error)
	ListTaxParamsByUF(ctx context.Context, uf string, onlyActive bool) ([]*entity.ICMSIPITaxParam, error)
	ListTaxParamsByItem(ctx context.Context, itemCode int64, onlyActive bool) ([]*entity.ICMSIPITaxParam, error)
	ListTaxParamsByNCM(ctx context.Context, ncmCode string, onlyActive bool) ([]*entity.ICMSIPITaxParam, error)

	// ── DAPI Transfer Reasons ──────────────────────────────────────────────
	CreateDAPITransferReason(ctx context.Context, r *entity.DAPITransferReason) (*entity.DAPITransferReason, error)
	UpdateDAPITransferReason(ctx context.Context, r *entity.DAPITransferReason) (*entity.DAPITransferReason, error)
	GetDAPITransferReasonByCode(ctx context.Context, code string) (*entity.DAPITransferReason, error)
	ListDAPITransferReasons(ctx context.Context, onlyActive bool) ([]*entity.DAPITransferReason, error)

	// ── ICMS Apuração Adjustment Codes (tabela 5.1.1) ─────────────────────
	CreateICMSApuracaoAdjCode(ctx context.Context, c *entity.ICMSApuracaoAdjustmentCode) (*entity.ICMSApuracaoAdjustmentCode, error)
	UpdateICMSApuracaoAdjCode(ctx context.Context, c *entity.ICMSApuracaoAdjustmentCode) (*entity.ICMSApuracaoAdjustmentCode, error)
	GetICMSApuracaoAdjCode(ctx context.Context, id int64) (*entity.ICMSApuracaoAdjustmentCode, error)
	ListICMSApuracaoAdjCodes(ctx context.Context, uf string, onlyActive bool) ([]*entity.ICMSApuracaoAdjustmentCode, error)

	// ── ICMS Adjustment Codes (tabelas 5.2/5.3/5.6/5.7) ──────────────────
	CreateICMSAdjustmentCode(ctx context.Context, c *entity.ICMSAdjustmentCode) (*entity.ICMSAdjustmentCode, error)
	UpdateICMSAdjustmentCode(ctx context.Context, c *entity.ICMSAdjustmentCode) (*entity.ICMSAdjustmentCode, error)
	GetICMSAdjustmentCode(ctx context.Context, id int64) (*entity.ICMSAdjustmentCode, error)
	ListICMSAdjustmentCodes(ctx context.Context, uf string, tableRef string, onlyActive bool) ([]*entity.ICMSAdjustmentCode, error)

	// ── ICMS Apuração Lines ────────────────────────────────────────────────
	CreateICMSApuracaoLine(ctx context.Context, l *entity.ICMSApuracaoLine) (*entity.ICMSApuracaoLine, error)
	UpdateICMSApuracaoLine(ctx context.Context, l *entity.ICMSApuracaoLine) (*entity.ICMSApuracaoLine, error)
	GetICMSApuracaoLine(ctx context.Context, code string) (*entity.ICMSApuracaoLine, error)
	ListICMSApuracaoLines(ctx context.Context, onlyActive bool) ([]*entity.ICMSApuracaoLine, error)

	// ── ICMS Summary Entries ───────────────────────────────────────────────
	CreateICMSSummaryEntry(ctx context.Context, e *entity.ICMSSummaryEntry) (*entity.ICMSSummaryEntry, error)
	UpdateICMSSummaryEntry(ctx context.Context, e *entity.ICMSSummaryEntry) (*entity.ICMSSummaryEntry, error)
	GetICMSSummaryEntry(ctx context.Context, id int64) (*entity.ICMSSummaryEntry, error)
	ListICMSSummaryEntries(ctx context.Context, period string, uf string) ([]*entity.ICMSSummaryEntry, error)
	AddICMSSummaryEntryNote(ctx context.Context, n *entity.ICMSSummaryEntryNote) (*entity.ICMSSummaryEntryNote, error)
	ListICMSSummaryEntryNotes(ctx context.Context, summaryEntryID int64) ([]*entity.ICMSSummaryEntryNote, error)

	// ── Simples Nacional Apuração ──────────────────────────────────────────
	CreateSimplesNacionalApuracao(ctx context.Context, s *entity.SimplesNacionalApuracao) (*entity.SimplesNacionalApuracao, error)
	UpdateSimplesNacionalApuracao(ctx context.Context, s *entity.SimplesNacionalApuracao) (*entity.SimplesNacionalApuracao, error)
	GetSimplesNacionalApuracao(ctx context.Context, period string, annex entity.SimplesNacionalAnnex) (*entity.SimplesNacionalApuracao, error)
	ListSimplesNacionalApuracoes(ctx context.Context, period string) ([]*entity.SimplesNacionalApuracao, error)

	// ── ICMS Reduction / Substitution / Deferral ───────────────────────────
	CreateICMSReductionSubstitution(ctx context.Context, r *entity.ICMSReductionSubstitution) (*entity.ICMSReductionSubstitution, error)
	UpdateICMSReductionSubstitution(ctx context.Context, r *entity.ICMSReductionSubstitution) (*entity.ICMSReductionSubstitution, error)
	GetICMSReductionSubstitution(ctx context.Context, id int64) (*entity.ICMSReductionSubstitution, error)
	ListICMSReductionSubstitutions(ctx context.Context, uf string, itemID *int64, onlyActive bool) ([]*entity.ICMSReductionSubstitution, error)
	FindICMSReductionSubstitution(ctx context.Context, uf string, itemID *int64, customerID *int64, opType entity.ICMSOperationType) (*entity.ICMSReductionSubstitution, error)

	// ── ICMS Summary Entry Additionals (aba Adicionais) ────────────────────
	AddICMSSummaryEntryAdditional(ctx context.Context, a *entity.ICMSSummaryEntryAdditional) (*entity.ICMSSummaryEntryAdditional, error)
	ListICMSSummaryEntryAdditionals(ctx context.Context, summaryEntryID int64) ([]*entity.ICMSSummaryEntryAdditional, error)

	// ── ICMS ST Restitution ────────────────────────────────────────────────
	CreateICMSSTRestitution(ctx context.Context, r *entity.ICMSSTRestitution) (*entity.ICMSSTRestitution, error)
	UpdateICMSSTRestitution(ctx context.Context, r *entity.ICMSSTRestitution) (*entity.ICMSSTRestitution, error)
	GetICMSSTRestitution(ctx context.Context, id int64) (*entity.ICMSSTRestitution, error)
	ListICMSSTRestitutions(ctx context.Context, empresaID int, period string, uf string) ([]*entity.ICMSSTRestitution, error)

	// ── Special Adjustment Notes ───────────────────────────────────────────
	CreateSpecialAdjustmentNote(ctx context.Context, n *entity.SpecialAdjustmentNote) (*entity.SpecialAdjustmentNote, error)
	UpdateSpecialAdjustmentNote(ctx context.Context, n *entity.SpecialAdjustmentNote) (*entity.SpecialAdjustmentNote, error)
	GetSpecialAdjustmentNote(ctx context.Context, id int64) (*entity.SpecialAdjustmentNote, error)
	ListSpecialAdjustmentNotes(ctx context.Context, empresaID int, period string) ([]*entity.SpecialAdjustmentNote, error)
	AddSpecialAdjustmentNoteItem(ctx context.Context, item *entity.SpecialAdjustmentNoteItem) (*entity.SpecialAdjustmentNoteItem, error)
	ListSpecialAdjustmentNoteItems(ctx context.Context, noteID int64) ([]*entity.SpecialAdjustmentNoteItem, error)
}
