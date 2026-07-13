package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_price_uc"
	priceRepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository"
	"github.com/go-chi/chi/v5"
)

type PurchasePriceHandler struct {
	uc *purchase_price_uc.PurchasePriceUseCase
}

func NewPurchasePriceHandler(uc *purchase_price_uc.PurchasePriceUseCase) *PurchasePriceHandler {
	return &PurchasePriceHandler{uc: uc}
}

func (h *PurchasePriceHandler) CreateTable(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePurchasePriceTableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.CreateTable(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *PurchasePriceHandler) UpdateTable(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdatePurchasePriceTableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.UpdateTable(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) GetTable(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	res, err := h.uc.GetTable(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	var supplier *int64
	if raw := r.URL.Query().Get("supplier_code"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid supplier_code")
			return
		}
		supplier = &v
	}
	res, err := h.uc.ListTables(r.Context(), supplier, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var classificationID *int64
	if raw := r.URL.Query().Get("classification_id"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid classification_id")
			return
		}
		classificationID = &v
	}
	mode, order := r.URL.Query().Get("mode"), r.URL.Query().Get("order")
	if mode == "" {
		mode = "INTERNAL"
	}
	if order == "" {
		order = "NUMERIC"
	}
	res, err := h.uc.ListCandidates(r.Context(), code, mode, order, classificationID)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) CopyAdjustments(w http.ResponseWriter, r *http.Request) {
	var dto request.CopyPriceAdjustmentsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if err := h.uc.CopyAdjustments(r.Context(), dto); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PurchasePriceHandler) ListSourcePrices(w http.ResponseWriter, r *http.Request) {
	start, err := time.Parse("2006-01-02", r.URL.Query().Get("start"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "start is required in YYYY-MM-DD")
		return
	}
	end, err := time.Parse("2006-01-02", r.URL.Query().Get("end"))
	if err != nil || end.Before(start) {
		jsonError(w, http.StatusBadRequest, "invalid end date")
		return
	}
	var supplier, table *int64
	if raw := r.URL.Query().Get("supplier_code"); raw != "" {
		v, e := strconv.ParseInt(raw, 10, 64)
		if e != nil {
			jsonError(w, http.StatusBadRequest, "invalid supplier_code")
			return
		}
		supplier = &v
	}
	if raw := r.URL.Query().Get("table_code"); raw != "" {
		v, e := strconv.ParseInt(raw, 10, 64)
		if e != nil {
			jsonError(w, http.StatusBadRequest, "invalid table_code")
			return
		}
		table = &v
	}
	res, err := h.uc.ListSourcePrices(r.Context(), priceRepo.SourceFilter{SupplierCode: supplier, TableCode: table, Start: start, End: end, Source: r.URL.Query().Get("source")})
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) ApplySourcePrices(w http.ResponseWriter, r *http.Request) {
	var dto request.ApplyPurchasePriceSourcesDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	n, err := h.uc.ApplySourcePrices(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]int64{"applied": n})
}

func (h *PurchasePriceHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	var dto request.AddPurchasePriceItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.AddItem(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *PurchasePriceHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	res, err := h.uc.ListItems(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *PurchasePriceHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.DeleteItem(r.Context(), id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
