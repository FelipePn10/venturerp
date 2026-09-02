package structure_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/google/uuid"
)

// configuratorPort é a fatia do configurador que a Estrutura de Produto consome
// pelo botão — sem tela própria, como no FoccoERP (FENG0210 → configurador).
type configuratorPort interface {
	StructurePanel(ctx context.Context, itemCode int64, itemName string) (*response.StructureConfiguratorPanelResponse, error)
	ValidateCombination(ctx context.Context, itemCode int64, answers []request.CfgMaskAnswerInput) ([]response.StructureConfiguratorViolation, error)
	GenerateMask(ctx context.Context, dto request.CfgGenerateMaskDTO) (*response.CfgGeneratedMaskResponse, error)
	FormulaVariables(ctx context.Context, answers []request.CfgMaskAnswerInput) map[string]float64
}

// structureResolver resolve a árvore da estrutura para uma configuração.
type structureResolver interface {
	Execute(ctx context.Context, dto request.ResolveStructureQueryDTO) (*response.StructureTreeResponse, error)
}

// StructureConfiguratorUseCase liga a Estrutura de Produto ao configurador: o
// painel do botão e a aplicação das respostas (restrições → máscara → estrutura
// resolvida com as fórmulas de quantidade avaliadas).
type StructureConfiguratorUseCase struct {
	Configurator configuratorPort
	Resolver     structureResolver
	Auth         ports.AuthService
	Items        any
}

func NewStructureConfiguratorUseCase(
	configurator configuratorPort,
	resolver structureResolver,
	auth ports.AuthService,
	items any,
) *StructureConfiguratorUseCase {
	return &StructureConfiguratorUseCase{Configurator: configurator, Resolver: resolver, Auth: auth, Items: items}
}

// resolveItem devolve o item da empresa autenticada a partir do código de texto.
func (uc *StructureConfiguratorUseCase) resolveItem(ctx context.Context, code request.TextCode) (*itementity.Item, error) {
	return resolveItem(ctx, uc.Items, code)
}

// Panel devolve tudo o que o botão "Configurador" da estrutura precisa exibir.
func (uc *StructureConfiguratorUseCase) Panel(ctx context.Context, code request.TextCode) (*response.StructureConfiguratorPanelResponse, error) {
	item, err := uc.resolveItem(ctx, code)
	if err != nil {
		return nil, err
	}
	panel, err := uc.Configurator.StructurePanel(ctx, int64(item.Code), item.Name)
	if err != nil {
		return nil, err
	}
	panel.ItemCode = string(item.BusinessCode)
	return panel, nil
}

// Apply valida as respostas contra as restrições, gera a máscara e — quando a
// configuração é gravada — devolve a estrutura resolvida com as fórmulas de
// quantidade já avaliadas.
func (uc *StructureConfiguratorUseCase) Apply(
	ctx context.Context,
	code request.TextCode,
	dto request.ApplyStructureConfigurationDTO,
) (*response.StructureConfiguratorApplyResponse, error) {
	item, err := uc.resolveItem(ctx, code)
	if err != nil {
		return nil, err
	}
	if len(dto.Answers) == 0 {
		return nil, errorsuc.NewValidationError("responda ao menos uma característica para configurar o item")
	}

	// Restrições e dependências (FENG0116) antes de qualquer gravação.
	violations, err := uc.Configurator.ValidateCombination(ctx, int64(item.Code), dto.Answers)
	if err != nil {
		return nil, err
	}
	if len(violations) > 0 {
		return nil, &RestrictionViolationError{Violations: violations}
	}

	actor := uuid.Nil
	if uc.Auth != nil {
		if actor, err = uc.Auth.UserID(ctx); err != nil {
			return nil, err
		}
	}
	generated, err := uc.Configurator.GenerateMask(ctx, request.CfgGenerateMaskDTO{
		ItemCode:  int64(item.Code),
		Answers:   dto.Answers,
		Persist:   dto.Persist,
		CreatedBy: actor,
	})
	if err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}

	out := &response.StructureConfiguratorApplyResponse{
		ItemCode:  string(item.BusinessCode),
		Mask:      generated.Mask,
		MaskHash:  generated.MaskHash,
		Persisted: generated.Persisted,
		MaskID:    generated.MaskID,
		Answers:   generated.Answers,
		Variables: uc.Configurator.FormulaVariables(ctx, dto.Answers),
		Warnings:  []string{},
	}

	// A estrutura só pode ser resolvida para uma configuração gravada: é a
	// máscara persistida que carrega as respostas usadas pelas fórmulas.
	if !generated.Persisted {
		out.Warnings = append(out.Warnings,
			"configuração simulada: grave-a para que a estrutura, o MRP e o custo passem a usá-la")
		return out, nil
	}
	if uc.Resolver == nil {
		return out, nil
	}
	tree, err := uc.Resolver.Execute(ctx, request.ResolveStructureQueryDTO{ItemCode: code, Mask: generated.Mask})
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("configuração gravada, mas a estrutura não pôde ser resolvida agora: %v", err))
		return out, nil
	}
	out.Structure = tree
	return out, nil
}

// RestrictionViolationError transporta as restrições violadas até a camada HTTP,
// que devolve 422 com a lista para a tela destacar as perguntas.
type RestrictionViolationError struct {
	Violations []response.StructureConfiguratorViolation
}

func (e *RestrictionViolationError) Error() string {
	if len(e.Violations) == 0 {
		return "a combinação de respostas viola as restrições cadastradas"
	}
	msg := "a combinação de respostas viola as restrições cadastradas: "
	for i, v := range e.Violations {
		if i > 0 {
			msg += "; "
		}
		msg += v.Message
	}
	return msg
}
