package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

func (h *BomHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req request.CreateBomUseCaseRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.BadRequest(w, "invalid request body")
		return
	}

	bom, err := h.createBomUC.Execute(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, errorsuc.ErrCreateBom):
			h.BadRequest(w, "failed create bom")
		case errors.Is(err, errorsuc.ErrCreateBomNotFound):
			h.NotFound(w, "try again later.")
		default:
			h.InternalError(w, r, err)
			return
		}
		return
	}

	h.OK(w, bom, "Create Bom Success!")
}
