package export

import (
	"archive/zip"
	"io"
	"strconv"
	"strings"
)

// EncodeDOCX writes the table as a minimal, valid .docx (Word) document.
//
// A .docx file is a ZIP of OOXML parts, just like .xlsx. We emit only the parts
// Word/LibreOffice require to open the file: content types, package
// relationships and a single WordprocessingML document body. The body carries
// the optional company letterhead, the report title/subtitle, and the data as a
// bordered Word table with a shaded, bold header row. No styles part is needed —
// all formatting is inline run/paragraph/table properties — so the writer stays
// dependency-free.
func EncodeDOCX(w io.Writer, t *Table) error {
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

	if err := write("[Content_Types].xml", docxContentTypesXML); err != nil {
		return err
	}
	if err := write("_rels/.rels", docxPackageRelsXML); err != nil {
		return err
	}
	if err := write("word/document.xml", documentXML(t)); err != nil {
		return err
	}
	return zw.Close()
}

// documentXML builds the WordprocessingML body: letterhead, title block and the
// data table.
func documentXML(t *Table) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	b.WriteString(`<w:body>`)

	// Letterhead.
	if br := t.Branding; br != nil && br.CompanyName != "" {
		b.WriteString(docxParagraph(br.CompanyName, true, 24)) // 12pt
		for _, ln := range br.infoLines() {
			b.WriteString(docxParagraph(ln, false, 16)) // 8pt
		}
	}

	// Title + subtitle/timestamp.
	b.WriteString(docxParagraph(t.Title, true, 30)) // 15pt
	meta := "Gerado em " + t.GeneratedAt.Format("02/01/2006 15:04")
	if t.Subtitle != "" {
		meta = t.Subtitle + " • " + meta
	}
	b.WriteString(docxParagraph(meta, false, 16))
	b.WriteString(docxParagraph("", false, 8)) // spacer

	// Data table.
	b.WriteString(docxTable(t))

	// Section properties (A4 portrait, sensible margins).
	b.WriteString(`<w:sectPr>` +
		`<w:pgSz w:w="11906" w:h="16838"/>` +
		`<w:pgMar w:top="720" w:right="720" w:bottom="720" w:left="720" w:header="0" w:footer="0" w:gutter="0"/>` +
		`</w:sectPr>`)

	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

// docxParagraph renders a single-run paragraph. sizeHalfPt is the font size in
// half-points (Word's unit: 24 = 12pt).
func docxParagraph(text string, bold bool, sizeHalfPt int) string {
	var b strings.Builder
	b.WriteString(`<w:p><w:pPr><w:spacing w:after="40" w:line="240" w:lineRule="auto"/></w:pPr>`)
	b.WriteString(docxRun(text, bold, sizeHalfPt))
	b.WriteString(`</w:p>`)
	return b.String()
}

// docxRun renders a text run with optional bold and explicit size.
func docxRun(text string, bold bool, sizeHalfPt int) string {
	var rpr strings.Builder
	rpr.WriteString(`<w:rPr>`)
	if bold {
		rpr.WriteString(`<w:b/>`)
	}
	if sizeHalfPt > 0 {
		sz := strconv.Itoa(sizeHalfPt)
		rpr.WriteString(`<w:sz w:val="` + sz + `"/><w:szCs w:val="` + sz + `"/>`)
	}
	rpr.WriteString(`</w:rPr>`)
	return `<w:r>` + rpr.String() +
		`<w:t xml:space="preserve">` + escapeXML(text) + `</w:t></w:r>`
}

// docxTable renders the table with thin borders and a shaded, bold header row.
func docxTable(t *Table) string {
	var b strings.Builder
	b.WriteString(`<w:tbl>`)
	b.WriteString(`<w:tblPr>` +
		`<w:tblW w:w="0" w:type="auto"/>` +
		`<w:tblBorders>` +
		docxBorder("top") + docxBorder("left") + docxBorder("bottom") +
		docxBorder("right") + docxBorder("insideH") + docxBorder("insideV") +
		`</w:tblBorders>` +
		`</w:tblPr>`)

	// Header row.
	b.WriteString(`<w:tr>`)
	for _, col := range t.Columns {
		b.WriteString(docxCell(col, true, "D9D9D9"))
	}
	b.WriteString(`</w:tr>`)

	// Data rows.
	for _, row := range t.Rows {
		b.WriteString(`<w:tr>`)
		for _, cell := range row {
			b.WriteString(docxCell(cell, false, ""))
		}
		b.WriteString(`</w:tr>`)
	}

	b.WriteString(`</w:tbl>`)
	return b.String()
}

// docxCell renders one table cell, optionally bold and shaded (fill is a hex
// RRGGBB colour, empty for none).
func docxCell(text string, bold bool, fill string) string {
	var tcpr strings.Builder
	tcpr.WriteString(`<w:tcPr><w:tcW w:w="0" w:type="auto"/>`)
	if fill != "" {
		tcpr.WriteString(`<w:shd w:val="clear" w:color="auto" w:fill="` + fill + `"/>`)
	}
	tcpr.WriteString(`</w:tcPr>`)
	return `<w:tc>` + tcpr.String() +
		`<w:p><w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr>` +
		docxRun(text, bold, 16) + `</w:p></w:tc>`
}

// docxBorder is a single thin (4 eighth-points ≈ 0.5pt) black border edge.
func docxBorder(edge string) string {
	return `<w:` + edge + ` w:val="single" w:sz="4" w:space="0" w:color="808080"/>`
}

const docxContentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
	`</Types>`

const docxPackageRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
	`</Relationships>`
