// Package pedidocompra desenha o pedido de compra que vai ao fornecedor.
package pedidocompra

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

// Gerador implementa purchase_order_uc.GeradorPDFPedido.
type Gerador struct{}

var _ purchase_order_uc.GeradorPDFPedido = Gerador{}

type coluna struct {
	titulo  string
	largura float64 // fração da largura útil
	direita bool
}

var colunas = []coluna{
	{"Seq", 0.05, false},
	{"Código", 0.12, false},
	{"Descrição", 0.31, false},
	{"UM", 0.05, false},
	{"Qtde", 0.09, true},
	{"Preço unit.", 0.11, true},
	{"Desc.%", 0.06, true},
	{"IPI%", 0.05, true},
	{"Entrega", 0.08, false},
	{"Total", 0.08, true},
}

// PedidoCompra devolve o PDF do pedido.
func (Gerador) PedidoCompra(d purchase_order_uc.DocumentoPedido) ([]byte, error) {
	if d.Dados == nil {
		return nil, fmt.Errorf("dados do pedido ausentes")
	}
	doc := pdfkit.New()
	th := pdfkit.DefaultTheme()
	if c, ok := pdfkit.ParseHexColor(d.Dados.CorMarca); ok {
		th.Brand, th.Title = c, c
	}
	var logo *pdfkit.Image
	if len(d.Dados.Logo) > 0 {
		logo, _ = doc.AddImage(d.Dados.Logo)
	}
	w, h := doc.Size()
	m := pdfkit.Margin
	util := w - 2*m
	rodape := h - m + 4
	nota := "Gerado em " + d.Gerado.Format("02/01/2006 15:04") + " — VentureERP"
	doc.SetFooter(func(p *pdfkit.Page, num, total int) { p.Footer(th, m, rodape, w-m, nota, num, total) })

	emp := d.Dados.Empresa
	co := pdfkit.Company{Name: emp.Nome, CNPJ: emp.CNPJCPF, IE: emp.IE, Address: juntar(emp.Endereco, cidadeUF(emp)), Phone: emp.Telefone, Email: emp.Email}
	pg := doc.AddPage()
	y := pg.Letterhead(th, co, logo, m, m, util, false)

	y += 22
	pg.TextCenter(w/2, y, pdfkit.FontBold, 15, th.Title, fmt.Sprintf("PEDIDO DE COMPRA Nº %d", d.Numero))
	y += 13
	sub := "Emissão " + d.Emissao.Format("02/01/2006") + " · Situação: " + d.Situacao
	pg.TextCenter(w/2, y, pdfkit.FontRegular, 9, th.Muted, sub)
	if d.Rascunho {
		y += 13
		pg.TextCenter(w/2, y, pdfkit.FontBold, 9, pdfkit.Color{R: 180, G: 83, B: 9}, "RASCUNHO — ainda não aprovado, sem valor de compra")
	}
	y += 14

	f := d.Dados.Fornecedor
	forn := []string{nomeComFantasia(f)}
	if ids := juntarSep("    ", rotulo("CNPJ/CPF: ", f.CNPJCPF), rotulo("IE: ", f.IE)); ids != "" {
		forn = append(forn, ids)
	}
	if end := juntar(f.Endereco, cidadeUF(f)); end != "" {
		forn = append(forn, end)
	}
	if f.Email != "" {
		forn = append(forn, "E-mail: "+f.Email)
	}
	y = secao(pg, th, m, y, util, "FORNECEDOR", forn)

	cond := []string{
		juntarSep("    ", rotulo("Condição de pagamento: ", valorOu(d.Dados.Condicao, "a combinar")), rotulo("Moeda: ", d.Moeda)),
		juntarSep("    ", rotulo("Frete: ", d.Frete), rotulo("Transportadora: ", d.Dados.Transportadora)),
	}
	if d.Entrega != nil {
		cond = append(cond, "Entrega prevista: "+d.Entrega.Format("02/01/2006"))
	}
	if d.Dados.Comprador != "" {
		cond = append(cond, "Comprador: "+d.Dados.Comprador)
	}
	y = secao(pg, th, m, y, util, "CONDIÇÕES", cond)

	// Itens.
	y += 4
	larguras := make([]float64, len(colunas))
	for i, c := range colunas {
		larguras[i] = c.largura * util
	}
	cabecalho := func(p *pdfkit.Page, top float64) float64 {
		p.FillRect(m, top, util, 16, th.Brand)
		x := m
		for i, c := range colunas {
			if c.direita {
				p.TextRight(x+larguras[i]-3, top+11, pdfkit.FontBold, 7.5, th.BrandText, c.titulo)
			} else {
				p.Text(x+3, top+11, pdfkit.FontBold, 7.5, th.BrandText, c.titulo)
			}
			x += larguras[i]
		}
		return top + 16
	}
	y = cabecalho(pg, y)
	limite := rodape - 24
	for n, l := range d.Dados.Linhas {
		desc := quebrar(l.Descricao, pdfkit.FontRegular, 7.5, larguras[2]-6)
		extra := []string{}
		if l.Observacao != "" {
			extra = quebrar("Obs.: "+l.Observacao, pdfkit.FontRegular, 6.5, larguras[2]-6)
		}
		if l.Cancelada > 0 {
			extra = append(extra, fmt.Sprintf("Saldo cancelado: %s", qtd(l.Cancelada)))
		}
		altura := 6 + float64(len(desc))*9 + float64(len(extra))*8
		if y+altura > limite {
			pg = doc.AddPage()
			y = cabecalho(pg, m)
		}
		if n%2 == 1 {
			pg.FillRect(m, y, util, altura, th.Zebra)
		}
		base := y + 10
		entrega := ""
		if l.Entrega != nil {
			entrega = l.Entrega.Format("02/01/06")
		}
		valores := []string{fmt.Sprint(l.Sequence), l.ItemCode, "", l.Unidade, qtd(l.Quantidade), dinheiro(l.PrecoUnit, 4),
			pct(l.DescontoPct), pct(l.IPIPct), entrega, dinheiro(l.Total, 2)}
		x := m
		for i, c := range colunas {
			if i == 2 {
				for k, linha := range desc {
					pg.Text(x+3, base+float64(k)*9, pdfkit.FontRegular, 7.5, th.Text, linha)
				}
				for k, linha := range extra {
					pg.Text(x+3, base+float64(len(desc))*9+float64(k)*8, pdfkit.FontRegular, 6.5, th.Muted, linha)
				}
			} else if c.direita {
				pg.TextRight(x+larguras[i]-3, base, pdfkit.FontRegular, 7.5, th.Text, valores[i])
			} else {
				pg.Text(x+3, base, pdfkit.FontRegular, 7.5, th.Text, cortar(valores[i], larguras[i]-6))
			}
			x += larguras[i]
		}
		y += altura
		pg.StrokeLine(m, y, m+util, y, 0.3, th.Rule)
	}

	// Totais.
	if y+90 > limite {
		pg = doc.AddPage()
		y = m
	}
	y += 12
	xTot := m + util*0.58
	linhaTotal := func(rot, val string, negrito bool) {
		fonte := pdfkit.FontRegular
		if negrito {
			fonte = pdfkit.FontBold
		}
		pg.Text(xTot, y, fonte, 9, th.Text, rot)
		pg.TextRight(m+util, y, fonte, 9, th.Text, val)
		y += 13
	}
	linhaTotal("Mercadoria", "R$ "+dec(d.Bruto), false)
	if d.Desconto.IsPositive() {
		linhaTotal("(−) Desconto", "R$ "+dec(d.Desconto), false)
	}
	if d.IPI.IsPositive() {
		linhaTotal("(+) IPI", "R$ "+dec(d.IPI), false)
	}
	if d.FreteFOB.IsPositive() {
		linhaTotal("(+) Frete (FOB)", "R$ "+dec(d.FreteFOB), false)
	}
	pg.StrokeLine(xTot, y-9, m+util, y-9, 0.5, th.Rule)
	y += 2
	linhaTotal("Total do pedido", "R$ "+dec(d.Liquido), true)

	if len(d.Parcelas) > 0 {
		linhas := make([]string, 0, len(d.Parcelas))
		for _, p := range d.Parcelas {
			linhas = append(linhas, fmt.Sprintf("Parcela %d — vencimento %s — R$ %s%s", p.Numero, p.Vencimento.Format("02/01/2006"),
				dinheiro(p.Valor, 2), estimado(p.Estimada)))
		}
		if y+float64(len(linhas))*12+30 > limite {
			pg = doc.AddPage()
			y = m
		}
		y += 6
		y = secao(pg, th, m, y, util, "PAGAMENTO PREVISTO (pela condição e pelas datas de entrega)", linhas)
	}
	if obs := strings.TrimSpace(d.Observacao); obs != "" {
		linhas := quebrar(obs, pdfkit.FontRegular, 8.5, util-12)
		if y+float64(len(linhas))*12+30 > limite {
			pg = doc.AddPage()
			y = m
		}
		y += 6
		y = secao(pg, th, m, y, util, "OBSERVAÇÕES", linhas)
	}
	if y+40 > limite {
		pg = doc.AddPage()
		y = m
	}
	y += 14
	pg.Text(m, y, pdfkit.FontRegular, 8, th.Muted, "Pedimos confirmar o recebimento deste pedido e a data de entrega. Mencione o número do pedido na nota fiscal.")
	return doc.Render(), nil
}

