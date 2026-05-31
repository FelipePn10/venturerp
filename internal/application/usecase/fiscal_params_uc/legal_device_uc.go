package fiscal_params_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type LegalDeviceUseCase struct {
	Repo repository.FiscalParamsRepository
}

var validLegalDeviceTypes = map[string]bool{
	"ICMS": true, "IPI": true, "LAUDO": true, "PIS": true, "COFINS": true,
}

func (uc *LegalDeviceUseCase) Create(ctx context.Context, dto request.CreateLegalDeviceDTO) (*entity.LegalDevice, error) {
	if dto.Description == "" {
		return nil, errors.New("description is required")
	}
	if !validLegalDeviceTypes[dto.Type] {
		return nil, errors.New("type must be one of: ICMS, IPI, LAUDO, PIS, COFINS")
	}
	d := &entity.LegalDevice{
		Type:        entity.LegalDeviceType(dto.Type),
		Description: dto.Description,
		IsActive:    true,
	}
	return uc.Repo.CreateLegalDevice(ctx, d)
}

func (uc *LegalDeviceUseCase) Update(ctx context.Context, dto request.UpdateLegalDeviceDTO) (*entity.LegalDevice, error) {
	if !validLegalDeviceTypes[dto.Type] {
		return nil, errors.New("type must be one of: ICMS, IPI, LAUDO, PIS, COFINS")
	}
	d := &entity.LegalDevice{
		ID:          dto.ID,
		Type:        entity.LegalDeviceType(dto.Type),
		Description: dto.Description,
		IsActive:    dto.IsActive,
	}
	return uc.Repo.UpdateLegalDevice(ctx, d)
}

func (uc *LegalDeviceUseCase) GetByCode(ctx context.Context, code int64) (*entity.LegalDevice, error) {
	return uc.Repo.GetLegalDeviceByCode(ctx, code)
}

func (uc *LegalDeviceUseCase) List(ctx context.Context, onlyActive bool) ([]*entity.LegalDevice, error) {
	return uc.Repo.ListLegalDevices(ctx, onlyActive)
}

func (uc *LegalDeviceUseCase) ListByType(ctx context.Context, devType string, onlyActive bool) ([]*entity.LegalDevice, error) {
	return uc.Repo.ListLegalDevicesByType(ctx, entity.LegalDeviceType(devType), onlyActive)
}
