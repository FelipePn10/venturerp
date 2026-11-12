// Package documentos desenha os documentos impressos do ERP. Cada documento
// tem o layout próprio dele — o pedido de venda é uma proposta para o cliente,
// o espelho da nota segue o quadro fiscal, o plano de materiais é uma árvore,
// a capacidade é um mapa de calor, a programação é um Gantt, o romaneio tem
// canhoto. Este arquivo só reúne o ferramental técnico comum (texto, quebra de
// linha, tabela paginada, formatos); o visual mora em cada documento.
package documentos

import (
	"fmt"
	"strings"
	"time"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Gerador implementa impressao_uc.Gerador.
type Gerador struct{}

var _ uc.Gerador = Gerador{}

type (
	cor   = pdfkit.Color
	fonte = pdfkit.Font
)

const (
	reg = pdfkit.FontRegular
	neg = pdfkit.FontBold
	ita = pdfkit.FontOblique
)

var (
	branco = pdfkit.White
	preto  = pdfkit.Black
	tinta  = cor{R: 30, G: 34, B: 40}
	cinza  = cor{R: 102, G: 110, B: 120}
	linha  = cor{R: 208, G: 213, B: 220}
	fundo  = cor{R: 245, G: 246, B: 248}
	verde  = cor{R: 26, G: 127, B: 55}
	verm   = cor{R: 180, G: 35, B: 24}
	ambar  = cor{R: 176, G: 110, B: 0}
)

// mistura aproxima c de b na fração f (0 = c, 1 = b).
func mistura(c, b cor, f float64) cor {
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*f) }
	return cor{R: m(c.R, b.R), G: m(c.G, b.G), B: m(c.B, b.B)}
}

// marca devolve a cor da empresa ou a padrão do documento.
func marca(hex string, padrao cor) cor {
	if c, ok := pdfkit.ParseHexColor(hex); ok {
		return c
	}
	return padrao
}

// ─── texto ──────────────────────────────────────────────────────────────────

func larg(f fonte, t float64, s string) float64 { return pdfkit.TextWidth(f, t, s) }

// fit corta com reticências para caber em w.
func fit(s string, f fonte, t, w float64) string {
	s = strings.TrimSpace(s)
	if w <= 0 || larg(f, t, s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && larg(f, t, string(r)+"…") > w {
		r = r[:len(r)-1]
	}
	return strings.TrimSpace(string(r)) + "…"
}

// quebrar divide o texto em linhas que cabem em w (por palavra), respeitando
// as quebras de linha do próprio texto.
func quebrar(s string, f fonte, t, w float64) []string {
	var out []string
	for _, par := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		palavras := strings.Fields(par)
		if len(palavras) == 0 {
			continue
		}
		atual := ""
		for _, p := range palavras {
			teste := strings.TrimSpace(atual + " " + p)
			if atual != "" && larg(f, t, teste) > w {
				out = append(out, atual)
				atual = p
				continue
			}
			atual = teste
		}
		for larg(f, t, atual) > w && len([]rune(atual)) > 1 { // palavra maior que a linha
			r := []rune(atual)
			n := len(r)
			for n > 1 && larg(f, t, string(r[:n])) > w {
				n--
			}
			out = append(out, string(r[:n]))
			atual = string(r[n:])
		}
		out = append(out, atual)
	}
	return out
}

// ─── formatos ───────────────────────────────────────────────────────────────

func milhar(s string) string {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	inteiro, dec, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if dec != "" {
		out += "," + dec
	}
	if neg {
		out = "-" + out
	}
	return out
}

func num(v decimal.Decimal, casas int32) string { return milhar(v.StringFixed(casas)) }
func moeda(v decimal.Decimal) string            { return "R$ " + num(v, 2) }

// qtd mostra até 4 casas, sem zeros à direita.
func qtd(v decimal.Decimal) string {
	s := v.Round(4).StringFixed(4)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" || s == "-0" {
		s = "0"
	}
	return milhar(s)
}

func pct(v decimal.Decimal) string { return num(v, 2) + "%" }

func data(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("02/01/2006")
}

func dataP(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return data(*t)
}

func dataHora(t time.Time) string { return t.Format("02/01/2006 15:04") }

func ou(s, padrao string) string {
	if strings.TrimSpace(s) == "" {
		return padrao
	}
	return strings.TrimSpace(s)
}

