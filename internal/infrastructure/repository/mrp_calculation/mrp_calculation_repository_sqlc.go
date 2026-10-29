package mrp_calculation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/shopspring/decimal"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepository "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *MRPCalculationRepositorySQLC) CreateProfile(
	ctx context.Context,
	p *entity.MRPItemProfile,
) (*entity.MRPItemProfile, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.CreateMRPItemProfile(ctx, sqlc.CreateMRPItemProfileParams{
		ItemCode:        p.ItemCode,
		PlanCode:        p.PlanCode,
		CalculationDate: pgutil.ToPgDate(p.CalculationDate),
		Demand:          pgutil.ToPgNumericFromDecimal(p.Demand),
		OrdersPlanned:   pgutil.ToPgNumericFromDecimal(p.OrdersPlanned),
		OrdersFirm:      pgutil.ToPgNumericFromDecimal(p.OrdersFirm),
		StockProjected:  pgutil.ToPgNumericFromDecimal(p.StockProjected),
		Llc:             int32(p.LLC),
		NeedDate:        pgutil.ToPgDate(p.NeedDate),
		EnterpriseID:    enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating MRP item profile: %w", err)
	}

	return profileToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) CreateProfileDetail(ctx context.Context, detail *entity.MRPProfileDetail) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if err := r.q.CreateMRPProfileDetail(ctx, sqlc.CreateMRPProfileDetailParams{
		PlanCode: detail.PlanCode, ItemCode: detail.ItemCode, NeedDate: pgutil.ToPgDate(detail.NeedDate),
		DetailType: detail.DetailType, SourceCode: detail.SourceCode, ParentItemCode: detail.ParentItemCode,
		Quantity: pgutil.ToPgNumericFromDecimal(detail.Quantity), EnterpriseID: enterpriseID,
	}); err != nil {
		return fmt.Errorf("creating MRP profile detail: %w", err)
	}
	return nil
}

func (r *MRPCalculationRepositorySQLC) DeleteProfileDetailsByPlan(ctx context.Context, planCode int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteMRPProfileDetailsByPlan(ctx, sqlc.DeleteMRPProfileDetailsByPlanParams{PlanCode: planCode, EnterpriseID: enterpriseID})
}

func (r *MRPCalculationRepositorySQLC) GetProfiles(
	ctx context.Context,
	itemCode,
	planCode int64,
) ([]*entity.MRPItemProfile, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.q.GetMRPItemProfiles(ctx, sqlc.GetMRPItemProfilesParams{
		ItemCode:     itemCode,
		PlanCode:     planCode,
		EnterpriseID: enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching MRP item profiles: %w", err)
	}

	return profilesToEntities(rows), nil
}

func (r *MRPCalculationRepositorySQLC) DeleteProfilesByPlan(
	ctx context.Context,
	planID int64,
) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteProfilesByPlan(ctx, sqlc.DeleteProfilesByPlanParams{PlanCode: planID, EnterpriseID: enterpriseID})
}

