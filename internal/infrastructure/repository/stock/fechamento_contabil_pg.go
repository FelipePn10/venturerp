package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/journal"
)

// contasDoFechamento são as contas dos lançamentos do fechamento de estoque.
type contasDoFechamento struct {
	plano, cmv, estoque, processo, absorvidos, provisao, despesaProvisao *int64
	producao                                                             bool
}

func lerContasDoFechamento(ctx context.Context, tx pgx.Tx, empresa int64) (contasDoFechamento, error) {
	var c contasDoFechamento
	err := tx.QueryRow(ctx,
		`SELECT plan_id, cmv_account_id, estoque_account_id, producao_processo_account_id, custos_absorvidos_account_id,
		        provisao_obsoletos_account_id, despesa_provisao_account_id, contabilizar_producao
		   FROM accounting_posting_params WHERE enterprise_id=$1`, empresa).
		Scan(&c.plano, &c.cmv, &c.estoque, &c.processo, &c.absorvidos, &c.provisao, &c.despesaProvisao, &c.producao)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return c, fmt.Errorf("lendo as contas do fechamento: %w", err)
	}
	return c, nil
}

// contabilizarFechamento leva ao resultado a diferença de custo do mês, a produção
// e a variação da provisão de obsoletos, num lote só: a reabertura estorna tudo.
func contabilizarFechamento(ctx context.Context, tx pgx.Tx, empresa, closing int64, janela, fim time.Time, liquida decimal.Decimal, simular bool, res *entity.ResultadoFechamento) error {
	contas, err := lerContasDoFechamento(ctx, tx, empresa)
	if err != nil {
		return err
	}
	competencia := fim.Format("01/2006")
	var ls []contabilizacao.Lancamento

	if !liquida.IsZero() {
		if contas.plano == nil || contas.cmv == nil || contas.estoque == nil {
			res.Avisos = append(res.Avisos, "A diferença de custo não foi contabilizada: configure as contas de CMV e Estoque nos parâmetros contábeis.")
		} else {
			debito, credito := *contas.cmv, *contas.estoque
			if liquida.IsNegative() {
				debito, credito = credito, debito
			}
			ls = append(ls, contabilizacao.Lancamento{DebitoID: debito, CreditoID: credito, Valor: liquida.Abs(),
				Historico: "Diferença de custo do fechamento de estoque " + competencia})
		}
	}

	if contas.producao {
		producao, err := contabilizarProducao(ctx, tx, empresa, closing, janela, fim, contas, simular, res)
		if err != nil {
			return err
		}
		ls = append(ls, producao...)
	}

	provisao, err := contabilizarProvisao(ctx, tx, empresa, closing, fim, contas, res)
	if err != nil {
		return err
	}
	ls = append(ls, provisao...)

	if len(ls) == 0 || contas.plano == nil {
		return nil
	}
	lote := contabilizacao.Lote{PlanID: *contas.plano, Data: fim, Prefixo: "EST", SourceType: origemFechamento, SourceID: closing, Lancamentos: ls}
	if err := journal.Gravar(ctx, tx, empresa, lote); err != nil {
		return fmt.Errorf("contabilizando o fechamento: %w", err)
	}
	res.Contabilizado = true
	return nil
}

// tipoDaProducaoSQL: a entrada é do produto; o resto é material. O estorno guarda a
// referência no movimento original, por isso a leitura passa por ele.
const entregaDaOrdemSQL = `COALESCE(o.movement_type, m.movement_type) IN ('EP','EPP','EPE')`

