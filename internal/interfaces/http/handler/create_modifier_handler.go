package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/modifier"
)

func (h *ModifierHandler) CreateModifier(w http.ResponseWriter, r *http.Request) {
	var req request.CreateModifierDTO

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	modifier, err := mapper.ToModifierEntity(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	created, err := h.createModifierUC.Execute(r.Context(), modifier)
	if err != nil {
		h.InternalError(w, r, err)
		return
	}

	h.Created(w, created, "modifier created succesfully")
}
