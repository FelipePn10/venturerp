package handler

import (
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *PlanningParamsHandler) List(w http.ResponseWriter, r *http.Request) {
	if !queryValida(w, r, filtrosDaQuery{Inteiros: []string{"limit", "offset"}}) {
		return
	}
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
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
		respondErrStatus(w, http.StatusNotFound, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PlanningParamsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdatePlanningParamDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
