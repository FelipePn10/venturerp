package tool_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/tool/repository"
)

type ToolUseCase struct {
	repo repository.ToolRepository
}

func New(repo repository.ToolRepository) *ToolUseCase {
	return &ToolUseCase{repo: repo}
}

func (uc *ToolUseCase) Create(ctx context.Context, dto request.CreateToolDTO) (*response.ToolResponse, error) {
	code, err := uc.repo.NextToolCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating tool code: %w", err)
	}
	t, err := entity.NewTool(code, dto.Name, dto.ToolType, dto.LifeType, dto.LifeLimit, dto.Cost, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateTool(ctx, t)
	if err != nil {
		return nil, err
	}
	return toToolResponse(created), nil
}

func (uc *ToolUseCase) Update(ctx context.Context, dto request.UpdateToolDTO) (*response.ToolResponse, error) {
	t, err := uc.repo.GetTool(ctx, dto.ID)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}
	if dto.LifeType != "" {
		t.LifeType = dto.LifeType
	}
	t.Name = dto.Name
	t.ToolType = dto.ToolType
	t.LifeLimit = dto.LifeLimit
	t.Cost = dto.Cost
	if dto.Status != "" {
		t.Status = dto.Status
	}
	updated, err := uc.repo.UpdateTool(ctx, t)
	if err != nil {
		return nil, err
	}
	return toToolResponse(updated), nil
}

func (uc *ToolUseCase) Get(ctx context.Context, id int64) (*response.ToolResponse, error) {
	t, err := uc.repo.GetTool(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}
	return toToolResponse(t), nil
}

func (uc *ToolUseCase) List(ctx context.Context, onlyActive bool) ([]*response.ToolResponse, error) {
	tools, err := uc.repo.ListTools(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ToolResponse, 0, len(tools))
	for _, t := range tools {
		out = append(out, toToolResponse(t))
	}
	return out, nil
}

func (uc *ToolUseCase) Deactivate(ctx context.Context, id int64) error {
	return uc.repo.DeactivateTool(ctx, id)
}

// ResetLife zeroes the consumed life and reactivates the tool (after replacement).
func (uc *ToolUseCase) ResetLife(ctx context.Context, id int64) (*response.ToolResponse, error) {
	t, err := uc.repo.ResetToolLife(ctx, id)
	if err != nil {
		return nil, err
	}
	return toToolResponse(t), nil
}

func (uc *ToolUseCase) ListNeedingReplacement(ctx context.Context) ([]*response.ToolResponse, error) {
	tools, err := uc.repo.ListToolsNeedingReplacement(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ToolResponse, 0, len(tools))
	for _, t := range tools {
		out = append(out, toToolResponse(t))
	}
	return out, nil
}

// ─── association ─────────────────────────────────────────────────────────────

func (uc *ToolUseCase) AddToOperation(ctx context.Context, dto request.AddRouteOpToolDTO) (*response.RouteOpToolResponse, error) {
	if dto.RouteOperationID <= 0 || dto.ToolID <= 0 {
		return nil, fmt.Errorf("route_operation_id and tool_id are required")
	}
	qty := dto.QtyRequired
	if qty <= 0 {
		qty = 1
	}
	added, err := uc.repo.AddRouteOpTool(ctx, &entity.RouteOpTool{
		RouteOperationID: dto.RouteOperationID,
		ToolID:           dto.ToolID,
		QtyRequired:      qty,
	})
	if err != nil {
		return nil, err
	}
	r := toRouteOpToolResponse(added)
	return &r, nil
}

func (uc *ToolUseCase) RemoveFromOperation(ctx context.Context, id int64) error {
	return uc.repo.RemoveRouteOpTool(ctx, id)
}

func (uc *ToolUseCase) ListByOperation(ctx context.Context, routeOperationID int64) ([]response.RouteOpToolResponse, error) {
	tools, err := uc.repo.ListToolsByRouteOp(ctx, routeOperationID)
	if err != nil {
		return nil, err
	}
	out := make([]response.RouteOpToolResponse, 0, len(tools))
	for _, t := range tools {
		out = append(out, toRouteOpToolResponse(t))
	}
	return out, nil
}

