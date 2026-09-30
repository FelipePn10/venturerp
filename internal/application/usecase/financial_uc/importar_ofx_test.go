package financial_uc

import (
	"strings"
	"testing"
)

const ofxValido = `OFXHEADER:100
DATA:OFXSGML
VERSION:102

<OFX>
<BANKMSGSRSV1><STMTTRNRS><STMTRS>
<CURDEF>BRL
<BANKACCTFROM><BANKID>341<ACCTID>12345-6<ACCTTYPE>CHECKING</BANKACCTFROM>
<BANKTRANLIST>
<DTSTART>20260901
<DTEND>20260930
<STMTTRN><TRNTYPE>CREDIT<DTPOSTED>20260903120000<TRNAMT>1500.00<FITID>A1<MEMO>Recebimento cliente</STMTTRN>
<STMTTRN><TRNTYPE>DEBIT<DTPOSTED>20260905120000<TRNAMT>-230.50<FITID>A2<MEMO>Pagamento fornecedor</STMTTRN>
</BANKTRANLIST>
</STMTRS></STMTTRNRS></BANKMSGSRSV1>
</OFX>`

// TestOFXRecusaArquivoQueNaoEhOFX é o teste do defeito relatado: importar um JSON
// respondia SUCESSO com zero lançamento, e quem importava concluía que o extrato
// estava vazio.
func TestOFXRecusaArquivoQueNaoEhOFX(t *testing.T) {
	casos := map[string]struct {
		conteudo    string
		esperaTexto string
	}{
		"json":        {`{"transacoes":[{"valor":100,"data":"2026-09-03"}]}`, "JSON"},
		"pdf":         {"%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>", "PDF"},
		"csv":         {"data;valor;descricao\n03/09/2026;1500,00;Recebimento", "delimitador"},
		"xlsx":        {"PK\x03\x04\x14\x00\x00\x00", "ZIP"},
		"xml não-OFX": {`<?xml version="1.0"?><nfeProc><NFe><infNFe/></NFe></nfeProc>`, "XML"},
		"texto solto": {"extrato do mes de setembro", "sem estrutura OFX"},
		"vazio":       {"   \n  ", "vazio"},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := parseOFX(caso.conteudo)
			if err == nil {
				t.Fatalf("%s foi aceito como extrato OFX", nome)
			}
			if !strings.Contains(err.Error(), caso.esperaTexto) {
				t.Fatalf("a mensagem não diz o que foi recebido (%q): %v", caso.esperaTexto, err)
			}
			// A recusa tem de ser legível: mensagem genérica aqui deixa a pessoa
			// sem saber qual arquivo baixar no banco.
			if !strings.Contains(err.Error(), "OFX") {
				t.Fatalf("a mensagem não diz qual formato usar: %v", err)
			}
		})
	}
}

// TestOFXRecusaArquivoValidoSemLancamento: OFX correto mas sem <STMTTRN> é período
// errado ou arquivo errado. Responder sucesso com zero importado é o mesmo defeito
// com outra roupa.
func TestOFXRecusaArquivoValidoSemLancamento(t *testing.T) {
	semLancamento := `OFXHEADER:100
<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS>
<BANKACCTFROM><BANKID>341<ACCTID>12345-6</BANKACCTFROM>
<BANKTRANLIST><DTSTART>20260901<DTEND>20260930</BANKTRANLIST>
</STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`
	_, err := parseOFX(semLancamento)
	if err == nil {
		t.Fatal("OFX sem nenhum lançamento foi aceito")
	}
	if !strings.Contains(err.Error(), "lançamento") {
		t.Fatalf("a mensagem não explica o que falta: %v", err)
	}
}

func TestOFXAceitaSGMLELeOsLancamentos(t *testing.T) {
	extrato, err := parseOFX(ofxValido)
	if err != nil {
		t.Fatalf("OFX válido recusado: %v", err)
	}
	if len(extrato.Transacoes) != 2 {
		t.Fatalf("leu %d lançamento(s), esperado 2", len(extrato.Transacoes))
	}
	if extrato.BankID != "341" || extrato.AccountID != "12345-6" {
		t.Fatalf("identificação da conta = banco %q conta %q", extrato.BankID, extrato.AccountID)
	}
	if extrato.DtStart != "20260901" || extrato.DtEnd != "20260930" {
		t.Fatalf("período = %q a %q", extrato.DtStart, extrato.DtEnd)
	}
	// O sinal decide crédito × débito: perder o sinal invertia a conciliação.
	if extrato.Transacoes[0].TrnAmt != "1500.00" || extrato.Transacoes[1].TrnAmt != "-230.50" {
		t.Fatalf("valores lidos: %q e %q", extrato.Transacoes[0].TrnAmt, extrato.Transacoes[1].TrnAmt)
	}
	if extrato.Transacoes[0].Memo != "Recebimento cliente" {
		t.Fatalf("histórico lido: %q", extrato.Transacoes[0].Memo)
	}
}

