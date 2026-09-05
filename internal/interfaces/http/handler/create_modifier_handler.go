package handler

import (
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/modifier"
)

func (h *ModifierHandler) CreateModifier(w http.ResponseWriter, r *http.Request) {
	var req request.CreateModifierDTO

	if !security.DecodeBody(w, r, &req) {
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
