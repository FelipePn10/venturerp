//go:build integration

package capa_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/capa_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/capa"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	capapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/capa"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func sp(s string) *string { return &s }

func TestCapa8DPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	// Ator próprio e ativo: a CAPA só aceita responsável ativo da empresa.
	actor := uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Líder CAPA',$2,'x')`, actor, actor.String()+"@example.test")
	item := 95_000_000 + u%1_000_000
	colega, estranho := uuid.New(), uuid.New()
	defer func() { _, _ = pool.Exec(bg, `UPDATE users SET is_active=FALSE WHERE id=$1`, actor) }()
	for _, x := range []uuid.UUID{colega, estranho} {
		testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,$2,$3,'x')`, x, "Usuário "+x.String()[:8], x.String()+"@example.test")
	}
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []int64{a.ID, b.ID} {
			_, _ = pool.Exec(bg, `ALTER TABLE quality_capa_eventos DISABLE TRIGGER trg_quality_capa_eventos_imutaveis`)
			_, _ = pool.Exec(bg, `DELETE FROM quality_capas WHERE enterprise_id=$1`, e)
			_, _ = pool.Exec(bg, `ALTER TABLE quality_capa_eventos ENABLE TRIGGER trg_quality_capa_eventos_imutaveis`)
		}
		_, _ = pool.Exec(bg, `DELETE FROM non_conformances WHERE item_code=$1`, item)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE code=$1`, item)
		_, _ = pool.Exec(bg, `DELETE FROM warehouse WHERE enterprise_id=$1`, a.ID)
		_, _ = pool.Exec(bg, `DELETE FROM user_enterprises WHERE user_id = ANY($1)`, []uuid.UUID{colega, estranho, actor})
		_, _ = pool.Exec(bg, `UPDATE users SET is_active=FALSE WHERE id = ANY($1)`, []uuid.UUID{colega, estranho})
	})
	testutil.Exec(t, pool, `INSERT INTO user_enterprises(user_id, enterprise_id, role) VALUES ($1,$3,'ADMIN'),($2,$3,'USER'),($4,$5,'USER') ON CONFLICT DO NOTHING`,
		actor, colega, a.ID, estranho, b.ID)
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Almox',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("CAPA-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,($1::bigint)::text,$2,$3,$4,'SUPORTE')`, item, almox, actor, a.ID)
	var nc int64
	if err := pool.QueryRow(bg, `INSERT INTO non_conformances(code,item_code,nonconform_qty,description,severity,created_by)
		VALUES((SELECT COALESCE(MAX(code),0)+1 FROM non_conformances),$1,12,'rebarba na furação','MAIOR',$2) RETURNING id`, item, actor).Scan(&nc); err != nil {
		t.Fatal(err)
	}

	uc := &capa_uc.UseCase{Repo: capapg.New(pool), Auth: &auth.AuthService{}}
	ctx := context.WithValue(bg, contextkey.UserKey, &security.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: a.ID, EnterpriseCode: a.Code})
	hoje := time.Now().Format("2006-01-02")

	if _, err := uc.Abrir(ctx, capa_uc.AberturaDTO{Metodo: "8D", Origem: "NC", Titulo: "x", NCs: []int64{nc},
		Equipe: []capa_uc.MembroDTO{{UserID: estranho.String(), Papel: "MEMBRO"}}}); err == nil {
		t.Fatal("usuário de outra empresa na equipe deveria ser recusado")
	}
	if _, err := uc.Abrir(b.Ctx, capa_uc.AberturaDTO{Metodo: "8D", Origem: "NC", Titulo: "x", NCs: []int64{nc}}); err == nil {
		t.Fatal("B abriu CAPA com a NC de A")
	}
	c, err := uc.Abrir(ctx, capa_uc.AberturaDTO{Metodo: "8D", Origem: "NC", Titulo: "Rebarba na furação do suporte", ItemCode: &item, NCs: []int64{nc}})
	if err != nil {
		t.Fatal(err)
	}
	if c.Numero != 1 || c.Etapa != "D0" || len(c.NCs) != 1 || c.Responsavel != actor {
		t.Fatalf("abertura: %+v", c.Capa)
	}
	if _, err := uc.Obter(b.Ctx, c.ID); err == nil {
		t.Fatal("B leu a CAPA de A")
	}
	ncs, err := uc.NCs(ctx, dom.FiltroNC{SemCapa: true})
	for _, n := range ncs {
		if n.ID == nc {
			t.Fatal("a NC tem CAPA e saiu no filtro sem CAPA")
		}
	}
	if ncs, err = uc.NCs(ctx, dom.FiltroNC{ItemCode: item}); err != nil || len(ncs) != 1 || len(ncs[0].CapaNumeros) != 1 || ncs[0].CapaNumeros[0] != 1 {
		t.Fatalf("NC com a CAPA: %+v %v", ncs, err)
	}
	if lista, _ := uc.NCs(b.Ctx, dom.FiltroNC{ItemCode: item}); len(lista) != 0 {
		t.Fatal("B listou a NC de A")
	}

	avancar := func(etapa string) {
		t.Helper()
		r, err := uc.Avancar(ctx, c.ID)
		if err != nil || r.Etapa != etapa {
			t.Fatalf("avançar para %s: %v", etapa, err)
		}
		c = r
	}
	avancar("D1")
	if _, err := uc.Avancar(ctx, c.ID); err == nil || !strings.Contains(err.Error(), "líder") {
		t.Fatalf("D1 sem equipe: %v", err)
	}
	if c, err = uc.Analisar(ctx, c.ID, capa_uc.AnaliseDTO{Equipe: &[]capa_uc.MembroDTO{{UserID: actor.String(), Papel: "LIDER"}, {UserID: colega.String()}}}); err != nil {
		t.Fatal(err)
	}
	avancar("D2")
	if c, err = uc.Analisar(ctx, c.ID, capa_uc.AnaliseDTO{Problema: &dom.Problema{OQue: "rebarba", Onde: "furo Ø8", Quando: "turno B", Quanto: "12%"}}); err != nil {
		t.Fatal(err)
	}
	avancar("D3")
	if c, err = uc.AdicionarAcao(ctx, c.ID, capa_uc.AcaoDTO{Tipo: "CONTENCAO", Descricao: "segregar e rebarbar o estoque", Responsavel: colega.String(), Prazo: hoje}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.AdicionarAcao(ctx, c.ID, capa_uc.AcaoDTO{Tipo: "CORRETIVA", Descricao: "x", Responsavel: estranho.String(), Prazo: hoje}); err == nil {
		t.Fatal("responsável de outra empresa deveria ser recusado")
	}
	avancar("D4")
	if _, err := uc.Analisar(ctx, c.ID, capa_uc.AnaliseDTO{Ishikawa: &map[string][]string{"DINHEIRO": {"x"}}}); err == nil {
		t.Fatal("categoria fora dos 6M deveria ser recusada")
	}
	if c, err = uc.Analisar(ctx, c.ID, capa_uc.AnaliseDTO{MetodoCausa: sp("5PORQUES"), CincoPorques: &[]string{"broca gasta", "trocada por tempo", "sem contador de vida"},
		CausaRaiz: sp("vida da broca não controlada"), PontoDeEscape: sp("inspeção visual sem lupa")}); err != nil {
		t.Fatal(err)
	}
	avancar("D5")
	if c, err = uc.AdicionarAcao(ctx, c.ID, capa_uc.AcaoDTO{Tipo: "CORRETIVA", Descricao: "contador de vida na máquina", Responsavel: actor.String(), Prazo: hoje}); err != nil {
		t.Fatal(err)
	}
	pend, err := uc.AcoesPendentes(ctx, false)
	if err != nil || len(pend) != 2 {
		t.Fatalf("ações pendentes: %d %v", len(pend), err)
	}
	if minhas, _ := uc.AcoesPendentes(ctx, true); len(minhas) != 1 || minhas[0].Descricao != "contador de vida na máquina" {
		t.Fatalf("minhas ações: %+v", minhas)
	}
	avancar("D6")
	if _, err := uc.Avancar(ctx, c.ID); err == nil {
		t.Fatal("D6 com ações abertas")
	}
	for _, x := range c.Acoes {
		if _, err := uc.AtualizarAcao(ctx, c.ID, x.ID, capa_uc.AndamentoDTO{Status: "CONCLUIDA"}); err == nil || !strings.Contains(err.Error(), "evidência") {
			t.Fatalf("concluir sem evidência: %v", err)
		}
		if c, err = uc.AtualizarAcao(ctx, c.ID, x.ID, capa_uc.AndamentoDTO{Status: "CONCLUIDA", Evidencia: sp("registro " + fmt.Sprint(x.ID))}); err != nil {
			t.Fatal(err)
		}
	}
	avancar("D7")
	if c, err = uc.Analisar(ctx, c.ID, capa_uc.AnaliseDTO{Prevencao: sp("vida de ferramenta no plano de manutenção")}); err != nil {
		t.Fatal(err)
	}
	avancar("D8")

	// Ineficaz: volta para D4 no ciclo 2, sem a causa anterior.
	if c, err = uc.Encerrar(ctx, c.ID, capa_uc.EncerramentoDTO{Eficaz: false, Verificacao: "rebarba voltou no lote 31"}); err != nil {
		t.Fatal(err)
	}
	if c.Etapa != "D4" || c.Ciclo != 2 || c.CausaRaiz != nil || c.Status != dom.Aberta {
		t.Fatalf("ineficaz: %+v", c.Capa)
	}
	velha := c.Acoes[0].ID
	if _, err := uc.AtualizarAcao(ctx, c.ID, velha, capa_uc.AndamentoDTO{Status: "CANCELADA"}); err == nil {
		t.Fatal("ação do ciclo anterior não muda mais")
	}
	if _, err := uc.AtualizarAcao(b.Ctx, c.ID, velha, capa_uc.AndamentoDTO{Status: "CANCELADA"}); err == nil {
		t.Fatal("B mexeu na ação de A")
	}
	cancelada, err := uc.Cancelar(ctx, c.ID, "substituída pela CAPA do fornecedor da broca")
	if err != nil || cancelada.Status != dom.Cancelada {
		t.Fatalf("cancelamento: %v", err)
	}
	tipos := map[string]int{}
	for _, ev := range cancelada.Eventos {
		tipos[ev.Evento]++
	}
	if tipos["ABERTA"] != 1 || tipos["ETAPA"] != 8 || tipos["INEFICAZ"] != 1 || tipos["CANCELADA"] != 1 || tipos["ACAO"] != 2 {
		t.Fatalf("histórico: %v", tipos)
	}
	if _, err := pool.Exec(bg, `UPDATE quality_capa_eventos SET detalhe='x' WHERE capa_id=$1`, c.ID); err == nil {
		t.Fatal("o histórico da CAPA deveria ser imutável")
	}

	// CAPA simples encerrada eficaz.
	s, err := uc.Abrir(ctx, capa_uc.AberturaDTO{Metodo: "SIMPLES", Origem: "INTERNA", Titulo: "Etiqueta trocada", Responsavel: colega.String()})
	if err != nil {
		t.Fatal(err)
	}
	if s, err = uc.Analisar(ctx, s.ID, capa_uc.AnaliseDTO{Problema: &dom.Problema{OQue: "etiqueta de outro item"}, MetodoCausa: sp("ISHIKAWA"),
		Ishikawa: &map[string][]string{"METODO": {"impressão em lote misturado"}}, CausaRaiz: sp("impressão sem conferência")}); err != nil {
		t.Fatal(err)
	}
	if s, err = uc.Avancar(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	if s, err = uc.AdicionarAcao(ctx, s.ID, capa_uc.AcaoDTO{Tipo: "CORRETIVA", Descricao: "leitura do código ao etiquetar", Responsavel: colega.String(), Prazo: hoje}); err != nil {
		t.Fatal(err)
	}
	if s, err = uc.AtualizarAcao(ctx, s.ID, s.Acoes[0].ID, capa_uc.AndamentoDTO{Status: "CONCLUIDA", Evidencia: sp("leitor instalado")}); err != nil {
		t.Fatal(err)
	}
	if s, err = uc.Avancar(ctx, s.ID); err != nil || s.Etapa != "VERIFICACAO" {
		t.Fatalf("simples para verificação: %v", err)
	}
	if s, err = uc.Encerrar(ctx, s.ID, capa_uc.EncerramentoDTO{Eficaz: true, Verificacao: "30 dias sem troca"}); err != nil || s.Status != dom.Encerrada || s.Numero != 2 {
		t.Fatalf("encerramento da simples: %v %+v", err, s.Capa)
	}
	if abertas, _ := uc.Listar(ctx, "ABERTA", false, false); len(abertas) != 0 {
		t.Fatalf("nenhuma CAPA aberta: %d", len(abertas))
	}
	if todas, _ := uc.Listar(b.Ctx, "", false, false); len(todas) != 0 {
		t.Fatal("B listou CAPAs de A")
	}
}
