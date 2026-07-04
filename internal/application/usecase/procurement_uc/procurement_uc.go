package procurement_uc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	procrepo "github.com/FelipePn10/panossoerp/internal/domain/procurement/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/google/uuid"
)

type UseCase struct {
	Repo      procrepo.Repository
	StockRepo stockrepo.StockRepository
	Auth      ports.AuthService
}

func (uc *UseCase) CreateRecord(ctx context.Context, dto request.CreateProcurementRecordDTO) (*response.ProcurementRecordResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, _ := uc.Auth.UserID(ctx)
	recordType := entity.RecordType(dto.RecordType)
	if !validRecordType(recordType) {
		return nil, fmt.Errorf("invalid record_type %q", dto.RecordType)
	}
	status := entity.RecordStatus(dto.Status)
	if status == "" {
		status = entity.StatusOpen
	}
	if !validStatus(status) {
		return nil, fmt.Errorf("invalid status %q", dto.Status)
	}
	payload := dto.Payload
	if len(payload) == 0 || !json.Valid(payload) {
		payload = json.RawMessage(`{}`)
	}
	rec := &entity.Record{
		RecordType:            recordType,
		Status:                status,
		SupplierCode:          dto.SupplierCode,
		PurchaseOrderCode:     dto.PurchaseOrderCode,
		PurchaseOrderItemCode: dto.PurchaseOrderItemCode,
		ItemCode:              dto.ItemCode,
		Mask:                  dto.Mask,
		WarehouseID:           dto.WarehouseID,
		Quantity:              dto.Quantity,
		Reference:             dto.Reference,
		Payload:               payload,
		CreatedBy:             &actor,
	}
	created, err := uc.Repo.CreateRecord(ctx, rec)
	if err != nil {
		return nil, err
	}
	return toRecordResponse(created), nil
}

func (uc *UseCase) GetRecord(ctx context.Context, id int64) (*response.ProcurementRecordResponse, error) {
	rec, err := uc.Repo.GetRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	return toRecordResponse(rec), nil
}

func (uc *UseCase) ListRecords(ctx context.Context, recordType, status string) ([]*response.ProcurementRecordResponse, error) {
	records, err := uc.Repo.ListRecords(ctx, recordType, status)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ProcurementRecordResponse, 0, len(records))
	for _, rec := range records {
		out = append(out, toRecordResponse(rec))
	}
	return out, nil
}

func (uc *UseCase) UpdateStatus(ctx context.Context, id int64, dto request.UpdateProcurementRecordStatusDTO) (*response.ProcurementRecordResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	status := entity.RecordStatus(dto.Status)
	if !validStatus(status) {
		return nil, fmt.Errorf("invalid status %q", dto.Status)
	}
	rec, err := uc.Repo.UpdateRecordStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}
	return toRecordResponse(rec), nil
}

