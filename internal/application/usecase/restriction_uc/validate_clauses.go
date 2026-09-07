package restriction_uc

import (
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	enumtypes "github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
)

// Um operador fora da lista chegava ao banco e virava 500 ("invalid input value
// for enum"), deixando a restrição gravada pela metade. A conferência acontece
// antes de qualquer escrita, e a recusa explica em português o que é aceito.
var operadoresAceitos = []entity.RestrictionOperator{
	entity.OperatorEqual, entity.OperatorDifferent, entity.OperatorGreater,
	entity.OperatorLess, entity.OperatorBelongs, entity.OperatorNotBelongs,
	entity.OperatorInvalid,
}

var condicoesAceitas = []entity.RestrictionCondition{entity.ConditionAnd, entity.ConditionOr}

func textos[T ~string](valores []T) []string {
	out := make([]string, 0, len(valores))
	for _, v := range valores {
		out = append(out, string(v))
	}
	return out
}

func validarOperador(campo, valor string) error {
	normalizado := strings.ToUpper(strings.TrimSpace(valor))
	for _, aceito := range operadoresAceitos {
		if normalizado == string(aceito) {
			return nil
		}
	}
	return enumtypes.NewInvalidValue(campo, valor, textos(operadoresAceitos)...)
}

func validarCondicao(campo, valor string) error {
	normalizado := strings.ToUpper(strings.TrimSpace(valor))
	if normalizado == "" {
		return nil // o padrão é E (AND)
	}
	for _, aceita := range condicoesAceitas {
		if normalizado == string(aceita) {
			return nil
		}
	}
	return enumtypes.NewInvalidValue(campo, valor, textos(condicoesAceitas)...)
}

// validarClausulas confere dominantes ("SE") e determinantes ("ENTÃO") antes de
// gravar qualquer linha.
func validarClausulas(dominantes []request.DominantDTO, determinantes []request.DeterminantDTO) error {
	for _, d := range dominantes {
		if err := validarOperador("Operador da condição (SE)", d.Operator); err != nil {
			return err
		}
		if err := validarCondicao("Conector da condição (E/OU)", d.ConditionType); err != nil {
			return err
		}
	}
	for _, d := range determinantes {
		if err := validarOperador("Operador da consequência (ENTÃO)", d.Operator); err != nil {
			return err
		}
	}
	return nil
}

// normalizarOperador e normalizarCondicao devolvem o valor na forma que o banco
// aceita — quem digita "equal" ou deixa o conector em branco não deveria ver
// erro por causa disso.
func normalizarOperador(valor string) entity.RestrictionOperator {
	return entity.RestrictionOperator(strings.ToUpper(strings.TrimSpace(valor)))
}

func normalizarCondicao(valor string) entity.RestrictionCondition {
	normalizado := strings.ToUpper(strings.TrimSpace(valor))
	if normalizado == "" {
		return entity.ConditionAnd
	}
	return entity.RestrictionCondition(normalizado)
}
