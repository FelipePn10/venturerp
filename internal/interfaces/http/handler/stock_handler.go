package handler

import (
	"encoding/json"
	"errors"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type StockHandler struct {
	createMovementUC  *stock_uc.CreateStockMovementUseCase
	listMovementsUC   *stock_uc.ListStockMovementsUseCase
	getBalanceUC      *stock_uc.GetStockBalanceUseCase
	reserveStockUC    *stock_uc.ReserveStockUseCase
	releaseReserveUC  *stock_uc.ReleaseReservationUseCase
	consumeReserveUC  *stock_uc.ConsumeReservationUseCase
	createInventoryUC *stock_uc.CreateInventoryUseCase
	countInventoryUC  *stock_uc.CountInventoryItemUseCase
	adjustInventoryUC *stock_uc.AdjustInventoryUseCase
	closeInventoryUC  *stock_uc.CloseInventoryUseCase
	getInventoryUC    *stock_uc.GetInventoryUseCase
	listInventoriesUC *stock_uc.ListInventoriesUseCase
	registerLotUC     *stock_uc.RegisterLotUseCase
	listLotBalancesUC *stock_uc.ListLotBalancesUseCase
	getGenealogyUC    *stock_uc.GetLotGenealogyUseCase
	recalcCMUC        *stock_uc.RecalcConsumptionAverageUseCase
	getCMUC           *stock_uc.GetConsumptionAverageUseCase
	separacaoUC       *stock_uc.SeparacaoUseCase
}

// WithSeparacao liga a separação por endereço e a sugestão FEFO/FIFO.
func (h *StockHandler) WithSeparacao(uc *stock_uc.SeparacaoUseCase) *StockHandler {
	h.separacaoUC = uc
	return h
}

// WithConsumptionAverage attaches the consumption-average use cases (consumo médio).
func (h *StockHandler) WithConsumptionAverage(
	recalcUC *stock_uc.RecalcConsumptionAverageUseCase,
	getUC *stock_uc.GetConsumptionAverageUseCase,
) *StockHandler {
	h.recalcCMUC = recalcUC
	h.getCMUC = getUC
	return h
}

// WithLot attaches the lot-traceability use cases (registro/saldo/genealogia).
func (h *StockHandler) WithLot(
	registerLotUC *stock_uc.RegisterLotUseCase,
	listLotBalancesUC *stock_uc.ListLotBalancesUseCase,
	getGenealogyUC *stock_uc.GetLotGenealogyUseCase,
) *StockHandler {
	h.registerLotUC = registerLotUC
	h.listLotBalancesUC = listLotBalancesUC
	h.getGenealogyUC = getGenealogyUC
	return h
}

func NewStockHandler(
	createMovementUC *stock_uc.CreateStockMovementUseCase,
	listMovementsUC *stock_uc.ListStockMovementsUseCase,
	getBalanceUC *stock_uc.GetStockBalanceUseCase,
	reserveStockUC *stock_uc.ReserveStockUseCase,
	releaseReserveUC *stock_uc.ReleaseReservationUseCase,
	consumeReserveUC *stock_uc.ConsumeReservationUseCase,
	createInventoryUC *stock_uc.CreateInventoryUseCase,
	countInventoryUC *stock_uc.CountInventoryItemUseCase,
	adjustInventoryUC *stock_uc.AdjustInventoryUseCase,
	closeInventoryUC *stock_uc.CloseInventoryUseCase,
	getInventoryUC *stock_uc.GetInventoryUseCase,
	listInventoriesUC *stock_uc.ListInventoriesUseCase,
) *StockHandler {
	return &StockHandler{
		createMovementUC:  createMovementUC,
		listMovementsUC:   listMovementsUC,
		getBalanceUC:      getBalanceUC,
		reserveStockUC:    reserveStockUC,
		releaseReserveUC:  releaseReserveUC,
		consumeReserveUC:  consumeReserveUC,
		createInventoryUC: createInventoryUC,
		countInventoryUC:  countInventoryUC,
		adjustInventoryUC: adjustInventoryUC,
		closeInventoryUC:  closeInventoryUC,
		getInventoryUC:    getInventoryUC,
		listInventoriesUC: listInventoriesUC,
	}
}

// ---------- Stock Movements ----------

func (h *StockHandler) CreateMovement(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateStockMovementDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createMovementUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *StockHandler) ListMovements(w http.ResponseWriter, r *http.Request) {
	results, err := h.listMovementsUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) ListMovementsByItem(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	results, err := h.listMovementsUC.ByItem(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) ListMovementsByWarehouse(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "warehouseId")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador do almoxarifado inválido")
		return
	}
	results, err := h.listMovementsUC.ByWarehouse(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ---------- Stock Balance ----------

func (h *StockHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	itemCodeStr := r.URL.Query().Get("item_code")
	mask := r.URL.Query().Get("mask")
	warehouseStr := r.URL.Query().Get("warehouse_id")

	itemCode, err := strconv.ParseInt(itemCodeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	warehouseID, err := strconv.ParseInt(warehouseStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador do almoxarifado inválido")
		return
	}

	result, err := h.getBalanceUC.Execute(r.Context(), itemCode, mask, warehouseID)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *StockHandler) ListBalances(w http.ResponseWriter, r *http.Request) {
	results, err := h.getBalanceUC.List(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) ListBalancesByWarehouse(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "warehouseId")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador do almoxarifado inválido")
		return
	}
	results, err := h.getBalanceUC.ByWarehouse(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	itemFilter := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("item_code")); raw != "" {
		itemFilter, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || itemFilter <= 0 {
			security.RespondError(w, http.StatusBadRequest, "código do item inválido")
			return
		}
	}
	lotFilter := strings.TrimSpace(r.URL.Query().Get("lot"))
	if lotFilter == "" {
		lotFilter = strings.TrimSpace(r.URL.Query().Get("mask"))
	}
	filtered := results[:0]
	for _, balance := range results {
		if itemFilter > 0 && balance.ItemCode != itemFilter {
			continue
		}
		if lotFilter != "" && balance.Mask != lotFilter {
			continue
		}
		filtered = append(filtered, balance)
	}
	results = filtered
	page, perPage := positiveQueryInt(r, "page", 1), positiveQueryInt(r, "per_page", 50)
	if perPage > 200 {
		perPage = 200
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(len(results)))
	start := (page - 1) * perPage
	if start >= len(results) {
		results = results[:0]
	} else {
		end := start + perPage
		if end > len(results) {
			end = len(results)
		}
		results = results[start:end]
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func positiveQueryInt(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (h *StockHandler) ListBalancesByItem(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	results, err := h.getBalanceUC.ByItem(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// GetATP returns the available-to-promise of an item (optionally filtered by
// ?mask=). Total available = on-hand − reservations across all warehouses.
func (h *StockHandler) GetATP(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	mask := r.URL.Query().Get("mask")
	result, err := h.getBalanceUC.ATP(r.Context(), code, mask)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---------- Consumption Average ----------

func (h *StockHandler) RecalcConsumptionAverage(w http.ResponseWriter, r *http.Request) {
	if h.recalcCMUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "consumo médio não configurado")
		return
	}
	var dto request.RecalcConsumptionAverageDTO
	// Body is optional: an empty body recalculates all items with the default window.
	_ = json.NewDecoder(r.Body).Decode(&dto)
	result, err := h.recalcCMUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *StockHandler) GetConsumptionAverage(w http.ResponseWriter, r *http.Request) {
	if h.getCMUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "consumo médio não configurado")
		return
	}
	code, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	result, err := h.getCMUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---------- Lot Traceability ----------

func (h *StockHandler) RegisterLot(w http.ResponseWriter, r *http.Request) {
	if h.registerLotUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "rastreabilidade de lote não configurada")
		return
	}
	var dto request.RegisterLotDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.registerLotUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *StockHandler) ListLotBalances(w http.ResponseWriter, r *http.Request) {
	if h.listLotBalancesUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "rastreabilidade de lote não configurada")
		return
	}
	code, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	results, err := h.listLotBalancesUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) GetLotGenealogy(w http.ResponseWriter, r *http.Request) {
	if h.getGenealogyUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "rastreabilidade de lote não configurada")
		return
	}
	code, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	lot := chi.URLParam(r, "lot")
	result, err := h.getGenealogyUC.Execute(r.Context(), code, lot)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---------- Stock Reservations ----------

