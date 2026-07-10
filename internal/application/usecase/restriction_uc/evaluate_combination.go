package restriction_uc

import (
	"context"
	"strings"

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
		ans, ok := answers[d.QuestionID]
		val := ""
		if d.AnswerValue != nil {
			val = *d.AnswerValue
		}
		switch d.Operator {
		case entity.OperatorInvalid:
			return false // forbidden combination
		case entity.OperatorEqual:
			if !ok || !strings.EqualFold(ans, val) {
				return false
			}
		case entity.OperatorDifferent:
			if ok && strings.EqualFold(ans, val) {
				return false
			}
		case entity.OperatorBelongs:
			if !ok || !inCSV(val, ans) {
				return false
			}
		case entity.OperatorNotBelongs:
			if ok && inCSV(val, ans) {
				return false
			}
		case entity.OperatorGreater:
			if !ok || !(ans > val) {
				return false
			}
		case entity.OperatorLess:
			if !ok || !(ans < val) {
				return false
			}
		}
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
