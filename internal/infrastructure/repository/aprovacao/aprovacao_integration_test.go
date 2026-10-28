//go:build integration

package aprovacao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	dom "github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Política e solicitações são da empresa; dois níveis avançam um de cada vez
// e aprovação concorrente do mesmo nível não pula nível.
func TestAprovacaoPorEmpresaEmDoisNiveis(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	actor := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		ids := []int64{a.ID, b.ID}
		tx, err := pool.Begin(base)
		if err != nil {
			return
		}
		defer tx.Rollback(base)
		_, _ = tx.Exec(base, `ALTER TABLE aprovacao_eventos DISABLE TRIGGER trg_aprovacao_eventos_imutaveis`)
		_, _ = tx.Exec(base, `DELETE FROM aprovacao_eventos WHERE enterprise_id = ANY($1)`, ids)
		_, _ = tx.Exec(base, `ALTER TABLE aprovacao_eventos ENABLE TRIGGER trg_aprovacao_eventos_imutaveis`)
		_, _ = tx.Exec(base, `DELETE FROM aprovacao_solicitacoes WHERE enterprise_id = ANY($1)`, ids)
		_, _ = tx.Exec(base, `DELETE FROM aprovacao_politicas WHERE enterprise_id = ANY($1)`, ids)
		_ = tx.Commit(base)
	})
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id, enterprise_id, role) VALUES ($1, $2, 'ADMIN')`, actor, a.ID)
	repo := aprovacao.New(pool)
	teto := decimal.NewFromInt(1000)
	p := &dom.Politica{Tipo: dom.LiberacaoCredito, Ativa: true, Segregacao: false,
		Niveis: []dom.Nivel{{ValorAte: &teto, UserID: &actor}, {UserID: &actor}}}
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	if err := repo.SalvarPolitica(a.Ctx, p, actor); err != nil {
		t.Fatal(err)
	}
	estranho := uuid.New()
	ruim := &dom.Politica{Tipo: dom.OrcamentoPoliticaComercial, Ativa: true, Niveis: []dom.Nivel{{UserID: &estranho}}}
	if err := repo.SalvarPolitica(a.Ctx, ruim, actor); err == nil {
		t.Fatal("aprovador de fora da empresa deveria ser recusado")
	}
	if pol, _ := repo.PoliticaAtiva(b.Ctx, dom.LiberacaoCredito); pol != nil {
		t.Fatal("B leu a política de A")
	}

	pol, err := repo.PoliticaAtiva(a.Ctx, dom.LiberacaoCredito)
	if err != nil || pol == nil || len(pol.Niveis) != 2 {
		t.Fatalf("política de A: %+v %v", pol, err)
	}
	valor := decimal.NewFromInt(5000)
	s, criada, err := repo.CriarSolicitacao(a.Ctx, &dom.Solicitacao{PoliticaID: pol.ID, Tipo: dom.LiberacaoCredito, DocumentoCodigo: 77,
		Valor: valor, Descricao: "acima do limite", Solicitante: actor, NiveisExigidos: pol.NiveisExigidos(valor)})
	if err != nil || !criada || s.NiveisExigidos != 2 {
		t.Fatalf("solicitação: %+v %v %v", s, criada, err)
	}
	if outra, criada2, _ := repo.CriarSolicitacao(a.Ctx, &dom.Solicitacao{PoliticaID: pol.ID, Tipo: dom.LiberacaoCredito, DocumentoCodigo: 77,
		Valor: valor, Descricao: "de novo", Solicitante: actor, NiveisExigidos: 2}); criada2 || outra.ID != s.ID {
		t.Fatal("a pendente do documento deveria ser reaproveitada")
	}
	if _, err := repo.BuscarSolicitacao(b.Ctx, s.ID); err == nil {
		t.Fatal("B leu a solicitação de A")
	}
	if err := repo.RegistrarAprovacaoDeNivel(b.Ctx, s.ID, 1, false, actor, nil); err == nil {
		t.Fatal("B aprovou a solicitação de A")
	}

	if err := repo.RegistrarAprovacaoDeNivel(a.Ctx, s.ID, 1, false, actor, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.RegistrarAprovacaoDeNivel(a.Ctx, s.ID, 1, false, actor, nil); err == nil {
		t.Fatal("o mesmo nível não se aprova duas vezes (aprovação concorrente)")
	}
	if err := repo.RegistrarAprovacaoDeNivel(a.Ctx, s.ID, 2, true, actor, nil); err != nil {
		t.Fatal(err)
	}
	final, err := repo.BuscarSolicitacao(a.Ctx, s.ID)
	if err != nil || final.Status != dom.Aprovada || len(final.Eventos) != 3 || final.ConcluidaAt == nil {
		t.Fatalf("final: %+v %v", final, err)
	}
	if pend, _ := repo.PendenteDoDocumento(a.Ctx, dom.LiberacaoCredito, 77); pend != nil {
		t.Fatal("aprovada não é mais pendente")
	}
}
