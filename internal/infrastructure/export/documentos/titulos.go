package documentos

import (
	"fmt"
	"sort"
	"strings"
	"time"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

var (
	verdeReceber = cor{R: 21, G: 111, B: 70}
	tijolo       = cor{R: 150, G: 52, B: 20}
)

func (g Gerador) Titulos(d uc.Titulos) []byte {
	if d.Receber {
		return g.receber(d)
	}
	return g.pagar(d)
}

func situacaoTitulo(t uc.Titulo, ref time.Time) (string, cor) {
	switch strings.ToUpper(t.Situacao) {
	case "RECEBIDO", "PAGO", "BAIXADO":
		return "Quitado", verde
	case "CANCELADO":
		return "Cancelado", cinza
	}
	if t.Pago.IsPositive() && t.Saldo().IsPositive() {
		return "Parcial", ambar
	}
	if t.Atrasado(ref) {
		return fmt.Sprintf("Vencido há %d d", -t.DiasParaVencer(ref)), verm
	}
	n := t.DiasParaVencer(ref)
	if n == 0 {
		return "Vence hoje", ambar
	}
	return fmt.Sprintf("Vence em %d d", n), tinta
}

type resumoTitulos struct{ total, aberto, vencido, quitado decimal.Decimal }

func resumir(ts []uc.Titulo, ref time.Time) resumoTitulos {
	var r resumoTitulos
	for _, t := range ts {
		if strings.EqualFold(t.Situacao, "CANCELADO") {
			continue
		}
		r.total = r.total.Add(t.Valor)
		r.quitado = r.quitado.Add(t.Pago)
		if t.Atrasado(ref) {
			r.vencido = r.vencido.Add(t.Saldo())
		} else {
			r.aberto = r.aberto.Add(t.Saldo())
		}
	}
	return r
}

// cartoes desenha os indicadores em linha.
func cartoes(f *folha, x, w float64, itens [][3]string, cores []cor) {
	n := float64(len(itens))
	gap := 8.0
	cw := (w - gap*(n-1)) / n
	for i, it := range itens {
		cx := x + float64(i)*(cw+gap)
		f.p.FillRect(cx, f.y, cw, 46, mistura(cores[i], branco, 0.9))
		f.p.FillRect(cx, f.y, cw, 2.5, cores[i])
		f.texto(cx+9, f.y+15, neg, 6.3, cores[i], strings.ToUpper(it[0]))
		f.texto(cx+9, f.y+31, neg, 12, tinta, fit(it[1], neg, 12, cw-18))
		f.texto(cx+9, f.y+41, reg, 6.4, cinza, fit(it[2], reg, 6.4, cw-18))
	}
	f.y += 56
}

// ─── Contas a receber: demonstrativo para o cliente ─────────────────────────

func (Gerador) receber(d uc.Titulos) []byte {
	doc := pdfkit.New()
	const mg = 32.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := verdeReceber
	rodapeSimples(doc, mg, ac, "Demonstrativo de títulos a receber · "+emitidoPor(d.Cabecalho)+" · posição em "+data(d.Referencia))
	f.continua = func() float64 {
		f.texto(mg, mg+10, neg, 8.5, ac, "DEMONSTRATIVO DE TÍTULOS A RECEBER (continuação)")
		f.p.FillRect(mg, mg+15, f.cw(), 1.5, ac)
		return mg + 26
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho: logo e empresa; à direita o título em bloco verde.
	x := mg
	if lw := desenharLogo(p, f.logo, x, mg, 86, 48); lw > 0 {
		x += lw + 12
	}
	yy := mg + 12.0
	for _, l := range quebrar(d.Emitente.Nome, neg, 9.5, cw-200-(x-mg)) {
		f.texto(x, yy, neg, 9.5, tinta, l)
		yy += 11
	}
	for _, l := range linhasDaParte(d.Emitente.Parte) {
		f.texto(x, yy, reg, 6.9, cinza, fit(l, reg, 6.9, cw-200-(x-mg)))
		yy += 8.5
	}
	bx := f.w - mg - 188
	p.FillRect(bx, mg, 188, 58, ac)
	f.texto(bx+12, mg+16, neg, 7, mistura(ac, branco, 0.65), "DEMONSTRATIVO DE")
	f.texto(bx+12, mg+33, neg, 15, branco, "Títulos a receber")
	f.texto(bx+12, mg+48, reg, 7.2, mistura(ac, branco, 0.78), "Posição em "+data(d.Referencia))
	f.y = maxF(yy, mg+58) + 12

	// Destinatário (quando um cliente só).
	clientes := map[int64]uc.Parte{}
	for _, t := range d.Titulos {
		clientes[t.Parte.Codigo] = t.Parte
	}
	if len(clientes) == 1 {
		var c uc.Parte
		for _, v := range clientes {
			c = v
		}
		p.StrokeRect(mg, f.y, cw, 40, 0.6, linha)
		p.FillRect(mg, f.y, 4, 40, ac)
		f.texto(mg+14, f.y+13, neg, 6.4, ac, "CLIENTE")
		f.texto(mg+14, f.y+27, neg, 10, tinta, fit(c.Nome, neg, 10, cw*0.6))
		f.texto(mg+14, f.y+36, reg, 7, cinza, juntar(" · ", rotulo("CNPJ ", cnpj(c.CNPJCPF)), rotulo("código ", fmt.Sprint(c.Codigo))))
		f.p.TextRight(f.w-mg-12, f.y+27, reg, 7.2, cinza, "Filtro: "+d.Filtro)
		f.y += 50
	} else {
		f.texto(mg, f.y+6, reg, 7.4, cinza, fmt.Sprintf("%d cliente(s) · filtro: %s", len(clientes), d.Filtro))
		f.y += 16
	}

	r := resumir(d.Titulos, d.Referencia)
	cartoes(f, mg, cw, [][3]string{
		{"Total dos títulos", moeda(r.total), fmt.Sprintf("%d parcela(s)", len(d.Titulos))},
		{"A vencer", moeda(r.aberto), "em dia"},
		{"Vencido", moeda(r.vencido), "em atraso na data"},
		{"Recebido", moeda(r.quitado), "baixas registradas"},
	}, []cor{ac, azulVenda, verm, verde})

	// Linha do tempo dos vencimentos.
	if len(d.Titulos) > 0 {
		f.cabe(70)
		f.texto(mg, f.y+6, neg, 8, ac, "CALENDÁRIO DE VENCIMENTOS")
		f.y += 14
		ini, fim := dia(d.Referencia), dia(d.Referencia)
		for _, t := range d.Titulos {
			if dia(t.Vencimento).Before(ini) {
				ini = dia(t.Vencimento)
			}
			if dia(t.Vencimento).After(fim) {
				fim = dia(t.Vencimento)
			}
		}
		fim = fim.AddDate(0, 0, 5)
		span := fim.Sub(ini).Hours()
		if span <= 0 {
			span = 24
		}
		lx, lw := mg+10, cw-20
		pos := func(t time.Time) float64 { return lx + lw*dia(t).Sub(ini).Hours()/span }
		base := f.y + 26
		p = f.p
		p.FillRect(lx, base-1.5, lw, 3, mistura(ac, branco, 0.75))
		hx := pos(d.Referencia)
		p.FillRect(hx-0.6, base-8, 1.2, 16, verm)
		p.Text(hx+3, base+12, neg, 6.2, verm, "hoje, "+d.Referencia.Format("02/01"))
		for i, t := range d.Titulos {
			tx := pos(t.Vencimento)
			c := ac
			if t.Atrasado(d.Referencia) {
				c = verm
			}
			p.FillPolygon(c, tx-5, base-11, tx+5, base-11, tx, base-3)
			yl := base - 15.0
			if i%2 == 1 {
				yl = base - 24
			}
			p.TextCenter(tx, yl, neg, 6.6, tinta, fmt.Sprintf("%s · %s", t.Vencimento.Format("02/01/06"), num(t.Saldo(), 2)))
		}
		f.y += 54
	}

	// Títulos.
	f.texto(mg, f.y+6, neg, 8, ac, "TÍTULOS")
	f.y += 11
	cols := []col{{"Documento", 0.9, pdfkit.AlignLeft}, {"Parc.", 0.5, pdfkit.AlignCenter}, {"Origem", 1.6, pdfkit.AlignLeft},
		{"Emissão", 0.75, pdfkit.AlignCenter}, {"Vencimento", 0.8, pdfkit.AlignCenter}, {"Valor", 0.9, pdfkit.AlignRight},
		{"Recebido", 0.8, pdfkit.AlignRight}, {"Saldo", 0.9, pdfkit.AlignRight}, {"Situação", 1, pdfkit.AlignLeft}}
	if len(clientes) > 1 {
		cols = append([]col{{"Cliente", 1.6, pdfkit.AlignLeft}}, cols...)
	}
	z := mistura(ac, branco, 0.94)
	t := f.tabela(mg, cw, cols, estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha, zebra: &z, altLinha: 17})
	t.cabecalho()
	saldo := decimal.Zero
	for _, ti := range d.Titulos {
		sit, c := situacaoTitulo(ti, d.Referencia)
		v := []string{ti.Documento, fmt.Sprintf("%d/%d", ti.Parcela, ti.Parcelas), ou(ti.Origem, "—"), data(ti.Emissao), data(ti.Vencimento),
			num(ti.Valor, 2), num(ti.Pago, 2), num(ti.Saldo(), 2), ""}
		if len(clientes) > 1 {
			v = append([]string{ti.Parte.Nome}, v...)
		}
		topo := t.linha(v, reg, tinta, nil)
		xs := t.xCol(len(cols) - 1)
		f.p.FillRect(xs+4, topo+5, 3, 7, c)
		f.texto(xs+10, topo+11, neg, 7.2, c, fit(sit, neg, 7.2, t.ws[len(cols)-1]-14))
		saldo = saldo.Add(ti.Saldo())
	}
	tot := make([]string, len(cols))
	tot[0] = "Em aberto"
	tot[len(cols)-2] = num(saldo, 2)
	t.linha(tot, neg, tinta, &fundo)
	f.y += 14

	// Como pagar.
	f.cabe(60)
	p = f.p
	h := 20 + 13*float64(max(len(d.Contas), 1))
	p.StrokeRect(mg, f.y, cw, h, 0.6, ac)
	f.texto(mg+12, f.y+14, neg, 7.4, ac, "COMO PAGAR")
	yy = f.y + 28
	if len(d.Contas) == 0 {
		f.texto(mg+12, yy, ita, 7.4, verm, "Nenhuma conta bancária cadastrada (VFIN0100): informe ao cliente os dados de depósito, boleto ou PIX.")
	}
	for _, c := range d.Contas {
		f.texto(mg+12, yy, reg, 7.6, tinta, juntar("  ·  ", c.Descricao, rotulo("Banco ", c.Banco), rotulo("Agência ", c.Agencia), rotulo("Conta ", c.Conta), rotulo("Titular ", c.Titular), rotulo("PIX ", c.Pix)))
		yy += 13
	}
	f.y += h + 10
	f.paragrafo(mg, f.y+6, cw, ita, 6.8, cinza, "Pagamentos após o vencimento estão sujeitos a multa e juros de mora conforme a condição comercial. Em caso de divergência, procure o financeiro informando o número do documento.", 9)
	return doc.Render()
}

// ─── Contas a pagar: programação e autorização ──────────────────────────────

func (Gerador) pagar(d uc.Titulos) []byte {
	doc := pdfkit.NewLandscape()
	const mg = 28.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := tijolo
	rodapeSimples(doc, mg, ac, "Programação de pagamentos · "+emitidoPor(d.Cabecalho)+" · "+d.Filtro)
	cab := func(cont bool) float64 {
		p := f.p
		p.FillRect(mg, mg, f.cw(), 40, ac)
		x := mg + 12
		if f.logo != nil && !cont {
			p.FillRect(mg+8, mg+5, 70, 30, branco)
			desenharLogo(p, f.logo, mg+12, mg+7, 62, 26)
			x = mg + 88
		}
		t := "PROGRAMAÇÃO E AUTORIZAÇÃO DE PAGAMENTOS"
		if cont {
			t += " (continuação)"
		}
		f.texto(x, mg+18, neg, 12, branco, t)
		f.texto(x, mg+31, reg, 7.4, mistura(ac, branco, 0.75), fit(d.Emitente.Nome+" · CNPJ "+cnpj(d.Emitente.CNPJCPF), reg, 7.4, f.cw()-x-180))
		f.p.TextRight(f.w-mg-12, mg+18, neg, 8, branco, "Posição em "+data(d.Referencia))
		f.p.TextRight(f.w-mg-12, mg+31, reg, 7, mistura(ac, branco, 0.75), d.Filtro)
		return mg + 50
	}
	f.y = cab(false)
	f.continua = func() float64 { return cab(true) }
	cw := f.cw()

	r := resumir(d.Titulos, d.Referencia)
	pend := decimal.Zero
	for _, t := range d.Titulos {
		if strings.EqualFold(t.Aprovacao, "PENDENTE") {
			pend = pend.Add(t.Saldo())
		}
	}
	cartoes(f, mg, cw, [][3]string{
		{"Total programado", moeda(r.total), fmt.Sprintf("%d título(s)", len(d.Titulos))},
		{"A pagar em dia", moeda(r.aberto), "dentro do vencimento"},
		{"Vencido", moeda(r.vencido), "pagar com prioridade"},
		{"Aguardando aprovação", moeda(pend), "sem alçada não se paga"},
		{"Pago", moeda(r.quitado), "baixas registradas"},
	}, []cor{ac, azulVenda, verm, ambar, verde})

	// Títulos agrupados por vencimento.
	cols := []col{{"Pagar", 0.55, pdfkit.AlignCenter}, {"Documento", 1, pdfkit.AlignLeft}, {"Fornecedor", 2.4, pdfkit.AlignLeft},
		{"CNPJ", 1.2, pdfkit.AlignLeft}, {"Origem", 1.4, pdfkit.AlignLeft}, {"Plano de contas", 1.9, pdfkit.AlignLeft},
		{"Parc.", 0.5, pdfkit.AlignCenter}, {"Aprovação", 0.9, pdfkit.AlignCenter}, {"Valor", 0.95, pdfkit.AlignRight}, {"Saldo", 0.95, pdfkit.AlignRight}}
	t := f.tabela(mg, cw, cols, estiloTabela{cabFundo: tinta, cabTexto: branco, grade: linha, altLinha: 16})
	t.cabecalho()
	grupos := map[string][]uc.Titulo{}
	var datas []string
	for _, ti := range d.Titulos {
		k := ti.Vencimento.Format("2006-01-02")
		if _, ok := grupos[k]; !ok {
			datas = append(datas, k)
		}
		grupos[k] = append(grupos[k], ti)
	}
	sort.Strings(datas)
	claro := mistura(ac, branco, 0.88)
	for _, k := range datas {
		g := grupos[k]
		sub := decimal.Zero
		for _, ti := range g {
			sub = sub.Add(ti.Saldo())
		}
		vt := g[0].Vencimento
		rot := "Vencimento " + data(vt) + " · " + diaSemana(vt)
		if g[0].Atrasado(d.Referencia) {
			rot += " · VENCIDO"
		}
		if f.cabe(18 + 16) {
			t.cabecalho()
		}
		f.p.FillRect(mg, f.y, cw, 16, claro)
		f.texto(mg+6, f.y+11, neg, 7.6, ac, rot)
		f.p.TextRight(mg+cw-6, f.y+11, neg, 7.6, ac, fmt.Sprintf("%d título(s) · %s", len(g), moeda(sub)))
		f.y += 16
		for _, ti := range g {
			topo := t.linha([]string{"", ti.Documento, ti.Parte.Nome, cnpj(ti.Parte.CNPJCPF), ou(ti.Origem, "—"), ou(ti.PlanoContas, "sem conta"),
				fmt.Sprintf("%d/%d", ti.Parcela, ti.Parcelas), rotuloAprov(ti.Aprovacao), num(ti.Valor, 2), num(ti.Saldo(), 2)}, reg, tinta, nil)
			caixa(f.p, t.xCol(0)+t.ws[0]/2-4, topo+4, tinta)
			if strings.EqualFold(ti.Situacao, "PAGO") {
				marcaOK(f.p, t.xCol(0)+t.ws[0]/2-3, topo+4, verde)
			}
		}
	}
	tot := make([]string, len(cols))
	tot[1] = "TOTAL"
	tot[9] = num(r.aberto.Add(r.vencido), 2)
	tot[8] = num(r.total, 2)
	t.linha(tot, neg, tinta, &fundo)
	f.y += 14

	// Resumos por fornecedor e por plano de contas.
	porF := agrupar(d.Titulos, func(t uc.Titulo) string { return t.Parte.Nome })
	porP := agrupar(d.Titulos, func(t uc.Titulo) string { return ou(t.PlanoContas, "Sem plano de contas") })
	n := max(len(porF), len(porP))
	f.cabe(30 + 15*float64(n))
	half := (cw - 16) / 2
	resumo := func(x float64, titulo string, itens []parValor) {
		y := f.y
		f.texto(x, y+6, neg, 7.6, ac, titulo)
		y += 11
		tt := f.tabela(x, half, []col{{"", 3, pdfkit.AlignLeft}, {"Títulos", 0.6, pdfkit.AlignCenter}, {"Saldo", 1, pdfkit.AlignRight}, {"%", 0.5, pdfkit.AlignRight}},
			estiloTabela{cabFundo: mistura(ac, branco, 0.88), cabTexto: ac, grade: linha})
		f.y = y
		tt.cabecalho()
		soma := decimal.Zero
		for _, it := range itens {
			soma = soma.Add(it.valor)
		}
		for _, it := range itens {
			pc := decimal.Zero
			if soma.IsPositive() {
				pc = it.valor.Div(soma).Mul(decimal.NewFromInt(100))
			}
			tt.linha([]string{it.nome, fmt.Sprint(it.n), num(it.valor, 2), num(pc, 1) + "%"}, reg, tinta, nil)
		}
	}
	y0 := f.y
	resumo(mg, "RESUMO POR FORNECEDOR", porF)
	yf := f.y
	f.y = y0
	resumo(mg+half+16, "RESUMO POR PLANO DE CONTAS", porP)
	f.y = maxF(yf, f.y) + 16

	// Autorização.
	f.cabe(70)
	bw := (cw - 30) / 4
	for i, rot := range []string{"Elaborado por (financeiro)", "Conferido por", "Aprovado por (alçada)", "Autorizado por (diretoria)"} {
		bx := mg + float64(i)*(bw+10)
		f.p.StrokeRect(bx, f.y, bw, 54, 0.6, linha)
		f.p.FillRect(bx, f.y, bw, 3, ac)
		f.texto(bx+8, f.y+14, neg, 6.4, cinza, strings.ToUpper(rot))
		f.p.StrokeLine(bx+8, f.y+38, bx+bw-8, f.y+38, 0.5, tinta)
		f.texto(bx+8, f.y+48, reg, 6.3, cinza, "Data  ____/____/________")
		if i == 0 && d.EmitidoPor != "" {
			f.texto(bx+8, f.y+34, reg, 7.4, tinta, fit(d.EmitidoPor, reg, 7.4, bw-16))
		}
	}
	f.y += 62
	return doc.Render()
}

type parValor struct {
	nome  string
	n     int
	valor decimal.Decimal
}

func agrupar(ts []uc.Titulo, chave func(uc.Titulo) string) []parValor {
	m := map[string]*parValor{}
	var ordem []string
	for _, t := range ts {
		k := chave(t)
		if m[k] == nil {
			m[k] = &parValor{nome: k}
			ordem = append(ordem, k)
		}
		m[k].n++
		m[k].valor = m[k].valor.Add(t.Saldo())
	}
	out := make([]parValor, 0, len(ordem))
	for _, k := range ordem {
		out = append(out, *m[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].valor.GreaterThan(out[j].valor) })
	return out
}

func rotuloAprov(s string) string {
	switch strings.ToUpper(s) {
	case "PENDENTE":
		return "Pendente"
	case "APROVADO":
		return "Aprovado"
	case "REJEITADO":
		return "Rejeitado"
	}
	return ou(s, "—")
}

func diaSemana(t time.Time) string {
	return [...]string{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"}[t.Weekday()]
}

func dia(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
