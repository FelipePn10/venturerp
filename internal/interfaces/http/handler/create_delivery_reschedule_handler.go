package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *DeliveryRescheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateDeliveryRescheduleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "Dados inválidos. O código do item deve ser informado como número.")
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *DeliveryRescheduleHandler) ListByOrder(w http.ResponseWriter, r *http.Request) {
	salesOrderCode, _ := strconv.ParseInt(chi.URLParam(r, "sales_order_code"), 10, 64)
	results, err := h.listUC.Execute(r.Context(), salesOrderCode)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *DeliveryRescheduleHandler) Preview(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "sales_order_code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do pedido inválido")
		return
	}
	result, err := h.planningUC.Preview(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *DeliveryRescheduleHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var dto request.IntegratedDeliveryRescheduleBatchDTO
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "dados inválidos para a reprogramação em lote")
		return
	}
	result, err := h.planningUC.CreateBatch(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	security.RespondJSON(w, status, result)
}
