package fiscal_params_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// ─── ICMSReductionSubstitution ────────────────────────────────────────────────

type ICMSReductionSubstitutionUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSReductionSubstitutionUseCase) Create(ctx context.Context, r *entity.ICMSReductionSubstitution) (*response.ICMSReductionSubstitutionResponse, error) {
	if r.UF == "" {
		return nil, errors.New("uf is required")
	}
	if r.OperationType == "" {
		r.OperationType = entity.ICMSOpAmbas
	}
	r.IsActive = true
	created, err := uc.Repo.CreateICMSReductionSubstitution(ctx, r)
	if err != nil {
		return nil, err
	}
	return toICMSReductionSubstitutionResponse(created), nil
}

func (uc *ICMSReductionSubstitutionUseCase) Update(ctx context.Context, r *entity.ICMSReductionSubstitution) (*response.ICMSReductionSubstitutionResponse, error) {
	if r.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.UpdateICMSReductionSubstitution(ctx, r)
	if err != nil {
		return nil, err
	}
	return toICMSReductionSubstitutionResponse(updated), nil
}

func (uc *ICMSReductionSubstitutionUseCase) GetByID(ctx context.Context, id int64) (*response.ICMSReductionSubstitutionResponse, error) {
	r, err := uc.Repo.GetICMSReductionSubstitution(ctx, id)
	if err != nil {
		return nil, err
	}
	return toICMSReductionSubstitutionResponse(r), nil
}

func (uc *ICMSReductionSubstitutionUseCase) List(ctx context.Context, uf string, itemID *int64, onlyActive bool) ([]*response.ICMSReductionSubstitutionResponse, error) {
	list, err := uc.Repo.ListICMSReductionSubstitutions(ctx, uf, itemID, onlyActive)
	if err != nil {
		return nil, err
	}
	return toICMSReductionSubstitutionResponses(list), nil
}

func (uc *ICMSReductionSubstitutionUseCase) Find(ctx context.Context, uf string, itemID *int64, customerID *int64, opType entity.ICMSOperationType) (*response.ICMSReductionSubstitutionResponse, error) {
	r, err := uc.Repo.FindICMSReductionSubstitution(ctx, uf, itemID, customerID, opType)
	if err != nil {
		return nil, err
	}
	return toICMSReductionSubstitutionResponse(r), nil
}

// ─── ICMS Summary Entry Additional (Aba Adicionais) ──────────────────────────

type ICMSSummaryAdditionalUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSSummaryAdditionalUseCase) Add(ctx context.Context, a *entity.ICMSSummaryEntryAdditional) (*response.ICMSSummaryEntryAdditionalResponse, error) {
	if a.SummaryEntryID == 0 {
		return nil, errors.New("summary_entry_id is required")
	}
	if a.ArrecadacaoIndicator == "" {
		return nil, errors.New("arrecadacao_indicator is required")
	}
	created, err := uc.Repo.AddICMSSummaryEntryAdditional(ctx, a)
	if err != nil {
		return nil, err
	}
	return toICMSSummaryEntryAdditionalResponse(created), nil
}

func (uc *ICMSSummaryAdditionalUseCase) List(ctx context.Context, summaryEntryID int64) ([]*response.ICMSSummaryEntryAdditionalResponse, error) {
	list, err := uc.Repo.ListICMSSummaryEntryAdditionals(ctx, summaryEntryID)
	if err != nil {
		return nil, err
	}
	return toICMSSummaryEntryAdditionalResponses(list), nil
}

// ─── ICMS ST Restitution ──────────────────────────────────────────────────────

type ICMSSTRestitutionUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSSTRestitutionUseCase) Create(ctx context.Context, r *entity.ICMSSTRestitution) (*response.ICMSSTRestitutionResponse, error) {
	if r.EmpresaID == 0 {
		return nil, errors.New("empresa_id is required")
	}
	if err := validatePeriod(r.Period); err != nil {
		return nil, err
	}
	if r.UF == "" {
		return nil, errors.New("uf is required")
	}
	if r.RestitutionType == "" {
		return nil, errors.New("restitution_type is required")
	}
	r.IsActive = true
	created, err := uc.Repo.CreateICMSSTRestitution(ctx, r)
	if err != nil {
		return nil, err
	}
	return toICMSSTRestitutionResponse(created), nil
}

