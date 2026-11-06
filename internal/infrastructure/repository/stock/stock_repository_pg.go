package stock

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// conexao é o que o repositório usa do banco: o pool ou uma transação já
// aberta (pgx.Tx). Dentro de uma transação, o Begin interno vira savepoint, e
// as gravações entram no tudo-ou-nada de quem chamou.
type conexao interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type StockRepositorySQLC struct {
	pool conexao
}

func NewStockRepositorySQLC(pool *pgxpool.Pool) *StockRepositorySQLC {
	return &StockRepositorySQLC{pool: pool}
}

// NewStockRepositoryTx devolve o repositório ligado a uma transação aberta.
func NewStockRepositoryTx(tx pgx.Tx) *StockRepositorySQLC {
	return &StockRepositorySQLC{pool: tx}
}

var _ repository.StockRepository = (*StockRepositorySQLC)(nil)

// SaldoParaSaida devolve o saldo do item no almoxarifado (todos os endereços)
// e se o almoxarifado aceita saldo negativo — o que a prévia da NF-e precisa
// para avisar ANTES da transmissão que a baixa não vai passar.
func (r *StockRepositorySQLC) SaldoParaSaida(ctx context.Context, itemCode int64, warehouseID int64) (decimal.Decimal, bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, false, err
	}
	var saldo decimal.Decimal
	var permite bool
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT SUM(quantity) FROM public.stock_balances
		                   WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3), 0),
		        COALESCE((SELECT allow_negative_stock FROM warehouse WHERE id=$3 AND enterprise_id=$1), FALSE)`,
		enterpriseID, itemCode, warehouseID).Scan(&saldo, &permite)
	if err != nil {
		return decimal.Zero, false, fmt.Errorf("lendo o saldo do item %d: %w", itemCode, err)
	}
	return saldo, permite, nil
}

// ---------- Stock Movements ----------

// CreateMovement records the movement and atomically updates the stock balance
// snapshot (on-hand quantity, weighted average cost and last cost) in the same
// transaction, so balances always reflect the movements that were posted.
func (r *StockRepositorySQLC) CreateMovement(ctx context.Context, m *entity.StockMovement) (*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning stock movement tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = CreateMovementTx(ctx, tx, enterpriseID, m)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing stock movement: %w", err)
	}
	return m, nil
}

// CreateMovementTx records a movement and updates its aggregate balances using
// the caller's transaction. It is exported for infrastructure coordinators
// that must settle production and stock atomically.
func CreateMovementTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if err := TravarItensTx(ctx, tx, enterpriseID, m.ItemCode); err != nil {
		return err
	}
	if err := conferirItemEAlmoxarifado(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	if err := conferirDataDoMovimento(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	if err := barrarSaidaDaInspecao(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	if err := conferirBloqueioDeInventario(ctx, tx, enterpriseID, m.WarehouseID, m.ItemCode, m.ReferenceType); err != nil {
		return err
	}
	entrada := signedDecimalQuantity(m.MovementType, movementQuantity(m)).IsPositive()
	if err := conferirEndereco(ctx, tx, enterpriseID, m.WarehouseID, enderecoDoSaldo(m), entrada, m.ReferenceType); err != nil {
		return err
	}
	if err := conferirRastreioDoItem(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	if err := conferirLoteLiberado(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	if err := valorizarSaidaPelaMedia(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	// Preço sem total: o valor do movimento é quantidade × preço. Sem isto os
	// chamadores internos que mandam só o preço gravavam valor zero — e o
	// movimento sumia do extrato valorizado e do estorno.
	if m.TotalPrice.IsZero() && !m.UnitPrice.IsZero() {
		m.TotalPrice = movementQuantity(m).Abs().Mul(m.UnitPrice).Round(4)
	}
	err := tx.QueryRow(ctx,
		`INSERT INTO public.stock_movements
			(item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
			 reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_by, enterprise_id,
			 address, address_to, movement_type_code, cost_center_code, reverses_movement_id, movement_date, backdated, wave_line_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,
		         COALESCE($22::date, CURRENT_DATE), COALESCE($22::date, CURRENT_DATE) < CURRENT_DATE, $23)
		 RETURNING id, created_at, movement_date`,
		m.ItemCode, m.Mask, m.WarehouseID, m.MovementType, movementQuantity(m), m.UnitPrice, m.TotalPrice,
		m.ReferenceType, m.ReferenceCode, m.Lot, m.SerialNumber, m.Batch, m.ExpirationDate, m.Notes, m.CreatedBy, enterpriseID,
		m.Address, m.AddressTo, m.MovementTypeCode, m.CostCenterCode, m.ReversesMovementID, m.MovementDate, m.LinhaDaOnda,
	).Scan(&m.ID, &m.CreatedAt, &m.MovementDate)
	if err != nil {
		return fmt.Errorf("creating stock movement: %w", err)
	}

	// A saída para a finalidade da demanda independente usa a peça reservada
	// para ela: a reserva sai antes de o saldo baixar.
	if err := consumirReservaDaSaidaTx(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	partes, err := applyMovementToBalance(ctx, tx, enterpriseID, m)
	if err != nil {
		return err
	}
	if err := applyMovementToLotBalance(ctx, tx, enterpriseID, m, partes); err != nil {
		return err
	}
	if m.LinhaDaOnda != nil {
		if _, err := tx.Exec(ctx, `UPDATE stock_picking_wave_lines SET invoiced_qty = invoiced_qty + $3 WHERE id=$1 AND enterprise_id=$2`,
			*m.LinhaDaOnda, enterpriseID, movementQuantity(m)); err != nil {
			return fmt.Errorf("marcando a separação faturada: %w", err)
		}
	}
	if signedDecimalQuantity(m.MovementType, movementQuantity(m)).IsNegative() {
		return reequilibrarReservadoTx(ctx, tx, enterpriseID, m.ItemCode, m.Mask, m.WarehouseID)
	}
	return nil
}

func loteInformado(m *entity.StockMovement) bool {
	return m.Lot != nil && strings.TrimSpace(*m.Lot) != ""
}

// applyMovementToLotBalance keeps the lot-segregated balance in sync, so a
// metallurgy shop can tell how much of each heat remains in each warehouse.
//
// A saída com lote sai das linhas do lote nos endereços que a baixa escolheu. A
// saída sem lote não pode deixar os lotes sobrando: o que passa do saldo do
// endereço é baixado do lote que vence antes, e a observação do movimento conta
// de quais lotes saiu.
func applyMovementToLotBalance(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement, partes map[string]decimal.Decimal) error {
	delta := signedDecimalQuantity(m.MovementType, movementQuantity(m))
	if delta.IsZero() {
		return nil
	}
	if !loteInformado(m) {
		if delta.IsPositive() {
			return nil
		}
		var resumo []string
		for _, endereco := range enderecosOrdenados(partes) {
			r, err := baixarLotesSemInformar(ctx, tx, enterpriseID, m, endereco)
			if err != nil {
				return err
			}
			resumo = append(resumo, r...)
		}
		if len(resumo) == 0 {
			return nil
		}
		nota := "Baixa automática: " + strings.Join(resumo, ", ")
		if _, err := tx.Exec(ctx,
			`UPDATE public.stock_movements SET notes = CASE WHEN COALESCE(notes,'') = '' THEN $2 ELSE notes || ' | ' || $2 END WHERE id=$1`,
			m.ID, nota); err != nil {
			return fmt.Errorf("registrando os lotes baixados: %w", err)
		}
		return nil
	}
	lote := strings.TrimSpace(*m.Lot)
	gravar := func(endereco string, q decimal.Decimal) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO public.stock_lot_balances
				(item_code, mask, warehouse_id, lot, quantity, last_cost, last_movement_at, enterprise_id, address)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, lot, address) WHERE enterprise_id IS NOT NULL DO UPDATE SET
				 quantity = public.stock_lot_balances.quantity + EXCLUDED.quantity,
				 last_cost = CASE WHEN EXCLUDED.last_cost > 0 THEN EXCLUDED.last_cost ELSE public.stock_lot_balances.last_cost END,
				 last_movement_at = EXCLUDED.last_movement_at,
				 updated_at = NOW()`,
			m.ItemCode, m.Mask, m.WarehouseID, lote, q, m.UnitPrice, m.CreatedAt, enterpriseID, endereco)
		if err != nil {
			return fmt.Errorf("updating lot balance: %w", err)
		}
		return nil
	}
	if delta.IsPositive() {
		if err := registrarLoteDaEntrada(ctx, tx, enterpriseID, m, lote); err != nil {
			return err
		}
		return gravar(enderecoDoSaldo(m), delta)
	}
	// Sem saldo em lote no almoxarifado, o lote da saída é só referência do
	// documento (a nota que entrou sem lote e passou pela inspeção, por exemplo):
	// gravar deixaria o lote negativo, e ele nunca teve saldo aqui.
	var temLote bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM public.stock_lot_balances
		                 WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND quantity > 0)`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID).Scan(&temLote); err != nil {
		return fmt.Errorf("conferindo o saldo em lote: %w", err)
	}
	if !temLote {
		return nil
	}
	for _, endereco := range enderecosOrdenados(partes) {
		saiu := partes[endereco]
		// O ajuste de contagem vale sobre a foto: se o lote já saiu depois dela, a
		// diferença fica só no saldo do endereço, sem deixar o lote negativo.
		if m.PermiteSaldoNegativo {
			var noLote decimal.Decimal
			if err := tx.QueryRow(ctx,
				`SELECT COALESCE(SUM(quantity), 0) FROM public.stock_lot_balances
				  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND lot=$5 AND address=$6`,
				enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, lote, endereco).Scan(&noLote); err != nil {
				return fmt.Errorf("lendo o saldo do lote: %w", err)
			}
			saiu = decimal.Min(saiu, decimal.Max(noLote, decimal.Zero))
			if !saiu.IsPositive() {
				continue
			}
		}
		if err := gravar(endereco, saiu.Neg()); err != nil {
			return err
		}
	}
	return nil
}

func enderecosOrdenados(partes map[string]decimal.Decimal) []string {
	out := make([]string, 0, len(partes))
	for e := range partes {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

// applyMovementToBalance atualiza o saldo na transação do movimento e devolve
// quanto saiu de cada endereço (vazio na entrada ou quando não há o que baixar
// no lote).
//
// A QUANTIDADE é por endereço (cada endereço tem a sua linha), mas o CUSTO MÉDIO
// é um só para o item no almoxarifado. Antes cada linha de endereço tinha o seu
// próprio custo: o mesmo tubo valia uma coisa na prateleira A e outra na B, e a
// saída sem endereço lia o custo de uma linha qualquer. Agora o custeio é feito
// sobre o total do almoxarifado e o custo resultante vale para todas as linhas.
//
// É também aqui que a saída é barrada quando deixaria o saldo negativo — no
// almoxarifado, no endereço ou no lote —, salvo onde o almoxarifado permite.
func applyMovementToBalance(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) (map[string]decimal.Decimal, error) {
	deltaExact := signedDecimalQuantity(m.MovementType, movementQuantity(m))
	if deltaExact.IsZero() {
		// Movement type does not affect on-hand quantity (e.g. reservation).
		return nil, nil
	}
	endereco := enderecoDoSaldo(m)

	// Trava todas as linhas do item no almoxarifado: o custo é comum a elas.
	rows, err := tx.Query(ctx,
		`SELECT address, quantity, reserved_qty, avg_cost, total_cost FROM public.stock_balances
		 WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$4 ORDER BY address FOR UPDATE`,
		m.ItemCode, m.Mask, m.WarehouseID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("reading stock balance for update: %w", err)
	}
	var linhas []linhaDeSaldo
	var qtdTotal, qtdPositiva, qtdEndereco, valorTotal, custoMedio decimal.Decimal
	temNegativo := false
	for rows.Next() {
		var l linhaDeSaldo
		var avg, total decimal.Decimal
		if err := rows.Scan(&l.endereco, &l.quantidade, &l.reservado, &avg, &total); err != nil {
			rows.Close()
			return nil, fmt.Errorf("reading stock balance for update: %w", err)
		}
		linhas = append(linhas, l)
		qtdTotal = qtdTotal.Add(l.quantidade)
		valorTotal = valorTotal.Add(total)
		if l.quantidade.IsPositive() {
			qtdPositiva = qtdPositiva.Add(l.quantidade)
		} else if l.quantidade.IsNegative() {
			temNegativo = true
		}
		if avg.GreaterThan(custoMedio) {
			custoMedio = avg
		}
		if l.endereco == endereco {
			qtdEndereco = l.quantidade
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading stock balance for update: %w", err)
	}
	// O valor está só nas linhas positivas: o médio é o delas. Dividir pelo
	// total, descontando uma linha negativa sem valor, inflava o custo.
	if qtdPositiva.IsPositive() {
		custoMedio = valorTotal.DivRound(qtdPositiva, entity.CasasCusto)
	}
	if temNegativo {
		valorTotal = decimal.Max(qtdTotal, decimal.Zero).Mul(custoMedio).Round(entity.CasasValor)
	}

	// Saída sem lote não leva o que está retido em lote vencido ou bloqueado.
	livres, totalLivre, enderecoLivre := linhas, qtdTotal, qtdEndereco
	retido := decimal.Zero
	if deltaExact.IsNegative() && !loteInformado(m) && !saidaLiberada(m) {
		retidos, err := quantidadesRetidas(ctx, tx, enterpriseID, m)
		if err != nil {
			return nil, err
		}
		livres = make([]linhaDeSaldo, len(linhas))
		copy(livres, linhas)
		for i := range livres {
			r := retidos[livres[i].endereco]
			livres[i].quantidade = livres[i].quantidade.Sub(r)
			retido = retido.Add(r)
		}
		totalLivre = qtdTotal.Sub(retido)
		enderecoLivre = qtdEndereco.Sub(retidos[endereco])
	}
	partes, err := partesDoMovimento(ctx, tx, enterpriseID, m, deltaExact, endereco, livres, totalLivre, enderecoLivre)
	if err != nil {
		if retido.IsPositive() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("%s — há %s retido em lote vencido ou bloqueado", err.Error(), entity.QuantidadeBR(retido)))
		}
		return nil, err
	}

	// Weighted-average costing is computed by the domain (single source of truth,
	// unit-tested); the repository only persists the result.
	custeio := entity.ApplyMovementCosting
	if m.SaidaPeloPreco && deltaExact.IsNegative() && m.UnitPrice.IsPositive() {
		custeio = entity.ApplyEntryReversalCosting
	}
	next, lastCost := custeio(
		entity.CostingState{Quantity: qtdTotal, AvgCost: custoMedio, TotalCost: valorTotal}, deltaExact, m.UnitPrice,
	)
	// Entrada sobre saldo negativo: o que ela cobre já saiu. Esse valor pagou uma
	// saída passada e não fica no estoque; vai para o resultado no fechamento.
	if deltaExact.IsPositive() && qtdTotal.IsNegative() {
		coberto := decimal.Min(qtdTotal.Abs(), deltaExact)
		if err := registrarDiferencaDeCusto(ctx, tx, enterpriseID, "SALDO_NEGATIVO", m, coberto.Mul(lastCost).Round(entity.CasasValor)); err != nil {
			return nil, err
		}
	}

	for _, e := range enderecosOrdenados(partes) {
		if _, err = tx.Exec(ctx,
			`INSERT INTO public.stock_balances
				(item_code, mask, warehouse_id, quantity, avg_cost, last_cost, total_cost, last_movement_at, enterprise_id, address)
			 VALUES ($1,$2,$3,$4,$5,$6,0,$7,$8,$9)
			 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, address) DO UPDATE
			 SET quantity = public.stock_balances.quantity + EXCLUDED.quantity,
			     last_movement_at = EXCLUDED.last_movement_at,
			     updated_at = NOW()`,
			m.ItemCode, m.Mask, m.WarehouseID, partes[e], next.AvgCost, lastCost, m.CreatedAt, enterpriseID, e); err != nil {
			return nil, fmt.Errorf("updating stock balance: %w", err)
		}
	}
	// O custo resultante vale para todas as linhas do item no almoxarifado; o
	// valor de cada linha é a sua quantidade pelo custo comum.
	if _, err = tx.Exec(ctx,
		`UPDATE public.stock_balances
		 SET avg_cost = $5, last_cost = $6,
		     total_cost = CASE WHEN quantity > 0 THEN quantity * $5 ELSE 0 END,
		     updated_at = NOW()
		 WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$4`,
		m.ItemCode, m.Mask, m.WarehouseID, enterpriseID, next.AvgCost, lastCost); err != nil {
		return nil, fmt.Errorf("updating stock balance cost: %w", err)
	}
	if deltaExact.IsPositive() {
		return nil, nil
	}
	saiu := make(map[string]decimal.Decimal, len(partes))
	for e, q := range partes {
		saiu[e] = q.Neg()
	}
	return saiu, nil
}

// partesDoMovimento decide em quais linhas de endereço o movimento mexe (valores
// com sinal) e barra a saída sem saldo.
func partesDoMovimento(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement, delta decimal.Decimal,
	endereco string, linhas []linhaDeSaldo, qtdTotal, qtdEndereco decimal.Decimal) (map[string]decimal.Decimal, error) {
	if delta.IsPositive() {
		return map[string]decimal.Decimal{endereco: delta}, nil
	}
	saida := delta.Neg()
	negar := func(partes map[string]decimal.Decimal) map[string]decimal.Decimal {
		out := make(map[string]decimal.Decimal, len(partes))
		for e, q := range partes {
			out[e] = q.Neg()
		}
		return out
	}
	// Com o lote informado, o endereço é onde o lote está.
	if loteInformado(m) {
		porLote, falta, controlado, err := alocarLoteInformado(ctx, tx, enterpriseID, m, endereco, saida)
		if err != nil {
			return nil, err
		}
		if controlado {
			if falta.IsPositive() && !m.PermiteSaldoNegativo && !loteDoDocumento(m) {
				if err := barrarLoteInsuficiente(ctx, tx, enterpriseID, m, saida.Sub(falta), saida); err != nil {
					return nil, err
				}
			}
			if falta.IsPositive() {
				porLote[endereco] = porLote[endereco].Add(falta)
			}
			if err := barrarSeFaltarSaldo(ctx, tx, enterpriseID, m, endereco, saida, qtdTotal, qtdEndereco); err != nil {
				return nil, err
			}
			return negar(porLote), nil
		}
	}
	if endereco != "" {
		if err := barrarSeFaltarSaldo(ctx, tx, enterpriseID, m, endereco, saida, qtdTotal, qtdEndereco); err != nil {
			return nil, err
		}
		return map[string]decimal.Decimal{endereco: delta}, nil
	}
	bloqueados, err := enderecosBloqueados(ctx, tx, enterpriseID, m.WarehouseID)
	if err != nil {
		return nil, err
	}
	for i := range linhas {
		linhas[i].bloqueado = bloqueados[linhas[i].endereco]
	}
	partes, falta := repartirSaida(linhas, saida)
	if falta.IsPositive() {
		if !m.PermiteSaldoNegativo {
			if err := barrarSaldoNegativo(ctx, tx, enterpriseID, m, "", qtdTotal.Sub(somaBloqueada(linhas)), decimal.Zero, saida); err != nil {
				return nil, err
			}
		}
		partes[""] = partes[""].Add(falta)
	}
	return negar(partes), nil
}

func somaBloqueada(linhas []linhaDeSaldo) decimal.Decimal {
	soma := decimal.Zero
	for _, l := range linhas {
		if l.bloqueado {
			soma = soma.Add(decimal.Max(l.quantidade, decimal.Zero))
		}
	}
	return soma
}

func barrarSeFaltarSaldo(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement, endereco string, saida, qtdTotal, qtdEndereco decimal.Decimal) error {
	if m.PermiteSaldoNegativo {
		return nil
	}
	if qtdTotal.Sub(saida).IsNegative() || (endereco != "" && qtdEndereco.Sub(saida).IsNegative()) {
		return barrarSaldoNegativo(ctx, tx, enterpriseID, m, endereco, qtdTotal, qtdEndereco, saida)
	}
	return nil
}

// loteDoDocumento: a decisão de uso move o lote da inspeção mesmo quando a
// entrada veio sem lote (nota de entrada) — ali o lote é o nome do lote de
// inspeção, não uma linha de saldo.
func loteDoDocumento(m *entity.StockMovement) bool {
	return m.ReferenceType != nil && *m.ReferenceType == "QUALITY_LOT"
}

// barrarLoteInsuficiente recusa tirar de um lote mais do que ele tem, salvo no
// almoxarifado que aceita saldo negativo.
func barrarLoteInsuficiente(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement, temNoLote, saida decimal.Decimal) error {
	var permite bool
	if err := tx.QueryRow(ctx, `SELECT COALESCE(allow_negative_stock, FALSE) FROM warehouse WHERE id=$1 AND enterprise_id=$2`, m.WarehouseID, enterpriseID).Scan(&permite); err != nil {
		return fmt.Errorf("conferindo se o almoxarifado permite saldo negativo: %w", err)
	}
	if permite {
		return nil
	}
	return errorsuc.NewValidationError(fmt.Sprintf(
		"saldo insuficiente no lote %s: ele tem %s neste almoxarifado e a saída é de %s — confira o lote ou tire de outro",
		strings.TrimSpace(*m.Lot), entity.QuantidadeBR(temNoLote), entity.QuantidadeBR(saida)))
}

// barrarSaldoNegativo recusa a saída que deixaria o saldo negativo, salvo no
// almoxarifado que permite (warehouse.allow_negative_stock).
func barrarSaldoNegativo(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement, endereco string, saldo, saldoEndereco, saida decimal.Decimal) error {
	var permite bool
	var almox, item string
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(w.allow_negative_stock, FALSE), COALESCE(w.code::text, $2::text),
		        COALESCE((SELECT COALESCE(NULLIF(i.business_code, ''), i.code::text) FROM items i WHERE i.code = $3 AND i.enterprise_id = $1), $3::text)
		   FROM (SELECT 1) x LEFT JOIN warehouse w ON w.id = $2 AND w.enterprise_id = $1`,
		enterpriseID, m.WarehouseID, m.ItemCode).Scan(&permite, &almox, &item)
	if err != nil {
		return fmt.Errorf("conferindo se o almoxarifado permite saldo negativo: %w", err)
	}
	if permite {
		return nil
	}
	if endereco != "" && saldoEndereco.Sub(saida).IsNegative() && !saldo.Sub(saida).IsNegative() {
		return errorsuc.NewValidationError(fmt.Sprintf(
			"saldo insuficiente no endereço %s: o item %s tem %s ali e a saída é de %s — tire de outro endereço ou transfira antes",
			endereco, item, entity.QuantidadeBR(saldoEndereco), entity.QuantidadeBR(saida)))
	}
	return errorsuc.NewValidationError(fmt.Sprintf(
		"saldo insuficiente: o item %s tem %s no almoxarifado %s e a saída é de %s. Lance a entrada antes ou, se a operação exige, permita saldo negativo no cadastro do almoxarifado",
		item, entity.QuantidadeBR(saldo), almox, entity.QuantidadeBR(saida)))
}

