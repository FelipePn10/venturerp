package service

import (
	"strings"

	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
)

type ItemQuestion struct {
	QuestionID int64
	Position   int32
}

// ChildMaskAnswerInput carrega os dados para persistir uma resposta de máscara propagada.
type ChildMaskAnswerInput struct {
	QuestionID int64
	OptionID   int64
	Position   int32
}

// PropagateMask deriva a string de máscara do filho a partir das respostas do pai.
// A máscara é a junção dos optionValues (na ordem das perguntas do filho) com "#".
// Retorna nil quando o filho não tem perguntas ou alguma resposta do pai está ausente.
func PropagateMask(parentAnswers []maskvo.MaskAnswer, childQuestions []ItemQuestion) *string {
	if len(childQuestions) == 0 {
		return nil
	}

	valueByQuestion := make(map[int64]string, len(parentAnswers))
	for _, a := range parentAnswers {
		valueByQuestion[a.QuestionID()] = a.OptionValue()
	}

	parts := make([]string, 0, len(childQuestions))
	for _, q := range childQuestions {
		v, ok := valueByQuestion[q.QuestionID]
		if !ok {
			return nil // propagação incompleta → trata como genérico
		}
		parts = append(parts, v)
	}

	mask := strings.Join(parts, "#")
	return &mask
}

// DeriveChildAnswers constrói as entradas de resposta necessárias para persistir
// uma máscara propagada, cruzando as perguntas do filho com as respostas do pai.
func DeriveChildAnswers(parentAnswers []maskvo.MaskAnswer, childQuestions []ItemQuestion) []ChildMaskAnswerInput {
	byQuestion := make(map[int64]maskvo.MaskAnswer, len(parentAnswers))
	for _, a := range parentAnswers {
		byQuestion[a.QuestionID()] = a
	}

	out := make([]ChildMaskAnswerInput, 0, len(childQuestions))
	for _, q := range childQuestions {
		if pa, ok := byQuestion[q.QuestionID]; ok {
			out = append(out, ChildMaskAnswerInput{
				QuestionID: q.QuestionID,
				OptionID:   pa.OptionID(),
				Position:   q.Position,
			})
		}
	}
	return out
}
