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

// camposPorRegistro: quantidade de campos (sem o REG) de cada registro no
// leiaute do Guia Prático da EFD ICMS/IPI.
var camposPorRegistro = map[string]int{
	"0000": 14, "0001": 1, "0005": 9, "0100": 13, "0150": 12, "0190": 2, "0200": 12, "0220": 3, "0990": 1,
	"B001": 1, "B990": 1, "C001": 1, "C100": 28, "C170": 37, "C190": 11, "C990": 1,
	"D001": 1, "D100": 24, "D190": 8, "D990": 1,
	"E001": 1, "E100": 2, "E110": 14, "E111": 3, "E116": 9, "E500": 3, "E510": 5, "E520": 7, "E990": 1,
	"G001": 1, "G990": 1, "H001": 1, "H005": 3, "H010": 10, "H990": 1, "K001": 1, "K990": 1,
	"1001": 1, "1010": 13, "1990": 1, "9001": 1, "9900": 2, "9990": 1, "9999": 1,
}

func linhas(out string) []string {
	return strings.Split(strings.TrimRight(out, "\r\n"), "\r\n")
}

// Validação estrutural: cada linha |REG|campos|, número de campos do leiaute,
// blocos na ordem, x990 = linhas do bloco, 9900 = contagem real de cada
// registro e 9999 = total de linhas.
func validar(t *testing.T, out string) {
	t.Helper()
	ls := linhas(out)
	contagem := map[string]int{}
	porBloco := map[string]int{}
	ordemBlocos := ""
	for _, ln := range ls {
		if !strings.HasPrefix(ln, "|") || !strings.HasSuffix(ln, "|") {
			t.Fatalf("linha mal formada: %q", ln)
		}
		p := strings.Split(ln[1:len(ln)-1], "|")
		reg := p[0]
		n, ok := camposPorRegistro[reg]
		if !ok {
			t.Fatalf("registro desconhecido %s", reg)
		}
		if len(p)-1 != n {
			t.Fatalf("registro %s com %d campos, o leiaute tem %d: %q", reg, len(p)-1, n, ln)
		}
		contagem[reg]++
		bl := reg[:1]
		porBloco[bl]++
		if !strings.HasSuffix(ordemBlocos, bl) {
			ordemBlocos += bl
		}
	}
	if ordemBlocos != "0BCDEGHK19" {
		t.Fatalf("ordem dos blocos: %s", ordemBlocos)
	}
	for _, ln := range ls {
		p := strings.Split(ln[1:len(ln)-1], "|")
		switch {
		case strings.HasSuffix(p[0], "990") && p[0] != "9990":
			if p[1] != itoa(porBloco[p[0][:1]]) {
				t.Fatalf("%s = %s, o bloco tem %d linhas", p[0], p[1], porBloco[p[0][:1]])
			}
		case p[0] == "9990":
			if p[1] != itoa(porBloco["9"]) {
				t.Fatalf("9990 = %s, o bloco 9 tem %d linhas", p[1], porBloco["9"])
			}
		case p[0] == "9900":
			if p[2] != itoa(contagem[p[1]]) {
				t.Fatalf("9900 de %s = %s, o arquivo tem %d", p[1], p[2], contagem[p[1]])
			}
		case p[0] == "9999":
			if p[1] != itoa(len(ls)) {
				t.Fatalf("9999 = %s, o arquivo tem %d linhas", p[1], len(ls))
			}
		}
	}
	if contagem["9900"] != len(contagem) {
		t.Fatalf("9900 tem %d linhas e o arquivo tem %d registros distintos", contagem["9900"], len(contagem))
	}
}

