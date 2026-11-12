// Package pedidocompra desenha o pedido de compra que vai ao fornecedor.
package pedidocompra

import (
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/documentos"
)

// Gerador implementa purchase_order_uc.GeradorPDFPedido; o layout mora em
// export/documentos, junto dos demais documentos impressos.
type Gerador struct{}

var _ purchase_order_uc.GeradorPDFPedido = Gerador{}

// PedidoCompra devolve o PDF do pedido.
func (Gerador) PedidoCompra(d purchase_order_uc.DocumentoPedido) ([]byte, error) {
	if d.Dados == nil {
		return nil, fmt.Errorf("dados do pedido ausentes")
	}
	return documentos.PedidoCompra(d), nil
}
