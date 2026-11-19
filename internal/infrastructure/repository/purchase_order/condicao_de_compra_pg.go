package purchase_order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	ppentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// CondicaoDeCompra: unidade de compra (fonte do fornecedor na VSUP0130 →
// unidade de compra do item → estoque), casas dela no catálogo e o preço
// vigente da tabela de compra. Tudo da empresa da sessão (enterprise_id).
func (r *PurchaseOrderRepositorySQLC) CondicaoDeCompra(ctx context.Context, itemCode, supplierCode int64, mask string) (purchase_order_uc.CondicaoDeCompra, error) {
	out := purchase_order_uc.CondicaoDeCompra{Casas: -1}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return out, err
	}
	var casas *int32
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(it.business_code, ''), it.code::text), c.unidade, (SELECT u.decimals::int FROM units_of_measure u
		                    WHERE u.enterprise_id = it.enterprise_id AND u.is_active AND u.code = c.unidade)
		  FROM items it
		  CROSS JOIN LATERAL (SELECT upper(btrim(COALESCE(
		         (SELECT NULLIF(btrim(s.uom), '') FROM item_preferred_suppliers s
		           WHERE s.enterprise_id = it.enterprise_id AND s.item_code = it.code AND s.supplier_code = $3
		             AND s.is_active AND NOT s.blocked AND s.mask IN ($4, '')
		           ORDER BY (s.mask = $4) DESC, s.ranking LIMIT 1),
		         NULLIF(btrim(it.supplies_purchase_uom::text), ''),
		         it.warehouse_unit_of_measurement::text, ''))) AS unidade) c
		 WHERE it.enterprise_id = $1 AND it.code = $2`,
		enterpriseID, itemCode, supplierCode, mask).Scan(&out.Rotulo, &out.Unidade, &casas)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, fmt.Errorf("item %d não encontrado nesta empresa", itemCode)
	}
	if err != nil {
		return out, fmt.Errorf("lendo a unidade de compra do item %d: %w", itemCode, err)
	}
	if casas != nil {
		out.Casas = *casas
	}

	// Preços: as linhas ativas de tabela vigente HOJE — CURRENT_DATE, a data do
	// banco no fuso da empresa, a mesma regra da linha manual do pedido —, do
	// fornecedor (na tabela ou na linha) antes da geral, faixa maior antes da
	// menor. A faixa está na unidade da linha da tabela: quem escolhe é o caso
	// de uso, que sabe converter a quantidade do pedido para ela.
	rows, err := r.db.Query(ctx, `
		SELECT i.id, i.price, upper(btrim(COALESCE(i.uom, ''))), COALESCE(i.min_qty, 0), t.code
		  FROM purchase_price_table_items i
		  JOIN purchase_price_tables t ON t.id = i.table_id
		 WHERE t.enterprise_id = $1 AND t.is_active AND i.is_active AND i.item_code = $2 AND i.price > 0
		   AND COALESCE(i.supplier_code, t.supplier_code, $3) = $3
		   AND (t.validity_start IS NULL OR t.validity_start <= CURRENT_DATE)
		   AND (t.validity_end IS NULL OR t.validity_end >= CURRENT_DATE)
		 ORDER BY (COALESCE(i.supplier_code, t.supplier_code) IS NOT NULL) DESC, COALESCE(i.min_qty, 0) DESC,
		          t.validity_start DESC NULLS LAST, i.id DESC`, enterpriseID, itemCode, supplierCode)
	if err != nil {
		return out, fmt.Errorf("lendo o preço de compra do item %d: %w", itemCode, err)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		var p purchase_order_uc.PrecoDeTabela
		if err := rows.Scan(&id, &p.Preco, &p.Unidade, &p.MinQtd, &p.Tabela); err != nil {
			return out, fmt.Errorf("lendo o preço de compra do item %d: %w", itemCode, err)
		}
		ids = append(ids, id)
		out.Precos = append(out.Precos, p)
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("lendo o preço de compra do item %d: %w", itemCode, err)
	}
	rows.Close()
	if len(ids) == 0 {
		return out, nil
	}
	// Descontos e acréscimos em cadeia de cada linha (VSUP0120): o preço que
	// vale é o líquido, a mesma conta da linha manual e da requisição. As
	// linhas já são da empresa (filtro acima); o ajuste vai pelo id delas.
	ajustes := map[int64][]*ppentity.PriceAdjustment{}
	arows, err := r.db.Query(ctx, `SELECT price_item_id, sequence, adjustment_kind, calculation_type, value
		  FROM purchase_price_item_adjustments WHERE price_item_id = ANY($1) ORDER BY price_item_id, sequence`, ids)
	if err != nil {
		return out, fmt.Errorf("lendo os descontos da tabela de compra do item %d: %w", itemCode, err)
	}
	defer arows.Close()
	for arows.Next() {
		a := &ppentity.PriceAdjustment{}
		if err := arows.Scan(&a.PriceItemID, &a.Sequence, &a.Kind, &a.CalculationType, &a.Value); err != nil {
			return out, fmt.Errorf("lendo os descontos da tabela de compra do item %d: %w", itemCode, err)
		}
		ajustes[a.PriceItemID] = append(ajustes[a.PriceItemID], a)
	}
	if err := arows.Err(); err != nil {
		return out, fmt.Errorf("lendo os descontos da tabela de compra do item %d: %w", itemCode, err)
	}
	for i, id := range ids {
		out.Precos[i].Preco = ppentity.PrecoLiquido(out.Precos[i].Preco, ajustes[id])
	}
	return out, nil
}

var _ purchase_order_uc.LeitorDeCondicaoDeCompra = (*PurchaseOrderRepositorySQLC)(nil)

// UnidadeDeCompraDoItem: a unidade de compra do cadastro do item, senão a de
// estoque (o item de serviço da operação externa é comprado nela).
func (r *PurchaseOrderRepositorySQLC) UnidadeDeCompraDoItem(ctx context.Context, itemCode int64) (string, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return "", err
	}
	var u string
	err = r.db.QueryRow(ctx, `
		SELECT upper(btrim(COALESCE(NULLIF(btrim(supplies_purchase_uom::text), ''), warehouse_unit_of_measurement::text, '')))
		  FROM items WHERE enterprise_id = $1 AND code = $2`, enterpriseID, itemCode).Scan(&u)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("item de serviço %d não encontrado nesta empresa", itemCode)
	}
	if err != nil {
		return "", fmt.Errorf("lendo a unidade de compra do item %d: %w", itemCode, err)
	}
	return u, nil
}
