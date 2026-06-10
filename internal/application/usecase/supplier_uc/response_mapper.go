package supplier_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/supplier/entity"
)

func toSupplierTypeResponse(t *entity.SupplierType) *response.SupplierTypeResponse {
	if t == nil {
		return nil
	}
	return &response.SupplierTypeResponse{
		ID:          t.ID,
		Code:        t.Code,
		Description: t.Description,
		Kind:        string(t.Kind),
		IsActive:    t.IsActive,
		CreatedAt:   t.CreatedAt,
	}
}

func toSupplierTypeResponses(list []*entity.SupplierType) []*response.SupplierTypeResponse {
	out := make([]*response.SupplierTypeResponse, 0, len(list))
	for _, t := range list {
		out = append(out, toSupplierTypeResponse(t))
	}
	return out
}

func toContactTypeResponse(t *entity.SupplierContactType) *response.SupplierContactTypeResponse {
	if t == nil {
		return nil
	}
	return &response.SupplierContactTypeResponse{
		ID:          t.ID,
		Code:        t.Code,
		Description: t.Description,
		IsActive:    t.IsActive,
		CreatedAt:   t.CreatedAt,
	}
}

func toContactTypeResponses(list []*entity.SupplierContactType) []*response.SupplierContactTypeResponse {
	out := make([]*response.SupplierContactTypeResponse, 0, len(list))
	for _, t := range list {
		out = append(out, toContactTypeResponse(t))
	}
	return out
}

func toSupplierResponse(s *entity.Supplier) *response.SupplierResponse {
	if s == nil {
		return nil
	}
	return &response.SupplierResponse{
		ID:                              s.ID,
		Code:                            s.Code,
		CorporateCode:                   s.CorporateCode,
		IsActive:                        s.IsActive,
		IsRepresentative:                s.IsRepresentative,
		IsCustomer:                      s.IsCustomer,
		Name:                            s.Name,
		TradeName:                       s.TradeName,
		PersonType:                      string(s.PersonType),
		DocumentType:                    string(s.DocumentType),
		DocumentNumber:                  s.DocumentNumber,
		StateRegistration:               s.StateRegistration,
		MunicipalRegistration:           s.MunicipalRegistration,
		SupplierTypeID:                  s.SupplierTypeID,
		PaymentConditionID:              s.PaymentConditionID,
		CarrierID:                       s.CarrierID,
		RegionID:                        s.RegionID,
		FreightType:                     string(s.FreightType),
		RegisterDate:                    s.RegisterDate,
		ViticolaObligation:              string(s.ViticolaObligation),
		GLNCode:                         s.GLNCode,
		AgricultureMinistryRegistration: s.AgricultureMinistryRegistration,
		ICMSContributor:                 string(s.ICMSContributor),
		IsMEI:                           s.IsMEI,
		TrackingPlatform:                string(s.TrackingPlatform),
		Homologated:                     s.Homologated,
		LastSefazQuery:                  s.LastSefazQuery,
		BillingReceiptStatus:            s.BillingReceiptStatus,
		LastSefazUpdate:                 s.LastSefazUpdate,
		SefazUpdateUser:                 s.SefazUpdateUser,
		Blocked:                         s.Blocked,
		BlockReason:                     s.BlockReason,
		CreatedAt:                       s.CreatedAt,
		CreatedBy:                       s.CreatedBy,
		UpdatedAt:                       s.UpdatedAt,
		Addresses:                       toAddressValues(s.Addresses),
		Phones:                          toPhoneValues(s.Phones),
		Emails:                          toEmailValues(s.Emails),
		DueDates:                        toDueDateValues(s.DueDates),
		Contacts:                        toContactValues(s.Contacts),
	}
}

func toSupplierResponses(list []*entity.Supplier) []*response.SupplierResponse {
	out := make([]*response.SupplierResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toSupplierResponse(s))
	}
	return out
}