// valorizarSaidaPelaMedia preenche o valor de uma saída que veio sem preço.
//
// O saldo já baixa pela média (ApplyMovementCosting consome ao custo médio
// corrente), mas o MOVIMENTO era gravado com o unit_price que o chamador
// mandasse — e as telas de baixa não mandam preço. O extrato e os relatórios de
// valorização ficavam com consumo a zero, embora o estoque tivesse sido baixado
// pelo valor certo. Aqui a saída passa a registrar o mesmo custo que consumiu:
// o custo médio do item no almoxarifado (todos os endereços).
func valorizarSaidaPelaMedia(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	delta := signedDecimalQuantity(m.MovementType, movementQuantity(m))
	semPreco := !m.UnitPrice.IsPositive() && !m.TotalPrice.IsPositive()
	switch {
	case m.ValorizarPelaMedia && delta.IsPositive():
		m.UnitPrice, m.TotalPrice = decimal.Zero, decimal.Zero
	case delta.IsPositive():
		// Sobra de ajuste sem preço entra pelo médio: a custo zero, ela diluía o
		// custo de todo o saldo.
		if m.MovementType != entity.MovementTypeAdjustment || !semPreco {
			return nil
		}
	case delta.IsNegative():
		// Saída baixa o saldo pelo médio, qualquer que seja o preço digitado: o
		// movimento grava o mesmo valor, senão extrato e saldo se separam. Só o
		// desfazer de uma entrada sai pelo preço dela.
		if m.SaidaPeloPreco {
			return nil
		}
	default:
		return nil
	}
	var avgCost decimal.NullDecimal
	err := tx.QueryRow(ctx,
		`SELECT CASE WHEN SUM(quantity) > 0 THEN ROUND(SUM(total_cost) / SUM(quantity), 10) ELSE MAX(avg_cost) END
		   FROM public.stock_balances
		  WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$4`,
		m.ItemCode, m.Mask, m.WarehouseID, enterpriseID).Scan(&avgCost)
	if err != nil {
		return fmt.Errorf("reading average cost to value the outbound movement: %w", err)
	}
	if !avgCost.Valid || !avgCost.Decimal.IsPositive() {
		return nil
	}
	m.UnitPrice = avgCost.Decimal
	m.TotalPrice = movementQuantity(m).Abs().Mul(avgCost.Decimal).Round(4)
	return nil
}

// conferirPeriodoFechado recusa lançamento com data dentro de um período de
// estoque fechado. Data nil = hoje.
func conferirPeriodoFechado(ctx context.Context, tx pgx.Tx, enterpriseID int64, data *time.Time) error {
	var de, ate *time.Time
	err := tx.QueryRow(ctx,
		`SELECT period_from, period_to FROM manufacturing_stock_closed_periods
		  WHERE enterprise_id = $1 AND COALESCE($2::date, CURRENT_DATE) BETWEEN period_from AND period_to
		  ORDER BY period_from LIMIT 1`, enterpriseID, data).Scan(&de, &ate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("conferindo o período de estoque fechado: %w", err)
	}
	return errorsuc.NewValidationError(fmt.Sprintf(
		"o estoque está fechado até %s: use uma data posterior ou reabra o mês", ate.Format("02/01/2006")))
}

// conferirDataDoMovimento: a data do documento precisa cair em período aberto. O
// documento que já existe e veio de um mês fechado entra hoje.
func conferirDataDoMovimento(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	err := conferirPeriodoFechado(ctx, tx, enterpriseID, m.MovementDate)
	if err == nil && m.MovementDate != nil && !m.DataAjustavel {
		return conferirLimiteRetroativo(ctx, tx, enterpriseID, *m.MovementDate)
	}
	if err == nil || !m.DataAjustavel || m.MovementDate == nil {
		return err
	}
	if _, ehValidacao := errorsuc.AsValidation(err); !ehValidacao {
		return err
	}
	m.MovementDate = nil
	return conferirPeriodoFechado(ctx, tx, enterpriseID, nil)
}

// conferirLimiteRetroativo recusa a data anterior ao que a empresa aceita. O
// documento que veio de outro módulo não passa aqui: ele entra hoje se o mês fechou.
func conferirLimiteRetroativo(ctx context.Context, tx pgx.Tx, enterpriseID int64, data time.Time) error {
	var meses *int
	var hoje time.Time
	err := tx.QueryRow(ctx,
		`SELECT (SELECT retroactive_months FROM stock_parameters WHERE enterprise_id=$1), CURRENT_DATE`, enterpriseID).Scan(&meses, &hoje)
	if err != nil {
		return fmt.Errorf("lendo o limite do retroativo: %w", err)
	}
	if meses == nil {
		return nil
	}
	limite := time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -*meses, 0)
	if time.Date(data.Year(), data.Month(), data.Day(), 0, 0, 0, 0, time.UTC).Before(limite) {
		return errorsuc.NewValidationError(fmt.Sprintf("esta empresa aceita lançamento com data a partir de %s", limite.Format("02/01/2006")))
	}
	return nil
}

// enderecoDoSaldo devolve o endereço que identifica a linha de saldo. Vazio
// significa almoxarifado sem endereçamento — o saldo continua sendo um só, como
// antes da migração 344.
func enderecoDoSaldo(m *entity.StockMovement) string {
	if m.Address == nil {
		return ""
	}
	return strings.TrimSpace(*m.Address)
}

func movementQuantity(m *entity.StockMovement) decimal.Decimal {
	return m.Quantity
}

func signedDecimalQuantity(movementType string, quantity decimal.Decimal) decimal.Decimal {
	return entity.SignedQuantity(movementType, quantity)
}

func (r *StockRepositorySQLC) ListMovements(ctx context.Context) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by,
		        movement_type_code, cost_center_code, reverses_movement_id, reversed_by_movement_id, movement_date
		 FROM public.stock_movements WHERE enterprise_id=$1 ORDER BY created_at DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

// ListMovementsPage lista os movimentos por página, com filtros de almoxarifado
// e período. A lista completa cresce sem limite e travaria a tela.
func (r *StockRepositorySQLC) ListMovementsPage(ctx context.Context, f entity.FiltroMovimentos) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	limite := f.Limite
	if limite <= 0 || limite > 2000 {
		limite = 500
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by,
		        movement_type_code, cost_center_code, reverses_movement_id, reversed_by_movement_id, movement_date
		 FROM public.stock_movements
		 WHERE enterprise_id=$1 AND ($2::bigint IS NULL OR warehouse_id=$2)
		   AND ($3::timestamptz IS NULL OR created_at >= $3) AND ($4::timestamptz IS NULL OR created_at < $4)
		 ORDER BY created_at DESC, id DESC LIMIT $5 OFFSET $6`,
		enterpriseID, f.Almoxarifado, f.De, f.Ate, limite, f.Deslocamento)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByItem(ctx context.Context, itemCode int64) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by,
		        movement_type_code, cost_center_code, reverses_movement_id, reversed_by_movement_id, movement_date
		 FROM public.stock_movements WHERE item_code = $1 AND enterprise_id=$2 ORDER BY created_at DESC`, itemCode, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements by item: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by,
		        movement_type_code, cost_center_code, reverses_movement_id, reversed_by_movement_id, movement_date
		 FROM public.stock_movements WHERE warehouse_id = $1 AND enterprise_id=$2 ORDER BY created_at DESC`, warehouseID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements by warehouse: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByDateRange(ctx context.Context, from, to time.Time) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by,
		        movement_type_code, cost_center_code, reverses_movement_id, reversed_by_movement_id, movement_date
		 FROM public.stock_movements WHERE created_at >= $1 AND created_at <= $2 AND enterprise_id=$3 ORDER BY created_at DESC`, from, to, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements by date range: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func scanMovements(rows pgx.Rows) ([]*entity.StockMovement, error) {
	var result []*entity.StockMovement
	for rows.Next() {
		var m entity.StockMovement
		if err := rows.Scan(
			&m.ID, &m.ItemCode, &m.Mask, &m.WarehouseID, &m.MovementType, &m.Quantity, &m.UnitPrice, &m.TotalPrice,
			&m.ReferenceType, &m.ReferenceCode, &m.Lot, &m.SerialNumber, &m.Batch, &m.ExpirationDate, &m.Notes,
			&m.Address, &m.AddressTo, &m.CreatedAt, &m.CreatedBy,
			&m.MovementTypeCode, &m.CostCenterCode, &m.ReversesMovementID, &m.ReversedByMovementID, &m.MovementDate,
		); err != nil {
			return nil, fmt.Errorf("scanning stock movement: %w", err)
		}
		result = append(result, &m)
	}
	return result, rows.Err()
}

// ---------- Stock Balance ----------

// GetBalance devolve o saldo do item no almoxarifado somando os endereços. Cada
// endereço tem a sua linha: ler uma só devolvia o saldo de um endereço qualquer,
// e o pedido reservava menos do que havia.
func (r *StockRepositorySQLC) GetBalance(ctx context.Context, itemCode int64, mask string, warehouseID int64) (*entity.StockBalance, error) {
	enterpriseID, tenantErr := tenant.ID(ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}
	var b entity.StockBalance
	err := r.pool.QueryRow(ctx,
		`SELECT MIN(id), item_code, mask, warehouse_id, SUM(quantity), SUM(reserved_qty), SUM(quantity) - SUM(reserved_qty),
		        MAX(minimum_stock), MAX(maximum_stock), MAX(safety_stock),
		        CASE WHEN SUM(quantity) > 0 THEN ROUND(SUM(total_cost) / SUM(quantity), 10) ELSE MAX(avg_cost) END,
		        MAX(last_cost), SUM(total_cost), MAX(last_movement_at), MAX(updated_at)
		 FROM public.stock_balances WHERE item_code = $1 AND mask = $2 AND warehouse_id = $3 AND enterprise_id=$4
		 GROUP BY item_code, mask, warehouse_id`,
		itemCode, mask, warehouseID, enterpriseID,
	).Scan(&b.ID, &b.ItemCode, &b.Mask, &b.WarehouseID, &b.Quantity, &b.ReservedQty, &b.AvailableQty,
		&b.MinimumStock, &b.MaximumStock, &b.SafetyStock, &b.AvgCost, &b.LastCost, &b.TotalCost,
		&b.LastMovementAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError("não há saldo deste item neste almoxarifado")
		}
		return nil, fmt.Errorf("getting stock balance: %w", err)
	}
	return &b, nil
}

