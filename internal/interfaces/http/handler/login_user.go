package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
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
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(userID, role, enterpriseID, authVersion, h.jwtSecret)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"token": token,
		"name":  name,
		"email": email,
		"role":  role,
	})
}
