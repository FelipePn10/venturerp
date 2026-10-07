// Package journal grava e estorna lançamentos contábeis automáticos dentro da
// transação de quem chama — o lançamento nasce e morre com o fato (baixa,
// autorização, cancelamento), nunca separado dele.
package journal

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
)

// Executor é o que uma transação (ou o pool) oferece.
type Executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func numero(prefixo string, id int64, n int) string {
	s := fmt.Sprintf("%s%d-%d", prefixo, id, n)
	if len(s) > 20 {
		s = s[len(s)-20:]
	}
	return s
}

// Gravar insere os lançamentos do lote.
func Gravar(ctx context.Context, q Executor, empresa int64, l contabilizacao.Lote) error {
	for i, lc := range l.Lancamentos {
		if !lc.Valor.IsPositive() {
			continue
		}
		hist := lc.Historico
		if len(hist) > 500 {
			hist = hist[:500]
		}
		if _, err := q.Exec(ctx,
			`INSERT INTO accounting_journal_entries
				(plan_id, empresa_id, entry_date, entry_number, debit_account_id, credit_account_id, debit_cc_id, credit_cc_id,
				 value, description, entry_type, source_type, source_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'AUTOMATICO',$11,$12)`,
			l.PlanID, empresa, l.Data, numero(l.Prefixo, l.SourceID, i+1), lc.DebitoID, lc.CreditoID, lc.DebitoCC, lc.CreditoCC,
			lc.Valor.Round(2), hist, l.SourceType, l.SourceID); err != nil {
			return fmt.Errorf("gravando lançamento contábil %s: %w", l.SourceType, err)
		}
	}
	return nil
}

// Estornar lança a inversão de tudo o que a origem lançou (e ainda não foi
// estornado), com a origem de estorno. Devolve quantos lançamentos inverteu.
func Estornar(ctx context.Context, q Executor, empresa int64, origem string, id int64, origemEstorno, prefixo string, data time.Time) (int, error) {
	var ja int
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM accounting_journal_entries WHERE empresa_id=$1 AND source_type=$2 AND source_id=$3`,
		empresa, origemEstorno, id).Scan(&ja); err != nil {
		return 0, err
	}
	if ja > 0 {
		return 0, nil // já estornado
	}
	tag, err := q.Exec(ctx,
		`INSERT INTO accounting_journal_entries
			(plan_id, empresa_id, entry_date, entry_number, batch_number, debit_account_id, credit_account_id, debit_cc_id, credit_cc_id,
			 value, history_code, description, entry_type, source_type, source_id)
		 SELECT plan_id, empresa_id, $5::date, right($6 || $7 || '-' || ROW_NUMBER() OVER (ORDER BY id), 20), batch_number,
		        credit_account_id, debit_account_id, credit_cc_id, debit_cc_id,
		        value, history_code, left('Estorno — ' || description, 500), 'ESTORNO', $4, source_id
		   FROM accounting_journal_entries
		  WHERE empresa_id=$1 AND source_type=$2 AND source_id=$3`,
		empresa, origem, id, origemEstorno, data, prefixo, fmt.Sprint(id))
	if err != nil {
		return 0, fmt.Errorf("estornando lançamentos de %s %d: %w", origem, id, err)
	}
	return int(tag.RowsAffected()), nil
}

// Existe diz se a origem já tem lançamentos (ex.: a nota foi contabilizada,
// então o título dela está em Fornecedores/Clientes).
func Existe(ctx context.Context, q Executor, empresa int64, origem string, id int64) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM accounting_journal_entries WHERE empresa_id=$1 AND source_type=$2 AND source_id=$3)`,
		empresa, origem, id).Scan(&ok)
	return ok, err
}
