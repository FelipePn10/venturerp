package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/audit"
)

// AuditHandler exposes read access to the audit trail. Writes happen
// automatically in middleware; this is the query side for administrators.
type AuditHandler struct {
	reader *audit.Reader
}

func NewAuditHandler(reader *audit.Reader) *AuditHandler {
	return &AuditHandler{reader: reader}
}

// List returns audit records, newest first. Query params (all optional):
//
//	user_id  filter by actor
//	route    filter by chi route pattern
//	from,to  RFC3339 timestamps bounding occurred_at
//	limit    page size (default 100, max 500)
//	offset   pagination offset
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := audit.Filter{
		UserID: q.Get("user_id"),
		Route:  q.Get("route"),
	}

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid 'from' (use RFC3339)")
			return
		}
		f.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid 'to' (use RFC3339)")
			return
		}
		f.To = t
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Offset = n
		}
	}

	records, err := h.reader.List(r.Context(), f)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, records)
}
