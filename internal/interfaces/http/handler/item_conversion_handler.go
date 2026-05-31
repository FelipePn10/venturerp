package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc"
	"github.com/go-chi/chi/v5"
)

type ItemConversionHandler struct {
	uc *item_conversion_uc.ItemConversionUseCase
}

func NewItemConversionHandler(uc *item_conversion_uc.ItemConversionUseCase) *ItemConversionHandler {
	return &ItemConversionHandler{uc: uc}
}

func (h *ItemConversionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateItemConversionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *ItemConversionHandler) ListByItem(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	res, err := h.uc.ListByItem(r.Context(), itemCode)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *ItemConversionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.Delete(r.Context(), id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Convert resolves a quantity conversion: GET ?item=&from=&to=&qty=
func (h *ItemConversionHandler) Convert(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	itemCode, _ := strconv.ParseInt(q.Get("item"), 10, 64)
	from := q.Get("from")
	to := q.Get("to")
	qty, _ := strconv.ParseFloat(q.Get("qty"), 64)
	if itemCode == 0 || from == "" || to == "" {
		jsonError(w, http.StatusBadRequest, "item, from and to are required")
		return
	}
	factor, found, err := h.uc.Factor(r.Context(), itemCode, from, to)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		jsonError(w, http.StatusNotFound, item_conversion_uc.ErrNoConversion.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{
		"item_code":     itemCode,
		"from_uom":      from,
		"to_uom":        to,
		"factor":        factor,
		"quantity":      qty,
		"converted_qty": qty * factor,
	})
}
