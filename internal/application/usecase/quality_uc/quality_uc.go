package quality_uc

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/repository"
)

type QualityUseCase struct {
	repo repository.QualityRepository
}

func New(repo repository.QualityRepository) *QualityUseCase {
	return &QualityUseCase{repo: repo}
}

// ─── inspection plans ─────────────────────────────────────────────────────────

func (uc *QualityUseCase) CreatePlan(ctx context.Context, dto request.CreateInspectionPlanDTO) (*response.InspectionPlanResponse, error) {
	plan, err := entity.NewInspectionPlan(
		dto.ItemCode,
		entity.InspectionPointType(dto.PointType),
		dto.Description,
		dto.SampleSize,
		dto.AcceptanceLevel,
		dto.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	plan.RouteOperationID = dto.RouteOperationID
	plan.Instructions = dto.Instructions

	created, err := uc.repo.CreatePlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("creating plan: %w", err)
	}
	return toPlanResponse(created, nil), nil
}

func (uc *QualityUseCase) GetPlan(ctx context.Context, id int64) (*response.InspectionPlanResponse, error) {
	plan, err := uc.repo.GetPlanByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}
	chars, _ := uc.repo.ListCharacteristics(ctx, id)
	return toPlanResponse(plan, chars), nil
}

func (uc *QualityUseCase) ListPlansByItem(ctx context.Context, itemCode int64) ([]*response.InspectionPlanResponse, error) {
	plans, err := uc.repo.ListPlansByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.InspectionPlanResponse, 0, len(plans))
	for _, p := range plans {
		chars, _ := uc.repo.ListCharacteristics(ctx, p.ID)
		out = append(out, toPlanResponse(p, chars))
	}
	return out, nil
}

func (uc *QualityUseCase) DeactivatePlan(ctx context.Context, id int64) error {
	return uc.repo.DeactivatePlan(ctx, id)
}

// ─── characteristics ──────────────────────────────────────────────────────────

func (uc *QualityUseCase) AddCharacteristic(ctx context.Context, dto request.AddCharacteristicDTO) (*response.CharacteristicResponse, error) {
	c := &entity.InspectionCharacteristic{
		PlanID:         dto.PlanID,
		Name:           dto.Name,
		Nominal:        dto.Nominal,
		ToleranceUpper: dto.ToleranceUpper,
		ToleranceLower: dto.ToleranceLower,
		Unit:           dto.Unit,
		IsCritical:     dto.IsCritical,
	}
	created, err := uc.repo.AddCharacteristic(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("adding characteristic: %w", err)
	}
	return toCharResponse(created), nil
}

// ─── quality records ──────────────────────────────────────────────────────────

func (uc *QualityUseCase) CreateRecord(ctx context.Context, dto request.CreateQualityRecordDTO) (*response.QualityRecordResponse, error) {
	if dto.InspectedQty <= 0 {
		return nil, errorsuc.NewValidationError("a quantidade inspecionada deve ser maior que zero")
	}
	rec := &entity.QualityRecord{
		PlanID:            dto.PlanID,
		ProductionOrderID: dto.ProductionOrderID,
		Lot:               dto.Lot,
		ItemCode:          dto.ItemCode,
		InspectedQty:      dto.InspectedQty,
		ApprovedQty:       dto.ApprovedQty,
		RejectedQty:       dto.RejectedQty,
		Result:            entity.InspectionResult(dto.Result),
		InspectorID:       dto.InspectorID,
		Notes:             dto.Notes,
		CreatedBy:         dto.CreatedBy,
	}
	created, err := uc.repo.CreateRecord(ctx, rec)
	if err != nil {
		return nil, fmt.Errorf("creating quality record: %w", err)
	}

	for _, m := range dto.Measurements {
		_ = uc.repo.AddMeasurement(ctx, &entity.QualityMeasurement{
			RecordID:         created.ID,
			CharacteristicID: m.CharacteristicID,
			MeasuredValue:    m.MeasuredValue,
			IsConformant:     m.IsConformant,
		})
	}

	return toRecordResponse(created), nil
}

func (uc *QualityUseCase) GetRecord(ctx context.Context, id int64) (*response.QualityRecordResponse, error) {
	rec, err := uc.repo.GetRecordByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("record not found: %w", err)
	}
	return toRecordResponse(rec), nil
}

func (uc *QualityUseCase) ListRecordsByOrder(ctx context.Context, orderID int64) ([]*response.QualityRecordResponse, error) {
	recs, err := uc.repo.ListRecordsByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return toRecordSlice(recs), nil
}

func (uc *QualityUseCase) ListRecordsByItem(ctx context.Context, itemCode int64) ([]*response.QualityRecordResponse, error) {
	recs, err := uc.repo.ListRecordsByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toRecordSlice(recs), nil
}

// ─── non-conformances ─────────────────────────────────────────────────────────

