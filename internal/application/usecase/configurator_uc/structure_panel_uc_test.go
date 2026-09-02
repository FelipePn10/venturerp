package configurator_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

func TestStructurePanel_RejectsInvalidItem(t *testing.T) {
	uc := &ConfiguratorUseCase{}
	_, err := uc.StructurePanel(context.Background(), 0, "")
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
}

// Sem motor de restrições cadastrado, nenhuma combinação é barrada.
func TestValidateCombination_NoOracleAcceptsEverything(t *testing.T) {
	uc := &ConfiguratorUseCase{}
	got, err := uc.ValidateCombination(context.Background(), 1, []request.CfgMaskAnswerInput{{CharacteristicID: 1, Value: "X"}})
	if err != nil || got != nil {
		t.Fatalf("violações = %v err=%v, quer nenhuma", got, err)
	}
}

type oracleWithoutExplain struct{ valid bool }

func (o *oracleWithoutExplain) EvaluateCombination(context.Context, *int64, *int64, *int64, map[int64]string) (bool, error) {
	return o.valid, nil
}

// Um oráculo que só sabe dizer sim/não não bloqueia o painel: sem detalhamento,
// a validação detalhada é simplesmente pulada.
func TestValidateCombination_OracleWithoutExplainIsSkipped(t *testing.T) {
	uc := (&ConfiguratorUseCase{}).WithRestrictions(&oracleWithoutExplain{valid: false})
	got, err := uc.ValidateCombination(context.Background(), 1, nil)
	if err != nil || got != nil {
		t.Fatalf("violações = %v err=%v, quer nenhuma", got, err)
	}
}

func TestViolationSummary_JoinsMessages(t *testing.T) {
	got := violationSummary([]response.StructureConfiguratorViolation{
		{Message: "Cor: a resposta não pode ser VERMELHO"},
		{Message: "Comprimento: a resposta precisa ser menor que 3000"},
	})
	want := "a combinação de respostas viola as restrições cadastradas: " +
		"Cor: a resposta não pode ser VERMELHO; Comprimento: a resposta precisa ser menor que 3000"
	if got != want {
		t.Fatalf("resumo = %q", got)
	}
}
