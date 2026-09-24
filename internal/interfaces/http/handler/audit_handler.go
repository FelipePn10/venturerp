package handler

import (
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
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
//	user_id     filter by actor
//	route       filter by chi route pattern
//	method      HTTP verb (POST/PUT/DELETE…) — the "action" the screen shows
//	search      substring of the request path
//	min_status  only records with status >= n (400 = only what failed)
//	from,to     RFC3339 timestamps bounding occurred_at
//	limit       page size (default 100, max 500)
//	offset      pagination offset
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := audit.Filter{
		UserID: q.Get("user_id"),
		Route:  q.Get("route"),
		Method: q.Get("method"),
		Search: q.Get("search"),
	}
	if v := q.Get("min_status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MinStatus = n
		}
	}

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "data inicial inválida: informe data e hora completas")
			return
		}
		f.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "data final inválida: informe data e hora completas")
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
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, records)
}
