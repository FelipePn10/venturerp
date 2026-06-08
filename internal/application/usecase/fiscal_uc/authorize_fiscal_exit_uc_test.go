package fiscal_uc

import (
	"context"
	"testing"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/google/uuid"
)

func strp(s string) *string { return &s }
func i64p(i int64) *int64   { return &i }

// ─── buildFocusItems: fiscal field mapping ──────────────────────────────────

func TestBuildFocusItems_DefaultsAndScaling(t *testing.T) {
	items := []*fiscalentity.FiscalExitItem{
		{
			ItemCode:   i64p(1001),
			Ncm:        strp("72142000"),
			Cfop:       "5101",
			Quantity:   2,
			UnitPrice:  100,
			TotalPrice: 200,
			BaseICMS:   200,
			AliqICMS:   0.18, // stored as fraction; must be emitted as 18
			ValorICMS:  36,
			AliqIPI:    0.05,
			AliqPIS:    0.0165,
			AliqCOFINS: 0.076,
			// CST pointers nil → defaults
			OrigemMercadoria: "3",
		},
	}
	cfg := &fiscalentity.FiscalConfig{}

	out := buildFocusItems(items, cfg)
	if len(out) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out))
	}
	it := out[0]
	if it.NumeroItem != 1 {
		t.Fatalf("NumeroItem = %d, want 1", it.NumeroItem)
	}
	if it.CodigoSituacaoTributariaICMS != "00" || it.CodigoSituacaoTributariaIPI != "50" ||
		it.CodigoSituacaoTributariaPIS != "01" || it.CodigoSituacaoTributariaCOFINS != "01" {
		t.Fatalf("CST defaults wrong: %+v", it)
	}
	if it.AliquotaICMS != 18 || it.AliquotaIPI != 5 {
		t.Fatalf("aliquota scaling wrong: icms=%v ipi=%v (want 18/5)", it.AliquotaICMS, it.AliquotaIPI)
	}
	if it.OrigemMercadoria != 3 {
		t.Fatalf("origem = %d, want 3", it.OrigemMercadoria)
	}
	if it.CodigoNCM != "72142000" || it.CFOP != "5101" {
		t.Fatalf("ncm/cfop wrong: %s / %s", it.CodigoNCM, it.CFOP)
	}
	// No description provided → fallback "Produto <code>".
	if it.Descricao != "Produto 1001" {
		t.Fatalf("description fallback = %q", it.Descricao)
	}
}

func TestBuildFocusItems_RespectsExplicitCSTAndDescription(t *testing.T) {
	items := []*fiscalentity.FiscalExitItem{{
		ItemCode:    i64p(9),
		Cfop:        "6101",
		Description: strp("Chapa de aço 3mm"),
		CstICMS:     strp("20"),
		CstIPI:      strp("99"),
		CstPIS:      strp("06"),
		CstCOFINS:   strp("06"),
	}}
	out := buildFocusItems(items, &fiscalentity.FiscalConfig{})
	it := out[0]
	if it.Descricao != "Chapa de aço 3mm" {
		t.Fatalf("description = %q", it.Descricao)
	}
	if it.CodigoSituacaoTributariaICMS != "20" || it.CodigoSituacaoTributariaIPI != "99" ||
		it.CodigoSituacaoTributariaPIS != "06" || it.CodigoSituacaoTributariaCOFINS != "06" {
		t.Fatalf("explicit CST not respected: %+v", it)
	}
}

func TestBuildFocusItems_SubstituicaoTributaria(t *testing.T) {
	items := []*fiscalentity.FiscalExitItem{{
		ItemCode:    i64p(5),
		Cfop:        "5401",
		CstICMS:     strp("10"),
		BaseICMSST:  300,
		AliqICMSST:  0.18,
		ValorICMSST: 54,
		MVA:         0.40,
	}}
	out := buildFocusItems(items, &fiscalentity.FiscalConfig{})
	it := out[0]
	if it.ModalidadeBaseCalculoICMSST == nil || *it.ModalidadeBaseCalculoICMSST != 4 {
		t.Fatalf("ST modalidade should be 4 (MVA)")
	}
	if it.PercentualMVAICMSST == nil || *it.PercentualMVAICMSST != 40 {
		t.Fatalf("MVA should be scaled to 40, got %v", it.PercentualMVAICMSST)
	}
	if it.AliquotaICMSST == nil || *it.AliquotaICMSST != 18 {
		t.Fatalf("ST aliquota should be scaled to 18")
	}
	if it.ValorICMSST == nil || *it.ValorICMSST != 54 {
		t.Fatalf("ST value should be 54")
	}
}

func TestBuildFocusItems_DiferimentoCST51(t *testing.T) {
	items := []*fiscalentity.FiscalExitItem{{
		ItemCode:          i64p(7),
		Cfop:              "5101",
		CstICMS:           strp("51"),
		ValorICMSDiferido: 12,
	}}
	cfg := &fiscalentity.FiscalConfig{IcmsDiferimentoPercentual: 0.33}
	out := buildFocusItems(items, cfg)
	it := out[0]
	if it.PercentualDiferimento == nil || *it.PercentualDiferimento != 33 {
		t.Fatalf("diferimento pct should be scaled to 33, got %v", it.PercentualDiferimento)
	}
	if it.ValorICMSDiferido == nil || *it.ValorICMSDiferido != 12 {
		t.Fatalf("diferido value should be 12")
	}
}

