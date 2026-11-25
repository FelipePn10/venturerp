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
