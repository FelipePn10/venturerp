//go:build integration

package procurement_test

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/procurement"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Aviso de recebimento sem transportadora herda a do pedido de compra, mas só
// enquanto ela estiver ativa.
func TestIntegration_Aviso_HerdaTransportadoraDoPedido(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		testutil.Exec(t, pool, `DELETE FROM receiving_notices WHERE enterprise_code=$1`, a.Code)
		testutil.Exec(t, pool, `DELETE FROM purchase_orders WHERE enterprise_code=$1`, a.Code)
		testutil.Exec(t, pool, `DELETE FROM shipping_carriers WHERE enterprise_id=$1`, a.ID)
		testutil.Exec(t, pool, `DELETE FROM suppliers WHERE enterprise_id=$1`, a.ID)
	})
	ator := testutil.Actor(t, pool)
	fornecedor := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'TRANSP AVISO','12345678000195',$2,$3)`, fornecedor, ator, a.ID)
	var carrierID, pedido int64
	if err := pool.QueryRow(bg, `INSERT INTO shipping_carriers(enterprise_id,supplier_code) VALUES($1,$2) RETURNING id`, a.ID, fornecedor).Scan(&carrierID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO purchase_orders(order_number,enterprise_code,status,supplier_code,created_by,shipping_carrier_id) VALUES(1,$1,'APPROVED',$2,$3,$4) RETURNING code`,
		a.Code, fornecedor, ator, carrierID).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	repo := procurement.New(pool)
	aviso := func() *int64 {
		n, err := repo.CreateReceivingNotice(a.Ctx, &entity.ReceivingNotice{SupplierCode: &fornecedor, PurchaseOrderCode: &pedido, Status: "SCHEDULED", CreatedBy: &ator})
		if err != nil {
			t.Fatal(err)
		}
		return n.ShippingCarrierID
	}
	if got := aviso(); got == nil || *got != carrierID {
		t.Fatalf("aviso: transportadora %v, quer %d", got, carrierID)
	}
	testutil.Exec(t, pool, `UPDATE shipping_carriers SET is_active=FALSE WHERE id=$1`, carrierID)
	if got := aviso(); got != nil {
		t.Fatalf("aviso herdou transportadora inativa %d", *got)
	}
}
