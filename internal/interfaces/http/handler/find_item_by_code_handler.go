package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/go-chi/chi/v5"
)

func (h *ItemHandler) FindItemByCodeHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	codeStr := chi.URLParam(r, "code")
	if codeStr == "" {
		h.BadRequest(w, "path param 'code' is required")
		return
	}

	codeInt, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		h.BadRequest(w, "invalid 'code'")
		return
	}

	req := request.FindItemByCodeDTO{
		Code: valueobject.ItemCode(codeInt),
	}

	item, err := h.findItemByCodeUC.Execute(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, errorsuc.ErrUnauthorized):
			h.BadRequest(w, "unauthorized")
			return
		case errors.Is(err, errorsuc.ErrProductNotFound):
			h.NotFound(w, "product not found")
			return
		default:
			h.InternalError(w, err)
			return
		}
	}

	h.OK(w, item, "Product Found")
}