func TestGenerate_LeiauteCompleto(t *testing.T) {
	out := Generate(sampleParams())
	validar(t, out)
	f := strings.Split(firstLine(out, "0000"), "|")
	if f[2] != "020" || f[4] != "01012026" || f[5] != "31012026" || f[7] != "12345678000199" || f[14] != "A" {
		t.Fatalf("0000: %v", f)
	}
	if !strings.Contains(firstLine(out, "C100"), "|1000,00|") {
		t.Fatalf("valores com vírgula decimal: %s", firstLine(out, "C100"))
	}
	if countReg(out, "E100") != 1 || countReg(out, "H005") != 1 {
		t.Fatal("E100 antes do E110 e H005 antes do H010 são obrigatórios")
	}
}

func TestGenerate_SemMovimentoAbreBlocosComIndicador1(t *testing.T) {
	p := sampleParams()
	p.DocumentosFiscais, p.Inventario, p.ApuracaoICMS, p.Empresa.ContabilistaNome = nil, nil, nil, ""
	out := Generate(p)
	validar(t, out)
	for reg, ind := range map[string]string{"C001": "1", "D001": "1", "H001": "1", "B001": "1", "G001": "1", "K001": "1", "E001": "0"} {
		if f := strings.Split(firstLine(out, reg), "|"); f[2] != ind {
			t.Fatalf("%s = %s, esperado %s", reg, f[2], ind)
		}
	}
	if countReg(out, "E110") != 1 {
		t.Fatal("o E110 é obrigatório mesmo sem movimento")
	}
}

// Documento cancelado: só a identificação, sem C170/C190.
func TestGenerate_DocumentoCancelado(t *testing.T) {
	p := sampleParams()
	p.DocumentosFiscais[0].CodSit = "02"
	out := Generate(p)
	validar(t, out)
	f := strings.Split(firstLine(out, "C100"), "|")
	if f[6] != "02" || f[5] != "55" || f[9] == "" || f[4] != "" || f[10] != "" || f[12] != "" {
		t.Fatalf("C100 cancelado: %v", f)
	}
	if countReg(out, "C170") != 0 || countReg(out, "C190") != 0 {
		t.Fatal("documento cancelado não tem itens nem analítico")
	}
}

func TestGenerate_ConhecimentoEIPI(t *testing.T) {
	p := sampleParams()
	p.Conhecimentos = []EFDConhecimento{{IndOper: "0", IndEmit: "1", CodPart: "P1", Ser: "1", NumDoc: "456", ChvCTe: strings.Repeat("4", 44),
		DtDoc: p.Periodo.DataInicial, DtAP: p.Periodo.DataInicial, VlDoc: 1200, VlServ: 1200, VlBcIcms: 1200, VlIcms: 144,
		Analiticos: []EFDD190{{CstIcms: "000", Cfop: "1353", AliqIcms: 12, VlOpr: 1200, VlBcIcms: 1200, VlIcms: 144}}}}
	p.ApuracaoIPI = &EFDApuracaoIPI{Linhas: []EFDE510{{Cfop: "5101", CstIpi: "50", VlCont: 1000, VlBcIpi: 1000, VlIpi: 50}}, Deb: 50, Cred: 80}
	out := Generate(p)
	validar(t, out)
	if countReg(out, "D100") != 1 || countReg(out, "D190") != 1 || countReg(out, "E520") != 1 {
		t.Fatal("D100/D190/E520")
	}
	if f := strings.Split(firstLine(out, "E520"), "|"); f[7] != "30,00" || f[8] != "0,00" {
		t.Fatalf("E520 saldo credor 30: %v", f)
	}
}

func TestFmtHelpers(t *testing.T) {
	if got := fmtVal(1234.5); got != "1234,50" {
		t.Errorf("fmtVal = %q", got)
	}
	if got := fmtQtd(2.5); got != "2,50000" {
		t.Errorf("fmtQtd = %q", got)
	}
	if got := fmtDate(time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)); got != "07022026" {
		t.Errorf("fmtDate = %q", got)
	}
	if CodVer(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) != "019" || CodVer(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) != "020" {
		t.Error("versão do leiaute")
	}
}

// itoa avoids importing strconv just for the assertions.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
