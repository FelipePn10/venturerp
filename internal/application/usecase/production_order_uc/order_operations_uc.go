package production_order_uc

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	toolentity "github.com/FelipePn10/panossoerp/internal/domain/tool/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// OrderOperationsUseCase manages production order operations (exploding route + advancing status).
type OrderOperationsUseCase struct {
	Q               *sqlc.Queries
	MachinePlanning interface {
		ApplyMachinePlanToProduction(context.Context, int64) error
	}
}

// ExplodeRoute creates production_order_operations from a manufacturing route.
// Called after creating a production order when route_id is provided.
//
// Cada etapa nasce com a quantidade que precisa PROCESSAR, não com a quantidade
// da ordem. Com refugo as duas divergem: para entregar 100 boas, a etapa final
// recebe 100 e a primeira recebe mais, acumulando o refugo de todas as
// seguintes. Sem isso o apontamento cobraria 100 peças de uma operação que
// tinha de fazer 106, e a ordem fecharia faltando peça.
func (uc *OrderOperationsUseCase) ExplodeRoute(ctx context.Context, orderID, routeID int64) ([]*response.ProductionOrderOperationResponse, error) {
	ops, err := uc.Q.GetRouteOpsForExplode(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("fetching route operations: %w", err)
	}

	entraPorEtapa := uc.quantidadePorEtapa(ctx, orderID, routeID, ops)

	out := make([]*response.ProductionOrderOperationResponse, 0, len(ops))
	for _, op := range ops {
		poo, err := uc.Q.CreateProductionOrderOperation(ctx, sqlc.CreateProductionOrderOperationParams{
			ProductionOrderID: orderID,
			RouteOperationID:  pgutil.ToPgInt8Ptr(&op.ID),
			Sequence:          op.Sequence,
			OperationName:     op.OperationName,
			WorkCenterID:      op.WorkCenterID,
			PlannedHours:      pgutil.ToPgNumericFromFloat64(op.PlannedHours),
			SetupHours:        pgutil.ToPgNumericFromFloat64(op.SetupHours),
			PlannedQty:        pgutil.ToPgNumericFromFloat64(entraPorEtapa[op.ID]),
		})
		if err != nil {
			return nil, fmt.Errorf("creating order operation seq %d: %w", op.Sequence, err)
		}
		out = append(out, pooToResponse(poo))
	}

	// Etapas marcadas como ponto de inspeção abrem o registro de inspeção junto
	// com a ordem. Falhar aqui não invalida a ordem: a ordem existe, a inspeção
	// é um controle sobre ela — derrubar a criação da ordem por causa do plano
	// de qualidade deixaria a fábrica sem ordem nenhuma.
	uc.abrirInspecoesDaOrdem(ctx, orderID, routeID, ops)
	if uc.MachinePlanning != nil {
		if err := uc.MachinePlanning.ApplyMachinePlanToProduction(ctx, orderID); err != nil {
			return nil, err
		}
		return uc.ListOperations(ctx, orderID)
	}
	return out, nil
}

// quantidadePorEtapa roda a cascata de refugo do roteiro sobre a quantidade boa
// da ordem, usando a MESMA função de domínio que o roteiro e o MRP usam.
// Se a quantidade da ordem não puder ser lida, cada etapa fica com a quantidade
// que o roteiro entrega — nunca com zero, que faria o apontamento parecer
// concluído antes de começar.
func (uc *OrderOperationsUseCase) quantidadePorEtapa(
	ctx context.Context, orderID, routeID int64, ops []sqlc.DBRouteOpForExplode,
) map[int64]float64 {
	boas, _, err := uc.Q.GetProductionOrderQty(ctx, orderID)
	if err != nil || boas <= 0 {
		boas = 1
	}

	etapas := make([]*routingentity.RouteOperation, 0, len(ops))
	for _, op := range ops {
		etapas = append(etapas, &routingentity.RouteOperation{
			ID: op.ID, Sequence: op.Sequence, EffectiveScrap: op.EffectiveScrap,
		})
	}

	var arestas []*routingentity.NetworkEdge
	if edges, errRede := uc.Q.GetRouteNetworkForExplode(ctx, routeID); errRede == nil {
		for _, e := range edges {
			arestas = append(arestas, &routingentity.NetworkEdge{
				PredecessorID: e.PredecessorID, SuccessorID: e.SuccessorID, OverlapPct: e.OverlapPct,
			})
		}
	}
	return routingentity.QuantidadePorOperacao(etapas, arestas, boas)
}

