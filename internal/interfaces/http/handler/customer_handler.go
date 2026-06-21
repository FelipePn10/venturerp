package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/customer_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export"
	"github.com/go-chi/chi/v5"
)

type CustomerHandler struct {
	uc *customer_uc.CustomerUseCase
}

func NewCustomerHandler(uc *customer_uc.CustomerUseCase) *CustomerHandler {
	return &CustomerHandler{uc: uc}
}

// ─── Regions ─────────────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateRegion(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRegionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateRegion(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) UpdateRegion(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateRegionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateRegion(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) GetRegion(w http.ResponseWriter, r *http.Request) {
	code, err := parseCustomerCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.uc.GetRegion(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListRegions(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListRegions(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Market Segments ──────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateMarketSegment(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateMarketSegmentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateMarketSegment(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListMarketSegments(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListMarketSegments(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Customer Contact Types ───────────────────────────────────────────────────

func (h *CustomerHandler) CreateContactType(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateContactTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateContactType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListContactTypes(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListContactTypes(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Customer Types ───────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateCustomerType(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCustomerTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateCustomerType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListCustomerTypes(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListCustomerTypes(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Carriers ─────────────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateCarrier(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCarrierDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateCarrier(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListCarriers(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListCarriers(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Carrier Groups ───────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateCarrierGroup(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCarrierGroupDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateCarrierGroup(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListCarrierGroups(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListCarrierGroups(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) AddCarrierToGroup(w http.ResponseWriter, r *http.Request) {
	var dto request.CarrierGroupMemberDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if err := h.uc.AddCarrierToGroup(r.Context(), dto); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Payment Conditions ───────────────────────────────────────────────────────

func (h *CustomerHandler) CreatePaymentCondition(w http.ResponseWriter, r *http.Request) {
	var dto request.CreatePaymentConditionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreatePaymentCondition(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListPaymentConditions(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListPaymentConditions(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) AddInstallment(w http.ResponseWriter, r *http.Request) {
	var dto request.AddInstallmentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.AddInstallment(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

// ─── Sales Tables ─────────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateSalesTable(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesTableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateSalesTable(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListSalesTables(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListSalesTables(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Invoice Types ────────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateInvoiceType(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateInvoiceTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateInvoiceType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) UpdateInvoiceType(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateInvoiceTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateInvoiceType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListInvoiceTypes(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListInvoiceTypes(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Tax Types ────────────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateTaxType(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateTaxTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateTaxType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListTaxTypes(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListTaxTypes(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Customers ────────────────────────────────────────────────────────────────

func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCustomerDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateCustomer(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	code, err := parseCustomerCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.uc.GetCustomer(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListCustomers(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// When ?format=xlsx|pdf|csv is present, stream the same rows as a file
	// instead of JSON. Any other list endpoint can opt in the same way.
	if done, _ := export.WriteSlice(w, r, "Clientes", "clientes", result); done {
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListEstablishments(w http.ResponseWriter, r *http.Request) {
	code, err := parseCustomerCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.uc.ListEstablishments(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) BlockCustomer(w http.ResponseWriter, r *http.Request) {
	var dto request.BlockCustomerDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if err := h.uc.BlockCustomer(r.Context(), dto); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CustomerHandler) UnblockCustomer(w http.ResponseWriter, r *http.Request) {
	code, err := parseCustomerCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := h.uc.UnblockCustomer(r.Context(), code); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Customer Addresses ───────────────────────────────────────────────────────

func (h *CustomerHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	var dto request.AddAddressDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.AddAddress(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

// ─── Customer Contacts ────────────────────────────────────────────────────────

func (h *CustomerHandler) AddContact(w http.ResponseWriter, r *http.Request) {
	var dto request.AddContactDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.AddContact(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

// ─── Sales Table Prices ───────────────────────────────────────────────────────

func (h *CustomerHandler) CreateSalesTablePrice(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesTablePriceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateSalesTablePrice(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) UpdateSalesTablePrice(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateSalesTablePriceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateSalesTablePrice(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) GetSalesTablePrice(w http.ResponseWriter, r *http.Request) {
	salesTableID, err := strconv.ParseInt(chi.URLParam(r, "tableID"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid table id")
		return
	}
	itemCode := chi.URLParam(r, "itemCode")
	result, err := h.uc.GetSalesTablePrice(r.Context(), salesTableID, itemCode)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListSalesTablePrices(w http.ResponseWriter, r *http.Request) {
	salesTableID, err := strconv.ParseInt(chi.URLParam(r, "tableID"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid table id")
		return
	}
	result, err := h.uc.ListSalesTablePrices(r.Context(), salesTableID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) DeleteSalesTablePrice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.uc.DeleteSalesTablePrice(r.Context(), id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func parseCustomerCode(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
}
