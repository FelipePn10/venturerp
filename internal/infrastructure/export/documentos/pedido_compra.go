package documentos

import (
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Pedido de compra — o documento que vai ao fornecedor: quem compra e onde
// entregar e faturar, condições, itens com prazo, totais, a previsão de
// pagamento, as instruções de fornecimento e o aceite do fornecedor.

var verdeSuprimentos = cor{R: 15, G: 107, B: 92}

func parteCompra(p purchase_order_uc.Parte) []string {
	var l []string
	if doc := juntar("   ", rotulo("CNPJ ", cnpj(p.CNPJCPF)), rotulo("IE ", p.IE)); doc != "" {
		l = append(l, doc)
	}
	if p.Endereco != "" {
		l = append(l, p.Endereco)
	}
	if c := juntar(" · ", juntar(" / ", p.Cidade, p.UF), rotulo("CEP ", cep(p.CEP))); c != "" {
		l = append(l, c)
	}
	if t := juntar(" · ", telefone(p.Telefone), p.Email); t != "" {
		l = append(l, t)
	}
	return l
}

func dec(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }

// PedidoCompra desenha o pedido de compra.
func PedidoCompra(d purchase_order_uc.DocumentoPedido) []byte {
	doc := pdfkit.New()
	const mg = 30.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Dados.Logo)
	ac := verdeSuprimentos
	claro := mistura(ac, branco, 0.9)
	emp, forn := d.Dados.Empresa, d.Dados.Fornecedor
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Pedido de compra nº %d · gerado em %s · VentureERP", d.Numero, dataHora(d.Gerado)))
	f.continua = func() float64 {
		f.p.FillRect(mg, mg, f.cw(), 20, ac)
		f.texto(mg+8, mg+13.5, neg, 8.5, branco, fit(emp.Nome, neg, 8.5, f.cw()/2))
		f.p.TextRight(f.w-mg-8, mg+13.5, neg, 8.5, branco, fmt.Sprintf("PEDIDO DE COMPRA Nº %d · continuação", d.Numero))
		return mg + 30
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho.
	x := mg
	if lw := desenharLogo(p, f.logo, x, mg, 90, 50); lw > 0 {
		x += lw + 12
	}
	lw := cw - 200 - (x - mg)
	yy := mg + 11.0
	for _, l := range quebrar(emp.Nome, neg, 10, lw) {
		f.texto(x, yy, neg, 10, tinta, l)
		yy += 11.5
	}
	for _, l := range parteCompra(emp) {
		f.texto(x, yy, reg, 7, cinza, fit(l, reg, 7, lw))
		yy += 8.6
	}
	bx := f.w - mg - 190
	p.StrokeRect(bx, mg, 190, 66, 1.2, ac)
	p.FillRect(bx, mg, 190, 18, ac)
	p.Text(bx+10, mg+12.5, neg, 7.6, branco, "PEDIDO DE COMPRA")
	p.Text(bx+10, mg+42, neg, 22, ac, fmt.Sprintf("Nº %d", d.Numero))
	p.Text(bx+10, mg+59, reg, 7, cinza, "Emissão "+data(d.Emissao)+" · "+strings.ToLower(ou(d.Situacao, "")))
	_ = p.DrawCode128B(fmt.Sprintf("PC%d", d.Numero), bx+104, mg+24, 80, 26, preto)
	f.y = maxF(yy, mg+66) + 12
	if d.Rascunho {
		p.FillRect(mg, f.y, cw, 18, mistura(verm, branco, 0.86))
		p.TextCenter(mg+cw/2, f.y+12.5, neg, 8, verm, "RASCUNHO — pedido ainda não aprovado: não enviar ao fornecedor")
		f.y += 24
	}

	// Fornecedor e entrega/faturamento.
	colW := (cw - 12) / 2
	h := 86.0
	card := func(x float64, titulo, nome string, linhas []string) {
		p.StrokeRect(x, f.y, colW, h, 0.6, linha)
		p.FillRect(x, f.y, colW, 16, claro)
		p.Text(x+8, f.y+11, neg, 7, ac, titulo)
		yy := f.y + 29
		for _, l := range quebrar(nome, neg, 9, colW-16) {
			p.Text(x+8, yy, neg, 9, tinta, l)
			yy += 10.5
		}
		for _, l := range linhas {
			if yy > f.y+h-4 {
				break
			}
			p.Text(x+8, yy, reg, 7.3, tinta, fit(l, reg, 7.3, colW-16))
			yy += 9.5
		}
	}
	fl := parteCompra(forn)
	if forn.Fantasia != "" {
		fl = append([]string{"Fantasia: " + forn.Fantasia}, fl...)
	}
	if len(fl) == 0 || forn.Endereco == "" {
		fl = append(fl, "Endereço do fornecedor não cadastrado")
	}
	card(mg, "FORNECEDOR", ou(forn.Nome, "fornecedor não informado"), fl)
	card(mg+colW+12, "FATURAR E ENTREGAR PARA", emp.Nome, parteCompra(emp))
	f.y += h + 10

	// Condições.
	cond := [][2]string{
		{"Condição de pagamento", ou(d.Dados.Condicao, "a combinar")},
		{"Frete", ou(rotuloFrete(d.Frete), "a combinar") + rotulo(" · ", valorSe(dec(d.ValorFrete)))},
		{"Transportadora", ou(d.Dados.Transportadora, "do fornecedor")},
		{"Entrega prevista", dataP(d.Entrega)},
		{"Comprador", ou(d.Dados.Comprador, "—")},
	}
	cwid := cw / float64(len(cond))
	for i, c := range cond {
		cx := mg + float64(i)*cwid
		p.FillRect(cx+1, f.y, cwid-2, 30, fundo)
		f.texto(cx+8, f.y+11, neg, 5.9, cinza, strings.ToUpper(c[0]))
		f.texto(cx+8, f.y+23, neg, 7.8, tinta, fit(c[1], neg, 7.8, cwid-16))
	}
	f.y += 40

	// Itens.
	f.texto(mg, f.y+6, neg, 8.5, ac, "ITENS")
	f.texto(mg+36, f.y+6, ita, 7, cinza, fmt.Sprintf("%d linha(s) · moeda %s", len(d.Dados.Linhas), ou(d.Moeda, "BRL")))
	f.y += 10
	cols := []col{{"#", 0.3, pdfkit.AlignCenter}, {"Código", 0.9, pdfkit.AlignLeft}, {"Descrição", 3, pdfkit.AlignLeft}, {"Un", 0.35, pdfkit.AlignCenter},
		{"Qtde", 0.85, pdfkit.AlignRight}, {"Preço unit.", 0.85, pdfkit.AlignRight}, {"Desc.", 0.5, pdfkit.AlignRight}, {"IPI", 0.45, pdfkit.AlignRight},
		{"Total", 0.95, pdfkit.AlignRight}, {"Entrega", 0.8, pdfkit.AlignCenter}}
	t := f.tabela(mg, cw, cols, estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha})
	t.cabecalho()
	if len(d.Dados.Linhas) == 0 {
		t.linha([]string{"", "", "Pedido sem itens."}, ita, cinza, nil)
	}
	for _, l := range d.Dados.Linhas {
		dw := t.ws[2] - 7
		desc := quebrar(l.Descricao, reg, 7.4, dw)
		if len(desc) > 3 {
			desc = desc[:3]
		}
		extra := []string{}
		if l.Observacao != "" {
			extra = quebrar("Obs.: "+l.Observacao, ita, 6.4, dw)
		}
		if l.Almoxarifado != "" || l.Cancelada > 0 {
			extra = append(extra, juntar(" · ", rotulo("almoxarifado ", l.Almoxarifado), rotulo("cancelado ", qtdSe(dec(l.Cancelada)))))
		}
		hh := 8 + 9*float64(len(desc)) + 8*float64(len(extra))
		hh = maxF(hh, 16)
		if f.cabe(hh) {
			t.cabecalho()
		}
		topo := t.linhaAlta([]string{fmt.Sprint(l.Sequence), l.ItemCode, "", l.Unidade, qtd(dec(l.Quantidade)), num(dec(l.PrecoUnit), 4),
			pctSe(dec(l.DescontoPct)), pctCurto(dec(l.IPIPct)), num(dec(l.Total), 2), dataP(l.Entrega)}, reg, tinta, nil, hh)
		dx := t.xCol(2) + 3.5
		y := topo + 10
		for _, s := range desc {
			f.texto(dx, y, reg, 7.4, tinta, s)
			y += 9
		}
		for _, s := range extra {
			f.texto(dx, y-1, ita, 6.4, cinza, s)
			y += 8
		}
	}
	f.y += 12

	// Previsão de pagamento e totais.
	altura := maxF(34+13*float64(max(len(d.Parcelas), 1)), 30+13*5+26)
	f.cabe(altura + 6)
	p = f.p
	topo := f.y
	pw := cw*0.52 - 8
	f.texto(mg, topo+8, neg, 8, ac, "PREVISÃO DE PAGAMENTO")
	ty := topo + 15
	p.FillRect(mg, ty, pw, 13, claro)
	p.Text(mg+6, ty+9, neg, 6.2, ac, "PARCELA")
	p.Text(mg+60, ty+9, neg, 6.2, ac, "VENCIMENTO")
	p.TextRight(mg+pw-6, ty+9, neg, 6.2, ac, "VALOR")
	ty += 13
	if len(d.Parcelas) == 0 {
		f.texto(mg+6, ty+9, ita, 7.2, cinza, "Sem condição de pagamento: a definir com o fornecedor.")
	}
	for _, pa := range d.Parcelas {
		f.texto(mg+6, ty+9, reg, 7.4, tinta, fmt.Sprintf("%dª", pa.Numero))
		s := data(pa.Vencimento)
		if pa.Estimada {
			s += " (estimada)"
		}
		f.texto(mg+60, ty+9, neg, 7.4, tinta, s)
		p.TextRight(mg+pw-6, ty+9, neg, 7.4, tinta, moeda(dec(pa.Valor)))
		p.StrokeLine(mg, ty+13, mg+pw, ty+13, 0.4, linha)
		ty += 13
	}
	tx := mg + pw + 16
	tw := f.w - mg - tx
	p.StrokeRect(tx, topo, tw, 30+13*5+20, 0.6, linha)
	f.texto(tx+10, topo+14, neg, 8, ac, "TOTAIS")
	ty = topo + 22
	for _, l := range [][2]string{{"Mercadoria", moeda(d.Bruto)}, {"(-) Desconto", moeda(d.Desconto)}, {"(+) IPI", moeda(d.IPI)}, {"(+) Frete FOB", moeda(d.FreteFOB)}} {
		f.texto(tx+10, ty+9, reg, 7.8, cinza, l[0])
		p.TextRight(tx+tw-10, ty+9, reg, 7.8, tinta, l[1])
		ty += 13
	}
	p.FillRect(tx, ty+6, tw, 26, ac)
	f.texto(tx+10, ty+23, neg, 8.5, branco, "TOTAL DO PEDIDO")
	p.TextRight(tx+tw-10, ty+24, neg, 13, branco, moeda(d.Liquido))
	f.y = topo + altura + 10

	// Instruções.
	inst := []string{
		fmt.Sprintf("Informe o número deste pedido (%d) e o código do item na nota fiscal; nota sem pedido pode ser recusada no recebimento.", d.Numero),
		"Entregue no endereço indicado acima; quantidades e preços acima da tolerância combinada podem ser devolvidos ou bloqueados.",
		"Envie o XML da NF-e para o e-mail da empresa no mesmo dia da emissão.",
		"Materiais com exigência de certificado devem vir acompanhados do certificado de qualidade/corrida.",
		"Confirme o recebimento deste pedido e a data de entrega.",
	}
	f.cabe(20 + 10*float64(len(inst)))
	f.texto(mg, f.y+6, neg, 8, ac, "INSTRUÇÕES DE FORNECIMENTO")
	f.y += 12
	for i, s := range inst {
		f.p.FillRect(mg, f.y+1, 11, 11, claro)
		f.p.TextCenter(mg+5.5, f.y+9.2, neg, 6.8, ac, fmt.Sprint(i+1))
		f.y += f.paragrafo(mg+17, f.y+9, cw-17, reg, 7.2, tinta, s, 9) + 3
	}
	f.y += 6
	if obs := strings.TrimSpace(d.Observacao); obs != "" {
		ls := quebrar(obs, reg, 7.4, cw-20)
		hh := 16 + float64(len(ls))*9.5
		f.cabe(hh + 8)
		f.p.StrokeRect(mg, f.y, cw, hh, 0.5, linha)
		f.texto(mg+10, f.y+11, neg, 6.6, ac, "OBSERVAÇÕES")
		for i, l := range ls {
			f.texto(mg+10, f.y+22+float64(i)*9.5, reg, 7.4, tinta, l)
		}
		f.y += hh + 12
	}

	// Aceite.
	f.cabe(60)
	aw := (cw - 30) / 3
	for i, r := range []string{"Comprador", "Aprovação (alçada)", "Aceite do fornecedor — nome e data"} {
		ax := mg + float64(i)*(aw+15)
		f.p.StrokeLine(ax, f.y+30, ax+aw, f.y+30, 0.6, tinta)
		f.p.TextCenter(ax+aw/2, f.y+40, reg, 6.8, cinza, r)
		if i == 0 && d.Dados.Comprador != "" {
			f.p.TextCenter(ax+aw/2, f.y+26, reg, 7.6, tinta, d.Dados.Comprador)
		}
	}
	return doc.Render()
}
