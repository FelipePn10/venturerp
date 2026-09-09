package machine

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *MachineRepositorySQLC) CreateType(ctx context.Context, mt *entity.MachineType) (*entity.MachineType, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateMachineType(ctx, sqlc.CreateMachineTypeParams{
		Code:             mt.Code,
		Name:             mt.Name,
		Description:      pgutil.ToPgTextFromPtr(mt.Description),
		Type:             sqlc.MachineTypeEnum(mt.Type),
		RequiresOperator: mt.RequiresOperator,
		IsActive:         mt.IsActive,
		CreatedBy:        pgutil.ToPgUUID(mt.CreatedBy),
		EnterpriseID:     &enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("create machine type: %w", err)
	}
	return machineTypeToEntity(row), nil
}

func (r *MachineRepositorySQLC) UpdateType(ctx context.Context, mt *entity.MachineType) (*entity.MachineType, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateMachineType(ctx, sqlc.UpdateMachineTypeParams{
		Code:             mt.Code,
		Name:             mt.Name,
		Description:      pgutil.ToPgTextFromPtr(mt.Description),
		Type:             sqlc.MachineTypeEnum(mt.Type),
		RequiresOperator: mt.RequiresOperator,
		IsActive:         mt.IsActive,
		EnterpriseID:     &enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("update machine type: %w", err)
	}
	return machineTypeToEntity(row), nil
}

func (r *MachineRepositorySQLC) GetTypeByCode(ctx context.Context, code int64) (*entity.MachineType, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetMachineTypeByCode(ctx, sqlc.GetMachineTypeByCodeParams{Code: code, EnterpriseID: &enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("tipo de máquina %d não encontrado", code))
		}
		return nil, err
	}
	return machineTypeToEntity(row), nil
}

func (r *MachineRepositorySQLC) ListTypes(ctx context.Context) ([]*entity.MachineType, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListMachineTypes(ctx, &enterpriseID)
	if err != nil {
		return nil, err
	}
	return machineTypesToEntities(rows), nil
}

func (r *MachineRepositorySQLC) ListActiveWorkCenterTypes(ctx context.Context, search string, limit, offset int) ([]*entity.MachineType, int64, error) {
	if r.pool == nil {
		return nil, 0, fmt.Errorf("consulta de centros de trabalho não configurada")
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, 0, err
	}
	search = strings.TrimSpace(search)
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM machine_types WHERE enterprise_id=$1 AND is_active=TRUE AND ($2='' OR code::text ILIKE '%'||$2||'%' OR name ILIKE '%'||$2||'%' OR COALESCE(description,'') ILIKE '%'||$2||'%')`, enterpriseID, search).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id,code,name,description,type,requires_operator,is_active,created_at,updated_at,created_by FROM machine_types WHERE enterprise_id=$1 AND is_active=TRUE AND ($2='' OR code::text ILIKE '%'||$2||'%' OR name ILIKE '%'||$2||'%' OR COALESCE(description,'') ILIKE '%'||$2||'%') ORDER BY code LIMIT $3 OFFSET $4`, enterpriseID, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]*entity.MachineType, 0)
	for rows.Next() {
		var row sqlc.MachineType
		if err := rows.Scan(&row.ID, &row.Code, &row.Name, &row.Description, &row.Type, &row.RequiresOperator, &row.IsActive, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy); err != nil {
			return nil, 0, err
		}
		out = append(out, machineTypeToEntity(row))
	}
	return out, total, rows.Err()
}

func (r *MachineRepositorySQLC) DeleteType(ctx context.Context, code int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteMachineType(ctx, sqlc.DeleteMachineTypeParams{Code: code, EnterpriseID: &enterpriseID})
}

