package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type ContasPagarPorPlanoHandler struct {
	uc *financial_uc.ContasPagarPorPlanoUseCase
}

func NewContasPagarPorPlanoHandler(uc *financial_uc.ContasPagarPorPlanoUseCase) *ContasPagarPorPlanoHandler {
	return &ContasPagarPorPlanoHandler{uc: uc}
}

func (h *ContasPagarPorPlanoHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := h.uc.Execute(r.Context(), financial_uc.ContasPagarPorPlanoDTO{
		StartDate:    textoDaQuery(q, "start_date"),
		EndDate:      textoDaQuery(q, "end_date"),
		DateField:    textoDaQuery(q, "date_field"),
		Status:       textoDaQuery(q, "status"),
		FornecedorID: inteiroDaQuery(q, "fornecedor_id"),
	})
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
