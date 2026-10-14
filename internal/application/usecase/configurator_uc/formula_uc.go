package configurator_uc

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/formula"
)

// normalizeCode maps a characteristic code to a formula identifier: uppercase,
// with any non [A-Z0-9_] run collapsed to a single underscore. Formulas reference
// other characteristics by this normalized code (e.g. "COR LAM EXT" → "COR_LAM_EXT").
func normalizeCode(code string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToUpper(strings.TrimSpace(code)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevUnderscore = false
		} else if !prevUnderscore {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

// formatNum renders a formula result without trailing zeros (66, 1.5, …).
func formatNum(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// formulaVars builds the variable map (normalized characteristic code → numeric
// value) from a set of answers, used to evaluate rule formulas. The numeric value
// is parsed from the chosen variable's mask composition (or the raw value).
func (uc *ConfiguratorUseCase) formulaVars(ctx context.Context, answers []request.CfgMaskAnswerInput) (map[string]float64, error) {
	vars := map[string]float64{}
	for _, a := range answers {
		// Pergunta ou variável apagada só fica de fora; falha de leitura é erro —
		// antes a fórmula era avaliada sem a variável e a quantidade saía errada.
		char, err := uc.Q.GetCfgCharacteristic(ctx, a.CharacteristicID)
		if errorsuc.EhNaoEncontrado(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("lendo a pergunta %d do configurador: %w", a.CharacteristicID, err)
		}
		numStr := a.Value
		v, err := uc.variavelDaResposta(ctx, a)
		if err != nil {
			return nil, err
		}
		if v != nil {
			numStr = v.MaskComposition
		}
		if n, ok := formula.ParseOptionValue(numStr); ok {
			vars[normalizeCode(char.Code)] = n
		}
	}
	return vars, nil
}

// variavelDaResposta lê a variável escolhida na resposta; nil quando a resposta
// é livre ou a variável não existe mais (vale o valor digitado).
func (uc *ConfiguratorUseCase) variavelDaResposta(ctx context.Context, a request.CfgMaskAnswerInput) (*sqlc.DBCfgVariable, error) {
	if a.VariableID == nil {
		return nil, nil
	}
	v, err := uc.Q.GetCfgVariable(ctx, *a.VariableID)
	if errorsuc.EhNaoEncontrado(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a variável %d do configurador: %w", *a.VariableID, err)
	}
	return &v, nil
}
