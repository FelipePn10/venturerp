package documentos

import (
	"fmt"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Apuração do custo de produção — o fechamento das ordens para a
// controladoria: resumo das OFs e, para cada uma, a composição do custo em
// barra, o real contra o padrão por componente, os materiais consumidos com
// o custo e as horas previstas e reais de cada etapa.

var (
	roxo       = cor{R: 91, G: 33, B: 182}
	corMat     = cor{R: 91, G: 33, B: 182}
	corMaq     = cor{R: 14, G: 116, B: 144}
	corMO      = cor{R: 217, G: 119, B: 6}
	corOver    = cor{R: 100, G: 116, B: 139}
	corTerc    = cor{R: 219, G: 39, B: 119}
	cemDecimal = decimal.NewFromInt(100)
)

func variacao(real, padrao decimal.Decimal) (string, cor) {
	if !padrao.IsPositive() {
		if real.IsPositive() {
			return "sem padrão", ambar
		}
		return "—", cinza
	}
	v := real.Sub(padrao).Div(padrao).Mul(cemDecimal)
	c := verde
	if v.GreaterThan(decimal.NewFromInt(5)) {
		c = verm
	} else if v.GreaterThan(decimal.Zero) {
		c = ambar
	}
	s := num(v, 1) + "%"
	if v.IsPositive() {
		s = "+" + s
	}
	return s, c
}

func (Gerador) CustoProducao(d uc.CustoProducao) []byte {
	doc := pdfkit.New()
	const mg = 28.0
	f := novaFolha(doc, mg, 22)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := roxo
	rodapeSimples(doc, mg, ac, "Apuração do custo de produção · "+d.Filtro+" · "+emitidoPor(d.Cabecalho))
	f.continua = func() float64 {
		f.p.FillRect(mg, mg, 5, 18, ac)
		f.texto(mg+12, mg+13, neg, 8.5, ac, "APURAÇÃO DO CUSTO DE PRODUÇÃO (continuação)")
		return mg + 28
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho: título grande à esquerda com filete roxo; logo à direita.
	p.FillRect(mg, mg, 6, 58, ac)
	f.texto(mg+16, mg+14, neg, 7.2, ac, "CONTROLADORIA · CUSTO REAL DAS ORDENS")
	f.texto(mg+16, mg+36, neg, 18, tinta, "Apuração do custo de produção")
	f.texto(mg+16, mg+51, reg, 7.6, cinza, fit(d.Emitente.Nome+" · "+d.Filtro, reg, 7.6, cw-140))
	desenharLogo(p, f.logo, f.w-mg-100, mg+6, 100, 44)
	f.y = mg + 72

	if len(d.Ordens) == 0 {
		f.texto(mg, f.y+12, ita, 8.5, cinza, "Nenhuma ordem com custo apurado no filtro.")
		return doc.Render()
	}

	var tReal, tPad, tMat, tMaq, tMO, tOver, tRealComPadrao decimal.Decimal
	for _, o := range d.Ordens {
		tReal, tPad = tReal.Add(o.Total), tPad.Add(o.PadraoTotal)
		if o.PadraoTotal.IsPositive() {
			tRealComPadrao = tRealComPadrao.Add(o.Total)
		}
		tMat, tMaq, tMO, tOver = tMat.Add(o.Material), tMaq.Add(o.Maquina), tMO.Add(o.MaoDeObra), tOver.Add(o.Overhead)
	}
	// A variação só compara ordens que têm custo-padrão.
	vs, vc := variacao(tRealComPadrao, tPad)
	cartoes(f, mg, cw, [][3]string{
		{"Ordens apuradas", fmt.Sprint(len(d.Ordens)), "encerradas com custo"},
		{"Custo real", moeda(tReal), "material + máquina + MO + overhead"},
		{"Custo padrão", moeda(tPad), "pelo custo-padrão vigente"},
		{"Variação", vs, "só ordens com custo-padrão"},
	}, []cor{ac, ac, cinza, vc})
	if !tMO.IsPositive() && !tOver.IsPositive() {
		f.p.FillRect(mg, f.y-4, cw, 18, mistura(ambar, branco, 0.85))
		f.texto(mg+8, f.y+8, neg, 7.4, cor{R: 130, G: 80, B: 0},
			"Mão de obra e overhead saíram zerados: cadastre o custo/hora dos centros de trabalho e os centros de custo (VCUS0100 / VCTB0102).")
		f.y += 22
	}

	// Resumo.
	f.texto(mg, f.y+6, neg, 8.5, ac, "RESUMO DAS ORDENS")
	f.y += 11
	rs := f.tabela(mg, cw, []col{{"OF", 0.45, pdfkit.AlignCenter}, {"Item", 0.95, pdfkit.AlignLeft}, {"Qtde", 0.5, pdfkit.AlignRight},
		{"Material", 0.9, pdfkit.AlignRight}, {"Máquina", 0.75, pdfkit.AlignRight}, {"M. obra", 0.7, pdfkit.AlignRight},
		{"Overhead", 0.7, pdfkit.AlignRight}, {"Total real", 0.95, pdfkit.AlignRight}, {"Unit. real", 0.8, pdfkit.AlignRight},
		{"Unit. padrão", 0.9, pdfkit.AlignRight}, {"Var.", 0.75, pdfkit.AlignRight}},
		estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha, altLinha: 15})
	rs.cabecalho()
	for _, o := range d.Ordens {
		up := decimal.Zero
		if o.Quantidade.IsPositive() {
			up = o.PadraoTotal.Div(o.Quantidade)
		}
		v, c := variacao(o.Total, o.PadraoTotal)
		topo := rs.linha([]string{fmt.Sprint(o.Ordem), o.Item, qtd(o.Quantidade), num(o.Material, 2), num(o.Maquina, 2), num(o.MaoDeObra, 2),
			num(o.Overhead, 2), num(o.Total, 2), num(o.Unitario, 2), num(up, 2), ""}, reg, tinta, nil)
		f.p.TextRight(rs.xCol(11)-3.5, topo+10.3, neg, 7.4, c, v)
	}
	rs.linha([]string{"", "Total", "", num(tMat, 2), num(tMaq, 2), num(tMO, 2), num(tOver, 2), num(tReal, 2), "", "", vs}, neg, tinta, &fundo)
	f.y += 18

	// Detalhe por OF.
	for _, o := range d.Ordens {
		f.cabe(150)
		p = f.p
		p.FillRect(mg, f.y, cw, 26, mistura(ac, branco, 0.9))
		p.FillRect(mg, f.y, 52, 26, ac)
		p.TextCenter(mg+26, f.y+11, neg, 6.4, mistura(ac, branco, 0.65), "OF")
		p.TextCenter(mg+26, f.y+21.5, neg, 10, branco, fmt.Sprint(o.Ordem))
		f.texto(mg+62, f.y+12, neg, 9, tinta, fit(o.Item+" · "+o.Descricao, neg, 9, cw-230))
		f.texto(mg+62, f.y+22, reg, 7, cinza, fmt.Sprintf("%s %s produzidos · início %s · fim %s · apurado %s", qtd(o.Quantidade), o.UM, dataP(o.Inicio), dataP(o.Fim), dataHoraP(o.Apurado)))
		p.TextRight(f.w-mg-10, f.y+12, neg, 10, ac, moeda(o.Unitario)+" / "+o.UM)
		p.TextRight(f.w-mg-10, f.y+22, reg, 7, cinza, "custo total "+moeda(o.Total))
		f.y += 34

		// Barra de composição.
		partes := []struct {
			n string
			v decimal.Decimal
			c cor
		}{{"Material", o.Material, corMat}, {"Máquina", o.Maquina, corMaq}, {"Mão de obra", o.MaoDeObra, corMO}, {"Overhead", o.Overhead, corOver}, {"Terceiros", o.Terceiros, corTerc}}
		bw := cw * 0.5
		x := mg
		if o.Total.IsPositive() {
			for _, pt := range partes {
				w := pt.v.Div(o.Total).InexactFloat64() * bw
				if w > 0 {
					p.FillRect(x, f.y, w, 14, pt.c)
					x += w
				}
			}
		}
		ly := f.y + 26
		for _, pt := range partes {
			pc := decimal.Zero
			if o.Total.IsPositive() {
				pc = pt.v.Div(o.Total).Mul(cemDecimal)
			}
			p.FillRect(mg, ly-7, 8, 8, pt.c)
			f.texto(mg+12, ly, reg, 7.2, tinta, pt.n)
			p.TextRight(mg+130, ly, reg, 7.2, tinta, moeda(pt.v))
			p.TextRight(mg+170, ly, neg, 7.2, cinza, num(pc, 1)+"%")
			ly += 11
		}
		// Real × padrão.
		rx := mg + bw + 20
		rw := cw - bw - 20
		f.texto(rx, f.y+9, neg, 7.2, ac, "REAL × PADRÃO")
		ty := f.y + 16
		p.FillRect(rx, ty, rw, 12, fundo)
		for i, h := range []string{"Componente", "Real", "Padrão", "Variação"} {
			xx := []float64{rx + 4, rx + rw*0.55, rx + rw*0.78, rx + rw - 4}[i]
			if i == 0 {
				p.Text(xx, ty+8.5, neg, 6, cinza, h)
			} else {
				p.TextRight(xx, ty+8.5, neg, 6, cinza, h)
			}
		}
		ty += 12
		linhas := []struct {
			n    string
			r, s decimal.Decimal
		}{{"Material", o.Material, o.PadraoMaterial}, {"Máquina", o.Maquina, o.PadraoMaquina}, {"Mão de obra", o.MaoDeObra, o.PadraoMaoDeObra},
			{"Overhead", o.Overhead, o.PadraoOverhead}, {"Total", o.Total, o.PadraoTotal}}
		for _, l := range linhas {
			ft := reg
			if l.n == "Total" {
				ft = neg
			}
			v, c := variacao(l.r, l.s)
			p.Text(rx+4, ty+9, ft, 7.2, tinta, l.n)
			p.TextRight(rx+rw*0.55, ty+9, ft, 7.2, tinta, num(l.r, 2))
			p.TextRight(rx+rw*0.78, ty+9, ft, 7.2, cinza, num(l.s, 2))
			p.TextRight(rx+rw-4, ty+9, neg, 7.2, c, v)
			p.StrokeLine(rx, ty+12, rx+rw, ty+12, 0.3, linha)
			ty += 12
		}
		f.y = maxF(ly, ty) + 8

		// Materiais consumidos.
		if len(o.Consumos) > 0 {
			tm := f.tabela(mg, cw, []col{{"Material consumido", 1, pdfkit.AlignLeft}, {"Descrição", 2.6, pdfkit.AlignLeft}, {"Qtde", 0.8, pdfkit.AlignRight},
				{"Un", 0.35, pdfkit.AlignCenter}, {"Custo unit.", 0.8, pdfkit.AlignRight}, {"Custo", 0.85, pdfkit.AlignRight}, {"% material", 0.7, pdfkit.AlignRight}},
				estiloTabela{cabFundo: mistura(ac, branco, 0.88), cabTexto: ac, grade: linha, altLinha: 14})
			tm.cabecalho()
			for _, c := range o.Consumos {
				un, pc := decimal.Zero, decimal.Zero
				if c.Quantidade.IsPositive() {
					un = c.Custo.Div(c.Quantidade)
				}
				if o.Material.IsPositive() {
					pc = c.Custo.Div(o.Material).Mul(cemDecimal)
				}
				tm.linha([]string{c.Codigo, c.Descricao, qtd(c.Quantidade), c.UM, num(un, 4), num(c.Custo, 2), num(pc, 1) + "%"}, reg, tinta, nil)
			}
			f.y += 6
		}
		// Etapas.
		if len(o.Etapas) > 0 {
			te := f.tabela(mg, cw, []col{{"Etapa", 0.45, pdfkit.AlignCenter}, {"Operação", 1.8, pdfkit.AlignLeft}, {"Centro", 1.4, pdfkit.AlignLeft},
				{"Horas previstas", 0.9, pdfkit.AlignRight}, {"Horas reais", 0.9, pdfkit.AlignRight}, {"Eficiência", 0.8, pdfkit.AlignRight}, {"Situação", 0.9, pdfkit.AlignLeft}},
				estiloTabela{cabFundo: mistura(ac, branco, 0.88), cabTexto: ac, grade: linha, altLinha: 14})
			te.cabecalho()
			for _, e := range o.Etapas {
				ef := "—"
				if e.Real.IsPositive() {
					ef = num(e.Prevista.Div(e.Real).Mul(cemDecimal), 0) + "%"
				}
				te.linha([]string{fmt.Sprint(e.Seq), e.Operacao, e.Centro, num(e.Prevista, 2), num(e.Real, 2), ef, rotuloEtapa(e.Situacao)}, reg, tinta, nil)
			}
		}
		f.y += 18
	}
	return doc.Render()
}

func rotuloEtapa(s string) string {
	switch s {
	case "DONE":
		return "Concluída"
	case "IN_PROGRESS":
		return "Em execução"
	case "PENDING":
		return "Pendente"
	case "SKIPPED":
		return "Pulada"
	}
	return s
}
