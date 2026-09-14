package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"io"
	"log/slog"
	"math"
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
	"item_base_cod": {}, "item_codes": {}, "item_from": {}, "item_to": {},
}

func isItemReferenceKey(r *http.Request, key string) bool {
	if key != "parent_code" && key != "child_code" {
		_, ok := itemReferenceKeys[key]
		return ok
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	return path == "/api/items/structure" || strings.HasPrefix(path, "/api/items/structure/")
}

// isItemInputReferenceKey decide o que é traduzido na ENTRADA (corpo e query).
//
// Difere da saída num ponto: `parent_code`/`child_code` da estrutura não são
// traduzidos aqui. O caso de uso da estrutura já resolve o código público por
// conta própria (itemresolution.Resolve: comercial primeiro, chave interna como
// alternativa), e traduzir também no middleware fazia a conversão acontecer
// duas vezes — o interno 5 produzido aqui era reinterpretado como o comercial
// "5" e virava o interno 2. Numa base onde códigos comerciais numéricos
// convivem com chaves internas ("1", "5", "8" ao lado dos internos 1..15), o
// componente era gravado sob OUTRO item: a gravação respondia 201 e a tela, ao
// recarregar o item certo, não encontrava nada.
//
// Na SAÍDA a tradução continua valendo: a tela lê e exibe código público.
func isItemInputReferenceKey(r *http.Request, key string) bool {
	if key == "parent_code" || key == "child_code" {
		path := strings.TrimSuffix(r.URL.Path, "/")
		if path == "/api/items/structure" || strings.HasPrefix(path, "/api/items/structure/") {
			return false
		}
	}
	return isItemReferenceKey(r, key)
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
			// A tradução do CAMINHO acontece em ItemPathTranslation, que é
			// middleware de raiz e roda antes do roteamento. Aqui ela chegaria
			// tarde demais (e traduzir de novo converteria o código errado).
			if !nativeItemBusinessCodeRequest(r) {
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
				body = translateItemResponse(r, pool, enterpriseID, body)
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

func nativeItemBusinessCodeRequest(r *http.Request) bool {
	path := strings.TrimSuffix(r.URL.Path, "/")
	return path == "/api/machine/time/create" || path == "/api/machine/time/list"
}

func nativeItemBusinessCodePath(r *http.Request) bool {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "items" {
		return false
	}

	// Estes handlers já recebem e resolvem o código comercial. Traduzir o
	// caminho antes deles cria uma colisão perigosa: RN-01001 pode virar a chave
	// interna 5 e, se existir um item cujo código comercial seja "5", o handler
	// abre esse outro item. A VENT0210 usa exatamente search e structure/resolve.
	if len(parts) == 4 && parts[2] == "search" {
		return true
	}
	if len(parts) == 5 && parts[2] == "structure" && parts[3] == "resolve" {
		return true
	}

	if len(parts) != 3 && len(parts) != 4 {
		return false
	}
	if parts[2] == "create" || parts[2] == "with-masks" || parts[2] == "structure" {
		return false
	}
	return len(parts) == 3 || (len(parts) == 4 && parts[3] == "activation-readiness")
}

// itemPathSegmentIndex devolve a posição do segmento do caminho que carrega o
// código do item, ou -1 quando a rota não referencia item. Separado para que o
// teste de guarda possa cobrar cada rota nova de api.go.
func itemPathSegmentIndex(parts []string) int {
	if len(parts) < 3 || parts[0] != "api" {
		return -1
	}
	if parts[1] == "items" {
		switch {
		case parts[2] == "search":
			if len(parts) < 4 {
				return -1
			}
			return 3
		case parts[2] == "structure":
			if len(parts) >= 5 && parts[3] == "resolve" {
				return 4
			}
			return -1
		case isStaticItemsPath(parts[2]):
			return -1
		}
		return 2
	}
	for _, pattern := range itemPathPatterns {
		if len(parts) <= pattern.index || len(parts) < len(pattern.prefix) {
			continue
		}
		matches := true
		for i, value := range pattern.prefix {
			// "*" casa qualquer segmento: algumas rotas têm um código variável
			// antes do código do item (.../sales-tables/{tableCode}/prices/…).
			if value != "*" && parts[i] != value {
				matches = false
				break
			}
		}
		if matches {
			return pattern.index
		}
	}
	return -1
}

var itemPathPatterns = []struct {
	prefix []string
	index  int
}{
	{[]string{"api", "stock", "movements", "item"}, 4},
	{[]string{"api", "stock", "balances", "item"}, 4},
	{[]string{"api", "stock", "balances", "atp"}, 4},
	{[]string{"api", "stock", "lots", "item"}, 4},
	{[]string{"api", "stock", "lots", "genealogy"}, 4},
	{[]string{"api", "stock", "consumption-average"}, 3},
	{[]string{"api", "bom-headers", "item"}, 3},
	{[]string{"api", "restriction", "item"}, 3},
	{[]string{"api", "sales-forecast", "item"}, 3},
	{[]string{"api", "item-conversions", "item"}, 3},
	{[]string{"api", "item-suppliers", "item"}, 3},
	{[]string{"api", "drawings", "item-code"}, 3},
	{[]string{"api", "standard-cost", "purchase-costs"}, 3},
	{[]string{"api", "quality", "records", "by-item"}, 4},
	{[]string{"api", "quality", "non-conformances", "by-item"}, 4},
	{[]string{"api", "mrp-reports", "explosion"}, 3},
	{[]string{"api", "independent-demand", "list-by-item"}, 3},
	{[]string{"api", "configurator", "items"}, 3},
	{[]string{"api", "quality", "plans", "by-item"}, 4},
	{[]string{"api", "standard-cost", "items"}, 3},
	{[]string{"api", "mrp-calculation", "profile"}, 3},
	{[]string{"api", "item-calendar-promise"}, 2},
	{[]string{"api", "financial", "relatorios", "ficha-tecnica"}, 4},
	{[]string{"api", "mrp-calculation", "configured-rules"}, 3},
	{[]string{"api", "fiscal", "support", "parametros-icms-ipi", "item"}, 5},
	{[]string{"api", "customers", "support", "sales-tables", "*", "prices"}, 6},
	{[]string{"api", "stock", "separation", "suggest"}, 4},
	{[]string{"api", "stock", "putaway", "suggest"}, 4},
}

func translateKnownItemURLPath(r *http.Request, pool *pgxpool.Pool, enterpriseID int64) error {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	index := itemPathSegmentIndex(parts)
	if index < 0 {
		return nil
	}
	publicCode := parts[index]
	id, err := resolveBusinessCode(r.Context(), pool, enterpriseID, publicCode)
	if err != nil {
		return err
	}
	parts[index] = strconv.FormatInt(id, 10)
	r.URL.Path = "/" + strings.Join(parts, "/")
	r.URL.RawPath = ""
	// In a chi middleware registered inside a routed Group, the outer router has
	// already captured URL parameters before this middleware runs. Rewriting only
	// URL.Path therefore leaves chi.URLParam with the public code. Keep both views
	// consistent so the real API router (not just an isolated middleware router)
	// hands the immutable internal ID to legacy handlers.
	interno := strconv.FormatInt(id, 10)
	if rc := chi.RouteContext(r.Context()); rc != nil {
		for i, key := range rc.URLParams.Keys {
			switch key {
			case "code", "itemCode", "item_code":
				if rc.URLParams.Values[i] == publicCode {
					rc.URLParams.Values[i] = interno
				}
			case "*":
				// Este é o caso que quebrava o estoque. `r.Route("/api/stock", …)`
				// monta um sub-roteador atrás de um curinga: quando este
				// middleware roda, o chi casou apenas `/api/stock/*` e guardou o
				// resto ("movements/item/MP-CH-3MM") neste parâmetro. É dele que
				// o sub-roteador tira a sub-rota — reescrever só `r.URL.Path`
				// não muda nada (embora o log passe a mostrar o código já
				// traduzido, o que despista a investigação).
				rc.URLParams.Values[i] = trocaSegmento(rc.URLParams.Values[i], publicCode, interno)
			}
		}
		if rc.RoutePath != "" {
			rc.RoutePath = trocaSegmento(rc.RoutePath, publicCode, interno)
		}
	}
	return nil
}

// trocaSegmento troca o segmento de caminho igual a `de` por `para`, comparando
// segmento inteiro. Um `strings.Replace` solto casaria pedaço de outro segmento
// (o código "10" dentro de "100").
func trocaSegmento(caminho, de, para string) string {
	partes := strings.Split(caminho, "/")
	for i, parte := range partes {
		if parte == de {
			partes[i] = para
		}
	}
	return strings.Join(partes, "/")
}

func isStaticItemsPath(segment string) bool {
	switch segment {
	case "create", "with-masks", "classifications":
		return true
	default:
		return false
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

// A tradução do caminho vive em itemPathSegmentIndex/translateKnownItemURLPath,
// chamada pelo middleware de raiz ItemPathTranslation. Não reintroduza uma
// segunda tradução por parâmetro do chi aqui: com as duas ativas, o código
// interno produzido na primeira era reinterpretado como código comercial na
// segunda e o registro ia parar em OUTRO item, respondendo 201.

func translateItemQuery(r *http.Request, pool *pgxpool.Pool, e int64) error {
	query := r.URL.Query()
	changed := false
	for key, values := range query {
		if !isItemInputReferenceKey(r, key) {
			continue
		}
		for i, value := range values {
			if strings.TrimSpace(value) == "" {
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
			if isItemInputReferenceKey(r, key) {
				if key == "item_code" && strings.HasPrefix(r.URL.Path, "/api/stock/cycle-counts") {
					continue
				}
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
		resolved, err := resolveBusinessCode(r.Context(), pool, e, typed)
		if err != nil {
			return nil, err
		}
		if textualPricingItemContract(r) {
			return strconv.FormatInt(resolved, 10), nil
		}
		return resolved, nil
	case float64:
		if !textualPricingItemContract(r) {
			return value, nil
		}
		if typed <= 0 || math.Trunc(typed) != typed || typed > math.MaxInt64 {
			return nil, fmt.Errorf("o código numérico legado do item é inválido")
		}
		resolved := int64(typed)
		var exists bool
		if err := pool.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM items WHERE enterprise_id=$1 AND code=$2)`,
			e, resolved,
		).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, errorsuc.NewNotFoundError("item não encontrado na empresa autenticada")
		}
		slog.WarnContext(r.Context(), "contrato numérico de item descontinuado",
			"operation", r.Method+" "+r.URL.Path,
			"field", "item_code",
		)
		return strconv.FormatInt(resolved, 10), nil
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

// textualPricingItemContract identifies the commercial pricing endpoints whose
// public contract uses textual item codes. Legacy clients may still send JSON
// numbers temporarily; translateInputReference normalizes both forms to a JSON
// string before the request DTO is decoded.
func textualPricingItemContract(r *http.Request) bool {
	path := r.URL.Path
	return strings.HasPrefix(path, "/api/customers/support/sales-tables/") ||
		strings.HasPrefix(path, "/api/sales-order/items/") ||
		strings.HasPrefix(path, "/api/sales-quotation/items/")
}
func resolveBusinessCode(ctx context.Context, pool *pgxpool.Pool, e int64, code string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT code FROM items WHERE enterprise_id=$1 AND business_code=upper(btrim($2))`, e, code).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return 0, err
	}
	// Alternativa pela chave interna, na mesma ordem que itemresolution.Resolve
	// usa no resto do backend: comercial primeiro, interna depois. Sem isto, um
	// chamador que ainda mande a chave interna no caminho passa a levar 422 —
	// antes ele funcionava por acidente, porque a tradução nem chegava ao
	// handler nas rotas montadas.
	if interno, convErr := strconv.ParseInt(strings.TrimSpace(code), 10, 64); convErr == nil {
		var existe int64
		if pool.QueryRow(ctx, `SELECT code FROM items WHERE enterprise_id=$1 AND code=$2`, e, interno).Scan(&existe) == nil {
			return existe, nil
		}
	}
	return 0, fmt.Errorf("item %q não encontrado na empresa autenticada", code)
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
func translateItemResponse(r *http.Request, pool *pgxpool.Pool, e int64, raw []byte) []byte {
	var payload any
	if json.Unmarshal(raw, &payload) != nil {
		return raw
	}
	cache := map[int64]string{}
	walkOutput(r, pool, e, payload, cache)
	out, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return out
}
func walkOutput(r *http.Request, pool *pgxpool.Pool, e int64, value any, cache map[int64]string) {
	switch node := value.(type) {
	case []any:
		for _, v := range node {
			walkOutput(r, pool, e, v, cache)
		}
	case map[string]any:
		for key, v := range node {
			if isItemReferenceKey(r, key) {
				translated, legacy, ok := translateOutputReference(r.Context(), pool, e, v, cache)
				if ok {
					node["legacy_"+key] = legacy
					node[key] = translated
				}
			}
			walkOutput(r, pool, e, v, cache)
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
