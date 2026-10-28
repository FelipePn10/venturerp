package quality

import (
	"context"
	"errors"
	"fmt"

	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/quality/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/domain/quality/entity"
)

type QualityRepositorySQLC struct {
	q *sqlc.Queries
}

func New(q *sqlc.Queries) domainrepo.QualityRepository {
	return &QualityRepositorySQLC{q: q}
}

// ─── inspection plans ─────────────────────────────────────────────────────────

func (r *QualityRepositorySQLC) CreatePlan(ctx context.Context, plan *entity.InspectionPlan) (*entity.InspectionPlan, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateInspectionPlan(ctx, sqlc.CreateInspectionPlanParams{
		EnterpriseID:     e,
		ItemCode:         plan.ItemCode,
		RouteOperationID: pgutil.ToPgInt8Ptr(plan.RouteOperationID),
		PointType:        sqlc.InspectionPointType(plan.PointType),
		Description:      plan.Description,
		SampleSize:       pgutil.ToPgNumericFromDecimal(plan.SampleSize),
		AcceptanceLevel:  pgutil.ToPgNumericFromDecimal(plan.AcceptanceLevel),
		Instructions:     pgutil.ToPgTextFromPtr(plan.Instructions),
		CreatedBy:        pgutil.ToPgUUID(plan.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating inspection plan: %w", err)
	}
	return planRowToEntity(row), nil
}

func (r *QualityRepositorySQLC) GetPlanByID(ctx context.Context, id int64) (*entity.InspectionPlan, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetInspectionPlanByID(ctx, id, e)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("plano de inspeção não encontrado")
		}
		return nil, fmt.Errorf("lendo o plano de inspeção %d: %w", id, err)
	}
	return planRowToEntity(row), nil
}

func (r *QualityRepositorySQLC) ListPlansByItem(ctx context.Context, itemCode int64) ([]*entity.InspectionPlan, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListPlansByItem(ctx, itemCode, e)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os planos do item %d: %w", itemCode, err)
	}
	out := make([]*entity.InspectionPlan, 0, len(rows))
	for _, row := range rows {
		out = append(out, planRowToEntity(row))
	}
	return out, nil
}

func (r *QualityRepositorySQLC) DeactivatePlan(ctx context.Context, id int64) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.DeactivateInspectionPlan(ctx, id, e)
}

// ─── characteristics ──────────────────────────────────────────────────────────

func (r *QualityRepositorySQLC) AddCharacteristic(ctx context.Context, c *entity.InspectionCharacteristic) (*entity.InspectionCharacteristic, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.AddCharacteristic(ctx, sqlc.AddCharacteristicParams{
		EnterpriseID:   e,
		PlanID:         c.PlanID,
		Name:           c.Name,
		Nominal:        pgutil.ToPgNumericFromDecimalPtr(c.Nominal),
		ToleranceUpper: pgutil.ToPgNumericFromDecimalPtr(c.ToleranceUpper),
		ToleranceLower: pgutil.ToPgNumericFromDecimalPtr(c.ToleranceLower),
		Unit:           pgutil.ToPgTextFromPtr(c.Unit),
		IsCritical:     c.IsCritical,
	})
	if err != nil {
		return nil, fmt.Errorf("adding characteristic: %w", err)
	}
	return charRowToEntity(row), nil
}

func (r *QualityRepositorySQLC) ListCharacteristics(ctx context.Context, planID int64) ([]*entity.InspectionCharacteristic, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListCharacteristics(ctx, planID, e)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar as características do plano %d: %w", planID, err)
	}
	out := make([]*entity.InspectionCharacteristic, 0, len(rows))
	for _, row := range rows {
		out = append(out, charRowToEntity(row))
	}
	return out, nil
}

// ─── quality records ──────────────────────────────────────────────────────────

