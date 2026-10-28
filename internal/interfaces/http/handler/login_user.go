package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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
		var escolha *user_uc.ErrEmpresaObrigatoria
		if errors.As(err, &escolha) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": "EMPRESA_OBRIGATORIA", "error": "Você tem acesso a mais de uma empresa: escolha em qual entrar.",
				"empresas": escolha.Empresas,
			})
			return
		}
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// O vencimento vai na resposta: sem ele o cliente só descobre que a sessão
	// caiu levando um 401 no meio do trabalho, e não tem como renovar antes.
	token, expiraEm, err := auth.GenerateSessionToken(
		userID, role, enterpriseID, authVersion, h.dataEnvironment, h.jwtSecret, time.Time{}, login.RememberMe,
	)
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
		"expires_at":  expiraEm.Format(time.RFC3339),
		"remember_me": login.RememberMe,
	})
}
