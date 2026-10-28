package fiscal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/journal"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ReferenciaSaidaCancelada marca a volta ao estoque da NF-e de saída cancelada.
const ReferenciaSaidaCancelada = "NF_SAIDA_CANCELADA"

// CustoDaSaida soma o custo (médio, no momento da saída) do que a NF-e baixou do estoque.
func (r *FiscalRepositoryPG) CustoDaSaida(ctx context.Context, exitID int64) (decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	var v decimal.Decimal
	err = r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_price),0) FROM stock_movements
		  WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3 AND movement_type=$4`,
		empresa, stockentity.ReferenceTypeNFExit, exitID, stockentity.MovementTypeOut).Scan(&v)
	return v, err
}

// TributosReformaDaSaida lê os totais de IBS e CBS que a autorização gravou
// junto do retrato transmitido.
func (r *FiscalRepositoryPG) TributosReformaDaSaida(ctx context.Context, exitID int64) (decimal.Decimal, decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	var ibs, cbs decimal.Decimal
	err = r.db.QueryRow(ctx,
		`SELECT COALESCE(valor_ibs,0), COALESCE(valor_cbs,0) FROM fiscal_exits WHERE id=$1 AND enterprise_id=$2`,
		exitID, empresa).Scan(&ibs, &cbs)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("lendo IBS/CBS da NF-e %d: %w", exitID, err)
	}
	return ibs, cbs, nil
}

// EstornarEstoqueDaSaida devolve ao estoque, pelo mesmo custo, o que a NF-e
// cancelada baixou. Idempotente: repetir o cancelamento não devolve de novo.
func (r *FiscalRepositoryPG) EstornarEstoqueDaSaida(ctx context.Context, exitID int64, userID uuid.UUID) (int, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("saida-estoque:%d:%d", empresa, exitID)); err != nil {
		return 0, err
	}
	var ja bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM stock_movements WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3)`,
		empresa, ReferenciaSaidaCancelada, exitID).Scan(&ja); err != nil {
		return 0, err
	}
	if ja {
		return 0, nil
	}
	rows, err := tx.Query(ctx,
		`SELECT item_code, mask, warehouse_id, quantity, total_price, lot, address FROM stock_movements
		  WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3 AND movement_type=$4 ORDER BY id`,
		empresa, stockentity.ReferenceTypeNFExit, exitID, stockentity.MovementTypeOut)
	if err != nil {
		return 0, err
	}
	var movs []*stockentity.StockMovement
	for rows.Next() {
		var m stockentity.StockMovement
		var qtd, total decimal.Decimal
		var addr *string
		if err := rows.Scan(&m.ItemCode, &m.Mask, &m.WarehouseID, &qtd, &total, &m.Lot, &addr); err != nil {
			rows.Close()
			return 0, err
		}
		m.Quantity, m.TotalPrice = qtd.Abs(), total
		if m.Quantity.IsPositive() {
			m.UnitPrice = total.DivRound(m.Quantity, stockentity.CasasCusto)
		}
		if addr != nil && *addr != "" {
			m.Address = addr
		}
		movs = append(movs, &m)
	}
	rows.Close()
	ref := ReferenciaSaidaCancelada
	nota := "volta ao estoque: NF-e de saída cancelada"
	for _, m := range movs {
		m.MovementType = stockentity.MovementTypeIn
		m.ReferenceType, m.ReferenceCode = &ref, &exitID
		m.Notes, m.CreatedBy = &nota, userID
		if err := stockrepo.CreateMovementTx(ctx, tx, empresa, m); err != nil {
			return 0, err
		}
	}
	return len(movs), tx.Commit(ctx)
}

// GravarLoteContabil grava os lançamentos (fora de uma transação de negócio:
// a NF-e já foi autorizada na SEFAZ e não se desfaz).
func (r *FiscalRepositoryPG) GravarLoteContabil(ctx context.Context, lote contabilizacao.Lote) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if ja, err := journal.Existe(ctx, tx, empresa, lote.SourceType, lote.SourceID); err != nil || ja {
		return err // idempotente
	}
	if err := journal.Gravar(ctx, tx, empresa, lote); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// EstornarLoteContabil inverte os lançamentos da origem (idempotente).
func (r *FiscalRepositoryPG) EstornarLoteContabil(ctx context.Context, origem string, id int64, origemEstorno, prefixo string, data time.Time) (int, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	return journal.Estornar(ctx, r.pool, empresa, origem, id, origemEstorno, prefixo, data)
}
