package technical_assistance_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	prodentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	prodrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	tarepo "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type UseCase struct {
	Repo             tarepo.Repository
	SalesOrders      orderrepo.SalesOrderRepository
	ProductionOrders prodrepo.ProductionOrderRepository
	Auth             ports.AuthService
}

func (uc *UseCase) tenantID(ctx context.Context) (int64, error) {
	if !uc.Auth.CanManageTechnicalAssistance(ctx) {
		return 0, errorsuc.ErrUnauthorized
	}
	tenantID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil || tenantID <= 0 {
		return 0, errorsuc.ErrUnauthorized
	}
	return tenantID, nil
}

func (uc *UseCase) CreateDefectGroup(ctx context.Context, dto request.CreateTADefectGroupDTO) (*response.TADefectGroupResponse, error) {
	if !uc.Auth.CanManageTechnicalAssistance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Description == "" {
		return nil, errorsuc.NewValidationError("description is required")
	}
	created, err := uc.Repo.CreateDefectGroup(ctx, &entity.DefectGroup{Description: dto.Description, IsActive: true, CreatedBy: dto.CreatedBy})
	if err != nil {
		return nil, err
	}
	return toDefectGroupResponse(created), nil
}

func (uc *UseCase) ListDefectGroups(ctx context.Context, onlyActive bool) ([]*response.TADefectGroupResponse, error) {
	if !uc.Auth.CanManageTechnicalAssistance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rows, err := uc.Repo.ListDefectGroups(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.TADefectGroupResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDefectGroupResponse(row))
	}
	return out, nil
}

