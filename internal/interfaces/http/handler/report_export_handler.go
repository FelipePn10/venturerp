package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// fiscalConfigReader is the slice of the fiscal repository this handler needs:
// the company's own fiscal/registration data, used to brand exported reports.
type fiscalConfigReader interface {
	GetFiscalConfig(ctx context.Context) (*fiscalentity.FiscalConfig, error)
}

// ReportExportHandler turns any client-supplied table into a downloadable file.
// The front-end already renders lists/reports; this endpoint lets it ship the
// exact rows the user sees to Excel, PDF, Word or CSV without each module needing
// a bespoke export route. The professional letterhead (company data, generation
// stamp) is injected here, server-side, so the client never sends company data.
type ReportExportHandler struct {
	*security.BaseHandler
	fiscal fiscalConfigReader
}

// NewReportExportHandler builds the handler. fiscal may be nil, in which case
// exports are produced without a company letterhead.
func NewReportExportHandler(fiscal fiscalConfigReader) *ReportExportHandler {
	return &ReportExportHandler{BaseHandler: &security.BaseHandler{}, fiscal: fiscal}
}

// exportRequest is the generic payload: a title plus the columns and rows to
// render. Rows whose length differs from the header are rejected by the encoder.
type exportRequest struct {
	Title    string     `json:"title"`
	Subtitle string     `json:"subtitle"`
	Filename string     `json:"filename"`
	Columns  []string   `json:"columns"`
	Rows     [][]string `json:"rows"`
}

// Export handles POST /api/reports/export?format=xlsx|pdf|csv|docx.
func (h *ReportExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.BadRequest(w, "invalid request body")
		return
	}
	if len(req.Columns) == 0 {
		h.BadRequest(w, "columns is required")
		return
	}

	table := &export.Table{
		Title:    req.Title,
		Subtitle: req.Subtitle,
		Columns:  req.Columns,
		Rows:     req.Rows,
		Branding: h.branding(r.Context()),
	}

	base := req.Filename
	if base == "" {
		base = req.Title
	}
	if err := export.WriteHTTP(w, r, table, base); err != nil {
		// WriteHTTP already wrote a 400 on validation errors; nothing else to do.
		return
	}
}

// branding loads the company letterhead from the fiscal configuration. It fails
// soft: any error (or no configured reader) simply yields an unbranded export
// rather than blocking the download.
func (h *ReportExportHandler) branding(ctx context.Context) *export.Branding {
	if h.fiscal == nil {
		return nil
	}
	cfg, err := h.fiscal.GetFiscalConfig(ctx)
	if err != nil || cfg == nil || cfg.RazaoSocial == "" {
		return nil
	}
	return brandingFromConfig(cfg)
}

// brandingFromConfig maps the company's fiscal config into the export letterhead.
func brandingFromConfig(c *fiscalentity.FiscalConfig) *export.Branding {
	b := &export.Branding{
		CompanyName: c.RazaoSocial,
		CNPJ:        formatCNPJMask(c.CnpjEmpresa),
		Address:     formatCompanyAddress(c),
		Logo:        c.Logo,
	}
	if c.IEEmpresa != nil {
		b.IE = *c.IEEmpresa
	}
	if c.Telefone != nil {
		b.Phone = *c.Telefone
	}
	if c.BrandColor != nil {
		b.BrandColorHex = *c.BrandColor
	}
	return b
}

// formatCompanyAddress assembles the single address line for the letterhead,
// skipping empty parts so it never shows stray separators.
func formatCompanyAddress(c *fiscalentity.FiscalConfig) string {
	var street string
	if c.Logradouro != "" {
		street = c.Logradouro
		if c.Numero != "" {
			street += ", " + c.Numero
		}
	}
	parts := make([]string, 0, 4)
	if street != "" {
		parts = append(parts, street)
	}
	if c.Bairro != "" {
		parts = append(parts, c.Bairro)
	}
	if c.Municipio != "" {
		city := c.Municipio
		if c.UFEmpresa != "" {
			city += "/" + c.UFEmpresa
		}
		parts = append(parts, city)
	}
	addr := strings.Join(parts, " - ")
	if c.CEP != "" {
		if addr != "" {
			addr += "  "
		}
		addr += "CEP " + formatCEPMask(c.CEP)
	}
	return addr
}

// formatCNPJMask renders a 14-digit CNPJ as 00.000.000/0000-00; non-conforming
// input is returned unchanged.
func formatCNPJMask(s string) string {
	d := digitsOnly(s)
	if len(d) != 14 {
		return s
	}
	return d[0:2] + "." + d[2:5] + "." + d[5:8] + "/" + d[8:12] + "-" + d[12:14]
}

// formatCEPMask renders an 8-digit CEP as 00000-000; other input is unchanged.
func formatCEPMask(s string) string {
	d := digitsOnly(s)
	if len(d) != 8 {
		return s
	}
	return d[0:5] + "-" + d[5:8]
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
