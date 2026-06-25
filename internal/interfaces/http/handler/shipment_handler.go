package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc"
	shipentity "github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	shiprepo "github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ShipmentHandler struct {
	uc         *shipment_uc.ShipmentUseCase
	autoFillUC *shipment_uc.ShipmentAutoFillUseCase
	exportUC   *shipment_uc.ShipmentExportUseCase
}

func NewShipmentHandler(uc *shipment_uc.ShipmentUseCase) *ShipmentHandler {
	return &ShipmentHandler{uc: uc}
}

func (h *ShipmentHandler) WithAutoFill(uc *shipment_uc.ShipmentAutoFillUseCase) *ShipmentHandler {
	h.autoFillUC = uc
	return h
}

func (h *ShipmentHandler) WithExport(uc *shipment_uc.ShipmentExportUseCase) *ShipmentHandler {
	h.exportUC = uc
	return h
}

// actingUser returns the authenticated user's ID from the request context, set
// by the JWT middleware. Falls back to uuid.Nil if absent (unauthenticated).
func actingUser(r *http.Request) uuid.UUID {
	if u, ok := r.Context().Value(contextkey.UserKey).(*appsecurity.AuthUser); ok && u != nil {
		if id, err := uuid.Parse(u.ID); err == nil {
			return id
		}
	}
	return uuid.Nil
}

type createShipmentRequest struct {
	ReferenceType       *string `json:"reference_type,omitempty"`
	SalesOrderCode      *int64  `json:"sales_order_code,omitempty"`
	PurchaseOrderCode   *int64  `json:"purchase_order_code,omitempty"`
	ProductionOrderCode *int64  `json:"production_order_code,omitempty"`
	CarrierCode         *int64  `json:"carrier_code,omitempty"`
	TotalVolumes        int     `json:"total_volumes"`
	TotalNetWeight      float64 `json:"total_net_weight"`
	TotalGrossWeight    float64 `json:"total_gross_weight"`
	TotalCubageM3       float64 `json:"total_cubage_m3"`
	Notes               *string `json:"notes,omitempty"`
}

func (h *ShipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	input := shipment_uc.CreateShipmentInput{
		SalesOrderCode:      req.SalesOrderCode,
		PurchaseOrderCode:   req.PurchaseOrderCode,
		ProductionOrderCode: req.ProductionOrderCode,
		CarrierCode:         req.CarrierCode,
		TotalVolumes:        req.TotalVolumes,
		TotalNetWeight:      req.TotalNetWeight,
		TotalGrossWeight:    req.TotalGrossWeight,
		TotalCubageM3:       req.TotalCubageM3,
		Notes:               req.Notes,
		CreatedBy:           actingUser(r),
	}
	if req.ReferenceType != nil {
		rt := shipentity.ShipmentReferenceType(*req.ReferenceType)
		input.ReferenceType = &rt
	}
	result, err := h.uc.Create(r.Context(), input)
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
	UnitNetWeight      float64 `json:"unit_net_weight"`
	UnitGrossWeight    float64 `json:"unit_gross_weight"`
	Notes              *string `json:"notes,omitempty"`
}

func (h *ShipmentHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	code, err := h.codeParam(r)
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
		UnitNetWeight:      req.UnitNetWeight,
		UnitGrossWeight:    req.UnitGrossWeight,
		Notes:              req.Notes,
	})
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ShipmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	code, err := h.codeParam(r)
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

// List supports query filters: ?status=&carrier_code=&from=&to=&limit=&offset=.
func (h *ShipmentHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f shiprepo.ShipmentFilter
	if s := q.Get("status"); s != "" {
		st := shipentity.ShipmentStatus(s)
		f.Status = &st
	}
	if c := q.Get("carrier_code"); c != "" {
		if v, err := strconv.ParseInt(c, 10, 64); err == nil {
			f.CarrierCode = &v
		}
	}
	if d := q.Get("from"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			f.From = &t
		}
	}
	if d := q.Get("to"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			t = t.AddDate(0, 0, 1) // inclusive end-of-day
			f.To = &t
		}
	}
	f.Limit, _ = strconv.Atoi(q.Get("limit"))
	f.Offset, _ = strconv.Atoi(q.Get("offset"))

	result, err := h.uc.List(r.Context(), f)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ShipmentHandler) ListBySalesOrder(w http.ResponseWriter, r *http.Request) {
	h.listByOrder(w, r, h.uc.ListBySalesOrder, "sales order")
}

