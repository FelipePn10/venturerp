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

// Plano de materiais (MRP) — o resultado do cálculo para o planejador: de onde
// veio a demanda, a árvore de explosão com o que cada nível pede, a linha do
// tempo das ordens, as compras com fornecedor e valor estimado, a fabricação
// com roteiro e horas, e as exceções.

var (
	indigo   = cor{R: 55, G: 48, B: 163}
	corFabr  = cor{R: 37, G: 99, B: 235}
	corCompr = cor{R: 217, G: 119, B: 6}
)

func tipoSug(t string) (string, string, cor) {
	if t == "COMPRA" || t == "PURCHASE" {
		return "C", "Compra", corCompr
	}
	return "F", "Fabricação", corFabr
}

func (Gerador) PlanoMateriais(d uc.PlanoMateriais) []byte {
	doc := pdfkit.New()
	const mg = 28.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := indigo
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Plano de materiais · plano %d — %s · %s", d.Plano, d.Nome, emitidoPor(d.Cabecalho)))
	f.continua = func() float64 {
		f.texto(mg, mg+10, neg, 8.5, ac, fmt.Sprintf("PLANO DE MATERIAIS · PLANO %d (continuação)", d.Plano))
		f.p.FillRect(mg, mg+14, f.cw(), 1.2, ac)
		return mg + 24
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho: faixa índigo à esquerda com o título; logo e empresa à direita.
	p.FillRect(mg, mg, cw*0.62, 66, ac)
	f.texto(mg+14, mg+18, neg, 7.2, mistura(ac, branco, 0.6), "MRP · PLANEJAMENTO DE NECESSIDADES DE MATERIAIS")
	f.texto(mg+14, mg+40, neg, 18, branco, "Plano de materiais")
	f.texto(mg+14, mg+56, reg, 8, mistura(ac, branco, 0.8), fit(fmt.Sprintf("Plano %d — %s", d.Plano, d.Nome), reg, 8, cw*0.62-28))
	rx := mg + cw*0.62 + 12
	rw := f.w - mg - rx
	lw := desenharLogo(p, f.logo, rx, mg+2, 70, 30)
	f.texto(rx+lw+8, mg+14, neg, 7.4, tinta, "")
	for i, l := range quebrar(d.Emitente.Nome, neg, 7, rw-lw-8) {
		if i < 3 {
			f.texto(rx+lw+8, mg+11+float64(i)*8.5, neg, 7, tinta, l)
		}
	}
	st, cs := statusMRP(d.Status)
	selo(p, rx, mg+40, st, cs, branco, 6.8)
	f.texto(rx, mg+62, reg, 7, cinza, "Calculado em "+dataHoraP(d.CalculadoEm))
	f.y = mg + 80

	// Indicadores.
	var fab, com, firm int
	valorCompra := decimal.Zero
	for _, s := range d.Sugestoes {
		if k, _, _ := tipoSug(s.Tipo); k == "C" {
			com++
			valorCompra = valorCompra.Add(s.Quantidade.Mul(s.PrecoReferencia))
		} else {
			fab++
		}
		if s.Firmada {
			firm++
		}
	}
	cartoes(f, mg, cw, [][3]string{
		{"Demandas", fmt.Sprint(len(d.Demandas)), "pedidos e demandas lidos"},
		{"Fabricação", fmt.Sprint(fab), "ordens sugeridas"},
		{"Compra", fmt.Sprint(com), "ordens sugeridas"},
		{"Compras estimadas", moeda(valorCompra.Round(2)), "pela tabela de preço"},
		{"Firmadas", fmt.Sprintf("%d de %d", firm, len(d.Sugestoes)), "viraram OF/pedido"},
	}, []cor{ac, corFabr, corCompr, corCompr, verde})

	sec := func(n int, t, nota string) {
		f.cabe(34)
		f.p.FillRect(mg, f.y, 18, 18, ac)
		f.p.TextCenter(mg+9, f.y+12.5, neg, 9, branco, fmt.Sprint(n))
		f.texto(mg+26, f.y+12.5, neg, 9.5, tinta, t)
		if nota != "" {
			f.texto(mg+32+larg(neg, 9.5, t), f.y+12.5, ita, 7, cinza, nota)
		}
		f.y += 26
	}

	// 1. Demanda.
	sec(1, "Demanda que originou o plano", "o MRP parte destas necessidades")
	td := f.tabela(mg, cw, []col{{"Origem", 1.35, pdfkit.AlignLeft}, {"Documento", 0.9, pdfkit.AlignLeft}, {"Cliente / motivo", 1.9, pdfkit.AlignLeft},
		{"Item", 0.9, pdfkit.AlignLeft}, {"Descrição", 2.3, pdfkit.AlignLeft}, {"Qtde", 0.6, pdfkit.AlignRight}, {"Un", 0.35, pdfkit.AlignCenter}, {"Data", 0.9, pdfkit.AlignCenter}},
		estiloTabela{cabFundo: mistura(ac, branco, 0.88), cabTexto: ac, grade: linha})
	td.cabecalho()
	if len(d.Demandas) == 0 {
		td.linha([]string{"", "", "Sem demanda em aberto para o plano."}, ita, cinza, nil)
	}
	for _, x := range d.Demandas {
		td.linha([]string{x.Origem, x.Documento, x.Cliente, x.Codigo, x.Descricao, qtd(x.Quantidade), x.UM, data(x.Data)}, reg, tinta, nil)
	}
	f.y += 14

	// 2. Árvore de explosão.
	sec(2, "Árvore de explosão", "cada nível com o que pede ao nível de baixo")
	f.arvoreMRP(d.Sugestoes, ac)
	f.y += 10

	// 3. Linha do tempo.
	if len(d.Sugestoes) > 0 {
		sec(3, "Linha do tempo das ordens", "do início (liberação) à data de necessidade")
		f.linhaDoTempoMRP(d.Sugestoes)
		f.y += 10
	}

	// 4. Compras.
	sec(4, "Compras sugeridas", "fornecedor preferencial e preço da tabela de compra")
	tc := f.tabela(mg, cw, []col{{"Nº", 0.45, pdfkit.AlignCenter}, {"Item", 0.75, pdfkit.AlignLeft}, {"Descrição", 2.1, pdfkit.AlignLeft},
		{"Qtde", 0.85, pdfkit.AlignRight}, {"Un", 0.35, pdfkit.AlignCenter}, {"Fornecedor", 1.55, pdfkit.AlignLeft}, {"Preço ref.", 0.75, pdfkit.AlignRight},
		{"Valor est.", 0.8, pdfkit.AlignRight}, {"Pedir até", 0.9, pdfkit.AlignCenter}, {"Gerou", 0.55, pdfkit.AlignCenter}},
		estiloTabela{cabFundo: corCompr, cabTexto: branco, grade: linha})
	tc.cabecalho()
	tot := decimal.Zero
	for _, s := range d.Sugestoes {
		if k, _, _ := tipoSug(s.Tipo); k != "C" {
			continue
		}
		v := s.Quantidade.Mul(s.PrecoReferencia).Round(2)
		tot = tot.Add(v)
		tc.linha([]string{fmt.Sprint(s.Numero), s.Item, s.Descricao, qtd(s.Quantidade), s.UM, ou(s.Fornecedor, "sem fornecedor"),
			numSe(s.PrecoReferencia, 4), numSe(v, 2), data(s.Inicio), ou(s.Gerou, "—")}, reg, tinta, nil)
	}
	tc.linha([]string{"", "", "Total estimado", "", "", "", "", num(tot, 2), "", ""}, neg, tinta, &fundo)
	f.y += 14

	// 5. Fabricação.
	sec(5, "Fabricação sugerida", "roteiro padrão e horas previstas da OF")
	tf := f.tabela(mg, cw, []col{{"Nº", 0.45, pdfkit.AlignCenter}, {"Item", 0.85, pdfkit.AlignLeft}, {"Descrição", 2.1, pdfkit.AlignLeft},
		{"Qtde", 0.6, pdfkit.AlignRight}, {"Para", 0.85, pdfkit.AlignLeft}, {"Roteiro", 0.8, pdfkit.AlignLeft}, {"Horas", 0.55, pdfkit.AlignRight},
		{"Início", 0.9, pdfkit.AlignCenter}, {"Necessidade", 0.95, pdfkit.AlignCenter}, {"Gerou", 0.55, pdfkit.AlignCenter}},
		estiloTabela{cabFundo: corFabr, cabTexto: branco, grade: linha})
	tf.cabecalho()
	for _, s := range d.Sugestoes {
		if k, _, _ := tipoSug(s.Tipo); k != "F" {
			continue
		}
		tf.linha([]string{fmt.Sprint(s.Numero), s.Item, s.Descricao, qtd(s.Quantidade), ou(s.Pai, "pedido"), ou(s.Roteiro, "sem roteiro"),
			numSe(s.HorasPrevistas, 2), data(s.Inicio), data(s.Necessidade), ou(s.Gerou, "—")}, reg, tinta, nil)
	}
	f.y += 14

	// 6. Exceções.
	sec(6, "Exceções e mensagens", "")
	if d.Erros != "" {
		f.cabe(22)
		f.p.FillRect(mg, f.y, cw, 18, mistura(verm, branco, 0.88))
		f.texto(mg+8, f.y+12, neg, 7.4, verm, fit("Erro no cálculo: "+d.Erros, neg, 7.4, cw-16))
		f.y += 24
	}
	if len(d.Excecoes) == 0 && d.Erros == "" {
		f.cabe(22)
		f.p.FillRect(mg, f.y, cw, 18, mistura(verde, branco, 0.88))
		marcaOK(f.p, mg+8, f.y+5, verde)
		f.texto(mg+22, f.y+12, neg, 7.6, verde, "Nenhuma exceção: todas as necessidades foram atendidas dentro do prazo.")
		f.y += 24
	}
	for _, e := range d.Excecoes {
		f.cabe(16)
		c := ambar
		if e.Tratada {
			c = cinza
		}
		f.p.FillRect(mg, f.y+3, 3, 10, c)
		f.texto(mg+8, f.y+11, neg, 7.4, tinta, e.Item+" · "+e.Tipo)
		f.texto(mg+140, f.y+11, reg, 7.2, tinta, fit(e.Descricao, reg, 7.2, cw-150))
		f.y += 15
	}
	return doc.Render()
}

func statusMRP(s string) (string, cor) {
	switch s {
	case "COMPLETED":
		return "CALCULADO", verde
	case "COMPLETED_WITH_ERRORS":
		return "CALCULADO COM ERROS", ambar
	case "ERROR":
		return "ERRO NO CÁLCULO", verm
	case "RUNNING":
		return "CALCULANDO", azulVenda
	}
	return strings.ToUpper(ou(s, "não calculado")), cinza
}

func dataHoraP(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return dataHora(*t)
}

func numSe(v decimal.Decimal, casas int32) string {
	if !v.IsPositive() {
		return "—"
	}
	return num(v, casas)
}

// arvoreMRP desenha a explosão: raízes (sem pai) e, abaixo de cada nó, as
// sugestões cujo pai é o item do nó, com conectores.
func (f *folha) arvoreMRP(ss []uc.SugestaoPlano, ac cor) {
	filhos := map[string][]uc.SugestaoPlano{}
	var raizes []uc.SugestaoPlano
	for _, s := range ss {
		if s.Pai == "" {
			raizes = append(raizes, s)
		} else {
			filhos[s.Pai] = append(filhos[s.Pai], s)
		}
	}
	sort.SliceStable(raizes, func(i, j int) bool { return raizes[i].Necessidade.Before(raizes[j].Necessidade) })
	mg, cw := f.mg, f.cw()
	const h = 17.0
	// Cabeçalho das colunas da árvore.
	f.p.FillRect(mg, f.y, cw, 13, mistura(ac, branco, 0.9))
	for _, c := range []struct {
		x  float64
		t  string
		al pdfkit.Align
	}{{mg + 6, "ESTRUTURA", pdfkit.AlignLeft}, {mg + cw*0.56, "QTDE", pdfkit.AlignRight}, {mg + cw*0.66, "INÍCIO", pdfkit.AlignLeft},
		{mg + cw*0.76, "NECESSIDADE", pdfkit.AlignLeft}, {mg + cw - 6, "GEROU", pdfkit.AlignRight}} {
		if c.al == pdfkit.AlignRight {
			f.p.TextRight(c.x, f.y+9, neg, 6, ac, c.t)
		} else {
			f.p.Text(c.x, f.y+9, neg, 6, ac, c.t)
		}
	}
	f.y += 15
	var desenhar func(s uc.SugestaoPlano, nivel int, visitados map[int64]bool)
	desenhar = func(s uc.SugestaoPlano, nivel int, visitados map[int64]bool) {
		if visitados[s.Codigo] || nivel > 8 {
			return
		}
		visitados[s.Codigo] = true
		f.cabe(h)
		p := f.p
		x := mg + 6 + float64(nivel)*16
		if nivel > 0 {
			p.StrokeLine(x-10, f.y-4, x-10, f.y+h/2, 0.6, linha)
			p.StrokeLine(x-10, f.y+h/2, x-2, f.y+h/2, 0.6, linha)
		}
		k, _, c := tipoSug(s.Tipo)
		p.FillRect(x, f.y+3.5, 11, 10, c)
		p.TextCenter(x+5.5, f.y+11, neg, 6.6, branco, k)
		nome := s.Item + "  " + s.Descricao
		ft := reg
		if nivel == 0 {
			ft = neg
		}
		p.Text(x+16, f.y+11, ft, 7.6, tinta, fit(nome, ft, 7.6, mg+cw*0.56-60-(x+16)))
		p.TextRight(mg+cw*0.56, f.y+11, neg, 7.6, tinta, qtd(s.Quantidade)+" "+s.UM)
		p.Text(mg+cw*0.66, f.y+11, reg, 7.4, cinza, data(s.Inicio))
		p.Text(mg+cw*0.76, f.y+11, reg, 7.4, tinta, data(s.Necessidade))
		g := ou(s.Gerou, "sugestão")
		cg := verde
		if s.Gerou == "" {
			cg = cinza
		}
		p.TextRight(mg+cw-6, f.y+11, neg, 7.4, cg, g)
		p.StrokeDashedLine(mg, f.y+h, mg+cw, f.y+h, 0.3, 1, 2, linha)
		f.y += h
		for _, fi := range filhos[s.Item] {
			desenhar(fi, nivel+1, visitados)
		}
	}
	vis := map[int64]bool{}
	for _, r := range raizes {
		desenhar(r, 0, vis)
	}
	// Sugestões cujo pai não está no plano (ex.: pai já em estoque).
	for _, s := range ss {
		if !vis[s.Codigo] {
			desenhar(s, 0, vis)
		}
	}
	f.y += 4
	f.p.FillRect(mg, f.y, 9, 8, corFabr)
	f.texto(mg+13, f.y+7, reg, 6.6, cinza, "F = fabricação (vira OF)")
	f.p.FillRect(mg+110, f.y, 9, 8, corCompr)
	f.texto(mg+123, f.y+7, reg, 6.6, cinza, "C = compra (vira pedido de compra)")
	f.y += 12
}

// linhaDoTempoMRP desenha uma barra por sugestão entre o início e a necessidade.
func (f *folha) linhaDoTempoMRP(ss []uc.SugestaoPlano) {
	ini, fim := ss[0].Inicio, ss[0].Necessidade
	for _, s := range ss {
		if s.Inicio.Before(ini) {
			ini = s.Inicio
		}
		if s.Necessidade.After(fim) {
			fim = s.Necessidade
		}
	}
	ini, fim = dia(ini), dia(fim).AddDate(0, 0, 1)
	dias := int(fim.Sub(ini).Hours()/24) + 0
	if dias < 1 {
		dias = 1
	}
	mg, cw := f.mg, f.cw()
	rot := 150.0
	gx, gw := mg+rot, cw-rot
	h := 11.0
	altura := 22 + h*float64(len(ss))
	f.cabe(altura + 8)
	p := f.p
	passo := gw / float64(dias)
	// Eixo de datas: marca a cada n dias para não amontoar.
	cada := 1
	for float64(cada)*passo < 34 {
		cada++
	}
	for i := 0; i <= dias; i++ {
		x := gx + float64(i)*passo
		dt := ini.AddDate(0, 0, i)
		if dt.Weekday() == time.Saturday || dt.Weekday() == time.Sunday {
			if i < dias {
				p.FillRect(x, f.y+14, passo, altura-14, fundo)
			}
		}
		if i%cada == 0 && i < dias {
			p.Text(x+1, f.y+9, reg, 5.8, cinza, dt.Format("02/01"))
			p.StrokeLine(x, f.y+12, x, f.y+altura, 0.3, linha)
		}
	}
	y := f.y + 18
	for _, s := range ss {
		_, _, c := tipoSug(s.Tipo)
		p.Text(mg, y+8, reg, 6.6, tinta, fit(s.Item+" · "+qtd(s.Quantidade)+" "+s.UM, reg, 6.6, rot-6))
		x0 := gx + passo*dia(s.Inicio).Sub(ini).Hours()/24
		x1 := gx + passo*(dia(s.Necessidade).Sub(ini).Hours()/24+1)
		p.FillRect(x0, y+1.5, maxF(x1-x0, 3), h-3, c)
		y += h
	}
	f.y += altura + 4
}
