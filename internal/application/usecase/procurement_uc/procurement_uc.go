package procurement_uc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/lote_uc"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	procrepo "github.com/FelipePn10/panossoerp/internal/domain/procurement/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type UseCase struct {
	Repo      procrepo.Repository
	StockRepo stockrepo.StockRepository
	Auth      ports.AuthService
	// Transferencias move o material entre almoxarifados numa transação, com a
	// entrada pelo custo da saída. Sem ela as pernas eram gravadas em chamadas
	// separadas e a entrada ia sem preço: o valor liberado da inspeção sumia.
	Transferencias Transferidor
	// Lotes é o lote de inspeção da qualidade: as rotas antigas de inspeção de
	// recebimento são adaptadores sobre ele (lote_compat.go).
	Lotes *lote_uc.UseCase
}

// Transferidor é a transferência transacional do estoque.
type Transferidor interface {
	TransferirEntreAlmoxarifados(ctx context.Context, t stockentity.TransferenciaEstoque) (*stockentity.ResultadoTransferencia, error)
}

func (uc *UseCase) CreateRecord(ctx context.Context, dto request.CreateProcurementRecordDTO) (*response.ProcurementRecordResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, errUsuario := uc.Auth.UserID(ctx)
	if errUsuario != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	recordType := entity.RecordType(dto.RecordType)
	if !validRecordType(recordType) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("tipo de registro %q inválido", dto.RecordType))
	}
	status := entity.RecordStatus(dto.Status)
	if status == "" {
		status = entity.StatusOpen
	}
	if !validStatus(status) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("situação %q inválida", dto.Status))
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
		return nil, errorsuc.NewValidationError(fmt.Sprintf("situação %q inválida", dto.Status))
	}
	rec, err := uc.Repo.UpdateRecordStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}
	return toRecordResponse(rec), nil
}

func (uc *UseCase) CreateSupplierScorecard(ctx context.Context, dto request.CreateSupplierScorecardDTO) (*response.SupplierScorecardResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, errUsuario := uc.Auth.UserID(ctx)
	if errUsuario != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	start, err := time.Parse("2006-01-02", dto.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("início do período inválido: %w", err)
	}
	end, err := time.Parse("2006-01-02", dto.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("fim do período inválido: %w", err)
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
	enterpriseCode, err := uc.Auth.EnterpriseCode(ctx)
	if err != nil {
		return nil, err
	}
	dto.EnterpriseCode = enterpriseCode
	if dto.Basis != "ITEM" && dto.Basis != "CLASSIFICATION" {
		return nil, errorsuc.NewValidationError("a base deve ser item ou classificação")
	}
	if dto.Basis == "ITEM" && dto.ItemCode == nil {
		return nil, errorsuc.NewValidationError("informe o item para uma regra por item")
	}
	if dto.Basis == "CLASSIFICATION" && (dto.ClassificationCode == nil || *dto.ClassificationCode == "") {
		return nil, errorsuc.NewValidationError("informe a classificação para uma regra por classificação")
	}
	if dto.InspectionWarehouseID <= 0 {
		return nil, errorsuc.NewValidationError("depósito de inspeção é obrigatório")
	}
	if len(dto.Steps) == 0 {
		return nil, errorsuc.NewValidationError("informe ao menos uma etapa de inspeção")
	}
	validFrom, err := parseOptionalDate(dto.ValidFrom, time.Now())
	if err != nil {
		return nil, fmt.Errorf("início da vigência inválido: %w", err)
	}
	validTo, err := parseDatePtr(dto.ValidTo)
	if err != nil {
		return nil, fmt.Errorf("fim da vigência inválido: %w", err)
	}
	actor, errUsuario := uc.Auth.UserID(ctx)
	if errUsuario != nil {
		return nil, errorsuc.ErrUnauthorized
	}
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
			return nil, errorsuc.NewValidationError("a sequência da etapa deve ser maior que zero")
		}
		if stepDTO.InspectionName == "" {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("informe o nome da inspeção da etapa %d", stepDTO.Sequence))
		}
		if !validInspectionKind(stepDTO.Kind) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("tipo %q inválido na etapa %d", stepDTO.Kind, stepDTO.Sequence))
		}
		if !validAppointmentMode(stepDTO.AppointmentMode) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("modo de apontamento %q inválido na etapa %d", stepDTO.AppointmentMode, stepDTO.Sequence))
		}
		if stepDTO.SampleQty <= 0 {
			stepDTO.SampleQty = 1
		}
		stepValidTo, err := parseDatePtr(stepDTO.ValidTo)
		if err != nil {
			return nil, fmt.Errorf("fim da vigência da etapa inválido: %w", err)
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
				return nil, errorsuc.NewValidationError(fmt.Sprintf("informe a descrição do atributo da etapa %d", stepDTO.Sequence))
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

func (uc *UseCase) LinkReceivingInspectionQualityReport(ctx context.Context, orderID int64, dto request.LinkReceivingInspectionQualityReportDTO) (*response.ReceivingInspectionQualityReportResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if orderID <= 0 || dto.QualityReportID <= 0 {
		return nil, errorsuc.NewValidationError("informe a ordem de inspeção e o laudo de qualidade")
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	link, err := uc.Repo.LinkReceivingInspectionQualityReport(ctx, enterpriseID, orderID, dto.QualityReportID, actor)
	if err != nil {
		return nil, err
	}
	return toReceivingInspectionQualityReportResponse(link), nil
}

func (uc *UseCase) ListReceivingInspectionQualityReports(ctx context.Context, orderID int64) ([]*response.ReceivingInspectionQualityReportResponse, error) {
	if orderID <= 0 {
		return nil, errorsuc.NewValidationError("informe a ordem de inspeção")
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}
	links, err := uc.Repo.ListReceivingInspectionQualityReports(ctx, enterpriseID, orderID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ReceivingInspectionQualityReportResponse, 0, len(links))
	for _, link := range links {
		out = append(out, toReceivingInspectionQualityReportResponse(link))
	}
	return out, nil
}

func (uc *UseCase) UnlinkReceivingInspectionQualityReport(ctx context.Context, orderID, reportID int64) error {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if orderID <= 0 || reportID <= 0 {
		return errorsuc.NewValidationError("informe a ordem de inspeção e o laudo de qualidade")
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return err
	}
	return uc.Repo.UnlinkReceivingInspectionQualityReport(ctx, enterpriseID, orderID, reportID)
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
		Quantity:       m.Quantity.InexactFloat64(),
		UnitPrice:      m.UnitPrice.InexactFloat64(),
		TotalPrice:     m.TotalPrice.InexactFloat64(),
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

func toReceivingInspectionQualityReportResponse(link *entity.ReceivingInspectionQualityReport) *response.ReceivingInspectionQualityReportResponse {
	return &response.ReceivingInspectionQualityReportResponse{
		QualityReportID: link.QualityReportID, InspectionOrderID: link.InspectionOrderID,
		ItemSupplierID: link.ItemSupplierID, RegisteredOn: link.RegisteredOn, Status: link.Status,
		FileName: link.FileName, ContentType: link.ContentType, Notes: link.Notes,
		LinkedAt: link.LinkedAt, LinkedBy: link.LinkedBy,
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