func (h *ShipmentHandler) ListByPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	h.listByOrder(w, r, h.uc.ListByPurchaseOrder, "purchase order")
}

func (h *ShipmentHandler) ListByProductionOrder(w http.ResponseWriter, r *http.Request) {
	h.listByOrder(w, r, h.uc.ListByProductionOrder, "production order")
}

func (h *ShipmentHandler) listByOrder(w http.ResponseWriter, r *http.Request, fn func(context.Context, int64) ([]*response.ShipmentResponse, error), label string) {
	code, err := strconv.ParseInt(chi.URLParam(r, "orderCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid "+label+" code")
		return
	}
	result, err := fn(r.Context(), code)
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
	code, err := h.codeParam(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	var req conferItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if err := h.uc.ConferItem(r.Context(), code, req.ItemID, req.ConferredQty); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "conferred"})
}

func (h *ShipmentHandler) Separate(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, func(ctx context.Context, code int64) error {
		return h.uc.Separate(ctx, code, actingUser(r))
	}, "separated")
}

func (h *ShipmentHandler) Confer(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, func(ctx context.Context, code int64) error {
		return h.uc.Confer(ctx, code, actingUser(r))
	}, "conferred")
}

type shipRequest struct {
	AcceptDivergences bool `json:"accept_divergences"`
}