func (uc *UseCase) DisposeInspection(ctx context.Context, id int64, dto request.DisposeReceivingInspectionDTO) (*response.ReceivingInspectionDispositionResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) || !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ApprovedQty < 0 || dto.RejectedQty < 0 || dto.ApprovedQty+dto.RejectedQty <= 0 {
		return nil, fmt.Errorf("approved_qty or rejected_qty must be positive")
	}
	actor, _ := uc.Auth.UserID(ctx)
	rec, err := uc.Repo.GetRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec.RecordType != entity.RecordReceivingInspection {
		return nil, fmt.Errorf("record %d is not a receiving inspection", id)
	}
	if rec.ItemCode == nil || rec.WarehouseID == nil {
		return nil, fmt.Errorf("inspection requires item_code and warehouse_id")
	}
	if dto.ApprovedQty > 0 && dto.DestinationWarehouseID == nil {
		return nil, fmt.Errorf("destination_warehouse_id is required when approved_qty is positive")
	}
	if dto.ApprovedQty+dto.RejectedQty > rec.Quantity+0.0001 {
		return nil, fmt.Errorf("disposition quantity exceeds inspected quantity")
	}

	disp := &entity.InspectionDisposition{
		RecordID:               id,
		ApprovedQty:            dto.ApprovedQty,
		RejectedQty:            dto.RejectedQty,
		QuarantineWarehouseID:  dto.QuarantineWarehouseID,
		DestinationWarehouseID: dto.DestinationWarehouseID,
		Reason:                 dto.Reason,
		DisposedBy:             &actor,
	}
	created, err := uc.Repo.CreateInspectionDisposition(ctx, disp)
	if err != nil {
		return nil, err
	}

	movements := make([]response.StockMovementResponse, 0, 2)
	refType := "RECEIVING_INSPECTION"
	refCode := id
	if dto.ApprovedQty > 0 {
		out, in, err := uc.transfer(ctx, rec, *rec.WarehouseID, *dto.DestinationWarehouseID, dto.ApprovedQty, refType, refCode, actor, dto.Reason)
		if err != nil {
			return nil, err
		}
		movements = append(movements, stockMovementResponse(out), stockMovementResponse(in))
	}
	if dto.RejectedQty > 0 && dto.QuarantineWarehouseID != nil && *dto.QuarantineWarehouseID != *rec.WarehouseID {
		out, in, err := uc.transfer(ctx, rec, *rec.WarehouseID, *dto.QuarantineWarehouseID, dto.RejectedQty, refType, refCode, actor, dto.Reason)
		if err != nil {
			return nil, err
		}
		movements = append(movements, stockMovementResponse(out), stockMovementResponse(in))
	}

	nextStatus := entity.StatusApproved
	if dto.ApprovedQty == 0 {
		nextStatus = entity.StatusRejected
	} else if dto.RejectedQty > 0 {
		nextStatus = entity.StatusPartial
	}
	updated, err := uc.Repo.UpdateRecordStatus(ctx, id, nextStatus)
	if err != nil {
		return nil, err
	}
	return &response.ReceivingInspectionDispositionResponse{
		ID:                     created.ID,
		RecordID:               created.RecordID,
		ApprovedQty:            created.ApprovedQty,
		RejectedQty:            created.RejectedQty,
		QuarantineWarehouseID:  created.QuarantineWarehouseID,
		DestinationWarehouseID: created.DestinationWarehouseID,
		Reason:                 created.Reason,
		DisposedAt:             created.DisposedAt,
		DisposedBy:             created.DisposedBy,
		Inspection:             toRecordResponse(updated),
		Movements:              movements,
	}, nil
}

func (uc *UseCase) CreateSupplierScorecard(ctx context.Context, dto request.CreateSupplierScorecardDTO) (*response.SupplierScorecardResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, _ := uc.Auth.UserID(ctx)
	start, err := time.Parse("2006-01-02", dto.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("invalid period_start: %w", err)
	}
	end, err := time.Parse("2006-01-02", dto.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid period_end: %w", err)
	}
	score := &entity.SupplierScorecard{
		SupplierCode:     dto.SupplierCode,
		PeriodStart:      start,
		PeriodEnd:        end,
		QualityScore:     defaultScore(dto.QualityScore),
		DeliveryScore:    defaultScore(dto.DeliveryScore),
		CommercialScore:  defaultScore(dto.CommercialScore),
		ServiceScore:     defaultScore(dto.ServiceScore),
		TotalReceipts:    dto.TotalReceipts,
		RejectedReceipts: dto.RejectedReceipts,
		LateReceipts:     dto.LateReceipts,
		Notes:            dto.Notes,
		CreatedBy:        &actor,
	}
	created, err := uc.Repo.CreateSupplierScorecard(ctx, score)
	if err != nil {
		return nil, err
	}
	return toScorecardResponse(created), nil
}

func (uc *UseCase) ListSupplierScorecards(ctx context.Context, supplierCode int64) ([]*response.SupplierScorecardResponse, error) {
	rows, err := uc.Repo.ListSupplierScorecards(ctx, supplierCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.SupplierScorecardResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScorecardResponse(row))
	}
	return out, nil
}

