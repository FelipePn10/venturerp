package documentos

import (
	"fmt"
	"strings"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Pedido de venda — documento comercial que vai ao cliente para conferência e
// aceite: cabeçalho com a marca, quadro do cliente e da entrega, condições
// comerciais, itens com a tributação de cada linha, parcelas previstas,
// totais em destaque, notas emitidas e o aceite.

var azulVenda = cor{R: 22, G: 72, B: 122}

func rotuloSituacaoVenda(s string, bloqueado bool) (string, cor) {
	if bloqueado {
		return "BLOQUEADO", verm
	}
	switch s {
	case "R":
		return "RASCUNHO", cinza
	case "A":
		return "EM ANÁLISE", ambar
	case "P":
		return "CONFIRMADO", azulVenda
	case "F":
		return "FATURADO", verde
	case "C":
		return "CANCELADO", verm
	}
	return ou(s, "—"), cinza
}

func (Gerador) PedidoVenda(d uc.PedidoVenda) []byte {
	doc := pdfkit.New()
	const mg = 30.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := azulVenda
	claro := mistura(ac, branco, 0.92)
	tot := d.Totais()
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Pedido de venda nº %d · %s · documento sem valor fiscal", d.Numero, emitidoPor(d.Cabecalho)))
	f.continua = func() float64 {
		f.p.FillRect(mg, mg, f.cw(), 20, ac)
		f.texto(mg+8, mg+13.5, neg, 8.5, branco, fit(d.Emitente.Nome, neg, 8.5, f.cw()/2))
		f.p.TextRight(f.w-mg-8, mg+13.5, neg, 8.5, branco, fmt.Sprintf("PEDIDO DE VENDA Nº %d · continuação", d.Numero))
		return mg + 30
	}

	// Cabeçalho: empresa à esquerda, identificação do pedido à direita.
	p := f.p
	top := mg
	x := mg
	if lw := desenharLogo(p, f.logo, x, top, 92, 52); lw > 0 {
		x += lw + 12
	}
	larguraEmp := f.w - mg - 190 - x
	yy := top + 11.0
	for _, l := range quebrar(d.Emitente.Nome, neg, 10.5, larguraEmp) {
		f.texto(x, yy, neg, 10.5, tinta, l)
		yy += 12
	}
	yy += 1
	for _, l := range linhasDaParte(d.Emitente.Parte) {
		f.texto(x, yy, reg, 7.2, cinza, fit(l, reg, 7.2, larguraEmp))
		yy += 9
	}
	bx := f.w - mg - 180
	p.FillRect(bx, top, 180, 62, ac)
	f.texto(bx+12, top+15, neg, 8, mistura(ac, branco, 0.65), "PEDIDO DE VENDA")
	f.texto(bx+12, top+40, neg, 24, branco, fmt.Sprintf("Nº %d", d.Numero))
	f.texto(bx+12, top+54, reg, 7.4, mistura(ac, branco, 0.75), "Emissão "+data(d.Emissao))
	sit, corSit := rotuloSituacaoVenda(d.Situacao, d.Bloqueado)
	sw := larg(neg, 7, sit) + 12
	p.FillRect(bx+180-sw-10, top+46, sw, 13, branco)
	p.Text(bx+180-sw-4, top+55.5, neg, 7, corSit, sit)
	f.y = top + 74
	p.FillRect(mg, f.y, f.cw(), 2.5, ac)
	f.y += 12

	// Cliente e entrega.
	colW := (f.cw() - 12) / 2
	h := 92.0
	quadro := func(x float64, titulo string) {
		p.FillRect(x, f.y, colW, h, claro)
		p.FillRect(x, f.y, 3, h, ac)
		f.texto(x+12, f.y+14, neg, 7, ac, titulo)
	}
	quadro(mg, "CLIENTE")
	cy := f.y + 29
	f.texto(mg+12, cy, neg, 10, tinta, fit(d.Cliente.Nome, neg, 10, colW-24))
	cy += 12
	if d.Cliente.Fantasia != "" || d.Cliente.Codigo != 0 {
		f.texto(mg+12, cy, reg, 7.3, cinza, fit(juntar(" · ", rotulo("Código ", fmt.Sprint(d.Cliente.Codigo)), d.Cliente.Fantasia), reg, 7.3, colW-24))
		cy += 10
	}
	for _, l := range linhasDaParte(d.Cliente) {
		f.texto(mg+12, cy, reg, 7.5, tinta, fit(l, reg, 7.5, colW-24))
		cy += 10
	}
	x2 := mg + colW + 12
	quadro(x2, "ENTREGA E TRANSPORTE")
	entrega := "Prevista para " + dataP(d.Entrega)
	if d.EntregaFirme {
		entrega = "Data firme: " + dataP(d.Entrega)
	}
	campos := [][2]string{
		{"Entrega", entrega},
		{"Local", ou(d.EnderecoEntrega, "o mesmo endereço do cliente")},
		{"Transportadora", ou(d.Transportadora, "a definir")},
		{"Frete", ou(rotuloFrete(d.TipoFrete), "a combinar") + rotulo(" · ", valorSe(d.Frete))},
		{"Peso total", fmt.Sprintf("%s kg líquido · %s kg bruto", num(tot.PesoLiquido, 2), num(tot.PesoBruto, 2))},
	}
	cy = f.y + 29
	for _, c := range campos {
		f.texto(x2+12, cy, neg, 6.4, cinza, strings.ToUpper(c[0]))
		f.texto(x2+80, cy, reg, 7.6, tinta, fit(c[1], reg, 7.6, colW-92))
		cy += 12.5
	}
	f.y += h + 12

	// Condições comerciais.
	cond := [][2]string{
		{"Condição de pagamento", ou(d.Condicao, "à vista")},
		{"Tabela de preço", ou(d.Tabela, "—")},
		{"Divisão de vendas", ou(d.Divisao, "—")},
		{"Representante", ou(d.Representante, "venda direta") + rotulo(" · ", pctSe(d.ComissaoPct))},
		{"Canal · moeda", juntar(" · ", rotuloCanal(d.Canal), d.Moeda)},
	}
	cwid := f.cw() / float64(len(cond))
	for i, c := range cond {
		cx := mg + float64(i)*cwid
		p.StrokeRect(cx, f.y, cwid, 30, 0.5, linha)
		f.texto(cx+7, f.y+10, neg, 5.9, cinza, strings.ToUpper(c[0]))
		f.texto(cx+7, f.y+23, neg, 8, tinta, fit(c[1], neg, 8, cwid-14))
	}
	f.y += 48

	// Itens.
	f.texto(mg, f.y, neg, 9, ac, "ITENS DO PEDIDO")
	f.texto(mg+90, f.y, ita, 7, cinza, fmt.Sprintf("%d item(ns) · %s unidade(s)", len(d.Itens), qtd(tot.Quantidade)))
	f.y += 6
	cols := []col{
		{"#", 0.35, pdfkit.AlignCenter}, {"Código", 1.1, pdfkit.AlignLeft}, {"Descrição", 3.3, pdfkit.AlignLeft},
		{"Qtde", 0.75, pdfkit.AlignRight}, {"Un", 0.4, pdfkit.AlignCenter}, {"Preço unit.", 1, pdfkit.AlignRight},
		{"IPI", 0.6, pdfkit.AlignRight}, {"ICMS", 0.6, pdfkit.AlignRight},
		{"Total", 1.1, pdfkit.AlignRight}, {"Entrega", 0.95, pdfkit.AlignCenter},
	}
	t := f.tabela(mg, f.cw(), cols, estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha, altLinha: 27})
	t.cabecalho()
	for _, it := range d.Itens {
		dw := t.ws[2] - 7
		desc := quebrar(it.Descricao, neg, 7.6, dw)
		if len(desc) > 2 {
			desc = desc[:2]
			desc[1] = fit(desc[1]+" …", neg, 7.6, dw)
		}
		h := 18 + 9*float64(len(desc))
		if f.cabe(h) {
			t.cabecalho()
		}
		topo := f.y
		vals := []string{fmt.Sprint(it.Seq), it.Codigo, "", qtd(it.Quantidade), it.UM, num(it.Preco, 2), pctCurto(it.IPIPct), pctCurto(it.ICMSPct), num(it.Total, 2), dataP(it.Entrega)}
		t.linhaAlta(vals, reg, tinta, nil, h)
		dx := t.xCol(2) + 3.5
		for i, l := range desc {
			f.texto(dx, topo+11+9*float64(i), neg, 7.6, tinta, l)
		}
		info := juntar(" · ", rotulo("NCM ", it.NCM), rotulo("dep. ", it.Deposito), pesoSe(it.PesoLiquido), rotulo("desc. ", pctSe(it.DescontoPct)),
			rotulo("ST ", valorSe(it.ST)))
		f.texto(dx, topo+h-6, reg, 6.3, cinza, fit(info, reg, 6.3, dw))
		if it.Atendida.IsPositive() || it.Cancelada.IsPositive() {
			s := juntar(" · ", rotulo("atendido ", qtdSe(it.Atendida)), rotulo("cancelado ", qtdSe(it.Cancelada)))
			f.p.TextRight(t.xCol(4)-3.5, topo+h-6, ita, 6.2, verde, s)
		}
	}
	f.y += 14

	// Pagamento e totais lado a lado.
	altPag := 30 + 13*float64(max(len(d.Parcelas), 1))
	altTot := 30 + 13*6 + 26.0
	f.cabe(maxF(altPag, altTot) + 10)
	topo := f.y
	pw := f.cw()*0.55 - 8
	f.texto(mg, topo+8, neg, 8, ac, "PAGAMENTO PREVISTO")
	f.texto(mg+100, topo+8, ita, 6.6, cinza, ou(d.Condicao, "à vista"))
	ty := topo + 16
	p = f.p
	p.FillRect(mg, ty, pw, 13, claro)
	for i, h := range []string{"Parcela", "Prazo", "Vencimento", "%", "Valor"} {
		xs := []float64{6, 50, 100, 170, pw - 6}
		if i == 4 {
			p.TextRight(mg+xs[i], ty+9, neg, 6.3, ac, strings.ToUpper(h))
		} else {
			p.Text(mg+xs[i], ty+9, neg, 6.3, ac, strings.ToUpper(h))
		}
	}
	ty += 13
	if len(d.Parcelas) == 0 {
		f.texto(mg+6, ty+9, ita, 7.3, cinza, "Pagamento à vista, na emissão da nota.")
		ty += 13
	}
	for _, par := range d.Parcelas {
		f.texto(mg+6, ty+9, reg, 7.5, tinta, fmt.Sprintf("%dª", par.Numero))
		f.texto(mg+50, ty+9, reg, 7.5, tinta, fmt.Sprintf("%d dias", par.Dias))
		f.texto(mg+100, ty+9, neg, 7.5, tinta, data(par.Vencimento))
		f.texto(mg+170, ty+9, reg, 7.5, cinza, pct(par.Percentual))
		p.TextRight(mg+pw-6, ty+9, neg, 7.5, tinta, moeda(par.Valor))
		p.StrokeLine(mg, ty+13, mg+pw, ty+13, 0.4, linha)
		ty += 13
	}
	f.texto(mg, ty+11, ita, 6.4, cinza, "Vencimentos contados da emissão; a nota fiscal confirma as datas e os valores.")

	tx := mg + pw + 16
	tw := f.w - mg - tx
	linhas := [][2]string{
		{"Produtos", moeda(tot.Produtos)}, {"Desconto", "- " + moeda(d.Desconto)}, {"Frete", moeda(d.Frete)},
		{"Seguro", moeda(d.Seguro)}, {"IPI", moeda(tot.IPI)}, {"ICMS-ST", moeda(tot.ST)},
	}
	ty = topo
	p.StrokeRect(tx, ty, tw, altTot-4, 0.6, linha)
	f.texto(tx+10, ty+14, neg, 8, ac, "TOTAIS")
	ty += 22
	for _, l := range linhas {
		f.texto(tx+10, ty+8, reg, 7.8, cinza, l[0])
		p.TextRight(tx+tw-10, ty+8, reg, 7.8, tinta, l[1])
		ty += 13
	}
	p.FillRect(tx, ty+2, tw, 26, ac)
	f.texto(tx+10, ty+19, neg, 8.5, branco, "TOTAL DO PEDIDO")
	p.TextRight(tx+tw-10, ty+20, neg, 13, branco, moeda(tot.Total))
	f.y = topo + maxF(altPag+14, altTot) + 8

	// Tributos informativos.
	f.cabe(40)
	trib := [][2]string{
		{"Base de cálculo ICMS", moeda(tot.Produtos)}, {"ICMS destacado", moeda(tot.ICMS)},
		{"IPI", moeda(tot.IPI)}, {"PIS (estimado)", moeda(estimar(d, true))}, {"COFINS (estimado)", moeda(estimar(d, false))},
	}
	tw2 := f.cw() / float64(len(trib))
	p = f.p
	p.FillRect(mg, f.y, f.cw(), 30, fundo)
	for i, c := range trib {
		cx := mg + float64(i)*tw2
		f.texto(cx+8, f.y+11, neg, 5.9, cinza, strings.ToUpper(c[0]))
		f.texto(cx+8, f.y+23, neg, 8, tinta, c[1])
	}
	f.y += 40

	if len(d.Notas) > 0 {
		f.cabe(30 + 15*float64(len(d.Notas)))
		f.texto(mg, f.y+6, neg, 8, ac, "FATURAMENTO")
		f.y += 10
		tn := f.tabela(mg, f.cw(), []col{{"NF-e", 0.8, pdfkit.AlignLeft}, {"Série", 0.4, pdfkit.AlignCenter}, {"Emissão", 0.8, pdfkit.AlignCenter},
			{"Situação", 0.9, pdfkit.AlignLeft}, {"Chave de acesso", 4, pdfkit.AlignLeft}, {"Valor", 1, pdfkit.AlignRight}},
			estiloTabela{cabFundo: claro, cabTexto: ac, grade: linha})
		tn.cabecalho()
		for _, n := range d.Notas {
			tn.linha([]string{fmt.Sprint(n.Numero), n.Serie, data(n.Emissao), rotuloNota(n.Situacao), chaveNFe(n.Chave), moeda(n.Valor)}, reg, tinta, nil)
		}
		f.y += 10
	}

	obs := strings.TrimSpace(juntar("\n", d.Observacao, rotulo("Ao cliente: ", d.ObservacaoCliente)))
	if obs != "" {
		ls := quebrar(obs, reg, 7.5, f.cw()-20)
		hh := 18 + float64(len(ls))*10
		f.cabe(hh + 8)
		f.p.StrokeRect(mg, f.y, f.cw(), hh, 0.5, linha)
		f.texto(mg+10, f.y+12, neg, 6.8, ac, "OBSERVAÇÕES")
		for i, l := range ls {
			f.texto(mg+10, f.y+23+float64(i)*10, reg, 7.5, tinta, l)
		}
		f.y += hh + 12
	}
	if d.Bloqueado && d.MotivoBloqueio != "" {
		f.cabe(26)
		f.p.FillRect(mg, f.y, f.cw(), 20, mistura(verm, branco, 0.88))
		f.texto(mg+10, f.y+13, neg, 7.5, verm, fit("Pedido bloqueado: "+d.MotivoBloqueio, neg, 7.5, f.cw()-20))
		f.y += 28
	}

	// Aceite.
	f.cabe(78)
	p = f.p
	f.texto(mg, f.y+6, neg, 8, ac, "ACEITE")
	f.paragrafo(mg+50, f.y+6, f.cw()-50, reg, 7, cinza,
		"Conferimos itens, quantidades, preços, prazos e condições deste pedido e estamos de acordo com o fornecimento.", 9)
	f.y += 40
	aw := (f.cw() - 30) / 3
	for i, r := range []string{"Cliente — de acordo", "Vendedor / representante", "Data"} {
		ax := mg + float64(i)*(aw+15)
		p.StrokeLine(ax, f.y+14, ax+aw, f.y+14, 0.6, tinta)
		p.TextCenter(ax+aw/2, f.y+24, reg, 6.8, cinza, r)
	}
	if d.Plano != "" {
		f.texto(mg, f.y+40, ita, 6.4, cinza, "Plano de produção: "+d.Plano)
	}
	return doc.Render()
}

