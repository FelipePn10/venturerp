package handler

import (
	"net/http"
	"strconv"
)

func paginate[T any](w http.ResponseWriter, r *http.Request, rows []T) []T {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	total := len(rows)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Offset", strconv.Itoa(offset))
	return rows[start:end]
}
