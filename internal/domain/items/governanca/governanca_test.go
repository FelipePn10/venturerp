package governanca

import (
	"encoding/json"
	"testing"
)

func ficha(tipo int, dados map[string]any) Ficha {
	f := Ficha{Tipo: tipo, Dados: map[string]json.RawMessage{}}
	for k, v := range dados {
		b, _ := json.Marshal(v)
		f.Dados[k] = b
	}
	return f
}

func TestPendenciasPorTipoEAjuste(t *testing.T) {
	base := map[string]any{"name": "SUPORTE", "pdm_group_code": 3, "pdm_description_technique": "SUPORTE ACO", "warehouse_code": 4,
		"accounting_sale_fiscal_classification_code": "73269090", "accounting_origin": 0}
	fab := ficha(Fabricado, base)
	p := Pendencias(fab, nil, nil)
	chaves := map[string]bool{}
	for _, x := range p {
		chaves[x.Campo] = true
	}
	for _, quer := range []string{"estrutura", "roteiro", "custo_padrao", "commercial_sale_type"} {
		if !chaves[quer] {
			t.Errorf("fabricado sem %s deveria ficar pendente: %+v", quer, p)
		}
	}
	if chaves["fornecedor"] || chaves["accounting_origin"] {
		t.Errorf("fornecedor não vale para fabricado; origem 0 (nacional) é preenchida: %+v", p)
	}
	fab.TemEstrutura, fab.TemRoteiro, fab.TemCustoPadrao = true, true, true
	if p := Pendencias(fab, []string{Engenharia}, nil); len(p) != 0 {
		t.Fatalf("engenharia completa: %+v", p)
	}
	if p := Pendencias(fab, []string{Comercial}, map[string]bool{"commercial_sale_type": false}); len(p) != 0 {
		t.Fatalf("a empresa desligou o tipo de venda: %+v", p)
	}
	if p := Pendencias(fab, []string{Engenharia}, map[string]bool{"engineering_weight": true}); len(p) != 1 || p[0].Campo != "engineering_weight" {
		t.Fatalf("a empresa ligou o peso: %+v", p)
	}
	comp := ficha(Comprado, base)
	comp.CompraEmOutraUM = true
	got := map[string]bool{}
	for _, x := range Pendencias(comp, []string{Suprimentos}, nil) {
		got[x.Campo] = true
	}
	if !got["fornecedor"] || !got["supplies_purchase_uom"] || !got["conversao_compra"] {
		t.Fatalf("comprado sem fornecedor/unidade/conversão: %v", got)
	}
	serv := ficha(Servico, map[string]any{"name": "MONTAGEM", "pdm_group_code": 1, "pdm_description_technique": "MONTAGEM"})
	if p := Pendencias(serv, nil, nil); len(p) != 0 {
		t.Fatalf("serviço não exige NCM, almoxarifado nem estrutura: %+v", p)
	}
}

func TestPreenchidoComJSONVazio(t *testing.T) {
	f := ficha(Comprado, map[string]any{"engineering_weight": map[string]any{"value": 0, "unit": ""}})
	if f.preenchido("engineering_weight") {
		t.Fatal("peso zerado não é preenchido")
	}
	f = ficha(Comprado, map[string]any{"engineering_weight": map[string]any{"value": 2.5, "unit": "KG"}})
	if !f.preenchido("engineering_weight") {
		t.Fatal("peso informado")
	}
}

func TestStatus(t *testing.T) {
	casos := []struct {
		atual string
		m     Mudanca
		ok    bool
	}{
		{Liberado, Mudanca{Para: Bloqueado, Motivo: "qualidade"}, true},
		{Liberado, Mudanca{Para: Bloqueado}, false},
		{EmCadastro, Mudanca{Para: Liberado, Motivo: "x"}, false},
		{Bloqueado, Mudanca{Para: Liberado, Motivo: "resolvido"}, true},
		{Descontinuado, Mudanca{Para: Bloqueado, Motivo: "x"}, false},
		{Liberado, Mudanca{Para: Bloqueado, Motivo: "x", Substituto: new(int64)}, false},
		{Liberado, Mudanca{Para: Descontinuado, Motivo: "fora de linha", Substituto: new(int64)}, true},
	}
	for i, c := range casos {
		if err := c.m.Conferir(c.atual); (err == nil) != c.ok {
			t.Errorf("caso %d: %v", i, err)
		}
	}
	lib := map[string]bool{Engenharia: true, Fiscal: true}
	if StatusAposLiberacao(EmCadastro, []string{Engenharia, Fiscal, Custos}, lib) != EmCadastro {
		t.Fatal("falta custos")
	}
	lib[Custos] = true
	if StatusAposLiberacao(EmCadastro, []string{Engenharia, Fiscal, Custos}, lib) != Liberado {
		t.Fatal("todas liberadas")
	}
	if StatusAposLiberacao(Bloqueado, nil, lib) != Bloqueado {
		t.Fatal("liberação não desbloqueia")
	}
}

func TestDuplicidade(t *testing.T) {
	if s := Semelhanca("Chapa de Aço 3mm", "CHAPA ACO 3MM"); s < 0.99 {
		t.Fatalf("mesmo item sem acento e sem 'de': %v", s)
	}
	if s := Semelhanca("PARAFUSO SEXTAVADO M8X20", "PARAFUSO SEXTAVADO M8 X 20"); s < LimiteDuplicidade {
		t.Fatalf("mesma medida escrita diferente: %v", s)
	}
	if s := Semelhanca("PARAFUSO M8", "CHAPA ACO 3MM"); s >= LimiteDuplicidade {
		t.Fatalf("itens diferentes: %v", s)
	}
	c := Classificar("Balança rodoviária 30t", "84238200", []Candidato{
		{ItemCode: 1, Nome: "BALANCA RODOVIARIA 30T", NCM: "84238200"},
		{ItemCode: 2, Nome: "BALANCA RODOVIARIA 60T", NCM: "84238200"},
		{ItemCode: 3, Nome: "MOTOR ELETRICO", NCM: "84238200"},
	})
	if len(c) != 2 || c[0].ItemCode != 1 || c[0].Motivo != "mesmo nome" {
		t.Fatalf("candidatos: %+v", c)
	}
}
