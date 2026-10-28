package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// CostCenterGestaoHandler: gestão, orçamento, árvore e aprovações do centro de
// custo único. As regras ficam no caso de uso.
type CostCenterGestaoHandler struct {
	UC *cost_center_uc.GestaoUseCase
}

func anoDaConsulta(r *http.Request) int {
	if a, err := strconv.Atoi(r.URL.Query().Get("ano")); err == nil {
		return a
	}
	return time.Now().Year()
}

func (h *CostCenterGestaoHandler) Centros(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Centros(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterGestaoHandler) SalvarGestao(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDoCentro(w, r)
	if !ok {
		return
	}
	var dto cost_center_uc.GestaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.UC.SalvarGestao(r.Context(), code, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterGestaoHandler) Orcamento(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Orcamento(r.Context(), anoDaConsulta(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterGestaoHandler) SalvarOrcamento(w http.ResponseWriter, r *http.Request) {
	var dto cost_center_uc.OrcamentoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	if err := h.UC.SalvarOrcamento(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	out, err := h.UC.Orcamento(r.Context(), dto.Ano)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterGestaoHandler) Arvore(w http.ResponseWriter, r *http.Request) {
	mes, _ := strconv.Atoi(r.URL.Query().Get("mes"))
	out, err := h.UC.Arvore(r.Context(), anoDaConsulta(r), mes)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterGestaoHandler) Aprovacoes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var req *int64
	if v, err := strconv.ParseInt(q.Get("requisicao"), 10, 64); err == nil && v > 0 {
		req = &v
	}
	out, err := h.UC.Aprovacoes(r.Context(), q.Get("status"), q.Get("minhas") == "true", req)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterGestaoHandler) decidir(w http.ResponseWriter, r *http.Request, aprovar bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		security.RespondUseCaseError(w, errorsuc.NewValidationError("aprovação inválida"))
		return
	}
	var dto struct {
		Justificativa string `json:"justificativa"`
	}
	if r.ContentLength != 0 && !security.DecodeBody(w, r, &dto) {
		return
	}
	if err := h.UC.Decidir(r.Context(), id, aprovar, dto.Justificativa); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CostCenterGestaoHandler) Aprovar(w http.ResponseWriter, r *http.Request) {
	h.decidir(w, r, true)
}
func (h *CostCenterGestaoHandler) Reprovar(w http.ResponseWriter, r *http.Request) {
	h.decidir(w, r, false)
}

func (h *CostCenterGestaoHandler) Usuarios(w http.ResponseWriter, r *http.Request) {
	out, err := h.UC.Usuarios(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