func rotuloFrete(s string) string {
	switch strings.ToUpper(s) {
	case "CIF", "0":
		return "CIF — por conta do emitente"
	case "FOB", "1":
		return "FOB — por conta do cliente"
	case "SEM_FRETE", "9":
		return "Sem frete"
	}
	return s
}

func rotuloCanal(s string) string {
	switch strings.ToUpper(s) {
	case "DIRETO":
		return "Venda direta"
	case "REPRESENTANTE":
		return "Representante"
	}
	return s
}

func rotuloNota(s string) string {
	switch s {
	case "AUTHORIZED":
		return "Autorizada"
	case "DRAFT":
		return "Rascunho"
	case "CANCELLED":
		return "Cancelada"
	case "REJECTED":
		return "Rejeitada"
	}
	return s
}

func valorSe(v decimal.Decimal) string {
	if !v.IsPositive() {
		return ""
	}
	return moeda(v)
}

// pctCurto mostra 12% e 3,25% (sem zeros à direita).
func pctCurto(v decimal.Decimal) string {
	s := strings.TrimRight(strings.TrimRight(v.StringFixed(2), "0"), ".")
	return milhar(s) + "%"
}

func pctSe(v decimal.Decimal) string {
	if !v.IsPositive() {
		return ""
	}
	return pct(v)
}

func qtdSe(v decimal.Decimal) string {
	if !v.IsPositive() {
		return ""
	}
	return qtd(v)
}

func pesoSe(v decimal.Decimal) string {
	if !v.IsPositive() {
		return ""
	}
	return num(v, 2) + " kg/un"
}

// estimar calcula PIS ou COFINS pelas alíquotas das linhas (informativo: a
// nota fiscal é quem apura).
func estimar(d uc.PedidoVenda, pis bool) decimal.Decimal {
	tot := decimal.Zero
	for _, it := range d.Itens {
		a := it.COFINSPct
		if pis {
			a = it.PISPct
		}
		tot = tot.Add(it.Total.Mul(a).Div(decimal.NewFromInt(100)))
	}
	return tot.Round(2)
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
