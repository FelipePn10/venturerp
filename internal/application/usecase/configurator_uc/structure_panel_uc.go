package configurator_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/configurator/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/formula"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

// typeLabels traduz o tipo da pergunta para o rótulo que a tela exibe.
var typeLabels = map[string]string{
	entity.TypeEscolha:     "Escolha única",
	entity.TypeEscolhaMult: "Escolha múltipla",
	entity.TypeFormula:     "Calculada por fórmula",
	entity.TypeDesenho:     "Desenho",
	entity.TypeInfCaracter: "Texto livre",
	entity.TypeInfNumerica: "Número",
	entity.TypeOpcao:       "Sim/Não",
	entity.TypeCampo:       "Campo do pedido",
	entity.TypeSequencial:  "Sequencial",
}

// CombinationExplainer devolve, além do sim/não, as restrições violadas — é o
// que permite ao painel dizer ao usuário qual dependência barrou a combinação.
type CombinationExplainer interface {
	ExplainCombination(ctx context.Context, itemCode, customerCode, divisionID *int64, answers map[int64]string) (bool, []response.StructureConfiguratorViolation, error)
}

// StructurePanel monta a carga do botão "Configurador" da Estrutura de Produto:
// as perguntas do item com suas respostas possíveis, as configurações já
// geradas e os componentes cuja quantidade sai de fórmula.
func (uc *ConfiguratorUseCase) StructurePanel(ctx context.Context, itemCode int64, itemName string) (*response.StructureConfiguratorPanelResponse, error) {
	if itemCode <= 0 {
		return nil, errorsuc.NewValidationError("informe o item da estrutura")
	}
	out := &response.StructureConfiguratorPanelResponse{
		ItemName:            itemName,
		RestrictionsEnabled: uc.Restrictions != nil,
		Questions:           []response.StructureConfiguratorQuestion{},
		Masks:               []response.StructureConfiguratorMask{},
		Formulas:            []response.StructureConfiguratorFormula{},
	}

	// Componentes com fórmula de quantidade: o painel mostra o que a
	// configuração alimenta e quais variáveis precisam de resposta.
	formulaRows, err := uc.Q.ListStructureQuantityFormulas(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("carregando fórmulas da estrutura: %w", err)
	}
	usedVars := map[string]bool{}
	for _, row := range formulaRows {
		vars := formula.Variables(row.Formula)
		for _, v := range vars {
			usedVars[v] = true
		}
		out.Formulas = append(out.Formulas, response.StructureConfiguratorFormula{
			ChildCode:        row.ChildCode,
			ChildDescription: row.ChildDescription,
			Formula:          row.Formula,
			Rounding:         row.Rounding,
			Scale:            row.Scale,
			NominalQuantity:  row.NominalQuantity,
			UnitOfMeasure:    row.UnitOfMeasure,
			Variables:        vars,
		})
	}

	itemChars, err := uc.Q.ListCfgItemCharacteristics(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("carregando características do item: %w", err)
	}
	answeredVars := map[string]bool{}
	for _, ic := range itemChars {
		question, qErr := uc.panelQuestion(ctx, ic, usedVars)
		if qErr != nil {
			return nil, qErr
		}
		answeredVars[normalizeCode(ic.CharCode)] = true
		out.Questions = append(out.Questions, question)
	}
	sort.SliceStable(out.Questions, func(i, j int) bool { return out.Questions[i].Sequence < out.Questions[j].Sequence })

	// Uma fórmula que referencia variável sem pergunta nunca será avaliável.
	missing := make([]string, 0)
	for v := range usedVars {
		if !answeredVars[v] {
			missing = append(missing, v)
		}
	}
	sort.Strings(missing)
	out.MissingFormulaVariables = missing

	masks, err := uc.Q.ListCfgItemMasks(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("carregando configurações do item: %w", err)
	}
	for _, m := range masks {
		out.Masks = append(out.Masks, response.StructureConfiguratorMask{
			ID: m.ID, Mask: m.Mask, Hash: m.MaskHash, Answered: m.Answered,
		})
	}

	out.Configurable = len(out.Questions) > 0
	if !out.Configurable {
		out.Message = "este item ainda não tem características de configuração cadastradas"
	}
	return out, nil
}