func (r *MRPCalculationRepositorySQLC) StartCalculation(
	ctx context.Context,
	planID int64,
) (*entity.MRPCalculationLog, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.StartMRPCalculation(ctx, sqlc.StartMRPCalculationParams{PlanCode: planID, EnterpriseID: enterpriseID})
	if err == nil {
		return logToEntity(row), nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "uq_mrp_single_running_calculation" {
		return nil, fmt.Errorf("starting MRP calculation: %w", err)
	}

	// A trava impede dois cálculos simultâneos no mesmo plano — correto. Mas um
	// cálculo que morreu no meio (o serviço reiniciou, o processo caiu) deixa o
	// registro RODANDO para sempre, e a partir daí TODA tentativa de planejar
	// devolve "já existe um cálculo em andamento". Da tela não há saída: o
	// planejamento fica bloqueado até alguém editar o banco.
	//
	// Um cálculo real leva segundos. Passado o limite abaixo, o que está ali é
	// destroço de uma execução interrompida: encerrar e seguir é o que o
	// usuário faria se pudesse.
	if _, errLiberar := r.q.AbandonarCalculoTravado(ctx, sqlc.AbandonarCalculoTravadoParams{
		PlanCode:     planID,
		Column2:      pgtype.Interval{Microseconds: int64(limiteDeCalculoTravado / time.Microsecond), Valid: true},
		EnterpriseID: enterpriseID,
	}); errLiberar != nil {
		return nil, fmt.Errorf("liberando cálculo do MRP interrompido: %w", errLiberar)
	}
	row, err = r.q.StartMRPCalculation(ctx, sqlc.StartMRPCalculationParams{PlanCode: planID, EnterpriseID: enterpriseID})
	if err == nil {
		return logToEntity(row), nil
	}
	// Só a mesma trava significa "em andamento" (o cálculo ainda é recente);
	// qualquer outra falha é do banco e tem de aparecer como tal.
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "uq_mrp_single_running_calculation" {
		return nil, fmt.Errorf("starting MRP calculation: %w", err)
	}
	return nil, mrprepository.ErrCalculationInProgress
}

// limiteDeCalculoTravado é a idade a partir da qual um cálculo RODANDO é tratado
// como interrompido. Generoso de propósito: o cálculo mais pesado observado
// termina em segundos, e liberar cedo demais permitiria dois cálculos
// concorrentes gravando sugestões no mesmo plano.
const limiteDeCalculoTravado = 30 * time.Minute

func (r *MRPCalculationRepositorySQLC) FinishCalculation(
	ctx context.Context,
	logCode int64,
	status string,
	errorsMap map[string]interface{},
	totalItems,
	totalOrders int,
) (*entity.MRPCalculationLog, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	errorsJSON, _ := json.Marshal(errorsMap)

	row, err := r.q.FinishMRPCalculation(ctx, sqlc.FinishMRPCalculationParams{
		Code:         pgutil.ToPgInt8Ptr(&logCode),
		Status:       status,
		Errors:       errorsJSON,
		TotalItems:   int32(totalItems),
		TotalOrders:  int32(totalOrders),
		EnterpriseID: enterpriseID,
	})

	if err != nil {
		return nil, fmt.Errorf("finishing MRP calculation: %w", err)
	}

	return logToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) GetCalculationLog(
	ctx context.Context,
	logCode int64,
) (*entity.MRPCalculationLog, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.GetMRPCalculationLog(
		ctx,
		sqlc.GetMRPCalculationLogParams{Code: pgutil.ToPgInt8Ptr(&logCode), EnterpriseID: enterpriseID},
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("registro de cálculo %d não encontrado", logCode))
		}

		return nil, fmt.Errorf("fetching calculation log: %w", err)
	}

	return logToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) ListCalculationLogs(
	ctx context.Context,
	planID int64,
) ([]*entity.MRPCalculationLog, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListMRPCalculationLogsByPlan(ctx, sqlc.ListMRPCalculationLogsByPlanParams{PlanCode: planID, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("listing calculation logs: %w", err)
	}

	return logsToEntities(rows), nil
}

func (r *MRPCalculationRepositorySQLC) CreateStockSnapshot(
	ctx context.Context,
	s *entity.StockSnapshot,
) (*entity.StockSnapshot, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.CreateStockSnapshot(ctx, sqlc.CreateStockSnapshotParams{
		ItemCode:      s.ItemCode,
		WarehouseCode: s.WarehouseCode,
		Quantity:      pgutil.ToPgNumericFromDecimal(s.Quantity),
		ReservedQty:   pgutil.ToPgNumericFromDecimal(s.ReservedQty),
		SafetyStock:   pgutil.ToPgNumericFromDecimal(s.SafetyStock),
		EnterpriseID:  enterpriseID,
	})

	if err != nil {
		return nil, fmt.Errorf("creating stock snapshot: %w", err)
	}

	return snapshotToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) GetStockSnapshot(
	ctx context.Context,
	itemCode int64,
) (*entity.StockSnapshot, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	// Try the manual snapshot table first.
	row, err := r.q.GetStockSnapshot(ctx, sqlc.GetStockSnapshotParams{ItemCode: itemCode, EnterpriseID: enterpriseID})
	if err == nil {
		return snapshotToEntity(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("fetching stock snapshot: %w", err)
	}

	// Fall back to live stock_balances aggregated per item.
	var qty, reservedQty, safetyStock decimal.Decimal
	err = r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(quantity), 0),
			COALESCE(SUM(reserved_qty), 0),
			COALESCE(SUM(safety_stock), 0)
		FROM stock_balances b
		WHERE item_code = $1 AND enterprise_id = $2 AND `+saldoPlanejavel+`
	`, itemCode, enterpriseID).Scan(&qty, &reservedQty, &safetyStock)
	if err != nil {
		return nil, nil
	}
	return &entity.StockSnapshot{
		ItemCode:    itemCode,
		Quantity:    qty,
		ReservedQty: reservedQty,
		SafetyStock: safetyStock,
	}, nil
}

// saldoPlanejavel: saldo que o MRP conta como disponível. Rejeição e
// almoxarifado sem disposição (sucata, avaria) não abastecem a produção; o
// saldo em inspeção conta, como no SAP — ele é liberado pela decisão de uso, e
// a parte recusada sai dele para a rejeição.
const saldoPlanejavel = `NOT EXISTS (SELECT 1 FROM warehouse w WHERE w.id = b.warehouse_id AND w.enterprise_id = b.enterprise_id
	AND (NOT w.disposition AND w.location::text <> 'INSPECAO' OR w.location::text = 'REJEICAO'))`

func (r *MRPCalculationRepositorySQLC) CreateSalesOrderDemand(
	ctx context.Context,
	d *entity.SalesOrderDemand,
) (*entity.SalesOrderDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	mask := ""
	if d.Mask != nil {
		mask = *d.Mask
	}

	row, err := r.q.CreateSalesOrderDemand(ctx, sqlc.CreateSalesOrderDemandParams{
		SalesOrderCode: d.SalesOrderCode,
		ItemCode:       d.ItemCode,
		Mask:           mask,
		Quantity:       pgutil.ToPgNumericFromDecimal(d.Quantity),
		DeliveredQty:   pgutil.ToPgNumericFromDecimal(d.DeliveredQty),
		DeliveryDate:   pgutil.ToPgDate(d.DeliveryDate),
		DivisionCode:   d.DivisionCode,
		Status:         d.Status,
		EnterpriseID:   enterpriseID,
	})

	if err != nil {
		return nil, fmt.Errorf("creating sales order demand: %w", err)
	}

	return salesDemandToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) GetSalesOrderDemand(
	ctx context.Context,
	code int64,
) (*entity.SalesOrderDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.GetSalesOrderDemandByCode(
		ctx,
		sqlc.GetSalesOrderDemandByCodeParams{Code: pgutil.ToPgInt8Ptr(&code), EnterpriseID: enterpriseID},
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("demanda %d do pedido de venda não encontrada", code))
		}

		return nil, fmt.Errorf("fetching sales order demand: %w", err)
	}

	return salesDemandToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) ListSalesOrderDemandsByItem(
	ctx context.Context,
	itemCode int64,
) ([]*entity.SalesOrderDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListSalesOrderDemandsByItem(ctx, sqlc.ListSalesOrderDemandsByItemParams{ItemCode: itemCode, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("listing sales order demands: %w", err)
	}

	return salesDemandsToEntities(rows), nil
}

func (r *MRPCalculationRepositorySQLC) UpdateSalesOrderDemandStatus(
	ctx context.Context,
	code int64,
	status string,
	deliveredQty decimal.Decimal,
) (*entity.SalesOrderDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.UpdateSalesOrderDemandStatus(
		ctx,
		sqlc.UpdateSalesOrderDemandStatusParams{
			Status:       status,
			DeliveredQty: pgutil.ToPgNumericFromDecimal(deliveredQty),
			Code:         pgutil.ToPgInt8Ptr(&code),
			EnterpriseID: enterpriseID,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("updating sales order demand status: %w", err)
	}

	return salesDemandToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) CreateConfiguredItemRule(
	ctx context.Context,
	rule *entity.ConfiguredItemRule,
) (*entity.ConfiguredItemRule, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.CreateConfiguredItemRule(
		ctx,
		sqlc.CreateConfiguredItemRuleParams{
			ItemCode:     rule.ItemCode,
			TableType:    rule.TableType,
			FieldName:    rule.FieldName,
			RuleType:     rule.RuleType,
			RuleValue:    rule.RuleValue,
			Sequence:     int32(rule.Sequence),
			CreatedBy:    pgutil.ToPgUUID(rule.CreatedBy),
			EnterpriseID: enterpriseID,
			CondicaoMascara: pgtype.Text{String: strings.TrimSpace(rule.CondicaoMascara),
				Valid: strings.TrimSpace(rule.CondicaoMascara) != ""},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("creating configured item rule: %w", err)
	}

	return ruleToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) GetConfiguredItemRules(
	ctx context.Context,
	itemCode int64,
) ([]*entity.ConfiguredItemRule, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetConfiguredItemRules(ctx, sqlc.GetConfiguredItemRulesParams{ItemCode: itemCode, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("fetching configured item rules: %w", err)
	}

	return rulesToEntities(rows), nil
}

func (r *MRPCalculationRepositorySQLC) DeleteConfiguredItemRule(
	ctx context.Context,
	id int64,
) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteConfiguredItemRule(
		ctx,
		sqlc.DeleteConfiguredItemRuleParams{Code: pgutil.ToPgInt8Ptr(&id), EnterpriseID: enterpriseID},
	)
}

func (r *MRPCalculationRepositorySQLC) CreatePlannedOrderSuggestion(
	ctx context.Context,
	s *entity.PlannedOrderSuggestion,
) (*entity.PlannedOrderSuggestion, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	startDate := pgutil.ToPgDate(s.NeedDate)
	if s.StartDate != nil {
		startDate = pgutil.ToPgDate(*s.StartDate)
	}

	row, err := r.q.CreateMRPPlannedSuggestion(ctx, sqlc.CreateMRPPlannedSuggestionParams{
		OrderNumber:          s.OrderNumber,
		PlanCode:             s.PlanCode,
		ItemCode:             s.ItemCode,
		Mask:                 s.Mask,
		Quantity:             pgutil.ToPgNumericFromDecimal(s.Quantity),
		NeedDate:             pgutil.ToPgDate(s.NeedDate),
		StartDate:            startDate,
		OrderType:            s.OrderType,
		DemandType:           s.DemandType,
		ParentItemCode:       s.ParentItemCode,
		Llc:                  int32(s.LLC),
		WarehouseCode:        s.WarehouseCode,
		InterFactory:         s.InterFactory,
		SourceEnterpriseCode: s.SourceEnterpriseCode,
		AutoRelease:          s.AutoRelease,
		RouteOperationID:     s.RouteOperationID,
		OperationID:          s.OperationID,
		SupplierCode:         s.SupplierCode,
		ServiceItemCode:      s.ServiceItemCode,
		RemittanceType:       pgutil.ToPgTextFromPtr(s.RemittanceType),
		EnterpriseID:         enterpriseID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, mrprepository.ErrOrderNumberInUse
	}
	if err != nil {
		return nil, fmt.Errorf("creating planned suggestion: %w", err)
	}

	return suggestionToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) GetSuggestionByCode(
	ctx context.Context,
	code int64,
) (*entity.PlannedOrderSuggestion, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	var row sqlc.MrpPlannedSuggestion
	err = r.db.QueryRow(ctx,
		`SELECT code, order_number, plan_code, item_code, mask, quantity, need_date, start_date,
		        order_type, demand_type, parent_item_code, llc, notes, warehouse_code,
		        inter_factory, source_enterprise_code, auto_release, route_operation_id,
		        operation_id, supplier_code, service_item_code, remittance_type
		   FROM public.mrp_planned_suggestions WHERE code = $1 AND enterprise_id = $2`, code, enterpriseID,
	).Scan(&row.Code, &row.OrderNumber, &row.PlanCode, &row.ItemCode, &row.Mask, &row.Quantity, &row.NeedDate, &row.StartDate,
		&row.OrderType, &row.DemandType, &row.ParentItemCode, &row.Llc, &row.Notes, &row.WarehouseCode,
		&row.InterFactory, &row.SourceEnterpriseCode, &row.AutoRelease, &row.RouteOperationID,
		&row.OperationID, &row.SupplierCode, &row.ServiceItemCode, &row.RemittanceType)
	if err != nil {
		return nil, fmt.Errorf("getting suggestion %d: %w", code, err)
	}
	out := suggestionToEntity(row)
	if err := r.loadMachineAllocations(ctx, []*entity.PlannedOrderSuggestion{out}); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *MRPCalculationRepositorySQLC) ListSuggestionsByPlan(
	ctx context.Context,
	planCode int64,
) ([]*entity.PlannedOrderSuggestion, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListMRPPlannedSuggestions(ctx, sqlc.ListMRPPlannedSuggestionsParams{PlanCode: planCode, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("listing planned suggestions: %w", err)
	}

	out := suggestionsToEntities(rows)
	if err := r.loadMachineAllocations(ctx, out); err != nil {
		return nil, err
	}
	if err := r.loadFirmedOrders(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *MRPCalculationRepositorySQLC) DeleteSuggestionsByPlan(
	ctx context.Context,
	planCode int64,
) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteMRPPlannedSuggestions(ctx, sqlc.DeleteMRPPlannedSuggestionsParams{PlanCode: planCode, EnterpriseID: enterpriseID})
}

func (r *MRPCalculationRepositorySQLC) UpdateItemLLCs(ctx context.Context, llcs map[int64]int) error {
	itemCodes := make([]int64, 0, len(llcs))
	levels := make([]int32, 0, len(llcs))
	for itemCode, level := range llcs {
		itemCodes = append(itemCodes, itemCode)
		levels = append(levels, int32(level))
	}
	if len(itemCodes) == 0 {
		return nil
	}
	if err := r.q.UpdateItemPlanningLLCs(ctx, sqlc.UpdateItemPlanningLLCsParams{ItemCodes: itemCodes, Llcs: levels}); err != nil {
		return fmt.Errorf("updating item LLCs: %w", err)
	}
	return nil
}

func (r *MRPCalculationRepositorySQLC) CreateExceptionMessage(
	ctx context.Context,
	msg *entity.MRPExceptionMessage,
) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	_, err = r.q.CreateMRPExceptionMessage(ctx, sqlc.CreateMRPExceptionMessageParams{
		PlanCode:      msg.PlanCode,
		ItemCode:      msg.ItemCode,
		MessageType:   string(msg.MessageType),
		SourceCode:    msg.SourceCode,
		SourceType:    pgutil.ToPgTextFromPtr(msg.SourceType),
		Description:   msg.Description,
		EnterpriseID:  enterpriseID,
		SuggestedDate: optionalPgDate(msg.SuggestedDate),
	})
	return err
}

func (r *MRPCalculationRepositorySQLC) ListExceptionsByPlan(
	ctx context.Context,
	planCode int64,
) ([]*entity.MRPExceptionMessage, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListMRPExceptionMessages(ctx, sqlc.ListMRPExceptionMessagesParams{PlanCode: planCode, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("listing exception messages: %w", err)
	}

	out := make([]*entity.MRPExceptionMessage, 0, len(rows))
	for _, row := range rows {
		m := exceptionFromRow(sqlc.MrpExceptionMessage{
			Code: row.Code, PlanCode: row.PlanCode, ItemCode: row.ItemCode, MessageType: row.MessageType,
			SourceCode: row.SourceCode, SourceType: row.SourceType, Description: row.Description, CreatedAt: row.CreatedAt,
			SuggestedDate: row.SuggestedDate, HandledAt: row.HandledAt, HandledBy: row.HandledBy,
			HandlingAction: row.HandlingAction, HandlingNote: row.HandlingNote,
		})
		m.HandledByName = row.HandledByName
		m.SourceOrderNumber = row.SourceOrderNumber
		if row.SourceNeedDate.Valid {
			d := row.SourceNeedDate.Time
			m.SourceNeedDate = &d
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *MRPCalculationRepositorySQLC) DeleteExceptionsByPlan(
	ctx context.Context,
	planCode int64,
) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteMRPExceptionMessages(ctx, sqlc.DeleteMRPExceptionMessagesParams{PlanCode: planCode, EnterpriseID: enterpriseID})
}

func (r *MRPCalculationRepositorySQLC) ListAllStockSnapshots(
	ctx context.Context,
) (map[int64]*entity.StockSnapshot, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// stock_snapshots is a manual freeze mechanism that may be empty;
	// fall back to stock_balances which always reflects the live position.
	// A reserva ativa de pedido de venda vem à parte: o MRP não a desconta,
	// porque o pedido já é demanda (DisponivelParaNecessidade). Só a do pedido
	// que o MRP lê (P/A, ativo, não bloqueado): a reserva de um pedido
	// bloqueado continua tirando o estoque, já que ele não entra como demanda.
	rows, err := r.db.Query(ctx, `
		WITH venda AS (
			SELECT r.item_code, SUM(r.quantity) AS qtd
			  FROM stock_reservations r
			  JOIN sales_orders so ON so.code = r.reference_code
			  JOIN enterprise e ON e.code = so.enterprise_code AND e.id = r.enterprise_id
			 WHERE r.enterprise_id = $1 AND r.status = 'ACTIVE' AND r.reference_type = 'SALES_ORDER'
			   AND so.is_active AND NOT so.is_blocked AND so.status IN ('P', 'A')
			   AND NOT EXISTS (SELECT 1 FROM warehouse w WHERE w.id = r.warehouse_id AND w.enterprise_id = r.enterprise_id
			                   AND (NOT w.disposition AND w.location::text <> 'INSPECAO' OR w.location::text = 'REJEICAO'))
			 GROUP BY r.item_code
		)
		SELECT
			b.item_code,
			COALESCE(SUM(b.quantity), 0)      AS quantity,
			COALESCE(SUM(b.reserved_qty), 0)  AS reserved_qty,
			COALESCE(MAX(v.qtd), 0)           AS sales_reserved_qty,
			COALESCE(SUM(b.safety_stock), 0)  AS safety_stock
		FROM stock_balances b
		LEFT JOIN venda v ON v.item_code = b.item_code
		WHERE b.enterprise_id = $1 AND `+saldoPlanejavel+`
		GROUP BY b.item_code
	`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("loading stock balances for MRP: %w", err)
	}
	defer rows.Close()

	m := make(map[int64]*entity.StockSnapshot)
	for rows.Next() {
		var itemCode int64
		var qty, reservedQty, salesReserved, safetyStock decimal.Decimal
		if err := rows.Scan(&itemCode, &qty, &reservedQty, &salesReserved, &safetyStock); err != nil {
			return nil, fmt.Errorf("scanning stock balance: %w", err)
		}
		m[itemCode] = &entity.StockSnapshot{
			ItemCode:         itemCode,
			Quantity:         qty,
			ReservedQty:      reservedQty,
			SalesReservedQty: salesReserved,
			SafetyStock:      safetyStock,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating stock balances: %w", err)
	}
	return m, nil
}

func (r *MRPCalculationRepositorySQLC) ListAllConfiguredRules(
	ctx context.Context,
) (map[int64][]*entity.ConfiguredItemRule, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListAllActiveConfiguredRules(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("loading all configured rules: %w", err)
	}

	m := make(map[int64][]*entity.ConfiguredItemRule, len(rows))
	for _, row := range rows {
		rule := ruleToEntity(row)
		m[row.ItemCode] = append(m[row.ItemCode], rule)
	}
	return m, nil
}

// =========================
// Helpers
// =========================

func profileToEntity(row sqlc.MrpItemProfile) *entity.MRPItemProfile {
	return &entity.MRPItemProfile{
		ItemCode:        row.ItemCode,
		PlanCode:        row.PlanCode,
		CalculationDate: pgutil.FromPgDate(row.CalculationDate),
		Demand:          pgutil.FromPgNumericToDecimal(row.Demand),
		OrdersPlanned:   pgutil.FromPgNumericToDecimal(row.OrdersPlanned),
		OrdersFirm:      pgutil.FromPgNumericToDecimal(row.OrdersFirm),
		StockProjected:  pgutil.FromPgNumericToDecimal(row.StockProjected),
		LLC:             int(row.Llc),
		NeedDate:        pgutil.FromPgDate(row.NeedDate),
		CreatedAt:       pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func profilesToEntities(
	rows []sqlc.MrpItemProfile,
) []*entity.MRPItemProfile {

	out := make([]*entity.MRPItemProfile, 0, len(rows))

	for _, row := range rows {
		out = append(out, profileToEntity(row))
	}

	return out
}

func logToEntity(
	row sqlc.MrpCalculationLog,
) *entity.MRPCalculationLog {

	e := &entity.MRPCalculationLog{
		Code:        row.Code.Int64,
		PlanCode:    row.PlanCode,
		StartedAt:   pgutil.FromPgTimestamptz(row.StartedAt),
		Status:      row.Status,
		TotalItems:  int(row.TotalItems),
		TotalOrders: int(row.TotalOrders),
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}

	if row.FinishedAt.Valid {
		v := row.FinishedAt.Time
		e.FinishedAt = &v
	}

	if row.Errors != nil {
		var errs map[string]interface{}
		_ = json.Unmarshal(row.Errors, &errs)
		e.Errors = errs
	}

	return e
}

func logsToEntities(
	rows []sqlc.MrpCalculationLog,
) []*entity.MRPCalculationLog {

	out := make([]*entity.MRPCalculationLog, 0, len(rows))

	for _, row := range rows {
		out = append(out, logToEntity(row))
	}

	return out
}

func snapshotToEntity(
	row sqlc.StockSnapshot,
) *entity.StockSnapshot {

	return &entity.StockSnapshot{
		ItemCode:      row.ItemCode,
		WarehouseCode: row.WarehouseCode,
		Quantity:      pgutil.FromPgNumericToDecimal(row.Quantity),
		ReservedQty:   pgutil.FromPgNumericToDecimal(row.ReservedQty),
		SafetyStock:   pgutil.FromPgNumericToDecimal(row.SafetyStock),
		SnapshotDate:  pgutil.FromPgTimestamptz(row.SnapshotDate),
		CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func salesDemandToEntity(
	row sqlc.SalesOrderDemand,
) *entity.SalesOrderDemand {

	var mask *string

	if row.Mask != "" {
		m := row.Mask
		mask = &m
	}

	return &entity.SalesOrderDemand{
		Code:           row.Code.Int64,
		SalesOrderCode: row.SalesOrderCode,
		ItemCode:       row.ItemCode,
		Mask:           mask,
		Quantity:       pgutil.FromPgNumericToDecimal(row.Quantity),
		DeliveredQty:   pgutil.FromPgNumericToDecimal(row.DeliveredQty),
		DeliveryDate:   pgutil.FromPgDate(row.DeliveryDate),
		DivisionCode:   row.DivisionCode,
		Status:         row.Status,
		IsActive:       row.IsActive,
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:      pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func salesDemandsToEntities(
	rows []sqlc.SalesOrderDemand,
) []*entity.SalesOrderDemand {

	out := make([]*entity.SalesOrderDemand, 0, len(rows))

	for _, row := range rows {
		out = append(out, salesDemandToEntity(row))
	}

	return out
}

func ruleToEntity(
	row sqlc.ConfiguredItemRule,
) *entity.ConfiguredItemRule {

	return &entity.ConfiguredItemRule{
		Code:            row.Code.Int64,
		ItemCode:        row.ItemCode,
		TableType:       row.TableType,
		FieldName:       row.FieldName,
		RuleType:        row.RuleType,
		RuleValue:       row.RuleValue,
		Sequence:        int(row.Sequence),
		IsActive:        row.IsActive,
		CreatedAt:       pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:       pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:       pgutil.FromPgUUID(row.CreatedBy),
		CondicaoMascara: row.CondicaoMascara.String,
	}
}

func suggestionToEntity(row sqlc.MrpPlannedSuggestion) *entity.PlannedOrderSuggestion {
	s := &entity.PlannedOrderSuggestion{
		Code:                 row.Code,
		OrderNumber:          row.OrderNumber,
		WarehouseCode:        row.WarehouseCode,
		InterFactory:         row.InterFactory,
		SourceEnterpriseCode: row.SourceEnterpriseCode,
		AutoRelease:          row.AutoRelease,
		PlanCode:             row.PlanCode,
		ItemCode:             row.ItemCode,
		Mask:                 row.Mask,
		Quantity:             pgutil.FromPgNumericToDecimal(row.Quantity),
		NeedDate:             pgutil.FromPgDate(row.NeedDate),
		OrderType:            row.OrderType,
		DemandType:           row.DemandType,
		LLC:                  int(row.Llc),
		Notes:                pgutil.FromPgTextPtr(row.Notes),
		RouteOperationID:     row.RouteOperationID,
		OperationID:          row.OperationID,
		SupplierCode:         row.SupplierCode,
		ServiceItemCode:      row.ServiceItemCode,
		RemittanceType:       pgutil.FromPgTextPtr(row.RemittanceType),
	}

	if sd := pgutil.FromPgDate(row.StartDate); !sd.IsZero() {
		s.StartDate = &sd
	}

	if row.ParentItemCode != nil {
		v := *row.ParentItemCode
		s.ParentItemCode = &v
	}

	return s
}

func suggestionsToEntities(rows []sqlc.MrpPlannedSuggestion) []*entity.PlannedOrderSuggestion {
	out := make([]*entity.PlannedOrderSuggestion, 0, len(rows))
	for _, row := range rows {
		out = append(out, suggestionToEntity(row))
	}
	return out
}

func rulesToEntities(
	rows []sqlc.ConfiguredItemRule,
) []*entity.ConfiguredItemRule {

	out := make([]*entity.ConfiguredItemRule, 0, len(rows))

	for _, row := range rows {
		out = append(out, ruleToEntity(row))
	}

	return out
}

// PrazoDeCompra é o prazo de entrega (dias) do fornecedor do item comprado: o
// marcado como preferencial, depois o de melhor ranking, entre os vínculos
// ativos e vigentes hoje. Sem fornecedor, zero. A data vai como parâmetro (o
// processo e o banco podem estar em fusos diferentes).
func (r *MRPCalculationRepositorySQLC) PrazoDeCompra(ctx context.Context, itemCode int64) (int, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	hoje := time.Now()
	var dias int
	err = r.db.QueryRow(ctx, `
		SELECT lead_time_days FROM item_preferred_suppliers
		 WHERE enterprise_id = $1 AND item_code = $2 AND is_active
		   AND (valid_from IS NULL OR valid_from <= $3::date) AND (valid_until IS NULL OR valid_until >= $3::date)
		 ORDER BY is_preferred DESC, ranking, id
		 LIMIT 1`, enterpriseID, itemCode, time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC)).Scan(&dias)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lendo o prazo do fornecedor do item %d: %w", itemCode, err)
	}
	if dias < 0 {
		dias = 0
	}
	return dias, nil
}
