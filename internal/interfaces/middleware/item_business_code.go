package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var itemReferenceKeys = map[string]struct{}{
	"item_code": {}, "parent_item_code": {}, "child_item_code": {}, "root_item_code": {},
	"material_item_code": {}, "band_item_code": {}, "scrap_item_code": {}, "service_item_code": {},
	"reference_item_code": {}, "order_item_code": {}, "packaging_item_code": {}, "substituted_item_code": {},
	"parent_code": {}, "child_code": {}, "item_base_cod": {}, "item_codes": {}, "class_item_codes": {}, "item_from": {}, "item_to": {},
}

// ItemBusinessCodeCompatibility translates public alphanumeric item references
// to immutable legacy IDs before handlers and translates IDs back in JSON
// responses. During rollout, every translated response also exposes legacy_*.
func ItemBusinessCodeCompatibility(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enterpriseID, err := tenant.ID(r.Context())
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if err = translateItemPath(r, pool, enterpriseID); err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			if err = translateItemQuery(r, pool, enterpriseID); err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			if requestHasJSON(r) && r.Body != nil {
				if err = translateItemBody(r, pool, enterpriseID); err != nil {
					http.Error(w, err.Error(), http.StatusUnprocessableEntity)
					return
				}
			}
			if bypassItemResponseTranslation(r) {
				next.ServeHTTP(w, r)
				return
			}
			recorder := &itemCodeResponseRecorder{header: make(http.Header)}
			next.ServeHTTP(recorder, r)
			copyHeader(w.Header(), recorder.header)
			body := recorder.body.Bytes()
			if strings.Contains(recorder.header.Get("Content-Type"), "application/json") && len(body) > 0 {
				body = translateItemResponse(r.Context(), pool, enterpriseID, body)
			}
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			_, _ = w.Write(body)
		})
	}
}

func bypassItemResponseTranslation(r *http.Request) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/reports/") || strings.Contains(path, "/download") || strings.Contains(path, "/attachments/") {
		return true
	}
	accept := r.Header.Get("Accept")
	return accept != "" && !strings.Contains(accept, "application/json") && accept != "*/*"
}

func requestHasJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Content-Type"), "application/json")
}
func translateItemPath(r *http.Request, pool *pgxpool.Pool, e int64) error {
	rc := chi.RouteContext(r.Context())
	if rc == nil {
		return nil
	}
	pattern := rc.RoutePattern()
	for i, key := range rc.URLParams.Keys {
		if !isItemReferencePath(pattern, key) {
			continue
		}
		value := rc.URLParams.Values[i]
		if _, err := strconv.ParseInt(value, 10, 64); err == nil {
			continue
		}
		id, err := resolveBusinessCode(r.Context(), pool, e, value)
		if err != nil {
			return err
		}
		rc.URLParams.Values[i] = strconv.FormatInt(id, 10)
	}
	return nil
}

func translateItemQuery(r *http.Request, pool *pgxpool.Pool, e int64) error {
	query := r.URL.Query()
	changed := false
	for key, values := range query {
		if _, ok := itemReferenceKeys[key]; !ok {
			continue
		}
		for i, value := range values {
			if strings.TrimSpace(value) == "" {
				continue
			}
			if _, err := strconv.ParseInt(value, 10, 64); err == nil {
				continue
			}
			parts := strings.Split(value, ",")
			translated := make([]string, len(parts))
			for j, part := range parts {
				id, err := resolveBusinessCode(r.Context(), pool, e, part)
				if err != nil {
					return fmt.Errorf("%s: %w", key, err)
				}
				translated[j] = strconv.FormatInt(id, 10)
			}
			values[i] = strings.Join(translated, ",")
			changed = true
		}
		query[key] = values
	}
	if changed {
		r.URL.RawQuery = query.Encode()
	}
	return nil
}

