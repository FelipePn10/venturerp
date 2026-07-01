package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/go-chi/chi/v5"
)

func (h *ModifierHandler) GetModifier(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.BadRequest(w, "invalid id")
		return
	}
	result, err := h.getModifierUC.Execute(r.Context(), id)
	if err != nil {
		h.NotFound(w, "modifier not found")
		return
	}
	h.OK(w, result)
}

func (h *ModifierHandler) ListModifiers(w http.ResponseWriter, r *http.Request) {
	results, err := h.listModifiersUC.Execute(r.Context())
	if err != nil {
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, results)
}

type updateModifierBody struct {
	Description string `json:"description"`
}

func (h *ModifierHandler) UpdateModifier(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.BadRequest(w, "invalid id")
		return
	}
	var body updateModifierBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.BadRequest(w, "invalid request body")
		return
	}
	result, err := h.updateModifierUC.Execute(r.Context(), id, body.Description)
	if err != nil {
		if v, ok := errorsuc.AsValidation(err); ok {
			h.UnprocessableEntity(w, v.Error())
			return
		}
		h.InternalError(w, r, err)
		return
	}
	h.OK(w, result, "modifier updated successfully")
}