func (uc *UseCase) CreateReceivingInspectionRoute(ctx context.Context, dto request.CreateReceivingInspectionRouteDTO) (*response.ReceivingInspectionRouteResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.EnterpriseCode == 0 {
		dto.EnterpriseCode = 1
	}
	if dto.Basis != "ITEM" && dto.Basis != "CLASSIFICATION" {
		return nil, fmt.Errorf("basis must be ITEM or CLASSIFICATION")
	}
	if dto.Basis == "ITEM" && dto.ItemCode == nil {
		return nil, fmt.Errorf("item_code is required for ITEM route")
	}
	if dto.Basis == "CLASSIFICATION" && (dto.ClassificationCode == nil || *dto.ClassificationCode == "") {
		return nil, fmt.Errorf("classification_code is required for CLASSIFICATION route")
	}
	if dto.InspectionWarehouseID <= 0 {
		return nil, fmt.Errorf("inspection_warehouse_id is required")
	}
	if len(dto.Steps) == 0 {
		return nil, fmt.Errorf("at least one inspection step is required")
	}
	validFrom, err := parseOptionalDate(dto.ValidFrom, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid valid_from: %w", err)
	}
	validTo, err := parseDatePtr(dto.ValidTo)
	if err != nil {
		return nil, fmt.Errorf("invalid valid_to: %w", err)
	}
	actor, _ := uc.Auth.UserID(ctx)
	route := &entity.ReceivingInspectionRoute{
		EnterpriseCode:        dto.EnterpriseCode,
		Basis:                 dto.Basis,
		ItemCode:              dto.ItemCode,
		ClassificationCode:    dto.ClassificationCode,
		Mask:                  dto.Mask,
		InspectionWarehouseID: dto.InspectionWarehouseID,
		HandlingType:          dto.HandlingType,
		StorageType:           dto.StorageType,
		RouteType:             dto.RouteType,
		MarketType:            dto.MarketType,
		InspectionType:        dto.InspectionType,
		ValidFrom:             validFrom,
		ValidTo:               validTo,
		CreatedBy:             &actor,
	}
	for _, stepDTO := range dto.Steps {
		if stepDTO.Sequence <= 0 {
			return nil, fmt.Errorf("step sequence must be positive")
		}
		if stepDTO.InspectionName == "" {
			return nil, fmt.Errorf("inspection_name is required for step %d", stepDTO.Sequence)
		}
		if !validInspectionKind(stepDTO.Kind) {
			return nil, fmt.Errorf("invalid kind %q for step %d", stepDTO.Kind, stepDTO.Sequence)
		}
		if !validAppointmentMode(stepDTO.AppointmentMode) {
			return nil, fmt.Errorf("invalid appointment_mode %q for step %d", stepDTO.AppointmentMode, stepDTO.Sequence)
		}
		if stepDTO.SampleQty <= 0 {
			stepDTO.SampleQty = 1
		}
		stepValidTo, err := parseDatePtr(stepDTO.ValidTo)
		if err != nil {
			return nil, fmt.Errorf("invalid step valid_to: %w", err)
		}
		step := &entity.ReceivingInspectionRouteStep{
			Sequence:        stepDTO.Sequence,
			InspectionName:  stepDTO.InspectionName,
			Kind:            stepDTO.Kind,
			AppointmentMode: stepDTO.AppointmentMode,
			IsRequired:      stepDTO.IsRequired,
			EmitsLabel:      stepDTO.EmitsLabel,
			InstrumentGroup: stepDTO.InstrumentGroup,
			SampleType:      stepDTO.SampleType,
			SampleUnit:      stepDTO.SampleUnit,
			SampleQty:       stepDTO.SampleQty,
			AcceptanceQty:   stepDTO.AcceptanceQty,
			RejectionQty:    stepDTO.RejectionQty,
			Norm:            stepDTO.Norm,
			Reference:       stepDTO.Reference,
			ValidTo:         stepValidTo,
			NominalValue:    stepDTO.NominalValue,
			MinValue:        stepDTO.MinValue,
			MaxValue:        stepDTO.MaxValue,
		}
		for _, attrDTO := range stepDTO.Attributes {
			if attrDTO.Description == "" {
				return nil, fmt.Errorf("attribute description is required for step %d", stepDTO.Sequence)
			}
			step.Attributes = append(step.Attributes, &entity.ReceivingInspectionStepAttribute{
				Description: attrDTO.Description,
				IsApproved:  attrDTO.IsApproved,
			})
		}
		route.Steps = append(route.Steps, step)
	}
	created, err := uc.Repo.CreateReceivingInspectionRoute(ctx, route)
	if err != nil {
		return nil, err
	}
	return toReceivingInspectionRouteResponse(created), nil
}

