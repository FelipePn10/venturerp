package restriction_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
)

// EvaluationResult is the processed outcome of evaluating restrictions against
// a set of answers. Fields are ready for direct consumption by the frontend or
// by the GenerateMask use case.
type EvaluationResult struct {
	// RestrictionCode is the code of the restriction that fired (0 if none).
	RestrictionCode int64 `json:"restriction_code"`

	// InvalidQuestionIDs lists questions that must be hidden and whose answers
	// must be discarded (operator INVALID on the determinant).
	InvalidQuestionIDs []int64 `json:"invalid_question_ids"`

	// LockedValues maps question_id → forced text value (operator EQUAL on the
	// determinant). The answer for that question must be replaced with this value.
	LockedValues map[int64]string `json:"locked_values"`

	// CleanedAnswers is the input Answers map after applying all determinants:
	// invalid question entries removed, locked values substituted. The frontend
	// can replace its in-progress answer state with this field directly.
	CleanedAnswers map[int64]string `json:"cleaned_answers"`
}


type EvaluateRestrictionsUseCase struct {
	Repo repository.RestrictionRepository
}

// Execute is the full public endpoint handler: evaluates restrictions and returns
// the enriched EvaluationResult ready for the frontend.
func (uc *EvaluateRestrictionsUseCase) Execute(
	ctx context.Context,
	dto request.EvaluateRestrictionDTO,
) (*EvaluationResult, error) {
	applied, restrictionCode, err := uc.findApplied(ctx, dto.CustomerCode, dto.ItemCode, dto.ClassificationType, dto.DivisionID, dto.Answers)
	if err != nil {
		return nil, err
	}
	if applied == nil {
		// No restriction fired — return cleaned answers identical to input.
		cleaned := make(map[int64]string, len(dto.Answers))
		for k, v := range dto.Answers {
			cleaned[k] = v
		}
		return &EvaluationResult{
			InvalidQuestionIDs: []int64{},
			LockedValues:       map[int64]string{},
			CleanedAnswers:     cleaned,
		}, nil
	}
	return buildResult(restrictionCode, applied, dto.Answers), nil
}

// ApplyRestrictions satisfies generate_mask_uc.RestrictionEvaluator.
// It evaluates restrictions for the given context and returns the two fields the
// mask pipeline needs: which question IDs to drop and which values to lock.
func (uc *EvaluateRestrictionsUseCase) ApplyRestrictions(
	ctx context.Context,
	itemCode, customerCode, divisionID *int64,
	answers map[int64]string,
) ([]int64, map[int64]string, error) {
	applied, _, err := uc.findApplied(ctx, customerCode, itemCode, nil /* classificationType */, divisionID, answers)
	if err != nil {
		return nil, nil, err
	}
	if applied == nil {
		return []int64{}, map[int64]string{}, nil
	}
	invalidIDs, lockedVals := splitDeterminants(applied)
	return invalidIDs, lockedVals, nil
}

// findApplied fetches all active restrictions, filters by context, picks the
// highest-weight one whose dominants evaluate to true, and returns its
// determinants alongside the restriction code.
func (uc *EvaluateRestrictionsUseCase) findApplied(
	ctx context.Context,
	customerCode, itemCode *int64,
	classificationType *string,
	divisionID *int64,
	answers map[int64]string,
) ([]*entity.RestrictionDeterminant, int64, error) {
	// Build a minimal EvaluateRestrictionDTO so contextMatches can be reused.
	ctxDTO := request.EvaluateRestrictionDTO{
		CustomerCode:       customerCode,
		ItemCode:           itemCode,
		ClassificationType: classificationType,
		DivisionID:         divisionID,
		Answers:            answers,
	}

	candidates, err := uc.Repo.ListActive(ctx)
	if err != nil {
		return nil, 0, err
	}

	// candidates are already sorted weight DESC by the DB query.
	for _, r := range candidates {
		if !contextMatches(r, ctxDTO) {
			continue
		}
		dominants, err := uc.Repo.GetDominants(ctx, r.ID)
		if err != nil {
			return nil, 0, err
		}
		if len(dominants) == 0 || evaluateDominants(dominants, answers) {
			dets, err := uc.Repo.GetDeterminants(ctx, r.ID)
			if err != nil {
				return nil, 0, err
			}
			return dets, r.Code, nil
		}
	}
	return nil, 0, nil
}

