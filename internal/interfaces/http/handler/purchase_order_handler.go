package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type PurchaseOrderHandler struct {
	createUC         *purchase_order_uc.CreatePurchaseOrderUseCase
	updateUC         *purchase_order_uc.UpdatePurchaseOrderUseCase
	getUC            *purchase_order_uc.GetPurchaseOrderUseCase
	listUC           *purchase_order_uc.ListPurchaseOrdersUseCase
	listBySupplierUC *purchase_order_uc.ListPurchaseOrdersBySupplierUseCase
	listByStatusUC   *purchase_order_uc.ListPurchaseOrdersByStatusUseCase
	cancelUC         *purchase_order_uc.CancelPurchaseOrderUseCase
	receiveUC        *purchase_order_uc.ReceivePurchaseOrderUseCase
	approveUC        *purchase_order_uc.ApprovePurchaseOrderUseCase
	consultUC        *purchase_order_uc.ConsultPurchaseOrdersUseCase
}

func NewPurchaseOrderHandler(
	createUC *purchase_order_uc.CreatePurchaseOrderUseCase,
	updateUC *purchase_order_uc.UpdatePurchaseOrderUseCase,
	getUC *purchase_order_uc.GetPurchaseOrderUseCase,
	listUC *purchase_order_uc.ListPurchaseOrdersUseCase,
	listBySupplierUC *purchase_order_uc.ListPurchaseOrdersBySupplierUseCase,
	listByStatusUC *purchase_order_uc.ListPurchaseOrdersByStatusUseCase,
	cancelUC *purchase_order_uc.CancelPurchaseOrderUseCase,
	receiveUC *purchase_order_uc.ReceivePurchaseOrderUseCase,
	approveUC *purchase_order_uc.ApprovePurchaseOrderUseCase,
	consultUC *purchase_order_uc.ConsultPurchaseOrdersUseCase,
) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{
		createUC:         createUC,
		updateUC:         updateUC,
		getUC:            getUC,
		listUC:           listUC,
		listBySupplierUC: listBySupplierUC,
		listByStatusUC:   listByStatusUC,
		cancelUC:         cancelUC,
		receiveUC:        receiveUC,
		approveUC:        approveUC,
		consultUC:        consultUC,
	}
}

func (h *PurchaseOrderHandler) Consult(w http.ResponseWriter, r *http.Request) {
	f, err := purchaseOrderConsultationFilter(r)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.consultUC.Execute(r.Context(), f)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PurchaseOrderHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	orderCode, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid order code")
		return
	}
	attachmentID, err := strconv.ParseInt(chi.URLParam(r, "attachmentID"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid attachment id")
		return
	}
	file, err := h.consultUC.DownloadAttachment(r.Context(), orderCode, attachmentID)
	if errors.Is(err, purchase_order_uc.ErrAttachmentNotFound) {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	name := strings.ReplaceAll(filepath.Base(file.FileName), `"`, "")
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.FileSize, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Content)
}

func purchaseOrderConsultationFilter(r *http.Request) (purchase_order_uc.PurchaseOrderConsultationFilter, error) {
	q := r.URL.Query()
	var f purchase_order_uc.PurchaseOrderConsultationFilter
	var err error
	for key, dst := range map[string]**int64{"order_from": &f.OrderFrom, "order_to": &f.OrderTo, "supplier_from": &f.SupplierFrom, "supplier_to": &f.SupplierTo, "request_type": &f.RequestTypeCode, "item_from": &f.ItemFrom, "item_to": &f.ItemTo, "buyer": &f.BuyerCode, "import_from": &f.ImportFrom, "import_to": &f.ImportTo} {
		if *dst, err = parseOptionalInt64(q.Get(key)); err != nil {
			return f, fmt.Errorf("invalid %s", key)
		}
	}
	for key, dst := range map[string]**time.Time{"base_date": &f.BaseDate, "emission_from": &f.EmissionFrom, "emission_to": &f.EmissionTo, "delivery_from": &f.DeliveryFrom, "delivery_to": &f.DeliveryTo} {
		if *dst, err = parseOptionalDate(q.Get(key)); err != nil {
			return f, fmt.Errorf("invalid %s", key)
		}
	}
	for key, dst := range map[string]*bool{"all_items": &f.AllItems, "convert": &f.Convert, "only_kanban": &f.OnlyKanban} {
		if *dst, err = parseOptionalBool(q.Get(key)); err != nil {
			return f, fmt.Errorf("invalid %s", key)
		}
	}
	f.Position = q.Get("position")
	f.TargetCurrency = q.Get("target_currency")
	f.OrderType = q.Get("type")
	f.Limit = 100
	if q.Get("limit") != "" {
		f.Limit, err = strconv.Atoi(q.Get("limit"))
		if err != nil {
			return f, fmt.Errorf("invalid limit")
		}
	}
	if q.Get("offset") != "" {
		f.Offset, err = strconv.Atoi(q.Get("offset"))
		if err != nil {
			return f, fmt.Errorf("invalid offset")
		}
	}
	return f, nil
}

func parseOptionalInt64(s string) (*int64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func parseOptionalDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	v, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
func parseOptionalBool(s string) (bool, error) {
	if s == "" {
		return false, nil
	}
	return strconv.ParseBool(s)
}

func (h *PurchaseOrderHandler) Approve(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.approveUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PurchaseOrderHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.approveUC.Authorize(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PurchaseOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePurchaseOrderDTO
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

func (h *PurchaseOrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.UpdatePurchaseOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PurchaseOrderHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *PurchaseOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *PurchaseOrderHandler) ListBySupplier(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "supplierCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid supplier code")
		return
	}
	results, err := h.listBySupplierUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *PurchaseOrderHandler) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	results, err := h.listByStatusUC.Execute(r.Context(), status)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *PurchaseOrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := h.cancelUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PurchaseOrderHandler) Receive(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.ReceivePurchaseOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.PurchaseOrderCode = code
	result, err := h.receiveUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
