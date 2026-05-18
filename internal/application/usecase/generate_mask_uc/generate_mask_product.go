package generate_mask_uc

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
)

type GenerateMaskForItemUseCase struct {
	Repo      repository.GenerateMaskForItemRepository
	Auth      ports.AuthService
	Evaluator RestrictionEvaluator // optional — nil skips restriction checks
}

func NewGenerateMaskItemUseCase(
	repo repository.GenerateMaskForItemRepository,
	auth ports.AuthService,
) *GenerateMaskForItemUseCase {
	return &GenerateMaskForItemUseCase{Repo: repo, Auth: auth}
}

func (uc *GenerateMaskForItemUseCase) Execute(
	ctx context.Context,
	dto request.GenerateMaskItemRequestDTO,
) (*entity.ItemMask, error) {
	if uc.Auth == nil {
		return nil, errors.New("auth mask not initialized")
	}
	if uc.Repo == nil {
		return nil, errors.New("repository not initialized")
	}
	if !uc.Auth.CanGenerateMaskForItem(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(dto.Answers) == 0 {
		return nil, errors.New("answers cannot be empty")
	}

	// ── Step 1: resolve each option_id → text value ──────────────────────────
	textAnswers := make(map[int64]string, len(dto.Answers))
	for _, a := range dto.Answers {
		val, err := uc.Repo.GetOptionValue(ctx, a.OptionID)
		if err != nil {
			return nil, fmt.Errorf("resolving option %d: %w", a.OptionID, err)
		}
		textAnswers[a.QuestionID] = val
	}

	// ── Step 2: apply restrictions (if an evaluator is wired) ────────────────
	if uc.Evaluator != nil {
		invalidIDs, lockedVals, err := uc.Evaluator.ApplyRestrictions(
			ctx,
			&dto.ItemCode,
			dto.CustomerCode,
			dto.DivisionID,
			textAnswers,
		)
		if err != nil {
			return nil, fmt.Errorf("evaluating restrictions: %w", err)
		}

		// Build a set for O(1) lookup.
		invalidSet := make(map[int64]struct{}, len(invalidIDs))
		for _, id := range invalidIDs {
			invalidSet[id] = struct{}{}
		}

		// Remove answers for INVALID questions.
		for qID := range invalidSet {
			delete(textAnswers, qID)
		}
		// Remove them from the DTO answers slice too (keep them in sync).
		filtered := dto.Answers[:0]
		for _, a := range dto.Answers {
			if _, bad := invalidSet[a.QuestionID]; !bad {
				filtered = append(filtered, a)
			}
		}
		dto.Answers = filtered

		// Inject locked values — replace or add the answer for locked questions.
		if len(lockedVals) > 0 {
			positions, err := uc.Repo.GetItemQuestionPositions(ctx, dto.ItemCode)
			if err != nil {
				return nil, fmt.Errorf("fetching question positions: %w", err)
			}
			for qID, val := range lockedVals {
				pos, ok := positions[qID]
				if !ok {
					continue // question not associated with item — skip
				}
				optionID, err := uc.Repo.GetOptionIDByQuestionAndValue(ctx, qID, val)
				if err != nil {
					return nil, fmt.Errorf("resolving locked option (question %d, value %q): %w", qID, val, err)
				}
				// Replace existing entry if present, otherwise append.
				replaced := false
				for i, a := range dto.Answers {
					if a.QuestionID == qID {
						dto.Answers[i] = request.MaskAnswerInput{
							QuestionID: qID,
							OptionID:   optionID,
							Position:   pos,
						}
						replaced = true
						break
					}
				}
				if !replaced {
					dto.Answers = append(dto.Answers, request.MaskAnswerInput{
						QuestionID: qID,
						OptionID:   optionID,
						Position:   pos,
					})
				}
				textAnswers[qID] = val
			}
		}
	}

	if len(dto.Answers) == 0 {
		return nil, errors.New("all answers were removed by active restrictions")
	}

	// ── Step 3: build MaskAnswer value objects ────────────────────────────────
	answers := make([]valueobject.MaskAnswer, 0, len(dto.Answers))
	for _, a := range dto.Answers {
		optionValue, err := uc.Repo.GetOptionValue(ctx, a.OptionID)
		if err != nil {
			return nil, err
		}
		answer, err := valueobject.NewMaskAnswer(a.QuestionID, a.OptionID, a.Position, optionValue)
		if err != nil {
			return nil, err
		}
		answers = append(answers, answer)
	}

	// ── Step 4: compose mask ──────────────────────────────────────────────────
	mask, err := valueobject.NewItemMask(dto.ItemCode, answers)
	if err != nil {
		return nil, err
	}

	itemMask, err := entity.NewItemMask(dto.ItemCode, mask.Value(), mask.Hash(), dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	itemMask.Answers = answers

	return uc.Repo.Generate(ctx, itemMask)
}
