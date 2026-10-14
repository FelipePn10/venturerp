//go:build integration

package fiscal_test

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// TestFiscalParamsAreIsolatedByTenant: parametrização fiscal, apurações e
// cenários não tinham empresa — uma empresa lia e alterava os da outra.
func TestFiscalParamsAreIsolatedByTenant(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ctx := context.Background()
		ids := []int64{a.ID, b.ID}
		for _, q := range []string{
			`DELETE FROM icms_ipi_tax_params WHERE enterprise_id = ANY($1)`,
			`DELETE FROM icms_reduction_substitutions WHERE enterprise_id = ANY($1)`,
			`DELETE FROM icms_apuracao_lines WHERE enterprise_id = ANY($1)`,
			`DELETE FROM icms_summary_entries WHERE enterprise_id = ANY($1)`,
			`DELETE FROM simples_nacional_apuracoes WHERE enterprise_id = ANY($1)`,
			`DELETE FROM tax_scenarios WHERE enterprise_id = ANY($1)`,
			`DELETE FROM items WHERE enterprise_id = ANY($1)`,
		} {
			if _, err := pool.Exec(ctx, q, ids); err != nil {
				t.Errorf("limpeza: %v", err)
			}
		}
	})
	repo := repository.NewFiscalParamsRepository(sqlc.New(pool), pool)

	// Parâmetro ICMS/IPI
	tp, err := repo.CreateTaxParam(a.Ctx, &entity.ICMSIPITaxParam{UF: "PR", OperationType: entity.TaxParamOperationSaida})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetTaxParamByID(b.Ctx, tp.ID); err == nil {
		t.Fatal("empresa B leu o parâmetro fiscal da A")
	}
	if l, err := repo.ListTaxParamsByUF(b.Ctx, "PR", false); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d parâmetro(s) da A (%v)", len(l), err)
	}
	if _, err := repo.UpdateTaxParam(b.Ctx, tp); err == nil {
		t.Fatal("empresa B alterou o parâmetro fiscal da A")
	}
	itemA := int64(1_730_000_000 + testutil.UniqueCode()%10_000_000)
	testutil.SeedItem(t, pool, a.Ctx, itemA, testutil.Actor(t, pool))
	if _, err := repo.CreateTaxParam(b.Ctx, &entity.ICMSIPITaxParam{UF: "PR", OperationType: entity.TaxParamOperationSaida, ItemCode: &itemA}); err == nil {
		t.Fatal("empresa B criou regra fiscal para item da A")
	}

	// Redução / substituição
	rs, err := repo.CreateICMSReductionSubstitution(a.Ctx, &entity.ICMSReductionSubstitution{UF: "SC", OperationType: entity.ICMSOpSaida,
		ICMSRedTargetContrib: entity.ICMSRedBase, ICMSRedTargetNonContrib: entity.ICMSRedBase, ICMSDeferralTarget: entity.ICMSRedBase,
		IPIRedTargetContrib: entity.ICMSRedBase, IPIRedTargetNonContrib: entity.ICMSRedBase})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetICMSReductionSubstitution(b.Ctx, rs.ID); err == nil {
		t.Fatal("empresa B leu a redução de ICMS da A")
	}
	if got, err := repo.FindICMSReductionSubstitution(b.Ctx, "SC", nil, nil, entity.ICMSOpSaida); err != nil || got != nil {
		t.Fatalf("o cálculo da empresa B usou a regra da A: %+v %v", got, err)
	}
	if got, err := repo.FindICMSReductionSubstitution(a.Ctx, "SC", nil, nil, entity.ICMSOpSaida); err != nil || got == nil || got.ID != rs.ID {
		t.Fatalf("o cálculo da empresa A não achou a própria regra: %+v %v", got, err)
	}
	if _, err := repo.UpdateICMSReductionSubstitution(b.Ctx, rs); err == nil {
		t.Fatal("empresa B alterou a redução de ICMS da A")
	}
	rs.IsActive = true
	if _, err := repo.UpdateICMSReductionSubstitution(a.Ctx, rs); err != nil {
		t.Fatalf("empresa A não alterou a própria redução de ICMS: %v", err)
	}
	tp.IsActive = true
	if _, err := repo.UpdateTaxParam(a.Ctx, tp); err != nil {
		t.Fatalf("empresa A não alterou o próprio parâmetro fiscal: %v", err)
	}

	// Linhas da apuração
	if _, err := repo.CreateICMSApuracaoLine(a.Ctx, &entity.ICMSApuracaoLine{Code: "T01", Description: "LINHA A", LineType: "DEBITO"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetICMSApuracaoLine(b.Ctx, "T01"); err == nil {
		t.Fatal("empresa B leu a linha de apuração da A")
	}
	if _, err := repo.CreateICMSApuracaoLine(b.Ctx, &entity.ICMSApuracaoLine{Code: "T01", Description: "LINHA B", LineType: "DEBITO"}); err != nil {
		t.Fatalf("o mesmo código de linha tem de poder existir na empresa B: %v", err)
	}

	// Totalizador da apuração e seus filhos
	tot, err := repo.CreateICMSSummaryEntry(a.Ctx, &entity.ICMSSummaryEntry{Period: "2026-01", UF: "PR"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetICMSSummaryEntry(b.Ctx, tot.ID); err == nil {
		t.Fatal("empresa B leu o totalizador de ICMS da A")
	}
	if l, err := repo.ListICMSSummaryEntries(b.Ctx, "2026-01", ""); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d totalizador(es) da A (%v)", len(l), err)
	}
	if _, err := repo.UpdateICMSSummaryEntry(b.Ctx, tot); err == nil {
		t.Fatal("empresa B alterou o totalizador da A")
	}
	if _, err := repo.AddICMSSummaryEntryNote(b.Ctx, &entity.ICMSSummaryEntryNote{SummaryEntryID: tot.ID, NoteNumber: "1", IssueDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)}); err == nil {
		t.Fatal("empresa B incluiu nota no totalizador da A")
	}
	if _, err := repo.AddICMSSummaryEntryAdditional(b.Ctx, &entity.ICMSSummaryEntryAdditional{SummaryEntryID: tot.ID, ArrecadacaoIndicator: "0"}); err == nil {
		t.Fatal("empresa B incluiu informação adicional no totalizador da A")
	}

	// Simples Nacional
	if _, err := repo.CreateSimplesNacionalApuracao(a.Ctx, &entity.SimplesNacionalApuracao{Period: "2026-01", Annex: "I"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetSimplesNacionalApuracao(b.Ctx, "2026-01", "I"); err == nil {
		t.Fatal("empresa B leu a apuração do Simples da A")
	}
	if _, err := repo.CreateSimplesNacionalApuracao(b.Ctx, &entity.SimplesNacionalApuracao{Period: "2026-01", Annex: "I"}); err != nil {
		t.Fatalf("o mesmo período do Simples tem de poder existir na empresa B: %v", err)
	}

	// Cenários fiscais
	if _, err := pool.Exec(context.Background(), `INSERT INTO tax_scenarios (scenario_name, destination_uf, destination_type, enterprise_id) VALUES ('CEN A','PR','CONTRIBUINTE',$1)`, a.ID); err != nil {
		t.Fatal(err)
	}
	fr := repository.NewFiscalRepositoryPG(pool)
	if l, err := fr.ListTaxScenarios(b.Ctx); err != nil || len(l) != 0 {
		t.Fatalf("empresa B listou %d cenário(s) da A (%v)", len(l), err)
	}
}
