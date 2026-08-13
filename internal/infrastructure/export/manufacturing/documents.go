// Package manufacturing renders industrial documents using the ERP pdfkit.
package manufacturing

import (
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

type Branding struct {
	Company       pdfkit.Company
	Logo          []byte
	BrandColorHex string
}
type Field struct{ Label, Value string }
type Column struct {
	Title  string
	Weight float64
	Align  pdfkit.Align
}
type OrderSection struct {
	Title   string
	Columns []Column
	Rows    [][]string
}
type ManufacturingOrder struct {
	Branding                        Branding
	Title, Number, Status, Subtitle string
	BarcodeValue                    string
	BarcodeValidUntil               *time.Time
	Generated                       time.Time
	Summary                         []Field
	Sections                        []OrderSection
	Approvals                       []Field
	Disclaimer                      string
}
type StructureNode struct {
	Level                                                                         int
	Position, Code, Description, Quantity, Unit, Loss, Origin, LeadTime, UnitCost string
}
type ItemStructure struct {
	Branding                           Branding
	Title, RootCode, Revision, BaseQty string
	Generated                          time.Time
	Summary                            []Field
	Nodes                              []StructureNode
}

func theme(b Branding) pdfkit.Theme {
	t := pdfkit.DefaultTheme()
	if c, ok := pdfkit.ParseHexColor(b.BrandColorHex); ok {
		t.Brand, t.Title, t.TotalBg = c, c, pdfkit.Color{R: 232, G: 245, B: 233}
	}
	return t
}

func RenderManufacturingOrder(d ManufacturingOrder) []byte {
	doc := pdfkit.NewLandscape()
	th := theme(d.Branding)
	logo, _ := doc.AddImage(d.Branding.Logo)
	const margin = pdfkit.Margin
	w, h := doc.Size()
	contentW, footTop := w-2*margin, h-margin+4
	generated := d.Generated
	if generated.IsZero() {
		generated = time.Now()
	}
	doc.SetFooter(func(p *pdfkit.Page, page, total int) {
		p.StrokeLine(margin, footTop-9, w-margin, footTop-9, .5, th.Rule)
		p.Text(margin, footTop+2, pdfkit.FontRegular, 7.5, th.Muted, "Documento industrial gerado pelo VentureERP em "+generated.Format("02/01/2006 15:04"))
		p.TextRight(w-margin, footTop+2, pdfkit.FontRegular, 7.5, th.Muted, fmt.Sprintf("Página %d de %d", page, total))
	})
	page := doc.AddPage()
	y := page.Letterhead(th, d.Branding.Company, logo, margin, margin, contentW, false) + 13
	y = orderTitle(page, th, d, margin, y, contentW)
	y = summaryCards(page, th, d.Summary, margin, y+10, contentW)
	if d.BarcodeValue != "" {
		y = scannerBarcode(page, th, d, margin, y+2, contentW)
	}
	for _, section := range d.Sections {
		if y+35+float64(len(section.Rows))*16 > footTop-28 {
			page = doc.AddPage()
			y = page.Letterhead(th, d.Branding.Company, logo, margin, margin, contentW, true) + 13
			page.Text(margin, y, pdfkit.FontBold, 12, th.Title, fit(d.Title+" — "+d.Number+" (continuação)", pdfkit.FontBold, 12, contentW))
			y += 14
		}
		y = sectionTable(page, th, section, margin, y, contentW) + 11
	}
	if len(d.Approvals) > 0 {
		if y+62 > footTop-10 {
			page = doc.AddPage()
			y = page.Letterhead(th, d.Branding.Company, logo, margin, margin, contentW, true) + 15
		}
		page.Text(margin, y, pdfkit.FontBold, 9, th.Title, "APROVAÇÕES E RESPONSABILIDADES")
		y += 12
		boxW := (contentW - 20) / 3
		for i, a := range d.Approvals {
			x, top := margin+float64(i%3)*(boxW+10), y+float64(i/3)*42
			page.StrokeRect(x, top, boxW, 34, .6, th.Rule)
			page.Text(x+8, top+12, pdfkit.FontBold, 7.5, th.Muted, strings.ToUpper(a.Label))
			page.Text(x+8, top+26, pdfkit.FontRegular, 9, th.Text, fit(a.Value, pdfkit.FontRegular, 9, boxW-16))
		}
	}
	if d.Disclaimer != "" {
		page.Text(margin, footTop-18, pdfkit.FontOblique, 7.5, th.Muted, fit(d.Disclaimer, pdfkit.FontOblique, 7.5, contentW))
	}
	return doc.Render()
}

func scannerBarcode(p *pdfkit.Page, th pdfkit.Theme, d ManufacturingOrder, x, y, w float64) float64 {
	const boxH = 70.0
	p.StrokeRect(x, y, w, boxH, .7, th.Rule)
	p.FillRect(x, y, 5, boxH, th.Brand)
	p.Text(x+14, y+14, pdfkit.FontBold, 8, th.Title, "LEITURA DA ORDEM DE FABRICAÇÃO")
	validity := "Token validado no servidor"
	if d.BarcodeValidUntil != nil {
		validity = "Válido até " + d.BarcodeValidUntil.Format("02/01/2006 15:04")
	}
	p.TextRight(x+w-14, y+14, pdfkit.FontRegular, 7.5, th.Muted, validity)
	if err := p.DrawCode128B(d.BarcodeValue, x+14, y+21, w-28, 34, pdfkit.Black); err != nil {
		p.Text(x+14, y+43, pdfkit.FontBold, 8, th.Text, "Código indisponível: "+err.Error())
	}
	p.TextCenter(x+w/2, y+65, pdfkit.FontRegular, 7.2, th.Muted, maskedToken(d.BarcodeValue))
	return y + boxH + 8
}

func maskedToken(token string) string {
	const visible = 8
	if len(token) <= visible*2 {
		return token
	}
	return token[:visible] + "…" + token[len(token)-visible:]
}

func orderTitle(p *pdfkit.Page, th pdfkit.Theme, d ManufacturingOrder, x, y, w float64) float64 {
	const statusW = 94.0
	p.Text(x, y+13, pdfkit.FontBold, 17, th.Title, fit(d.Title, pdfkit.FontBold, 17, w-statusW-14))
	p.FillRect(x+w-statusW, y, statusW, 23, th.Brand)
	p.TextCenter(x+w-statusW/2, y+15, pdfkit.FontBold, 8.5, th.BrandText, strings.ToUpper(d.Status))
	y += 28
	p.Text(x, y, pdfkit.FontBold, 11, th.Text, d.Number)
	if d.Subtitle != "" {
		p.Text(x+145, y, pdfkit.FontRegular, 9, th.Muted, fit(d.Subtitle, pdfkit.FontRegular, 9, w-145))
	}
	return y + 7
}

func summaryCards(p *pdfkit.Page, th pdfkit.Theme, fields []Field, x, y, w float64) float64 {
	if len(fields) == 0 {
		return y
	}
	const cols = 4
	cardW := (w - float64(cols-1)*8) / cols
	rows := (len(fields) + cols - 1) / cols
	for i, f := range fields {
		cx, cy := x+float64(i%cols)*(cardW+8), y+float64(i/cols)*43
		p.FillRect(cx, cy, cardW, 36, th.Zebra)
		p.StrokeRect(cx, cy, cardW, 36, .5, th.Rule)
		p.FillRect(cx, cy, 4, 36, th.Brand)
		p.Text(cx+11, cy+12, pdfkit.FontBold, 7, th.Muted, strings.ToUpper(f.Label))
		p.Text(cx+11, cy+27, pdfkit.FontBold, 9.5, th.Text, fit(f.Value, pdfkit.FontBold, 9.5, cardW-20))
	}
	return y + float64(rows)*43
}

func sectionTable(p *pdfkit.Page, th pdfkit.Theme, s OrderSection, x, y, w float64) float64 {
	p.Text(x, y+9, pdfkit.FontBold, 10, th.Title, strings.ToUpper(s.Title))
	y += 15
	ws := widths(s.Columns, w)
	p.FillRect(x, y, w, 17, th.Brand)
	cx := x
	for i, c := range s.Columns {
		cell(p, c.Title, pdfkit.FontBold, 7.7, th.BrandText, c.Align, cx, y+11.5, ws[i])
		cx += ws[i]
	}
	y += 17
	for i, row := range s.Rows {
		if i%2 == 1 {
			p.FillRect(x, y, w, 16, th.Zebra)
		}
		cx = x
		for col, c := range s.Columns {
			v := ""
			if col < len(row) {
				v = row[col]
			}
			cell(p, v, pdfkit.FontRegular, 8, th.Text, c.Align, cx, y+11, ws[col])
			cx += ws[col]
		}
		y += 16
	}
	p.StrokeLine(x, y, x+w, y, .5, th.Rule)
	return y
}

func RenderItemStructure(s ItemStructure) []byte {
	doc := pdfkit.NewLandscape()
	th := theme(s.Branding)
	logo, _ := doc.AddImage(s.Branding.Logo)
	const margin = pdfkit.Margin
	w, h := doc.Size()
	contentW, footTop := w-2*margin, h-margin+4
	generated := s.Generated
	if generated.IsZero() {
		generated = time.Now()
	}
	doc.SetFooter(func(p *pdfkit.Page, page, total int) {
		p.StrokeLine(margin, footTop-9, w-margin, footTop-9, .5, th.Rule)
		p.Text(margin, footTop+2, pdfkit.FontRegular, 7.5, th.Muted, "Estrutura de produto gerada pelo VentureERP em "+generated.Format("02/01/2006 15:04"))
		p.TextRight(w-margin, footTop+2, pdfkit.FontRegular, 7.5, th.Muted, fmt.Sprintf("Página %d de %d", page, total))
	})
	page := doc.AddPage()
	y := page.Letterhead(th, s.Branding.Company, logo, margin, margin, contentW, false) + 14
	page.Text(margin, y+13, pdfkit.FontBold, 17, th.Title, fit(s.Title, pdfkit.FontBold, 17, contentW))
	y += 30
	base := []Field{{"Item raiz", s.RootCode}, {"Revisão", s.Revision}, {"Quantidade-base", s.BaseQty}}
	y = summaryCards(page, th, append(base, s.Summary...), margin, y, contentW) + 4
	cols := []Column{{"Nível", 46, pdfkit.AlignCenter}, {"Posição", 68, pdfkit.AlignLeft}, {"Código", 118, pdfkit.AlignLeft}, {"Descrição e hierarquia", 270, pdfkit.AlignLeft}, {"Qtd.", 66, pdfkit.AlignRight}, {"UM", 34, pdfkit.AlignCenter}, {"Perda", 52, pdfkit.AlignRight}, {"Origem", 84, pdfkit.AlignLeft}, {"Prazo", 58, pdfkit.AlignLeft}, {"Custo", 76, pdfkit.AlignRight}}
	ws := widths(cols, contentW)
	drawHeader := func() {
		page.FillRect(margin, y, contentW, 18, th.Brand)
		cx := margin
		for i, c := range cols {
			cell(page, c.Title, pdfkit.FontBold, 7.7, th.BrandText, c.Align, cx, y+12, ws[i])
			cx += ws[i]
		}
		y += 18
	}
	drawHeader()
	for i, n := range s.Nodes {
		if y+18 > footTop-12 {
			page = doc.AddPage()
			y = page.Letterhead(th, s.Branding.Company, logo, margin, margin, contentW, true) + 14
			page.Text(margin, y, pdfkit.FontBold, 12, th.Title, s.Title+" — "+s.RootCode+" (continuação)")
			y += 10
			drawHeader()
		}
		if i%2 == 1 {
			page.FillRect(margin, y, contentW, 18, th.Zebra)
		}
		vals := []string{fmt.Sprintf("%d", n.Level), n.Position, n.Code, n.Description, n.Quantity, n.Unit, n.Loss, n.Origin, n.LeadTime, n.UnitCost}
		cx := margin
		for c, col := range cols {
			cellX, cellW := cx, ws[c]
			if c == 3 {
				indent := float64(n.Level) * 13
				page.FillRect(cx+4+indent, y+5, 4, 8, levelColor(th, n.Level))
				cellX, cellW = cx+9+indent, ws[c]-9-indent
			}
			cell(page, vals[c], pdfkit.FontRegular, 8, th.Text, col.Align, cellX, y+12, cellW)
			cx += ws[c]
		}
		y += 18
	}
	page.StrokeLine(margin, y, margin+contentW, y, .5, th.Rule)
	return doc.Render()
}

func levelColor(th pdfkit.Theme, level int) pdfkit.Color {
	if level == 0 {
		return th.Brand
	}
	if level == 1 {
		return pdfkit.Color{R: 67, G: 160, B: 71}
	}
	if level == 2 {
		return pdfkit.Color{R: 102, G: 187, B: 106}
	}
	return pdfkit.Color{R: 165, G: 214, B: 167}
}
func widths(cols []Column, total float64) []float64 {
	sum := 0.0
	for _, c := range cols {
		if c.Weight > 0 {
			sum += c.Weight
		} else {
			sum++
		}
	}
	out := make([]float64, len(cols))
	for i, c := range cols {
		weight := c.Weight
		if weight <= 0 {
			weight = 1
		}
		out[i] = total * weight / sum
	}
	return out
}
func cell(p *pdfkit.Page, s string, font pdfkit.Font, size float64, color pdfkit.Color, align pdfkit.Align, x, baseline, width float64) {
	const pad = 4.0
	s = fit(s, font, size, width-2*pad)
	switch align {
	case pdfkit.AlignRight:
		p.TextRight(x+width-pad, baseline, font, size, color, s)
	case pdfkit.AlignCenter:
		p.TextCenter(x+width/2, baseline, font, size, color, s)
	default:
		p.Text(x+pad, baseline, font, size, color, s)
	}
}
func fit(s string, font pdfkit.Font, size, maxW float64) string {
	if pdfkit.TextWidth(font, size, s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && pdfkit.TextWidth(font, size, string(r)+"…") > maxW {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}
