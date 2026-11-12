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

// Capacidade (CRP) — mapa de calor da carga por centro de trabalho e dia, com
// o total de horas de cada centro contra o disponível e a lista das ordens que
// formam a carga. Paisagem: o eixo é o calendário.

var petroleo = cor{R: 14, G: 98, B: 120}

// corCarga é a escala do mapa de calor.
func corCarga(p float64) (cor, cor) {
	switch {
	case p <= 0:
		return branco, cinza
	case p < 50:
		return cor{R: 220, G: 242, B: 228}, verde
	case p < 85:
		return cor{R: 252, G: 236, B: 190}, cor{R: 130, G: 90, B: 0}
	case p <= 100:
		return cor{R: 253, G: 205, B: 150}, cor{R: 150, G: 70, B: 0}
	}
	return cor{R: 239, G: 120, B: 110}, branco
}

func (Gerador) Capacidade(d uc.Capacidade) []byte {
	doc := pdfkit.NewLandscape()
	const mg = 26.0
	f := novaFolha(doc, mg, 20)
	f.logo = logo(doc, d.Emitente.Logo)
	ac := petroleo
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Capacidade (CRP) · plano %d — %s · %s", d.Plano, d.Nome, emitidoPor(d.Cabecalho)))
	cab := func(cont bool) float64 {
		p := f.p
		x := mg
		if !cont {
			x += desenharLogo(p, f.logo, mg, mg, 66, 30) + 12
		}
		t := "CAPACIDADE DE PRODUÇÃO · CRP"
		if cont {
			t += " (continuação)"
		}
		f.texto(x, mg+13, neg, 13, ac, t)
		f.texto(x, mg+26, reg, 7.6, cinza, fit(fmt.Sprintf("Plano %d — %s · %s", d.Plano, d.Nome, d.Emitente.Nome), reg, 7.6, f.cw()-x-120))
		p.TextRight(f.w-mg, mg+13, neg, 7.6, tinta, "Horas requeridas × disponíveis")
		p.TextRight(f.w-mg, mg+25, reg, 7, cinza, "calendário da máquina, por dia")
		p.FillRect(mg, mg+34, f.cw(), 2, ac)
		return mg + 44
	}
	f.y = cab(false)
	f.continua = func() float64 { return cab(true) }
	cw := f.cw()

	if len(d.Cargas) == 0 {
		f.texto(mg, f.y+14, ita, 9, cinza, "O plano ainda não tem carga calculada: rode o CRP (VPRO0200) ou o pipeline de planejamento.")
		return doc.Render()
	}

	// Agrupa por centro e por dia.
	type dc struct{ req, disp, pct decimal.Decimal }
	carga := map[string]map[string]dc{}
	var centros []string
	ini, fim := d.Cargas[0].Data, d.Cargas[0].Data
	totReq, totDisp := map[string]decimal.Decimal{}, map[string]decimal.Decimal{}
	maxPct, maxCentro, maxDia := decimal.Zero, "", time.Time{}
	sobre := 0
	for _, c := range d.Cargas {
		if carga[c.Centro] == nil {
			carga[c.Centro] = map[string]dc{}
			centros = append(centros, c.Centro)
		}
		carga[c.Centro][c.Data.Format("2006-01-02")] = dc{c.Requerido, c.Disponivel, c.CargaPct}
		totReq[c.Centro] = totReq[c.Centro].Add(c.Requerido)
		totDisp[c.Centro] = totDisp[c.Centro].Add(c.Disponivel)
		if c.Data.Before(ini) {
			ini = c.Data
		}
		if c.Data.After(fim) {
			fim = c.Data
		}
		if c.CargaPct.GreaterThan(maxPct) {
			maxPct, maxCentro, maxDia = c.CargaPct, c.Centro, c.Data
		}
		if c.CargaPct.GreaterThan(decimal.NewFromInt(100)) {
			sobre++
		}
	}
	sort.Strings(centros)
	horas := decimal.Zero
	for _, v := range totReq {
		horas = horas.Add(v)
	}
	cartoes(f, mg, cw, [][3]string{
		{"Centros com carga", fmt.Sprint(len(centros)), fmt.Sprintf("de %s a %s", data(ini), data(fim))},
		{"Horas requeridas", num(horas, 1) + " h", "soma das operações das ordens"},
		{"Maior carga", pct(maxPct), fit(maxCentro+" · "+maxDia.Format("02/01"), reg, 6.4, 120)},
		{"Dias em sobrecarga", fmt.Sprint(sobre), "carga acima de 100%"},
	}, []cor{ac, ac, ambar, verm})
	if sobre == 0 {
		f.p.FillRect(mg, f.y-4, cw, 16, mistura(verde, branco, 0.88))
		marcaOK(f.p, mg+8, f.y-0.5, verde)
		f.texto(mg+22, f.y+7, neg, 7.6, verde, "Plano viável em capacidade: nenhum centro passa de 100% em nenhum dia.")
		f.y += 20
	}

	// Mapa de calor em blocos de até 20 dias.
	var dias []time.Time
	for dt := dia(ini); !dt.After(dia(fim)); dt = dt.AddDate(0, 0, 1) {
		dias = append(dias, dt)
	}
	const porBloco = 20
	rotW := 150.0
	for b := 0; b < len(dias); b += porBloco {
		bloco := dias[b:min(b+porBloco, len(dias))]
		cel := minF((cw-rotW-70)/float64(len(bloco)), 64)
		altLinha := 26.0
		f.cabe(30 + altLinha*float64(len(centros)))
		p := f.p
		f.texto(mg, f.y+6, neg, 8.5, ac, "MAPA DE CARGA POR CENTRO E DIA")
		f.y += 12
		// Cabeçalho dos dias.
		for i, dt := range bloco {
			x := mg + rotW + float64(i)*cel
			fd := mistura(ac, branco, 0.88)
			if dt.Weekday() == time.Saturday || dt.Weekday() == time.Sunday {
				fd = fundo
			}
			p.FillRect(x+1, f.y, cel-2, 22, fd)
			p.TextCenter(x+cel/2, f.y+9, neg, 7.2, ac, dt.Format("02/01"))
			p.TextCenter(x+cel/2, f.y+18, reg, 5.8, cinza, diaSemana(dt)[:3])
		}
		p.TextCenter(mg+rotW+float64(len(bloco))*cel+35, f.y+13, neg, 6.4, ac, "TOTAL")
		f.y += 25
		for _, c := range centros {
			if f.cabe(altLinha) {
				p = f.p
			}
			p = f.p
			f.texto(mg, f.y+11, neg, 7.6, tinta, fit(c, neg, 7.6, rotW-6))
			f.texto(mg, f.y+20, reg, 6, cinza, fit(strings.Join(d.Maquinas[c], ", "), reg, 6, rotW-6))
			for i, dt := range bloco {
				x := mg + rotW + float64(i)*cel
				v, ok := carga[c][dt.Format("2006-01-02")]
				if !ok {
					p.StrokeRect(x+1, f.y+1, cel-2, altLinha-2, 0.3, linha)
					continue
				}
				bg, fg := corCarga(v.pct.InexactFloat64())
				p.FillRect(x+1, f.y+1, cel-2, altLinha-2, bg)
				p.TextCenter(x+cel/2, f.y+12, neg, 7.4, fg, num(v.pct, 0)+"%")
				p.TextCenter(x+cel/2, f.y+20.5, reg, 5.6, fg, num(v.req, 1)+"/"+num(v.disp, 1)+"h")
			}
			tx := mg + rotW + float64(len(bloco))*cel + 4
			p.FillRect(tx, f.y+1, 62, altLinha-2, fundo)
			p.TextCenter(tx+31, f.y+12, neg, 7.4, tinta, num(totReq[c], 1)+" h")
			p.TextCenter(tx+31, f.y+20.5, reg, 5.6, cinza, "de "+num(totDisp[c], 1)+" h")
			f.y += altLinha
		}
		f.y += 6
		// Legenda.
		lx := mg
		for _, l := range []struct {
			v float64
			t string
		}{{10, "até 50%"}, {60, "50 a 85%"}, {90, "85 a 100%"}, {120, "acima de 100% (sobrecarga)"}} {
			bg, _ := corCarga(l.v)
			f.p.FillRect(lx, f.y, 14, 9, bg)
			f.texto(lx+18, f.y+7.5, reg, 6.6, cinza, l.t)
			lx += 30 + larg(reg, 6.6, l.t)
		}
		f.texto(lx+10, f.y+7.5, ita, 6.6, cinza, "Cada célula: carga % e horas requeridas / disponíveis no dia.")
		f.y += 20
	}

	// Barras por centro.
	f.cabe(30 + 16*float64(len(centros)))
	f.texto(mg, f.y+6, neg, 8.5, ac, "OCUPAÇÃO NO PERÍODO (dias com carga)")
	f.y += 14
	maxH := decimal.Zero
	for _, c := range centros {
		if totDisp[c].GreaterThan(maxH) {
			maxH = totDisp[c]
		}
	}
	bx, bw := mg+rotW, cw*0.55
	for _, c := range centros {
		p := f.p
		f.texto(mg, f.y+9, reg, 7.4, tinta, fit(c, reg, 7.4, rotW-6))
		disp := totDisp[c].Div(maxH).InexactFloat64() * bw
		req := totReq[c].Div(maxH).InexactFloat64() * bw
		p.FillRect(bx, f.y+2, disp, 10, fundo)
		p.StrokeRect(bx, f.y+2, disp, 10, 0.4, linha)
		cr := ac
		if totReq[c].GreaterThan(totDisp[c]) {
			cr = verm
		}
		p.FillRect(bx, f.y+2, minF(req, bw), 10, cr)
		ocup := decimal.Zero
		if totDisp[c].IsPositive() {
			ocup = totReq[c].Div(totDisp[c]).Mul(decimal.NewFromInt(100))
		}
		f.texto(bx+bw+10, f.y+10, neg, 7.4, tinta, fmt.Sprintf("%s h de %s h · %s", num(totReq[c], 1), num(totDisp[c], 1), pct(ocup)))
		f.y += 16
	}
	f.y += 10

	// Ordens que formam a carga.
	if len(d.Ordens) > 0 {
		f.cabe(40)
		f.texto(mg, f.y+6, neg, 8.5, ac, "ORDENS QUE FORMAM A CARGA")
		f.y += 12
		t := f.tabela(mg, cw, []col{{"Centro", 1.5, pdfkit.AlignLeft}, {"Ordem", 0.7, pdfkit.AlignCenter}, {"Item", 1.1, pdfkit.AlignLeft},
			{"Operação", 2, pdfkit.AlignLeft}, {"Data planejada", 0.9, pdfkit.AlignCenter}, {"Horas (prep. + exec.)", 1.1, pdfkit.AlignRight}},
			estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha, altLinha: 14})
		t.cabecalho()
		ant := ""
		for _, o := range d.Ordens {
			centro := o.Centro
			if centro == ant {
				centro = ""
			}
			ant = o.Centro
			t.linha([]string{centro, o.Ordem, o.Item, o.Operacao, data(o.Data), num(o.Horas, 2) + " h"}, reg, tinta, nil)
		}
	}
	return doc.Render()
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
