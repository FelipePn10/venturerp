package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/journal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

const (
	origemFechamento        = "FECHAMENTO_ESTOQUE"
	origemFechamentoEstorno = "FECH_ESTOQUE_ESTORNO"
)

// registrarDiferencaDeCusto guarda um valor que saiu do estoque sem virar saldo.
func registrarDiferencaDeCusto(ctx context.Context, tx pgx.Tx, enterpriseID int64, tipo string, m *entity.StockMovement, valor decimal.Decimal) error {
	if valor.IsZero() {
		return nil
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO stock_cost_variances(enterprise_id,kind,item_code,mask,warehouse_id,movement_id,value,occurred_on)
		 VALUES($1,$2,$3,$4,$5,$6,$7,COALESCE($8::date,CURRENT_DATE))`,
		enterpriseID, tipo, m.ItemCode, m.Mask, m.WarehouseID, m.ID, valor, m.MovementDate); err != nil {
		return fmt.Errorf("registrando a diferença de custo: %w", err)
	}
	return nil
}

// Fechamentos lista os meses fechados e reabertos, do mais recente ao mais antigo.
func (r *StockRepositorySQLC) Fechamentos(ctx context.Context) ([]entity.Fechamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, to_char(competence,'YYYY-MM-DD'), status, total_value, recalculated_items, variance_value, journal_posted, closed_at, reopened_at, reopen_reason
		   FROM stock_closings WHERE enterprise_id=$1 ORDER BY competence DESC, id DESC LIMIT 120`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os fechamentos de estoque: %w", err)
	}
	defer rows.Close()
	out := []entity.Fechamento{}
	for rows.Next() {
		var f entity.Fechamento
		if err := rows.Scan(&f.ID, &f.Competencia, &f.Status, &f.ValorTotal, &f.ItensRecalculados, &f.Diferenca, &f.Contabilizado, &f.FechadoEm, &f.ReabertoEm, &f.MotivoReabertura); err != nil {
			return nil, fmt.Errorf("lendo o fechamento de estoque: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// FecharMes fecha a competência: recalcula quem teve lançamento retroativo, grava
// a foto de saldo e custo, leva as diferenças ao resultado e fecha o período. Tudo
// numa transação; a simulação faz o mesmo e desfaz no fim.
func (r *StockRepositorySQLC) FecharMes(ctx context.Context, competencia time.Time, ator uuid.UUID, simular bool) (*entity.ResultadoFechamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	inicio := time.Date(competencia.Year(), competencia.Month(), 1, 0, 0, 0, 0, time.UTC)
	fim := inicio.AddDate(0, 1, -1)
	res := &entity.ResultadoFechamento{Competencia: inicio.Format("2006-01-02"), Simulacao: simular, Pendencias: []string{}, Avisos: []string{}}

	inconsistencias, err := r.ListarInconsistencias(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo o fechamento de estoque: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, chaveDoFechamento(empresa)); err != nil {
		return nil, fmt.Errorf("travando o fechamento: %w", err)
	}

	janela, err := conferirCompetencia(ctx, tx, empresa, inicio, fim, res)
	if err != nil {
		return nil, err
	}
	if len(inconsistencias) > 0 {
		res.Pendencias = append(res.Pendencias, fmt.Sprintf("%d inconsistência(s) de saldo: acerte em Estoque › Conferência do saldo", len(inconsistencias)))
	}
	if len(res.Pendencias) > 0 && !simular {
		return nil, errorsuc.NewValidationError("o mês não pode ser fechado: " + strings.Join(res.Pendencias, "; "))
	}

	var id int64
	if err := tx.QueryRow(ctx, `INSERT INTO stock_closings(enterprise_id,competence,closed_by) VALUES($1,$2,$3) RETURNING id`,
		empresa, inicio.Format("2006-01-02"), ator).Scan(&id); err != nil {
		return nil, fmt.Errorf("gravando o fechamento: %w", err)
	}
	if err := recalcularRetroativos(ctx, tx, empresa, id, janela, fim, res); err != nil {
		return nil, err
	}
	if err := gravarFoto(ctx, tx, empresa, id, fim, res); err != nil {
		return nil, err
	}
	liquida, err := apurarDiferencas(ctx, tx, empresa, id, fim, res)
	if err != nil {
		return nil, err
	}
	if err := contabilizarFechamento(ctx, tx, empresa, id, fim, liquida, res); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO manufacturing_stock_closed_periods(enterprise_id,period_from,period_to) VALUES($1,$2,$3)`,
		empresa, janela.Format("2006-01-02"), fim.Format("2006-01-02")); err != nil {
		return nil, fmt.Errorf("fechando o período: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_movements SET backdated=FALSE WHERE enterprise_id=$1 AND backdated AND movement_date <= $2::date`,
		empresa, fim.Format("2006-01-02")); err != nil {
		return nil, fmt.Errorf("baixando os lançamentos retroativos: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE stock_closings SET total_value=$2, recalculated_items=$3, variance_value=$4, journal_posted=$5 WHERE id=$1`,
		id, res.ValorTotal, res.ItensRecalculados, liquida, res.Contabilizado); err != nil {
		return nil, fmt.Errorf("gravando o resumo do fechamento: %w", err)
	}
	if simular {
		return res, nil // o defer desfaz
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmando o fechamento: %w", err)
	}
	return res, nil
}

// conferirCompetencia aponta o que impede fechar e devolve o primeiro dia aberto.
// Sem fechamento anterior, o primeiro mês fechado leva junto tudo o que veio antes.
func conferirCompetencia(ctx context.Context, tx pgx.Tx, empresa int64, inicio, fim time.Time, res *entity.ResultadoFechamento) (time.Time, error) {
	janela := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	var terminou bool
	if err := tx.QueryRow(ctx, `SELECT $1::date < CURRENT_DATE`, fim.Format("2006-01-02")).Scan(&terminou); err != nil {
		return janela, fmt.Errorf("conferindo a competência: %w", err)
	}
	if !terminou {
		res.Pendencias = append(res.Pendencias, "o mês ainda não terminou")
	}
	var ultima time.Time
	err := tx.QueryRow(ctx, `SELECT competence FROM stock_closings WHERE enterprise_id=$1 AND status='FECHADO' ORDER BY competence DESC LIMIT 1`, empresa).Scan(&ultima)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return janela, fmt.Errorf("lendo o último fechamento: %w", err)
	}
	if err == nil {
		proxima := ultima.AddDate(0, 1, 0)
		janela = proxima
		if !proxima.Equal(inicio) {
			res.Pendencias = append(res.Pendencias, fmt.Sprintf("o próximo mês a fechar é %s", proxima.Format("01/2006")))
		}
	}
	var andamento int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM physical_inventories WHERE enterprise_id=$1 AND status IN ('OPEN','EM_CONFERENCIA','AGUARDANDO_APROVACAO')`, empresa).Scan(&andamento); err != nil {
		return janela, fmt.Errorf("conferindo os inventários: %w", err)
	}
	if andamento > 0 {
		res.Pendencias = append(res.Pendencias, fmt.Sprintf("%d inventário(s) em andamento: termine antes de fechar", andamento))
	}
	return janela, nil
}

type retroativo struct {
	item     int64
	primeira time.Time
	// entradaMudou: a entrada do produto já foi repassada; o saldo acompanha mesmo
	// sem nenhuma saída a refazer.
	entradaMudou bool
}

// recalcularRetroativos refaz o custo só dos itens com lançamento fora de ordem,
// a partir do primeiro deles. Os demais já estão certos: o custo é em tempo real.
func recalcularRetroativos(ctx context.Context, tx pgx.Tx, empresa, closing int64, janela, fim time.Time, res *entity.ResultadoFechamento) error {
	rows, err := tx.Query(ctx,
		`SELECT item_code, MIN(movement_date) FROM stock_movements
		  WHERE enterprise_id=$1 AND backdated AND movement_date >= $2::date AND movement_date <= $3::date GROUP BY item_code ORDER BY item_code`,
		empresa, janela.Format("2006-01-02"), fim.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("listando os lançamentos retroativos: %w", err)
	}
	var itens []retroativo
	for rows.Next() {
		var x retroativo
		if err := rows.Scan(&x.item, &x.primeira); err != nil {
			rows.Close()
			return fmt.Errorf("lendo o lançamento retroativo: %w", err)
		}
		itens = append(itens, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("lendo os lançamentos retroativos: %w", err)
	}
	// Em níveis: o consumo da ordem mudou de valor, a entrada do produto acompanha,
	// e o produto vira a próxima rodada. O limite é a defesa contra estrutura em ciclo.
	fila := itens
	vistos := map[int64]int{}
	for len(fila) > 0 {
		x := fila[0]
		fila = fila[1:]
		if vistos[x.item]++; vistos[x.item] > 20 {
			continue
		}
		ordens, err := recalcularItem(ctx, tx, empresa, closing, x.item, x.primeira, x.entradaMudou, res)
		if err != nil {
			return err
		}
		for ordem, diferenca := range ordens {
			produto, data, err := repassarAoProduto(ctx, tx, empresa, closing, ordem, diferenca)
			if err != nil {
				return err
			}
			if produto != 0 {
				fila = append(fila, retroativo{item: produto, primeira: data, entradaMudou: true})
			}
		}
	}
	return nil
}

// repassarAoProduto soma ao valor das entradas da ordem a diferença do material
// consumido, repartida pela quantidade. Devolve o produto e a data a recalcular.
func repassarAoProduto(ctx context.Context, tx pgx.Tx, empresa, closing, ordem int64, diferenca decimal.Decimal) (int64, time.Time, error) {
	type entrada struct {
		id         int64
		qtd, total decimal.Decimal
		unit       decimal.Decimal
		item       int64
		data       time.Time
	}
	rows, err := tx.Query(ctx,
		`SELECT id, quantity, total_price, unit_price, item_code, movement_date FROM stock_movements
		  WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3 AND movement_type IN ('EP','EPP','EPE')
		    AND reversed_by_movement_id IS NULL ORDER BY id`, empresa, entity.ReferenceTypeProductionOrder, ordem)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("lendo a entrada da ordem: %w", err)
	}
	var entradas []entrada
	soma := decimal.Zero
	for rows.Next() {
		var e entrada
		if err := rows.Scan(&e.id, &e.qtd, &e.total, &e.unit, &e.item, &e.data); err != nil {
			rows.Close()
			return 0, time.Time{}, fmt.Errorf("lendo a entrada da ordem: %w", err)
		}
		entradas = append(entradas, e)
		soma = soma.Add(e.qtd)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, time.Time{}, fmt.Errorf("lendo a entrada da ordem: %w", err)
	}
	if !soma.IsPositive() {
		return 0, time.Time{}, nil
	}
	for _, e := range entradas {
		novoTotal := e.total.Add(diferenca.Mul(e.qtd).Div(soma)).Round(entity.CasasValor)
		novoUnit := novoTotal.DivRound(e.qtd, entity.CasasCusto)
		if _, err := tx.Exec(ctx, `UPDATE stock_movements SET unit_price=$2, total_price=$3 WHERE id=$1`, e.id, novoUnit, novoTotal); err != nil {
			return 0, time.Time{}, fmt.Errorf("gravando a entrada da ordem: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO stock_cost_recalc_log(closing_id,movement_id,old_unit,new_unit,old_total,new_total) VALUES($1,$2,$3,$4,$5,$6)`,
			closing, e.id, e.unit, novoUnit, e.total, novoTotal); err != nil {
			return 0, time.Time{}, fmt.Errorf("registrando a entrada da ordem: %w", err)
		}
	}
	return entradas[0].item, entradas[0].data, nil
}

// recalcularItem refaz o custo do item e devolve, por ordem de produção, quanto o
// consumo mudou de valor: essa diferença vai para o produto, não para o resultado.
func recalcularItem(ctx context.Context, tx pgx.Tx, empresa, closing, item int64, primeira time.Time, entradaMudou bool, res *entity.ResultadoFechamento) (map[int64]decimal.Decimal, error) {
	data := primeira.Format("2006-01-02")
	estados := map[entity.ChaveCusto]entity.CostingState{}
	rows, err := tx.Query(ctx,
		`SELECT m.warehouse_id, m.mask, SUM(`+qtdSQL+`), SUM(`+valorSQL+`),
		        COALESCE((SELECT MAX(b.avg_cost) FROM stock_balances b WHERE b.enterprise_id=m.enterprise_id AND b.item_code=m.item_code AND b.mask=m.mask AND b.warehouse_id=m.warehouse_id), 0)
		   FROM stock_movements m WHERE m.enterprise_id=$1 AND m.item_code=$2 AND m.movement_date < $3::date
		  GROUP BY m.enterprise_id, m.item_code, m.warehouse_id, m.mask`, empresa, item, data)
	if err != nil {
		return nil, fmt.Errorf("lendo o saldo antes do retroativo: %w", err)
	}
	for rows.Next() {
		var k entity.ChaveCusto
		var st entity.CostingState
		var medioAtual decimal.Decimal
		if err := rows.Scan(&k.Almoxarifado, &k.Mask, &st.Quantity, &st.TotalCost, &medioAtual); err != nil {
			rows.Close()
			return nil, fmt.Errorf("lendo o saldo antes do retroativo: %w", err)
		}
		st.AvgCost = medioAtual
		if st.Quantity.IsPositive() {
			st.AvgCost = st.TotalCost.DivRound(st.Quantity, entity.CasasCusto)
		}
		estados[k] = st
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lendo o saldo antes do retroativo: %w", err)
	}
	rows, err = tx.Query(ctx,
		`SELECT m.id, m.warehouse_id, m.mask, m.movement_type, m.quantity, m.unit_price, m.total_price, COALESCE(m.reference_type,''),
		        m.reverses_movement_id, COALESCE(NOT t.updates_avg_cost, FALSE), m.movement_date
		   FROM stock_movements m
		   LEFT JOIN stock_movement_types t ON t.enterprise_id=m.enterprise_id AND t.sigla=m.movement_type_code
		  WHERE m.enterprise_id=$1 AND m.item_code=$2 AND m.movement_date >= $3::date
		  ORDER BY m.movement_date, m.id`, empresa, item, data)
	if err != nil {
		return nil, fmt.Errorf("lendo os movimentos do item: %w", err)
	}
	var movs []entity.MovimentoCusto
	for rows.Next() {
		var m entity.MovimentoCusto
		if err := rows.Scan(&m.ID, &m.Almoxarifado, &m.Mask, &m.Tipo, &m.Quantidade, &m.Unitario, &m.Total, &m.Referencia, &m.EstornaDe, &m.SemMedia, &m.Data); err != nil {
			rows.Close()
			return nil, fmt.Errorf("lendo o movimento do item: %w", err)
		}
		movs = append(movs, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lendo os movimentos do item: %w", err)
	}
	mudancas := entity.RecalcularCusto(estados, movs)
	if len(mudancas) == 0 && !entradaMudou {
		return nil, nil
	}
	res.ItensRecalculados++
	ordens := map[int64]decimal.Decimal{}
	ordemDoConsumo, err := ordensDoConsumo(ctx, tx, empresa, item, data)
	if err != nil {
		return nil, err
	}
	for _, c := range mudancas {
		res.MovimentosAlterados++
		if _, err := tx.Exec(ctx, `UPDATE stock_movements SET unit_price=$2, total_price=$3 WHERE id=$1`, c.MovimentoID, c.UnitarioDepois, c.TotalDepois); err != nil {
			return nil, fmt.Errorf("gravando o custo recalculado: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO stock_cost_recalc_log(closing_id,movement_id,old_unit,new_unit,old_total,new_total) VALUES($1,$2,$3,$4,$5,$6)`,
			closing, c.MovimentoID, c.UnitarioAntes, c.UnitarioDepois, c.TotalAntes, c.TotalDepois); err != nil {
			return nil, fmt.Errorf("registrando o custo recalculado: %w", err)
		}
		m := &entity.StockMovement{ID: c.MovimentoID, ItemCode: item, Mask: c.Mask, WarehouseID: c.Almoxarifado}
		data := c.Data
		m.MovementDate = &data
		if ordem, ok := ordemDoConsumo[c.MovimentoID]; ok {
			ordens[ordem] = ordens[ordem].Add(c.TotalDepois.Sub(c.TotalAntes))
			continue
		}
		if err := registrarDiferencaDeCusto(ctx, tx, empresa, "RECALCULO", m, c.DiferencaResultado); err != nil {
			return nil, err
		}
	}
	for k, st := range estados {
		if _, err := tx.Exec(ctx,
			`UPDATE stock_balances SET avg_cost=$5, total_cost = CASE WHEN quantity > 0 THEN quantity * $5 ELSE 0 END, updated_at=NOW()
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4`, empresa, item, k.Mask, k.Almoxarifado, st.AvgCost); err != nil {
			return nil, fmt.Errorf("gravando o custo do saldo: %w", err)
		}
	}
	return ordens, nil
}

// ordensDoConsumo: saídas do item para ordem de produção que teve entrada.
func ordensDoConsumo(ctx context.Context, tx pgx.Tx, empresa, item int64, desde string) (map[int64]int64, error) {
	rows, err := tx.Query(ctx,
		`SELECT m.id, m.reference_code FROM stock_movements m
		  WHERE m.enterprise_id=$1 AND m.item_code=$2 AND m.movement_date >= $3::date AND m.reference_type=$4
		    AND m.movement_type IN ('OUT','REP') AND m.reference_code IS NOT NULL
		    AND EXISTS (SELECT 1 FROM stock_movements e WHERE e.enterprise_id=m.enterprise_id AND e.reference_type=m.reference_type
		                 AND e.reference_code=m.reference_code AND e.movement_type IN ('EP','EPP','EPE') AND e.reversed_by_movement_id IS NULL)`,
		empresa, item, desde, entity.ReferenceTypeProductionOrder)
	if err != nil {
		return nil, fmt.Errorf("lendo o consumo das ordens: %w", err)
	}
	defer rows.Close()
	out := map[int64]int64{}
	for rows.Next() {
		var id, ordem int64
		if err := rows.Scan(&id, &ordem); err != nil {
			return nil, fmt.Errorf("lendo o consumo das ordens: %w", err)
		}
		out[id] = ordem
	}
	return out, rows.Err()
}

// gravarFoto guarda o saldo e o valor no fim do mês, o saldo inicial do seguinte.
func gravarFoto(ctx context.Context, tx pgx.Tx, empresa, closing int64, fim time.Time, res *entity.ResultadoFechamento) error {
	tag, err := tx.Exec(ctx,
		`INSERT INTO stock_closing_balances(closing_id,item_code,mask,warehouse_id,quantity,avg_cost,total_value)
		 SELECT $1, m.item_code, m.mask, m.warehouse_id, SUM(`+qtdSQL+`),
		        CASE WHEN SUM(`+qtdSQL+`) > 0 THEN ROUND(SUM(`+valorSQL+`) / SUM(`+qtdSQL+`), 10) ELSE 0 END, SUM(`+valorSQL+`)
		   FROM stock_movements m WHERE m.enterprise_id=$2 AND m.movement_date <= $3::date
		  GROUP BY m.item_code, m.mask, m.warehouse_id
		 HAVING SUM(`+qtdSQL+`) <> 0`, closing, empresa, fim.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("gravando a foto do fechamento: %w", err)
	}
	res.ItensNaFoto = int(tag.RowsAffected())
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(total_value),0) FROM stock_closing_balances WHERE closing_id=$1`, closing).Scan(&res.ValorTotal); err != nil {
		return fmt.Errorf("somando a foto do fechamento: %w", err)
	}
	res.ValorTotal = res.ValorTotal.Round(2)
	return nil
}

func apurarDiferencas(ctx context.Context, tx pgx.Tx, empresa, closing int64, fim time.Time, res *entity.ResultadoFechamento) (decimal.Decimal, error) {
	if _, err := tx.Exec(ctx, `UPDATE stock_cost_variances SET closing_id=$3 WHERE enterprise_id=$1 AND closing_id IS NULL AND occurred_on <= $2::date`,
		empresa, fim.Format("2006-01-02"), closing); err != nil {
		return decimal.Zero, fmt.Errorf("apurando as diferenças de custo: %w", err)
	}
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(value) FILTER (WHERE kind='RECALCULO'),0), COALESCE(SUM(value) FILTER (WHERE kind='SALDO_NEGATIVO'),0)
		   FROM stock_cost_variances WHERE closing_id=$1`, closing).Scan(&res.DiferencaRecalculo, &res.DiferencaSaldoNeg); err != nil {
		return decimal.Zero, fmt.Errorf("somando as diferenças de custo: %w", err)
	}
	return res.DiferencaRecalculo.Add(res.DiferencaSaldoNeg).Round(2), nil
}

// contabilizarFechamento leva a diferença líquida ao resultado: débito no CMV e
// crédito no estoque, ou o contrário. Sem as contas configuradas, avisa.
func contabilizarFechamento(ctx context.Context, tx pgx.Tx, empresa, closing int64, fim time.Time, liquida decimal.Decimal, res *entity.ResultadoFechamento) error {
	if liquida.IsZero() {
		return nil
	}
	var plano, cmv, estoque *int64
	err := tx.QueryRow(ctx, `SELECT plan_id, cmv_account_id, estoque_account_id FROM accounting_posting_params WHERE enterprise_id=$1`, empresa).Scan(&plano, &cmv, &estoque)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lendo as contas do fechamento: %w", err)
	}
	if plano == nil || cmv == nil || estoque == nil {
		res.Avisos = append(res.Avisos, "A diferença de custo não foi contabilizada: configure as contas de CMV e Estoque nos parâmetros contábeis.")
		return nil
	}
	debito, credito := *cmv, *estoque
	if liquida.IsNegative() {
		debito, credito = credito, debito
	}
	lote := contabilizacao.Lote{PlanID: *plano, Data: fim, Prefixo: "EST", SourceType: origemFechamento, SourceID: closing,
		Lancamentos: []contabilizacao.Lancamento{{DebitoID: debito, CreditoID: credito, Valor: liquida.Abs(),
			Historico: "Diferença de custo do fechamento de estoque " + fim.Format("01/2006")}}}
	if err := journal.Gravar(ctx, tx, empresa, lote); err != nil {
		return fmt.Errorf("contabilizando o fechamento: %w", err)
	}
	res.Contabilizado = true
	return nil
}

