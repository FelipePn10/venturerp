package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_commission_uc"
	commissionentity "github.com/FelipePn10/panossoerp/internal/domain/sales_commission/entity"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

// SalesCommissionHandler atende o rateio de comissão do pedido e do orçamento.
// O tipo de documento vem da rota, nunca do corpo: assim uma requisição não
// consegue escrever na tabela do outro documento.
type SalesCommissionHandler struct {
	uc *sales_commission_uc.UseCase
}

func NewSalesCommissionHandler(uc *sales_commission_uc.UseCase) *SalesCommissionHandler {
	return &SalesCommissionHandler{uc: uc}
}

func (h *SalesCommissionHandler) ListOrder(w http.ResponseWriter, r *http.Request) {
	h.listar(w, r, commissionentity.DocumentoPedido)
}

func (h *SalesCommissionHandler) SaveOrder(w http.ResponseWriter, r *http.Request) {
	h.salvar(w, r, commissionentity.DocumentoPedido)
}

func (h *SalesCommissionHandler) ListQuotation(w http.ResponseWriter, r *http.Request) {
	h.listar(w, r, commissionentity.DocumentoOrcamento)
}

func (h *SalesCommissionHandler) SaveQuotation(w http.ResponseWriter, r *http.Request) {
	h.salvar(w, r, commissionentity.DocumentoOrcamento)
}

func (h *SalesCommissionHandler) listar(w http.ResponseWriter, r *http.Request, doc commissionentity.Documento) {
	code, ok := parseCommissionDocumentCode(w, r)
	if !ok {
		return
	}
	out, err := h.uc.Listar(r.Context(), doc, code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func (h *SalesCommissionHandler) salvar(w http.ResponseWriter, r *http.Request, doc commissionentity.Documento) {
	code, ok := parseCommissionDocumentCode(w, r)
	if !ok {
		return
	}
	var dto request.SalvarRateioComissaoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "não foi possível ler o rateio de comissão enviado")
		return
	}
	out, err := h.uc.Substituir(r.Context(), doc, code, &dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

func parseCommissionDocumentCode(w http.ResponseWriter, r *http.Request) (int64, bool) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código do documento inválido")
		return 0, false
	}
	return code, true
}