func (r *MachineRepositorySQLC) Create(ctx context.Context, m *entity.Machine) (*entity.Machine, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateMachine(ctx, sqlc.CreateMachineParams{
		Code:                             m.Code,
		Name:                             m.Name,
		MachineTypeCode:                  m.MachineTypeCode,
		CostCenterCode:                   m.CostCenterCode,
		Capacity:                         pgutil.ToPgNumericFromFloat64(m.Capacity),
		CapacityPeriod:                   sqlc.CapacityPeriodEnum(m.CapacityPeriod),
		CapacityUnit:                     sqlc.MachineCapacityUnitEnum(m.CapacityUnit),
		EfficiencyRate:                   pgutil.ToPgNumericFromFloat64(m.EfficiencyRate),
		IsActive:                         m.IsActive,
		ResourceGroupID:                  m.ResourceGroupID,
		CalendarID:                       m.CalendarID,
		Location:                         pgutil.ToPgTextFromPtr(m.Location),
		IsCritical:                       m.IsCritical,
		UsageDescription:                 pgutil.ToPgTextFromPtr(m.UsageDescription),
		AcquiredOn:                       dataOuNula(m.AcquiredOn),
		PreparationTime:                  pgutil.ToPgNumericFromFloat64(m.PreparationTime),
		PreparationTimeUnit:              unidadeOuPadrao(m.PreparationTimeUnit),
		SupplierCode:                     m.SupplierCode,
		Brand:                            pgutil.ToPgTextFromPtr(m.Brand),
		IsPreferred:                      m.IsPreferred,
		MaintenanceResponsibleEmployeeID: m.MaintenanceResponsibleEmployeeID,
		CreatedBy:                        pgutil.ToPgUUID(m.CreatedBy),
		EnterpriseID:                     &enterpriseID,
	})
	if err != nil {
		return nil, fmt.Errorf("create machine: %w", err)
	}
	return machineToEntity(row), nil
}

func (r *MachineRepositorySQLC) Update(ctx context.Context, m *entity.Machine) (*entity.Machine, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateMachine(ctx, sqlc.UpdateMachineParams{
		Name:                             m.Name,
		MachineTypeCode:                  m.MachineTypeCode,
		CostCenterCode:                   m.CostCenterCode,
		Capacity:                         pgutil.ToPgNumericFromFloat64(m.Capacity),
		CapacityPeriod:                   sqlc.CapacityPeriodEnum(m.CapacityPeriod),
		CapacityUnit:                     sqlc.MachineCapacityUnitEnum(m.CapacityUnit),
		EfficiencyRate:                   pgutil.ToPgNumericFromFloat64(m.EfficiencyRate),
		IsActive:                         m.IsActive,
		ResourceGroupID:                  m.ResourceGroupID,
		CalendarID:                       m.CalendarID,
		Location:                         pgutil.ToPgTextFromPtr(m.Location),
		IsCritical:                       m.IsCritical,
		UsageDescription:                 pgutil.ToPgTextFromPtr(m.UsageDescription),
		AcquiredOn:                       dataOuNula(m.AcquiredOn),
		PreparationTime:                  pgutil.ToPgNumericFromFloat64(m.PreparationTime),
		PreparationTimeUnit:              unidadeOuPadrao(m.PreparationTimeUnit),
		SupplierCode:                     m.SupplierCode,
		Brand:                            pgutil.ToPgTextFromPtr(m.Brand),
		IsPreferred:                      m.IsPreferred,
		MaintenanceResponsibleEmployeeID: m.MaintenanceResponsibleEmployeeID,
		Code:                             m.Code,
		EnterpriseID:                     &enterpriseID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("máquina %d não encontrada nesta empresa", m.Code))
		}
		return nil, fmt.Errorf("update machine: %w", err)
	}
	return machineToEntity(row), nil
}

func (r *MachineRepositorySQLC) GetByCode(ctx context.Context, code int64) (*entity.Machine, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetMachineByCode(ctx, sqlc.GetMachineByCodeParams{Code: code, EnterpriseID: &enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("máquina %d não encontrada", code))
		}
		return nil, err
	}
	return machineToEntity(row), nil
}

