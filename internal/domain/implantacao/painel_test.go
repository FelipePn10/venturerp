package implantacao

import "testing"

func TestPlacarPonderaPelaPendencia(t *testing.T) {
	tudoCerto := map[string]Contagem{}
	if p := Montar(tudoCerto); p.Percentual != 100 || p.Prontos != len(Catalogo) {
		t.Fatalf("sem pendência = 100%%: %+v", p)
	}
	// Config fiscal faltando (peso 3 de 23) e metade dos fornecedores sem endereço (peso 2 × 0,5).
	p := Montar(map[string]Contagem{"config_fiscal": {Pendentes: 1, Total: 1}, "fornecedores_endereco": {Pendentes: 5, Total: 10}})
	if p.Percentual != 82 || p.Prontos != len(Catalogo)-2 {
		t.Fatalf("placar = %d%% (%d prontos), quer 82%%", p.Percentual, p.Prontos)
	}
	for _, it := range p.Itens {
		if it.Chave == "fornecedores_endereco" && (it.Percentual != 50 || it.Pronto || it.Tela == "") {
			t.Fatalf("fornecedores: %+v", it)
		}
	}
}
