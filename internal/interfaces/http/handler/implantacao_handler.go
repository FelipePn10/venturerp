package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/implantacao_uc"
)

// ImplantacaoHandler expõe o painel de implantação.
type ImplantacaoHandler struct{ Painel *implantacao_uc.PainelUseCase }

// GetPainel: GET /api/implantacao/painel — placar "pronto para operar" da empresa.
func (h *ImplantacaoHandler) GetPainel(w http.ResponseWriter, r *http.Request) {
	p, err := h.Painel.Painel(r.Context())
	if err != nil {
		respondErr(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, p)
}
