package routing_uc

import (
	"context"
	"fmt"
	enumtypes "github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"github.com/google/uuid"
)

type OperationUseCase struct {
	repo  repository.RoutingRepository
	items any
	auth  ports.AuthService
}

func NewOperationUseCase(repo repository.RoutingRepository, deps ...any) *OperationUseCase {
	uc := &OperationUseCase{repo: repo}
	for _, dep := range deps {
		if auth, ok := dep.(ports.AuthService); ok {
			uc.auth = auth
		} else {
			uc.items = dep
		}
	}
	return uc
}

func (uc *OperationUseCase) Create(ctx context.Context, dto request.CreateOperationDTO) (*response.OperationResponse, error) {
	if dto.Name == "" {
		return nil, errorsuc.NewValidationError("informe o nome")
	}
	if !validTimeUnit(dto.TimeUnit) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("unidade de tempo %q inválida: use minuto, hora ou dia", dto.TimeUnit))
	}
	origin := entity.OperationOrigin(dto.Origin)
	if origin == "" {
		origin = entity.OriginInternal
	}
	remittance, err := normalizeThirdPartyRemittance(dto.ThirdPartyRemittance)
	if err != nil {
		return nil, err
	}
	serviceItemCode, err := resolveOptionalItemCode(ctx, uc.items, dto.ServiceItemCode)
	if err != nil {
		return nil, err
	}
	var actor uuid.UUID
	if uc.auth != nil {
		actor, err = uc.auth.UserID(ctx)
		if err != nil {
			return nil, errorsuc.ErrUnauthorized
		}
	}

	code, err := uc.repo.NextOperationCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating operation code: %w", err)
	}

	op, err := entity.NewOperation(code, dto.Name, dto.Description, origin,
		dto.DefaultWorkCenterID, dto.StandardTime, dto.SetupTime, actor)
	if err != nil {
		return nil, err
	}
	applyOperationTime(op, dto.RunTime, dto.LaborTime, dto.RunBaseQty,
		dto.QueueTime, dto.WaitTime, dto.MoveTime, dto.CrewSize, dto.TimeUnit)
	op.SupplierID = dto.SupplierID
	op.ServiceItemCode = serviceItemCode
	op.CostPerUnit = dto.CostPerUnit
	op.LeadTimeDays = dto.LeadTimeDays
	op.ThirdPartyRemittance = remittance

	created, err := uc.repo.CreateOperation(ctx, op)
	if err != nil {
		return nil, err
	}
	return toOperationResponse(created), nil
}

func (uc *OperationUseCase) Update(ctx context.Context, dto request.UpdateOperationDTO) (*response.OperationResponse, error) {
	if !validTimeUnit(dto.TimeUnit) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("unidade de tempo %q inválida: use minuto, hora ou dia", dto.TimeUnit))
	}
	op, err := uc.repo.GetOperationByID(ctx, dto.ID)
	if err != nil {
		return nil, fmt.Errorf("operação não encontrada: %w", err)
	}
	remittance, err := normalizeThirdPartyRemittance(dto.ThirdPartyRemittance)
	if err != nil {
		return nil, err
	}
	serviceItemCode, err := resolveOptionalItemCode(ctx, uc.items, dto.ServiceItemCode)
	if err != nil {
		return nil, err
	}
	nextOrigin := entity.OperationOrigin(dto.Origin)
	if (op.Origin == entity.OriginExternal || op.Origin == entity.OriginThirdPart) && nextOrigin == entity.OriginInternal {
		used, usedErr := uc.repo.OperationUsedInRoutes(ctx, dto.ID)
		if usedErr != nil {
			return nil, usedErr
		}
		if used {
			return nil, errorsuc.NewValidationError("operação externa usada em um roteiro de fabricação não pode virar interna")
		}
	}
	op.Name = dto.Name
	op.Description = dto.Description
	op.Origin = nextOrigin
	op.Situation = entity.OperationSituation(dto.Situation)
	op.DefaultWorkCenterID = dto.DefaultWorkCenterID
	op.StandardTime = dto.StandardTime
	op.SetupTime = dto.SetupTime
	applyOperationTime(op, dto.RunTime, dto.LaborTime, dto.RunBaseQty,
		dto.QueueTime, dto.WaitTime, dto.MoveTime, dto.CrewSize, dto.TimeUnit)
	op.SupplierID = dto.SupplierID
	op.ServiceItemCode = serviceItemCode
	op.CostPerUnit = dto.CostPerUnit
	op.LeadTimeDays = dto.LeadTimeDays
	op.ThirdPartyRemittance = remittance

	updated, err := uc.repo.UpdateOperation(ctx, op)
	if err != nil {
		return nil, err
	}
	return toOperationResponse(updated), nil
}

func normalizeThirdPartyRemittance(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		value = "DEMAND_ITEMS"
	}
	if !map[string]bool{"DEMAND_ITEMS": true, "ORDER_ITEM": true, "GENERIC": true, "NONE": true}[value] {
		// Recusa em português, dizendo o que é aceito — antes era
		// "invalid third_party_remittance" e virava 422 sem pista nenhuma.
		return "", enumtypes.NewInvalidValue(
			"O que remeter ao terceiro", value,
			"DEMAND_ITEMS", "ORDER_ITEM", "GENERIC", "NONE")
	}
	return value, nil
}

func (uc *OperationUseCase) GetByID(ctx context.Context, id int64) (*response.OperationResponse, error) {
	op, err := uc.repo.GetOperationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("operação não encontrada: %w", err)
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
		ID:                   op.ID,
		Code:                 op.Code,
		Name:                 op.Name,
		Description:          op.Description,
		Origin:               string(op.Origin),
		Situation:            string(op.Situation),
		DefaultWorkCenterID:  op.DefaultWorkCenterID,
		StandardTime:         op.StandardTime,
		SetupTime:            op.SetupTime,
		RunTime:              op.RunTime,
		LaborTime:            op.LaborTime,
		RunBaseQty:           op.RunBaseQty,
		QueueTime:            op.QueueTime,
		WaitTime:             op.WaitTime,
		MoveTime:             op.MoveTime,
		CrewSize:             op.CrewSize,
		TimeUnit:             op.TimeUnit,
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          op.CostPerUnit,
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: op.ThirdPartyRemittance,
		IsActive:             op.IsActive,
		CreatedAt:            op.CreatedAt,
	}
}
