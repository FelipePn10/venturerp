package mrp_calculation

import (
	"context"
	"errors"
	"fmt"
	machine "github.com/FelipePn10/panossoerp/internal/domain/machine/service"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routing "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"time"
)

func (r *MRPCalculationRepositorySQLC) WithMachinePlanningTransaction(ctx context.Context, work func(domainrepo.MRPCalculationRepository) error) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	beginner, ok := r.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fmt.Errorf("transação de planejamento de máquinas não configurada")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Serializes machine replanning within this tenant, including different MRP plans.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(348,$1::int)`, e); err != nil {
		return err
	}
	// O agendamento traz paradas e sequências de colunas timestamptz para hora de
	// parede com ::timestamp, e essa conversão usa o fuso da SESSÃO. Sem fixá-lo,
	// o resultado depende da configuração do servidor: com o Postgres em UTC — o
	// padrão da imagem oficial — uma parada das 08:00 vira 11:00 e bloqueia três
	// horas erradas do turno, sem erro nenhum. O fuso da empresa é o mesmo já
	// usado para agendar notificações.
	if _, err = tx.Exec(ctx, `SELECT set_config('TimeZone',COALESCE((SELECT NULLIF(timezone,'') FROM enterprise_notification_settings WHERE enterprise_id=$1),'America/Sao_Paulo'),true)`, e); err != nil {
		return err
	}
	child := NewMRPCalculationRepositorySQLC(r.q.WithTx(tx), tx)
	if err = work(child); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *MRPCalculationRepositorySQLC) MachinePlanningWindows(ctx context.Context, machineID, plan int64, from, to time.Time) ([]machine.CapacityWindow, []machine.CapacityWindow, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Turno com fim menor ou igual ao início atravessa a meia-noite: 22:00–06:00
	// é uma janela contínua de segunda 22:00 a terça 06:00. Sem o dia extra a
	// janela nasceria invertida e seria descartada em silêncio pelo alocador.
	rows, err := r.db.Query(ctx, `SELECT day::date+COALESCE(i.start_time,TIME '00:00'),
 day::date+COALESCE(i.end_time,TIME '00:00')+CASE WHEN i.id IS NULL THEN make_interval(secs=>(COALESCE(m.available_hours_per_day,mt.capacity_hours)*3600)::double precision) ELSE INTERVAL '0' END
 +CASE WHEN i.id IS NOT NULL AND i.end_time<=i.start_time THEN INTERVAL '1 day' ELSE INTERVAL '0' END
 FROM generate_series($3::date,$4::date,'1 day') day
 JOIN machines m ON m.id=$1 AND m.enterprise_id=$2 AND m.is_active
 JOIN machine_types mt ON mt.code=m.machine_type_code AND mt.enterprise_id=m.enterprise_id AND mt.is_active
 LEFT JOIN machine_calendars c ON c.id=m.calendar_id AND c.enterprise_id=$2
 LEFT JOIN machine_calendar_intervals i ON i.calendar_id=c.id AND i.weekday=EXTRACT(DOW FROM day)
 WHERE i.id IS NOT NULL OR m.calendar_id IS NULL AND EXTRACT(ISODOW FROM day)<6 ORDER BY 1`, machineID, e, from, to)
	if err != nil {
		return nil, nil, err
	}
	windows, err := scanCapacityWindows(rows)
	if err != nil {
		return nil, nil, err
	}
	rows, err = r.db.Query(ctx, `SELECT starts_at::timestamp,COALESCE(ends_at,NOW())::timestamp FROM machine_downtimes WHERE machine_id=$1 AND enterprise_id=$2 AND starts_at<$4 AND COALESCE(ends_at,NOW())>$3
 UNION ALL SELECT ps.scheduled_start::timestamp,ps.scheduled_end::timestamp FROM production_sequences ps JOIN production_orders po ON po.id=ps.production_order_id AND po.enterprise_id=ps.enterprise_id WHERE ps.machine_id=$1 AND ps.enterprise_id=$2 AND po.is_active AND po.status NOT IN ('COMPLETED','CANCELLED') AND ps.scheduled_start<$4 AND ps.scheduled_end>$3
 UNION ALL SELECT sl.starts_at,sl.ends_at FROM mrp_machine_allocation_slots sl JOIN mrp_machine_allocations a ON a.suggestion_code=sl.suggestion_code JOIN mrp_planned_suggestions s ON s.code=a.suggestion_code AND s.enterprise_id=a.enterprise_id WHERE sl.machine_id=$1 AND a.enterprise_id=$2 AND (s.plan_code<>$5 OR EXISTS(SELECT 1 FROM planned_orders po WHERE po.mrp_suggestion_code=s.code AND po.enterprise_id=$2 AND po.is_active AND po.status<>'CANCELLED')) AND sl.starts_at<$4 AND sl.ends_at>$3 AND NOT EXISTS(SELECT 1 FROM planned_orders po WHERE po.mrp_suggestion_code=s.code AND po.enterprise_id=$2 AND (NOT po.is_active OR po.status='CANCELLED')) AND NOT EXISTS(SELECT 1 FROM production_orders prod JOIN planned_orders po ON po.id=prod.planned_order_id AND po.enterprise_id=prod.enterprise_id WHERE po.mrp_suggestion_code=s.code AND prod.enterprise_id=$2 AND prod.status IN ('COMPLETED','CANCELLED'))
 UNION ALL SELECT ms.schedule_date+ms.start_time,ms.schedule_date+ms.end_time FROM machine_schedules ms JOIN machines m ON m.code=ms.machine_code AND m.enterprise_id=ms.enterprise_id WHERE m.id=$1 AND ms.enterprise_id=$2 AND ms.is_active AND ms.status IN ('SCHEDULED','IN_PROGRESS') AND ms.start_time IS NOT NULL AND ms.end_time>ms.start_time AND ms.schedule_date BETWEEN $3::date AND $4::date
 UNION ALL SELECT o.scheduled_date::timestamp,o.scheduled_date::timestamp+INTERVAL '1 day' FROM maintenance_orders o LEFT JOIN maintenance_plans p ON p.id=o.plan_id JOIN machines m ON m.id=COALESCE(o.machine_id,p.machine_id) WHERE m.id=$1 AND m.enterprise_id=$2 AND o.is_active AND o.status IN ('PLANNED','IN_PROGRESS') AND o.scheduled_date BETWEEN $3::date AND $4::date`, machineID, e, from, to, plan)
	if err != nil {
		return nil, nil, err
	}
	busy, err := scanCapacityWindows(rows)
	return windows, busy, err
}
func scanCapacityWindows(rows pgx.Rows) ([]machine.CapacityWindow, error) {
	defer rows.Close()
	out := []machine.CapacityWindow{}
	for rows.Next() {
		var w machine.CapacityWindow
		if err := rows.Scan(&w.Start, &w.End); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
func (r *MRPCalculationRepositorySQLC) SaveMachineSlots(ctx context.Context, suggestion, machineID int64, operationID *int64, slots []machine.CapacityWindow, reset bool) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if len(slots) == 0 {
		return fmt.Errorf("programação vazia")
	}
	if reset {
		if _, err = r.db.Exec(ctx, `DELETE FROM mrp_machine_allocation_slots sl USING mrp_machine_allocations a WHERE sl.suggestion_code=a.suggestion_code AND a.suggestion_code=$1 AND a.enterprise_id=$2`, suggestion, e); err != nil {
			return err
		}
	}
	for _, w := range slots {
		if _, err = r.db.Exec(ctx, `INSERT INTO mrp_machine_allocation_slots(suggestion_code,starts_at,ends_at,machine_id,route_operation_id) SELECT suggestion_code,$2,$3,$5,$6 FROM mrp_machine_allocations WHERE suggestion_code=$1 AND enterprise_id=$4`, suggestion, w.Start, w.End, e, machineID, operationID); err != nil {
			return err
		}
	}
	_, err = r.db.Exec(ctx, `UPDATE mrp_machine_allocations SET scheduled_start=(SELECT MIN(starts_at) FROM mrp_machine_allocation_slots WHERE suggestion_code=$1),scheduled_end=(SELECT MAX(ends_at) FROM mrp_machine_allocation_slots WHERE suggestion_code=$1),schedule_date=(SELECT MIN(starts_at)::date FROM mrp_machine_allocation_slots WHERE suggestion_code=$1) WHERE suggestion_code=$1 AND enterprise_id=$2`, suggestion, e)
	return err
}

func (r *MRPCalculationRepositorySQLC) SuggestionHasOrder(ctx context.Context, code int64) (bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM planned_orders WHERE mrp_suggestion_code=$1 AND enterprise_id=$2 AND is_active AND status<>'CANCELLED')`, code, e).Scan(&exists)
	return exists, err
}

