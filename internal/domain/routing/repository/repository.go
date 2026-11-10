package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	// ListRoutesByItemWithInactive inclui os desativados, para a tela poder reativá-los.
	ListRoutesByItemWithInactive(ctx context.Context, itemCode int64) ([]*entity.ManufacturingRoute, error)
	// ReactivateRoute devolve false se o roteiro não existe ou já está ativo.
	ReactivateRoute(ctx context.Context, id int64) (bool, error)

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
	// CopyRoute cria `destino` e copia as etapas da origem e tudo o que pende
	// delas, numa transação só.
	CopyRoute(ctx context.Context, sourceRouteID int64, destino *entity.ManufacturingRoute) (*entity.ManufacturingRoute, entity.RouteCopyResult, error)
	// ListRouteTemplates lista os roteiros-modelo (sem item).
	ListRouteTemplates(ctx context.Context, incluirInativos bool) ([]*entity.ManufacturingRoute, error)
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

// escolhePorQuantidade é o pedaço do repositório que considera a faixa de lote.
type escolhePorQuantidade interface {
	GetRouteForItemQty(ctx context.Context, itemCode int64, mask string, qty *decimal.Decimal) (*entity.ManufacturingRoute, error)
}

type escolheSemQuantidade interface {
	GetRouteForItem(ctx context.Context, itemCode int64, mask string) (*entity.ManufacturingRoute, error)
}

type externasPorOrdem interface {
	GetExternalOpsForOrder(ctx context.Context, itemCode int64, mask string, qty decimal.Decimal) ([]*entity.ExternalOp, error)
}

type externasPorItem interface {
	GetExternalOpsByItem(ctx context.Context, itemCode int64) ([]*entity.ExternalOp, error)
}

// OperacoesExternasDaOrdem: etapas externas do roteiro que a OP da máscara e
// quantidade vai usar — a ordem de serviço nasce do mesmo roteiro que a OP
// explode. Repositório que não conhece máscara/quantidade cai no genérico.
func OperacoesExternasDaOrdem(ctx context.Context, repo externasPorItem, itemCode int64, mask string, qty decimal.Decimal) ([]*entity.ExternalOp, error) {
	if r, ok := repo.(externasPorOrdem); ok {
		return r.GetExternalOpsForOrder(ctx, itemCode, mask, qty)
	}
	return repo.GetExternalOpsByItem(ctx, itemCode)
}

// RoteiroParaQuantidade escolhe o roteiro do item para a quantidade da ordem:
// (GetRouteForItemQty fica FORA de RoutingRepository de propósito: fakes que
// embutem a interface responderiam à asserção e chamariam um método nil.)
// com a faixa de lote quando o repositório sabe considerá-la, senão pela regra
// sem quantidade. Existe para os consumidores com interfaces estreitas (OP,
// ordem planejada, capacidade, custo) não precisarem mudar de contrato.
func RoteiroParaQuantidade(ctx context.Context, repo escolheSemQuantidade, itemCode int64, mask string, qty decimal.Decimal) (*entity.ManufacturingRoute, error) {
	if r, ok := repo.(escolhePorQuantidade); ok && qty.IsPositive() {
		return r.GetRouteForItemQty(ctx, itemCode, mask, &qty)
	}
	return repo.GetRouteForItem(ctx, itemCode, mask)
}
