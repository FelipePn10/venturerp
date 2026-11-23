package handler

import "testing"

func TestDocumentoValido(t *testing.T) {
	casos := []struct {
		tipo, corpo string
		ok          bool
	}{
		{"danfe", "%PDF-1.4 ...", true},
		{"danfe", "<html>erro</html>", false},
		{"xml", `<?xml version="1.0"?><nfeProc versao="4.00">`, true},
		{"xml", "\ufeff<?xml version=\"1.0\"?>\n<NFe xmlns=", true},
		{"xml", "<html><body>Not found</body></html>", false},
		{"xml", "%PDF-1.4", false},
	}
	for _, c := range casos {
		if got := documentoValido(c.tipo, []byte(c.corpo)); got != c.ok {
			t.Errorf("%s %q: %v", c.tipo, c.corpo, got)
		}
	}
}