func toAddressResponse(a *entity.SupplierAddress) *response.SupplierAddressResponse {
	if a == nil {
		return nil
	}
	return &response.SupplierAddressResponse{
		ID:           a.ID,
		SupplierID:   a.SupplierID,
		AddressType:  string(a.AddressType),
		ZipCode:      a.ZipCode,
		Street:       a.Street,
		Number:       a.Number,
		Complement:   a.Complement,
		Neighborhood: a.Neighborhood,
		City:         a.City,
		UF:           a.UF,
		Country:      a.Country,
		IsDefault:    a.IsDefault,
		CreatedAt:    a.CreatedAt,
	}
}

func toAddressValues(list []*entity.SupplierAddress) []response.SupplierAddressResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.SupplierAddressResponse, 0, len(list))
	for _, a := range list {
		out = append(out, *toAddressResponse(a))
	}
	return out
}

func toAddressResponses(list []*entity.SupplierAddress) []*response.SupplierAddressResponse {
	out := make([]*response.SupplierAddressResponse, 0, len(list))
	for _, a := range list {
		out = append(out, toAddressResponse(a))
	}
	return out
}

func toPhoneResponse(p *entity.SupplierPhone) *response.SupplierPhoneResponse {
	if p == nil {
		return nil
	}
	return &response.SupplierPhoneResponse{
		ID:         p.ID,
		SupplierID: p.SupplierID,
		Number:     p.Number,
		Ranking:    p.Ranking,
		IsActive:   p.IsActive,
		CreatedAt:  p.CreatedAt,
	}
}

func toPhoneValues(list []*entity.SupplierPhone) []response.SupplierPhoneResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.SupplierPhoneResponse, 0, len(list))
	for _, p := range list {
		out = append(out, *toPhoneResponse(p))
	}
	return out
}

func toEmailResponse(e *entity.SupplierEmail) *response.SupplierEmailResponse {
	if e == nil {
		return nil
	}
	return &response.SupplierEmailResponse{
		ID:         e.ID,
		SupplierID: e.SupplierID,
		Email:      e.Email,
		Ranking:    e.Ranking,
		IsActive:   e.IsActive,
		CreatedAt:  e.CreatedAt,
	}
}

func toEmailValues(list []*entity.SupplierEmail) []response.SupplierEmailResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.SupplierEmailResponse, 0, len(list))
	for _, e := range list {
		out = append(out, *toEmailResponse(e))
	}
	return out
}

func toDueDateResponse(d *entity.SupplierDueDate) *response.SupplierDueDateResponse {
	if d == nil {
		return nil
	}
	return &response.SupplierDueDateResponse{
		ID:                 d.ID,
		SupplierID:         d.SupplierID,
		Description:        d.Description,
		Ranking:            d.Ranking,
		BaseDate:           string(d.BaseDate),
		PaymentConditionID: d.PaymentConditionID,
		PaymentType:        string(d.PaymentType),
		SubsequentMonth:    d.SubsequentMonth,
		Rounding:           string(d.Rounding),
		ReceiptStartTime:   d.ReceiptStartTime,
		ReceiptEndTime:     d.ReceiptEndTime,
		AvgUnloadMinutes:   d.AvgUnloadMinutes,
		CreatedAt:          d.CreatedAt,
	}
}

func toDueDateValues(list []*entity.SupplierDueDate) []response.SupplierDueDateResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.SupplierDueDateResponse, 0, len(list))
	for _, d := range list {
		out = append(out, *toDueDateResponse(d))
	}
	return out
}

func toContactResponse(c *entity.SupplierContact) *response.SupplierContactResponse {
	if c == nil {
		return nil
	}
	return &response.SupplierContactResponse{
		ID:               c.ID,
		SupplierID:       c.SupplierID,
		ContactTypeID:    c.ContactTypeID,
		Name:             c.Name,
		Position:         c.Position,
		Department:       c.Department,
		Ranking:          c.Ranking,
		Observation:      c.Observation,
		PurchaseOrderTag: c.PurchaseOrderTag,
		IsActive:         c.IsActive,
		CreatedAt:        c.CreatedAt,
		Phones:           toContactPhoneValues(c.Phones),
		Emails:           toContactEmailValues(c.Emails),
	}
}

