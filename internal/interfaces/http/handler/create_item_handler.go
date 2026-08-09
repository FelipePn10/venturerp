package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/item"
)

func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var req request.CreateItemDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	item, err := mapper.ToItemEntity(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	created, err := h.createItemUC.Execute(r.Context(), item)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidReference) {
			jsonError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, created, "item created successfully")
}
