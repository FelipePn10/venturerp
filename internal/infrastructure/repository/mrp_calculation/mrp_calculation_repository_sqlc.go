package mrp_calculation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
)

func (r *MRPCalculationRepositorySQLC) CreateProfile(
	ctx context.Context,
	p *entity.MRPItemProfile,
) (*entity.MRPItemProfile, error) {

	row, err := r.q.CreateMRPItemProfile(ctx, sqlc.CreateMRPItemProfileParams{
		ItemCode:        p.ItemCode,
		PlanCode:        p.PlanCode,
		CalculationDate: pgutil.ToPgDate(p.CalculationDate),
		Demand:          pgutil.ToPgNumericFromFloat64(p.Demand),
		OrdersPlanned:   pgutil.ToPgNumericFromFloat64(p.OrdersPlanned),
		OrdersFirm:      pgutil.ToPgNumericFromFloat64(p.OrdersFirm),
		StockProjected:  pgutil.ToPgNumericFromFloat64(p.StockProjected),
		Llc:             int32(p.LLC),
		NeedDate:        pgutil.ToPgDate(p.NeedDate),
	})
	if err != nil {
		return nil, fmt.Errorf("creating MRP item profile: %w", err)
	}

	return profileToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) GetProfiles(
	ctx context.Context,
	itemCode,
	planCode int64,
) ([]*entity.MRPItemProfile, error) {

	rows, err := r.q.GetMRPItemProfiles(ctx, sqlc.GetMRPItemProfilesParams{
		ItemCode: itemCode,
		PlanCode: planCode,
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

	return r.q.DeleteProfilesByPlan(ctx, planID)
}

func (r *MRPCalculationRepositorySQLC) StartCalculation(
	ctx context.Context,
	planID int64,
) (*entity.MRPCalculationLog, error) {

	row, err := r.q.StartMRPCalculation(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("starting MRP calculation: %w", err)
	}

	return logToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) FinishCalculation(
	ctx context.Context,
	logCode int64,
	status string,
	errorsMap map[string]interface{},
	totalItems,
	totalOrders int,
) (*entity.MRPCalculationLog, error) {

	errorsJSON, _ := json.Marshal(errorsMap)

	row, err := r.q.FinishMRPCalculation(ctx, sqlc.FinishMRPCalculationParams{
		Code:        pgutil.ToPgInt8Ptr(&logCode),
		Status:      status,
		Errors:      errorsJSON,
		TotalItems:  int32(totalItems),
		TotalOrders: int32(totalOrders),
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

	row, err := r.q.GetMRPCalculationLog(
		ctx,
		pgutil.ToPgInt8Ptr(&logCode),
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("calculation log %d not found", logCode)
		}

		return nil, fmt.Errorf("fetching calculation log: %w", err)
	}

	return logToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) ListCalculationLogs(
	ctx context.Context,
	planID int64,
) ([]*entity.MRPCalculationLog, error) {

	rows, err := r.q.ListMRPCalculationLogsByPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("listing calculation logs: %w", err)
	}

	return logsToEntities(rows), nil
}

func (r *MRPCalculationRepositorySQLC) CreateStockSnapshot(
	ctx context.Context,
	s *entity.StockSnapshot,
) (*entity.StockSnapshot, error) {

	row, err := r.q.CreateStockSnapshot(ctx, sqlc.CreateStockSnapshotParams{
		ItemCode:      s.ItemCode,
		WarehouseCode: s.WarehouseCode,
		Quantity:      pgutil.ToPgNumericFromFloat64(s.Quantity),
		ReservedQty:   pgutil.ToPgNumericFromFloat64(s.ReservedQty),
		SafetyStock:   pgutil.ToPgNumericFromFloat64(s.SafetyStock),
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

	row, err := r.q.GetStockSnapshot(ctx, itemCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("fetching stock snapshot: %w", err)
	}

	return snapshotToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) CreateSalesOrderDemand(
	ctx context.Context,
	d *entity.SalesOrderDemand,
) (*entity.SalesOrderDemand, error) {

	mask := ""
	if d.Mask != nil {
		mask = *d.Mask
	}

	row, err := r.q.CreateSalesOrderDemand(ctx, sqlc.CreateSalesOrderDemandParams{
		SalesOrderCode: d.SalesOrderCode,
		ItemCode:       d.ItemCode,
		Mask:           mask,
		Quantity:       pgutil.ToPgNumericFromFloat64(d.Quantity),
		DeliveredQty:   pgutil.ToPgNumericFromFloat64(d.DeliveredQty),
		DeliveryDate:   pgutil.ToPgDate(d.DeliveryDate),
		DivisionCode:   d.DivisionCode,
		Status:         d.Status,
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

	row, err := r.q.GetSalesOrderDemandByCode(
		ctx,
		pgutil.ToPgInt8Ptr(&code),
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("sales order demand %d not found", code)
		}

		return nil, fmt.Errorf("fetching sales order demand: %w", err)
	}

	return salesDemandToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) ListSalesOrderDemandsByItem(
	ctx context.Context,
	itemCode int64,
) ([]*entity.SalesOrderDemand, error) {

	rows, err := r.q.ListSalesOrderDemandsByItem(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing sales order demands: %w", err)
	}

	return salesDemandsToEntities(rows), nil
}

func (r *MRPCalculationRepositorySQLC) UpdateSalesOrderDemandStatus(
	ctx context.Context,
	code int64,
	status string,
	deliveredQty float64,
) (*entity.SalesOrderDemand, error) {

	row, err := r.q.UpdateSalesOrderDemandStatus(
		ctx,
		sqlc.UpdateSalesOrderDemandStatusParams{
			Status:       status,
			DeliveredQty: pgutil.ToPgNumericFromFloat64(deliveredQty),
			Code:         pgutil.ToPgInt8Ptr(&code),
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

	row, err := r.q.CreateConfiguredItemRule(
		ctx,
		sqlc.CreateConfiguredItemRuleParams{
			ItemCode:  rule.ItemCode,
			TableType: rule.TableType,
			FieldName: rule.FieldName,
			RuleType:  rule.RuleType,
			RuleValue: rule.RuleValue,
			Sequence:  int32(rule.Sequence),
			CreatedBy: pgutil.ToPgUUID(rule.CreatedBy),
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

	rows, err := r.q.GetConfiguredItemRules(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("fetching configured item rules: %w", err)
	}

	return rulesToEntities(rows), nil
}

func (r *MRPCalculationRepositorySQLC) DeleteConfiguredItemRule(
	ctx context.Context,
	id int64,
) error {

	return r.q.DeleteConfiguredItemRule(
		ctx,
		pgutil.ToPgInt8Ptr(&id),
	)
}

func (r *MRPCalculationRepositorySQLC) CreatePlannedOrderSuggestion(
	ctx context.Context,
	s *entity.PlannedOrderSuggestion,
) (*entity.PlannedOrderSuggestion, error) {

	startDate := pgutil.ToPgDate(s.NeedDate)
	if s.StartDate != nil {
		startDate = pgutil.ToPgDate(*s.StartDate)
	}

	row, err := r.q.CreateMRPPlannedSuggestion(ctx, sqlc.CreateMRPPlannedSuggestionParams{
		PlanCode:       s.PlanCode,
		ItemCode:       s.ItemCode,
		Quantity:       pgutil.ToPgNumericFromFloat64(s.Quantity),
		NeedDate:       pgutil.ToPgDate(s.NeedDate),
		StartDate:      startDate,
		OrderType:      s.OrderType,
		DemandType:     s.DemandType,
		ParentItemCode: s.ParentItemCode,
		Llc:            int32(s.LLC),
	})
	if err != nil {
		return nil, fmt.Errorf("creating planned suggestion: %w", err)
	}

	return suggestionToEntity(row), nil
}

func (r *MRPCalculationRepositorySQLC) ListSuggestionsByPlan(
	ctx context.Context,
	planCode int64,
) ([]*entity.PlannedOrderSuggestion, error) {

	rows, err := r.q.ListMRPPlannedSuggestions(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("listing planned suggestions: %w", err)
	}

	return suggestionsToEntities(rows), nil
}

func (r *MRPCalculationRepositorySQLC) DeleteSuggestionsByPlan(
	ctx context.Context,
	planCode int64,
) error {

	return r.q.DeleteMRPPlannedSuggestions(ctx, planCode)
}

func (r *MRPCalculationRepositorySQLC) CreateExceptionMessage(
	ctx context.Context,
	msg *entity.MRPExceptionMessage,
) error {

	_, err := r.q.CreateMRPExceptionMessage(ctx, sqlc.CreateMRPExceptionMessageParams{
		PlanCode:    msg.PlanCode,
		ItemCode:    msg.ItemCode,
		MessageType: string(msg.MessageType),
		SourceCode:  msg.SourceCode,
		SourceType:  pgutil.ToPgTextFromPtr(msg.SourceType),
		Description: msg.Description,
	})
	return err
}

func (r *MRPCalculationRepositorySQLC) ListExceptionsByPlan(
	ctx context.Context,
	planCode int64,
) ([]*entity.MRPExceptionMessage, error) {

	rows, err := r.q.ListMRPExceptionMessages(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("listing exception messages: %w", err)
	}

	out := make([]*entity.MRPExceptionMessage, 0, len(rows))
	for _, row := range rows {
		m := &entity.MRPExceptionMessage{
			Code:        row.Code,
			PlanCode:    row.PlanCode,
			ItemCode:    row.ItemCode,
			MessageType: entity.ExceptionMessageType(row.MessageType),
			Description: row.Description,
			CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		}
		if row.SourceCode != nil {
			v := *row.SourceCode
			m.SourceCode = &v
		}
		if row.SourceType.Valid {
			v := row.SourceType.String
			m.SourceType = &v
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *MRPCalculationRepositorySQLC) DeleteExceptionsByPlan(
	ctx context.Context,
	planCode int64,
) error {
	return r.q.DeleteMRPExceptionMessages(ctx, planCode)
}

func (r *MRPCalculationRepositorySQLC) ListAllStockSnapshots(
	ctx context.Context,
) (map[int64]*entity.StockSnapshot, error) {

	rows, err := r.q.ListLatestStockSnapshots(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading all stock snapshots: %w", err)
	}

	m := make(map[int64]*entity.StockSnapshot, len(rows))
	for _, row := range rows {
		m[row.ItemCode] = snapshotToEntity(row)
	}
	return m, nil
}

func (r *MRPCalculationRepositorySQLC) ListAllConfiguredRules(
	ctx context.Context,
) (map[int64][]*entity.ConfiguredItemRule, error) {

	rows, err := r.q.ListAllActiveConfiguredRules(ctx)
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
		Demand:          pgutil.FromPgNumericToFloat64(row.Demand),
		OrdersPlanned:   pgutil.FromPgNumericToFloat64(row.OrdersPlanned),
		OrdersFirm:      pgutil.FromPgNumericToFloat64(row.OrdersFirm),
		StockProjected:  pgutil.FromPgNumericToFloat64(row.StockProjected),
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
		Quantity:      pgutil.FromPgNumericToFloat64(row.Quantity),
		ReservedQty:   pgutil.FromPgNumericToFloat64(row.ReservedQty),
		SafetyStock:   pgutil.FromPgNumericToFloat64(row.SafetyStock),
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
		Quantity:       pgutil.FromPgNumericToFloat64(row.Quantity),
		DeliveredQty:   pgutil.FromPgNumericToFloat64(row.DeliveredQty),
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
		Code:      row.Code.Int64,
		ItemCode:  row.ItemCode,
		TableType: row.TableType,
		FieldName: row.FieldName,
		RuleType:  row.RuleType,
		RuleValue: row.RuleValue,
		Sequence:  int(row.Sequence),
		IsActive:  row.IsActive,
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt: pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
	}
}

func suggestionToEntity(row sqlc.MrpPlannedSuggestion) *entity.PlannedOrderSuggestion {
	s := &entity.PlannedOrderSuggestion{
		Code:       row.Code,
		PlanCode:   row.PlanCode,
		ItemCode:   row.ItemCode,
		Quantity:   pgutil.FromPgNumericToFloat64(row.Quantity),
		NeedDate:   pgutil.FromPgDate(row.NeedDate),
		OrderType:  row.OrderType,
		DemandType: row.DemandType,
		LLC:        int(row.Llc),
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
