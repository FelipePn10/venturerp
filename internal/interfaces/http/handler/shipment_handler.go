package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ShipmentHandler exposes outbound logistics (expedição/romaneio).
type ShipmentHandler struct {
	uc *shipment_uc.ShipmentUseCase
}

func NewShipmentHandler(uc *shipment_uc.ShipmentUseCase) *ShipmentHandler {
	return &ShipmentHandler{uc: uc}
}

type createShipmentRequest struct {
	SalesOrderCode *int64  `json:"sales_order_code,omitempty"`
	CarrierCode    *int64  `json:"carrier_code,omitempty"`
	TotalVolumes   int     `json:"total_volumes"`
	TotalWeight    float64 `json:"total_weight"`
	Notes          *string `json:"notes,omitempty"`
	CreatedBy      string  `json:"created_by"`
}

func (h *ShipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	createdBy, _ := uuid.Parse(req.CreatedBy)
	result, err := h.uc.Create(r.Context(), shipment_uc.CreateShipmentInput{
		SalesOrderCode: req.SalesOrderCode,
		CarrierCode:    req.CarrierCode,
		TotalVolumes:   req.TotalVolumes,
		TotalWeight:    req.TotalWeight,
		Notes:          req.Notes,
		CreatedBy:      createdBy,
	})
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

type addShipmentItemRequest struct {
	Sequence           int     `json:"sequence"`
	ItemCode           int64   `json:"item_code"`
	SalesOrderItemCode *int64  `json:"sales_order_item_code,omitempty"`
	WarehouseID        *int64  `json:"warehouse_id,omitempty"`
	Quantity           float64 `json:"quantity"`
	Notes              *string `json:"notes,omitempty"`
}

func (h *ShipmentHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	var req addShipmentItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.AddItem(r.Context(), shipment_uc.AddShipmentItemInput{
		ShipmentCode:       code,
		Sequence:           req.Sequence,
		ItemCode:           req.ItemCode,
		SalesOrderItemCode: req.SalesOrderItemCode,
		WarehouseID:        req.WarehouseID,
		Quantity:           req.Quantity,
		Notes:              req.Notes,
	})
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ShipmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	result, err := h.uc.Get(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ShipmentHandler) List(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.List(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

type conferItemRequest struct {
	ItemID       int64   `json:"item_id"`
	ConferredQty float64 `json:"conferred_qty"`
}

func (h *ShipmentHandler) ConferItem(w http.ResponseWriter, r *http.Request) {
	var req conferItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if err := h.uc.ConferItem(r.Context(), req.ItemID, req.ConferredQty); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "conferred"})
}

func (h *ShipmentHandler) Confer(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.uc.Confer, "conferred")
}

func (h *ShipmentHandler) Ship(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.uc.Ship, "shipped")
}

func (h *ShipmentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.uc.Cancel, "cancelled")
}

func (h *ShipmentHandler) transition(w http.ResponseWriter, r *http.Request, fn func(context.Context, int64) error, ok string) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	if err := fn(r.Context(), code); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": ok})
}
