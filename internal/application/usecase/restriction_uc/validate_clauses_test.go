package restriction_uc

import (
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func TestValidarClausulasRecusaOperadorForaDaLista(t *testing.T) {
	err := validarClausulas(
		[]request.DominantDTO{{QuestionID: 1, Operator: "EQ", ConditionType: "AND", AnswerValue: "PT"}},
		nil,
	)
	if err == nil {
		t.Fatal("operador \"EQ\" não existe no banco e deveria ser recusado antes da gravação")
	}
	msg := err.Error()
	for _, esperado := range []string{"EQ", "EQUAL", "DIFFERENT", "INVALID"} {
		if !strings.Contains(msg, esperado) {
			t.Fatalf("a recusa deveria citar %q para o usuário saber o que usar; veio: %s", esperado, msg)
		}
	}
}

func TestValidarClausulasRecusaConectorInvalido(t *testing.T) {
	err := validarClausulas(
		[]request.DominantDTO{{QuestionID: 1, Operator: "EQUAL", ConditionType: "XOR"}},
		nil,
	)
	if err == nil {
		t.Fatal("conector \"XOR\" deveria ser recusado")
	}
	if !strings.Contains(err.Error(), "OR") {
		t.Fatalf("a recusa deveria citar os conectores aceitos; veio: %s", err.Error())
	}
}

func TestValidarClausulasAceitaOsOperadoresDoFoccoEmMinusculas(t *testing.T) {
	valor := "600"
	err := validarClausulas(
		[]request.DominantDTO{
			{QuestionID: 1, Operator: "equal", ConditionType: "and", AnswerValue: "PT"},
			{QuestionID: 2, Operator: "belongs", ConditionType: "or", AnswerValue: "400,450"},
		},
		[]request.DeterminantDTO{{QuestionID: 3, Operator: "invalid", AnswerValue: &valor}},
	)
	if err != nil {
		t.Fatalf("operadores válidos deveriam passar independentemente da caixa: %v", err)
	}
}

func TestValidarClausulasAceitaConectorVazio(t *testing.T) {
	if err := validarClausulas([]request.DominantDTO{{QuestionID: 1, Operator: "EQUAL"}}, nil); err != nil {
		t.Fatalf("a primeira condição não precisa de conector: %v", err)
	}
}

func TestNormalizacaoDeixaOValorNaFormaQueOBancoAceita(t *testing.T) {
	if got := normalizarOperador(" equal "); string(got) != "EQUAL" {
		t.Fatalf("operador deveria virar EQUAL, veio %q", got)
	}
	if got := normalizarCondicao(""); string(got) != "AND" {
		t.Fatalf("conector vazio deveria virar AND (E), veio %q", got)
	}
	if got := normalizarCondicao("or"); string(got) != "OR" {
		t.Fatalf("conector deveria virar OR, veio %q", got)
	}
}
