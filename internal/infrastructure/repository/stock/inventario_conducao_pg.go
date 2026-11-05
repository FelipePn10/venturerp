package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ResumoDoFechamento diz o que o inventário ajustaria se fechasse agora: é o que
// decide se o ajuste vai para a alçada.
func (r *StockRepositorySQLC) ResumoDoFechamento(ctx context.Context, inv *entity.PhysicalInventory, zerarNaoContados bool) (entity.ResumoFechamento, error) {
	var resumo entity.ResumoFechamento
	itens, err := r.ListInventoryItems(ctx, inv.ID)
	if err != nil {
		return resumo, err
	}
	tolerancia, err := r.toleranciaDoInventario(ctx, inv)
	if err != nil {
		return resumo, err
	}
	for _, it := range itens {
		dif := decimal.Zero
		switch {
		case it.CountedQty != nil && it.DifferenceQty != nil:
			dif = *it.DifferenceQty
		case zerarNaoContados:
			dif = it.SystemQty.Neg()
		default:
			resumo.NaoContados++
			continue
		}
		if dif.IsZero() {
			continue
		}
		custo := decimal.Zero
		if it.UnitCost != nil {
			custo = *it.UnitCost
		}
		resumo.ComDiferenca++
		valor := dif.Abs().Mul(custo)
		resumo.ValorAbsoluto = resumo.ValorAbsoluto.Add(valor)
		resumo.MaiorLinha = decimal.Max(resumo.MaiorLinha, valor)
		if entity.ForaDaTolerancia(it.SystemQty, dif, custo, tolerancia(it.ItemCode), inv.ToleranceValue) {
			resumo.ForaTolerancia++
		}
	}
	return resumo, nil
}

func (r *StockRepositorySQLC) mudarSituacaoDoInventario(ctx context.Context, id int64, de []string, para, recusa string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE public.physical_inventories SET status=$3, updated_at=NOW() WHERE id=$1 AND enterprise_id=$2 AND status = ANY($4)`,
		id, empresa, para, de)
	if err != nil {
		return fmt.Errorf("mudando a situação do inventário: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError(recusa)
	}
	return nil
}

// EncerrarContagem leva o inventário à conferência.
func (r *StockRepositorySQLC) EncerrarContagem(ctx context.Context, id int64) error {
	return r.mudarSituacaoDoInventario(ctx, id, []string{entity.InventarioAberto}, entity.InventarioEmConferencia,
		"somente inventário em contagem pode ir para a conferência")
}

// DevolverInventarioParaConferencia: a alçada reprovou o ajuste.
func (r *StockRepositorySQLC) DevolverInventarioParaConferencia(ctx context.Context, id int64) error {
	return r.mudarSituacaoDoInventario(ctx, id, []string{entity.InventarioAguardandoAprovacao}, entity.InventarioEmConferencia,
		"o inventário não está aguardando aprovação")
}

// AguardarAprovacaoDoInventario guarda como o fechamento foi pedido, para a
// aprovação fechar do mesmo jeito.
func (r *StockRepositorySQLC) AguardarAprovacaoDoInventario(ctx context.Context, id int64, zerarNaoContados bool, motivo string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE public.physical_inventories
		    SET status='AGUARDANDO_APROVACAO', zero_uncounted=$3, close_reason=NULLIF($4,''), updated_at=NOW()
		  WHERE id=$1 AND enterprise_id=$2 AND status IN ('OPEN','EM_CONFERENCIA')`, id, empresa, zerarNaoContados, motivo)
	if err != nil {
		return fmt.Errorf("enviando o inventário para aprovação: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("somente inventário em contagem ou conferência vai para aprovação")
	}
	return nil
}

