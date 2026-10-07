package fiscal_uc

import (
	"os"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// NT 2025.002 (reforma tributária): IBS/CBS por item e nos totais, imposto
// seletivo, retenções federais e ISS retido.
func TestLerNFeReformaTributariaERetencoes(t *testing.T) {
	base, err := os.ReadFile("testdata/nfe_mp_epi.xml")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(base), "</COFINS>\n        </imposto>",
		`</COFINS>
          <IBSCBS><CST>000</CST><cClassTrib>000001</cClassTrib><gIBSCBS><vBC>48000.00</vBC>
            <gIBSUF><pIBSUF>0.10</pIBSUF><vIBSUF>48.00</vIBSUF></gIBSUF>
            <gIBSMun><pIBSMun>0.00</pIBSMun><vIBSMun>0.00</vIBSMun></gIBSMun>
            <gCBS><pCBS>0.90</pCBS><vCBS>432.00</vCBS></gCBS></gIBSCBS></IBSCBS>
          <IS><vIS>12.34</vIS></IS>
        </imposto>`, 1)
	s = strings.Replace(s, "</ICMSTot>", `</ICMSTot><ISSQNtot><vISSRet>25.00</vISSRet></ISSQNtot>
        <retTrib><vRetPIS>65.00</vRetPIS><vRetCOFINS>300.00</vRetCOFINS><vRetCSLL>100.00</vRetCSLL><vBCIRRF>10000.00</vBCIRRF><vIRRF>150.00</vIRRF><vBCRetPrev>1000.00</vBCRetPrev><vRetPrev>110.00</vRetPrev></retTrib>
        <ISTot><vIS>12.34</vIS></ISTot>`, 1)
	n, err := LerNFe([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	it := n.Itens[0]
	if it.CSTIBSCBS != "000" || it.ClassTrib != "000001" || !it.ValorIBS.Equal(decimal.NewFromInt(48)) || !it.ValorCBS.Equal(decimal.NewFromInt(432)) || !it.ValorIS.Equal(decimal.RequireFromString("12.34")) {
		t.Fatalf("IBS/CBS/IS do item: %+v", it)
	}
	// Sem <IBSCBSTot>, o total vem da soma dos itens.
	if !n.ValorIBS.Equal(decimal.NewFromInt(48)) || !n.ValorCBS.Equal(decimal.NewFromInt(432)) || !n.BaseIBSCBS.Equal(decimal.NewFromInt(48000)) {
		t.Fatalf("totais IBS/CBS: %s %s %s", n.ValorIBS, n.ValorCBS, n.BaseIBSCBS)
	}
	if !n.ValorIS.Equal(decimal.RequireFromString("12.34")) {
		t.Fatalf("IS total: %s", n.ValorIS)
	}
	if !n.TotalRetencoes().Equal(decimal.NewFromInt(65 + 300 + 100 + 150 + 110 + 25)) {
		t.Fatalf("retenções: %s", n.TotalRetencoes())
	}
}

// O leitor recebe arquivo de qualquer origem: nunca pode entrar em pânico.
func FuzzLerNFe(f *testing.F) {
	base, err := os.ReadFile("testdata/nfe_mp_epi.xml")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(base)
	f.Add([]byte("<NFe><infNFe></infNFe></NFe>"))
	f.Add(append([]byte{0xEF, 0xBB, 0xBF}, []byte(`<?xml version="1.0"?><nfeProc/>`)...))
	f.Add([]byte("não é xml"))
	f.Fuzz(func(t *testing.T, b []byte) {
		n, err := LerNFe(b)
		if err == nil && (n == nil || len(n.Itens) == 0) {
			t.Fatal("nota aceita sem itens")
		}
	})
}
