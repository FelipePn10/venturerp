package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc"
	taentity "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	tarepo "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/go-chi/chi/v5"
)

type TechnicalAssistanceHandler struct {
	uc *technical_assistance_uc.UseCase
}

func NewTechnicalAssistanceHandler(uc *technical_assistance_uc.UseCase) *TechnicalAssistanceHandler {
	return &TechnicalAssistanceHandler{uc: uc}
}

func (h *TechnicalAssistanceHandler) CreateDefectGroup(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateTADefectGroupDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateDefectGroup(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *TechnicalAssistanceHandler) ListDefectGroups(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListDefectGroups(r.Context(), r.URL.Query().Get("active") != "false")
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) CreateDefectReason(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateTADefectReasonDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateDefectReason(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *TechnicalAssistanceHandler) ListDefectReasons(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListDefectReasons(r.Context(), int64Query(r, "group_code"), r.URL.Query().Get("active") != "false")
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) CreateWarrantyResponsible(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateTAWarrantyResponsibleDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateWarrantyResponsible(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *TechnicalAssistanceHandler) ListWarrantyResponsibles(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListWarrantyResponsibles(r.Context(), r.URL.Query().Get("active") != "false")
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) CreateCall(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateTechnicalAssistanceCallDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateCall(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *TechnicalAssistanceHandler) GetCall(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.GetCall(r.Context(), code)
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) ListCalls(w http.ResponseWriter, r *http.Request) {
	var status *taentity.CallStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		v := taentity.CallStatus(raw)
		status = &v
	}
	result, err := h.uc.ListCalls(r.Context(), tarepo.CallFilter{
		Status:       status,
		CustomerCode: int64Query(r, "customer_code"),
		From:         parseHandlerDatePtr(r.URL.Query().Get("from")),
		To:           parseHandlerDatePtr(r.URL.Query().Get("to")),
		OnlyActive:   r.URL.Query().Get("active") != "false",
	})
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) AddCallItem(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateTechnicalAssistanceCallItemDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := pathInt64(w, r, "code"); ok {
		dto.CallCode = code
	}
	result, err := h.uc.AddCallItem(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *TechnicalAssistanceHandler) AddReturnNote(w http.ResponseWriter, r *http.Request) {
	var dto request.AddTechnicalAssistanceReturnNoteDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := pathInt64(w, r, "code"); ok {
		dto.CallCode = code
	}
	result, err := h.uc.AddReturnNote(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *TechnicalAssistanceHandler) GenerateOrders(w http.ResponseWriter, r *http.Request) {
	var dto request.GenerateTechnicalAssistanceOrdersDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := pathInt64(w, r, "code"); ok {
		dto.CallCode = code
	}
	result, err := h.uc.GenerateOrders(r.Context(), dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateTechnicalAssistanceCallStatusDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := pathInt64(w, r, "code"); ok {
		dto.Code = code
	}
	result, err := h.uc.UpdateStatus(r.Context(), dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) Report(w http.ResponseWriter, r *http.Request) {
	var status *taentity.CallStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		v := taentity.CallStatus(raw)
		status = &v
	}
	result, err := h.uc.Report(r.Context(), tarepo.ReportFilter{
		From:         parseHandlerDatePtr(r.URL.Query().Get("from")),
		To:           parseHandlerDatePtr(r.URL.Query().Get("to")),
		CustomerCode: int64Query(r, "customer_code"),
		Status:       status,
	})
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) respond(w http.ResponseWriter, result any, err error, status int) {
	if err != nil {
		switch {
		case errors.Is(err, errorsuc.ErrUnauthorized):
			security.RespondError(w, http.StatusUnauthorized, err.Error())
		default:
			security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		}
		return
	}
	security.RespondJSON(w, status, result)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid body")
		return false
	}
	return true
}

func pathInt64(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return value, true
}

func int64Query(r *http.Request, name string) *int64 {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &value
}

func parseHandlerDatePtr(s string) *time.Time {
	return datetime.ParseDatePtr(&s)
}