func (uc *UseCase) GetReceivingInspectionRoute(ctx context.Context, id int64) (*response.ReceivingInspectionRouteResponse, error) {
	route, err := uc.Repo.GetReceivingInspectionRoute(ctx, id)
	if err != nil {
		return nil, err
	}
	return toReceivingInspectionRouteResponse(route), nil
}

func (uc *UseCase) GenerateReceivingInspectionOrder(ctx context.Context, dto request.GenerateReceivingInspectionOrderDTO) (*response.ReceivingInspectionOrderResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ItemCode <= 0 || dto.WarehouseID <= 0 || dto.Quantity <= 0 {
		return nil, fmt.Errorf("item_code, warehouse_id and quantity are required")
	}
	if dto.Source == "" {
		dto.Source = "MANUAL"
	}
	if !validInspectionSource(dto.Source) {
		return nil, fmt.Errorf("invalid source %q", dto.Source)
	}
	route, err := uc.Repo.FindReceivingInspectionRoute(ctx, 1, dto.ItemCode, dto.Mask, dto.ClassificationCode)
	var routeID *int64
	if err == nil {
		routeID = &route.ID
	}
	actor, _ := uc.Auth.UserID(ctx)
	order := &entity.ReceivingInspectionOrder{
		RouteID:               routeID,
		Source:                dto.Source,
		SupplierCode:          dto.SupplierCode,
		PurchaseOrderCode:     dto.PurchaseOrderCode,
		PurchaseOrderItemCode: dto.PurchaseOrderItemCode,
		FiscalEntryCode:       dto.FiscalEntryCode,
		ReceivingNoticeCode:   dto.ReceivingNoticeCode,
		ItemCode:              dto.ItemCode,
		Mask:                  dto.Mask,
		Lot:                   dto.Lot,
		SerialNumber:          dto.SerialNumber,
		WarehouseID:           dto.WarehouseID,
		Quantity:              dto.Quantity,
		Certificate:           dto.Certificate,
		SupplierNote:          dto.SupplierNote,
		Model:                 dto.Model,
		Notes:                 dto.Notes,
		CreatedBy:             &actor,
	}
	created, err := uc.Repo.CreateReceivingInspectionOrder(ctx, order)
	if err != nil {
		return nil, err
	}
	return toReceivingInspectionOrderResponse(created), nil
}

func (uc *UseCase) ListReceivingInspectionOrders(ctx context.Context, status string) ([]*response.ReceivingInspectionOrderResponse, error) {
	orders, err := uc.Repo.ListReceivingInspectionOrders(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ReceivingInspectionOrderResponse, 0, len(orders))
	for _, order := range orders {
		out = append(out, toReceivingInspectionOrderResponse(order))
	}
	return out, nil
}

func (uc *UseCase) RecordReceivingInspectionResult(ctx context.Context, orderID int64, dto request.RecordReceivingInspectionResultDTO) (*response.ReceivingInspectionResultResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if orderID <= 0 || dto.Sequence <= 0 {
		return nil, fmt.Errorf("order_id and sequence are required")
	}
	if dto.SampleIndex <= 0 {
		dto.SampleIndex = 1
	}
	actor, _ := uc.Auth.UserID(ctx)
	result := &entity.ReceivingInspectionResult{
		OrderID:              orderID,
		StepID:               dto.StepID,
		Sequence:             dto.Sequence,
		SampleIndex:          dto.SampleIndex,
		MeasuredValue:        dto.MeasuredValue,
		MinValue:             dto.MinValue,
		MaxValue:             dto.MaxValue,
		AttributeDescription: dto.AttributeDescription,
		IsApproved:           dto.IsApproved,
		Notes:                dto.Notes,
		CreatedBy:            &actor,
	}
	created, err := uc.Repo.CreateReceivingInspectionResult(ctx, result)
	if err != nil {
		return nil, err
	}
	return toReceivingInspectionResultResponse(created), nil
}