func (uc *ICMSSTRestitutionUseCase) Update(ctx context.Context, r *entity.ICMSSTRestitution) (*response.ICMSSTRestitutionResponse, error) {
	if r.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.UpdateICMSSTRestitution(ctx, r)
	if err != nil {
		return nil, err
	}
	return toICMSSTRestitutionResponse(updated), nil
}

func (uc *ICMSSTRestitutionUseCase) GetByID(ctx context.Context, id int64) (*response.ICMSSTRestitutionResponse, error) {
	r, err := uc.Repo.GetICMSSTRestitution(ctx, id)
	if err != nil {
		return nil, err
	}
	return toICMSSTRestitutionResponse(r), nil
}

func (uc *ICMSSTRestitutionUseCase) List(ctx context.Context, empresaID int, period, uf string) ([]*response.ICMSSTRestitutionResponse, error) {
	if err := validatePeriod(period); err != nil {
		return nil, err
	}
	list, err := uc.Repo.ListICMSSTRestitutions(ctx, empresaID, period, uf)
	if err != nil {
		return nil, err
	}
	return toICMSSTRestitutionResponses(list), nil
}

// ─── Special Adjustment Note ──────────────────────────────────────────────────

type SpecialAdjustmentNoteUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *SpecialAdjustmentNoteUseCase) Create(ctx context.Context, n *entity.SpecialAdjustmentNote) (*response.SpecialAdjustmentNoteResponse, error) {
	if n.EmpresaID == 0 {
		return nil, errors.New("empresa_id is required")
	}
	if n.Purpose == "" {
		return nil, errors.New("purpose is required")
	}
	if err := validatePeriod(n.Period); err != nil {
		return nil, err
	}
	if n.IssueDate.IsZero() {
		return nil, errors.New("issue_date is required")
	}
	n.Status = entity.SpecialNoteRascunho
	created, err := uc.Repo.CreateSpecialAdjustmentNote(ctx, n)
	if err != nil {
		return nil, err
	}
	return toSpecialAdjustmentNoteResponse(created), nil
}

func (uc *SpecialAdjustmentNoteUseCase) Update(ctx context.Context, n *entity.SpecialAdjustmentNote) (*response.SpecialAdjustmentNoteResponse, error) {
	if n.ID == 0 {
		return nil, errors.New("id is required")
	}
	updated, err := uc.Repo.UpdateSpecialAdjustmentNote(ctx, n)
	if err != nil {
		return nil, err
	}
	return toSpecialAdjustmentNoteResponse(updated), nil
}

func (uc *SpecialAdjustmentNoteUseCase) GetByID(ctx context.Context, id int64) (*response.SpecialAdjustmentNoteResponse, error) {
	n, err := uc.Repo.GetSpecialAdjustmentNote(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSpecialAdjustmentNoteResponse(n), nil
}

func (uc *SpecialAdjustmentNoteUseCase) List(ctx context.Context, empresaID int, period string) ([]*response.SpecialAdjustmentNoteResponse, error) {
	if err := validatePeriod(period); err != nil {
		return nil, err
	}
	list, err := uc.Repo.ListSpecialAdjustmentNotes(ctx, empresaID, period)
	if err != nil {
		return nil, err
	}
	return toSpecialAdjustmentNoteResponses(list), nil
}

func (uc *SpecialAdjustmentNoteUseCase) AddItem(ctx context.Context, item *entity.SpecialAdjustmentNoteItem) (*response.SpecialAdjustmentNoteItemResponse, error) {
	if item.NoteID == 0 {
		return nil, errors.New("note_id is required")
	}
	created, err := uc.Repo.AddSpecialAdjustmentNoteItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return toSpecialAdjustmentNoteItemResponse(created), nil
}

func (uc *SpecialAdjustmentNoteUseCase) ListItems(ctx context.Context, noteID int64) ([]*response.SpecialAdjustmentNoteItemResponse, error) {
	list, err := uc.Repo.ListSpecialAdjustmentNoteItems(ctx, noteID)
	if err != nil {
		return nil, err
	}
	return toSpecialAdjustmentNoteItemResponses(list), nil
}
