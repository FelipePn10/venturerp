package handler

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// CostCenterManutencaoHandler: listar com inativos, ver uso, alterar e excluir o
// centro de custo industrial. As regras ficam nos casos de uso.
type CostCenterManutencaoHandler struct {
	List   *cost_center_uc.ListCostCentersCadastroUseCase
	Uso    *cost_center_uc.UsoCostCenterUseCase
	Update *cost_center_uc.UpdateCostCenterUseCase
	Delete *cost_center_uc.DeleteCostCenterUseCase
}

func codigoDoCentro(w http.ResponseWriter, r *http.Request) (int32, bool) {
	v, ok := codigoDaRota(w, r, "costCenterCode")
	if !ok {
		return 0, false
	}
	if v <= 0 || v > math.MaxInt32 {
		respondErr(w, errorsuc.NewValidationError("código de centro de custo inválido"))
		return 0, false
	}
	return int32(v), true
}

func (h *CostCenterManutencaoHandler) ListCadastro(w http.ResponseWriter, r *http.Request) {
	out, err := h.List.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterManutencaoHandler) UsoCentro(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDoCentro(w, r)
	if !ok {
		return
	}
	out, err := h.Uso.Execute(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterManutencaoHandler) UpdateCentro(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDoCentro(w, r)
	if !ok {
		return
	}
	var dto request.UpdateCostCenterDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.Update.Execute(r.Context(), code, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *CostCenterManutencaoHandler) DeleteCentro(w http.ResponseWriter, r *http.Request) {
	code, ok := codigoDoCentro(w, r)
	if !ok {
		return
	}
	if err := h.Delete.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
