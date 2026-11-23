//go:build integration

package shipment_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	shipmentrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Bug M: cancelar a NF-e deixava o romaneio "expedido" e ligado à nota. O
// cancelamento desliga a nota e volta o expedido para conferido; a empresa B
// não enxerga nem reabre o romaneio da A.
func TestNotaCanceladaReabreORomaneio(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		pool.Exec(base, `DELETE FROM shipment_events WHERE shipment_id IN (SELECT id FROM shipments WHERE enterprise_id = ANY($1))`, ids)
		pool.Exec(base, `DELETE FROM shipments WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM shipment_sequences WHERE id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM fiscal_exits WHERE enterprise_id = ANY($1)`, ids)
	})
	var nota int64
	if err := pool.QueryRow(base, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status)
		VALUES($1,CURRENT_DATE,'5102','Venda',$2,$3,'CANCELLED') RETURNING id`, testutil.UniqueCode()%1_000_000, ator, a.ID).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	repo := shipmentrepo.NewShipmentRepositoryPG(pool)
	s, err := repo.Create(a.Ctx, &entity.Shipment{Code: 1, Status: entity.ShipmentStatusShipped, CreatedBy: ator})
	if err != nil {
		t.Fatal(err)
	}
	num := int64(55)
	if err := repo.SetFiscalExit(a.Ctx, s.Code, &nota, &num, nil, &ator); err != nil {
		t.Fatal(err)
	}

	if cods, err := repo.RomaneiosDaNota(b.Ctx, nota); err != nil || len(cods) != 0 {
		t.Fatalf("a empresa B achou romaneio da A: %v err=%v", cods, err)
	}
	if err := repo.ReabrirPelaNotaCancelada(b.Ctx, nota); err != nil {
		t.Fatal(err)
	}
	if cods, err := repo.RomaneiosDaNota(a.Ctx, nota); err != nil || len(cods) != 1 || cods[0] != s.Code {
		t.Fatalf("romaneios da nota = %v err=%v", cods, err)
	}

	if err := repo.ReabrirPelaNotaCancelada(a.Ctx, nota); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByCode(a.Ctx, s.Code)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != entity.ShipmentStatusConferred || got.FiscalExitID != nil || got.NFeNumber != nil {
		t.Fatalf("romaneio depois do cancelamento: status=%s nota=%v número=%v", got.Status, got.FiscalExitID, got.NFeNumber)
	}
	if cods, _ := repo.RomaneiosDaNota(a.Ctx, nota); len(cods) != 0 {
		t.Fatalf("o romaneio continua ligado à nota cancelada: %v", cods)
	}
}
