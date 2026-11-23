package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

func TestEnviaPeloModelo(t *testing.T) {
	var recebido map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/PHONE/messages" || r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&recebido)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid"}]}`))
	}))
	defer srv.Close()
	c := New(Config{Token: "tok", PhoneNumberID: "PHONE", Template: "aviso_erp"})
	c.base = srv.URL
	if err := c.Enviar(context.Background(), ports.MensagemCurta{Para: "5544999990000", Texto: "Documento: NF 1\nValor: R$ 10,00"}); err != nil {
		t.Fatal(err)
	}
	tpl := recebido["template"].(map[string]any)
	p := tpl["components"].([]any)[0].(map[string]any)["parameters"].([]any)[0].(map[string]any)
	if tpl["name"] != "aviso_erp" || p["text"] != "Documento: NF 1 · Valor: R$ 10,00" || recebido["to"] != "5544999990000" {
		t.Fatalf("corpo: %+v", recebido)
	}
}

func TestNumeroInternacional(t *testing.T) {
	for in, want := range map[string]string{"(44) 99999-0000": "5544999990000", "4433334444": "554433334444", "+55 44 99999-0000": "5544999990000", "044999990000": "5544999990000"} {
		if got := NumeroInternacional(in); got != want {
			t.Errorf("%s → %s, quer %s", in, got, want)
		}
	}
}
