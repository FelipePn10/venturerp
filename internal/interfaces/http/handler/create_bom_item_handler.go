package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

func (h *BomItemHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req request.CreateBomItemsRequestDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.BadRequest(w, "invalid request body")
		return
	}

	if req.BomID < 0 || req.ComponentID < 0 {
		h.BadRequest(w, "missing required fields")
		return
	}

	bomItem, err := h.createBomItemUC.Execute(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, errorsuc.ErrCreateBomItem):
			h.UnprocessableEntity(w, "invalid bom item data")
		case errors.Is(err, errorsuc.ErrCreateBomItemNotFound):
			h.NotFound(w, "related resource not found")
		default:
			h.InternalError(w, r, err)
		}
		return
	}

	h.Created(w, bomItem, "bom item created successfully")
}
