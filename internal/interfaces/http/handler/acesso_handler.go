package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/acesso_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	sec "github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// LeitorDeAcesso devolve o acesso do usuário da sessão (o controle de acesso
// do middleware, com o mesmo cache).
type LeitorDeAcesso interface {
	AcessoDe(ctx context.Context, user *security.AuthUser) (acesso.AcessoDoUsuario, error)
}

// AcessoHandler expõe os perfis de acesso da empresa (ADMIN) e as permissões do
// próprio usuário (qualquer perfil — o desktop esconde o que não pode abrir).
type AcessoHandler struct {
	uc       *acesso_uc.UseCase
	controle LeitorDeAcesso
}

func NewAcessoHandler(uc *acesso_uc.UseCase, controle LeitorDeAcesso) *AcessoHandler {
	return &AcessoHandler{uc: uc, controle: controle}
}

func (h *AcessoHandler) Catalogo(w http.ResponseWriter, _ *http.Request) {
	sec.RespondJSON(w, http.StatusOK, acesso_uc.Catalogo())
}

type minhasPermissoesResponse struct {
	Papel      string                     `json:"papel"`
	Restrito   bool                       `json:"restrito"`
	Perfis     []string                   `json:"perfis"`
	Permissoes []acesso_uc.PermissaoDTO   `json:"permissoes"`
	Modulos    []acesso_uc.ModuloResponse `json:"modulos"`
}

func (h *AcessoHandler) MinhasPermissoes(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser)
	if !ok {
		sec.RespondError(w, http.StatusUnauthorized, "sessão não encontrada")
		return
	}
	a, err := h.controle.AcessoDe(r.Context(), user)
	if err != nil {
		sec.RespondUseCaseError(w, err)
		return
	}
	out := minhasPermissoesResponse{Papel: user.Role, Restrito: a.Restrito, Perfis: a.Perfis, Permissoes: []acesso_uc.PermissaoDTO{}, Modulos: acesso_uc.Catalogo()}
	if out.Perfis == nil {
		out.Perfis = []string{}
	}
	for _, p := range a.Permissoes.Lista() {
		out.Permissoes = append(out.Permissoes, acesso_uc.PermissaoDTO{Modulo: string(p.Modulo), Acao: string(p.Acao)})
	}
	sec.RespondJSON(w, http.StatusOK, out)
}

func (h *AcessoHandler) ListarPerfis(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarPerfis(r.Context())
	if err != nil {
		sec.RespondUseCaseError(w, err)
		return
	}
	sec.RespondJSON(w, http.StatusOK, out)
}

func (h *AcessoHandler) CriarPerfil(w http.ResponseWriter, r *http.Request) {
	var dto acesso_uc.PerfilDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		sec.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.CriarPerfil(r.Context(), dto)
	if err != nil {
		sec.RespondUseCaseError(w, err)
		return
	}
	sec.RespondJSON(w, http.StatusCreated, out)
}

func (h *AcessoHandler) AtualizarPerfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		sec.RespondError(w, http.StatusBadRequest, "perfil inválido")
		return
	}
	var dto acesso_uc.PerfilDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		sec.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.AtualizarPerfil(r.Context(), id, dto)
	if err != nil {
		sec.RespondUseCaseError(w, err)
		return
	}
	sec.RespondJSON(w, http.StatusOK, out)
}

func (h *AcessoHandler) ExcluirPerfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		sec.RespondError(w, http.StatusBadRequest, "perfil inválido")
		return
	}
	if err := h.uc.ExcluirPerfil(r.Context(), id); err != nil {
		sec.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AcessoHandler) ListarUsuarios(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarUsuarios(r.Context())
	if err != nil {
		sec.RespondUseCaseError(w, err)
		return
	}
	sec.RespondJSON(w, http.StatusOK, out)
}

func (h *AcessoHandler) DefinirPerfisDoUsuario(w http.ResponseWriter, r *http.Request) {
	var dto acesso_uc.PerfisDoUsuarioDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		sec.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.uc.DefinirPerfisDoUsuario(r.Context(), chi.URLParam(r, "userId"), dto); err != nil {
		sec.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
