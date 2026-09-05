package valueobject

import (
	"errors"
	"fmt"
	"math/rand"
)

type ProductCodeMask struct {
	value string
}

func NewProductCodeMask(groupCode string) (ProductCodeMask, error) {
	if len(groupCode) < 2 {
		return ProductCodeMask{}, errors.New("o código do grupo precisa de ao menos 2 caracteres")
	}
	group := groupCode[:2]
	random := rand.Intn(10000)
	code := fmt.Sprintf("%s%02d", group, random)

	return ProductCodeMask{value: code}, nil

}
func (p ProductCodeMask) String() string {
	return p.value
}