func (uc *UseCase) AnalyzeReceivingInspectionOrder(ctx context.Context, orderID int64, dto request.AnalyzeReceivingInspectionOrderDTO) (*response.ReceivingInspectionAnalysisResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if orderID <= 0 {
		return nil, fmt.Errorf("order_id is required")
	}
	if !validInspectionTreatment(dto.Treatment) {
		return nil, fmt.Errorf("invalid treatment %q", dto.Treatment)
	}
	if dto.ConformQty+dto.RejectedQty+dto.ReworkQty+dto.RestrictedQty <= 0 {
		return nil, fmt.Errorf("analysis quantities must be positive")
	}
	if dto.MoveStock && !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	order, err := uc.Repo.GetReceivingInspectionOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if dto.MoveStock {
		if (dto.ConformQty > 0 || dto.RestrictedQty > 0) && dto.DestinationWarehouseID == nil && dto.RestrictedWarehouseID == nil {
			return nil, fmt.Errorf("destination_warehouse_id is required to release approved quantity")
		}
		if dto.ConformQty+dto.RejectedQty+dto.ReworkQty+dto.RestrictedQty > order.Quantity+0.0001 {
			return nil, fmt.Errorf("analysis quantity exceeds inspected order quantity")
		}
	}
	actor, _ := uc.Auth.UserID(ctx)
	analysis := &entity.ReceivingInspectionAnalysis{
		OrderID:              orderID,
		ConformQty:           dto.ConformQty,
		RejectedQty:          dto.RejectedQty,
		ReworkQty:            dto.ReworkQty,
		RestrictedQty:        dto.RestrictedQty,
		Treatment:            dto.Treatment,
		AffectsSupplierScore: dto.AffectsSupplierScore,
		Notes:                dto.Notes,
		AnalyzedBy:           &actor,
	}
	created, err := uc.Repo.CreateReceivingInspectionAnalysis(ctx, analysis)
	if err != nil {
		return nil, err
	}
	out := toReceivingInspectionAnalysisResponse(created)
	if dto.MoveStock {
		movements, err := uc.releaseInspectionStock(ctx, order, dto, actor)
		if err != nil {
			return nil, err
		}
		out.Movements = movements
	}
	return out, nil
}

// releaseInspectionStock moves analyzed quantities out of the inspection
// warehouse toward available (conform + restricted), rework and rejection
// warehouses. It mirrors the non-transactional movement pattern already used by
// DisposeInspection so quality analysis and physical stock stay aligned.
func (uc *UseCase) releaseInspectionStock(ctx context.Context, order *entity.ReceivingInspectionOrder, dto request.AnalyzeReceivingInspectionOrderDTO, actor uuid.UUID) ([]response.StockMovementResponse, error) {
	refType := "RECEIVING_INSPECTION_ANALYSIS"
	refCode := order.ID
	movements := make([]response.StockMovementResponse, 0, 8)
	for _, l := range planInspectionStockLegs(order.WarehouseID, dto) {
		out, in, err := uc.transferStock(ctx, order.ItemCode, order.Mask, order.WarehouseID, l.To, l.Qty, refType, refCode, actor, dto.Notes)
		if err != nil {
			return nil, err
		}
		movements = append(movements, stockMovementResponse(out), stockMovementResponse(in))
	}
	return movements, nil
}

// inspectionStockLeg is a single quarantine-out transfer computed from an analysis.
type inspectionStockLeg struct {
	Qty float64
	To  int64
}