// ReabrirMes reabre o último mês fechado. É exceção: desfaz o período e o
// lançamento contábil; os custos recalculados ficam e a foto vira histórico.
func (r *StockRepositorySQLC) ReabrirMes(ctx context.Context, id int64, motivo string, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrindo a reabertura: %w", err)
	}
	defer tx.Rollback(ctx)
	var ultimo int64
	var competencia time.Time
	var contabilizado bool
	err = tx.QueryRow(ctx,
		`SELECT id, competence, journal_posted FROM stock_closings WHERE enterprise_id=$1 AND status='FECHADO' ORDER BY competence DESC LIMIT 1 FOR UPDATE`,
		empresa).Scan(&ultimo, &competencia, &contabilizado)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewValidationError("não há mês fechado para reabrir")
	}
	if err != nil {
		return fmt.Errorf("lendo o último fechamento: %w", err)
	}
	if ultimo != id {
		return errorsuc.NewValidationError("só o último mês fechado pode ser reaberto")
	}
	fim := competencia.AddDate(0, 1, -1)
	if _, err := tx.Exec(ctx, `DELETE FROM manufacturing_stock_closed_periods WHERE enterprise_id=$1 AND period_to=$2::date`, empresa, fim.Format("2006-01-02")); err != nil {
		return fmt.Errorf("reabrindo o período: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_cost_variances SET closing_id=NULL WHERE closing_id=$1 AND kind='SALDO_NEGATIVO'`, id); err != nil {
		return fmt.Errorf("devolvendo as diferenças de custo: %w", err)
	}
	if contabilizado {
		if _, err := journal.Estornar(ctx, tx, empresa, origemFechamento, id, origemFechamentoEstorno, "ESTE", fim); err != nil {
			return fmt.Errorf("estornando o lançamento do fechamento: %w", err)
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE stock_closings SET status='REABERTO', reopened_by=$2, reopened_at=NOW(), reopen_reason=$3 WHERE id=$1`, id, ator, motivo); err != nil {
		return fmt.Errorf("gravando a reabertura: %w", err)
	}
	return tx.Commit(ctx)
}
