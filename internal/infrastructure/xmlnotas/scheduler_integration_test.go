//go:build integration

package xmlnotas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/nfse_uc"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	nfserepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/nfse"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

type semLog struct{}

func (semLog) Info(string, ...any)  {}
func (semLog) Error(string, ...any) {}

// A rotina guarda o XML das notas transmitidas só das empresas com token, na
// sessão da empresa da nota; a nota que a Focus não tem fica para o dia seguinte
// e não volta no próximo ciclo.
func TestRotinaGuardaXMLDasNotasAntigas(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	ctx := context.Background()
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{`DELETE FROM focus_nfe_logs WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id = ANY($1))`,
			`DELETE FROM fiscal_exits WHERE enterprise_id = ANY($1)`, `DELETE FROM nfse WHERE enterprise_id = ANY($1)`,
			`DELETE FROM fiscal_configs WHERE enterprise_id = ANY($1)`} {
			_, _ = pool.Exec(context.Background(), q, []int64{a.ID, b.ID})
		}
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/nfe/"), "/nfse/")
		switch {
		case strings.HasPrefix(r.URL.Path, "/arquivos/"):
			_, _ = w.Write([]byte(`<doc chave="` + strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/arquivos/"), ".xml") + `"/>`))
		case ref == "sumiu":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"codigo":"nao_encontrado","mensagem":"Nota fiscal não encontrada"}`))
		default:
			_, _ = w.Write([]byte(`{"status":"autorizado","caminho_xml_nota_fiscal":"/arquivos/` + ref + `.xml"}`))
		}
	}))
	defer srv.Close()
	t.Setenv("FOCUSNFE_BASE_URL", srv.URL)

	if _, err := pool.Exec(ctx, `INSERT INTO fiscal_configs(enterprise_id,cnpj_empresa,razao_social,updated_by,focus_nfe_token,focus_nfe_ambiente)
		VALUES ($1,'11222333000181','A',$2,'tok','homologacao')`, a.ID, ator); err != nil {
		t.Fatal(err)
	}
	nota := func(emp int64, ref, chave string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,status,focus_ref,chave_acesso)
			VALUES ($1,$2::date,'5102','Venda',$3,$4,'AUTHORIZED',$5,$6) RETURNING id`,
			testutil.UniqueCode()%1_000_000, time.Now(), ator, emp, ref, chave).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	daA, sumiu, daB := nota(a.ID, "a1", "a1"), nota(a.ID, "sumiu", "x"), nota(b.ID, "b1", "b1")
	var nfse int64
	if err := pool.QueryRow(ctx, `INSERT INTO nfse(data_emissao,item_lista_servico,discriminacao,codigo_municipio,valor_servicos,enterprise_id,status,focus_ref,created_by)
		VALUES ($1::date,'14.01','S','4106902',100,$2,'AUTORIZADA','s1',$3) RETURNING id`, time.Now(), a.ID, ator).Scan(&nfse); err != nil {
		t.Fatal(err)
	}

	auth := &infraauth.AuthService{}
	fr := fiscalrepo.NewFiscalRepositoryPG(pool)
	nr := nfserepo.NewNFSeRepositoryPG(pool)
	s := NewScheduler(fr.(*fiscalrepo.FiscalRepositoryPG),
		&fiscal_uc.GetExitXMLUseCase{Repo: fr, Auth: auth},
		&nfse_uc.GetNFSeXMLUseCase{Repo: nr, Config: fr, Auth: auth}, semLog{})
	s.porCiclo, s.espaco = 1000, 0

	s.Ciclo(ctx)
	xml := func(tabela string, id int64) string {
		var x string
		_ = pool.QueryRow(ctx, `SELECT COALESCE(xml_content,'') FROM `+tabela+` WHERE id=$1`, id).Scan(&x)
		return x
	}
	if xml("fiscal_exits", daA) != `<doc chave="a1"/>` || xml("nfse", nfse) != `<doc chave="s1"/>` {
		t.Fatalf("XML da empresa A não guardado: nfe=%q nfse=%q", xml("fiscal_exits", daA), xml("nfse", nfse))
	}
	if xml("fiscal_exits", daB) != "" {
		t.Fatal("a rotina baixou nota da empresa B, que não tem token")
	}
	if xml("fiscal_exits", sumiu) != "" {
		t.Fatal("gravou XML da nota que a Focus não tem")
	}
	alvos, err := fr.(*fiscalrepo.FiscalRepositoryPG).NotasSemXML(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range alvos {
		if n.ID == sumiu || n.ID == daA {
			t.Fatalf("nota %d voltou à fila no mesmo dia", n.ID)
		}
	}
}
