package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_params_uc"
	"github.com/go-chi/chi/v5"
)

type FiscalParamsHandler struct {
	legalDeviceUC *fiscal_params_uc.LegalDeviceUseCase
	cfopUC        *fiscal_params_uc.CFOPUseCase
	taxParamUC    *fiscal_params_uc.TaxParamUseCase
}

func NewFiscalParamsHandler(
	legalDeviceUC *fiscal_params_uc.LegalDeviceUseCase,
	cfopUC *fiscal_params_uc.CFOPUseCase,
	taxParamUC *fiscal_params_uc.TaxParamUseCase,
) *FiscalParamsHandler {
	return &FiscalParamsHandler{
		legalDeviceUC: legalDeviceUC,
		cfopUC:        cfopUC,
		taxParamUC:    taxParamUC,
	}
}

// ─── Legal Devices ─────────────────────────────────────────────────────────────

func (h *FiscalParamsHandler) CreateLegalDevice(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateLegalDeviceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.legalDeviceUC.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *FiscalParamsHandler) UpdateLegalDevice(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateLegalDeviceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.legalDeviceUC.Update(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) GetLegalDevice(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.legalDeviceUC.GetByCode(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) ListLegalDevices(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.legalDeviceUC.List(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) ListLegalDevicesByType(w http.ResponseWriter, r *http.Request) {
	deviceType := chi.URLParam(r, "type")
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.legalDeviceUC.ListByType(r.Context(), deviceType, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── CFOP ─────────────────────────────────────────────────────────────────────

func (h *FiscalParamsHandler) CreateCFOP(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCFOPDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.cfopUC.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *FiscalParamsHandler) UpdateCFOP(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateCFOPDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.cfopUC.Update(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) GetCFOP(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 32)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.cfopUC.GetByCode(r.Context(), int32(code))
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) ListCFOPs(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.cfopUC.List(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) ListCFOPsByDirection(w http.ResponseWriter, r *http.Request) {
	direction := chi.URLParam(r, "direction")
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.cfopUC.ListByDirection(r.Context(), direction, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Tax Params ───────────────────────────────────────────────────────────────

func (h *FiscalParamsHandler) CreateTaxParam(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateTaxParamDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.taxParamUC.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *FiscalParamsHandler) UpdateTaxParam(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateTaxParamDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.taxParamUC.Update(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) GetTaxParam(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.taxParamUC.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) ListTaxParams(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.taxParamUC.List(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) ListTaxParamsByUF(w http.ResponseWriter, r *http.Request) {
	uf := chi.URLParam(r, "uf")
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.taxParamUC.ListByUF(r.Context(), uf, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) ListTaxParamsByItem(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.taxParamUC.ListByItem(r.Context(), itemCode, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *FiscalParamsHandler) ListTaxParamsByNCM(w http.ResponseWriter, r *http.Request) {
	ncmCode := chi.URLParam(r, "ncmCode")
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.taxParamUC.ListByNCM(r.Context(), ncmCode, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
