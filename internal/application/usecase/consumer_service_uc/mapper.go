package consumer_service_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
)

func toCallTypeResponse(v *entity.CallType) *response.ConsumerServiceCallTypeResponse {
	return &response.ConsumerServiceCallTypeResponse{Code: v.Code, Description: v.Description, IsComplaint: v.IsComplaint, IsActive: v.IsActive, CreatedAt: v.CreatedAt}
}

func toKnowledgeSourceResponse(v *entity.KnowledgeSource) *response.ConsumerServiceKnowledgeSourceResponse {
	return &response.ConsumerServiceKnowledgeSourceResponse{Code: v.Code, Description: v.Description, IsActive: v.IsActive, CreatedAt: v.CreatedAt}
}

func toConsumerResponse(v *entity.Consumer) *response.ConsumerResponse {
	out := &response.ConsumerResponse{
		Code: v.Code, Name: v.Name, IsActive: v.IsActive, PersonType: v.PersonType, CPF: v.CPF, RG: v.RG,
		CNPJ: v.CNPJ, StateRegistration: v.StateRegistration, ZipCode: v.ZipCode, City: v.City, State: v.State,
		Address: v.Address, AddressNumber: v.AddressNumber, Complement: v.Complement, District: v.District,
		MarketSegmentCode: v.MarketSegmentCode, KnowledgeCode: v.KnowledgeCode, Notes: v.Notes, CreatedAt: v.CreatedAt,
	}
	for _, phone := range v.Phones {
		out.Phones = append(out.Phones, response.ConsumerPhoneResponse{Code: phone.Code, PhoneType: phone.PhoneType, Number: phone.Number, IsPrimary: phone.IsPrimary, ContactCode: phone.ContactCode})
	}
	for _, email := range v.Emails {
		out.Emails = append(out.Emails, response.ConsumerEmailResponse{Code: email.Code, Email: email.Email, IsPrimary: email.IsPrimary, ContactCode: email.ContactCode})
	}
	for _, contact := range v.Contacts {
		out.Contacts = append(out.Contacts, response.ConsumerContactResponse{Code: contact.Code, Name: contact.Name, Role: contact.Role, ContactType: contact.ContactType, Notes: contact.Notes})
	}
	return out
}

func toCustomerContactResponse(v *entity.CustomerContactHistory) *response.CustomerContactHistoryResponse {
	return &response.CustomerContactHistoryResponse{
		Code: v.Code, CustomerCode: v.CustomerCode, OpenedAt: v.OpenedAt, ScheduledAt: v.ScheduledAt,
		UserCode: v.UserCode, ContactType: v.ContactType, Description: v.Description, CreatedAt: v.CreatedAt,
	}
}

func toCallResponse(v *entity.Call) *response.ConsumerServiceCallResponse {
	out := &response.ConsumerServiceCallResponse{
		Code: v.Code, CallNumber: v.CallNumber, EnterpriseCode: v.EnterpriseCode, ConsumerCode: v.ConsumerCode,
		CustomerCode: v.CustomerCode, CallTypeCode: v.CallTypeCode, Direction: string(v.Direction),
		InWarranty: v.InWarranty, DefectGroupCode: v.DefectGroupCode, DefectReasonCode: v.DefectReasonCode,
		ResponsibleUserCode: v.ResponsibleUserCode, Position: string(v.Position), Situation: string(v.Situation),
		OpenedAt: v.OpenedAt, ReturnDate: v.ReturnDate, VisitRequestedDate: v.VisitRequestedDate,
		VisitReturnedDate: v.VisitReturnedDate, SaleStoreCode: v.SaleStoreCode, EstablishmentCode: v.EstablishmentCode,
		TechnicianDescription: v.TechnicianDescription, Symptoms: v.Symptoms, ForwardedStoreCode: v.ForwardedStoreCode,
		Subject: v.Subject, Description: v.Description, Solution: v.Solution, ChecklistCode: v.ChecklistCode,
		IsActive: v.IsActive, CreatedAt: v.CreatedAt,
	}
	for _, ret := range v.Returns {
		out.Returns = append(out.Returns, *toCallReturnResponse(ret))
	}
	for _, att := range v.Attachments {
		out.Attachments = append(out.Attachments, *toCallAttachmentResponse(att))
	}
	for _, item := range v.ChecklistItems {
		out.ChecklistItems = append(out.ChecklistItems, *toChecklistItemResponse(item))
	}
	return out
}

func toCallReturnResponse(v *entity.CallReturn) *response.ConsumerServiceCallReturnResponse {
	return &response.ConsumerServiceCallReturnResponse{Code: v.Code, CallCode: v.CallCode, ContactedAt: v.ContactedAt, ContactType: v.ContactType, Description: v.Description, NextReturnAt: v.NextReturnAt, UserCode: v.UserCode}
}

func toCallAttachmentResponse(v *entity.CallAttachment) *response.ConsumerServiceCallAttachmentResponse {
	return &response.ConsumerServiceCallAttachmentResponse{Code: v.Code, CallCode: v.CallCode, FileName: v.FileName, FilePath: v.FilePath, ContentType: v.ContentType, Notes: v.Notes}
}

func toChecklistItemResponse(v *entity.CallChecklistItem) *response.ConsumerServiceChecklistItemResponse {
	return &response.ConsumerServiceChecklistItemResponse{Code: v.Code, CallCode: v.CallCode, Sequence: v.Sequence, Description: v.Description, IsDone: v.IsDone, DoneAt: v.DoneAt, Notes: v.Notes}
}