func (r *StockRepositorySQLC) ListBalances(ctx context.Context) ([]*entity.StockBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE enterprise_id=$1 ORDER BY item_code`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *StockRepositorySQLC) ListBalancesByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE warehouse_id = $1 AND enterprise_id=$2 ORDER BY item_code`, warehouseID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances by warehouse: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *StockRepositorySQLC) ListBalancesByItem(ctx context.Context, itemCode int64) ([]*entity.StockBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE item_code = $1 AND enterprise_id=$2 ORDER BY warehouse_id`, itemCode, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances by item: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func scanBalances(rows pgx.Rows) ([]*entity.StockBalance, error) {
	var result []*entity.StockBalance
	for rows.Next() {
		var b entity.StockBalance
		if err := rows.Scan(
			&b.ID, &b.ItemCode, &b.Mask, &b.WarehouseID, &b.Quantity, &b.ReservedQty, &b.AvailableQty,
			&b.MinimumStock, &b.MaximumStock, &b.SafetyStock, &b.AvgCost, &b.LastCost, &b.TotalCost,
			&b.LastMovementAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning stock balance: %w", err)
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}

// ---------- Stock Reservations ----------

// CreateReservation records the reservation and atomically increases the
// reserved quantity of the balance, so available_qty (= quantity − reserved_qty)
// reflects the reservation immediately.
func (r *StockRepositorySQLC) CreateReservation(ctx context.Context, res *entity.StockReservation) (*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning reservation tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := TravarItensTx(ctx, tx, enterpriseID, res.ItemCode); err != nil {
		return nil, err
	}
	if err := conferirAlmoxAceitaReserva(ctx, tx, enterpriseID, res.WarehouseID); err != nil {
		return nil, err
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO public.stock_reservations
			(item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
			 reservation_date, expiration_date, status, notes, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING id, created_at, updated_at`,
		res.ItemCode, res.Mask, res.WarehouseID, res.Quantity, res.ReferenceType, res.ReferenceCode,
		res.ReferenceItemCode, res.ReservationDate, res.ExpirationDate, res.Status, res.Notes, res.CreatedBy, enterpriseID,
	).Scan(&res.ID, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating stock reservation: %w", err)
	}

	if res.Status == "ACTIVE" {
		if err := adjustReservedTx(ctx, tx, enterpriseID, res.ItemCode, res.Mask, res.WarehouseID, res.Quantity); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing reservation: %w", err)
	}
	return res, nil
}

// conferirAlmoxAceitaReserva faz valer o "aceita reservas" do cadastro do
// almoxarifado, que até aqui era gravado e ignorado.
func conferirAlmoxAceitaReserva(ctx context.Context, tx pgx.Tx, enterpriseID, warehouseID int64) error {
	var aceita bool
	err := tx.QueryRow(ctx, `SELECT reservations_allowed FROM warehouse WHERE id=$1 AND enterprise_id=$2`, warehouseID, enterpriseID).Scan(&aceita)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("conferindo se o almoxarifado aceita reservas: %w", err)
	}
	if !aceita {
		return repository.ErrAlmoxSemReserva
	}
	return nil
}

// adjustReservedTx soma delta ao reservado do item no almoxarifado. Cada
// endereço tem a sua linha: o UPDATE direto somava o delta em TODAS elas (reservar
// 5 com dois endereços prendia 10) e exigia que um endereço sozinho cobrisse a
// reserva. Reservar reparte pelo livre de cada linha; liberar devolve primeiro o
// que não é de reserva presa a um endereço (onda de separação).
func adjustReservedTx(ctx context.Context, tx pgx.Tx, enterpriseID, itemCode int64, mask string, warehouseID int64, delta decimal.Decimal) error {
	rows, err := tx.Query(ctx,
		`SELECT b.id, b.quantity, b.reserved_qty,
		        COALESCE((SELECT SUM(r.quantity) FROM public.stock_reservations r
		                   WHERE r.enterprise_id=b.enterprise_id AND r.item_code=b.item_code AND r.mask=b.mask
		                     AND r.warehouse_id=b.warehouse_id AND r.address=b.address AND r.address<>'' AND r.status='ACTIVE'), 0)
		   FROM public.stock_balances b
		  WHERE b.enterprise_id=$1 AND b.item_code=$2 AND b.mask=$3 AND b.warehouse_id=$4
		  ORDER BY b.address FOR UPDATE OF b`,
		enterpriseID, itemCode, mask, warehouseID)
	if err != nil {
		return fmt.Errorf("adjusting reserved quantity: %w", err)
	}
	type linha struct {
		id                   int64
		qtd, reservado, fixa decimal.Decimal
	}
	var linhas []linha
	for rows.Next() {
		var l linha
		if err := rows.Scan(&l.id, &l.qtd, &l.reservado, &l.fixa); err != nil {
			rows.Close()
			return fmt.Errorf("adjusting reserved quantity: %w", err)
		}
		linhas = append(linhas, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("adjusting reserved quantity: %w", err)
	}
	gravar := func(id int64, q decimal.Decimal) error {
		if _, err := tx.Exec(ctx, `UPDATE public.stock_balances SET reserved_qty = GREATEST(reserved_qty + $2, 0), updated_at = NOW() WHERE id = $1`, id, q); err != nil {
			return fmt.Errorf("adjusting reserved quantity: %w", err)
		}
		return nil
	}
	if delta.IsPositive() {
		livre := decimal.Zero
		for _, l := range linhas {
			livre = livre.Add(decimal.Max(l.qtd.Sub(l.reservado), decimal.Zero))
		}
		if livre.LessThan(delta) {
			return repository.ErrInsufficientStock
		}
		resta := delta
		for _, l := range linhas {
			q := decimal.Min(decimal.Max(l.qtd.Sub(l.reservado), decimal.Zero), resta)
			if !q.IsPositive() {
				continue
			}
			if err := gravar(l.id, q); err != nil {
				return err
			}
			resta = resta.Sub(q)
		}
		return nil
	}
	resta := delta.Neg()
	for _, soLivre := range []bool{true, false} {
		for i := range linhas {
			l := &linhas[i]
			teto := l.reservado
			if soLivre {
				teto = decimal.Max(l.reservado.Sub(l.fixa), decimal.Zero)
			}
			q := decimal.Min(teto, resta)
			if !q.IsPositive() {
				continue
			}
			if err := gravar(l.id, q.Neg()); err != nil {
				return err
			}
			l.reservado = l.reservado.Sub(q)
			resta = resta.Sub(q)
		}
	}
	return nil
}

func (r *StockRepositorySQLC) HasActiveReservationByReference(ctx context.Context, referenceType string, referenceCode int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM public.stock_reservations
			WHERE reference_type = $1 AND reference_code = $2 AND status = 'ACTIVE' AND enterprise_id=$3
		 )`, referenceType, referenceCode, enterpriseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking active reservations: %w", err)
	}
	return exists, nil
}

func (r *StockRepositorySQLC) GetReservation(ctx context.Context, id int64) (*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var res entity.StockReservation
	err = r.pool.QueryRow(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE id = $1 AND enterprise_id=$2`, id, enterpriseID,
	).Scan(&res.ID, &res.ItemCode, &res.Mask, &res.WarehouseID, &res.Quantity, &res.ReferenceType,
		&res.ReferenceCode, &res.ReferenceItemCode, &res.ReservationDate, &res.ExpirationDate,
		&res.Status, &res.Notes, &res.CreatedAt, &res.UpdatedAt, &res.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("reserva de estoque %d não encontrada", id))
		}
		return nil, fmt.Errorf("getting stock reservation: %w", err)
	}
	return &res, nil
}

func (r *StockRepositorySQLC) ListReservations(ctx context.Context) ([]*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE enterprise_id=$1 ORDER BY created_at DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing reservations: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) ListReservationsByItem(ctx context.Context, itemCode int64) ([]*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE item_code = $1 AND enterprise_id=$2 ORDER BY created_at DESC`, itemCode, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing reservations by item: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) ListActiveReservations(ctx context.Context) ([]*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE status = 'ACTIVE' AND enterprise_id=$1 ORDER BY created_at DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing active reservations: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) CancelReservation(ctx context.Context, id int64) error {
	return r.closeReservation(ctx, id, "CANCELLED")
}

func (r *StockRepositorySQLC) ConsumeReservation(ctx context.Context, id int64) error {
	return r.closeReservation(ctx, id, "CONSUMED")
}

// closeReservation moves a reservation to a terminal status and, if it was still
// ACTIVE, releases its quantity from the balance's reserved_qty so available_qty
// is restored. No-op on the reserved_qty if the reservation was already closed.
func (r *StockRepositorySQLC) closeReservation(ctx context.Context, id int64, status string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning reservation close tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := travarItemDaReservaTx(ctx, tx, enterpriseID, id); err != nil {
		return err
	}
	var itemCode, warehouseID int64
	var mask, prevStatus string
	var qty decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT item_code, mask, warehouse_id, quantity, status
		 FROM public.stock_reservations WHERE id = $1 AND enterprise_id=$2 FOR UPDATE`, id, enterpriseID,
	).Scan(&itemCode, &mask, &warehouseID, &qty, &prevStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errorsuc.NewNotFoundError(fmt.Sprintf("reserva de estoque %d não encontrada", id))
		}
		return fmt.Errorf("reading reservation %d: %w", id, err)
	}

	// Reserva encerrada não muda de situação: consumida que vira cancelada some
	// da conferência do pedido sem nada ter voltado ao estoque.
	if prevStatus != "ACTIVE" {
		if prevStatus == status {
			return nil
		}
		return errorsuc.NewValidationError(fmt.Sprintf("a reserva %d já está encerrada e não pode mudar de situação", id))
	}

	if _, err = tx.Exec(ctx,
		`UPDATE public.stock_reservations SET status = $2, updated_at = NOW() WHERE id = $1 AND enterprise_id=$3`, id, status, enterpriseID); err != nil {
		return fmt.Errorf("closing reservation %d: %w", id, err)
	}

	if err := adjustReservedTx(ctx, tx, enterpriseID, itemCode, mask, warehouseID, qty.Neg()); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing reservation close: %w", err)
	}
	return nil
}

// ConsumirReservaParcial consome até qtd de uma reserva ativa. Com a quantidade
// inteira ela encerra como consumida; senão só diminui, e o resto segue reservado.
func (r *StockRepositorySQLC) ConsumirReservaParcial(ctx context.Context, id int64, qtd decimal.Decimal) (decimal.Decimal, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return decimal.Zero, fmt.Errorf("beginning reservation consume tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := travarItemDaReservaTx(ctx, tx, enterpriseID, id); err != nil {
		return decimal.Zero, err
	}
	var itemCode, warehouseID int64
	var mask, status string
	var reservado decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT item_code, mask, warehouse_id, quantity, status FROM public.stock_reservations WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`,
		id, enterpriseID).Scan(&itemCode, &mask, &warehouseID, &reservado, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, errorsuc.NewNotFoundError(fmt.Sprintf("reserva de estoque %d não encontrada", id))
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("reading reservation %d: %w", id, err)
	}
	if status != "ACTIVE" || !qtd.IsPositive() {
		return decimal.Zero, nil
	}
	consumido := decimal.Min(qtd, reservado)
	if consumido.Equal(reservado) {
		_, err = tx.Exec(ctx, `UPDATE public.stock_reservations SET status='CONSUMED', updated_at=NOW() WHERE id=$1`, id)
	} else {
		_, err = tx.Exec(ctx, `UPDATE public.stock_reservations SET quantity = quantity - $2, updated_at=NOW() WHERE id=$1`, id, consumido)
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("consuming reservation %d: %w", id, err)
	}
	if err := adjustReservedTx(ctx, tx, enterpriseID, itemCode, mask, warehouseID, consumido.Neg()); err != nil {
		return decimal.Zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return decimal.Zero, fmt.Errorf("committing reservation consume: %w", err)
	}
	return consumido, nil
}

// travarItemDaReservaTx trava o item antes da linha da reserva, na mesma ordem
// da saída de estoque.
func travarItemDaReservaTx(ctx context.Context, tx pgx.Tx, enterpriseID, id int64) error {
	var item int64
	err := tx.QueryRow(ctx, `SELECT item_code FROM public.stock_reservations WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID).Scan(&item)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lendo o item da reserva: %w", err)
	}
	return TravarItensTx(ctx, tx, enterpriseID, item)
}

func scanReservations(rows pgx.Rows) ([]*entity.StockReservation, error) {
	var result []*entity.StockReservation
	for rows.Next() {
		var res entity.StockReservation
		if err := rows.Scan(
			&res.ID, &res.ItemCode, &res.Mask, &res.WarehouseID, &res.Quantity, &res.ReferenceType,
			&res.ReferenceCode, &res.ReferenceItemCode, &res.ReservationDate, &res.ExpirationDate,
			&res.Status, &res.Notes, &res.CreatedAt, &res.UpdatedAt, &res.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning reservation: %w", err)
		}
		result = append(result, &res)
	}
	return result, rows.Err()
}

// ---------- Consumption Average ----------

// RecalcConsumptionAverage computes the average monthly consumption of an item
// from its outbound movements over the trailing window and upserts the result.
// consumoSQL: a saída que é CONSUMO. Transferência não é consumo (o material só
// mudou de lugar) e contava como tal; estorno e movimento estornado se anulam;
// e o tipo de movimento marcado como "não considera consumo" fica de fora.
const consumoSQL = `m.movement_type IN ('OUT','SAIDA','REP')
		   AND m.reverses_movement_id IS NULL AND m.reversed_by_movement_id IS NULL
		   -- Desfazer uma entrada (nota ou recebimento cancelado), devolver ao
		   -- fornecedor, sucatear na inspeção, perder no transporte e entregar a outro
		   -- estabelecimento tiram do saldo sem ser consumo.
		   AND COALESCE(m.reference_type,'') NOT IN ('NF_ENTRADA','PURCHASE_ORDER','QUALITY_LOT','TRANSITO','TRANSF_CNPJ')
		   AND NOT (m.reference_type = 'NF_SAIDA' AND EXISTS (SELECT 1 FROM fiscal_return_settlements frs
		                                                       WHERE frs.enterprise_id = m.enterprise_id AND frs.fiscal_exit_id = m.reference_code))
		   AND NOT EXISTS (SELECT 1 FROM stock_movement_types t
		                    WHERE t.enterprise_id = m.enterprise_id AND t.sigla = m.movement_type_code
		                      AND NOT t.considers_consumption)`

