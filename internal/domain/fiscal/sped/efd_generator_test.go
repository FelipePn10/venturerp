package sped

import (
	"strings"
	"testing"
	"time"
)

func sampleParams() EFDParams {
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	}
	return EFDParams{
		Empresa: EFDEmpresa{
			CNPJ: "12345678000199", Nome: "Metalúrgica Teste LTDA", UF: "PR",
			IE: "9012345678", CodigoMunicipio: "4106902", RegimeTributario: "A",
			CodigoFinalizacao: "0", ContabilistaNome: "Contador X", ContabilistaCPF: "11122233344",
		},
		Periodo: EFDPeriodo{
			DataInicial: d(2026, time.January, 1), DataFinal: d(2026, time.January, 31),
			IndicadorSituacaoEspecial: "0",
		},
		Participantes: []EFDParticipante{{CodPart: "P1", Nome: "Fornecedor A", CNPJ: "99887766000155"}},
		Unidades:      []EFDUnidade{{CodUnd: "UN", DescUnd: "Unidade"}, {CodUnd: "KG", DescUnd: "Quilograma"}},
		Itens: []EFDItem{
			{CodItem: "1001", DescItem: "Chapa aço", UnCom: "KG", TipoItem: "01", CodNCM: "72142000", AliqICMS: 18},
		},
		DocumentosFiscais: []EFDDocumentoFiscal{{
			IndOper: "1", IndEmit: "0", CodPart: "P1", CodMod: "55", CodSit: "00",
			SerDoc: "1", NumDoc: "123", ChvNfe: strings.Repeat("1", 44),
			DtDoc: d(2026, time.January, 10), DtES: d(2026, time.January, 10),
			VlDoc: 1000, VlMerc: 1000, VlBcIcms: 1000, VlIcms: 180,
			Itens: []EFDItemDoc{
				{NumItem: 1, CodItem: "1001", Qtd: 100, UnCom: "KG", VlUnt: 10, CstIcms: "00", CfopC170: "5101", AliqIcms: 18, VlBcIcms: 1000, VlIcms: 180},
			},
			AnaliticosICMS: []EFDC190{
				{CstIcms: "00", Cfop: "5101", AliqIcms: 18, VlOpr: 1000, VlBcIcms: 1000, VlIcms: 180},
			},
		}},
		ApuracaoICMS: &EFDApuracaoICMS{
			VlTotDebitos: 180, VlTotCreditos: 0, VlIcmsRecolher: 180,
			Ajustes: []EFDApuracaoAjuste{{CodAjApur: "PR000001", DescCompl: "ajuste", VlAjApur: 0}},
		},
		Inventario: []EFDInventarioItem{
			{DtInv: d(2026, time.January, 31), CodItem: "1001", Unid: "KG", Qtd: 50, VlUnit: 9.5, VlItem: 475, IndProp: "0"},
		},
	}
}

// records returns, for each pipe-delimited line, its register code.
func records(out string) []string {
	var regs []string
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		parts := strings.Split(ln, "|")
		if len(parts) >= 2 {
			regs = append(regs, parts[1])
		}
	}
	return regs
}

func countReg(out, reg string) int {
	n := 0
	for _, r := range records(out) {
		if r == reg {
			n++
		}
	}
	return n
}

// firstLine returns the first full line whose register equals reg.
func firstLine(out, reg string) string {
	for _, ln := range strings.Split(out, "\n") {
		p := strings.Split(ln, "|")
		if len(p) >= 2 && p[1] == reg {
			return ln
		}
	}
	return ""
}

func TestGenerate_OpeningRecordAndDates(t *testing.T) {
	out := Generate(sampleParams())

	l0000 := firstLine(out, "0000")
	if l0000 == "" {
		t.Fatal("missing 0000 opening record")
	}
	f := strings.Split(l0000, "|")
	// |0000|010|0|01012026|31012026|Nome|CNPJ|...
	if f[2] != "010" {
		t.Fatalf("COD_VER = %q, want 010", f[2])
	}
	if f[4] != "01012026" || f[5] != "31012026" {
		t.Fatalf("dates wrong: DT_INI=%q DT_FIN=%q (want ddmmyyyy)", f[4], f[5])
	}
	if f[7] != "12345678000199" {
		t.Fatalf("CNPJ = %q", f[7])
	}
	// Every line must start and end with a pipe.
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if !strings.HasPrefix(ln, "|") || !strings.HasSuffix(ln, "|") {
			t.Fatalf("malformed SPED line (pipes): %q", ln)
		}
	}
}

