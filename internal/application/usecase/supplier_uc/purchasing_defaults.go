package supplier_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

// GetPurchasingDefaults implements ports.SupplierPurchasingDefaultsProvider.
// It resolves the supplier-derived defaults used when creating a purchase order
// or an entry NF. enterpriseCode == 0 skips the per-enterprise binding fields.
func (uc *SupplierUseCase) GetPurchasingDefaults(ctx context.Context, supplierCode, enterpriseCode int64) (*ports.SupplierPurchasingDefaults, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, supplierCode)
	if err != nil {
		return nil, err
	}

	out := &ports.SupplierPurchasingDefaults{
		SupplierCode:       s.Code,
		SupplierName:       s.Name,
		IsActive:           s.IsActive,
		PaymentConditionID: s.PaymentConditionID,
		FreightType:        string(s.FreightType),
		ICMSContributor:    string(s.ICMSContributor),
		StateRegistration:  s.StateRegistration,
	}

	// Fallback for payment condition: lowest-ranking "vencimento" carrying one.
	if out.PaymentConditionID == nil {
		if dueDates, err := uc.repo.ListDueDates(ctx, s.ID); err == nil {
			best := int32(1 << 30)
			for _, d := range dueDates {
				if d.PaymentConditionID != nil && d.Ranking <= best {
					best = d.Ranking
					out.PaymentConditionID = d.PaymentConditionID
				}
			}
		}
	}

	// Per-enterprise binding (pasta Empresas).
	if enterpriseCode != 0 {
		if links, err := uc.repo.ListEnterprises(ctx, s.ID); err == nil {
			for _, l := range links {
				if l.EnterpriseCode == enterpriseCode && l.IsActive {
					out.FinancialAccount = l.FinancialAccount
					out.DefaultInvoiceTypeID = l.DefaultInvoiceTypeID
					out.PurchasePriceTableID = l.PurchasePriceTableID
					out.AppliesIPI = l.AppliesIPI
					break
				}
			}
		}
	}

	return out, nil
}

// FindSupplierCodeByDocument implements ports.SupplierPurchasingDefaultsProvider.
// It normalises the document to digits and looks up a supplier. A missing match
// is reported as (0, false, nil) — callers treat it as best-effort.
func (uc *SupplierUseCase) FindSupplierCodeByDocument(ctx context.Context, document string) (int64, bool, error) {
	digits := onlyDigits(document)
	if digits == "" {
		return 0, false, nil
	}
	s, err := uc.repo.GetSupplierByDocument(ctx, digits)
	if err != nil || s == nil {
		return 0, false, nil
	}
	return s.Code, true, nil
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
