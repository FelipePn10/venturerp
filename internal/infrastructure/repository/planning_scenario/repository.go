package planning_scenario

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
	mrpentity "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// Repository guarda cenários de simulação e mantém o plano oculto de cada um.
type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// lockCodes serializa a escolha de código novo (plano oculto, demanda
// promovida): os códigos são únicos no banco inteiro.
const lockCodes = 4211

var errNotFound = errorsuc.NewNotFoundError("cenário não encontrado nesta empresa")

// scenarioPlanBase é o início da faixa de códigos dos planos ocultos.
const scenarioPlanBase int64 = 1_000_000_000

// ScenarioAdjustments lê os ajustes de demanda, estoque e prazo do cenário dono
// do plano — o que o motor do MRP aplica em memória.
func (r *Repository) ScenarioAdjustments(ctx context.Context, planCode int64) (*mrpentity.ScenarioAdjustments, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.kind, c.item_code, c.mask, c.demand_ref_kind, c.demand_ref_code,
		       c.quantity::float8, c.need_date, c.lead_time_days
		  FROM planning_scenario_changes c
		  JOIN planning_scenarios s ON s.id = c.scenario_id
		 WHERE s.scenario_plan_code = $1 AND s.enterprise_id = $2 AND c.enterprise_id = $2
		   AND c.kind IN ('DEMANDA', 'ESTOQUE', 'LEAD_TIME')
		 ORDER BY c.id`, planCode, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &mrpentity.ScenarioAdjustments{StockDelta: map[int64]float64{}, LeadTimeDays: map[int64]int{}}
	for rows.Next() {
		var (
			id            int64
			kind, mask    string
			item, refCode *int64
			refKind       *string
			qtd           *float64
			need          *time.Time
			lead          *int
		)
		if err := rows.Scan(&id, &kind, &item, &mask, &refKind, &refCode, &qtd, &need, &lead); err != nil {
			return nil, err
		}
		switch kind {
		case planning_uc.ChangeDemand:
			c := mrpentity.ScenarioDemandChange{ID: id, RefCode: refCode, Mask: mask, Quantity: qtd, NeedDate: need}
			if refKind != nil {
				c.RefKind = *refKind
			}
			if item != nil {
				c.ItemCode = *item
			}
			out.Demands = append(out.Demands, c)
		case planning_uc.ChangeStock:
			if item != nil && qtd != nil {
				out.StockDelta[*item] += *qtd
			}
		case planning_uc.ChangeLeadTime:
			// O ajuste mais recente do item vale.
			if item != nil && lead != nil {
				out.LeadTimeDays[*item] = *lead
			}
		}
	}
	return out, rows.Err()
}

func (r *Repository) tx(ctx context.Context, work func(pgx.Tx, int64) error) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := work(tx, e); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Create cria o cenário e o plano oculto, clonado do base.
func (r *Repository) Create(ctx context.Context, name, description string, basePlan int64, actor uuid.UUID) (*planning_uc.Scenario, error) {
	var id int64
	err := r.tx(ctx, func(tx pgx.Tx, e int64) error {
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_plans WHERE code=$1 AND enterprise_id=$2 AND is_active AND NOT is_scenario)`,
			basePlan, e).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewNotFoundError("plano base não encontrado nesta empresa")
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, 0)`, lockCodes); err != nil {
			return err
		}
		var plano int64
		// Faixa reservada: o código do plano é escolhido pelo usuário e único no
		// banco; com MAX+1 o plano oculto tomava o próximo número e o usuário
		// recebia "já existe" ao cadastrar um plano que ele não enxerga.
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(code), $1 - 1) + 1 FROM production_plans WHERE code >= $1`, scenarioPlanBase).Scan(&plano); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO production_plans (code, name, independent_demands, group_same_date_orders, planning_types,
			                              classification, class_item_codes, order_item_code, parameters,
			                              created_by, enterprise_id, is_scenario)
			SELECT $1, LEFT('Cenário: ' || $2, 200), independent_demands, group_same_date_orders, planning_types,
			       classification, class_item_codes, order_item_code, parameters, $3, enterprise_id, TRUE
			  FROM production_plans WHERE code = $4 AND enterprise_id = $5`, plano, name, actor, basePlan, e); err != nil {
			return fmt.Errorf("criando o plano do cenário: %w", err)
		}
		if err := copyInterFactories(ctx, tx, e, basePlan, plano); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			INSERT INTO planning_scenarios (enterprise_id, name, description, base_plan_code, scenario_plan_code, created_by)
			VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6) RETURNING id`,
			e, name, description, basePlan, plano, actor).Scan(&id)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func copyInterFactories(ctx context.Context, tx pgx.Tx, e, from, to int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM production_plan_inter_factories WHERE plan_code=$1 AND enterprise_id=$2`, to, e); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO production_plan_inter_factories (plan_code, enterprise_id, source_enterprise_id, auto_release)
		SELECT $1, enterprise_id, source_enterprise_id, auto_release
		  FROM production_plan_inter_factories WHERE plan_code=$2 AND enterprise_id=$3`, to, from, e)
	return err
}