func (uc *UseCase) CreateDefectReason(ctx context.Context, dto request.CreateTADefectReasonDTO) (*response.TADefectReasonResponse, error) {
	if !uc.Auth.CanManageTechnicalAssistance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.GroupCode == 0 || dto.Description == "" {
		return nil, errorsuc.NewValidationError("group_code and description are required")
	}
	created, err := uc.Repo.CreateDefectReason(ctx, &entity.DefectReason{
		GroupCode:                dto.GroupCode,
		Description:              dto.Description,
		AllowsComplement:         dto.AllowsComplement,
		GeneratesRevenue:         dto.GeneratesRevenue,
		RequiresReturnNote:       dto.RequiresReturnNote,
		GeneratesSalesOrder:      dto.GeneratesSalesOrder,
		GeneratesProductionOrder: dto.GeneratesProductionOrder,
		IsReplacement:            dto.IsReplacement,
		IsService:                dto.IsService,
		AvailableWeb:             dto.AvailableWeb,
		IsActive:                 true,
		CreatedBy:                dto.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	return toDefectReasonResponse(created), nil
}

func (uc *UseCase) ListDefectReasons(ctx context.Context, groupCode *int64, onlyActive bool) ([]*response.TADefectReasonResponse, error) {
	if !uc.Auth.CanManageTechnicalAssistance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rows, err := uc.Repo.ListDefectReasons(ctx, groupCode, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.TADefectReasonResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDefectReasonResponse(row))
	}
	return out, nil
}

func (uc *UseCase) CreateWarrantyResponsible(ctx context.Context, dto request.CreateTAWarrantyResponsibleDTO) (*response.TAWarrantyResponsibleResponse, error) {
	if !uc.Auth.CanManageTechnicalAssistance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Name == "" || (dto.EmployeeCode == nil && dto.CustomerCode == nil) {
		return nil, errorsuc.NewValidationError("name and employee_code or customer_code are required")
	}
	created, err := uc.Repo.CreateWarrantyResponsible(ctx, &entity.WarrantyResponsible{
		Name: dto.Name, EmployeeCode: dto.EmployeeCode, CustomerCode: dto.CustomerCode,
		Email: dto.Email, Phone: dto.Phone, IsActive: true, CreatedBy: dto.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	return toWarrantyResponsibleResponse(created), nil
}

func (uc *UseCase) ListWarrantyResponsibles(ctx context.Context, onlyActive bool) ([]*response.TAWarrantyResponsibleResponse, error) {
	if !uc.Auth.CanManageTechnicalAssistance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rows, err := uc.Repo.ListWarrantyResponsibles(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.TAWarrantyResponsibleResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toWarrantyResponsibleResponse(row))
	}
	return out, nil
}

func (uc *UseCase) CreateCall(ctx context.Context, dto request.CreateTechnicalAssistanceCallDTO) (*response.TechnicalAssistanceCallResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.CustomerCode == 0 || dto.Subject == "" {
		return nil, errorsuc.NewValidationError("customer_code and subject are required")
	}
	openedAt := time.Now()
	if dto.OpenedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, dto.OpenedAt); err == nil {
			openedAt = parsed
		}
	}
	callNumber, err := uc.Repo.NextCallNumber(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	priority := dto.Priority
	if priority == "" {
		priority = "NORMAL"
	}
	call := &entity.Call{
		CallNumber:              callNumber,
		CustomerCode:            dto.CustomerCode,
		ConsumerName:            dto.ConsumerName,
		ConsumerDocument:        dto.ConsumerDocument,
		TechnicalAssistantCode:  dto.TechnicalAssistantCode,
		WarrantyResponsibleCode: dto.WarrantyResponsibleCode,
		Status:                  entity.CallStatusPending,
		Priority:                priority,
		OpenedAt:                openedAt,
		PromisedDate:            parseDatePtr(dto.PromisedDate),
		Subject:                 dto.Subject,
		Description:             dto.Description,
		ReturnNoteRequired:      dto.ReturnNoteRequired,
		CreatedBy:               dto.CreatedBy,
	}
	created, err := uc.Repo.CreateCall(ctx, tenantID, call)
	if err != nil {
		return nil, err
	}
	for i, item := range dto.Items {
		item.CallCode = created.Code
		if item.Sequence == 0 {
			item.Sequence = i + 1
		}
		if _, err := uc.AddCallItem(ctx, item); err != nil {
			return nil, err
		}
	}
	return uc.GetCall(ctx, created.Code)
}

func (uc *UseCase) AddCallItem(ctx context.Context, dto request.CreateTechnicalAssistanceCallItemDTO) (*response.TechnicalAssistanceCallItemResponse, error) {
	tenantID, tenantErr := uc.tenantID(ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}
	if dto.CallCode == 0 || dto.ItemCode == 0 {
		return nil, errorsuc.NewValidationError("call_code and item_code are required")
	}
	if dto.Quantity <= 0 {
		dto.Quantity = 1
	}
	action := dto.RequestedAction
	if action == "" {
		action = "REPAIR"
	}
	var reason *entity.DefectReason
	var err error
	if dto.DefectReasonCode != nil {
		reason, err = uc.Repo.GetDefectReason(ctx, *dto.DefectReasonCode)
		if err != nil {
			return nil, err
		}
		if reason.AllowsComplement && (dto.DefectComplement == nil || *dto.DefectComplement == "") {
			return nil, errorsuc.NewValidationError("defect_complement is required for this defect reason")
		}
	}
	inWarranty := false
	var warrantyUntil *time.Time
	if dto.PurchaseInvoiceDate != "" && dto.WarrantyDays > 0 {
		purchaseDate := parseDatePtr(dto.PurchaseInvoiceDate)
		if purchaseDate != nil {
			until := purchaseDate.AddDate(0, 0, dto.WarrantyDays)
			warrantyUntil = &until
			inWarranty = !time.Now().After(until)
		}
	}
	generatesRevenue := false
	if reason != nil {
		generatesRevenue = reason.GeneratesRevenue
	}
	created, err := uc.Repo.AddCallItem(ctx, tenantID, &entity.CallItem{
		CallCode:              dto.CallCode,
		Sequence:              dto.Sequence,
		ItemCode:              dto.ItemCode,
		Mask:                  dto.Mask,
		SerialNumber:          dto.SerialNumber,
		Quantity:              dto.Quantity,
		DefectReasonCode:      dto.DefectReasonCode,
		DefectComplement:      dto.DefectComplement,
		PurchaseInvoiceNumber: dto.PurchaseInvoiceNumber,
		PurchaseInvoiceDate:   parseDatePtr(dto.PurchaseInvoiceDate),
		WarrantyDays:          dto.WarrantyDays,
		WarrantyUntil:         warrantyUntil,
		InWarranty:            inWarranty,
		GeneratesRevenue:      generatesRevenue,
		RequestedAction:       action,
		Status:                "OPEN",
		Notes:                 dto.Notes,
	})
	if err != nil {
		return nil, err
	}
	return toCallItemResponse(created), nil
}

func (uc *UseCase) GetCall(ctx context.Context, code int64) (*response.TechnicalAssistanceCallResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	call, err := uc.Repo.GetCall(ctx, tenantID, code)
	if err != nil {
		return nil, err
	}
	return toCallResponse(call), nil
}

func (uc *UseCase) ListCalls(ctx context.Context, filter tarepo.CallFilter) ([]*response.TechnicalAssistanceCallResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := uc.Repo.ListCalls(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*response.TechnicalAssistanceCallResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCallResponse(row))
	}
	return out, nil
}

