package purchase_order

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

// FornecedorSugerido é o fornecedor que o MRP escolheu para a sugestão de
// compra de onde a ordem planejada nasceu (preferencial, ranking ou cota em
// VSUP0130). Sem sugestão de origem ou sem fornecedor: false.
func (r *PurchaseOrderRepositorySQLC) FornecedorSugerido(ctx context.Context, plannedCode int64) (int64, bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, false, err
	}
	var fornecedor int64
	// Da ordem (gravado ao firmar); ordem antiga, da sugestão se ainda existir.
	err = r.db.QueryRow(ctx, `SELECT COALESCE(p.supplier_code, s.supplier_code) FROM planned_orders p
		LEFT JOIN mrp_planned_suggestions s ON s.code = p.mrp_suggestion_code AND s.enterprise_id = p.enterprise_id
		WHERE p.enterprise_id = $1 AND p.code = $2 AND COALESCE(p.supplier_code, s.supplier_code) IS NOT NULL`, enterpriseID, plannedCode).Scan(&fornecedor)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return fornecedor, true, nil
}

// FornecedoresSugeridos: o fornecedor escolhido pelo MRP para cada ordem
// planejada de compra ativa da empresa.
func (r *PurchaseOrderRepositorySQLC) FornecedoresSugeridos(ctx context.Context) (map[int64]int64, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT p.code, COALESCE(p.supplier_code, s.supplier_code) FROM planned_orders p
		LEFT JOIN mrp_planned_suggestions s ON s.code = p.mrp_suggestion_code AND s.enterprise_id = p.enterprise_id
		WHERE p.enterprise_id = $1 AND p.is_active AND COALESCE(p.supplier_code, s.supplier_code) IS NOT NULL`, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int64{}
	for rows.Next() {
		var code, fornecedor int64
		if err := rows.Scan(&code, &fornecedor); err != nil {
			return nil, err
		}
		out[code] = fornecedor
	}
	return out, rows.Err()
}

// GravarFornecedor guarda na ordem planejada o fornecedor que o MRP escolheu
// (VSUP0130). A sugestão some no recálculo do plano; a ordem fica.
func (r *PurchaseOrderRepositorySQLC) GravarFornecedor(ctx context.Context, plannedCode, supplier int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	// planned_orders é exceção da conferência de empresa do banco (planejamento
	// entre fábricas): o fornecedor tem de ser da empresa da ordem, conferido aqui.
	// Fornecedor fora da empresa não grava (e não derruba a firmação: a ordem
	// já existe; a aprovação pede o fornecedor). Só erro de banco volta.
	_, err = r.db.Exec(ctx, `UPDATE planned_orders SET supplier_code=$3, updated_at=NOW()
		WHERE enterprise_id=$1 AND code=$2
		  AND EXISTS (SELECT 1 FROM suppliers f WHERE f.code=$3 AND f.enterprise_id=$1)`, enterpriseID, plannedCode, supplier)
	return err
}
