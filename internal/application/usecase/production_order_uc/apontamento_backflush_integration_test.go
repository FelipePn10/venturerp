//go:build integration

// Apontamento e baixa dos componentes (backflush) são tudo ou nada: se um
// componente não sai do estoque, o apontamento também não fica. Antes o
// apontamento era gravado e a falha na baixa sumia.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run ApontamentoComBackflush ./internal/application/usecase/production_order_uc/
package production_order_uc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structurerepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	productionpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

type estruturaDeDois struct {
	structurerepo.ItemStructureRepository
	filhos []*structentity.ItemStructure
}

func (e estruturaDeDois) GetAllDirectChildren(context.Context, int64) ([]*structentity.ItemStructure, error) {
	return e.filhos, nil
}

// segundaBaixaFalha deixa a primeira baixa entrar (na transação) e recusa a segunda.
type segundaBaixaFalha struct {
	stockrepo.StockRepository
	chamadas *int
}

func (s segundaBaixaFalha) CreateMovement(ctx context.Context, m *stockentity.StockMovement) (*stockentity.StockMovement, error) {
	*s.chamadas++
	if *s.chamadas == 2 {
		return nil, errors.New("saldo bloqueado para inventário")
	}
	return s.StockRepository.CreateMovement(ctx, m)
}

func TestApontamentoComBackflushTudoOuNada(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	empresaCode := int64(1_960_000_000 + u%30_000_000)
	var empresaID, of, operador int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Apontamento backflush') RETURNING id`, empresaCode).Scan(&empresaID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: empresaID, EnterpriseCode: empresaCode})
	pai, filhoA, filhoB := u, u+1, u+2
	for _, c := range []int64{pai, filhoA, filhoB} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,3,$2,$3)`, c, actor, empresaID)
	}
	if err := pool.QueryRow(bg, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id)
		VALUES($1,$2,10,'IN_PROGRESS',$3,$4) RETURNING id`, u, pai, actor, empresaID).Scan(&of); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO employees(code,name,created_by,enterprise_id) VALUES($1,'Operador',$2,$3) RETURNING id`, u%1_000_000_000, actor, empresaID).Scan(&operador); err != nil {
		t.Fatal(err)
	}
	// Saldo dos componentes: sem ele a primeira baixa já seria recusada (saldo
	// negativo bloqueado), e o teste é sobre a SEGUNDA falhar.
	for _, c := range []int64{filhoA, filhoB} {
		if _, err := stockpg.NewStockRepositorySQLC(pool).CreateMovement(ctx, &stockentity.StockMovement{
			ItemCode: c, WarehouseID: 3, MovementType: stockentity.MovementTypeIn, Quantity: 100, UnitPrice: 1, CreatedBy: actor,
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM production_appointments WHERE production_order_id=$1`, of)
		_, _ = pool.Exec(bg, `DELETE FROM stock_movements WHERE enterprise_id=$1`, empresaID)
		_, _ = pool.Exec(bg, `DELETE FROM stock_balances WHERE enterprise_id=$1`, empresaID)
		_, _ = pool.Exec(bg, `DELETE FROM production_orders WHERE id=$1`, of)
		_, _ = pool.Exec(bg, `DELETE FROM employees WHERE id=$1`, operador)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE enterprise_id=$1`, empresaID)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresaID)
	})

	chamadas := 0
	falhar := true
	deposito := int64(3)
	uc := &production_order_uc.AddAppointmentUseCase{
		Repo: productionpg.NewProductionOrderRepositoryPGX(pool),
		Auth: &auth.AuthService{},
		StructureRepo: estruturaDeDois{filhos: []*structentity.ItemStructure{
			{ChildCode: filhoA, Quantity: 2},
			{ChildCode: filhoB, Quantity: 1},
		}},
		StockRepo: stockpg.NewStockRepositorySQLC(pool),
		Transacao: func(ctx context.Context, fn func(production_order_uc.ApontamentoRepos) error) error {
			tx, err := pool.Begin(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback(ctx) }()
			var estoque stockrepo.StockRepository = stockpg.NewStockRepositoryTx(tx)
			if falhar {
				estoque = segundaBaixaFalha{StockRepository: estoque, chamadas: &chamadas}
			}
			if err := fn(production_order_uc.ApontamentoRepos{
				Ordens:  productionpg.NewProductionOrderRepositoryPGX(tx),
				Estoque: estoque,
			}); err != nil {
				return err
			}
			return tx.Commit(ctx)
		},
	}
	contar := func(sql string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(bg, sql, of).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	dto := request.AddAppointmentDTO{ProductionOrderID: of, EmployeeID: &operador, ProducedQty: 5, BackflushWarehouseID: &deposito}

	_, err := uc.Execute(ctx, dto)
	if err == nil || !strings.Contains(err.Error(), "saldo bloqueado") {
		t.Fatalf("erro = %v, quer a falha da segunda baixa", err)
	}
	if n := contar(`SELECT count(*) FROM production_appointments WHERE production_order_id=$1`); n != 0 {
		t.Fatalf("apontamentos = %d depois da falha, quer 0", n)
	}
	if n := contar(`SELECT count(*) FROM stock_movements WHERE reference_code=$1 AND reference_type='PRODUCTION_ORDER'`); n != 0 {
		t.Fatalf("baixas = %d depois da falha, quer 0 (a primeira tinha de ser desfeita)", n)
	}

	falhar = false
	if _, err := uc.Execute(ctx, dto); err != nil {
		t.Fatal(err)
	}
	if n := contar(`SELECT count(*) FROM production_appointments WHERE production_order_id=$1`); n != 1 {
		t.Fatalf("apontamentos = %d, quer 1", n)
	}
	var baixas int
	if err := pool.QueryRow(bg, `SELECT count(*) FROM stock_movements WHERE enterprise_id=$1 AND movement_type='OUT'`, empresaID).Scan(&baixas); err != nil {
		t.Fatal(err)
	}
	if baixas != 2 {
		t.Fatalf("baixas = %d, quer os 2 componentes", baixas)
	}
}
