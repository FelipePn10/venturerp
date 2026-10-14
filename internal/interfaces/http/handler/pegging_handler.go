package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// PeggingHandler expõe o pegging do último cálculo de um plano.
type PeggingHandler struct {
	uc *mrp_calculation_uc.PeggingUseCase
}

func NewPeggingHandler(uc *mrp_calculation_uc.PeggingUseCase) *PeggingHandler {
	return &PeggingHandler{uc: uc}
}

func peggingPlan(w http.ResponseWriter, r *http.Request) (int64, bool) {
	plan, err := strconv.ParseInt(chi.URLParam(r, "plan_code"), 10, 64)
	if err != nil || plan <= 0 {
		security.RespondErrorCode(w, http.StatusBadRequest, "PEGGING_PLANO_INVALIDO", "plano inválido")
		return 0, false
	}
	return plan, true
}

func optionalInt64(w http.ResponseWriter, raw, campo string) (*int64, bool) {
	if raw == "" {
		return nil, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "PEGGING_PARAMETRO_INVALIDO", campo+" inválido")
		return nil, false
	}
	return &v, true
}

// Roots: GET /api/mrp-calculation/pegging/{plan_code}/roots
func (h *PeggingHandler) Roots(w http.ResponseWriter, r *http.Request) {
	plan, ok := peggingPlan(w, r)
	if !ok {
		return
	}
	out, err := h.uc.Roots(r.Context(), plan)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// Down: GET .../down?demand_kind=PEDIDO_VENDA&demand_code=&item_code=&need_date=
func (h *PeggingHandler) Down(w http.ResponseWriter, r *http.Request) {
	plan, ok := peggingPlan(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	code, ok := optionalInt64(w, q.Get("demand_code"), "demand_code")
	if !ok {
		return
	}
	item, ok := optionalInt64(w, q.Get("item_code"), "item_code")
	if !ok {
		return
	}
	var need *time.Time
	if raw := q.Get("need_date"); raw != "" {
		d, err := time.Parse("2006-01-02", raw)
		if err != nil {
			security.RespondErrorCode(w, http.StatusBadRequest, "PEGGING_PARAMETRO_INVALIDO", "need_date inválida: use ano-mês-dia")
			return
		}
		need = &d
	}
	out, err := h.uc.Down(r.Context(), plan, q.Get("demand_kind"), code, item, need)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// Up: GET .../up?supply_kind=SUGESTAO|ORDEM_FIRME&supply_code=
func (h *PeggingHandler) Up(w http.ResponseWriter, r *http.Request) {
	plan, ok := peggingPlan(w, r)
	if !ok {
		return
	}
	code, ok := optionalInt64(w, r.URL.Query().Get("supply_code"), "supply_code")
	if !ok {
		return
	}
	if code == nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "PEGGING_PARAMETRO_INVALIDO", "informe supply_code")
		return
	}
	out, err := h.uc.Up(r.Context(), plan, r.URL.Query().Get("supply_kind"), *code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}

// SalesOrder: GET /api/mrp-calculation/pegging/sales-order/{order_code} — a
// cadeia de atendimento dos itens do pedido em cada plano que o considerou.
func (h *PeggingHandler) SalesOrder(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "order_code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondErrorCode(w, http.StatusBadRequest, "PEGGING_PEDIDO_INVALIDO", "pedido inválido")
		return
	}
	out, err := h.uc.SalesOrder(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, out)
}