// ─── serials (physical instances of a tool master) ───────────────────────────

func (uc *ToolUseCase) CreateSerial(ctx context.Context, dto request.CreateToolSerialDTO) (*response.ToolSerialResponse, error) {
	if _, err := uc.repo.GetTool(ctx, dto.ToolID); err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}
	s, err := entity.NewToolSerial(dto.ToolID, dto.SerialNumber, dto.Status, dto.Location, dto.Notes, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.CreateToolSerial(ctx, s)
	if err != nil {
		return nil, err
	}
	return toToolSerialResponse(created), nil
}

func (uc *ToolUseCase) UpdateSerial(ctx context.Context, dto request.UpdateToolSerialDTO) (*response.ToolSerialResponse, error) {
	s, err := uc.repo.GetToolSerial(ctx, dto.ID)
	if err != nil {
		return nil, fmt.Errorf("tool serial not found: %w", err)
	}
	if dto.SerialNumber != "" {
		s.SerialNumber = dto.SerialNumber
	}
	if dto.Status != "" {
		s.Status = dto.Status
	}
	s.Location = dto.Location
	s.Notes = dto.Notes
	updated, err := uc.repo.UpdateToolSerial(ctx, s)
	if err != nil {
		return nil, err
	}
	return toToolSerialResponse(updated), nil
}

func (uc *ToolUseCase) GetSerial(ctx context.Context, id int64) (*response.ToolSerialResponse, error) {
	s, err := uc.repo.GetToolSerial(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tool serial not found: %w", err)
	}
	return toToolSerialResponse(s), nil
}

func (uc *ToolUseCase) ListSerials(ctx context.Context, toolID int64, onlyActive bool) ([]*response.ToolSerialResponse, error) {
	serials, err := uc.repo.ListToolSerials(ctx, toolID, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ToolSerialResponse, 0, len(serials))
	for _, s := range serials {
		out = append(out, toToolSerialResponse(s))
	}
	return out, nil
}

func (uc *ToolUseCase) DeactivateSerial(ctx context.Context, id int64) error {
	return uc.repo.DeactivateToolSerial(ctx, id)
}

// ─── mappers ─────────────────────────────────────────────────────────────────

func toToolSerialResponse(s *entity.ToolSerial) *response.ToolSerialResponse {
	return &response.ToolSerialResponse{
		ID:           s.ID,
		ToolID:       s.ToolID,
		SerialNumber: s.SerialNumber,
		Status:       s.Status,
		LifeUsed:     s.LifeUsed,
		Location:     s.Location,
		Notes:        s.Notes,
		IsActive:     s.IsActive,
		Available:    s.Available(),
		CreatedAt:    s.CreatedAt,
		ToolCode:     s.ToolCode,
		ToolName:     s.ToolName,
	}
}

func toToolResponse(t *entity.Tool) *response.ToolResponse {
	return &response.ToolResponse{
		ID:               t.ID,
		Code:             t.Code,
		Name:             t.Name,
		ToolType:         t.ToolType,
		LifeType:         t.LifeType,
		LifeLimit:        t.LifeLimit,
		LifeUsed:         t.LifeUsed,
		RemainingLife:    t.RemainingLife(),
		NeedsReplacement: t.NeedsReplacement(),
		Cost:             t.Cost,
		Status:           t.Status,
		IsActive:         t.IsActive,
		CreatedAt:        t.CreatedAt,
	}
}

func toRouteOpToolResponse(t *entity.RouteOpTool) response.RouteOpToolResponse {
	needs := t.LifeLimit > 0 && t.LifeUsed >= t.LifeLimit
	return response.RouteOpToolResponse{
		ID:               t.ID,
		RouteOperationID: t.RouteOperationID,
		ToolID:           t.ToolID,
		ToolCode:         t.ToolCode,
		ToolName:         t.ToolName,
		QtyRequired:      t.QtyRequired,
		LifeType:         t.LifeType,
		LifeLimit:        t.LifeLimit,
		LifeUsed:         t.LifeUsed,
		NeedsReplacement: needs,
		Status:           t.Status,
	}
}
