// Package unit_of_measure_uc mantém o catálogo de unidades da empresa, os
// sinônimos (texto da nota → unidade) e as conversões padrão.
package unit_of_measure_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
)

type UseCase struct {
	repo repository.UnitRepository
	auth ports.AuthService
}

func New(repo repository.UnitRepository, auth ports.AuthService) *UseCase {
	return &UseCase{repo: repo, auth: auth}
}

func (uc *UseCase) actor(ctx context.Context) (*uuid.UUID, error) {
	if uc.auth == nil {
		return nil, nil
	}
	id, err := uc.auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func toUnit(u *entity.Unit) *response.UnitOfMeasureResponse {
	return &response.UnitOfMeasureResponse{ID: u.ID, Code: u.Code, Description: u.Description, Dimension: string(u.Dimension),
		FiscalCode: u.FiscalCode, Decimals: u.Decimals, IsSystem: u.IsSystem, IsActive: u.IsActive, UpdatedAt: u.UpdatedAt}
}

func (uc *UseCase) ListUnits(ctx context.Context, includeInactive bool) ([]*response.UnitOfMeasureResponse, error) {
	list, err := uc.repo.ListUnits(ctx, includeInactive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.UnitOfMeasureResponse, 0, len(list))
	for _, u := range list {
		out = append(out, toUnit(u))
	}
	return out, nil
}

func (uc *UseCase) CreateUnit(ctx context.Context, dto request.UnitOfMeasureDTO) (*response.UnitOfMeasureResponse, error) {
	u, err := entity.NewUnit(dto.Code, dto.Description, entity.Dimensao(strings.ToUpper(strings.TrimSpace(dto.Dimension))), dto.FiscalCode, dto.Decimals)
	if err != nil {
		return nil, err
	}
	if u.CreatedBy, err = uc.actor(ctx); err != nil {
		return nil, err
	}
	// Um código que já é sinônimo de outra unidade criaria ambiguidade na nota.
	res, err := uc.repo.Resolver(ctx, nil, []string{u.Code})
	if err != nil {
		return nil, err
	}
	if r := res[u.Code]; r.Unidade != "" && r.Unidade != u.Code && strings.HasPrefix(r.Via, "SINONIMO") {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("%s já é sinônimo de %s: exclua o sinônimo antes de criar a unidade", u.Code, r.Unidade))
	}
	created, err := uc.repo.CreateUnit(ctx, u)
	if err != nil {
		return nil, err
	}
	return toUnit(created), nil
}

// UpdateUnit: o código não muda. Mudar a dimensão de unidade em uso exige
// cuidado: conversões padrão que dependem dela deixam de valer.
func (uc *UseCase) UpdateUnit(ctx context.Context, code string, dto request.UnitOfMeasureDTO) (*response.UnitOfMeasureResponse, error) {
	atual, err := uc.repo.GetUnit(ctx, code)
	if err != nil {
		return nil, err
	}
	u, err := entity.NewUnit(atual.Code, dto.Description, entity.Dimensao(strings.ToUpper(strings.TrimSpace(dto.Dimension))), dto.FiscalCode, dto.Decimals)
	if err != nil {
		return nil, err
	}
	if atual.IsSystem && u.Dimension != atual.Dimension {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s é uma unidade padrão do sistema: a dimensão (%s) não pode mudar", atual.Code, atual.Dimension))
	}
	if u.Dimension != atual.Dimension {
		// A conversão padrão só vale dentro da mesma dimensão: trocar a
		// dimensão deixaria M→X (por exemplo) ligando grandezas diferentes.
		padroes, err := uc.repo.ListStandardConversions(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range padroes {
			if c.FromUOM == atual.Code || c.ToUOM == atual.Code {
				return nil, errorsuc.NewConflictError(fmt.Sprintf("%s tem a conversão padrão %s → %s: exclua-a antes de mudar a dimensão da unidade",
					atual.Code, c.FromUOM, c.ToUOM))
			}
		}
	}
	salvo, err := uc.repo.UpdateUnit(ctx, u)
	if err != nil {
		return nil, err
	}
	return toUnit(salvo), nil
}

// SetActive inativa ou reativa. Unidade em uso não é inativada: itens,
// estruturas e conversões que a usam ficariam apontando para o nada.
func (uc *UseCase) SetActive(ctx context.Context, code string, active bool) error {
	if !active {
		n, err := uc.repo.UnitUsage(ctx, code)
		if err != nil {
			return err
		}
		if n > 0 {
			return errorsuc.NewConflictError(fmt.Sprintf("a unidade %s está em uso em %d cadastro(s) (itens, estruturas ou conversões) e não pode ser inativada",
				entity.NormalizarCodigo(code), n))
		}
	}
	return uc.repo.SetUnitActive(ctx, code, active)
}

func toAlias(a *entity.Alias) *response.UnitAliasResponse {
	return &response.UnitAliasResponse{ID: a.ID, Alias: a.Alias, UnitCode: a.UnitCode, SupplierCode: a.SupplierCode,
		SupplierName: a.SupplierName, IsSystem: a.IsSystem, CreatedAt: a.CreatedAt}
}

func (uc *UseCase) ListAliases(ctx context.Context) ([]*response.UnitAliasResponse, error) {
	list, err := uc.repo.ListAliases(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.UnitAliasResponse, 0, len(list))
	for _, a := range list {
		out = append(out, toAlias(a))
	}
	return out, nil
}

func (uc *UseCase) CreateAlias(ctx context.Context, dto request.UnitAliasDTO) (*response.UnitAliasResponse, error) {
	a, err := entity.NewAlias(dto.Alias, dto.UnitCode, dto.SupplierCode)
	if err != nil {
		return nil, err
	}
	if a.CreatedBy, err = uc.actor(ctx); err != nil {
		return nil, err
	}
	unidade, err := uc.repo.GetUnit(ctx, a.UnitCode)
	if err != nil {
		return nil, err
	}
	if !unidade.IsActive {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a unidade %s está inativa", unidade.Code))
	}
	if a.SupplierCode != nil {
		ok, err := uc.repo.SupplierExists(ctx, *a.SupplierCode)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("fornecedor %d não encontrado nesta empresa", *a.SupplierCode))
		}
	}
	// O sinônimo não pode ser o código de OUTRA unidade do catálogo.
	if _, err := uc.repo.GetUnit(ctx, a.Alias); err == nil {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("%s já é o código de uma unidade do catálogo: não pode ser sinônimo de %s", a.Alias, a.UnitCode))
	}
	created, err := uc.repo.CreateAlias(ctx, a)
	if err != nil {
		return nil, err
	}
	return toAlias(created), nil
}