// MarcarRecontagem devolve linhas à contagem. A contagem anterior fica guardada
// e o inventário volta a aceitar contagem.
func (r *StockRepositorySQLC) MarcarRecontagem(ctx context.Context, inv *entity.PhysicalInventory, linhas []int64) (int, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	if inv.Status != entity.InventarioEmConferencia {
		return 0, errorsuc.NewValidationError("a recontagem é pedida na conferência: encerre a contagem antes")
	}
	alvo := linhas
	if len(alvo) == 0 {
		itens, err := r.ListInventoryItems(ctx, inv.ID)
		if err != nil {
			return 0, err
		}
		tolerancia, err := r.toleranciaDoInventario(ctx, inv)
		if err != nil {
			return 0, err
		}
		for _, it := range itens {
			if it.CountedQty == nil || it.DifferenceQty == nil || it.DifferenceQty.IsZero() {
				continue
			}
			custo := decimal.Zero
			if it.UnitCost != nil {
				custo = *it.UnitCost
			}
			pct := tolerancia(it.ItemCode)
			if (pct == nil && inv.ToleranceValue == nil) || entity.ForaDaTolerancia(it.SystemQty, *it.DifferenceQty, custo, pct, inv.ToleranceValue) {
				alvo = append(alvo, it.ID)
			}
		}
	}
	if len(alvo) == 0 {
		return 0, errorsuc.NewValidationError("nenhuma linha com diferença para recontar")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("abrindo a recontagem: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx,
		`UPDATE public.physical_inventory_items i SET needs_recount=TRUE
		  WHERE i.inventory_id=$1 AND i.id = ANY($2)
		    AND EXISTS (SELECT 1 FROM public.physical_inventories p WHERE p.id=i.inventory_id AND p.enterprise_id=$3)`,
		inv.ID, alvo, empresa)
	if err != nil {
		return 0, fmt.Errorf("marcando a recontagem: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE public.physical_inventories SET status='OPEN', updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, inv.ID, empresa); err != nil {
		return 0, fmt.Errorf("reabrindo a contagem: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("confirmando a recontagem: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// consultaInconsistencias compara o saldo gravado com o que deveria ser: a soma
// dos movimentos, o saldo que cobre os lotes e as reservas ativas.
const consultaInconsistencias = `
WITH saldo AS (
    SELECT b.item_code, b.mask, b.warehouse_id, SUM(b.quantity) q, SUM(b.reserved_qty) r
      FROM stock_balances b WHERE b.enterprise_id=$1 GROUP BY 1,2,3),
mov AS (
    SELECT m.item_code, m.mask, m.warehouse_id, SUM(` + qtdSQL + `) q
      FROM stock_movements m WHERE m.enterprise_id=$1 GROUP BY 1,2,3),
lote AS (
    SELECT l.item_code, l.mask, l.warehouse_id, SUM(l.quantity) q, MIN(l.quantity) menor
      FROM stock_lot_balances l WHERE l.enterprise_id=$1 GROUP BY 1,2,3),
reserva AS (
    SELECT v.item_code, v.mask, v.warehouse_id, SUM(v.quantity) q
      FROM stock_reservations v WHERE v.enterprise_id=$1 AND v.status='ACTIVE' GROUP BY 1,2,3),
achados AS (
    SELECT 'SALDO_X_MOVIMENTOS' tipo, COALESCE(s.item_code,m.item_code) item_code, COALESCE(s.mask,m.mask) mask,
           COALESCE(s.warehouse_id,m.warehouse_id) warehouse_id, COALESCE(s.q,0) gravado, COALESCE(m.q,0) esperado
      FROM saldo s FULL JOIN mov m USING (item_code, mask, warehouse_id)
     WHERE COALESCE(s.q,0) <> COALESCE(m.q,0)
    UNION ALL
    SELECT 'LOTE_X_SALDO', l.item_code, l.mask, l.warehouse_id, l.q, LEAST(l.q, GREATEST(COALESCE(s.q,0),0))
      FROM lote l LEFT JOIN saldo s USING (item_code, mask, warehouse_id)
     WHERE l.q > GREATEST(COALESCE(s.q,0),0) OR l.menor < 0
    UNION ALL
    SELECT 'RESERVADO_X_RESERVAS', s.item_code, s.mask, s.warehouse_id, s.r, COALESCE(v.q,0)
      FROM saldo s LEFT JOIN reserva v USING (item_code, mask, warehouse_id)
     WHERE s.r <> COALESCE(v.q,0))
SELECT a.tipo, a.item_code, COALESCE(NULLIF(i.business_code,''), a.item_code::text), COALESCE(i.name,''), a.mask, a.warehouse_id, a.gravado, a.esperado
  FROM achados a LEFT JOIN items i ON i.code=a.item_code AND i.enterprise_id=$1
 WHERE ($2::bigint IS NULL OR a.warehouse_id=$2) AND ($3::bigint IS NULL OR a.item_code=$3)
 ORDER BY 1, 3, 6`

// ListarInconsistencias lista as divergências do saldo da empresa.
func (r *StockRepositorySQLC) ListarInconsistencias(ctx context.Context, almox, item *int64) ([]entity.InconsistenciaEstoque, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, consultaInconsistencias, empresa, almox, item)
	if err != nil {
		return nil, fmt.Errorf("consultando as inconsistências de estoque: %w", err)
	}
	defer rows.Close()
	out := []entity.InconsistenciaEstoque{}
	for rows.Next() {
		var i entity.InconsistenciaEstoque
		if err := rows.Scan(&i.Tipo, &i.ItemCode, &i.Codigo, &i.Nome, &i.Mask, &i.Almoxarifado, &i.Gravado, &i.Esperado); err != nil {
			return nil, fmt.Errorf("lendo a inconsistência de estoque: %w", err)
		}
		i.Diferenca = i.Esperado.Sub(i.Gravado)
		out = append(out, i)
	}
	return out, rows.Err()
}

// CorrigirInconsistencia refaz o saldo a partir da fonte certa e registra o que
// mudou. Os movimentos são a verdade: nenhum movimento é criado nem alterado.
// Corrige quantidade; o custo segue o médio do item.
func (r *StockRepositorySQLC) CorrigirInconsistencia(ctx context.Context, tipo string, item int64, mask string, almox int64, motivo string, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	achadas, err := r.ListarInconsistencias(ctx, &almox, &item)
	if err != nil {
		return err
	}
	var alvo *entity.InconsistenciaEstoque
	for i := range achadas {
		if achadas[i].Tipo == tipo && achadas[i].Mask == mask {
			alvo = &achadas[i]
		}
	}
	if alvo == nil {
		return errorsuc.NewValidationError("esta inconsistência não existe mais: atualize a lista")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrindo o acerto de estoque: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := conferirPeriodoFechado(ctx, tx, empresa, nil); err != nil {
		return err
	}
	// Trava as linhas: o acerto não pode correr junto com um movimento.
	if _, err := tx.Exec(ctx, `SELECT 1 FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 FOR UPDATE`,
		empresa, item, mask, almox); err != nil {
		return fmt.Errorf("travando o saldo: %w", err)
	}
	switch tipo {
	case entity.InconsistenciaSaldo:
		err = acertarSaldoPelosMovimentos(ctx, tx, empresa, item, mask, almox, alvo.Diferenca)
	case entity.InconsistenciaLote:
		err = acertarLotes(ctx, tx, empresa, item, mask, almox)
	case entity.InconsistenciaReservado:
		err = acertarReservado(ctx, tx, empresa, item, mask, almox)
	default:
		err = errorsuc.NewValidationError("tipo de inconsistência desconhecido")
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO stock_inconsistency_fixes(enterprise_id,kind,item_code,mask,warehouse_id,before_qty,after_qty,reason,fixed_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		empresa, tipo, item, mask, almox, alvo.Gravado, alvo.Esperado, motivo, ator); err != nil {
		return fmt.Errorf("registrando o acerto: %w", err)
	}
	return tx.Commit(ctx)
}

func acertarSaldoPelosMovimentos(ctx context.Context, tx pgx.Tx, empresa, item int64, mask string, almox int64, delta decimal.Decimal) error {
	var linhas []linhaDeSaldo
	rows, err := tx.Query(ctx, `SELECT address, quantity, reserved_qty FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4`,
		empresa, item, mask, almox)
	if err != nil {
		return fmt.Errorf("lendo o saldo: %w", err)
	}
	for rows.Next() {
		var l linhaDeSaldo
		if err := rows.Scan(&l.endereco, &l.quantidade, &l.reservado); err != nil {
			rows.Close()
			return fmt.Errorf("lendo o saldo: %w", err)
		}
		linhas = append(linhas, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("lendo o saldo: %w", err)
	}
	partes := map[string]decimal.Decimal{"": delta}
	if delta.IsNegative() {
		saidas, falta := repartirSaida(linhas, delta.Neg())
		partes = map[string]decimal.Decimal{}
		for e, q := range saidas {
			partes[e] = q.Neg()
		}
		if falta.IsPositive() {
			partes[""] = partes[""].Sub(falta)
		}
	}
	for _, e := range enderecosOrdenados(partes) {
		if _, err := tx.Exec(ctx,
			`INSERT INTO stock_balances(item_code,mask,warehouse_id,quantity,enterprise_id,address) VALUES($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (enterprise_id,item_code,mask,warehouse_id,address) DO UPDATE
			 SET quantity = stock_balances.quantity + EXCLUDED.quantity, updated_at = NOW()`,
			item, mask, almox, partes[e], empresa, e); err != nil {
			return fmt.Errorf("acertando o saldo: %w", err)
		}
	}
	// O valor acompanha a quantidade, pelo custo médio do item.
	if _, err := tx.Exec(ctx,
		`UPDATE stock_balances SET total_cost = CASE WHEN quantity > 0 THEN quantity * avg_cost ELSE 0 END, updated_at = NOW()
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4`, empresa, item, mask, almox); err != nil {
		return fmt.Errorf("acertando o valor do saldo: %w", err)
	}
	return nil
}

func acertarLotes(ctx context.Context, tx pgx.Tx, empresa, item int64, mask string, almox int64) error {
	if _, err := tx.Exec(ctx,
		`UPDATE stock_lot_balances SET quantity = 0, reserved_qty = 0, updated_at = NOW()
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND quantity < 0`, empresa, item, mask, almox); err != nil {
		return fmt.Errorf("zerando lote negativo: %w", err)
	}
	rows, err := tx.Query(ctx,
		`SELECT DISTINCT address FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND quantity > 0`,
		empresa, item, mask, almox)
	if err != nil {
		return fmt.Errorf("lendo os lotes: %w", err)
	}
	var enderecos []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			rows.Close()
			return fmt.Errorf("lendo os lotes: %w", err)
		}
		enderecos = append(enderecos, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("lendo os lotes: %w", err)
	}
	m := &entity.StockMovement{ItemCode: item, Mask: mask, WarehouseID: almox}
	if err := tx.QueryRow(ctx, `SELECT NOW()`).Scan(&m.CreatedAt); err != nil {
		return fmt.Errorf("lendo a hora do acerto: %w", err)
	}
	for _, e := range enderecos {
		if _, err := baixarLotesSemInformar(ctx, tx, empresa, m, e); err != nil {
			return err
		}
	}
	return nil
}

// acertarReservado refaz o reservado do saldo: o que tem endereço volta ao
// endereço; o resto é repartido pelo livre.
func acertarReservado(ctx context.Context, tx pgx.Tx, empresa, item int64, mask string, almox int64) error {
	if _, err := tx.Exec(ctx,
		`UPDATE stock_balances b SET reserved_qty = COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r
		     WHERE r.enterprise_id=b.enterprise_id AND r.item_code=b.item_code AND r.mask=b.mask AND r.warehouse_id=b.warehouse_id
		       AND r.address=b.address AND r.address<>'' AND r.status='ACTIVE'), 0), updated_at = NOW()
		  WHERE b.enterprise_id=$1 AND b.item_code=$2 AND b.mask=$3 AND b.warehouse_id=$4`, empresa, item, mask, almox); err != nil {
		return fmt.Errorf("refazendo o reservado por endereço: %w", err)
	}
	var geral decimal.Decimal
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity),0) FROM stock_reservations
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND status='ACTIVE' AND address=''`,
		empresa, item, mask, almox).Scan(&geral); err != nil {
		return fmt.Errorf("somando as reservas: %w", err)
	}
	if !geral.IsPositive() {
		return nil
	}
	err := adjustReservedTx(ctx, tx, empresa, item, mask, almox, geral)
	if errors.Is(err, repository.ErrInsufficientStock) {
		return errorsuc.NewValidationError("as reservas ativas passam do saldo do item: libere as que não valem mais antes de acertar")
	}
	return err
}

// toleranciaDoInventario devolve a tolerância em percentual de cada item: a do
// inventário e, sem ela, a da classe ABC cadastrada nos parâmetros.
func (r *StockRepositorySQLC) toleranciaDoInventario(ctx context.Context, inv *entity.PhysicalInventory) (func(int64) *decimal.Decimal, error) {
	if inv.TolerancePct != nil {
		return func(int64) *decimal.Decimal { return inv.TolerancePct }, nil
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	param, err := lerParametrosDeEstoque(ctx, r.pool, empresa)
	if err != nil {
		return nil, err
	}
	if !param.TemToleranciaPorClasse() {
		return func(int64) *decimal.Decimal { return nil }, nil
	}
	classes := map[int64]string{}
	rows, err := r.pool.Query(ctx, `SELECT code, COALESCE(planning_abc_class::text,'') FROM items WHERE enterprise_id=$1 AND planning_abc_class IS NOT NULL`, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo as classes ABC: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item int64
		var classe string
		if err := rows.Scan(&item, &classe); err != nil {
			return nil, fmt.Errorf("lendo a classe do item: %w", err)
		}
		classes[item] = classe
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return func(item int64) *decimal.Decimal { return param.ToleranciaDaClasse(classes[item]) }, nil
}
