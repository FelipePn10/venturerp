// Package configurator_uc implements the Product Configurator (Fase 1):
// Conjuntos/Variáveis, Características (com tipos) e Características do Item, além
// da geração de máscara com ponte para item_masks. Usa *sqlc.Queries direto
// (como tool_sheet_uc/order_operations_uc).
package configurator_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/configurator/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

// RestrictionOracle validates a fully-formed combination of configurator answers
// against the restriction/dependency rules. Implemented by restriction_uc so the
// configurator reuses the existing engine without depending on its package.
type RestrictionOracle interface {
	EvaluateCombination(ctx context.Context, itemCode, customerCode, divisionID *int64, answers map[int64]string) (bool, error)
}

type ConfiguratorUseCase struct {
	Q            *sqlc.Queries
	Restrictions RestrictionOracle // optional; nil ⇒ no restriction filtering
}

func New(q *sqlc.Queries) *ConfiguratorUseCase { return &ConfiguratorUseCase{Q: q} }

// WithRestrictions wires the restriction oracle used by the cartesian generator.
func (uc *ConfiguratorUseCase) WithRestrictions(o RestrictionOracle) *ConfiguratorUseCase {
	uc.Restrictions = o
	return uc
}

// ─── Conjuntos ────────────────────────────────────────────────────────────────

func (uc *ConfiguratorUseCase) CreateSet(ctx context.Context, dto request.CreateCfgSetDTO) (*response.CfgSetResponse, error) {
	s, err := entity.NewSet(dto.Description, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	row, err := uc.Q.CreateCfgSet(ctx, s.Description, pgutil.ToPgUUID(dto.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("criando conjunto: %w", err)
	}
	return setToResponse(row), nil
}

func (uc *ConfiguratorUseCase) UpdateSet(ctx context.Context, dto request.UpdateCfgSetDTO) (*response.CfgSetResponse, error) {
	if dto.Description == "" {
		return nil, fmt.Errorf("descrição do conjunto é obrigatória")
	}
	row, err := uc.Q.UpdateCfgSet(ctx, dto.ID, dto.Description, dto.IsActive)
	if err != nil {
		return nil, fmt.Errorf("atualizando conjunto: %w", err)
	}
	return setToResponse(row), nil
}

func (uc *ConfiguratorUseCase) GetSet(ctx context.Context, id int64) (*response.CfgSetResponse, error) {
	row, err := uc.Q.GetCfgSet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("conjunto não encontrado: %w", err)
	}
	return setToResponse(row), nil
}

func (uc *ConfiguratorUseCase) ListSets(ctx context.Context, onlyActive bool) ([]*response.CfgSetResponse, error) {
	rows, err := uc.Q.ListCfgSets(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgSetResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, setToResponse(r))
	}
	return out, nil
}

func (uc *ConfiguratorUseCase) DeactivateSet(ctx context.Context, id int64) error {
	return uc.Q.DeactivateCfgSet(ctx, id)
}

// ─── Variáveis ────────────────────────────────────────────────────────────────

func (uc *ConfiguratorUseCase) CreateVariable(ctx context.Context, dto request.CreateCfgVariableDTO) (*response.CfgVariableResponse, error) {
	v, err := entity.NewVariable(dto.SetID, dto.Code, dto.Description, dto.MaskComposition, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	if _, err := uc.Q.GetCfgSet(ctx, dto.SetID); err != nil {
		return nil, fmt.Errorf("conjunto %d não encontrado", dto.SetID)
	}
	row, err := uc.Q.CreateCfgVariable(ctx, sqlc.CreateCfgVariableParams{
		SetID:              v.SetID,
		Code:               v.Code,
		Description:        v.Description,
		MaskComposition:    v.MaskComposition,
		IsSpecial:          dto.IsSpecial,
		IncludeDescription: dto.IncludeDescription,
		SpecialData:        textOrNull(dto.SpecialData),
		Marketing:          dto.Marketing,
		CreatedBy:          pgutil.ToPgUUID(dto.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("criando variável: %w", err)
	}
	return variableToResponse(row, nil), nil
}

func (uc *ConfiguratorUseCase) UpdateVariable(ctx context.Context, dto request.UpdateCfgVariableDTO) (*response.CfgVariableResponse, error) {
	if dto.Code == "" || dto.Description == "" {
		return nil, fmt.Errorf("código e descrição da variável são obrigatórios")
	}
	maskComp := dto.MaskComposition
	if maskComp == "" {
		maskComp = dto.Code
	}
	row, err := uc.Q.UpdateCfgVariable(ctx, sqlc.UpdateCfgVariableParams{
		ID:                 dto.ID,
		Code:               dto.Code,
		Description:        dto.Description,
		MaskComposition:    maskComp,
		IsActive:           dto.IsActive,
		IsSpecial:          dto.IsSpecial,
		IncludeDescription: dto.IncludeDescription,
		SpecialData:        textOrNull(dto.SpecialData),
		Marketing:          dto.Marketing,
	})
	if err != nil {
		return nil, fmt.Errorf("atualizando variável: %w", err)
	}
	langs, _ := uc.Q.ListCfgVariableLanguages(ctx, dto.ID)
	return variableToResponse(row, langs), nil
}

func (uc *ConfiguratorUseCase) GetVariable(ctx context.Context, id int64) (*response.CfgVariableResponse, error) {
	row, err := uc.Q.GetCfgVariable(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("variável não encontrada: %w", err)
	}
	langs, _ := uc.Q.ListCfgVariableLanguages(ctx, id)
	return variableToResponse(row, langs), nil
}

func (uc *ConfiguratorUseCase) ListVariablesBySet(ctx context.Context, setID int64, onlyActive bool) ([]*response.CfgVariableResponse, error) {
	rows, err := uc.Q.ListCfgVariablesBySet(ctx, setID, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CfgVariableResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, variableToResponse(r, nil))
	}
	return out, nil
}

func (uc *ConfiguratorUseCase) DeactivateVariable(ctx context.Context, id int64) error {
	return uc.Q.DeactivateCfgVariable(ctx, id)
}

// Variable languages

func (uc *ConfiguratorUseCase) SetVariableLanguage(ctx context.Context, variableID int64, dto request.CfgVariableLanguageDTO) (*response.CfgVariableLanguageResponse, error) {
	if dto.Language == "" || dto.Translation == "" {
		return nil, fmt.Errorf("idioma e tradução são obrigatórios")
	}
	row, err := uc.Q.UpsertCfgVariableLanguage(ctx, variableID, dto.Language, textOrNull(dto.Country), dto.Translation)
	if err != nil {
		return nil, fmt.Errorf("gravando idioma da variável: %w", err)
	}
	return &response.CfgVariableLanguageResponse{
		ID: row.ID, VariableID: row.VariableID, Language: row.Language,
		Country: pgutil.FromPgText(row.Country), Translation: row.Translation,
	}, nil
}

func (uc *ConfiguratorUseCase) DeleteVariableLanguage(ctx context.Context, id int64) error {
	return uc.Q.DeleteCfgVariableLanguage(ctx, id)
}
