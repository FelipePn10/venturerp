//go:build integration

package shipment_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	shiprepo "github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// A carga grava a transportadora (shipping_carriers) só da própria empresa e ativa.
func TestIntegration_Carga_TransportadoraDaEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	repo := shipment.NewShipmentRepositoryPG(pool)

	empresa := func(nome string) (int64, context.Context) {
		var id int64
		code := int64(1_700_000_000 + testutil.UniqueCode()%100_000_000)
		if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,$2) RETURNING id`, code, nome).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM enterprise WHERE id=$1", id) })
		return id, context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: id, EnterpriseCode: code})
	}
	a, ctxA := empresa("Carga A")
	_, ctxB := empresa("Carga B")
	user := uuid.New()
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Carga',$2,'x')`, user, user.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM users WHERE id=$1", user) })

	fornecedor := testutil.UniqueCode()
	var supplierID, carrierID int64
	if err := pool.QueryRow(bg, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id)
		VALUES($1,'TRANSPORTES CARGA LTDA','12345678000195',$2,$3) RETURNING id`, fornecedor, user, a).Scan(&supplierID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO shipping_carriers(enterprise_id,supplier_code) VALUES($1,$2) RETURNING id`, a, fornecedor).Scan(&carrierID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM shipment_loads WHERE shipping_carrier_id=$1", carrierID)
		testutil.Exec(t, pool, "DELETE FROM shipping_carriers WHERE id=$1", carrierID)
		testutil.Exec(t, pool, "DELETE FROM suppliers WHERE id=$1", supplierID)
	})

	carga, err := repo.CreateLoad(ctxA, shiprepo.CreateLoadInput{ShippingCarrierID: &carrierID, CreatedBy: user})
	if err != nil {
		t.Fatal(err)
	}
	lida, err := repo.GetLoadByCode(ctxA, carga.Code)
	if err != nil {
		t.Fatal(err)
	}
	if lida.ShippingCarrierID == nil || *lida.ShippingCarrierID != carrierID {
		t.Fatalf("transportadora da carga: %v", lida.ShippingCarrierID)
	}
	if _, err := repo.CreateLoad(ctxB, shiprepo.CreateLoadInput{ShippingCarrierID: &carrierID, CreatedBy: user}); err == nil {
		t.Fatal("empresa B gravou a transportadora da A")
	}
	testutil.Exec(t, pool, "UPDATE shipping_carriers SET is_active = FALSE WHERE id=$1", carrierID)
	if _, err := repo.CreateLoad(ctxA, shiprepo.CreateLoadInput{ShippingCarrierID: &carrierID, CreatedBy: user}); err == nil {
		t.Fatal("transportadora inativa não pode ir para a carga")
	}
}

// Romaneio de pedido de venda sem transportadora herda a preferencial do
// cliente; pedido sem frete fica sem.
func TestIntegration_Romaneio_HerdaTransportadoraDoCliente(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	repo := shipment.NewShipmentRepositoryPG(pool)

	var empresaID int64
	empresaCode := int64(1_800_000_000 + testutil.UniqueCode()%100_000_000)
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Romaneio herda') RETURNING id`, empresaCode).Scan(&empresaID); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: empresaID, EnterpriseCode: empresaCode})
	user := uuid.New()
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Romaneio',$2,'x')`, user, user.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	fornecedor, cliente := testutil.UniqueCode(), testutil.UniqueCode()
	var carrierID int64
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'TRANSP HERDA','12345678000195',$2,$3)`, fornecedor, user, empresaID)
	if err := pool.QueryRow(bg, `INSERT INTO shipping_carriers(enterprise_id,supplier_code) VALUES($1,$2) RETURNING id`, empresaID, fornecedor).Scan(&carrierID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id,shipping_carrier_id) VALUES($1,'CLIENTE HERDA','11444777000161',$2,$3,$4)`, cliente, user, empresaID, carrierID)
	pedido := func(n int, frete *string) int64 {
		var code int64
		if err := pool.QueryRow(bg, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,freight_type,created_by) VALUES($1,$2,'R',$3,$4,$5) RETURNING code`,
			n, empresaCode, cliente, frete, user).Scan(&code); err != nil {
			t.Fatal(err)
		}
		return code
	}
	semFrete := "SEM FRETE"
	comFrete, semFretePedido := pedido(1, nil), pedido(2, &semFrete)
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM shipments WHERE enterprise_id=$1", empresaID)
		testutil.Exec(t, pool, "DELETE FROM sales_orders WHERE enterprise_code=$1", empresaCode)
		testutil.Exec(t, pool, "DELETE FROM customers WHERE enterprise_id=$1", empresaID)
		testutil.Exec(t, pool, "DELETE FROM shipping_carriers WHERE id=$1", carrierID)
		testutil.Exec(t, pool, "DELETE FROM suppliers WHERE enterprise_id=$1", empresaID)
		testutil.Exec(t, pool, "DELETE FROM users WHERE id=$1", user)
		testutil.Exec(t, pool, "DELETE FROM enterprise WHERE id=$1", empresaID)
	})

	romaneio := func(so int64) *int64 {
		code, err := repo.NextCode(ctx)
		if err != nil {
			t.Fatal(err)
		}
		s, err := repo.Create(ctx, &entity.Shipment{Code: code, SalesOrderCode: &so, Status: entity.ShipmentStatusOpen, CreatedBy: user})
		if err != nil {
			t.Fatal(err)
		}
		return s.ShippingCarrierID
	}
	if got := romaneio(comFrete); got == nil || *got != carrierID {
		t.Fatalf("romaneio do pedido com frete: transportadora %v, quer %d", got, carrierID)
	}
	if got := romaneio(semFretePedido); got != nil {
		t.Fatalf("pedido sem frete herdou transportadora %d", *got)
	}

	var compra int64
	if err := pool.QueryRow(bg, `INSERT INTO purchase_orders(order_number,enterprise_code,status,supplier_code,created_by,shipping_carrier_id) VALUES(1,$1,'APPROVED',$2,$3,$4) RETURNING code`,
		empresaCode, fornecedor, user, carrierID).Scan(&compra); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM purchase_orders WHERE code=$1", compra) })
	code, err := repo.NextCode(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s, err := repo.Create(ctx, &entity.Shipment{Code: code, PurchaseOrderCode: &compra, Status: entity.ShipmentStatusOpen, CreatedBy: user})
	if err != nil {
		t.Fatal(err)
	}
	if s.ShippingCarrierID == nil || *s.ShippingCarrierID != carrierID {
		t.Fatalf("romaneio do pedido de compra: transportadora %v, quer %d", s.ShippingCarrierID, carrierID)
	}
}
