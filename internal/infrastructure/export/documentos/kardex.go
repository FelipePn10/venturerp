package documentos

import (
	"fmt"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Kardex — ficha de estoque no formato de livro razão: resumo do período por
// item e, para cada item, saldo anterior, cada entrada e saída com a origem,
// o saldo corrente em quantidade e valor, e o saldo final com o custo médio.

var ardosia = cor{R: 51, G: 65, B: 85}

func rotuloMovimento(t string) string {
	switch t {
	case "IN":
		return "Entrada (NF)"
	case "OUT":
		return "Saída"
	case "REP":
		return "Requisição p/ OF"
	case "EP":
		return "Produção (entrada)"
	case "ENTRADA":
		return "Ajuste de entrada"
	case "SAIDA":
		return "Ajuste de saída"
	case "TRANSFER_IN":
		return "Transferência (entrada)"
	case "TRANSFER_OUT":
		return "Transferência (saída)"
	case "DEV":
		return "Devolução"
	}
	return t
}

func (Gerador) Kardex(d uc.Kardex) []byte {
	doc := pdfkit.New()
	const mg = 28.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := ardosia
	periodo := data(d.De) + " a " + data(d.Ate)
	rodapeSimples(doc, mg, ac, "Kardex · "+periodo+" · "+ou(d.Deposito, "todos os depósitos")+" · "+emitidoPor(d.Cabecalho))
	f.continua = func() float64 {
		f.p.StrokeLine(mg, mg+16, f.w-mg, mg+16, 0.8, ac)
		f.texto(mg, mg+11, neg, 8.5, ac, "KARDEX · FICHA DE ESTOQUE (continuação)")
		f.p.TextRight(f.w-mg, mg+11, reg, 7.4, cinza, periodo)
		return mg + 26
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho em moldura dupla, como livro.
	p.StrokeRect(mg, mg, cw, 64, 1.4, ac)
	p.StrokeRect(mg+3, mg+3, cw-6, 58, 0.5, ac)
	x := mg + 12
	x += desenharLogo(p, f.logo, x, mg+12, 70, 40) + 12
	f.texto(x, mg+24, neg, 16, ac, "KARDEX")
	f.texto(x, mg+37, neg, 8, tinta, "Ficha de movimentação e saldo de estoque")
	f.texto(x, mg+49, reg, 7.2, cinza, fit(d.Emitente.Nome, reg, 7.2, cw-x-170))
	rx := f.w - mg - 160
	p.StrokeLine(rx, mg+10, rx, mg+54, 0.5, linha)
	for i, l := range [][2]string{{"PERÍODO", periodo}, {"DEPÓSITO", ou(d.Deposito, "Todos")}, {"ITENS", fmt.Sprint(len(d.Itens))}} {
		f.texto(rx+10, mg+18+float64(i)*14, neg, 5.8, cinza, l[0])
		f.texto(rx+60, mg+18+float64(i)*14, neg, 7.6, tinta, fit(l[1], neg, 7.6, 95))
	}
	f.y = mg + 76

	if len(d.Itens) == 0 {
		f.texto(mg, f.y+12, ita, 8.5, cinza, "Nenhuma movimentação no período.")
		return doc.Render()
	}

	// Resumo.
	f.texto(mg, f.y+6, neg, 8.5, ac, "RESUMO DO PERÍODO")
	f.y += 11
	rs := f.tabela(mg, cw, []col{{"Item", 0.9, pdfkit.AlignLeft}, {"Descrição", 2.4, pdfkit.AlignLeft}, {"Un", 0.35, pdfkit.AlignCenter},
		{"Saldo ant.", 0.85, pdfkit.AlignRight}, {"Entradas", 0.85, pdfkit.AlignRight}, {"Saídas", 0.85, pdfkit.AlignRight},
		{"Saldo final", 0.85, pdfkit.AlignRight}, {"Custo médio", 0.9, pdfkit.AlignRight}, {"Valor final", 0.95, pdfkit.AlignRight}},
		estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha, altLinha: 14})
	rs.cabecalho()
	totalValor := decimal.Zero
	for _, it := range d.Itens {
		ent, sai := decimal.Zero, decimal.Zero
		for _, m := range it.Movimentos {
			if m.Entrada {
				ent = ent.Add(m.Quantidade)
			} else {
				sai = sai.Add(m.Quantidade)
			}
		}
		v := it.SaldoFinal.Mul(it.CustoMedio).Round(2)
		totalValor = totalValor.Add(v)
		rs.linha([]string{it.Codigo, it.Descricao, it.UM, qtd(it.SaldoInicial), qtd(ent), qtd(sai), qtd(it.SaldoFinal), num(it.CustoMedio, 4), num(v, 2)}, reg, tinta, nil)
	}
	rs.linha([]string{"", "Valor do estoque dos itens no fim do período", "", "", "", "", "", "", num(totalValor, 2)}, neg, tinta, &fundo)
	f.y += 16

	// Fichas por item.
	cols := []col{{"Data", 0.95, pdfkit.AlignLeft}, {"Dep.", 0.85, pdfkit.AlignCenter}, {"Movimento", 1.5, pdfkit.AlignLeft},
		{"Origem", 1.65, pdfkit.AlignLeft}, {"Entrada", 0.8, pdfkit.AlignRight}, {"Saída", 0.8, pdfkit.AlignRight},
		{"Saldo", 0.85, pdfkit.AlignRight}, {"Custo unit.", 0.85, pdfkit.AlignRight}, {"Valor saldo", 0.95, pdfkit.AlignRight}}
	z := mistura(ac, branco, 0.95)
	for _, it := range d.Itens {
		f.cabe(22 + 15*3)
		f.p.FillRect(mg, f.y, cw, 20, mistura(ac, branco, 0.86))
		f.p.FillRect(mg, f.y, 4, 20, ac)
		f.texto(mg+10, f.y+13.5, neg, 8.6, ac, it.Codigo)
		f.texto(mg+14+larg(neg, 8.6, it.Codigo), f.y+13.5, reg, 8, tinta, fit(it.Descricao+" · "+it.UM, reg, 8, cw-200))
		f.p.TextRight(f.w-mg-8, f.y+13.5, reg, 7, cinza, fmt.Sprintf("%d movimento(s)", len(it.Movimentos)))
		f.y += 22
		t := f.tabela(mg, cw, cols, estiloTabela{cabFundo: branco, cabTexto: cinza, grade: linha, zebra: &z, altLinha: 14, tam: 7.2})
		t.cabecalho()
		f.p.StrokeLine(mg, f.y, mg+cw, f.y, 0.8, ac)
		saldo, valor := it.SaldoInicial, it.CustoInicial
		t.linha([]string{data(d.De), "", "Saldo anterior", "", "", "", qtd(saldo), "", num(valor, 2)}, ita, cinza, nil)
		for _, m := range it.Movimentos {
			e, s := "", ""
			if m.Entrada {
				e = qtd(m.Quantidade)
				saldo, valor = saldo.Add(m.Quantidade), valor.Add(m.Quantidade.Mul(m.Custo))
			} else {
				s = qtd(m.Quantidade)
				saldo, valor = saldo.Sub(m.Quantidade), valor.Sub(m.Quantidade.Mul(m.Custo))
			}
			topo := t.linha([]string{m.Data.Format("02/01/06 15:04"), m.Deposito, "", juntar(" · ", m.Origem, rotulo("lote ", m.Lote)), e, s,
				qtd(saldo), num(m.Custo, 4), num(valor, 2)}, reg, tinta, nil)
			f.texto(t.xCol(2)+13, topo+10.2, reg, 7.2, tinta, fit(rotuloMovimento(m.Tipo), reg, 7.2, t.ws[2]-16))
			c, sinal := verde, "+"
			if !m.Entrada {
				c, sinal = verm, "−"
			}
			if sinal == "−" {
				f.p.FillRect(t.xCol(2)+4, topo+6.6, 5, 1.3, c)
			} else {
				f.p.FillRect(t.xCol(2)+4, topo+6.6, 5, 1.3, c)
				f.p.FillRect(t.xCol(2)+5.85, topo+4.75, 1.3, 5, c)
			}
		}
		t.linha([]string{data(d.Ate), "", "Saldo final", "", "", "", qtd(it.SaldoFinal), num(it.CustoMedio, 4), num(it.SaldoFinal.Mul(it.CustoMedio), 2)}, neg, ac, &fundo)
		f.y += 14
	}
	return doc.Render()
}
