package quality_uc

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/lote_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/repository"
	"github.com/shopspring/decimal"
)

type QualityUseCase struct {
	repo repository.QualityRepository
	// Lotes: o registro de qualidade é o lote de inspeção (lote_compat.go).
	Lotes *lote_uc.UseCase
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
		return nil, err
	}
	chars, errCaract := uc.repo.ListCharacteristics(ctx, id)
	if errCaract != nil {
		return nil, fmt.Errorf("lendo as características do plano de inspeção: %w", errCaract)
	}
	return toPlanResponse(plan, chars), nil
}

func (uc *QualityUseCase) ListPlansByItem(ctx context.Context, itemCode int64) ([]*response.InspectionPlanResponse, error) {
	plans, err := uc.repo.ListPlansByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*response.InspectionPlanResponse, 0, len(plans))
	for _, p := range plans {
		chars, errCaract := uc.repo.ListCharacteristics(ctx, p.ID)
		if errCaract != nil {
			return nil, fmt.Errorf("lendo as características do plano de inspeção: %w", errCaract)
		}
		out = append(out, toPlanResponse(p, chars))
	}
	return out, nil
}

func (uc *QualityUseCase) DeactivatePlan(ctx context.Context, id int64) error {
	return uc.repo.DeactivatePlan(ctx, id)
}

// ─── characteristics ──────────────────────────────────────────────────────────

func (uc *QualityUseCase) AddCharacteristic(ctx context.Context, dto request.AddCharacteristicDTO) (*response.CharacteristicResponse, error) {
	if dto.PlanID <= 0 {
		return nil, errorsuc.NewValidationError("informe o plano de inspeção")
	}
	if strings.TrimSpace(dto.Name) == "" {
		return nil, errorsuc.NewValidationError("informe o nome da característica")
	}
	if dto.ToleranceLower != nil && dto.ToleranceUpper != nil && dto.ToleranceLower.GreaterThan(*dto.ToleranceUpper) {
		return nil, errorsuc.NewValidationError("a tolerância inferior não pode ser maior que a superior")
	}
	// Antes o plano inexistente (ou de outra empresa) só era recusado pela
	// chave estrangeira, e o usuário lia a mensagem do banco.
	if _, err := uc.repo.GetPlanByID(ctx, dto.PlanID); err != nil {
		return nil, err
	}
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
	nc.QualityRecordID, nc.LotID = uc.loteDaNC(ctx, dto.QualityRecordID)
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
		return nil, errorsuc.NewNotFoundError("não conformidade não encontrada")
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
		SampleSize:       p.SampleSize.InexactFloat64(),
		AcceptanceLevel:  p.AcceptanceLevel.InexactFloat64(),
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
		Nominal:        decimalPtrToFloat(c.Nominal),
		ToleranceUpper: decimalPtrToFloat(c.ToleranceUpper),
		ToleranceLower: decimalPtrToFloat(c.ToleranceLower),
		Unit:           c.Unit,
		IsCritical:     c.IsCritical,
	}
}

func toNCResponse(nc *entity.NonConformance) *response.NonConformanceResponse {
	r := &response.NonConformanceResponse{
		ID:                nc.ID,
		Code:              nc.Code,
		QualityRecordID:   nc.QualityRecordID,
		ProductionOrderID: nc.ProductionOrderID,
		ItemCode:          nc.ItemCode,
		Lot:               nc.Lot,
		NonConformQty:     nc.NonConformQty.InexactFloat64(),
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

// decimalPtrToFloat converte só na fronteira JSON: a resposta segue numérica,
// como no estoque, enquanto domínio e banco guardam o valor exato.
func decimalPtrToFloat(v *decimal.Decimal) *float64 {
	if v == nil {
		return nil
	}
	f := v.InexactFloat64()
	return &f
}