func (r *MachineRepositorySQLC) List(ctx context.Context) ([]*entity.Machine, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListMachines(ctx, &enterpriseID)
	if err != nil {
		return nil, err
	}
	return machinesToEntities(rows), nil
}

func (r *MachineRepositorySQLC) ListByType(ctx context.Context, typeCode int64) ([]*entity.Machine, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListMachinesByType(ctx, sqlc.ListMachinesByTypeParams{MachineTypeCode: typeCode, EnterpriseID: &enterpriseID})
	if err != nil {
		return nil, err
	}
	return machinesToEntities(rows), nil
}

func (r *MachineRepositorySQLC) Delete(ctx context.Context, code int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.q.DeleteMachine(ctx, sqlc.DeleteMachineParams{Code: code, EnterpriseID: &enterpriseID})
}

func (r *MachineRepositorySQLC) CreateItemMachineTime(ctx context.Context, imt *entity.ItemMachineTime) (*entity.ItemMachineTime, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateItemMachineTime(ctx, sqlc.CreateItemMachineTimeParams{
		ItemCode:           imt.ItemCode,
		Mask:               ptrToString(imt.Mask),
		MachineCode:        imt.MachineCode,
		ProductionTime:     pgutil.ToPgNumericFromFloat64(imt.ProductionTime),
		ProductionTimeUnit: sqlc.CapacityPeriodEnum(imt.ProductionTimeUnit),
		ProductionBaseQty:  int32(imt.ProductionBaseQty),
		SetupTime:          pgutil.ToPgNumericFromFloat64(imt.SetupTime),
		Priority:           int32(imt.Priority),
		EnterpriseID:       &enterpriseID,
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListItemMachineTimes(ctx, sqlc.ListItemMachineTimesParams{ItemCode: itemCode, EnterpriseID: &enterpriseID})
	if err != nil {
		return nil, err
	}

	return itemMachineTimesToEntities(rows), nil
}

func (r *MachineRepositorySQLC) ListItemsByMachine(
	ctx context.Context,
	machineCode int64,
) ([]*entity.ItemMachineTime, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListItemsByMachine(ctx, sqlc.ListItemsByMachineParams{MachineCode: machineCode, EnterpriseID: &enterpriseID})
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateSchedule(ctx, sqlc.CreateScheduleParams{
		EnterpriseID:     &enterpriseID,
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetSchedule(ctx, sqlc.GetScheduleParams{Code: code, EnterpriseID: &enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("programação %d não encontrada", code))
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

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListSchedules(ctx, sqlc.ListSchedulesParams{
		MachineCode:  machineCode,
		ScheduleDate: pgutil.ToPgDate(date),
		EnterpriseID: &enterpriseID,
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

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListSchedulesByRange(ctx, sqlc.ListSchedulesByRangeParams{
		MachineCode:    machineCode,
		ScheduleDate:   pgutil.ToPgDate(start),
		ScheduleDate_2: pgutil.ToPgDate(end),
		EnterpriseID:   &enterpriseID,
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

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateScheduleSequence(ctx, sqlc.UpdateScheduleSequenceParams{
		Code:             code,
		Sequence:         int32(sequence),
		PriorityOverride: int32Ptr(priorityOverride),
		EnterpriseID:     &enterpriseID,
	})
	if err != nil {
		return nil, semLinhaVira404(err, code)
	}

	return scheduleToEntity(row), nil
}

func (r *MachineRepositorySQLC) UpdateScheduleStatus(
	ctx context.Context,
	code int64,
	status string,
	producedQty float64,
) (*entity.MachineSchedule, error) {

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateScheduleStatus(ctx, sqlc.UpdateScheduleStatusParams{
		Code:         code,
		Status:       status,
		ProducedQty:  pgutil.ToPgNumericFromFloat64(producedQty),
		EnterpriseID: &enterpriseID,
	})
	if err != nil {
		return nil, semLinhaVira404(err, code)
	}

	return scheduleToEntity(row), nil
}

func (r *MachineRepositorySQLC) UpdateScheduleTimes(
	ctx context.Context,
	code int64,
	startTime,
	endTime *time.Time,
) (*entity.MachineSchedule, error) {

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateScheduleTimes(ctx, sqlc.UpdateScheduleTimesParams{
		Code:         code,
		StartTime:    toPgTimePtr(startTime),
		EndTime:      toPgTimePtr(endTime),
		EnterpriseID: &enterpriseID,
	})
	if err != nil {
		return nil, semLinhaVira404(err, code)
	}

	return scheduleToEntity(row), nil
}

func (r *MachineRepositorySQLC) DeleteSchedule(
	ctx context.Context,
	code int64,
) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	linhas, err := r.q.DeleteSchedule(ctx, sqlc.DeleteScheduleParams{Code: code, EnterpriseID: &enterpriseID})
	if err != nil {
		return err
	}
	if linhas == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("programação %d não encontrada", code))
	}
	return nil
}

// semLinhaVira404 traduz "nenhuma linha atualizada" para não encontrado: com o
// filtro por empresa, um código de outra empresa some do UPDATE em vez de dar
// erro, e o usuário precisa saber que o registro não é dele.
func semLinhaVira404(err error, code int64) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("programação %d não encontrada", code))
	}
	return err
}

func machineTypeToEntity(row sqlc.MachineType) *entity.MachineType {
	return &entity.MachineType{
		ID:               row.ID,
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
		ID:              row.ID,
		Code:            row.Code,
		Name:            row.Name,
		MachineTypeCode: row.MachineTypeCode,
		CostCenterCode:  row.CostCenterCode,
		Capacity:        pgutil.FromPgNumericToFloat64(row.Capacity),
		CapacityPeriod:  types.CapacityPeriod(row.CapacityPeriod),
		CapacityUnit:    types.MachineCapacityUnit(row.CapacityUnit),
		EfficiencyRate:  pgutil.FromPgNumericToFloat64(row.EfficiencyRate),
		IsActive:        row.IsActive,

		ResourceGroupID:                  row.ResourceGroupID,
		CalendarID:                       row.CalendarID,
		Location:                         textoOuNil(row.Location),
		IsCritical:                       row.IsCritical,
		UsageDescription:                 textoOuNil(row.UsageDescription),
		AcquiredOn:                       dataOuNil(row.AcquiredOn),
		PreparationTime:                  pgutil.FromPgNumericToFloat64(row.PreparationTime),
		PreparationTimeUnit:              row.PreparationTimeUnit,
		SupplierCode:                     row.SupplierCode,
		Brand:                            textoOuNil(row.Brand),
		IsPreferred:                      row.IsPreferred,
		MaintenanceResponsibleEmployeeID: row.MaintenanceResponsibleEmployeeID,

		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt: pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
	}
}

// dataOuNil é o caminho de volta de dataOuNula.
func dataOuNil(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
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
		// Sem estes dois a prioridade manual e a observação eram gravadas e
		// nunca liam de volta: a coluna aparecia vazia na tela logo depois de
		// o programador digitar o valor.
		PriorityOverride: int32PtrToIntPtr(row.PriorityOverride),
		Notes:            textoOuNil(row.Notes),
		CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

// dataOuNula converte a data de aquisição, aceitando "não informada".
func dataOuNula(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

// unidadeOuPadrao garante a unidade do tempo de preparação (coluna NOT NULL,
// com CHECK que só aceita MINUTE ou HOUR).
func unidadeOuPadrao(u string) string {
	if strings.ToUpper(strings.TrimSpace(u)) == "HOUR" {
		return "HOUR"
	}
	return "MINUTE"
}

// textoOuNil devolve nil para coluna nula, em vez de string vazia.
func textoOuNil(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

// int32PtrToIntPtr converte a prioridade manual (int32 no banco) para o int do
// domínio, preservando "não informado".
func int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
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
