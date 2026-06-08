package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	// Reports
	getLivroEntradasUC        *financial_uc.GetLivroEntradasUseCase
	getLivroSaidasUC          *financial_uc.GetLivroSaidasUseCase
	getImpostosSaidasUC       *financial_uc.GetImpostosSaidasUseCase
	getImpostosEntradasUC     *financial_uc.GetImpostosEntradasUseCase
	getDREUC                  *financial_uc.GetDREUseCase
	getAgingReceberDetUC      *financial_uc.GetAgingReceberDetalhadoUseCase
	getAgingPagarDetUC        *financial_uc.GetAgingPagarDetalhadoUseCase
	getExtratoPorFornecedorUC *financial_uc.GetExtratoPorFornecedorUseCase
	getExtratoPorClienteUC    *financial_uc.GetExtratoPorClienteUseCase
	getProdutosVendidosUC     *financial_uc.GetProdutosVendidosUseCase
	getProdutosProduzidosUC   *financial_uc.GetProdutosProduzidosUseCase
	getHistoricoCustosUC      *financial_uc.GetHistoricoCustosUseCase
	getFichaTecnicaCustoUC    *financial_uc.GetFichaTecnicaCustoUseCase
	getCurvaABCClientesUC     *financial_uc.GetCurvaABCClientesUseCase
	getCurvaABCProdutosUC     *financial_uc.GetCurvaABCProdutosUseCase
	getComprasPeriodoUC       *financial_uc.GetComprasPeriodoUseCase
	// Conciliação
	importarOFXUC *financial_uc.ImportarOFXUseCase
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
	getLivroEntradasUC *financial_uc.GetLivroEntradasUseCase,
	getLivroSaidasUC *financial_uc.GetLivroSaidasUseCase,
	getImpostosSaidasUC *financial_uc.GetImpostosSaidasUseCase,
	getImpostosEntradasUC *financial_uc.GetImpostosEntradasUseCase,
	getDREUC *financial_uc.GetDREUseCase,
	getAgingReceberDetUC *financial_uc.GetAgingReceberDetalhadoUseCase,
	getAgingPagarDetUC *financial_uc.GetAgingPagarDetalhadoUseCase,
	getExtratoPorFornecedorUC *financial_uc.GetExtratoPorFornecedorUseCase,
	getExtratoPorClienteUC *financial_uc.GetExtratoPorClienteUseCase,
	getProdutosVendidosUC *financial_uc.GetProdutosVendidosUseCase,
	getProdutosProduzidosUC *financial_uc.GetProdutosProduzidosUseCase,
	getHistoricoCustosUC *financial_uc.GetHistoricoCustosUseCase,
	getFichaTecnicaCustoUC *financial_uc.GetFichaTecnicaCustoUseCase,
	getCurvaABCClientesUC *financial_uc.GetCurvaABCClientesUseCase,
	getCurvaABCProdutosUC *financial_uc.GetCurvaABCProdutosUseCase,
	getComprasPeriodoUC *financial_uc.GetComprasPeriodoUseCase,
	importarOFXUC *financial_uc.ImportarOFXUseCase,
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
		getLivroEntradasUC:        getLivroEntradasUC,
		getLivroSaidasUC:          getLivroSaidasUC,
		getImpostosSaidasUC:       getImpostosSaidasUC,
		getImpostosEntradasUC:     getImpostosEntradasUC,
		getDREUC:                  getDREUC,
		getAgingReceberDetUC:      getAgingReceberDetUC,
		getAgingPagarDetUC:        getAgingPagarDetUC,
		getExtratoPorFornecedorUC: getExtratoPorFornecedorUC,
		getExtratoPorClienteUC:    getExtratoPorClienteUC,
		getProdutosVendidosUC:     getProdutosVendidosUC,
		getProdutosProduzidosUC:   getProdutosProduzidosUC,
		getHistoricoCustosUC:      getHistoricoCustosUC,
		getFichaTecnicaCustoUC:    getFichaTecnicaCustoUC,
		getCurvaABCClientesUC:     getCurvaABCClientesUC,
		getCurvaABCProdutosUC:     getCurvaABCProdutosUC,
		getComprasPeriodoUC:       getComprasPeriodoUC,
		importarOFXUC:             importarOFXUC,
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

