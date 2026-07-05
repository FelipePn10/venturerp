package representative_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/representative/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/representative/repository"
)

func toTypeResponse(t *entity.RepresentativeType) *response.RepresentativeTypeResponse {
	if t == nil {
		return nil
	}
	return &response.RepresentativeTypeResponse{Code: t.Code, Description: t.Description, IsFree: t.IsFree, IgnoresDirectBilling: t.IgnoresDirectBilling, IsActive: t.IsActive, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}

func toResponse(rep *entity.Representative) *response.RepresentativeResponse {
	if rep == nil {
		return nil
	}
	out := &response.RepresentativeResponse{
		Code: rep.Code, IsCustomer: rep.IsCustomer, CustomerCode: rep.CustomerCode, IsSupplier: rep.IsSupplier, SupplierCode: rep.SupplierCode,
		Name: rep.Name, TradeName: rep.TradeName, TypeCode: rep.TypeCode, CategoryCode: rep.CategoryCode, RegisterDate: rep.RegisterDate,
		CoreNumber: rep.CoreNumber, DocumentNumber: rep.DocumentNumber, PostalCode: rep.PostalCode, City: rep.City, State: rep.State,
		FullAddress: rep.FullAddress, Street: rep.Street, StreetNumber: rep.StreetNumber, Complement: rep.Complement, District: rep.District,
		MainPhone: rep.MainPhone, MainEmail: rep.MainEmail, DeviceQuantity: rep.DeviceQuantity, IsActive: rep.IsActive, Blocked: rep.Blocked,
		BlockReason: rep.BlockReason, CreatedAt: rep.CreatedAt, UpdatedAt: rep.UpdatedAt,
	}
	for _, row := range rep.Enterprises {
		out.Enterprises = append(out.Enterprises, *toEnterpriseResponse(row))
	}
	for _, row := range rep.Accounting {
		out.Accounting = append(out.Accounting, *toAccountingResponse(row))
	}
	for _, row := range rep.Regions {
		out.Regions = append(out.Regions, *toRegionResponse(row))
	}
	for _, row := range rep.Segments {
		out.Segments = append(out.Segments, *toSegmentResponse(row))
	}
	for _, row := range rep.SalesPlans {
		out.SalesPlans = append(out.SalesPlans, *toSalesPlanResponse(row))
	}
	for _, row := range rep.Interests {
		out.Interests = append(out.Interests, *toInterestResponse(row))
	}
	for _, row := range rep.Phones {
		out.Phones = append(out.Phones, *toPhoneResponse(row))
	}
	for _, row := range rep.Emails {
		out.Emails = append(out.Emails, *toEmailResponse(row))
	}
	for _, row := range rep.CorrespondenceAddresses {
		out.CorrespondenceAddresses = append(out.CorrespondenceAddresses, *toCorrespondenceResponse(row))
	}
	for _, row := range rep.Contacts {
		out.Contacts = append(out.Contacts, *toContactResponse(row))
	}
	return out
}

func toEnterpriseResponse(row *entity.RepresentativeEnterprise) *response.RepresentativeEnterpriseResponse {
	return &response.RepresentativeEnterpriseResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, EnterpriseCode: row.EnterpriseCode, EnterpriseName: row.EnterpriseName, CommissionPatternCode: row.CommissionPatternCode, CommissionPct: row.CommissionPct, IsDefault: row.IsDefault, IsActive: row.IsActive, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toAccountingResponse(row *entity.RepresentativeAccounting) *response.RepresentativeAccountingResponse {
	return &response.RepresentativeAccountingResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, EnterpriseCode: row.EnterpriseCode, EventType: row.EventType, DebitAccountCode: row.DebitAccountCode, DebitCostCenterCode: row.DebitCostCenterCode, CreditAccountCode: row.CreditAccountCode, CreditCostCenterCode: row.CreditCostCenterCode, HistoryCode: row.HistoryCode, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toRegionResponse(row *entity.RepresentativeRegion) *response.RepresentativeRegionResponse {
	return &response.RepresentativeRegionResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, EnterpriseCode: row.EnterpriseCode, RegionCode: row.RegionCode, MicroregionCode: row.MicroregionCode, IsActive: row.IsActive, CreatedAt: row.CreatedAt}
}

func toSegmentResponse(row *entity.RepresentativeSegment) *response.RepresentativeSegmentResponse {
	return &response.RepresentativeSegmentResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, EnterpriseCode: row.EnterpriseCode, MicroregionCode: row.MicroregionCode, MarketSegmentCode: row.MarketSegmentCode, IsActive: row.IsActive, CreatedAt: row.CreatedAt}
}

func toSalesPlanResponse(row *entity.RepresentativeSalesPlan) *response.RepresentativeSalesPlanResponse {
	return &response.RepresentativeSalesPlanResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, EnterpriseCode: row.EnterpriseCode, MicroregionCode: row.MicroregionCode, SalesPlanCode: row.SalesPlanCode, IsActive: row.IsActive, CreatedAt: row.CreatedAt}
}