func (h *StockHandler) ReserveStock(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateReservationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.reserveStockUC.Execute(r.Context(), dto)
	if err != nil {
		if errors.Is(err, stockrepo.ErrInsufficientStock) {
			security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *StockHandler) ReleaseReservation(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador do inventário inválido")
		return
	}
	if err := h.releaseReserveUC.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockHandler) ConsumeReservation(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador do inventário inválido")
		return
	}
	if err := h.consumeReserveUC.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Physical Inventory ----------

func (h *StockHandler) CreateInventory(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateInventoryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createInventoryUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *StockHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador do inventário inválido")
		return
	}
	result, err := h.getInventoryUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *StockHandler) ListInventories(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	if statusFilter != "" {
		results, err := h.listInventoriesUC.ByStatus(r.Context(), statusFilter)
		if err != nil {
			security.RespondUseCaseError(w, err)
			return
		}
		security.RespondJSON(w, http.StatusOK, results)
		return
	}
	results, err := h.listInventoriesUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) CountInventoryItem(w http.ResponseWriter, r *http.Request) {
	var dto request.CountInventoryItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.countInventoryUC.Execute(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockHandler) AdjustInventoryItem(w http.ResponseWriter, r *http.Request) {
	var dto request.AdjustInventoryItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.adjustInventoryUC.Execute(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockHandler) CloseInventory(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador do inventário inválido")
		return
	}
	if err := h.closeInventoryUC.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockHandler) ListInventoryItems(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador do inventário inválido")
		return
	}
	results, err := h.getInventoryUC.ListItems(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// SugerirSeparacao responde de quais lotes e endereços tirar a quantidade
// pedida. GET /api/stock/separation/suggest/{itemCode}?qty=&warehouse_id=&mask=&rule=
func (h *StockHandler) SugerirSeparacao(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	quantidade, err := strconv.ParseFloat(r.URL.Query().Get("qty"), 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "informe a quantidade a separar em \"qty\"")
		return
	}
	almoxarifado, _ := strconv.ParseInt(r.URL.Query().Get("warehouse_id"), 10, 64)
	resultado, err := h.separacaoUC.SugerirFEFO(
		r.Context(), itemCode, r.URL.Query().Get("mask"), almoxarifado, quantidade, r.URL.Query().Get("rule"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, resultado)
}

// SaldoPorEndereco lista o estoque quebrado por endereço.
// GET /api/stock/balances/by-address?warehouse_id=&item_code=
func (h *StockHandler) SaldoPorEndereco(w http.ResponseWriter, r *http.Request) {
	almoxarifado, _ := strconv.ParseInt(r.URL.Query().Get("warehouse_id"), 10, 64)
	item, _ := strconv.ParseInt(r.URL.Query().Get("item_code"), 10, 64)
	saldos, err := h.separacaoUC.SaldoPorEndereco(r.Context(), almoxarifado, item)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, saldos)
}

// TransferirEndereco move material entre endereços do mesmo almoxarifado.
// POST /api/stock/separation/transfer
func (h *StockHandler) TransferirEndereco(w http.ResponseWriter, r *http.Request) {
	var dto request.TransferenciaEnderecoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	movimento, err := h.separacaoUC.TransferirEndereco(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, movimento)
}

// ApurarCurvaABC recalcula a classe ABC dos itens pelo valor consumido.
// POST /api/stock/abc/recalc  {"window_months":12,"cut_a_pct":80,"cut_b_pct":95}
func (h *StockHandler) ApurarCurvaABC(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		WindowMonths int     `json:"window_months"`
		CutAPct      float64 `json:"cut_a_pct"`
		CutBPct      float64 `json:"cut_b_pct"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&corpo)
	}
	resumo, err := h.separacaoUC.ApurarABC(r.Context(), corpo.WindowMonths, corpo.CutAPct, corpo.CutBPct)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, resumo)
}

// SugerirGuarda recomenda onde guardar o material recebido.
// GET /api/stock/putaway/suggest/{itemCode}?qty=&warehouse_id=&zone=&mask=
func (h *StockHandler) SugerirGuarda(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
		return
	}
	quantidade, err := strconv.ParseFloat(r.URL.Query().Get("qty"), 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "informe a quantidade a guardar em \"qty\"")
		return
	}
	almoxarifado, _ := strconv.ParseInt(r.URL.Query().Get("warehouse_id"), 10, 64)
	sugestoes, err := h.separacaoUC.SugerirGuarda(
		r.Context(), itemCode, r.URL.Query().Get("mask"), almoxarifado, quantidade, r.URL.Query().Get("zone"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, sugestoes)
}

// CriarOnda agrupa necessidades numa caminhada só. POST /api/stock/waves
func (h *StockHandler) CriarOnda(w http.ResponseWriter, r *http.Request) {
	var dto request.OndaDeSeparacaoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	onda, err := h.separacaoUC.CriarOnda(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, onda)
}

// ConfirmarOnda baixa o separado. POST /api/stock/waves/{code}/confirm
func (h *StockHandler) ConfirmarOnda(w http.ResponseWriter, r *http.Request) {
	h.encerrarOnda(w, r, true)
}

// CancelarOnda devolve o reservado. POST /api/stock/waves/{code}/cancel
func (h *StockHandler) CancelarOnda(w http.ResponseWriter, r *http.Request) {
	h.encerrarOnda(w, r, false)
}

func (h *StockHandler) encerrarOnda(w http.ResponseWriter, r *http.Request, confirmar bool) {
	codigo, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "número da onda inválido")
		return
	}
	var onda *stockentity.OndaDeSeparacao
	if confirmar {
		onda, err = h.separacaoUC.ConfirmarOnda(r.Context(), codigo)
	} else {
		onda, err = h.separacaoUC.CancelarOnda(r.Context(), codigo)
	}
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, onda)
}