// contabilizarProducao apura cada ordem com movimento no mês ou com saldo em
// processo, e grava o que foi levado. Ligada a contabilização, faltar conta para o
// fechamento: o mês não volta, e a produção dele ficaria sem lançamento para sempre.
func contabilizarProducao(ctx context.Context, tx pgx.Tx, empresa, closing int64, janela, fim time.Time, contas contasDoFechamento, simular bool, res *entity.ResultadoFechamento) ([]contabilizacao.Lancamento, error) {
	rows, err := tx.Query(ctx,
		`WITH mov AS (
		     SELECT COALESCE(o.reference_code, m.reference_code) ordem,
		            SUM(CASE WHEN `+entregaDaOrdemSQL+` THEN 0 ELSE -`+valorSQL+` END) consumo,
		            SUM(CASE WHEN `+entregaDaOrdemSQL+` THEN `+valorSQL+` ELSE 0 END) entrega
		       FROM stock_movements m
		       LEFT JOIN stock_movements o ON o.id = m.reverses_movement_id AND o.enterprise_id = m.enterprise_id
		      WHERE m.enterprise_id=$1 AND m.movement_date BETWEEN $2::date AND $3::date
		        AND COALESCE(o.reference_type, m.reference_type) = 'PRODUCTION_ORDER'
		      GROUP BY 1),
		  antes AS (
		     SELECT p.production_order_id ordem, SUM(p.consumed_value) c, SUM(p.delivered_value) e,
		            SUM(p.absorbed_value) a, SUM(p.variance_value) v
		       FROM stock_closing_production p
		       JOIN stock_closings f ON f.id = p.closing_id
		      WHERE f.enterprise_id=$1 AND f.status='FECHADO' AND f.id <> $4
		      GROUP BY 1)
		 SELECT ordens.id, COALESCE(mov.consumo,0), COALESCE(mov.entrega,0),
		        COALESCE(antes.c,0), COALESCE(antes.e,0), COALESCE(antes.a,0), COALESCE(antes.v,0),
		        ordens.status IN ('COMPLETED','CLOSED','CANCELLED')
		   FROM mov FULL JOIN antes USING (ordem)
		   JOIN production_orders ordens ON ordens.id = COALESCE(mov.ordem, antes.ordem) AND ordens.enterprise_id = $1
		  ORDER BY ordens.id`,
		empresa, janela.Format("2006-01-02"), fim.Format("2006-01-02"), closing)
	if err != nil {
		return nil, fmt.Errorf("lendo a produção do mês: %w", err)
	}
	defer rows.Close()
	type linha struct {
		ordem int64
		a     contabilizacao.ApuracaoOrdem
	}
	var linhas []linha
	var total contabilizacao.ApuracaoOrdem
	for rows.Next() {
		var l linha
		var o contabilizacao.OrdemNoMes
		if err := rows.Scan(&l.ordem, &o.Consumo, &o.Entrega, &o.ConsumoAntes, &o.EntregaAntes, &o.AbsorvidoAntes, &o.VariacaoAntes, &o.Encerrada); err != nil {
			return nil, fmt.Errorf("lendo a produção do mês: %w", err)
		}
		l.a = contabilizacao.ApurarOrdem(o)
		if l.a.Consumo.IsZero() && l.a.Entrega.IsZero() && l.a.Absorvido.IsZero() && l.a.Variacao.IsZero() {
			continue
		}
		total.Consumo = total.Consumo.Add(l.a.Consumo)
		total.Entrega = total.Entrega.Add(l.a.Entrega)
		total.Absorvido = total.Absorvido.Add(l.a.Absorvido)
		total.Variacao = total.Variacao.Add(l.a.Variacao)
		linhas = append(linhas, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	res.ProducaoConsumo, res.ProducaoEntrega, res.ProducaoAbsorvida, res.ProducaoVariacao = total.Consumo, total.Entrega, total.Absorvido, total.Variacao
	if len(linhas) == 0 {
		return nil, nil
	}
	ls, faltando := contabilizacao.Producao(total, contabilizacao.ContasProducao{Estoque: contas.estoque, Processo: contas.processo, Absorvido: contas.absorvidos, Variacao: contas.cmv}, fim.Format("01/2006"))
	if contas.plano == nil {
		faltando = append(faltando, "falta o plano contábil")
	}
	if len(faltando) > 0 {
		motivo := "a contabilização da produção está ligada, mas " + strings.Join(faltando, "; ")
		if !simular {
			return nil, errorsuc.NewValidationError("o mês não pode ser fechado: " + motivo)
		}
		res.Pendencias = append(res.Pendencias, motivo)
		return nil, nil
	}
	for _, l := range linhas {
		if _, err := tx.Exec(ctx,
			`INSERT INTO stock_closing_production(closing_id,production_order_id,consumed_value,delivered_value,absorbed_value,variance_value)
			 VALUES($1,$2,$3,$4,$5,$6)`, closing, l.ordem, l.a.Consumo, l.a.Entrega, l.a.Absorvido, l.a.Variacao); err != nil {
			return nil, fmt.Errorf("gravando a produção do fechamento: %w", err)
		}
	}
	return ls, nil
}

// contabilizarProvisao recalcula a provisão pela foto do mês e lança a variação.
// Sem conta, o saldo fica como estava e o mês seguinte lança a variação acumulada.
func contabilizarProvisao(ctx context.Context, tx pgx.Tx, empresa, closing int64, fim time.Time, contas contasDoFechamento, res *entity.ResultadoFechamento) ([]contabilizacao.Lancamento, error) {
	param, err := lerParametrosDeEstoque(ctx, tx, empresa)
	if err != nil {
		return nil, err
	}
	saldo := decimal.Zero
	if param.ObsoleteDays != nil && param.ObsoleteProvisionPct != nil {
		if err := tx.QueryRow(ctx,
			`WITH foto AS (
			     SELECT item_code, SUM(total_value) v FROM stock_closing_balances WHERE closing_id=$1 GROUP BY item_code HAVING SUM(quantity) > 0),
			  ultimo AS (
			     SELECT item_code, MAX(movement_date) quando FROM stock_movements
			      WHERE enterprise_id=$2 AND movement_date <= $3::date AND `+movimentoQueGiraSQL+`
			      GROUP BY item_code)
			 SELECT COALESCE(SUM(ROUND(f.v * $4::numeric / 100, 2)),0)
			   FROM foto f LEFT JOIN ultimo u USING (item_code)
			  WHERE u.quando IS NULL OR $3::date - u.quando >= $5::int`,
			closing, empresa, fim.Format("2006-01-02"), *param.ObsoleteProvisionPct, *param.ObsoleteDays).Scan(&saldo); err != nil {
			return nil, fmt.Errorf("calculando a provisão de obsoletos: %w", err)
		}
	}
	var anterior decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT obsolete_provision FROM stock_closings WHERE enterprise_id=$1 AND status='FECHADO' AND id<>$2 ORDER BY competence DESC LIMIT 1`,
		empresa, closing).Scan(&anterior)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lendo a provisão anterior: %w", err)
	}
	res.ProvisaoObsoletos, res.ProvisaoVariacao = saldo, saldo.Sub(anterior)
	ls, faltando := contabilizacao.ProvisaoObsoletos(saldo, anterior, contas.despesaProvisao, contas.provisao, fim.Format("01/2006"))
	if contas.plano == nil && !res.ProvisaoVariacao.IsZero() {
		faltando = append(faltando, "falta o plano contábil")
	}
	gravado := saldo
	if len(faltando) > 0 {
		res.Avisos = append(res.Avisos, "A provisão de obsoletos não foi contabilizada: "+strings.Join(faltando, "; ")+".")
		gravado, ls = anterior, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_closings SET obsolete_provision=$2 WHERE id=$1`, closing, gravado); err != nil {
		return nil, fmt.Errorf("gravando a provisão: %w", err)
	}
	return ls, nil
}