// SyncPlan recopia a configuração do plano base para o plano oculto: o
// cenário sempre compara com o plano base como ele está hoje.
func (r *Repository) SyncPlan(ctx context.Context, id int64) error {
	return r.tx(ctx, func(tx pgx.Tx, e int64) error {
		var base, plano int64
		if err := tx.QueryRow(ctx, `SELECT base_plan_code, scenario_plan_code FROM planning_scenarios WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`,
			id, e).Scan(&base, &plano); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errNotFound
			}
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE production_plans p
			   SET independent_demands = b.independent_demands, group_same_date_orders = b.group_same_date_orders,
			       planning_types = b.planning_types, classification = b.classification,
			       class_item_codes = b.class_item_codes, order_item_code = b.order_item_code,
			       parameters = b.parameters, updated_at = NOW()
			  FROM production_plans b
			 WHERE p.code = $1 AND p.enterprise_id = $3 AND p.is_scenario
			   AND b.code = $2 AND b.enterprise_id = $3 AND b.is_active`, plano, base, e)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errorsuc.NewValidationError("o plano base do cenário não está mais ativo")
		}
		return copyInterFactories(ctx, tx, e, base, plano)
	})
}

const scenarioColumns = `s.id, s.name, COALESCE(s.description, ''), s.base_plan_code, COALESCE(b.name, ''),
	s.scenario_plan_code, s.status, s.calculated_at, COALESCE(s.calculation_notes, ''), s.promoted_at,
	COALESCE(pu.name, ''), COALESCE(cu.name, ''), s.created_at,
	(SELECT COUNT(*) FROM planning_scenario_changes c WHERE c.scenario_id = s.id)::int`

const scenarioFrom = ` FROM planning_scenarios s
	LEFT JOIN production_plans b ON b.code = s.base_plan_code AND b.enterprise_id = s.enterprise_id
	LEFT JOIN users pu ON pu.id = s.promoted_by
	LEFT JOIN users cu ON cu.id = s.created_by`

func scanScenario(row pgx.Row) (*planning_uc.Scenario, error) {
	var s planning_uc.Scenario
	err := row.Scan(&s.ID, &s.Name, &s.Description, &s.BasePlanCode, &s.BasePlanName, &s.ScenarioPlanCode,
		&s.Status, &s.CalculatedAt, &s.CalculationNotes, &s.PromotedAt, &s.PromotedByName, &s.CreatedByName,
		&s.CreatedAt, &s.ChangesCount)
	return &s, err
}

func (r *Repository) List(ctx context.Context) ([]planning_uc.Scenario, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+scenarioColumns+scenarioFrom+` WHERE s.enterprise_id=$1 ORDER BY s.created_at DESC, s.id DESC`, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []planning_uc.Scenario{}
	for rows.Next() {
		s, err := scanScenario(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*planning_uc.Scenario, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := scanScenario(r.pool.QueryRow(ctx, `SELECT `+scenarioColumns+scenarioFrom+` WHERE s.id=$1 AND s.enterprise_id=$2`, id, e))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *Repository) UpdateMeta(ctx context.Context, id int64, name, description string) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE planning_scenarios SET name=$3, description=NULLIF($4,''), updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`,
		id, e, name, description)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errNotFound
	}
	_, err = r.pool.Exec(ctx, `UPDATE production_plans SET name=LEFT('Cenário: ' || $3, 200), updated_at=NOW()
		WHERE code=(SELECT scenario_plan_code FROM planning_scenarios WHERE id=$1 AND enterprise_id=$2) AND enterprise_id=$2 AND is_scenario`,
		id, e, name)
	return err
}

