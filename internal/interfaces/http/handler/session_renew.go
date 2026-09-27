package handler

import (
	"encoding/json"
	"net/http"
	"time"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// RenewSessionHandler troca um token válido por outro com prazo cheio.
//
// É o que faz o "manter conectado" existir de verdade. O token vale 24 h e não
// havia renovação: quem abre o ERP todo dia tinha de digitar a senha de novo, e
// a caixa "manter conectado" na tela de login não mudava nada.
//
// A rota passa pelo mesmo middleware de sempre, então o token já foi conferido
// contra o banco (perfil, empresa e `auth_version` — trocar a senha continua
// derrubando a sessão). O que muda aqui é só o prazo, e ele nunca passa do teto
// contado do login de verdade (`auth.SessionMaxDuration`): renovar não pode
// virar uma sessão eterna.
func (h *UserHandler) RenewSessionHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextkey.UserKey).(*appsecurity.AuthUser)
	if !ok || user == nil || user.ID == "" {
		http.Error(w, `{"error": "sessão não identificada"}`, http.StatusUnauthorized)
		return
	}

	sessao, _ := r.Context().Value(contextkey.SessionKey).(appsecurity.SessionInfo)
	inicio := sessao.Start
	if !inicio.IsZero() && time.Since(inicio) >= auth.SessionMaxDuration {
		// Fim do "manter conectado": a senha volta a ser pedida.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":  "SESSION_EXPIRED",
			"error": "a sessão passou do prazo máximo de 30 dias; entre com a senha novamente",
		})
		return
	}

	// `auth_version` vem do que o middleware já validou contra o banco: reemitir
	// com o valor do token antigo manteria uma sessão que deveria ter caído.
	authVersion := int64(0)
	if h.loginUC != nil && h.loginUC.Repo != nil {
		atual, err := h.loginUC.Repo.CurrentAuthorization(r.Context(), user.ID, user.EnterpriseID)
		if err != nil {
			h.InternalError(w, r, err)
			return
		}
		authVersion = atual.AuthVersion
	}

	token, expiraEm, err := auth.GenerateSessionToken(
		user.ID, user.Role, user.EnterpriseID, authVersion, h.dataEnvironment, h.jwtSecret, inicio, sessao.Remember,
	)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"remember_me":   sessao.Remember,
		"token":         token,
		"expires_at":    expiraEm.Format(time.RFC3339),
		"session_start": sessionStartOrNow(inicio).Format(time.RFC3339),
		"role":          user.Role,
		"environment":   h.dataEnvironment,
	})
}

func sessionStartOrNow(inicio time.Time) time.Time {
	if inicio.IsZero() {
		return time.Now()
	}
	return inicio
}
