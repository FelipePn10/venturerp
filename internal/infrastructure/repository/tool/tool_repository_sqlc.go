package tool

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/tool/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type ToolRepositorySQLC struct {
	q *sqlc.Queries
}

func New(q *sqlc.Queries) domainrepo.ToolRepository {
	return &ToolRepositorySQLC{q: q}
}

// ─── master ────────────────────────────────────────────────────────────────────

func (r *ToolRepositorySQLC) CreateTool(ctx context.Context, t *entity.Tool) (*entity.Tool, error) {
	row, err := r.q.CreateTool(ctx, sqlc.CreateToolParams{
		Code:      t.Code,
		Name:      t.Name,
		ToolType:  t.ToolType,
		LifeType:  t.LifeType,
		LifeLimit: pgutil.ToPgNumericFromFloat64(t.LifeLimit),
		Cost:      pgutil.ToPgNumericFromFloat64(t.Cost),
		Status:    t.Status,
		CreatedBy: pgutil.ToPgUUID(t.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating tool: %w", err)
	}
	return toolRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) UpdateTool(ctx context.Context, t *entity.Tool) (*entity.Tool, error) {
	row, err := r.q.UpdateTool(ctx, sqlc.UpdateToolParams{
		ID:        t.ID,
		Name:      t.Name,
		ToolType:  t.ToolType,
		LifeType:  t.LifeType,
		LifeLimit: pgutil.ToPgNumericFromFloat64(t.LifeLimit),
		Cost:      pgutil.ToPgNumericFromFloat64(t.Cost),
		Status:    t.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("updating tool: %w", err)
	}
	return toolRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) GetTool(ctx context.Context, id int64) (*entity.Tool, error) {
	row, err := r.q.GetTool(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching tool %d: %w", id, err)
	}
	return toolRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) ListTools(ctx context.Context, onlyActive bool) ([]*entity.Tool, error) {
	rows, err := r.q.ListTools(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing tools: %w", err)
	}
	return toolRowsToEntities(rows), nil
}

func (r *ToolRepositorySQLC) DeactivateTool(ctx context.Context, id int64) error {
	return r.q.DeactivateTool(ctx, id)
}

func (r *ToolRepositorySQLC) NextToolCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextToolCode(ctx)
	return int64(v), err
}

// ─── life ────────────────────────────────────────────────────────────────────

func (r *ToolRepositorySQLC) ConsumeToolLife(ctx context.Context, id int64, amount float64) (*entity.Tool, error) {
	row, err := r.q.ConsumeToolLife(ctx, sqlc.ConsumeToolLifeParams{
		ID:       id,
		LifeUsed: pgutil.ToPgNumericFromFloat64(amount),
	})
	if err != nil {
		return nil, fmt.Errorf("consuming tool life %d: %w", id, err)
	}
	return toolRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) ResetToolLife(ctx context.Context, id int64) (*entity.Tool, error) {
	row, err := r.q.ResetToolLife(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("resetting tool life %d: %w", id, err)
	}
	return toolRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) ListToolsNeedingReplacement(ctx context.Context) ([]*entity.Tool, error) {
	rows, err := r.q.ListToolsNeedingReplacement(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing tools needing replacement: %w", err)
	}
	return toolRowsToEntities(rows), nil
}

// ─── association ─────────────────────────────────────────────────────────────

func (r *ToolRepositorySQLC) AddRouteOpTool(ctx context.Context, rt *entity.RouteOpTool) (*entity.RouteOpTool, error) {
	row, err := r.q.AddRouteOpTool(ctx, sqlc.AddRouteOpToolParams{
		RouteOperationID: rt.RouteOperationID,
		ToolID:           rt.ToolID,
		QtyRequired:      pgutil.ToPgNumericFromFloat64(rt.QtyRequired),
	})
	if err != nil {
		return nil, fmt.Errorf("adding route op tool: %w", err)
	}
	return &entity.RouteOpTool{
		ID:               row.ID,
		RouteOperationID: row.RouteOperationID,
		ToolID:           row.ToolID,
		QtyRequired:      pgutil.FromPgNumericToFloat64(row.QtyRequired),
	}, nil
}

func (r *ToolRepositorySQLC) RemoveRouteOpTool(ctx context.Context, id int64) error {
	return r.q.RemoveRouteOpTool(ctx, id)
}

func (r *ToolRepositorySQLC) ListToolsByRouteOp(ctx context.Context, routeOperationID int64) ([]*entity.RouteOpTool, error) {
	rows, err := r.q.ListToolsByRouteOp(ctx, routeOperationID)
	if err != nil {
		return nil, fmt.Errorf("listing tools for route op %d: %w", routeOperationID, err)
	}
	out := make([]*entity.RouteOpTool, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.RouteOpTool{
			ID:               row.ID,
			RouteOperationID: row.RouteOperationID,
			ToolID:           row.ToolID,
			QtyRequired:      pgutil.FromPgNumericToFloat64(row.QtyRequired),
			ToolCode:         row.ToolCode,
			ToolName:         row.ToolName,
			LifeType:         row.LifeType,
			LifeLimit:        pgutil.FromPgNumericToFloat64(row.LifeLimit),
			LifeUsed:         pgutil.FromPgNumericToFloat64(row.LifeUsed),
			Status:           row.Status,
		})
	}
	return out, nil
}

func (r *ToolRepositorySQLC) ListToolsByRoute(ctx context.Context, routeID int64) ([]*entity.RouteOpTool, error) {
	rows, err := r.q.ListToolsByRoute(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("listing tools for route %d: %w", routeID, err)
	}
	out := make([]*entity.RouteOpTool, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.RouteOpTool{
			ID:               row.ID,
			RouteOperationID: row.RouteOperationID,
			ToolID:           row.ToolID,
			QtyRequired:      pgutil.FromPgNumericToFloat64(row.QtyRequired),
			ToolCode:         row.ToolCode,
			ToolName:         row.ToolName,
			LifeType:         row.LifeType,
			LifeLimit:        pgutil.FromPgNumericToFloat64(row.LifeLimit),
			LifeUsed:         pgutil.FromPgNumericToFloat64(row.LifeUsed),
			Status:           row.Status,
		})
	}
	return out, nil
}

