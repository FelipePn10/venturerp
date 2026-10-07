package fiscal_uc

import (
	"os"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestLerCTe(t *testing.T) {
	b, err := os.ReadFile("testdata/cte_frete.xml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := LerCTe(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.Chave != "41261011222333000181570010000004561000004560" || c.Numero != 456 || c.Serie != "1" || c.CFOP != "5353" {
		t.Fatalf("identificação: %+v", c)
	}
	if c.EmitenteCNPJ != "11222333000181" || c.EmitenteUF != "PR" || c.TomadorCNPJ != "98765432000110" {
		t.Fatalf("partes: emitente %s/%s tomador %s", c.EmitenteCNPJ, c.EmitenteUF, c.TomadorCNPJ)
	}
	if !c.ValorPrestacao.Equal(decimal.NewFromInt(1200)) || !c.ValorICMS.Equal(decimal.NewFromInt(144)) || !c.AliqICMS.Equal(decimal.NewFromInt(12)) {
		t.Fatalf("valores: %s %s %s", c.ValorPrestacao, c.ValorICMS, c.AliqICMS)
	}
	if len(c.ChavesNFe) != 1 || c.ChavesNFe[0] != "41261012345678000190550010000123451000123459" {
		t.Fatalf("notas transportadas: %v", c.ChavesNFe)
	}
	if c.DataEmissao.Format("2006-01-02") != "2026-10-02" {
		t.Fatalf("emissão: %v", c.DataEmissao)
	}
	// Sem o envelope e com tomador "outros" (toma4).
	s := string(b)
	i, j := strings.Index(s, "<CTe "), strings.Index(s, "</CTe>")+len("</CTe>")
	direto := strings.Replace(s[i:j], "<toma3><toma>3</toma></toma3>", "<toma4><toma>4</toma><CNPJ>55666777000199</CNPJ></toma4>", 1)
	c2, err := LerCTe([]byte(direto))
	if err != nil {
		t.Fatal(err)
	}
	if c2.Chave != c.Chave || c2.TomadorCNPJ != "55666777000199" {
		t.Fatalf("CT-e sem envelope: chave %s tomador %s", c2.Chave, c2.TomadorCNPJ)
	}
	if _, err := LerCTe([]byte("<NFe><infNFe/></NFe>")); err == nil {
		t.Fatal("NF-e não é CT-e")
	}
	if _, err := LerCTe([]byte(strings.Replace(s, "<vTPrest>1200.00</vTPrest>", "<vTPrest>0</vTPrest>", 1))); err == nil {
		t.Fatal("CT-e sem valor deveria ser recusado")
	}
}

func FuzzLerCTe(f *testing.F) {
	b, _ := os.ReadFile("testdata/cte_frete.xml")
	f.Add(b)
	f.Add([]byte("<CTe/>"))
	f.Fuzz(func(t *testing.T, x []byte) {
		c, err := LerCTe(x)
		if err == nil && (c == nil || !c.ValorPrestacao.IsPositive()) {
			t.Fatal("CT-e aceito sem valor")
		}
	})
}