// ─── settleStockAndOrder: stock write-down + order settlement ────────────────

type fakeStockRepo struct {
	stockrepo.StockRepository
	movements    []*stockentity.StockMovement
	reservations []*stockentity.StockReservation
	consumed     []int64
}

func (f *fakeStockRepo) CreateMovement(_ context.Context, m *stockentity.StockMovement) (*stockentity.StockMovement, error) {
	f.movements = append(f.movements, m)
	return m, nil
}
func (f *fakeStockRepo) ListActiveReservations(context.Context) ([]*stockentity.StockReservation, error) {
	return f.reservations, nil
}
func (f *fakeStockRepo) ConsumeReservation(_ context.Context, id int64) error {
	f.consumed = append(f.consumed, id)
	return nil
}

type fakeSalesRepo struct {
	salesrepo.SalesOrderRepository
	items   []*salesentity.SalesOrderItem
	changed map[int64]salesentity.SalesOrderStatus
}

func (f *fakeSalesRepo) ListItems(context.Context, int64) ([]*salesentity.SalesOrderItem, error) {
	return f.items, nil
}
func (f *fakeSalesRepo) ChangeStatus(_ context.Context, code int64, status salesentity.SalesOrderStatus) error {
	if f.changed == nil {
		f.changed = map[int64]salesentity.SalesOrderStatus{}
	}
	f.changed[code] = status
	return nil
}

func TestSettleStockAndOrder_WritesDownResolvableItemsOnly(t *testing.T) {
	stock := &fakeStockRepo{
		reservations: []*stockentity.StockReservation{
			{ID: 42, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: 500},
			{ID: 99, ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: 777}, // other order
		},
	}
	sales := &fakeSalesRepo{
		items: []*salesentity.SalesOrderItem{
			{ItemCode: 1001, WarehouseCode: i64p(7)},
			{ItemCode: 1002, WarehouseCode: i64p(8)},
			// 1003 has no warehouse → item should be skipped
		},
	}

	uc := AuthorizeFiscalExitUseCase{StockRepo: stock, SalesOrderRepo: sales}
	exit := &fiscalentity.FiscalExit{ID: 99, SalesOrderCode: i64p(500)}
	items := []*fiscalentity.FiscalExitItem{
		{ItemCode: i64p(1001), Quantity: 5, UnitPrice: 10, TotalPrice: 50},
		{ItemCode: i64p(1002), Quantity: 3, UnitPrice: 20, TotalPrice: 60},
		{ItemCode: i64p(1003), Quantity: 1}, // no warehouse → skipped
		{ItemCode: nil, Quantity: 9},        // nil code → skipped
	}

	uc.settleStockAndOrder(context.Background(), exit, items, uuid.New())

	if len(stock.movements) != 2 {
		t.Fatalf("expected 2 OUT movements (resolvable items), got %d", len(stock.movements))
	}
	for _, m := range stock.movements {
		if m.MovementType != stockentity.MovementTypeOut {
			t.Fatalf("movement must be OUT, got %s", m.MovementType)
		}
		if m.ReferenceType == nil || *m.ReferenceType != stockentity.ReferenceTypeNFExit {
			t.Fatalf("reference type must be NF_SAIDA")
		}
	}
	// Warehouse resolution: 1001→7, 1002→8.
	byItem := map[int64]int64{}
	for _, m := range stock.movements {
		byItem[m.ItemCode] = m.WarehouseID
	}
	if byItem[1001] != 7 || byItem[1002] != 8 {
		t.Fatalf("warehouse resolution wrong: %+v", byItem)
	}

	// Only the reservation tied to order 500 is consumed.
	if len(stock.consumed) != 1 || stock.consumed[0] != 42 {
		t.Fatalf("consumed reservations = %v, want [42]", stock.consumed)
	}

	// The linked order is flagged invoiced.
	if got := sales.changed[500]; got != salesentity.SalesOrderStatusInvoiced {
		t.Fatalf("order 500 status = %v, want invoiced", got)
	}
}

func TestSettleStockAndOrder_NoStockRepoIsNoOp(t *testing.T) {
	sales := &fakeSalesRepo{}
	uc := AuthorizeFiscalExitUseCase{SalesOrderRepo: sales} // StockRepo nil
	exit := &fiscalentity.FiscalExit{ID: 1, SalesOrderCode: i64p(10)}

	// Must not panic and should still flag the order invoiced.
	uc.settleStockAndOrder(context.Background(), exit, []*fiscalentity.FiscalExitItem{
		{ItemCode: i64p(1), Quantity: 1},
	}, uuid.New())

	if sales.changed[10] != salesentity.SalesOrderStatusInvoiced {
		t.Fatal("order should be flagged invoiced even without a stock repo")
	}
}
