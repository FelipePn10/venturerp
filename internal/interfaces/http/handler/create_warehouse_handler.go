package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
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
