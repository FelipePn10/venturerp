//go:build integration

package production_order_test

import (
	"context"
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

	const warehouse int64 = 7
	if err := repo.ConfigureWarehouseAddress(ctx, warehouse, "B-02", true); err != nil {
		t.Fatalf("não configurou o endereço: %v", err)
	}

	all, err := repo.ListWarehouseAddresses(ctx, nil)
	if err != nil {
		t.Fatalf("listagem sem filtro falhou: %v", err)
	}
	found := false
	for _, address := range all {
		if address.WarehouseID == warehouse && address.Address == "B-02" {
			found = true
			if !address.IsActive {
				t.Fatal("endereço ativo veio como inativo")
			}
		}
	}
	if !found {
		t.Fatalf("endereço configurado não apareceu na listagem: %+v", all)
	}

	filtered, err := repo.ListWarehouseAddresses(ctx, &[]int64{warehouse}[0])
	if err != nil {
		t.Fatalf("listagem filtrada falhou: %v", err)
	}
	for _, address := range filtered {
		if address.WarehouseID != warehouse {
			t.Fatalf("filtro por almoxarifado vazou outro registro: %+v", address)
		}
	}
}
