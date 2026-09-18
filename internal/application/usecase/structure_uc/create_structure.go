package structure_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

// CreateStructureComponentUseCase adiciona um componente (filho) à estrutura
type CreateStructureComponentUseCase struct {
	Repo repository.ItemStructureRepository
	Auth ports.AuthService
	// Items resolve o código público do item; UOM converte a unidade da
	// estrutura para a de estoque quando as duas diferem.
	Items any
	UOM   ports.UOMConverter
}

func NewCreateStructureComponentUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
	items ...any,
) *CreateStructureComponentUseCase {
	uc := &CreateStructureComponentUseCase{
		Repo: repo,
		Auth: auth,
	}
	// As dependências chegam soltas: a de item e a de conversão são
	// reconhecidas pelo tipo, o que evita quebrar as chamadas existentes.
	for _, dep := range items {
		if conv, ok := dep.(ports.UOMConverter); ok {
			uc.UOM = conv
			continue
		}
		if uc.Items == nil {
			uc.Items = dep
		}
	}
	return uc
}

func (uc *CreateStructureComponentUseCase) Execute(
	ctx context.Context,
	dto request.CreateStructureComponentDTO,
) (*entity.ItemStructure, error) {
	if !uc.Auth.CanCreateStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Sequence < 1 {
		return nil, errorsuc.NewValidationError("a posição do componente é obrigatória e deve ser positiva")
	}
	parentCode, err := resolveItemCode(ctx, uc.Items, dto.ParentCode)
	if err != nil {
		return nil, err
	}
	childItem, err := itemresolution.Resolve(ctx, uc.Items, dto.ChildCode)
	if err != nil {
		return nil, err
	}
	childCode := int64(childItem.Code)
	unidadeDeEstoque := childItem.Warehouse.UnitOfMeasurement
	if dto.UnitOfMeasurement == "" {
		dto.UnitOfMeasurement = unidadeDeEstoque
	}
	// A unidade da estrutura pode ser diferente da de estoque — desenhar em m²
	// uma chapa estocada em kg é legítimo. O que não pode é a diferença passar
	// sem conversão: daí em diante todo o sistema leria o número na unidade
	// errada.
	qtdeEmEstoque, fator, err := resolverUnidadeDeEstoque(
		ctx, uc.UOM, childCode, string(dto.ChildCode),
		dto.Quantity, string(dto.UnitOfMeasurement), string(unidadeDeEstoque))
	if err != nil {
		return nil, err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	parentExists, err := uc.Repo.ItemExists(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("checking parent item: %w", err)
	}
	if !parentExists {
		return nil, errorsuc.NewNotFoundError("item pai não encontrado")
	}

	childExists, err := uc.Repo.ItemExists(ctx, childCode)
	if err != nil {
		return nil, fmt.Errorf("checking child item: %w", err)
	}
	if !childExists {
		return nil, errorsuc.NewNotFoundError("item filho não encontrado")
	}

	// Situação vazia chegava ao banco e estourava no enum, com o erro cru do
	// Postgres na tela ("invalid input value for enum health_enum").
	if strings.TrimSpace(string(dto.Health)) == "" {
		dto.Health = "ATIVO"
	}

	if parentCode == childCode {
		return nil, errorsuc.NewValidationError("um item não pode ser componente de si mesmo")
	}

	// Ao adicionar pai → filho, só há ciclo se o filho já alcançar o pai.
	hasCycle, err := uc.Repo.HasCyclicReference(ctx, childCode, parentCode)
	if err != nil {
		return nil, err
	}
	if hasCycle {
		// Dizer só "criaria um ciclo" deixa o usuário sem saída: ele não sabe
		// QUAL relação existente está no caminho, e é comum ela estar invertida
		// por engano. Nomeamos os dois itens para que dê para ir corrigir lá.
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"não é possível incluir %s dentro de %s: o item %s já contém %s na sua estrutura "+
				"(direta ou indiretamente). Verifique se a estrutura existente não está invertida.",
			dto.ChildCode, dto.ParentCode, dto.ChildCode, dto.ParentCode))
	}

	exists, err := uc.Repo.SequenceExists(ctx, parentCode, dto.Sequence)
	if err != nil {
		return nil, err
	}
	if exists {
		// Sem tipo isto virava 500: a mensagem já estava em português, mas o
		// classificador trabalha por tipo e o ramo final é "erro interno".
		return nil, errorsuc.NewConflictError(fmt.Sprintf(
			"a posição %d já está em uso na estrutura deste item — escolha outra sequência", dto.Sequence))
	}

	structure, err := entity.NewItemStructure(
		parentCode,
		childCode,
		dto.ParentMask,
		dto.Quantity,
		dto.UnitOfMeasurement,
		dto.Health,
		dto.LossPercentage,
		dto.Sequence,
		dto.Notes,
		dto.IsActive,
		dto.Inherit,
		dto.StartDate,
		dto.EndDate,
		dto.LossFormula,
		actor,
		dto.QuantityFormula,
		dto.QuantityRounding,
		dto.QuantityScale,
	)
	if err != nil {
		return nil, err
	}
	structure.IsCoproduct = dto.IsCoproduct
	structure.IsFixedQty = dto.IsFixedQty
	structure.SetSubstitute(dto.SubstituteGroup, dto.SubstitutePriority)
	structure.WarehouseCode = dto.WarehouseCode
	structure.LineWarehouseCode = dto.LineWarehouseCode
	structure.SetupLoss = dto.SetupLoss
	structure.CostLossType = entity.NormalizeCostLossType(dto.CostLossType)
	structure.CostLoss = dto.CostLoss
	structure.CostCenterCode = dto.CostCenterCode
	structure.IsCriticalMPS = dto.IsCriticalMPS
	structure.GeneratesInspection = dto.GeneratesInspection
	structure.QuantityStockUOM = qtdeEmEstoque
	structure.ConversionFactor = fator

	return uc.Repo.Create(ctx, structure)
}