// Delete descarta o cenário com tudo o que o cálculo dele gravou.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	return r.tx(ctx, func(tx pgx.Tx, e int64) error {
		var plano int64
		if err := tx.QueryRow(ctx, `SELECT scenario_plan_code FROM planning_scenarios WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, e).Scan(&plano); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errNotFound
			}
			return err
		}
		limpeza := []string{
			`DELETE FROM mrp_pegging WHERE plan_code=$1 AND enterprise_id=$2`,
			`DELETE FROM mrp_exception_messages WHERE plan_code=$1 AND enterprise_id=$2`,
			`DELETE FROM mrp_profile_details WHERE plan_code=$1 AND enterprise_id=$2`,
			`DELETE FROM mrp_item_profiles WHERE plan_code=$1 AND enterprise_id=$2`,
			// capacity_requirements não tem empresa: o plano (código único) é da empresa, conferido acima.
			`DELETE FROM capacity_requirements WHERE plan_code=$1 AND EXISTS(SELECT 1 FROM production_plans p WHERE p.code=$1 AND p.enterprise_id=$2)`,
			`DELETE FROM mrp_machine_allocation_slots WHERE suggestion_code IN (SELECT code FROM mrp_planned_suggestions WHERE plan_code=$1 AND enterprise_id=$2)`,
			`DELETE FROM mrp_machine_allocations WHERE suggestion_code IN (SELECT code FROM mrp_planned_suggestions WHERE plan_code=$1 AND enterprise_id=$2)`,
			`DELETE FROM mrp_planned_suggestions WHERE plan_code=$1 AND enterprise_id=$2`,
			`DELETE FROM mrp_calculation_logs WHERE plan_code=$1 AND enterprise_id=$2`,
			`DELETE FROM production_plan_inter_factories WHERE plan_code=$1 AND enterprise_id=$2`,
		}
		for _, q := range limpeza {
			if _, err := tx.Exec(ctx, q, plano, e); err != nil {
				return fmt.Errorf("descartando os resultados do cenário: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM planning_scenarios WHERE id=$1 AND enterprise_id=$2`, id, e); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM production_plans WHERE code=$1 AND enterprise_id=$2 AND is_scenario`, plano, e)
		return err
	})
}

func (r *Repository) Changes(ctx context.Context, id int64) ([]planning_uc.ScenarioChange, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.kind, c.item_code, COALESCE(i.name, ''), c.mask, COALESCE(c.demand_ref_kind, ''), c.demand_ref_code,
		       CASE c.demand_ref_kind
		            WHEN 'PEDIDO_VENDA' THEN (SELECT 'Pedido ' || so.order_number FROM sales_order_items soi JOIN sales_orders so ON so.code = soi.sales_order_code WHERE soi.code = c.demand_ref_code)
		            WHEN 'DEMANDA_INDEPENDENTE' THEN 'Demanda independente ' || c.demand_ref_code
		            ELSE '' END,
		       c.quantity, c.need_date, c.lead_time_days, c.machine_id, COALESCE(m.name, ''), c.hours_per_day,
		       c.stop_from, c.stop_to, COALESCE(c.notes, ''), c.created_at
		  FROM planning_scenario_changes c
		  JOIN planning_scenarios s ON s.id = c.scenario_id AND s.enterprise_id = $2
		  LEFT JOIN items i ON i.code = c.item_code AND i.enterprise_id = $2
		  LEFT JOIN machines m ON m.id = c.machine_id AND m.enterprise_id = $2
		 WHERE c.scenario_id = $1 AND c.enterprise_id = $2
		 ORDER BY c.id`, id, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []planning_uc.ScenarioChange{}
	for rows.Next() {
		var c planning_uc.ScenarioChange
		var refLabel *string
		if err := rows.Scan(&c.ID, &c.Kind, &c.ItemCode, &c.ItemDescription, &c.Mask, &c.DemandRefKind, &c.DemandRefCode,
			&refLabel, &c.Quantity, &c.NeedDate, &c.LeadTimeDays, &c.MachineID, &c.MachineName, &c.HoursPerDay,
			&c.StopFrom, &c.StopTo, &c.Notes, &c.CreatedAt); err != nil {
			return nil, err
		}
		if refLabel != nil {
			c.DemandRefLabel = *refLabel
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddChange grava o ajuste e volta o cenário para rascunho: os resultados
// calculados deixam de refletir os ajustes.
func (r *Repository) AddChange(ctx context.Context, id int64, c planning_uc.ScenarioChange, actor uuid.UUID) (*planning_uc.ScenarioChange, error) {
	var changeID int64
	err := r.tx(ctx, func(tx pgx.Tx, e int64) error {
		var demandRef *string
		if c.DemandRefKind != "" {
			demandRef = &c.DemandRefKind
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO planning_scenario_changes (enterprise_id, scenario_id, kind, item_code, mask, demand_ref_kind,
			       demand_ref_code, quantity, need_date, lead_time_days, machine_id, hours_per_day, stop_from, stop_to,
			       notes, created_by)
			SELECT $1, s.id, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NULLIF($15, ''), $16
			  FROM planning_scenarios s WHERE s.id = $2 AND s.enterprise_id = $1
			RETURNING id`,
			e, id, c.Kind, c.ItemCode, c.Mask, demandRef, c.DemandRefCode, c.Quantity, c.NeedDate, c.LeadTimeDays,
			c.MachineID, c.HoursPerDay, c.StopFrom, c.StopTo, c.Notes, actor).Scan(&changeID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errNotFound
			}
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE planning_scenarios SET status='RASCUNHO', updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, e)
		return err
	})
	if err != nil {
		return nil, err
	}
	changes, err := r.Changes(ctx, id)
	if err != nil {
		return nil, err
	}
	for i := range changes {
		if changes[i].ID == changeID {
			return &changes[i], nil
		}
	}
	return nil, errNotFound
}

func (r *Repository) DeleteChange(ctx context.Context, id, changeID int64) error {
	return r.tx(ctx, func(tx pgx.Tx, e int64) error {
		tag, err := tx.Exec(ctx, `DELETE FROM planning_scenario_changes WHERE id=$1 AND scenario_id=$2 AND enterprise_id=$3`, changeID, id, e)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errorsuc.NewNotFoundError("ajuste não encontrado neste cenário")
		}
		_, err = tx.Exec(ctx, `UPDATE planning_scenarios SET status='RASCUNHO', updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, e)
		return err
	})
}

func (r *Repository) MarkCalculated(ctx context.Context, id int64, notes string) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE planning_scenarios SET status='CALCULADO', calculated_at=NOW(), calculation_notes=NULLIF($3,''), updated_at=NOW()
		WHERE id=$1 AND enterprise_id=$2`, id, e, notes)
	return err
}

// BaseDemands lista as demandas de origem que o último cálculo do plano base
// considerou (demanda independente e pedido de venda), somadas por origem.
func (r *Repository) BaseDemands(ctx context.Context, basePlan int64) ([]planning_uc.BaseDemand, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT CASE d.detail_type WHEN 'SALES_ORDER' THEN 'PEDIDO_VENDA' ELSE 'DEMANDA_INDEPENDENTE' END,
		       d.source_code, d.item_code, COALESCE(i.name, ''), MIN(d.need_date), SUM(d.quantity),
		       CASE d.detail_type
		            WHEN 'SALES_ORDER' THEN COALESCE((SELECT 'Pedido ' || so.order_number || COALESCE(' · ' || COALESCE(c.trade_name, c.name), '')
		                                                FROM sales_order_items soi JOIN sales_orders so ON so.code = soi.sales_order_code
		                                                JOIN enterprise en ON en.code = so.enterprise_code
		                                                LEFT JOIN customers c ON c.code = so.customer_code AND c.enterprise_id = en.id
		                                               WHERE soi.code = d.source_code), 'Pedido de venda')
		            ELSE 'Demanda independente ' || d.source_code END
		  FROM mrp_profile_details d
		  LEFT JOIN items i ON i.code = d.item_code AND i.enterprise_id = $2
		 WHERE d.plan_code = $1 AND d.enterprise_id = $2 AND d.source_code IS NOT NULL
		   AND d.detail_type IN ('SALES_ORDER', 'INDEPENDENT_DEMAND')
		 GROUP BY d.detail_type, d.source_code, d.item_code, i.name
		 ORDER BY MIN(d.need_date), d.item_code`, basePlan, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []planning_uc.BaseDemand{}
	for rows.Next() {
		var d planning_uc.BaseDemand
		if err := rows.Scan(&d.RefKind, &d.RefCode, &d.ItemCode, &d.ItemDescription, &d.NeedDate, &d.Quantity, &d.Label); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) exists(ctx context.Context, q string, args ...any) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, q, args...).Scan(&ok)
	return ok, err
}

