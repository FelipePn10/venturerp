package aps

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *APSRepositorySQLC) UpsertResourceGroup(ctx context.Context, code, description string) (domainrepo.ResourceGroup, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return domainrepo.ResourceGroup{}, err
	}
	var v domainrepo.ResourceGroup
	err = r.pool.QueryRow(ctx, `INSERT INTO production_resource_groups(enterprise_id,code,description) VALUES($1,$2,$3) ON CONFLICT(enterprise_id,code) DO UPDATE SET description=EXCLUDED.description,updated_at=NOW() RETURNING id,code,description`, enterpriseID, code, description).Scan(&v.ID, &v.Code, &v.Description)
	return v, err
}
func (r *APSRepositorySQLC) ListResourceGroups(ctx context.Context) ([]domainrepo.ResourceGroup, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id,code,description FROM production_resource_groups WHERE enterprise_id=$1 ORDER BY code`, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domainrepo.ResourceGroup{}
	for rows.Next() {
		var v domainrepo.ResourceGroup
		if err := rows.Scan(&v.ID, &v.Code, &v.Description); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *APSRepositorySQLC) UpsertMachineCalendar(ctx context.Context, code int64, description string, intervals []domainrepo.MachineCalendarInterval) (domainrepo.MachineCalendar, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return domainrepo.MachineCalendar{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domainrepo.MachineCalendar{}, err
	}
	defer tx.Rollback(ctx)
	v := domainrepo.MachineCalendar{Code: code, Description: description, Intervals: intervals}
	if err = tx.QueryRow(ctx, `INSERT INTO machine_calendars(enterprise_id,code,description) VALUES($1,$2,$3) ON CONFLICT(enterprise_id,code) DO UPDATE SET description=EXCLUDED.description,updated_at=NOW() RETURNING id`, enterpriseID, code, description).Scan(&v.ID); err != nil {
		return v, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM machine_calendar_intervals WHERE calendar_id=$1`, v.ID); err != nil {
		return v, err
	}
	for _, in := range intervals {
		if _, err = tx.Exec(ctx, `INSERT INTO machine_calendar_intervals(calendar_id,weekday,start_time,end_time) VALUES($1,$2,$3::time,$4::time)`, v.ID, in.Weekday, in.Start, in.End); err != nil {
			return v, err
		}
	}
	return v, tx.Commit(ctx)
}
func (r *APSRepositorySQLC) ListMachineCalendars(ctx context.Context) ([]domainrepo.MachineCalendar, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT c.id,c.code,c.description,i.weekday,COALESCE(to_char(i.start_time,'HH24:MI'),''),COALESCE(to_char(i.end_time,'HH24:MI'),'') FROM machine_calendars c LEFT JOIN machine_calendar_intervals i ON i.calendar_id=c.id WHERE c.enterprise_id=$1 ORDER BY c.code,i.weekday,i.start_time`, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domainrepo.MachineCalendar{}
	index := map[int64]int{}
	for rows.Next() {
		var id, code int64
		var desc string
		var weekday *int
		var start, end string
		if err := rows.Scan(&id, &code, &desc, &weekday, &start, &end); err != nil {
			return nil, err
		}
		pos, ok := index[id]
		if !ok {
			pos = len(out)
			index[id] = pos
			out = append(out, domainrepo.MachineCalendar{ID: id, Code: code, Description: desc})
		}
		if weekday != nil {
			out[pos].Intervals = append(out[pos].Intervals, domainrepo.MachineCalendarInterval{Weekday: *weekday, Start: start, End: end})
		}
	}
	return out, rows.Err()
}
func (r *APSRepositorySQLC) UpdateSequencingSettings(ctx context.Context, active bool) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO manufacturing_sequencing_settings(enterprise_id,list_only_active_resources) VALUES($1,$2) ON CONFLICT(enterprise_id) DO UPDATE SET list_only_active_resources=$2,updated_at=NOW()`, enterpriseID, active)
	return err
}
func (r *APSRepositorySQLC) UpdateWorkCenterSequencing(ctx context.Context, id int64, machineCC, laborCC *int64, capacity string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE machine_types SET machine_cost_center_id=$3,labor_cost_center_id=$4,capacity_hours=$5::numeric,updated_at=NOW() WHERE id=$2 AND enterprise_id=$1`, enterpriseID, id, machineCC, laborCC, capacity)
	if err == nil && tag.RowsAffected() == 0 {
		return fmt.Errorf("work center not found")
	}
	return err
}
func (r *APSRepositorySQLC) UpdateResourceSequencing(ctx context.Context, id int64, group, calendar *int64, location string, critical, active bool) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE machines SET resource_group_id=$3,calendar_id=$4,location=NULLIF($5,''),is_critical=$6,is_active=$7,updated_at=NOW() WHERE id=$2 AND enterprise_id=$1 AND ($3::bigint IS NULL OR EXISTS(SELECT 1 FROM production_resource_groups g WHERE g.id=$3 AND g.enterprise_id=$1)) AND ($4::bigint IS NULL OR EXISTS(SELECT 1 FROM machine_calendars c WHERE c.id=$4 AND c.enterprise_id=$1))`, enterpriseID, id, group, calendar, strings.TrimSpace(location), critical, active)
	if err == nil && tag.RowsAffected() == 0 {
		return fmt.Errorf("resource or tenant configuration not found")
	}
	return err
}
func (r *APSRepositorySQLC) DeleteResourceGroup(ctx context.Context, id int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM production_resource_groups WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID)
	if err == nil && tag.RowsAffected() == 0 {
		return fmt.Errorf("resource group not found")
	}
	return err
}
func (r *APSRepositorySQLC) DeleteMachineCalendar(ctx context.Context, id int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM machine_calendars WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID)
	if err == nil && tag.RowsAffected() == 0 {
		return fmt.Errorf("machine calendar not found")
	}
	return err
}
func (r *APSRepositorySQLC) CreateMachineDowntime(ctx context.Context, v domainrepo.MachineDowntime) (domainrepo.MachineDowntime, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return v, err
	}
	err = r.pool.QueryRow(ctx, `INSERT INTO machine_downtimes(enterprise_id,machine_id,starts_at,ends_at,downtime_type,reason,maintenance_order_id) SELECT $1,$2,$3,$4,$5,$6,$7 WHERE EXISTS(SELECT 1 FROM machines WHERE id=$2 AND enterprise_id=$1) RETURNING id`, enterpriseID, v.MachineID, v.StartsAt, v.EndsAt, v.DowntimeType, v.Reason, v.MaintenanceOrderID).Scan(&v.ID)
	return v, err
}
func (r *APSRepositorySQLC) ListMachineDowntimes(ctx context.Context, machineID int64, from, to time.Time) ([]domainrepo.MachineDowntime, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id,machine_id,starts_at,ends_at,downtime_type,reason,maintenance_order_id FROM machine_downtimes WHERE enterprise_id=$1 AND ($2=0 OR machine_id=$2) AND starts_at<$4 AND ends_at>$3 ORDER BY starts_at`, enterpriseID, machineID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domainrepo.MachineDowntime{}
	for rows.Next() {
		var v domainrepo.MachineDowntime
		if err := rows.Scan(&v.ID, &v.MachineID, &v.StartsAt, &v.EndsAt, &v.DowntimeType, &v.Reason, &v.MaintenanceOrderID); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *APSRepositorySQLC) DeleteMachineDowntime(ctx context.Context, id int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM machine_downtimes WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID)
	if err == nil && tag.RowsAffected() == 0 {
		return fmt.Errorf("downtime not found")
	}
	return err
}
func (r *APSRepositorySQLC) UpsertEmployeeSequencingProfile(ctx context.Context, id int64, p domainrepo.EmployeeSequencingProfile) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND enterprise_id=$2)`, id, enterpriseID).Scan(&exists); err != nil || !exists {
		if err == nil {
			err = fmt.Errorf("employee not found")
		}
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM employee_contacts WHERE employee_id=$1 AND enterprise_id=$2`, id, enterpriseID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM employee_functions WHERE employee_id=$1 AND enterprise_id=$2`, id, enterpriseID); err != nil {
		return err
	}
	for _, v := range p.Contacts {
		if _, err = tx.Exec(ctx, `INSERT INTO employee_contacts(enterprise_id,employee_id,contact_type,value,is_primary) VALUES($1,$2,$3,$4,$5)`, enterpriseID, id, v.ContactType, v.Value, v.IsPrimary); err != nil {
			return err
		}
	}
	for _, v := range p.Functions {
		if _, err = tx.Exec(ctx, `INSERT INTO employee_functions(enterprise_id,employee_id,function_name,cost_center_id,is_supervisor,is_manager) VALUES($1,$2,$3,$4,$5,$6)`, enterpriseID, id, v.FunctionName, v.CostCenterID, v.IsSupervisor, v.IsManager); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO employee_credit_limits(enterprise_id,employee_id,credit_limit,valid_until) VALUES($1,$2,$3::numeric,$4) ON CONFLICT(employee_id) DO UPDATE SET credit_limit=$3::numeric,valid_until=$4,updated_at=NOW()`, enterpriseID, id, p.CreditLimit, p.ValidUntil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *APSRepositorySQLC) UpsertMachineIndustrialProfile(ctx context.Context, id int64, p domainrepo.MachineIndustrialProfile) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE machines SET usage_description=NULLIF($3,''),acquired_on=$4,preparation_time=$5::numeric,preparation_time_unit=$6,supplier_code=$7,brand=NULLIF($8,''),is_preferred=$9,maintenance_responsible_employee_id=$10,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID, p.UsageDescription, p.AcquiredOn, p.PreparationTime, p.PreparationTimeUnit, p.SupplierCode, p.Brand, p.IsPreferred, p.MaintenanceResponsibleEmployeeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("machine not found")
	}
	if _, err = tx.Exec(ctx, `DELETE FROM machine_preventive_services WHERE machine_id=$1 AND enterprise_id=$2`, id, enterpriseID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM machine_special_values WHERE machine_id=$1 AND enterprise_id=$2`, id, enterpriseID); err != nil {
		return err
	}
	for _, s := range p.Services {
		var serviceID int64
		if err = tx.QueryRow(ctx, `INSERT INTO preventive_services(enterprise_id,code,description,service_type) VALUES($1,$2,$3,$4) ON CONFLICT(enterprise_id,code) DO UPDATE SET description=$3,service_type=$4,updated_at=NOW() RETURNING id`, enterpriseID, s.ServiceCode, s.Description, s.ServiceType).Scan(&serviceID); err != nil {
			return err
		}
		var linkID int64
		if err = tx.QueryRow(ctx, `INSERT INTO machine_preventive_services(enterprise_id,machine_id,service_id,frequency_value,frequency_unit,max_tolerance,supplier_code,implemented_on,last_executed_on,notes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')) RETURNING id`, enterpriseID, id, serviceID, s.FrequencyValue, s.FrequencyUnit, s.MaxTolerance, s.SupplierCode, s.ImplementedOn, s.LastExecutedOn, s.Notes).Scan(&linkID); err != nil {
			return err
		}
		for _, item := range s.Items {
			if _, err = tx.Exec(ctx, `INSERT INTO machine_service_items(enterprise_id,machine_service_id,item_code,quantity,notes) VALUES($1,$2,$3,$4::numeric,NULLIF($5,''))`, enterpriseID, linkID, item.ItemCode, item.Quantity, item.Notes); err != nil {
				return err
			}
		}
		for _, employeeID := range s.ResponsibleEmployeeIDs {
			if _, err = tx.Exec(ctx, `INSERT INTO machine_service_responsibles(machine_service_id,employee_id,enterprise_id) SELECT $1,$2,$3 WHERE EXISTS(SELECT 1 FROM employees WHERE id=$2 AND enterprise_id=$3)`, linkID, employeeID, enterpriseID); err != nil {
				return err
			}
		}
	}
	for _, v := range p.SpecialValues {
		var fieldID int64
		if err = tx.QueryRow(ctx, `INSERT INTO machine_special_fields(enterprise_id,name,value_type,max_length) VALUES($1,$2,$3,$4) ON CONFLICT(enterprise_id,name) DO UPDATE SET value_type=$3,max_length=$4 RETURNING id`, enterpriseID, v.Name, v.ValueType, v.MaxLength).Scan(&fieldID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO machine_special_values(machine_id,field_id,enterprise_id,text_value,numeric_value) VALUES($1,$2,$3,NULLIF($4,''),NULLIF($5,'')::numeric)`, id, fieldID, enterpriseID, v.TextValue, v.NumericValue); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (r *APSRepositorySQLC) GetEmployeeSequencingProfile(ctx context.Context, id int64) (domainrepo.EmployeeSequencingProfile, error) {
	enterpriseID, err := tenant.ID(ctx)
	p := domainrepo.EmployeeSequencingProfile{}
	if err != nil {
		return p, err
	}
	if err = r.pool.QueryRow(ctx, `SELECT credit_limit::text,valid_until FROM employee_credit_limits WHERE employee_id=$1 AND enterprise_id=$2`, id, enterpriseID).Scan(&p.CreditLimit, &p.ValidUntil); err != nil {
		return p, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id,contact_type,value,is_primary FROM employee_contacts WHERE employee_id=$1 AND enterprise_id=$2 ORDER BY id`, id, enterpriseID)
	if err != nil {
		return p, err
	}
	for rows.Next() {
		var v domainrepo.EmployeeContact
		if err = rows.Scan(&v.ID, &v.ContactType, &v.Value, &v.IsPrimary); err != nil {
			rows.Close()
			return p, err
		}
		p.Contacts = append(p.Contacts, v)
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT id,function_name,cost_center_id,is_supervisor,is_manager FROM employee_functions WHERE employee_id=$1 AND enterprise_id=$2 ORDER BY id`, id, enterpriseID)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domainrepo.EmployeeFunction
		if err = rows.Scan(&v.ID, &v.FunctionName, &v.CostCenterID, &v.IsSupervisor, &v.IsManager); err != nil {
			return p, err
		}
		p.Functions = append(p.Functions, v)
	}
	return p, rows.Err()
}
func (r *APSRepositorySQLC) GetMachineIndustrialProfile(ctx context.Context, id int64) (domainrepo.MachineIndustrialProfile, error) {
	enterpriseID, err := tenant.ID(ctx)
	p := domainrepo.MachineIndustrialProfile{}
	if err != nil {
		return p, err
	}
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(usage_description,''),acquired_on,preparation_time::text,preparation_time_unit,supplier_code,COALESCE(brand,''),is_preferred,maintenance_responsible_employee_id FROM machines WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID).Scan(&p.UsageDescription, &p.AcquiredOn, &p.PreparationTime, &p.PreparationTimeUnit, &p.SupplierCode, &p.Brand, &p.IsPreferred, &p.MaintenanceResponsibleEmployeeID)
	if err != nil {
		return p, err
	}
	rows, err := r.pool.Query(ctx, `SELECT ms.id,s.code,s.description,s.service_type,ms.frequency_value,ms.frequency_unit,ms.max_tolerance,ms.supplier_code,ms.implemented_on,ms.last_executed_on,COALESCE(ms.notes,'') FROM machine_preventive_services ms JOIN preventive_services s ON s.id=ms.service_id WHERE ms.machine_id=$1 AND ms.enterprise_id=$2 ORDER BY ms.id`, id, enterpriseID)
	if err != nil {
		return p, err
	}
	for rows.Next() {
		var v domainrepo.MachineService
		if err = rows.Scan(&v.ID, &v.ServiceCode, &v.Description, &v.ServiceType, &v.FrequencyValue, &v.FrequencyUnit, &v.MaxTolerance, &v.SupplierCode, &v.ImplementedOn, &v.LastExecutedOn, &v.Notes); err != nil {
			rows.Close()
			return p, err
		}
		itemRows, e := r.pool.Query(ctx, `SELECT id,item_code,quantity::text,COALESCE(notes,'') FROM machine_service_items WHERE machine_service_id=$1 AND enterprise_id=$2 ORDER BY id`, v.ID, enterpriseID)
		if e != nil {
			rows.Close()
			return p, e
		}
		for itemRows.Next() {
			var item domainrepo.ServiceItem
			if e = itemRows.Scan(&item.ID, &item.ItemCode, &item.Quantity, &item.Notes); e != nil {
				itemRows.Close()
				rows.Close()
				return p, e
			}
			v.Items = append(v.Items, item)
		}
		itemRows.Close()
		respRows, e := r.pool.Query(ctx, `SELECT employee_id FROM machine_service_responsibles WHERE machine_service_id=$1 AND enterprise_id=$2`, v.ID, enterpriseID)
		if e != nil {
			rows.Close()
			return p, e
		}
		for respRows.Next() {
			var employeeID int64
			if e = respRows.Scan(&employeeID); e != nil {
				respRows.Close()
				rows.Close()
				return p, e
			}
			v.ResponsibleEmployeeIDs = append(v.ResponsibleEmployeeIDs, employeeID)
		}
		respRows.Close()
		p.Services = append(p.Services, v)
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT f.id,f.name,f.value_type,f.max_length,COALESCE(v.text_value,''),COALESCE(v.numeric_value::text,'') FROM machine_special_values v JOIN machine_special_fields f ON f.id=v.field_id WHERE v.machine_id=$1 AND v.enterprise_id=$2 ORDER BY f.id`, id, enterpriseID)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domainrepo.SpecialValue
		if err = rows.Scan(&v.FieldID, &v.Name, &v.ValueType, &v.MaxLength, &v.TextValue, &v.NumericValue); err != nil {
			return p, err
		}
		p.SpecialValues = append(p.SpecialValues, v)
	}
	return p, rows.Err()
}

