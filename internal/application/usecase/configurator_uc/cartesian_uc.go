package configurator_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/configurator/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
)

// maxCombinations caps the cartesian explosion; the caller must restrict enough
// characteristics to stay under it.
const maxCombinations = 20000

// cartOption is one candidate answer of an ESCOLHA characteristic in the product.
type cartOption struct {
	characteristicID int64
	position         int
	variableID       int64
	code             string // canonical answer for the restriction engine
	maskComposition  string // value placed in the mask
}

// GenerateMasks produces every valid combination of the item's ESCOLHA
// characteristics (Geração de Máscara para Itens Configurados). Fixed selections
// (Restrict) narrow each characteristic to a subset of variables; restrictions/
// dependencies drop invalid combinations; valid masks are optionally persisted.
func (uc *ConfiguratorUseCase) GenerateMasks(ctx context.Context, dto request.CfgGenerateMasksDTO) (*response.CfgGeneratedMasksResponse, error) {
	if dto.ItemCode <= 0 {
		return nil, fmt.Errorf("item_code é obrigatório")
	}
	if len(dto.Restrict) == 0 {
		return nil, fmt.Errorf("é obrigatório restringir ao menos uma característica para reduzir o volume")
	}
	fixed := map[int64][]int64{}
	for _, r := range dto.Restrict {
		fixed[r.CharacteristicID] = r.VariableIDs
	}

	itemChars, err := uc.Q.ListCfgItemCharacteristics(ctx, dto.ItemCode)
	if err != nil {
		return nil, fmt.Errorf("carregando características do item: %w", err)
	}

	// Build the per-characteristic option lists (only ESCOLHA participates).
	dims := make([][]cartOption, 0, len(itemChars))
	for _, ic := range itemChars {
		if ic.CharType != entity.TypeEscolha {
			continue
		}
		char, err := uc.Q.GetCfgCharacteristic(ctx, ic.CharacteristicID)
		if err != nil || !char.SetID.Valid {
			continue
		}
		vars, err := uc.Q.ListCfgVariablesBySet(ctx, char.SetID.Int64, true)
		if err != nil {
			return nil, fmt.Errorf("carregando variáveis do conjunto: %w", err)
		}
		allow := fixed[ic.CharacteristicID]
		opts := make([]cartOption, 0, len(vars))
		for _, v := range vars {
			if len(allow) > 0 && !containsInt64(allow, v.ID) {
				continue
			}
			opts = append(opts, cartOption{
				characteristicID: ic.CharacteristicID,
				position:         int(ic.Sequence),
				variableID:       v.ID,
				code:             v.Code,
				maskComposition:  v.MaskComposition,
			})
		}
		if len(opts) == 0 {
			return nil, fmt.Errorf("característica %s não possui variáveis ativas válidas", ic.CharCode)
		}
		dims = append(dims, opts)
	}
	if len(dims) == 0 {
		return nil, fmt.Errorf("item %d não possui características do tipo Escolha configuradas", dto.ItemCode)
	}

	total := 1
	for _, d := range dims {
		total *= len(d)
		if total > maxCombinations {
			return nil, fmt.Errorf("produto cartesiano excede %d combinações — restrinja mais características", maxCombinations)
		}
	}

	out := &response.CfgGeneratedMasksResponse{ItemCode: dto.ItemCode, TotalCombinations: total}
	seen := map[string]struct{}{}

	// Iterate the cartesian product via a mixed-radix counter.
	idx := make([]int, len(dims))
	for {
		combo := make([]cartOption, len(dims))
		for i := range dims {
			combo[i] = dims[i][idx[i]]
		}

		valid, err := uc.comboValid(ctx, dto, combo)
		if err != nil {
			return nil, err
		}
		if valid {
			segments := make([]entity.MaskSegment, 0, len(combo))
			answers := make([]response.CfgMaskAnswerResponse, 0, len(combo))
			for _, o := range combo {
				segments = append(segments, entity.MaskSegment{Position: o.position, Value: o.maskComposition})
				vid := o.variableID
				answers = append(answers, response.CfgMaskAnswerResponse{
					Position: o.position, CharacteristicID: o.characteristicID, VariableID: &vid, Value: o.maskComposition,
				})
			}
			mask, hash := entity.BuildMask(segments)
			if _, dup := seen[mask]; !dup {
				seen[mask] = struct{}{}
				out.ValidCount++
				out.Masks = append(out.Masks, response.CfgGeneratedMaskItem{Mask: mask, MaskHash: hash, Answers: answers})
				if dto.Persist {
					if err := uc.persistMask(ctx, dto, mask, hash, answers); err != nil {
						return nil, err
					}
					out.Persisted++
				}
			}
		}

		// increment the mixed-radix counter
		pos := len(dims) - 1
		for pos >= 0 {
			idx[pos]++
			if idx[pos] < len(dims[pos]) {
				break
			}
			idx[pos] = 0
			pos--
		}
		if pos < 0 {
			break
		}
	}
	return out, nil
}

// comboValid runs the restriction oracle (when wired) over the combination.
func (uc *ConfiguratorUseCase) comboValid(ctx context.Context, dto request.CfgGenerateMasksDTO, combo []cartOption) (bool, error) {
	if uc.Restrictions == nil {
		return true, nil
	}
	answers := make(map[int64]string, len(combo))
	for _, o := range combo {
		answers[o.characteristicID] = o.code
	}
	itemCode := dto.ItemCode
	return uc.Restrictions.EvaluateCombination(ctx, &itemCode, dto.CustomerCode, dto.DivisionID, answers)
}

func (uc *ConfiguratorUseCase) persistMask(ctx context.Context, dto request.CfgGenerateMasksDTO, mask, hash string, answers []response.CfgMaskAnswerResponse) error {
	maskID, err := uc.Q.PersistCfgItemMask(ctx, dto.ItemCode, mask, hash, pgutil.ToPgUUID(dto.CreatedBy))
	if err != nil {
		return fmt.Errorf("persistindo máscara: %w", err)
	}
	for _, a := range answers {
		_ = uc.Q.InsertCfgItemMaskAnswer(ctx, maskID, a.CharacteristicID, pgutil.ToPgInt8Ptr(a.VariableID), a.Value, int32(a.Position))
	}
	return nil
}

func containsInt64(s []int64, v int64) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
