package routing

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Adaptadores do checklist de prontidão do roteiro.
//
// As duas perguntas que ele faz não pertencem a nenhum repositório existente:
// "este centro tem tarifa por hora?" mora no custo padrão e "a que centro esta
// máquina pertence?" é a tradução máquina → tipo de máquina, que é o que o
// roteiro chama de centro de trabalho. São consultas de leitura, pequenas e de
// um cliente só.

// ReadinessAdapters responde às duas perguntas do checklist.
type ReadinessAdapters struct{ pool *pgxpool.Pool }

func NewReadinessAdapters(pool *pgxpool.Pool) *ReadinessAdapters {
	return &ReadinessAdapters{pool: pool}
}

// HourlyRate devolve a tarifa por hora do centro e se ela existe. Tarifa
// ausente não é zero: zero é uma tarifa cadastrada como zero, e o checklist
// precisa distinguir "de graça" de "ninguém cadastrou".
func (a *ReadinessAdapters) HourlyRate(ctx context.Context, workCenterID int64) (float64, bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, false, err
	}
	var taxa float64
	err = a.pool.QueryRow(ctx,
		`SELECT cost_per_hour FROM work_center_costs
		 WHERE work_center_id = $1 AND enterprise_id = $2`, workCenterID, empresa).Scan(&taxa)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return taxa, true, nil
}

// InspectionStepsWithoutPlan devolve as sequências marcadas como ponto de
// inspeção que não têm plano ativo.
//
// A ordem de fabricação não é mais recusada por isso — ela nasce com a etapa
// marcada como pendência. Mas a hora barata de descobrir é aqui, montando o
// roteiro, e não com a ordem já solta no chão de fábrica.
func (a *ReadinessAdapters) InspectionStepsWithoutPlan(ctx context.Context, routeID int64) ([]int16, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := a.pool.Query(ctx,
		`SELECT ro.sequence
		 FROM route_operations ro
		 JOIN manufacturing_routes mr ON mr.id = ro.route_id
		 LEFT JOIN inspection_plans ip ON ip.route_operation_id = ro.id AND ip.is_active = TRUE
		 WHERE ro.route_id = $1 AND ro.is_active AND ro.inspection_required
		   AND mr.enterprise_id = $2 AND ip.id IS NULL
		 ORDER BY ro.sequence`, routeID, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int16
	for rows.Next() {
		var seq int16
		if err := rows.Scan(&seq); err != nil {
			return nil, err
		}
		out = append(out, seq)
	}
	return out, rows.Err()
}

// WorkCenterOfMachine traduz o código da máquina no id do tipo dela — que é o
// que o roteiro guarda como centro de trabalho.
func (a *ReadinessAdapters) WorkCenterOfMachine(ctx context.Context, machineCode int64) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var centro int64
	err = a.pool.QueryRow(ctx,
		`SELECT mt.id FROM machines m
		 JOIN machine_types mt ON mt.code = m.machine_type_code AND mt.enterprise_id = m.enterprise_id
		 WHERE m.code = $1 AND m.enterprise_id = $2 AND m.is_active AND mt.is_active`,
		machineCode, empresa).Scan(&centro)
	if err != nil {
		return 0, err
	}
	return centro, nil
}
