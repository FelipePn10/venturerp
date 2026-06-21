package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/export"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// ReportExportHandler turns any client-supplied table into a downloadable file.
// The front-end already renders lists/reports; this endpoint lets it ship the
// exact rows the user sees to Excel, PDF or CSV without each module needing a
// bespoke export route.
type ReportExportHandler struct {
	*security.BaseHandler
}

func NewReportExportHandler() *ReportExportHandler {
	return &ReportExportHandler{BaseHandler: &security.BaseHandler{}}
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

// Export handles POST /api/reports/export?format=xlsx|pdf|csv.
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
