package configurator_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc"
)

// O painel só valida restrições se o motor real satisfizer a interface; sem esta
// verificação, uma mudança de assinatura desativaria silenciosamente o FENG0116.
var _ CombinationExplainer = (*restriction_uc.EvaluateRestrictionsUseCase)(nil)
var _ RestrictionOracle = (*restriction_uc.EvaluateRestrictionsUseCase)(nil)

func TestRestrictionEngineSatisfiesConfiguratorPorts(t *testing.T) {}