func (r *APSRepositorySQLC) UpdateEmployeeContact(ctx context.Context, employeeID, contactID int64, v domainrepo.EmployeeContact) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE employee_contacts SET contact_type=$4,value=$5,is_primary=$6 WHERE id=$3 AND employee_id=$2 AND enterprise_id=$1`, enterpriseID, employeeID, contactID, v.ContactType, v.Value, v.IsPrimary)
	return affectedOrNotFound(tag, err, "employee contact")
}

func (r *APSRepositorySQLC) DeleteEmployeeContact(ctx context.Context, employeeID, contactID int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM employee_contacts WHERE id=$3 AND employee_id=$2 AND enterprise_id=$1`, enterpriseID, employeeID, contactID)
	return affectedOrNotFound(tag, err, "employee contact")
}

func (r *APSRepositorySQLC) UpdateEmployeeFunction(ctx context.Context, employeeID, functionID int64, v domainrepo.EmployeeFunction) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE employee_functions f SET function_name=$4,cost_center_id=$5,is_supervisor=$6,is_manager=$7 WHERE f.id=$3 AND f.employee_id=$2 AND f.enterprise_id=$1 AND ($5::bigint IS NULL OR EXISTS(SELECT 1 FROM cost_centers c WHERE c.id=$5))`, enterpriseID, employeeID, functionID, v.FunctionName, v.CostCenterID, v.IsSupervisor, v.IsManager)
	return affectedOrNotFound(tag, err, "employee function")
}

func (r *APSRepositorySQLC) DeleteEmployeeFunction(ctx context.Context, employeeID, functionID int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM employee_functions WHERE id=$3 AND employee_id=$2 AND enterprise_id=$1`, enterpriseID, employeeID, functionID)
	return affectedOrNotFound(tag, err, "employee function")
}

