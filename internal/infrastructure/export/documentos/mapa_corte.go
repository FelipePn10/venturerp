package documentos

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Mapa de corte — o documento da seccionadora/laser/serra: um resumo do plano
// (material, parâmetros de corte, consumo e aproveitamento) e uma página por
// padrão com o desenho em escala, cotas, cor por peça, a lista do que sai de
// cada chapa ou barra e o controle de execução do operador.

// MapaCorte implementa cutting_plan_uc.GeradorMapaCorte.
type MapaCorte struct{}

var _ cutting_plan_uc.GeradorMapaCorte = MapaCorte{}

var aco = cor{R: 71, G: 85, B: 105}

var paletaPecas = []cor{
	{R: 191, G: 219, B: 254}, {R: 254, G: 215, B: 170}, {R: 187, G: 247, B: 208}, {R: 233, G: 213, B: 255},
	{R: 254, G: 202, B: 202}, {R: 165, G: 243, B: 252}, {R: 254, G: 240, B: 138}, {R: 251, G: 207, B: 232},
	{R: 199, G: 210, B: 254}, {R: 217, G: 249, B: 157},
}

func rotuloTipoCorte(t entity.CutType) string {
	switch t {
	case entity.CutTypeLinear1D:
		return "Linear (barras e perfis)"
	case entity.CutTypeGuillotine2D:
		return "Chapa — guilhotinado 2D"
	case entity.CutTypeTrueShape2D:
		return "Chapa — formato real (nesting)"
	}
	return string(t)
}

func mm(v float64) string { return num(decimal.NewFromFloat(v), 0) }

func (MapaCorte) MapaCorte(d cutting_plan_uc.MapaCorte) []byte {
	doc := pdfkit.NewLandscape()
	const mg = 26.0
	f := novaFolha(doc, mg, 20)
	f.logo = logo(doc, d.Logo)
	ac := aco
	pl := d.Plano
	rodapeSimples(doc, mg, ac, fmt.Sprintf("Mapa de corte · plano %d · %s · gerado em %s · VCUT0100", pl.Code, ou(d.MaterialNome, d.MaterialCodigo), dataHora(d.Gerado)))
	cab := func(sub string) float64 {
		p := f.p
		p.FillRect(mg, mg, f.cw(), 36, ac)
		x := mg + 12
		if f.logo != nil {
			p.FillRect(mg+6, mg+4, 64, 28, branco)
			desenharLogo(p, f.logo, mg+9, mg+6, 58, 24)
			x = mg + 80
		}
		f.texto(x, mg+16, neg, 12, branco, fmt.Sprintf("MAPA DE CORTE · PLANO %d", pl.Code))
		f.texto(x, mg+28, reg, 7.2, mistura(ac, branco, 0.75), fit(juntar(" · ", d.Empresa, sub), reg, 7.2, f.cw()-x-180))
		p.TextRight(f.w-mg-12, mg+16, neg, 8.6, branco, fit(juntar(" ", d.MaterialCodigo, d.MaterialNome), neg, 8.6, 260))
		p.TextRight(f.w-mg-12, mg+28, reg, 7, mistura(ac, branco, 0.75), rotuloTipoCorte(pl.CutType)+" · "+strings.ToLower(string(pl.Status)))
		return mg + 46
	}
	f.y = cab("resumo do plano")
	f.continua = func() float64 { return cab("continuação") }
	cw := f.cw()

	// Cores por peça (mesmo rótulo, mesma cor em todo o plano).
	corDe := map[string]cor{}
	var rotulos []string
	totalPecas, consumo := 0, 0
	area, usado := 0.0, 0.0
	for _, pt := range d.Padroes {
		consumo += pt.RepeatCount
		for _, x := range pt.Placements {
			totalPecas += pt.RepeatCount
			if _, ok := corDe[x.Label]; !ok {
				corDe[x.Label] = paletaPecas[len(rotulos)%len(paletaPecas)]
				rotulos = append(rotulos, x.Label)
			}
		}
		if pt.StockWidthMM > 0 && pt.StockHeightMM > 0 {
			area += pt.StockWidthMM * pt.StockHeightMM * float64(pt.RepeatCount)
			usado += pt.UsedAreaMM2 * float64(pt.RepeatCount)
		} else {
			area += pt.StockLengthMM * float64(pt.RepeatCount)
			usado += pt.UsedMM * float64(pt.RepeatCount)
		}
	}
	aprov := 0.0
	if area > 0 {
		aprov = usado / area * 100
	}
	unidade := "chapa(s)"
	if pl.CutType == entity.CutTypeLinear1D {
		unidade = "barra(s)"
	}
	cartoes(f, mg, cw, [][3]string{
		{"Material consumido", fmt.Sprintf("%d %s", consumo, unidade), fmt.Sprintf("%d padrão(ões) de corte", len(d.Padroes))},
		{"Peças cortadas", fmt.Sprint(totalPecas), fmt.Sprintf("%d peça(s) diferente(s)", len(rotulos))},
		{"Aproveitamento", num(decimal.NewFromFloat(aprov), 1) + "%", "ponderado pelas repetições"},
		{"Parâmetros de corte", "kerf " + num(decimal.NewFromFloat(pl.KerfMM), 1) + " mm", "refilo " + mm(pl.TrimMM) + " mm · retalho mín. " + mm(pl.MinRemnantMM) + " mm"},
	}, []cor{ac, ac, verde, cinza})

	// Tabela de padrões.
	f.texto(mg, f.y+6, neg, 8.5, ac, "PADRÕES DE CORTE")
	f.y += 11
	t := f.tabela(mg, cw, []col{{"Padrão", 0.6, pdfkit.AlignCenter}, {"Origem", 0.9, pdfkit.AlignLeft}, {"Dimensão (mm)", 1.3, pdfkit.AlignLeft},
		{"Repetir", 0.6, pdfkit.AlignRight}, {"Peças por unid.", 0.8, pdfkit.AlignRight}, {"Composição", 4, pdfkit.AlignLeft},
		{"Aproveit.", 0.7, pdfkit.AlignRight}, {"Sobra / retalho", 1.3, pdfkit.AlignLeft}, {"Página", 0.5, pdfkit.AlignCenter}},
		estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha})
	t.cabecalho()
	for i, pt := range d.Padroes {
		origem := "estoque"
		if pt.IsRemnant {
			origem = "retalho"
		}
		dim, sobra := mm(pt.StockLengthMM), mm(pt.RemnantMM)+" mm"
		if pt.StockWidthMM > 0 && pt.StockHeightMM > 0 {
			dim = mm(pt.StockWidthMM) + " × " + mm(pt.StockHeightMM)
			sobra = "—"
			if pt.RemnantWidthMM > 0 && pt.RemnantHeightMM > 0 {
				sobra = mm(pt.RemnantWidthMM) + " × " + mm(pt.RemnantHeightMM)
			}
		}
		t.linha([]string{fmt.Sprint(pt.Sequence), origem, dim, fmt.Sprintf("%d×", pt.RepeatCount), fmt.Sprint(len(pt.Placements)),
			composicao(pt), num(decimal.NewFromFloat(pt.UtilizationPct), 1) + "%", sobra, fmt.Sprint(i + 2)}, reg, tinta, nil)
	}

	// Uma página por padrão.
	for _, pt := range d.Padroes {
		f.p = doc.AddPage()
		f.y = cab(fmt.Sprintf("padrão %d de %d", pt.Sequence, len(d.Padroes)))
		f.padraoDeCorte(pt, corDe)
	}
	return doc.Render()
}

