//go:build integration

package production_order_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// A tabela manufacturing_warehouse_addresses tem chave composta e não possui
// coluna id. A consulta chegou a selecioná-la, o que fazia
// GET /api/warehouse-addresses devolver 500 em qualquer chamada.
func TestListWarehouseAddressesMatchesCompositeKeyTable(t *testing.T) {
	_, pool := testutil.Queries(t)
	base := context.Background()
	var enterpriseID int64
	if err := pool.QueryRow(base, "SELECT MIN(id) FROM enterprise").Scan(&enterpriseID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(base, contextkey.UserKey, &security.AuthUser{EnterpriseID: enterpriseID})
	repo := production_order.NewProductionOrderRepositoryPGX(pool)

	// TEST_DATABASE_URL é persistente e não há rollback entre execuções: o
	// endereço precisa ser único e removido ao final para não contaminar
	// consultas de outros testes.
	warehouse := testutil.UniqueCode()
	address := fmt.Sprintf("END-%d", warehouse)
	if err := repo.ConfigureWarehouseAddress(ctx, warehouse, address, true); err != nil {
		t.Fatalf("não configurou o endereço: %v", err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool,
			"DELETE FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1 AND warehouse_id=$2 AND address=$3",
			enterpriseID, warehouse, address)
	})

	all, err := repo.ListWarehouseAddresses(ctx, nil)
	if err != nil {
		t.Fatalf("listagem sem filtro falhou: %v", err)
	}
	found := false
	for _, item := range all {
		if item.WarehouseID == warehouse && item.Address == address {
			found = true
			if !item.IsActive {
				t.Fatal("endereço ativo veio como inativo")
			}
		}
	}
	if !found {
		t.Fatalf("endereço configurado não apareceu na listagem de %d registro(s)", len(all))
	}

	filtered, err := repo.ListWarehouseAddresses(ctx, &warehouse)
	if err != nil {
		t.Fatalf("listagem filtrada falhou: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Address != address {
		t.Fatalf("filtro por almoxarifado devolveu %+v, esperado apenas %q", filtered, address)
	}
}
