package handler

import (
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *OrderPriorityHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/find/{value}", h.FindByValue)
	return r
}

func (h *OrderPriorityHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateOrderPriorityDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *OrderPriorityHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *OrderPriorityHandler) FindByValue(w http.ResponseWriter, r *http.Request) {
	value, err := strconv.ParseFloat(strings.TrimSpace(chi.URLParam(r, "value")), 64)
	if err != nil {
		respondErr(w, errorsuc.NewValidationError(fmt.Sprintf("valor de prioridade inválido no endereço (%q)", chi.URLParam(r, "value"))))
		return
	}
	result, err := h.findUC.Execute(r.Context(), value)
	if err != nil {
		respondErrStatus(w, http.StatusNotFound, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
