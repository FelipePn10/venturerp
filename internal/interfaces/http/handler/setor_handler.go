package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/setor_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/setor"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// SetorHandler: setores, pessoas e responsáveis por grupo do PDM.
type SetorHandler struct{ uc *setor_uc.UseCase }

func NewSetorHandler(uc *setor_uc.UseCase) *SetorHandler { return &SetorHandler{uc: uc} }

func (h *SetorHandler) Setores(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Setores(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *SetorHandler) SalvarSetor(w http.ResponseWriter, r *http.Request) {
	var s dom.Setor
	if !decodificar(w, r, &s) {
		return
	}
	responder(w, http.StatusOK, s, h.uc.SalvarSetor(r.Context(), s))
}

func (h *SetorHandler) DefinirMembros(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Usuarios []uuid.UUID `json:"user_ids"`
	}
	if !decodificar(w, r, &corpo) {
		return
	}
	responder(w, http.StatusOK, map[string]any{"ok": true}, h.uc.DefinirMembros(r.Context(), chi.URLParam(r, "codigo"), corpo.Usuarios))
}

func grupoDaURL(w http.ResponseWriter, r *http.Request) (int, bool) {
	g, err := strconv.Atoi(chi.URLParam(r, "grupo"))
	if err != nil || g <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código do grupo inválido")
		return 0, false
	}
	return g, true
}

func (h *SetorHandler) ResponsaveisDoGrupo(w http.ResponseWriter, r *http.Request) {
	g, ok := grupoDaURL(w, r)
	if !ok {
		return
	}
	out, err := h.uc.ResponsaveisDoGrupo(r.Context(), g)
	responder(w, http.StatusOK, out, err)
}

func (h *SetorHandler) SalvarResponsaveis(w http.ResponseWriter, r *http.Request) {
	g, ok := grupoDaURL(w, r)
	if !ok {
		return
	}
	var resp dom.Responsaveis
	if !decodificar(w, r, &resp) {
		return
	}
	resp.Grupo = g
	if err := h.uc.SalvarResponsaveis(r.Context(), resp); err != nil {
		responder(w, http.StatusOK, nil, err)
		return
	}
	out, err := h.uc.ResponsaveisDoGrupo(r.Context(), g)
	responder(w, http.StatusOK, out, err)
}

func (h *SetorHandler) ResponsaveisDoItem(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ResponsaveisDoItem(r.Context(), itemCodePathParam(r, "item"))
	responder(w, http.StatusOK, out, err)
}
