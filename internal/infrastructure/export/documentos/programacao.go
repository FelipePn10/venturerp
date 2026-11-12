package documentos

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/shopspring/decimal"
)

// Programação da produção (APS) — Gantt por máquina comprimido nos dias com
// programação (o turno vira a escala de cada dia), legenda por ordem e a
// sequência de cada máquina para o chão de fábrica, com a folga até a data
// de necessidade.

var noite = cor{R: 24, G: 33, B: 52}

var paletaOF = []cor{
	{R: 37, G: 99, B: 235}, {R: 217, G: 119, B: 6}, {R: 22, G: 163, B: 74}, {R: 147, G: 51, B: 234},
	{R: 220, G: 38, B: 38}, {R: 8, G: 145, B: 178}, {R: 202, G: 138, B: 4}, {R: 219, G: 39, B: 119},
	{R: 79, G: 70, B: 229}, {R: 101, G: 163, B: 13},
}

func (Gerador) Programacao(d uc.Programacao) []byte {
	doc := pdfkit.NewLandscape()
	const mg = 24.0
	f := novaFolha(doc, mg, 20)
	f.logo = logo(doc, d.Emitente.Logo)
	periodo := data(d.De) + " a " + data(d.Ate)
	rodapeSimples(doc, mg, noite, "Programação da produção (APS) · "+periodo+" · "+emitidoPor(d.Cabecalho)+" · a sequência válida é a do sistema (VPRO0210)")
	cab := func(cont bool) float64 {
		p := f.p
		p.FillRect(mg, mg, f.cw(), 38, noite)
		x := mg + 12
		if !cont && f.logo != nil {
			p.FillRect(mg+6, mg+5, 66, 28, branco)
			desenharLogo(p, f.logo, mg+9, mg+7, 60, 24)
			x = mg + 84
		}
		t := "PROGRAMAÇÃO DA PRODUÇÃO · APS"
		if cont {
			t += " (continuação)"
		}
		f.texto(x, mg+17, neg, 12.5, branco, t)
		f.texto(x, mg+29, reg, 7.2, mistura(noite, branco, 0.7), fit("Sequenciamento em capacidade finita · "+ou(d.Grupo, "todos os recursos")+" · "+d.Emitente.Nome, reg, 7.2, f.w-mg-x-150))
		p.TextRight(f.w-mg-12, mg+17, neg, 9, branco, periodo)
		p.TextRight(f.w-mg-12, mg+29, reg, 7, mistura(noite, branco, 0.7), fmt.Sprintf("%d operação(ões) programada(s)", len(d.Operacoes)))
		return mg + 48
	}
	f.y = cab(false)
	f.continua = func() float64 { return cab(true) }
	cw := f.cw()
	if len(d.Operacoes) == 0 {
		f.texto(mg, f.y+14, ita, 9, cinza, "Nenhuma operação programada no período: rode o sequenciamento (VPRO0210).")
		return doc.Render()
	}

	// Cores por ordem e eixo comprimido.
	corDe := map[int64]cor{}
	var ordens []int64
	diasSet := map[string]time.Time{}
	h0, h1 := 24.0, 0.0
	maqs := map[string]bool{}
	var maquinas []string
	for _, o := range d.Operacoes {
		if _, ok := corDe[o.Ordem]; !ok {
			corDe[o.Ordem] = paletaOF[len(ordens)%len(paletaOF)]
			ordens = append(ordens, o.Ordem)
		}
		for dt := dia(o.Inicio); !dt.After(dia(o.Fim)); dt = dt.AddDate(0, 0, 1) {
			diasSet[dt.Format("2006-01-02")] = dt
		}
		h0 = math.Min(h0, horaDec(o.Inicio))
		h1 = math.Max(h1, horaDec(o.Fim))
		if !maqs[o.Maquina] {
			maqs[o.Maquina] = true
			maquinas = append(maquinas, o.Maquina)
		}
	}
	h0, h1 = math.Floor(h0), math.Ceil(h1)
	if h1 <= h0 {
		h1 = h0 + 1
	}
	var dias []time.Time
	for _, v := range diasSet {
		dias = append(dias, v)
	}
	sort.Slice(dias, func(i, j int) bool { return dias[i].Before(dias[j]) })
	idx := map[string]int{}
	for i, dt := range dias {
		idx[dt.Format("2006-01-02")] = i
	}
	sort.Strings(maquinas)

	// Gantt.
	rot := 150.0
	gx, gw := mg+rot, cw-rot
	dw := gw / float64(len(dias))
	linhaH := 24.0
	f.cabe(40 + linhaH*float64(len(maquinas)))
	p := f.p
	agrup := ou(d.Agrupamento, "máquina")
	f.texto(mg, f.y+8, neg, 8.5, noite, "GANTT POR "+strings.ToUpper(agrup))
	f.texto(mg+larg(neg, 8.5, "GANTT POR "+strings.ToUpper(agrup))+12, f.y+8, ita, 6.8, cinza, fmt.Sprintf("só os dias com programação · escala de cada dia: %02.0f:00 às %02.0f:00", h0, h1))
	f.y += 14
	for i, dt := range dias {
		x := gx + float64(i)*dw
		p.FillRect(x+0.5, f.y, dw-1, 24, mistura(noite, branco, 0.9))
		p.TextCenter(x+dw/2, f.y+10, neg, 7.6, noite, dt.Format("02/01")+" · "+diaSemana(dt)[:3])
		// Marcas de hora.
		passoH := math.Max(1, math.Ceil((h1-h0)/math.Max(1, dw/26)))
		for h := h0; h <= h1; h += passoH {
			hx := x + dw*(h-h0)/(h1-h0)
			p.StrokeLine(hx, f.y+15, hx, f.y+19, 0.4, cinza)
			if hx < x+dw-8 {
				p.Text(hx+1, f.y+22, reg, 5.2, cinza, fmt.Sprintf("%02.0fh", h))
			}
		}
	}
	f.y += 26
	topoGantt := f.y
	for _, m := range maquinas {
		p.FillRect(mg, f.y, cw, linhaH, branco)
		p.StrokeLine(mg, f.y+linhaH, mg+cw, f.y+linhaH, 0.4, linha)
		centro := ""
		for _, o := range d.Operacoes {
			if o.Maquina == m {
				centro = o.Centro
				break
			}
		}
		f.texto(mg+4, f.y+11, neg, 7.4, tinta, fit(m, neg, 7.4, rot-8))
		f.texto(mg+4, f.y+19.5, reg, 6, cinza, fit(centro, reg, 6, rot-8))
		for _, o := range d.Operacoes {
			if o.Maquina != m {
				continue
			}
			for dt := dia(o.Inicio); !dt.After(dia(o.Fim)); dt = dt.AddDate(0, 0, 1) {
				i := idx[dt.Format("2006-01-02")]
				a, b := h0, h1
				if dia(o.Inicio).Equal(dt) {
					a = horaDec(o.Inicio)
				}
				if dia(o.Fim).Equal(dt) {
					b = horaDec(o.Fim)
				}
				x0 := gx + float64(i)*dw + dw*(a-h0)/(h1-h0)
				x1 := gx + float64(i)*dw + dw*(b-h0)/(h1-h0)
				c := corDe[o.Ordem]
				p.FillRect(x0, f.y+4, math.Max(x1-x0, 1.5), linhaH-8, c)
				lab := fmt.Sprintf("OF %d·%d", o.Ordem, o.Etapa)
				if larg(neg, 6, lab)+4 < x1-x0 {
					p.Text(x0+2, f.y+linhaH/2+2.2, neg, 6, branco, lab)
				}
			}
		}
		f.y += linhaH
	}
	for i := range dias {
		x := gx + float64(i)*dw
		p.StrokeLine(x, topoGantt, x, f.y, 0.5, linha)
	}
	f.y += 8

	// Legenda das ordens.
	f.cabe(16 * float64((len(ordens)+2)/3+1))
	f.texto(mg, f.y+8, neg, 7.6, noite, "ORDENS")
	f.y += 12
	colW := cw / 3
	for i, o := range ordens {
		var op uc.OperacaoProgramada
		var fim time.Time
		for _, x := range d.Operacoes {
			if x.Ordem == o {
				op = x
				if x.Fim.After(fim) {
					fim = x.Fim
				}
			}
		}
		x := mg + float64(i%3)*colW
		y := f.y + float64(i/3)*14
		f.p.FillRect(x, y+1, 10, 9, corDe[o])
		txt := fmt.Sprintf("OF %d · %s%s · termina %s", o, op.Item, rotulo(" · ", qtdUn(op.Quantidade)), fim.Format("02/01 15:04"))
		if op.Necessidade != nil {
			txt += " · precisa " + op.Necessidade.Format("02/01")
		}
		f.texto(x+14, y+8.5, reg, 6.9, tinta, fit(txt, reg, 6.9, colW-20))
	}
	f.y += 14*float64((len(ordens)+2)/3) + 12

	// Sequência por máquina.
	f.cabe(40)
	f.texto(mg, f.y+8, neg, 8.5, noite, "SEQUÊNCIA POR "+strings.ToUpper(agrup))
	f.texto(mg+larg(neg, 8.5, "SEQUÊNCIA POR "+strings.ToUpper(agrup))+12, f.y+8, ita, 6.8, cinza, "o operador segue a ordem e marca o que concluiu")
	f.y += 14
	cols := []col{{"#", 0.3, pdfkit.AlignCenter}, {"Início", 0.9, pdfkit.AlignCenter}, {"Fim", 0.9, pdfkit.AlignCenter}, {"Duração", 0.6, pdfkit.AlignRight},
		{"OF", 0.5, pdfkit.AlignCenter}, {"Etapa", 0.45, pdfkit.AlignCenter}, {"Item", 0.95, pdfkit.AlignLeft}, {"Descrição", 2.4, pdfkit.AlignLeft},
		{"Operação", 1.5, pdfkit.AlignLeft}, {"Qtde", 0.55, pdfkit.AlignRight}, {"Folga", 0.75, pdfkit.AlignRight}, {"Feito", 0.45, pdfkit.AlignCenter}}
	t := f.tabela(mg, cw, cols, estiloTabela{cabFundo: noite, cabTexto: branco, grade: linha, altLinha: 15})
	for _, m := range maquinas {
		f.cabe(18 + 15 + 15)
		f.p.FillRect(mg, f.y, cw, 16, mistura(noite, branco, 0.88))
		f.texto(mg+6, f.y+11, neg, 7.8, noite, m)
		f.y += 16
		t.cabecalho()
		n := 0
		for _, o := range d.Operacoes {
			if o.Maquina != m {
				continue
			}
			n++
			dur := o.Fim.Sub(o.Inicio)
			folga := "—"
			cf := tinta
			if o.Necessidade != nil {
				fd := dia(*o.Necessidade).Sub(dia(o.Fim)).Hours() / 24
				folga = fmt.Sprintf("%.0f d", fd)
				if fd < 0 {
					cf = verm
				}
			}
			topo := t.linha([]string{fmt.Sprint(n), o.Inicio.Format("02/01 15:04"), o.Fim.Format("02/01 15:04"), duracao(dur), fmt.Sprint(o.Ordem),
				fmt.Sprint(o.Etapa), o.Item, o.Descricao, o.Operacao, ou(qtdSe(o.Quantidade), "—"), "", ""}, reg, tinta, nil)
			f.p.FillRect(t.xCol(4)+2, topo+3, 2.5, 9, corDe[o.Ordem])
			f.p.TextRight(t.xCol(11)-3.5, topo+10.3, neg, 7.4, cf, folga)
			caixa(f.p, t.xCol(11)+t.ws[11]/2-4, topo+3.5, tinta)
			if o.Situacao == "DONE" || o.Situacao == "COMPLETED" {
				marcaOK(f.p, t.xCol(11)+t.ws[11]/2-3, topo+3.5, verde)
			}
		}
		f.y += 8
	}
	return doc.Render()
}

func qtdUn(v decimal.Decimal) string {
	if !v.IsPositive() {
		return ""
	}
	return qtd(v) + " un"
}

func horaDec(t time.Time) float64 { return float64(t.Hour()) + float64(t.Minute())/60 }

func duracao(d time.Duration) string {
	m := int(d.Minutes() + 0.5)
	return fmt.Sprintf("%dh%02d", m/60, m%60)
}
