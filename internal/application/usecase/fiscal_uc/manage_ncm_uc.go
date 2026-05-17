package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type UpsertNcmTaxUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *UpsertNcmTaxUseCase) Execute(ctx context.Context, dto request.UpsertNcmTaxDTO) (*entity.NcmTaxTable, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	n := &entity.NcmTaxTable{
		Ncm:         dto.Ncm,
		AliqIPI:     dto.AliqIPI,
		AliqPis:     dto.AliqPis,
		AliqCofins:  dto.AliqCofins,
		CstPis:      dto.CstPis,
		CstCofins:   dto.CstCofins,
		CstIPI:      dto.CstIPI,
		Description: dto.Description,
		IsActive:    true,
	}
	return uc.Repo.UpsertNcmTax(ctx, n)
}

type ListNcmTaxesUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListNcmTaxesUseCase) Execute(ctx context.Context) ([]*entity.NcmTaxTable, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListNcmTaxes(ctx)
}

type DeleteNcmTaxUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *DeleteNcmTaxUseCase) Execute(ctx context.Context, ncm string) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.DeleteNcmTax(ctx, ncm)
}

type UpsertICMSInterstateUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *UpsertICMSInterstateUseCase) Execute(ctx context.Context, dto request.UpsertICMSInterstateDTO) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.UpsertICMSInterstate(ctx, dto.OriginUF, dto.DestinationUF, dto.AliqICMS)
}

type ListICMSInterstateUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListICMSInterstateUseCase) Execute(ctx context.Context) (map[string]float64, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListICMSInterstate(ctx)
}

type UpsertICMSInternalUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *UpsertICMSInternalUseCase) Execute(ctx context.Context, dto request.UpsertICMSInternalDTO) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.UpsertICMSInternal(ctx, dto.UF, dto.AliqICMS, dto.AliqFCP)
}

type ListICMSInternalUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListICMSInternalUseCase) Execute(ctx context.Context) (map[string]struct{ ICMS, FCP float64 }, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListICMSInternal(ctx)
}
