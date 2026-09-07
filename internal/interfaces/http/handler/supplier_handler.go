package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type SupplierHandler struct {
	uc *supplier_uc.SupplierUseCase
}

func NewSupplierHandler(uc *supplier_uc.SupplierUseCase) *SupplierHandler {
	return &SupplierHandler{uc: uc}
}

// ─── Supplier Types ─────────────────────────────────────────────────────────

func (h *SupplierHandler) CreateSupplierType(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSupplierTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.CreateSupplierType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) UpdateSupplierType(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateSupplierTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.UpdateSupplierType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *SupplierHandler) ListSupplierTypes(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListSupplierTypes(r.Context(), onlyActiveParam(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// ─── Supplier Contact Types ───────────────────────────────────────────────────

func (h *SupplierHandler) CreateContactType(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSupplierContactTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.CreateContactType(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) ListContactTypes(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListContactTypes(r.Context(), onlyActiveParam(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// ─── Suppliers ────────────────────────────────────────────────────────────────

func (h *SupplierHandler) CreateSupplier(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSupplierDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.CreateSupplier(r.Context(), dto)
	if err != nil {
		// Duplicate-document conflict surfaces as 409.
		if strings.Contains(err.Error(), "já existe um fornecedor") {
			jsonError(w, http.StatusConflict, err.Error())
			return
		}
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) UpdateSupplier(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateSupplierDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.UpdateSupplier(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *SupplierHandler) GetSupplier(w http.ResponseWriter, r *http.Request) {
	code, err := parseSupplierCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código inválido")
		return
	}
	res, err := h.uc.GetSupplier(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *SupplierHandler) ListSuppliers(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListSuppliers(r.Context(), onlyActiveParam(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	if done, _ := export.WriteSlice(w, r, "Fornecedores", "fornecedores", res); done {
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *SupplierHandler) ListEstablishments(w http.ResponseWriter, r *http.Request) {
	code, err := parseSupplierCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código inválido")
		return
	}
	res, err := h.uc.ListEstablishments(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *SupplierHandler) BlockSupplier(w http.ResponseWriter, r *http.Request) {
	code, err := parseSupplierCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código de fornecedor inválido")
		return
	}
	var dto request.BlockSupplierDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "payload inválido: "+err.Error())
		return
	}
	dto.Code = code
	if strings.TrimSpace(dto.Reason) == "" {
		jsonError(w, http.StatusBadRequest, "motivo do bloqueio é obrigatório")
		return
	}
	if err := h.uc.BlockSupplier(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SupplierHandler) UnblockSupplier(w http.ResponseWriter, r *http.Request) {
	code, err := parseSupplierCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código de fornecedor inválido")
		return
	}
	if err := h.uc.UnblockSupplier(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SupplierHandler) DeleteSupplier(w http.ResponseWriter, r *http.Request) {
	code, err := parseSupplierCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código inválido")
		return
	}
	if err := h.uc.DeleteSupplier(r.Context(), code); err != nil {
		jsonError(w, http.StatusConflict, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Folders ──────────────────────────────────────────────────────────────────

func (h *SupplierHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	var dto request.AddSupplierAddressDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.AddAddress(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) AddPhone(w http.ResponseWriter, r *http.Request) {
	var dto request.AddSupplierPhoneDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.AddPhone(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) AddEmail(w http.ResponseWriter, r *http.Request) {
	var dto request.AddSupplierEmailDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.AddEmail(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) AddDueDate(w http.ResponseWriter, r *http.Request) {
	var dto request.AddSupplierDueDateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.AddDueDate(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) AddContact(w http.ResponseWriter, r *http.Request) {
	var dto request.AddSupplierContactDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.AddContact(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) AddContactPhone(w http.ResponseWriter, r *http.Request) {
	var dto request.AddSupplierContactPhoneDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.AddContactPhone(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) AddContactEmail(w http.ResponseWriter, r *http.Request) {
	var dto request.AddSupplierContactEmailDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.AddContactEmail(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

// ─── Enterprise links ──────────────────────────────────────────────────────

func (h *SupplierHandler) AddEnterprise(w http.ResponseWriter, r *http.Request) {
	var dto request.AddSupplierEnterpriseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.AddEnterprise(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *SupplierHandler) UpdateEnterprise(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateSupplierEnterpriseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.UpdateEnterprise(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *SupplierHandler) ListEnterprises(w http.ResponseWriter, r *http.Request) {
	code, err := parseSupplierCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código inválido")
		return
	}
	res, err := h.uc.ListEnterprises(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// GetPurchasingDefaults resolves the supplier-derived defaults (payment
// condition, freight, financial account, default invoice type, ICMS contributor)
// consumed by the Purchase Order and Fiscal flows. The enterprise comes only
// from the authenticated tenant.
func (h *SupplierHandler) GetPurchasingDefaults(w http.ResponseWriter, r *http.Request) {
	code, err := parseSupplierCode(r)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código inválido")
		return
	}
	res, err := h.uc.GetPurchasingDefaults(r.Context(), code, 0)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// ─── Parameters ─────────────────────────────────────────────────────────────

func (h *SupplierHandler) GetParameters(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.GetParameters(r.Context(), 0)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *SupplierHandler) UpsertParameters(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertSupplierParametersDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	res, err := h.uc.UpsertParameters(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func parseSupplierCode(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
}

func onlyActiveParam(r *http.Request) bool {
	return r.URL.Query().Get("only_active") != "false"
}
