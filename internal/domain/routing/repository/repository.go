package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/google/uuid"
)

type RoutingRepository interface {
	// Operations
	CreateOperation(ctx context.Context, op *entity.Operation) (*entity.Operation, error)
	UpdateOperation(ctx context.Context, op *entity.Operation) (*entity.Operation, error)
	GetOperationByID(ctx context.Context, id int64) (*entity.Operation, error)
	ListOperations(ctx context.Context, onlyActive bool) ([]*entity.Operation, error)
	DeactivateOperation(ctx context.Context, id int64) error
	OperationUsedInRoutes(ctx context.Context, id int64) (bool, error)

	// Routes
	CreateRoute(ctx context.Context, r *entity.ManufacturingRoute) (*entity.ManufacturingRoute, error)
	UpdateRoute(ctx context.Context, r *entity.ManufacturingRoute) (*entity.ManufacturingRoute, error)
	GetRouteByID(ctx context.Context, id int64) (*entity.ManufacturingRoute, error)
	GetRouteByItemCode(ctx context.Context, itemCode int64, mask string, alternative int16) (*entity.ManufacturingRoute, error)
	ListRoutesByItem(ctx context.Context, itemCode int64) ([]*entity.ManufacturingRoute, error)
	DeactivateRoute(ctx context.Context, id int64) error

	// Route operations
	AddRouteOperation(ctx context.Context, op *entity.RouteOperation) (*entity.RouteOperation, error)
	UpdateRouteOperation(ctx context.Context, op *entity.RouteOperation) (*entity.RouteOperation, error)
	GetRouteOperations(ctx context.Context, routeID int64) ([]*entity.RouteOperation, error)
	RemoveRouteOperation(ctx context.Context, id int64) error

	// Network edges
	SetNetworkEdge(ctx context.Context, edge *entity.NetworkEdge) (*entity.NetworkEdge, error)
	DeleteNetworkEdge(ctx context.Context, predecessorID, successorID int64) error
	GetNetworkEdges(ctx context.Context, routeID int64) ([]*entity.NetworkEdge, error)

	// Alternative resources per route operation
	AddRouteOpResource(ctx context.Context, res *entity.RouteOpResource) (*entity.RouteOpResource, error)
	UpdateRouteOpResource(ctx context.Context, res *entity.RouteOpResource) (*entity.RouteOpResource, error)
	GetRouteOpResource(ctx context.Context, id int64) (*entity.RouteOpResource, error)
	RemoveRouteOpResource(ctx context.Context, id int64) error
	SetRouteOpResourcePrimary(ctx context.Context, id, routeOperationID, workCenterID int64) (*entity.RouteOpResource, error)
	ListResourcesByRouteOp(ctx context.Context, routeOperationID int64) ([]*entity.RouteOpResource, error)
	ListResourcesByRoute(ctx context.Context, routeID int64) ([]*entity.RouteOpResource, error)

	// MRP
	GetRouteForItem(ctx context.Context, itemCode int64, mask string) (*entity.ManufacturingRoute, error)
	ItemHasRoute(ctx context.Context, itemCode int64) (bool, error)
	NextRouteCode(ctx context.Context) (int64, error)
	NextOperationCode(ctx context.Context) (int64, error)
	CreatedByFromUUID(v uuid.UUID) uuid.UUID // identity helper for DI

	// Documentos de processo (desenho, instrução, ficha). O vínculo é com a
	// operação de biblioteca OU com a etapa do roteiro — nunca com os dois.
	CreateOperationDocument(ctx context.Context, doc *entity.OperationDocument) (*entity.OperationDocument, error)
	UpdateOperationDocument(ctx context.Context, doc *entity.OperationDocument) (*entity.OperationDocument, error)
	DeactivateOperationDocument(ctx context.Context, id int64) error
	ListDocumentsByOperation(ctx context.Context, operationID int64) ([]*entity.OperationDocument, error)
	ListDocumentsForRouteOperation(ctx context.Context, routeOperationID int64) ([]*entity.OperationDocument, error)
	ListDocumentsByRoute(ctx context.Context, routeID int64) ([]*entity.OperationDocument, error)

	// RouteIDOfOperation diz a que roteiro uma etapa pertence, já filtrando por
	// empresa: a etapa é endereçada por id e sem o filtro seria alcançável de
	// outra empresa.
	RouteIDOfOperation(ctx context.Context, routeOperationID int64) (int64, error)

	// Pontos de inspeção amarrados às etapas do roteiro.
	CreateRouteInspection(ctx context.Context, ins *entity.RouteInspection) (*entity.RouteInspection, error)
	ListInspectionsByRoute(ctx context.Context, routeID int64) ([]*entity.RouteInspection, error)
	DeactivateRouteInspection(ctx context.Context, id int64) error

	// GetExternalOpsByItem returns external/third-party operations from the
	// standard route of the given item. Returns empty slice when no route exists.
	GetExternalOpsByItem(ctx context.Context, itemCode int64) ([]*entity.ExternalOp, error)
}