// planInspectionStockLegs resolves which analyzed quantities move out of the
// inspection warehouse and to where. Restricted (accept-with-restriction) falls
// back to the available destination when no dedicated restricted warehouse is
// given. Zero quantities, missing targets and self-transfers are skipped.
func planInspectionStockLegs(fromWarehouse int64, dto request.AnalyzeReceivingInspectionOrderDTO) []inspectionStockLeg {
	restrictedTo := dto.RestrictedWarehouseID
	if restrictedTo == nil {
		restrictedTo = dto.DestinationWarehouseID
	}
	candidates := []struct {
		qty float64
		to  *int64
	}{
		{dto.ConformQty, dto.DestinationWarehouseID},
		{dto.RestrictedQty, restrictedTo},
		{dto.ReworkQty, dto.ReworkWarehouseID},
		{dto.RejectedQty, dto.RejectionWarehouseID},
	}
	legs := make([]inspectionStockLeg, 0, len(candidates))
	for _, c := range candidates {
		if c.qty <= 0 || c.to == nil || *c.to == fromWarehouse {
			continue
		}
		legs = append(legs, inspectionStockLeg{Qty: c.qty, To: *c.to})
	}
	return legs
}

func (uc *UseCase) transfer(ctx context.Context, rec *entity.Record, from, to int64, qty float64, refType string, refCode int64, actor uuid.UUID, notes *string) (*stockentity.StockMovement, *stockentity.StockMovement, error) {
	return uc.transferStock(ctx, *rec.ItemCode, rec.Mask, from, to, qty, refType, refCode, actor, notes)
}

func (uc *UseCase) transferStock(ctx context.Context, itemCode int64, mask string, from, to int64, qty float64, refType string, refCode int64, actor uuid.UUID, notes *string) (*stockentity.StockMovement, *stockentity.StockMovement, error) {
	out := &stockentity.StockMovement{
		ItemCode:      itemCode,
		Mask:          mask,
		WarehouseID:   from,
		MovementType:  stockentity.MovementTypeTransferOut,
		Quantity:      qty,
		ReferenceType: &refType,
		ReferenceCode: &refCode,
		Notes:         notes,
		CreatedBy:     actor,
	}
	createdOut, err := uc.StockRepo.CreateMovement(ctx, out)
	if err != nil {
		return nil, nil, err
	}
	in := &stockentity.StockMovement{
		ItemCode:      itemCode,
		Mask:          mask,
		WarehouseID:   to,
		MovementType:  stockentity.MovementTypeTransferIn,
		Quantity:      qty,
		ReferenceType: &refType,
		ReferenceCode: &refCode,
		Notes:         notes,
		CreatedBy:     actor,
	}
	createdIn, err := uc.StockRepo.CreateMovement(ctx, in)
	if err != nil {
		return nil, nil, err
	}
	return createdOut, createdIn, nil
}

func validRecordType(t entity.RecordType) bool {
	switch t {
	case entity.RecordReceivingInspection, entity.RecordReceivingNotice, entity.RecordSupplierEvaluation,
		entity.RecordApprovalLimit, entity.RecordSupplierContract, entity.RecordReceivingChecklist,
		entity.RecordReceivingLabel, entity.RecordSupplierEDI, entity.RecordImportProcess:
		return true
	default:
		return false
	}
}

func validStatus(s entity.RecordStatus) bool {
	switch s {
	case entity.StatusDraft, entity.StatusOpen, entity.StatusInReview, entity.StatusApproved,
		entity.StatusRejected, entity.StatusPartial, entity.StatusClosed, entity.StatusCancelled:
		return true
	default:
		return false
	}
}

func validInspectionKind(value string) bool {
	switch value {
	case "VALUE", "ATTRIBUTE", "STRUCTURE":
		return true
	default:
		return false
	}
}

func validAppointmentMode(value string) bool {
	switch value {
	case "ALL_MEASUREMENTS", "SINGLE_INTERVAL", "MULTIPLE_INTERVAL", "STATUS_ONLY":
		return true
	default:
		return false
	}
}

func validInspectionSource(value string) bool {
	switch value {
	case "PURCHASE_RECEIPT", "RECEIVING_NOTICE", "FISCAL_ENTRY", "MANUAL":
		return true
	default:
		return false
	}
}

func validInspectionTreatment(value string) bool {
	switch value {
	case "ACCEPT_WITH_RESTRICTION", "RETURN_TO_SUPPLIER", "SCRAP", "REWORK", "SORTING", "CONCESSION":
		return true
	default:
		return false
	}
}