// abrirInspecoesDaOrdem cria um registro de inspeção PENDENTE para cada etapa
// marcada no roteiro que tenha plano de inspeção ativo. Etapa marcada sem plano
// não gera registro: o registro precisa dizer o que inspecionar.
func (uc *OrderOperationsUseCase) abrirInspecoesDaOrdem(
	ctx context.Context, orderID, routeID int64, ops []sqlc.DBRouteOpForExplode,
) {
	temInspecao := false
	for _, op := range ops {
		if op.InspectionRequired {
			temInspecao = true
			break
		}
	}
	if !temInspecao {
		return
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return
	}
	etapas, err := uc.Q.ListInspectionStepsForOrderRoute(ctx, sqlc.ListInspectionStepsForOrderRouteParams{
		RouteID: routeID, EnterpriseID: empresa,
	})
	if err != nil {
		return
	}
	for _, etapa := range etapas {
		if !etapa.PlanID.Valid || etapa.ItemCode == nil {
			continue // etapa marcada mas ainda sem plano: nada a inspecionar
		}
		_, _ = uc.Q.CreateQualityRecord(ctx, sqlc.CreateQualityRecordParams{
			PlanID:            etapa.PlanID.Int64,
			ProductionOrderID: pgutil.ToPgInt8Ptr(&orderID),
			ItemCode:          *etapa.ItemCode,
			// Quantidade zerada e resultado PENDENTE: o registro nasce aberto,
			// esperando o inspetor. Quem preenche é a tela de qualidade.
			InspectedQty: 0,
			Result:       sqlc.InspectionResultEnum("PENDENTE"),
		})
	}
}

// ListOperations lists operations for a production order.
func (uc *OrderOperationsUseCase) ListOperations(ctx context.Context, orderID int64) ([]*response.ProductionOrderOperationResponse, error) {
	poos, err := uc.Q.ListProductionOrderOperations(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("listing operations: %w", err)
	}
	out := make([]*response.ProductionOrderOperationResponse, 0, len(poos))
	for _, poo := range poos {
		out = append(out, pooToResponse(poo))
	}
	return out, nil
}

// AdvanceOperation changes an operation status (PENDING → IN_PROGRESS → DONE).
func (uc *OrderOperationsUseCase) AdvanceOperation(ctx context.Context, dto request.AdvanceOperationDTO) (*response.ProductionOrderOperationResponse, error) {
	if dto.OperationID == 0 {
		return nil, errorsuc.NewValidationError("informe a operação")
	}
	switch dto.Status {
	case "PENDING", "IN_PROGRESS", "DONE", "SKIPPED":
	default:
		return nil, errorsuc.NewValidationError(fmt.Sprintf("situação %q inválida: use pendente, em andamento, concluída ou dispensada", dto.Status))
	}
	// Capture the prior status so tool-life is consumed only on the real transition
	// INTO DONE (advancing an already-DONE operation must not double-consume).
	prior, priorErr := uc.Q.GetProductionOrderOperation(ctx, dto.OperationID)
	wasDone := priorErr == nil && prior.Status == "DONE"

	poo, err := uc.Q.AdvanceProductionOrderOperation(ctx, dto.OperationID, dto.Status)
	if err != nil {
		return nil, fmt.Errorf("advancing operation: %w", err)
	}
	if dto.ActualHours > 0 {
		_ = uc.Q.AddActualHours(ctx, dto.OperationID, pgutil.ToPgNumericFromFloat64(dto.ActualHours))
	}
	resp := pooToResponse(poo)
	// On the first completion, consume the useful life of the tools used by this
	// operation and surface any that reached their replacement limit.
	if dto.Status == "DONE" && !wasDone && poo.RouteOperationID.Valid {
		resp.ToolAlerts = uc.consumeToolLife(ctx, poo.ID, poo.RouteOperationID.Int64, dto.ProducedQty, dto.ActualHours)
	}
	return resp, nil
}

