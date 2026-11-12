package documentos

import (
	"fmt"
	"strings"
	"time"

	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/certificado"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

// Certificado de qualidade — documento formal entregue ao cliente: moldura,
// identificação do material e do cliente, cada lote de inspeção com a decisão
// em destaque e o resultado de cada característica, a declaração com o selo e
// a assinatura do responsável. Certificado cancelado sai marcado.
//
// O conteúdo é a foto gravada na emissão; só a logo vem da configuração atual.

var verdeQualidade = cor{R: 20, G: 83, B: 45}

func especCert(c dom.CaracResumo) string {
	if c.Tipo != "VARIAVEL" {
		return "atributo (verificação)"
	}
	u := ""
	if c.Unidade != "" {
		u = " " + c.Unidade
	}
	switch {
	case c.Minimo != "" && c.Maximo != "":
		return virgula(c.Minimo) + " a " + virgula(c.Maximo) + u
	case c.Minimo != "":
		return "mín. " + virgula(c.Minimo) + u
	case c.Maximo != "":
		return "máx. " + virgula(c.Maximo) + u
	case c.Nominal != "":
		return virgula(c.Nominal) + u
	}
	return "—"
}

func virgula(s string) string {
	if s == "" {
		return "—"
	}
	return strings.Replace(s, ".", ",", 1)
}

func decisaoCert(l dom.LoteResumo) (string, cor) {
	d := strings.ToUpper(l.Decisao)
	switch {
	case l.Concessao:
		return "APROVADO SOB CONCESSÃO", ambar
	case strings.Contains(d, "REJEIT") || strings.Contains(d, "REPROV") || strings.Contains(d, "SUCATA"):
		return strings.ReplaceAll(d, "_", " "), verm
	case d == "":
		return "SEM DECISÃO", cinza
	}
	return strings.ReplaceAll(d, "_", " "), verde
}

// CertificadoQualidade desenha o certificado.
func CertificadoQualidade(c *dom.Certificado, gerado time.Time, logoEmpresa []byte) []byte {
	doc := pdfkit.New()
	const mg = 34.0
	f := novaFolha(doc, mg, 24)
	f.logo = logo(doc, logoEmpresa)
	ac := verdeQualidade
	ct := c.Conteudo
	cancelado := strings.EqualFold(c.Status, "CANCELADO")
	w, h := doc.Size()
	doc.SetFooter(func(p *pdfkit.Page, n, total int) {
		// Moldura dupla em todas as páginas.
		p.StrokeRect(16, 16, w-32, h-32, 1.4, ac)
		p.StrokeRect(20, 20, w-40, h-40, 0.5, ac)
		y := h - mg + 6
		p.Text(mg, y, reg, 6.6, cinza, fit(fmt.Sprintf("Certificado de qualidade nº %d · emitido em %s · gerado em %s", c.Numero, dataHora(c.EmitidoEm), dataHora(gerado)), reg, 6.6, w-2*mg-70))
		p.TextRight(w-mg, y, neg, 6.8, ac, fmt.Sprintf("Página %d de %d", n, total))
	})
	f.continua = func() float64 {
		f.texto(mg, mg+10, neg, 8.5, ac, fmt.Sprintf("CERTIFICADO DE QUALIDADE Nº %d (continuação)", c.Numero))
		f.p.StrokeLine(mg, mg+15, f.w-mg, mg+15, 0.6, ac)
		return mg + 26
	}
	p := f.p
	cw := f.cw()

	// Cabeçalho centrado, como um certificado.
	top := mg
	if f.logo != nil {
		lw, lh := f.logo.Fit(110, 44)
		p.DrawImage(f.logo, f.w/2-lw/2, top, lw, lh)
		top += lh + 8
	}
	p.TextCenter(f.w/2, top+10, neg, 9.5, tinta, fit(ct.Empresa.Nome, neg, 9.5, cw))
	if ct.Empresa.CNPJ != "" {
		p.TextCenter(f.w/2, top+21, reg, 7.4, cinza, "CNPJ "+cnpj(ct.Empresa.CNPJ))
	}
	top += 34
	p.FillRect(mg+cw/2-150, top, 300, 1.2, ac)
	p.TextCenter(f.w/2, top+24, neg, 17, ac, "CERTIFICADO DE QUALIDADE")
	p.TextCenter(f.w/2, top+40, neg, 10, tinta, fmt.Sprintf("Nº %d · emitido em %s", c.Numero, data(c.EmitidoEm)))
	p.FillRect(mg+cw/2-150, top+48, 300, 1.2, ac)
	f.y = top + 60
	if cancelado {
		p.FillRect(mg, f.y, cw, 22, mistura(verm, branco, 0.85))
		motivo := ""
		if c.MotivoCancelamento != nil {
			motivo = " — " + *c.MotivoCancelamento
		}
		p.TextCenter(f.w/2, f.y+15, neg, 9.5, verm, fit("CERTIFICADO CANCELADO"+motivo+": sem validade", neg, 9.5, cw-20))
		f.y += 30
	}

	// Material e cliente.
	colW := (cw - 12) / 2
	quadro := func(x float64, titulo string, campos [][2]string) float64 {
		y := f.y
		p.FillRect(x, y, colW, 16, mistura(ac, branco, 0.88))
		p.Text(x+8, y+11, neg, 7, ac, titulo)
		y += 22
		for _, c := range campos {
			p.Text(x+8, y+8, neg, 6, cinza, strings.ToUpper(c[0]))
			ls := quebrar(ou(c[1], "—"), reg, 8, colW-96)
			for i, l := range ls {
				p.Text(x+90, y+8+float64(i)*9.5, reg, 8, tinta, l)
			}
			y += 9.5*float64(max(len(ls), 1)) + 3
		}
		return y
	}
	item := fmt.Sprintf("%d — %s", ct.ItemCode, ct.ItemNome)
	if ct.Mask != "" {
		item += " (máscara " + ct.Mask + ")"
	}
	y1 := quadro(mg, "MATERIAL CERTIFICADO", [][2]string{{"Item", item}, {"Quantidade", virgula(ct.Quantidade)}, {"Lote", ct.Lote}, {"Corrida", ct.Corrida}, {"Cert. da usina", ct.CertUsina}})
	cli := [][2]string{{"Cliente", "—"}}
	if ct.Cliente != nil {
		cli = [][2]string{{"Cliente", ct.Cliente.Nome}, {"CNPJ", cnpj(ct.Cliente.CNPJ)}}
	}
	if ct.NF != nil {
		cli = append(cli, [2]string{"Nota fiscal", fmt.Sprintf("NF-e %d série %s de %s", ct.NF.Numero, ou(ct.NF.Serie, "1"), data(ct.NF.Data))})
	}
	if ct.PedidoVenda != nil {
		cli = append(cli, [2]string{"Pedido de venda", fmt.Sprint(*ct.PedidoVenda)})
	}
	y2 := quadro(mg+colW+12, "CLIENTE E DOCUMENTOS", cli)
	f.y = maxF(y1, y2) + 12

	// Lotes de inspeção.
	for _, l := range ct.Lotes {
		f.cabe(80)
		p = f.p
		dec, cd := decisaoCert(l)
		p.FillRect(mg, f.y, cw, 22, mistura(ac, branco, 0.93))
		p.FillRect(mg, f.y, 4, 22, ac)
		f.texto(mg+12, f.y+14.5, neg, 8.6, tinta, fmt.Sprintf("Lote de inspeção %d · %s", l.Numero, strings.ToLower(strings.ReplaceAll(l.Origem, "_", " "))))
		sw := larg(neg, 7, dec) + 12
		p.FillRect(f.w-mg-sw-8, f.y+5, sw, 12, cd)
		p.Text(f.w-mg-sw-2, f.y+13.5, neg, 7, branco, dec)
		f.y += 22
		info := juntar(" · ", "quantidade aproveitada "+virgula(l.Aproveitada), "decisão em "+data(l.DecididoEm), rotulo("amostragem ", l.Amostra), rotulo("tratamento ", l.Tratamento))
		f.texto(mg+12, f.y+10, reg, 7, cinza, fit(info, reg, 7, cw-24))
		f.y += 15
		t := f.tabela(mg, cw, []col{{"Característica", 2.2, pdfkit.AlignLeft}, {"Especificação", 1.6, pdfkit.AlignLeft}, {"n", 0.35, pdfkit.AlignRight},
			{"Mínimo", 0.7, pdfkit.AlignRight}, {"Máximo", 0.7, pdfkit.AlignRight}, {"Média", 0.7, pdfkit.AlignRight}, {"NC", 0.35, pdfkit.AlignRight},
			{"Instrumento", 1.5, pdfkit.AlignLeft}, {"Resultado", 1.25, pdfkit.AlignLeft}},
			estiloTabela{cabFundo: ac, cabTexto: branco, grade: linha, altLinha: 15})
		t.cabecalho()
		if len(l.Caracteristicas) == 0 {
			t.linha([]string{"Sem características medidas neste lote."}, ita, cinza, nil)
		}
		for _, c := range l.Caracteristicas {
			nome := c.Nome
			if c.Critica {
				nome += " (crítica)"
			}
			mn, mx, md := virgula(c.MedidoMinimo), virgula(c.MedidoMaximo), virgula(c.Media)
			if c.Tipo != "VARIAVEL" {
				mn, mx, md = "—", "—", "—"
			}
			topo := t.linha([]string{nome, especCert(c), fmt.Sprint(c.N), mn, mx, md, fmt.Sprint(c.NaoConformes), strings.Join(c.Instrumentos, ", "), ""}, reg, tinta, nil)
			xr := t.xCol(8) + 4
			if c.NaoConformes == 0 {
				marcaOK(f.p, xr, topo+4, verde)
				f.p.Text(xr+11, topo+10.3, neg, 7.2, verde, "Conforme")
			} else {
				f.p.FillRect(xr, topo+4, 7, 7, verm)
				f.p.Text(xr+11, topo+10.3, neg, 7.2, verm, "Não conforme")
			}
		}
		f.y += 14
	}
	if len(ct.Lotes) == 0 {
		f.cabe(20)
		f.texto(mg, f.y+8, ita, 7.6, cinza, "Certificado sem lote de inspeção vinculado.")
		f.y += 18
	}

	if obs := strings.TrimSpace(ct.Observacoes); obs != "" {
		ls := quebrar(obs, reg, 7.6, cw-20)
		hh := 18 + float64(len(ls))*10
		f.cabe(hh + 8)
		f.p.StrokeRect(mg, f.y, cw, hh, 0.5, linha)
		f.texto(mg+10, f.y+11, neg, 6.8, ac, "OBSERVAÇÕES")
		for i, l := range ls {
			f.texto(mg+10, f.y+22+float64(i)*10, reg, 7.6, tinta, l)
		}
		f.y += hh + 12
	}

	// Declaração com selo e assinatura.
	decl := quebrar(ou(ct.Declaracao, "Declaramos que o material acima atende às especificações indicadas."), reg, 8.4, cw-140)
	hh := maxF(24+float64(len(decl))*11, 80)
	f.cabe(hh + 70)
	p = f.p
	p.StrokeRect(mg, f.y, cw, hh, 0.8, ac)
	for i, l := range decl {
		f.texto(mg+14, f.y+20+float64(i)*11, reg, 8.4, tinta, l)
	}
	// Selo: dois quadrados concêntricos com o resultado.
	sx, sy := f.w-mg-112, f.y+(hh-70)/2
	selo := "APROVADO"
	cs := ac
	if cancelado {
		selo, cs = "CANCELADO", verm
	}
	p.StrokeRect(sx, sy, 100, 70, 1.6, cs)
	p.StrokeRect(sx+4, sy+4, 92, 62, 0.5, cs)
	p.TextCenter(sx+50, sy+22, neg, 6.4, cs, "CONTROLE DA QUALIDADE")
	p.TextCenter(sx+50, sy+42, neg, 11, cs, selo)
	p.TextCenter(sx+50, sy+57, reg, 6.6, cs, data(c.EmitidoEm))
	f.y += hh + 40
	p.StrokeLine(mg, f.y, mg+230, f.y, 0.7, tinta)
	f.texto(mg, f.y+11, reg, 7.6, cinza, "Responsável pela qualidade — nome, registro e assinatura")
	p.StrokeLine(f.w-mg-170, f.y, f.w-mg, f.y, 0.7, tinta)
	f.texto(f.w-mg-170, f.y+11, reg, 7.6, cinza, "Local e data")
	return doc.Render()
}
