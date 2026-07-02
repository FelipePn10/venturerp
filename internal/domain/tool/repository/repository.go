package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
)

type ToolRepository interface {
	// Master
	CreateTool(ctx context.Context, t *entity.Tool) (*entity.Tool, error)
	UpdateTool(ctx context.Context, t *entity.Tool) (*entity.Tool, error)
	GetTool(ctx context.Context, id int64) (*entity.Tool, error)
	ListTools(ctx context.Context, onlyActive bool) ([]*entity.Tool, error)
	DeactivateTool(ctx context.Context, id int64) error
	NextToolCode(ctx context.Context) (int64, error)

	// Life
	ConsumeToolLife(ctx context.Context, id int64, amount float64) (*entity.Tool, error)
	ResetToolLife(ctx context.Context, id int64) (*entity.Tool, error)
	ListToolsNeedingReplacement(ctx context.Context) ([]*entity.Tool, error)

	// Association
	AddRouteOpTool(ctx context.Context, rt *entity.RouteOpTool) (*entity.RouteOpTool, error)
	RemoveRouteOpTool(ctx context.Context, id int64) error
	ListToolsByRouteOp(ctx context.Context, routeOperationID int64) ([]*entity.RouteOpTool, error)
	ListToolsByRoute(ctx context.Context, routeID int64) ([]*entity.RouteOpTool, error)
}
