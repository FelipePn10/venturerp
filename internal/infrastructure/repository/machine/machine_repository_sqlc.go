package machine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *MachineRepositorySQLC) CreateType(ctx context.Context, mt *entity.MachineType) (*entity.MachineType, error) {
	row, err := r.q.CreateMachineType(ctx, sqlc.CreateMachineTypeParams{
		Code:             mt.Code,
		Name:             mt.Name,
		Description:      pgutil.ToPgTextFromPtr(mt.Description),
		Type:             sqlc.MachineTypeEnum(mt.Type),
		RequiresOperator: mt.RequiresOperator,
		IsActive:         mt.IsActive,
		CreatedBy:        pgutil.ToPgUUID(mt.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("create machine type: %w", err)
	}
	return machineTypeToEntity(row), nil
}

func (r *MachineRepositorySQLC) UpdateType(ctx context.Context, mt *entity.MachineType) (*entity.MachineType, error) {
	row, err := r.q.UpdateMachineType(ctx, sqlc.UpdateMachineTypeParams{
		Code:             mt.Code,
		Name:             mt.Name,
		Description:      pgutil.ToPgTextFromPtr(mt.Description),
		Type:             sqlc.MachineTypeEnum(mt.Type),
		RequiresOperator: mt.RequiresOperator,
		IsActive:         mt.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("update machine type: %w", err)
	}
	return machineTypeToEntity(row), nil
}

func (r *MachineRepositorySQLC) GetTypeByCode(ctx context.Context, code int64) (*entity.MachineType, error) {
	row, err := r.q.GetMachineTypeByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("machine type %d not found", code)
		}
		return nil, err
	}
	return machineTypeToEntity(row), nil
}

func (r *MachineRepositorySQLC) ListTypes(ctx context.Context) ([]*entity.MachineType, error) {
	rows, err := r.q.ListMachineTypes(ctx)
	if err != nil {
		return nil, err
	}
	return machineTypesToEntities(rows), nil
}

func (r *MachineRepositorySQLC) DeleteType(ctx context.Context, code int64) error {
	return r.q.DeleteMachineType(ctx, code)
}

func (r *MachineRepositorySQLC) Create(ctx context.Context, m *entity.Machine) (*entity.Machine, error) {
	row, err := r.q.CreateMachine(ctx, sqlc.CreateMachineParams{
		Code:            m.Code,
		Name:            m.Name,
		MachineTypeCode: m.MachineTypeCode,
		CostCenterCode:  m.CostCenterCode,
		Capacity:        pgutil.ToPgNumericFromFloat64(m.Capacity),
		CapacityPeriod:  sqlc.CapacityPeriodEnum(m.CapacityPeriod),
		CapacityUnit:    sqlc.MachineCapacityUnitEnum(m.CapacityUnit),
		EfficiencyRate:  pgutil.ToPgNumericFromFloat64(m.EfficiencyRate),
		CreatedBy:       pgutil.ToPgUUID(m.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("create machine: %w", err)
	}
	return machineToEntity(row), nil
}

func (r *MachineRepositorySQLC) Update(ctx context.Context, m *entity.Machine) (*entity.Machine, error) {
	row, err := r.q.UpdateMachine(ctx, sqlc.UpdateMachineParams{
		Name:            m.Name,
		MachineTypeCode: m.MachineTypeCode,
		CostCenterCode:  m.CostCenterCode,
		Capacity:        pgutil.ToPgNumericFromFloat64(m.Capacity),
		CapacityPeriod:  sqlc.CapacityPeriodEnum(m.CapacityPeriod),
		CapacityUnit:    sqlc.MachineCapacityUnitEnum(m.CapacityUnit),
		EfficiencyRate:  pgutil.ToPgNumericFromFloat64(m.EfficiencyRate),
	})
	if err != nil {
		return nil, fmt.Errorf("update machine: %w", err)
	}
	return machineToEntity(row), nil
}

func (r *MachineRepositorySQLC) GetByCode(ctx context.Context, code int64) (*entity.Machine, error) {
	row, err := r.q.GetMachineByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("machine %d not found", code)
		}
		return nil, err
	}
	return machineToEntity(row), nil
}

func (r *MachineRepositorySQLC) List(ctx context.Context) ([]*entity.Machine, error) {
	rows, err := r.q.ListMachines(ctx)
	if err != nil {
		return nil, err
	}
	return machinesToEntities(rows), nil
}

func (r *MachineRepositorySQLC) ListByType(ctx context.Context, typeCode int64) ([]*entity.Machine, error) {
	rows, err := r.q.ListMachinesByType(ctx, typeCode)
	if err != nil {
		return nil, err
	}
	return machinesToEntities(rows), nil
}

func (r *MachineRepositorySQLC) Delete(ctx context.Context, code int64) error {
	return r.q.DeleteMachine(ctx, code)
}

func (r *MachineRepositorySQLC) CreateItemMachineTime(ctx context.Context, imt *entity.ItemMachineTime) (*entity.ItemMachineTime, error) {
	row, err := r.q.CreateItemMachineTime(ctx, sqlc.CreateItemMachineTimeParams{
		ItemCode:           imt.ItemCode,
		Mask:               ptrToString(imt.Mask),
		MachineCode:        imt.MachineCode,
		ProductionTime:     pgutil.ToPgNumericFromFloat64(imt.ProductionTime),
		ProductionTimeUnit: sqlc.CapacityPeriodEnum(imt.ProductionTimeUnit),
		ProductionBaseQty:  int32(imt.ProductionBaseQty),
		SetupTime:          pgutil.ToPgNumericFromFloat64(imt.SetupTime),
		Priority:           int32(imt.Priority),
	})
	if err != nil {
		return nil, fmt.Errorf("create item machine time: %w", err)
	}
	return itemMachineTimeToEntity(row), nil
}

//func (r *MachineRepositorySQLC) GetItemMachineTime(
//	ctx context.Context,
//	code int64,
//) (*entity.ItemMachineTime, error) {
//
//	row, err := r.q.GetItemMachineTime(ctx, code)
//	if err != nil {
//		if errors.Is(err, pgx.ErrNoRows) {
//			return nil, fmt.Errorf("item machine time %d not found", code)
//		}
//
//		return nil, err
//	}
//
//	return itemMachineTimeToEntity(row), nil
//}

func (r *MachineRepositorySQLC) ListItemMachineTimes(
	ctx context.Context,
	itemCode int64,
) ([]*entity.ItemMachineTime, error) {

	rows, err := r.q.ListItemMachineTimes(ctx, itemCode)
	if err != nil {
		return nil, err
	}

	return itemMachineTimesToEntities(rows), nil
}

func (r *MachineRepositorySQLC) ListItemsByMachine(
	ctx context.Context,
	machineCode int64,
) ([]*entity.ItemMachineTime, error) {

	rows, err := r.q.ListItemsByMachine(ctx, machineCode)
	if err != nil {
		return nil, err
	}

	return itemMachineTimesToEntities(rows), nil
}

//func (r *MachineRepositorySQLC) DeleteItemMachineTime(
//	ctx context.Context,
//	code int64,
//) error {
//
//	return r.q.DeleteItemMachineTime(ctx, code)
//}

func (r *MachineRepositorySQLC) CreateSchedule(ctx context.Context, s *entity.MachineSchedule) (*entity.MachineSchedule, error) {
	row, err := r.q.CreateSchedule(ctx, sqlc.CreateScheduleParams{
		MachineCode:      s.MachineCode,
		OrderCode:        s.OrderCode,
		ScheduleDate:     pgutil.ToPgDate(s.ScheduleDate),
		StartTime:        toPgTime(s.StartTime),
		EndTime:          toPgTime(s.EndTime),
		PlannedQty:       pgutil.ToPgNumericFromFloat64(s.PlannedQty),
		Sequence:         int32(s.Sequence),
		PriorityOverride: int32Ptr(s.PriorityOverride),
		Notes:            pgutil.ToPgTextFromPtr(s.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("create schedule: %w", err)
	}
	return scheduleToEntity(row), nil
}

func (r *MachineRepositorySQLC) GetSchedule(ctx context.Context, code int64) (*entity.MachineSchedule, error) {
	row, err := r.q.GetSchedule(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("schedule %d not found", code)
		}
		return nil, err
	}

	return scheduleToEntity(row), nil
}

func (r *MachineRepositorySQLC) ListSchedules(
	ctx context.Context,
	machineCode int64,
	date time.Time,
) ([]*entity.MachineSchedule, error) {

	rows, err := r.q.ListSchedules(ctx, sqlc.ListSchedulesParams{
		MachineCode:  machineCode,
		ScheduleDate: pgutil.ToPgDate(date),
	})
	if err != nil {
		return nil, err
	}

	out := make([]*entity.MachineSchedule, 0, len(rows))

	for _, r := range rows {
		out = append(out, scheduleToEntity(r))
	}

	return out, nil
}

func (r *MachineRepositorySQLC) ListSchedulesByRange(
	ctx context.Context,
	machineCode int64,
	start,
	end time.Time,
) ([]*entity.MachineSchedule, error) {

	rows, err := r.q.ListSchedulesByRange(ctx, sqlc.ListSchedulesByRangeParams{
		MachineCode:    machineCode,
		ScheduleDate:   pgutil.ToPgDate(start),
		ScheduleDate_2: pgutil.ToPgDate(end),
	})
	if err != nil {
		return nil, err
	}

	out := make([]*entity.MachineSchedule, 0, len(rows))

	for _, r := range rows {
		out = append(out, scheduleToEntity(r))
	}

	return out, nil
}

func (r *MachineRepositorySQLC) UpdateScheduleSequence(
	ctx context.Context,
	code int64,
	sequence int,
	priorityOverride *int,
) (*entity.MachineSchedule, error) {

	row, err := r.q.UpdateScheduleSequence(ctx, sqlc.UpdateScheduleSequenceParams{
		Code:             code,
		Sequence:         int32(sequence),
		PriorityOverride: int32Ptr(priorityOverride),
	})
	if err != nil {
		return nil, err
	}

	return scheduleToEntity(row), nil
}

func (r *MachineRepositorySQLC) UpdateScheduleStatus(
	ctx context.Context,
	code int64,
	status string,
	producedQty float64,
) (*entity.MachineSchedule, error) {

	row, err := r.q.UpdateScheduleStatus(ctx, sqlc.UpdateScheduleStatusParams{
		Code:        code,
		Status:      status,
		ProducedQty: pgutil.ToPgNumericFromFloat64(producedQty),
	})
	if err != nil {
		return nil, err
	}

	return scheduleToEntity(row), nil
}

func (r *MachineRepositorySQLC) UpdateScheduleTimes(
	ctx context.Context,
	code int64,
	startTime,
	endTime *time.Time,
) (*entity.MachineSchedule, error) {

	row, err := r.q.UpdateScheduleTimes(ctx, sqlc.UpdateScheduleTimesParams{
		Code:      code,
		StartTime: toPgTimePtr(startTime),
		EndTime:   toPgTimePtr(endTime),
	})
	if err != nil {
		return nil, err
	}

	return scheduleToEntity(row), nil
}

func (r *MachineRepositorySQLC) DeleteSchedule(
	ctx context.Context,
	code int64,
) error {
	return r.q.DeleteSchedule(ctx, code)
}

func machineTypeToEntity(row sqlc.MachineType) *entity.MachineType {
	return &entity.MachineType{
		Code:             row.Code,
		Name:             row.Name,
		Type:             types.MachineTypeEnum(row.Type),
		Description:      pgutil.FromPgTextPtr(row.Description),
		RequiresOperator: row.RequiresOperator,
		IsActive:         row.IsActive,
		CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:        pgutil.FromPgUUID(row.CreatedBy),
	}
}

func machineToEntity(row sqlc.Machine) *entity.Machine {
	return &entity.Machine{
		Code:            row.Code,
		Name:            row.Name,
		MachineTypeCode: row.MachineTypeCode,
		CostCenterCode:  row.CostCenterCode,
		Capacity:        pgutil.FromPgNumericToFloat64(row.Capacity),
		CapacityPeriod:  types.CapacityPeriod(row.CapacityPeriod),
		CapacityUnit:    types.MachineCapacityUnit(row.CapacityUnit),
		EfficiencyRate:  pgutil.FromPgNumericToFloat64(row.EfficiencyRate),
		IsActive:        row.IsActive,
		CreatedAt:       pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:       pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:       pgutil.FromPgUUID(row.CreatedBy),
	}
}

func itemMachineTimeToEntity(row sqlc.ItemMachineTime) *entity.ItemMachineTime {
	return &entity.ItemMachineTime{
		ItemCode:           row.ItemCode,
		MachineCode:        row.MachineCode,
		Mask:               &row.Mask,
		ProductionTime:     pgutil.FromPgNumericToFloat64(row.ProductionTime),
		SetupTime:          pgutil.FromPgNumericToFloat64(row.SetupTime),
		Priority:           int(row.Priority),
		ProductionTimeUnit: types.CapacityPeriod(row.ProductionTimeUnit),
		ProductionBaseQty:  int(row.ProductionBaseQty),
		CreatedAt:          pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:          pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func scheduleToEntity(row sqlc.MachineSchedule) *entity.MachineSchedule {
	return &entity.MachineSchedule{
		Code:         row.Code,
		MachineCode:  row.MachineCode,
		OrderCode:    row.OrderCode,
		ScheduleDate: pgutil.FromPgDate(row.ScheduleDate),
		StartTime:    fromPgTime(row.StartTime),
		EndTime:      fromPgTime(row.EndTime),
		PlannedQty:   pgutil.FromPgNumericToFloat64(row.PlannedQty),
		ProducedQty:  pgutil.FromPgNumericToFloat64(row.ProducedQty),
		Status:       row.Status,
		Sequence:     int(row.Sequence),
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func machineTypesToEntities(rows []sqlc.MachineType) []*entity.MachineType {
	out := make([]*entity.MachineType, 0, len(rows))
	for _, r := range rows {
		out = append(out, machineTypeToEntity(r))
	}
	return out
}

func machinesToEntities(rows []sqlc.Machine) []*entity.Machine {
	out := make([]*entity.Machine, 0, len(rows))
	for _, r := range rows {
		out = append(out, machineToEntity(r))
	}
	return out
}

func itemMachineTimesToEntities(rows []sqlc.ItemMachineTime) []*entity.ItemMachineTime {
	out := make([]*entity.ItemMachineTime, 0, len(rows))
	for _, r := range rows {
		out = append(out, itemMachineTimeToEntity(r))
	}
	return out
}

func int32Ptr(i *int) *int32 {
	if i == nil {
		return nil
	}
	v := int32(*i)
	return &v
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toPgTime(t *time.Time) pgtype.Time {
	if t == nil {
		return pgtype.Time{}
	}

	us := int64(t.Hour())*3600*1e6 +
		int64(t.Minute())*60*1e6 +
		int64(t.Second())*1e6 +
		int64(t.Nanosecond()/1e3)

	return pgtype.Time{
		Microseconds: us,
		Valid:        true,
	}
}

func fromPgTime(t pgtype.Time) *time.Time {
	if !t.Valid {
		return nil
	}

	totalSec := t.Microseconds / 1e6

	h := int(totalSec / 3600)
	m := int((totalSec % 3600) / 60)
	s := int(totalSec % 60)

	tt := time.Date(0, 1, 1, h, m, s, 0, time.UTC)
	return &tt
}

func toPgTimePtr(t *time.Time) pgtype.Time {
	if t == nil {
		return pgtype.Time{}
	}

	microseconds :=
		int64(t.Hour())*3600*1_000_000 +
			int64(t.Minute())*60*1_000_000 +
			int64(t.Second())*1_000_000 +
			int64(t.Nanosecond()/1000)

	return pgtype.Time{
		Microseconds: microseconds,
		Valid:        true,
	}
}

func fromPgTimePtr(t pgtype.Time) *time.Time {
	if !t.Valid {
		return nil
	}

	totalSeconds := t.Microseconds / 1_000_000

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	tt := time.Date(
		0,
		time.January,
		1,
		int(hours),
		int(minutes),
		int(seconds),
		0,
		time.UTC,
	)

	return &tt
}
