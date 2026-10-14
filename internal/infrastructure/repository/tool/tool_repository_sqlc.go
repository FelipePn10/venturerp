package tool

import (
	"errors"

	"context"
	"fmt"
	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/tool/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

type ToolRepositorySQLC struct {
	q *sqlc.Queries
}

func New(q *sqlc.Queries) domainrepo.ToolRepository {
	return &ToolRepositorySQLC{q: q}
}

// Ferramenta é da empresa da sessão (tools.enterprise_id); série segue a
// ferramenta; vínculo com operação segue o roteiro (manufacturing_routes).
// Registro de outra empresa responde como inexistente.

func (r *ToolRepositorySQLC) posse(ctx context.Context, check func(context.Context, int64, int64) (bool, error), id int64, oque string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	ok, err := check(ctx, id, empresa)
	if err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewNotFoundError(fmt.Sprintf("%s %d não encontrado(a)", oque, id))
	}
	return nil
}

func (r *ToolRepositorySQLC) ferramenta(ctx context.Context, id int64) error {
	return r.posse(ctx, r.q.ToolBelongsToEnterprise, id, "ferramenta")
}

func (r *ToolRepositorySQLC) serie(ctx context.Context, id int64) error {
	return r.posse(ctx, r.q.ToolSerialBelongsToEnterprise, id, "número de série")
}

// ─── master ────────────────────────────────────────────────────────────────────

func (r *ToolRepositorySQLC) CreateTool(ctx context.Context, t *entity.Tool) (*entity.Tool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateTool(ctx, sqlc.CreateToolParams{
		EnterpriseID: empresa,
		Code:         t.Code,
		Name:         t.Name,
		ToolType:     t.ToolType,
		LifeType:     t.LifeType,
		LifeLimit:    pgutil.ToPgNumericFromFloat64(t.LifeLimit),
		Cost:         pgutil.ToPgNumericFromFloat64(t.Cost),
		Status:       t.Status,
		CreatedBy:    pgutil.ToPgUUID(t.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating tool: %w", err)
	}
	return toolRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) UpdateTool(ctx context.Context, t *entity.Tool) (*entity.Tool, error) {
	if err := r.ferramenta(ctx, t.ID); err != nil {
		return nil, err
	}
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
	if err := r.ferramenta(ctx, id); err != nil {
		return nil, err
	}
	row, err := r.q.GetTool(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching tool %d: %w", id, err)
	}
	return toolRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) ListTools(ctx context.Context, onlyActive bool) ([]*entity.Tool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListTools(ctx, sqlc.ListToolsParams{Column1: onlyActive, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("listing tools: %w", err)
	}
	return toolRowsToEntities(rows), nil
}

func (r *ToolRepositorySQLC) DeactivateTool(ctx context.Context, id int64) error {
	if err := r.ferramenta(ctx, id); err != nil {
		return err
	}
	return r.q.DeactivateTool(ctx, id)
}

func (r *ToolRepositorySQLC) NextToolCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextToolCode(ctx)
	return int64(v), err
}

// ─── life ────────────────────────────────────────────────────────────────────

func (r *ToolRepositorySQLC) ConsumeToolLife(ctx context.Context, id int64, amount float64) (*entity.Tool, error) {
	if err := r.ferramenta(ctx, id); err != nil {
		return nil, err
	}
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
	if err := r.ferramenta(ctx, id); err != nil {
		return nil, err
	}
	row, err := r.q.ResetToolLife(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("resetting tool life %d: %w", id, err)
	}
	return toolRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) ListToolsNeedingReplacement(ctx context.Context, threshold float64) ([]*entity.Tool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListToolsNeedingReplacement(ctx, sqlc.ListToolsNeedingReplacementParams{Threshold: pgutil.ToPgNumericFromFloat64(threshold), EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("listing tools needing replacement: %w", err)
	}
	return toolRowsToEntities(rows), nil
}

// ─── association ─────────────────────────────────────────────────────────────

func (r *ToolRepositorySQLC) AddRouteOpTool(ctx context.Context, rt *entity.RouteOpTool) (*entity.RouteOpTool, error) {
	if err := r.ferramenta(ctx, rt.ToolID); err != nil {
		return nil, err
	}
	if err := r.posse(ctx, r.q.RouteOperationBelongsToEnterprise, rt.RouteOperationID, "operação do roteiro"); err != nil {
		return nil, err
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.AddRouteOpTool(ctx, sqlc.AddRouteOpToolParams{
		EnterpriseID:     empresa,
		RouteOperationID: rt.RouteOperationID,
		ToolID:           rt.ToolID,
		QtyRequired:      pgutil.ToPgNumericFromFloat64(rt.QtyRequired),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("etapa do roteiro ou ferramenta não encontrada")
		}
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
	if err := r.posse(ctx, r.q.RouteOpToolBelongsToEnterprise, id, "ferramenta da operação"); err != nil {
		return err
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	n, err := r.q.RemoveRouteOpTool(ctx, sqlc.RemoveRouteOpToolParams{ID: id, EnterpriseID: empresa})
	if err != nil {
		return err
	}
	if n == 0 {
		return errorsuc.NewNotFoundError("ferramenta da operação não encontrada")
	}
	return nil
}

func (r *ToolRepositorySQLC) ListToolsByRouteOp(ctx context.Context, routeOperationID int64) ([]*entity.RouteOpTool, error) {
	if err := r.posse(ctx, r.q.RouteOperationBelongsToEnterprise, routeOperationID, "operação do roteiro"); err != nil {
		return nil, err
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListToolsByRouteOp(ctx, sqlc.ListToolsByRouteOpParams{RouteOperationID: routeOperationID, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar as ferramentas da operação %d do roteiro: %w", routeOperationID, err)
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
	if err := r.posse(ctx, r.q.RouteBelongsToEnterprise, routeID, "roteiro"); err != nil {
		return nil, err
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListToolsByRoute(ctx, sqlc.ListToolsByRouteParams{RouteID: routeID, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar as ferramentas do roteiro %d: %w", routeID, err)
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
	if err := r.ferramenta(ctx, s.ToolID); err != nil {
		return nil, err
	}
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
	if err := r.serie(ctx, s.ID); err != nil {
		return nil, err
	}
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
	if err := r.serie(ctx, id); err != nil {
		return nil, err
	}
	row, err := r.q.GetToolSerial(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("número de série %d da ferramenta não encontrado: %w", id, err)
	}
	return toolSerialRowToEntity(row), nil
}

func (r *ToolRepositorySQLC) ListToolSerials(ctx context.Context, toolID int64, onlyActive bool) ([]*entity.ToolSerial, error) {
	if err := r.ferramenta(ctx, toolID); err != nil {
		return nil, err
	}
	rows, err := r.q.ListToolSerials(ctx, sqlc.ListToolSerialsParams{ToolID: toolID, OnlyActive: onlyActive})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os números de série da ferramenta %d: %w", toolID, err)
	}
	out := make([]*entity.ToolSerial, 0, len(rows))
	for _, row := range rows {
		out = append(out, toolSerialRowToEntity(row))
	}
	return out, nil
}

func (r *ToolRepositorySQLC) DeactivateToolSerial(ctx context.Context, id int64) error {
	if err := r.serie(ctx, id); err != nil {
		return err
	}
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
