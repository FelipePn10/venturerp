package middleware

import (
	"net/http"
	"strings"
)

// Rotas cuja credencial vai no caminho: o token do portal do cliente e o
// segredo do aviso do banco. Quem lê log, auditoria ou rastro não pode abri-las.
var prefixosComSegredo = []string{"/api/portal/", "/api/integracoes/inter/webhook/"}

// CaminhoSemSegredo troca o segmento secreto por "***" para log e auditoria.
func CaminhoSemSegredo(p string) string {
	for _, pre := range prefixosComSegredo {
		if strings.HasPrefix(p, pre) {
			resto := strings.TrimPrefix(p, pre)
			if i := strings.IndexByte(resto, '/'); i >= 0 {
				return pre + "***" + resto[i:]
			}
			return pre + "***"
		}
	}
	return p
}

// RastrearRequisicao: filtro do OpenTelemetry. As rotas com segredo no caminho
// não viram rastro (o span guardaria a URL inteira).
func RastrearRequisicao(r *http.Request) bool {
	return CaminhoSemSegredo(r.URL.Path) == r.URL.Path
}
