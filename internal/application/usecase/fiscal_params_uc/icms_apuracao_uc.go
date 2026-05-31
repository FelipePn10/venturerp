package fiscal_params_uc

import (
	"context"
	"errors"
	"regexp"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

var reYYYYMM = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func validatePeriod(period string) error {
	if !reYYYYMM.MatchString(period) {
		return errors.New("period must be in YYYY-MM format (e.g. 2024-01)")
	}
	return nil
}

// ─── DAPI Transfer Reasons ────────────────────────────────────────────────────

type DAPITransferReasonUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *DAPITransferReasonUseCase) Create(ctx context.Context, d *entity.DAPITransferReason) (*entity.DAPITransferReason, error) {
	if d.Code == "" {
		return nil, errors.New("code is required")
	}
	if d.Reason == "" {
		return nil, errors.New("reason is required")
	}
	d.IsActive = true
	return uc.Repo.CreateDAPITransferReason(ctx, d)
}

func (uc *DAPITransferReasonUseCase) Update(ctx context.Context, d *entity.DAPITransferReason) (*entity.DAPITransferReason, error) {
	if d.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.UpdateDAPITransferReason(ctx, d)
}

func (uc *DAPITransferReasonUseCase) GetByCode(ctx context.Context, code string) (*entity.DAPITransferReason, error) {
	return uc.Repo.GetDAPITransferReasonByCode(ctx, code)
}

func (uc *DAPITransferReasonUseCase) List(ctx context.Context, onlyActive bool) ([]*entity.DAPITransferReason, error) {
	return uc.Repo.ListDAPITransferReasons(ctx, onlyActive)
}

// ─── ICMS Apuração Adjustment Codes ──────────────────────────────────────────

type ICMSApuracaoAdjCodeUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSApuracaoAdjCodeUseCase) Create(ctx context.Context, c *entity.ICMSApuracaoAdjustmentCode) (*entity.ICMSApuracaoAdjustmentCode, error) {
	if c.Code == "" {
		return nil, errors.New("code is required")
	}
	if c.UF == "" {
		return nil, errors.New("uf is required")
	}
	if c.Description == "" {
		return nil, errors.New("description is required")
	}
	c.IsActive = true
	return uc.Repo.CreateICMSApuracaoAdjCode(ctx, c)
}

func (uc *ICMSApuracaoAdjCodeUseCase) Update(ctx context.Context, c *entity.ICMSApuracaoAdjustmentCode) (*entity.ICMSApuracaoAdjustmentCode, error) {
	if c.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.UpdateICMSApuracaoAdjCode(ctx, c)
}

func (uc *ICMSApuracaoAdjCodeUseCase) GetByID(ctx context.Context, id int64) (*entity.ICMSApuracaoAdjustmentCode, error) {
	return uc.Repo.GetICMSApuracaoAdjCode(ctx, id)
}

func (uc *ICMSApuracaoAdjCodeUseCase) List(ctx context.Context, uf string, onlyActive bool) ([]*entity.ICMSApuracaoAdjustmentCode, error) {
	return uc.Repo.ListICMSApuracaoAdjCodes(ctx, uf, onlyActive)
}

// ─── ICMS Adjustment Codes ────────────────────────────────────────────────────

var validTableRefs = map[string]bool{"5.2": true, "5.3": true, "5.6": true, "5.7": true}

type ICMSAdjustmentCodeUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSAdjustmentCodeUseCase) Create(ctx context.Context, c *entity.ICMSAdjustmentCode) (*entity.ICMSAdjustmentCode, error) {
	if c.UF == "" {
		return nil, errors.New("uf is required")
	}
	if c.Code == "" {
		return nil, errors.New("code is required")
	}
	if !validTableRefs[string(c.TableRef)] {
		return nil, errors.New("table_ref must be one of: 5.2, 5.3, 5.6, 5.7")
	}
	c.IsActive = true
	return uc.Repo.CreateICMSAdjustmentCode(ctx, c)
}

func (uc *ICMSAdjustmentCodeUseCase) Update(ctx context.Context, c *entity.ICMSAdjustmentCode) (*entity.ICMSAdjustmentCode, error) {
	if c.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.UpdateICMSAdjustmentCode(ctx, c)
}

func (uc *ICMSAdjustmentCodeUseCase) GetByID(ctx context.Context, id int64) (*entity.ICMSAdjustmentCode, error) {
	return uc.Repo.GetICMSAdjustmentCode(ctx, id)
}

func (uc *ICMSAdjustmentCodeUseCase) List(ctx context.Context, uf, tableRef string, onlyActive bool) ([]*entity.ICMSAdjustmentCode, error) {
	return uc.Repo.ListICMSAdjustmentCodes(ctx, uf, tableRef, onlyActive)
}

// ─── ICMS Apuração Lines ──────────────────────────────────────────────────────

var validLineTypes = map[string]bool{
	"DEBITO": true, "CREDITO": true, "SALDO": true, "DEDUCAO": true, "OUTROS": true,
}

type ICMSApuracaoLineUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSApuracaoLineUseCase) Create(ctx context.Context, l *entity.ICMSApuracaoLine) (*entity.ICMSApuracaoLine, error) {
	if l.Code == "" {
		return nil, errors.New("code is required")
	}
	if l.Description == "" {
		return nil, errors.New("description is required")
	}
	if !validLineTypes[string(l.LineType)] {
		l.LineType = entity.LineTypeOutros
	}
	l.IsActive = true
	l.AcceptsEntries = true
	return uc.Repo.CreateICMSApuracaoLine(ctx, l)
}

func (uc *ICMSApuracaoLineUseCase) Update(ctx context.Context, l *entity.ICMSApuracaoLine) (*entity.ICMSApuracaoLine, error) {
	if l.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.UpdateICMSApuracaoLine(ctx, l)
}

func (uc *ICMSApuracaoLineUseCase) GetByCode(ctx context.Context, code string) (*entity.ICMSApuracaoLine, error) {
	return uc.Repo.GetICMSApuracaoLine(ctx, code)
}

func (uc *ICMSApuracaoLineUseCase) List(ctx context.Context, onlyActive bool) ([]*entity.ICMSApuracaoLine, error) {
	return uc.Repo.ListICMSApuracaoLines(ctx, onlyActive)
}

// ─── ICMS Summary Entries ─────────────────────────────────────────────────────

type ICMSSummaryEntryUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSSummaryEntryUseCase) Create(ctx context.Context, e *entity.ICMSSummaryEntry) (*entity.ICMSSummaryEntry, error) {
	if err := validatePeriod(e.Period); err != nil {
		return nil, err
	}
	if e.UF == "" {
		return nil, errors.New("uf is required")
	}
	if e.ICMSBase < 0 || e.ICMSValue < 0 {
		return nil, errors.New("icms_base and icms_value must be >= 0")
	}
	e.IsActive = true
	return uc.Repo.CreateICMSSummaryEntry(ctx, e)
}

func (uc *ICMSSummaryEntryUseCase) Update(ctx context.Context, e *entity.ICMSSummaryEntry) (*entity.ICMSSummaryEntry, error) {
	if e.ID == 0 {
		return nil, errors.New("id is required")
	}
	if e.Period != "" {
		if err := validatePeriod(e.Period); err != nil {
			return nil, err
		}
	}
	if e.ICMSBase < 0 || e.ICMSValue < 0 {
		return nil, errors.New("icms_base and icms_value must be >= 0")
	}
	return uc.Repo.UpdateICMSSummaryEntry(ctx, e)
}

func (uc *ICMSSummaryEntryUseCase) GetByID(ctx context.Context, id int64) (*entity.ICMSSummaryEntry, error) {
	return uc.Repo.GetICMSSummaryEntry(ctx, id)
}

func (uc *ICMSSummaryEntryUseCase) List(ctx context.Context, period, uf string) ([]*entity.ICMSSummaryEntry, error) {
	return uc.Repo.ListICMSSummaryEntries(ctx, period, uf)
}

func (uc *ICMSSummaryEntryUseCase) AddNote(ctx context.Context, n *entity.ICMSSummaryEntryNote) (*entity.ICMSSummaryEntryNote, error) {
	if n.SummaryEntryID == 0 {
		return nil, errors.New("summary_entry_id is required")
	}
	if n.NoteNumber == "" {
		return nil, errors.New("note_number is required")
	}
	return uc.Repo.AddICMSSummaryEntryNote(ctx, n)
}

func (uc *ICMSSummaryEntryUseCase) ListNotes(ctx context.Context, summaryEntryID int64) ([]*entity.ICMSSummaryEntryNote, error) {
	return uc.Repo.ListICMSSummaryEntryNotes(ctx, summaryEntryID)
}

// ─── Simples Nacional Apuração ────────────────────────────────────────────────

var validAnnexes = map[string]bool{"I": true, "II": true, "III": true, "IV": true, "V": true, "VI": true}

type SimplesNacionalUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *SimplesNacionalUseCase) Create(ctx context.Context, s *entity.SimplesNacionalApuracao) (*entity.SimplesNacionalApuracao, error) {
	if err := validatePeriod(s.Period); err != nil {
		return nil, err
	}
	if !validAnnexes[string(s.Annex)] {
		return nil, errors.New("annex must be one of: I, II, III, IV, V, VI")
	}
	if s.ReceitaInterna < 0 || s.ReceitaExterna < 0 {
		return nil, errors.New("receita values must be >= 0")
	}
	s.IsActive = true
	return uc.Repo.CreateSimplesNacionalApuracao(ctx, s)
}

func (uc *SimplesNacionalUseCase) Update(ctx context.Context, s *entity.SimplesNacionalApuracao) (*entity.SimplesNacionalApuracao, error) {
	if s.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.UpdateSimplesNacionalApuracao(ctx, s)
}

func (uc *SimplesNacionalUseCase) Get(ctx context.Context, period string, annex entity.SimplesNacionalAnnex) (*entity.SimplesNacionalApuracao, error) {
	return uc.Repo.GetSimplesNacionalApuracao(ctx, period, annex)
}

func (uc *SimplesNacionalUseCase) List(ctx context.Context, period string) ([]*entity.SimplesNacionalApuracao, error) {
	return uc.Repo.ListSimplesNacionalApuracoes(ctx, period)
}
