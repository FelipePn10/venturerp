//go:build integration

package delivery_promise_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity"
	paramsentity "github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise_params"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestDeliveryPromiseIsIsolatedByTenant: reservas de tanque e parâmetros da
// promessa eram do banco inteiro — a promessa de uma empresa enxergava a
// ocupação dos tanques da outra.
func TestDeliveryPromiseIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	item := int64(1_760_000_000 + testutil.UniqueCode()%10_000_000)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		_, _ = pool.Exec(base, `DELETE FROM delivery_tank_reservations WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
		_, _ = pool.Exec(base, `DELETE FROM delivery_promise_params WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
		_, _ = pool.Exec(base, `DELETE FROM items WHERE code=$1`, item)
	})
	testutil.SeedItem(t, pool, a.Ctx, item, testutil.Actor(t, pool))
	repo := delivery_promise.New(pool)
	hoje := time.Date(2030, 3, 10, 0, 0, 0, 0, time.UTC)
	res := &entity.TankReservation{Code: testutil.UniqueCode(), ItemCode: item, TankCode: 1, RequestedQty: decimal.NewFromFloat(10), ReservedQty: decimal.NewFromFloat(10),
		AllocationDate: hoje, ExpiresAt: hoje.AddDate(0, 0, 5), Status: entity.TankReservationActive, CreatedBy: uuid.New()}
	if _, err := repo.CreateReservation(b.Ctx, res); err == nil {
		t.Fatal("empresa B reservou tanque para item da A")
	}
	if _, err := repo.CreateReservation(a.Ctx, res); err != nil {
		t.Fatal(err)
	}
	if l, err := repo.ListActiveReservations(b.Ctx, hoje, hoje, nil); err != nil || len(l) != 0 {
		t.Fatalf("a promessa da B viu %d reserva(s) da A (%v)", len(l), err)
	}
	if err := repo.CancelReservation(b.Ctx, res.Code); err == nil {
		t.Fatal("empresa B cancelou a reserva da A")
	}
	if n, err := repo.ExpireReservations(b.Ctx, hoje.AddDate(1, 0, 0)); err != nil || n != 0 {
		t.Fatalf("empresa B expirou %d reserva(s) da A (%v)", n, err)
	}
	if l, err := repo.ListActiveReservations(a.Ctx, hoje, hoje, nil); err != nil || len(l) != 1 {
		t.Fatalf("a reserva da A sumiu: %d (%v)", len(l), err)
	}

	params := delivery_promise_params.NewDeliveryPromiseParamsRepositorySQLC(sqlc.New(pool))
	if _, err := params.Save(a.Ctx, &paramsentity.DeliveryPromiseParams{UseDeliveryPromise: true, DefaultOrderSort: "DATA", ShowOrderValues: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := params.Get(b.Ctx); err == nil {
		t.Fatal("empresa B leu os parâmetros de promessa da A")
	}
	if _, err := params.Save(b.Ctx, &paramsentity.DeliveryPromiseParams{UseDeliveryPromise: false, DefaultOrderSort: "DATA", ShowOrderValues: 1}); err != nil {
		t.Fatal(err)
	}
	if got, err := params.Get(a.Ctx); err != nil || !got.UseDeliveryPromise {
		t.Fatalf("a configuração da B sobrescreveu a da A: %+v (%v)", got, err)
	}
}

// TestNextReservationCodeCriaASequencia: a base da Tecnofer não tinha a linha da
// sequência das reservas, e o UPDATE puro respondia "não encontrado" a toda
// reserva gravada. Sem a linha, o próximo código é 1; com ela, segue contando.
func TestNextReservationCodeCriaASequencia(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	var antes *int64
	_ = pool.QueryRow(ctx, `SELECT last_code FROM delivery_tank_reservation_sequences WHERE id = 1`).Scan(&antes)
	t.Cleanup(func() {
		if antes != nil {
			_, _ = pool.Exec(ctx, `UPDATE delivery_tank_reservation_sequences SET last_code = GREATEST(last_code, $1) WHERE id = 1`, *antes)
		}
	})
	if _, err := pool.Exec(ctx, `DELETE FROM delivery_tank_reservation_sequences`); err != nil {
		t.Fatal(err)
	}
	repo := delivery_promise.New(pool)
	if c, err := repo.NextReservationCode(ctx); err != nil || c != 1 {
		t.Fatalf("sem a linha da sequência: código %d, erro %v; esperado 1", c, err)
	}
	if c, err := repo.NextReservationCode(ctx); err != nil || c != 2 {
		t.Fatalf("segundo código %d, erro %v; esperado 2", c, err)
	}
}
