package fiscal_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type catalogoFiscalFake map[string]string

func (c catalogoFiscalFake) CodigosFiscais(_ context.Context, codigos ...string) (map[string]string, error) {
	out := map[string]string{}
	for _, k := range codigos {
		if v, ok := c[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

// A nota sai com o código fiscal do catálogo (TONELADA → T); a unidade que
// nem assim cabe em 6 caracteres é recusada antes de transmitir.
func TestNFeUsaCodigoFiscalDaUnidade(t *testing.T) {
	cat := catalogoFiscalFake{"TONELADA": "T", "MICROMETRO": "UM", "UN": "UN"}
	p := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{
		{UnidadeComercial: "tonelada"},
		{UnidadeComercial: "UN", UnidadeTributavel: "MICROMETRO"},
		{UnidadeComercial: "KG"},
	}}
	if err := aplicarCodigosFiscaisDeUnidade(context.Background(), cat, &p); err != nil {
		t.Fatal(err)
	}
	if p.Items[0].UnidadeComercial != "T" || p.Items[1].UnidadeTributavel != "UM" || p.Items[1].UnidadeComercial != "UN" || p.Items[2].UnidadeComercial != "KG" {
		t.Fatalf("unidades traduzidas erradas: %+v", p.Items)
	}

	ruim := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{UnidadeComercial: "BOBINAS"}}}
	err := aplicarCodigosFiscaisDeUnidade(context.Background(), cat, &ruim)
	if err == nil || !strings.Contains(err.Error(), "código fiscal") {
		t.Fatalf("unidade de 7 caracteres sem código fiscal deveria ser recusada: %v", err)
	}
	// Sem catálogo injetado: não traduz, mas ainda recusa o que a SEFAZ recusaria.
	sem := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{UnidadeComercial: "PC"}}}
	if err := aplicarCodigosFiscaisDeUnidade(context.Background(), nil, &sem); err != nil || sem.Items[0].UnidadeComercial != "PC" {
		t.Fatalf("sem catálogo, PC passa como está: %v %+v", err, sem.Items)
	}
}
