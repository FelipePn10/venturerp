//go:build integration

package amostragem_uc_test

import (
	"sync"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/amostragem_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	repo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/amostragem"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Procedimento e estado são da empresa; a troca de severidade sobrevive entre
// chamadas com histórico; lotes concorrentes do mesmo fornecedor não perdem
// resultado; suspensão bloqueia e a liberação recomeça em apertada.
func TestAmostragemPorEmpresaComTrocaESuspensao(t *testing.T) {
	pool := testutil.Pool(t)
	a, b := testutil.DuasEmpresas(t, pool, nil) // a exclusão da empresa leva procedimentos, estados e histórico
	uc := &amostragem_uc.UseCase{Repo: repo.New(pool), Auth: &infraauth.AuthService{}}

	p, err := uc.Salvar(a.Ctx, 0, amostragem_uc.ProcedimentoDTO{Codigo: "rec-ii-1", Descricao: "Recebimento NQA 1,0", Tipo: "norma", Nivel: "ii", NQA: "1,0"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Codigo != "REC-II-1" || p.NQA != "1.0" || p.Nivel != dom.NivelII {
		t.Fatalf("normalização: %+v", p)
	}
	if _, err := uc.Salvar(a.Ctx, 0, amostragem_uc.ProcedimentoDTO{Codigo: "REC-II-1", Descricao: "dup", Tipo: "CEM"}); err == nil {
		t.Fatal("código duplicado na empresa deveria ser recusado")
	}
	// Mesmo código em B é outro procedimento.
	if _, err := uc.Salvar(b.Ctx, 0, amostragem_uc.ProcedimentoDTO{Codigo: "REC-II-1", Descricao: "de B", Tipo: "CEM"}); err != nil {
		t.Fatalf("B pode ter o mesmo código: %v", err)
	}
	if _, err := uc.ObterProcedimento(b.Ctx, p.ID); err == nil {
		t.Fatal("B leu o procedimento de A")
	}
	if _, err := uc.Salvar(b.Ctx, p.ID, amostragem_uc.ProcedimentoDTO{Codigo: "X", Descricao: "invasão", Tipo: "CEM"}); err == nil {
		t.Fatal("B alterou o procedimento de A")
	}

	const item = 990001
	chave := dom.ChaveFornecedor(4242)
	// Lote de 1000 unidades, nível II → J; normal NQA 1,0: n 80, Ac 2.
	pl, err := uc.PlanoDoLote(a.Ctx, p.ID, item, chave, "1000.000000")
	if err != nil || pl.Plano.N != 80 || pl.Plano.Ac != 2 || pl.Estado.Severidade != dom.Normal || !pl.Inspecionar {
		t.Fatalf("plano do 1º lote: %+v %v", pl, err)
	}
	if _, err := uc.PlanoDoLote(a.Ctx, p.ID, item, chave, "12.5"); err == nil {
		t.Fatal("lote fracionado pela norma deveria ser recusado")
	}
	if _, err := uc.PlanoDoLote(b.Ctx, p.ID, item, chave, "1000"); err == nil {
		t.Fatal("B calculou com o procedimento de A")
	}

	// Duas recusas → apertada, com evento.
	for _, d := range []int{3, 0, 4} {
		if _, err := uc.RegistrarResultado(a.Ctx, p.ID, item, chave, pl.Plano, d, true, "NF 1"); err != nil {
			t.Fatal(err)
		}
	}
	pl2, err := uc.PlanoDoLote(a.Ctx, p.ID, item, chave, "1000")
	if err != nil || pl2.Estado.Severidade != dom.Apertada || pl2.Plano.Ac != 1 || pl2.Plano.Severidade != dom.Apertada {
		t.Fatalf("depois de 2 recusas: apertada J NQA 1,0 = n 80 Ac 1: %+v %v", pl2, err)
	}
	ev, err := uc.Eventos(a.Ctx, item)
	if err != nil || len(ev) != 1 || ev[0].Para != dom.Apertada || ev[0].Referencia == nil || *ev[0].Referencia != "NF 1" {
		t.Fatalf("histórico: %+v %v", ev, err)
	}
	if evB, _ := uc.Eventos(b.Ctx, item); len(evB) != 0 {
		t.Fatal("B viu o histórico de A")
	}
	if _, err := pool.Exec(a.Ctx, `UPDATE quality_sampling_events SET motivo='x' WHERE id=$1`, ev[0].ID); err == nil {
		t.Fatal("o histórico deveria ser imutável")
	}

	// Concorrência: 5 recusas em paralelo no mesmo estado. Sem a trava, dois
	// lotes leriam o mesmo estado e a 5ª recusa se perderia.
	var wg sync.WaitGroup
	erros := make(chan error, 5)
	for k := 0; k < 5; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := uc.RegistrarResultado(a.Ctx, p.ID, item, chave, pl2.Plano, 5, true, "paralelo"); err != nil {
				erros <- err
			}
		}()
	}
	wg.Wait()
	close(erros)
	for err := range erros {
		t.Fatalf("registro paralelo: %v", err)
	}
	est, err := uc.Estados(a.Ctx, item)
	if err != nil || len(est) != 1 || !est[0].Estado.Suspensa || est[0].Estado.RecusasEmApertada != 5 {
		t.Fatalf("5 recusas em apertada suspendem (nenhuma perdida): %+v %v", est, err)
	}
	if _, err := uc.RegistrarResultado(a.Ctx, p.ID, item, chave, pl2.Plano, 0, true, ""); err == nil {
		t.Fatal("lote em item suspenso não entra na norma")
	}
	if _, err := uc.LiberarSuspensao(b.Ctx, amostragem_uc.LiberacaoDTO{ProcedimentoID: p.ID, ItemCode: item, Chave: chave, Motivo: "8D aceito"}); err == nil {
		t.Fatal("B liberou a suspensão de A")
	}
	if _, err := uc.LiberarSuspensao(a.Ctx, amostragem_uc.LiberacaoDTO{ProcedimentoID: p.ID, ItemCode: item, Chave: chave}); err == nil {
		t.Fatal("liberação sem motivo deveria ser recusada")
	}
	lib, err := uc.LiberarSuspensao(a.Ctx, amostragem_uc.LiberacaoDTO{ProcedimentoID: p.ID, ItemCode: item, Chave: chave, Motivo: "8D 12 aceito"})
	if err != nil || lib.Suspensa || lib.Severidade != dom.Apertada {
		t.Fatalf("liberação: %+v %v", lib, err)
	}
	ev, _ = uc.Eventos(a.Ctx, item)
	tipos := map[string]int{}
	for _, e := range ev {
		tipos[e.Tipo]++
	}
	if tipos["TROCA"] != 1 || tipos["SUSPENSAO"] != 1 || tipos["LIBERACAO"] != 1 {
		t.Fatalf("eventos: %v", tipos)
	}

	// Skip-lot num plano fixo: depois de 2 aceitos, 1 a cada 3.
	f, err := uc.Salvar(a.Ctx, 0, amostragem_uc.ProcedimentoDTO{Codigo: "FIX", Descricao: "Fixo 5/0/1", Tipo: "FIXO",
		AmostraFixa: ptr(int64(5)), AcFixo: ptr(0), ReFixo: ptr(1), SkipApos: 2, SkipFrequencia: 3})
	if err != nil {
		t.Fatal(err)
	}
	geral := ""
	for k := 0; k < 2; k++ {
		pf, err := uc.PlanoDoLote(a.Ctx, f.ID, item, geral, "12.5") // plano fixo aceita lote fracionado
		if err != nil || !pf.Inspecionar || pf.Plano.N != 1 || !pf.Plano.CemPorCento {
			t.Fatalf("plano fixo, lote fracionado: %+v %v", pf, err)
		}
		if _, err := uc.RegistrarResultado(a.Ctx, f.ID, item, geral, dom.Plano{N: 5, Ac: 0, Re: 1}, 0, true, ""); err != nil {
			t.Fatal(err)
		}
	}
	for k := 0; k < 2; k++ {
		pf, _ := uc.PlanoDoLote(a.Ctx, f.ID, item, geral, "100")
		if pf.Inspecionar {
			t.Fatalf("lote %d deveria ser pulado", k+1)
		}
		if _, err := uc.PularLote(a.Ctx, f.ID, item, geral, "skip"); err != nil {
			t.Fatal(err)
		}
	}
	if pf, _ := uc.PlanoDoLote(a.Ctx, f.ID, item, geral, "100"); !pf.Inspecionar {
		t.Fatal("o 3º lote é inspecionado")
	}
	if _, err := uc.PularLote(a.Ctx, f.ID, item, geral, "skip"); err == nil {
		t.Fatal("não se pula lote que precisa de inspeção")
	}
}

func ptr[T any](v T) *T { return &v }
