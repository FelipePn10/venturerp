package fiscal_params_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// ─── ICMSReductionSubstitution ────────────────────────────────────────────────

type ICMSReductionSubstitutionUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSReductionSubstitutionUseCase) Create(ctx context.Context, r *entity.ICMSReductionSubstitution) (*entity.ICMSReductionSubstitution, error) {
	if r.UF == "" {
		return nil, errors.New("uf is required")
	}
	if r.OperationType == "" {
		r.OperationType = entity.ICMSOpAmbas
	}
	r.IsActive = true
	return uc.Repo.CreateICMSReductionSubstitution(ctx, r)
}

func (uc *ICMSReductionSubstitutionUseCase) Update(ctx context.Context, r *entity.ICMSReductionSubstitution) (*entity.ICMSReductionSubstitution, error) {
	if r.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.UpdateICMSReductionSubstitution(ctx, r)
}

func (uc *ICMSReductionSubstitutionUseCase) GetByID(ctx context.Context, id int64) (*entity.ICMSReductionSubstitution, error) {
	return uc.Repo.GetICMSReductionSubstitution(ctx, id)
}

func (uc *ICMSReductionSubstitutionUseCase) List(ctx context.Context, uf string, itemID *int64, onlyActive bool) ([]*entity.ICMSReductionSubstitution, error) {
	return uc.Repo.ListICMSReductionSubstitutions(ctx, uf, itemID, onlyActive)
}

func (uc *ICMSReductionSubstitutionUseCase) Find(ctx context.Context, uf string, itemID *int64, customerID *int64, opType entity.ICMSOperationType) (*entity.ICMSReductionSubstitution, error) {
	return uc.Repo.FindICMSReductionSubstitution(ctx, uf, itemID, customerID, opType)
}

// ─── ICMS Summary Entry Additional (Aba Adicionais) ──────────────────────────

type ICMSSummaryAdditionalUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSSummaryAdditionalUseCase) Add(ctx context.Context, a *entity.ICMSSummaryEntryAdditional) (*entity.ICMSSummaryEntryAdditional, error) {
	if a.SummaryEntryID == 0 {
		return nil, errors.New("summary_entry_id is required")
	}
	if a.ArrecadacaoIndicator == "" {
		return nil, errors.New("arrecadacao_indicator is required")
	}
	return uc.Repo.AddICMSSummaryEntryAdditional(ctx, a)
}

func (uc *ICMSSummaryAdditionalUseCase) List(ctx context.Context, summaryEntryID int64) ([]*entity.ICMSSummaryEntryAdditional, error) {
	return uc.Repo.ListICMSSummaryEntryAdditionals(ctx, summaryEntryID)
}

// ─── ICMS ST Restitution ──────────────────────────────────────────────────────

type ICMSSTRestitutionUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *ICMSSTRestitutionUseCase) Create(ctx context.Context, r *entity.ICMSSTRestitution) (*entity.ICMSSTRestitution, error) {
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
	return uc.Repo.CreateICMSSTRestitution(ctx, r)
}

func (uc *ICMSSTRestitutionUseCase) Update(ctx context.Context, r *entity.ICMSSTRestitution) (*entity.ICMSSTRestitution, error) {
	if r.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.UpdateICMSSTRestitution(ctx, r)
}

func (uc *ICMSSTRestitutionUseCase) GetByID(ctx context.Context, id int64) (*entity.ICMSSTRestitution, error) {
	return uc.Repo.GetICMSSTRestitution(ctx, id)
}

func (uc *ICMSSTRestitutionUseCase) List(ctx context.Context, empresaID int, period, uf string) ([]*entity.ICMSSTRestitution, error) {
	if err := validatePeriod(period); err != nil {
		return nil, err
	}
	return uc.Repo.ListICMSSTRestitutions(ctx, empresaID, period, uf)
}

// ─── Special Adjustment Note ──────────────────────────────────────────────────

type SpecialAdjustmentNoteUseCase struct {
	Repo repository.FiscalParamsRepository
}

func (uc *SpecialAdjustmentNoteUseCase) Create(ctx context.Context, n *entity.SpecialAdjustmentNote) (*entity.SpecialAdjustmentNote, error) {
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
	return uc.Repo.CreateSpecialAdjustmentNote(ctx, n)
}

func (uc *SpecialAdjustmentNoteUseCase) Update(ctx context.Context, n *entity.SpecialAdjustmentNote) (*entity.SpecialAdjustmentNote, error) {
	if n.ID == 0 {
		return nil, errors.New("id is required")
	}
	return uc.Repo.UpdateSpecialAdjustmentNote(ctx, n)
}

func (uc *SpecialAdjustmentNoteUseCase) GetByID(ctx context.Context, id int64) (*entity.SpecialAdjustmentNote, error) {
	return uc.Repo.GetSpecialAdjustmentNote(ctx, id)
}

func (uc *SpecialAdjustmentNoteUseCase) List(ctx context.Context, empresaID int, period string) ([]*entity.SpecialAdjustmentNote, error) {
	if err := validatePeriod(period); err != nil {
		return nil, err
	}
	return uc.Repo.ListSpecialAdjustmentNotes(ctx, empresaID, period)
}

func (uc *SpecialAdjustmentNoteUseCase) AddItem(ctx context.Context, item *entity.SpecialAdjustmentNoteItem) (*entity.SpecialAdjustmentNoteItem, error) {
	if item.NoteID == 0 {
		return nil, errors.New("note_id is required")
	}
	return uc.Repo.AddSpecialAdjustmentNoteItem(ctx, item)
}

func (uc *SpecialAdjustmentNoteUseCase) ListItems(ctx context.Context, noteID int64) ([]*entity.SpecialAdjustmentNoteItem, error) {
	return uc.Repo.ListSpecialAdjustmentNoteItems(ctx, noteID)
}
