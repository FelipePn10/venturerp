//go:build integration

package shipment_uc_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	shipmentpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O número do romaneio é por empresa: cada empresa tem o seu romaneio 1. Com a
// unicidade global (até a migração 000440), a segunda empresa recebia "já
// existe um registro com os dados informados" ao criar o primeiro romaneio.
func TestRomaneioNumeradoPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			_, _ = pool.Exec(bg, `DELETE FROM shipment_events WHERE shipment_id IN (SELECT id FROM shipments WHERE enterprise_id=$1)`, e.ID)
			_, _ = pool.Exec(bg, `DELETE FROM shipments WHERE enterprise_id=$1`, e.ID)
			_, _ = pool.Exec(bg, `DELETE FROM shipment_sequences WHERE id=$1`, e.ID)
		}
	})
	uc := &shipment_uc.ShipmentUseCase{Repo: shipmentpg.NewShipmentRepositoryPG(pool)}
	ra, err := uc.Create(a.Ctx, shipment_uc.CreateShipmentInput{CreatedBy: ator})
	if err != nil {
		t.Fatal(err)
	}
	rb, err := uc.Create(b.Ctx, shipment_uc.CreateShipmentInput{CreatedBy: ator})
	if err != nil {
		t.Fatalf("a segunda empresa cria o seu romaneio: %v", err)
	}
	if ra.Code != 1 || rb.Code != 1 {
		t.Fatalf("cada empresa começa no romaneio 1: A=%d B=%d", ra.Code, rb.Code)
	}
	if _, err := uc.Get(b.Ctx, 1); err != nil {
		t.Fatalf("a empresa B acha o seu romaneio 1: %v", err)
	}
}
