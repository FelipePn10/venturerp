package export

import (
	"net/http"
	"strings"
	"time"
)

// Requested reports whether the client asked for a file export, i.e. the
// request carries a recognised `format` (or `export`) query parameter. List
// handlers use this to decide between their normal JSON response and a file.
func Requested(r *http.Request) bool {
	_, ok := formatParam(r)
	return ok
}

func formatParam(r *http.Request) (Format, bool) {
	q := r.URL.Query()
	raw := q.Get("format")
	if raw == "" {
		raw = q.Get("export")
	}
	if raw == "" {
		return "", false
	}
	return ParseFormat(raw)
}

// Encode writes a table to w in the given format.
func Encode(w interface {
	Write([]byte) (int, error)
}, f Format, t *Table) error {
	switch f {
	case FormatCSV:
		return EncodeCSV(w, t)
	case FormatXLSX:
		return EncodeXLSX(w, t)
	case FormatPDF:
		return EncodePDF(w, t)
	default:
		return EncodeCSV(w, t)
	}
}

// WriteHTTP streams the table to the client as a downloadable file, choosing the
// format from the request's `format` query parameter (defaulting to CSV). The
// baseName seeds the download filename (without extension); a timestamp is
// appended so repeated exports don't overwrite each other.
func WriteHTTP(w http.ResponseWriter, r *http.Request, t *Table, baseName string) error {
	f, ok := formatParam(r)
	if !ok {
		f = FormatCSV
	}
	if err := t.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	filename := sanitizeName(baseName) + "_" +
		time.Now().Format("20060102_1504") + "." + f.Extension()

	w.Header().Set("Content-Type", f.ContentType())
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	return Encode(w, f, t)
}

// WriteSlice is the one-liner list endpoints use: reflect the DTO slice into a
// Table and stream it. Returns false (writing nothing) if no export was
// requested, so the caller can fall through to its JSON response.
func WriteSlice(w http.ResponseWriter, r *http.Request, title, baseName string, data any) (bool, error) {
	if !Requested(r) {
		return false, nil
	}
	t, err := TableFromSlice(title, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return true, err
	}
	return true, WriteHTTP(w, r, t, baseName)
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "relatorio"
	}
	return b.String()
}
