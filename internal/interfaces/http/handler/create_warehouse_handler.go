package handler

import (
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/go-chi/chi/v5"
)

func (h *WarehouseHandler) CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req request.CreateWarehouseRequestDTO

	if !security.DecodeBody(w, r, &req) {
		return
	}

	warehouse, err := h.createWarehouseUC.Execute(r.Context(), req)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, warehouse, "almoxarifado criado com sucesso")
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

// UpdateWarehouse altera o almoxarifado (PUT /api/warehouse/{code}).
func (h *WarehouseHandler) UpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateWarehouseRequestDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UpdateWarehouseUC.Execute(r.Context(), chi.URLParam(r, "code"), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
