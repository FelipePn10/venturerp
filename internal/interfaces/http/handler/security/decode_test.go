package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// O corpo ruim volta com mensagem em português, sem o texto do decodificador.
func TestDecodeBodyMensagemLegivel(t *testing.T) {
	type dto struct {
		Hora int `json:"run_hour"`
	}
	casos := []struct {
		corpo  string
		status int
	}{
		{`{"run_hour": "duas"}`, http.StatusUnprocessableEntity},
		{`{"run_hour": 2,}`, http.StatusBadRequest},
		{`{"run_hour": `, http.StatusBadRequest},
		{``, http.StatusBadRequest},
	}
	for _, c := range casos {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(c.corpo))
		var d dto
		if DecodeBody(w, r, &d) {
			t.Fatalf("%q aceito", c.corpo)
		}
		if w.Code != c.status {
			t.Fatalf("%q: status %d, esperado %d", c.corpo, w.Code, c.status)
		}
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		texto := w.Body.String()
		for _, ruim := range []string{"invalid character", "unexpected EOF", "cannot unmarshal", "Go struct"} {
			if strings.Contains(texto, ruim) {
				t.Fatalf("%q: mensagem técnica vazou: %s", c.corpo, texto)
			}
		}
	}
}