func TestOFXAceitaXMLVersao2(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<?OFX OFXHEADER="200" VERSION="211"?>
<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS>
<BANKACCTFROM><BANKID>033</BANKID><ACCTID>98765</ACCTID></BANKACCTFROM>
<BANKTRANLIST><DTSTART>20260901</DTSTART><DTEND>20260930</DTEND>
<STMTTRN><TRNTYPE>CREDIT</TRNTYPE><DTPOSTED>20260910</DTPOSTED><TRNAMT>99.90</TRNAMT><FITID>X1</FITID><MEMO>PIX recebido</MEMO></STMTTRN>
</BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`
	extrato, err := parseOFX(xml)
	if err != nil {
		t.Fatalf("OFX 2.x recusado: %v", err)
	}
	if len(extrato.Transacoes) != 1 || extrato.Transacoes[0].FitID != "X1" {
		t.Fatalf("leitura do OFX 2.x errada: %+v", extrato.Transacoes)
	}
	if extrato.AccountID != "98765" {
		t.Fatalf("conta do arquivo = %q", extrato.AccountID)
	}
}

// TestIdentificacaoDaContaNaoVemDeDentroDoLancamento: a tag de conta lida depois
// dos lançamentos pegaria valor de dentro de um <STMTTRN>, e o aviso de "conta
// diferente" apontaria a conta errada.
func TestIdentificacaoDaContaNaoVemDeDentroDoLancamento(t *testing.T) {
	comRuido := `OFXHEADER:100
<OFX><BANKACCTFROM><BANKID>341<ACCTID>11111</BANKACCTFROM>
<BANKTRANLIST><DTSTART>20260901<DTEND>20260930
<STMTTRN><TRNTYPE>CREDIT<DTPOSTED>20260903<TRNAMT>10.00<FITID>Z1<ACCTID>99999<MEMO>ruido</STMTTRN>
</BANKTRANLIST></OFX>`
	extrato, err := parseOFX(comRuido)
	if err != nil {
		t.Fatalf("recusado: %v", err)
	}
	if extrato.AccountID != "11111" {
		t.Fatalf("conta do arquivo = %q, esperado 11111 (a do cabeçalho)", extrato.AccountID)
	}
}

func TestComparacaoDeContaIgnoraFormatacao(t *testing.T) {
	if !mesmoNumero("12345-6", "123456") {
		t.Fatal("conta com dígito separado deveria casar com a mesma sem separador")
	}
	if !mesmoNumero("0001 / 12.345-6", "123456") {
		t.Fatal("conta com máscara deveria casar")
	}
	if mesmoNumero("12345-6", "65432-1") {
		t.Fatal("contas diferentes não podem casar")
	}
	// Piso de 4 dígitos: sem ele, "6" casaria com qualquer conta terminada em 6 e
	// o aviso de conta trocada nunca apareceria.
	if mesmoNumero("123", "999123") {
		t.Fatal("número curto não deve casar por sufixo")
	}
}

func TestDataOFXAceitaOsTresFormatos(t *testing.T) {
	casos := map[string]string{
		"20260903":                   "03/09/2026",
		"20260903120000":             "03/09/2026",
		"20260903120000.000[-3:BRT]": "03/09/2026",
	}
	for bruto, esperado := range casos {
		t.Run(bruto, func(t *testing.T) {
			d, err := parseOFXDate(bruto)
			if err != nil {
				t.Fatalf("data %q recusada: %v", bruto, err)
			}
			if got := d.Format("02/01/2006"); got != esperado {
				t.Fatalf("data %q lida como %s, esperado %s", bruto, got, esperado)
			}
		})
	}
	if _, err := parseOFXDate("03/09/2026"); err == nil {
		t.Fatal("data no formato brasileiro deveria ser recusada: não é data OFX")
	}
}
