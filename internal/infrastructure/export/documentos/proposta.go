package documentos

import (
	"fmt"
	"strings"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

// Proposta comercial — o orçamento apresentado ao cliente: carta de
// apresentação, itens com prazo, investimento em destaque, condições
// comerciais em lista (pagamento, frete, entrega, validade), e o aceite com o
// número do pedido do cliente.

var vinho = cor{R: 122, G: 31, B: 61}

func (Gerador) Proposta(d uc.Proposta) []byte {
	doc := pdfkit.New()
	const mg = 34.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := vinho
	claro := mistura(ac, branco, 0.92)
	tot := d.Totais()
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Proposta comercial nº %d · %s", d.Numero, emitidoPor(d.Cabecalho)))
	f.continua = func() float64 {
		f.texto(mg, mg+10, neg, 8.5, ac, fmt.Sprintf("PROPOSTA COMERCIAL Nº %d (continuação)", d.Numero))
		f.p.StrokeLine(mg, mg+15, f.w-mg, mg+15, 0.8, ac)
		return mg + 26
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho: logo e empresa; à direita o título com a validade em destaque.
	x := mg
	if lw := desenharLogo(p, f.logo, x, mg, 96, 52); lw > 0 {
		x += lw + 14
	}
	lw := cw - 200 - (x - mg)
	yy := mg + 11.0
	for _, l := range quebrar(d.Emitente.Nome, neg, 9.5, lw) {
		f.texto(x, yy, neg, 9.5, tinta, l)
		yy += 11
	}
	for _, l := range linhasDaParte(d.Emitente.Parte) {
		f.texto(x, yy, reg, 6.9, cinza, fit(l, reg, 6.9, lw))
		yy += 8.5
	}
	rx := f.w - mg - 186
	f.texto(rx, mg+12, neg, 8, ac, "PROPOSTA COMERCIAL")
	f.texto(rx, mg+36, neg, 24, tinta, fmt.Sprintf("Nº %d", d.Numero))
	f.texto(rx, mg+49, reg, 7.4, cinza, "Emitida em "+data(d.Emissao))
	p.FillRect(rx, mg+55, 186, 18, ac)
	val := "Válida até " + dataP(d.Validade)
	if d.Validade != nil {
		if n := int(dia(*d.Validade).Sub(dia(d.Gerado)).Hours() / 24); n >= 0 {
			val += fmt.Sprintf(" · %d dia(s)", n)
		} else {
			val += " · vencida"
		}
	}
	p.Text(rx+8, mg+67.5, neg, 8, branco, val)
	f.y = maxF(yy, mg+73) + 16
	p.StrokeLine(mg, f.y, f.w-mg, f.y, 0.6, linha)
	f.y += 16

	// Carta de apresentação.
	f.texto(mg, f.y, neg, 7, ac, "PARA")
	f.texto(mg+40, f.y, neg, 10, tinta, fit(d.Cliente.Nome, neg, 10, cw-40))
	f.y += 11
	if l := juntar(" · ", rotulo("CNPJ ", cnpj(d.Cliente.CNPJCPF)), cidadeUF(d.Cliente), rotulo("A/C pedido ", d.PedidoCliente)); l != "" {
		f.texto(mg+40, f.y, reg, 7.4, cinza, fit(l, reg, 7.4, cw-40))
		f.y += 10
	}
	f.y += 8
	f.y += f.paragrafo(mg, f.y, cw, reg, 8.4, tinta,
		fmt.Sprintf("Agradecemos a consulta. Apresentamos a seguir nossa proposta para o fornecimento dos itens solicitados, nas condições comerciais descritas abaixo, válida até %s.", dataP(d.Validade)), 11.5)
	f.y += 10
	if d.Convertida != 0 {
		p.FillRect(mg, f.y, cw, 18, mistura(verde, branco, 0.86))
		marcaOK(p, mg+8, f.y+5, verde)
		f.texto(mg+22, f.y+12.5, neg, 7.8, verde, fmt.Sprintf("Proposta aprovada: convertida no pedido de venda nº %d.", d.Convertida))
		f.y += 26
	}

	// Itens.
	cols := []col{{"#", 0.3, pdfkit.AlignCenter}, {"Código", 1, pdfkit.AlignLeft}, {"Descrição", 3.2, pdfkit.AlignLeft}, {"Qtde", 0.7, pdfkit.AlignRight},
		{"Un", 0.35, pdfkit.AlignCenter}, {"Preço unit.", 0.95, pdfkit.AlignRight}, {"IPI", 0.5, pdfkit.AlignRight}, {"Total", 1, pdfkit.AlignRight}, {"Entrega", 0.85, pdfkit.AlignCenter}}
	t := f.tabela(mg, cw, cols, estiloTabela{cabFundo: branco, cabTexto: ac, grade: linha})
	t.cabecalho()
	f.p.StrokeLine(mg, f.y, mg+cw, f.y, 1.2, ac)
	if len(d.Itens) == 0 {
		t.linha([]string{"", "", "Orçamento sem itens."}, ita, cinza, nil)
	}
	for _, it := range d.Itens {
		dw := t.ws[2] - 7
		desc := quebrar(it.Descricao, neg, 7.6, dw)
		if len(desc) > 2 {
			desc = desc[:2]
		}
		info := juntar(" · ", rotulo("NCM ", it.NCM), rotulo("desc. ", pctSe(it.DescontoPct)), pesoSe(it.PesoLiquido), it.Situacao)
		h := 10 + 9*float64(len(desc))
		if info != "" {
			h += 8
		}
		if f.cabe(h) {
			t.cabecalho()
		}
		topo := t.linhaAlta([]string{fmt.Sprint(it.Seq), it.Codigo, "", qtd(it.Quantidade), it.UM, num(it.Preco, 2), pctCurto(it.IPIPct), num(it.Total, 2), dataP(it.Entrega)}, reg, tinta, nil, h)
		y := topo + 11
		for _, l := range desc {
			f.texto(t.xCol(2)+3.5, y, neg, 7.6, tinta, l)
			y += 9
		}
		if info != "" {
			f.texto(t.xCol(2)+3.5, y-1, reg, 6.2, cinza, fit(info, reg, 6.2, dw))
		}
	}
	f.y += 16

	// Condições à esquerda; investimento à direita.
	f.cabe(150)
	p = f.p
	topo := f.y
	lw2 := cw*0.56 - 10
	f.texto(mg, topo+8, neg, 8.5, ac, "CONDIÇÕES COMERCIAIS")
	ly := topo + 22
	pag := ou(d.Condicao, "à vista")
	if len(d.Parcelas) > 0 {
		var ps []string
		for _, pa := range d.Parcelas {
			ps = append(ps, fmt.Sprintf("%dª em %d dias (%s)", pa.Numero, pa.Dias, moeda(pa.Valor)))
		}
		pag += ": " + strings.Join(ps, "; ")
	}
	entrega := "conforme cada item"
	if d.Entrega != nil {
		entrega = "até " + data(*d.Entrega)
	}
	conds := [][2]string{
		{"Pagamento", pag},
		{"Frete", ou(rotuloFrete(d.TipoFrete), "a combinar") + rotulo(" · ", valorSe(d.Frete))},
		{"Prazo de entrega", entrega},
		{"Validade", "esta proposta vale até " + dataP(d.Validade) + "; depois disso preços e prazos devem ser confirmados"},
		{"Impostos", "IPI destacado por item; ICMS conforme a operação e a UF de destino"},
		{"Tabela de preço", ou(d.Tabela, "—")},
		{"Atendimento", ou(d.Representante, "venda direta")},
	}
	for _, c := range conds {
		p.FillRect(mg, ly-6.5, 4, 4, ac)
		f.texto(mg+10, ly, neg, 7.4, tinta, c[0])
		ly += f.paragrafo(mg+90, ly, lw2-90, reg, 7.4, tinta, c[1], 9.5) + 4
	}
	tx := mg + lw2 + 20
	tw := f.w - mg - tx
	p.FillRect(tx, topo, tw, 132, claro)
	f.texto(tx+12, topo+16, neg, 8, ac, "INVESTIMENTO")
	ty := topo + 30
	for _, l := range [][2]string{{"Produtos", moeda(tot.Produtos)}, {"IPI", moeda(tot.IPI)}, {"ICMS-ST", moeda(tot.ST)}, {"Frete e seguro", moeda(d.Frete.Add(d.Seguro))}, {"Desconto", "- " + moeda(d.Desconto)}} {
		f.texto(tx+12, ty, reg, 7.6, cinza, l[0])
		p.TextRight(tx+tw-12, ty, reg, 7.6, tinta, l[1])
		ty += 12
	}
	p.FillRect(tx, topo+98, tw, 34, ac)
	f.texto(tx+12, topo+112, neg, 7.4, mistura(ac, branco, 0.7), "TOTAL DA PROPOSTA")
	p.TextRight(tx+tw-12, topo+126, neg, 14, branco, moeda(tot.Total))
	f.y = maxF(ly, topo+132) + 16

	if obs := strings.TrimSpace(juntar("\n", d.ObservacaoCliente, d.Observacao)); obs != "" {
		ls := quebrar(obs, reg, 7.6, cw-20)
		hh := 18 + float64(len(ls))*10
		f.cabe(hh + 8)
		f.p.FillRect(mg, f.y, 3, hh, ac)
		f.texto(mg+12, f.y+10, neg, 7, ac, "OBSERVAÇÕES")
		for i, l := range ls {
			f.texto(mg+12, f.y+22+float64(i)*10, reg, 7.6, tinta, l)
		}
		f.y += hh + 14
	}

	// Aceite.
	f.cabe(96)
	p = f.p
	p.StrokeRect(mg, f.y, cw, 86, 0.6, linha)
	f.texto(mg+12, f.y+16, neg, 8, ac, "DE ACORDO DO CLIENTE")
	f.paragrafo(mg+12, f.y+29, cw-24, reg, 7.2, cinza, "Aprovamos esta proposta nas condições acima. Pedido de compra nº ______________________", 9)
	aw := (cw - 48) / 3
	for i, r := range []string{"Nome e cargo", "Assinatura", "Data"} {
		ax := mg + 12 + float64(i)*(aw+12)
		p.StrokeLine(ax, f.y+66, ax+aw, f.y+66, 0.6, tinta)
		p.TextCenter(ax+aw/2, f.y+76, reg, 6.6, cinza, r)
	}
	f.y += 96
	if d.Representante != "" {
		f.texto(mg, f.y+4, reg, 7.4, tinta, "Atenciosamente, "+d.Representante+" — "+d.Emitente.Nome)
	} else {
		f.texto(mg, f.y+4, reg, 7.4, tinta, "Atenciosamente, "+d.Emitente.Nome)
	}
	return doc.Render()
}