func (r *Repository) ItemExists(ctx context.Context, item int64) (bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	return r.exists(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2)`, item, e)
}

func (r *Repository) MachineExists(ctx context.Context, machine int64) (bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	return r.exists(ctx, `SELECT EXISTS(SELECT 1 FROM machines WHERE id=$1 AND enterprise_id=$2)`, machine, e)
}

func (r *Repository) DemandRefExists(ctx context.Context, kind string, code int64) (bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	switch kind {
	case planning_uc.DemandRefIndependent:
		return r.exists(ctx, `SELECT EXISTS(SELECT 1 FROM independent_demands WHERE code=$1 AND enterprise_id=$2 AND is_active)`, code, e)
	case planning_uc.DemandRefSalesOrder:
		return r.exists(ctx, `SELECT EXISTS(SELECT 1 FROM sales_order_items soi JOIN sales_orders so ON so.code=soi.sales_order_code
			JOIN enterprise en ON en.code=so.enterprise_code WHERE soi.code=$1 AND en.id=$2)`, code, e)
	}
	return false, nil
}

// Promote aplica ao plano real só os ajustes de demanda: demanda nova vira
// demanda independente; ajuste de demanda independente altera quantidade e
// data (quantidade 0 desativa). Pedido de venda é do cliente e não é alterado.
func (r *Repository) Promote(ctx context.Context, id int64, actor uuid.UUID) (*planning_uc.PromoteResult, error) {
	out := &planning_uc.PromoteResult{}
	err := r.tx(ctx, func(tx pgx.Tx, e int64) error {
		var promovido *time.Time
		if err := tx.QueryRow(ctx, `SELECT promoted_at FROM planning_scenarios WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, e).Scan(&promovido); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errNotFound
			}
			return err
		}
		if promovido != nil {
			return errorsuc.NewConflictError("este cenário já foi aplicado ao plano real")
		}
		rows, err := tx.Query(ctx, `SELECT demand_ref_kind, demand_ref_code, item_code, mask, quantity, need_date
			FROM planning_scenario_changes WHERE scenario_id=$1 AND enterprise_id=$2 AND kind='DEMANDA' ORDER BY id`, id, e)
		if err != nil {
			return err
		}
		type ajuste struct {
			kind string
			code *int64
			item *int64
			mask string
			qtd  *decimal.Decimal
			need *time.Time
		}
		var ajustes []ajuste
		for rows.Next() {
			var a ajuste
			if err := rows.Scan(&a.kind, &a.code, &a.item, &a.mask, &a.qtd, &a.need); err != nil {
				rows.Close()
				return err
			}
			ajustes = append(ajustes, a)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(ajustes) == 0 {
			return errorsuc.NewValidationError("o cenário não tem ajuste de demanda para aplicar")
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, 0)`, lockCodes); err != nil {
			return err
		}
		for _, a := range ajustes {
			switch a.kind {
			case planning_uc.DemandRefNew:
				var code int64
				if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(code), 0) + 1 FROM independent_demands`).Scan(&code); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `INSERT INTO independent_demands (code, item_code, mask, quantity, demand_date, created_by, enterprise_id)
					VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7)`, code, a.item, a.mask, a.qtd, a.need, actor, e); err != nil {
					return fmt.Errorf("criando a demanda independente: %w", err)
				}
				out.Created++
			case planning_uc.DemandRefIndependent:
				if a.qtd != nil && a.qtd.IsZero() {
					tag, err := tx.Exec(ctx, `UPDATE independent_demands SET is_active=FALSE, updated_at=NOW() WHERE code=$1 AND enterprise_id=$2 AND is_active`, a.code, e)
					if err != nil {
						return err
					}
					out.Removed += int(tag.RowsAffected())
					continue
				}
				tag, err := tx.Exec(ctx, `UPDATE independent_demands SET quantity=COALESCE($3, quantity), demand_date=COALESCE($4, demand_date), updated_at=NOW()
					WHERE code=$1 AND enterprise_id=$2 AND is_active`, a.code, e, a.qtd, a.need)
				if err != nil {
					return err
				}
				out.Updated += int(tag.RowsAffected())
			default:
				out.Skipped++
			}
		}
		_, err = tx.Exec(ctx, `UPDATE planning_scenarios SET promoted_at=NOW(), promoted_by=$3, updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, e, actor)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) WorkCenterNames(ctx context.Context, ids []int64) (map[int64]string, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id, name FROM machine_types WHERE id = ANY($1::bigint[]) AND enterprise_id = $2`, ids, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

// LastCalculatedAt é quando o plano foi calculado pela última vez.
func (r *Repository) LastCalculatedAt(ctx context.Context, planCode int64) (*time.Time, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var at *time.Time
	err = r.pool.QueryRow(ctx, `SELECT last_calculated_at FROM production_plans WHERE code=$1 AND enterprise_id=$2`, planCode, e).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return at, err
}