func (uc *UseCase) AddReturnNote(ctx context.Context, dto request.AddTechnicalAssistanceReturnNoteDTO) (*response.TechnicalAssistanceReturnNoteResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.CallCode == 0 || dto.NoteNumber == "" || dto.EmissionDate == "" {
		return nil, errorsuc.NewValidationError("call_code, note_number and emission_date are required")
	}
	op := dto.OperationType
	if op == "" {
		op = "RETURN"
	}
	created, err := uc.Repo.AddReturnNote(ctx, tenantID, &entity.ReturnNote{
		CallCode: dto.CallCode, NoteNumber: dto.NoteNumber, NoteSeries: dto.NoteSeries,
		EmissionDate: *parseDatePtr(dto.EmissionDate), CustomerCode: dto.CustomerCode,
		OperationType: op, AccessKey: dto.AccessKey, TotalValue: dto.TotalValue, Notes: dto.Notes, CreatedBy: dto.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	return toReturnNoteResponse(created), nil
}

func (uc *UseCase) GenerateOrders(ctx context.Context, dto request.GenerateTechnicalAssistanceOrdersDTO) (*response.TechnicalAssistanceOrderGenerationResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	call, err := uc.Repo.GetCall(ctx, tenantID, dto.CallCode)
	if err != nil {
		return nil, err
	}
	if call.Status == entity.CallStatusCancelled || call.Status == entity.CallStatusClosed {
		return nil, errorsuc.NewValidationError("cancelled or closed calls cannot generate orders")
	}
	out := &response.TechnicalAssistanceOrderGenerationResponse{CallCode: call.Code}
	if uc.needsSalesOrder(ctx, call.Items) {
		salesOrderCode, err := uc.generateSalesOrder(ctx, call, dto)
		if err != nil {
			return nil, err
		}
		call.SalesOrderCode = &salesOrderCode
		out.SalesOrderCode = &salesOrderCode
		out.GeneratedLinks++
	}
	for _, item := range call.Items {
		if !uc.itemNeedsProductionOrder(ctx, item) {
			continue
		}
		prodID, err := uc.generateProductionOrder(ctx, call, item, dto)
		if err != nil {
			return nil, err
		}
		call.ProductionOrderID = &prodID
		out.ProductionOrderID = &prodID
		out.GeneratedLinks++
	}
	call.Status = entity.CallStatusWaitingOrder
	_, _ = uc.Repo.UpdateCall(ctx, tenantID, call)
	return out, nil
}

func (uc *UseCase) UpdateStatus(ctx context.Context, dto request.UpdateTechnicalAssistanceCallStatusDTO) (*response.TechnicalAssistanceCallResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	call, err := uc.Repo.GetCall(ctx, tenantID, dto.Code)
	if err != nil {
		return nil, err
	}
	next := entity.CallStatus(dto.Status)
	now := time.Now()
	if next == entity.CallStatusAttended || next == entity.CallStatusClosed {
		if err := uc.validateCanAttend(ctx, call); err != nil {
			return nil, err
		}
		call.AttendedAt = &now
		if next == entity.CallStatusClosed {
			call.ClosedAt = &now
		}
	}
	call.Status = next
	call.Diagnosis = dto.Diagnosis
	call.Solution = dto.Solution
	call.ServiceInvoiceNumber = dto.ServiceInvoiceNumber
	call.CloseReason = dto.CloseReason
	updated, err := uc.Repo.UpdateCall(ctx, tenantID, call)
	if err != nil {
		return nil, err
	}
	return toCallResponse(updated), nil
}

func (uc *UseCase) Report(ctx context.Context, filter tarepo.ReportFilter) (*response.TechnicalAssistanceReportResponse, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Repo.Report(ctx, tenantID, filter)
}

func (uc *UseCase) generateSalesOrder(ctx context.Context, call *entity.Call, dto request.GenerateTechnicalAssistanceOrdersDTO) (int64, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return 0, err
	}
	orderNumber, err := uc.SalesOrders.NextOrderNumber(ctx, call.EnterpriseCode)
	if err != nil {
		return 0, err
	}
	order := &orderentity.SalesOrder{
		OrderNumber:       orderNumber,
		EnterpriseCode:    call.EnterpriseCode,
		Status:            orderentity.SalesOrderStatusOrder,
		Origin:            orderentity.SalesOrderOriginAssistance,
		EmissionDate:      time.Now(),
		DigitDate:         time.Now(),
		CustomerCode:      &call.CustomerCode,
		SalesDivisionCode: dto.SalesDivisionCode,
		PriceTableCode:    dto.PriceTableCode,
		PaymentTermCode:   dto.PaymentTermCode,
		CurrencyCode:      "BRL",
		Notes:             strPtr(fmt.Sprintf("Pedido de assistência técnica gerado pelo chamado %d", call.CallNumber)),
		CreatedBy:         dto.CreatedBy,
	}
	created, err := uc.SalesOrders.Create(ctx, order)
	if err != nil {
		return 0, err
	}
	for i, item := range call.Items {
		if !uc.itemNeedsSalesOrder(ctx, item) {
			continue
		}
		_, _ = uc.SalesOrders.CreateItem(ctx, &orderentity.SalesOrderItem{
			SalesOrderCode: created.Code,
			Sequence:       i + 1,
			ItemCode:       item.ItemCode,
			Mask:           item.Mask,
			DigitDate:      time.Now(),
			WarehouseCode:  dto.WarehouseCode,
			RequestedQty:   item.Quantity,
			UnitPrice:      0,
			Status:         orderentity.SalesOrderItemStatusOpen,
			Notes:          strPtr("Item gerado por assistência técnica"),
		})
	}
	_, _ = uc.Repo.AddOrderLink(ctx, tenantID, &entity.OrderLink{CallCode: call.Code, GeneratedType: "SALES_ORDER", SalesOrderCode: &created.Code, CreatedBy: dto.CreatedBy})
	return created.Code, nil
}

func (uc *UseCase) generateProductionOrder(ctx context.Context, call *entity.Call, item *entity.CallItem, dto request.GenerateTechnicalAssistanceOrdersDTO) (int64, error) {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return 0, err
	}
	orderNumber, err := uc.ProductionOrders.GetNextOrderNumber(ctx)
	if err != nil {
		orderNumber = 1
	}
	prod, err := uc.ProductionOrders.Create(ctx, &prodentity.ProductionOrder{
		OrderNumber: orderNumber,
		ItemCode:    item.ItemCode,
		Mask:        item.Mask,
		PlannedQty:  item.Quantity,
		Status:      prodentity.StatusOpen,
		Notes:       strPtr(fmt.Sprintf("OFT gerada pelo chamado de assistência técnica %d", call.CallNumber)),
		CreatedBy:   dto.CreatedBy,
		IsActive:    true,
	})
	if err != nil {
		return 0, err
	}
	_, _ = uc.Repo.AddOrderLink(ctx, tenantID, &entity.OrderLink{CallCode: call.Code, CallItemCode: &item.Code, GeneratedType: "PRODUCTION_ORDER", ProductionOrderID: &prod.ID, CreatedBy: dto.CreatedBy})
	return prod.ID, nil
}

