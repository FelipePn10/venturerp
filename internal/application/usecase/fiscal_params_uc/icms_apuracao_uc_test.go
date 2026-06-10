package fiscal_params_uc

import (
	"context"
	"testing"

	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// ── period validator ──────────────────────────────────────────────────────────

func TestValidatePeriod(t *testing.T) {
	ok := []string{"2024-01", "2023-12", "2025-06"}
	for _, p := range ok {
		if err := validatePeriod(p); err != nil {
			t.Errorf("expected %q to be valid, got: %v", p, err)
		}
	}
	bad := []string{"", "2024", "2024-13", "24-01", "2024/01", "2024-00"}
	for _, p := range bad {
		if err := validatePeriod(p); err == nil {
			t.Errorf("expected %q to fail validation", p)
		}
	}
}

// ── stub repo ─────────────────────────────────────────────────────────────────

type stubFiscalRepo struct{}

// ensure interface satisfaction at compile time
var _ fiscalRepo = (*stubFiscalRepo)(nil)

// Use the package's repository interface type alias to avoid import cycle.
type fiscalRepo = interface {
	CreateLegalDevice(context.Context, *fiscalEntity.LegalDevice) (*fiscalEntity.LegalDevice, error)
	UpdateLegalDevice(context.Context, *fiscalEntity.LegalDevice) (*fiscalEntity.LegalDevice, error)
	GetLegalDeviceByCode(context.Context, int64) (*fiscalEntity.LegalDevice, error)
	ListLegalDevices(context.Context, bool) ([]*fiscalEntity.LegalDevice, error)
	ListLegalDevicesByType(context.Context, fiscalEntity.LegalDeviceType, bool) ([]*fiscalEntity.LegalDevice, error)
	NextLegalDeviceCode(context.Context) (int64, error)
	CreateCFOP(context.Context, *fiscalEntity.CFOP) (*fiscalEntity.CFOP, error)
	UpdateCFOP(context.Context, *fiscalEntity.CFOP) (*fiscalEntity.CFOP, error)
	GetCFOPByCode(context.Context, int32) (*fiscalEntity.CFOP, error)
	ListCFOPs(context.Context, bool) ([]*fiscalEntity.CFOP, error)
	ListCFOPsByDirection(context.Context, string, bool) ([]*fiscalEntity.CFOP, error)
	CreateTaxParam(context.Context, *fiscalEntity.ICMSIPITaxParam) (*fiscalEntity.ICMSIPITaxParam, error)
	UpdateTaxParam(context.Context, *fiscalEntity.ICMSIPITaxParam) (*fiscalEntity.ICMSIPITaxParam, error)
	GetTaxParamByID(context.Context, int64) (*fiscalEntity.ICMSIPITaxParam, error)
	ListTaxParams(context.Context, bool) ([]*fiscalEntity.ICMSIPITaxParam, error)
	ListTaxParamsByUF(context.Context, string, bool) ([]*fiscalEntity.ICMSIPITaxParam, error)
	ListTaxParamsByItem(context.Context, int64, bool) ([]*fiscalEntity.ICMSIPITaxParam, error)
	ListTaxParamsByNCM(context.Context, string, bool) ([]*fiscalEntity.ICMSIPITaxParam, error)
	CreateDAPITransferReason(context.Context, *fiscalEntity.DAPITransferReason) (*fiscalEntity.DAPITransferReason, error)
	UpdateDAPITransferReason(context.Context, *fiscalEntity.DAPITransferReason) (*fiscalEntity.DAPITransferReason, error)
	GetDAPITransferReasonByCode(context.Context, string) (*fiscalEntity.DAPITransferReason, error)
	ListDAPITransferReasons(context.Context, bool) ([]*fiscalEntity.DAPITransferReason, error)
	CreateICMSApuracaoAdjCode(context.Context, *fiscalEntity.ICMSApuracaoAdjustmentCode) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error)
	UpdateICMSApuracaoAdjCode(context.Context, *fiscalEntity.ICMSApuracaoAdjustmentCode) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error)
	GetICMSApuracaoAdjCode(context.Context, int64) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error)
	ListICMSApuracaoAdjCodes(context.Context, string, bool) ([]*fiscalEntity.ICMSApuracaoAdjustmentCode, error)
	CreateICMSAdjustmentCode(context.Context, *fiscalEntity.ICMSAdjustmentCode) (*fiscalEntity.ICMSAdjustmentCode, error)
	UpdateICMSAdjustmentCode(context.Context, *fiscalEntity.ICMSAdjustmentCode) (*fiscalEntity.ICMSAdjustmentCode, error)
	GetICMSAdjustmentCode(context.Context, int64) (*fiscalEntity.ICMSAdjustmentCode, error)
	ListICMSAdjustmentCodes(context.Context, string, string, bool) ([]*fiscalEntity.ICMSAdjustmentCode, error)
	CreateICMSApuracaoLine(context.Context, *fiscalEntity.ICMSApuracaoLine) (*fiscalEntity.ICMSApuracaoLine, error)
	UpdateICMSApuracaoLine(context.Context, *fiscalEntity.ICMSApuracaoLine) (*fiscalEntity.ICMSApuracaoLine, error)
	GetICMSApuracaoLine(context.Context, string) (*fiscalEntity.ICMSApuracaoLine, error)
	ListICMSApuracaoLines(context.Context, bool) ([]*fiscalEntity.ICMSApuracaoLine, error)
	CreateICMSSummaryEntry(context.Context, *fiscalEntity.ICMSSummaryEntry) (*fiscalEntity.ICMSSummaryEntry, error)
	UpdateICMSSummaryEntry(context.Context, *fiscalEntity.ICMSSummaryEntry) (*fiscalEntity.ICMSSummaryEntry, error)
	GetICMSSummaryEntry(context.Context, int64) (*fiscalEntity.ICMSSummaryEntry, error)
	ListICMSSummaryEntries(context.Context, string, string) ([]*fiscalEntity.ICMSSummaryEntry, error)
	AddICMSSummaryEntryNote(context.Context, *fiscalEntity.ICMSSummaryEntryNote) (*fiscalEntity.ICMSSummaryEntryNote, error)
	ListICMSSummaryEntryNotes(context.Context, int64) ([]*fiscalEntity.ICMSSummaryEntryNote, error)
	CreateSimplesNacionalApuracao(context.Context, *fiscalEntity.SimplesNacionalApuracao) (*fiscalEntity.SimplesNacionalApuracao, error)
	UpdateSimplesNacionalApuracao(context.Context, *fiscalEntity.SimplesNacionalApuracao) (*fiscalEntity.SimplesNacionalApuracao, error)
	GetSimplesNacionalApuracao(context.Context, string, fiscalEntity.SimplesNacionalAnnex) (*fiscalEntity.SimplesNacionalApuracao, error)
	ListSimplesNacionalApuracoes(context.Context, string) ([]*fiscalEntity.SimplesNacionalApuracao, error)
}

