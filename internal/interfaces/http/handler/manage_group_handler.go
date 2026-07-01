package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/go-chi/chi/v5"
)

func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(chi.URLParam(r, "code"))
	if err != nil {
		h.BadRequest(w, "invalid code")
		return
	}
	result, err := h.getGroupUC.Execute(r.Context(), code)
	if err != nil {
		h.NotFound(w, "group not found")
		return
	}
	h.OK(w, result)
}

func (h *GroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	results, err := h.listGroupsUC.Execute(r.Context())
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, results)
}

type updateGroupBody struct {
	Description  string `json:"description"`
	EnterpriseID int    `json:"enterprise_id"`
}

func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(chi.URLParam(r, "code"))
	if err != nil {
		h.BadRequest(w, "invalid code")
		return
	}
	var body updateGroupBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.BadRequest(w, "invalid request body")
		return
	}
	result, err := h.updateGroupUC.Execute(r.Context(), code, body.Description, body.EnterpriseID)
	if err != nil {
		if v, ok := errorsuc.AsValidation(err); ok {
			h.UnprocessableEntity(w, v.Error())
			return
		}
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, result, "group updated successfully")
}