// ─── serials (physical instances) ────────────────────────────────────────────

func (r *ToolRepositorySQLC) CreateToolSerial(ctx context.Context, s *entity.ToolSerial) (*entity.ToolSerial, error) {
	row, err := r.q.CreateToolSerial(ctx, sqlc.CreateToolSerialParams{
		ToolID:       s.ToolID,
		SerialNumber: s.SerialNumber,
		Status:       s.Status,
		Location:     pgutil.ToPgTextFromString(s.Location),
		Notes:        pgutil.ToPgTextFromString(s.Notes),
		CreatedBy:    pgutil.ToPgUUID(s.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating tool serial: %w", err)
	}
	return toolSerialRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) UpdateToolSerial(ctx context.Context, s *entity.ToolSerial) (*entity.ToolSerial, error) {
	row, err := r.q.UpdateToolSerial(ctx, sqlc.UpdateToolSerialParams{
		ID:           s.ID,
		SerialNumber: s.SerialNumber,
		Status:       s.Status,
		Location:     pgutil.ToPgTextFromString(s.Location),
		Notes:        pgutil.ToPgTextFromString(s.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("updating tool serial %d: %w", s.ID, err)
	}
	return toolSerialRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) GetToolSerial(ctx context.Context, id int64) (*entity.ToolSerial, error) {
	row, err := r.q.GetToolSerial(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tool serial %d not found: %w", id, err)
	}
	return toolSerialRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) ListToolSerials(ctx context.Context, toolID int64, onlyActive bool) ([]*entity.ToolSerial, error) {
	rows, err := r.q.ListToolSerials(ctx, sqlc.ListToolSerialsParams{ToolID: toolID, OnlyActive: onlyActive})
	if err != nil {
		return nil, fmt.Errorf("listing serials for tool %d: %w", toolID, err)
	}
	out := make([]*entity.ToolSerial, 0, len(rows))
	for _, row := range rows {
		out = append(out, toolSerialRowToEntity(row))
	}
	return out, nil
}

func (r *ToolRepositorySQLC) DeactivateToolSerial(ctx context.Context, id int64) error {
	if err := r.q.DeactivateToolSerial(ctx, id); err != nil {
		return fmt.Errorf("deactivating tool serial %d: %w", id, err)
	}
	return nil
}

// ─── mappers ─────────────────────────────────────────────────────────────────

func toolSerialRowToEntity(row sqlc.DBToolSerial) *entity.ToolSerial {
	return &entity.ToolSerial{
		ID:           row.ID,
		ToolID:       row.ToolID,
		SerialNumber: row.SerialNumber,
		Status:       row.Status,
		LifeUsed:     pgutil.FromPgNumericToFloat64(row.LifeUsed),
		Location:     pgutil.FromPgText(row.Location),
		Notes:        pgutil.FromPgText(row.Notes),
		IsActive:     row.IsActive,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:    pgutil.FromPgUUID(row.CreatedBy),
	}
}

func toolRowToEntity(row sqlc.Tool) *entity.Tool {
	return &entity.Tool{
		ID:        row.ID,
		Code:      row.Code,
		Name:      row.Name,
		ToolType:  row.ToolType,
		LifeType:  row.LifeType,
		LifeLimit: pgutil.FromPgNumericToFloat64(row.LifeLimit),
		LifeUsed:  pgutil.FromPgNumericToFloat64(row.LifeUsed),
		Cost:      pgutil.FromPgNumericToFloat64(row.Cost),
		Status:    row.Status,
		IsActive:  row.IsActive,
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt: pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
	}
}

func toolRowsToEntities(rows []sqlc.Tool) []*entity.Tool {
	out := make([]*entity.Tool, 0, len(rows))
	for _, row := range rows {
		out = append(out, toolRowToEntity(row))
	}
	return out
}
