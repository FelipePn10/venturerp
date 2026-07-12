package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type ProductionOrderHandler struct {
	*security.BaseHandler
	createUC             *production_order_uc.CreateProductionOrderUseCase
	getByCodeUC          *production_order_uc.GetProductionOrderUseCase
	listUC               *production_order_uc.ListProductionOrdersUseCase
	startUC              *production_order_uc.StartProductionOrderUseCase
	addAppointmentUC     *production_order_uc.AddAppointmentUseCase
	addConsumptionUC     *production_order_uc.AddConsumptionUseCase
	completeUC           *production_order_uc.CompleteProductionOrderUseCase
	closeUC              *production_order_uc.CloseProductionOrderUseCase
	cancelUC             *production_order_uc.CancelProductionOrderUseCase
	getAppointmentsUC    *production_order_uc.GetAppointmentsUseCase
	getConsumptionsUC    *production_order_uc.GetConsumptionsUseCase
	orderOpsUC           *production_order_uc.OrderOperationsUseCase
	settleCostUC         *production_order_uc.SettleProductionCostUseCase
	getCostUC            *production_order_uc.GetProductionCostUseCase
	returnScrapUC        *production_order_uc.ReturnScrapUseCase
	operationalUC        *production_order_uc.OperationalConsultationUseCase
	materialControlUC    *production_order_uc.ProductionMaterialControlUseCase
	maintainUC           *production_order_uc.MaintainProductionOrderUseCase
	deliveryCandidatesUC *production_order_uc.ListDeliveryCandidatesUseCase
}

func (h *ProductionOrderHandler) DeliveryCandidates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	result, err := h.deliveryCandidatesUC.Execute(r.Context(), production_order_uc.DeliveryCandidateFilter{OrderFrom: reportInt64(q.Get("order_from")), OrderTo: reportInt64(q.Get("order_to")), ItemFrom: reportInt64(q.Get("item_from")), ItemTo: reportInt64(q.Get("item_to")), PlannerFrom: reportInt64(q.Get("planner_from")), PlannerTo: reportInt64(q.Get("planner_to")), From: reportDate(q.Get("from")), To: reportDate(q.Get("to")), OrderType: q.Get("order_type")})
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) Maintain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.MaintainProductionOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.ID = id
	result, err := h.maintainUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) ListMaterials(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.materialControlUC.List(r.Context(), orderID, r.URL.Query().Get("kind"))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) AddMaterial(w http.ResponseWriter, r *http.Request) {
	var dto request.AddProductionMaterialDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.materialControlUC.Add(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProductionOrderHandler) ReplaceMaterial(w http.ResponseWriter, r *http.Request) {
	var dto request.ReplaceProductionMaterialDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.materialControlUC.Replace(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) DeleteMaterial(w http.ResponseWriter, r *http.Request) {
	materialID, err := strconv.ParseInt(chi.URLParam(r, "materialID"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid material id")
		return
	}
	if err := h.materialControlUC.Delete(r.Context(), materialID); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductionOrderHandler) AllocateLots(w http.ResponseWriter, r *http.Request) {
	var dto request.AllocateProductionLotsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.materialControlUC.AllocateLots(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) AllocateLotsBatch(w http.ResponseWriter, r *http.Request) {
	var dto request.BatchAllocateProductionLotsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.materialControlUC.AllocateLotsBatch(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) AddScrapDestination(w http.ResponseWriter, r *http.Request) {
	var dto request.AddScrapDestinationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.materialControlUC.AddScrap(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProductionOrderHandler) DeleteScrapDestination(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "destinationID"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid destination id")
		return
	}
	if err := h.materialControlUC.DeleteScrap(r.Context(), id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *ProductionOrderHandler) UpdateScrapDestination(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "destinationID"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid destination id")
		return
	}
	var dto request.AddScrapDestinationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.materialControlUC.UpdateScrap(r.Context(), id, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
func (h *ProductionOrderHandler) Maintenance(w http.ResponseWriter, r *http.Request) {
	var id *int64
	if raw := r.URL.Query().Get("id"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		id = &value
	}
	result, err := h.materialControlUC.Maintenance(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
func (h *ProductionOrderHandler) ConfigureManufacturingStock(w http.ResponseWriter, r *http.Request) {
	var dto request.ConfigureManufacturingStockDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.materialControlUC.ConfigureStock(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *ProductionOrderHandler) ConfigureManufacturingItemStock(w http.ResponseWriter, r *http.Request) {
	var dto request.ConfigureManufacturingItemStockDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.materialControlUC.ConfigureItemStock(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *ProductionOrderHandler) ConfigureWarehouseAddress(w http.ResponseWriter, r *http.Request) {
	var dto request.ConfigureWarehouseAddressDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.materialControlUC.ConfigureAddress(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *ProductionOrderHandler) ConfigureTemporaryLot(w http.ResponseWriter, r *http.Request) {
	var dto request.ConfigureTemporaryProductionLotDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.materialControlUC.ConfigureTemporaryLot(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) ConfigureWMS(w http.ResponseWriter, r *http.Request) {
	var dto request.ConfigureWMSWarehouseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.materialControlUC.ConfigureWMS(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) Operational(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.operationalUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateProductionOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProductionOrderHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.getByCodeUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *ProductionOrderHandler) Start(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.StartProductionOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.ID = id
	result, err := h.startUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) AddAppointment(w http.ResponseWriter, r *http.Request) {
	var dto request.AddAppointmentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.addAppointmentUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProductionOrderHandler) AddConsumption(w http.ResponseWriter, r *http.Request) {
	var dto request.AddConsumptionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.addConsumptionUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProductionOrderHandler) Complete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.CompleteProductionOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.ID = id
	result, err := h.completeUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) Close(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.closeUC.Execute(r.Context(), request.CloseProductionOrderDTO{ID: id})
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.cancelUC.Execute(r.Context(), request.CancelProductionOrderDTO{ID: id})
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) GetAppointments(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	results, err := h.getAppointmentsUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *ProductionOrderHandler) GetConsumptions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	results, err := h.getConsumptionsUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ─── cost settlement (custo real da OF) ───────────────────────────────────────

func (h *ProductionOrderHandler) SettleCost(w http.ResponseWriter, r *http.Request) {
	if h.settleCostUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "cost settlement not configured")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.settleCostUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) GetCost(w http.ResponseWriter, r *http.Request) {
	if h.getCostUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "cost settlement not configured")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.getCostUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ─── scrap return (sucata valorizada) ─────────────────────────────────────────

func (h *ProductionOrderHandler) ReturnScrap(w http.ResponseWriter, r *http.Request) {
	if h.returnScrapUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "scrap return not configured")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.ReturnScrapDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.ProductionOrderID = id
	result, err := h.returnScrapUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

// ─── order operations (route explosion) ──────────────────────────────────────

func (h *ProductionOrderHandler) ExplodeRoute(w http.ResponseWriter, r *http.Request) {
	if h.orderOpsUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "order operations not configured")
		return
	}
	var dto request.ExplodeRouteDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.orderOpsUC.ExplodeRoute(r.Context(), dto.OrderID, dto.RouteID)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProductionOrderHandler) ListOrderOperations(w http.ResponseWriter, r *http.Request) {
	if h.orderOpsUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "order operations not configured")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.orderOpsUC.ListOperations(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProductionOrderHandler) AdvanceOperation(w http.ResponseWriter, r *http.Request) {
	if h.orderOpsUC == nil {
		security.RespondError(w, http.StatusNotImplemented, "order operations not configured")
		return
	}
	var dto request.AdvanceOperationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.orderOpsUC.AdvanceOperation(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
