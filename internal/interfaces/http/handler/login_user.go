package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
)

func (h *UserHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var login request.LoginUserDTO

	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, role, name, email, enterpriseID, authVersion, err := h.loginUC.Execute(
		r.Context(),
		login,
	)
	if err != nil {
		if errors.Is(err, user_uc.ErrIdentitySync) {
			h.InternalError(w, r, err)
			return
		}
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateTokenForEnvironment(userID, role, enterpriseID, authVersion, h.dataEnvironment, h.jwtSecret)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"token":       token,
		"name":        name,
		"email":       email,
		"role":        role,
		"environment": h.dataEnvironment,
	})
}
