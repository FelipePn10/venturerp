package export

import (
	"io"
	"regexp"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

// EncodePDF renders the table as a professional, paginated PDF: a branded
// company letterhead (logo + colour band), the report title and filters, then
// the data as a table with a coloured header band, zebra rows and a paginated
// footer. Layout is delegated to the dependency-free pdfkit, which uses
// proportional Helvetica with real font metrics so columns and numbers align.
func EncodePDF(w io.Writer, t *Table) error {
	if err := t.Validate(); err != nil {
		return err
	}
	t.normalize()

	report := &pdfkit.TableReport{
		Theme:       theme(t.Branding),
		Company:     company(t.Branding),
		Logo:        logoBytes(t.Branding),
		Title:       t.Title,
		Subtitle:    t.Subtitle,
		Columns:     deriveColumns(t),
		Rows:        t.Rows,
		GeneratedAt: t.GeneratedAt,
	}

	_, err := w.Write(report.Render())
	return err
}

func theme(b *Branding) pdfkit.Theme {
	th := pdfkit.DefaultTheme()
	if b != nil && b.BrandColorHex != "" {
		if c, ok := pdfkit.ParseHexColor(b.BrandColorHex); ok {
			th.Brand = c
			th.Title = c
		}
	}
	return th
}

func company(b *Branding) pdfkit.Company {
	if b == nil {
		return pdfkit.Company{}
	}
	return pdfkit.Company{
		Name:    b.CompanyName,
		CNPJ:    b.CNPJ,
		IE:      b.IE,
		Address: b.Address,
		Phone:   b.Phone,
		Email:   b.Email,
	}
}

func logoBytes(b *Branding) []byte {
	if b == nil {
		return nil
	}
	return b.Logo
}

// deriveColumns infers each column's alignment and relative width from the data:
// columns whose cells are all numeric/currency are right-aligned, and widths are
// weighted by the widest rendered cell so long text columns get more room.
func deriveColumns(t *Table) []pdfkit.Column {
	cols := make([]pdfkit.Column, len(t.Columns))
	for c, title := range t.Columns {
		numeric := len(t.Rows) > 0
		maxW := pdfkit.TextWidth(pdfkit.FontBold, 8.5, title)
		for _, row := range t.Rows {
			if c >= len(row) {
				continue
			}
			cell := row[c]
			if w := pdfkit.TextWidth(pdfkit.FontRegular, 8.5, cell); w > maxW {
				maxW = w
			}
			if strings.TrimSpace(cell) != "" && !isNumericCell(cell) {
				numeric = false
			}
		}
		align := pdfkit.AlignLeft
		if numeric {
			align = pdfkit.AlignRight
		}
		cols[c] = pdfkit.Column{Title: title, Align: align, Weight: maxW + 8}
	}
	return cols
}

var numericCell = regexp.MustCompile(`^[R$\s]*-?[\d.]+(,\d+)?%?$`)

// isNumericCell reports whether a display string looks like a pt-BR number,
// currency or percentage (e.g. "R$ 12.500,00", "1.234", "9,5%").
func isNumericCell(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	return numericCell.MatchString(s)
}