func parseOptionalDate(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	return time.Parse("2006-01-02", value)
}

func parseDatePtr(value *string) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", *value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func defaultScore(value float64) float64 {
	if value <= 0 {
		return 100
	}
	return value
}

func toRecordResponse(rec *entity.Record) *response.ProcurementRecordResponse {
	return &response.ProcurementRecordResponse{
		ID:                    rec.ID,
		RecordType:            string(rec.RecordType),
		Status:                string(rec.Status),
		SupplierCode:          rec.SupplierCode,
		PurchaseOrderCode:     rec.PurchaseOrderCode,
		PurchaseOrderItemCode: rec.PurchaseOrderItemCode,
		ItemCode:              rec.ItemCode,
		Mask:                  rec.Mask,
		WarehouseID:           rec.WarehouseID,
		Quantity:              rec.Quantity,
		Reference:             rec.Reference,
		Payload:               rec.Payload,
		OpenedAt:              rec.OpenedAt,
		ClosedAt:              rec.ClosedAt,
		CreatedBy:             rec.CreatedBy,
		UpdatedAt:             rec.UpdatedAt,
	}
}

func toScorecardResponse(s *entity.SupplierScorecard) *response.SupplierScorecardResponse {
	return &response.SupplierScorecardResponse{
		ID:               s.ID,
		SupplierCode:     s.SupplierCode,
		PeriodStart:      s.PeriodStart,
		PeriodEnd:        s.PeriodEnd,
		QualityScore:     s.QualityScore,
		DeliveryScore:    s.DeliveryScore,
		CommercialScore:  s.CommercialScore,
		ServiceScore:     s.ServiceScore,
		OverallScore:     s.OverallScore,
		TotalReceipts:    s.TotalReceipts,
		RejectedReceipts: s.RejectedReceipts,
		LateReceipts:     s.LateReceipts,
		Notes:            s.Notes,
		CreatedAt:        s.CreatedAt,
		CreatedBy:        s.CreatedBy,
	}
}

func stockMovementResponse(m *stockentity.StockMovement) response.StockMovementResponse {
	return response.StockMovementResponse{
		ID:             m.ID,
		ItemCode:       m.ItemCode,
		Mask:           m.Mask,
		WarehouseID:    m.WarehouseID,
		MovementType:   m.MovementType,
		Quantity:       m.Quantity,
		UnitPrice:      m.UnitPrice,
		TotalPrice:     m.TotalPrice,
		ReferenceType:  m.ReferenceType,
		ReferenceCode:  m.ReferenceCode,
		Lot:            m.Lot,
		SerialNumber:   m.SerialNumber,
		Batch:          m.Batch,
		ExpirationDate: m.ExpirationDate,
		Notes:          m.Notes,
		CreatedAt:      m.CreatedAt,
		CreatedBy:      m.CreatedBy,
	}
}

func toReceivingInspectionRouteResponse(route *entity.ReceivingInspectionRoute) *response.ReceivingInspectionRouteResponse {
	out := &response.ReceivingInspectionRouteResponse{
		ID:                    route.ID,
		EnterpriseCode:        route.EnterpriseCode,
		Basis:                 route.Basis,
		ItemCode:              route.ItemCode,
		ClassificationCode:    route.ClassificationCode,
		Mask:                  route.Mask,
		InspectionWarehouseID: route.InspectionWarehouseID,
		HandlingType:          route.HandlingType,
		StorageType:           route.StorageType,
		RouteType:             route.RouteType,
		MarketType:            route.MarketType,
		InspectionType:        route.InspectionType,
		ValidFrom:             route.ValidFrom,
		ValidTo:               route.ValidTo,
		IsActive:              route.IsActive,
		CreatedAt:             route.CreatedAt,
		CreatedBy:             route.CreatedBy,
	}
	for _, step := range route.Steps {
		stepOut := response.ReceivingInspectionStepResponse{
			ID:              step.ID,
			RouteID:         step.RouteID,
			Sequence:        step.Sequence,
			InspectionName:  step.InspectionName,
			Kind:            step.Kind,
			AppointmentMode: step.AppointmentMode,
			IsRequired:      step.IsRequired,
			EmitsLabel:      step.EmitsLabel,
			InstrumentGroup: step.InstrumentGroup,
			SampleType:      step.SampleType,
			SampleUnit:      step.SampleUnit,
			SampleQty:       step.SampleQty,
			AcceptanceQty:   step.AcceptanceQty,
			RejectionQty:    step.RejectionQty,
			Norm:            step.Norm,
			Reference:       step.Reference,
			ValidTo:         step.ValidTo,
			NominalValue:    step.NominalValue,
			MinValue:        step.MinValue,
			MaxValue:        step.MaxValue,
		}
		for _, attr := range step.Attributes {
			stepOut.Attributes = append(stepOut.Attributes, response.ReceivingInspectionAttributeResponse{
				ID:          attr.ID,
				StepID:      attr.StepID,
				Description: attr.Description,
				IsApproved:  attr.IsApproved,
			})
		}
		out.Steps = append(out.Steps, stepOut)
	}
	return out
}

