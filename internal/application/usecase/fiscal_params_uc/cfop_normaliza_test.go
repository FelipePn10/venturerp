package fiscal_params_uc

import (
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// Campo vazio recebe o padrão (o PUT não grava enum vazio); valor inválido é
// recusado com os valores aceitos; origem IPI vazia vira "sem origem".
func TestNormalizaCFOP(t *testing.T) {
	vazio := ""
	c := &entity.CFOP{Code: 5949, OrigemClasIPI: &vazio}
	if err := normalizaCFOP(c); err != nil {
		t.Fatal(err)
	}
	if c.Utilization != entity.CfopUtilizationIndustrializacaoComercio || c.IndOperacao != entity.CfopIndOperacaoNormal ||
		c.TipoUtilizacao != entity.CfopTipoUtilizacaoNormal || c.OrigemClasIPI != nil {
		t.Fatalf("padrões não aplicados: %+v", c)
	}
	for nome, caso := range map[string]struct {
		cfop  entity.CFOP
		aceit string
	}{
		"código zero":        {entity.CFOP{}, "código do CFOP"},
		"utilização":         {entity.CFOP{Code: 1, Utilization: "XPTO"}, "INDUSTRIALIZACAO_COMERCIO"},
		"indicador":          {entity.CFOP{Code: 1, IndOperacao: "X"}, "ENERGIA_ELETRICA"},
		"tipo de utilização": {entity.CFOP{Code: 1, TipoUtilizacao: "Z"}, "EXPORTACAO"},
	} {
		c := caso.cfop
		err := normalizaCFOP(&c)
		if err == nil || !strings.Contains(err.Error(), caso.aceit) {
			t.Fatalf("%s: erro = %v; esperado citar %q", nome, err, caso.aceit)
		}
	}
}
