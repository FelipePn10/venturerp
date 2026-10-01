package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type UpsertNcmTaxUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *UpsertNcmTaxUseCase) Execute(ctx context.Context, dto request.UpsertNcmTaxDTO) (*response.NcmTaxTableResponse, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	ncm, err := ncmValido(dto.Ncm)
	if err != nil {
		return nil, err
	}
	n := &entity.NcmTaxTable{
		Ncm:         ncm,
		AliqIPI:     dto.AliqIPI,
		AliqPis:     dto.AliqPis,
		AliqCofins:  dto.AliqCofins,
		CstPis:      dto.CstPis,
		CstCofins:   dto.CstCofins,
		CstIPI:      dto.CstIPI,
		Description: dto.Description,
		IsActive:    true,
	}
	saved, err := uc.Repo.UpsertNcmTax(ctx, n)
	if err != nil {
		return nil, err
	}
	return toNcmTaxTableResponse(saved), nil
}

type ListNcmTaxesUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListNcmTaxesUseCase) Execute(ctx context.Context) ([]*response.NcmTaxTableResponse, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListNcmTaxes(ctx)
	if err != nil {
		return nil, err
	}
	return toNcmTaxTableResponses(list), nil
}

type DeleteNcmTaxUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *DeleteNcmTaxUseCase) Execute(ctx context.Context, ncm string) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	limpo, err := ncmValido(ncm)
	if err != nil {
		return err
	}
	return uc.Repo.DeleteNcmTax(ctx, limpo)
}

type UpsertICMSInterstateUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *UpsertICMSInterstateUseCase) Execute(ctx context.Context, dto request.UpsertICMSInterstateDTO) error {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return errorsuc.ErrUnauthorized
	}
	origem, err := ufValida(dto.OriginUF, "origin_uf")
	if err != nil {
		return err
	}
	destino, err := ufValida(dto.DestinationUF, "destination_uf")
	if err != nil {
		return err
	}
	return uc.Repo.UpsertICMSInterstate(ctx, origem, destino, dto.AliqICMS)
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
	uf, err := ufValida(dto.UF, "uf")
	if err != nil {
		return err
	}
	return uc.Repo.UpsertICMSInternal(ctx, uf, dto.AliqICMS, dto.AliqFCP)
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

// ufValida devolve a UF em maiúsculas, recusando o que não é sigla de estado.
//
// A gravação precisa normalizar porque o motor fiscal procura a UF do destinatário
// em MAIÚSCULAS (internalTable[params.DestinoUF]). Uma linha cadastrada como "sp"
// nunca seria encontrada e a nota sairia com ICMS zerado, sem erro nenhum.
func ufValida(valor, campo string) (string, error) {
	uf := strings.ToUpper(strings.TrimSpace(valor))
	if _, ok := ufsDoBrasil[uf]; !ok {
		return "", errorsuc.NewValidationError(fmt.Sprintf(
			"%s: %q não é uma sigla de estado — informe a sigla de duas letras, por exemplo SP", campo, valor))
	}
	return uf, nil
}

var ufsDoBrasil = map[string]struct{}{
	"AC": {}, "AL": {}, "AP": {}, "AM": {}, "BA": {}, "CE": {}, "DF": {}, "ES": {},
	"GO": {}, "MA": {}, "MT": {}, "MS": {}, "MG": {}, "PA": {}, "PB": {}, "PR": {},
	"PE": {}, "PI": {}, "RJ": {}, "RN": {}, "RS": {}, "RO": {}, "RR": {}, "SC": {},
	"SP": {}, "SE": {}, "TO": {},
}

// ncmValido devolve o NCM com 8 dígitos, sem máscara.
//
// Guardar sempre na mesma forma é o que impede o defeito silencioso descrito em
// engine.NormalizarNCM: tabela e item precisam casar, e a SEFAZ recusa máscara no
// campo <NCM> do XML.
func ncmValido(valor string) (string, error) {
	ncm := engine.NormalizarNCM(valor)
	if ncm == "" {
		return "", errorsuc.NewValidationError(
			"ncm: informe a classificação fiscal (8 dígitos), por exemplo 8466.20.90")
	}
	if len(ncm) != 8 {
		return "", errorsuc.NewValidationError(fmt.Sprintf(
			"ncm: %q tem %d dígitos e o NCM tem 8 — confira a classificação na tabela TIPI",
			valor, len(ncm)))
	}
	return ncm, nil
}
