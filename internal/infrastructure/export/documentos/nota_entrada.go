package documentos

import (
	"fmt"
	"strings"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Espelho da nota fiscal de entrada — documento de conferência do recebimento
// fiscal, no formato de quadros do DANFE: identificação com a chave, emitente
// e destinatário, cálculo do imposto, itens com a classificação fiscal,
// conferência com o pedido de compra, créditos, destino no estoque com o
// custo, parcelas no contas a pagar e o carimbo de aprovação.

var (
	grafite = cor{R: 44, G: 52, B: 64}
	ambarNF = cor{R: 196, G: 112, B: 8}
)

type quadroNF struct {
	f *folha
}

// campo é uma célula de formulário fiscal: rótulo minúsculo e valor.
func (q quadroNF) campo(x, y, w, h float64, rot, val string, destaque bool) {
	p := q.f.p
	p.StrokeRect(x, y, w, h, 0.6, grafite)
	p.Text(x+3.5, y+7, neg, 5.4, cinza, fit(strings.ToUpper(rot), neg, 5.4, w-7))
	ft, t := reg, 8.0
	if destaque {
		ft, t = neg, 8.8
	}
	p.Text(x+3.5, y+h-4.5, ft, t, tinta, fit(val, ft, t, w-7))
}

func (q quadroNF) titulo(s string) {
	f := q.f
	f.cabe(26)
	f.texto(f.mg, f.y+8, neg, 7.4, grafite, s)
	f.p.FillRect(f.mg, f.y+11, 26, 1.8, ambarNF)
	f.y += 15
}

func situacaoEntrada(s string) (string, cor) {
	switch s {
	case "APPROVED":
		return "APROVADA", verde
	case "PENDING":
		return "PENDENTE", ambarNF
	case "CONFERRED":
		return "CONFERIDA", azulVenda
	case "CANCELLED":
		return "CANCELADA", verm
	case "WRITTEN_OFF":
		return "BAIXADA", cinza
	}
	return ou(s, "—"), cinza
}

func (Gerador) NotaEntrada(d uc.NotaEntrada) []byte {
	doc := pdfkit.New()
	const mg = 26.0
	f := novaFolha(doc, mg, 20)
	f.logo = logo(doc, d.Emitente.Logo)
	q := quadroNF{f}
	rodapeSimples(doc, mg, grafite, fmt.Sprintf("Espelho da NF de entrada %d/%s · %s · conferência interna, não substitui o DANFE", d.Numero, ou(d.Serie, "1"), emitidoPor(d.Cabecalho)))
	f.continua = func() float64 {
		f.p.StrokeRect(mg, mg, f.cw(), 20, 0.8, grafite)
		f.texto(mg+8, mg+13.5, neg, 8, grafite, "ESPELHO DA NF DE ENTRADA")
		f.p.TextRight(f.w-mg-8, mg+13.5, neg, 8, grafite, fmt.Sprintf("NF %d · série %s · %s (continuação)", d.Numero, ou(d.Serie, "1"), fit(d.Fornecedor.Nome, neg, 8, 200)))
		return mg + 28
	}
	p := f.p
	cw := f.cw()

	// Identificação.
	top := mg
	esq := cw - 170
	p.StrokeRect(mg, top, esq, 66, 0.8, grafite)
	x := mg + 8
	if lw := desenharLogo(p, f.logo, x, top+8, 70, 34); lw > 0 {
		x += lw + 10
	}
	f.texto(x, top+20, neg, 13, grafite, "ESPELHO DA NOTA FISCAL DE ENTRADA")
	f.texto(x, top+32, reg, 7.2, cinza, "Conferência do recebimento fiscal: imposto, pedido de compra, estoque e financeiro")
	f.texto(x, top+44, neg, 7.4, tinta, fit(d.Emitente.Nome, neg, 7.4, mg+esq-x-8))
	sit, cs := situacaoEntrada(d.Situacao)
	selo(p, x, top+50, sit, cs, branco, 6.8)
	rx := mg + esq
	p.FillRect(rx, top, 170, 66, grafite)
	f.texto(rx+12, top+15, neg, 7, mistura(grafite, branco, 0.6), "NOTA FISCAL Nº")
	f.texto(rx+12, top+37, neg, 20, branco, milhar(fmt.Sprint(d.Numero)))
	f.texto(rx+12, top+52, reg, 7.4, mistura(grafite, branco, 0.75), fmt.Sprintf("Série %s · Modelo %s", ou(d.Serie, "1"), ou(d.Modelo, "55")))
	f.texto(rx+12, top+61, reg, 7, mistura(grafite, branco, 0.75), "Tipo: 0 — Entrada")
	f.y = top + 72

	// Chave de acesso.
	if len(strings.TrimSpace(d.Chave)) == 44 {
		p.StrokeRect(mg, f.y, cw, 46, 0.6, grafite)
		p.Text(mg+4, f.y+7, neg, 5.4, cinza, "CHAVE DE ACESSO")
		_ = p.DrawCode128B(d.Chave, mg+6, f.y+10, cw-12, 22, preto)
		p.TextCenter(mg+cw/2, f.y+42, neg, 8.4, tinta, chaveNFe(d.Chave))
		f.y += 46
	} else {
		p.StrokeRect(mg, f.y, cw, 30, 0.6, grafite)
		p.Text(mg+4, f.y+7, neg, 5.4, cinza, "CHAVE DE ACESSO")
		p.Text(mg+6, f.y+21, ita, 7.8, ambarNF, "Nota digitada manualmente, sem chave de acesso: confira com o documento físico do fornecedor.")
		f.y += 30
	}
	ws := []float64{cw * 0.34, cw * 0.14, cw * 0.14, cw * 0.18, cw * 0.20}
	vals := [][2]string{{"Natureza da operação", ou(d.Natureza, "Compra para industrialização")}, {"Emissão", data(d.Emissao)},
		{"Entrada", data(d.Entrada)}, {"Pedido de compra", pedidoSe(d.PedidoCompra)}, {"Protocolo de autorização", ou(d.Protocolo, "—")}}
	xx := mg
	for i, v := range vals {
		q.campo(xx, f.y, ws[i], 24, v[0], v[1], i == 3)
		xx += ws[i]
	}
	f.y += 32

	// Emitente e destinatário.
	q.titulo("EMITENTE (FORNECEDOR) E DESTINATÁRIO")
	half := cw / 2
	parte := func(x float64, rot string, pt uc.Parte, cod int64) {
		p.StrokeRect(x, f.y, half, 62, 0.6, grafite)
		p.Text(x+4, f.y+7, neg, 5.4, cinza, rot)
		nome := pt.Nome
		if cod != 0 {
			nome = fmt.Sprintf("%s  (cód. %d)", pt.Nome, cod)
		}
		p.Text(x+6, f.y+19, neg, 8.6, tinta, fit(nome, neg, 8.6, half-12))
		ly := f.y + 30.0
		ls := linhasDaParte(pt)
		if strings.TrimSpace(pt.Endereco) == "" {
			ls = append(ls, "Endereço não cadastrado")
		}
		for _, l := range ls {
			p.Text(x+6, ly, reg, 7.2, tinta, fit(l, reg, 7.2, half-12))
			ly += 9
		}
	}
	parte(mg, "EMITENTE", d.Fornecedor, d.Fornecedor.Codigo)
	parte(mg+half, "DESTINATÁRIO", d.Emitente.Parte, 0)
	f.y += 70

	// Cálculo do imposto.
	q.titulo("CÁLCULO DO IMPOSTO")
	l1 := [][2]string{{"Base de cálculo do ICMS", num(d.BaseICMS, 2)}, {"Valor do ICMS", num(d.ICMS, 2)}, {"Base ICMS ST", "0,00"},
		{"Valor do ICMS ST", num(d.ICMSST, 2)}, {"Valor total dos produtos", num(d.Produtos, 2)}}
	l2 := [][2]string{{"Frete", num(d.Frete, 2)}, {"Seguro", num(d.Seguro, 2)}, {"Desconto", num(d.Desconto, 2)}, {"Outras despesas", num(d.Outras, 2)},
		{"Valor do IPI", num(d.IPI, 2)}, {"PIS", num(d.PIS, 2)}, {"COFINS", num(d.COFINS, 2)}}
	linhaCampos := func(vs [][2]string, ult string) {
		w := (cw - 110) / float64(len(vs))
		for i, v := range vs {
			q.campo(mg+float64(i)*w, f.y, w, 24, v[0], v[1], false)
		}
		if ult != "" {
			p.FillRect(mg+cw-110, f.y, 110, 24, mistura(ambarNF, branco, 0.85))
			q.campo(mg+cw-110, f.y, 110, 24, "Valor total da nota", ult, true)
		} else {
			q.campo(mg+cw-110, f.y, 110, 24, "Itens", fmt.Sprint(len(d.Itens)), false)
		}
		f.y += 24
	}
	linhaCampos(l1, "")
	linhaCampos(l2, moeda(d.Total))
	f.y += 10

	// Itens.
	q.titulo("DADOS DOS PRODUTOS")
	cols := []col{{"#", 0.3, pdfkit.AlignCenter}, {"Código", 0.85, pdfkit.AlignLeft}, {"Descrição", 2.6, pdfkit.AlignLeft},
		{"NCM", 0.75, pdfkit.AlignCenter}, {"CST", 0.4, pdfkit.AlignCenter}, {"CFOP", 0.45, pdfkit.AlignCenter}, {"Un", 0.35, pdfkit.AlignCenter},
		{"Qtde", 0.8, pdfkit.AlignRight}, {"V. unit.", 0.75, pdfkit.AlignRight}, {"V. total", 0.85, pdfkit.AlignRight},
		{"BC ICMS", 0.8, pdfkit.AlignRight}, {"ICMS", 0.65, pdfkit.AlignRight}, {"Alíq.", 0.55, pdfkit.AlignRight}, {"IPI", 0.5, pdfkit.AlignRight}}
	st := estiloTabela{cabFundo: grafite, cabTexto: branco, grade: linha, vertical: true, tam: 6.9, tamCab: 5.7, altLinha: 15}
	t := f.tabela(mg, cw, cols, st)
	t.cabecalho()
	for _, it := range d.Itens {
		t.linha([]string{fmt.Sprint(it.Seq), it.Codigo, it.Descricao, it.NCM, it.CST, it.CFOP, it.UM, qtd(it.Quantidade), num(it.Unitario, 4),
			num(it.Total, 2), num(it.BaseICMS, 2), num(it.ICMS, 2), pctCurto(it.AliqICMS), num(it.IPI, 2)}, reg, tinta, nil)
	}
	f.y += 12

	// Conferência com o pedido.
	q.titulo("CONFERÊNCIA COM O PEDIDO DE COMPRA")
	cc := []col{{"Item", 1.5, pdfkit.AlignLeft}, {"Pedido/linha", 0.95, pdfkit.AlignCenter}, {"Qtd. pedido", 0.95, pdfkit.AlignRight},
		{"Qtd. nota", 0.95, pdfkit.AlignRight}, {"Dif.", 0.6, pdfkit.AlignRight}, {"R$ pedido", 0.9, pdfkit.AlignRight},
		{"R$ nota", 0.9, pdfkit.AlignRight}, {"Var.", 0.6, pdfkit.AlignRight}, {"Resultado", 1.1, pdfkit.AlignLeft}}
	tc := f.tabela(mg, cw, cc, estiloTabela{cabFundo: mistura(grafite, branco, 0.85), cabTexto: grafite, grade: linha, altLinha: 16})
	tc.cabecalho()
	for _, it := range d.Itens {
		if it.Pedido == 0 {
			tc.linha([]string{it.Codigo, "sem pedido", "—", qtd(it.Quantidade), "—", "—", num(it.Unitario, 4), "—", "Entrada sem pedido"}, reg, ambarNF, nil)
			continue
		}
		difQ := it.Quantidade.Sub(it.QtdPedido)
		varP := decimal.Zero
		if it.PrecoPedido.IsPositive() {
			varP = it.Unitario.Sub(it.PrecoPedido).Div(it.PrecoPedido).Mul(decimal.NewFromInt(100))
		}
		res, c := "Confere", verde
		if difQ.Abs().GreaterThan(decimal.RequireFromString("0.001")) || varP.Abs().GreaterThan(decimal.RequireFromString("0.5")) {
			res, c = "Divergente", verm
		}
		topo := tc.linha([]string{it.Codigo, fmt.Sprintf("%d · %d", it.Pedido, it.LinhaPedido), qtd(it.QtdPedido), qtd(it.Quantidade), qtd(difQ),
			num(it.PrecoPedido, 4), num(it.Unitario, 4), pct(varP), ""}, reg, tinta, nil)
		xr := tc.xCol(8) + 5
		if res == "Confere" {
			marcaOK(f.p, xr, topo+4, c)
		} else {
			f.p.FillRect(xr, topo+4.5, 7, 7, c)
		}
		f.p.Text(xr+12, topo+10.6, neg, 7.4, c, res)
	}
	f.y += 12

	// Créditos, estoque e custo.
	q.titulo("CRÉDITOS FISCAIS, DESTINO NO ESTOQUE E CUSTO")
	ce := []col{{"Item", 1.3, pdfkit.AlignLeft}, {"ICMS", 0.55, pdfkit.AlignCenter}, {"IPI", 0.45, pdfkit.AlignCenter}, {"PIS", 0.45, pdfkit.AlignCenter},
		{"COFINS", 0.55, pdfkit.AlignCenter}, {"Crédito", 0.8, pdfkit.AlignRight}, {"Depósito", 0.75, pdfkit.AlignCenter},
		{"Qtd. estoque", 0.95, pdfkit.AlignRight}, {"Custo unit.", 0.8, pdfkit.AlignRight}, {"Custo total", 0.9, pdfkit.AlignRight}, {"Plano de contas", 2, pdfkit.AlignLeft}}
	te := f.tabela(mg, cw, ce, estiloTabela{cabFundo: mistura(grafite, branco, 0.85), cabTexto: grafite, grade: linha, altLinha: 16})
	te.cabecalho()
	totCred, totCusto := decimal.Zero, decimal.Zero
	for _, it := range d.Itens {
		cred := decimal.Zero
		if it.CreditaICMS {
			cred = cred.Add(it.ICMS)
		}
		if it.CreditaIPI {
			cred = cred.Add(it.IPI)
		}
		if it.CreditaPIS {
			cred = cred.Add(it.PIS)
		}
		if it.CreditaCOFINS {
			cred = cred.Add(it.COFINS)
		}
		custo := it.CustoAquisicao.Round(2)
		unit := decimal.Zero
		if it.QtdEstoque.IsPositive() {
			unit = it.CustoAquisicao.Div(it.QtdEstoque)
		}
		totCred, totCusto = totCred.Add(cred), totCusto.Add(custo)
		topo := te.linha([]string{it.Codigo, "", "", "", "", num(cred, 2), ou(it.Deposito, "—"), qtd(it.QtdEstoque), num(unit, 4), num(custo, 2), ou(it.PlanoContas, "sem conta")}, reg, tinta, nil)
		for k, ok := range []bool{it.CreditaICMS, it.CreditaIPI, it.CreditaPIS, it.CreditaCOFINS} {
			cx := te.xCol(1+k) + te.ws[1+k]/2 - 4
			if ok {
				marcaOK(f.p, cx, topo+4.5, verde)
			} else {
				f.p.StrokeLine(cx+1, topo+8, cx+7, topo+8, 1, cinza)
			}
		}
	}
	te.linha([]string{"Totais", "", "", "", "", num(totCred, 2), "", "", "", num(totCusto, 2), ""}, neg, tinta, &fundo)
	f.y += 12

	// Financeiro.
	q.titulo("FINANCEIRO — PARCELAS DO CONTAS A PAGAR")
	if len(d.Parcelas) == 0 {
		f.cabe(18)
		f.texto(mg, f.y+8, ita, 7.4, cinza, "Nota sem parcelas: não gera contas a pagar.")
		f.y += 16
	} else {
		tf := f.tabela(mg, cw*0.7, []col{{"Parcela", 0.6, pdfkit.AlignCenter}, {"Documento", 1.3, pdfkit.AlignLeft}, {"Vencimento", 0.9, pdfkit.AlignCenter},
			{"Valor", 1, pdfkit.AlignRight}, {"Título", 1.6, pdfkit.AlignLeft}}, estiloTabela{cabFundo: mistura(grafite, branco, 0.85), cabTexto: grafite, grade: linha})
		tf.cabecalho()
		for _, pa := range d.Parcelas {
			tf.linha([]string{fmt.Sprint(pa.Numero), pa.Documento, data(pa.Vencimento), moeda(pa.Valor), rotuloTitulo(pa.Titulo)}, reg, tinta, nil)
		}
	}
	f.y += 12

	if obs := strings.TrimSpace(d.Observacao); obs != "" {
		ls := quebrar(obs, reg, 7.4, cw-16)
		h := 16 + float64(len(ls))*9.5
		f.cabe(h + 8)
		f.p.StrokeRect(mg, f.y, cw, h, 0.6, grafite)
		f.p.Text(mg+4, f.y+7, neg, 5.4, cinza, "INFORMAÇÕES COMPLEMENTARES / OBSERVAÇÕES")
		for i, l := range ls {
			f.texto(mg+8, f.y+18+float64(i)*9.5, reg, 7.4, tinta, l)
		}
		f.y += h + 12
	}

	// Aprovação.
	f.cabe(70)
	p = f.p
	bw := (cw - 20) / 3
	for i, r := range []string{"Recebimento físico (almoxarifado)", "Conferência fiscal", "Aprovação da nota"} {
		bx := mg + float64(i)*(bw+10)
		p.StrokeRect(bx, f.y, bw, 56, 0.6, grafite)
		p.Text(bx+5, f.y+9, neg, 5.8, cinza, strings.ToUpper(r))
		if i == 2 && d.AprovadaEm != nil {
			p.FillRect(bx+1, f.y+13, bw-2, 30, mistura(verde, branco, 0.88))
			p.Text(bx+8, f.y+26, neg, 9, verde, "APROVADA")
			p.Text(bx+8, f.y+37, reg, 7, tinta, fit(dataHora(*d.AprovadaEm)+rotulo(" · ", d.AprovadaPor), reg, 7, bw-16))
		} else {
			p.StrokeLine(bx+8, f.y+38, bx+bw-8, f.y+38, 0.5, tinta)
		}
		p.Text(bx+8, f.y+50, reg, 6.3, cinza, "Data  ____/____/________")
	}
	f.y += 64
	return doc.Render()
}

func pedidoSe(n int64) string {
	if n == 0 {
		return "sem pedido"
	}
	return fmt.Sprintf("Nº %d", n)
}

func rotuloTitulo(s string) string {
	r := strings.NewReplacer("PENDENTE ·", "Em aberto ·", "PAGO", "Pago", "CANCELADO", "Cancelado", "PENDENTE", "Em aberto")
	return r.Replace(s)
}
