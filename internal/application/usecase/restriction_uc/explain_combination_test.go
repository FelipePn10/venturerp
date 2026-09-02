package restriction_uc

import (
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
)

func ptr(s string) *string { return &s }

func TestDeterminantSatisfied_PerOperator(t *testing.T) {
	answers := map[int64]string{1: "AZUL", 2: "10"}
	cases := []struct {
		name string
		det  *entity.RestrictionDeterminant
		want bool
	}{
		{"igual atende", &entity.RestrictionDeterminant{QuestionID: 1, Operator: entity.OperatorEqual, AnswerValue: ptr("AZUL")}, true},
		{"igual não atende", &entity.RestrictionDeterminant{QuestionID: 1, Operator: entity.OperatorEqual, AnswerValue: ptr("VERDE")}, false},
		{"diferente atende", &entity.RestrictionDeterminant{QuestionID: 1, Operator: entity.OperatorDifferent, AnswerValue: ptr("VERDE")}, true},
		{"diferente não atende", &entity.RestrictionDeterminant{QuestionID: 1, Operator: entity.OperatorDifferent, AnswerValue: ptr("azul")}, false},
		{"pertence atende", &entity.RestrictionDeterminant{QuestionID: 1, Operator: entity.OperatorBelongs, AnswerValue: ptr("AZUL, VERDE")}, true},
		{"não pertence atende", &entity.RestrictionDeterminant{QuestionID: 1, Operator: entity.OperatorNotBelongs, AnswerValue: ptr("VERMELHO")}, true},
		{"inválida sempre recusa", &entity.RestrictionDeterminant{QuestionID: 1, Operator: entity.OperatorInvalid}, false},
		{"sem resposta recusa igualdade", &entity.RestrictionDeterminant{QuestionID: 99, Operator: entity.OperatorEqual, AnswerValue: ptr("X")}, false},
		{"sem resposta aceita diferença", &entity.RestrictionDeterminant{QuestionID: 99, Operator: entity.OperatorDifferent, AnswerValue: ptr("X")}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := determinantSatisfied(tc.det, answers); got != tc.want {
				t.Fatalf("determinantSatisfied = %v, quer %v", got, tc.want)
			}
		})
	}
}

// As mensagens vão para a tela do configurador: precisam estar em PT-BR e citar
// o valor esperado e o respondido.
func TestViolationMessage_IsPortugueseAndInformative(t *testing.T) {
	cases := []struct {
		op       entity.RestrictionOperator
		contains []string
	}{
		{entity.OperatorInvalid, []string{"não é permitida"}},
		{entity.OperatorEqual, []string{"precisa ser", "AZUL", "VERDE"}},
		{entity.OperatorDifferent, []string{"não pode ser", "AZUL"}},
		{entity.OperatorBelongs, []string{"entre", "AZUL"}},
		{entity.OperatorNotBelongs, []string{"não pode estar entre", "AZUL"}},
		{entity.OperatorGreater, []string{"maior que", "AZUL"}},
		{entity.OperatorLess, []string{"menor que", "AZUL"}},
	}
	for _, tc := range cases {
		got := violationMessage(tc.op, "AZUL", "VERDE")
		for _, needle := range tc.contains {
			if !strings.Contains(got, needle) {
				t.Errorf("mensagem de %s = %q, esperava conter %q", tc.op, got, needle)
			}
		}
	}
}

func TestViolationMessage_UnansweredIsExplicit(t *testing.T) {
	got := violationMessage(entity.OperatorEqual, "AZUL", "")
	if !strings.Contains(got, "sem resposta") {
		t.Fatalf("mensagem = %q, esperava indicar a ausência de resposta", got)
	}
}