func (uc *UseCase) DeleteAlias(ctx context.Context, id int64) error {
	return uc.repo.DeleteAlias(ctx, id)
}

func toStandard(c *entity.StandardConversion) *response.StandardConversionResponse {
	out := &response.StandardConversionResponse{ID: c.ID, FromUOM: c.FromUOM, ToUOM: c.ToUOM, FromQuantity: c.FromQty, ToQuantity: c.ToQty,
		Factor: c.Factor, Dimension: string(c.FromDim), IsSystem: c.IsSystem}
	if c.ToQty.IsPositive() {
		out.InverseFactor = c.FromQty.DivRound(c.ToQty, 10)
	}
	return out
}

func (uc *UseCase) ListStandardConversions(ctx context.Context) ([]*response.StandardConversionResponse, error) {
	list, err := uc.repo.ListStandardConversions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.StandardConversionResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toStandard(c))
	}
	return out, nil
}

// SaveStandardConversion: só entre unidades da mesma dimensão — a padrão vale
// para todos os itens, e KG→UN não é verdade para todos os itens.
func (uc *UseCase) SaveStandardConversion(ctx context.Context, dto request.StandardConversionDTO) (*response.StandardConversionResponse, error) {
	de, err := uc.repo.GetUnit(ctx, dto.FromUOM)
	if err != nil {
		return nil, err
	}
	para, err := uc.repo.GetUnit(ctx, dto.ToUOM)
	if err != nil {
		return nil, err
	}
	for _, u := range []*entity.Unit{de, para} {
		if !u.IsActive {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a unidade %s está inativa no catálogo", u.Code))
		}
	}
	c, err := entity.NewStandardConversion(de.Code, para.Code, dto.FromQuantity, dto.ToQuantity, de.Dimension, para.Dimension)
	if err != nil {
		return nil, err
	}
	salvo, err := uc.repo.UpsertStandardConversion(ctx, c)
	if err != nil {
		return nil, err
	}
	return toStandard(salvo), nil
}

func (uc *UseCase) DeleteStandardConversion(ctx context.Context, id int64) error {
	return uc.repo.DeleteStandardConversion(ctx, id)
}

// Resolve traduz textos recebidos (uCom da nota) para o catálogo.
func (uc *UseCase) Resolve(ctx context.Context, supplier *int64, textos []string) ([]response.UnitResolutionResponse, error) {
	res, err := uc.repo.Resolver(ctx, supplier, textos)
	if err != nil {
		return nil, err
	}
	out := make([]response.UnitResolutionResponse, 0, len(textos))
	for _, t := range textos {
		r := res[t]
		out = append(out, response.UnitResolutionResponse{Text: t, Unit: r.Unidade, Via: r.Via})
	}
	return out, nil
}
