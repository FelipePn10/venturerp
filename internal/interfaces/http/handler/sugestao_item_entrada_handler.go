package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// SugestaoItemEntradaHandler apoia o lançamento manual da nota de entrada.
type SugestaoItemEntradaHandler struct {
	uc *fiscal_uc.SugestaoItemEntradaUseCase
}

func NewSugestaoItemEntradaHandler(uc *fiscal_uc.SugestaoItemEntradaUseCase) *SugestaoItemEntradaHandler {
	return &SugestaoItemEntradaHandler{uc: uc}
}

func (h *SugestaoItemEntradaHandler) Sugerir(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	consulta := fiscal_uc.ConsultaSugestaoItem{
		CodigoFornecedor: strings.TrimSpace(q.Get("codigo_fornecedor")),
		UFEmitente:       strings.ToUpper(strings.TrimSpace(q.Get("uf_emitente"))),
	}
	for campo, destino := range map[string]**int64{"supplier_code": &consulta.SupplierCode, "item_code": &consulta.ItemCode} {
		v := strings.TrimSpace(q.Get(campo))
		if v == "" {
			continue
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "código inválido")
			return
		}
		*destino = &n
	}
	out, err := h.uc.Execute(r.Context(), consulta)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
