package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc"
)

func (h *UserHandler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var login request.RegisterUserDTO

	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.registerUC.Execute(r.Context(), login); err != nil {
		if errors.Is(err, user_uc.ErrRegisterUserForbidden) {
			h.writeJSON(w, http.StatusForbidden, map[string]string{
				"error":   "forbidden",
				"message": "cadastro restrito à empresa autenticada",
			})
			return
		}
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			h.writeJSON(w, http.StatusConflict, map[string]string{
				"error":   "conflict",
				"message": "e-mail já cadastrado",
			})
			return
		}
		h.InternalError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *UserHandler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
