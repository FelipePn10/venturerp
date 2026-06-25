// Package export turns tabular data into the file formats companies expect for
// reports — CSV, XLSX (Excel) and PDF — with no external dependencies.
//
// The whole package is built around a single neutral value, Table, so any list
// or report endpoint can be exported the same way. Encoders are deliberately
// kept dependency-free (encoding/csv, archive/zip and a hand-rolled PDF writer)
// so the binary stays lean and offline-buildable with the vendored tree.
package export

import (
	"fmt"
	"strings"
	"time"
)

// Format is one of the supported export formats.
type Format string

const (
	FormatCSV  Format = "csv"
	FormatXLSX Format = "xlsx"
	FormatPDF  Format = "pdf"
	FormatDOCX Format = "docx"
)

// ParseFormat normalises a user-supplied format string. It accepts a few common
// aliases (xls→xlsx, excel→xlsx, word/doc→docx) and reports whether the value
// was recognised.
func ParseFormat(s string) (Format, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "csv":
		return FormatCSV, true
	case "xlsx", "xls", "excel":
		return FormatXLSX, true
	case "pdf":
		return FormatPDF, true
	case "docx", "doc", "word":
		return FormatDOCX, true
	default:
		return "", false
	}
}

// Table is the neutral representation every encoder consumes. Rows are plain
// strings already formatted for display; the caller owns locale/number
// formatting so the encoders stay simple and predictable.
type Table struct {
	Title       string     // report heading (e.g. "Clientes")
	Subtitle    string     // optional second line (filters, company, period)
	Columns     []string   // header labels
	Rows        [][]string // each row must have len(Columns) cells
	GeneratedAt time.Time  // stamped on PDF/XLSX footers; defaults to now
	Branding    *Branding  // optional company letterhead (PDF/XLSX/DOCX)
}

// Branding is the company letterhead rendered at the top of PDF, XLSX and DOCX
// exports so reports look like the documents an industrial company issues. Only
// non-sensitive identifying data lives here; it is filled server-side from the
// company's fiscal configuration, never by the client. CSV stays branding-free
// on purpose, as it is a raw data-interchange format.
type Branding struct {
	CompanyName string
	CNPJ        string
	IE          string
	Address     string // single pre-formatted line (street, city/UF, CEP)
	Phone       string
	Email       string

	// BrandColorHex tints the PDF letterhead/table header (e.g. "#1B3A5B").
	// Empty falls back to the default corporate navy.
	BrandColorHex string
	// Logo is the company logo as raw PNG or JPEG bytes, embedded in the PDF
	// letterhead. Optional; a decode failure simply omits the image.
	Logo []byte
}

// infoLines returns the letterhead's secondary lines (identification, address,
// contact), each already joined for display. Empty fields are skipped so the
// block never shows dangling labels.
func (b *Branding) infoLines() []string {
	var lines []string
	var ids []string
	if b.CNPJ != "" {
		ids = append(ids, "CNPJ: "+b.CNPJ)
	}
	if b.IE != "" {
		ids = append(ids, "IE: "+b.IE)
	}
	if len(ids) > 0 {
		lines = append(lines, strings.Join(ids, "   "))
	}
	if b.Address != "" {
		lines = append(lines, b.Address)
	}
	var contact []string
	if b.Phone != "" {
		contact = append(contact, "Tel: "+b.Phone)
	}
	if b.Email != "" {
		contact = append(contact, "Email: "+b.Email)
	}
	if len(contact) > 0 {
		lines = append(lines, strings.Join(contact, "   "))
	}
	return lines
}

// Validate guards the invariants the encoders rely on.
func (t *Table) Validate() error {
	if len(t.Columns) == 0 {
		return fmt.Errorf("export: table has no columns")
	}
	for i, row := range t.Rows {
		if len(row) != len(t.Columns) {
			return fmt.Errorf("export: row %d has %d cells, expected %d", i, len(row), len(t.Columns))
		}
	}
	return nil
}

// normalize fills defaults so encoders never have to second-guess the table.
func (t *Table) normalize() {
	if t.GeneratedAt.IsZero() {
		t.GeneratedAt = time.Now()
	}
	if t.Title == "" {
		t.Title = "Relatório"
	}
}

// ContentType returns the MIME type for a format.
func (f Format) ContentType() string {
	switch f {
	case FormatCSV:
		return "text/csv; charset=utf-8"
	case FormatXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case FormatPDF:
		return "application/pdf"
	case FormatDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

// Extension returns the filename extension (without dot) for a format.
func (f Format) Extension() string { return string(f) }
