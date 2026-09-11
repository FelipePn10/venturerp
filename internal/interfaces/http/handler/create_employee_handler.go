package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *EmployeeHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateEmployeeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *EmployeeHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *EmployeeHandler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	result, err := h.getUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *EmployeeHandler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateEmployeeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *EmployeeHandler) DeactivateEmployee(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	if err := h.deactivateUC.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "employee deactivated"})
}

// ListEmployeesByRole filtra funcionários por função. É o que o cadastro de
// máquina usa para oferecer só mecânicos como responsável pela manutenção —
// no FoccoERP (FENG0111) a lista já vem filtrada, e oferecer a empresa inteira
// convida ao erro.
func (h *EmployeeHandler) ListEmployeesByRole(w http.ResponseWriter, r *http.Request) {
	role := strings.TrimSpace(chi.URLParam(r, "role"))
	if role == "" {
		security.RespondError(w, http.StatusBadRequest, "informe a função")
		return
	}
	result, err := h.listByRoleUC.Execute(r.Context(), role)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
