package fiscal_uc

import "testing"

// A devolução de venda emitida pela empresa é NF-e de ENTRADA (tipo 0); as
// outras notas continuam de saída (tipo 1), inclusive as antigas sem o campo.
func TestPayloadRespeitaOTipoDaNota(t *testing.T) {
	nota := notaBase()
	if p, _ := montarPayloadNFe(nota, duasLinhas(), configBase(), PlanoDaNota{}, TributacaoReforma{}); p.TipoDocumento != 1 {
		t.Fatalf("nota sem tipo é saída: %d", p.TipoDocumento)
	}
	zero := 0
	nota.TipoDocumento = &zero
	nota.Finalidade = 4
	if p, _ := montarPayloadNFe(nota, duasLinhas(), configBase(), PlanoDaNota{}, TributacaoReforma{}); p.TipoDocumento != 0 || p.FinalidadeEmissao != 4 {
		t.Fatalf("devolução de venda: tipo %d finalidade %d", p.TipoDocumento, p.FinalidadeEmissao)
	}
}
