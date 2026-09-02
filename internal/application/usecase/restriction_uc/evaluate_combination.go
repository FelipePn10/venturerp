package restriction_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
)

// EvaluateCombination is the generation-oriented counterpart of Execute: instead
// of cleaning the frontend's in-progress answers, it answers a single yes/no —
// "is this fully-formed combination of answers valid?" — for the cartesian mask
// generator.
//
// Semantics: the highest-weight applicable restriction whose dominants fire acts
// as a dependency guard; the combination is valid iff that restriction's
// determinants are all satisfied by the combination. `INVALID` determinants mark
// the fired dominant pattern as forbidden (combination discarded). `answers`
// maps characteristic_id → the chosen variable's canonical value (its code).
func (uc *EvaluateRestrictionsUseCase) EvaluateCombination(
	ctx context.Context,
	itemCode, customerCode, divisionID *int64,
	answers map[int64]string,
) (bool, error) {
	dets, _, err := uc.findApplied(ctx, customerCode, itemCode, nil /* classificationType */, divisionID, answers)
	if err != nil {
		return false, err
	}
	if dets == nil {
		return true, nil // no restriction fired → combination is valid
	}
	return determinantsSatisfied(dets, answers), nil
}

// determinantsSatisfied reports whether every determinant of a fired restriction
// holds for the combination. A single violated determinant invalidates it.
func determinantsSatisfied(dets []*entity.RestrictionDeterminant, answers map[int64]string) bool {
	for _, d := range dets {
		if !determinantSatisfied(d, answers) {
			return false
		}
	}
	return true
}

// determinantSatisfied avalia um único determinante contra a combinação.
func determinantSatisfied(d *entity.RestrictionDeterminant, answers map[int64]string) bool {
	ans, ok := answers[d.QuestionID]
	val := ""
	if d.AnswerValue != nil {
		val = *d.AnswerValue
	}
	switch d.Operator {
	case entity.OperatorInvalid:
		return false // forbidden combination
	case entity.OperatorEqual:
		return ok && strings.EqualFold(ans, val)
	case entity.OperatorDifferent:
		return !ok || !strings.EqualFold(ans, val)
	case entity.OperatorBelongs:
		return ok && inCSV(val, ans)
	case entity.OperatorNotBelongs:
		return !ok || !inCSV(val, ans)
	case entity.OperatorGreater:
		return ok && ans > val
	case entity.OperatorLess:
		return ok && ans < val
	}
	return true
}

func inCSV(csv, needle string) bool {
	for _, v := range strings.Split(csv, ",") {
		if strings.EqualFold(strings.TrimSpace(v), needle) {
			return true
		}
	}
	return false
}

// ExplainCombination avalia a combinação e, quando ela é inválida, devolve as
// restrições violadas em PT-BR — é o que o configurador embutido na Estrutura de
// Produto mostra ao usuário no lugar de um simples "combinação inválida".
// Combinação válida devolve (true, nil, nil).
func (uc *EvaluateRestrictionsUseCase) ExplainCombination(
	ctx context.Context,
	itemCode, customerCode, divisionID *int64,
	answers map[int64]string,
) (bool, []response.StructureConfiguratorViolation, error) {
	dets, restrictionCode, err := uc.findApplied(ctx, customerCode, itemCode, nil /* classificationType */, divisionID, answers)
	if err != nil {
		return false, nil, err
	}
	if dets == nil {
		return true, nil, nil
	}
	violations := make([]response.StructureConfiguratorViolation, 0)
	for _, d := range dets {
		if determinantSatisfied(d, answers) {
			continue
		}
		expected := ""
		if d.AnswerValue != nil {
			expected = *d.AnswerValue
		}
		violations = append(violations, response.StructureConfiguratorViolation{
			RestrictionCode:  restrictionCode,
			CharacteristicID: d.QuestionID,
			Operator:         string(d.Operator),
			ExpectedValue:    expected,
			AnsweredValue:    answers[d.QuestionID],
			Message:          violationMessage(d.Operator, expected, answers[d.QuestionID]),
		})
	}
	return len(violations) == 0, violations, nil
}

// violationMessage traduz o operador da restrição para uma frase de tela.
func violationMessage(op entity.RestrictionOperator, expected, answered string) string {
	if answered == "" {
		answered = "sem resposta"
	}
	switch op {
	case entity.OperatorInvalid:
		return "esta combinação de respostas não é permitida"
	case entity.OperatorEqual:
		return "a resposta precisa ser " + expected + " (respondido: " + answered + ")"
	case entity.OperatorDifferent:
		return "a resposta não pode ser " + expected
	case entity.OperatorBelongs:
		return "a resposta precisa estar entre " + expected + " (respondido: " + answered + ")"
	case entity.OperatorNotBelongs:
		return "a resposta não pode estar entre " + expected
	case entity.OperatorGreater:
		return "a resposta precisa ser maior que " + expected + " (respondido: " + answered + ")"
	case entity.OperatorLess:
		return "a resposta precisa ser menor que " + expected + " (respondido: " + answered + ")"
	default:
		return "a resposta não atende à restrição cadastrada"
	}
}
