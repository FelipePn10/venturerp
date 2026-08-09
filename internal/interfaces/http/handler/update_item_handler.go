package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/go-chi/chi/v5"
)

func (h *ItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		h.BadRequest(w, "invalid 'code'")
		return
	}
	var dto request.UpdateItemDTO
	if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.BadRequest(w, "invalid request body")
		return
	}
	updated, err := h.updateItemUC.Execute(r.Context(), code, dto)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.NotFound(w, "item not found")
			return
		}
		if errors.Is(err, repository.ErrInvalidReference) {
			jsonError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	h.OK(w, updated, "item updated successfully")
}