func (uc *QualityUseCase) CreateNC(ctx context.Context, dto request.CreateNCDTO) (*response.NonConformanceResponse, error) {
	code, err := uc.repo.NextNCCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating NC code: %w", err)
	}
	nc, err := entity.NewNonConformance(code, dto.ItemCode, dto.Description, dto.NonConformQty,
		entity.NCSeverity(dto.Severity), dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	nc.QualityRecordID = dto.QualityRecordID
	nc.ProductionOrderID = dto.ProductionOrderID
	nc.Lot = dto.Lot

	created, err := uc.repo.CreateNC(ctx, nc)
	if err != nil {
		return nil, fmt.Errorf("creating NC: %w", err)
	}
	return toNCResponse(created), nil
}

func (uc *QualityUseCase) GetNC(ctx context.Context, id int64) (*response.NonConformanceResponse, error) {
	nc, err := uc.repo.GetNCByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("NC not found: %w", err)
	}
	return toNCResponse(nc), nil
}

func (uc *QualityUseCase) ListOpenNCs(ctx context.Context) ([]*response.NonConformanceResponse, error) {
	ncs, err := uc.repo.ListOpenNCs(ctx)
	if err != nil {
		return nil, err
	}
	return toNCSlice(ncs), nil
}

func (uc *QualityUseCase) ListNCsByItem(ctx context.Context, itemCode int64) ([]*response.NonConformanceResponse, error) {
	ncs, err := uc.repo.ListNCsByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toNCSlice(ncs), nil
}

func (uc *QualityUseCase) DispositionNC(ctx context.Context, id int64, dto request.DispositionNCDTO) error {
	return uc.repo.DispositionNC(ctx, id, entity.NCDisposition(dto.Disposition), dto.DisposedBy)
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func toPlanResponse(p *entity.InspectionPlan, chars []*entity.InspectionCharacteristic) *response.InspectionPlanResponse {
	r := &response.InspectionPlanResponse{
		ID:               p.ID,
		ItemCode:         p.ItemCode,
		RouteOperationID: p.RouteOperationID,
		PointType:        string(p.PointType),
		Description:      p.Description,
		SampleSize:       p.SampleSize,
		AcceptanceLevel:  p.AcceptanceLevel,
		Instructions:     p.Instructions,
		IsActive:         p.IsActive,
		CreatedAt:        p.CreatedAt,
		CreatedBy:        p.CreatedBy,
	}
	for _, c := range chars {
		r.Characteristics = append(r.Characteristics, *toCharResponse(c))
	}
	return r
}

func toCharResponse(c *entity.InspectionCharacteristic) *response.CharacteristicResponse {
	return &response.CharacteristicResponse{
		ID:             c.ID,
		PlanID:         c.PlanID,
		Name:           c.Name,
		Nominal:        c.Nominal,
		ToleranceUpper: c.ToleranceUpper,
		ToleranceLower: c.ToleranceLower,
		Unit:           c.Unit,
		IsCritical:     c.IsCritical,
	}
}

func toRecordResponse(r *entity.QualityRecord) *response.QualityRecordResponse {
	return &response.QualityRecordResponse{
		ID:                r.ID,
		PlanID:            r.PlanID,
		ProductionOrderID: r.ProductionOrderID,
		Lot:               r.Lot,
		ItemCode:          r.ItemCode,
		InspectedQty:      r.InspectedQty,
		ApprovedQty:       r.ApprovedQty,
		RejectedQty:       r.RejectedQty,
		Result:            string(r.Result),
		InspectorID:       r.InspectorID,
		InspectedAt:       r.InspectedAt,
		Notes:             r.Notes,
		CreatedAt:         r.CreatedAt,
		CreatedBy:         r.CreatedBy,
	}
}

func toRecordSlice(recs []*entity.QualityRecord) []*response.QualityRecordResponse {
	out := make([]*response.QualityRecordResponse, 0, len(recs))
	for _, r := range recs {
		out = append(out, toRecordResponse(r))
	}
	return out
}

func toNCResponse(nc *entity.NonConformance) *response.NonConformanceResponse {
	r := &response.NonConformanceResponse{
		ID:                nc.ID,
		Code:              nc.Code,
		QualityRecordID:   nc.QualityRecordID,
		ProductionOrderID: nc.ProductionOrderID,
		ItemCode:          nc.ItemCode,
		Lot:               nc.Lot,
		NonConformQty:     nc.NonConformQty,
		Description:       nc.Description,
		Severity:          string(nc.Severity),
		RootCause:         nc.RootCause,
		CorrectiveAction:  nc.CorrectiveAction,
		IsOpen:            nc.IsOpen,
		CreatedAt:         nc.CreatedAt,
		CreatedBy:         nc.CreatedBy,
	}
	if nc.Disposition != nil {
		s := string(*nc.Disposition)
		r.Disposition = &s
	}
	r.DisposedAt = nc.DisposedAt
	r.DisposedBy = nc.DisposedBy
	return r
}

func toNCSlice(ncs []*entity.NonConformance) []*response.NonConformanceResponse {
	out := make([]*response.NonConformanceResponse, 0, len(ncs))
	for _, nc := range ncs {
		out = append(out, toNCResponse(nc))
	}
	return out
}
