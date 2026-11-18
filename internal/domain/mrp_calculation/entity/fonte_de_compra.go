package entity

import "github.com/shopspring/decimal"

// FonteDeCompra é um fornecedor do item comprado, válido hoje (ativo,
// desbloqueado e vigente — VSUP0130), com a política de compra dele.
type FonteDeCompra struct {
	SupplierCode  int64
	Mask          string // vazio = vale para qualquer máscara
	Preferred     bool
	Ranking       int32
	LeadTimeDays  int
	MinOrderQty   decimal.Decimal
	OrderMultiple decimal.Decimal
	// QuotaPercent: cota do fornecedor. nil = fora da divisão por cota.
	QuotaPercent *decimal.Decimal
	// Alocado: o que já foi pedido a ele desde o início da cota (pedidos não
	// cancelados) mais o que este cálculo já mandou para ele.
	Alocado decimal.Decimal
}
