package fiscal_uc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

const chaveXMLTeste = "41251112345678000123550010000001231000001234"

type authXML struct{ ports.AuthService }

func (authXML) CanGetFiscalExit(context.Context) bool { return true }

type repoXML struct {
	repository.FiscalRepository
	exit     *entity.FiscalExit
	xml      string
	xmlCanc  string
	saves    int
	savesCan int
}

func (r *repoXML) GetExitByID(context.Context, int64) (*entity.FiscalExit, error) { return r.exit, nil }
func (r *repoXML) ExitXML(context.Context, int64) (string, string, error) {
	return r.xml, r.xmlCanc, nil
}
func (r *repoXML) SaveExitCancelXML(_ context.Context, _ int64, xml string) error {
	r.xmlCanc = xml
	r.savesCan++
	return nil
}
func (r *repoXML) SaveExitXML(_ context.Context, _ int64, xml string) error {
	r.xml = xml
	r.saves++
	return nil
}
func (r *repoXML) SaveFocusLog(context.Context, int64, string, string, string, string, int, int) error {
	return nil
}
func (r *repoXML) GetFiscalConfig(context.Context) (*entity.FiscalConfig, error) {
	tok := "tok"
	return &entity.FiscalConfig{FocusNfeToken: &tok, FocusNfeAmbiente: "homologacao"}, nil
}

// focusXML simula a Focus como ela responde de verdade (conferido na
// homologação): a consulta exige o token, o arquivo é público e chega depois de
// dois redirecionamentos (/arquivos → /notas_fiscais → arquivo).
func focusXML(t *testing.T, arquivo string) (*int, func()) {
	return focusXMLComCancelamento(t, arquivo, "")
}