func toContactValues(list []*entity.SupplierContact) []response.SupplierContactResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.SupplierContactResponse, 0, len(list))
	for _, c := range list {
		out = append(out, *toContactResponse(c))
	}
	return out
}

func toContactResponses(list []*entity.SupplierContact) []*response.SupplierContactResponse {
	out := make([]*response.SupplierContactResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toContactResponse(c))
	}
	return out
}

func toContactPhoneResponse(p *entity.SupplierContactPhone) *response.SupplierContactPhoneResponse {
	if p == nil {
		return nil
	}
	return &response.SupplierContactPhoneResponse{
		ID:        p.ID,
		ContactID: p.ContactID,
		Value:     p.Value,
		Ranking:   p.Ranking,
	}
}

func toContactPhoneValues(list []*entity.SupplierContactPhone) []response.SupplierContactPhoneResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.SupplierContactPhoneResponse, 0, len(list))
	for _, p := range list {
		out = append(out, *toContactPhoneResponse(p))
	}
	return out
}

func toContactEmailResponse(e *entity.SupplierContactEmail) *response.SupplierContactEmailResponse {
	if e == nil {
		return nil
	}
	return &response.SupplierContactEmailResponse{
		ID:        e.ID,
		ContactID: e.ContactID,
		Value:     e.Value,
		Ranking:   e.Ranking,
	}
}

func toContactEmailValues(list []*entity.SupplierContactEmail) []response.SupplierContactEmailResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.SupplierContactEmailResponse, 0, len(list))
	for _, e := range list {
		out = append(out, *toContactEmailResponse(e))
	}
	return out
}

func toEnterpriseResponse(e *entity.SupplierEnterprise) *response.SupplierEnterpriseResponse {
	if e == nil {
		return nil
	}
	return &response.SupplierEnterpriseResponse{
		ID:                   e.ID,
		SupplierID:           e.SupplierID,
		EnterpriseCode:       e.EnterpriseCode,
		FinancialAccount:     e.FinancialAccount,
		AppliesIPI:           e.AppliesIPI,
		DefaultInvoiceTypeID: e.DefaultInvoiceTypeID,
		PurchasePriceTableID: e.PurchasePriceTableID,
		IsActive:             e.IsActive,
		CreatedAt:            e.CreatedAt,
	}
}

func toEnterpriseResponses(list []*entity.SupplierEnterprise) []*response.SupplierEnterpriseResponse {
	out := make([]*response.SupplierEnterpriseResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toEnterpriseResponse(e))
	}
	return out
}

func toParametersResponse(p *entity.SupplierParameters) *response.SupplierParametersResponse {
	if p == nil {
		return nil
	}
	return &response.SupplierParametersResponse{
		ID:                        p.ID,
		EnterpriseCode:            p.EnterpriseCode,
		DefaultFinancialAccount:   p.DefaultFinancialAccount,
		UniqueItemCodePerSupplier: p.UniqueItemCodePerSupplier,
		RequiresFinancialAccount:  p.RequiresFinancialAccount,
		PurchaseSupplierTypeID:    p.PurchaseSupplierTypeID,
		CopyObsToPurchaseOrder:    p.CopyObsToPurchaseOrder,
		CopyObsToEntryInvoice:     p.CopyObsToEntryInvoice,
		HomologationDefault:       p.HomologationDefault,
		UseStockUOM:               p.UseStockUOM,
		GenericSupplierCode:       p.GenericSupplierCode,
		DefaultDueBaseDate:        string(p.DefaultDueBaseDate),
		CreatedAt:                 p.CreatedAt,
		UpdatedAt:                 p.UpdatedAt,
	}
}