func TestGenerate_BlocksAndDocumentRecords(t *testing.T) {
	out := Generate(sampleParams())

	// One document with one item and one analytic line.
	if countReg(out, "C100") != 1 {
		t.Errorf("C100 count = %d, want 1", countReg(out, "C100"))
	}
	if countReg(out, "C170") != 1 {
		t.Errorf("C170 count = %d, want 1", countReg(out, "C170"))
	}
	if countReg(out, "C190") != 1 {
		t.Errorf("C190 count = %d, want 1", countReg(out, "C190"))
	}
	// Apuração + ajuste, inventory, contabilista.
	if countReg(out, "E110") != 1 || countReg(out, "E111") != 1 {
		t.Errorf("E110/E111 missing")
	}
	if countReg(out, "H010") != 1 {
		t.Errorf("H010 (inventory) missing")
	}
	if countReg(out, "0100") != 1 {
		t.Errorf("0100 (contabilista) missing")
	}
	if countReg(out, "0200") != 1 || countReg(out, "0150") != 1 || countReg(out, "0190") != 2 {
		t.Errorf("0150/0190/0200 counts wrong")
	}
}

func TestGenerate_BlockClosingCounts(t *testing.T) {
	out := Generate(sampleParams())

	// C990 field = total C-block lines incl. C001 and C990 itself.
	expectedC := countReg(out, "C100") + countReg(out, "C170") + countReg(out, "C190") + 2
	c990 := strings.Split(firstLine(out, "C990"), "|")
	if c990[2] != itoa(expectedC) {
		t.Fatalf("C990 count = %s, want %d", c990[2], expectedC)
	}

	// E990 = E110 + E111 + 2 (E001 + E990).
	expectedE := countReg(out, "E110") + countReg(out, "E111") + 2
	e990 := strings.Split(firstLine(out, "E990"), "|")
	if e990[2] != itoa(expectedE) {
		t.Fatalf("E990 count = %s, want %d", e990[2], expectedE)
	}

	// H990 = H010 + 2.
	expectedH := countReg(out, "H010") + 2
	h990 := strings.Split(firstLine(out, "H990"), "|")
	if h990[2] != itoa(expectedH) {
		t.Fatalf("H990 count = %s, want %d", h990[2], expectedH)
	}
}

func TestGenerate_OmitsOptionalBlocks(t *testing.T) {
	p := sampleParams()
	p.ApuracaoICMS = nil
	p.Inventario = nil
	p.Empresa.ContabilistaNome = ""
	out := Generate(p)

	if countReg(out, "E110") != 0 {
		t.Error("no apuração → no E110")
	}
	if countReg(out, "H010") != 0 {
		t.Error("no inventory → no H010")
	}
	if countReg(out, "0100") != 0 {
		t.Error("no contabilista → no 0100")
	}
	// Block skeleton must still be present.
	for _, reg := range []string{"0000", "C001", "C990", "E001", "E990", "H001", "H990", "9999"} {
		if countReg(out, reg) == 0 {
			t.Errorf("missing mandatory record %s", reg)
		}
	}
}

func TestGenerate_9900HasOneLinePerRegister(t *testing.T) {
	out := Generate(sampleParams())
	// There must be a 9900 line for 0000 reporting count 1.
	found := false
	for _, ln := range strings.Split(out, "\n") {
		p := strings.Split(ln, "|")
		if len(p) >= 4 && p[1] == "9900" && p[2] == "0000" {
			if p[3] != "1" {
				t.Fatalf("9900 for 0000 reports %q, want 1", p[3])
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing 9900 summary line for register 0000")
	}
}

func TestFmtHelpers(t *testing.T) {
	if got := fmtVal(1234.5); got != "1234.50" {
		t.Errorf("fmtVal = %q, want 1234.50", got)
	}
	if got := fmtQtd(2.5); got != "2.5000" {
		t.Errorf("fmtQtd = %q, want 2.5000", got)
	}
	if got := fmtAliq(18); got != "18.00" {
		t.Errorf("fmtAliq = %q, want 18.00", got)
	}
	if got := fmtDate(time.Time{}); got != "" {
		t.Errorf("fmtDate(zero) = %q, want empty", got)
	}
	if got := fmtDate(time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)); got != "07022026" {
		t.Errorf("fmtDate = %q, want 07022026", got)
	}
}

// itoa avoids importing strconv just for the assertions.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
