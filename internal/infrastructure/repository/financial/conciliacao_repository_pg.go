package financial

import (
	"context"
	"errors"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// Conciliação bancária: o extrato importado (OFX) contra o fluxo de caixa.
//
// Posse: `extrato_bancario` não tem coluna de empresa — herda a da conta bancária
// (`contas_bancarias.enterprise_id`). `fluxo_caixa` tem `enterprise_id` próprio.
// Toda consulta abaixo passa por uma das duas, e as escritas travam as linhas
// (FOR UPDATE) para que duas pessoas conciliando ao mesmo tempo não casem o mesmo
// movimento com dois lançamentos do extrato.

// descricaoMaxima é o tamanho de `fluxo_caixa.descricao` (varchar 200).
const descricaoMaxima = 200

func (r *FinancialRepositoryPG) ListExtrato(ctx context.Context, contaID int64, filtro repository.ExtratoFiltro) ([]*repository.ExtratoLinha, error) {
	if err := r.contaDaEmpresa(ctx, contaID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT e.id, e.conta_bancaria_id, e.data_transacao, e.valor, e.tipo,
		        COALESCE(e.descricao, ''), COALESCE(e.fitid, ''), e.extrato_hash, e.conciliado, e.fluxo_caixa_id,
		        f.data, f.descricao, f.valor, f.contas_pagar_id, f.contas_receber_id,
		        COALESCE(f.extrato_hash = e.extrato_hash AND f.contas_pagar_id IS NULL AND f.contas_receber_id IS NULL, false)
		   FROM extrato_bancario e
		   LEFT JOIN fluxo_caixa f ON f.id = e.fluxo_caixa_id
		  WHERE e.conta_bancaria_id = $1
		    AND ($2 = '' OR ($2 = 'PENDENTE' AND NOT e.conciliado) OR ($2 = 'CONCILIADO' AND e.conciliado))
		    AND ($3::date IS NULL OR e.data_transacao >= $3::date)
		    AND ($4::date IS NULL OR e.data_transacao <= $4::date)
		  ORDER BY e.data_transacao, e.id`,
		contaID, string(filtro.Situacao), filtro.Inicio, filtro.Fim)
	if err != nil {
		return nil, fmt.Errorf("lendo o extrato da conta %d: %w", contaID, err)
	}
	defer rows.Close()

	out := []*repository.ExtratoLinha{}
	for rows.Next() {
		var l repository.ExtratoLinha
		var tipo string
		if err := rows.Scan(&l.ID, &l.ContaBancariaID, &l.Data, &l.Valor, &tipo,
			&l.Descricao, &l.FitID, &l.Hash, &l.Conciliado, &l.FluxoCaixaID,
			&l.MovimentoData, &l.MovimentoDescricao, &l.MovimentoValor, &l.ContasPagarID, &l.ContasReceberID,
			&l.LancadoPeloExtrato); err != nil {
			return nil, fmt.Errorf("lendo linha do extrato: %w", err)
		}
		l.Tipo = entity.ExtratoTipo(tipo)
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) CandidatosConciliacao(ctx context.Context, extratoID int64) ([]*repository.CandidatoConciliacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	e, err := lerExtrato(ctx, r.pool, extratoID, empresa, false)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT f.id, f.data, f.valor, f.tipo, f.descricao, f.contas_pagar_id, f.contas_receber_id,
		        COALESCE(cp.numero_documento, cr.numero_documento),
		        ABS(f.data - $5::date)
		   FROM fluxo_caixa f
		   LEFT JOIN contas_pagar cp ON cp.id = f.contas_pagar_id
		   LEFT JOIN contas_receber cr ON cr.id = f.contas_receber_id
		  WHERE f.conta_bancaria_id = $1 AND f.enterprise_id = $2 AND f.tipo = $3
		    AND f.conciliado = false AND ABS(f.valor - $4) < 0.005
		    AND NOT EXISTS (SELECT 1 FROM extrato_bancario x WHERE x.fluxo_caixa_id = f.id)
		  ORDER BY ABS(f.data - $5::date), f.id
		  LIMIT 50`,
		e.ContaBancariaID, empresa, string(e.TipoNoCaixa()), e.Valor, e.Data)
	if err != nil {
		return nil, fmt.Errorf("procurando movimentos para o extrato %d: %w", extratoID, err)
	}
	defer rows.Close()

	out := []*repository.CandidatoConciliacao{}
	for rows.Next() {
		var c repository.CandidatoConciliacao
		var tipo string
		if err := rows.Scan(&c.FluxoCaixaID, &c.Data, &c.Valor, &tipo, &c.Descricao,
			&c.ContasPagarID, &c.ContasReceberID, &c.Documento, &c.DiferencaDias); err != nil {
			return nil, fmt.Errorf("lendo movimento candidato: %w", err)
		}
		c.Tipo = entity.FluxoCaixaTipo(tipo)
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *FinancialRepositoryPG) ConciliarExtrato(ctx context.Context, extratoID, fluxoID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	e, err := lerExtrato(ctx, tx, extratoID, empresa, true)
	if err != nil {
		return err
	}
	f, err := lerFluxo(ctx, tx, fluxoID, empresa)
	if err != nil {
		return err
	}
	if motivo := e.MotivoParaNaoConciliar(*f); motivo != "" {
		return errorsuc.NewValidationError(motivo)
	}
	// `fluxo_caixa.conciliado` pode ter sido marcado à parte; o vínculo é o que vale.
	var jaVinculado bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM extrato_bancario WHERE fluxo_caixa_id = $1 AND id <> $2)`,
		fluxoID, extratoID).Scan(&jaVinculado); err != nil {
		return fmt.Errorf("conferindo o vínculo do movimento %d: %w", fluxoID, err)
	}
	if jaVinculado {
		return errorsuc.NewValidationError("o movimento escolhido já está conciliado com outro lançamento do extrato")
	}

	if _, err := tx.Exec(ctx,
		`UPDATE extrato_bancario SET conciliado = true, fluxo_caixa_id = $1 WHERE id = $2`,
		fluxoID, extratoID); err != nil {
		return fmt.Errorf("conciliando o extrato %d: %w", extratoID, err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE fluxo_caixa SET conciliado = true WHERE id = $1 AND enterprise_id = $2`,
		fluxoID, empresa); err != nil {
		return fmt.Errorf("marcando o movimento %d como conciliado: %w", fluxoID, err)
	}
	return tx.Commit(ctx)
}

