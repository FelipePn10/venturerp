package fiscal_uc

import (
	"context"
	"testing"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
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

// ─── fakes dos efeitos da NF-e (ver efeitos_nfe_test.go) ─────────────────────

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

func (f *fakeSalesRepo) GetByCode(_ context.Context, code int64) (*salesentity.SalesOrder, error) {
	return &salesentity.SalesOrder{Code: code, Status: salesentity.SalesOrderStatusOrder}, nil
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
