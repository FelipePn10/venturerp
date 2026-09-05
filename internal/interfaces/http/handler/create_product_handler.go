package handler

import (
	"encoding/json"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req request.CreateProductDTO

	if !security.DecodeBody(w, r, &req) {
		return
	}

	product, err := h.createProductUC.Execute(r.Context(), req)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}
