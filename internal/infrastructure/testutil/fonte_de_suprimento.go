//go:build integration

package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FonteDeSuprimento cadastra o fornecedor como fonte preferencial do item
// (VSUP0130), na empresa do ITEM. Requisição, pedido de compra e ordem de
// serviço de terceiro exigem fonte (migração 000502). fornecedor 0 = um
// código qualquer: basta para a requisição, que ainda não tem fornecedor.
func FonteDeSuprimento(t *testing.T, pool *pgxpool.Pool, item, fornecedor int64) {
	t.Helper()
	if fornecedor == 0 {
		fornecedor = UniqueCode()
	}
	tag, err := pool.Exec(context.Background(), `
		INSERT INTO item_preferred_suppliers (enterprise_id, item_code, supplier_code, mask, ranking, is_preferred, created_by)
		SELECT i.enterprise_id, i.code, $2, '', 1, TRUE, (SELECT id FROM users ORDER BY created_at LIMIT 1)
		  FROM items i WHERE i.code = $1
		ON CONFLICT (enterprise_id, item_code, supplier_code, mask) WHERE enterprise_id IS NOT NULL
		DO UPDATE SET is_active = TRUE, blocked = FALSE, blocked_reason = NULL, valid_from = NULL, valid_until = NULL`, item, fornecedor)
	if err != nil {
		t.Fatalf("cadastrando a fonte de suprimento do item %d: %v", item, err)
	}
	if tag.RowsAffected() == 0 {
		t.Fatalf("fonte de suprimento: item %d não existe", item)
	}
	// Roda antes da limpeza do próprio teste (ordem inversa). Se uma nota de
	// entrada ainda aponta para o vínculo, fica para a limpeza da empresa.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM item_preferred_suppliers s WHERE s.item_code = $1 AND s.supplier_code = $2
			AND NOT EXISTS (SELECT 1 FROM fiscal_entry_items f WHERE f.item_supplier_id = s.id)`, item, fornecedor)
	})
}
