package production_order_uc

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"math"
	"strings"

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
	Routing interface {
		GetRouteForItem(context.Context, int64, string) (*routingentity.ManufacturingRoute, error)
		GetRouteByID(context.Context, int64) (*routingentity.ManufacturingRoute, error)
		GetRouteOperations(context.Context, int64) ([]*routingentity.RouteOperation, error)
		GetNetworkEdges(context.Context, int64) ([]*routingentity.NetworkEdge, error)
	}

	Q                 *sqlc.Queries
	Auth              ports.AuthService
	WithinTransaction func(context.Context, func(*sqlc.Queries) error) error
	MachinePlanning   interface {
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
	if uc.WithinTransaction == nil {
		return uc.explodeRoute(ctx, orderID, routeID)
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	created := false
	err = uc.WithinTransaction(ctx, func(q *sqlc.Queries) error {
		order, err := q.LockProductionExecution(ctx, sqlc.LockProductionExecutionParams{ID: orderID, EnterpriseID: &empresa})
		if err != nil {
			return errorsuc.NewValidationError("ordem não encontrada na empresa autenticada")
		}
		existing, err := q.ListProductionOrderOperations(ctx, orderID)
		if err != nil {
			return err
		}
		if len(existing) > 0 {

			return nil
		}
		if order.Status != "OPEN" && order.Status != "IN_PROGRESS" {
			return errorsuc.NewValidationError("não é possível gerar etapas para uma ordem encerrada")
		}
		if routeID <= 0 {
			if uc.Routing == nil {
				return errorsuc.NewValidationError("selecione o roteiro do item")
			}
			route, err := uc.Routing.GetRouteForItem(ctx, order.ItemCode, order.Mask)
			if err != nil {
				return errorsuc.NewValidationError("o item não possui roteiro aprovado para esta máscara")
			}
			routeID = route.ID
		}
		inner := *uc
		inner.Q = q
		inner.WithinTransaction = nil
		inner.MachinePlanning = nil
		_, err = inner.explodeRoute(ctx, orderID, routeID)
		created = err == nil
		return err
	})
	if err != nil {
		return nil, err
	}
	if created && uc.MachinePlanning != nil {
		if err := uc.MachinePlanning.ApplyMachinePlanToProduction(ctx, orderID); err != nil {
			return nil, err
		}
	}
	return uc.ListOperations(ctx, orderID)
}

func (uc *OrderOperationsUseCase) explodeRoute(ctx context.Context, orderID, routeID int64) ([]*response.ProductionOrderOperationResponse, error) {
	ops, err := uc.Q.GetRouteOpsForExplode(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("fetching route operations: %w", err)
	}

	entraPorEtapa := uc.quantidadePorEtapa(ctx, orderID, routeID, ops)
	if uc.Routing != nil {
		route, err := uc.Routing.GetRouteByID(ctx, routeID)
		if err != nil {
			return nil, err
		}
		qty, item, err := uc.Q.GetProductionOrderQty(ctx, orderID)
		if err != nil {
			return nil, err
		}
		if route.ItemCode != item {
			return nil, errorsuc.NewValidationError("o roteiro deve pertencer ao item da ordem")
		}
		resolved, err := uc.Routing.GetRouteOperations(ctx, routeID)
		if err != nil {
			return nil, err
		}
		edges, err := uc.Routing.GetNetworkEdges(ctx, routeID)
		if err != nil {
			return nil, err
		}
		if len(resolved) == 0 {
			return nil, errorsuc.NewValidationError("cadastre as etapas do roteiro antes de gerar a ordem")
		}
		if routingentity.CriticalPath(resolved, edges, qty).HasCycle() {
			return nil, errorsuc.NewValidationError("o roteiro contém um ciclo de dependências")
		}
		entraPorEtapa = routingentity.QuantidadePorOperacao(resolved, edges, qty)
		ops = make([]sqlc.DBRouteOpForExplode, 0, len(resolved))
		for _, op := range resolved {
			ops = append(ops, sqlc.DBRouteOpForExplode{ID: op.ID, Sequence: op.Sequence, OperationName: op.OperationName,
				WorkCenterID: pgutil.ToPgInt8Ptr(op.EffectiveWorkCenterID),
				PlannedHours: op.EffTime.Run * op.EffTime.Batches(entraPorEtapa[op.ID]), SetupHours: op.EffTime.Setup,
				EffectiveScrap: op.EffectiveScrap, InspectionRequired: op.InspectionRequired})
		}
	}

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

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if err := uc.Q.FreezeProductionRoutingDependencies(ctx, sqlc.FreezeProductionRoutingDependenciesParams{ID: orderID, EnterpriseID: &enterpriseID}); err != nil {
		return nil, err
	}
	if err := uc.Q.MarkProductionRoutingFrozen(ctx, sqlc.MarkProductionRoutingFrozenParams{ID: orderID, EnterpriseID: &enterpriseID}); err != nil {
		return nil, err
	}

	// A inspeção é um controle SOBRE a ordem, não um pré-requisito dela: falta de
	// plano vira aviso na etapa, não recusa da ordem.
	avisos, err := uc.abrirInspecoesDaOrdem(ctx, orderID, routeID, ops)
	if err != nil {
		return nil, err
	}
	for _, resp := range out {
		if aviso, marcada := avisos[int16(resp.Sequence)]; marcada {
			resp.Warnings = append(resp.Warnings, aviso)
		}
	}
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
// marcada no roteiro que tenha plano ativo, e devolve um AVISO por etapa marcada
// que não tenha.
//
// Nem recusar nem esquecer. A versão anterior recusava a ordem inteira: uma
// lacuna de cadastro no módulo de qualidade parava a fábrica, e o operador via
// "cadastre um plano ativo" sem poder resolver — o plano é de outra tela e de
// outra pessoa. A versão antes dela pulava em silêncio, e o controle de
// qualidade simplesmente não acontecia sem ninguém notar.
//
// A ordem nasce, e a etapa sem plano volta marcada. Quem quiser a trava ANTES
// tem a conferência de prontidão do roteiro (`/api/routing/routes/{id}/readiness`),
// que acusa a mesma falta enquanto o roteiro está sendo montado.
func (uc *OrderOperationsUseCase) abrirInspecoesDaOrdem(
	ctx context.Context, orderID, routeID int64, ops []sqlc.DBRouteOpForExplode,
) (map[int16]string, error) {
	temInspecao := false
	for _, op := range ops {
		if op.InspectionRequired {
			temInspecao = true
			break
		}
	}
	if !temInspecao {
		return nil, nil
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	etapas, err := uc.Q.ListInspectionStepsForOrderRoute(ctx, sqlc.ListInspectionStepsForOrderRouteParams{
		RouteID: routeID, EnterpriseID: empresa,
	})
	if err != nil {
		return nil, err
	}
	if uc.Auth == nil {
		return nil, errorsuc.NewValidationError("autenticação não configurada para abrir inspeções")
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	avisos := map[int16]string{}
	for _, etapa := range etapas {
		if !etapa.PlanID.Valid || etapa.ItemCode == nil {
			avisos[etapa.Sequence] = fmt.Sprintf(
				"esta etapa é ponto de inspeção, mas não há plano de inspeção ativo para ela: a ordem foi criada e a conferência não será aberta até o plano existir (etapa %d)",
				etapa.Sequence)
			continue
		}
		_, err = uc.Q.CreateQualityRecord(ctx, sqlc.CreateQualityRecordParams{
			PlanID:            etapa.PlanID.Int64,
			CreatedBy:         pgutil.ToPgUUID(actor),
			ProductionOrderID: pgutil.ToPgInt8Ptr(&orderID),
			ItemCode:          *etapa.ItemCode,
			// Quantidade zerada e resultado PENDENTE: o registro nasce aberto,
			// esperando o inspetor. Quem preenche é a tela de qualidade.
			InspectedQty: 0,
			Result:       sqlc.InspectionResultEnum("PENDENTE"),
		})
		if err != nil {
			return nil, err
		}
	}
	return avisos, nil
}

// ListOperations lists operations for a production order.
func (uc *OrderOperationsUseCase) ListOperations(ctx context.Context, orderID int64) ([]*response.ProductionOrderOperationResponse, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := uc.Q.GetProductionExecution(ctx, sqlc.GetProductionExecutionParams{ID: orderID, EnterpriseID: &empresa}); err != nil {
		return nil, errorsuc.NewValidationError("ordem não encontrada na empresa autenticada")
	}
	poos, err := uc.Q.ListProductionOrderOperations(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("listing operations: %w", err)
	}
	// Pendência de inspeção: dura até alguém cadastrar o plano, então é lida a
	// cada listagem. Computá-la só na explosão do roteiro fazia o aviso sumir no
	// primeiro recarregamento da tela — e a explosão devolve esta mesma listagem,
	// então o aviso nem chegava a aparecer uma vez.
	semPlano := map[int]bool{}
	pendentes, err := uc.Q.ListOrderStepsMissingInspectionPlan(ctx, sqlc.ListOrderStepsMissingInspectionPlanParams{
		ProductionOrderID: orderID, EnterpriseID: &empresa,
	})
	if err != nil {
		return nil, err
	}
	for _, seq := range pendentes {
		semPlano[int(seq)] = true
	}

	out := make([]*response.ProductionOrderOperationResponse, 0, len(poos))
	for _, poo := range poos {
		r := pooToResponse(poo)
		if semPlano[r.Sequence] {
			r.Warnings = append(r.Warnings, "esta etapa é ponto de inspeção e ainda não tem plano de inspeção ativo: a conferência de qualidade não será aberta até o plano existir")
		}
		blocked, err := uc.Q.OperationPredecessorsPending(ctx, poo.ID)
		if err != nil {
			return nil, err
		}
		r.CanStart = !blocked && (poo.Status == "PENDING" || poo.Status == "PAUSED" || poo.Status == "INTERRUPTED")
		events, err := uc.Q.ListProductionExecutionEvents(ctx, sqlc.ListProductionExecutionEventsParams{OperationID: poo.ID, EnterpriseID: empresa})
		if err != nil {
			return nil, err
		}
		for _, event := range events {
			r.ExecutionHistory = append(r.ExecutionHistory, response.ProductionOperationEvent{ID: event.ID, PreviousStatus: event.OldStatus, Status: event.NewStatus, Actor: event.ActorName, OccurredAt: event.OccurredAt.Time, ActualHours: pgutil.FromPgNumericToFloat64(event.ActualHoursDelta)})
		}
		out = append(out, r)
	}
	return out, nil
}

// AdvanceOperation changes an operation status (PENDING → IN_PROGRESS → DONE).
func (uc *OrderOperationsUseCase) AdvanceOperation(ctx context.Context, dto request.AdvanceOperationDTO) (*response.ProductionOrderOperationResponse, error) {
	if uc.Auth == nil || !uc.Auth.CanReportProduction(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.OperationID <= 0 {
		return nil, errorsuc.NewValidationError("informe a operação")
	}
	if math.IsNaN(dto.ActualHours) || math.IsInf(dto.ActualHours, 0) || dto.ActualHours < 0 || math.IsNaN(dto.ProducedQty) || math.IsInf(dto.ProducedQty, 0) || dto.ProducedQty < 0 {
		return nil, errorsuc.NewValidationError("horas e quantidade devem ser finitas e não negativas")
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if uc.WithinTransaction == nil {
		return nil, errorsuc.NewValidationError("execução transacional de etapas não configurada")
	}
	var result *response.ProductionOrderOperationResponse
	err = uc.WithinTransaction(ctx, func(q *sqlc.Queries) error {
		prior, err := q.LockOperationExecution(ctx, sqlc.LockOperationExecutionParams{ID: dto.OperationID, EnterpriseID: &empresa})
		if err != nil {
			return errorsuc.NewValidationError("etapa não encontrada na empresa autenticada")
		}
		if prior.OrderStatus != "IN_PROGRESS" {
			return errorsuc.NewValidationError("inicie a ordem antes de executar suas etapas; ordens encerradas não podem ser alteradas")
		}
		reason := strings.TrimSpace(dto.Reason)
		if err := orderentity.ValidateOperationTransition(prior.Status, dto.Status, reason); err != nil {
			return errorsuc.NewValidationError(err.Error())
		}
		if dto.Status == "IN_PROGRESS" || dto.Status == "SKIPPED" {
			blocked, err := q.OperationPredecessorsPending(ctx, dto.OperationID)
			if err != nil {
				return err
			}
			if blocked {
				return errorsuc.NewValidationError("conclua as etapas predecessoras antes de iniciar esta etapa")
			}
		}
		actor, err := uc.Auth.UserID(ctx)
		if err != nil {
			return err
		}
		if err := q.SetProductionExecutionActor(ctx, sqlc.SetProductionExecutionActorParams{Actor: actor.String(), Source: "OPERATION"}); err != nil {
			return err
		}
		if err := q.UpdateOperationExecution(ctx, sqlc.UpdateOperationExecutionParams{ID: dto.OperationID, Status: dto.Status, Hours: pgutil.ToPgNumericFromFloat64(dto.ActualHours), Reason: reason}); err != nil {
			return err
		}
		op, err := q.GetProductionOrderOperation(ctx, dto.OperationID)
		if err != nil {
			return err
		}
		result = pooToResponse(op)
		// Começar a etapa é o último instante em que trocar a ferramenta ainda é
		// barato: depois dela rodar, a peça já saiu fora de medida. O aviso não
		// bloqueia — quem está no posto pode ter um motivo —, mas aparece.
		if dto.Status == "IN_PROGRESS" && op.RouteOperationID.Valid {
			transactional := *uc
			transactional.Q = q
			result.ToolAlerts = transactional.ferramentasVencidas(ctx, op.RouteOperationID.Int64)
		}
		if dto.Status == "DONE" && op.RouteOperationID.Valid {
			transactional := *uc
			transactional.Q = q
			result.ToolAlerts, err = transactional.consumeToolLife(ctx, op.ID, op.RouteOperationID.Int64, dto.ProducedQty, pgutil.FromPgNumericToFloat64(op.ActualHours))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

// ferramentasVencidas lista as ferramentas da etapa que já passaram do limite de
// vida. É consulta de aviso: falhar aqui não pode impedir o operador de
// começar a etapa — a ordem existe, o alerta é um cuidado sobre ela.
func (uc *OrderOperationsUseCase) ferramentasVencidas(ctx context.Context, routeOpID int64) []string {
	tools, err := uc.Q.ListToolsByRouteOp(ctx, routeOpID)
	if err != nil {
		return nil
	}
	var alerts []string
	for _, t := range tools {
		limite := pgutil.FromPgNumericToFloat64(t.LifeLimit)
		usado := pgutil.FromPgNumericToFloat64(t.LifeUsed)
		if limite <= 0 || usado < limite {
			continue
		}
		alerts = append(alerts, fmt.Sprintf(
			"ferramenta %d (%s) já usou %.0f de %.0f %s de vida útil: troque antes de produzir",
			t.ToolCode, t.ToolName, usado, limite, strings.ToLower(t.LifeType)))
	}
	return alerts
}

// consumeToolLife charges each tool linked to the route operation for the work just
// completed (produced pieces for GOLPES/PECAS, actual hours for HORAS) and returns
// alerts for tools that reached their useful-life limit. When the tool production
// sheet has bound a physical serial to this operation/tool, the same amount is
// charged to that serial too, so per-instance wear stays in sync with the master.
func (uc *OrderOperationsUseCase) consumeToolLife(ctx context.Context, operationID, routeOpID int64, produced, hours float64) ([]string, error) {
	tools, err := uc.Q.ListToolsByRouteOp(ctx, routeOpID)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		// Charge the physical serial bound to this operation/tool, if any.
		if binding, err := uc.Q.GetOperationToolSerial(ctx, sqlc.GetOperationToolSerialParams{
			OperationID: operationID, ToolID: t.ToolID,
		}); err == nil {
			_, err = uc.Q.ConsumeToolSerialLife(ctx, sqlc.ConsumeToolSerialLifeParams{
				ID:       binding.ToolSerialID,
				LifeUsed: pgutil.ToPgNumericFromFloat64(amount),
			})
			if err != nil {
				return nil, err
			}
		}
		limit := pgutil.FromPgNumericToFloat64(updated.LifeLimit)
		used := pgutil.FromPgNumericToFloat64(updated.LifeUsed)
		if limit > 0 && used >= limit {
			alerts = append(alerts, fmt.Sprintf(
				"ferramenta %d (%s) atingiu o limite de vida útil (%.0f/%.0f %s) — troca necessária",
				updated.Code, updated.Name, used, limit, updated.LifeType))
		}
	}
	return alerts, nil
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
