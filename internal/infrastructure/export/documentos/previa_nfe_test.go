package documentos

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
)

func previaDeTeste(itens int) response.PreviaNFeResponse {
	var linhas []map[string]any
	for i := 0; i < itens; i++ {
		linhas = append(linhas, map[string]any{
			"codigo_produto": "RN-01001", "descricao": strings.Repeat("Chapa de aço carbono SAE 1020 cortada a laser com dobra ", 3),
			"codigo_ncm": "72085200", "icms_origem": "0", "icms_situacao_tributaria": "00", "cfop": "5101",
			"unidade_comercial": "UN", "quantidade_comercial": 12.5, "valor_unitario_comercial": "150.1234",
			"valor_bruto": 1876.54, "icms_base_calculo": 1876.54, "icms_valor": 337.78, "icms_aliquota": 18,
			"ipi_valor": 0, "ipi_aliquota": 0,
		})
	}
	payload, _ := json.Marshal(map[string]any{
		"natureza_operacao": "Venda de produção do estabelecimento", "tipo_documento": "1", "items": linhas,
		"modalidade_frete": "0", "nome_transportador": "RODOSUL TRANSPORTES LTDA", "cnpj_transportador": "11444777000161",
		"informacoes_adicionais_contribuinte": "Pedido 1234. Entregar no portão 2.",
		"notas_referenciadas":                 []map[string]any{{"chave_nfe": "41261111444777000161550010000000011000000015"}},
	})
	return response.PreviaNFeResponse{
		FiscalExitID: 9, NumeroNF: 0, Serie: "1", DataEmissao: "2026-11-17", NaturezaOperacao: "Venda",
		Emitente:       response.PreviaParteNFe{Nome: "TECNOFER INDÚSTRIA LTDA", Documento: "11444777000161", IE: "", Logradouro: "Rua A", Numero: "10", Municipio: "Apucarana", UF: "PR", CEP: "86800000"},
		Destinatario:   response.PreviaParteNFe{Nome: "CLIENTE EXEMPLO LTDA", Documento: "04252011000110", Logradouro: "Av. B", Numero: "200", Bairro: "Centro", Municipio: "Curitiba", UF: "PR", CEP: "80010000"},
		Totais:         response.PreviaTotaisNFe{ValorProdutos: 1876.54 * float64(itens), ValorICMS: 337.78 * float64(itens), ValorTotalNF: 1876.54 * float64(itens)},
		Pagamento:      response.PreviaPagamentoNFe{Parcelas: []response.PreviaParcelaNFe{{Numero: 1, Valor: 900, Vencimento: "2026-12-17"}, {Numero: 2, Valor: 976.54, Vencimento: "2027-01-17"}}},
		PayloadEnviado: string(payload),
	}
}

func TestPreviaNFeGeraPDF(t *testing.T) {
	for _, n := range []int{0, 1, 60} {
		pdf := Gerador{}.PreviaNFe(previaDeTeste(n))
		if !bytes.HasPrefix(pdf, []byte("%PDF")) || len(pdf) < 1000 {
			t.Fatalf("%d itens: PDF inválido (%d bytes)", n, len(pdf))
		}
		if n == 60 && bytes.Count(pdf, []byte("/Type /Page\n"))+bytes.Count(pdf, []byte("/Type /Page "))+bytes.Count(pdf, []byte("/Type/Page")) < 2 {
			// 60 itens com descrição em 2-3 linhas não cabem numa folha.
			if !bytes.Contains(pdf, []byte("/Count 2")) && !bytes.Contains(pdf, []byte("/Count 3")) && !bytes.Contains(pdf, []byte("/Count 4")) {
				t.Errorf("60 itens deveriam quebrar a página")
			}
		}
		if out := os.Getenv("PREVIA_PDF_OUT"); out != "" && n == 60 {
			_ = os.WriteFile(out, pdf, 0o600)
		}
	}
	// Payload inválido não derruba: sai a folha sem itens.
	d := previaDeTeste(1)
	d.PayloadEnviado = "{quebrado"
	if pdf := (Gerador{}).PreviaNFe(d); !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("payload inválido derrubou o PDF")
	}
}
