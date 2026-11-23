package shipment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// RegrasDoRomaneioPG lê o cadastro dos itens da carga, da empresa da sessão.
type RegrasDoRomaneioPG struct{ pool *pgxpool.Pool }

func NewRegrasDoRomaneioPG(pool *pgxpool.Pool) *RegrasDoRomaneioPG {
	return &RegrasDoRomaneioPG{pool: pool}
}

func (r *RegrasDoRomaneioPG) ItensQueExigemVolume(ctx context.Context, itens []int64) ([]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT business_code FROM items
		 WHERE enterprise_id = $1 AND code = ANY($2) AND commercial_assemble_shipping_volumes ORDER BY business_code`, empresa, itens)
	if err != nil {
		return nil, fmt.Errorf("lendo os itens que exigem volume: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *RegrasDoRomaneioPG) VolumesPorUnidade(ctx context.Context, itens []int64) (map[int64]decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Volumes por unidade vence o fator: um é "uma unidade ocupa N caixas", o outro
	// é "uma unidade ocupa esta fração da caixa".
	rows, err := r.pool.Query(ctx, `SELECT code, COALESCE(commercial_volumes_per_unit::numeric, commercial_volume_conversion_factor) FROM items
		 WHERE enterprise_id = $1 AND code = ANY($2)
		   AND (commercial_volumes_per_unit IS NOT NULL OR commercial_volume_conversion_factor IS NOT NULL)`, empresa, itens)
	if err != nil {
		return nil, fmt.Errorf("lendo os volumes por unidade: %w", err)
	}
	defer rows.Close()
	out := map[int64]decimal.Decimal{}
	for rows.Next() {
		var codigo int64
		var n decimal.Decimal
		if err := rows.Scan(&codigo, &n); err != nil {
			return nil, err
		}
		out[codigo] = n
	}
	return out, rows.Err()
}

func (r *RegrasDoRomaneioPG) Embalagens(ctx context.Context, itens []int64) (map[int64]shipment_uc.EmbalagemDoItem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT i.code, e.code, e.warehouse_code FROM items i
		  JOIN items e ON e.code = i.commercial_packaging_item_code AND e.enterprise_id = i.enterprise_id
		 WHERE i.enterprise_id = $1 AND i.code = ANY($2)`, empresa, itens)
	if err != nil {
		return nil, fmt.Errorf("lendo as embalagens dos itens: %w", err)
	}
	defer rows.Close()
	out := map[int64]shipment_uc.EmbalagemDoItem{}
	for rows.Next() {
		var item int64
		var e shipment_uc.EmbalagemDoItem
		if err := rows.Scan(&item, &e.ItemCode, &e.WarehouseID); err != nil {
			return nil, err
		}
		out[item] = e
	}
	return out, rows.Err()
}

func (r *RegrasDoRomaneioPG) SaldoDaEmbalagem(ctx context.Context, e shipment_uc.EmbalagemDoItem) (decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	var saldo decimal.Decimal
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(quantity - reserved_qty), 0) FROM stock_balances
		 WHERE enterprise_id = $1 AND item_code = $2 AND warehouse_id = $3 AND mask = ''`, empresa, e.ItemCode, e.WarehouseID).Scan(&saldo)
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo o saldo da embalagem: %w", err)
	}
	return saldo, nil
}

// MascarasDasLinhas: a máscara (item configurado) de cada linha do pedido de
// venda — o romaneio não a guarda, e a reserva completada pela separação precisa dela.
func (r *RegrasDoRomaneioPG) MascarasDasLinhas(ctx context.Context, linhas []int64) (map[int64]string, error) {
	out := map[int64]string{}
	if len(linhas) == 0 {
		return out, nil
	}
	codigo, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT l.code, COALESCE(l.mask,'') FROM sales_order_items l
		JOIN sales_orders so ON so.code = l.sales_order_code AND so.enterprise_code = $2
		WHERE l.code = ANY($1)`, linhas, codigo)
	if err != nil {
		return nil, fmt.Errorf("lendo a máscara das linhas do pedido: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var m string
		if err := rows.Scan(&id, &m); err != nil {
			return nil, err
		}
		out[id] = m
	}
	return out, rows.Err()
}

// DepositosPadrao: o depósito padrão de cada item (cadastro do item). A NF-e já
// baixa por ele quando a linha do pedido não tem depósito; a separação também.
func (r *RegrasDoRomaneioPG) DepositosPadrao(ctx context.Context, itens []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(itens) == 0 {
		return out, nil
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT code, warehouse_code FROM items
		WHERE code = ANY($1) AND enterprise_id = $2 AND warehouse_code IS NOT NULL AND warehouse_code > 0`, itens, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo o depósito padrão dos itens: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item, dep int64
		if err := rows.Scan(&item, &dep); err != nil {
			return nil, err
		}
		out[item] = dep
	}
	return out, rows.Err()
}
