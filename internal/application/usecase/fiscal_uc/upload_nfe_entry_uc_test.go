package fiscal_uc

import (
	"errors"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// O arquivo que o contador manda é o que a SEFAZ devolveu na autorização:
// <nfeProc><NFe>…</NFe><protNFe>…</protNFe></nfeProc>. Aceitar só <NFe> na raiz
// recusava justamente o caso mais comum, e o usuário lia "não é um XML de NF-e
// válido" olhando para o XML correto da nota dele.
//
// O outro lado importa igual: XML bem formado que NÃO é nota precisa ser recusado.
// Sem a conferência da raiz, `xml.Unmarshal` em `<foo/>` passaria e a entrada
// fiscal seria gravada com todos os campos zerados — o mesmo defeito que o OFX
// tinha, de aceitar qualquer arquivo e dizer "sucesso".
func TestLerXMLdeNFeAceitaEnvelopeDaSefazERecusaOResto(t *testing.T) {
	nota := `<NFe><infNFe><ide><nNF>4321</nNF><serie>1</serie></ide>` +
		`<emit><CNPJ>23208854000163</CNPJ><xNome>FORNECEDOR</xNome></emit></infNFe></NFe>`

	casos := []struct {
		nome   string
		xml    string
		aceita bool
		numero string
	}{
		{"NFe na raiz", nota, true, "4321"},
		{"dentro do envelope nfeProc", `<nfeProc>` + nota + `<protNFe/></nfeProc>`, true, "4321"},
		{"com espaço e quebra de linha antes", "\n  " + nota + "\n", true, "4321"},
		{"envelope sem a nota dentro", `<nfeProc><protNFe/></nfeProc>`, false, ""},
		{"XML bem formado que não é NF-e", `<foo/>`, false, ""},
		{"outra raiz", `<pedido><item>1</item></pedido>`, false, ""},
		{"não é XML", `isto nao e xml`, false, ""},
		{"JSON", `{"a":1}`, false, ""},
		{"vazio", ``, false, ""},
		{"só espaços", "   \n ", false, ""},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			lida, err := lerXMLdeNFe(c.xml)
			if c.aceita {
				if err != nil {
					t.Fatalf("esperava aceitar, recusou: %v", err)
				}
				if lida.InfNFe.Ide.NF != c.numero {
					t.Errorf("nNF = %q, esperado %q", lida.InfNFe.Ide.NF, c.numero)
				}
				return
			}
			if err == nil {
				t.Fatal("esperava recusar, aceitou — a entrada fiscal entraria com dado zerado")
			}
			// Recusa tem de ser de validação (→ 422), nunca falha genérica (→ 500).
			var validacao *errorsuc.ValidationError
			if !errors.As(err, &validacao) {
				t.Errorf("erro deveria ser de validação, veio %T: %v", err, err)
			}
		})
	}
}
