//go:build integration

// Separar e cancelar o romaneio são tudo ou nada: reservas de estoque e
// situação entram juntas. Antes a reserva podia falhar (ou o status) e o
// romaneio ficava separado sem reserva — ou cancelado com a reserva presa.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run RomaneioTudoOuNada ./internal/application/usecase/shipment_uc/
package shipment_uc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	shipmentpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// situacaoQueFalha deixa a reserva entrar na transação e recusa a mudança de status.
type situacaoQueFalha struct{ repository.ShipmentRepository }

func (situacaoQueFalha) UpdateStatus(context.Context, int64, entity.ShipmentStatus, *uuid.UUID, string) error {
	return errors.New("romaneio travado por outra conferência")
}

func TestRomaneioTudoOuNada(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	code := int64(1_975_000_000 + u%15_000_000)
	var empresa int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Romaneio transação') RETURNING id`, code).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: code})
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,4,$2,$3)`, u, ator, empresa)
	testutil.Exec(t, pool, `INSERT INTO stock_balances(item_code,warehouse_id,quantity,reserved_qty,enterprise_id) VALUES($1,4,50,0,$2)`, u, empresa)
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM stock_reservations WHERE enterprise_id=$1`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM stock_balances WHERE enterprise_id=$1`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM shipment_events WHERE shipment_id IN (SELECT id FROM shipments WHERE enterprise_id=$1)`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM shipment_items WHERE shipment_id IN (SELECT id FROM shipments WHERE enterprise_id=$1)`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM shipments WHERE enterprise_id=$1`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE code=$1`, u)
		_, _ = pool.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresa)
	})

	falhar := true
	uc := &shipment_uc.ShipmentUseCase{
		Repo:  shipmentpg.NewShipmentRepositoryPG(pool),
		Stock: stockpg.NewStockRepositorySQLC(pool),
		Transacao: func(ctx context.Context, fn func(shipment_uc.RomaneioRepos) error) error {
			tx, err := pool.Begin(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback(ctx) }()
			var romaneios repository.ShipmentRepository = shipmentpg.NewShipmentRepositoryTx(tx)
			if falhar {
				romaneios = situacaoQueFalha{romaneios}
			}
			if err := fn(shipment_uc.RomaneioRepos{Romaneios: romaneios, Estoque: stockpg.NewStockRepositoryTx(tx)}); err != nil {
				return err
			}
			return tx.Commit(ctx)
		},
	}
	criado, err := uc.Create(ctx, shipment_uc.CreateShipmentInput{CreatedBy: ator})
	if err != nil {
		t.Fatal(err)
	}
	deposito := int64(4)
	if _, err := uc.AddItem(ctx, shipment_uc.AddShipmentItemInput{ShipmentCode: criado.Code, Sequence: 1, ItemCode: u, WarehouseID: &deposito, Quantity: 6}); err != nil {
		t.Fatal(err)
	}
	estado := func() (status string, reservas int) {
		t.Helper()
		if err := pool.QueryRow(bg, `SELECT status FROM shipments WHERE code=$1 AND enterprise_id=$2`, criado.Code, empresa).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(bg, `SELECT count(*) FROM stock_reservations WHERE enterprise_id=$1 AND reference_type='SHIPMENT' AND reference_code=$2 AND status='ACTIVE'`, empresa, criado.Code).Scan(&reservas); err != nil {
			t.Fatal(err)
		}
		return
	}

	// Separar com a mudança de status falhando: a reserva feita antes é desfeita.
	if err := uc.Separate(ctx, criado.Code, ator); err == nil || !strings.Contains(err.Error(), "travado") {
		t.Fatalf("erro = %v, quer a falha do status", err)
	}
	if st, n := estado(); st != string(entity.ShipmentStatusOpen) || n != 0 {
		t.Fatalf("depois da falha: status=%s reservas=%d, quer aberto e 0", st, n)
	}
	falhar = false
	if err := uc.Separate(ctx, criado.Code, ator); err != nil {
		t.Fatal(err)
	}
	if st, n := estado(); st != string(entity.ShipmentStatusSeparated) || n != 1 {
		t.Fatalf("separado: status=%s reservas=%d, quer separado e 1", st, n)
	}

	// Cancelar com a mudança de status falhando: a reserva continua de pé.
	falhar = true
	if err := uc.Cancel(ctx, criado.Code, ator, "teste"); err == nil {
		t.Fatal("cancelamento deveria falhar")
	}
	if st, n := estado(); st != string(entity.ShipmentStatusSeparated) || n != 1 {
		t.Fatalf("depois da falha no cancelamento: status=%s reservas=%d, quer separado e 1", st, n)
	}
	falhar = false
	if err := uc.Cancel(ctx, criado.Code, ator, "teste"); err != nil {
		t.Fatal(err)
	}
	if st, n := estado(); st != string(entity.ShipmentStatusCancelled) || n != 0 {
		t.Fatalf("cancelado: status=%s reservas=%d, quer cancelado e 0", st, n)
	}
}
