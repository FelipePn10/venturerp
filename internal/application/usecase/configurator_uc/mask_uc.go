package configurator_uc

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/configurator/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/formula"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

// GenerateMask assembles the configured item's mask from the answers, ordered by
// the item-characteristic sequence, and (optionally) persists it to item_masks so
// downstream (structure/sales/MRP) sees it. It is bit-compatible with the legacy
// mask value object (# join + 8-char sha256).
func (uc *ConfiguratorUseCase) GenerateMask(ctx context.Context, dto request.CfgGenerateMaskDTO) (*response.CfgGeneratedMaskResponse, error) {
	if dto.ItemCode <= 0 {
		return nil, fmt.Errorf("item_code é obrigatório")
	}
	itemChars, err := uc.Q.ListCfgItemCharacteristics(ctx, dto.ItemCode)
	if err != nil {
		return nil, fmt.Errorf("carregando características do item: %w", err)
	}
	if len(itemChars) == 0 {
		return nil, fmt.Errorf("item %d não possui características configuradas", dto.ItemCode)
	}
	// gather provided answers per characteristic (allows multi-select)
	answers := map[int64][]request.CfgMaskAnswerInput{}
	for _, a := range dto.Answers {
		answers[a.CharacteristicID] = append(answers[a.CharacteristicID], a)
	}

	segments := make([]entity.MaskSegment, 0, len(itemChars))
	respAnswers := make([]response.CfgMaskAnswerResponse, 0, len(itemChars))

	// vars accumulates numeric answers (normalized characteristic code → value)
	// so a FORMULA characteristic can reference earlier characteristics' answers.
	vars := map[string]float64{}

	for _, ic := range itemChars {
		char, err := uc.Q.GetCfgCharacteristic(ctx, ic.CharacteristicID)
		if err != nil {
			return nil, fmt.Errorf("característica %d não encontrada", ic.CharacteristicID)
		}
		var value string
		var variableID *int64
		if char.CharType == entity.TypeFormula {
			value, err = uc.evalFormulaAnswer(ic, char, vars)
		} else {
			value, variableID, err = uc.resolveAnswer(ctx, ic, char, answers[ic.CharacteristicID])
		}
		if err != nil {
			return nil, err
		}
		if value == "" {
			continue // unanswered optional characteristic
		}
		// register this answer for subsequent formulas
		if n, ok := formula.ParseOptionValue(value); ok {
			vars[normalizeCode(char.Code)] = n
		}
		pos := int(ic.Sequence)
		segments = append(segments, entity.MaskSegment{Position: pos, Value: value})
		respAnswers = append(respAnswers, response.CfgMaskAnswerResponse{
			Position: pos, CharacteristicID: ic.CharacteristicID, VariableID: variableID, Value: value,
		})
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("nenhuma resposta válida para compor a máscara")
	}

	mask, hash := entity.BuildMask(segments)
	out := &response.CfgGeneratedMaskResponse{
		ItemCode: dto.ItemCode, Mask: mask, MaskHash: hash, Answers: respAnswers,
	}

	if dto.Persist {
		maskID, err := uc.Q.PersistCfgItemMask(ctx, dto.ItemCode, mask, hash, pgutil.ToPgUUID(dto.CreatedBy))
		if err != nil {
			return nil, fmt.Errorf("persistindo máscara: %w", err)
		}
		for _, a := range respAnswers {
			_ = uc.Q.InsertCfgItemMaskAnswer(ctx, maskID, a.CharacteristicID,
				pgutil.ToPgInt8Ptr(a.VariableID), a.Value, int32(a.Position))
		}
		out.Persisted = true
		out.MaskID = &maskID
	}
	return out, nil
}

// resolveAnswer computes the mask value for one item-characteristic, applying
// type-specific rules and defaults. Returns the value, the (single) variable id
// used (when applicable) and any validation error.
func (uc *ConfiguratorUseCase) resolveAnswer(
	ctx context.Context,
	ic sqlc.DBCfgItemCharacteristic,
	char sqlc.DBCfgCharacteristic,
	provided []request.CfgMaskAnswerInput,
) (string, *int64, error) {
	switch char.CharType {
	case entity.TypeEscolha:
		return uc.resolveChoice(ctx, ic, provided)
	case entity.TypeEscolhaMult:
		return uc.resolveMultiChoice(ctx, ic, provided)
	case entity.TypeInfCaracter:
		val := firstValue(provided)
		if val == "" && char.IsRequired {
			return "", nil, fmt.Errorf("característica %s é de preenchimento obrigatório", char.Code)
		}
		return val, nil, nil
	case entity.TypeInfNumerica:
		return resolveNumeric(char, firstValue(provided))
	case entity.TypeOpcao:
		return resolveOption(char, firstValue(provided)), nil, nil
	case entity.TypeDesenho, entity.TypeCampo, entity.TypeSequencial, entity.TypeFormula:
		// value supplied by the caller (drawing code, field value, computed formula)
		return firstValue(provided), nil, nil
	default:
		return firstValue(provided), nil, nil
	}
}

