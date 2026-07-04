package routing_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
)

type OperationUseCase struct {
	repo repository.RoutingRepository
}

func NewOperationUseCase(repo repository.RoutingRepository) *OperationUseCase {
	return &OperationUseCase{repo: repo}
}

func (uc *OperationUseCase) Create(ctx context.Context, dto request.CreateOperationDTO) (*response.OperationResponse, error) {
	if dto.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if !validTimeUnit(dto.TimeUnit) {
		return nil, fmt.Errorf("invalid time_unit %q (expected MIN, HORA or DIA)", dto.TimeUnit)
	}
	origin := entity.OperationOrigin(dto.Origin)
	if origin == "" {
		origin = entity.OriginInternal
	}

	code, err := uc.repo.NextOperationCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating operation code: %w", err)
	}

	op, err := entity.NewOperation(code, dto.Name, dto.Description, origin,
		dto.DefaultWorkCenterID, dto.StandardTime, dto.SetupTime, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	applyOperationTime(op, dto.RunTime, dto.LaborTime, dto.RunBaseQty,
		dto.QueueTime, dto.WaitTime, dto.MoveTime, dto.CrewSize, dto.TimeUnit)
	op.SupplierID = dto.SupplierID
	op.ServiceItemCode = dto.ServiceItemCode
	op.CostPerUnit = dto.CostPerUnit
	op.LeadTimeDays = dto.LeadTimeDays

	created, err := uc.repo.CreateOperation(ctx, op)
	if err != nil {
		return nil, err
	}
	return toOperationResponse(created), nil
}

func (uc *OperationUseCase) Update(ctx context.Context, dto request.UpdateOperationDTO) (*response.OperationResponse, error) {
	if !validTimeUnit(dto.TimeUnit) {
		return nil, fmt.Errorf("invalid time_unit %q (expected MIN, HORA or DIA)", dto.TimeUnit)
	}
	op, err := uc.repo.GetOperationByID(ctx, dto.ID)
	if err != nil {
		return nil, fmt.Errorf("operation not found: %w", err)
	}
	op.Name = dto.Name
	op.Description = dto.Description
	op.Origin = entity.OperationOrigin(dto.Origin)
	op.Situation = entity.OperationSituation(dto.Situation)
	op.DefaultWorkCenterID = dto.DefaultWorkCenterID
	op.StandardTime = dto.StandardTime
	op.SetupTime = dto.SetupTime
	applyOperationTime(op, dto.RunTime, dto.LaborTime, dto.RunBaseQty,
		dto.QueueTime, dto.WaitTime, dto.MoveTime, dto.CrewSize, dto.TimeUnit)
	op.SupplierID = dto.SupplierID
	op.ServiceItemCode = dto.ServiceItemCode
	op.CostPerUnit = dto.CostPerUnit
	op.LeadTimeDays = dto.LeadTimeDays

	updated, err := uc.repo.UpdateOperation(ctx, op)
	if err != nil {
		return nil, err
	}
	return toOperationResponse(updated), nil
}

func (uc *OperationUseCase) GetByID(ctx context.Context, id int64) (*response.OperationResponse, error) {
	op, err := uc.repo.GetOperationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("operation not found: %w", err)
	}
	return toOperationResponse(op), nil
}

func (uc *OperationUseCase) List(ctx context.Context, onlyActive bool) ([]*response.OperationResponse, error) {
	ops, err := uc.repo.ListOperations(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.OperationResponse, 0, len(ops))
	for _, op := range ops {
		out = append(out, toOperationResponse(op))
	}
	return out, nil
}

func (uc *OperationUseCase) Deactivate(ctx context.Context, id int64) error {
	return uc.repo.DeactivateOperation(ctx, id)
}

// validTimeUnit reports whether u is an accepted time-unit code (empty ⇒ default).
func validTimeUnit(u string) bool {
	switch u {
	case "", entity.TimeUnitMinute, entity.TimeUnitHour, entity.TimeUnitDay:
		return true
	default:
		return false
	}
}

// applyOperationTime fills the rich time model on an operation, applying sane
// defaults: TimeUnit=HORA, RunBaseQty>=1, CrewSize>=1, and RunTime falling back
// to the legacy StandardTime when not supplied. The legacy StandardTime column is
// kept mirrored to RunTime so consumers that still read it (interim cost roll-up,
// external-operation hours) stay consistent until they migrate to the rich model.
func applyOperationTime(op *entity.Operation, run, labor, baseQty, queue, wait, move, crew float64, unit string) {
	op.RunTime = run
	if op.RunTime == 0 && op.StandardTime > 0 {
		op.RunTime = op.StandardTime
	}
	op.StandardTime = op.RunTime // mirror legacy column
	op.LaborTime = labor
	op.RunBaseQty = baseQty
	if op.RunBaseQty <= 0 {
		op.RunBaseQty = 1
	}
	op.QueueTime = queue
	op.WaitTime = wait
	op.MoveTime = move
	op.CrewSize = crew
	if op.CrewSize <= 0 {
		op.CrewSize = 1
	}
	op.TimeUnit = unit
	if op.TimeUnit == "" {
		op.TimeUnit = entity.TimeUnitHour
	}
}

func toOperationResponse(op *entity.Operation) *response.OperationResponse {
	return &response.OperationResponse{
		ID:                  op.ID,
		Code:                op.Code,
		Name:                op.Name,
		Description:         op.Description,
		Origin:              string(op.Origin),
		Situation:           string(op.Situation),
		DefaultWorkCenterID: op.DefaultWorkCenterID,
		StandardTime:        op.StandardTime,
		SetupTime:           op.SetupTime,
		RunTime:             op.RunTime,
		LaborTime:           op.LaborTime,
		RunBaseQty:          op.RunBaseQty,
		QueueTime:           op.QueueTime,
		WaitTime:            op.WaitTime,
		MoveTime:            op.MoveTime,
		CrewSize:            op.CrewSize,
		TimeUnit:            op.TimeUnit,
		SupplierID:          op.SupplierID,
		ServiceItemCode:     op.ServiceItemCode,
		CostPerUnit:         op.CostPerUnit,
		LeadTimeDays:        op.LeadTimeDays,
		IsActive:            op.IsActive,
		CreatedAt:           op.CreatedAt,
	}
}