func (r *MRPCalculationRepositorySQLC) MachineRoute(ctx context.Context, item int64, mask string) ([]*routingentity.RouteOperation, []*routingentity.NetworkEdge, error) {
	repo := routing.New(r.q)
	route, err := repo.GetRouteForItem(ctx, item, mask)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	ops, err := repo.GetRouteOperations(ctx, route.ID)
	if err != nil {
		return nil, nil, err
	}
	edges, err := repo.GetNetworkEdges(ctx, route.ID)
	return ops, edges, err
}

func (r *MRPCalculationRepositorySQLC) ClearUnreleasedMachineAllocations(ctx context.Context, plan int64) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `DELETE FROM mrp_machine_allocations a USING mrp_planned_suggestions s WHERE a.suggestion_code=s.code AND s.plan_code=$1 AND s.enterprise_id=$2 AND a.enterprise_id=$2 AND NOT EXISTS(SELECT 1 FROM planned_orders po WHERE po.mrp_suggestion_code=s.code AND po.enterprise_id=$2 AND po.is_active AND po.status<>'CANCELLED')`, plan, e)
	return err
}

// ApplyMachinePlanToProduction preserves the measured processing duration when
// the approved routing is exploded. Setup is already included in slot duration.
func (r *MRPCalculationRepositorySQLC) ApplyMachinePlanToProduction(ctx context.Context, orderID int64) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	beginner, ok := r.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fmt.Errorf("transação de liberação não configurada")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(348,$1::int)`, e); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE production_order_operations op SET planned_hours=v.hours,setup_hours=0,work_center_id=v.work_center_id,updated_at=NOW()
 FROM (SELECT sl.route_operation_id,mt.id work_center_id,SUM(EXTRACT(EPOCH FROM(sl.ends_at-sl.starts_at))/3600) hours
 FROM production_orders prod JOIN planned_orders po ON po.id=prod.planned_order_id AND po.enterprise_id=prod.enterprise_id
 JOIN mrp_machine_allocations a ON a.suggestion_code=po.mrp_suggestion_code AND a.enterprise_id=po.enterprise_id
 JOIN mrp_machine_allocation_slots sl ON sl.suggestion_code=a.suggestion_code
 JOIN machines m ON m.id=sl.machine_id AND m.enterprise_id=prod.enterprise_id
 JOIN machine_types mt ON mt.code=m.machine_type_code AND mt.enterprise_id=m.enterprise_id
 WHERE prod.id=$1 AND prod.enterprise_id=$2 GROUP BY sl.route_operation_id,mt.id) v
 WHERE op.production_order_id=$1 AND op.enterprise_id=$2 AND op.route_operation_id=v.route_operation_id AND op.status='PENDING'`, orderID, e)
	if err != nil {
		return err
	}
	// One sequence per productive slice preserves shift gaps; retries do not duplicate.
	_, err = tx.Exec(ctx, `INSERT INTO production_sequences(production_order_id,operation_id,work_center_id,machine_id,sequence_position,scheduled_start,scheduled_end,status,enterprise_id)
 SELECT prod.id,op.id,op.work_center_id,sl.machine_id,op.sequence,sl.starts_at,sl.ends_at,'SCHEDULED',prod.enterprise_id
 FROM production_orders prod JOIN planned_orders po ON po.id=prod.planned_order_id AND po.enterprise_id=prod.enterprise_id
 JOIN mrp_machine_allocations a ON a.suggestion_code=po.mrp_suggestion_code AND a.enterprise_id=po.enterprise_id
 JOIN mrp_machine_allocation_slots sl ON sl.suggestion_code=a.suggestion_code
 JOIN production_order_operations op ON op.production_order_id=prod.id AND op.enterprise_id=prod.enterprise_id AND op.route_operation_id=sl.route_operation_id
 WHERE prod.id=$1 AND prod.enterprise_id=$2 AND op.status='PENDING'
 AND NOT EXISTS(SELECT 1 FROM production_sequences ps WHERE ps.production_order_id=prod.id AND ps.operation_id=op.id AND ps.enterprise_id=$2 AND ps.scheduled_start=sl.starts_at AND ps.scheduled_end=sl.ends_at)`, orderID, e)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *MRPCalculationRepositorySQLC) SaveMachineCompletion(ctx context.Context, code int64, end time.Time) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `UPDATE mrp_machine_allocations SET scheduled_end=$3 WHERE suggestion_code=$1 AND enterprise_id=$2`, code, e, end)
	return err
}
