package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type FinancialHandler struct {
	createContaBancariaUC     *financial_uc.CreateContaBancariaUseCase
	listContasBancariasUC     *financial_uc.ListContasBancariasUseCase
	createCondicaoPagamentoUC *financial_uc.CreateCondicaoPagamentoUseCase
	listCondicoesPagamentoUC  *financial_uc.ListCondicoesPagamentoUseCase
	createPlanoContasUC       *financial_uc.CreatePlanoContasUseCase
	listPlanoContasUC         *financial_uc.ListPlanoContasUseCase
	createCentroCustoUC       *financial_uc.CreateCentroCustoUseCase
	listCentrosCustoUC        *financial_uc.ListCentrosCustoUseCase
	createContaPagarUC        *financial_uc.CreateContaPagarUseCase
	listContasPagarUC         *financial_uc.ListContasPagarUseCase
	getContaPagarUC           *financial_uc.GetContaPagarUseCase
	approveContaPagarUC       *financial_uc.ApproveContaPagarUseCase
	baixarContaPagarUC        *financial_uc.BaixarContaPagarUseCase
	cancelContaPagarUC        *financial_uc.CancelContaPagarUseCase
	getAgingPagarUC           *financial_uc.GetAgingPagarUseCase
	createContaReceberUC      *financial_uc.CreateContaReceberUseCase
	listContasReceberUC       *financial_uc.ListContasReceberUseCase
	getContaReceberUC         *financial_uc.GetContaReceberUseCase
	baixarContaReceberUC      *financial_uc.BaixarContaReceberUseCase
	cancelContaReceberUC      *financial_uc.CancelContaReceberUseCase
	getAgingReceberUC         *financial_uc.GetAgingReceberUseCase
	getFluxoCaixaUC           *financial_uc.GetFluxoCaixaUseCase
	getFluxoProjetadoUC       *financial_uc.GetFluxoProjetadoUseCase
	getSaldoContasUC          *financial_uc.GetSaldoContasUseCase
	apurarImpostosUC          *financial_uc.ApurarImpostosUseCase
	getTaxAssessmentUC        *financial_uc.GetTaxAssessmentUseCase
}

func NewFinancialHandler(
	createContaBancariaUC *financial_uc.CreateContaBancariaUseCase,
	listContasBancariasUC *financial_uc.ListContasBancariasUseCase,
	createCondicaoPagamentoUC *financial_uc.CreateCondicaoPagamentoUseCase,
	listCondicoesPagamentoUC *financial_uc.ListCondicoesPagamentoUseCase,
	createPlanoContasUC *financial_uc.CreatePlanoContasUseCase,
	listPlanoContasUC *financial_uc.ListPlanoContasUseCase,
	createCentroCustoUC *financial_uc.CreateCentroCustoUseCase,
	listCentrosCustoUC *financial_uc.ListCentrosCustoUseCase,
	createContaPagarUC *financial_uc.CreateContaPagarUseCase,
	listContasPagarUC *financial_uc.ListContasPagarUseCase,
	getContaPagarUC *financial_uc.GetContaPagarUseCase,
	approveContaPagarUC *financial_uc.ApproveContaPagarUseCase,
	baixarContaPagarUC *financial_uc.BaixarContaPagarUseCase,
	cancelContaPagarUC *financial_uc.CancelContaPagarUseCase,
	getAgingPagarUC *financial_uc.GetAgingPagarUseCase,
	createContaReceberUC *financial_uc.CreateContaReceberUseCase,
	listContasReceberUC *financial_uc.ListContasReceberUseCase,
	getContaReceberUC *financial_uc.GetContaReceberUseCase,
	baixarContaReceberUC *financial_uc.BaixarContaReceberUseCase,
	cancelContaReceberUC *financial_uc.CancelContaReceberUseCase,
	getAgingReceberUC *financial_uc.GetAgingReceberUseCase,
	getFluxoCaixaUC *financial_uc.GetFluxoCaixaUseCase,
	getFluxoProjetadoUC *financial_uc.GetFluxoProjetadoUseCase,
	getSaldoContasUC *financial_uc.GetSaldoContasUseCase,
	apurarImpostosUC *financial_uc.ApurarImpostosUseCase,
	getTaxAssessmentUC *financial_uc.GetTaxAssessmentUseCase,
) *FinancialHandler {
	return &FinancialHandler{
		createContaBancariaUC:     createContaBancariaUC,
		listContasBancariasUC:     listContasBancariasUC,
		createCondicaoPagamentoUC: createCondicaoPagamentoUC,
		listCondicoesPagamentoUC:  listCondicoesPagamentoUC,
		createPlanoContasUC:       createPlanoContasUC,
		listPlanoContasUC:         listPlanoContasUC,
		createCentroCustoUC:       createCentroCustoUC,
		listCentrosCustoUC:        listCentrosCustoUC,
		createContaPagarUC:        createContaPagarUC,
		listContasPagarUC:         listContasPagarUC,
		getContaPagarUC:           getContaPagarUC,
		approveContaPagarUC:       approveContaPagarUC,
		baixarContaPagarUC:        baixarContaPagarUC,
		cancelContaPagarUC:        cancelContaPagarUC,
		getAgingPagarUC:           getAgingPagarUC,
		createContaReceberUC:      createContaReceberUC,
		listContasReceberUC:       listContasReceberUC,
		getContaReceberUC:         getContaReceberUC,
		baixarContaReceberUC:      baixarContaReceberUC,
		cancelContaReceberUC:      cancelContaReceberUC,
		getAgingReceberUC:         getAgingReceberUC,
		getFluxoCaixaUC:           getFluxoCaixaUC,
		getFluxoProjetadoUC:       getFluxoProjetadoUC,
		getSaldoContasUC:          getSaldoContasUC,
		apurarImpostosUC:          apurarImpostosUC,
		getTaxAssessmentUC:        getTaxAssessmentUC,
	}
}