func juntar(sep string, partes ...string) string {
	var out []string
	for _, p := range partes {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return strings.Join(out, sep)
}

func cnpj(s string) string {
	d := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	switch len(d) {
	case 14:
		return d[0:2] + "." + d[2:5] + "." + d[5:8] + "/" + d[8:12] + "-" + d[12:]
	case 11:
		return d[0:3] + "." + d[3:6] + "." + d[6:9] + "-" + d[9:]
	}
	return strings.TrimSpace(s)
}

func cep(s string) string {
	if len(s) == 8 {
		return s[:5] + "-" + s[5:]
	}
	return s
}

func telefone(s string) string {
	d := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	switch len(d) {
	case 10:
		return "(" + d[:2] + ") " + d[2:6] + "-" + d[6:]
	case 11:
		return "(" + d[:2] + ") " + d[2:7] + "-" + d[7:]
	}
	return s
}

func cidadeUF(p uc.Parte) string {
	return juntar(" / ", p.Cidade, p.UF)
}

// linhasDaParte devolve as linhas de identificação de uma empresa.
func linhasDaParte(p uc.Parte) []string {
	var l []string
	if doc := juntar("   ", rotulo("CNPJ ", cnpj(p.CNPJCPF)), rotulo("IE ", p.IE)); doc != "" {
		l = append(l, doc)
	}
	if e := juntar(" · ", p.Endereco, p.Bairro); e != "" {
		l = append(l, e)
	}
	if c := juntar(" · ", cidadeUF(p), rotulo("CEP ", cep(p.CEP))); c != "" {
		l = append(l, c)
	}
	if t := juntar(" · ", telefone(p.Telefone), p.Email); t != "" {
		l = append(l, t)
	}
	return l
}

func rotulo(prefixo, v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return prefixo + v
}

// chaveNFe agrupa a chave de 44 dígitos de 4 em 4.
func chaveNFe(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ─── página ─────────────────────────────────────────────────────────────────

// folha controla a página corrente e a quebra: cada documento passa a função
// que desenha o cabeçalho de continuação.
type folha struct {
	doc      *pdfkit.Doc
	p        *pdfkit.Page
	w, h, mg float64
	y        float64
	base     float64 // limite inferior útil
	continua func() float64
	logo     *pdfkit.Image
}

func novaFolha(doc *pdfkit.Doc, mg, rodape float64) *folha {
	w, h := doc.Size()
	f := &folha{doc: doc, w: w, h: h, mg: mg, base: h - mg - rodape}
	f.p = doc.AddPage()
	f.y = mg
	return f
}

func (f *folha) cw() float64 { return f.w - 2*f.mg }

// cabe garante altura; abre página nova quando falta e devolve true.
func (f *folha) cabe(h float64) bool {
	if f.y+h <= f.base {
		return false
	}
	f.p = f.doc.AddPage()
	f.y = f.mg
	if f.continua != nil {
		f.y = f.continua()
	}
	return true
}

func (f *folha) texto(x, y float64, ft fonte, t float64, c cor, s string) {
	f.p.Text(x, y, ft, t, c, s)
}

// paragrafo escreve texto quebrado em w e devolve a altura usada.
func (f *folha) paragrafo(x, y, w float64, ft fonte, t float64, c cor, s string, entre float64) float64 {
	ls := quebrar(s, ft, t, w)
	for i, l := range ls {
		f.p.Text(x, y+float64(i)*entre, ft, t, c, l)
	}
	return float64(len(ls)) * entre
}

// ─── tabela ─────────────────────────────────────────────────────────────────

type col struct {
	titulo string
	peso   float64
	al     pdfkit.Align
}

type estiloTabela struct {
	cabFundo, cabTexto cor
	zebra              *cor
	grade              cor
	vertical           bool // linhas verticais entre colunas
	altCab, altLinha   float64
	tamCab, tam        float64
}

type tabela struct {
	f    *folha
	cols []col
	ws   []float64
	x, w float64
	e    estiloTabela
	n    int
}

func (f *folha) tabela(x, w float64, cols []col, e estiloTabela) *tabela {
	if e.altCab == 0 {
		e.altCab = 15
	}
	if e.altLinha == 0 {
		e.altLinha = 15
	}
	if e.tamCab == 0 {
		e.tamCab = 6.5
	}
	if e.tam == 0 {
		e.tam = 7.6
	}
	t := &tabela{f: f, cols: cols, x: x, w: w, e: e}
	soma := 0.0
	for _, c := range cols {
		soma += c.peso
	}
	for _, c := range cols {
		t.ws = append(t.ws, w*c.peso/soma)
	}
	return t
}

func celula(p *pdfkit.Page, s string, ft fonte, t float64, c cor, al pdfkit.Align, x, base, w float64) {
	s = fit(s, ft, t, w-7)
	switch al {
	case pdfkit.AlignRight:
		p.TextRight(x+w-3.5, base, ft, t, c, s)
	case pdfkit.AlignCenter:
		p.TextCenter(x+w/2, base, ft, t, c, s)
	default:
		p.Text(x+3.5, base, ft, t, c, s)
	}
}

func (t *tabela) cabecalho() {
	f := t.f
	f.cabe(t.e.altCab + t.e.altLinha)
	f.p.FillRect(t.x, f.y, t.w, t.e.altCab, t.e.cabFundo)
	x := t.x
	for i, c := range t.cols {
		titulo := strings.ToUpper(c.titulo)
		tam := t.e.tamCab
		for tam > 4.6 && larg(neg, tam, titulo) > t.ws[i]-8 {
			tam -= 0.2
		}
		celula(f.p, titulo, neg, tam, t.e.cabTexto, c.al, x, f.y+t.e.altCab/2+2.3, t.ws[i])
		x += t.ws[i]
	}
	f.y += t.e.altCab
	t.n = 0
}

// linha desenha uma linha; quebra a página repetindo o cabeçalho.
// linha devolve o topo em que a linha foi desenhada (já na página nova,
// quando quebrou) — é onde os desenhos sobrepostos à linha devem ir.
func (t *tabela) linha(vals []string, ft fonte, c cor, destaque *cor) float64 {
	return t.linhaAlta(vals, ft, c, destaque, t.e.altLinha)
}

func (t *tabela) linhaAlta(vals []string, ft fonte, c cor, destaque *cor, h float64) float64 {
	f := t.f
	if f.cabe(h) {
		t.cabecalho()
	}
	topo := f.y
	switch {
	case destaque != nil:
		f.p.FillRect(t.x, f.y, t.w, h, *destaque)
	case t.e.zebra != nil && t.n%2 == 1:
		f.p.FillRect(t.x, f.y, t.w, h, *t.e.zebra)
	}
	x := t.x
	for i, cc := range t.cols {
		if i < len(vals) {
			celula(f.p, vals[i], ft, t.e.tam, c, cc.al, x, f.y+h/2+2.6, t.ws[i])
		}
		if t.e.vertical && i > 0 {
			f.p.StrokeLine(x, f.y, x, f.y+h, 0.4, t.e.grade)
		}
		x += t.ws[i]
	}
	f.p.StrokeLine(t.x, f.y+h, t.x+t.w, f.y+h, 0.4, t.e.grade)
	f.y += h
	t.n++
	return topo
}

// xCol devolve o x inicial da coluna i.
func (t *tabela) xCol(i int) float64 {
	x := t.x
	for j := 0; j < i; j++ {
		x += t.ws[j]
	}
	return x
}

// ─── peças visuais pequenas ─────────────────────────────────────────────────

// selo desenha um rótulo em pílula e devolve a largura.
func selo(p *pdfkit.Page, x, top float64, s string, bg, fg cor, t float64) float64 {
	w := larg(neg, t, s) + 12
	p.FillRect(x, top, w, t+7, bg)
	p.Text(x+6, top+t+2.2, neg, t, fg, s)
	return w
}

// marcaOK desenha um visto (✓) com dois traços.
func marcaOK(p *pdfkit.Page, x, top float64, c cor) {
	p.StrokeLine(x, top+4, x+2.6, top+7, 1.3, c)
	p.StrokeLine(x+2.6, top+7, x+7.5, top+0.5, 1.3, c)
}

// caixa de seleção vazia.
func caixa(p *pdfkit.Page, x, top float64, c cor) { p.StrokeRect(x, top, 8, 8, 0.7, c) }

// rodapeSimples: linha, nota à esquerda e paginação à direita.
func rodapeSimples(doc *pdfkit.Doc, mg float64, c cor, nota string) {
	w, h := doc.Size()
	doc.SetFooter(func(p *pdfkit.Page, n, total int) {
		y := h - mg + 6
		p.StrokeLine(mg, y-9, w-mg, y-9, 0.5, linha)
		p.Text(mg, y, reg, 6.6, cinza, fit(nota, reg, 6.6, w-2*mg-70))
		p.TextRight(w-mg, y, neg, 6.8, c, fmt.Sprintf("Página %d de %d", n, total))
	})
}

func emitidoPor(c uc.Cabecalho) string {
	s := "Emitido em " + dataHora(c.Gerado)
	if strings.TrimSpace(c.EmitidoPor) != "" {
		s += " por " + c.EmitidoPor
	}
	return s
}

// logo adiciona a logo da empresa ao documento (nil quando não há).
func logo(doc *pdfkit.Doc, b []byte) *pdfkit.Image {
	if len(b) == 0 {
		return nil
	}
	img, err := doc.AddImage(b)
	if err != nil {
		return nil
	}
	return img
}

func desenharLogo(p *pdfkit.Page, img *pdfkit.Image, x, top, maxW, maxH float64) float64 {
	if img == nil {
		return 0
	}
	w, h := img.Fit(maxW, maxH)
	p.DrawImage(img, x, top+(maxH-h)/2, w, h)
	return w
}

var zero = decimal.Zero