// resolveChoice returns the mask composition of the chosen variable, defaulting
// to the item-level default variable when unanswered.
func (uc *ConfiguratorUseCase) resolveChoice(ctx context.Context, ic sqlc.DBCfgItemCharacteristic, provided []request.CfgMaskAnswerInput) (string, *int64, error) {
	var varID *int64
	if len(provided) > 0 && provided[0].VariableID != nil {
		varID = provided[0].VariableID
	} else if ic.DefaultVariableID.Valid {
		v := ic.DefaultVariableID.Int64
		varID = &v
	}
	if varID == nil {
		return "", nil, nil // unanswered, no default
	}
	v, err := uc.Q.GetCfgVariable(ctx, *varID)
	if err != nil {
		return "", nil, fmt.Errorf("variável %d não encontrada", *varID)
	}
	return v.MaskComposition, varID, nil
}

// resolveMultiChoice joins the mask compositions of all selected variables with
// '+', defaulting to the item-characteristic's registered default answers.
func (uc *ConfiguratorUseCase) resolveMultiChoice(ctx context.Context, ic sqlc.DBCfgItemCharacteristic, provided []request.CfgMaskAnswerInput) (string, *int64, error) {
	ids := make([]int64, 0, len(provided))
	for _, a := range provided {
		if a.VariableID != nil {
			ids = append(ids, *a.VariableID)
		}
	}
	if len(ids) == 0 {
		defaults, _ := uc.Q.ListCfgItemCharDefaultAnswers(ctx, ic.ID)
		ids = defaults
	}
	if len(ids) == 0 {
		return "", nil, nil
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		v, err := uc.Q.GetCfgVariable(ctx, id)
		if err != nil {
			return "", nil, fmt.Errorf("variável %d não encontrada", id)
		}
		parts = append(parts, v.MaskComposition)
	}
	return strings.Join(parts, "+"), nil, nil
}

func resolveNumeric(char sqlc.DBCfgCharacteristic, val string) (string, *int64, error) {
	if val == "" {
		return "", nil, nil
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil {
		return "", nil, fmt.Errorf("característica %s exige um número", char.Code)
	}
	if char.NumMin.Valid && n < pgutil.FromPgNumericToFloat64(char.NumMin) {
		return "", nil, fmt.Errorf("característica %s: valor abaixo do mínimo", char.Code)
	}
	if char.NumMax.Valid && n > pgutil.FromPgNumericToFloat64(char.NumMax) {
		return "", nil, fmt.Errorf("característica %s: valor acima do máximo", char.Code)
	}
	if char.NumMultiple.Valid {
		m := pgutil.FromPgNumericToFloat64(char.NumMultiple)
		if m > 0 && math.Mod(n, m) != 0 {
			return "", nil, fmt.Errorf("característica %s: valor deve ser múltiplo de %g", char.Code, m)
		}
	}
	return val, nil, nil
}

func resolveOption(char sqlc.DBCfgCharacteristic, val string) string {
	up := strings.ToUpper(strings.TrimSpace(val))
	switch up {
	case "SIM", "S", "YES", "TRUE", "1":
		if t := pgutil.FromPgText(char.OptionTrue); t != "" {
			return t
		}
		return "SIM"
	case "":
		return ""
	default:
		if f := pgutil.FromPgText(char.OptionFalse); f != "" {
			return f
		}
		return "NAO"
	}
}

func firstValue(a []request.CfgMaskAnswerInput) string {
	if len(a) == 0 {
		return ""
	}
	return strings.TrimSpace(a[0].Value)
}

// evalFormulaAnswer computes a FORMULA characteristic's answer from its formula
// (item-level, falling back to the characteristic default) using the numeric
// answers already resolved (only earlier sequences are available).
func (uc *ConfiguratorUseCase) evalFormulaAnswer(ic sqlc.DBCfgItemCharacteristic, char sqlc.DBCfgCharacteristic, vars map[string]float64) (string, error) {
	expr := pgutil.FromPgText(ic.Formula)
	if expr == "" {
		expr = pgutil.FromPgText(char.Formula)
	}
	if expr == "" {
		return "", fmt.Errorf("característica %s é do tipo fórmula mas não possui fórmula", char.Code)
	}
	r, err := formula.Evaluate(expr, vars)
	if err != nil {
		return "", fmt.Errorf("erro ao calcular a fórmula da característica %s: %w", char.Code, err)
	}
	return formatNum(r), nil
}