func focusXMLComCancelamento(t *testing.T, arquivo, cancelamento string) (*int, func()) {
	t.Helper()
	baixou := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/nfe/"):
			if u, _, ok := r.BasicAuth(); !ok || u != "tok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			corpo := `{"status":"autorizado","chave_nfe":"NFe` + chaveXMLTeste + `","caminho_xml_nota_fiscal":"/arquivos/x/` + chaveXMLTeste + `-nfe.xml"`
			if cancelamento != "" {
				corpo += `,"caminho_xml_cancelamento":"/arquivos/x/` + chaveXMLTeste + `-can.xml"`
			}
			_, _ = w.Write([]byte(corpo + "}"))
		case strings.HasPrefix(r.URL.Path, "/arquivos/"):
			http.Redirect(w, r, "/notas_fiscais"+strings.TrimPrefix(r.URL.Path, "/arquivos/x"), http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/notas_fiscais/"):
			http.Redirect(w, r, "/arquivos_development"+strings.TrimPrefix(r.URL.Path, "/notas_fiscais"), http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/arquivos_development/"):
			baixou++
			if strings.HasSuffix(r.URL.Path, "-can.xml") {
				_, _ = w.Write([]byte(cancelamento))
				return
			}
			_, _ = w.Write([]byte(arquivo))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Setenv("FOCUSNFE_BASE_URL", srv.URL)
	return &baixou, srv.Close
}

func notaAutorizada() *entity.FiscalExit {
	chave, ref := chaveXMLTeste, "ref1"
	return &entity.FiscalExit{ID: 7, NumeroNF: 123, Serie: "1", Status: entity.ExitStatusAuthorized, ChaveAcesso: &chave, FocusRef: &ref}
}

func TestExitXML_BaixaUmaVezEGuarda(t *testing.T) {
	xml := `<?xml version="1.0"?><nfeProc><NFe><infNFe Id="NFe` + chaveXMLTeste + `"><total><ICMSTot><vICMS>18.00</vICMS></ICMSTot></total></infNFe></NFe></nfeProc>`
	baixou, fim := focusXML(t, "\ufeff"+xml)
	defer fim()
	repo := &repoXML{exit: notaAutorizada()}
	uc := &GetExitXMLUseCase{Repo: repo, Auth: authXML{}}

	for i := 0; i < 2; i++ {
		res, err := uc.Execute(context.Background(), 7)
		if err != nil {
			t.Fatal(err)
		}
		if res.XML != xml || res.ChaveAcesso != chaveXMLTeste || res.NumeroNF != 123 {
			t.Fatalf("resultado: %+v", res)
		}
	}
	if *baixou != 1 || repo.saves != 1 {
		t.Fatalf("baixou %d vez(es) e gravou %d: a segunda consulta deveria sair do banco", *baixou, repo.saves)
	}
}

func TestExitXML_RecusaArquivoQueNaoEDaNota(t *testing.T) {
	for nome, arquivo := range map[string]string{
		"pagina de erro": "<html><body>Not found</body></html>",
		"outra chave":    `<nfeProc><infNFe Id="NFe41251199999999000199550010000009991000009999"/></nfeProc>`,
	} {
		t.Run(nome, func(t *testing.T) {
			_, fim := focusXML(t, arquivo)
			defer fim()
			repo := &repoXML{exit: notaAutorizada()}
			if _, err := (&GetExitXMLUseCase{Repo: repo, Auth: authXML{}}).Execute(context.Background(), 7); err == nil {
				t.Fatal("esperava erro")
			}
			if repo.saves != 0 {
				t.Fatal("gravou um arquivo que não é o XML da nota")
			}
		})
	}
}

func TestExitXML_RascunhoESemEmissoraNaoTemXML(t *testing.T) {
	rascunho := notaAutorizada()
	rascunho.Status = entity.ExitStatusDraft
	legada := notaAutorizada()
	legada.FocusRef = nil
	for _, e := range []*entity.FiscalExit{rascunho, legada} {
		_, err := (&GetExitXMLUseCase{Repo: &repoXML{exit: e}, Auth: authXML{}}).Execute(context.Background(), 7)
		if err == nil {
			t.Fatalf("nota %s sem XML deveria ser recusada", e.Status)
		}
	}
}

func TestExitXML_CanceladaGuardaOsDoisXML(t *testing.T) {
	aut := `<nfeProc><infNFe Id="NFe` + chaveXMLTeste + `"/></nfeProc>`
	can := `<procEventoNFe><evento><infEvento><chNFe>` + chaveXMLTeste + `</chNFe><tpEvento>110111</tpEvento></infEvento></evento></procEventoNFe>`
	baixou, fim := focusXMLComCancelamento(t, aut, can)
	defer fim()
	nota := notaAutorizada()
	nota.Status = entity.ExitStatusCancelled
	repo := &repoXML{exit: nota}
	uc := &GetExitXMLUseCase{Repo: repo, Auth: authXML{}}
	for i := 0; i < 2; i++ {
		res, err := uc.Execute(context.Background(), 7)
		if err != nil {
			t.Fatal(err)
		}
		if res.XML != aut || res.XMLCancelamento != can {
			t.Fatalf("resultado: %+v", res)
		}
	}
	if *baixou != 2 || repo.saves != 1 || repo.savesCan != 1 {
		t.Fatalf("baixou %d, gravou %d autorizado e %d cancelamento", *baixou, repo.saves, repo.savesCan)
	}
}

// Sem o XML do cancelamento na Focus, a nota cancelada ainda mostra o autorizado.
func TestExitXML_CanceladaSemEventoMostraOAutorizado(t *testing.T) {
	_, fim := focusXML(t, "")
	defer fim()
	nota := notaAutorizada()
	nota.Status = entity.ExitStatusCancelled
	repo := &repoXML{exit: nota, xml: "<nfeProc>guardado</nfeProc>"}
	res, err := (&GetExitXMLUseCase{Repo: repo, Auth: authXML{}}).Execute(context.Background(), 7)
	if err != nil || res.XML != "<nfeProc>guardado</nfeProc>" || res.XMLCancelamento != "" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}