func (r *StockRepositorySQLC) RecalcConsumptionAverage(ctx context.Context, itemCode int64, windowMonths int) (*entity.ItemConsumptionAverage, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if windowMonths <= 0 {
		windowMonths = 6
	}

	var totalConsumed decimal.Decimal
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(ABS(m.quantity)), 0)
		 FROM public.stock_movements m
		 WHERE m.item_code = $1
		   AND `+consumoSQL+`
		   AND m.created_at >= NOW() - make_interval(months => $2)
		   AND m.enterprise_id=$3`, itemCode, windowMonths, enterpriseID).Scan(&totalConsumed)
	if err != nil {
		return nil, fmt.Errorf("summing item consumption: %w", err)
	}

	avg := totalConsumed.DivRound(decimal.NewFromInt(int64(windowMonths)), 6)

	var out entity.ItemConsumptionAverage
	err = r.pool.QueryRow(ctx,
		`INSERT INTO public.item_consumption_averages
			(item_code, avg_monthly_consumption, total_consumed, window_months, calculated_at, enterprise_id)
		 VALUES ($1,$2,$3,$4,NOW(),$5)
		 ON CONFLICT (enterprise_id,item_code) DO UPDATE SET
			 avg_monthly_consumption = EXCLUDED.avg_monthly_consumption,
			 total_consumed          = EXCLUDED.total_consumed,
			 window_months           = EXCLUDED.window_months,
			 calculated_at           = NOW()
		 RETURNING id, item_code, avg_monthly_consumption, total_consumed, window_months, calculated_at`,
		itemCode, avg, totalConsumed, windowMonths, enterpriseID,
	).Scan(&out.ID, &out.ItemCode, &out.AvgMonthlyConsumption, &out.TotalConsumed, &out.WindowMonths, &out.CalculatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting consumption average: %w", err)
	}
	return &out, nil
}

// RecalcAllConsumptionAverages recomputes the average for every item that had
// outbound movements within the window, returning how many items were updated.
func (r *StockRepositorySQLC) RecalcAllConsumptionAverages(ctx context.Context, windowMonths int) (int, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	if windowMonths <= 0 {
		windowMonths = 6
	}
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT m.item_code FROM public.stock_movements m
		 WHERE `+consumoSQL+`
		   AND m.created_at >= NOW() - make_interval(months => $1) AND m.enterprise_id=$2`, windowMonths, enterpriseID)
	if err != nil {
		return 0, fmt.Errorf("listing items with consumption: %w", err)
	}
	var items []int64
	for rows.Next() {
		var code int64
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scanning item code: %w", err)
		}
		items = append(items, code)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	count := 0
	for _, code := range items {
		if _, err := r.RecalcConsumptionAverage(ctx, code, windowMonths); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (r *StockRepositorySQLC) GetConsumptionAverage(ctx context.Context, itemCode int64) (*entity.ItemConsumptionAverage, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var out entity.ItemConsumptionAverage
	err = r.pool.QueryRow(ctx,
		`SELECT id, item_code, avg_monthly_consumption, total_consumed, window_months, calculated_at
		 FROM public.item_consumption_averages WHERE item_code = $1 AND enterprise_id=$2`, itemCode, enterpriseID,
	).Scan(&out.ID, &out.ItemCode, &out.AvgMonthlyConsumption, &out.TotalConsumed, &out.WindowMonths, &out.CalculatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError("este item ainda não tem consumo médio apurado")
		}
		return nil, fmt.Errorf("getting consumption average: %w", err)
	}
	return &out, nil
}

// ---------- Lot Traceability ----------

func (r *StockRepositorySQLC) UpsertLot(ctx context.Context, lot *entity.StockLot) (*entity.StockLot, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO public.stock_lots
			(item_code, mask, lot, heat_number, certificate, supplier_code, received_at, expires_at, notes, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT (enterprise_id,item_code,mask,lot) DO UPDATE SET
			 heat_number   = EXCLUDED.heat_number,
			 certificate   = EXCLUDED.certificate,
			 supplier_code = EXCLUDED.supplier_code,
			 received_at   = EXCLUDED.received_at,
			 expires_at    = EXCLUDED.expires_at,
			 notes         = EXCLUDED.notes
		 RETURNING id, created_at`,
		lot.ItemCode, lot.Mask, lot.Lot, lot.HeatNumber, lot.Certificate, lot.SupplierCode, lot.ReceivedAt, lot.ExpiresAt, lot.Notes, lot.CreatedBy, enterpriseID,
	).Scan(&lot.ID, &lot.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting stock lot: %w", err)
	}
	return lot, nil
}

func (r *StockRepositorySQLC) GetLot(ctx context.Context, itemCode int64, lot string) (*entity.StockLot, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var l entity.StockLot
	err = r.pool.QueryRow(ctx,
		`SELECT id, item_code, mask, lot, heat_number, certificate, supplier_code, received_at, expires_at, notes, created_at, created_by
		 FROM public.stock_lots WHERE item_code = $1 AND lot = $2 AND enterprise_id=$3 ORDER BY mask LIMIT 1`, itemCode, lot, enterpriseID,
	).Scan(&l.ID, &l.ItemCode, &l.Mask, &l.Lot, &l.HeatNumber, &l.Certificate, &l.SupplierCode, &l.ReceivedAt, &l.ExpiresAt, &l.Notes, &l.CreatedAt, &l.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting stock lot: %w", err)
	}
	return &l, nil
}

func (r *StockRepositorySQLC) ListLotBalancesByItem(ctx context.Context, itemCode int64) ([]*entity.StockLotBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT b.id, b.item_code, b.mask, b.warehouse_id, b.lot, b.quantity, b.last_cost, b.last_movement_at, b.updated_at,
		        l.status_code, l.grade, l.expires_at
		 FROM public.stock_lot_balances b
		 LEFT JOIN public.stock_lots l ON l.enterprise_id=b.enterprise_id AND l.item_code=b.item_code AND l.mask=b.mask AND l.lot=b.lot
		 WHERE b.item_code = $1 AND b.enterprise_id=$2 ORDER BY b.lot, b.warehouse_id`, itemCode, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing lot balances: %w", err)
	}
	defer rows.Close()
	return scanLotBalances(rows)
}

func scanLotBalances(rows pgx.Rows) ([]*entity.StockLotBalance, error) {
	var out []*entity.StockLotBalance
	for rows.Next() {
		var b entity.StockLotBalance
		if err := rows.Scan(&b.ID, &b.ItemCode, &b.Mask, &b.WarehouseID, &b.Lot, &b.Quantity, &b.LastCost, &b.LastMovementAt, &b.UpdatedAt,
			&b.StatusCode, &b.Grade, &b.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scanning lot balance: %w", err)
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

// GetLotGenealogy traces an item lot in both directions: the production orders
// that consumed it (where this raw material went) and the production orders that
// produced it together with the input lots that compose it.
func (r *StockRepositorySQLC) GetLotGenealogy(ctx context.Context, itemCode int64, lot string) (*entity.LotGenealogy, error) {
	enterpriseID, tenantErr := tenant.ID(ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}
	g := &entity.LotGenealogy{ItemCode: itemCode, Lot: lot}

	registry, err := r.GetLot(ctx, itemCode, lot)
	if err != nil {
		return nil, err
	}
	g.Registry = registry

	balRows, err := r.pool.Query(ctx,
		`SELECT b.id, b.item_code, b.mask, b.warehouse_id, b.lot, b.quantity, b.last_cost, b.last_movement_at, b.updated_at,
		        l.status_code, l.grade, l.expires_at
		 FROM public.stock_lot_balances b
		 LEFT JOIN public.stock_lots l ON l.enterprise_id=b.enterprise_id AND l.item_code=b.item_code AND l.mask=b.mask AND l.lot=b.lot
		 WHERE b.item_code = $1 AND b.lot = $2 AND b.enterprise_id=$3 ORDER BY b.warehouse_id`, itemCode, lot, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing lot genealogy balances: %w", err)
	}
	balances, err := scanLotBalances(balRows)
	balRows.Close()
	if err != nil {
		return nil, err
	}
	g.Balances = balances

	// Forward: production orders that consumed this lot.
	consRows, err := r.pool.Query(ctx,
		`SELECT pc.production_order_id, po.order_number, po.item_code, pc.consumed_qty
		 FROM public.production_consumptions pc
		 JOIN public.production_orders po ON po.id = pc.production_order_id
		 WHERE pc.item_code = $1 AND pc.lot = $2 AND po.enterprise_id=$3
		 ORDER BY pc.production_order_id`, itemCode, lot, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing lot consumptions: %w", err)
	}
	for consRows.Next() {
		var c entity.LotConsumption
		if err := consRows.Scan(&c.ProductionOrderID, &c.OrderNumber, &c.ProducedItemCode, &c.ConsumedQty); err != nil {
			consRows.Close()
			return nil, fmt.Errorf("scanning lot consumption: %w", err)
		}
		g.ConsumedIn = append(g.ConsumedIn, c)
	}
	consRows.Close()

	// Backward: production orders that produced this lot (recorded on the IN
	// movement) and the input lots that went into each of them.
	prodRows, err := r.pool.Query(ctx,
		`SELECT sm.reference_code, po.order_number, COALESCE(SUM(sm.quantity), 0)
		 FROM public.stock_movements sm
		 JOIN public.production_orders po ON po.id = sm.reference_code
		 WHERE sm.item_code = $1 AND sm.lot = $2
		   AND sm.reference_type = 'PRODUCTION_ORDER' AND sm.movement_type IN ('IN','EP','EPP','EPE')
		   AND sm.reversed_by_movement_id IS NULL
		   AND sm.enterprise_id=$3 AND po.enterprise_id=$3
		 GROUP BY sm.reference_code, po.order_number
		 ORDER BY sm.reference_code`, itemCode, lot, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing lot productions: %w", err)
	}
	var productions []entity.LotProduction
	for prodRows.Next() {
		var p entity.LotProduction
		if err := prodRows.Scan(&p.ProductionOrderID, &p.OrderNumber, &p.ProducedQty); err != nil {
			prodRows.Close()
			return nil, fmt.Errorf("scanning lot production: %w", err)
		}
		productions = append(productions, p)
	}
	prodRows.Close()

	for i := range productions {
		inputRows, err := r.pool.Query(ctx,
			`SELECT pc.item_code, COALESCE(pc.lot, ''), pc.consumed_qty
			 FROM public.production_consumptions pc
			 JOIN public.production_orders po ON po.id=pc.production_order_id
			 WHERE pc.production_order_id = $1 AND pc.lot IS NOT NULL AND pc.lot <> '' AND po.enterprise_id=$2
			 ORDER BY pc.item_code`, productions[i].ProductionOrderID, enterpriseID)
		if err != nil {
			return nil, fmt.Errorf("listing lot inputs: %w", err)
		}
		for inputRows.Next() {
			var in entity.LotInput
			if err := inputRows.Scan(&in.ItemCode, &in.Lot, &in.ConsumedQty); err != nil {
				inputRows.Close()
				return nil, fmt.Errorf("scanning lot input: %w", err)
			}
			productions[i].InputLots = append(productions[i].InputLots, in)
		}
		inputRows.Close()
	}
	g.ProducedBy = productions

	return g, nil
}

// ---------- Physical Inventory ----------

// CreateInventory abre o inventário e grava a FOTO do saldo de cada item do
// almoxarifado (quantidade somada dos endereços e custo médio) — é contra ela
// que a contagem é comparada. Um almoxarifado não tem dois inventários abertos.
func (r *StockRepositorySQLC) CreateInventory(ctx context.Context, inv *entity.PhysicalInventory) (*entity.PhysicalInventory, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning inventory tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var aberto int64
	err = tx.QueryRow(ctx,
		`SELECT code FROM public.physical_inventories
		  WHERE enterprise_id=$1 AND warehouse_id=$2 AND status IN ('OPEN','EM_CONFERENCIA','AGUARDANDO_APROVACAO') LIMIT 1`, enterpriseID, inv.WarehouseID).Scan(&aberto)
	if err == nil {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("o almoxarifado já tem o inventário %d em andamento: feche-o antes de abrir outro", aberto))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("checking open inventory: %w", err)
	}

	var filtro []int64
	if len(inv.ItemCodes) > 0 {
		filtro = inv.ItemCodes
	}
	inv.Partial = len(filtro) > 0
	if inv.ApprovalMode == "" {
		inv.ApprovalMode = entity.AprovacaoNunca
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO public.physical_inventories
			(code, description, warehouse_id, start_date, end_date, status, notes, created_by, enterprise_id,
			 blocks_movements, blind_count, partial, approval_mode, tolerance_pct, tolerance_value)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 RETURNING id, created_at, updated_at`,
		inv.Code, inv.Description, inv.WarehouseID, inv.StartDate, inv.EndDate, inv.Status, inv.Notes, inv.CreatedBy, enterpriseID,
		inv.BlocksMovements, inv.BlindCount, inv.Partial, inv.ApprovalMode, inv.TolerancePct, inv.ToleranceValue,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating physical inventory: %w", err)
	}

	// A foto é por endereço e lote: cada lote do endereço é uma linha, e o que o
	// endereço tem além dos lotes é a linha sem lote. O custo é o médio do item.
	tag, err := tx.Exec(ctx,
		`WITH custo AS (
		     SELECT item_code, mask,
		            CASE WHEN SUM(quantity) > 0 THEN SUM(total_cost) / SUM(quantity) ELSE MAX(avg_cost) END AS unit_cost
		       FROM public.stock_balances
		      WHERE enterprise_id = $2 AND warehouse_id = $3 GROUP BY item_code, mask),
		  linhas AS (
		     SELECT item_code, mask, address, lot, quantity FROM public.stock_lot_balances
		      WHERE enterprise_id = $2 AND warehouse_id = $3 AND quantity <> 0
		     UNION ALL
		     SELECT s.item_code, s.mask, s.address, '', s.quantity - COALESCE(x.total, 0)
		       FROM public.stock_balances s
		       LEFT JOIN LATERAL (SELECT SUM(l.quantity) total FROM public.stock_lot_balances l
		                           WHERE l.enterprise_id = s.enterprise_id AND l.item_code = s.item_code AND l.mask = s.mask
		                             AND l.warehouse_id = s.warehouse_id AND l.address = s.address) x ON TRUE
		      WHERE s.enterprise_id = $2 AND s.warehouse_id = $3 AND s.quantity - COALESCE(x.total, 0) <> 0)
		 INSERT INTO public.physical_inventory_items (inventory_id, item_code, mask, warehouse_id, address, lot, system_qty, unit_cost)
		 SELECT $1, l.item_code, l.mask, $3, l.address, l.lot, l.quantity, c.unit_cost
		   FROM linhas l JOIN custo c ON c.item_code = l.item_code AND c.mask = l.mask
		  WHERE ($4::bigint[] IS NULL OR l.item_code = ANY($4::bigint[]))`,
		inv.ID, enterpriseID, inv.WarehouseID, filtro)
	if err != nil {
		return nil, fmt.Errorf("snapshotting inventory balances: %w", err)
	}
	// Inventário parcial: item pedido sem saldo também entra (foto zero), para
	// a contagem dele aparecer na lista.
	if len(filtro) > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO public.physical_inventory_items (inventory_id, item_code, mask, warehouse_id, system_qty, unit_cost)
			 SELECT $1, i.code, '', $3, 0, 0 FROM items i
			  WHERE i.enterprise_id = $2 AND i.code = ANY($4::bigint[])
			    AND NOT EXISTS (SELECT 1 FROM public.physical_inventory_items x WHERE x.inventory_id = $1 AND x.item_code = i.code)`,
			inv.ID, enterpriseID, inv.WarehouseID, filtro); err != nil {
			return nil, fmt.Errorf("adding requested items to the inventory: %w", err)
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE public.physical_inventories
		    SET total_items = (SELECT COUNT(*) FROM public.physical_inventory_items WHERE inventory_id = $1)
		  WHERE id = $1`, inv.ID); err != nil {
		return nil, fmt.Errorf("counting inventory items: %w", err)
	}
	inv.TotalItems = int(tag.RowsAffected())
	if err := tx.QueryRow(ctx, `SELECT total_items FROM public.physical_inventories WHERE id=$1`, inv.ID).Scan(&inv.TotalItems); err != nil {
		return nil, fmt.Errorf("counting inventory items: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing inventory: %w", err)
	}
	return inv, nil
}

const colunasInventario = `id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by, closed_by, adjustment_value,
		        blocks_movements, blind_count, partial, approval_mode, tolerance_pct, tolerance_value, close_reason, zero_uncounted`

func camposInventario(i *entity.PhysicalInventory) []any {
	return []any{&i.ID, &i.Code, &i.Description, &i.WarehouseID, &i.StartDate, &i.EndDate, &i.Status,
		&i.TotalItems, &i.CountedItems, &i.Notes, &i.CreatedAt, &i.UpdatedAt, &i.CreatedBy, &i.ClosedBy, &i.AdjustmentValue,
		&i.BlocksMovements, &i.BlindCount, &i.Partial, &i.ApprovalMode, &i.TolerancePct, &i.ToleranceValue, &i.CloseReason, &i.ZeroUncounted}
}

func (r *StockRepositorySQLC) GetInventory(ctx context.Context, id int64) (*entity.PhysicalInventory, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var inv entity.PhysicalInventory
	err = r.pool.QueryRow(ctx,
		`SELECT `+colunasInventario+`
		 FROM public.physical_inventories WHERE id = $1 AND enterprise_id=$2`, id, enterpriseID,
	).Scan(camposInventario(&inv)...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("inventário físico %d não encontrado", id))
		}
		return nil, fmt.Errorf("getting physical inventory: %w", err)
	}
	return &inv, nil
}

func (r *StockRepositorySQLC) GetInventoryByCode(ctx context.Context, code int64) (*entity.PhysicalInventory, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var inv entity.PhysicalInventory
	err = r.pool.QueryRow(ctx,
		`SELECT `+colunasInventario+`
		 FROM public.physical_inventories WHERE code = $1 AND enterprise_id=$2`, code, enterpriseID,
	).Scan(camposInventario(&inv)...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("inventário físico %d não encontrado", code))
		}
		return nil, fmt.Errorf("getting physical inventory by code: %w", err)
	}
	return &inv, nil
}

func (r *StockRepositorySQLC) ListInventories(ctx context.Context) ([]*entity.PhysicalInventory, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+colunasInventario+`
		 FROM public.physical_inventories WHERE enterprise_id=$1 ORDER BY created_at DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing physical inventories: %w", err)
	}
	defer rows.Close()
	return scanInventories(rows)
}

func (r *StockRepositorySQLC) ListInventoriesByStatus(ctx context.Context, status string) ([]*entity.PhysicalInventory, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+colunasInventario+`
		 FROM public.physical_inventories WHERE status = $1 AND enterprise_id=$2 ORDER BY created_at DESC`, status, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing physical inventories by status: %w", err)
	}
	defer rows.Close()
	return scanInventories(rows)
}

func (r *StockRepositorySQLC) UpdateInventoryStatus(ctx context.Context, id int64, status string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE public.physical_inventories SET status = $1, updated_at = NOW() WHERE id = $2 AND enterprise_id=$3`, status, id, enterpriseID)
	if err != nil {
		return fmt.Errorf("updating inventory status: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) CloseInventory(ctx context.Context, id int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE public.physical_inventories SET status = 'CLOSED', end_date = CURRENT_DATE, updated_at = NOW() WHERE id = $1 AND enterprise_id=$2`, id, enterpriseID)
	if err != nil {
		return fmt.Errorf("closing inventory: %w", err)
	}
	return nil
}

