package entry_operation_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/repository"
)

type EntryOperationUseCase struct {
	repo repository.EntryOperationRepository
}

func NewEntryOperationUseCase(repo repository.EntryOperationRepository) *EntryOperationUseCase {
	return &EntryOperationUseCase{repo: repo}
}

// ─── State Groups ─────────────────────────────────────────────────────────────

func (uc *EntryOperationUseCase) CreateStateGroup(ctx context.Context, dto request.CreateStateGroupDTO) (*entity.StateGroup, error) {
	code, err := uc.repo.NextStateGroupCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	g, err := entity.NewStateGroup(code, dto.Description, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateStateGroup(ctx, g)
	if err != nil {
		return nil, err
	}
	for _, uf := range dto.UFs {
		uf = strings.ToUpper(strings.TrimSpace(uf))
		if uf == "" {
			continue
		}
		if aerr := uc.repo.AddStateGroupUF(ctx, created.Code, uf); aerr != nil {
			return nil, aerr
		}
		created.UFs = append(created.UFs, uf)
	}
	return created, nil
}

func (uc *EntryOperationUseCase) GetStateGroup(ctx context.Context, code int64) (*entity.StateGroup, error) {
	return uc.repo.GetStateGroupByCode(ctx, code)
}

func (uc *EntryOperationUseCase) ListStateGroups(ctx context.Context) ([]*entity.StateGroup, error) {
	return uc.repo.ListStateGroups(ctx)
}

func (uc *EntryOperationUseCase) AddStateGroupUF(ctx context.Context, dto request.AddStateGroupUFDTO) error {
	uf := strings.ToUpper(strings.TrimSpace(dto.UF))
	if uf == "" {
		return fmt.Errorf("uf is required")
	}
	return uc.repo.AddStateGroupUF(ctx, dto.StateGroupCode, uf)
}

// ─── Entry Operation Types ────────────────────────────────────────────────────

func (uc *EntryOperationUseCase) CreateEntryOperation(ctx context.Context, dto request.CreateEntryOperationDTO) (*entity.EntryOperationType, error) {
	code, err := uc.repo.NextEntryOperationCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}
	o, err := entity.NewEntryOperationType(code, dto.Description, dto.NatureOperation, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	o.InvoiceTypeCode = dto.InvoiceTypeCode
	o.ClassificationType = dto.ClassificationType
	o.ClassificationCode = dto.ClassificationCode
	o.StateGroupCode = dto.StateGroupCode
	o.SupplierTypeCode = dto.SupplierTypeCode
	return uc.repo.CreateEntryOperation(ctx, o)
}

func (uc *EntryOperationUseCase) UpdateEntryOperation(ctx context.Context, dto request.UpdateEntryOperationDTO) (*entity.EntryOperationType, error) {
	o, err := uc.repo.GetEntryOperationByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	o.Description = dto.Description
	if dto.NatureOperation != "" {
		o.NatureOperation = dto.NatureOperation
	}
	o.InvoiceTypeCode = dto.InvoiceTypeCode
	o.ClassificationType = dto.ClassificationType
	o.ClassificationCode = dto.ClassificationCode
	o.StateGroupCode = dto.StateGroupCode
	o.SupplierTypeCode = dto.SupplierTypeCode
	o.IsActive = dto.IsActive
	return uc.repo.UpdateEntryOperation(ctx, o)
}

func (uc *EntryOperationUseCase) GetEntryOperation(ctx context.Context, code int64) (*entity.EntryOperationType, error) {
	return uc.repo.GetEntryOperationByCode(ctx, code)
}

func (uc *EntryOperationUseCase) ListEntryOperations(ctx context.Context, onlyActive bool) ([]*entity.EntryOperationType, error) {
	return uc.repo.ListEntryOperations(ctx, onlyActive)
}

// ValidateUF applies the UF × Grupo de Estado rule for an entry operation type.
func (uc *EntryOperationUseCase) ValidateUF(ctx context.Context, code int64, enterpriseUF string) error {
	o, err := uc.repo.GetEntryOperationByCode(ctx, code)
	if err != nil {
		return err
	}
	enterpriseUF = strings.ToUpper(strings.TrimSpace(enterpriseUF))
	ufInGroup := false
	if o.StateGroupCode != nil {
		if in, gerr := uc.repo.UFInGroup(ctx, *o.StateGroupCode, enterpriseUF); gerr == nil {
			ufInGroup = in
		}
	}
	return o.ValidateUF(enterpriseUF, ufInGroup)
}