func toInterestResponse(row *entity.RepresentativeInterest) *response.RepresentativeInterestResponse {
	return &response.RepresentativeInterestResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, ItemClassificationCode: row.ItemClassificationCode, IsActive: row.IsActive, CreatedAt: row.CreatedAt}
}

func toPhoneResponse(row *entity.RepresentativePhone) *response.RepresentativePhoneResponse {
	return &response.RepresentativePhoneResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, DDI: row.DDI, DDD: row.DDD, Phone: row.Phone, PhoneType: row.PhoneType, Ranking: row.Ranking, CreatedAt: row.CreatedAt}
}

func toEmailResponse(row *entity.RepresentativeEmail) *response.RepresentativeEmailResponse {
	return &response.RepresentativeEmailResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, Email: row.Email, Ranking: row.Ranking, CreatedAt: row.CreatedAt}
}

func toCorrespondenceResponse(row *entity.RepresentativeCorrespondenceAddress) *response.RepresentativeCorrespondenceAddressResponse {
	return &response.RepresentativeCorrespondenceAddressResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, PostalCode: row.PostalCode, City: row.City, State: row.State, FullAddress: row.FullAddress, Street: row.Street, StreetNumber: row.StreetNumber, Complement: row.Complement, District: row.District, IsDefault: row.IsDefault, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toContactResponse(row *entity.RepresentativeContact) *response.RepresentativeContactResponse {
	return &response.RepresentativeContactResponse{ID: row.ID, RepresentativeCode: row.RepresentativeCode, ContactTypeCode: row.ContactTypeCode, Name: row.Name, Role: row.Role, Phone: row.Phone, Email: row.Email, Notes: row.Notes, IsActive: row.IsActive, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toReportRowResponse(row repository.RepresentativeReportRow) response.RepresentativeReportRowResponse {
	return response.RepresentativeReportRowResponse{Code: row.Code, Name: row.Name, TradeName: row.TradeName, TypeCode: row.TypeCode, TypeDescription: row.TypeDescription, State: row.State, City: row.City, MainPhone: row.MainPhone, MainEmail: row.MainEmail, RegionCodes: row.RegionCodes, IsActive: row.IsActive, CommissionPct: row.CommissionPct, DebitAccountCode: row.DebitAccountCode, CreditAccountCode: row.CreditAccountCode, GeneratedHistoryCode: row.GeneratedHistoryCode}
}

func toFollowUpResponse(row repository.RepresentativeFollowUp) response.RepresentativeFollowUpResponse {
	out := response.RepresentativeFollowUpResponse{RepresentativeCode: row.RepresentativeCode, RepresentativeName: row.RepresentativeName, CustomerCount: row.CustomerCount, QuotationCount: row.QuotationCount, OrderCount: row.OrderCount, TotalQuoted: row.TotalQuoted, TotalOrdered: row.TotalOrdered, AverageTicket: row.AverageTicket, CommissionBase: row.CommissionBase, CommissionValue: row.CommissionValue, LastQuotationDate: row.LastQuotationDate, LastOrderDate: row.LastOrderDate}
	for _, customer := range row.Customers {
		out.Customers = append(out.Customers, response.RepresentativeCustomerFollowUpResponse(customer))
	}
	return out
}
