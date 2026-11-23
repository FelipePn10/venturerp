package financial

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ContasComMovimento: o plano de contas (o dos parâmetros contábeis, ou o
// informado) com débitos e créditos dos lançamentos do período, por conta.
func (r *FinancialRepositoryPG) ContasComMovimento(ctx context.Context, plano *int64, de, ate time.Time) ([]entity.ContaDoResultado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var planoID int64
	if plano != nil {
		planoID = *plano
	} else if err := r.pool.QueryRow(ctx, `SELECT plan_id FROM accounting_posting_params WHERE enterprise_id = $1 AND plan_id IS NOT NULL`, empresa).Scan(&planoID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewValidationError("informe o plano de contas (ou configure-o nos parâmetros contábeis, VCTB0200)")
		}
		return nil, fmt.Errorf("lendo o plano dos parâmetros contábeis: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		WITH mov AS (
		    SELECT j.debit_account_id conta, j.value d, 0::numeric c FROM accounting_journal_entries j
		     WHERE j.empresa_id = $1 AND j.plan_id = $2 AND j.entry_date BETWEEN $3::date AND $4::date
		    UNION ALL
		    SELECT j.credit_account_id, 0, j.value FROM accounting_journal_entries j
		     WHERE j.empresa_id = $1 AND j.plan_id = $2 AND j.entry_date BETWEEN $3::date AND $4::date)
		SELECT a.id, COALESCE(a.parent_id, 0), a.account_number, COALESCE(a.description,''), a.is_analytic,
		       COALESCE(SUM(m.d), 0), COALESCE(SUM(m.c), 0)
		  FROM accounting_accounts a
		  JOIN accounting_plans p ON p.id = a.plan_id AND p.enterprise_id = $1
		  LEFT JOIN mov m ON m.conta = a.id
		 WHERE a.plan_id = $2
		 GROUP BY a.id, a.parent_id, a.account_number, a.description, a.is_analytic
		 ORDER BY a.account_number`, empresa, planoID, de.Format("2006-01-02"), ate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("lendo o movimento do plano de contas: %w", err)
	}
	defer rows.Close()
	var out []entity.ContaDoResultado
	for rows.Next() {
		var c entity.ContaDoResultado
		if err := rows.Scan(&c.ID, &c.ParentID, &c.Numero, &c.Nome, &c.Analitica, &c.Debitos, &c.Creditos); err != nil {
			return nil, fmt.Errorf("lendo o movimento do plano de contas: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
