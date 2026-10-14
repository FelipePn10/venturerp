package handler

import (
	"errors"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/item"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var req request.CreateItemDTO
	if !security.DecodeBody(w, r, &req) {
		return
	}

	item, err := mapper.ToItemEntity(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	created, err := h.createItemUC.Execute(r.Context(), item)
	if err != nil {
		if errors.Is(err, item_uc.ErrItemBaseNotFound) {
			respondErr(w, err)
			return
		}
		if errors.Is(err, repository.ErrInvalidReference) {
			respondErr(w, err)
			return
		}
		if errors.Is(err, repository.ErrConflict) {
			respondErrStatus(w, http.StatusConflict, err)
			return
		}
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, created, "item created successfully")
}