func secao(p *pdfkit.Page, th pdfkit.Theme, x, top, w float64, titulo string, linhas []string) float64 {
	vis := linhas[:0:0]
	for _, l := range linhas {
		if strings.TrimSpace(l) != "" {
			vis = append(vis, l)
		}
	}
	p.FillRect(x, top, w, 14, th.Brand)
	p.Text(x+6, top+10, pdfkit.FontBold, 8.5, th.BrandText, titulo)
	altura := 14 + float64(len(vis))*12 + 4
	p.StrokeRect(x, top, w, altura, 0.5, th.Rule)
	y := top + 14
	for _, l := range vis {
		y += 12
		p.Text(x+6, y-2, pdfkit.FontRegular, 8.5, th.Text, cortar(l, w-12))
	}
	return top + altura + 6
}

func quebrar(s string, f pdfkit.Font, tam, largura float64) []string {
	palavras := strings.Fields(s)
	if len(palavras) == 0 {
		return []string{""}
	}
	var out []string
	atual := ""
	for _, p := range palavras {
		cand := strings.TrimSpace(atual + " " + p)
		if atual != "" && pdfkit.TextWidth(f, tam, cand) > largura {
			out = append(out, atual)
			atual = p
			continue
		}
		atual = cand
	}
	out = append(out, atual)
	if len(out) > 4 {
		out = append(out[:3], cortar(strings.Join(out[3:], " "), largura))
	}
	return out
}

