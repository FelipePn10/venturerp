//go:build integration

package fiscal_test

import (
	"context"
	"errors"
	"testing"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	repository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A nota especial de ajuste (VFIS0560) alimenta o E111 da EFD: tem de ser da
// empresa do usuário — antes a empresa vinha do corpo da requisição (a tela
// mandava 1 fixo) e a consulta por id não filtrava empresa.
func TestNotasEspeciaisIsoladasPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	codeA := int64(1_650_000_000 + testutil.UniqueCode()%90_000_000)
	var a, b int64
	for i, dst := range []*int64{&a, &b} {
		if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Notas especiais') RETURNING id`, codeA+int64(i)).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, e := range []int64{a, b} {
			_, _ = pool.Exec(bg, `DELETE FROM special_adjustment_note_items WHERE note_id IN (SELECT id FROM special_adjustment_notes WHERE empresa_id=$1)`, e)
			_, _ = pool.Exec(bg, `DELETE FROM special_adjustment_notes WHERE empresa_id=$1`, e)
			_, _ = pool.Exec(bg, `DELETE FROM icms_st_restitutions WHERE empresa_id=$1`, e)
			if _, err := pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, e); err != nil {
				t.Errorf("limpeza: %v", err)
			}
		}
	})
	repo := repository.NewFiscalParamsRepository(nil, pool)
	ctxA, ctxB := fiscalTenantContext(a), fiscalTenantContext(b)

	// O corpo diz empresa B; vale a do usuário (A).
	n, err := repo.CreateSpecialAdjustmentNote(ctxA, &entity.SpecialAdjustmentNote{EmpresaID: int(b), Purpose: entity.SpecialNotePurpose("AJUSTE"),
		Status: entity.SpecialNoteStatus("RASCUNHO"), IssueDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Period: "2026-08"})
	if err != nil {
		t.Fatal(err)
	}
	if n.EmpresaID != int(a) {
		t.Fatalf("empresa gravada = %d, want %d", n.EmpresaID, a)
	}
	var nf *errorsuc.NotFoundError
	if _, err := repo.GetSpecialAdjustmentNote(ctxB, n.ID); !errors.As(err, &nf) {
		t.Errorf("empresa B lê a nota de A: %v", err)
	}
	if _, err := repo.UpdateSpecialAdjustmentNote(ctxB, &entity.SpecialAdjustmentNote{ID: n.ID, Status: entity.SpecialNoteStatus("EMITIDA")}); !errors.As(err, &nf) {
		t.Errorf("empresa B altera a nota de A: %v", err)
	}
	if l, err := repo.ListSpecialAdjustmentNotes(ctxB, int(a), "2026-08"); err != nil || len(l) != 0 {
		t.Errorf("empresa B lista as notas de A pedindo empresa_id=%d: %v %d", a, err, len(l))
	}
	if l, err := repo.ListSpecialAdjustmentNotes(ctxA, 0, "2026-08"); err != nil || len(l) != 1 {
		t.Errorf("empresa A não vê a própria nota: %v %d", err, len(l))
	}
	if _, err := repo.AddSpecialAdjustmentNoteItem(ctxB, &entity.SpecialAdjustmentNoteItem{NoteID: n.ID}); !errors.As(err, &nf) {
		t.Errorf("empresa B inclui item na nota de A: %v", err)
	}
	if _, err := repo.ListSpecialAdjustmentNoteItems(ctxB, n.ID); !errors.As(err, &nf) {
		t.Errorf("empresa B lê os itens da nota de A: %v", err)
	}

	r, err := repo.CreateICMSSTRestitution(ctxA, &entity.ICMSSTRestitution{EmpresaID: int(b), Period: "2026-08", RestitutionType: entity.ICMSSTRestitutionType("RESTITUICAO"), UF: "PR"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetICMSSTRestitution(ctxB, r.ID); !errors.As(err, &nf) {
		t.Errorf("empresa B lê a restituição de A: %v", err)
	}
	if l, err := repo.ListICMSSTRestitutions(ctxB, int(a), "2026-08", ""); err != nil || len(l) != 0 {
		t.Errorf("empresa B lista as restituições de A: %v %d", err, len(l))
	}
}
