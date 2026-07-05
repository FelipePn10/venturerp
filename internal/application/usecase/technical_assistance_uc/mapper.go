package technical_assistance_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
)

func toDefectGroupResponse(v *entity.DefectGroup) *response.TADefectGroupResponse {
	return &response.TADefectGroupResponse{Code: v.Code, Description: v.Description, IsActive: v.IsActive, CreatedAt: v.CreatedAt}
}

func toDefectReasonResponse(v *entity.DefectReason) *response.TADefectReasonResponse {
	return &response.TADefectReasonResponse{
		Code: v.Code, GroupCode: v.GroupCode, Description: v.Description, AllowsComplement: v.AllowsComplement,
		GeneratesRevenue: v.GeneratesRevenue, RequiresReturnNote: v.RequiresReturnNote,
		GeneratesSalesOrder: v.GeneratesSalesOrder, GeneratesProductionOrder: v.GeneratesProductionOrder,
		IsReplacement: v.IsReplacement, IsService: v.IsService, AvailableWeb: v.AvailableWeb,
		IsActive: v.IsActive, CreatedAt: v.CreatedAt,
	}
}

func toWarrantyResponsibleResponse(v *entity.WarrantyResponsible) *response.TAWarrantyResponsibleResponse {
	return &response.TAWarrantyResponsibleResponse{
		Code: v.Code, Name: v.Name, EmployeeCode: v.EmployeeCode, CustomerCode: v.CustomerCode,
		Email: v.Email, Phone: v.Phone, IsActive: v.IsActive, CreatedAt: v.CreatedAt,
	}
}

func toCallResponse(v *entity.Call) *response.TechnicalAssistanceCallResponse {
	out := &response.TechnicalAssistanceCallResponse{
		Code: v.Code, CallNumber: v.CallNumber, EnterpriseCode: v.EnterpriseCode, CustomerCode: v.CustomerCode,
		Status: string(v.Status), Priority: v.Priority, OpenedAt: v.OpenedAt, PromisedDate: v.PromisedDate,
		AttendedAt: v.AttendedAt, ClosedAt: v.ClosedAt, Subject: v.Subject, Description: v.Description,
		Diagnosis: v.Diagnosis, Solution: v.Solution, ReturnNoteRequired: v.ReturnNoteRequired,
		SalesOrderCode: v.SalesOrderCode, ProductionOrderID: v.ProductionOrderID,
		ServiceInvoiceNumber: v.ServiceInvoiceNumber, CloseReason: v.CloseReason,
	}
	for _, item := range v.Items {
		out.Items = append(out.Items, *toCallItemResponse(item))
	}
	for _, note := range v.ReturnNotes {
		out.ReturnNotes = append(out.ReturnNotes, *toReturnNoteResponse(note))
	}
	return out
}

func toCallItemResponse(v *entity.CallItem) *response.TechnicalAssistanceCallItemResponse {
	return &response.TechnicalAssistanceCallItemResponse{
		Code: v.Code, Sequence: v.Sequence, ItemCode: v.ItemCode, Mask: v.Mask, Quantity: v.Quantity,
		DefectReasonCode: v.DefectReasonCode, WarrantyUntil: v.WarrantyUntil, InWarranty: v.InWarranty,
		GeneratesRevenue: v.GeneratesRevenue, RequestedAction: v.RequestedAction, Status: v.Status,
	}
}

func toReturnNoteResponse(v *entity.ReturnNote) *response.TechnicalAssistanceReturnNoteResponse {
	return &response.TechnicalAssistanceReturnNoteResponse{
		Code: v.Code, CallCode: v.CallCode, NoteNumber: v.NoteNumber,
		EmissionDate: v.EmissionDate, OperationType: v.OperationType, TotalValue: v.TotalValue,
	}
}