func (s *stubFiscalRepo) CreateLegalDevice(_ context.Context, d *fiscalEntity.LegalDevice) (*fiscalEntity.LegalDevice, error) {
	return d, nil
}
func (s *stubFiscalRepo) UpdateLegalDevice(_ context.Context, d *fiscalEntity.LegalDevice) (*fiscalEntity.LegalDevice, error) {
	return d, nil
}
func (s *stubFiscalRepo) GetLegalDeviceByCode(_ context.Context, id int64) (*fiscalEntity.LegalDevice, error) {
	return &fiscalEntity.LegalDevice{ID: id}, nil
}
func (s *stubFiscalRepo) ListLegalDevices(_ context.Context, _ bool) ([]*fiscalEntity.LegalDevice, error) {
	return nil, nil
}
func (s *stubFiscalRepo) ListLegalDevicesByType(_ context.Context, _ fiscalEntity.LegalDeviceType, _ bool) ([]*fiscalEntity.LegalDevice, error) {
	return nil, nil
}
func (s *stubFiscalRepo) NextLegalDeviceCode(_ context.Context) (int64, error) { return 1, nil }
func (s *stubFiscalRepo) CreateCFOP(_ context.Context, c *fiscalEntity.CFOP) (*fiscalEntity.CFOP, error) {
	return c, nil
}
func (s *stubFiscalRepo) UpdateCFOP(_ context.Context, c *fiscalEntity.CFOP) (*fiscalEntity.CFOP, error) {
	return c, nil
}
func (s *stubFiscalRepo) GetCFOPByCode(_ context.Context, _ int32) (*fiscalEntity.CFOP, error) {
	return &fiscalEntity.CFOP{}, nil
}
func (s *stubFiscalRepo) ListCFOPs(_ context.Context, _ bool) ([]*fiscalEntity.CFOP, error) {
	return nil, nil
}
func (s *stubFiscalRepo) ListCFOPsByDirection(_ context.Context, _ string, _ bool) ([]*fiscalEntity.CFOP, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateTaxParam(_ context.Context, tp *fiscalEntity.ICMSIPITaxParam) (*fiscalEntity.ICMSIPITaxParam, error) {
	return tp, nil
}
func (s *stubFiscalRepo) UpdateTaxParam(_ context.Context, tp *fiscalEntity.ICMSIPITaxParam) (*fiscalEntity.ICMSIPITaxParam, error) {
	return tp, nil
}
func (s *stubFiscalRepo) GetTaxParamByID(_ context.Context, _ int64) (*fiscalEntity.ICMSIPITaxParam, error) {
	return &fiscalEntity.ICMSIPITaxParam{}, nil
}
func (s *stubFiscalRepo) ListTaxParams(_ context.Context, _ bool) ([]*fiscalEntity.ICMSIPITaxParam, error) {
	return nil, nil
}
func (s *stubFiscalRepo) ListTaxParamsByUF(_ context.Context, _ string, _ bool) ([]*fiscalEntity.ICMSIPITaxParam, error) {
	return nil, nil
}
func (s *stubFiscalRepo) ListTaxParamsByItem(_ context.Context, _ int64, _ bool) ([]*fiscalEntity.ICMSIPITaxParam, error) {
	return nil, nil
}
func (s *stubFiscalRepo) ListTaxParamsByNCM(_ context.Context, _ string, _ bool) ([]*fiscalEntity.ICMSIPITaxParam, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateDAPITransferReason(_ context.Context, d *fiscalEntity.DAPITransferReason) (*fiscalEntity.DAPITransferReason, error) {
	return d, nil
}
func (s *stubFiscalRepo) UpdateDAPITransferReason(ctx context.Context, d *fiscalEntity.DAPITransferReason) (*fiscalEntity.DAPITransferReason, error) {
	return d, nil
}
func (s *stubFiscalRepo) GetDAPITransferReasonByCode(ctx context.Context, code string) (*fiscalEntity.DAPITransferReason, error) {
	return &fiscalEntity.DAPITransferReason{}, nil
}
func (s *stubFiscalRepo) ListDAPITransferReasons(ctx context.Context, onlyActive bool) ([]*fiscalEntity.DAPITransferReason, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateICMSApuracaoAdjCode(ctx context.Context, c *fiscalEntity.ICMSApuracaoAdjustmentCode) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error) {
	return c, nil
}
func (s *stubFiscalRepo) UpdateICMSApuracaoAdjCode(ctx context.Context, c *fiscalEntity.ICMSApuracaoAdjustmentCode) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error) {
	return c, nil
}
func (s *stubFiscalRepo) GetICMSApuracaoAdjCode(ctx context.Context, id int64) (*fiscalEntity.ICMSApuracaoAdjustmentCode, error) {
	return &fiscalEntity.ICMSApuracaoAdjustmentCode{}, nil
}
func (s *stubFiscalRepo) ListICMSApuracaoAdjCodes(ctx context.Context, uf string, onlyActive bool) ([]*fiscalEntity.ICMSApuracaoAdjustmentCode, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateICMSAdjustmentCode(ctx context.Context, c *fiscalEntity.ICMSAdjustmentCode) (*fiscalEntity.ICMSAdjustmentCode, error) {
	return c, nil
}
func (s *stubFiscalRepo) UpdateICMSAdjustmentCode(ctx context.Context, c *fiscalEntity.ICMSAdjustmentCode) (*fiscalEntity.ICMSAdjustmentCode, error) {
	return c, nil
}
func (s *stubFiscalRepo) GetICMSAdjustmentCode(ctx context.Context, id int64) (*fiscalEntity.ICMSAdjustmentCode, error) {
	return &fiscalEntity.ICMSAdjustmentCode{}, nil
}
func (s *stubFiscalRepo) ListICMSAdjustmentCodes(ctx context.Context, uf, tableRef string, onlyActive bool) ([]*fiscalEntity.ICMSAdjustmentCode, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateICMSApuracaoLine(ctx context.Context, l *fiscalEntity.ICMSApuracaoLine) (*fiscalEntity.ICMSApuracaoLine, error) {
	return l, nil
}
func (s *stubFiscalRepo) UpdateICMSApuracaoLine(ctx context.Context, l *fiscalEntity.ICMSApuracaoLine) (*fiscalEntity.ICMSApuracaoLine, error) {
	return l, nil
}
func (s *stubFiscalRepo) GetICMSApuracaoLine(ctx context.Context, code string) (*fiscalEntity.ICMSApuracaoLine, error) {
	return &fiscalEntity.ICMSApuracaoLine{}, nil
}
func (s *stubFiscalRepo) ListICMSApuracaoLines(ctx context.Context, onlyActive bool) ([]*fiscalEntity.ICMSApuracaoLine, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateICMSSummaryEntry(ctx context.Context, e *fiscalEntity.ICMSSummaryEntry) (*fiscalEntity.ICMSSummaryEntry, error) {
	return e, nil
}
func (s *stubFiscalRepo) UpdateICMSSummaryEntry(ctx context.Context, e *fiscalEntity.ICMSSummaryEntry) (*fiscalEntity.ICMSSummaryEntry, error) {
	return e, nil
}
func (s *stubFiscalRepo) GetICMSSummaryEntry(ctx context.Context, id int64) (*fiscalEntity.ICMSSummaryEntry, error) {
	return &fiscalEntity.ICMSSummaryEntry{}, nil
}
func (s *stubFiscalRepo) ListICMSSummaryEntries(ctx context.Context, period, uf string) ([]*fiscalEntity.ICMSSummaryEntry, error) {
	return nil, nil
}
func (s *stubFiscalRepo) AddICMSSummaryEntryNote(ctx context.Context, n *fiscalEntity.ICMSSummaryEntryNote) (*fiscalEntity.ICMSSummaryEntryNote, error) {
	return n, nil
}
func (s *stubFiscalRepo) ListICMSSummaryEntryNotes(ctx context.Context, summaryEntryID int64) ([]*fiscalEntity.ICMSSummaryEntryNote, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateSimplesNacionalApuracao(ctx context.Context, ap *fiscalEntity.SimplesNacionalApuracao) (*fiscalEntity.SimplesNacionalApuracao, error) {
	return ap, nil
}
func (s *stubFiscalRepo) UpdateSimplesNacionalApuracao(ctx context.Context, ap *fiscalEntity.SimplesNacionalApuracao) (*fiscalEntity.SimplesNacionalApuracao, error) {
	return ap, nil
}
func (s *stubFiscalRepo) GetSimplesNacionalApuracao(ctx context.Context, period string, annex fiscalEntity.SimplesNacionalAnnex) (*fiscalEntity.SimplesNacionalApuracao, error) {
	return &fiscalEntity.SimplesNacionalApuracao{}, nil
}
func (s *stubFiscalRepo) ListSimplesNacionalApuracoes(ctx context.Context, period string) ([]*fiscalEntity.SimplesNacionalApuracao, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateICMSReductionSubstitution(ctx context.Context, r *fiscalEntity.ICMSReductionSubstitution) (*fiscalEntity.ICMSReductionSubstitution, error) {
	return r, nil
}
func (s *stubFiscalRepo) UpdateICMSReductionSubstitution(ctx context.Context, r *fiscalEntity.ICMSReductionSubstitution) (*fiscalEntity.ICMSReductionSubstitution, error) {
	return r, nil
}
func (s *stubFiscalRepo) GetICMSReductionSubstitution(ctx context.Context, id int64) (*fiscalEntity.ICMSReductionSubstitution, error) {
	return &fiscalEntity.ICMSReductionSubstitution{}, nil
}
func (s *stubFiscalRepo) ListICMSReductionSubstitutions(ctx context.Context, uf string, itemID *int64, onlyActive bool) ([]*fiscalEntity.ICMSReductionSubstitution, error) {
	return nil, nil
}
func (s *stubFiscalRepo) FindICMSReductionSubstitution(ctx context.Context, uf string, itemID *int64, customerID *int64, opType fiscalEntity.ICMSOperationType) (*fiscalEntity.ICMSReductionSubstitution, error) {
	return nil, nil
}
func (s *stubFiscalRepo) AddICMSSummaryEntryAdditional(ctx context.Context, a *fiscalEntity.ICMSSummaryEntryAdditional) (*fiscalEntity.ICMSSummaryEntryAdditional, error) {
	return a, nil
}
func (s *stubFiscalRepo) ListICMSSummaryEntryAdditionals(ctx context.Context, summaryEntryID int64) ([]*fiscalEntity.ICMSSummaryEntryAdditional, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateICMSSTRestitution(ctx context.Context, r *fiscalEntity.ICMSSTRestitution) (*fiscalEntity.ICMSSTRestitution, error) {
	return r, nil
}
func (s *stubFiscalRepo) UpdateICMSSTRestitution(ctx context.Context, r *fiscalEntity.ICMSSTRestitution) (*fiscalEntity.ICMSSTRestitution, error) {
	return r, nil
}
func (s *stubFiscalRepo) GetICMSSTRestitution(ctx context.Context, id int64) (*fiscalEntity.ICMSSTRestitution, error) {
	return &fiscalEntity.ICMSSTRestitution{}, nil
}
func (s *stubFiscalRepo) ListICMSSTRestitutions(ctx context.Context, empresaID int, period string, uf string) ([]*fiscalEntity.ICMSSTRestitution, error) {
	return nil, nil
}
func (s *stubFiscalRepo) CreateSpecialAdjustmentNote(ctx context.Context, n *fiscalEntity.SpecialAdjustmentNote) (*fiscalEntity.SpecialAdjustmentNote, error) {
	return n, nil
}
func (s *stubFiscalRepo) UpdateSpecialAdjustmentNote(ctx context.Context, n *fiscalEntity.SpecialAdjustmentNote) (*fiscalEntity.SpecialAdjustmentNote, error) {
	return n, nil
}
func (s *stubFiscalRepo) GetSpecialAdjustmentNote(ctx context.Context, id int64) (*fiscalEntity.SpecialAdjustmentNote, error) {
	return &fiscalEntity.SpecialAdjustmentNote{}, nil
}
func (s *stubFiscalRepo) ListSpecialAdjustmentNotes(ctx context.Context, empresaID int, period string) ([]*fiscalEntity.SpecialAdjustmentNote, error) {
	return nil, nil
}
func (s *stubFiscalRepo) AddSpecialAdjustmentNoteItem(ctx context.Context, item *fiscalEntity.SpecialAdjustmentNoteItem) (*fiscalEntity.SpecialAdjustmentNoteItem, error) {
	return item, nil
}
func (s *stubFiscalRepo) ListSpecialAdjustmentNoteItems(ctx context.Context, noteID int64) ([]*fiscalEntity.SpecialAdjustmentNoteItem, error) {
	return nil, nil
}

var stub = &stubFiscalRepo{}

// ── DAPI ─────────────────────────────────────────────────────────────────────

func TestDAPICreate_MissingCode(t *testing.T) {
	uc := &DAPITransferReasonUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.DAPITransferReason{Reason: "test"})
	if err == nil {
		t.Fatal("expected error for missing code")
	}
}

func TestDAPICreate_MissingReason(t *testing.T) {
	uc := &DAPITransferReasonUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.DAPITransferReason{Code: "01"})
	if err == nil {
		t.Fatal("expected error for missing reason")
	}
}

func TestDAPICreate_OK(t *testing.T) {
	uc := &DAPITransferReasonUseCase{Repo: stub}
	d, err := uc.Create(context.Background(), &fiscalEntity.DAPITransferReason{Code: "01", Reason: "teste"})
	if err != nil {
		t.Fatal(err)
	}
	if !d.IsActive {
		t.Error("expected IsActive=true")
	}
}

// ── ICMSAdjCode ───────────────────────────────────────────────────────────────

func TestICMSAdjCode_InvalidTableRef(t *testing.T) {
	uc := &ICMSAdjustmentCodeUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.ICMSAdjustmentCode{
		UF: "SP", Code: "SP001", TableRef: "5.9",
	})
	if err == nil {
		t.Fatal("expected error for invalid table_ref")
	}
}

func TestICMSAdjCode_ValidTableRefs(t *testing.T) {
	uc := &ICMSAdjustmentCodeUseCase{Repo: stub}
	for _, ref := range []string{"5.2", "5.3", "5.6", "5.7"} {
		_, err := uc.Create(context.Background(), &fiscalEntity.ICMSAdjustmentCode{
			UF: "SP", Code: "SP001", TableRef: fiscalEntity.ICMSAdjustmentTableRef(ref),
		})
		if err != nil {
			t.Errorf("ref %q should be valid, got: %v", ref, err)
		}
	}
}

// ── ICMSApuracaoLine ──────────────────────────────────────────────────────────

func TestApuracaoLine_MissingCode(t *testing.T) {
	uc := &ICMSApuracaoLineUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.ICMSApuracaoLine{Description: "desc"})
	if err == nil {
		t.Fatal("expected error for missing code")
	}
}

func TestApuracaoLine_MissingDescription(t *testing.T) {
	uc := &ICMSApuracaoLineUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.ICMSApuracaoLine{Code: "E110"})
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestApuracaoLine_UnknownLineTypeDefaultsToOutros(t *testing.T) {
	uc := &ICMSApuracaoLineUseCase{Repo: stub}
	d, err := uc.Create(context.Background(), &fiscalEntity.ICMSApuracaoLine{
		Code: "E110", Description: "saldo", LineType: "INVALIDO",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.LineType != string(fiscalEntity.LineTypeOutros) {
		t.Errorf("expected LineTypeOutros, got %v", d.LineType)
	}
}

// ── ICMSSummaryEntry ──────────────────────────────────────────────────────────

func TestSummaryEntry_BadPeriod(t *testing.T) {
	uc := &ICMSSummaryEntryUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.ICMSSummaryEntry{
		Period: "2024/01", UF: "SP",
	})
	if err == nil {
		t.Fatal("expected error for bad period")
	}
}

func TestSummaryEntry_NegativeICMS(t *testing.T) {
	uc := &ICMSSummaryEntryUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.ICMSSummaryEntry{
		Period: "2024-01", UF: "SP", ICMSValue: -10,
	})
	if err == nil {
		t.Fatal("expected error for negative icms_value")
	}
}

func TestSummaryEntry_OK(t *testing.T) {
	uc := &ICMSSummaryEntryUseCase{Repo: stub}
	e, err := uc.Create(context.Background(), &fiscalEntity.ICMSSummaryEntry{
		Period: "2024-01", UF: "SP", ICMSBase: 10000, ICMSValue: 1200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !e.IsActive {
		t.Error("expected IsActive=true")
	}
}

// ── SimplesNacional ───────────────────────────────────────────────────────────

func TestSimples_BadPeriod(t *testing.T) {
	uc := &SimplesNacionalUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.SimplesNacionalApuracao{
		Period: "01/2024", Annex: "I",
	})
	if err == nil {
		t.Fatal("expected error for bad period")
	}
}

func TestSimples_InvalidAnnex(t *testing.T) {
	uc := &SimplesNacionalUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.SimplesNacionalApuracao{
		Period: "2024-01", Annex: "VII",
	})
	if err == nil {
		t.Fatal("expected error for invalid annex")
	}
}

func TestSimples_NegativeReceita(t *testing.T) {
	uc := &SimplesNacionalUseCase{Repo: stub}
	_, err := uc.Create(context.Background(), &fiscalEntity.SimplesNacionalApuracao{
		Period: "2024-01", Annex: "I", ReceitaInterna: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative receita")
	}
}

func TestSimples_OK(t *testing.T) {
	uc := &SimplesNacionalUseCase{Repo: stub}
	s, err := uc.Create(context.Background(), &fiscalEntity.SimplesNacionalApuracao{
		Period: "2024-06", Annex: "III", ReceitaInterna: 50000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !s.IsActive {
		t.Error("expected IsActive=true")
	}
}
