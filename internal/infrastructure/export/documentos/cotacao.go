package documentos

import (
	"fmt"
	"strings"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Cotação de compra em dois documentos:
//   - Solicitação de cotação: vai ao fornecedor, com os itens e os campos que ele
//     devolve preenchidos (preço, IPI, prazo, condições).
//   - Mapa comparativo: uso interno, itens × fornecedores com o melhor preço
//     destacado, o vencedor escolhido e o total de cada fornecedor.

var oliva = cor{R: 63, G: 98, B: 18}

func situacaoCotacao(s string) string {
	switch s {
	case "OPEN":
		return "Aberta"
	case "QUOTED", "PARTIAL":
		return "Em cotação"
	case "CLOSED", "ATTENDED":
		return "Encerrada"
	case "CANCELLED":
		return "Cancelada"
	}
	return ou(s, "—")
}

func (Gerador) SolicitacaoCotacao(d uc.Cotacao) []byte {
	doc := pdfkit.New()
	const mg = 30.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := oliva
	claro := mistura(ac, branco, 0.9)
	forn := uc.Parte{}
	if d.Fornecedor != nil {
		forn = *d.Fornecedor
	}
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Solicitação de cotação nº %d · %s · %s", d.Codigo, forn.Nome, emitidoPor(d.Cabecalho)))
	f.continua = func() float64 {
		f.p.FillRect(mg, mg, f.cw(), 20, ac)
		f.texto(mg+8, mg+13.5, neg, 8.5, branco, fmt.Sprintf("SOLICITAÇÃO DE COTAÇÃO Nº %d (continuação)", d.Codigo))
		return mg + 30
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho em faixa oliva com logo em cartão branco.
	p.FillRect(mg, mg, cw, 62, ac)
	x := mg + 12
	if f.logo != nil {
		p.FillRect(mg+8, mg+8, 80, 46, branco)
		desenharLogo(p, f.logo, mg+12, mg+11, 72, 40)
		x = mg + 100
	}
	f.texto(x, mg+20, neg, 7.2, mistura(ac, branco, 0.65), "SUPRIMENTOS · PEDIDO DE PREÇOS")
	f.texto(x, mg+40, neg, 16, branco, "Solicitação de cotação")
	f.texto(x, mg+53, reg, 7.4, mistura(ac, branco, 0.78), "Emitida em "+data(d.Emissao))
	p.FillRect(f.w-mg-110, mg+10, 100, 42, branco)
	p.TextCenter(f.w-mg-60, mg+24, neg, 6.4, cinza, "COTAÇÃO Nº")
	p.TextCenter(f.w-mg-60, mg+44, neg, 18, ac, fmt.Sprint(d.Codigo))
	f.y = mg + 74

	// Para / De.
	colW := (cw - 12) / 2
	h := 86.0
	quadro := func(x float64, rot, nome string, ls []string) {
		p.StrokeRect(x, f.y, colW, h, 0.6, linha)
		p.FillRect(x, f.y, 4, h, ac)
		p.Text(x+12, f.y+12, neg, 6.6, ac, rot)
		y := f.y + 25
		for _, l := range quebrar(nome, neg, 8.8, colW-20) {
			p.Text(x+12, y, neg, 8.8, tinta, l)
			y += 10.5
		}
		for _, l := range ls {
			for _, q := range quebrar(l, reg, 7.2, colW-20) {
				if y > f.y+h-4 {
					return
				}
				p.Text(x+12, y, reg, 7.2, tinta, q)
				y += 9.3
			}
		}
	}
	quadro(mg, "PARA (FORNECEDOR)", forn.Nome, linhasDaParte(forn))
	de := linhasDaParte(d.Emitente.Parte)
	if d.Comprador != "" {
		de = append([]string{"Comprador: " + d.Comprador}, de...)
	}
	quadro(mg+colW+12, "DE (COMPRADOR)", d.Emitente.Nome, de)
	f.y += h + 12

	f.y += f.paragrafo(mg, f.y+8, cw, reg, 8, tinta,
		"Solicitamos o seu melhor preço e prazo para os itens abaixo. Preencha as colunas em branco e as condições ao final, e devolva esta solicitação assinada ou envie a proposta citando o número da cotação.", 10.5)
	f.y += 12

	// Itens com campos para preencher.
	cols := []col{{"#", 0.3, pdfkit.AlignCenter}, {"Código", 0.85, pdfkit.AlignLeft}, {"Descrição", 2.5, pdfkit.AlignLeft}, {"Un", 0.35, pdfkit.AlignCenter},
		{"Qtde", 0.7, pdfkit.AlignRight}, {"Entrega", 0.75, pdfkit.AlignCenter}, {"Preço unit.", 0.85, pdfkit.AlignCenter}, {"IPI %", 0.5, pdfkit.AlignCenter},
		{"Prazo (dias)", 0.65, pdfkit.AlignCenter}, {"Total", 0.85, pdfkit.AlignCenter}}
	t := f.tabela(mg, cw, cols, estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha, vertical: true, altLinha: 24})
	t.cabecalho()
	if len(d.Itens) == 0 {
		t.linha([]string{"", "", "Cotação sem itens."}, ita, cinza, nil)
	}
	for _, it := range d.Itens {
		topo := t.linha([]string{fmt.Sprint(it.Seq), it.Codigo, "", it.UM, qtd(it.Quantidade), dataP(it.Entrega), "", "", "", ""}, reg, tinta, nil)
		ls := quebrar(it.Descricao, reg, 7.4, t.ws[2]-7)
		if len(ls) > 2 {
			ls = ls[:2]
		}
		for i, l := range ls {
			f.texto(t.xCol(2)+3.5, topo+10+float64(i)*9, reg, 7.4, tinta, l)
		}
		f.p.FillRect(t.xCol(6)+1, topo+1, t.xCol(10)-t.xCol(6)-2, 22, mistura(ac, branco, 0.96))
	}
	f.y += 14

	// Condições a informar.
	f.cabe(150)
	p = f.p
	f.texto(mg, f.y+6, neg, 8.5, ac, "CONDIÇÕES DA PROPOSTA (preenchimento do fornecedor)")
	f.y += 14
	linhaForm := func(rot string, w float64, x float64) {
		p.Text(x, f.y+10, neg, 6.6, cinza, strings.ToUpper(rot))
		p.StrokeLine(x+larg(neg, 6.6, strings.ToUpper(rot))+6, f.y+11, x+w, f.y+11, 0.5, tinta)
	}
	half := (cw - 20) / 2
	linhaForm("Condição de pagamento", half, mg)
	linhaForm("Validade da proposta", half, mg+half+20)
	f.y += 20
	p.Text(mg, f.y+10, neg, 6.6, cinza, "FRETE")
	fx := mg + 40
	for _, o := range []string{"CIF (por conta do fornecedor)", "FOB (por nossa conta)"} {
		caixa(p, fx, f.y+3, tinta)
		p.Text(fx+12, f.y+10, reg, 7.4, tinta, o)
		fx += 24 + larg(reg, 7.4, o)
	}
	linhaForm("Valor do frete", half-80, mg+half+100)
	f.y += 20
	linhaForm("Marca / norma / observações", cw, mg)
	f.y += 18
	p.StrokeLine(mg, f.y+11, mg+cw, f.y+11, 0.5, tinta)
	f.y += 26
	p.FillRect(mg, f.y, cw, 1, claro)
	f.y += 10
	aw := (cw - 24) / 3
	for i, r := range []string{"Responsável pela proposta", "Assinatura e carimbo", "Data"} {
		ax := mg + float64(i)*(aw+12)
		p.StrokeLine(ax, f.y+26, ax+aw, f.y+26, 0.6, tinta)
		p.TextCenter(ax+aw/2, f.y+36, reg, 6.6, cinza, r)
	}
	f.y += 48
	if obs := strings.TrimSpace(d.Observacao); obs != "" {
		f.cabe(30)
		f.y += f.paragrafo(mg, f.y+8, cw, ita, 7.4, cinza, "Observações do comprador: "+obs, 9.5)
	}
	return doc.Render()
}

func (Gerador) MapaCotacao(d uc.Cotacao) []byte {
	doc := pdfkit.NewLandscape()
	const mg = 26.0
	f := novaFolha(doc, mg, 20)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := oliva
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Mapa comparativo da cotação nº %d · uso interno · %s", d.Codigo, emitidoPor(d.Cabecalho)))
	cab := func(cont bool) float64 {
		p := f.p
		x := mg
		if !cont {
			x += desenharLogo(p, f.logo, mg, mg, 64, 30) + 12
		}
		t := fmt.Sprintf("MAPA COMPARATIVO · COTAÇÃO Nº %d", d.Codigo)
		if cont {
			t += " (continuação)"
		}
		f.texto(x, mg+13, neg, 13, ac, t)
		f.texto(x, mg+26, reg, 7.4, cinza, fmt.Sprintf("Emitida em %s · %s · %d fornecedor(es) · %d item(ns) · %s", data(d.Emissao), situacaoCotacao(d.Situacao), len(d.Fornecedores), len(d.Itens), d.Emitente.Nome))
		p.FillRect(mg, mg+34, f.cw(), 2, ac)
		return mg + 44
	}
	f.y = cab(false)
	f.continua = func() float64 { return cab(true) }
	cw := f.cw()
	if len(d.Fornecedores) == 0 {
		f.texto(mg, f.y+14, ita, 9, cinza, "Nenhum fornecedor convidado nesta cotação.")
		return doc.Render()
	}

	// Totais por fornecedor (só itens cotados).
	type resumo struct {
		total           decimal.Decimal
		cotados, ganhos int
	}
	res := map[int64]*resumo{}
	for _, fo := range d.Fornecedores {
		res[fo.Codigo] = &resumo{}
	}
	melhor := map[int64]int64{} // item → fornecedor de menor preço
	for _, it := range d.Itens {
		var best *uc.PrecoCotacao
		for i := range it.Precos {
			pr := &it.Precos[i]
			if r := res[pr.Fornecedor]; r != nil && pr.Unitario.IsPositive() {
				r.total = r.total.Add(pr.Unitario.Mul(it.Quantidade))
				r.cotados++
				if best == nil || pr.Unitario.LessThan(best.Unitario) {
					best = pr
				}
			}
		}
		if best != nil {
			melhor[it.ID] = best.Fornecedor
			res[best.Fornecedor].ganhos++
		}
	}

	// Matriz: itens nas linhas, fornecedores nas colunas (até 5 por bloco).
	const porBloco = 5
	for b := 0; b < len(d.Fornecedores); b += porBloco {
		fs := d.Fornecedores[b:min(b+porBloco, len(d.Fornecedores))]
		fixa := 250.0
		fw := (cw - fixa) / float64(len(fs))
		f.cabe(60)
		p := f.p
		// Cabeçalho dos fornecedores.
		p.FillRect(mg, f.y, fixa, 30, ac)
		p.Text(mg+6, f.y+19, neg, 7, branco, "ITEM · QUANTIDADE")
		for i, fo := range fs {
			x := mg + fixa + float64(i)*fw
			p.FillRect(x+1, f.y, fw-1, 30, mistura(ac, branco, 0.85))
			p.Text(x+6, f.y+12, neg, 7, ac, fit(fo.Nome, neg, 7, fw-12))
			p.Text(x+6, f.y+22, reg, 6, cinza, "unit. · total · prazo · cond.")
		}
		f.y += 30
		for n, it := range d.Itens {
			if f.cabe(26) {
				p = f.p
			}
			p = f.p
			if n%2 == 1 {
				p.FillRect(mg, f.y, cw, 26, fundo)
			}
			p.Text(mg+6, f.y+11, neg, 7.4, tinta, fit(it.Codigo+"  "+it.Descricao, neg, 7.4, fixa-12))
			p.Text(mg+6, f.y+21, reg, 6.6, cinza, qtd(it.Quantidade)+" "+it.UM+rotulo(" · entrega ", dataP(it.Entrega)))
			for i, fo := range fs {
				x := mg + fixa + float64(i)*fw
				var pr *uc.PrecoCotacao
				for k := range it.Precos {
					if it.Precos[k].Fornecedor == fo.Codigo {
						pr = &it.Precos[k]
					}
				}
				if pr == nil || !pr.Unitario.IsPositive() {
					p.Text(x+6, f.y+15, ita, 7, cinza, "sem preço")
					continue
				}
				if melhor[it.ID] == fo.Codigo {
					p.FillRect(x+1, f.y+1, fw-2, 24, mistura(verde, branco, 0.82))
				}
				p.Text(x+6, f.y+11, neg, 7.6, tinta, num(pr.Unitario, 4))
				p.TextRight(x+fw-6, f.y+11, reg, 7.2, tinta, num(pr.Unitario.Mul(it.Quantidade), 2))
				p.Text(x+6, f.y+21, reg, 6.4, cinza, fit(juntar(" · ", fmt.Sprintf("%d d", pr.PrazoDias), pr.Condicao), reg, 6.4, fw-30))
				if pr.Escolhido {
					marcaOK(p, x+fw-14, f.y+14, verde)
				}
			}
			p.StrokeLine(mg, f.y+26, mg+cw, f.y+26, 0.3, linha)
			f.y += 26
		}
		// Totais do bloco.
		p = f.p
		p.FillRect(mg, f.y, cw, 30, mistura(ac, branco, 0.9))
		p.Text(mg+6, f.y+13, neg, 7.4, ac, "TOTAL COTADO")
		p.Text(mg+6, f.y+23, reg, 6.4, cinza, "soma dos itens com preço")
		for i, fo := range fs {
			x := mg + fixa + float64(i)*fw
			r := res[fo.Codigo]
			p.TextRight(x+fw-6, f.y+13, neg, 8.4, tinta, moeda(r.total))
			p.Text(x+6, f.y+24, reg, 6.4, cinza, fmt.Sprintf("%d de %d itens · menor preço em %d", r.cotados, len(d.Itens), r.ganhos))
		}
		f.y += 40
	}

	// Legenda e aprovação.
	f.cabe(80)
	p := f.p
	p.FillRect(mg, f.y, 14, 9, mistura(verde, branco, 0.82))
	f.texto(mg+18, f.y+7.5, reg, 6.8, cinza, "menor preço do item")
	marcaOK(p, mg+120, f.y, verde)
	f.texto(mg+132, f.y+7.5, reg, 6.8, cinza, "fornecedor escolhido no sistema (VSUP0400)")
	f.y += 26
	aw := (cw - 30) / 3
	for i, r := range []string{"Comprador", "Aprovação (alçada)", "Data"} {
		ax := mg + float64(i)*(aw+15)
		p.StrokeLine(ax, f.y+24, ax+aw, f.y+24, 0.6, tinta)
		p.TextCenter(ax+aw/2, f.y+34, reg, 6.6, cinza, r)
		if i == 0 && d.Comprador != "" {
			p.TextCenter(ax+aw/2, f.y+20, reg, 7.4, tinta, d.Comprador)
		}
	}
	return doc.Render()
}
