package structure_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/formula"
	"github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository"
)

// ConsultStructureUseCase implementa VENG0401 — Consulta de Estrutura de Produto.
type ConsultStructureUseCase struct {
	Repo  repository.StructureQueryRepository
	Items any
}

func NewConsultStructureUseCase(repo repository.StructureQueryRepository, items ...any) *ConsultStructureUseCase {
	uc := &ConsultStructureUseCase{Repo: repo}
	if len(items) > 0 {
		uc.Items = items[0]
	}
	return uc
}

func (uc *ConsultStructureUseCase) Execute(
	ctx context.Context,
	dto request.ConsultStructureDTO,
) (*response.ConsultStructureResponse, error) {

	itemCode, err := resolveItemCode(ctx, uc.Items, dto.ItemCode)
	if err != nil {
		return nil, err
	}

	var rows []response.ConsultStructureRowResponse
	if err := uc.descend(ctx, dto, itemCode, dto.Mask, 1, &rows); err != nil {
		return nil, err
	}

	return &response.ConsultStructureResponse{
		RootItemCode: itemCode,
		Mask:         dto.Mask,
		Rows:         rows,
	}, nil
}

// descend percorre a BOM recursivamente.
//
// parentCode — código do item cujos filhos serão consultados agora
// parentMask — máscara corrente do pai (passada ao SQL para filtro e à lógica de herança)
// level      — nível de profundidade atual (1 = filhos diretos da raiz)
func (uc *ConsultStructureUseCase) descend(
	ctx context.Context,
	dto request.ConsultStructureDTO,
	parentCode int64,
	parentMask string,
	level int,
	out *[]response.ConsultStructureRowResponse,
) error {
	if dto.Levels > 0 && level > dto.Levels {
		return nil
	}

	children, err := uc.Repo.ConsultChildren(ctx, parentCode, parentMask, dto.EffectivenessDate)
	if err != nil {
		return fmt.Errorf("consultando filhos de %d (nível %d): %w", parentCode, level, err)
	}

	for _, child := range children {
		row, childMask, err := uc.buildRow(ctx, child.ItemStructure, child.WarehouseCode, child.TypeStruct, parentMask, level)
		if err != nil {
			return err
		}
		*out = append(*out, row)

		if err := uc.descend(ctx, dto, child.ChildCode, childMask, level+1, out); err != nil {
			return err
		}
	}
	return nil
}

// buildRow constrói uma linha da grade resolvendo a máscara do filho e
// avaliando a fórmula de perda quando possível.
func (uc *ConsultStructureUseCase) buildRow(
	ctx context.Context,
	s *entity.ItemStructure,
	warehouseCode int64,
	structureType int16,
	parentMask string,
	level int,
) (row response.ConsultStructureRowResponse, childMask string, err error) {

	// Resolve a máscara efetiva do filho para a próxima descida.
	// Se o componente herda a configuração do pai, propaga a máscara do pai.
	// Caso contrário, busca a última máscara registrada para o item filho.
	if s.Inherit && parentMask != "" {
		childMask = parentMask
	} else {
		childMask, err = uc.Repo.GetLatestMaskForItem(ctx, s.ChildCode)
		if err != nil {
			return row, "", fmt.Errorf("buscando máscara do item %d: %w", s.ChildCode, err)
		}
	}

	// As variáveis da configuração (as "perguntas" respondidas) alimentam tanto a
	// fórmula de quantidade quanto a de perda.
	vars := map[string]float64{}
	if (s.HasQuantityFormula() || (s.LossFormula != nil && *s.LossFormula != "")) && childMask != "" {
		vars, _ = uc.Repo.GetMaskAnswersWithNames(ctx, s.ChildCode, childMask)
		if len(vars) == 0 && parentMask != "" {
			// Fórmula escrita com as variáveis do pai (caso mais comum em
			// FENG0210: COMPRIMENTO/PROFUNDIDADE são perguntas do produto pai).
			vars, _ = uc.Repo.GetMaskAnswersWithNames(ctx, s.ParentCode, parentMask)
		}
	}

	quantity, formulaApplied := s.ResolvedQuantity(vars)
	corrected := quantity * (1 + s.LossPercentage/100.0)

	if s.LossFormula != nil && *s.LossFormula != "" {
		if result, ok := formula.EvaluateSafe(*s.LossFormula, vars); ok {
			corrected = result
		}
	}

	var maskPtr *string
	if childMask != "" {
		m := childMask
		maskPtr = &m
	}

	row = response.ConsultStructureRowResponse{
		Level:             level,
		ParentCode:        s.ParentCode,
		ItemCode:          s.ChildCode,
		Description:       s.ChildDescription,
		Sequence:          s.Sequence,
		StartDate:         s.StartDate,
		EndDate:           s.EndDate,
		Quantity:          quantity,
		NominalQuantity:   s.Quantity,
		QuantityFormula:   s.QuantityFormula,
		FormulaApplied:    formulaApplied,
		WarehouseCode:     warehouseCode,
		LossFormula:       s.LossFormula,
		LossPercentage:    s.LossPercentage,
		CorrectedQuantity: corrected,
		StructureType:     structureType,
		Mask:              maskPtr,
	}
	return row, childMask, nil
}
