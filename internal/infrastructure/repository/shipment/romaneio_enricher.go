package shipment

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	itemsrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	itemvo "github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	supplierrepo "github.com/FelipePn10/panossoerp/internal/domain/supplier/repository"
	romaneioExport "github.com/FelipePn10/panossoerp/internal/infrastructure/export/romaneio"
)

// RomaneioEnricherAdapter fills a romaneio with real company, party, carrier and
// item data drawn from the respective repositories. It is the production
// implementation of shipment_uc.RomaneioEnricher, including the branding (logo +
// brand colour) pulled from the company's fiscal config.
type RomaneioEnricherAdapter struct {
	Fiscal     fiscalrepo.FiscalRepository
	Customers  customerrepo.CustomerRepository
	Suppliers  supplierrepo.SupplierRepository
	Items      itemsrepo.ItemRepository
	Sales      *SalesOrderAdapter
	Purchases  *PurchaseOrderAdapter
	Production *ProductionOrderAdapter
}

var _ shipment_uc.RomaneioEnricher = (*RomaneioEnricherAdapter)(nil)

func (a *RomaneioEnricherAdapter) GetEnterprise(ctx context.Context) (romaneioExport.CompanyInfo, error) {
	cfg, err := a.Fiscal.GetFiscalConfig(ctx)
	if err != nil || cfg == nil {
		return romaneioExport.CompanyInfo{}, err
	}
	return romaneioExport.CompanyInfo{
		Name:     cfg.RazaoSocial,
		CNPJCPF:  maskCNPJ(cfg.CnpjEmpresa),
		IE:       deref(cfg.IEEmpresa),
		Street:   cfg.Logradouro,
		Number:   cfg.Numero,
		District: cfg.Bairro,
		City:     cfg.Municipio,
		UF:       cfg.UFEmpresa,
		CEP:      cfg.CEP,
		Phone:    deref(cfg.Telefone),
	}, nil
}

func (a *RomaneioEnricherAdapter) GetBranding(ctx context.Context) ([]byte, string, error) {
	cfg, err := a.Fiscal.GetFiscalConfig(ctx)
	if err != nil || cfg == nil {
		return nil, "", err
	}
	return cfg.Logo, deref(cfg.BrandColor), nil
}

func (a *RomaneioEnricherAdapter) GetCustomer(ctx context.Context, code int64) (romaneioExport.CompanyInfo, error) {
	c, err := a.Customers.GetCustomerByCode(ctx, code)
	if err != nil || c == nil {
		return romaneioExport.CompanyInfo{}, err
	}
	return romaneioExport.CompanyInfo{
		Name:    c.Name,
		CNPJCPF: maskCNPJ(c.DocumentNumber),
		IE:      deref(c.StateRegistration),
	}, nil
}

func (a *RomaneioEnricherAdapter) GetSupplier(ctx context.Context, code int64) (romaneioExport.CompanyInfo, error) {
	s, err := a.Suppliers.GetSupplierByCode(ctx, code)
	if err != nil || s == nil {
		return romaneioExport.CompanyInfo{}, err
	}
	return romaneioExport.CompanyInfo{
		Name:    s.Name,
		CNPJCPF: maskCNPJ(s.DocumentNumber),
		IE:      deref(s.StateRegistration),
	}, nil
}

func (a *RomaneioEnricherAdapter) GetCarrier(ctx context.Context, code int64) (romaneioExport.CarrierInfo, error) {
	c, err := a.Customers.GetCarrierByCode(ctx, code)
	if err != nil || c == nil {
		return romaneioExport.CarrierInfo{}, err
	}
	return romaneioExport.CarrierInfo{
		Name:        c.Description,
		FreightType: string(c.BillingType),
	}, nil
}

func (a *RomaneioEnricherAdapter) GetItemDetails(ctx context.Context, code int64) (romaneioExport.RomaneioItem, error) {
	vo, err := itemvo.NewItemCode(code)
	if err != nil {
		return romaneioExport.RomaneioItem{}, err
	}
	it, err := a.Items.FindItemByCode(ctx, vo)
	if err != nil || it == nil {
		return romaneioExport.RomaneioItem{}, err
	}
	return romaneioExport.RomaneioItem{
		ItemCode:    code,
		Description: it.PDM.DescriptionTechnique,
		Unit:        string(it.Warehouse.UnitOfMeasurement),
	}, nil
}

func (a *RomaneioEnricherAdapter) GetSalesOrder(ctx context.Context, code int64) (*shipment_uc.SalesOrderHeader, error) {
	return a.Sales.GetByCode(ctx, code)
}

func (a *RomaneioEnricherAdapter) GetPurchaseOrder(ctx context.Context, code int64) (*shipment_uc.PurchaseOrderHeader, error) {
	return a.Purchases.GetByCode(ctx, code)
}

func (a *RomaneioEnricherAdapter) GetProductionOrder(ctx context.Context, code int64) (*shipment_uc.ProductionOrderHeader, error) {
	return a.Production.GetByCode(ctx, code)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// maskCNPJ formats a 14-digit CNPJ as 00.000.000/0000-00; a CPF (11 digits) as
// 000.000.000-00; anything else is returned unchanged.
func maskCNPJ(s string) string {
	d := digits(s)
	switch len(d) {
	case 14:
		return d[0:2] + "." + d[2:5] + "." + d[5:8] + "/" + d[8:12] + "-" + d[12:14]
	case 11:
		return d[0:3] + "." + d[3:6] + "." + d[6:9] + "-" + d[9:11]
	default:
		return s
	}
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
