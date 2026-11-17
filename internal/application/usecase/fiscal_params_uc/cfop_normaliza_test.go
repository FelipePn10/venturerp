package fiscal_params_uc

import (
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// Código zero é recusado; origem IPI vazia vira "sem origem". (Utilização,
// indicador e tipo de utilização saíram na v1.20.0: nada os usava.)
func TestNormalizaCFOP(t *testing.T) {
	vazio := ""
	c := &entity.CFOP{Code: 5949, OrigemClasIPI: &vazio}
	if err := normalizaCFOP(c); err != nil {
		t.Fatal(err)
	}
	if c.OrigemClasIPI != nil {
		t.Fatalf("origem IPI vazia não virou nil: %+v", c)
	}
	for nome, caso := range map[string]struct {
		cfop  entity.CFOP
		aceit string
	}{
		"código zero": {entity.CFOP{}, "código do CFOP"},
	} {
		c := caso.cfop
		err := normalizaCFOP(&c)
		if err == nil || !strings.Contains(err.Error(), caso.aceit) {
			t.Fatalf("%s: erro = %v; esperado citar %q", nome, err, caso.aceit)
		}
	}
}
