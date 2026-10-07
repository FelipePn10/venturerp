package fiscal_uc

import (
	"errors"
	"os"
	"testing"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
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
func TestLerNFeAceitaEnvelopeDaSefazERecusaOResto(t *testing.T) {
	nota := `<NFe><infNFe><ide><nNF>4321</nNF><serie>1</serie><dhEmi>2026-10-01T10:00:00-03:00</dhEmi></ide>` +
		`<emit><CNPJ>23208854000163</CNPJ><xNome>FORNECEDOR</xNome></emit>` +
		`<det nItem="1"><prod><cProd>A</cProd><xProd>X</xProd><qCom>1</qCom><vUnCom>1</vUnCom><vProd>1</vProd></prod></det>` +
		`</infNFe></NFe>`

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
			lida, err := LerNFe([]byte(c.xml))
			if c.aceita {
				if err != nil {
					t.Fatalf("esperava aceitar, recusou: %v", err)
				}
				if lida.Numero != c.numero {
					t.Errorf("nNF = %q, esperado %q", lida.Numero, c.numero)
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

// O XML real tem namespace, grupos de ICMS por CST (ICMS00, ICMSSN102…), o
// IPI depois do <cEnq> e as duplicatas em <cobr>. Antes, o leitor procurava
// <ICMS><CST> direto e nunca achava nada: todo item entrava com ICMS zero.
func TestLerNFeLeImpostosDuplicatasEChave(t *testing.T) {
	conteudo, err := os.ReadFile("testdata/nfe_mp_epi.xml")
	if err != nil {
		t.Fatal(err)
	}
	n, err := LerNFe(conteudo)
	if err != nil {
		t.Fatal(err)
	}
	if n.ChaveAcesso != "41261012345678000190550010000123451000123459" {
		t.Errorf("chave = %q", n.ChaveAcesso)
	}
	if n.Protocolo != "141260000012345" || n.NaturezaOperacao != "VENDA DE MERCADORIA" {
		t.Errorf("protocolo/natureza = %q/%q", n.Protocolo, n.NaturezaOperacao)
	}
	if n.DataEmissao.Format("2006-01-02") != "2026-10-01" || n.DataSaidaEntrada == nil || n.DataSaidaEntrada.Format("2006-01-02") != "2026-10-02" {
		t.Errorf("datas = %v / %v", n.DataEmissao, n.DataSaidaEntrada)
	}
	if n.DestinatarioCNPJ != "98765432000110" || n.EmitenteUF != "PR" {
		t.Errorf("destinatário/UF = %q/%q", n.DestinatarioCNPJ, n.EmitenteUF)
	}
	if len(n.Itens) != 2 {
		t.Fatalf("itens = %d", len(n.Itens))
	}
	chapa, luva := n.Itens[0], n.Itens[1]
	if chapa.CSTICMS != "00" || !chapa.ValorICMS.Equal(decimal.RequireFromString("5820")) || !chapa.AliqICMS.Equal(decimal.RequireFromString("12")) {
		t.Errorf("ICMS da chapa = CST %q valor %s aliq %s", chapa.CSTICMS, chapa.ValorICMS, chapa.AliqICMS)
	}
	if chapa.CSTIPI != "50" || !chapa.ValorIPI.Equal(decimal.RequireFromString("1500")) {
		t.Errorf("IPI da chapa = CST %q valor %s", chapa.CSTIPI, chapa.ValorIPI)
	}
	if chapa.EAN != "" {
		t.Errorf("'SEM GTIN' deveria virar EAN vazio, veio %q", chapa.EAN)
	}
	if chapa.PedidoCompra != "PC-77" {
		t.Errorf("xPed = %q", chapa.PedidoCompra)
	}
	if !chapa.ValorContabil().Equal(decimal.RequireFromString("50000")) {
		t.Errorf("valor contábil da chapa = %s, esperado 50000 (produto + frete + IPI)", chapa.ValorContabil())
	}
	if luva.CSTICMS != "102" || luva.CSTIPI != "53" || luva.EAN != "7891234567895" || luva.Unidade != "CX" {
		t.Errorf("luva = CSOSN %q IPI %q EAN %q UN %q", luva.CSTICMS, luva.CSTIPI, luva.EAN, luva.Unidade)
	}
	if len(n.Duplicatas) != 3 || n.Duplicatas[2].Vencimento.Format("2006-01-02") != "2026-12-30" {
		t.Fatalf("duplicatas = %+v", n.Duplicatas)
	}
	if n.SemPagamento() {
		t.Error("nota com boleto não é 'sem pagamento'")
	}
}

// A nota de 100 mil (50 mil de matéria-prima, 50 mil de EPI) em 3 parcelas:
// cada parcela nasce com a distribuição proporcional, e a soma por conta
// fecha com os itens.
func TestMontarEntradaDoXMLGeraParcelasDistribuidas(t *testing.T) {
	conteudo, _ := os.ReadFile("testdata/nfe_mp_epi.xml")
	n, err := LerNFe(conteudo)
	if err != nil {
		t.Fatal(err)
	}
	e, err := montarEntradaDoXML(n, string(conteudo), n.DataEmissao)
	if err != nil {
		t.Fatal(err)
	}
	if e.Itens[0].ItemCode != nil {
		t.Fatal("o código do fornecedor não pode virar código do nosso cadastro")
	}
	if len(e.Parcelas) != 3 || e.Parcelas[0].Origem != "XML" {
		t.Fatalf("parcelas = %+v", e.Parcelas)
	}
	mp, epi := int64(10), int64(20)
	code := int64(1)
	e.Itens[0].ItemCode, e.Itens[0].PlanoContasID = &code, &mp
	e.Itens[1].ItemCode, e.Itens[1].PlanoContasID = &code, &epi
	if err := entrada.DistribuirProporcional(e.Parcelas, entrada.TotalPorConta(e.Itens)); err != nil {
		t.Fatal(err)
	}
	// Material que movimenta estoque precisa de depósito para a nota fechar.
	if !entrada.TemImpedimento(entrada.ConferirEntrada(e)) {
		t.Fatal("item sem depósito deveria impedir a aprovação")
	}
	dep := int64(1)
	e.Itens[0].WarehouseID, e.Itens[1].WarehouseID = &dep, &dep
	if pend := entrada.ConferirEntrada(e); !entrada.TemImpedimento(pend) || pend[0].Campo != "supplier_code" {
		t.Fatalf("emitente sem cadastro deveria impedir: %+v", pend)
	}
	forn := int64(77001)
	e.SupplierCode = &forn
	if pend := entrada.ConferirEntrada(e); entrada.TemImpedimento(pend) {
		t.Fatalf("pendências: %+v", pend)
	}
}
