package pendencias

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
)

func TestPendenciasDoPerfil(t *testing.T) {
	c := map[string]Contagem{"ofs_atrasadas": {Quantidade: 3}, "pagar_vencendo": {Quantidade: 2}, "inventarios_abertos": {}}
	if p := DoPerfil(c, nil); len(p) != 2 {
		t.Fatalf("sem filtro: %+v", p)
	}
	soProducao := func(m acesso.Modulo) bool { return m == acesso.Producao }
	if p := DoPerfil(c, soProducao); len(p) != 1 || p[0].Chave != "ofs_atrasadas" || p[0].Quantidade != 3 {
		t.Fatalf("perfil da produção: %+v", p)
	}
}
