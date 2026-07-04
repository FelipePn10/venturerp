package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
)

type CustomerRepository interface {
	// ── Regions ──────────────────────────────────────────────────────────────
	CreateRegion(ctx context.Context, r *entity.Region) (*entity.Region, error)
	UpdateRegion(ctx context.Context, r *entity.Region) (*entity.Region, error)
	GetRegionByCode(ctx context.Context, code int64) (*entity.Region, error)
	ListRegions(ctx context.Context, onlyActive bool) ([]*entity.Region, error)
	NextRegionCode(ctx context.Context) (int64, error)

	// ── Market Segments ───────────────────────────────────────────────────────
	CreateMarketSegment(ctx context.Context, s *entity.MarketSegment) (*entity.MarketSegment, error)
	UpdateMarketSegment(ctx context.Context, s *entity.MarketSegment) (*entity.MarketSegment, error)
	GetMarketSegmentByCode(ctx context.Context, code int64) (*entity.MarketSegment, error)
	ListMarketSegments(ctx context.Context, onlyActive bool) ([]*entity.MarketSegment, error)
	NextMarketSegmentCode(ctx context.Context) (int64, error)

	// ── Customer Contact Types ────────────────────────────────────────────────
	CreateContactType(ctx context.Context, ct *entity.CustomerContactType) (*entity.CustomerContactType, error)
	UpdateContactType(ctx context.Context, ct *entity.CustomerContactType) (*entity.CustomerContactType, error)
	GetContactTypeByCode(ctx context.Context, code int64) (*entity.CustomerContactType, error)
	ListContactTypes(ctx context.Context, onlyActive bool) ([]*entity.CustomerContactType, error)
	NextContactTypeCode(ctx context.Context) (int64, error)

	// ── Customer Types ────────────────────────────────────────────────────────
	CreateCustomerType(ctx context.Context, ct *entity.CustomerType) (*entity.CustomerType, error)
	UpdateCustomerType(ctx context.Context, ct *entity.CustomerType) (*entity.CustomerType, error)
	GetCustomerTypeByCode(ctx context.Context, code int64) (*entity.CustomerType, error)
	ListCustomerTypes(ctx context.Context, onlyActive bool) ([]*entity.CustomerType, error)

	// ── Carriers ──────────────────────────────────────────────────────────────
	CreateCarrier(ctx context.Context, c *entity.Carrier) (*entity.Carrier, error)
	UpdateCarrier(ctx context.Context, c *entity.Carrier) (*entity.Carrier, error)
	GetCarrierByCode(ctx context.Context, code int64) (*entity.Carrier, error)
	ListCarriers(ctx context.Context, onlyActive bool) ([]*entity.Carrier, error)
	NextCarrierCode(ctx context.Context) (int64, error)

	// ── Carrier Groups ────────────────────────────────────────────────────────
	CreateCarrierGroup(ctx context.Context, g *entity.CarrierGroup) (*entity.CarrierGroup, error)
	GetCarrierGroupByCode(ctx context.Context, code int64) (*entity.CarrierGroup, error)
	ListCarrierGroups(ctx context.Context) ([]*entity.CarrierGroup, error)
	AddCarrierToGroup(ctx context.Context, groupID, carrierID int64) error
	RemoveCarrierFromGroup(ctx context.Context, groupID, carrierID int64) error
	NextCarrierGroupCode(ctx context.Context) (int64, error)

	// ── Payment Conditions ────────────────────────────────────────────────────
	CreatePaymentCondition(ctx context.Context, pc *entity.PaymentCondition) (*entity.PaymentCondition, error)
	UpdatePaymentCondition(ctx context.Context, pc *entity.PaymentCondition) (*entity.PaymentCondition, error)
	GetPaymentConditionByCode(ctx context.Context, code int64) (*entity.PaymentCondition, error)
	ListPaymentConditions(ctx context.Context, onlyActive bool) ([]*entity.PaymentCondition, error)
	AddInstallment(ctx context.Context, inst *entity.PaymentInstallment) (*entity.PaymentInstallment, error)
	ListInstallments(ctx context.Context, paymentConditionID int64) ([]*entity.PaymentInstallment, error)
	DeleteInstallment(ctx context.Context, id int64) error
	NextPaymentConditionCode(ctx context.Context) (int64, error)

	// ── Sales Tables ──────────────────────────────────────────────────────────
	CreateSalesTable(ctx context.Context, st *entity.SalesTable) (*entity.SalesTable, error)
	UpdateSalesTable(ctx context.Context, st *entity.SalesTable) (*entity.SalesTable, error)
	GetSalesTableByID(ctx context.Context, id int64) (*entity.SalesTable, error)
	GetSalesTableByCode(ctx context.Context, code int64) (*entity.SalesTable, error)
	ListSalesTables(ctx context.Context, onlyActive bool) ([]*entity.SalesTable, error)
	NextSalesTableCode(ctx context.Context) (int64, error)

	// ── Sales Price Policies ─────────────────────────────────────────────────
	CreateSalesPricePolicy(ctx context.Context, p *entity.SalesPricePolicy) (*entity.SalesPricePolicy, error)
	UpdateSalesPricePolicy(ctx context.Context, p *entity.SalesPricePolicy) (*entity.SalesPricePolicy, error)
	GetSalesPricePolicyByCode(ctx context.Context, code int64) (*entity.SalesPricePolicy, error)
	ListSalesPricePolicies(ctx context.Context, onlyActive bool) ([]*entity.SalesPricePolicy, error)
	NextSalesPricePolicyCode(ctx context.Context) (int64, error)

	// ── Commercial Policies ──────────────────────────────────────────────────
	CreateCommercialPolicy(ctx context.Context, p *entity.CommercialPolicy) (*entity.CommercialPolicy, error)
	UpdateCommercialPolicy(ctx context.Context, p *entity.CommercialPolicy) (*entity.CommercialPolicy, error)
	GetCommercialPolicyByCode(ctx context.Context, code int64) (*entity.CommercialPolicy, error)
	ListCommercialPolicies(ctx context.Context, onlyActive bool, kind *entity.CommercialPolicyKind) ([]*entity.CommercialPolicy, error)
	NextCommercialPolicyCode(ctx context.Context) (int64, error)
	AddCommercialPolicyLine(ctx context.Context, line *entity.CommercialPolicyLine) (*entity.CommercialPolicyLine, error)
	ListCommercialPolicyLines(ctx context.Context, policyCode int64) ([]*entity.CommercialPolicyLine, error)
	AddCommercialPolicySpecificItem(ctx context.Context, item *entity.CommercialPolicySpecificItem) (*entity.CommercialPolicySpecificItem, error)
	ListCommercialPolicySpecificItems(ctx context.Context, policyCode int64) ([]*entity.CommercialPolicySpecificItem, error)

	// ── Sales Table Prices ────────────────────────────────────────────────────
	CreateSalesTablePrice(ctx context.Context, p *entity.SalesTablePrice) (*entity.SalesTablePrice, error)
	UpsertSalesTablePrice(ctx context.Context, p *entity.SalesTablePrice) (*entity.SalesTablePrice, *float64, error)
	UpdateSalesTablePrice(ctx context.Context, p *entity.SalesTablePrice) (*entity.SalesTablePrice, error)
	GetSalesTablePriceByID(ctx context.Context, id int64) (*entity.SalesTablePrice, error)
	GetSalesTablePrice(ctx context.Context, salesTableID int64, itemCode string) (*entity.SalesTablePrice, error)
	ListSalesTablePrices(ctx context.Context, salesTableID int64) ([]*entity.SalesTablePrice, error)
	DeleteSalesTablePrice(ctx context.Context, id int64) error
	ResolveSalesCost(ctx context.Context, itemCode int64, mask string, source entity.SalesCostSource, warehouseID *int64) (float64, string, error)
	CreateSalesTablePriceHistory(ctx context.Context, h *entity.SalesTablePriceHistory) (*entity.SalesTablePriceHistory, error)
	ListSalesTablePriceHistory(ctx context.Context, salesTableCode int64, itemCode *string) ([]*entity.SalesTablePriceHistory, error)

	// ── Invoice Types ─────────────────────────────────────────────────────────
	CreateInvoiceType(ctx context.Context, it *entity.InvoiceType) (*entity.InvoiceType, error)
	UpdateInvoiceType(ctx context.Context, it *entity.InvoiceType) (*entity.InvoiceType, error)
	GetInvoiceTypeByCode(ctx context.Context, code int64) (*entity.InvoiceType, error)
	ListInvoiceTypes(ctx context.Context, onlyActive bool) ([]*entity.InvoiceType, error)
	NextInvoiceTypeCode(ctx context.Context) (int64, error)

	// ── Tax Types ─────────────────────────────────────────────────────────────
	CreateTaxType(ctx context.Context, tt *entity.TaxType) (*entity.TaxType, error)
	UpdateTaxType(ctx context.Context, tt *entity.TaxType) (*entity.TaxType, error)
	GetTaxTypeByCode(ctx context.Context, code int64) (*entity.TaxType, error)
	ListTaxTypes(ctx context.Context, onlyActive bool) ([]*entity.TaxType, error)
	NextTaxTypeCode(ctx context.Context) (int64, error)

	// ── Customers ─────────────────────────────────────────────────────────────
	CreateCustomer(ctx context.Context, c *entity.Customer) (*entity.Customer, error)
	UpdateCustomer(ctx context.Context, c *entity.Customer) (*entity.Customer, error)
	GetCustomerByCode(ctx context.Context, code int64) (*entity.Customer, error)
	GetCustomerByDocument(ctx context.Context, document string) (*entity.Customer, error)
	ListCustomers(ctx context.Context, onlyActive bool) ([]*entity.Customer, error)
	ListEstablishments(ctx context.Context, corporateCode int64) ([]*entity.Customer, error)
	BlockCustomer(ctx context.Context, code int64, reason string) error
	UnblockCustomer(ctx context.Context, code int64) error
	NextCustomerCode(ctx context.Context) (int64, error)

	// ── Customer Addresses ────────────────────────────────────────────────────
	AddAddress(ctx context.Context, addr *entity.CustomerAddress) (*entity.CustomerAddress, error)
	UpdateAddress(ctx context.Context, addr *entity.CustomerAddress) (*entity.CustomerAddress, error)
	ListAddresses(ctx context.Context, customerID int64) ([]*entity.CustomerAddress, error)
	DeleteAddress(ctx context.Context, id int64) error

	// ── Customer Contacts ─────────────────────────────────────────────────────
	AddContact(ctx context.Context, c *entity.CustomerContact) (*entity.CustomerContact, error)
	UpdateContact(ctx context.Context, c *entity.CustomerContact) (*entity.CustomerContact, error)
	ListContacts(ctx context.Context, customerID int64) ([]*entity.CustomerContact, error)
	DeleteContact(ctx context.Context, id int64) error
}