// Contas Bancarias

func (h *FinancialHandler) CreateContaBancaria(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateContaBancariaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createContaBancariaUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FinancialHandler) ListContasBancarias(w http.ResponseWriter, r *http.Request) {
	results, err := h.listContasBancariasUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// Condicoes Pagamento

func (h *FinancialHandler) CreateCondicaoPagamento(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCondicaoPagamentoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createCondicaoPagamentoUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FinancialHandler) ListCondicoesPagamento(w http.ResponseWriter, r *http.Request) {
	results, err := h.listCondicoesPagamentoUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// Plano de Contas

func (h *FinancialHandler) CreatePlanoContas(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePlanoContasDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createPlanoContasUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FinancialHandler) ListPlanoContas(w http.ResponseWriter, r *http.Request) {
	results, err := h.listPlanoContasUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// Centros de Custo

func (h *FinancialHandler) CreateCentroCusto(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCentroCustoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createCentroCustoUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FinancialHandler) ListCentrosCusto(w http.ResponseWriter, r *http.Request) {
	results, err := h.listCentrosCustoUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// Contas a Pagar

func (h *FinancialHandler) CreateContaPagar(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateContaPagarDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createContaPagarUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FinancialHandler) ListContasPagar(w http.ResponseWriter, r *http.Request) {
	var dto request.ListContasPagarFilter
	_ = json.NewDecoder(r.Body).Decode(&dto)
	results, err := h.listContasPagarUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *FinancialHandler) GetContaPagar(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.getContaPagarUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FinancialHandler) ApproveContaPagar(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.approveContaPagarUC.Execute(r.Context(), id); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FinancialHandler) BaixarContaPagar(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.BaixarContaPagarDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.baixarContaPagarUC.Execute(r.Context(), id, dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FinancialHandler) CancelContaPagar(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.cancelContaPagarUC.Execute(r.Context(), id); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FinancialHandler) GetAgingPagar(w http.ResponseWriter, r *http.Request) {
	results, err := h.getAgingPagarUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// Contas a Receber

func (h *FinancialHandler) CreateContaReceber(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateContaReceberDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createContaReceberUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FinancialHandler) ListContasReceber(w http.ResponseWriter, r *http.Request) {
	var dto request.ListContasReceberFilter
	_ = json.NewDecoder(r.Body).Decode(&dto)
	results, err := h.listContasReceberUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *FinancialHandler) GetContaReceber(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.getContaReceberUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FinancialHandler) BaixarContaReceber(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.BaixarContaReceberDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.baixarContaReceberUC.Execute(r.Context(), id, dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FinancialHandler) CancelContaReceber(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.cancelContaReceberUC.Execute(r.Context(), id); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FinancialHandler) GetAgingReceber(w http.ResponseWriter, r *http.Request) {
	results, err := h.getAgingReceberUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// Fluxo de Caixa

func (h *FinancialHandler) GetFluxoCaixa(w http.ResponseWriter, r *http.Request) {
	var dto request.GetFluxoCaixaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	results, err := h.getFluxoCaixaUC.Execute(r.Context(), dto.StartDate, dto.EndDate)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *FinancialHandler) GetFluxoProjetado(w http.ResponseWriter, r *http.Request) {
	var dto request.GetFluxoProjetadoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	results, err := h.getFluxoProjetadoUC.Execute(r.Context(), dto.StartDate)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *FinancialHandler) GetSaldoContas(w http.ResponseWriter, r *http.Request) {
	results, err := h.getSaldoContasUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// Tax Assessment

func (h *FinancialHandler) ApurarImpostos(w http.ResponseWriter, r *http.Request) {
	var dto request.ApurarImpostosDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	results, err := h.apurarImpostosUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, results)
}

func (h *FinancialHandler) GetTaxAssessment(w http.ResponseWriter, r *http.Request) {
	competencia := chi.URLParam(r, "competencia")
	results, err := h.getTaxAssessmentUC.List(r.Context(), competencia)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}
