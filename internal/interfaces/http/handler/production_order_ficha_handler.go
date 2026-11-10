package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/manufacturing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

func (h *ProductionOrderHandler) WithFicha(uc *production_order_uc.FichaDaOrdemUseCase) *ProductionOrderHandler {
	h.fichaUC = uc
	return h
}

// PDF devolve a ordem de fabricação impressa (etapas, materiais por etapa,
// ferramentas e serviços em terceiro).
func (h *ProductionOrderHandler) PDF(w http.ResponseWriter, r *http.Request) {
	if h.fichaUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "a impressão da ordem não está configurada")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}
	impressa, err := h.fichaUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	e := impressa.Empresa
	b := manufacturing.Branding{
		Company:       pdfkit.Company{Name: e.Nome, CNPJ: e.CNPJ, IE: e.IE, Address: e.Endereco, Phone: e.Telefone, Email: e.Email},
		Logo:          e.Logo,
		BrandColorHex: e.Cor,
	}
	pdf := manufacturing.RenderFichaDaOrdemPor(impressa.Ficha, b, time.Now(), impressa.EmitidaPor)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="OF-%d.pdf"`, impressa.Ficha.ID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func (h *ProductionOrderHandler) WithCadastrosDoPosto(uc *production_order_uc.CadastrosDoPostoUseCase) *ProductionOrderHandler {
	h.cadastrosDoPostoUC = uc
	return h
}

// CadastrosDoPosto: listas mínimas para o apontamento (máquina, operador, depósito).
func (h *ProductionOrderHandler) CadastrosDoPosto(w http.ResponseWriter, r *http.Request) {
	if h.cadastrosDoPostoUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "as listas do posto não estão configuradas")
		return
	}
	out, err := h.cadastrosDoPostoUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}

// ReapurarCustos refaz a apuração de todas as OFs já apuradas pela regra atual.
func (h *ProductionOrderHandler) ReapurarCustos(w http.ResponseWriter, r *http.Request) {
	if h.settleCostUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "a apuração de custo não está configurada")
		return
	}
	out, err := h.settleCostUC.ReapurarTodas(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, out)
}
