package handler

import (
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *ModifierHandler) GetModifier(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.BadRequest(w, "código inválido")
		return
	}
	result, err := h.getModifierUC.Execute(r.Context(), id)
	if err != nil {
		respondErrStatus(w, http.StatusInternalServerError, err)
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
	IsActive    *bool  `json:"is_active"`
}

func (h *ModifierHandler) UpdateModifier(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.BadRequest(w, "código inválido")
		return
	}
	var body updateModifierBody
	if !security.DecodeBody(w, r, &body) {
		return
	}
	result, err := h.updateModifierUC.Execute(r.Context(), id, body.Description, body.IsActive)
	if err != nil {
		respondErrStatus(w, http.StatusInternalServerError, err)
		return
	}
	h.OK(w, result, "modifier updated successfully")
}
