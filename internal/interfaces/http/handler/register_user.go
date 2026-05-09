package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func (h *UserHandler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var login request.RegisterUserDTO

	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.registerUC.Execute(
		r.Context(),
		login,
	); err != nil {
		h.InternalError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
