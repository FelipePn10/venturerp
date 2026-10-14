package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// ScenarioHandler expõe os cenários de simulação do planejamento.
type ScenarioHandler struct{ uc *planning_uc.ScenarioUseCase }

func NewScenarioHandler(uc *planning_uc.ScenarioUseCase) *ScenarioHandler {
	return &ScenarioHandler{uc: uc}
}

func scenarioID(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil || id <= 0 {
		security.RespondErrorCode(w, http.StatusBadRequest, "CENARIO_INVALIDO", "código inválido")
		return 0, false
	}
	return id, true
}

func decodeScenario(w http.ResponseWriter, r *http.Request, dst any) bool {
	if !security.DecodeBody(w, r, dst) {
		return false
	}
	return true
}

func respondScenario(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, status, out)
}

func (h *ScenarioHandler) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.List(r.Context())
	respondScenario(w, http.StatusOK, out, err)
}

func (h *ScenarioHandler) Get(w http.ResponseWriter, r *http.Request) {
	if id, ok := scenarioID(w, r, "id"); ok {
		out, err := h.uc.Get(r.Context(), id)
		respondScenario(w, http.StatusOK, out, err)
	}
}

func (h *ScenarioHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateScenarioDTO
	if !decodeScenario(w, r, &dto) {
		return
	}
	out, err := h.uc.Create(r.Context(), dto)
	respondScenario(w, http.StatusCreated, out, err)
}

func (h *ScenarioHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := scenarioID(w, r, "id")
	if !ok {
		return
	}
	var dto request.UpdateScenarioDTO
	if !decodeScenario(w, r, &dto) {
		return
	}
	out, err := h.uc.Update(r.Context(), id, dto)
	respondScenario(w, http.StatusOK, out, err)
}

func (h *ScenarioHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if id, ok := scenarioID(w, r, "id"); ok {
		if err := h.uc.Delete(r.Context(), id); err != nil {
			security.RespondUseCaseError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *ScenarioHandler) AddChange(w http.ResponseWriter, r *http.Request) {
	id, ok := scenarioID(w, r, "id")
	if !ok {
		return
	}
	var dto request.ScenarioChangeDTO
	if !decodeScenario(w, r, &dto) {
		return
	}
	out, err := h.uc.AddChange(r.Context(), id, dto)
	respondScenario(w, http.StatusCreated, out, err)
}

func (h *ScenarioHandler) DeleteChange(w http.ResponseWriter, r *http.Request) {
	id, ok := scenarioID(w, r, "id")
	if !ok {
		return
	}
	changeID, ok := scenarioID(w, r, "changeID")
	if !ok {
		return
	}
	if err := h.uc.DeleteChange(r.Context(), id, changeID); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ScenarioHandler) BaseDemands(w http.ResponseWriter, r *http.Request) {
	if id, ok := scenarioID(w, r, "id"); ok {
		out, err := h.uc.BaseDemands(r.Context(), id)
		respondScenario(w, http.StatusOK, out, err)
	}
}

func (h *ScenarioHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	if id, ok := scenarioID(w, r, "id"); ok {
		out, err := h.uc.Calculate(r.Context(), id)
		respondScenario(w, http.StatusOK, out, err)
	}
}

func (h *ScenarioHandler) Compare(w http.ResponseWriter, r *http.Request) {
	if id, ok := scenarioID(w, r, "id"); ok {
		out, err := h.uc.Compare(r.Context(), id)
		respondScenario(w, http.StatusOK, out, err)
	}
}

func (h *ScenarioHandler) Promote(w http.ResponseWriter, r *http.Request) {
	if id, ok := scenarioID(w, r, "id"); ok {
		out, err := h.uc.Promote(r.Context(), id)
		respondScenario(w, http.StatusOK, out, err)
	}
}
