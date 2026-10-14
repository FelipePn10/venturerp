//go:build integration

package procurement_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/procurement"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestProcurementIsIsolatedByTenant: a sonda cruzada lia, da outra empresa,
// registros de suprimentos, mensagens EDI, processos de importação e avisos de
// recebimento — por id e na lista. Cada empresa só lê e altera o que é seu.
func TestProcurementIsIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ctx := context.Background()
		codes := []int64{a.Code, b.Code}
		for _, tabela := range []string{"receiving_inspection_orders", "supplier_contracts", "supplier_scorecard_snapshots", "receiving_divergences", "receiving_notices", "supplier_edi_messages",
			"import_processes", "supplier_homologations", "procurement_records"} {
			if _, err := pool.Exec(ctx, `DELETE FROM `+tabela+` WHERE enterprise_code = ANY($1)`, codes); err != nil {
				t.Errorf("limpando %s: %v", tabela, err)
			}
		}
	})
	repo := procurement.New(pool)
	supplier := int64(1_690_000_000 + testutil.UniqueCode()%10_000_000)

	rec, err := repo.CreateRecord(a.Ctx, &entity.Record{RecordType: "SUPPLIER_EDI", Status: "OPEN", SupplierCode: &supplier, Payload: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetRecord(b.Ctx, rec.ID); err == nil {
		t.Fatal("empresa B leu o registro de suprimentos da A")
	}
	if l, err := repo.ListRecords(b.Ctx, "SUPPLIER_EDI", ""); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d registro(s) da A (%v)", len(l), err)
	}
	if got, err := repo.GetRecord(a.Ctx, rec.ID); err != nil || got.ID != rec.ID {
		t.Fatalf("empresa A não leu o próprio registro: %v", err)
	}

	edi, err := repo.CreateEDIMessage(a.Ctx, &entity.SupplierEDIMessage{SupplierCode: &supplier, Direction: "INBOUND", MessageType: "ASN", Status: "RECEIVED", Payload: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if edi.EnterpriseCode != a.Code {
		t.Fatalf("mensagem EDI gravada na empresa %d, esperado %d", edi.EnterpriseCode, a.Code)
	}
	if _, err := repo.GetEDIMessage(b.Ctx, edi.ID); err == nil {
		t.Fatal("empresa B leu a mensagem EDI da A")
	}
	if l, err := repo.ListEDIMessages(b.Ctx, &supplier); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d mensagem(ns) EDI da A (%v)", len(l), err)
	}

	imp, err := repo.CreateImportProcess(a.Ctx, &entity.ImportProcess{SupplierCode: &supplier, Currency: "USD", ExchangeRate: 5, ApportionBasis: "VALUE", Status: "OPEN"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetImportProcess(b.Ctx, imp.ID); err == nil {
		t.Fatal("empresa B leu o processo de importação da A")
	}
	if l, err := repo.ListImportProcesses(b.Ctx, ""); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d processo(s) de importação da A (%v)", len(l), err)
	}
	if _, err := repo.UpdateImportProcessStatus(b.Ctx, imp.ID, "CANCELLED"); err == nil {
		t.Fatal("empresa B cancelou o processo de importação da A")
	}
	if got, err := repo.GetImportProcess(a.Ctx, imp.ID); err != nil || got.Status != "OPEN" {
		t.Fatalf("processo da A mudou: %+v %v", got, err)
	}

	aviso, err := repo.CreateReceivingNotice(a.Ctx, &entity.ReceivingNotice{SupplierCode: &supplier, Status: "SCHEDULED"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetReceivingNotice(b.Ctx, aviso.ID); err == nil {
		t.Fatal("empresa B leu o aviso de recebimento da A")
	}
	if l, err := repo.ListReceivingNotices(b.Ctx, ""); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d aviso(s) da A (%v)", len(l), err)
	}
	if _, err := repo.UpdateReceivingNoticeStatus(b.Ctx, aviso.ID, "BLOCKED", true); err == nil {
		t.Fatal("empresa B bloqueou o aviso de recebimento da A")
	}
	if _, err := repo.CreateReceivingDivergence(b.Ctx, &entity.ReceivingDivergence{NoticeID: &aviso.ID, DivergenceType: "SHORTAGE", Resolution: "PENDING"}); err == nil {
		t.Fatal("empresa B registrou divergência no aviso da A")
	}
	div, err := repo.CreateReceivingDivergence(a.Ctx, &entity.ReceivingDivergence{NoticeID: &aviso.ID, SupplierCode: &supplier, DivergenceType: "SHORTAGE", Resolution: "PENDING"})
	if err != nil {
		t.Fatal(err)
	}
	if l, err := repo.ListReceivingDivergences(b.Ctx, &supplier, ""); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d divergência(s) da A (%v)", len(l), err)
	}
	if _, err := repo.ResolveReceivingDivergence(b.Ctx, div.ID, "WAIVED"); err == nil {
		t.Fatal("empresa B resolveu a divergência da A")
	}
	if l, err := repo.ListReceivingDivergences(a.Ctx, &supplier, ""); err != nil || len(l) != 1 {
		t.Fatalf("empresa A deveria ver a própria divergência: %d (%v)", len(l), err)
	}

	insp, err := repo.CreateReceivingInspectionOrder(a.Ctx, &entity.ReceivingInspectionOrder{Source: "MANUAL", SupplierCode: &supplier, ItemCode: supplier, WarehouseID: 1, Quantity: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetReceivingInspectionOrder(b.Ctx, insp.ID); err == nil {
		t.Fatal("empresa B leu a ordem de inspeção da A")
	}
	if l, err := repo.ListReceivingInspectionOrders(b.Ctx, ""); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d ordem(ns) de inspeção da A (%v)", len(l), err)
	}

	hoje := time.Now().UTC().Truncate(24 * time.Hour)
	contrato, err := repo.CreateSupplierContract(a.Ctx, &entity.SupplierContract{SupplierCode: supplier, ContractNumber: "CT-A", Status: "ACTIVE", Currency: "BRL", ValidFrom: hoje,
		Items: []*entity.SupplierContractItem{{ItemCode: supplier, ContractedQty: 10, UnitPrice: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetSupplierContract(b.Ctx, contrato.ID); err == nil {
		t.Fatal("empresa B leu o contrato da A")
	}
	if l, err := repo.ListSupplierContracts(b.Ctx, supplier, ""); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d contrato(s) da A (%v)", len(l), err)
	}
	if _, err := repo.UpdateSupplierContractStatus(b.Ctx, contrato.ID, "CANCELLED"); err == nil {
		t.Fatal("empresa B cancelou o contrato da A")
	}
	if _, err := repo.FindContractItem(b.Ctx, contrato.ID, supplier, ""); err == nil {
		t.Fatal("empresa B achou item do contrato da A")
	}
	if _, err := repo.ConsumeContractItem(b.Ctx, contrato.Items[0].ID, 1); err == nil {
		t.Fatal("empresa B consumiu saldo do contrato da A")
	}

	if _, err := repo.CreateSupplierScorecard(a.Ctx, &entity.SupplierScorecard{SupplierCode: supplier, PeriodStart: hoje, PeriodEnd: hoje}); err != nil {
		t.Fatal(err)
	}
	if l, err := repo.ListSupplierScorecards(b.Ctx, supplier); err != nil || len(l) != 0 {
		t.Fatalf("empresa B viu %d IQF da A (%v)", len(l), err)
	}
	if _, err := repo.CreateSupplierScorecard(b.Ctx, &entity.SupplierScorecard{SupplierCode: supplier, PeriodStart: hoje, PeriodEnd: hoje}); err != nil {
		t.Fatalf("o mesmo período do fornecedor tem de poder existir na empresa B: %v", err)
	}

	if _, err := repo.CreateHomologation(a.Ctx, &entity.SupplierHomologation{SupplierCode: supplier, Status: "HOMOLOGATED"}); err != nil {
		t.Fatal(err)
	}
	if l, err := repo.ListHomologations(b.Ctx, supplier); err != nil || len(l) != 0 {
		t.Fatalf("empresa B viu %d homologação(ões) da A (%v)", len(l), err)
	}
	if l, err := repo.ListHomologations(a.Ctx, supplier); err != nil || len(l) != 1 {
		t.Fatalf("empresa A deveria ver a própria homologação: %d (%v)", len(l), err)
	}
}

// Gerar vínculos item × fornecedor pelo histórico de compras (FFOR0204): o ator
// ia sem tipo no SELECT DISTINCT, virava texto e a coluna created_by é uuid —
// a rotina falhava para qualquer fornecedor.
func TestGeraVinculosItemFornecedorPeloHistorico(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids, codes := []int64{a.ID, b.ID}, []int64{a.Code, b.Code}
		pool.Exec(base, `DELETE FROM item_preferred_suppliers WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM purchase_order_items WHERE purchase_order_code IN (SELECT code FROM purchase_orders WHERE enterprise_code = ANY($1))`, codes)
		pool.Exec(base, `DELETE FROM purchase_orders WHERE enterprise_code = ANY($1)`, codes)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
	})
	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, actor)
	fornecedor := int64(1_690_000_000 + testutil.UniqueCode()%10_000_000)
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR',$2,$3,$4)`,
		fornecedor, fmt.Sprintf("%014d", fornecedor), actor, a.ID)
	var pedido int64
	if err := pool.QueryRow(base, `INSERT INTO purchase_orders(order_number,enterprise_code,created_by,supplier_code) VALUES($1,$2,$3,$4) RETURNING code`,
		testutil.UniqueCode(), a.Code, actor, fornecedor).Scan(&pedido); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO purchase_order_items(purchase_order_code,item_code) VALUES($1,$2)`, pedido, item)

	criados, err := procurement.New(pool).GenerateItemSuppliersFromHistory(a.Ctx, fornecedor, actor)
	if err != nil {
		t.Fatalf("gerando vínculos pelo histórico: %v", err)
	}
	if criados != 1 {
		t.Fatalf("%d vínculo(s) criado(s), esperado 1", criados)
	}
	if criados, err = procurement.New(pool).GenerateItemSuppliersFromHistory(a.Ctx, fornecedor, actor); err != nil || criados != 0 {
		t.Fatalf("segunda rodada criou %d (%v), esperado 0", criados, err)
	}
}
