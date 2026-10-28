package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/aprovacao_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// AprovacaoHandler expõe o motor de aprovação: solicitações (para os
// aprovadores) e políticas de alçada (ADMIN).
type AprovacaoHandler struct{ uc *aprovacao_uc.UseCase }

func NewAprovacaoHandler(uc *aprovacao_uc.UseCase) *AprovacaoHandler {
	return &AprovacaoHandler{uc: uc}
}

func (h *AprovacaoHandler) ListarSolicitacoes(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarSolicitacoes(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AprovacaoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "solicitação inválida")
		return
	}
	out, err := h.uc.Obter(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AprovacaoHandler) decidir(w http.ResponseWriter, r *http.Request, aprovar bool) {
	id, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "solicitação inválida")
		return
	}
	var dto aprovacao_uc.DecisaoDTO
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
			return
		}
	}
	var (
		out *aprovacao_uc.SolicitacaoResponse
		err error
	)
	if aprovar {
		out, err = h.uc.Aprovar(r.Context(), id, dto)
	} else {
		out, err = h.uc.Reprovar(r.Context(), id, dto)
	}
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AprovacaoHandler) Aprovar(w http.ResponseWriter, r *http.Request)  { h.decidir(w, r, true) }
func (h *AprovacaoHandler) Reprovar(w http.ResponseWriter, r *http.Request) { h.decidir(w, r, false) }

func (h *AprovacaoHandler) ListarPoliticas(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarPoliticas(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *AprovacaoHandler) SalvarPolitica(w http.ResponseWriter, r *http.Request) {
	var dto aprovacao_uc.PoliticaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.uc.SalvarPolitica(r.Context(), chi.URLParam(r, "tipo"), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
