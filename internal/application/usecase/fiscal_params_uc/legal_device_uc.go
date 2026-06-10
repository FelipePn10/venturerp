package fiscal_params_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type LegalDeviceUseCase struct {
	Repo repository.FiscalParamsRepository
}

var validLegalDeviceTypes = map[string]bool{
	"ICMS": true, "IPI": true, "LAUDO": true, "PIS": true, "COFINS": true,
}

func (uc *LegalDeviceUseCase) Create(ctx context.Context, dto request.CreateLegalDeviceDTO) (*response.LegalDeviceResponse, error) {
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
	created, err := uc.Repo.CreateLegalDevice(ctx, d)
	if err != nil {
		return nil, err
	}
	return toLegalDeviceResponse(created), nil
}

func (uc *LegalDeviceUseCase) Update(ctx context.Context, dto request.UpdateLegalDeviceDTO) (*response.LegalDeviceResponse, error) {
	if !validLegalDeviceTypes[dto.Type] {
		return nil, errors.New("type must be one of: ICMS, IPI, LAUDO, PIS, COFINS")
	}
	d := &entity.LegalDevice{
		ID:          dto.ID,
		Type:        entity.LegalDeviceType(dto.Type),
		Description: dto.Description,
		IsActive:    dto.IsActive,
	}
	updated, err := uc.Repo.UpdateLegalDevice(ctx, d)
	if err != nil {
		return nil, err
	}
	return toLegalDeviceResponse(updated), nil
}

func (uc *LegalDeviceUseCase) GetByCode(ctx context.Context, code int64) (*response.LegalDeviceResponse, error) {
	d, err := uc.Repo.GetLegalDeviceByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toLegalDeviceResponse(d), nil
}

func (uc *LegalDeviceUseCase) List(ctx context.Context, onlyActive bool) ([]*response.LegalDeviceResponse, error) {
	list, err := uc.Repo.ListLegalDevices(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	return toLegalDeviceResponses(list), nil
}

func (uc *LegalDeviceUseCase) ListByType(ctx context.Context, devType string, onlyActive bool) ([]*response.LegalDeviceResponse, error) {
	list, err := uc.Repo.ListLegalDevicesByType(ctx, entity.LegalDeviceType(devType), onlyActive)
	if err != nil {
		return nil, err
	}
	return toLegalDeviceResponses(list), nil
}
