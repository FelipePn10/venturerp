package fiscal_params_uc

import (
	"context"
	"errors"
	"regexp"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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

func (uc *DAPITransferReasonUseCase) Create(ctx context.Context, d *entity.DAPITransferReason) (*response.DAPITransferReasonResponse, error) {
	if d.Code == "" {
		return nil, errors.New("code is required")
	}
	if d.Reason == "" {
		return nil, errors.New("reason is required")
	}
	d.IsActive = true
	created, err := uc.Repo.CreateDAPITransferReason(ctx, d)
	if err != nil {
		return nil, err
	}
	return toDAPITransferReasonResponse(created), nil
}

func (uc *DAPITransferReasonUseCase) Update(ctx context.Context, d *entity.DAPITransferReason) (*response.DAPITransferReasonResponse, error) {
	if d.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.UpdateDAPITransferReason(ctx, d)
	if err != nil {
		return nil, err
	}
	return toDAPITransferReasonResponse(updated), nil
}

func (uc *DAPITransferReasonUseCase) GetByCode(ctx context.Context, code string) (*response.DAPITransferReasonResponse, error) {
	d, err := uc.Repo.GetDAPITransferReasonByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toDAPITransferReasonResponse(d), nil
}

func (uc *DAPITransferReasonUseCase) List(ctx context.Context, onlyActive bool) ([]*response.DAPITransferReasonResponse, error) {
	list, err := uc.Repo.ListDAPITransferReasons(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toDAPITransferReasonResponses(list), nil
}

// ─── ICMS Apuração Adjustment Codes ──────────────────────────────────────────

type ICMSApuracaoAdjCodeUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSApuracaoAdjCodeUseCase) Create(ctx context.Context, c *entity.ICMSApuracaoAdjustmentCode) (*response.ICMSApuracaoAdjCodeResponse, error) {
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
	created, err := uc.Repo.CreateICMSApuracaoAdjCode(ctx, c)
	if err != nil {
		return nil, err
	}
	return toICMSApuracaoAdjCodeResponse(created), nil
}

func (uc *ICMSApuracaoAdjCodeUseCase) Update(ctx context.Context, c *entity.ICMSApuracaoAdjustmentCode) (*response.ICMSApuracaoAdjCodeResponse, error) {
	if c.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.UpdateICMSApuracaoAdjCode(ctx, c)
	if err != nil {
		return nil, err
	}
	return toICMSApuracaoAdjCodeResponse(updated), nil
}

func (uc *ICMSApuracaoAdjCodeUseCase) GetByID(ctx context.Context, id int64) (*response.ICMSApuracaoAdjCodeResponse, error) {
	c, err := uc.Repo.GetICMSApuracaoAdjCode(ctx, id)
	if err != nil {
		return nil, err
	}
	return toICMSApuracaoAdjCodeResponse(c), nil
}

func (uc *ICMSApuracaoAdjCodeUseCase) List(ctx context.Context, uf string, onlyActive bool) ([]*response.ICMSApuracaoAdjCodeResponse, error) {
	list, err := uc.Repo.ListICMSApuracaoAdjCodes(ctx, uf, onlyActive)
	if err != nil {
		return nil, err
	}
	return toICMSApuracaoAdjCodeResponses(list), nil
}

// ─── ICMS Adjustment Codes ────────────────────────────────────────────────────

var validTableRefs = map[string]bool{"5.2": true, "5.3": true, "5.6": true, "5.7": true}

type ICMSAdjustmentCodeUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSAdjustmentCodeUseCase) Create(ctx context.Context, c *entity.ICMSAdjustmentCode) (*response.ICMSAdjustmentCodeResponse, error) {
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
	created, err := uc.Repo.CreateICMSAdjustmentCode(ctx, c)
	if err != nil {
		return nil, err
	}
	return toICMSAdjustmentCodeResponse(created), nil
}

func (uc *ICMSAdjustmentCodeUseCase) Update(ctx context.Context, c *entity.ICMSAdjustmentCode) (*response.ICMSAdjustmentCodeResponse, error) {
	if c.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.UpdateICMSAdjustmentCode(ctx, c)
	if err != nil {
		return nil, err
	}
	return toICMSAdjustmentCodeResponse(updated), nil
}

func (uc *ICMSAdjustmentCodeUseCase) GetByID(ctx context.Context, id int64) (*response.ICMSAdjustmentCodeResponse, error) {
	c, err := uc.Repo.GetICMSAdjustmentCode(ctx, id)
	if err != nil {
		return nil, err
	}
	return toICMSAdjustmentCodeResponse(c), nil
}

func (uc *ICMSAdjustmentCodeUseCase) List(ctx context.Context, uf, tableRef string, onlyActive bool) ([]*response.ICMSAdjustmentCodeResponse, error) {
	list, err := uc.Repo.ListICMSAdjustmentCodes(ctx, uf, tableRef, onlyActive)
	if err != nil {
		return nil, err
	}
	return toICMSAdjustmentCodeResponses(list), nil
}

// ─── ICMS Apuração Lines ──────────────────────────────────────────────────────

var validLineTypes = map[string]bool{
	"DEBITO": true, "CREDITO": true, "SALDO": true, "DEDUCAO": true, "OUTROS": true,
}

type ICMSApuracaoLineUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSApuracaoLineUseCase) Create(ctx context.Context, l *entity.ICMSApuracaoLine) (*response.ICMSApuracaoLineResponse, error) {
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
	created, err := uc.Repo.CreateICMSApuracaoLine(ctx, l)
	if err != nil {
		return nil, err
	}
	return toICMSApuracaoLineResponse(created), nil
}

