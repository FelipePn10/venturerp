// Package certificado desenha o certificado de qualidade em PDF com o pdfkit
// do sistema: timbre, dados do material e do cliente, uma tabela de medições
// por lote de inspeção, a decisão (com a concessão, se houver), a declaração e
// a assinatura.
package certificado

import (
	"fmt"
	"strings"
	"time"

	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/certificado"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

var colunas = []struct {
	nome  string
	peso  float64
	align pdfkit.Align
}{
	{"Característica", 3.2, pdfkit.AlignLeft},
	{"Especificação", 2.4, pdfkit.AlignLeft},
	{"n", 0.6, pdfkit.AlignRight},
	{"Mínimo", 1.2, pdfkit.AlignRight},
	{"Máximo", 1.2, pdfkit.AlignRight},
	{"Média", 1.2, pdfkit.AlignRight},
	{"NC", 0.6, pdfkit.AlignRight},
	{"Instrumento", 1.6, pdfkit.AlignLeft},
}

func espec(c dom.CaracResumo) string {
	if c.Tipo != "VARIAVEL" {
		return "conforme"
	}
	u := ""
	if c.Unidade != "" {
		u = " " + c.Unidade
	}
	switch {
	case c.Minimo != "" && c.Maximo != "":
		return br(c.Minimo) + " a " + br(c.Maximo) + u
	case c.Minimo != "":
		return "mín. " + br(c.Minimo) + u
	case c.Maximo != "":
		return "máx. " + br(c.Maximo) + u
	case c.Nominal != "":
		return br(c.Nominal) + u
	}
	return "—"
}

func br(s string) string { return strings.Replace(s, ".", ",", 1) }

// Gerar devolve o PDF do certificado.
func Gerar(c *dom.Certificado, agora time.Time) []byte {
	doc := pdfkit.New()
	th := pdfkit.DefaultTheme()
	w, h := doc.Size()
	m := pdfkit.Margin
	cw := w - 2*m
	nota := fmt.Sprintf("Certificado de qualidade nº %d — emitido em %s", c.Numero, c.EmitidoEm.Format("02/01/2006 15:04"))
	doc.SetFooter(func(p *pdfkit.Page, num, total int) { p.Footer(th, m, h-m+4, w-m, nota, num, total) })
	ct := c.Conteudo
	page := doc.AddPage()
	y := page.Letterhead(th, pdfkit.Company{Name: ct.Empresa.Nome, CNPJ: ct.Empresa.CNPJ}, nil, m, m, cw, false)
	y += 22
	page.TextCenter(w/2, y, pdfkit.FontBold, 15, th.Title, fmt.Sprintf("CERTIFICADO DE QUALIDADE Nº %d", c.Numero))
	if c.Status == "CANCELADO" {
		y += 16
		page.TextCenter(w/2, y, pdfkit.FontBold, 12, pdfkit.Color{R: 192, G: 57, B: 43}, "CANCELADO")
	}
	y += 18

	secao := func(rotulo string, linhas []string) {
		page.FillRect(m, y, cw, 14, th.Brand)
		page.Text(m+6, y+10, pdfkit.FontBold, 8.5, th.BrandText, rotulo)
		page.StrokeRect(m, y, cw, 14+float64(len(linhas))*12+4, 0.5, th.Rule)
		y += 25
		for _, l := range linhas {
			page.Text(m+6, y, pdfkit.FontRegular, 9, th.Text, l)
			y += 12
		}
		y += 8
	}
	material := []string{fmt.Sprintf("Item: %d — %s%s", ct.ItemCode, ct.ItemNome, map[bool]string{true: "  (máscara " + ct.Mask + ")", false: ""}[ct.Mask != ""]),
		fmt.Sprintf("Quantidade certificada: %s", br(ct.Quantidade))}
	if ct.Lote != "" || ct.Corrida != "" {
		material = append(material, fmt.Sprintf("Lote: %s    Corrida: %s    Certificado da usina: %s", vazio(ct.Lote), vazio(ct.Corrida), vazio(ct.CertUsina)))
	}
	secao("MATERIAL", material)
	cliente := []string{}
	if ct.Cliente != nil {
		cliente = append(cliente, fmt.Sprintf("%s    %s", ct.Cliente.Nome, ct.Cliente.CNPJ))
	}
	if ct.NF != nil {
		cliente = append(cliente, fmt.Sprintf("NF-e %d série %s de %s", ct.NF.Numero, ct.NF.Serie, ct.NF.Data.Format("02/01/2006")))
	}
	if ct.PedidoVenda != nil {
		cliente = append(cliente, fmt.Sprintf("Pedido de venda %d", *ct.PedidoVenda))
	}
	if len(cliente) > 0 {
		secao("CLIENTE", cliente)
	}

	soma := 0.0
	for _, c := range colunas {
		soma += c.peso
	}
	larguras := make([]float64, len(colunas))
	for i, c := range colunas {
		larguras[i] = cw * c.peso / soma
	}
	linha := func(vals []string, fonte pdfkit.Font, fundo *pdfkit.Color, cor pdfkit.Color) {
		if y > h-m-60 {
			page = doc.AddPage()
			y = m + 10
		}
		if fundo != nil {
			page.FillRect(m, y, cw, 14, *fundo)
		}
		x := m
		for i, v := range vals {
			maxW := larguras[i] - 8
			// Corta por runa (o texto tem acento) até caber, com reticências.
			for r := []rune(v); pdfkit.TextWidth(fonte, 8, v) > maxW && len(r) > 2; {
				r = r[:len(r)-2]
				v = string(r) + "…"
			}
			switch colunas[i].align {
			case pdfkit.AlignRight:
				page.TextRight(x+larguras[i]-4, y+10, fonte, 8, cor, v)
			default:
				page.Text(x+4, y+10, fonte, 8, cor, v)
			}
			x += larguras[i]
		}
		y += 14
	}
	for _, l := range ct.Lotes {
		if y > h-m-120 {
			page = doc.AddPage()
			y = m + 10
		}
		dec := strings.ToLower(strings.ReplaceAll(l.Decisao, "_", " "))
		if l.Concessao {
			dec += " — SOB CONCESSÃO"
		}
		page.Text(m, y+10, pdfkit.FontBold, 9.5, th.Title, fmt.Sprintf("Lote de inspeção %d (%s) — %s, %s aprovado em %s", l.Numero, strings.ToLower(l.Origem), dec,
			br(l.Aproveitada), l.DecididoEm.Format("02/01/2006")))
		y += 14
		if l.Amostra != "" {
			page.Text(m, y+9, pdfkit.FontRegular, 8, th.Muted, "Amostragem: "+l.Amostra)
			y += 13
		}
		cab := make([]string, len(colunas))
		for i, c := range colunas {
			cab[i] = c.nome
		}
		bg := th.Brand
		linha(cab, pdfkit.FontBold, &bg, th.BrandText)
		for i, c := range l.Caracteristicas {
			nome := c.Nome
			if c.Critica {
				nome += " (crítica)"
			}
			vals := []string{nome, espec(c), fmt.Sprint(c.N), br(c.MedidoMinimo), br(c.MedidoMaximo), br(c.Media), fmt.Sprint(c.NaoConformes), strings.Join(c.Instrumentos, ", ")}
			if c.Tipo != "VARIAVEL" {
				vals[3], vals[4], vals[5] = "—", "—", "—"
			}
			var z *pdfkit.Color
			if i%2 == 1 {
				zz := th.Zebra
				z = &zz
			}
			linha(vals, pdfkit.FontRegular, z, th.Text)
		}
		if len(l.Caracteristicas) == 0 {
			linha([]string{"sem características medidas", "", "", "", "", "", "", ""}, pdfkit.FontRegular, nil, th.Muted)
		}
		y += 10
	}
	if ct.Observacoes != "" {
		secao("OBSERVAÇÕES", quebrar(ct.Observacoes, 110))
	}
	if y > h-m-110 {
		page = doc.AddPage()
		y = m + 10
	}
	for _, l := range quebrar(ct.Declaracao, 115) {
		page.Text(m, y+10, pdfkit.FontRegular, 9, th.Text, l)
		y += 12
	}
	y += 40
	page.StrokeLine(m, y, m+220, y, 0.6, th.Text)
	page.Text(m, y+12, pdfkit.FontRegular, 8.5, th.Muted, "Responsável pela qualidade")
	page.TextRight(w-m, y+12, pdfkit.FontRegular, 8.5, th.Muted, "Gerado em "+agora.Format("02/01/2006 15:04"))
	return doc.Render()
}

func vazio(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func quebrar(s string, max int) []string {
	var out []string
	linha := ""
	for _, p := range strings.Fields(s) {
		if len([]rune(linha))+len([]rune(p))+1 > max && linha != "" {
			out = append(out, linha)
			linha = p
			continue
		}
		if linha == "" {
			linha = p
		} else {
			linha += " " + p
		}
	}
	if linha != "" {
		out = append(out, linha)
	}
	return out
}
