package focusnfe

import (
	"encoding/json"
	"testing"
)

func TestPayloadLevaTransportadorEntregaEComplemento(t *testing.T) {
	p := NFEPayload{
		ModalidadeFrete: 1,
		Destinatario: NFEDestinatario{CNPJCPF: "11.222.333/0001-81", Nome: "CLIENTE", Logradouro: "RUA A", Numero: "1", Complemento: "SALA 2",
			Bairro: "CENTRO", Municipio: "SAO PAULO", CodigoMunicipio: "3550308", UF: "SP", CEP: "01001-000", IndicadorIE: 9},
		Transportador: &NFETransportador{CNPJCPF: "12.345.678/0001-95", Nome: "TRANSPORTES LTDA", IE: "123.456.789", Endereco: "RUA B, 2", Municipio: "CURITIBA", UF: "PR"},
		Entrega:       &NFELocalEntrega{CNPJCPF: "11222333000181", Nome: "CLIENTE", Logradouro: "RUA DA FABRICA", Numero: "100", Bairro: "DISTRITO", Municipio: "CAMPINAS", UF: "SP", CEP: "13050-000"},
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	quer := map[string]any{
		"modalidade_frete": 1.0, "complemento_destinatario": "SALA 2", "codigo_municipio_destinatario": "3550308",
		"cnpj_transportador": "12345678000195", "nome_transportador": "TRANSPORTES LTDA", "inscricao_estadual_transportador": "123456789",
		"endereco_transportador": "RUA B, 2", "municipio_transportador": "CURITIBA", "uf_transportador": "PR",
		"cnpj_entrega": "11222333000181", "logradouro_entrega": "RUA DA FABRICA", "numero_entrega": "100", "bairro_entrega": "DISTRITO",
		"municipio_entrega": "CAMPINAS", "uf_entrega": "SP", "cep_entrega": "13050000",
	}
	for k, v := range quer {
		if m[k] != v {
			t.Errorf("%s = %v, quer %v", k, m[k], v)
		}
	}

	p.Transportador, p.Entrega = nil, nil
	raw, _ = json.Marshal(p)
	m = map[string]any{}
	_ = json.Unmarshal(raw, &m)
	for _, k := range []string{"cnpj_transportador", "nome_transportador", "cnpj_entrega", "logradouro_entrega"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s não devia ir sem transportadora/entrega", k)
		}
	}
}
