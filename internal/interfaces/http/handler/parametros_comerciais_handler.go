package handler

import (
	"net/http"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/parametros_comerciais_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/parametroscomerciais"
)

// ParametrosComerciaisHandler expõe a alçada de desconto do pedido.
type ParametrosComerciaisHandler struct {
	uc *parametros_comerciais_uc.UseCase
}

func NewParametrosComerciaisHandler(uc *parametros_comerciais_uc.UseCase) *ParametrosComerciaisHandler {
	return &ParametrosComerciaisHandler{uc: uc}
}

func (h *ParametrosComerciaisHandler) Obter(w http.ResponseWriter, r *http.Request) {
	out, err := h.uc.Obter(r.Context())
	responder(w, http.StatusOK, out, err)
}

func (h *ParametrosComerciaisHandler) Salvar(w http.ResponseWriter, r *http.Request) {
	var dto parametroscomerciais.Parametros
	if !decodificar(w, r, &dto) {
		return
	}
	out, err := h.uc.Salvar(r.Context(), dto)
	responder(w, http.StatusOK, out, err)
}
