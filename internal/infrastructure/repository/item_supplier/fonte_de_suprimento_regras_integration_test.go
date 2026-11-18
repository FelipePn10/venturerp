//go:build integration

package item_supplier_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	entity_mrp "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	repo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_supplier"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/shopspring/decimal"
)

// Regras finas da fonte de suprimento: cota reservada pelo bloqueado, fonte
// por máscara para a geração de pedidos e a janela comum da cota no MRP.
func TestFonteDeSuprimentoRegras(t *testing.T) {
	q, pool := testutil.Queries(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids, codes := []int64{a.ID, b.ID}, []int64{a.Code, b.Code}
		pool.Exec(base, `DELETE FROM planned_orders WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM purchase_order_items WHERE purchase_order_code IN (SELECT code FROM purchase_orders WHERE enterprise_code = ANY($1))`, codes)
		pool.Exec(base, `DELETE FROM purchase_orders WHERE enterprise_code = ANY($1)`, codes)
		pool.Exec(base, `DELETE FROM procurement_parameters WHERE enterprise_code = ANY($1)`, codes)
		pool.Exec(base, `DELETE FROM item_supplier_block_history WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM item_preferred_suppliers WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM items WHERE enterprise_id = ANY($1)`, ids)
		pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, ids)
	})
	r := repo.New(q, pool)

	item := testutil.UniqueCode()
	testutil.SeedItem(t, pool, a.Ctx, item, ator)
	forn := func() int64 {
		c := 1_700_000_000 + testutil.UniqueCode()%100_000_000
		testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`,
			c, fmt.Sprintf("FORN %d", c), fmt.Sprintf("%014d", c), ator, a.ID)
		return c
	}
	f1, f2, f3 := forn(), forn(), forn()
	pct := func(v int64) *decimal.Decimal { d := decimal.NewFromInt(v); return &d }
	hoje := time.Now()
	dia := func(d int) time.Time {
		return time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
	}
	salvar := func(fornecedor int64, mask string, ranking int32, cota *decimal.Decimal, preferencial bool, desde time.Time) *entity.ItemPreferredSupplier {
		t.Helper()
		s, err := entity.NewItemPreferredSupplier(a.ID, item, fornecedor, mask, ranking, ator)
		if err != nil {
			t.Fatal(err)
		}
		s.QuotaPercent, s.IsPreferred, s.ValidFrom = cota, preferencial, &desde
		v, err := r.Upsert(a.Ctx, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	// f1 com cota há mais de um ano; f2 entrou há 10 dias.
	v1 := salvar(f1, "", 1, pct(60), true, dia(-400))
	v2 := salvar(f2, "", 2, pct(40), false, dia(-10))
	// As cotas foram definidas na entrada de cada um (o cadastro grava hoje).
	testutil.Exec(t, pool, `UPDATE item_preferred_suppliers SET quota_since=valid_from WHERE id = ANY($1)`, []int64{v1.ID, v2.ID})

	// ── Janela comum: o histórico de f1 anterior à entrada de f2 não conta.
	po := func(fornecedor int64, emissao time.Time, qtd int64) {
		t.Helper()
		var code int64
		if err := pool.QueryRow(base, `INSERT INTO purchase_orders(order_number,enterprise_code,supplier_code,created_by,emission_date) VALUES($1,$2,$3,$4,$5::date) RETURNING code`,
			testutil.UniqueCode()%1_000_000, a.Code, fornecedor, ator, emissao).Scan(&code); err != nil {
			t.Fatal(err)
		}
		testutil.Exec(t, pool, `INSERT INTO purchase_order_items(purchase_order_code,item_code,requested_qty) VALUES($1,$2,$3)`, code, item, qtd)
	}
	po(f1, dia(-200), 1000) // antes da janela
	po(f1, dia(-2), 30)     // dentro
	// Ordem de compra firmada para f2, ainda sem pedido: já conta para f2.
	testutil.Exec(t, pool, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,supplier_code,is_firm,enterprise_id)
		VALUES($1,$2,20,'PURCHASE','REPLENISHMENT',$3::date,$4,0,$5,TRUE,$6)`, testutil.UniqueCode()%1_000_000_000, item, dia(5), ator, f2, a.ID)
	mrp := mrprepo.NewMRPCalculationRepositorySQLC(q, pool)
	fontes, err := mrp.FontesDeCompra(a.Ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	alocado := map[int64]decimal.Decimal{}
	for _, f := range fontes {
		alocado[f.SupplierCode] = f.Alocado
	}
	if !alocado[f1].Equal(decimal.NewFromInt(30)) || !alocado[f2].Equal(decimal.NewFromInt(20)) {
		t.Fatalf("alocado: f1=%s f2=%s (esperado 30 — só dentro da janela comum — e 20 da ordem firmada)", alocado[f1], alocado[f2])
	}

	// ── Regravar sem mudar a cota não recomeça a conta; mudar recomeça.
	salvar(f1, "", 1, pct(60), true, dia(-400))
	if f, _ := mrp.FontesDeCompra(a.Ctx, item); !alocadoDe(f, f1).Equal(decimal.NewFromInt(30)) {
		t.Fatalf("regravar a mesma cota recomeçou a conta: %s", alocadoDe(f, f1))
	}
	salvar(f1, "", 1, pct(55), true, dia(-400))
	if f, _ := mrp.FontesDeCompra(a.Ctx, item); !alocadoDe(f, f1).IsZero() || !alocadoDe(f, f2).Equal(decimal.NewFromInt(20)) {
		t.Fatalf("mudar a cota não recomeçou a conta: f1=%s f2=%s", alocadoDe(f, f1), alocadoDe(f, f2))
	}
	salvar(f1, "", 1, pct(60), true, dia(-400))

	// ── Geral + máscara não passam de 100%: 60 + 40 gerais, +10 na M1 é recusado.
	sm, _ := entity.NewItemPreferredSupplier(a.ID, item, f3, "M1", 3, ator)
	sm.QuotaPercent = pct(10)
	if _, err = r.Upsert(a.Ctx, sm); err == nil || !strings.Contains(err.Error(), "100%") {
		t.Fatalf("cota da máscara somada às gerais passou de 100%%: %v", err)
	}

	// ── Cota do bloqueado continua reservada: 60 + 40 (bloqueado) + 10 > 100.
	if _, err = r.SetBlocked(a.Ctx, &entity.BlockEvent{EnterpriseID: a.ID, ItemSupplierID: v2.ID, Action: entity.AcaoBloqueio, Reason: "atraso", ActorID: ator}); err != nil {
		t.Fatal(err)
	}
	s3, _ := entity.NewItemPreferredSupplier(a.ID, item, f3, "", 3, ator)
	s3.QuotaPercent = pct(10)
	if _, err = r.Upsert(a.Ctx, s3); err == nil || !strings.Contains(err.Error(), "100%") {
		t.Fatalf("cota somando 110%% com um bloqueado foi aceita: %v", err)
	}

	// ── Fonte do item por máscara (geração de pedidos a partir de requisição).
	salvar(f3, "M1", 3, nil, false, dia(-1))
	if got, ok, err := r.FonteDoItem(a.Ctx, a.ID, item, "M2", nil); err != nil || !ok || got != f1 {
		t.Fatalf("sem informado deveria vir o preferencial f1: %d %v %v", got, ok, err)
	}
	if _, ok, err := r.FonteDoItem(a.Ctx, a.ID, item, "M2", &f3); err != nil || ok {
		t.Fatalf("f3 só é fonte da máscara M1 e foi aceito em M2: %v %v", ok, err)
	}
	if got, ok, err := r.FonteDoItem(a.Ctx, a.ID, item, "M1", &f3); err != nil || !ok || got != f3 {
		t.Fatalf("f3 é fonte da máscara M1: %d %v %v", got, ok, err)
	}
	if _, ok, err := r.FonteDoItem(a.Ctx, a.ID, item, "", &f2); err != nil || ok {
		t.Fatalf("fornecedor bloqueado aceito como fonte: %v %v", ok, err)
	}
	// Empresa B não enxerga a fonte da A.
	if _, ok, err := r.FonteDoItem(b.Ctx, b.ID, item, "", nil); err != nil || ok {
		t.Fatalf("empresa B achou fonte no item da A: %v %v", ok, err)
	}
	// Exigência desligada: o informado vale mesmo sem vínculo.
	testutil.Exec(t, pool, `INSERT INTO procurement_parameters(enterprise_code,domain,param_key,param_value,value_type) VALUES($1,'SUPPLIER','EXIGE_FONTE_DE_SUPRIMENTO','N','BOOL')`, a.Code)
	if got, ok, err := r.FonteDoItem(a.Ctx, a.ID, item, "", &f2); err != nil || !ok || got != f2 {
		t.Fatalf("com a exigência desligada o informado deveria valer: %d %v %v", got, ok, err)
	}
}

func alocadoDe(fontes []entity_mrp.FonteDeCompra, fornecedor int64) decimal.Decimal {
	for _, f := range fontes {
		if f.SupplierCode == fornecedor {
			return f.Alocado
		}
	}
	return decimal.Zero
}
