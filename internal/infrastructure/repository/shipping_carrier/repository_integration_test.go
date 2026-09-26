//go:build integration

package shipping_carrier_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/repository"
	carrierpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipping_carrier"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/shopspring/decimal"
)

func texto(v string) *string { return &v }

// O cadastro de transportadora usa `enterprise_id` (a convenção do pai,
// `suppliers`). Este teste prova as duas coisas que mais quebram: o filtro de
// empresa e a substituição de frota/regiões.
func TestTransportadoraIsolaEmpresaESubstituiFilhos(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := testutil.TenantContext(t, pool)
	enterprise := testutil.EnterpriseID(t, ctx)
	actor := testutil.Actor(t, pool)
	repo := carrierpg.New(pool)

	fornecedor := testutil.UniqueCode()
	documento := strconv.FormatInt(fornecedor, 10)
	if _, err := pool.Exec(ctx, `INSERT INTO suppliers(code,name,document_number,is_active,created_by,enterprise_id)
		VALUES($1,'Transportadora de Teste',$2,TRUE,$3,$4)`, fornecedor, documento, actor, enterprise); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM shipping_carriers WHERE supplier_code=$1`, fornecedor)
		_, _ = pool.Exec(context.Background(), `DELETE FROM suppliers WHERE code=$1`, fornecedor)
	})

	transportadora := &entity.Transportadora{
		SupplierCode:    fornecedor,
		ANTTRNTRC:       texto("12345678"),
		Modal:           entity.ModalRodoviario,
		IssuesCTe:       true,
		FreightMinValue: decimal.NewFromInt(80),
		FreightKgRate:   decimal.RequireFromString("0.9"),
		IsActive:        true,
		Vehicles: []*entity.Veiculo{
			{Plate: "ABC1D23", CapacityKg: decimal.NewFromInt(5000), IsActive: true},
			{Plate: "XYZ4567", CapacityKg: decimal.NewFromInt(12000), IsActive: true},
		},
		ServiceAreas: []*entity.RegiaoAtendida{
			{State: texto("SP"), LeadDays: 3, KgRate: decimal.RequireFromString("0.8"), IsActive: true},
		},
	}
	salva, err := repo.Salvar(ctx, transportadora)
	if err != nil {
		t.Fatal(err)
	}
	if len(salva.Vehicles) != 2 || len(salva.ServiceAreas) != 1 {
		t.Fatalf("frota/regiões não gravadas: %d veículo(s), %d região(ões)", len(salva.Vehicles), len(salva.ServiceAreas))
	}

	// Gravar de novo com UM veículo substitui a frota inteira — é assim que a
	// tela envia, e frota pela metade é pior que nenhuma.
	salva.Vehicles = []*entity.Veiculo{{Plate: "QRS9A88", CapacityKg: decimal.NewFromInt(3000), IsActive: true}}
	salva.ServiceAreas = append(salva.ServiceAreas, &entity.RegiaoAtendida{State: texto("RJ"), LeadDays: 5, IsActive: true})
	segunda, err := repo.Salvar(ctx, salva)
	if err != nil {
		t.Fatal(err)
	}
	if len(segunda.Vehicles) != 1 || segunda.Vehicles[0].Plate != "QRS9A88" {
		t.Fatalf("frota não foi substituída: %+v", segunda.Vehicles)
	}
	if len(segunda.ServiceAreas) != 2 {
		t.Fatalf("regiões não foram substituídas: %d", len(segunda.ServiceAreas))
	}

	// A mesma placa duas vezes é conflito, não erro técnico.
	segunda.Vehicles = []*entity.Veiculo{{Plate: "QRS9A88", IsActive: true}, {Plate: "QRS9A88", IsActive: true}}
	if _, err := repo.Salvar(ctx, segunda); err == nil {
		t.Fatal("placa repetida na frota deveria ser recusada")
	}

	// Outra empresa não vê nem consegue alterar.
	outra := context.WithValue(context.Background(), contextkey.UserKey, &security.AuthUser{
		Role: "ADMIN", EnterpriseID: enterprise + 90_000, EnterpriseCode: enterprise + 90_000,
	})
	if lista, err := repo.Listar(outra, domrepo.Filtro{}); err != nil || len(lista) != 0 {
		t.Fatalf("outra empresa viu %d transportadora(s) (err=%v)", len(lista), err)
	}
	if _, err := repo.Obter(outra, segunda.ID); err == nil {
		t.Fatal("outra empresa leu a transportadora pelo id")
	}
	if err := repo.DefinirSituacao(outra, segunda.ID, false); err == nil {
		t.Fatal("outra empresa inativou a transportadora")
	}
}
