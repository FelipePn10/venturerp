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
)

// ParseFormat normalises a user-supplied format string. It accepts a few common
// aliases (xls→xlsx, excel→xlsx) and reports whether the value was recognised.
func ParseFormat(s string) (Format, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "csv":
		return FormatCSV, true
	case "xlsx", "xls", "excel":
		return FormatXLSX, true
	case "pdf":
		return FormatPDF, true
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
	default:
		return "application/octet-stream"
	}
}

// Extension returns the filename extension (without dot) for a format.
func (f Format) Extension() string { return string(f) }
