package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// ReformaTributariaHandler expõe a parametrização do IBS/CBS: tabela oficial de
// cClassTrib, alíquotas por ano, cClassTrib por CFOP e o da linha da nota.
type ReformaTributariaHandler struct {
	uc *fiscal_uc.ReformaTributariaUseCase
}

func NewReformaTributariaHandler(uc *fiscal_uc.ReformaTributariaUseCase) *ReformaTributariaHandler {
	return &ReformaTributariaHandler{uc: uc}
}

func (h *ReformaTributariaHandler) ListarClassificacoes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := h.uc.ListarClassificacoes(r.Context(), q.Get("cst"), q.Get("busca"), q.Get("somente_nfe") != "false")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ReformaTributariaHandler) ListarAliquotas(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarAliquotas(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func anoDaRota(r *http.Request) (int, bool) {
	ano, err := strconv.Atoi(chi.URLParam(r, "ano"))
	return ano, err == nil
}

func (h *ReformaTributariaHandler) SalvarAliquotas(w http.ResponseWriter, r *http.Request) {
	ano, ok := anoDaRota(r)
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "ano inválido")
		return
	}
	var dto fiscal_uc.SalvarAliquotasIBSCBSDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.uc.SalvarAliquotas(r.Context(), ano, dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReformaTributariaHandler) ExcluirAliquotas(w http.ResponseWriter, r *http.Request) {
	ano, ok := anoDaRota(r)
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "ano inválido")
		return
	}
	if err := h.uc.ExcluirAliquotas(r.Context(), ano); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReformaTributariaHandler) ListarCFOP(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.ListarCFOP(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ReformaTributariaHandler) SalvarCFOP(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.SalvarCFOPClassTribDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.uc.SalvarCFOP(r.Context(), chi.URLParam(r, "cfop"), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReformaTributariaHandler) ExcluirCFOP(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.ExcluirCFOP(r.Context(), chi.URLParam(r, "cfop")); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReformaTributariaHandler) DefinirClassTribDaLinha(w http.ResponseWriter, r *http.Request) {
	exitID, ok := idDaRota(r, "id")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "nota inválida")
		return
	}
	itemID, ok := idDaRota(r, "itemId")
	if !ok {
		security.RespondError(w, http.StatusBadRequest, "linha inválida")
		return
	}
	var dto fiscal_uc.DefinirClassTribDaLinhaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.uc.DefinirClassTribDaLinha(r.Context(), exitID, itemID, dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReformaTributariaHandler) Modo(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Modo(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ReformaTributariaHandler) SalvarModo(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.SalvarModoReformaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	if err := h.uc.SalvarModo(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
