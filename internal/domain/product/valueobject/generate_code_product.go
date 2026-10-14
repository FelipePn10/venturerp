package valueobject

import (
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"math/rand"
)

type ProductCode struct {
	value string
}

func NewProductCode(groupCode string) (ProductCode, error) {
	if len(groupCode) < 2 {
		return ProductCode{}, domainerr.NewValidation("o código do grupo precisa de ao menos 2 caracteres")
	}

	group := groupCode[:2]
	random := rand.Intn(1000)

	code := fmt.Sprintf("%s%02d", group, random)

	return ProductCode{value: code}, nil
}

func (p ProductCode) String() string {
	return p.value
}
