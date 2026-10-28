package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// SimularTributacaoHandler expõe o simulador de tributação: o mesmo cálculo da
// nota, sem gravar nada.
type SimularTributacaoHandler struct {
	uc *fiscal_uc.SimularTributacaoUseCase
}

func NewSimularTributacaoHandler(uc *fiscal_uc.SimularTributacaoUseCase) *SimularTributacaoHandler {
	return &SimularTributacaoHandler{uc: uc}
}

func (h *SimularTributacaoHandler) Simular(w http.ResponseWriter, r *http.Request) {
	var dto fiscal_uc.SimularTributacaoDTO
	if !security.DecodeBody(w, r, &dto) {
		return
	}
	out, err := h.uc.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
