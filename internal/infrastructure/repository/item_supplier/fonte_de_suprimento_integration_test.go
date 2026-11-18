//go:build integration

package item_supplier_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	repo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_supplier"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

// Fonte de suprimento (migração 000502) com DUAS empresas: cota, bloqueio com
// histórico, o que a NF-e aprendeu, e a trava nos documentos de compra.
func TestFonteDeSuprimentoDuasEmpresas(t *testing.T) {
	q, pool := testutil.Queries(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids, codes := []int64{a.ID, b.ID}, []int64{a.Code, b.Code}
		pool.Exec(base, `DELETE FROM purchase_order_items WHERE purchase_order_code IN (SELECT code FROM purchase_orders WHERE enterprise_code = ANY($1))`, codes)
		pool.Exec(base, `DELETE FROM purchase_orders WHERE enterprise_code = ANY($1)`, codes)
		pool.Exec(base, `DELETE FROM purchase_requisition_items WHERE requisition_code IN (SELECT code FROM purchase_requisitions WHERE enterprise_code = ANY($1))`, codes)
		pool.Exec(base, `DELETE FROM purchase_requisitions WHERE enterprise_code = ANY($1)`, codes)
		pool.Exec(base, `DELETE FROM procurement_parameters WHERE enterprise_code = ANY($1)`, codes)
		pool.Exec(base, `DELETE FROM item_preferred_suppliers WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
	})
	r := repo.New(q, pool)

	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, ator)
	forn := func(e testutil.Empresa) int64 {
		c := 1_700_000_000 + testutil.UniqueCode()%100_000_000
		testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`,
			c, fmt.Sprintf("FORN %d", c), fmt.Sprintf("%014d", c), ator, e.ID)
		return c
	}
	f1, f2, f3 := forn(a), forn(a), forn(a)

	novo := func(fornecedor int64, ranking int32) *entity.ItemPreferredSupplier {
		s, err := entity.NewItemPreferredSupplier(a.ID, item, fornecedor, "", ranking, ator)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	pct := func(v int64) *decimal.Decimal { d := decimal.NewFromInt(v); return &d }

	// ── Cadastro: cota somando até 100%, política de compra e prazo.
	s1 := novo(f1, 1)
	s1.IsPreferred, s1.QuotaPercent, s1.MinOrderQty, s1.LeadTimeDays = true, pct(70), decimal.NewFromInt(50), 12
	v1, err := r.Upsert(a.Ctx, s1)
	if err != nil {
		t.Fatal(err)
	}
	s2 := novo(f2, 2)
	s2.QuotaPercent, s2.OrderMultiple = pct(30), decimal.NewFromInt(12)
	if _, err = r.Upsert(a.Ctx, s2); err != nil {
		t.Fatal(err)
	}
	s3 := novo(f3, 3)
	s3.QuotaPercent = pct(10)
	if _, err = r.Upsert(a.Ctx, s3); err == nil || !strings.Contains(err.Error(), "100%") {
		t.Fatalf("cota somando 110%% aceita: %v", err)
	}
	s3.QuotaPercent = nil
	if _, err = r.Upsert(a.Ctx, s3); err != nil {
		t.Fatal(err)
	}

	// ── O que a NF-e aprendeu não some quando a tela regrava o vínculo.
	testutil.Exec(t, pool, `UPDATE item_preferred_suppliers SET supplier_item_code='ABC-77', supplier_description='PARAFUSO M8', uom='CX' WHERE id=$1`, v1.ID)
	s1.Ranking = 1
	regravado, err := r.Upsert(a.Ctx, s1)
	if err != nil {
		t.Fatal(err)
	}
	if regravado.SupplierItemCode == nil || *regravado.SupplierItemCode != "ABC-77" || regravado.SupplierDescription == nil || regravado.UOM == nil {
		t.Fatalf("regravar apagou o aprendido: %+v", regravado)
	}
	if !regravado.MinOrderQty.Equal(decimal.NewFromInt(50)) || regravado.LeadTimeDays != 12 || regravado.QuotaPercent == nil {
		t.Fatalf("política não gravada: %+v", regravado)
	}

	// ── A empresa B não lista, não bloqueia, não limpa nada da A (nem pelo id).
	if l, err := r.ListByItem(b.Ctx, b.ID, item); err != nil || len(l) != 0 {
		t.Fatalf("B listou %d vínculo(s) da A: %v", len(l), err)
	}
	ev, _ := entity.NewBlockEvent(b.ID, v1.ID, entity.AcaoBloqueio, "invasão", ator)
	if _, err := r.SetBlocked(b.Ctx, ev); err == nil {
		t.Fatal("B bloqueou o vínculo da A")
	}
	if _, err := r.ClearLearned(b.Ctx, b.ID, v1.ID); err == nil {
		t.Fatal("B limpou o código aprendido da A")
	}
	if h, err := r.ListBlockHistory(b.Ctx, b.ID, v1.ID); err != nil || len(h) != 0 {
		t.Fatalf("B leu o histórico da A: %d %v", len(h), err)
	}

	// ── MRP: fontes válidas, com o alocado; a B não vê as fontes da A.
	mrp := mrprepo.NewMRPCalculationRepositorySQLC(q, pool)
	fontes, err := mrp.FontesDeCompra(a.Ctx, item)
	if err != nil || len(fontes) != 3 || fontes[0].SupplierCode != f1 || fontes[0].LeadTimeDays != 12 {
		t.Fatalf("fontes da A: %+v %v", fontes, err)
	}
	if fb, err := mrp.FontesDeCompra(b.Ctx, item); err != nil || len(fb) != 0 {
		t.Fatalf("B viu %d fonte(s) da A: %v", len(fb), err)
	}

	// ── Trava: pedido só com fonte e o fornecedor do pedido tem de ser fonte.
	// A requisição é livre (pedido interno; nasce sozinha ao firmar a OF).
	regra := func(err error) bool {
		var pg *pgconn.PgError
		return errors.As(err, &pg) && pg.Code == "P0001"
	}
	semFonte := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, semFonte, ator)
	reqCode := testutil.UniqueCode() % 1_000_000_000
	testutil.Exec(t, pool, `INSERT INTO purchase_requisitions(code,enterprise_code,created_by) VALUES($1,$2,$3)`, reqCode, a.Code, ator)
	if _, err := pool.Exec(base, `INSERT INTO purchase_requisition_items(requisition_code,item_code,quantity) VALUES($1,$2,1)`, reqCode, semFonte); err != nil {
		t.Fatalf("requisição de item sem fornecedor deveria ser livre: %v", err)
	}
	pedido := func(fornecedor int64) int64 {
		var code int64
		if err := pool.QueryRow(base, `INSERT INTO purchase_orders(order_number,enterprise_code,supplier_code,created_by) VALUES($1,$2,$3,$4) RETURNING code`,
			testutil.UniqueCode()%1_000_000, a.Code, fornecedor, ator).Scan(&code); err != nil {
			t.Fatal(err)
		}
		return code
	}
	linha := func(po int64) error {
		_, err := pool.Exec(base, `INSERT INTO purchase_order_items(purchase_order_code,item_code,requested_qty) VALUES($1,$2,1)`, po, item)
		return err
	}
	estranho := forn(a)
	if err := linha(pedido(estranho)); !regra(err) {
		t.Fatalf("pedido com fornecedor que não é fonte aceito: %v", err)
	}
	linhaSemFonte := func(po int64) error {
		_, err := pool.Exec(base, `INSERT INTO purchase_order_items(purchase_order_code,item_code,requested_qty) VALUES($1,$2,1)`, po, semFonte)
		return err
	}
	if err := linhaSemFonte(pedido(estranho)); !regra(err) {
		t.Fatalf("pedido de item sem fornecedor aceito: %v", err)
	}
	poF2 := pedido(f2)
	if err := linha(poF2); err != nil {
		t.Fatalf("pedido com fonte recusado: %v", err)
	}
	if _, err := pool.Exec(base, `UPDATE purchase_orders SET supplier_code=$2 WHERE code=$1`, poF2, estranho); !regra(err) {
		t.Fatalf("trocou o fornecedor do pedido para um que não é fonte: %v", err)
	}

	// ── Bloqueio com motivo: deixa de ser fonte; histórico guarda os dois eventos.
	ev, _ = entity.NewBlockEvent(a.ID, v1.ID, entity.AcaoBloqueio, "atraso recorrente", ator)
	bloq, err := r.SetBlocked(a.Ctx, ev)
	if err != nil || !bloq.Blocked || bloq.BlockedReason == nil || bloq.Situacao(time.Now()) != entity.SituacaoBloqueado {
		t.Fatalf("bloqueio: %+v %v", bloq, err)
	}
	if _, err := r.SetBlocked(a.Ctx, ev); err == nil {
		t.Fatal("bloqueou duas vezes")
	}
	if err := linha(pedido(f1)); !regra(err) {
		t.Fatalf("pedido com fornecedor bloqueado aceito: %v", err)
	}
	if pref, err := r.GetPreferred(a.Ctx, a.ID, item); err != nil || pref == nil || pref.SupplierCode == f1 {
		t.Fatalf("preferencial bloqueado ainda é o preferencial: %+v %v", pref, err)
	}
	ev, _ = entity.NewBlockEvent(a.ID, v1.ID, entity.AcaoDesbloqueio, "plano de ação aprovado", ator)
	if _, err := r.SetBlocked(a.Ctx, ev); err != nil {
		t.Fatal(err)
	}
	if h, err := r.ListBlockHistory(a.Ctx, a.ID, v1.ID); err != nil || len(h) != 2 || h[0].Action != entity.AcaoDesbloqueio {
		t.Fatalf("histórico: %+v %v", h, err)
	}
	if err := linha(pedido(f1)); err != nil {
		t.Fatalf("desbloqueado continua barrado: %v", err)
	}

	// ── Vigência: vínculo vencido não é fonte.
	testutil.Exec(t, pool, `UPDATE item_preferred_suppliers SET valid_until=$2::date WHERE id=$1`, v1.ID, time.Now().AddDate(0, 0, -10))
	if err := linha(pedido(f1)); !regra(err) {
		t.Fatalf("pedido com vínculo vencido aceito: %v", err)
	}

	// ── Limpar o aprendido.
	limpo, err := r.ClearLearned(a.Ctx, a.ID, v1.ID)
	if err != nil || limpo.SupplierItemCode != nil || limpo.SupplierDescription != nil {
		t.Fatalf("limpar: %+v %v", limpo, err)
	}

	// ── A empresa pode desligar a exigência.
	testutil.Exec(t, pool, `INSERT INTO procurement_parameters(enterprise_code,domain,param_key,param_value,value_type) VALUES($1,'SUPPLIER','EXIGE_FONTE_DE_SUPRIMENTO','N','BOOL')`, a.Code)
	if err := linhaSemFonte(pedido(estranho)); err != nil {
		t.Fatalf("com a exigência desligada, o pedido foi recusado: %v", err)
	}

	var validacao bool
	if _, err := r.Upsert(a.Ctx, &entity.ItemPreferredSupplier{EnterpriseID: b.ID, ItemCode: item, SupplierCode: f2, Ranking: 1, CreatedBy: ator}); err != nil {
		_, validacao = errorsuc.AsNotFound(err)
	}
	if !validacao {
		t.Fatal("B vinculou fornecedor ao item da A")
	}
}
