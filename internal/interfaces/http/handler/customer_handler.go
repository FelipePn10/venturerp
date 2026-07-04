package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/customer_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
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

func (h *CustomerHandler) GetSalesTable(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "tableCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid table code")
		return
	}
	result, err := h.uc.GetSalesTable(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) UpdateSalesTable(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateSalesTableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if dto.Code == 0 {
		code, err := strconv.ParseInt(chi.URLParam(r, "tableCode"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid table code")
			return
		}
		dto.Code = code
	}
	result, err := h.uc.UpdateSalesTable(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Sales Price Policies ────────────────────────────────────────────────────

func (h *CustomerHandler) CreateSalesPricePolicy(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesPricePolicyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateSalesPricePolicy(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) UpdateSalesPricePolicy(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateSalesPricePolicyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if dto.Code == 0 {
		code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid code")
			return
		}
		dto.Code = code
	}
	result, err := h.uc.UpdateSalesPricePolicy(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) GetSalesPricePolicy(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.uc.GetSalesPricePolicy(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListSalesPricePolicies(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListSalesPricePolicies(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Commercial Policies ─────────────────────────────────────────────────────

func (h *CustomerHandler) CreateCommercialPolicy(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCommercialPolicyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateCommercialPolicy(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) UpdateCommercialPolicy(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateCommercialPolicyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if dto.Code == 0 {
		code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid code")
			return
		}
		dto.Code = code
	}
	result, err := h.uc.UpdateCommercialPolicy(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) GetCommercialPolicy(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.uc.GetCommercialPolicy(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListCommercialPolicies(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	kind := r.URL.Query().Get("kind")
	result, err := h.uc.ListCommercialPolicies(r.Context(), onlyActive, kind)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if done, _ := export.WriteSlice(w, r, "Politicas Comerciais", "politicas-comerciais", result); done {
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) AddCommercialPolicySpecificItem(w http.ResponseWriter, r *http.Request) {
	var dto request.CommercialPolicySpecificItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if dto.PolicyCode == 0 {
		code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid code")
			return
		}
		dto.PolicyCode = code
	}
	result, err := h.uc.AddCommercialPolicySpecificItem(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) AddCommercialPolicyLine(w http.ResponseWriter, r *http.Request) {
	var dto request.CommercialPolicyLineDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if dto.PolicyCode == 0 {
		code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid code")
			return
		}
		dto.PolicyCode = code
	}
	result, err := h.uc.AddCommercialPolicyLine(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *CustomerHandler) ListCommercialPolicyLines(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.uc.ListCommercialPolicyLines(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListCommercialPolicySpecificItems(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.uc.ListCommercialPolicySpecificItems(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) EvaluateCommercialPolicies(w http.ResponseWriter, r *http.Request) {
	var dto request.EvaluateCommercialPoliciesDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.EvaluateCommercialPolicies(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
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

func (h *CustomerHandler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	code, err := parseCustomerCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.CreateCustomerDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateCustomer(r.Context(), code, dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
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
	if dto.SalesTableCode == 0 {
		code, err := strconv.ParseInt(chi.URLParam(r, "tableCode"), 10, 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid table code")
			return
		}
		dto.SalesTableCode = code
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
	salesTableCode, err := strconv.ParseInt(chi.URLParam(r, "tableCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid table code")
		return
	}
	itemCode := chi.URLParam(r, "itemCode")
	result, err := h.uc.GetSalesTablePriceByCode(r.Context(), salesTableCode, itemCode)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListSalesTablePrices(w http.ResponseWriter, r *http.Request) {
	salesTableCode, err := strconv.ParseInt(chi.URLParam(r, "tableCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid table code")
		return
	}
	result, err := h.uc.ListSalesTablePricesByCode(r.Context(), salesTableCode)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) GenerateSalesTablePrices(w http.ResponseWriter, r *http.Request) {
	var dto request.GenerateSalesTablePricesDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.GenerateSalesTablePrices(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) ListSalesTablePriceHistory(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "tableCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid table code")
		return
	}
	var itemCode *string
	if raw := r.URL.Query().Get("item_code"); raw != "" {
		itemCode = &raw
	}
	result, err := h.uc.ListSalesTablePriceHistory(r.Context(), code, itemCode)
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

func (h *CustomerHandler) PriceSalesItem(w http.ResponseWriter, r *http.Request) {
	var dto request.PriceSalesItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.PriceSalesItem(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *CustomerHandler) FormSalesPrice(w http.ResponseWriter, r *http.Request) {
	var dto request.FormSalesPriceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.FormSalesPrice(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func parseCustomerCode(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
}
