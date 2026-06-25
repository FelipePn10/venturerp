package romaneio

import (
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

// GenerateRomaneioPDF renders a romaneio de expedição as a professional PDF:
// branded letterhead, framed party/transport sections, a coloured/zebra item
// table, totals, signature lines and a paginated footer — all via the shared
// pdfkit so it matches the rest of the system's documents.
func GenerateRomaneioPDF(d *RomaneioData) ([]byte, error) {
	doc := pdfkit.New()
	th := pdfkit.DefaultTheme()
	if c, ok := pdfkit.ParseHexColor(d.BrandColorHex); ok {
		th.Brand, th.Title = c, c
	}
	var logo *pdfkit.Image
	if len(d.Logo) > 0 {
		logo, _ = doc.AddImage(d.Logo)
	}

	w, h := doc.Size()
	m := pdfkit.Margin
	contentW := w - 2*m
	footTop := h - m + 4

	note := "Documento gerado em " + d.GeneratedAt.Format("02/01/2006 15:04:05")
	doc.SetFooter(func(p *pdfkit.Page, num, total int) {
		p.Footer(th, m, footTop, w-m, note, num, total)
	})

	co := companyFrom(d.Enterprise)
	page := doc.AddPage()
	y := page.Letterhead(th, co, logo, m, m, contentW, false)

	// Title.
	y += 22
	page.TextCenter(w/2, y, pdfkit.FontBold, 15, th.Title, d.Title)
	if d.Subtitle != "" {
		y += 13
		page.TextCenter(w/2, y, pdfkit.FontRegular, 9, th.Muted, d.Subtitle)
	}
	y += 16

	// Romaneio header data.
	headerLines := []string{
		fmt.Sprintf("Nº: %d        Data: %s        Status: %s", d.Code, d.Date.Format("02/01/2006"), d.Status),
		fmt.Sprintf("Referência: %s %d", d.ReferenceType, d.ReferenceCode),
	}
	if d.NFeNumber > 0 || d.NFeKey != "" {
		nfe := "NF-e: "
		if d.NFeNumber > 0 {
			nfe += fmt.Sprintf("%d", d.NFeNumber)
		}
		if d.NFeKey != "" {
			nfe += "    Chave: " + d.NFeKey
		}
		headerLines = append(headerLines, nfe)
	}
	y = section(page, th, m, y, contentW, "DADOS DO ROMANEIO", headerLines)

	// Recipient / sender.
	if d.Destinatario.Name != "" {
		lines := []string{d.Destinatario.Name + "    " + d.Destinatario.CNPJCPF}
		if addr := addressLine(d.Destinatario); addr != "" {
			lines = append(lines, addr)
		}
		y = section(page, th, m, y, contentW, "DESTINATÁRIO / REMETENTE", lines)
	}

	// Carrier.
	if d.Carrier.Name != "" || d.Carrier.Plate != "" {
		carrierLines := []string{
			d.Carrier.Name + "    " + d.Carrier.CNPJCPF,
			fmt.Sprintf("Placa: %s    Motorista: %s    ANTT: %s    Frete: %s",
				d.Carrier.Plate, d.Carrier.Driver, d.Carrier.ANTT, d.Carrier.FreightType),
		}
		if d.Seals != "" {
			carrierLines = append(carrierLines, "Lacres: "+d.Seals)
		}
		y = section(page, th, m, y, contentW, "TRANSPORTADORA", carrierLines)
	}

	// Items table (paginates if needed).
	y += 4
	page, y = drawItemsTable(doc, page, th, co, logo, m, y, contentW, d.Items, footTop)

	// Volumes / packing detail.
	if len(d.Volumes) > 0 {
		y += 10
		page, y = drawVolumesTable(doc, page, th, co, logo, m, y, contentW, d.Volumes, footTop)
	}

	// Totals (right aligned block).
	y += 10
	totalsX := m + contentW*0.55
	page.StrokeLine(totalsX, y, m+contentW, y, 0.5, th.Rule)
	y += 12
	tline := func(label, val string, bold bool) {
		font := pdfkit.FontRegular
		if bold {
			font = pdfkit.FontBold
		}
		page.Text(totalsX, y, font, 9, th.Text, label)
		page.TextRight(m+contentW, y, font, 9, th.Text, val)
		y += 13
	}
	tline("Peso Líquido", fmt.Sprintf("%.3f kg", d.TransportInfo.NetWeight), false)
	tline("Peso Bruto", fmt.Sprintf("%.3f kg", d.TransportInfo.GrossWeight), false)
	tline("Volumes", fmt.Sprintf("%.0f %s", d.TransportInfo.VolumeQuantity, d.TransportInfo.VolumeType), false)
	if d.TotalGross > 0 {
		tline("Total Bruto", fmt.Sprintf("R$ %s", money(d.TotalGross)), true)
		tline("Total Líquido", fmt.Sprintf("R$ %s", money(d.TotalNet)), true)
	}
	if d.TransportInfo.FreightValue > 0 {
		tline("Frete", fmt.Sprintf("R$ %s", money(d.TransportInfo.FreightValue)), false)
	}
	if d.TransportInfo.InsuranceValue > 0 {
		tline("Seguro", fmt.Sprintf("R$ %s", money(d.TransportInfo.InsuranceValue)), false)
	}
	if d.TransportInfo.EstimatedDelivery != "" {
		tline("Previsão de Entrega", d.TransportInfo.EstimatedDelivery, false)
	}

	// Signatures.
	y += 24
	drawSignatures(page, th, m, y, contentW)

	return doc.Render(), nil
}

// section draws a brand title bar plus its content lines, returning the y below.
func section(p *pdfkit.Page, th pdfkit.Theme, x, top, w float64, label string, lines []string) float64 {
	p.FillRect(x, top, w, 14, th.Brand)
	p.Text(x+6, top+10, pdfkit.FontBold, 8.5, th.BrandText, label)
	y := top + 14
	p.StrokeRect(x, top, w, 14+float64(len(lines))*12+4, 0.5, th.Rule)
	y += 11
	for _, ln := range lines {
		p.Text(x+6, y, pdfkit.FontRegular, 9, th.Text, ln)
		y += 12
	}
	return y + 8
}

// ---- items table ----

type itemCol struct {
	label  string
	align  pdfkit.Align
	weight float64
}

var itemCols = []itemCol{
	{"Seq", pdfkit.AlignLeft, 0.7},
	{"Código", pdfkit.AlignLeft, 1.2},
	{"Descrição", pdfkit.AlignLeft, 4.2},
	{"NCM", pdfkit.AlignLeft, 1.6},
	{"CFOP", pdfkit.AlignLeft, 1.0},
	{"Qtd", pdfkit.AlignRight, 1.2},
	{"UN", pdfkit.AlignCenter, 0.7},
	{"V.Unit", pdfkit.AlignRight, 1.5},
	{"V.Total", pdfkit.AlignRight, 1.6},
	{"ICMS", pdfkit.AlignRight, 1.1},
	{"IPI", pdfkit.AlignRight, 1.1},
	{"Peso L.", pdfkit.AlignRight, 1.3},
}

const itemRowH = 13.0

func drawItemsTable(doc *pdfkit.Doc, page *pdfkit.Page, th pdfkit.Theme, co pdfkit.Company, logo *pdfkit.Image, x, y, w float64, items []RomaneioItem, footTop float64) (*pdfkit.Page, float64) {
	widths := itemWidths(w)
	y = itemsHeader(page, th, widths, x, y)
	zebra := false
	for _, it := range items {
		if y+itemRowH > footTop-90 {
			page = doc.AddPage()
			top := page.Letterhead(th, co, logo, x, pdfkit.Margin, w, true)
			y = top + 12
			y = itemsHeader(page, th, widths, x, y)
			zebra = false
		}
		vals := []string{
			fmt.Sprintf("%d", it.Sequence),
			fmt.Sprintf("%d", it.ItemCode),
			it.Description,
			it.NCM,
			it.CFOP,
			fmt.Sprintf("%.2f", it.Quantity),
			it.Unit,
			money(it.UnitPrice),
			money(it.TotalPrice),
			fmt.Sprintf("%.1f%%", it.ICMSPct),
			fmt.Sprintf("%.1f%%", it.IPIPct),
			fmt.Sprintf("%.3f", it.WeightNet),
		}
		if zebra {
			page.FillRect(x, y, w, itemRowH, th.Zebra)
		}
		cx := x
		for i, v := range vals {
			cell(page, v, pdfkit.FontRegular, 7.5, th.Text, itemCols[i].align, cx, y+9, widths[i])
			cx += widths[i]
		}
		y += itemRowH
		zebra = !zebra
	}
	page.StrokeLine(x, y, x+w, y, 0.5, th.Rule)
	return page, y
}

var volCols = []itemCol{
	{"Vol", pdfkit.AlignLeft, 0.8},
	{"Espécie", pdfkit.AlignLeft, 2.0},
	{"Peso Líq (kg)", pdfkit.AlignRight, 2.0},
	{"Peso Bruto (kg)", pdfkit.AlignRight, 2.2},
	{"Dimensões (cm)", pdfkit.AlignCenter, 2.6},
	{"Cubagem (m³)", pdfkit.AlignRight, 2.0},
	{"Marca", pdfkit.AlignLeft, 2.4},
}

// drawVolumesTable renders the packing detail (handling units) as a branded,
// zebra table, paginating like the items table.
func drawVolumesTable(doc *pdfkit.Doc, page *pdfkit.Page, th pdfkit.Theme, co pdfkit.Company, logo *pdfkit.Image, x, y, w float64, volumes []RomaneioVolume, footTop float64) (*pdfkit.Page, float64) {
	page.Text(x, y, pdfkit.FontBold, 9, th.Title, "VOLUMES")
	y += 6
	widths := genericWidths(volCols, w)
	y = genericHeader(page, th, volCols, widths, x, y)
	zebra := false
	for _, v := range volumes {
		if y+itemRowH > footTop-40 {
			page = doc.AddPage()
			top := page.Letterhead(th, co, logo, x, pdfkit.Margin, w, true)
			y = top + 12
			y = genericHeader(page, th, volCols, widths, x, y)
			zebra = false
		}
		dims := fmt.Sprintf("%.0f x %.0f x %.0f", v.LengthCm, v.WidthCm, v.HeightCm)
		vals := []string{
			fmt.Sprintf("%d", v.Number),
			v.PackageType,
			money(v.NetWeight),
			money(v.GrossWeight),
			dims,
			fmt.Sprintf("%.3f", v.CubageM3),
			v.Marking,
		}
		if zebra {
			page.FillRect(x, y, w, itemRowH, th.Zebra)
		}
		cx := x
		for i, val := range vals {
			cell(page, val, pdfkit.FontRegular, 7.5, th.Text, volCols[i].align, cx, y+9, widths[i])
			cx += widths[i]
		}
		y += itemRowH
		zebra = !zebra
	}
	page.StrokeLine(x, y, x+w, y, 0.5, th.Rule)
	return page, y
}

// genericWidths/genericHeader render any itemCol-based table header.
func genericWidths(cols []itemCol, total float64) []float64 {
	s := 0.0
	for _, c := range cols {
		s += c.weight
	}
	out := make([]float64, len(cols))
	for i, c := range cols {
		out[i] = total * c.weight / s
	}
	return out
}

func genericHeader(p *pdfkit.Page, th pdfkit.Theme, cols []itemCol, widths []float64, x, top float64) float64 {
	p.FillRect(x, top, sum(widths), 15, th.Brand)
	cx := x
	for i, c := range cols {
		cell(p, c.label, pdfkit.FontBold, 7.5, th.BrandText, c.align, cx, top+10.5, widths[i])
		cx += widths[i]
	}
	return top + 15
}

func itemsHeader(p *pdfkit.Page, th pdfkit.Theme, widths []float64, x, top float64) float64 {
	p.FillRect(x, top, sum(widths), 15, th.Brand)
	cx := x
	for i, c := range itemCols {
		cell(p, c.label, pdfkit.FontBold, 7.5, th.BrandText, c.align, cx, top+10.5, widths[i])
		cx += widths[i]
	}
	return top + 15
}

func itemWidths(total float64) []float64 {
	s := 0.0
	for _, c := range itemCols {
		s += c.weight
	}
	out := make([]float64, len(itemCols))
	for i, c := range itemCols {
		out[i] = total * c.weight / s
	}
	return out
}

// cell draws aligned, ellipsised text inside [cx, cx+width].
func cell(p *pdfkit.Page, s string, font pdfkit.Font, size float64, c pdfkit.Color, align pdfkit.Align, cx, baseline, width float64) {
	const pad = 3.0
	inner := width - 2*pad
	for pdfkit.TextWidth(font, size, s) > inner && len([]rune(s)) > 1 {
		r := []rune(s)
		s = string(r[:len(r)-1])
	}
	switch align {
	case pdfkit.AlignRight:
		p.TextRight(cx+width-pad, baseline, font, size, c, s)
	case pdfkit.AlignCenter:
		p.TextCenter(cx+width/2, baseline, font, size, c, s)
	default:
		p.Text(cx+pad, baseline, font, size, c, s)
	}
}

func drawSignatures(p *pdfkit.Page, th pdfkit.Theme, x, top, w float64) {
	sigW := (w - 40) / 3
	p.Text(x, top, pdfkit.FontBold, 9, th.Text, "ASSINATURAS")
	top += 28
	labels := []string{"Emitente", "Transportadora", "Destinatário"}
	for i, lbl := range labels {
		lx := x + float64(i)*(sigW+20)
		p.StrokeLine(lx, top, lx+sigW, top, 0.5, th.Text)
		p.TextCenter(lx+sigW/2, top+11, pdfkit.FontRegular, 8.5, th.Muted, lbl)
	}
}

// ---- mapping helpers ----

func companyFrom(c CompanyInfo) pdfkit.Company {
	return pdfkit.Company{
		Name:    c.Name,
		CNPJ:    c.CNPJCPF,
		IE:      c.IE,
		Address: addressLine(c),
		Phone:   c.Phone,
		Email:   c.Email,
	}
}

func addressLine(c CompanyInfo) string {
	var street string
	if c.Street != "" {
		street = c.Street
		if c.Number != "" {
			street += ", " + c.Number
		}
	}
	parts := make([]string, 0, 3)
	if street != "" {
		parts = append(parts, street)
	}
	if c.District != "" {
		parts = append(parts, c.District)
	}
	if c.City != "" {
		city := c.City
		if c.UF != "" {
			city += "/" + c.UF
		}
		parts = append(parts, city)
	}
	addr := strings.Join(parts, " - ")
	if c.CEP != "" {
		if addr != "" {
			addr += "  "
		}
		addr += "CEP " + c.CEP
	}
	return addr
}

// money formats a value with pt-BR thousands/decimal separators (1.234,56).
func money(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%.2f", v) // "1234.56"
	intPart, dec := s[:len(s)-3], s[len(s)-2:]
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	out := b.String() + "," + dec
	if neg {
		out = "-" + out
	}
	return out
}

func sum(xs []float64) float64 {
	t := 0.0
	for _, x := range xs {
		t += x
	}
	return t
}
