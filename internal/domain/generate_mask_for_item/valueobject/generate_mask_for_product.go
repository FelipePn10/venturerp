package valueobject

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type MaskAnswer struct {
	questionID  int64
	optionID    int64
	optionValue string
	position    int
}

type ItemMask struct {
	itemCode  int64
	createdBy uuid.UUID
	answers   []MaskAnswer
	mask      string
	hash      string
}

func NewMaskAnswer(questionID, optionID int64, position int, value string) (MaskAnswer, error) {
	if questionID <= 0 {
		return MaskAnswer{}, domainerr.NewValidation("pergunta inválida")
	}
	if optionID <= 0 {
		return MaskAnswer{}, domainerr.NewValidation("opção inválida")
	}
	if position <= 0 {
		return MaskAnswer{}, domainerr.NewValidation("a posição da resposta na máscara deve ser maior que zero")
	}
	if value == "" {
		return MaskAnswer{}, domainerr.NewValidation("valor de opção inválido")
	}

	return MaskAnswer{
		questionID:  questionID,
		optionID:    optionID,
		optionValue: value,
		position:    position,
	}, nil
}

func NewItemMask(itemCode int64, answers []MaskAnswer) (ItemMask, error) {
	if itemCode < 0 {
		return ItemMask{}, domainerr.NewValidation("código de item inválido")
	}
	if len(answers) == 0 {
		return ItemMask{}, domainerr.NewValidation("a máscara precisa de ao menos uma resposta")
	}

	mask := generateMask(answers)

	h := sha256.Sum256([]byte(mask))
	hash := hex.EncodeToString(h[:])[:8]

	return ItemMask{
		itemCode: itemCode,
		answers:  answers,
		mask:     mask,
		hash:     hash,
	}, nil
}

func generateMask(answers []MaskAnswer) string {
	copied := make([]MaskAnswer, len(answers))
	copy(copied, answers)

	sort.Slice(copied, func(i, j int) bool {
		return copied[i].position < copied[j].position
	})

	values := make([]string, 0, len(copied))
	for _, a := range copied {
		values = append(values, a.optionValue)
	}

	return strings.Join(values, "#")
}

// Getters
func (pm ItemMask) Value() string {
	return pm.mask
}

func (pm ItemMask) Hash() string {
	return pm.hash
}

func (ma MaskAnswer) QuestionID() int64 {
	return ma.questionID
}

func (ma MaskAnswer) OptionID() int64 {
	return ma.optionID
}

func (ma MaskAnswer) Position() int {
	return ma.position
}

func (ma MaskAnswer) OptionValue() string {
	return ma.optionValue
}