func (r *FinancialRepositoryPG) LancarExtratoNoCaixa(ctx context.Context, extratoID int64, descricao string) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	e, err := lerExtrato(ctx, tx, extratoID, empresa, true)
	if err != nil {
		return 0, err
	}
	if motivo := e.MotivoParaNaoLancar(); motivo != "" {
		return 0, errorsuc.NewValidationError(motivo)
	}
	if descricao == "" {
		descricao = e.Descricao
	}
	if descricao == "" {
		descricao = "Lançamento do extrato bancário"
	}
	descricao = cortar(descricao, descricaoMaxima)

	// O hash da linha no movimento é a marca de "nasceu do extrato": é por ela que
	// o desfazer sabe que deve apagar o movimento em vez de só soltá-lo.
	var fluxoID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO fluxo_caixa (data, tipo, valor, conta_bancaria_id, descricao, conciliado, extrato_hash, enterprise_id)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $7)
		 RETURNING id`,
		e.Data, string(e.TipoNoCaixa()), e.Valor, e.ContaBancariaID, descricao, e.Hash, empresa,
	).Scan(&fluxoID); err != nil {
		return 0, fmt.Errorf("lançando o extrato %d no caixa: %w", extratoID, err)
	}

	// Mesma convenção da baixa e do adiantamento: `saldo_inicial` é o saldo
	// corrente da conta, e todo movimento que entra no caixa o atualiza.
	if err := ajustarSaldo(ctx, tx, e.ContaBancariaID, empresa, e.TipoNoCaixa(), e.Valor); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE extrato_bancario SET conciliado = true, fluxo_caixa_id = $1 WHERE id = $2`,
		fluxoID, extratoID); err != nil {
		return 0, fmt.Errorf("conciliando o extrato %d: %w", extratoID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return fluxoID, nil
}

func (r *FinancialRepositoryPG) DesfazerConciliacao(ctx context.Context, extratoID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	e, err := lerExtrato(ctx, tx, extratoID, empresa, true)
	if err != nil {
		return err
	}
	if !e.Conciliado {
		return errorsuc.NewValidationError("este lançamento do extrato não está conciliado")
	}
	if _, err := tx.Exec(ctx,
		`UPDATE extrato_bancario SET conciliado = false, fluxo_caixa_id = NULL WHERE id = $1`, extratoID); err != nil {
		return fmt.Errorf("desfazendo a conciliação do extrato %d: %w", extratoID, err)
	}

	if e.FluxoCaixaID != nil {
		f, err := lerFluxo(ctx, tx, *e.FluxoCaixaID, empresa)
		if err != nil {
			return err
		}
		if e.MovimentoLancadoPeloExtrato(*f) {
			// O movimento só existia por causa da linha: sai do caixa e devolve o saldo.
			if _, err := tx.Exec(ctx,
				`DELETE FROM fluxo_caixa WHERE id = $1 AND enterprise_id = $2`, f.ID, empresa); err != nil {
				return fmt.Errorf("removendo o movimento %d: %w", f.ID, err)
			}
			if err := ajustarSaldo(ctx, tx, e.ContaBancariaID, empresa, inverso(f.Tipo), f.Valor); err != nil {
				return err
			}
		} else if _, err := tx.Exec(ctx,
			`UPDATE fluxo_caixa SET conciliado = false WHERE id = $1 AND enterprise_id = $2`, f.ID, empresa); err != nil {
			return fmt.Errorf("desmarcando o movimento %d: %w", f.ID, err)
		}
	}
	return tx.Commit(ctx)
}

// AutoMatchExtrato casa cada linha pendente com o movimento de mesmo valor e
// sentido mais próximo da data (até 3 dias). Uma linha que outra pessoa conciliou
// no meio do caminho é pulada, não interrompe as demais.
func (r *FinancialRepositoryPG) AutoMatchExtrato(ctx context.Context, contaID int64) (int, error) {
	if err := r.contaDaEmpresa(ctx, contaID); err != nil {
		return 0, err
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, data_transacao, valor, tipo FROM extrato_bancario
		  WHERE conta_bancaria_id = $1 AND conciliado = false
		  ORDER BY data_transacao, id`, contaID)
	if err != nil {
		return 0, fmt.Errorf("lendo o extrato para a conciliação automática: %w", err)
	}
	var pendentes []entity.ExtratoBancario
	for rows.Next() {
		var e entity.ExtratoBancario
		var tipo string
		if err := rows.Scan(&e.ID, &e.Data, &e.Valor, &tipo); err != nil {
			rows.Close()
			return 0, err
		}
		e.Tipo = entity.ExtratoTipo(tipo)
		pendentes = append(pendentes, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	matched := 0
	for _, e := range pendentes {
		var fluxoID int64
		err := r.pool.QueryRow(ctx,
			`SELECT f.id FROM fluxo_caixa f
			  WHERE f.conta_bancaria_id = $1 AND f.tipo = $2 AND f.enterprise_id = $3
			    AND f.conciliado = false AND ABS(f.valor - $4) < 0.005
			    AND f.data BETWEEN $5::date - 3 AND $5::date + 3
			    AND NOT EXISTS (SELECT 1 FROM extrato_bancario x WHERE x.fluxo_caixa_id = f.id)
			  ORDER BY ABS(f.data - $5::date), f.id
			  LIMIT 1`,
			contaID, string(e.TipoNoCaixa()), empresa, e.Valor, e.Data,
		).Scan(&fluxoID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue // sem movimento correspondente: fica para conciliar à mão
		}
		if err != nil {
			return matched, fmt.Errorf("procurando movimento para o extrato %d: %w", e.ID, err)
		}
		if err := r.ConciliarExtrato(ctx, e.ID, fluxoID); err != nil {
			if _, recusado := errorsuc.AsValidation(err); recusado {
				continue
			}
			return matched, fmt.Errorf("conciliando o extrato %d: %w", e.ID, err)
		}
		matched++
	}
	return matched, nil
}

// consultor é o que lerExtrato/lerFluxo precisam: o pool ou uma transação.
type consultor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// lerExtrato lê a linha do extrato se a conta dela for da empresa. travar=true
// trava a linha até o fim da transação.
func lerExtrato(ctx context.Context, q consultor, extratoID, empresa int64, travar bool) (*entity.ExtratoBancario, error) {
	sql := `SELECT e.id, e.conta_bancaria_id, e.data_transacao, e.valor, e.tipo,
	               COALESCE(e.descricao, ''), COALESCE(e.fitid, ''), e.extrato_hash, e.conciliado, e.fluxo_caixa_id
	          FROM extrato_bancario e
	          JOIN contas_bancarias cb ON cb.id = e.conta_bancaria_id AND cb.enterprise_id = $2
	         WHERE e.id = $1`
	if travar {
		sql += ` FOR UPDATE OF e`
	}
	var e entity.ExtratoBancario
	var tipo string
	err := q.QueryRow(ctx, sql, extratoID, empresa).Scan(&e.ID, &e.ContaBancariaID, &e.Data, &e.Valor, &tipo,
		&e.Descricao, &e.FitID, &e.Hash, &e.Conciliado, &e.FluxoCaixaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("lançamento %d do extrato não encontrado", extratoID))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o lançamento %d do extrato: %w", extratoID, err)
	}
	e.Tipo = entity.ExtratoTipo(tipo)
	return &e, nil
}

// lerFluxo lê e trava o movimento do caixa da empresa.
func lerFluxo(ctx context.Context, q consultor, fluxoID, empresa int64) (*entity.FluxoCaixa, error) {
	var f entity.FluxoCaixa
	var tipo string
	err := q.QueryRow(ctx,
		`SELECT id, data, tipo, valor, conta_bancaria_id, contas_pagar_id, contas_receber_id,
		        descricao, conciliado, extrato_hash
		   FROM fluxo_caixa WHERE id = $1 AND enterprise_id = $2
		    FOR UPDATE`, fluxoID, empresa,
	).Scan(&f.ID, &f.Data, &tipo, &f.Valor, &f.ContaBancariaID, &f.ContasPagarID, &f.ContasReceberID,
		&f.Descricao, &f.Conciliado, &f.ExtratoHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("movimento %d do caixa não encontrado", fluxoID))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o movimento %d do caixa: %w", fluxoID, err)
	}
	f.Tipo = entity.FluxoCaixaTipo(tipo)
	return &f, nil
}

// ajustarSaldo soma a entrada (ou subtrai a saída) do saldo corrente da conta.
func ajustarSaldo(ctx context.Context, tx pgx.Tx, contaID, empresa int64, tipo entity.FluxoCaixaTipo, valor decimal.Decimal) error {
	if tipo == entity.FluxoCaixaTipoSaida {
		valor = valor.Neg()
	}
	tag, err := tx.Exec(ctx,
		`UPDATE contas_bancarias SET saldo_inicial = saldo_inicial + $1, updated_at = NOW()
		  WHERE id = $2 AND enterprise_id = $3`, valor, contaID, empresa)
	if err != nil {
		return fmt.Errorf("atualizando o saldo da conta %d: %w", contaID, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("conta bancária %d não encontrada", contaID))
	}
	return nil
}

func inverso(t entity.FluxoCaixaTipo) entity.FluxoCaixaTipo {
	if t == entity.FluxoCaixaTipoSaida {
		return entity.FluxoCaixaTipoEntrada
	}
	return entity.FluxoCaixaTipoSaida
}

// cortar limita o texto a n caracteres (não bytes: o memo do banco tem acento).
func cortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