func toReceivingInspectionOrderResponse(order *entity.ReceivingInspectionOrder) *response.ReceivingInspectionOrderResponse {
	return &response.ReceivingInspectionOrderResponse{
		ID:                    order.ID,
		OrderNumber:           order.OrderNumber,
		RouteID:               order.RouteID,
		ProcurementRecordID:   order.ProcurementRecordID,
		Source:                order.Source,
		SupplierCode:          order.SupplierCode,
		PurchaseOrderCode:     order.PurchaseOrderCode,
		PurchaseOrderItemCode: order.PurchaseOrderItemCode,
		FiscalEntryCode:       order.FiscalEntryCode,
		ReceivingNoticeCode:   order.ReceivingNoticeCode,
		ItemCode:              order.ItemCode,
		Mask:                  order.Mask,
		Lot:                   order.Lot,
		SerialNumber:          order.SerialNumber,
		WarehouseID:           order.WarehouseID,
		Quantity:              order.Quantity,
		InspectedQty:          order.InspectedQty,
		ApprovedQty:           order.ApprovedQty,
		RejectedQty:           order.RejectedQty,
		ReworkQty:             order.ReworkQty,
		RestrictedQty:         order.RestrictedQty,
		Status:                order.Status,
		Certificate:           order.Certificate,
		SupplierNote:          order.SupplierNote,
		Model:                 order.Model,
		Notes:                 order.Notes,
		CreatedAt:             order.CreatedAt,
		CreatedBy:             order.CreatedBy,
	}
}

func toReceivingInspectionResultResponse(result *entity.ReceivingInspectionResult) *response.ReceivingInspectionResultResponse {
	return &response.ReceivingInspectionResultResponse{
		ID:                   result.ID,
		OrderID:              result.OrderID,
		StepID:               result.StepID,
		Sequence:             result.Sequence,
		SampleIndex:          result.SampleIndex,
		MeasuredValue:        result.MeasuredValue,
		MinValue:             result.MinValue,
		MaxValue:             result.MaxValue,
		AttributeDescription: result.AttributeDescription,
		IsApproved:           result.IsApproved,
		Notes:                result.Notes,
		CreatedAt:            result.CreatedAt,
		CreatedBy:            result.CreatedBy,
	}
}

func toReceivingInspectionAnalysisResponse(analysis *entity.ReceivingInspectionAnalysis) *response.ReceivingInspectionAnalysisResponse {
	return &response.ReceivingInspectionAnalysisResponse{
		ID:                   analysis.ID,
		OrderID:              analysis.OrderID,
		ConformQty:           analysis.ConformQty,
		RejectedQty:          analysis.RejectedQty,
		ReworkQty:            analysis.ReworkQty,
		RestrictedQty:        analysis.RestrictedQty,
		Treatment:            analysis.Treatment,
		AffectsSupplierScore: analysis.AffectsSupplierScore,
		Notes:                analysis.Notes,
		AnalyzedAt:           analysis.AnalyzedAt,
		AnalyzedBy:           analysis.AnalyzedBy,
		Order:                toReceivingInspectionOrderResponse(analysis.Order),
	}
}
