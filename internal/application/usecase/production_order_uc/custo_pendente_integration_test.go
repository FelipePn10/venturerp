//go:build integration

// OF encerrada com a apuração de custo falhando: o encerramento vale e a OF fica
// marcada com "custo pendente" e o motivo, até "Apurar custo" dar certo.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run CustoPendente ./internal/application/usecase/production_order_uc/
package production_order_uc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	stdentity "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	stdrepo "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	productionpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type custoPadraoIndisponivel struct{ stdrepo.StandardCostRepository }

func (custoPadraoIndisponivel) GetItemStandardCost(context.Context, int64, string) (*stdentity.ItemStandardCost, error) {
	return nil, errors.New("conexão com o banco de custos recusada")
}

func TestCustoPendenteNoEncerramentoDaOF(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	empresaCode := int64(1_970_000_000 + u%20_000_000)
	var empresaID, of int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Custo pendente') RETURNING id`, empresaCode).Scan(&empresaID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresaID, EnterpriseCode: empresaCode})
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,3,$2,$3)`, u, actor, empresaID)
	if err := pool.QueryRow(bg, `INSERT INTO production_orders(order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id)
		VALUES($1,$1,10,10,'COMPLETED',$2,$3) RETURNING id`, u, actor, empresaID).Scan(&of); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM production_order_costs WHERE production_order_id=$1`, of)
		_, _ = pool.Exec(bg, `DELETE FROM production_orders WHERE id=$1`, of)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE code=$1`, u)
		_, _ = pool.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, empresaID)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresaID)
	})

	repo := productionpg.NewProductionOrderRepositoryPGX(pool)
	autz := &auth.AuthService{}
	fechar := &production_order_uc.CloseProductionOrderUseCase{Repo: repo, Auth: autz,
		SettleUC: &production_order_uc.SettleProductionCostUseCase{Repo: repo, Auth: autz, StdCostRepo: custoPadraoIndisponivel{}}}
	consultar := &production_order_uc.GetProductionOrderUseCase{Repo: repo, Auth: autz}

	ordem, err := fechar.Execute(ctx, request.CloseProductionOrderDTO{ID: of})
	if err != nil {
		t.Fatalf("o encerramento tem de valer mesmo sem custo: %v", err)
	}
	if ordem.CostPending == nil || !strings.Contains(*ordem.CostPending, "conexão com o banco de custos recusada") {
		t.Fatalf("custo pendente na resposta = %v, quer o motivo", ordem.CostPending)
	}
	visto, err := consultar.Execute(ctx, of)
	if err != nil {
		t.Fatal(err)
	}
	if visto.CostPending == nil || !strings.Contains(*visto.CostPending, "custo real não foi apurado") {
		t.Fatalf("custo pendente na consulta = %v, quer o motivo gravado", visto.CostPending)
	}

	// "Apurar custo" dando certo limpa a pendência.
	apurar := &production_order_uc.SettleProductionCostUseCase{Repo: repo, Auth: autz}
	if _, err := apurar.Execute(ctx, of); err != nil {
		t.Fatal(err)
	}
	visto, err = consultar.Execute(ctx, of)
	if err != nil {
		t.Fatal(err)
	}
	if visto.CostPending != nil {
		t.Fatalf("pendência continuou depois da apuração: %q", *visto.CostPending)
	}
}
