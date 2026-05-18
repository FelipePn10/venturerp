package generate_mask_uc

import "context"

// RestrictionEvaluator is a port consumed by GenerateMaskForItemUseCase.
// It is satisfied by *restriction_uc.EvaluateRestrictionsUseCase via its
// ApplyRestrictions method, without creating a circular import.
type RestrictionEvaluator interface {
	// ApplyRestrictions evaluates which restrictions are active for the given
	// context and answer set, then returns:
	//   - invalidQuestionIDs: questions that must be removed from the mask
	//   - lockedValues:       question_id → text value that must be forced
	ApplyRestrictions(
		ctx context.Context,
		itemCode, customerCode, divisionID *int64,
		answers map[int64]string,
	) (invalidQuestionIDs []int64, lockedValues map[int64]string, err error)
}