// buildResult converts raw determinants + original answers into the enriched
// EvaluationResult.
func buildResult(code int64, dets []*entity.RestrictionDeterminant, originalAnswers map[int64]string) *EvaluationResult {
	invalidIDs, lockedVals := splitDeterminants(dets)

	// Build a set for O(1) lookup.
	invalidSet := make(map[int64]struct{}, len(invalidIDs))
	for _, id := range invalidIDs {
		invalidSet[id] = struct{}{}
	}

	// Clean the answers: remove invalid ones, apply locked values.
	cleaned := make(map[int64]string, len(originalAnswers))
	for qID, val := range originalAnswers {
		if _, isInvalid := invalidSet[qID]; !isInvalid {
			cleaned[qID] = val
		}
	}
	for qID, val := range lockedVals {
		cleaned[qID] = val // override with the locked value
	}

	return &EvaluationResult{
		RestrictionCode:    code,
		InvalidQuestionIDs: invalidIDs,
		LockedValues:       lockedVals,
		CleanedAnswers:     cleaned,
	}
}

// splitDeterminants separates a list of determinants into INVALID question IDs
// and EQUAL locked values.
func splitDeterminants(dets []*entity.RestrictionDeterminant) (invalidIDs []int64, lockedVals map[int64]string) {
	lockedVals = make(map[int64]string)
	for _, d := range dets {
		switch d.Operator {
		case entity.OperatorInvalid:
			invalidIDs = append(invalidIDs, d.QuestionID)
		case entity.OperatorEqual:
			if d.AnswerValue != nil {
				lockedVals[d.QuestionID] = *d.AnswerValue
			}
		}
	}
	return invalidIDs, lockedVals
}

// ─── context & dominants evaluation ─────────────────────────────────────────

func contextMatches(r *entity.Restriction, dto request.EvaluateRestrictionDTO) bool {
	// Global restriction (no scope) always applies.
	if r.CustomerCode == nil && r.ItemCode == nil && r.ClassificationType == nil && r.DivisionID == nil {
		return true
	}
	if r.CustomerCode != nil && dto.CustomerCode != nil && *r.CustomerCode == *dto.CustomerCode {
		if r.ItemCode == nil || (dto.ItemCode != nil && *r.ItemCode == *dto.ItemCode) {
			return true
		}
	}
	if r.ItemCode != nil && dto.ItemCode != nil && *r.ItemCode == *dto.ItemCode && r.CustomerCode == nil {
		return true
	}
	if r.ClassificationType != nil && dto.ClassificationType != nil &&
		*r.ClassificationType == *dto.ClassificationType {
		return true
	}
	if r.DivisionID != nil && dto.DivisionID != nil && *r.DivisionID == *dto.DivisionID {
		return true
	}
	return false
}

// evaluateDominants implements the E/OU grouping logic per spec:
//
//	E  A=1  → group 1
//	E  B=2  → group 1 (continues)
//	OU C=3  → group 2 starts
//	E  D=4  → group 2 (continues)
//
// → (A=1 AND B=2) OR (C=3 AND D=4)
func evaluateDominants(dominants []*entity.RestrictionDominant, answers map[int64]string) bool {
	if len(dominants) == 0 {
		return true
	}

	type group struct{ conditions []*entity.RestrictionDominant }
	var groups []group
	var current group

	for i, d := range dominants {
		if i == 0 || d.ConditionType == entity.ConditionAnd {
			current.conditions = append(current.conditions, d)
		} else {
			groups = append(groups, current)
			current = group{conditions: []*entity.RestrictionDominant{d}}
		}
	}
	groups = append(groups, current)

	for _, g := range groups {
		allTrue := true
		for _, cond := range g.conditions {
			if !evalCondition(cond, answers) {
				allTrue = false
				break
			}
		}
		if allTrue {
			return true
		}
	}
	return false
}

func evalCondition(d *entity.RestrictionDominant, answers map[int64]string) bool {
	answer, ok := answers[d.QuestionID]
	if !ok {
		return false
	}
	switch d.Operator {
	case entity.OperatorEqual:
		return strings.EqualFold(answer, d.AnswerValue)
	case entity.OperatorDifferent:
		return !strings.EqualFold(answer, d.AnswerValue)
	case entity.OperatorGreater:
		return answer > d.AnswerValue
	case entity.OperatorLess:
		return answer < d.AnswerValue
	case entity.OperatorBelongs:
		for _, v := range strings.Split(d.AnswerValue, ",") {
			if strings.EqualFold(strings.TrimSpace(v), answer) {
				return true
			}
		}
	case entity.OperatorNotBelongs:
		for _, v := range strings.Split(d.AnswerValue, ",") {
			if strings.EqualFold(strings.TrimSpace(v), answer) {
				return false
			}
		}
		return true
	}
	return false
}
