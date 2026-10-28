package sales_order_uc

import (
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// normalizarFinalidade valida a finalidade da compra do pedido. Vazia vale
// (a alteração usa vazio para limpar); nil continua nil.
func normalizarFinalidade(f *string) (*string, error) {
	if f == nil {
		return nil, nil
	}
	v := strings.ToUpper(strings.TrimSpace(*f))
	switch v {
	case "", "REVENDA", "INDUSTRIALIZACAO", "USO_CONSUMO":
		return &v, nil
	}
	return nil, errorsuc.NewValidationError("finalidade da compra inválida: use REVENDA, INDUSTRIALIZACAO ou USO_CONSUMO")
}
