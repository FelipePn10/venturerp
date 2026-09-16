package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// MachineConsumableHandler expõe o cadastro de consumíveis da máquina.
type MachineConsumableHandler struct{ uc *machine_uc.ConsumableUseCase }

func NewMachineConsumableHandler(uc *machine_uc.ConsumableUseCase) *MachineConsumableHandler {
	return &MachineConsumableHandler{uc: uc}
}

func (h *MachineConsumableHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	var dto request.MachineConsumableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondUseCaseError(w, errorsuc.NewValidationError("corpo da requisição inválido"))
		return
	}
	result, err := h.uc.Upsert(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// List aceita machine_code opcional: sem ele, devolve os consumíveis de todas as
// máquinas da empresa, que é o que a tela precisa para montar a seleção.
func (h *MachineConsumableHandler) List(w http.ResponseWriter, r *http.Request) {
	var machineCode int64
	if v := r.URL.Query().Get("machine_code"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			security.RespondUseCaseError(w, errorsuc.NewValidationError("máquina inválida"))
			return
		}
		machineCode = n
	}
	results, err := h.uc.List(r.Context(), machineCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineConsumableHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondUseCaseError(w, errorsuc.NewValidationError("consumível inválido"))
		return
	}
	if err := h.uc.Delete(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "consumível removido"})
}