func isItemReferencePath(pattern, key string) bool {
	if key == "code" {
		return strings.HasPrefix(pattern, "/api/items/")
	}
	if key == "item_code" {
		return strings.HasPrefix(pattern, "/api/mrp-calculation/") ||
			strings.HasPrefix(pattern, "/api/item-calendar-promise/") ||
			strings.Contains(pattern, "/ficha-tecnica/{")
	}
	if key != "itemCode" && key != "parentItemCode" && key != "childItemCode" && key != "rootItemCode" {
		return false
	}
	// These legacy routes call a line/checklist identifier itemCode even though
	// it is not an item master reference. Everything else named itemCode in the
	// public API is a product/item reference and must accept the business code.
	for _, nonItem := range []string{
		"/api/sales-orders/items/{itemCode}",
		"/api/sales-quotations/items/{itemCode}",
		"/api/consumer-service/calls/checklist/{itemCode}",
	} {
		if strings.HasPrefix(pattern, nonItem) {
			return false
		}
	}
	return true
}
func translateItemBody(r *http.Request, pool *pgxpool.Pool, e int64) error {
	const maximumJSONBody = 16 << 20
	raw, err := io.ReadAll(io.LimitReader(r.Body, maximumJSONBody+1))
	if err != nil {
		return err
	}
	_ = r.Body.Close()
	if len(raw) > maximumJSONBody {
		return fmt.Errorf("corpo JSON excede o limite de 16 MiB")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		r.Body = io.NopCloser(bytes.NewReader(raw))
		return nil
	}
	var payload any
	if err = json.Unmarshal(raw, &payload); err != nil {
		r.Body = io.NopCloser(bytes.NewReader(raw))
		return nil
	}
	if err = walkInput(r, pool, e, payload); err != nil {
		return err
	}
	translated, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(translated))
	r.ContentLength = int64(len(translated))
	return nil
}
func walkInput(r *http.Request, pool *pgxpool.Pool, e int64, value any) error {
	switch node := value.(type) {
	case []any:
		for _, v := range node {
			if err := walkInput(r, pool, e, v); err != nil {
				return err
			}
		}
	case map[string]any:
		for key, v := range node {
			if _, ok := itemReferenceKeys[key]; ok {
				translated, err := translateInputReference(r, pool, e, v)
				if err != nil {
					return fmt.Errorf("%s: %w", key, err)
				}
				node[key] = translated
			}
			if err := walkInput(r, pool, e, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func translateInputReference(r *http.Request, pool *pgxpool.Pool, e int64, value any) (any, error) {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return value, nil
		}
		return resolveBusinessCode(r.Context(), pool, e, typed)
	case []any:
		out := make([]any, len(typed))
		for i, v := range typed {
			translated, err := translateInputReference(r, pool, e, v)
			if err != nil {
				return nil, err
			}
			out[i] = translated
		}
		return out, nil
	default:
		return value, nil
	}
}
func resolveBusinessCode(ctx context.Context, pool *pgxpool.Pool, e int64, code string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT code FROM items WHERE enterprise_id=$1 AND business_code=upper(btrim($2))`, e, code).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("item %q nao encontrado na empresa autenticada", code)
		}
		return 0, err
	}
	return id, nil
}

type itemCodeResponseRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (r *itemCodeResponseRecorder) Header() http.Header { return r.header }
func (r *itemCodeResponseRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}
func (r *itemCodeResponseRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(p)
}
func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}
func translateItemResponse(ctx context.Context, pool *pgxpool.Pool, e int64, raw []byte) []byte {
	var payload any
	if json.Unmarshal(raw, &payload) != nil {
		return raw
	}
	cache := map[int64]string{}
	walkOutput(ctx, pool, e, payload, cache)
	out, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return out
}
func walkOutput(ctx context.Context, pool *pgxpool.Pool, e int64, value any, cache map[int64]string) {
	switch node := value.(type) {
	case []any:
		for _, v := range node {
			walkOutput(ctx, pool, e, v, cache)
		}
	case map[string]any:
		for key, v := range node {
			if _, ok := itemReferenceKeys[key]; ok {
				translated, legacy, ok := translateOutputReference(ctx, pool, e, v, cache)
				if ok {
					node["legacy_"+key] = legacy
					node[key] = translated
				}
			}
			walkOutput(ctx, pool, e, v, cache)
		}
	}
}

func translateOutputReference(ctx context.Context, pool *pgxpool.Pool, e int64, value any, cache map[int64]string) (any, any, bool) {
	switch typed := value.(type) {
	case float64:
		if typed <= 0 || typed != float64(int64(typed)) {
			return nil, nil, false
		}
		id := int64(typed)
		code, found := cache[id]
		if !found {
			if pool.QueryRow(ctx, `SELECT business_code FROM items WHERE enterprise_id=$1 AND code=$2`, e, id).Scan(&code) != nil {
				return nil, nil, false
			}
			cache[id] = code
		}
		return code, id, true
	case []any:
		translated := make([]any, len(typed))
		legacy := make([]any, len(typed))
		changed := false
		for i, v := range typed {
			tv, lv, ok := translateOutputReference(ctx, pool, e, v, cache)
			if ok {
				translated[i] = tv
				legacy[i] = lv
				changed = true
			} else {
				translated[i] = v
				legacy[i] = v
			}
		}
		return translated, legacy, changed
	default:
		return nil, nil, false
	}
}
