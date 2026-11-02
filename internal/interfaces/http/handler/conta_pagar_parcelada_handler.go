package handler

import (
	"encoding/json"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type ContaPagarParceladaHandler struct {
	uc *financial_uc.CreateContaPagarParceladaUseCase
}

func NewContaPagarParceladaHandler(uc *financial_uc.CreateContaPagarParceladaUseCase) *ContaPagarParceladaHandler {
	return &ContaPagarParceladaHandler{uc: uc}
}

func (h *ContaPagarParceladaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateContaPagarParceladaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	out, err := h.uc.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, map[string]int{"parcelas": len(out)})
}
