package structure_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// VerificarConfiguracao aponta, no cadastro da estrutura, os componentes cuja
// configuração não fecha — antes de a ordem de produção ou o MRP tropeçarem
// nisso.
//
// Uma característica do filho precisa ter origem. Os ERPs de configuração
// resolvem isso de três formas, e o sistema aceita as três:
//
//  1. o pai responde a mesma característica (herança direta);
//  2. uma regra de equivalência leva a resposta do pai para a do filho — é o
//     procedure do SAP, a equivalência do Focco;
//  3. a característica tem resposta padrão no item — o opcional padrão do
//     Protheus.
//
// O que não se encaixa em nenhuma das três deixa a configuração incompleta
// (o "incomplete configuration" do Oracle). Antes disso o sistema caía em
// silêncio na estrutura genérica do filho.
func (uc *ResolveStructureQueryUseCase) VerificarConfiguracao(
	ctx context.Context,
	code request.TextCode,
	mask string,
) ([]response.StructureConfigurationCheckResponse, error) {
	if !uc.Auth.CanResolveStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	itemCode, err := resolveItemCode(ctx, uc.Items, code)
	if err != nil {
		return nil, err
	}

	componentes, err := uc.Repo.GetDirectChildrenForMask(ctx, itemCode, mask)
	if err != nil {
		return nil, fmt.Errorf("buscando componentes: %w", err)
	}
	perguntasDoPai, err := uc.Repo.GetItemQuestions(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("buscando características do item: %w", err)
	}
	noPai := make(map[int64]bool, len(perguntasDoPai))
	for _, q := range perguntasDoPai {
		noPai[q.QuestionID] = true
	}
	regras, err := uc.Repo.ListEquivalentRules(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("buscando regras de equivalência: %w", err)
	}

	saida := make([]response.StructureConfigurationCheckResponse, 0, len(componentes))
	for _, comp := range componentes {
		perguntas, err := uc.Repo.GetItemQuestions(ctx, comp.ChildCode)
		if err != nil {
			return nil, fmt.Errorf("buscando características do componente %d: %w", comp.ChildCode, err)
		}
		item := response.StructureConfigurationCheckResponse{
			ChildCode:        comp.ChildCode,
			ChildDescription: comp.ChildDescription,
			Inherits:         comp.Inherit,
			Configured:       len(perguntas) > 0,
		}
		// Configurado sem herdar: a subestrutura não é explodida sozinha.
		item.RequiresMask = len(perguntas) > 0 && !comp.Inherit
		if len(perguntas) == 0 || !comp.Inherit {
			saida = append(saida, item)
			continue
		}

		caracteristicas, err := uc.Repo.ListItemCharacteristics(ctx, comp.ChildCode)
		if err != nil {
			return nil, fmt.Errorf("buscando características do componente %d: %w", comp.ChildCode, err)
		}
		temPadrao := make(map[int64]bool, len(caracteristicas))
		codigo := make(map[int64]string, len(caracteristicas))
		for _, c := range caracteristicas {
			temPadrao[c.CharacteristicID] = c.DefaultVariableID != nil
			codigo[c.CharacteristicID] = c.Code
		}
		porEquivalencia := make(map[int64]bool)
		for _, regra := range regras {
			if regra.ChildItemCode == comp.ChildCode && regra.ChildVariableID != nil {
				porEquivalencia[regra.ChildCharacteristicID] = true
			}
		}

		for _, q := range perguntas {
			if noPai[q.QuestionID] || porEquivalencia[q.QuestionID] || temPadrao[q.QuestionID] {
				continue
			}
			nome := codigo[q.QuestionID]
			if nome == "" {
				nome = fmt.Sprintf("característica %d", q.QuestionID)
			}
			item.MissingCharacteristics = append(item.MissingCharacteristics, nome)
		}
		saida = append(saida, item)
	}
	return saida, nil
}