// panelQuestion monta uma pergunta do painel com suas respostas possíveis.
func (uc *ConfiguratorUseCase) panelQuestion(ctx context.Context, ic sqlc.DBCfgItemCharacteristic, usedVars map[string]bool) (response.StructureConfiguratorQuestion, error) {
	char, err := uc.Q.GetCfgCharacteristic(ctx, ic.CharacteristicID)
	if err != nil {
		return response.StructureConfiguratorQuestion{}, errorsuc.NewNotFoundError(
			fmt.Sprintf("característica %d não encontrada", ic.CharacteristicID))
	}
	q := response.StructureConfiguratorQuestion{
		CharacteristicID: ic.CharacteristicID,
		ItemCharID:       ic.ID,
		Sequence:         int(ic.Sequence),
		Code:             char.Code,
		Description:      char.Description,
		Type:             char.CharType,
		TypeLabel:        typeLabels[char.CharType],
		Required:         char.IsRequired,
		AllowsMultiple:   char.CharType == entity.TypeEscolhaMult,
		Mask:             pgutil.FromPgText(char.Mask),
		NumMin:           numPtr(char.NumMin),
		NumMax:           numPtr(char.NumMax),
		NumMultiple:      numPtr(char.NumMultiple),
		OptionTrue:       pgutil.FromPgText(char.OptionTrue),
		OptionFalse:      pgutil.FromPgText(char.OptionFalse),
		UsedByFormula:    usedVars[normalizeCode(char.Code)],
		Options:          []response.StructureConfiguratorOption{},
	}
	// A fórmula da característica pode vir do vínculo item↔característica.
	if f := pgutil.FromPgText(ic.Formula); f != "" {
		q.Formula = f
	} else {
		q.Formula = pgutil.FromPgText(char.Formula)
	}

	// Só perguntas de escolha têm variáveis para listar.
	if char.CharType != entity.TypeEscolha && char.CharType != entity.TypeEscolhaMult {
		return q, nil
	}
	if !char.SetID.Valid {
		return q, nil
	}
	variables, err := uc.Q.ListCfgVariablesBySet(ctx, char.SetID.Int64, true)
	if err != nil {
		return response.StructureConfiguratorQuestion{}, fmt.Errorf("carregando variáveis da característica %s: %w", char.Code, err)
	}
	defaults := map[int64]bool{}
	if ic.DefaultVariableID.Valid {
		defaults[ic.DefaultVariableID.Int64] = true
	} else if char.DefaultVariableID.Valid {
		defaults[char.DefaultVariableID.Int64] = true
	}
	if char.CharType == entity.TypeEscolhaMult {
		if ids, listErr := uc.Q.ListCfgItemCharDefaultAnswers(ctx, ic.ID); listErr == nil {
			for _, id := range ids {
				defaults[id] = true
			}
		}
	}
	for _, v := range variables {
		q.Options = append(q.Options, response.StructureConfiguratorOption{
			VariableID:      v.ID,
			Code:            v.Code,
			MaskComposition: v.MaskComposition,
			Description:     v.Description,
			IsDefault:       defaults[v.ID],
		})
	}
	return q, nil
}

// ValidateCombination roda o motor de restrições/dependências (FENG0116) sobre
// as respostas e devolve as violações em PT-BR. Sem motor configurado, aceita.
func (uc *ConfiguratorUseCase) ValidateCombination(ctx context.Context, itemCode int64, answers []request.CfgMaskAnswerInput) ([]response.StructureConfiguratorViolation, error) {
	explainer, ok := uc.Restrictions.(CombinationExplainer)
	if !ok || uc.Restrictions == nil {
		return nil, nil
	}
	codes := uc.resolveAnswerCodes(ctx, answers)
	valid, violations, err := explainer.ExplainCombination(ctx, &itemCode, nil, nil, codes)
	if err != nil {
		return nil, err
	}
	if valid {
		return nil, nil
	}
	// Enriquece cada violação com o nome da pergunta para a mensagem de tela.
	for i := range violations {
		if char, charErr := uc.Q.GetCfgCharacteristic(ctx, violations[i].CharacteristicID); charErr == nil {
			violations[i].Question = char.Description
			violations[i].Message = char.Description + ": " + violations[i].Message
		}
	}
	return violations, nil
}

// FormulaVariables devolve variável → valor numérico das respostas, no formato
// que as fórmulas de quantidade da estrutura consomem.
func (uc *ConfiguratorUseCase) FormulaVariables(ctx context.Context, answers []request.CfgMaskAnswerInput) map[string]float64 {
	return uc.formulaVars(ctx, answers)
}

// violationSummary resume as violações em uma frase única para o erro 422.
func violationSummary(violations []response.StructureConfiguratorViolation) string {
	parts := make([]string, 0, len(violations))
	for _, v := range violations {
		parts = append(parts, v.Message)
	}
	return "a combinação de respostas viola as restrições cadastradas: " + strings.Join(parts, "; ")
}