func (r *QualityRepositorySQLC) CreateRecord(ctx context.Context, rec *entity.QualityRecord) (*entity.QualityRecord, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateQualityRecord(ctx, sqlc.CreateQualityRecordParams{
		EnterpriseID:      e,
		PlanID:            rec.PlanID,
		ProductionOrderID: pgutil.ToPgInt8Ptr(rec.ProductionOrderID),
		Lot:               pgutil.ToPgTextFromPtr(rec.Lot),
		ItemCode:          rec.ItemCode,
		InspectedQty:      pgutil.ToPgNumericFromDecimal(rec.InspectedQty),
		ApprovedQty:       pgutil.ToPgNumericFromDecimal(rec.ApprovedQty),
		RejectedQty:       pgutil.ToPgNumericFromDecimal(rec.RejectedQty),
		Result:            sqlc.InspectionResultEnum(rec.Result),
		InspectorID:       pgutil.ToPgInt8Ptr(rec.InspectorID),
		Notes:             pgutil.ToPgTextFromPtr(rec.Notes),
		CreatedBy:         pgutil.ToPgUUID(rec.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating quality record: %w", err)
	}
	return recordRowToEntity(row), nil
}

func (r *QualityRepositorySQLC) AddMeasurement(ctx context.Context, m *entity.QualityMeasurement) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.AddMeasurement(ctx, sqlc.AddMeasurementParams{
		EnterpriseID:     e,
		RecordID:         m.RecordID,
		CharacteristicID: m.CharacteristicID,
		MeasuredValue:    pgutil.ToPgNumericFromDecimal(m.MeasuredValue),
		IsConformant:     m.IsConformant,
	})
}

func (r *QualityRepositorySQLC) GetRecordByID(ctx context.Context, id int64) (*entity.QualityRecord, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetQualityRecordByID(ctx, id, e)
	if err != nil {
		return nil, fmt.Errorf("fetching quality record %d: %w", id, err)
	}
	return recordRowToEntity(row), nil
}

func (r *QualityRepositorySQLC) ListRecordsByOrder(ctx context.Context, orderID int64) ([]*entity.QualityRecord, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRecordsByOrder(ctx, orderID, e)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os registros da ordem %d: %w", orderID, err)
	}
	out := make([]*entity.QualityRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, recordRowToEntity(row))
	}
	return out, nil
}

func (r *QualityRepositorySQLC) ListRecordsByItem(ctx context.Context, itemCode int64) ([]*entity.QualityRecord, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRecordsByItem(ctx, itemCode, e)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar os registros do item %d: %w", itemCode, err)
	}
	out := make([]*entity.QualityRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, recordRowToEntity(row))
	}
	return out, nil
}

// ─── non-conformances ─────────────────────────────────────────────────────────

func (r *QualityRepositorySQLC) CreateNC(ctx context.Context, nc *entity.NonConformance) (*entity.NonConformance, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateNC(ctx, sqlc.CreateNCParams{
		EnterpriseID:      e,
		Code:              nc.Code,
		QualityRecordID:   pgutil.ToPgInt8Ptr(nc.QualityRecordID),
		ProductionOrderID: pgutil.ToPgInt8Ptr(nc.ProductionOrderID),
		ItemCode:          nc.ItemCode,
		Lot:               pgutil.ToPgTextFromPtr(nc.Lot),
		NonConformQty:     pgutil.ToPgNumericFromDecimal(nc.NonConformQty),
		Description:       nc.Description,
		Severity:          sqlc.NcSeverityEnum(nc.Severity),
		CreatedBy:         pgutil.ToPgUUID(nc.CreatedBy),
		LotID:             pgutil.ToPgInt8Ptr(nc.LotID),
	})
	if err != nil {
		return nil, fmt.Errorf("creating non-conformance: %w", err)
	}
	return ncRowToEntity(row), nil
}

func (r *QualityRepositorySQLC) GetNCByID(ctx context.Context, id int64) (*entity.NonConformance, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetNCByID(ctx, id, e)
	if err != nil {
		return nil, fmt.Errorf("fetching NC %d: %w", id, err)
	}
	return ncRowToEntity(row), nil
}

func (r *QualityRepositorySQLC) ListOpenNCs(ctx context.Context) ([]*entity.NonConformance, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListOpenNCs(ctx, e)
	if err != nil {
		return nil, fmt.Errorf("listing open NCs: %w", err)
	}
	out := make([]*entity.NonConformance, 0, len(rows))
	for _, row := range rows {
		out = append(out, ncRowToEntity(row))
	}
	return out, nil
}

func (r *QualityRepositorySQLC) ListNCsByItem(ctx context.Context, itemCode int64) ([]*entity.NonConformance, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListNCsByItem(ctx, itemCode, e)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar as não conformidades do item %d: %w", itemCode, err)
	}
	out := make([]*entity.NonConformance, 0, len(rows))
	for _, row := range rows {
		out = append(out, ncRowToEntity(row))
	}
	return out, nil
}

