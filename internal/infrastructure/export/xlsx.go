package export

import (
	"archive/zip"
	"io"
	"strconv"
	"strings"
)

// EncodeXLSX writes the table as a minimal, valid .xlsx workbook.
//
// An .xlsx file is a ZIP of OOXML parts. We emit only what Excel/LibreOffice
// require to open the file: content types, package + workbook relationships,
// a workbook with one sheet, a tiny stylesheet (bold header), and the sheet
// itself. Cell values use inline strings (t="inlineStr"), which avoids a
// shared-strings table while still round-tripping text faithfully.
func EncodeXLSX(w io.Writer, t *Table) error {
	if err := t.Validate(); err != nil {
		return err
	}
	t.normalize()

	zw := zip.NewWriter(w)
	write := func(name, body string) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(f, body)
		return err
	}

	if err := write("[Content_Types].xml", contentTypesXML); err != nil {
		return err
	}
	if err := write("_rels/.rels", packageRelsXML); err != nil {
		return err
	}
	if err := write("xl/workbook.xml", workbookXML); err != nil {
		return err
	}
	if err := write("xl/_rels/workbook.xml.rels", workbookRelsXML); err != nil {
		return err
	}
	if err := write("xl/styles.xml", stylesXML); err != nil {
		return err
	}
	if err := write("xl/worksheets/sheet1.xml", sheetXML(t)); err != nil {
		return err
	}
	return zw.Close()
}

// sheetXML renders the rows. The optional letterhead and the report title sit
// above the table; the column header row uses style index 1 (bold).
func sheetXML(t *Table) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	b.WriteString(`<sheetData>`)

	row := 1
	textRow := func(text string, style int) {
		b.WriteString(`<row r="` + strconv.Itoa(row) + `">`)
		b.WriteString(inlineCell(cellRef(0, row), text, style))
		b.WriteString(`</row>`)
		row++
	}

	// Letterhead.
	if br := t.Branding; br != nil && br.CompanyName != "" {
		textRow(br.CompanyName, 1)
		for _, ln := range br.infoLines() {
			textRow(ln, 0)
		}
		row++ // blank spacer
	}

	// Title + subtitle/timestamp.
	textRow(t.Title, 1)
	meta := "Gerado em " + t.GeneratedAt.Format("02/01/2006 15:04")
	if t.Subtitle != "" {
		meta = t.Subtitle + " • " + meta
	}
	textRow(meta, 0)
	row++ // blank spacer before the table

	// Column header.
	b.WriteString(`<row r="` + strconv.Itoa(row) + `">`)
	for c, col := range t.Columns {
		b.WriteString(inlineCell(cellRef(c, row), col, 1))
	}
	b.WriteString(`</row>`)
	row++

	// Data rows.
	for _, dataRow := range t.Rows {
		b.WriteString(`<row r="` + strconv.Itoa(row) + `">`)
		for c, val := range dataRow {
			b.WriteString(inlineCell(cellRef(c, row), val, 0))
		}
		b.WriteString(`</row>`)
		row++
	}

	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

// inlineCell builds a single inline-string cell. style 0 = default, 1 = bold.
func inlineCell(ref, value string, style int) string {
	s := ""
	if style != 0 {
		s = ` s="` + strconv.Itoa(style) + `"`
	}
	return `<c r="` + ref + `"` + s + ` t="inlineStr"><is><t xml:space="preserve">` +
		escapeXML(value) + `</t></is></c>`
}

// cellRef converts a 0-based column and 1-based row into an A1 reference.
func cellRef(col, row int) string {
	return columnName(col) + strconv.Itoa(row)
}

// columnName converts a 0-based column index into spreadsheet letters (A, B, …,
// Z, AA, AB, …).
func columnName(col int) string {
	name := ""
	for col >= 0 {
		name = string(rune('A'+col%26)) + name
		col = col/26 - 1
	}
	return name
}

// escapeXML escapes the five XML predefined entities.
func escapeXML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
	`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
	`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
	`</Types>`

const packageRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
	`</Relationships>`

const workbookXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
	`<sheets><sheet name="Relatorio" sheetId="1" r:id="rId1"/></sheets>` +
	`</workbook>`

const workbookRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
	`</Relationships>`

// stylesXML defines two cell formats: index 0 (default) and index 1 (bold),
// used for the header row.
const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
	`<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font>` +
	`<font><b/><sz val="11"/><name val="Calibri"/></font></fonts>` +
	`<fills count="1"><fill><patternFill patternType="none"/></fill></fills>` +
	`<borders count="1"><border/></borders>` +
	`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
	`<cellXfs count="2">` +
	`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
	`<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
	`</cellXfs>` +
	`</styleSheet>`