func (r *APSRepositorySQLC) UpdateMachineService(ctx context.Context, machineID, linkID int64, v domainrepo.MachineService) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var serviceID int64
	err = tx.QueryRow(ctx, `UPDATE preventive_services s SET code=$4,description=$5,service_type=$6,updated_at=NOW() FROM machine_preventive_services ms WHERE ms.id=$3 AND ms.machine_id=$2 AND ms.enterprise_id=$1 AND s.id=ms.service_id AND s.enterprise_id=$1 RETURNING s.id`, enterpriseID, machineID, linkID, v.ServiceCode, v.Description, v.ServiceType).Scan(&serviceID)
	if err != nil {
		return fmt.Errorf("machine service not found: %w", err)
	}
	tag, err := tx.Exec(ctx, `UPDATE machine_preventive_services SET frequency_value=$4,frequency_unit=$5,max_tolerance=$6,supplier_code=$7,implemented_on=$8,last_executed_on=$9,notes=NULLIF($10,'') WHERE id=$3 AND machine_id=$2 AND enterprise_id=$1`, enterpriseID, machineID, linkID, v.FrequencyValue, v.FrequencyUnit, v.MaxTolerance, v.SupplierCode, v.ImplementedOn, v.LastExecutedOn, v.Notes)
	if err = affectedOrNotFound(tag, err, "machine service"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM machine_service_responsibles WHERE machine_service_id=$1 AND enterprise_id=$2`, linkID, enterpriseID); err != nil {
		return err
	}
	for _, employeeID := range v.ResponsibleEmployeeIDs {
		tag, err = tx.Exec(ctx, `INSERT INTO machine_service_responsibles(machine_service_id,employee_id,enterprise_id) SELECT $1,$2,$3 WHERE EXISTS(SELECT 1 FROM employees WHERE id=$2 AND enterprise_id=$3) ON CONFLICT DO NOTHING`, linkID, employeeID, enterpriseID)
		if err != nil || tag.RowsAffected() == 0 {
			if err == nil {
				err = fmt.Errorf("responsible employee not found")
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *APSRepositorySQLC) DeleteMachineService(ctx context.Context, machineID, linkID int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM machine_preventive_services WHERE id=$3 AND machine_id=$2 AND enterprise_id=$1`, enterpriseID, machineID, linkID)
	return affectedOrNotFound(tag, err, "machine service")
}