func (uc *UseCase) validateCanAttend(ctx context.Context, call *entity.Call) error {
	tenantID, err := uc.tenantID(ctx)
	if err != nil {
		return err
	}
	if uc.needsReturnNote(ctx, call) {
		notes, err := uc.Repo.ListReturnNotes(ctx, tenantID, call.Code)
		if err != nil {
			return err
		}
		if len(notes) == 0 {
			return errorsuc.NewValidationError("return note is required before attending the call")
		}
	}
	if uc.needsSalesOrder(ctx, call.Items) && call.SalesOrderCode == nil {
		return errorsuc.NewValidationError("sales order is required before attending the call")
	}
	for _, item := range call.Items {
		if uc.itemNeedsProductionOrder(ctx, item) && call.ProductionOrderID == nil {
			return errorsuc.NewValidationError("production order is required before attending the call")
		}
	}
	return nil
}

func (uc *UseCase) itemNeedsProductionOrder(ctx context.Context, item *entity.CallItem) bool {
	if item.DefectReasonCode == nil {
		return true
	}
	reason, err := uc.Repo.GetDefectReason(ctx, *item.DefectReasonCode)
	return err != nil || reason.GeneratesProductionOrder
}

func (uc *UseCase) needsSalesOrder(ctx context.Context, items []*entity.CallItem) bool {
	for _, item := range items {
		if uc.itemNeedsSalesOrder(ctx, item) {
			return true
		}
	}
	return false
}

func (uc *UseCase) itemNeedsSalesOrder(ctx context.Context, item *entity.CallItem) bool {
	if item.GeneratesRevenue {
		return true
	}
	if item.DefectReasonCode == nil {
		return false
	}
	reason, err := uc.Repo.GetDefectReason(ctx, *item.DefectReasonCode)
	return err == nil && reason.GeneratesSalesOrder
}

func (uc *UseCase) needsReturnNote(ctx context.Context, call *entity.Call) bool {
	if call.ReturnNoteRequired {
		return true
	}
	for _, item := range call.Items {
		if item.DefectReasonCode == nil {
			continue
		}
		reason, err := uc.Repo.GetDefectReason(ctx, *item.DefectReasonCode)
		if err == nil && reason.RequiresReturnNote {
			return true
		}
	}
	return false
}

func strPtr(s string) *string { return &s }

func parseDatePtr(s string) *time.Time {
	return datetime.ParseDatePtr(&s)
}
