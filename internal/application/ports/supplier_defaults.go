package ports

import "context"

// SupplierPurchasingDefaults carries the supplier-derived defaults consumed by
// the Purchase Order and Fiscal (NF entry) flows. It is the single source of
// truth for "what does this supplier default to" when a document is created.
type SupplierPurchasingDefaults struct {
	SupplierCode       int64   `json:"supplier_code"`
	SupplierName       string  `json:"supplier_name"`
	IsActive           bool    `json:"is_active"`
	PaymentConditionID *int64  `json:"payment_condition_id,omitempty"`
	FreightType        string  `json:"freight_type"`
	ICMSContributor    string  `json:"icms_contributor"`
	StateRegistration  *string `json:"state_registration,omitempty"`
	// Per-enterprise binding (pasta Empresas). Zero/nil when the supplier has no
	// link for the requested enterprise.
	FinancialAccount     *string `json:"financial_account,omitempty"`
	DefaultInvoiceTypeID *int64  `json:"default_invoice_type_id,omitempty"`
	PurchasePriceTableID *int64  `json:"purchase_price_table_id,omitempty"`
	// PurchasePriceTableCode é o CÓDIGO da tabela acima, que é o que o pedido
	// guarda e o que a busca de preço usa. O vínculo guarda o id: usar o id
	// como código levava o pedido a outra tabela, ou a nenhuma.
	PurchasePriceTableCode *int64 `json:"purchase_price_table_code,omitempty"`
	AppliesIPI             bool   `json:"applies_ipi"`
}

// TabelaDeCompraPorID traduz o id da tabela de preço de compra no código dela,
// na empresa da sessão.
type TabelaDeCompraPorID interface {
	CodigoDaTabela(ctx context.Context, id int64) (code int64, achou bool, err error)
}

// SupplierPurchasingDefaultsProvider is the outbound port implemented by the
// supplier module and consumed by the purchase_order and fiscal use cases.
// Pass nil to disable the integration (behaviour without supplier defaults).
type SupplierPurchasingDefaultsProvider interface {
	// GetPurchasingDefaults resolves the defaults for a supplier within an
	// enterprise. enterpriseCode may be 0 when the caller has no enterprise
	// context (per-enterprise fields are then left empty).
	GetPurchasingDefaults(ctx context.Context, supplierCode, enterpriseCode int64) (*SupplierPurchasingDefaults, error)
	// FindSupplierCodeByDocument returns the supplier code matching a CNPJ/CPF
	// (digits only), and whether a match was found. Used to link an incoming
	// NF-e emitter to a registered supplier.
	FindSupplierCodeByDocument(ctx context.Context, document string) (int64, bool, error)
}
