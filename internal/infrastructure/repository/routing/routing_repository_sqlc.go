package routing

import (
	"encoding/json"
	"strings"

	"context"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
)

// semLinha traduz "nenhuma linha" — o filtro de empresa não achou o registro
// (ou ele é de outra empresa) — para "não encontrado".
func semLinha(err error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(msg)
	}
	return err
}

// sequenciaRepetida: a etapa bateu no índice único de sequência ativa.
func sequenciaRepetida(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_route_operations_sequencia_ativa"
}

// afetou: gravação que não pegou linha nenhuma é registro de outra empresa
// ou inexistente.
func afetou(n int64, err error, msg string) error {
	if err != nil {
		return err
	}
	if n == 0 {
		return errorsuc.NewNotFoundError(msg)
	}
	return nil
}

const (
	msgEtapa   = "etapa do roteiro não encontrada"
	msgRecurso = "recurso da etapa do roteiro não encontrado"
)

type RoutingRepositorySQLC struct {
	q *sqlc.Queries
}

func New(q *sqlc.Queries) domainrepo.RoutingRepository {
	return &RoutingRepositorySQLC{q: q}
}

func (r *RoutingRepositorySQLC) CreatedByFromUUID(v uuid.UUID) uuid.UUID { return v }

// ─── operations ──────────────────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) CreateOperation(ctx context.Context, op *entity.Operation) (*entity.Operation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateOperation(ctx, sqlc.CreateOperationParams{
		Code:                 op.Code,
		Name:                 op.Name,
		Description:          pgutil.ToPgTextFromPtr(op.Description),
		Origin:               sqltypes.OperationOriginEnum(op.Origin),
		Situation:            sqltypes.OperationSituationEnum(op.Situation),
		DefaultWorkCenterID:  op.DefaultWorkCenterID,
		StandardTime:         pgutil.ToPgNumericFromFloat64(op.StandardTime),
		SetupTime:            pgutil.ToPgNumericFromFloat64(op.SetupTime),
		RunTime:              pgutil.ToPgNumericFromFloat64(op.RunTime),
		LaborTime:            pgutil.ToPgNumericFromFloat64(op.LaborTime),
		RunTimeBaseQty:       pgutil.ToPgNumericFromFloat64(op.RunBaseQty),
		QueueTime:            pgutil.ToPgNumericFromFloat64(op.QueueTime),
		WaitTime:             pgutil.ToPgNumericFromFloat64(op.WaitTime),
		MoveTime:             pgutil.ToPgNumericFromFloat64(op.MoveTime),
		CrewSize:             pgutil.ToPgNumericFromFloat64(op.CrewSize),
		TimeUnit:             op.TimeUnit,
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          pgutil.ToPgNumericFromDecimalPtr(op.CostPerUnit),
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: operationRemittance(op.ThirdPartyRemittance),
		ScrapPct:             pgutil.ToPgNumericFromFloat64(op.ScrapPct),
		TimeBasis:            baseOuPeca(op.TimeBasis),
		TimeBasisAttribute:   pgutil.ToPgTextFromPtr(op.TimeBasisAttribute),
		CreatedBy:            pgutil.ToPgUUID(op.CreatedBy),
		EnterpriseID:         enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating operation: %w", err)
	}
	return operationRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateOperation(ctx context.Context, op *entity.Operation) (*entity.Operation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateOperation(ctx, sqlc.UpdateOperationParams{
		ID:                   op.ID,
		Name:                 op.Name,
		Description:          pgutil.ToPgTextFromPtr(op.Description),
		Origin:               sqltypes.OperationOriginEnum(op.Origin),
		Situation:            sqltypes.OperationSituationEnum(op.Situation),
		DefaultWorkCenterID:  op.DefaultWorkCenterID,
		StandardTime:         pgutil.ToPgNumericFromFloat64(op.StandardTime),
		SetupTime:            pgutil.ToPgNumericFromFloat64(op.SetupTime),
		RunTime:              pgutil.ToPgNumericFromFloat64(op.RunTime),
		LaborTime:            pgutil.ToPgNumericFromFloat64(op.LaborTime),
		RunTimeBaseQty:       pgutil.ToPgNumericFromFloat64(op.RunBaseQty),
		QueueTime:            pgutil.ToPgNumericFromFloat64(op.QueueTime),
		WaitTime:             pgutil.ToPgNumericFromFloat64(op.WaitTime),
		MoveTime:             pgutil.ToPgNumericFromFloat64(op.MoveTime),
		CrewSize:             pgutil.ToPgNumericFromFloat64(op.CrewSize),
		TimeUnit:             op.TimeUnit,
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          pgutil.ToPgNumericFromDecimalPtr(op.CostPerUnit),
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: operationRemittance(op.ThirdPartyRemittance),
		ScrapPct:             pgutil.ToPgNumericFromFloat64(op.ScrapPct),
		TimeBasis:            baseOuPeca(op.TimeBasis),
		TimeBasisAttribute:   pgutil.ToPgTextFromPtr(op.TimeBasisAttribute),
		EnterpriseID:         enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("updating operation: %w", err)
	}
	return operationRowToEntity(row), nil
}

func baseOuPeca(base string) string {
	if base == "" {
		return entity.BasePeca
	}
	return base
}

func operationRemittance(value string) string {
	if value == "" {
		return "DEMAND_ITEMS"
	}
	return value
}

func (r *RoutingRepositorySQLC) GetOperationByID(ctx context.Context, id int64) (*entity.Operation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetOperationByID(ctx, sqlc.GetOperationByIDParams{ID: id, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("fetching operation %d: %w", id, err)
	}
	return operationRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) ListOperations(ctx context.Context, onlyActive bool) ([]*entity.Operation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListOperations(ctx, sqlc.ListOperationsParams{Column1: onlyActive, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("listing operations: %w", err)
	}
	out := make([]*entity.Operation, 0, len(rows))
	for _, row := range rows {
		out = append(out, operationRowToEntity(row))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) DeactivateOperation(ctx context.Context, id int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.DeactivateOperation(ctx, sqlc.DeactivateOperationParams{ID: id, EnterpriseID: enterpriseID})
}

func (r *RoutingRepositorySQLC) OperationUsedInRoutes(ctx context.Context, id int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	return r.q.OperationUsedInRoutes(ctx, sqlc.OperationUsedInRoutesParams{OperationID: id, EnterpriseID: enterpriseID})
}

func (r *RoutingRepositorySQLC) NextOperationCode(ctx context.Context) (int64, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	v, err := r.q.NextOperationCode(ctx, enterpriseID)
	return int64(v), err
}

// ─── manufacturing_routes ─────────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) CreateRoute(ctx context.Context, rt *entity.ManufacturingRoute) (*entity.ManufacturingRoute, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var row sqlc.ManufacturingRoute
	err = r.q.InTx(ctx, func(q *sqlc.Queries) error {
		var errGrava error
		row, errGrava = q.CreateRoute(ctx, sqlc.CreateRouteParams{
			Code:         rt.Code,
			ItemCode:     itemDoRoteiro(rt),
			IsTemplate:   rt.IsTemplate,
			LotMin:       pgutil.ToPgNumericFromDecimalPtr(rt.LotMin),
			LotMax:       pgutil.ToPgNumericFromDecimalPtr(rt.LotMax),
			Mask:         pgutil.ToPgTextFromPtr(rt.Mask),
			Alternative:  rt.Alternative,
			Description:  pgutil.ToPgTextFromPtr(rt.Description),
			Situation:    sqltypes.RouteSituationEnum(rt.Situation),
			IsStandard:   rt.IsStandard,
			ValidFrom:    pgutil.ToPgDateFromPtr(rt.ValidFrom),
			ValidTo:      pgutil.ToPgDateFromPtr(rt.ValidTo),
			CreatedBy:    pgutil.ToPgUUID(rt.CreatedBy),
			EnterpriseID: enterpriseID,
		})
		if errGrava != nil || !row.IsStandard {
			return errGrava
		}
		return q.ClearOverlappingStandardRoutes(ctx, sqlc.ClearOverlappingStandardRoutesParams{ID: row.ID, EnterpriseID: enterpriseID})
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, errorsuc.NewConflictError(
				"já existe um roteiro com esta alternativa para este item e máscara; use outro número de alternativa ou altere o roteiro existente")
		}
		return nil, fmt.Errorf("creating route: %w", err)
	}
	return routeRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateRoute(ctx context.Context, rt *entity.ManufacturingRoute) (*entity.ManufacturingRoute, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var row sqlc.ManufacturingRoute
	err = r.q.InTx(ctx, func(q *sqlc.Queries) error {
		var errGrava error
		row, errGrava = q.UpdateRoute(ctx, sqlc.UpdateRouteParams{
			ID:           rt.ID,
			Description:  pgutil.ToPgTextFromPtr(rt.Description),
			Situation:    sqltypes.RouteSituationEnum(rt.Situation),
			IsStandard:   rt.IsStandard,
			ValidFrom:    pgutil.ToPgDateFromPtr(rt.ValidFrom),
			ValidTo:      pgutil.ToPgDateFromPtr(rt.ValidTo),
			LotMin:       pgutil.ToPgNumericFromDecimalPtr(rt.LotMin),
			LotMax:       pgutil.ToPgNumericFromDecimalPtr(rt.LotMax),
			EnterpriseID: enterpriseID,
		})
		if errGrava != nil || !row.IsStandard {
			return errGrava
		}
		return q.ClearOverlappingStandardRoutes(ctx, sqlc.ClearOverlappingStandardRoutesParams{ID: row.ID, EnterpriseID: enterpriseID})
	})
	if err != nil {
		return nil, fmt.Errorf("updating route: %w", semLinha(err, "roteiro não encontrado nesta empresa"))
	}
	return routeRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) GetRouteByID(ctx context.Context, id int64) (*entity.ManufacturingRoute, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetRouteByID(ctx, sqlc.GetRouteByIDParams{ID: id, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("fetching route %d: %w", id, err)
	}
	return routeRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) GetRouteByItemCode(ctx context.Context, itemCode int64, mask string, alternative int16) (*entity.ManufacturingRoute, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetRouteByItemAndAlternative(ctx, sqlc.GetRouteByItemAndAlternativeParams{
		ItemCode:     &itemCode,
		Mask:         pgutil.ToPgText(mask),
		Alternative:  alternative,
		EnterpriseID: enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar o roteiro do item %d: %w", itemCode, err)
	}
	return routeRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) ListRoutesByItem(ctx context.Context, itemCode int64) ([]*entity.ManufacturingRoute, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRoutesByItem(ctx, sqlc.ListRoutesByItemParams{ItemCode: &itemCode, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os roteiros do item %d: %w", itemCode, err)
	}
	out := make([]*entity.ManufacturingRoute, 0, len(rows))
	for _, row := range rows {
		out = append(out, routeRowToEntity(row))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) ListRoutesByItemWithInactive(ctx context.Context, itemCode int64) ([]*entity.ManufacturingRoute, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRoutesByItemWithInactive(ctx, sqlc.ListRoutesByItemWithInactiveParams{ItemCode: &itemCode, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os roteiros do item %d: %w", itemCode, err)
	}
	out := make([]*entity.ManufacturingRoute, 0, len(rows))
	for _, row := range rows {
		out = append(out, routeRowToEntity(row))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) ReactivateRoute(ctx context.Context, id int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	n, err := r.q.ReactivateRoute(ctx, sqlc.ReactivateRouteParams{ID: id, EnterpriseID: enterpriseID})
	if err != nil {
		return false, fmt.Errorf("reativando o roteiro %d: %w", id, err)
	}
	return n > 0, nil
}

func (r *RoutingRepositorySQLC) DeactivateRoute(ctx context.Context, id int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.DeactivateRoute(ctx, sqlc.DeactivateRouteParams{ID: id, EnterpriseID: enterpriseID})
}

func (r *RoutingRepositorySQLC) NextRouteCode(ctx context.Context) (int64, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	v, err := r.q.NextRouteCode(ctx, enterpriseID)
	return int64(v), err
}

func (r *RoutingRepositorySQLC) ItemHasRoute(ctx context.Context, itemCode int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	return r.q.ItemHasRoute(ctx, sqlc.ItemHasRouteParams{ItemCode: &itemCode, EnterpriseID: enterpriseID})
}

func (r *RoutingRepositorySQLC) GetRouteForItem(ctx context.Context, itemCode int64, mask string) (*entity.ManufacturingRoute, error) {
	return r.GetRouteForItemQty(ctx, itemCode, mask, nil)
}

// GetRouteForItemQty escolhe o roteiro pela regra única (erp_roteiro_vigente):
// com a quantidade, a faixa de lote do roteiro entra na escolha.
func (r *RoutingRepositorySQLC) GetRouteForItemQty(ctx context.Context, itemCode int64, mask string, qty *decimal.Decimal) (*entity.ManufacturingRoute, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetStandardRouteForItem(ctx, sqlc.GetStandardRouteForItemParams{
		EnterpriseID: enterpriseID,
		ItemCode:     itemCode,
		Mask:         pgutil.ToPgText(mask),
		Quantity:     pgutil.ToPgNumericFromDecimalPtr(qty),
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar o roteiro padrão do item %d: %w", itemCode, err)
	}
	return routeRowToEntity(row), nil
}

// ListRouteTemplates lista os roteiros-modelo da empresa.
func (r *RoutingRepositorySQLC) ListRouteTemplates(ctx context.Context, incluirInativos bool) ([]*entity.ManufacturingRoute, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRouteTemplates(ctx, sqlc.ListRouteTemplatesParams{EnterpriseID: enterpriseID, IncludeInactive: incluirInativos})
	if err != nil {
		return nil, fmt.Errorf("listando os roteiros-modelo: %w", err)
	}
	out := make([]*entity.ManufacturingRoute, 0, len(rows))
	for _, row := range rows {
		out = append(out, routeRowToEntity(row))
	}
	return out, nil
}

func itemDoRoteiro(rt *entity.ManufacturingRoute) *int64 {
	if rt.IsTemplate || rt.ItemCode <= 0 {
		return nil
	}
	v := rt.ItemCode
	return &v
}

// ─── route_operations ────────────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) AddRouteOperation(ctx context.Context, op *entity.RouteOperation) (*entity.RouteOperation, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.AddRouteOperation(ctx, sqlc.AddRouteOperationParams{
		EnterpriseID:         empresa,
		RouteID:              op.RouteID,
		Sequence:             op.Sequence,
		OperationID:          op.OperationID,
		WorkCenterID:         op.WorkCenterID,
		StandardTime:         pgutil.ToPgNumericFromFloat64Ptr(op.StandardTime),
		SetupTime:            pgutil.ToPgNumericFromFloat64Ptr(op.SetupTime),
		RunTime:              pgutil.ToPgNumericFromFloat64Ptr(op.RunTime),
		LaborTime:            pgutil.ToPgNumericFromFloat64Ptr(op.LaborTime),
		RunTimeBaseQty:       pgutil.ToPgNumericFromFloat64Ptr(op.RunBaseQty),
		QueueTime:            pgutil.ToPgNumericFromFloat64Ptr(op.QueueTime),
		WaitTime:             pgutil.ToPgNumericFromFloat64Ptr(op.WaitTime),
		MoveTime:             pgutil.ToPgNumericFromFloat64Ptr(op.MoveTime),
		CrewSize:             pgutil.ToPgNumericFromFloat64Ptr(op.CrewSize),
		TimeUnit:             pgutil.ToPgTextFromPtr(op.TimeUnit),
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          pgutil.ToPgNumericFromDecimalPtr(op.CostPerUnit),
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: pgutil.ToPgTextFromPtr(op.ThirdPartyRemittance),
		ScrapPct:             pgutil.ToPgNumericFromFloat64Ptr(op.ScrapPct),
		InspectionRequired:   op.InspectionRequired,
		Situation:            sqltypes.RouteOpSituationEnum(op.Situation),
		Notes:                pgutil.ToPgTextFromPtr(op.Notes),
		TimeBasis:            pgutil.ToPgTextFromPtr(op.TimeBasis),
		TimeBasisAttribute:   pgutil.ToPgTextFromPtr(op.TimeBasisAttribute),
	})
	if err != nil {
		if sequenciaRepetida(err) {
			return nil, errorsuc.NewConflictError(fmt.Sprintf(
				"o roteiro já tem uma etapa na sequência %d; use outra sequência ou edite a etapa existente", op.Sequence))
		}
		return nil, fmt.Errorf("adding route operation: %w", semLinha(err, "roteiro, operação ou centro de trabalho não encontrado"))
	}
	return routeOpRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateRouteOperation(ctx context.Context, op *entity.RouteOperation) (*entity.RouteOperation, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var row sqlc.RouteOperation
	err = r.q.InTx(ctx, func(q *sqlc.Queries) error {
		if op.WorkCenterID != nil {
			daEmpresa, errCentro := q.WorkCenterOfEnterprise(ctx, sqlc.WorkCenterOfEnterpriseParams{ID: *op.WorkCenterID, EnterpriseID: &empresa})
			if errCentro != nil {
				return errCentro
			}
			if !daEmpresa {
				return errorsuc.NewNotFoundError("centro de trabalho não encontrado nesta empresa")
			}
		}
		var errGrava error
		row, errGrava = q.UpdateRouteOperation(ctx, sqlc.UpdateRouteOperationParams{
			EnterpriseID:         empresa,
			ID:                   op.ID,
			WorkCenterID:         op.WorkCenterID,
			StandardTime:         pgutil.ToPgNumericFromFloat64Ptr(op.StandardTime),
			SetupTime:            pgutil.ToPgNumericFromFloat64Ptr(op.SetupTime),
			RunTime:              pgutil.ToPgNumericFromFloat64Ptr(op.RunTime),
			LaborTime:            pgutil.ToPgNumericFromFloat64Ptr(op.LaborTime),
			RunTimeBaseQty:       pgutil.ToPgNumericFromFloat64Ptr(op.RunBaseQty),
			QueueTime:            pgutil.ToPgNumericFromFloat64Ptr(op.QueueTime),
			WaitTime:             pgutil.ToPgNumericFromFloat64Ptr(op.WaitTime),
			MoveTime:             pgutil.ToPgNumericFromFloat64Ptr(op.MoveTime),
			CrewSize:             pgutil.ToPgNumericFromFloat64Ptr(op.CrewSize),
			TimeUnit:             pgutil.ToPgTextFromPtr(op.TimeUnit),
			SupplierID:           op.SupplierID,
			ServiceItemCode:      op.ServiceItemCode,
			CostPerUnit:          pgutil.ToPgNumericFromDecimalPtr(op.CostPerUnit),
			LeadTimeDays:         op.LeadTimeDays,
			ThirdPartyRemittance: pgutil.ToPgTextFromPtr(op.ThirdPartyRemittance),
			ScrapPct:             pgutil.ToPgNumericFromFloat64Ptr(op.ScrapPct),
			InspectionRequired:   op.InspectionRequired,
			Situation:            sqltypes.RouteOpSituationEnum(op.Situation),
			Notes:                pgutil.ToPgTextFromPtr(op.Notes),
			TimeBasis:            pgutil.ToPgTextFromPtr(op.TimeBasis),
			TimeBasisAttribute:   pgutil.ToPgTextFromPtr(op.TimeBasisAttribute),
		})
		if errGrava != nil {
			return errGrava
		}
		return q.SyncPrimaryResourceWithStep(ctx, sqlc.SyncPrimaryResourceWithStepParams{ID: row.ID, EnterpriseID: empresa})
	})
	if err != nil {
		return nil, fmt.Errorf("updating route operation: %w", semLinha(err, "etapa do roteiro ou centro de trabalho não encontrado"))
	}
	return routeOpRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) GetRouteOperations(ctx context.Context, routeID int64) ([]*entity.RouteOperation, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetRouteOperations(ctx, sqlc.GetRouteOperationsParams{RouteID: routeID, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar as operações do roteiro %d: %w", routeID, err)
	}
	out := make([]*entity.RouteOperation, 0, len(rows))
	for _, row := range rows {
		out = append(out, routeOpRowWithNamesToEntity(row))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) RemoveRouteOperation(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	n, err := r.q.RemoveRouteOperation(ctx, sqlc.RemoveRouteOperationParams{ID: id, EnterpriseID: empresa})
	return afetou(n, err, msgEtapa)
}

// ─── network edges ────────────────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) SetNetworkEdge(ctx context.Context, edge *entity.NetworkEdge) (*entity.NetworkEdge, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpsertNetworkEdge(ctx, sqlc.UpsertNetworkEdgeParams{
		EnterpriseID:  empresa,
		PredecessorID: edge.PredecessorID,
		SuccessorID:   edge.SuccessorID,
		OverlapPct:    pgutil.ToPgNumericFromFloat64(edge.OverlapPct),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting network edge: %w", semLinha(err, "as duas etapas precisam ser do mesmo roteiro da empresa"))
	}
	return &entity.NetworkEdge{
		ID:            row.ID,
		PredecessorID: row.PredecessorID,
		SuccessorID:   row.SuccessorID,
		OverlapPct:    pgutil.FromPgNumericToFloat64(row.OverlapPct),
		CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
	}, nil
}

func (r *RoutingRepositorySQLC) DeleteNetworkEdge(ctx context.Context, predecessorID, successorID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	n, err := r.q.DeleteNetworkEdge(ctx, sqlc.DeleteNetworkEdgeParams{
		PredecessorID: predecessorID,
		SuccessorID:   successorID,
		EnterpriseID:  empresa,
	})
	return afetou(n, err, "precedência do roteiro não encontrada")
}

func (r *RoutingRepositorySQLC) GetNetworkEdges(ctx context.Context, routeID int64) ([]*entity.NetworkEdge, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetNetworkEdges(ctx, sqlc.GetNetworkEdgesParams{RouteID: routeID, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar as precedências do roteiro %d: %w", routeID, err)
	}
	out := make([]*entity.NetworkEdge, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.NetworkEdge{
			ID:            row.ID,
			PredecessorID: row.PredecessorID,
			SuccessorID:   row.SuccessorID,
			OverlapPct:    pgutil.FromPgNumericToFloat64(row.OverlapPct),
			CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
		})
	}
	return out, nil
}

// ─── route operation resources (alternatives) ─────────────────────────────────

func (r *RoutingRepositorySQLC) AddRouteOpResource(ctx context.Context, res *entity.RouteOpResource) (*entity.RouteOpResource, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.AddRouteOpResource(ctx, sqlc.AddRouteOpResourceParams{
		EnterpriseID:     empresa,
		RouteOperationID: res.RouteOperationID,
		WorkCenterID:     res.WorkCenterID,
		Priority:         res.Priority,
		TimeFactor:       pgutil.ToPgNumericFromFloat64(res.TimeFactor),
		IsPrimary:        res.IsPrimary,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, errorsuc.NewConflictError("este centro já está na lista de centros desta etapa")
		}
		return nil, fmt.Errorf("adding route op resource: %w", semLinha(err, "etapa do roteiro ou centro de trabalho não encontrado"))
	}
	return resourceRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateRouteOpResource(ctx context.Context, res *entity.RouteOpResource) (*entity.RouteOpResource, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateRouteOpResource(ctx, sqlc.UpdateRouteOpResourceParams{
		EnterpriseID: empresa,
		ID:           res.ID,
		Priority:     res.Priority,
		TimeFactor:   pgutil.ToPgNumericFromFloat64(res.TimeFactor),
	})
	if err != nil {
		return nil, fmt.Errorf("updating route op resource: %w", semLinha(err, msgRecurso))
	}
	return resourceRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) GetRouteOpResource(ctx context.Context, id int64) (*entity.RouteOpResource, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetRouteOpResource(ctx, sqlc.GetRouteOpResourceParams{ID: id, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("fetching route op resource %d: %w", id, semLinha(err, msgRecurso))
	}
	return resourceRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) RemoveRouteOpResource(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	n, err := r.q.RemoveRouteOpResource(ctx, sqlc.RemoveRouteOpResourceParams{ID: id, EnterpriseID: empresa})
	return afetou(n, err, msgRecurso)
}

// SetRouteOpResourcePrimary makes one resource the primary for its operation:
// it clears any other primary, flags this one, and mirrors the choice onto
// route_operations.work_center_id (so cost/CRP/lead-time use it).
func (r *RoutingRepositorySQLC) SetRouteOpResourcePrimary(ctx context.Context, id, routeOperationID, workCenterID int64) (*entity.RouteOpResource, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Três gravações que só fazem sentido juntas: sem a transação, uma falha no
	// meio deixava a etapa sem recurso principal (ou com o centro desatualizado).
	var row sqlc.RouteOperationResource
	err = r.q.InTx(ctx, func(q *sqlc.Queries) error {
		if err := q.KeepStepCenterAsAlternative(ctx, sqlc.KeepStepCenterAsAlternativeParams{
			ResourceID: id, RouteOperationID: routeOperationID, EnterpriseID: empresa,
		}); err != nil {
			return fmt.Errorf("guardando o centro atual como alternativo: %w", err)
		}
		if err := q.ClearPrimaryResources(ctx, sqlc.ClearPrimaryResourcesParams{RouteOperationID: routeOperationID, EnterpriseID: empresa}); err != nil {
			return fmt.Errorf("clearing primary resources: %w", err)
		}
		var err error
		row, err = q.SetResourcePrimary(ctx, sqlc.SetResourcePrimaryParams{ID: id, EnterpriseID: empresa})
		if err != nil {
			return fmt.Errorf("setting resource primary: %w", semLinha(err, msgRecurso))
		}
		if row.RouteOperationID != routeOperationID {
			return errorsuc.NewValidationError("o recurso não pertence a esta etapa do roteiro")
		}
		if err := q.SetRouteOpWorkCenter(ctx, sqlc.SetRouteOpWorkCenterParams{
			ID:           routeOperationID,
			WorkCenterID: &workCenterID,
			EnterpriseID: empresa,
		}); err != nil {
			return fmt.Errorf("syncing route op work center: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resourceRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) ListResourcesByRouteOp(ctx context.Context, routeOperationID int64) ([]*entity.RouteOpResource, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListResourcesByRouteOp(ctx, sqlc.ListResourcesByRouteOpParams{RouteOperationID: routeOperationID, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os recursos da operação %d do roteiro: %w", routeOperationID, err)
	}
	out := make([]*entity.RouteOpResource, 0, len(rows))
	for _, row := range rows {
		e := &entity.RouteOpResource{
			ID:               row.ID,
			RouteOperationID: row.RouteOperationID,
			WorkCenterID:     row.WorkCenterID,
			Priority:         row.Priority,
			TimeFactor:       pgutil.FromPgNumericToFloat64(row.TimeFactor),
			IsPrimary:        row.IsPrimary,
			CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
			UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
		}
		if row.WorkCenterName.Valid {
			e.WorkCenterName = row.WorkCenterName.String
		}
		out = append(out, e)
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) ListResourcesByRoute(ctx context.Context, routeID int64) ([]*entity.RouteOpResource, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListResourcesByRoute(ctx, sqlc.ListResourcesByRouteParams{RouteID: routeID, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os recursos do roteiro %d: %w", routeID, err)
	}
	out := make([]*entity.RouteOpResource, 0, len(rows))
	for _, row := range rows {
		e := &entity.RouteOpResource{
			ID:               row.ID,
			RouteOperationID: row.RouteOperationID,
			WorkCenterID:     row.WorkCenterID,
			Priority:         row.Priority,
			TimeFactor:       pgutil.FromPgNumericToFloat64(row.TimeFactor),
			IsPrimary:        row.IsPrimary,
			CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
			UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
		}
		if row.WorkCenterName.Valid {
			e.WorkCenterName = row.WorkCenterName.String
		}
		out = append(out, e)
	}
	return out, nil
}

func resourceRowToEntity(row sqlc.RouteOperationResource) *entity.RouteOpResource {
	return &entity.RouteOpResource{
		ID:               row.ID,
		RouteOperationID: row.RouteOperationID,
		WorkCenterID:     row.WorkCenterID,
		Priority:         row.Priority,
		TimeFactor:       pgutil.FromPgNumericToFloat64(row.TimeFactor),
		IsPrimary:        row.IsPrimary,
		CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

// ─── mappers ─────────────────────────────────────────────────────────────────

func operationRowToEntity(row sqlc.Operation) *entity.Operation {
	e := &entity.Operation{
		ID:                   row.ID,
		Code:                 row.Code,
		Name:                 row.Name,
		Origin:               entity.OperationOrigin(row.Origin),
		Situation:            entity.OperationSituation(row.Situation),
		StandardTime:         pgutil.FromPgNumericToFloat64(row.StandardTime),
		SetupTime:            pgutil.FromPgNumericToFloat64(row.SetupTime),
		RunTime:              pgutil.FromPgNumericToFloat64(row.RunTime),
		LaborTime:            pgutil.FromPgNumericToFloat64(row.LaborTime),
		RunBaseQty:           pgutil.FromPgNumericToFloat64(row.RunTimeBaseQty),
		QueueTime:            pgutil.FromPgNumericToFloat64(row.QueueTime),
		WaitTime:             pgutil.FromPgNumericToFloat64(row.WaitTime),
		MoveTime:             pgutil.FromPgNumericToFloat64(row.MoveTime),
		CrewSize:             pgutil.FromPgNumericToFloat64(row.CrewSize),
		TimeUnit:             row.TimeUnit,
		SupplierID:           row.SupplierID,
		ServiceItemCode:      row.ServiceItemCode,
		CostPerUnit:          pgutil.FromPgNumericToDecimalPtr(row.CostPerUnit),
		LeadTimeDays:         row.LeadTimeDays,
		ThirdPartyRemittance: row.ThirdPartyRemittance,
		ScrapPct:             pgutil.FromPgNumericToFloat64(row.ScrapPct),
		IsActive:             row.IsActive,
		CreatedAt:            pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:            pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:            pgutil.FromPgUUID(row.CreatedBy),
		DefaultWorkCenterID:  row.DefaultWorkCenterID,
	}
	if row.Description.Valid {
		v := row.Description.String
		e.Description = &v
	}
	e.TimeBasis = row.TimeBasis
	e.TimeBasisAttribute = textToPtr(row.TimeBasisAttribute)
	return e
}

// numToPtr converts a nullable pg numeric to *float64 (nil when NULL).
func numToPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	v := pgutil.FromPgNumericToFloat64(n)
	return &v
}

// textToPtr converts a nullable pg text to *string (nil when NULL).
func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}

func routeRowToEntity(row sqlc.ManufacturingRoute) *entity.ManufacturingRoute {
	e := &entity.ManufacturingRoute{
		ID:          row.ID,
		Code:        row.Code,
		Alternative: row.Alternative,
		IsTemplate:  row.IsTemplate,
		LotMin:      pgutil.FromPgNumericToDecimalPtr(row.LotMin),
		LotMax:      pgutil.FromPgNumericToDecimalPtr(row.LotMax),
		Situation:   entity.RouteSituation(row.Situation),
		IsStandard:  row.IsStandard,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:   pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:   pgutil.FromPgUUID(row.CreatedBy),
		ValidFrom:   pgutil.FromPgDateToPtr(row.ValidFrom),
		ValidTo:     pgutil.FromPgDateToPtr(row.ValidTo),
	}
	if row.ItemCode != nil {
		e.ItemCode = *row.ItemCode
	}
	if row.Mask.Valid {
		v := row.Mask.String
		e.Mask = &v
	}
	if row.Description.Valid {
		v := row.Description.String
		e.Description = &v
	}
	return e
}

func routeOpRowToEntity(row sqlc.RouteOperation) *entity.RouteOperation {
	e := &entity.RouteOperation{
		ID:           row.ID,
		RouteID:      row.RouteID,
		Sequence:     row.Sequence,
		OperationID:  row.OperationID,
		WorkCenterID: row.WorkCenterID,
		Situation:    entity.RouteOpSituation(row.Situation),
		IsActive:     row.IsActive,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
	if row.StandardTime.Valid {
		v := pgutil.FromPgNumericToFloat64(row.StandardTime)
		e.StandardTime = &v
	}
	if row.SetupTime.Valid {
		v := pgutil.FromPgNumericToFloat64(row.SetupTime)
		e.SetupTime = &v
	}
	e.RunTime = numToPtr(row.RunTime)
	e.LaborTime = numToPtr(row.LaborTime)
	e.RunBaseQty = numToPtr(row.RunTimeBaseQty)
	e.QueueTime = numToPtr(row.QueueTime)
	e.WaitTime = numToPtr(row.WaitTime)
	e.MoveTime = numToPtr(row.MoveTime)
	e.CrewSize = numToPtr(row.CrewSize)
	e.TimeUnit = textToPtr(row.TimeUnit)
	e.SupplierID = row.SupplierID
	e.ServiceItemCode = row.ServiceItemCode
	e.CostPerUnit = pgutil.FromPgNumericToDecimalPtr(row.CostPerUnit)
	e.LeadTimeDays = row.LeadTimeDays
	e.ThirdPartyRemittance = textToPtr(row.ThirdPartyRemittance)
	e.ScrapPct = numToPtr(row.ScrapPct)
	e.TimeBasis = textToPtr(row.TimeBasis)
	e.TimeBasisAttribute = textToPtr(row.TimeBasisAttribute)
	e.InspectionRequired = row.InspectionRequired
	if row.Notes.Valid {
		v := row.Notes.String
		e.Notes = &v
	}
	return e
}

func routeOpRowWithNamesToEntity(row sqlc.GetRouteOperationsRow) *entity.RouteOperation {
	e := routeOpRowToEntity(sqlc.RouteOperation{
		ID:                   row.ID,
		RouteID:              row.RouteID,
		Sequence:             row.Sequence,
		OperationID:          row.OperationID,
		WorkCenterID:         row.WorkCenterID,
		StandardTime:         row.StandardTime,
		SetupTime:            row.SetupTime,
		RunTime:              row.RunTime,
		LaborTime:            row.LaborTime,
		RunTimeBaseQty:       row.RunTimeBaseQty,
		QueueTime:            row.QueueTime,
		WaitTime:             row.WaitTime,
		MoveTime:             row.MoveTime,
		CrewSize:             row.CrewSize,
		TimeUnit:             row.TimeUnit,
		Situation:            row.Situation,
		Notes:                row.Notes,
		IsActive:             row.IsActive,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
		SupplierID:           row.SupplierID,
		ServiceItemCode:      row.ServiceItemCode,
		CostPerUnit:          row.CostPerUnit,
		LeadTimeDays:         row.LeadTimeDays,
		ThirdPartyRemittance: row.ThirdPartyRemittance,
		ScrapPct:             row.ScrapPct,
		InspectionRequired:   row.InspectionRequired,
		TimeBasis:            row.TimeBasis,
		TimeBasisAttribute:   row.TimeBasisAttribute,
	})
	e.OperationName = row.OperationName
	e.OperationOrigin = entity.OperationOrigin(row.OperationOrigin)
	e.RequiresOperator = row.RequiresOperator
	e.EffectiveWorkCenterID = row.EffectiveWorkCenterID
	if row.WorkCenterName.Valid {
		e.WorkCenterName = row.WorkCenterName.String
	}

	// Resolve the quantity-aware time model (route-op overrides ∘ operation defaults),
	// normalised to hours.
	def := entity.TimeComponents{
		Setup:      pgutil.FromPgNumericToFloat64(row.OpSetupTime),
		Run:        entity.TempoDeMaquinaDaOperacao(pgutil.FromPgNumericToFloat64(row.OpRunTime), pgutil.FromPgNumericToFloat64(row.OpStandardTime)),
		Labor:      pgutil.FromPgNumericToFloat64(row.OpLaborTime),
		RunBaseQty: pgutil.FromPgNumericToFloat64(row.OpRunTimeBaseQty),
		Queue:      pgutil.FromPgNumericToFloat64(row.OpQueueTime),
		Wait:       pgutil.FromPgNumericToFloat64(row.OpWaitTime),
		Move:       pgutil.FromPgNumericToFloat64(row.OpMoveTime),
		CrewSize:   pgutil.FromPgNumericToFloat64(row.OpCrewSize),
		Unit:       row.OpTimeUnit,
	}
	e.EffTime = entity.ResolveOperationTime(e.Overrides(), def)

	// Base do tempo (etapa ∘ operação) e quanto dela a peça do roteiro tem.
	e.EffectiveTimeBasis = row.OpTimeBasis
	e.EffectiveTimeBasisAttribute = textToPtr(row.OpTimeBasisAttribute)
	if e.TimeBasis != nil && *e.TimeBasis != "" {
		e.EffectiveTimeBasis = *e.TimeBasis
		e.EffectiveTimeBasisAttribute = e.TimeBasisAttribute
	}
	if e.EffectiveTimeBasis != "" && e.EffectiveTimeBasis != entity.BasePeca {
		if fator, ok := entity.FatorDaBase(e.EffectiveTimeBasis, e.EffectiveTimeBasisAttribute,
			medidasDoItem(row.ItemDimensions, row.ItemPdmAttributes)); ok {
			e.EffTime.UnitsPerPiece = fator
		} else {
			e.EffTime.SemMedida = true
		}
	}
	e.EffectiveStdTime = e.EffTime.Run
	e.EffectiveSetup = e.EffTime.Setup
	e.EffectiveScrap = e.EffectiveScrapPct(pgutil.FromPgNumericToFloat64(row.OpScrapPct))
	e.EffectiveSubcontractDays = row.EffectiveLeadTimeDays
	e.EffectiveCostPerUnit = pgutil.FromPgNumericToDecimalPtr(row.EffectiveCostPerUnit)
	e.EffectiveSupplierID = row.EffectiveSupplierID
	return e
}

// GetExternalOpsByItem devolve as etapas externas do MESMO roteiro que a OP e
// o MRP usam (o vigente, com a máscara genérica como alternativa). Item sem
// roteiro vigente não tem etapa externa: devolve lista vazia.
func (r *RoutingRepositorySQLC) GetExternalOpsByItem(ctx context.Context, itemCode int64) ([]*entity.ExternalOp, error) {
	return r.GetExternalOpsForOrder(ctx, itemCode, "", decimal.Zero)
}

// GetExternalOpsForOrder: as etapas externas do roteiro que a OP desta máscara
// e quantidade vai usar (com faixa de lote, o alternativo pode ser outro).
func (r *RoutingRepositorySQLC) GetExternalOpsForOrder(ctx context.Context, itemCode int64, mask string, qty decimal.Decimal) ([]*entity.ExternalOp, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var q *decimal.Decimal
	if qty.IsPositive() {
		q = &qty
	}
	route, err := r.GetRouteForItemQty(ctx, itemCode, mask, q)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*entity.ExternalOp{}, nil
		}
		return nil, err
	}
	rows, err := r.q.GetExternalRouteOpsForRoute(ctx, sqlc.GetExternalRouteOpsForRouteParams{ID: route.ID, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar as operações externas do item %d: %w", itemCode, err)
	}
	out := make([]*entity.ExternalOp, 0, len(rows))
	for _, row := range rows {
		var supplierID *int64
		if row.SupplierID > 0 {
			value := row.SupplierID
			supplierID = &value
		}
		op := &entity.ExternalOp{
			RouteOpID:       row.ID,
			OperationID:     row.OperationID,
			OperationName:   row.OperationName,
			EffectiveHours:  pgutil.FromPgNumericToFloat64(row.EffectiveHours),
			Origin:          entity.OperationOrigin(row.Origin),
			WorkCenterID:    row.WorkCenterID,
			SupplierID:      supplierID,
			ServiceItemCode: row.ServiceItemCode,
			CostPerUnit:     pgutil.FromPgNumericToDecimal(row.CostPerUnit),
			LeadTimeDays:    row.LeadTimeDays,
			RemittanceType:  row.RemittanceType,
		}
		out = append(out, op)
	}
	return out, nil
}

// medidasDoItem lê as dimensões de engenharia (mm) e os atributos do PDM do
// item do roteiro. Cadastro ilegível vira "sem medida", nunca erro: a etapa
// por medida aparece na prontidão como pendência, e o resto do roteiro segue.
func medidasDoItem(dimensoes, pdm []byte) entity.MedidasDaPeca {
	m := entity.MedidasDaPeca{Atributos: map[string]string{}}
	var d struct{ Length, Width json.Number }
	if json.Unmarshal(dimensoes, &d) == nil {
		m.ComprimentoMM, _ = d.Length.Float64()
		m.LarguraMM, _ = d.Width.Float64()
	}
	var atributos []struct{ Name, Value string }
	if json.Unmarshal(pdm, &atributos) == nil {
		for _, a := range atributos {
			m.Atributos[strings.ToUpper(strings.TrimSpace(a.Name))] = strings.TrimSpace(a.Value)
		}
	}
	return m
}

// CopyRoute cria o roteiro de destino e copia tudo o que pende das etapas da
// origem numa transação só. Antes a tela copiava etapa por etapa pela API: uma
// falha no meio deixava um roteiro pela metade, e centros alternativos,
// ferramentas, documentos e inspeções ficavam para trás.
func (r *RoutingRepositorySQLC) CopyRoute(ctx context.Context, sourceRouteID int64, destino *entity.ManufacturingRoute) (*entity.ManufacturingRoute, entity.RouteCopyResult, error) {
	var res entity.RouteCopyResult
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, res, err
	}
	var criado sqlc.ManufacturingRoute
	err = r.q.InTx(ctx, func(q *sqlc.Queries) error {
		var errTx error
		criado, errTx = q.CreateRoute(ctx, sqlc.CreateRouteParams{
			Code:         destino.Code,
			ItemCode:     itemDoRoteiro(destino),
			Mask:         pgutil.ToPgTextFromPtr(destino.Mask),
			Alternative:  destino.Alternative,
			Description:  pgutil.ToPgTextFromPtr(destino.Description),
			Situation:    sqltypes.RouteSituationEnum(destino.Situation),
			IsStandard:   destino.IsStandard,
			ValidFrom:    pgutil.ToPgDateFromPtr(destino.ValidFrom),
			ValidTo:      pgutil.ToPgDateFromPtr(destino.ValidTo),
			CreatedBy:    pgutil.ToPgUUID(destino.CreatedBy),
			EnterpriseID: empresa,
			IsTemplate:   destino.IsTemplate,
			LotMin:       pgutil.ToPgNumericFromDecimalPtr(destino.LotMin),
			LotMax:       pgutil.ToPgNumericFromDecimalPtr(destino.LotMax),
		})
		if errTx != nil {
			return errTx
		}
		if criado.IsStandard {
			if errTx = q.ClearOverlappingStandardRoutes(ctx, sqlc.ClearOverlappingStandardRoutesParams{ID: criado.ID, EnterpriseID: empresa}); errTx != nil {
				return errTx
			}
		}
		ids := struct{ src, dst int64 }{sourceRouteID, criado.ID}
		if res.Steps, errTx = q.CopyRouteSteps(ctx, sqlc.CopyRouteStepsParams{TargetRouteID: ids.dst, SourceRouteID: ids.src, EnterpriseID: empresa}); errTx != nil {
			return fmt.Errorf("copiando as etapas: %w", errTx)
		}
		if res.Steps == 0 {
			return errorsuc.NewValidationError("o roteiro de origem não tem etapas para copiar (ou não é desta empresa)")
		}
		if res.Dependencies, errTx = q.CopyRouteNetwork(ctx, sqlc.CopyRouteNetworkParams{TargetRouteID: ids.dst, SourceRouteID: ids.src, EnterpriseID: empresa}); errTx != nil {
			return fmt.Errorf("copiando a rede: %w", errTx)
		}
		if res.Resources, errTx = q.CopyRouteResources(ctx, sqlc.CopyRouteResourcesParams{TargetRouteID: ids.dst, SourceRouteID: ids.src, EnterpriseID: empresa}); errTx != nil {
			return fmt.Errorf("copiando os centros alternativos: %w", errTx)
		}
		if res.Tools, errTx = q.CopyRouteTools(ctx, sqlc.CopyRouteToolsParams{TargetRouteID: ids.dst, SourceRouteID: ids.src, EnterpriseID: empresa}); errTx != nil {
			return fmt.Errorf("copiando as ferramentas: %w", errTx)
		}
		if res.Documents, errTx = q.CopyRouteStepDocuments(ctx, sqlc.CopyRouteStepDocumentsParams{TargetRouteID: ids.dst, SourceRouteID: ids.src, EnterpriseID: empresa, CreatedBy: pgutil.ToPgUUID(destino.CreatedBy)}); errTx != nil {
			return fmt.Errorf("copiando os documentos: %w", errTx)
		}
		// Plano de inspeção é do ITEM: modelo não tem item, então não leva.
		if destino.IsTemplate || destino.ItemCode <= 0 {
			return nil
		}
		planos, errTx := q.ListActiveInspectionPlanIDsByRoute(ctx, sqlc.ListActiveInspectionPlanIDsByRouteParams{RouteID: ids.src, EnterpriseID: empresa})
		if errTx != nil {
			return fmt.Errorf("lendo os planos de inspeção: %w", errTx)
		}
		for _, plano := range planos {
			novo, errPlano := q.CopyRouteInspectionPlan(ctx, sqlc.CopyRouteInspectionPlanParams{
				TargetItemCode: destino.ItemCode, CreatedBy: pgutil.ToPgUUID(destino.CreatedBy),
				EnterpriseID: empresa, TargetRouteID: ids.dst, SourcePlanID: plano,
			})
			if errPlano != nil {
				return fmt.Errorf("copiando o plano de inspeção %d: %w", plano, errPlano)
			}
			if _, errPlano = q.CopyInspectionCharacteristics(ctx, sqlc.CopyInspectionCharacteristicsParams{TargetPlanID: novo, SourcePlanID: plano}); errPlano != nil {
				return fmt.Errorf("copiando as características do plano %d: %w", plano, errPlano)
			}
			res.Inspections++
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_mfg_routes_unique" {
			return nil, res, errorsuc.NewConflictError(
				"o item de destino já tem um roteiro com esta alternativa e máscara; escolha outra alternativa")
		}
		return nil, res, err
	}
	return routeRowToEntity(criado), res, nil
}
