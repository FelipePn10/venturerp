package handler

import (
	"errors"
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/cnpj_uc"
	cnpjsvc "github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// CNPJHandler exposes the auto-fill lookup used by the cadastro screens: given a
// CNPJ it returns razão social, inscrição estadual, endereço and more so the
// operator doesn't retype data already held by the Receita Federal.
type CNPJHandler struct {
	*security.BaseHandler
	lookupUC *cnpj_uc.LookupCNPJUseCase
}

func NewCNPJHandler(lookupUC *cnpj_uc.LookupCNPJUseCase) *CNPJHandler {
	return &CNPJHandler{BaseHandler: &security.BaseHandler{}, lookupUC: lookupUC}
}

// Lookup handles GET /api/cnpj/{cnpj}.
func (h *CNPJHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	cnpj := chi.URLParam(r, "cnpj")
	if cnpj == "" {
		h.BadRequest(w, "cnpj é obrigatório")
		return
	}

	result, err := h.lookupUC.Execute(r.Context(), cnpj)
	switch {
	case err == nil:
		h.OK(w, result)
	case errors.Is(err, cnpj_uc.ErrInvalidCNPJ):
		h.BadRequest(w, err.Error())
	case errors.Is(err, cnpjsvc.ErrNotFound):
		h.NotFound(w, "CNPJ não encontrado na base da Receita")
	case errors.Is(err, cnpjsvc.ErrUnavailable):
		// 502: our dependency failed, not the client's fault.
		security.WriteError(w, http.StatusBadGateway, "cnpj_provider_unavailable",
			"serviço de consulta de CNPJ indisponível, tente novamente em instantes")
	default:
		h.InternalError(w, r, err)
	}
}
