package focusnfe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// O caminho absoluto (DANFSE no S3 da Focus) não recebe o token da empresa.
func TestBaixarArquivo_TokenSoParaAFocus(t *testing.T) {
	comToken := map[string]bool{}
	handler := func(nome string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, _, ok := r.BasicAuth()
			comToken[nome] = ok
			_, _ = w.Write([]byte("<xml/>"))
		}
	}
	focus := httptest.NewServer(handler("focus"))
	defer focus.Close()
	externo := httptest.NewServer(handler("externo"))
	defer externo.Close()
	t.Setenv("FOCUSNFE_BASE_URL", focus.URL)
	cli := NewClient("tok", "homologacao")

	if _, err := cli.BaixarArquivo(context.Background(), "/arquivos/a.xml"); err != nil {
		t.Fatal(err)
	}
	if _, err := cli.BaixarArquivo(context.Background(), externo.URL+"/arquivos/a.xml"); err != nil {
		t.Fatal(err)
	}
	if !comToken["focus"] || comToken["externo"] {
		t.Fatalf("token enviado: %v", comToken)
	}
}

// Acima do limite, o download é recusado em vez de cortado.
func TestBaixarArquivo_RecusaAcimaDoLimite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<xml>0123456789</xml>"))
	}))
	defer srv.Close()
	t.Setenv("FOCUSNFE_BASE_URL", srv.URL)
	antes := limiteArquivo
	defer func() { limiteArquivo = antes }()
	limiteArquivo = 10
	if _, err := NewClient("tok", "homologacao").BaixarArquivo(context.Background(), "/arquivos/a.xml"); err == nil {
		t.Fatal("arquivo acima do limite foi aceito")
	}
	limiteArquivo = 21
	if b, err := NewClient("tok", "homologacao").BaixarArquivo(context.Background(), "/arquivos/a.xml"); err != nil || len(b) != 21 {
		t.Fatalf("arquivo no limite: %d %v", len(b), err)
	}
}