func (r *QualityRepositorySQLC) DispositionNC(ctx context.Context, id int64, disposition entity.NCDisposition, disposedBy string) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(disposedBy)
	if err != nil {
		return fmt.Errorf("usuário responsável pelo descarte inválido: %w", err)
	}
	return r.q.DispositionNC(ctx, sqlc.DispositionNCParams{
		EnterpriseID: e,
		ID:           id,
		Disposition:  sqlc.NcDispositionEnum(disposition),
		DisposedBy:   pgutil.ToPgUUID(uid),
	})
}

func (r *QualityRepositorySQLC) NextNCCode(ctx context.Context) (int64, error) {
	return r.q.NextNCCode(ctx)
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func planRowToEntity(row sqlc.InspectionPlan) *entity.InspectionPlan {
	e := &entity.InspectionPlan{
		ID:               row.ID,
		ItemCode:         row.ItemCode,
		RouteOperationID: row.RouteOperationID,
		PointType:        entity.InspectionPointType(row.PointType),
		Description:      row.Description,
		SampleSize:       pgutil.FromPgNumericToDecimal(row.SampleSize),
		AcceptanceLevel:  pgutil.FromPgNumericToDecimal(row.AcceptanceLevel),
		IsActive:         row.IsActive,
		CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:        pgutil.FromPgUUID(row.CreatedBy),
	}
	if row.Instructions.Valid {
		v := row.Instructions.String
		e.Instructions = &v
	}
	return e
}

func charRowToEntity(row sqlc.InspectionPlanCharacteristic) *entity.InspectionCharacteristic {
	e := &entity.InspectionCharacteristic{
		ID:         row.ID,
		PlanID:     row.PlanID,
		Name:       row.Name,
		IsCritical: row.IsCritical,
	}
	e.Nominal = pgutil.FromPgNumericToDecimalPtr(row.Nominal)
	e.ToleranceUpper = pgutil.FromPgNumericToDecimalPtr(row.ToleranceUpper)
	e.ToleranceLower = pgutil.FromPgNumericToDecimalPtr(row.ToleranceLower)
	if row.Unit.Valid {
		v := row.Unit.String
		e.Unit = &v
	}
	return e
}

func recordRowToEntity(row sqlc.QualityRecord) *entity.QualityRecord {
	e := &entity.QualityRecord{
		ID:                row.ID,
		PlanID:            row.PlanID,
		ProductionOrderID: row.ProductionOrderID,
		ItemCode:          row.ItemCode,
		InspectedQty:      pgutil.FromPgNumericToDecimal(row.InspectedQty),
		ApprovedQty:       pgutil.FromPgNumericToDecimal(row.ApprovedQty),
		RejectedQty:       pgutil.FromPgNumericToDecimal(row.RejectedQty),
		Result:            entity.InspectionResult(row.Result),
		InspectorID:       row.InspectorID,
		InspectedAt:       pgutil.FromPgTimestamptz(row.InspectedAt),
		CreatedAt:         pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:         pgutil.FromPgUUID(row.CreatedBy),
	}
	if row.Lot.Valid {
		v := row.Lot.String
		e.Lot = &v
	}
	if row.Notes.Valid {
		v := row.Notes.String
		e.Notes = &v
	}
	return e
}

func ncRowToEntity(row sqlc.NonConformance) *entity.NonConformance {
	e := &entity.NonConformance{
		ID:                row.ID,
		Code:              row.Code,
		QualityRecordID:   row.QualityRecordID,
		ProductionOrderID: row.ProductionOrderID,
		ItemCode:          row.ItemCode,
		NonConformQty:     pgutil.FromPgNumericToDecimal(row.NonconformQty),
		Description:       row.Description,
		Severity:          entity.NCSeverity(row.Severity),
		IsOpen:            row.IsOpen,
		CreatedAt:         pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:         pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:         pgutil.FromPgUUID(row.CreatedBy),
	}
	if row.Lot.Valid {
		v := row.Lot.String
		e.Lot = &v
	}
	if row.RootCause.Valid {
		v := row.RootCause.String
		e.RootCause = &v
	}
	if row.CorrectiveAction.Valid {
		v := row.CorrectiveAction.String
		e.CorrectiveAction = &v
	}
	if row.Disposition.Valid {
		d := entity.NCDisposition(string(row.Disposition.NcDispositionEnum))
		e.Disposition = &d
	}
	if row.DisposedAt.Valid {
		t := pgutil.FromPgTimestamptz(row.DisposedAt)
		e.DisposedAt = &t
	}
	if row.DisposedBy.Valid {
		uid := pgutil.FromPgUUID(row.DisposedBy)
		e.DisposedBy = &uid
	}
	return e
}