// ─── Reports helper ───────────────────────────────────────────────────────────

func parseDateRange(r *http.Request) (start, end string) {
	start = r.URL.Query().Get("start")
	end = r.URL.Query().Get("end")
	return
}

func parseTime(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// ─── R01 Livro de Entradas ────────────────────────────────────────────────────

func (h *FinancialHandler) GetLivroEntradas(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getLivroEntradasUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R02 Livro de Saídas ─────────────────────────────────────────────────────

func (h *FinancialHandler) GetLivroSaidas(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getLivroSaidasUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R03 Impostos Saídas ─────────────────────────────────────────────────────

func (h *FinancialHandler) GetImpostosSaidas(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getImpostosSaidasUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R04 Impostos Entradas ───────────────────────────────────────────────────

func (h *FinancialHandler) GetImpostosEntradas(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getImpostosEntradasUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R06 DRE ─────────────────────────────────────────────────────────────────

func (h *FinancialHandler) GetDRE(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	result, err := h.getDREUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ─── R09 Aging Receber Detalhado ─────────────────────────────────────────────

func (h *FinancialHandler) GetAgingReceberDetalhado(w http.ResponseWriter, r *http.Request) {
	results, err := h.getAgingReceberDetUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R10 Aging Pagar Detalhado ───────────────────────────────────────────────

func (h *FinancialHandler) GetAgingPagarDetalhado(w http.ResponseWriter, r *http.Request) {
	results, err := h.getAgingPagarDetUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R11 Extrato por Fornecedor ──────────────────────────────────────────────

func (h *FinancialHandler) GetExtratoPorFornecedor(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	results, err := h.getExtratoPorFornecedorUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R12 Extrato por Cliente ─────────────────────────────────────────────────

func (h *FinancialHandler) GetExtratoPorCliente(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	results, err := h.getExtratoPorClienteUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R13 Produtos Vendidos ───────────────────────────────────────────────────

func (h *FinancialHandler) GetProdutosVendidos(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getProdutosVendidosUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R14 Produtos Produzidos ─────────────────────────────────────────────────

func (h *FinancialHandler) GetProdutosProduzidos(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getProdutosProduzidosUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R15 Histórico de Custos ─────────────────────────────────────────────────

func (h *FinancialHandler) GetHistoricoCustos(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getHistoricoCustosUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R16 Ficha Técnica com Custo ─────────────────────────────────────────────

func (h *FinancialHandler) GetFichaTecnicaCusto(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "item_code")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item_code")
		return
	}
	results, err := h.getFichaTecnicaCustoUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R17 Curva ABC Clientes ──────────────────────────────────────────────────

func (h *FinancialHandler) GetCurvaABCClientes(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getCurvaABCClientesUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R18 Curva ABC Produtos ──────────────────────────────────────────────────

func (h *FinancialHandler) GetCurvaABCProdutos(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getCurvaABCProdutosUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── R19 Compras no Período ──────────────────────────────────────────────────

func (h *FinancialHandler) GetComprasPeriodo(w http.ResponseWriter, r *http.Request) {
	start, end := parseDateRange(r)
	results, err := h.getComprasPeriodoUC.Execute(r.Context(),
		mustParseDate(parseTime(start, "2000-01-01")),
		mustParseDate(parseTime(end, "2099-12-31")))
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── Conciliação Bancária ────────────────────────────────────────────────────

func (h *FinancialHandler) ImportarOFX(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "conta_id")
	contaID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid conta_id")
		return
	}
	var body struct {
		OFXContent string `json:"ofx_content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.importarOFXUC.Execute(r.Context(), contaID, body.OFXContent)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t
}
