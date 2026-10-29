package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_policy_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// PlanningPolicyHandler é a política de planejamento por item ou
// classificação (VPLA0310).
type PlanningPolicyHandler struct{ uc *planning_policy_uc.UseCase }

func NewPlanningPolicyHandler(uc *planning_policy_uc.UseCase) *PlanningPolicyHandler {
	return &PlanningPolicyHandler{uc: uc}
}

// List: ?escopo=ITEM|CLASSIFICACAO&busca=texto
func (h *PlanningPolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := h.uc.Listar(r.Context(), q.Get("escopo"), q.Get("busca"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *PlanningPolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaPolitica(w, r)
	if !ok {
		return
	}
	out, err := h.uc.Obter(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *PlanningPolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.PoliticaPlanejamentoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.uc.Criar(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, out)
}

func (h *PlanningPolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaPolitica(w, r)
	if !ok {
		return
	}
	var dto request.PoliticaPlanejamentoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.uc.Alterar(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *PlanningPolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaPolitica(w, r)
	if !ok {
		return
	}
	if err := h.uc.Excluir(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Effective: ?item=<código de negócio> — a política que vale para o item.
func (h *PlanningPolicyHandler) Effective(w http.ResponseWriter, r *http.Request) {
	item := strings.TrimSpace(r.URL.Query().Get("item"))
	if item == "" {
		security.RespondError(w, http.StatusBadRequest, "informe o item")
		return
	}
	out, err := h.uc.Efetiva(r.Context(), request.TextCode(item))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func idDaPolitica(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		security.RespondError(w, http.StatusBadRequest, "política inválida")
		return 0, false
	}
	return id, true
}
