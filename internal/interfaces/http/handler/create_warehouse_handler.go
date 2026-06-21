package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/go-chi/chi/v5"
)

func (h *WarehouseHandler) CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req request.CreateWarehouseRequestDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	warehouse, err := h.createWarehouseUC.Execute(r.Context(), req)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, warehouse, "warehouse created succesfully")
}

func (h *WarehouseHandler) ListWarehouses(w http.ResponseWriter, r *http.Request) {
	list, err := h.listWarehousesUC.Execute(r.Context())
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, list)
}

func (h *WarehouseHandler) GetWarehouse(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	wh, err := h.getWarehouseUC.Execute(r.Context(), code)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, wh)
}
