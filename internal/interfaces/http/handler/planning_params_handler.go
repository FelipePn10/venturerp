package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *PlanningParamsHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	search := r.URL.Query().Get("number")
	if search != "" {
		number, parseErr := strconv.Atoi(search)
		if parseErr != nil || number <= 0 {
			security.RespondErrorCode(w, http.StatusUnprocessableEntity, "PARAMETRO_NUMERO_INVALIDO", "o número do parâmetro deve ser positivo")
			return
		}
		filtered := results[:0]
		for _, result := range results {
			if result.ParamNumber == number {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}
	security.RespondJSON(w, http.StatusOK, paginate(w, r, results))
}

func (h *PlanningParamsHandler) GetByNumber(w http.ResponseWriter, r *http.Request) {
	num, err := strconv.Atoi(chi.URLParam(r, "number"))
	if err != nil {
		security.RespondErrorCode(w, http.StatusBadRequest, "PARAMETRO_NUMERO_INVALIDO", "número do parâmetro inválido")
		return
	}
	result, err := h.getUC.Execute(r.Context(), num)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PlanningParamsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdatePlanningParamDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