// composicao resume as peças do padrão: "2× LATERAL · 1× FECHAMENTO".
func composicao(pt *entity.CuttingPattern) string {
	cont := map[string]int{}
	var ordem []string
	for _, x := range pt.Placements {
		if cont[x.Label] == 0 {
			ordem = append(ordem, x.Label)
		}
		cont[x.Label]++
	}
	var out []string
	for _, l := range ordem {
		out = append(out, fmt.Sprintf("%d× %s", cont[l], l))
	}
	return strings.Join(out, " · ")
}

func (f *folha) padraoDeCorte(pt *entity.CuttingPattern, corDe map[string]cor) {
	mg, cw := f.mg, f.cw()
	p := f.p
	is2D := pt.StockWidthMM > 0 && pt.StockHeightMM > 0
	sw, sh := pt.StockLengthMM, 0.0
	if is2D {
		sw, sh = pt.StockWidthMM, pt.StockHeightMM
	}
	// Faixa de identificação do padrão.
	dim := mm(sw) + " mm"
	if is2D {
		dim = mm(sw) + " × " + mm(sh) + " mm"
	}
	p.FillRect(mg, f.y, cw, 24, mistura(aco, branco, 0.9))
	f.texto(mg+10, f.y+16, neg, 10, tinta, fmt.Sprintf("Padrão %d · %s · repetir %d vez(es)", pt.Sequence, dim, pt.RepeatCount))
	p.TextRight(mg+cw-10, f.y+16, neg, 9, verde, "aproveitamento "+num(decimal.NewFromFloat(pt.UtilizationPct), 1)+"%")
	f.y += 34

	// Área do desenho (à esquerda) e lista de peças (à direita).
	listaW := 230.0
	dx, dy := mg+18, f.y+12
	dw, dh := cw-listaW-40, f.base-f.y-70
	if !is2D {
		dh = 60
	}
	escala := dw / math.Max(sw, 1)
	if is2D {
		escala = math.Min(escala, dh/math.Max(sh, 1))
	}
	altura := sh * escala
	if !is2D {
		altura = 46
	}
	larguraDes := sw * escala
	// Material.
	p.FillRect(dx, dy, larguraDes, altura, fundo)
	p.StrokeRect(dx, dy, larguraDes, altura, 1.2, tinta)
	// Peças.
	for _, x := range pt.Placements {
		var rx, ry, rw, rh float64
		if is2D {
			rx, ry, rw, rh = dx+x.PosXMM*escala, dy+x.PosYMM*escala, x.WidthMM*escala, x.HeightMM*escala
		} else {
			rx, ry, rw, rh = dx+x.OffsetMM*escala, dy, x.LengthMM*escala, altura
		}
		c := corDe[x.Label]
		if len(x.Outline) >= 3 {
			pts := make([]float64, 0, len(x.Outline)*2)
			for _, q := range x.Outline {
				pts = append(pts, rx+q[0]*escala, ry+q[1]*escala)
			}
			p.FillPolygon(c, pts...)
			p.StrokeRect(rx, ry, rw, rh, 0.3, linha)
		} else {
			p.FillRect(rx, ry, rw, rh, c)
			p.StrokeRect(rx, ry, rw, rh, 0.7, tinta)
		}
		rot := x.Label
		if x.Rotated {
			rot += " (girada)"
		}
		tam := math.Min(7, math.Max(4.8, rh/3))
		if larg(neg, tam, rot) < rw-4 && rh > tam+4 {
			p.TextCenter(rx+rw/2, ry+rh/2+tam/3, neg, tam, tinta, rot)
			med := mm(rw / escala)
			if is2D {
				med += "×" + mm(rh/escala)
			}
			if larg(reg, tam-1, med) < rw-4 && rh > 2*tam+6 {
				p.TextCenter(rx+rw/2, ry+rh/2+tam/3+tam+1, reg, tam-1, cinza, med)
			}
		}
	}
	// Cotas do material.
	p.StrokeLine(dx, dy-6, dx+larguraDes, dy-6, 0.5, cinza)
	p.StrokeLine(dx, dy-9, dx, dy-3, 0.5, cinza)
	p.StrokeLine(dx+larguraDes, dy-9, dx+larguraDes, dy-3, 0.5, cinza)
	p.TextCenter(dx+larguraDes/2, dy-8, neg, 7, cinza, mm(sw)+" mm")
	if is2D {
		p.StrokeLine(dx-6, dy, dx-6, dy+altura, 0.5, cinza)
		p.TextRight(dx-8, dy+altura/2+2, neg, 6.6, cinza, mm(sh))
	}
	sobra := ""
	if !is2D && pt.RemnantMM > 0 {
		sobra = "sobra " + mm(pt.RemnantMM) + " mm"
	} else if is2D && pt.RemnantWidthMM > 0 {
		sobra = "maior retalho " + mm(pt.RemnantWidthMM) + " × " + mm(pt.RemnantHeightMM) + " mm"
	}
	f.texto(dx, dy+altura+14, reg, 7, cinza, juntar(" · ", sobra, "perda de corte "+mm(pt.KerfLossMM)+" mm", ternario(pt.IsRemnant, "cortar de RETALHO", "")))

	// Lista de peças do padrão.
	lx := mg + cw - listaW
	ly := f.y
	p.FillRect(lx, ly, listaW, 16, aco)
	p.Text(lx+8, ly+11, neg, 7, branco, "PEÇAS DESTE PADRÃO")
	p.TextRight(lx+listaW-8, ly+11, neg, 7, branco, "POR UNID. · TOTAL")
	ly += 16
	cont := map[string]int{}
	medida := map[string]string{}
	var ordem []string
	for _, x := range pt.Placements {
		if cont[x.Label] == 0 {
			ordem = append(ordem, x.Label)
			if is2D {
				medida[x.Label] = mm(x.WidthMM) + " × " + mm(x.HeightMM)
			} else {
				medida[x.Label] = mm(x.LengthMM) + " mm"
			}
		}
		cont[x.Label]++
	}
	sort.SliceStable(ordem, func(i, j int) bool { return cont[ordem[i]] > cont[ordem[j]] })
	for _, l := range ordem {
		p.FillRect(lx, ly, 10, 18, corDe[l])
		p.Text(lx+16, ly+8, neg, 7.4, tinta, fit(l, neg, 7.4, listaW-90))
		p.Text(lx+16, ly+15.5, reg, 6.2, cinza, medida[l])
		p.TextRight(lx+listaW-8, ly+12, neg, 8, tinta, fmt.Sprintf("%d · %d", cont[l], cont[l]*pt.RepeatCount))
		p.StrokeLine(lx, ly+18, lx+listaW, ly+18, 0.3, linha)
		ly += 18
	}
	// Controle do operador.
	ly += 12
	p.StrokeRect(lx, ly, listaW, 74, 0.6, linha)
	p.Text(lx+8, ly+12, neg, 6.6, cinza, "CONTROLE DE EXECUÇÃO")
	p.Text(lx+8, ly+28, reg, 7.6, tinta, fmt.Sprintf("Cortado ______ de %d %s", pt.RepeatCount, ternario(is2D, "chapa(s)", "barra(s)")))
	p.Text(lx+8, ly+44, reg, 7.6, tinta, "Lote do material ______________")
	p.StrokeLine(lx+8, ly+62, lx+listaW-8, ly+62, 0.5, tinta)
	p.Text(lx+8, ly+71, reg, 6.2, cinza, "Operador · data")
}

func ternario(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}
