package romaneio

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"time"
)

func GenerateRomaneioXLSX(d *RomaneioData) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	writeFile := func(name string, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(content))
		return err
	}

	if err := writeFile("[Content_Types].xml", xlsxContentTypes); err != nil {
		return nil, fmt.Errorf("xlsx content types: %w", err)
	}
	if err := writeFile("_rels/.rels", xlsxRels); err != nil {
		return nil, fmt.Errorf("xlsx rels: %w", err)
	}
	if err := writeFile("xl/workbook.xml", xlsxWorkbook); err != nil {
		return nil, fmt.Errorf("xlsx workbook: %w", err)
	}
	if err := writeFile("xl/_rels/workbook.xml.rels", xlsxWorkbookRels); err != nil {
		return nil, fmt.Errorf("xlsx workbook rels: %w", err)
	}
	if err := writeFile("xl/styles.xml", xlsxStyles); err != nil {
		return nil, fmt.Errorf("xlsx styles: %w", err)
	}

	sheet := buildSheet(d)
	if err := writeFile("xl/worksheets/sheet1.xml", sheet); err != nil {
		return nil, fmt.Errorf("xlsx sheet: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("xlsx zip close: %w", err)
	}
	return buf.Bytes(), nil
}

const xlsxContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`

const xlsxRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

const xlsxWorkbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
          xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="Romaneio" sheetId="1" r:id="rId1"/></sheets>
</workbook>`

const xlsxWorkbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

const xlsxStyles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <fonts count="3">
    <font><sz val="10"/><name val="Calibri"/></font>
    <font><b/><sz val="12"/><name val="Calibri"/></font>
    <font><b/><sz val="10"/><name val="Calibri"/></font>
  </fonts>
  <fills count="2">
    <fill><patternFill patternType="none"/></fill>
    <fill><patternFill patternType="gray125"/></fill>
  </fills>
  <borders count="2">
    <border><left/><right/><top/><bottom/><diagonal/></border>
    <border><left/><right/><top style="thin"><color auto="1"/></top><bottom style="thin"><color auto="1"/></bottom><diagonal/></border>
  </borders>
  <cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
  <cellXfs count="3">
    <xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="2" fillId="0" borderId="1" xfId="0"/>
  </cellXfs>
</styleSheet>`

func buildSheet(d *RomaneioData) string {
	var rows []string

	addRow := func(style int, cells ...string) {
		var xmlCells strings.Builder
		col := 'A'
		for _, c := range cells {
			xmlCells.WriteString(fmt.Sprintf(`<c r="%c%d" s="%d" t="inlineStr"><is><t>%s</t></is></c>`,
				col, len(rows)+1, style, xmlEscape(c)))
			col++
		}
		rows = append(rows, fmt.Sprintf("<row r=\"%d\">%s</row>", len(rows)+1, xmlCells.String()))
	}

	addRow(1, d.Title)
	addRow(0, fmt.Sprintf("Romaneio: %d", d.Code))
	addRow(0, fmt.Sprintf("Data: %s", d.Date.Format("02/01/2006")))
	addRow(0, fmt.Sprintf("Status: %s", d.Status))
	addRow(0, fmt.Sprintf("Referencia: %s %d", d.ReferenceType, d.ReferenceCode))
	addRow(0, "")

	addRow(2, "Emitente", "", "", "", "", "", "", "", "", "", "")
	addRow(0, d.Enterprise.Name, d.Enterprise.CNPJCPF, d.Enterprise.IE, d.Enterprise.Street+", "+d.Enterprise.Number, d.Enterprise.City+"/"+d.Enterprise.UF, d.Enterprise.CEP)
	addRow(0, "")

	if d.Destinatario.Name != "" {
		addRow(2, "Destinatario / Remetente", "", "", "", "", "", "", "", "", "", "")
		addRow(0, d.Destinatario.Name, d.Destinatario.CNPJCPF, d.Destinatario.Street+", "+d.Destinatario.Number, d.Destinatario.City+"/"+d.Destinatario.UF)
		addRow(0, "")
	}

	if d.Carrier.Name != "" {
		addRow(2, "Transportadora", "", "", "", "", "", "", "", "", "", "")
		addRow(0, d.Carrier.Name, d.Carrier.CNPJCPF, d.Carrier.Plate, d.Carrier.Driver, d.Carrier.ANTT, d.Carrier.FreightType)
		addRow(0, "")
	}

	addRow(2, "ITENS", "", "", "", "", "", "", "", "", "", "", "")
	addRow(2, "Seq", "Codigo", "Descricao", "NCM", "CFOP", "Qtd", "UN", "V.Unit", "V.Total", "ICMS", "IPI", "Peso Liq")

	for _, it := range d.Items {
		addRow(0,
			fmt.Sprintf("%d", it.Sequence),
			fmt.Sprintf("%d", it.ItemCode),
			it.Description,
			it.NCM,
			it.CFOP,
			fmt.Sprintf("%.2f", it.Quantity),
			it.Unit,
			fmt.Sprintf("%.2f", it.UnitPrice),
			fmt.Sprintf("%.2f", it.TotalPrice),
			fmt.Sprintf("%.2f%%", it.ICMSPct),
			fmt.Sprintf("%.2f%%", it.IPIPct),
			fmt.Sprintf("%.3f", it.WeightNet),
		)
	}

	addRow(0, "")
	addRow(2, "TOTAIS", "", "", "", "", "", "", "", "", "", "", "")
	addRow(0, "", "", "", "", "", "", "", fmt.Sprintf("Total Bruto: R$ %.2f", d.TotalGross), fmt.Sprintf("Total Liq: R$ %.2f", d.TotalNet), "", "", "")
	addRow(0, "", "", "", "", "", "", "", fmt.Sprintf("Peso Liq: %.3f kg", d.TransportInfo.NetWeight), fmt.Sprintf("Peso Bruto: %.3f kg", d.TransportInfo.GrossWeight), "", "")
	addRow(0, "", "", "", "", "", "", "", fmt.Sprintf("Volumes: %.0f %s", d.TransportInfo.VolumeQuantity, d.TransportInfo.VolumeType), fmt.Sprintf("Frete: R$ %.2f", d.TransportInfo.FreightValue), "", "")
	addRow(0, "")
	addRow(0, fmt.Sprintf("Gerado em: %s", d.GeneratedAt.Format("02/01/2006 15:04:05")))

	var buf strings.Builder
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buf.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	buf.WriteString(`<cols>`)
	for i := 0; i < 12; i++ {
		w := 14.0
		switch i {
		case 0:
			w = 6
		case 1:
			w = 10
		case 2:
			w = 36
		case 3:
			w = 12
		case 4:
			w = 8
		case 7:
			w = 14
		case 8:
			w = 14
		}
		buf.WriteString(fmt.Sprintf(`<col min="%d" max="%d" width="%.1f" customWidth="1"/>`, i+1, i+1, w))
	}
	buf.WriteString(`</cols>`)
	buf.WriteString(`<sheetData>`)
	for _, r := range rows {
		buf.WriteString(r)
	}
	buf.WriteString(`</sheetData></worksheet>`)

	return buf.String()
}

func xmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

func init() {
	_ = time.Now()
}