func (uc *ICMSApuracaoLineUseCase) Update(ctx context.Context, l *entity.ICMSApuracaoLine) (*response.ICMSApuracaoLineResponse, error) {
	if l.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.UpdateICMSApuracaoLine(ctx, l)
	if err != nil {
		return nil, err
	}
	return toICMSApuracaoLineResponse(updated), nil
}

func (uc *ICMSApuracaoLineUseCase) GetByCode(ctx context.Context, code string) (*response.ICMSApuracaoLineResponse, error) {
	l, err := uc.Repo.GetICMSApuracaoLine(ctx, code)
	if err != nil {
		return nil, err
	}
	return toICMSApuracaoLineResponse(l), nil
}

func (uc *ICMSApuracaoLineUseCase) List(ctx context.Context, onlyActive bool) ([]*response.ICMSApuracaoLineResponse, error) {
	list, err := uc.Repo.ListICMSApuracaoLines(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toICMSApuracaoLineResponses(list), nil
}

// ─── ICMS Summary Entries ─────────────────────────────────────────────────────

type ICMSSummaryEntryUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSSummaryEntryUseCase) Create(ctx context.Context, e *entity.ICMSSummaryEntry) (*response.ICMSSummaryEntryResponse, error) {
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
	created, err := uc.Repo.CreateICMSSummaryEntry(ctx, e)
	if err != nil {
		return nil, err
	}
	return toICMSSummaryEntryResponse(created), nil
}

func (uc *ICMSSummaryEntryUseCase) Update(ctx context.Context, e *entity.ICMSSummaryEntry) (*response.ICMSSummaryEntryResponse, error) {
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
	updated, err := uc.Repo.UpdateICMSSummaryEntry(ctx, e)
	if err != nil {
		return nil, err
	}
	return toICMSSummaryEntryResponse(updated), nil
}

func (uc *ICMSSummaryEntryUseCase) GetByID(ctx context.Context, id int64) (*response.ICMSSummaryEntryResponse, error) {
	e, err := uc.Repo.GetICMSSummaryEntry(ctx, id)
	if err != nil {
		return nil, err
	}
	return toICMSSummaryEntryResponse(e), nil
}

func (uc *ICMSSummaryEntryUseCase) List(ctx context.Context, period, uf string) ([]*response.ICMSSummaryEntryResponse, error) {
	list, err := uc.Repo.ListICMSSummaryEntries(ctx, period, uf)
	if err != nil {
		return nil, err
	}
	return toICMSSummaryEntryResponses(list), nil
}

func (uc *ICMSSummaryEntryUseCase) AddNote(ctx context.Context, n *entity.ICMSSummaryEntryNote) (*response.ICMSSummaryEntryNoteResponse, error) {
	if n.SummaryEntryID == 0 {
		return nil, errors.New("summary_entry_id is required")
	}
	if n.NoteNumber == "" {
		return nil, errors.New("note_number is required")
	}
	created, err := uc.Repo.AddICMSSummaryEntryNote(ctx, n)
	if err != nil {
		return nil, err
	}
	return toICMSSummaryEntryNoteResponse(created), nil
}

func (uc *ICMSSummaryEntryUseCase) ListNotes(ctx context.Context, summaryEntryID int64) ([]*response.ICMSSummaryEntryNoteResponse, error) {
	list, err := uc.Repo.ListICMSSummaryEntryNotes(ctx, summaryEntryID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ICMSSummaryEntryNoteResponse, 0, len(list))
	for _, n := range list {
		out = append(out, toICMSSummaryEntryNoteResponse(n))
	}
	return out, nil
}

// ─── Simples Nacional Apuração ────────────────────────────────────────────────

var validAnnexes = map[string]bool{"I": true, "II": true, "III": true, "IV": true, "V": true, "VI": true}

type SimplesNacionalUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *SimplesNacionalUseCase) Create(ctx context.Context, s *entity.SimplesNacionalApuracao) (*response.SimplesNacionalApuracaoResponse, error) {
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
	created, err := uc.Repo.CreateSimplesNacionalApuracao(ctx, s)
	if err != nil {
		return nil, err
	}
	return toSimplesNacionalApuracaoResponse(created), nil
}

func (uc *SimplesNacionalUseCase) Update(ctx context.Context, s *entity.SimplesNacionalApuracao) (*response.SimplesNacionalApuracaoResponse, error) {
	if s.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.UpdateSimplesNacionalApuracao(ctx, s)
	if err != nil {
		return nil, err
	}
	return toSimplesNacionalApuracaoResponse(updated), nil
}

func (uc *SimplesNacionalUseCase) Get(ctx context.Context, period string, annex entity.SimplesNacionalAnnex) (*response.SimplesNacionalApuracaoResponse, error) {
	s, err := uc.Repo.GetSimplesNacionalApuracao(ctx, period, annex)
	if err != nil {
		return nil, err
	}
	return toSimplesNacionalApuracaoResponse(s), nil
}

func (uc *SimplesNacionalUseCase) List(ctx context.Context, period string) ([]*response.SimplesNacionalApuracaoResponse, error) {
	list, err := uc.Repo.ListSimplesNacionalApuracoes(ctx, period)
	if err != nil {
		return nil, err
	}
	return toSimplesNacionalApuracaoResponses(list), nil
}
