package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

// Crédito, situação na Receita, ciclo de vida e painel do cliente (VCLI0500).

func dateValue(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("02/01/2006")
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	v, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || v <= 0 {
		jsonError(w, http.StatusUnprocessableEntity, name+" inválido")
		return 0, false
	}
	return v, true
}

func noContent(w http.ResponseWriter, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreditSuggestion: GET /api/customers/credit-suggestion?document_type=CNPJ&document=...
// (cliente ainda não cadastrado) ou GET /api/customers/{code}/credit-suggestion.
func (h *CustomerHandler) CreditSuggestion(w http.ResponseWriter, r *http.Request) {
	var code *int64
	if chi.URLParam(r, "code") != "" {
		c, ok := customerPathCode(w, r)
		if !ok {
			return
		}
		code = &c
	}
	q := r.URL.Query()
	result, err := h.uc.SuggestCredit(r.Context(), code, q.Get("document_type"), q.Get("document"))
	customerRespond(w, result, err)
}

func (h *CustomerHandler) SetCreditLimit(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	var dto request.SetCreditLimitDTO
	if !ok || !customerDecode(w, r, &dto) {
		return
	}
	result, err := h.uc.SetCreditLimit(r.Context(), code, dto)
	customerRespond(w, result, err)
}

func (h *CustomerHandler) ListCreditReviews(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	if !ok {
		return
	}
	result, err := h.uc.ListCreditReviews(r.Context(), code)
	customerRespond(w, result, err)
}

func (h *CustomerHandler) GetCreditPolicy(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.GetCreditPolicy(r.Context())
	customerRespond(w, result, err)
}

func (h *CustomerHandler) SaveCreditPolicy(w http.ResponseWriter, r *http.Request) {
	var dto request.CreditPolicyDTO
	if !customerDecode(w, r, &dto) {
		return
	}
	result, err := h.uc.SaveCreditPolicy(r.Context(), dto)
	customerRespond(w, result, err)
}

func (h *CustomerHandler) RefreshRegistration(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	if !ok {
		return
	}
	result, err := h.uc.RefreshRegistration(r.Context(), code)
	customerRespond(w, result, err)
}

func (h *CustomerHandler) CustomerUsage(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	if !ok {
		return
	}
	result, err := h.uc.CustomerUsageReport(r.Context(), code)
	customerRespond(w, result, err)
}

func (h *CustomerHandler) CustomerOverview(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	if !ok {
		return
	}
	result, err := h.uc.CustomerOverview(r.Context(), code)
	customerRespond(w, result, err)
}

func (h *CustomerHandler) InactivateCustomer(w http.ResponseWriter, r *http.Request) {
	if code, ok := customerPathCode(w, r); ok {
		noContent(w, h.uc.InactivateCustomer(r.Context(), code))
	}
}

func (h *CustomerHandler) ReactivateCustomer(w http.ResponseWriter, r *http.Request) {
	if code, ok := customerPathCode(w, r); ok {
		noContent(w, h.uc.ReactivateCustomer(r.Context(), code))
	}
}

func (h *CustomerHandler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	if code, ok := customerPathCode(w, r); ok {
		noContent(w, h.uc.DeleteCustomer(r.Context(), code))
	}
}

func (h *CustomerHandler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	var dto request.AddAddressDTO
	if !ok || !customerDecode(w, r, &dto) {
		return
	}
	result, err := h.uc.UpdateAddress(r.Context(), code, id, dto)
	customerRespond(w, result, err)
}

func (h *CustomerHandler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	if !ok {
		return
	}
	if id, ok := pathID(w, r, "id"); ok {
		noContent(w, h.uc.DeleteAddress(r.Context(), code, id))
	}
}

func (h *CustomerHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "id")
	var dto request.AddContactDTO
	if !ok || !customerDecode(w, r, &dto) {
		return
	}
	result, err := h.uc.UpdateContact(r.Context(), code, id, dto)
	customerRespond(w, result, err)
}

func (h *CustomerHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	if !ok {
		return
	}
	if id, ok := pathID(w, r, "id"); ok {
		noContent(w, h.uc.DeleteContact(r.Context(), code, id))
	}
}

// DeleteSupport atende DELETE /api/customers/support/{resource}/{code}.
func (h *CustomerHandler) DeleteSupport(resource string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if code, ok := customerPathCode(w, r); ok {
			noContent(w, h.uc.DeleteSupport(r.Context(), resource, code))
		}
	}
}

// SetSupportActive atende PATCH /api/customers/support/{resource}/{code}/active {is_active}.
func (h *CustomerHandler) SetSupportActive(resource string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code, ok := customerPathCode(w, r)
		var dto struct {
			IsActive *bool `json:"is_active"`
		}
		if !ok || !customerDecode(w, r, &dto) {
			return
		}
		if dto.IsActive == nil {
			customerRespond(w, nil, errorsuc.NewValidationError("informe is_active"))
			return
		}
		noContent(w, h.uc.SetSupportActive(r.Context(), resource, code, *dto.IsActive))
	}
}

func (h *CustomerHandler) UpdateTaxType(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	var dto request.CreateTaxTypeDTO
	if !ok || !customerDecode(w, r, &dto) {
		return
	}
	result, err := h.uc.UpdateTaxType(r.Context(), code, dto)
	customerRespond(w, result, err)
}

// UpdateInvoiceTypeByCode: PUT /invoice-types/{code} (a tela manda o código na
// rota; PUT /invoice-types com o código no corpo continua valendo).
func (h *CustomerHandler) UpdateInvoiceTypeByCode(w http.ResponseWriter, r *http.Request) {
	code, ok := customerPathCode(w, r)
	var dto request.UpdateInvoiceTypeDTO
	if !ok || !customerDecode(w, r, &dto) {
		return
	}
	dto.Code = code
	result, err := h.uc.UpdateInvoiceType(r.Context(), dto)
	customerRespond(w, result, err)
}
