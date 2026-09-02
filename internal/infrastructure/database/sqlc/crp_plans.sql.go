package sqlc

// Catálogo de planos com CRP calculado, usado pelo modal da tela de CRP
// (VPRO0200) — sem ele o usuário precisa adivinhar o código do plano e a
// exportação sai vazia.

import "context"

// CRPPlanRow resume um plano de produção do ponto de vista do CRP.
type CRPPlanRow struct {
	PlanCode       int64
	Name           string
	TotalEntries   int64
	OverloadCount  int64
	MaxLoadPct     float64
	LastCalculated *string
}

// ListCRPPlans devolve os planos da empresa com o resumo da carga já calculada,
// dos mais recentes para os mais antigos.
func (q *Queries) ListCRPPlans(ctx context.Context, enterpriseID int64, onlyCalculated bool) ([]CRPPlanRow, error) {
	const sql = `SELECT p.code, p.name,
		COUNT(c.id) AS total_entries,
		COUNT(c.id) FILTER (WHERE c.required_hours > c.available_hours) AS overload_count,
		COALESCE(MAX(c.load_pct), 0)::float8 AS max_load_pct,
		to_char(p.last_calculated_at, 'YYYY-MM-DD"T"HH24:MI:SSOF:00') AS last_calculated_at
		FROM production_plans p
		LEFT JOIN capacity_requirements c ON c.plan_code = p.code
		WHERE p.enterprise_id = $1 AND p.is_active
		GROUP BY p.code, p.name, p.last_calculated_at
		HAVING ($2::boolean = FALSE OR COUNT(c.id) > 0)
		ORDER BY p.last_calculated_at DESC NULLS LAST, p.code DESC`
	rows, err := q.db.Query(ctx, sql, enterpriseID, onlyCalculated)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CRPPlanRow, 0)
	for rows.Next() {
		var i CRPPlanRow
		if err := rows.Scan(&i.PlanCode, &i.Name, &i.TotalEntries, &i.OverloadCount, &i.MaxLoadPct, &i.LastCalculated); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// CRPPlanBelongsToEnterprise informa se o plano pertence à empresa autenticada.
func (q *Queries) CRPPlanBelongsToEnterprise(ctx context.Context, planCode, enterpriseID int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM production_plans WHERE code = $1 AND enterprise_id = $2)`,
		planCode, enterpriseID).Scan(&exists)
	return exists, err
}