func cortar(s string, largura float64) string {
	if pdfkit.TextWidth(pdfkit.FontRegular, 8.5, s) <= largura {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && pdfkit.TextWidth(pdfkit.FontRegular, 8.5, string(r)+"…") > largura {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func nomeComFantasia(f purchase_order_uc.Parte) string {
	if f.Fantasia != "" && !strings.EqualFold(f.Fantasia, f.Nome) {
		return f.Nome + " (" + f.Fantasia + ")"
	}
	return valorOu(f.Nome, "Fornecedor não informado")
}

func cidadeUF(p purchase_order_uc.Parte) string {
	return juntarSep(" ", juntarSep("/", p.Cidade, p.UF), rotulo("CEP ", p.CEP))
}

func juntar(partes ...string) string { return juntarSep(" — ", partes...) }

func juntarSep(sep string, partes ...string) string {
	var v []string
	for _, p := range partes {
		if strings.TrimSpace(p) != "" {
			v = append(v, strings.TrimSpace(p))
		}
	}
	return strings.Join(v, sep)
}

func rotulo(r, v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return r + v
}

func valorOu(v, padrao string) string {
	if strings.TrimSpace(v) == "" {
		return padrao
	}
	return v
}

func estimado(e bool) string {
	if e {
		return " (estimada)"
	}
	return ""
}

func dec(v decimal.Decimal) string {
	f, _ := v.Float64()
	return dinheiro(f, 2)
}

// dinheiro formata no padrão brasileiro (1.234,56).
func dinheiro(v float64, casas int) string {
	s := decimal.NewFromFloat(v).StringFixed(int32(casas))
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	inteiro, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if frac != "" {
		out += "," + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}

func qtd(v float64) string {
	s := dinheiro(v, 4)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ",")
}

func pct(v float64) string {
	if v == 0 {
		return "—"
	}
	return qtd(v)
}
