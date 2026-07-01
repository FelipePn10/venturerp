package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise"
	"github.com/go-chi/chi/v5"
)

func (h *EnterpriseHandler) CreateEnterprise(w http.ResponseWriter, r *http.Request) {
	var req request.CreateEnterpriseDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	enterprise, err := mapper.ToEnterpriseEntity(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	created, err := h.createEnterpriseUC.Execute(r.Context(), enterprise)
	if err != nil {
		// A duplicate code is a client conflict (409), not a server error.
		if c, ok := errorsuc.AsConflict(err); ok {
			h.Conflict(w, c.Error())
			return
		}
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, created, "enterprise created succesfully")
}

func (h *EnterpriseHandler) GetEnterprise(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(chi.URLParam(r, "code"))
	if err != nil {
		h.BadRequest(w, "invalid code")
		return
	}
	result, err := h.getEnterpriseUC.Execute(r.Context(), code)
	if err != nil {
		h.NotFound(w, "enterprise not found")
		return
	}
	h.OK(w, result)
}

func (h *EnterpriseHandler) ListEnterprises(w http.ResponseWriter, r *http.Request) {
	results, err := h.listEnterprisesUC.Execute(r.Context())
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, results)
}
