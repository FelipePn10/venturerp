package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc"
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
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *StockHandler) ListMovements(w http.ResponseWriter, r *http.Request) {
	results, err := h.listMovementsUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) ListMovementsByItem(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	results, err := h.listMovementsUC.ByItem(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) ListMovementsByWarehouse(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "warehouseId")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid warehouse id")
		return
	}
	results, err := h.listMovementsUC.ByWarehouse(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
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
		security.RespondError(w, http.StatusBadRequest, "invalid item_code")
		return
	}
	warehouseID, err := strconv.ParseInt(warehouseStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid warehouse_id")
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
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) ListBalancesByWarehouse(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "warehouseId")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid warehouse id")
		return
	}
	results, err := h.getBalanceUC.ByWarehouse(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) ListBalancesByItem(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	results, err := h.getBalanceUC.ByItem(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
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
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	mask := r.URL.Query().Get("mask")
	result, err := h.getBalanceUC.ATP(r.Context(), code, mask)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---------- Consumption Average ----------

func (h *StockHandler) RecalcConsumptionAverage(w http.ResponseWriter, r *http.Request) {
	if h.recalcCMUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "consumption average not configured")
		return
	}
	var dto request.RecalcConsumptionAverageDTO
	// Body is optional: an empty body recalculates all items with the default window.
	_ = json.NewDecoder(r.Body).Decode(&dto)
	result, err := h.recalcCMUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *StockHandler) GetConsumptionAverage(w http.ResponseWriter, r *http.Request) {
	if h.getCMUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "consumption average not configured")
		return
	}
	code, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
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
		security.RespondError(w, http.StatusNotImplemented, "lot traceability not configured")
		return
	}
	var dto request.RegisterLotDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.registerLotUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *StockHandler) ListLotBalances(w http.ResponseWriter, r *http.Request) {
	if h.listLotBalancesUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "lot traceability not configured")
		return
	}
	code, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	results, err := h.listLotBalancesUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *StockHandler) GetLotGenealogy(w http.ResponseWriter, r *http.Request) {
	if h.getGenealogyUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "lot traceability not configured")
		return
	}
	code, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	lot := chi.URLParam(r, "lot")
	result, err := h.getGenealogyUC.Execute(r.Context(), code, lot)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
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
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *StockHandler) ReleaseReservation(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.releaseReserveUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockHandler) ConsumeReservation(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.consumeReserveUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
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
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *StockHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
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
			security.RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		security.RespondJSON(w, http.StatusOK, results)
		return
	}
	results, err := h.listInventoriesUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
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
		security.RespondError(w, http.StatusInternalServerError, err.Error())
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
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockHandler) CloseInventory(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.closeInventoryUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StockHandler) ListInventoryItems(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "id")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	results, err := h.getInventoryUC.ListItems(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}
