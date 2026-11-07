package handler

import (
	"errors"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/items/repository"
)

// ImpactoDaAlteracao simula a alteração sem gravar.
func (h *ItemHandler) ImpactoDaAlteracao(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateItemDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.updateItemUC.Impacto(r.Context(), itemCodePathParam(r, "code"), dto)
	if err != nil {
		respondErr(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *ItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	code := itemCodePathParam(r, "code")
	if code == "" {
		h.BadRequest(w, "invalid 'code'")
		return
	}
	var dto request.UpdateItemDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	updated, err := h.updateItemUC.ExecuteBusinessCode(r.Context(), code, dto)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.NotFound(w, "item não encontrado")
			return
		}
		if errors.Is(err, repository.ErrInvalidReference) {
			respondErr(w, err)
			return
		}
		respondErr(w, err)
		return
	}
	h.OK(w, updated, "item updated successfully")
}
