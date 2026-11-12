// Package certificado entrega o certificado de qualidade em PDF; o layout mora
// em export/documentos, junto dos demais documentos impressos.
package certificado

import (
	"time"

	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/certificado"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/documentos"
)

// Gerar devolve o PDF do certificado, sem logo.
func Gerar(c *dom.Certificado, agora time.Time) []byte { return GerarComLogo(c, agora, nil) }

// GerarComLogo devolve o PDF do certificado com a logo atual da empresa.
func GerarComLogo(c *dom.Certificado, agora time.Time, logo []byte) []byte {
	return documentos.CertificadoQualidade(c, agora, logo)
}