// FecharInventario lança o ajuste de cada diferença e fecha o inventário.
//
// O ajuste é a DIFERENÇA contra a foto da abertura (contado − sistema na
// abertura), aplicada ao saldo atual: o que entrou ou saiu depois da abertura
// continua valendo. A sobra entra ao custo médio do item; a falta sai ao custo
// médio. Tudo numa transação — ou o inventário fecha com todos os ajustes, ou
// nada muda.
func (r *StockRepositorySQLC) FecharInventario(ctx context.Context, id int64, ator uuid.UUID, zerarNaoContados bool, motivo string) (*entity.ResultadoFechamentoInventario, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning inventory close tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var codigo, almox int64
	var status string
	var dataDaFoto time.Time
	err = tx.QueryRow(ctx,
		`SELECT code, warehouse_id, status, start_date FROM public.physical_inventories WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`,
		id, enterpriseID).Scan(&codigo, &almox, &status, &dataDaFoto)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("inventário %d não encontrado", id))
	}
	if err != nil {
		return nil, fmt.Errorf("locking inventory: %w", err)
	}
	if !entity.InventarioEmAndamento(status) {
		return nil, errorsuc.NewValidationError("somente inventário em andamento pode ser fechado")
	}

	res := &entity.ResultadoFechamentoInventario{}
	if zerarNaoContados {
		tag, err := tx.Exec(ctx,
			`UPDATE public.physical_inventory_items
			    SET counted_qty = 0, difference_qty = -system_qty, counted_at = NOW(), counted_by = $2,
			        adjustment_reason = COALESCE(adjustment_reason, 'não contado: zerado no fechamento')
			  WHERE inventory_id = $1 AND counted_qty IS NULL`, id, ator)
		if err != nil {
			return nil, fmt.Errorf("zeroing uncounted items: %w", err)
		}
		res.NaoContadosZerados = int(tag.RowsAffected())
	}
	var faltando int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM public.physical_inventory_items WHERE inventory_id=$1 AND counted_qty IS NULL`, id).Scan(&faltando); err != nil {
		return nil, fmt.Errorf("checking uncounted items: %w", err)
	}
	if faltando > 0 {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"%d item(ns) ainda sem contagem: conte-os ou feche zerando os não contados", faltando))
	}

	rows, err := tx.Query(ctx,
		`SELECT id, item_code, mask, address, lot, difference_qty, COALESCE(unit_cost, 0)
		   FROM public.physical_inventory_items
		  WHERE inventory_id=$1 AND difference_qty IS NOT NULL AND difference_qty <> 0
		  ORDER BY item_code`, id)
	if err != nil {
		return nil, fmt.Errorf("listing inventory differences: %w", err)
	}
	type diferenca struct {
		id, item             int64
		mask, endereco, lote string
		qtd                  decimal.Decimal
		custo                decimal.Decimal
	}
	var difs []diferenca
	for rows.Next() {
		var d diferenca
		if err := rows.Scan(&d.id, &d.item, &d.mask, &d.endereco, &d.lote, &d.qtd, &d.custo); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning inventory difference: %w", err)
		}
		difs = append(difs, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	ref := entity.ReferenceTypeInventory
	nota := fmt.Sprintf("Ajuste do inventário %d", codigo)
	if m := strings.TrimSpace(motivo); m != "" {
		nota += ": " + m
	}
	for _, d := range difs {
		mov := &entity.StockMovement{
			ItemCode: d.item, Mask: d.mask, WarehouseID: almox,
			MovementType: entity.MovementTypeAdjustment, Quantity: d.qtd,
			ReferenceType: &ref, ReferenceCode: &codigo, Notes: &nota, CreatedBy: ator,
			PermiteSaldoNegativo: true, MovementDate: &dataDaFoto, DataAjustavel: true,
		}
		if d.endereco != "" {
			endereco := d.endereco
			mov.Address = &endereco
		}
		if d.lote != "" {
			lote := d.lote
			mov.Lot = &lote
		}
		if d.qtd.IsPositive() {
			// Sobra entra ao custo médio atual (o da foto, se o item zerou).
			custo := d.custo
			var atual decimal.NullDecimal
			if err := tx.QueryRow(ctx,
				`SELECT CASE WHEN SUM(quantity) > 0 THEN ROUND(SUM(total_cost)/SUM(quantity), 10) ELSE MAX(avg_cost) END
				   FROM public.stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4`,
				enterpriseID, d.item, d.mask, almox).Scan(&atual); err != nil {
				return nil, fmt.Errorf("reading average cost for the inventory adjustment: %w", err)
			}
			if atual.Valid && atual.Decimal.IsPositive() {
				custo = atual.Decimal
			}
			mov.UnitPrice, mov.TotalPrice = custo, d.qtd.Mul(custo).Round(4)
			res.Sobras = res.Sobras.Add(d.qtd)
		} else {
			res.Faltas = res.Faltas.Add(d.qtd.Neg())
		}
		if err := CreateMovementTx(ctx, tx, enterpriseID, mov); err != nil {
			return nil, fmt.Errorf("lançando o ajuste do item %d: %w", d.item, err)
		}
		if d.qtd.IsPositive() {
			res.ValorAjustado = res.ValorAjustado.Add(mov.TotalPrice)
		} else {
			res.ValorAjustado = res.ValorAjustado.Sub(mov.TotalPrice)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE public.physical_inventory_items
			    SET is_adjusted = TRUE, adjustment_movement_id = $2,
			        adjustment_type = CASE WHEN difference_qty > 0 THEN 'IN' ELSE 'OUT' END
			  WHERE id = $1`, d.id, mov.ID); err != nil {
			return nil, fmt.Errorf("linking the adjustment to the inventory item: %w", err)
		}
		res.Ajustes++
	}
	if _, err := tx.Exec(ctx,
		`UPDATE public.physical_inventories
		    SET status='CLOSED', end_date=CURRENT_DATE, closed_by=$2, adjustment_value=$3,
		        close_reason=COALESCE(NULLIF($4,''), close_reason), updated_at=NOW()
		  WHERE id=$1`, id, ator, res.ValorAjustado, strings.TrimSpace(motivo)); err != nil {
		return nil, fmt.Errorf("closing inventory: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing inventory close: %w", err)
	}
	return res, nil
}

func scanInventories(rows pgx.Rows) ([]*entity.PhysicalInventory, error) {
	var result []*entity.PhysicalInventory
	for rows.Next() {
		var inv entity.PhysicalInventory
		if err := rows.Scan(camposInventario(&inv)...); err != nil {
			return nil, fmt.Errorf("scanning physical inventory: %w", err)
		}
		result = append(result, &inv)
	}
	return result, rows.Err()
}

// ---------- Physical Inventory Items ----------