func (r *APSRepositorySQLC) UpdateMachineServiceItem(ctx context.Context, machineID, linkID, itemID int64, v domainrepo.ServiceItem) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE machine_service_items i SET item_code=$5,quantity=$6::numeric,notes=NULLIF($7,'') FROM machine_preventive_services ms WHERE i.id=$4 AND i.machine_service_id=$3 AND i.enterprise_id=$1 AND ms.id=$3 AND ms.machine_id=$2 AND ms.enterprise_id=$1`, enterpriseID, machineID, linkID, itemID, v.ItemCode, v.Quantity, v.Notes)
	return affectedOrNotFound(tag, err, "machine service item")
}

func (r *APSRepositorySQLC) DeleteMachineServiceItem(ctx context.Context, machineID, linkID, itemID int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM machine_service_items i USING machine_preventive_services ms WHERE i.id=$4 AND i.machine_service_id=$3 AND i.enterprise_id=$1 AND ms.id=$3 AND ms.machine_id=$2 AND ms.enterprise_id=$1`, enterpriseID, machineID, linkID, itemID)
	return affectedOrNotFound(tag, err, "machine service item")
}

func (r *APSRepositorySQLC) UpdateMachineSpecialValue(ctx context.Context, machineID, fieldID int64, v domainrepo.SpecialValue) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE machine_special_fields f SET name=$4,value_type=$5,max_length=$6 FROM machine_special_values v WHERE f.id=$3 AND f.enterprise_id=$1 AND v.field_id=f.id AND v.machine_id=$2 AND v.enterprise_id=$1`, enterpriseID, machineID, fieldID, v.Name, v.ValueType, v.MaxLength)
	if err = affectedOrNotFound(tag, err, "machine special value"); err != nil {
		return err
	}
	tag, err = tx.Exec(ctx, `UPDATE machine_special_values SET text_value=NULLIF($4,''),numeric_value=NULLIF($5,'')::numeric WHERE machine_id=$2 AND field_id=$3 AND enterprise_id=$1`, enterpriseID, machineID, fieldID, v.TextValue, v.NumericValue)
	if err = affectedOrNotFound(tag, err, "machine special value"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *APSRepositorySQLC) DeleteMachineSpecialValue(ctx context.Context, machineID, fieldID int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM machine_special_values WHERE machine_id=$2 AND field_id=$3 AND enterprise_id=$1`, enterpriseID, machineID, fieldID)
	return affectedOrNotFound(tag, err, "machine special value")
}

func affectedOrNotFound(tag interface{ RowsAffected() int64 }, err error, resource string) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s not found", resource)
	}
	return nil
}
