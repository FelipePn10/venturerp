//go:build integration

package bom_header_test

import (
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
	bomheaderrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom_header"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Versão da estrutura: numeração automática, aprovação com foto das linhas,
// uma aprovada por vez (a nova torna a anterior obsoleta), ciclo fechado e
// isolamento por empresa.
func TestIntegration_BomHeader_AprovacaoComFoto(t *testing.T) {
	q, pool := testutil.Queries(t)
	repo := bomheaderrepo.New(q, pool)
	actor := testutil.Actor(t, pool)
	pai, filho, outro := testutil.UniqueCode(), testutil.UniqueCode()+1, testutil.UniqueCode()+2
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		testutil.Exec(t, pool, `DELETE FROM bom_headers WHERE item_code=$1`, pai)
		testutil.Exec(t, pool, `DELETE FROM item_structures WHERE parent_code=$1`, pai)
	})
	testutil.NovosItens(t, pool, a.ID, actor, pai, filho, outro)
	ctx := a.Ctx

	novaVersao := func() *entity.BomHeader {
		v, err := repo.NextVersion(ctx, pai, "")
		if err != nil {
			t.Fatal(err)
		}
		h, _ := entity.NewBomHeader(pai, nil, entity.TypeMBOM, v, nil, actor)
		criada, err := repo.Create(ctx, h)
		if err != nil {
			t.Fatal(err)
		}
		return criada
	}

	v1 := novaVersao()
	if v1.Version != 1 || v1.Status != entity.StatusDraft {
		t.Fatalf("v1 = %+v", v1)
	}
	if _, err := repo.Approve(ctx, v1.ID, actor, time.Now()); err == nil {
		t.Fatal("aprovou versão com estrutura vazia")
	}

	testutil.Exec(t, pool, `INSERT INTO item_structures(parent_code,child_code,quantity,unit_of_measurement,sequence,created_by,is_active) VALUES($1,$2,2,'UN',1,$3,TRUE)`, pai, filho, actor)
	aprovada, err := repo.Approve(ctx, v1.ID, actor, time.Now())
	if err != nil || aprovada.Status != entity.StatusApproved || aprovada.ApprovedBy == nil || aprovada.ApprovedAt == nil {
		t.Fatalf("aprovação v1 = %+v err=%v", aprovada, err)
	}
	foto, err := repo.ApprovedLines(ctx, v1.ID)
	if err != nil || len(foto) != 1 || foto[0].ChildCode != filho {
		t.Fatalf("foto da v1 = %+v err=%v", foto, err)
	}

	// A estrutura muda: a foto não, e a comparação aponta a diferença.
	testutil.Exec(t, pool, `INSERT INTO item_structures(parent_code,child_code,quantity,unit_of_measurement,sequence,created_by,is_active) VALUES($1,$2,1,'UN',2,$3,TRUE)`, pai, outro, actor)
	atual, err := repo.CurrentLines(ctx, pai, "")
	if err != nil || len(atual) != 2 {
		t.Fatalf("linhas atuais = %d err=%v", len(atual), err)
	}
	if mud := entity.CompararLinhas(foto, atual); len(mud) != 1 || mud[0].Kind != "INCLUIDO" {
		t.Fatalf("comparação = %+v", mud)
	}

	// Aprovar a v2 torna a v1 obsoleta; a v1 não volta.
	v2 := novaVersao()
	if _, err := repo.Approve(ctx, v2.ID, actor, time.Now()); err != nil {
		t.Fatal(err)
	}
	if h, _ := repo.GetByID(ctx, v1.ID); h.Status != entity.StatusObsolete || h.ObsoletedAt == nil {
		t.Fatalf("v1 deveria estar obsoleta: %+v", h)
	}
	if _, err := repo.Approve(ctx, v1.ID, actor, time.Now()); err == nil {
		t.Fatal("versão obsoleta voltou a ser aprovada")
	}
	if foto2, _ := repo.ApprovedLines(ctx, v2.ID); len(foto2) != 2 {
		t.Fatalf("foto da v2 = %d linhas, esperava 2", len(foto2))
	}

	// A outra empresa não lê, não aprova nem torna obsoleta.
	if _, err := repo.GetByID(b.Ctx, v2.ID); err == nil {
		t.Fatal("versão de outra empresa foi lida")
	}
	if _, err := repo.Obsolete(b.Ctx, v2.ID); err == nil {
		t.Fatal("versão de outra empresa foi alterada")
	}
	if lista, err := repo.ListByItem(b.Ctx, pai); err != nil || len(lista) != 0 {
		t.Fatalf("listagem cruzou empresas: %d err=%v", len(lista), err)
	}
}
