package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req request.CreateGroupDTO

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "corpo inválido; enterprise_id e created_by não são aceitos", http.StatusBadRequest)
		return
	}

	created, err := h.createGroupUC.Execute(r.Context(), req)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, created, "group created succesfully")
}
