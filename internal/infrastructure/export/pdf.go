package export

import (
	"io"
	"math"
	"regexp"
	"sort"
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

	cols := deriveColumns(t)
	orientation := t.Orientation
	// Sem orientação pedida, tabela larga demais para o retrato vai em paisagem.
	if orientation == "" && naturalWidth(cols) > (pdfkit.A4Width-2*pdfkit.Margin)*1.15 {
		orientation = "paisagem"
	}
	report := &pdfkit.TableReport{
		Orientation: orientation,
		Theme:       theme(t.Branding),
		Company:     company(t.Branding),
		Logo:        logoBytes(t.Branding),
		Title:       t.Title,
		Subtitle:    t.Subtitle,
		Columns:     cols,
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
// columns whose cells are all numeric/currency are right-aligned. Text columns
// are sized by the 90th-percentile cell (not the longest one, which would starve
// every other column) — longer cells wrap — and never below the longest word of
// the header, so titles do not break mid-word.
func deriveColumns(t *Table) []pdfkit.Column {
	cols := make([]pdfkit.Column, len(t.Columns))
	for c, title := range t.Columns {
		numeric := len(t.Rows) > 0
		var ws []float64
		for _, row := range t.Rows {
			if c >= len(row) {
				continue
			}
			cell := strings.TrimSpace(row[c])
			ws = append(ws, pdfkit.TextWidth(pdfkit.FontRegular, 8.5, cell))
			if cell != "" && !isNumericCell(cell) {
				numeric = false
			}
		}
		minW := 0.0
		for _, word := range strings.Fields(title) {
			if w := pdfkit.TextWidth(pdfkit.FontBold, 7.6, word); w > minW {
				minW = w
			}
		}
		natural := percentil(ws, 0.9)
		if numeric {
			natural = percentil(ws, 1)
		}
		natural = math.Min(math.Max(natural, minW), 230)
		align := pdfkit.AlignLeft
		if numeric {
			align = pdfkit.AlignRight
		}
		cols[c] = pdfkit.Column{Title: title, Align: align, Weight: math.Max(natural, 22) + 10}
	}
	return cols
}

// naturalWidth soma a largura que as colunas pedem — decide a orientação.
func naturalWidth(cols []pdfkit.Column) float64 {
	total := 0.0
	for _, c := range cols {
		total += c.Weight
	}
	return total
}

func percentil(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(math.Ceil(p*float64(len(s)))) - 1
	if i < 0 {
		i = 0
	}
	return s[i]
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
