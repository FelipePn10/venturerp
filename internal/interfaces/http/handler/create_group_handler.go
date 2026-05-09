package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/group"
)

func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req request.CreateGroupDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	group, err := mapper.ToGroupEntity(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	created, err := h.createGroupUC.Execute(r.Context(), group)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, created, "group created succesfully")
}