func (r *StockRepositorySQLC) UpsertInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var existingID int64
	err = r.pool.QueryRow(ctx,
		`SELECT item.id FROM public.physical_inventory_items item
		 JOIN physical_inventories inventory ON inventory.id=item.inventory_id
		 WHERE item.inventory_id=$1 AND item.item_code=$2 AND item.mask=$3 AND item.warehouse_id=$4 AND inventory.enterprise_id=$5`,
		item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, enterpriseID,
	).Scan(&existingID)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("checking existing inventory item: %w", err)
	}
	if err == pgx.ErrNoRows {
		_, err = r.pool.Exec(ctx,
			`INSERT INTO public.physical_inventory_items
				(inventory_id, item_code, mask, warehouse_id, system_qty)
			 SELECT $1,$2,$3,$4,$5 WHERE EXISTS (SELECT 1 FROM physical_inventories WHERE id=$1 AND enterprise_id=$6)`,
			item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, item.SystemQty, enterpriseID)
	} else {
		_, err = r.pool.Exec(ctx,
			`UPDATE public.physical_inventory_items SET system_qty = $1 WHERE id = $2`,
			item.SystemQty, existingID)
	}
	if err != nil {
		return fmt.Errorf("upserting inventory item: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) ListInventoryItems(ctx context.Context, inventoryID int64) ([]*entity.PhysicalInventoryItem, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, inventory_id, item_code, mask, warehouse_id, system_qty, counted_qty,
		        difference_qty, unit_cost, adjustment_type, adjustment_reason, counted_by,
		        counted_at, is_adjusted, created_at, count_number, previous_counted_qty, adjustment_movement_id,
		        address, lot, needs_recount, notebook_number
		 FROM public.physical_inventory_items item WHERE inventory_id=$1 AND EXISTS
		 (SELECT 1 FROM physical_inventories inventory WHERE inventory.id=item.inventory_id AND inventory.enterprise_id=$2)
		 ORDER BY address, item_code, lot`, inventoryID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing inventory items: %w", err)
	}
	defer rows.Close()
	return scanInventoryItems(rows)
}

// CountInventoryItem grava a contagem e calcula a diferença contra a foto da
// abertura. Item que não estava na foto (achado no almoxarifado) entra com saldo
// do sistema zero. Contar de novo é recontagem: a anterior fica guardada.
//
// Antes era um UPDATE numa linha que nunca era criada — não gravava nada e
// respondia sucesso.
func (r *StockRepositorySQLC) CountInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning inventory count tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var existe bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2)`, item.ItemCode, enterpriseID).Scan(&existe); err != nil {
		return fmt.Errorf("checking inventory item: %w", err)
	}
	if !existe {
		return errorsuc.NewNotFoundError("item não encontrado nesta empresa")
	}
	tag, err := tx.Exec(ctx,
		`INSERT INTO public.physical_inventory_items
			(inventory_id, item_code, mask, warehouse_id, address, lot, system_qty, unit_cost,
			 counted_qty, difference_qty, counted_by, counted_at, count_number)
		 SELECT $1::bigint, $2::bigint, $3::text, $4::bigint, $8::text, $9::text, 0,
		        COALESCE((SELECT CASE WHEN SUM(b.quantity) > 0 THEN SUM(b.total_cost)/SUM(b.quantity) ELSE MAX(b.avg_cost) END
		                    FROM public.stock_balances b
		                   WHERE b.enterprise_id=$7 AND b.item_code=$2::bigint AND b.mask=$3::text AND b.warehouse_id=$4::bigint), 0),
		        $5::numeric, $5::numeric, $6::uuid, NOW(), 1
		  WHERE EXISTS (SELECT 1 FROM public.physical_inventories inv WHERE inv.id=$1 AND inv.enterprise_id=$7 AND inv.status='OPEN')
		 ON CONFLICT (inventory_id, item_code, mask, warehouse_id, address, lot) DO UPDATE SET
		     previous_counted_qty = public.physical_inventory_items.counted_qty,
		     needs_recount  = FALSE,
		     counted_qty    = EXCLUDED.counted_qty,
		     difference_qty = EXCLUDED.counted_qty - public.physical_inventory_items.system_qty,
		     counted_by     = EXCLUDED.counted_by,
		     counted_at     = NOW(),
		     count_number   = public.physical_inventory_items.count_number + 1,
		     is_adjusted    = FALSE`,
		item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, item.CountedQty, item.CountedBy, enterpriseID, item.Address, item.Lot)
	if err != nil {
		return fmt.Errorf("counting inventory item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("somente inventário aberto desta empresa pode receber contagem")
	}
	if item.AdjustmentReason != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE public.physical_inventory_items SET adjustment_reason=$5
			  WHERE inventory_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$6 AND lot=$7`,
			item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, item.AdjustmentReason, item.Address, item.Lot); err != nil {
			return fmt.Errorf("recording count reason: %w", err)
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE public.physical_inventories SET
		     total_items   = (SELECT COUNT(*) FROM public.physical_inventory_items WHERE inventory_id=$1),
		     counted_items = (SELECT COUNT(*) FROM public.physical_inventory_items WHERE inventory_id=$1 AND counted_qty IS NOT NULL),
		     updated_at = NOW()
		  WHERE id=$1`, item.InventoryID); err != nil {
		return fmt.Errorf("updating inventory progress: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *StockRepositorySQLC) AdjustInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE public.physical_inventory_items SET
		     adjustment_type = $1, adjustment_reason = $2
		 WHERE inventory_id=$3 AND item_code=$4 AND mask=$5 AND warehouse_id=$6 AND EXISTS
		 (SELECT 1 FROM physical_inventories inventory WHERE inventory.id=$3 AND inventory.enterprise_id=$7)`,
		item.AdjustmentType, item.AdjustmentReason, item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, enterpriseID)
	if err != nil {
		return fmt.Errorf("adjusting inventory item: %w", err)
	}
	return nil
}

func scanInventoryItems(rows pgx.Rows) ([]*entity.PhysicalInventoryItem, error) {
	var result []*entity.PhysicalInventoryItem
	for rows.Next() {
		var item entity.PhysicalInventoryItem
		if err := rows.Scan(
			&item.ID, &item.InventoryID, &item.ItemCode, &item.Mask, &item.WarehouseID, &item.SystemQty,
			&item.CountedQty, &item.DifferenceQty, &item.UnitCost, &item.AdjustmentType, &item.AdjustmentReason,
			&item.CountedBy, &item.CountedAt, &item.IsAdjusted, &item.CreatedAt,
			&item.CountNumber, &item.PreviousCountedQty, &item.AdjustmentMovementID,
			&item.Address, &item.Lot, &item.NeedsRecount, &item.Notebook,
		); err != nil {
			return nil, fmt.Errorf("scanning inventory item: %w", err)
		}
		result = append(result, &item)
	}
	return result, rows.Err()
}

// SugerirSeparacaoFEFO devolve de quais lotes e endereços tirar a quantidade
// pedida, na ordem da regra escolhida.
//
// FEFO ("first expired, first out") ordena por validade: o que vence antes sai
// antes. É o que a norma sanitária e a maioria das indústrias exige, e o que o
// sistema não tinha — havia apenas FIFO, e só dentro do plano de corte. Lote sem
// validade cai para o fim e desempata pela data de recebimento, que é o FIFO.
//
// Lote vencido nunca é sugerido: ele é contado à parte para que a tela possa
// avisar em vez de fingir que o saldo está disponível.
func (r *StockRepositorySQLC) SugerirSeparacaoFEFO(
	ctx context.Context, itemCode int64, mask string, warehouseID int64, necessario decimal.Decimal, regra string,
) (*entity.ResultadoFEFO, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ordem := `l.expires_at IS NULL, l.expires_at, l.received_at NULLS LAST, b.lot`
	if strings.EqualFold(regra, "FIFO") {
		regra = "FIFO"
		ordem = `l.received_at NULLS LAST, l.expires_at IS NULL, l.expires_at, b.lot`
	} else {
		regra = "FEFO"
	}

	// O livre do lote é limitado pelo livre do endereço: reserva de pedido fica na
	// linha do endereço, não na do lote. Sem isso a separação prometia peça
	// reservada. A segunda parte traz o saldo que não é de lote nenhum — item sem
	// lote também é separado. Inspeção, rejeição e almoxarifado indisponível não
	// entregam para separação.
	linhas, err := r.pool.Query(ctx, `
		WITH saldo AS (
			SELECT b.lot, b.address, b.warehouse_id,
			       LEAST(b.quantity - b.reserved_qty, s.quantity - s.reserved_qty) AS disponivel,
			       l.heat_number, l.certificate, l.expires_at, l.received_at,
			       `+loteSemSaidaSQL+` AS retido
			  FROM public.stock_lot_balances b
			  JOIN public.stock_balances s
			         ON s.enterprise_id = b.enterprise_id AND s.item_code = b.item_code AND s.mask = b.mask
			        AND s.warehouse_id = b.warehouse_id AND s.address = b.address
			  LEFT JOIN public.stock_lots l
			         ON l.enterprise_id = b.enterprise_id AND l.item_code = b.item_code
			        AND l.mask = b.mask AND l.lot = b.lot
			 WHERE b.enterprise_id = $1 AND b.item_code = $2 AND b.mask = $3
			   AND ($4::bigint = 0 OR b.warehouse_id = $4::bigint)
			UNION ALL
			SELECT ''::varchar, s.address, s.warehouse_id,
			       s.quantity - s.reserved_qty - COALESCE((SELECT SUM(x.quantity) FROM public.stock_lot_balances x
			            WHERE x.enterprise_id = s.enterprise_id AND x.item_code = s.item_code AND x.mask = s.mask
			              AND x.warehouse_id = s.warehouse_id AND x.address = s.address AND x.quantity > 0), 0),
			       NULL, NULL, NULL, NULL, FALSE
			  FROM public.stock_balances s
			 WHERE s.enterprise_id = $1 AND s.item_code = $2 AND s.mask = $3
			   AND ($4::bigint = 0 OR s.warehouse_id = $4::bigint)
		)
		SELECT b.lot, b.address, b.warehouse_id, b.disponivel,
		       b.heat_number, b.certificate, b.expires_at, b.received_at,
		       COALESCE(b.expires_at < CURRENT_DATE, false) AS vencido,
		       COALESCE(a.zone, ''), COALESCE(a.pick_sequence, 0),
		       COALESCE(a.is_blocked, false) OR b.retido AS bloqueado
		  FROM saldo b
		  JOIN public.warehouse w ON w.id = b.warehouse_id AND w.enterprise_id = $1
		  LEFT JOIN public.manufacturing_warehouse_addresses a
		         ON a.enterprise_id = $1 AND a.warehouse_id = b.warehouse_id AND a.address = b.address
		 WHERE b.disponivel > 0 AND `+almoxPrometivelSQL+`
		 ORDER BY `+strings.ReplaceAll(ordem, "l.", "b."),
		enterpriseID, itemCode, mask, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("consultando lotes para separação: %w", err)
	}
	defer linhas.Close()

	resultado := &entity.ResultadoFEFO{
		ItemCode: itemCode, Mask: mask, Regra: regra, Necessario: necessario.InexactFloat64(),
		Linhas: []entity.SugestaoFEFO{},
	}
	restante := necessario
	for linhas.Next() {
		var s entity.SugestaoFEFO
		var vencido, bloqueado bool
		var disponivel decimal.Decimal
		if err := linhas.Scan(&s.Lot, &s.Address, &s.WarehouseID, &disponivel,
			&s.HeatNumber, &s.Certificate, &s.ExpiresAt, &s.ReceivedAt, &vencido,
			&s.Zone, &s.PickSequence, &bloqueado); err != nil {
			return nil, fmt.Errorf("lendo lote para separação: %w", err)
		}
		s.Disponivel = disponivel.InexactFloat64()
		if vencido {
			resultado.VencidosFora++
			continue
		}
		// Endereço bloqueado (inventário, avaria, quarentena) não entra na
		// separação: o saldo existe, mas não pode sair.
		if bloqueado {
			resultado.BloqueadosFora++
			continue
		}
		if !restante.IsPositive() {
			continue
		}
		sugerido := decimal.Min(disponivel, restante)
		s.Sugerido = sugerido.InexactFloat64()
		restante = restante.Sub(sugerido)
		resultado.Linhas = append(resultado.Linhas, s)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("lendo lotes para separação: %w", err)
	}
	resultado.Atendido = necessario.Sub(restante).InexactFloat64()
	if restante.IsPositive() {
		resultado.EmFalta = restante.InexactFloat64()
	}
	// O FEFO decide QUAIS lotes; a lista é entregue na ordem de caminhada, que é
	// o que o separador segue no galpão. Sem sequência cadastrada (0) a ordem
	// vira a do endereço, que é melhor que a ordem de validade para caminhar.
	sort.SliceStable(resultado.Linhas, func(i, j int) bool {
		a, b := resultado.Linhas[i], resultado.Linhas[j]
		if a.PickSequence != b.PickSequence {
			if a.PickSequence == 0 || b.PickSequence == 0 {
				return b.PickSequence == 0
			}
			return a.PickSequence < b.PickSequence
		}
		return a.Address < b.Address
	})
	return resultado, nil
}

// ListarSaldoPorEndereco devolve o saldo quebrado por endereço de um
// almoxarifado. É a consulta que o Focco expõe na FEST0332; aqui ela também
// traz o lote, para que a conferência física não precise de uma segunda tela.
func (r *StockRepositorySQLC) ListarSaldoPorEndereco(
	ctx context.Context, warehouseID int64, itemCode int64,
) ([]*entity.StockLotBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// A segunda parte é o saldo do endereço que não pertence a lote nenhum: lendo
	// só os lotes, item sem controle de lote não aparecia no endereço onde está.
	linhas, err := r.pool.Query(ctx, `
		SELECT id, item_code, mask, warehouse_id, lot, address, quantity, last_cost, last_movement_at, updated_at
		  FROM public.stock_lot_balances
		 WHERE enterprise_id = $1
		   AND ($2::bigint = 0 OR warehouse_id = $2::bigint)
		   AND ($3::bigint = 0 OR item_code = $3::bigint)
		   AND quantity <> 0
		UNION ALL
		SELECT s.id, s.item_code, s.mask, s.warehouse_id, ''::varchar, s.address,
		       s.quantity - COALESCE(x.total, 0), s.last_cost, s.last_movement_at, s.updated_at
		  FROM public.stock_balances s
		  LEFT JOIN LATERAL (SELECT SUM(l.quantity) total FROM public.stock_lot_balances l
		                      WHERE l.enterprise_id = s.enterprise_id AND l.item_code = s.item_code AND l.mask = s.mask
		                        AND l.warehouse_id = s.warehouse_id AND l.address = s.address) x ON TRUE
		 WHERE s.enterprise_id = $1
		   AND ($2::bigint = 0 OR s.warehouse_id = $2::bigint)
		   AND ($3::bigint = 0 OR s.item_code = $3::bigint)
		   AND s.quantity - COALESCE(x.total, 0) <> 0
		 ORDER BY 4, 6, 2, 5`,
		enterpriseID, warehouseID, itemCode)
	if err != nil {
		return nil, fmt.Errorf("consultando saldo por endereço: %w", err)
	}
	defer linhas.Close()
	var saldos []*entity.StockLotBalance
	for linhas.Next() {
		var b entity.StockLotBalance
		if err := linhas.Scan(&b.ID, &b.ItemCode, &b.Mask, &b.WarehouseID, &b.Lot, &b.Address,
			&b.Quantity, &b.LastCost, &b.LastMovementAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("lendo saldo por endereço: %w", err)
		}
		saldos = append(saldos, &b)
	}
	return saldos, linhas.Err()
}

// TransferirEntreEnderecos move quantidade de um endereço para outro dentro do
// MESMO almoxarifado, numa única transação.
//
// É um movimento só, com origem e destino, e não dois espelhados: dois
// movimentos dobrariam a quantidade nos relatórios de giro e fariam a mesma peça
// aparecer duas vezes no extrato. O saldo do almoxarifado não muda — muda de
// endereço —, por isso a baixa e o crédito são aplicados aqui explicitamente em
// vez de passarem pela apuração normal de saldo.
func (r *StockRepositorySQLC) TransferirEntreEnderecos(
	ctx context.Context, m *entity.StockMovement, origem, destino string,
) (*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	quantidade := movementQuantity(m)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo transação da transferência: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := TravarItensTx(ctx, tx, enterpriseID, m.ItemCode); err != nil {
		return nil, err
	}
	if err := conferirPeriodoFechado(ctx, tx, enterpriseID, nil); err != nil {
		return nil, err
	}
	if err := conferirBloqueioDeInventario(ctx, tx, enterpriseID, m.WarehouseID, m.ItemCode, nil); err != nil {
		return nil, err
	}
	if err := conferirEndereco(ctx, tx, enterpriseID, m.WarehouseID, origem, false, nil); err != nil {
		return nil, err
	}
	if err := conferirEndereco(ctx, tx, enterpriseID, m.WarehouseID, destino, true, nil); err != nil {
		return nil, err
	}

	// A origem precisa cobrir a quantidade com o que está LIVRE: o reservado fica
	// preso ao endereço e a onda de separação iria buscá-lo no lugar errado.
	var disponivel decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT quantity - reserved_qty FROM public.stock_balances
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5 FOR UPDATE`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem).Scan(&disponivel)
	if err == pgx.ErrNoRows {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("não há saldo do item no endereço %s", origem))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo saldo da origem: %w", err)
	}
	if disponivel.LessThan(quantidade) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"o endereço %s tem %s livre e a transferência pede %s", origem, entity.QuantidadeBR(disponivel), entity.QuantidadeBR(quantidade)))
	}
	if err := conferirRastreioDoItem(ctx, tx, enterpriseID, m); err != nil {
		return nil, err
	}
	if !loteInformado(m) {
		// Sem o lote, só anda o que está fora de lote: o resto ficaria com o lote
		// no endereço antigo e o material no novo.
		var emLote, saldo decimal.Decimal
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE((SELECT SUM(l.quantity) FROM public.stock_lot_balances l
			                   WHERE l.enterprise_id=$1 AND l.item_code=$2 AND l.mask=$3 AND l.warehouse_id=$4 AND l.address=$5 AND l.quantity > 0), 0),
			        COALESCE((SELECT s.quantity FROM public.stock_balances s
			                   WHERE s.enterprise_id=$1 AND s.item_code=$2 AND s.mask=$3 AND s.warehouse_id=$4 AND s.address=$5), 0)`,
			enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem).Scan(&emLote, &saldo); err != nil {
			return nil, fmt.Errorf("lendo os lotes da origem: %w", err)
		}
		if semLote := decimal.Max(saldo.Sub(emLote), decimal.Zero); emLote.IsPositive() && semLote.LessThan(quantidade) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"o endereço %s tem %s sem lote e a transferência pede %s: informe o lote a transferir", origem, entity.QuantidadeBR(semLote), entity.QuantidadeBR(quantidade)))
		}
	}
	if loteInformado(m) {
		saida := *m
		saida.MovementType = entity.MovementTypeTransferOut
		if err := conferirLoteLiberado(ctx, tx, enterpriseID, &saida); err != nil {
			return nil, err
		}
		var noLote decimal.Decimal
		err := tx.QueryRow(ctx,
			`SELECT quantity - reserved_qty FROM public.stock_lot_balances
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5 AND lot=$6 FOR UPDATE`,
			enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem, strings.TrimSpace(*m.Lot)).Scan(&noLote)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("lendo saldo do lote na origem: %w", err)
		}
		if noLote.LessThan(quantidade) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"o lote %s tem %s livre no endereço %s e a transferência pede %s", strings.TrimSpace(*m.Lot), entity.QuantidadeBR(noLote), origem, entity.QuantidadeBR(quantidade)))
		}
	}

	var custoMedio, ultimoCusto decimal.Decimal
	if err := tx.QueryRow(ctx,
		`SELECT avg_cost, last_cost FROM public.stock_balances
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem).Scan(&custoMedio, &ultimoCusto); err != nil {
		return nil, fmt.Errorf("lendo o custo do saldo de origem: %w", err)
	}
	valorMovido := quantidade.Mul(custoMedio).Round(4)

	err = tx.QueryRow(ctx,
		`INSERT INTO public.stock_movements
			(item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
			 reference_type, reference_code, lot, notes, created_by, enterprise_id, address, address_to)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 RETURNING id, created_at`,
		m.ItemCode, m.Mask, m.WarehouseID, entity.MovementTypeAddressTransfer, quantidade,
		custoMedio, valorMovido, m.ReferenceType, m.ReferenceCode, m.Lot, m.Notes, m.CreatedBy,
		enterpriseID, origem, destino,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("gravando a transferência: %w", err)
	}

	// Baixa na origem, mantendo o custo médio (o material é o mesmo).
	if _, err = tx.Exec(ctx,
		`UPDATE public.stock_balances
		    SET quantity = quantity - $6, total_cost = GREATEST(quantity - $6, 0) * avg_cost, last_movement_at = $7, updated_at = NOW()
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem, quantidade, m.CreatedAt); err != nil {
		return nil, fmt.Errorf("baixando o endereço de origem: %w", err)
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO public.stock_balances
			(item_code, mask, warehouse_id, quantity, avg_cost, last_cost, total_cost, last_movement_at, enterprise_id, address)
		 VALUES ($1,$2,$3,$4,$5,$6,$4::numeric * $5::numeric,$7,$8,$9)
		 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, address) DO UPDATE
		 SET quantity = public.stock_balances.quantity + EXCLUDED.quantity,
		     total_cost = GREATEST(public.stock_balances.quantity + EXCLUDED.quantity, 0) * EXCLUDED.avg_cost,
		     avg_cost = EXCLUDED.avg_cost,
		     last_movement_at = EXCLUDED.last_movement_at, updated_at = NOW()`,
		m.ItemCode, m.Mask, m.WarehouseID, quantidade, custoMedio, ultimoCusto, m.CreatedAt, enterpriseID, destino); err != nil {
		return nil, fmt.Errorf("creditando o endereço de destino: %w", err)
	}

	// Lote acompanha o material: sem isso a rastreabilidade apontaria o endereço
	// antigo depois da transferência.
	if m.Lot != nil && *m.Lot != "" {
		if _, err = tx.Exec(ctx,
			`UPDATE public.stock_lot_balances SET quantity = quantity - $7, updated_at = NOW()
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5 AND lot=$6`,
			enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem, *m.Lot, quantidade); err != nil {
			return nil, fmt.Errorf("baixando o lote na origem: %w", err)
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO public.stock_lot_balances
				(item_code, mask, warehouse_id, lot, quantity, last_cost, last_movement_at, enterprise_id, address)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, lot, address) WHERE enterprise_id IS NOT NULL DO UPDATE
			 SET quantity = public.stock_lot_balances.quantity + EXCLUDED.quantity,
			     last_movement_at = EXCLUDED.last_movement_at, updated_at = NOW()`,
			m.ItemCode, m.Mask, m.WarehouseID, *m.Lot, quantidade, custoMedio, m.CreatedAt, enterpriseID, destino); err != nil {
			return nil, fmt.Errorf("creditando o lote no destino: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmando a transferência: %w", err)
	}
	return m, nil
}

// ApurarCurvaABC classifica os itens pelo VALOR consumido na janela, não pela
// quantidade: mil parafusos baratos não são um item A.
//
// Pareto clássico — A até 80% do valor acumulado, B até 95%, C o resto. Grava em
// items.planning_abc_class (a coluna que o resto do sistema já lê) junto do valor
// e da participação, para a tela poder explicar a classe em vez de exibir só a
// letra. Antes a classe era digitada à mão e envelhecia sem ninguém notar.
func (r *StockRepositorySQLC) ApurarCurvaABC(ctx context.Context, janelaMeses int, corteA, corteB float64) (*entity.ResumoABC, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if janelaMeses <= 0 {
		janelaMeses = 12
	}
	if corteA <= 0 || corteA >= 100 {
		corteA = 80
	}
	if corteB <= corteA || corteB >= 100 {
		corteB = 95
	}

	linhas, err := r.pool.Query(ctx, `
		WITH consumo AS (
			SELECT m.item_code,
			       SUM(ABS(m.quantity) * COALESCE(NULLIF(m.unit_price,0), b.avg_cost, 0)) AS valor
			  FROM stock_movements m
			  LEFT JOIN LATERAL (
			       SELECT AVG(avg_cost) avg_cost FROM stock_balances s
			        WHERE s.enterprise_id = m.enterprise_id AND s.item_code = m.item_code
			  ) b ON TRUE
			 WHERE m.enterprise_id = $1
			   AND m.created_at >= NOW() - make_interval(months => $2)
			   AND `+consumoSQL+`
			 GROUP BY m.item_code
			HAVING SUM(ABS(m.quantity) * COALESCE(NULLIF(m.unit_price,0), b.avg_cost, 0)) > 0
		), total AS (
			SELECT COALESCE(SUM(valor),0) t FROM consumo
		)
		SELECT c.item_code, c.valor,
		       100 * c.valor / NULLIF(t.t,0) AS participacao,
		       100 * SUM(c.valor) OVER (ORDER BY c.valor DESC, c.item_code) / NULLIF(t.t,0) AS acumulado,
		       t.t
		  FROM consumo c CROSS JOIN total t
		 ORDER BY c.valor DESC, c.item_code`, enterpriseID, janelaMeses)
	if err != nil {
		return nil, fmt.Errorf("apurando consumo para a curva ABC: %w", err)
	}
	defer linhas.Close()

	resumo := &entity.ResumoABC{JanelaMeses: janelaMeses, CorteA: corteA, CorteB: corteB, Itens: []entity.ClasseABC{}}
	for linhas.Next() {
		var it entity.ClasseABC
		if err := linhas.Scan(&it.ItemCode, &it.ValorConsumido, &it.ParticipacaoPct, &it.AcumuladoPct, &resumo.ValorTotal); err != nil {
			return nil, fmt.Errorf("lendo consumo do item: %w", err)
		}
		it.Classe = entity.ClassificarABC(it.AcumuladoPct, it.ParticipacaoPct, corteA, corteB)
		resumo.Itens = append(resumo.Itens, it)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("lendo a curva ABC: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo a gravação da curva ABC: %w", err)
	}
	defer tx.Rollback(ctx)
	apurados := make([]int64, 0, len(resumo.Itens))
	for _, it := range resumo.Itens {
		if _, err := tx.Exec(ctx, `
			UPDATE items SET planning_abc_class = $3, abc_consumption_value = $4,
			                 abc_share_pct = $5, abc_calculated_at = NOW()
			 WHERE enterprise_id = $1 AND code = $2`,
			enterpriseID, it.ItemCode, it.Classe, it.ValorConsumido, it.ParticipacaoPct); err != nil {
			return nil, fmt.Errorf("gravando a classe ABC do item %d: %w", it.ItemCode, err)
		}
		apurados = append(apurados, it.ItemCode)
		resumo.Classificados++
	}
	// Quem parou de consumir cai para C: seguir como A mantém a contagem cíclica
	// frequente num item parado.
	if _, err := tx.Exec(ctx, `
		UPDATE items SET planning_abc_class = 'C', abc_consumption_value = 0, abc_share_pct = 0, abc_calculated_at = NOW()
		 WHERE enterprise_id = $1 AND planning_abc_class IS NOT NULL AND planning_abc_class <> ''
		   AND NOT (code = ANY($2::bigint[]))`, enterpriseID, apurados); err != nil {
		return nil, fmt.Errorf("reclassificando os itens sem consumo: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("gravando a curva ABC: %w", err)
	}
	return resumo, nil
}

// SugerirEnderecoDeGuarda recomenda onde guardar o material que chegou.
//
// A ordem das estratégias é a que reduz deslocamento e fragmentação, e é a mesma
// que SAP e Focco usam como padrão:
//  1. endereço fixo do item (o "fixed bin") — se existe, é ele e acabou;
//  2. consolidação: endereço que JÁ tem o mesmo item, para não espalhar o saldo;
//  3. endereço vazio da zona pedida, na ordem da rota de separação;
//  4. qualquer endereço ativo com espaço.
//
// Endereço bloqueado nunca é sugerido, e a capacidade declarada é respeitada —
// sugerir um endereço que não cabe é pior que não sugerir nada.
func (r *StockRepositorySQLC) SugerirEnderecoDeGuarda(
	ctx context.Context, itemCode int64, mask string, warehouseID int64, quantidade float64, zona string,
) ([]*entity.SugestaoGuarda, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	linhas, err := r.pool.Query(ctx, `
		WITH enderecos AS (
			SELECT a.address, a.zone, a.capacity, a.pick_sequence, a.fixed_item_code,
			       COALESCE(s.quantity, 0) AS saldo,
			       COALESCE(ocupado.total, 0) AS ocupado
			  FROM manufacturing_warehouse_addresses a
			  LEFT JOIN stock_balances s
			         ON s.enterprise_id = a.enterprise_id AND s.warehouse_id = a.warehouse_id
			        AND s.address = a.address AND s.item_code = $2 AND s.mask = $3
			  LEFT JOIN LATERAL (
			       SELECT SUM(quantity) total FROM stock_balances x
			        WHERE x.enterprise_id = a.enterprise_id AND x.warehouse_id = a.warehouse_id
			          AND x.address = a.address
			  ) ocupado ON TRUE
			 WHERE a.enterprise_id = $1 AND a.warehouse_id = $4
			   AND a.is_active AND NOT a.is_blocked
			   -- Endereço fixo é de um item só: oferecê-lo a outro quebra a regra.
			   AND (a.fixed_item_code IS NULL OR a.fixed_item_code = $2)
		)
		SELECT address, zone, capacity, pick_sequence, saldo,
		       CASE
		         WHEN fixed_item_code = $2                     THEN 'endereço fixo do item'
		         WHEN saldo > 0                                THEN 'consolida com o saldo que já está aqui'
		         WHEN ocupado = 0 AND ($5 = '' OR zone = $5)    THEN 'endereço vazio na zona pedida'
		         ELSE 'endereço ativo com espaço'
		       END AS motivo,
		       (capacity IS NULL OR capacity - ocupado >= $6) AS cabe
		  FROM enderecos
		 WHERE (capacity IS NULL OR capacity - ocupado >= $6)
		 ORDER BY (fixed_item_code = $2) DESC NULLS LAST,
		          (saldo > 0) DESC,
		          ($5 <> '' AND zone = $5) DESC,
		          pick_sequence,
		          address
		 LIMIT 10`, enterpriseID, itemCode, mask, warehouseID, strings.TrimSpace(zona), quantidade)
	if err != nil {
		return nil, fmt.Errorf("sugerindo endereço de guarda: %w", err)
	}
	defer linhas.Close()
	var sugestoes []*entity.SugestaoGuarda
	for linhas.Next() {
		var s entity.SugestaoGuarda
		if err := linhas.Scan(&s.Address, &s.Zone, &s.Capacidade, &s.PickSeq, &s.SaldoAtual, &s.Motivo, &s.Cabe); err != nil {
			return nil, fmt.Errorf("lendo sugestão de guarda: %w", err)
		}
		sugestoes = append(sugestoes, &s)
	}
	return sugestoes, linhas.Err()
}

// CriarOndaDeSeparacao aloca as necessidades por FEFO, RESERVA o que alocou no
// endereço/lote e devolve a lista já na ordem da rota.
//
// Reservar é o ponto: sem isso duas ondas criadas no mesmo minuto mandam o
// separador ao mesmo endereço atrás da mesma peça. A reserva é feita dentro da
// mesma transação da alocação, com o saldo travado — entre ler o disponível e
// reservar não pode caber outra onda.
//
// A onda é criada mesmo quando falta saldo: separar o que existe e sinalizar a
// falta é mais útil ao almoxarifado que recusar a onda inteira.
func (r *StockRepositorySQLC) CriarOndaDeSeparacao(
	ctx context.Context, codigo, warehouseID int64, regra string, necessidades []entity.NecessidadeOnda, ator uuid.UUID,
) (*entity.OndaDeSeparacao, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Sem regra na onda, cada item sai pela sua regra de baixa.
	switch strings.ToUpper(strings.TrimSpace(regra)) {
	case "FIFO":
		regra = "FIFO"
	case "FEFO":
		regra = "FEFO"
	default:
		regra = "ITEM"
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo transação da onda: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := conferirAlmoxAceitaReserva(ctx, tx, enterpriseID, warehouseID); err != nil {
		return nil, err
	}
	itensDaOnda := make([]int64, 0, len(necessidades))
	for _, n := range necessidades {
		itensDaOnda = append(itensDaOnda, n.ItemCode)
	}
	if err := TravarItensTx(ctx, tx, enterpriseID, itensDaOnda...); err != nil {
		return nil, err
	}
	if codigo <= 0 {
		// A trava serializa a numeração: duas ondas ao mesmo tempo tirariam o mesmo código.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('stock_wave_code', $1))`, enterpriseID); err != nil {
			return nil, fmt.Errorf("travando a numeração da onda: %w", err)
		}
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(code),0)+1 FROM stock_picking_waves WHERE enterprise_id=$1`, enterpriseID).Scan(&codigo); err != nil {
			return nil, fmt.Errorf("numerando a onda: %w", err)
		}
	}
	onda := &entity.OndaDeSeparacao{
		Code: codigo, WarehouseID: warehouseID, Status: "ABERTA", Rule: regra,
		Linhas: []entity.LinhaOnda{}, EmFalta: []entity.FaltaNaOnda{},
	}
	if err = tx.QueryRow(ctx,
		`INSERT INTO stock_picking_waves(enterprise_id,code,warehouse_id,status,rule,created_by)
		 VALUES($1,$2,$3,'ABERTA',$4,$5) RETURNING id, created_at`,
		enterpriseID, codigo, warehouseID, regra, ator).Scan(&onda.ID, &onda.CreatedAt); err != nil {
		return nil, fmt.Errorf("criando a onda: %w", err)
	}

	for _, necessidade := range necessidades {
		if !necessidade.Quantity.IsPositive() {
			continue
		}
		regraDoItem := regra
		if regra == "ITEM" {
			regraDoItem = entity.BaixaFEFO
			err := tx.QueryRow(ctx, `SELECT lot_issue_rule FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`,
				enterpriseID, necessidade.ItemCode).Scan(&regraDoItem)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("lendo a regra de baixa do item: %w", err)
			}
		}
		ordem := `l.expires_at IS NULL, l.expires_at, l.received_at NULLS LAST, b.lot`
		if regraDoItem == "FIFO" {
			ordem = `l.received_at NULLS LAST, l.expires_at IS NULL, l.expires_at, b.lot`
		}
		restante := necessidade.Quantity
		// A reserva do pedido prende o saldo que a onda dele vai separar: sai
		// daqui e volta ao encerrar, para a nota fiscal encontrá-la.
		doPedido := decimal.Zero
		if necessidade.EhDoPedido() {
			if doPedido, err = soltarReservaDoPedidoTx(ctx, tx, enterpriseID, *necessidade.ReferenceCode, necessidade.ItemCode, necessidade.Mask, warehouseID, restante); err != nil {
				return nil, err
			}
		}
		// Trava a linha do endereço junto com a do lote: é nela que está a reserva
		// do pedido, que a onda precisa respeitar.
		candidatos, err := tx.Query(ctx, `
			SELECT b.id, b.lot, b.address, GREATEST(b.quantity - b.reserved_qty, 0) AS disponivel,
			       l.heat_number, l.expires_at,
			       COALESCE(a.zone,''), COALESCE(a.pick_sequence,0),
			       GREATEST(s.quantity - s.reserved_qty, 0) AS livre_endereco
			  FROM stock_lot_balances b
			  JOIN stock_balances s ON s.enterprise_id=b.enterprise_id AND s.item_code=b.item_code AND s.mask=b.mask
			                       AND s.warehouse_id=b.warehouse_id AND s.address=b.address
			  LEFT JOIN stock_lots l ON l.enterprise_id=b.enterprise_id AND l.item_code=b.item_code
			                        AND l.mask=b.mask AND l.lot=b.lot
			  LEFT JOIN manufacturing_warehouse_addresses a
			         ON a.enterprise_id=b.enterprise_id AND a.warehouse_id=b.warehouse_id AND a.address=b.address
			 WHERE b.enterprise_id=$1 AND b.item_code=$2 AND b.mask=$3 AND b.warehouse_id=$4
			   AND b.quantity - b.reserved_qty > 0
			   AND COALESCE(l.expires_at >= CURRENT_DATE, true)
			   AND COALESCE(a.is_blocked,false) = false
			   AND NOT `+loteSemSaidaSQL+`
			 ORDER BY `+ordem+`
			 FOR UPDATE OF b, s`,
			enterpriseID, necessidade.ItemCode, necessidade.Mask, warehouseID)
		if err != nil {
			return nil, fmt.Errorf("alocando o item %d na onda: %w", necessidade.ItemCode, err)
		}
		type alocacao struct {
			saldoID              int64
			lote, endereco, zona string
			quantidade           decimal.Decimal
			corrida              *string
			validade             *time.Time
			sequencia            int
		}
		var alocacoes []alocacao
		usadoNoEndereco := map[string]decimal.Decimal{}
		for candidatos.Next() {
			var c alocacao
			var disponivel, livreEndereco decimal.Decimal
			if err := candidatos.Scan(&c.saldoID, &c.lote, &c.endereco, &disponivel,
				&c.corrida, &c.validade, &c.zona, &c.sequencia, &livreEndereco); err != nil {
				candidatos.Close()
				return nil, fmt.Errorf("lendo candidato da onda: %w", err)
			}
			if !restante.IsPositive() {
				continue
			}
			// Vários lotes no mesmo endereço disputam o livre dele.
			livreEndereco = livreEndereco.Sub(usadoNoEndereco[c.endereco])
			c.quantidade = decimal.Min(decimal.Min(disponivel, livreEndereco), restante)
			if !c.quantidade.IsPositive() {
				continue
			}
			usadoNoEndereco[c.endereco] = usadoNoEndereco[c.endereco].Add(c.quantidade)
			restante = restante.Sub(c.quantidade)
			alocacoes = append(alocacoes, c)
		}
		errCandidatos := candidatos.Err()
		candidatos.Close()
		if errCandidatos != nil {
			return nil, fmt.Errorf("lendo candidatos da onda: %w", errCandidatos)
		}

		// Saldo sem lote: item que não controla lote também vai para a onda. Antes
		// ele aparecia inteiro como falta, por maior que fosse o estoque.
		if restante.IsPositive() {
			semLote, err := tx.Query(ctx, `
				SELECT s.address, COALESCE(a.zone,''), COALESCE(a.pick_sequence,0),
				       s.quantity - s.reserved_qty - COALESCE((SELECT SUM(x.quantity) FROM stock_lot_balances x
				            WHERE x.enterprise_id=s.enterprise_id AND x.item_code=s.item_code AND x.mask=s.mask
				              AND x.warehouse_id=s.warehouse_id AND x.address=s.address AND x.quantity > 0), 0)
				  FROM stock_balances s
				  LEFT JOIN manufacturing_warehouse_addresses a
				         ON a.enterprise_id=s.enterprise_id AND a.warehouse_id=s.warehouse_id AND a.address=s.address
				 WHERE s.enterprise_id=$1 AND s.item_code=$2 AND s.mask=$3 AND s.warehouse_id=$4
				   AND COALESCE(a.is_blocked,false) = false
				 ORDER BY COALESCE(a.pick_sequence,0), s.address
				 FOR UPDATE OF s`,
				enterpriseID, necessidade.ItemCode, necessidade.Mask, warehouseID)
			if err != nil {
				return nil, fmt.Errorf("alocando o saldo sem lote do item %d: %w", necessidade.ItemCode, err)
			}
			for semLote.Next() {
				var c alocacao
				var livre decimal.Decimal
				if err := semLote.Scan(&c.endereco, &c.zona, &c.sequencia, &livre); err != nil {
					semLote.Close()
					return nil, fmt.Errorf("lendo saldo sem lote da onda: %w", err)
				}
				c.quantidade = decimal.Min(livre.Sub(usadoNoEndereco[c.endereco]), restante)
				if !c.quantidade.IsPositive() {
					continue
				}
				usadoNoEndereco[c.endereco] = usadoNoEndereco[c.endereco].Add(c.quantidade)
				restante = restante.Sub(c.quantidade)
				alocacoes = append(alocacoes, c)
			}
			errSemLote := semLote.Err()
			semLote.Close()
			if errSemLote != nil {
				return nil, fmt.Errorf("lendo saldo sem lote da onda: %w", errSemLote)
			}
		}

		for _, c := range alocacoes {
			var reservaID int64
			if err = tx.QueryRow(ctx,
				`INSERT INTO stock_reservations(item_code,mask,warehouse_id,quantity,reference_type,reference_code,
				     status,created_by,enterprise_id,address,lot)
				 VALUES($1,$2,$3,$4,'ONDA_SEPARACAO',$5,'ACTIVE',$6,$7,$8,$9) RETURNING id`,
				necessidade.ItemCode, necessidade.Mask, warehouseID, c.quantidade, codigo, ator,
				enterpriseID, c.endereco, c.lote).Scan(&reservaID); err != nil {
				return nil, fmt.Errorf("reservando o lote %s: %w", c.lote, err)
			}
			if c.saldoID != 0 {
				if _, err = tx.Exec(ctx,
					`UPDATE stock_lot_balances SET reserved_qty = reserved_qty + $2, updated_at=NOW() WHERE id=$1`,
					c.saldoID, c.quantidade); err != nil {
					return nil, fmt.Errorf("marcando a reserva no saldo do lote: %w", err)
				}
			}
			if _, err = tx.Exec(ctx,
				`UPDATE stock_balances SET reserved_qty = reserved_qty + $6, updated_at=NOW()
				  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
				enterpriseID, necessidade.ItemCode, necessidade.Mask, warehouseID, c.endereco, c.quantidade); err != nil {
				return nil, fmt.Errorf("marcando a reserva no saldo do endereço: %w", err)
			}
			var linhaID int64
			daReserva := decimal.Min(doPedido, c.quantidade)
			doPedido = doPedido.Sub(daReserva)
			if err = tx.QueryRow(ctx,
				`INSERT INTO stock_picking_wave_lines(enterprise_id,wave_id,item_code,mask,lot,address,
				     pick_sequence,quantity,reservation_id,reference_type,reference_code,order_reserved_qty)
				 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
				enterpriseID, onda.ID, necessidade.ItemCode, necessidade.Mask, c.lote, c.endereco,
				c.sequencia, c.quantidade, reservaID, necessidade.ReferenceType, necessidade.ReferenceCode, daReserva).Scan(&linhaID); err != nil {
				return nil, fmt.Errorf("gravando a linha da onda: %w", err)
			}
			onda.Linhas = append(onda.Linhas, entity.LinhaOnda{
				ID: linhaID, ItemCode: necessidade.ItemCode, Mask: necessidade.Mask, Lot: c.lote,
				Address: c.endereco, Zone: c.zona, PickSequence: c.sequencia, Quantity: c.quantidade.InexactFloat64(),
				HeatNumber: c.corrida, ExpiresAt: c.validade,
				ReferenceType: necessidade.ReferenceType, ReferenceCode: necessidade.ReferenceCode,
			})
		}
		// O que a onda não alocou segue reservado para o pedido.
		if doPedido.IsPositive() {
			if err := reservarParaPedidoTx(ctx, tx, enterpriseID, *necessidade.ReferenceCode, necessidade.ItemCode, necessidade.Mask, warehouseID, doPedido, ator); err != nil {
				return nil, err
			}
		}
		if restante.IsPositive() {
			onda.EmFalta = append(onda.EmFalta, entity.FaltaNaOnda{
				ItemCode: necessidade.ItemCode, Quantity: restante.InexactFloat64(),
			})
		}
	}

	// Uma caminhada só: a ordem é a do galpão, não a das necessidades.
	sort.SliceStable(onda.Linhas, func(i, j int) bool {
		a, b := onda.Linhas[i], onda.Linhas[j]
		if a.PickSequence != b.PickSequence {
			if a.PickSequence == 0 || b.PickSequence == 0 {
				return b.PickSequence == 0
			}
			return a.PickSequence < b.PickSequence
		}
		return a.Address < b.Address
	})

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmando a criação da onda: %w", err)
	}
	return onda, nil
}

// ConfirmarOndaDeSeparacao baixa o estoque do que foi separado e libera as
// reservas. Sem este passo a reserva ficaria presa para sempre e o saldo, embora
// fisicamente entregue, continuaria aparecendo como disponível. Linha de pedido
// não baixa: o separado volta a ser reserva do pedido e a nota fiscal dá a saída.
func (r *StockRepositorySQLC) ConfirmarOndaDeSeparacao(ctx context.Context, codigo int64, separado map[int64]decimal.Decimal, ator uuid.UUID) (*entity.OndaDeSeparacao, error) {
	return r.encerrarOnda(ctx, codigo, separado, ator, true)
}

// CancelarOndaDeSeparacao devolve o reservado sem mexer no estoque físico.
func (r *StockRepositorySQLC) CancelarOndaDeSeparacao(ctx context.Context, codigo int64, ator uuid.UUID) (*entity.OndaDeSeparacao, error) {
	return r.encerrarOnda(ctx, codigo, nil, ator, false)
}

func (r *StockRepositorySQLC) encerrarOnda(ctx context.Context, codigo int64, separado map[int64]decimal.Decimal, ator uuid.UUID, confirmar bool) (*entity.OndaDeSeparacao, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo transação da onda: %w", err)
	}
	defer tx.Rollback(ctx)

	var ondaID, warehouseID int64
	var status, regra string
	if err = tx.QueryRow(ctx,
		`SELECT id, warehouse_id, status, rule FROM stock_picking_waves
		  WHERE enterprise_id=$1 AND code=$2 FOR UPDATE`, enterpriseID, codigo).
		Scan(&ondaID, &warehouseID, &status, &regra); err == pgx.ErrNoRows {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("onda de separação %d não encontrada", codigo))
	} else if err != nil {
		return nil, fmt.Errorf("lendo a onda: %w", err)
	}
	if status != "ABERTA" {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a onda %d já está %s", codigo, strings.ToLower(status)))
	}
	var itensDaOnda []int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(array_agg(DISTINCT item_code), '{}') FROM stock_picking_wave_lines WHERE wave_id=$1`, ondaID).Scan(&itensDaOnda); err != nil {
		return nil, fmt.Errorf("lendo os itens da onda: %w", err)
	}
	if err := TravarItensTx(ctx, tx, enterpriseID, itensDaOnda...); err != nil {
		return nil, err
	}

	linhas, err := tx.Query(ctx,
		`SELECT id, item_code, mask, lot, address, quantity, reservation_id, reference_type, reference_code, order_reserved_qty
		   FROM stock_picking_wave_lines WHERE wave_id=$1 ORDER BY pick_sequence, address`, ondaID)
	if err != nil {
		return nil, fmt.Errorf("lendo as linhas da onda: %w", err)
	}
	type linha struct {
		id, itemCode         int64
		mask, lote, endereco string
		quantidade, doPedido decimal.Decimal
		reservaID, refCodigo *int64
		refTipo              *string
	}
	var todas []linha
	for linhas.Next() {
		var l linha
		if err := linhas.Scan(&l.id, &l.itemCode, &l.mask, &l.lote, &l.endereco, &l.quantidade, &l.reservaID, &l.refTipo, &l.refCodigo, &l.doPedido); err != nil {
			linhas.Close()
			return nil, fmt.Errorf("lendo linha da onda: %w", err)
		}
		todas = append(todas, l)
	}
	errLinhas := linhas.Err()
	linhas.Close()
	if errLinhas != nil {
		return nil, fmt.Errorf("lendo linhas da onda: %w", errLinhas)
	}

	conhecidas := map[int64]bool{}
	for _, l := range todas {
		conhecidas[l.id] = true
		if q, ok := separado[l.id]; ok && (q.IsNegative() || q.GreaterThan(l.quantidade)) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("endereço %s: o separado precisa ficar entre zero e %s", l.endereco, entity.QuantidadeBR(l.quantidade)))
		}
	}
	for linha := range separado {
		if !conhecidas[linha] {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("uma das linhas informadas não é da onda %d", codigo))
		}
	}

	for _, l := range todas {
		pego := l.quantidade
		if q, ok := separado[l.id]; ok {
			pego = q
		}
		if !confirmar {
			pego = decimal.Zero
		}
		// A reserva sai nos dois casos: confirmada vira baixa física, cancelada
		// simplesmente volta a ficar disponível.
		if _, err = tx.Exec(ctx,
			`UPDATE stock_lot_balances SET reserved_qty = GREATEST(reserved_qty - $5, 0), updated_at=NOW()
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND lot=$4 AND address=$6 AND warehouse_id=$7`,
			enterpriseID, l.itemCode, l.mask, l.lote, l.quantidade, l.endereco, warehouseID); err != nil {
			return nil, fmt.Errorf("liberando a reserva do lote: %w", err)
		}
		if _, err = tx.Exec(ctx,
			`UPDATE stock_balances SET reserved_qty = GREATEST(reserved_qty - $6, 0), updated_at=NOW()
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
			enterpriseID, l.itemCode, l.mask, warehouseID, l.endereco, l.quantidade); err != nil {
			return nil, fmt.Errorf("liberando a reserva do endereço: %w", err)
		}
		if l.reservaID != nil {
			estado := "CANCELLED"
			if pego.IsPositive() {
				estado = "CONSUMED"
			}
			if _, err = tx.Exec(ctx, `UPDATE stock_reservations SET status=$2, updated_at=NOW() WHERE id=$1`,
				*l.reservaID, estado); err != nil {
				return nil, fmt.Errorf("encerrando a reserva: %w", err)
			}
		}
		if l.refTipo != nil && *l.refTipo == entity.ReferenceTypeSalesOrder && l.refCodigo != nil {
			if q := decimal.Max(pego, l.doPedido); q.IsPositive() {
				if err := reservarParaPedidoTx(ctx, tx, enterpriseID, *l.refCodigo, l.itemCode, l.mask, warehouseID, q, ator); err != nil {
					return nil, err
				}
			}
			if _, err = tx.Exec(ctx, `UPDATE stock_picking_wave_lines SET picked_qty=$2 WHERE id=$1`, l.id, pego); err != nil {
				return nil, fmt.Errorf("marcando a separação da linha: %w", err)
			}
			continue
		}
		if !pego.IsPositive() {
			continue
		}
		endereco := l.endereco
		movimento := &entity.StockMovement{
			ItemCode: l.itemCode, Mask: l.mask, WarehouseID: warehouseID,
			MovementType: entity.MovementTypeOut, Quantity: pego,
			CreatedBy: ator,
		}
		if l.lote != "" {
			lote := l.lote
			movimento.Lot = &lote
		}
		if l.endereco != "" {
			movimento.Address = &endereco
		}
		referencia := "ONDA_SEPARACAO"
		movimento.ReferenceType = &referencia
		movimento.ReferenceCode = &codigo
		if err = CreateMovementTx(ctx, tx, enterpriseID, movimento); err != nil {
			return nil, fmt.Errorf("baixando o estoque da linha da onda: %w", err)
		}
		if _, err = tx.Exec(ctx, `UPDATE stock_picking_wave_lines SET picked_qty=$2 WHERE id=$1`, l.id, pego); err != nil {
			return nil, fmt.Errorf("marcando a separação da linha: %w", err)
		}
	}

	novoStatus := "CANCELADA"
	coluna := "cancelled_at"
	if confirmar {
		novoStatus = "SEPARADA"
		coluna = "confirmed_at"
	}
	if _, err = tx.Exec(ctx,
		`UPDATE stock_picking_waves SET status=$2, `+coluna+`=NOW() WHERE id=$1`, ondaID, novoStatus); err != nil {
		return nil, fmt.Errorf("encerrando a onda: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmando o encerramento da onda: %w", err)
	}
	onda := &entity.OndaDeSeparacao{
		ID: ondaID, Code: codigo, WarehouseID: warehouseID, Status: novoStatus, Rule: regra,
		Linhas: []entity.LinhaOnda{}, EmFalta: []entity.FaltaNaOnda{},
	}
	if onda.Linhas, err = r.linhasDaOnda(ctx, enterpriseID, ondaID); err != nil {
		return nil, err
	}
	return onda, nil
}

// AjustarCustoTx soma `valor` ao custo do saldo do item no almoxarifado (valor
// negativo estorna), recalculando o custo médio, e registra o movimento
// AJUSTE_CUSTO de quantidade zero. Com o saldo em vários endereços, o valor é
// repartido pela quantidade de cada um. Sem saldo, nada é aplicado. O estorno
// nunca leva o custo total abaixo de zero. Devolve o id do movimento e o
// valor efetivamente aplicado.
func AjustarCustoTx(ctx context.Context, tx pgx.Tx, enterpriseID, itemCode int64, mask string, warehouseID int64,
	valor decimal.Decimal, refType string, refCode int64, notes string, createdBy uuid.UUID) (int64, decimal.Decimal, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, quantity, total_cost FROM public.stock_balances
		  WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$4 AND quantity > 0 ORDER BY id FOR UPDATE`,
		itemCode, mask, warehouseID, enterpriseID)
	if err != nil {
		return 0, decimal.Zero, fmt.Errorf("lendo o saldo para o ajuste de custo: %w", err)
	}
	type saldo struct {
		id         int64
		qtd, custo decimal.Decimal
	}
	var ss []saldo
	total := decimal.Zero
	for rows.Next() {
		var s saldo
		if err := rows.Scan(&s.id, &s.qtd, &s.custo); err != nil {
			rows.Close()
			return 0, decimal.Zero, err
		}
		ss = append(ss, s)
		total = total.Add(s.qtd)
	}
	rows.Close()
	if len(ss) == 0 || !total.IsPositive() || valor.IsZero() {
		return 0, decimal.Zero, nil
	}
	aplicado := decimal.Zero
	restante := valor
	for i, s := range ss {
		parte := valor.Mul(s.qtd).Div(total).Round(4)
		if i == len(ss)-1 {
			parte = restante
		}
		if s.custo.Add(parte).IsNegative() {
			parte = s.custo.Neg()
		}
		restante = restante.Sub(parte)
		novo := s.custo.Add(parte)
		if _, err := tx.Exec(ctx,
			`UPDATE public.stock_balances SET total_cost=$2, avg_cost=$2/quantity, updated_at=NOW() WHERE id=$1`, s.id, novo); err != nil {
			return 0, decimal.Zero, fmt.Errorf("aplicando o ajuste de custo: %w", err)
		}
		aplicado = aplicado.Add(parte)
	}
	var movID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO public.stock_movements (item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		     reference_type, reference_code, notes, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,0,0,$5,$6,$7,$8,$9,$10) RETURNING id`,
		itemCode, mask, warehouseID, entity.MovementTypeCostAdjustment, aplicado, refType, refCode, notes, createdBy, enterpriseID).Scan(&movID); err != nil {
		return 0, decimal.Zero, fmt.Errorf("registrando o ajuste de custo: %w", err)
	}
	return movID, aplicado, nil
}

// saidasDaInspecao: referências que podem tirar material de um almoxarifado de
// inspeção. A decisão de uso do lote (QUALITY_LOT) é o caminho normal; o
// estorno/cancelamento da entrada, o estorno de movimento e o inventário
// precisam continuar funcionando; as referências antigas da inspeção de
// recebimento ficam para lançamentos já existentes.
var saidasDaInspecao = map[string]bool{
	"QUALITY_LOT": true, entity.ReferenceTypeNFEntry: true, entity.ReferenceTypePurchaseOrder: true,
	entity.ReferenceTypeReversal: true, entity.ReferenceTypeInventory: true, entity.ReferenceTypeCycleCount: true,
	"RECEIVING_INSPECTION": true, "RECEIVING_INSPECTION_ANALYSIS": true,
}

// barrarSaidaDaInspecao é o bloqueio do material em inspeção: enquanto o lote
// não tem decisão de uso, ele não é consumido, separado, faturado nem
// transferido à mão. Antes o almoxarifado de inspeção só ficava fora do ATP e
// da reposição — qualquer requisição, apontamento ou transferência o tirava.
func barrarSaidaDaInspecao(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	saida := false
	switch m.MovementType {
	case entity.MovementTypeOut, entity.MovementTypeTransferOut, entity.MovementTypePlannedRequisition, "SAIDA":
		saida = true
	case entity.MovementTypeAdjustment:
		saida = m.Quantity.IsNegative()
	}
	if !saida {
		return nil
	}
	if m.ReferenceType != nil && saidasDaInspecao[*m.ReferenceType] {
		return nil
	}
	var local string
	err := tx.QueryRow(ctx, `SELECT location::text FROM warehouse WHERE id=$1 AND enterprise_id=$2`, m.WarehouseID, enterpriseID).Scan(&local)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("conferindo o almoxarifado da saída: %w", err)
	}
	if local != "INSPECAO" {
		return nil
	}
	return errorsuc.NewValidationError(fmt.Sprintf(
		"o item %d está no almoxarifado de inspeção e só sai pela decisão de uso do lote de inspeção (VINS0300): aceite, rejeite ou devolva o lote primeiro", m.ItemCode))
}
