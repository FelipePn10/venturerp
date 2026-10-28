package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// FonteDeAcesso lê os perfis ativos do usuário na empresa.
type FonteDeAcesso interface {
	AcessoDoUsuario(ctx context.Context, userID uuid.UUID, enterpriseID int64) (acesso.AcessoDoUsuario, error)
}

// ControleDeAcesso aplica os perfis de acesso da empresa depois do papel.
// Guarda o acesso de cada usuário por alguns segundos (a consulta seria feita
// em toda requisição) e esquece tudo quando um perfil ou vínculo muda.
type ControleDeAcesso struct {
	fonte FonteDeAcesso
	ttl   time.Duration
	mu    sync.Mutex
	cache map[string]acessoEmCache
}

type acessoEmCache struct {
	acesso acesso.AcessoDoUsuario
	ate    time.Time
}

func NewControleDeAcesso(fonte FonteDeAcesso) *ControleDeAcesso {
	return &ControleDeAcesso{fonte: fonte, ttl: 15 * time.Second, cache: map[string]acessoEmCache{}}
}

// Invalidar descarta o acesso guardado (chamado ao gravar perfil ou vínculo).
func (c *ControleDeAcesso) Invalidar() {
	c.mu.Lock()
	c.cache = map[string]acessoEmCache{}
	c.mu.Unlock()
}

// PodeNoModulo responde se o usuário da sessão pode a ação no módulo — para o
// caso de uso que decide pela área (ex.: liberar a visão fiscal do item exige
// aprovar no módulo fiscal), além da permissão da rota.
func (c *ControleDeAcesso) PodeNoModulo(ctx context.Context, m acesso.Modulo, a acesso.Acao) (bool, error) {
	user, ok := ctx.Value(contextkey.UserKey).(*security.AuthUser)
	if !ok || user == nil {
		return false, nil
	}
	ac, err := c.AcessoDe(ctx, user)
	if err != nil {
		return false, err
	}
	return ac.Pode(m, a), nil
}

// AcessoDe devolve o acesso do usuário na empresa, do cache quando recente.
func (c *ControleDeAcesso) AcessoDe(ctx context.Context, user *security.AuthUser) (acesso.AcessoDoUsuario, error) {
	if strings.EqualFold(user.Role, "ADMIN") {
		return acesso.AcessoDoUsuario{}, nil
	}
	uid, err := uuid.Parse(user.ID)
	if err != nil {
		return acesso.AcessoDoUsuario{}, fmt.Errorf("usuário da sessão inválido: %w", err)
	}
	chave := fmt.Sprintf("%s|%d", uid, user.EnterpriseID)
	c.mu.Lock()
	if v, ok := c.cache[chave]; ok && time.Now().Before(v.ate) {
		c.mu.Unlock()
		return v.acesso, nil
	}
	c.mu.Unlock()
	a, err := c.fonte.AcessoDoUsuario(ctx, uid, user.EnterpriseID)
	if err != nil {
		return acesso.AcessoDoUsuario{}, err
	}
	c.mu.Lock()
	c.cache[chave] = acessoEmCache{acesso: a, ate: time.Now().Add(c.ttl)}
	c.mu.Unlock()
	return a, nil
}

// Middleware recusa (403) o que o perfil do usuário não permite. Roda depois do
// JWT; o papel (RequireRole) continua valendo em cada rota.
func (c *ControleDeAcesso) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser)
		if !ok || strings.EqualFold(user.Role, "ADMIN") {
			next.ServeHTTP(w, r)
			return
		}
		rota := ClassificarRota(r.Method, r.URL.Path)
		if rota.Livre {
			next.ServeHTTP(w, r)
			return
		}
		a, err := c.AcessoDe(r.Context(), user)
		if err != nil {
			slog.Error("lendo os perfis de acesso do usuário", "usuario", user.ID, "empresa", user.EnterpriseID, "erro", err)
			respondJSONErro(w, http.StatusServiceUnavailable, "ACESSO_INDISPONIVEL",
				"não foi possível conferir o seu perfil de acesso agora; tente novamente")
			return
		}
		if !a.Restrito {
			next.ServeHTTP(w, r)
			return
		}
		if !rota.Conhecida {
			respondJSONErro(w, http.StatusForbidden, "ACESSO_NEGADO",
				"esta operação ainda não pertence a nenhum módulo de permissão; peça ao administrador")
			return
		}
		if rota.LeituraReferencia || a.Pode(rota.Modulo, rota.Acao) {
			next.ServeHTTP(w, r)
			return
		}
		nome := string(rota.Modulo)
		if info, ok := acesso.InfoDo(rota.Modulo); ok {
			nome = info.Nome
		}
		respondJSONErro(w, http.StatusForbidden, "ACESSO_NEGADO", fmt.Sprintf(
			"seu perfil de acesso (%s) não permite %s em %s. Peça ao administrador para revisar os seus perfis nesta empresa.",
			strings.Join(a.Perfis, ", "), acesso.RotuloAcao(rota.Acao), nome))
	})
}