func (h *ShipmentHandler) Ship(w http.ResponseWriter, r *http.Request) {
	var req shipRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body optional
	h.transition(w, r, func(ctx context.Context, code int64) error {
		return h.uc.Ship(ctx, code, actingUser(r), req.AcceptDivergences)
	}, "shipped")
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

func (h *ShipmentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	var req cancelRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	h.transition(w, r, func(ctx context.Context, code int64) error {
		return h.uc.Cancel(ctx, code, actingUser(r), req.Reason)
	}, "cancelled")
}

type updateTransportRequest struct {
	CarrierCode       *int64  `json:"carrier_code,omitempty"`
	FreightModality   *string `json:"freight_modality,omitempty"`
	FreightValue      float64 `json:"freight_value"`
	InsuranceValue    float64 `json:"insurance_value"`
	VehiclePlate      *string `json:"vehicle_plate,omitempty"`
	DriverName        *string `json:"driver_name,omitempty"`
	DriverDocument    *string `json:"driver_document,omitempty"`
	ANTTCode          *string `json:"antt_code,omitempty"`
	Seals             *string `json:"seals,omitempty"`
	EstimatedDelivery *string `json:"estimated_delivery,omitempty"` // YYYY-MM-DD
}

func (h *ShipmentHandler) UpdateTransport(w http.ResponseWriter, r *http.Request) {
	code, err := h.codeParam(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	var req updateTransportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	t := shiprepo.TransportInput{
		CarrierCode:     req.CarrierCode,
		FreightModality: req.FreightModality,
		FreightValue:    req.FreightValue,
		InsuranceValue:  req.InsuranceValue,
		VehiclePlate:    req.VehiclePlate,
		DriverName:      req.DriverName,
		DriverDocument:  req.DriverDocument,
		ANTTCode:        req.ANTTCode,
		Seals:           req.Seals,
	}
	if req.EstimatedDelivery != nil && *req.EstimatedDelivery != "" {
		if d, err := time.Parse("2006-01-02", *req.EstimatedDelivery); err == nil {
			t.EstimatedDelivery = &d
		}
	}
	result, err := h.uc.UpdateTransport(r.Context(), code, t, actingUser(r))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

type addVolumeRequest struct {
	VolumeNumber int     `json:"volume_number"`
	PackageType  string  `json:"package_type"`
	NetWeight    float64 `json:"net_weight"`
	GrossWeight  float64 `json:"gross_weight"`
	LengthCm     float64 `json:"length_cm"`
	WidthCm      float64 `json:"width_cm"`
	HeightCm     float64 `json:"height_cm"`
	Marking      *string `json:"marking,omitempty"`
	Contents     *string `json:"contents,omitempty"`
}

func (h *ShipmentHandler) AddVolume(w http.ResponseWriter, r *http.Request) {
	code, err := h.codeParam(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	var req addVolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.AddVolume(r.Context(), shipment_uc.AddVolumeInput{
		ShipmentCode: code,
		VolumeNumber: req.VolumeNumber,
		PackageType:  req.PackageType,
		NetWeight:    req.NetWeight,
		GrossWeight:  req.GrossWeight,
		LengthCm:     req.LengthCm,
		WidthCm:      req.WidthCm,
		HeightCm:     req.HeightCm,
		Marking:      req.Marking,
		Contents:     req.Contents,
	})
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ShipmentHandler) ListVolumes(w http.ResponseWriter, r *http.Request) {
	code, err := h.codeParam(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	result, err := h.uc.ListVolumes(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ShipmentHandler) DeleteVolume(w http.ResponseWriter, r *http.Request) {
	code, err := h.codeParam(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	volumeID, err := strconv.ParseInt(chi.URLParam(r, "volumeID"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid volume id")
		return
	}
	if err := h.uc.DeleteVolume(r.Context(), code, volumeID); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type linkFiscalExitRequest struct {
	FiscalExitID *int64  `json:"fiscal_exit_id,omitempty"`
	NFeNumber    *int64  `json:"nfe_number,omitempty"`
	NFeKey       *string `json:"nfe_key,omitempty"`
}

func (h *ShipmentHandler) LinkFiscalExit(w http.ResponseWriter, r *http.Request) {
	code, err := h.codeParam(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	var req linkFiscalExitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if err := h.uc.LinkFiscalExit(r.Context(), code, req.FiscalExitID, req.NFeNumber, req.NFeKey, actingUser(r)); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "linked"})
}

func (h *ShipmentHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	code, err := h.codeParam(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	result, err := h.uc.ListEvents(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

type autoFillSalesOrderRequest struct {
	SalesOrderCode int64 `json:"sales_order_code"`
}

func (h *ShipmentHandler) AutoFillFromSalesOrder(w http.ResponseWriter, r *http.Request) {
	if h.autoFillUC == nil {
		jsonError(w, http.StatusNotImplemented, "auto-fill not configured")
		return
	}
	var req autoFillSalesOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.autoFillUC.AutoFillFromSalesOrder(r.Context(), req.SalesOrderCode, actingUser(r))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

type autoFillPurchaseOrderRequest struct {
	PurchaseOrderCode int64 `json:"purchase_order_code"`
}

func (h *ShipmentHandler) AutoFillFromPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	if h.autoFillUC == nil {
		jsonError(w, http.StatusNotImplemented, "auto-fill not configured")
		return
	}
	var req autoFillPurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.autoFillUC.AutoFillFromPurchaseOrder(r.Context(), req.PurchaseOrderCode, actingUser(r))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

type autoFillProductionOrderRequest struct {
	ProductionOrderCode int64 `json:"production_order_code"`
}

func (h *ShipmentHandler) AutoFillFromProductionOrder(w http.ResponseWriter, r *http.Request) {
	if h.autoFillUC == nil {
		jsonError(w, http.StatusNotImplemented, "auto-fill not configured")
		return
	}
	var req autoFillProductionOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.autoFillUC.AutoFillFromProductionOrder(r.Context(), req.ProductionOrderCode, actingUser(r))
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ShipmentHandler) ExportPDF(w http.ResponseWriter, r *http.Request) {
	if h.exportUC == nil {
		jsonError(w, http.StatusNotImplemented, "export not configured")
		return
	}
	h.exportFile(w, r, "pdf", h.exportUC.GeneratePDF)
}

func (h *ShipmentHandler) ExportXLSX(w http.ResponseWriter, r *http.Request) {
	if h.exportUC == nil {
		jsonError(w, http.StatusNotImplemented, "export not configured")
		return
	}
	h.exportFile(w, r, "xlsx", h.exportUC.GenerateXLSX)
}

func (h *ShipmentHandler) exportFile(w http.ResponseWriter, r *http.Request, format string, generate func(context.Context, int64) ([]byte, error)) {
	code, err := h.codeParam(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid shipment code")
		return
	}
	data, err := generate(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	contentType := "application/pdf"
	ext := "pdf"
	if format == "xlsx" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		ext = "xlsx"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="romaneio_%d.%s"`, code, ext))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *ShipmentHandler) transition(w http.ResponseWriter, r *http.Request, fn func(context.Context, int64) error, ok string) {
	code, err := h.codeParam(r)
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

func (h *ShipmentHandler) codeParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
}
