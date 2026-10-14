package valueobject

import (
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"math/rand"
)

type ComponentCode struct {
	value string
}

func NewComponentCode(groupCode string) (ComponentCode, error) {
	if len(groupCode) < 2 {
		return ComponentCode{}, domainerr.NewValidation("o código do grupo precisa de ao menos 2 caracteres")
	}

	group := groupCode[:2]
	random := rand.Intn(10000)

	code := fmt.Sprintf("%s%04d", group, random)
	return ComponentCode{value: code}, nil
}

func (c ComponentCode) String() string {
	return c.value
}