// consumeToolLife charges each tool linked to the route operation for the work just
// completed (produced pieces for GOLPES/PECAS, actual hours for HORAS) and returns
// alerts for tools that reached their useful-life limit. When the tool production
// sheet has bound a physical serial to this operation/tool, the same amount is
// charged to that serial too, so per-instance wear stays in sync with the master.
func (uc *OrderOperationsUseCase) consumeToolLife(ctx context.Context, operationID, routeOpID int64, produced, hours float64) []string {
	tools, err := uc.Q.ListToolsByRouteOp(ctx, routeOpID)
	if err != nil {
		return nil
	}
	var alerts []string
	for _, t := range tools {
		amount := produced
		if t.LifeType == toolentity.LifeHours {
			amount = hours
		}
		if amount <= 0 {
			continue
		}
		updated, err := uc.Q.ConsumeToolLife(ctx, sqlc.ConsumeToolLifeParams{
			ID:       t.ToolID,
			LifeUsed: pgutil.ToPgNumericFromFloat64(amount),
		})
		if err != nil {
			continue
		}
		// Charge the physical serial bound to this operation/tool, if any.
		if binding, err := uc.Q.GetOperationToolSerial(ctx, sqlc.GetOperationToolSerialParams{
			OperationID: operationID, ToolID: t.ToolID,
		}); err == nil {
			_, _ = uc.Q.ConsumeToolSerialLife(ctx, sqlc.ConsumeToolSerialLifeParams{
				ID:       binding.ToolSerialID,
				LifeUsed: pgutil.ToPgNumericFromFloat64(amount),
			})
		}
		limit := pgutil.FromPgNumericToFloat64(updated.LifeLimit)
		used := pgutil.FromPgNumericToFloat64(updated.LifeUsed)
		if limit > 0 && used >= limit {
			alerts = append(alerts, fmt.Sprintf(
				"ferramenta %d (%s) atingiu o limite de vida útil (%.0f/%.0f %s) — troca necessária",
				updated.Code, updated.Name, used, limit, updated.LifeType))
		}
	}
	return alerts
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func pooToResponse(p sqlc.DBProductionOrderOperation) *response.ProductionOrderOperationResponse {
	r := &response.ProductionOrderOperationResponse{
		ID:                p.ID,
		ProductionOrderID: p.ProductionOrderID,
		Sequence:          int(p.Sequence),
		OperationName:     p.OperationName,
		PlannedHours:      pgutil.FromPgNumericToFloat64(p.PlannedHours),
		SetupHours:        pgutil.FromPgNumericToFloat64(p.SetupHours),
		ActualHours:       pgutil.FromPgNumericToFloat64(p.ActualHours),
		Status:            p.Status,
	}
	if p.RouteOperationID.Valid {
		v := p.RouteOperationID.Int64
		r.RouteOperationID = &v
	}
	if p.WorkCenterID.Valid {
		v := p.WorkCenterID.Int64
		r.WorkCenterID = &v
	}
	if p.StartedAt.Valid {
		t := pgutil.FromPgTimestamptz(p.StartedAt)
		r.StartedAt = &t
	}
	if p.CompletedAt.Valid {
		t := pgutil.FromPgTimestamptz(p.CompletedAt)
		r.CompletedAt = &t
	}
	if p.Notes.Valid {
		r.Notes = &p.Notes.String
	}
	return r
}
