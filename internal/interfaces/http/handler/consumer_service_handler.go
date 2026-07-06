package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/consumer_service_uc"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity"
	csrepo "github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type ConsumerServiceHandler struct {
	uc *consumer_service_uc.UseCase
}

func NewConsumerServiceHandler(uc *consumer_service_uc.UseCase) *ConsumerServiceHandler {
	return &ConsumerServiceHandler{uc: uc}
}

func (h *ConsumerServiceHandler) CreateCallType(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateConsumerServiceCallTypeDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateCallType(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) ListCallTypes(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListCallTypes(r.Context(), r.URL.Query().Get("active") != "false")
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) CreateKnowledgeSource(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateConsumerServiceKnowledgeSourceDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateKnowledgeSource(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) ListKnowledgeSources(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListKnowledgeSources(r.Context(), r.URL.Query().Get("active") != "false")
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) CreateConsumer(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateConsumerDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateConsumer(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) ListConsumers(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListConsumers(r.Context(), csrepo.ConsumerFilter{
		Search:     stringQuery(r, "search"),
		State:      stringQuery(r, "state"),
		City:       stringQuery(r, "city"),
		OnlyActive: r.URL.Query().Get("active") != "false",
	})
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) GetConsumer(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.GetConsumer(r.Context(), code)
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) UpdateConsumer(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	var dto request.UpdateConsumerDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.UpdateConsumer(r.Context(), code, dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) AddConsumerPhone(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateConsumerPhoneDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := optionalPathInt64(r, "code"); ok {
		dto.ConsumerCode = code
	}
	result, err := h.uc.AddConsumerPhone(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) AddConsumerEmail(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateConsumerEmailDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := optionalPathInt64(r, "code"); ok {
		dto.ConsumerCode = code
	}
	result, err := h.uc.AddConsumerEmail(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) AddConsumerContact(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateConsumerContactDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := optionalPathInt64(r, "code"); ok {
		dto.ConsumerCode = code
	}
	result, err := h.uc.AddConsumerContact(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) CreateCustomerContact(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCustomerContactHistoryDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateCustomerContact(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) ListCustomerContacts(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListCustomerContacts(r.Context(), csrepo.CustomerContactFilter{
		CustomerCode: int64Query(r, "customer_code"),
		From:         parseHandlerDatePtr(r.URL.Query().Get("from")),
		To:           parseHandlerDatePtr(r.URL.Query().Get("to")),
		ContactType:  stringQuery(r, "contact_type"),
	})
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) CreateCall(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateConsumerServiceCallDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateCall(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) ListCalls(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListCalls(r.Context(), h.callFilter(r))
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) GetCall(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.GetCall(r.Context(), code)
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) UpdateCall(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	var dto request.UpdateConsumerServiceCallDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.UpdateCall(r.Context(), code, dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) AddCallReturn(w http.ResponseWriter, r *http.Request) {
	var dto request.AddConsumerServiceCallReturnDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := optionalPathInt64(r, "code"); ok {
		dto.CallCode = code
	}
	result, err := h.uc.AddCallReturn(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) AddCallAttachment(w http.ResponseWriter, r *http.Request) {
	var dto request.AddConsumerServiceCallAttachmentDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := optionalPathInt64(r, "code"); ok {
		dto.CallCode = code
	}
	result, err := h.uc.AddCallAttachment(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) AddChecklistItem(w http.ResponseWriter, r *http.Request) {
	var dto request.AddConsumerServiceChecklistItemDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := optionalPathInt64(r, "code"); ok {
		dto.CallCode = code
	}
	result, err := h.uc.AddChecklistItem(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) SetChecklistItemDone(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "itemCode")
	if !ok {
		return
	}
	var dto request.SetConsumerServiceChecklistItemDoneDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.SetChecklistItemDone(r.Context(), code, dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) ReportCalls(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ReportCalls(r.Context(), h.callFilter(r))
	h.respond(w, result, err, http.StatusOK)
}

func (h *ConsumerServiceHandler) callFilter(r *http.Request) csrepo.CallFilter {
	var position *entity.CallPosition
	if raw := r.URL.Query().Get("position"); raw != "" {
		v := entity.CallPosition(strings.ToUpper(raw))
		position = &v
	}
	var situation *entity.CallSituation
	if raw := r.URL.Query().Get("situation"); raw != "" {
		v := entity.CallSituation(strings.ToUpper(raw))
		situation = &v
	}
	return csrepo.CallFilter{
		CallNumber:          int64Query(r, "call_number"),
		CallTypeCode:        int64Query(r, "call_type_code"),
		ConsumerCode:        int64Query(r, "consumer_code"),
		ResponsibleUserCode: int64Query(r, "responsible_user_code"),
		DefectGroupCode:     int64Query(r, "defect_group_code"),
		DefectReasonCode:    int64Query(r, "defect_reason_code"),
		Position:            position,
		Situation:           situation,
		From:                parseHandlerDatePtr(r.URL.Query().Get("from")),
		To:                  parseHandlerDatePtr(r.URL.Query().Get("to")),
		ReturnFrom:          parseHandlerDatePtr(r.URL.Query().Get("return_from")),
		ReturnTo:            parseHandlerDatePtr(r.URL.Query().Get("return_to")),
		VisitState:          stringQuery(r, "visit_state"),
		OnlyActive:          r.URL.Query().Get("active") != "false",
	}
}

func (h *ConsumerServiceHandler) respond(w http.ResponseWriter, result any, err error, status int) {
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

func stringQuery(r *http.Request, name string) *string {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return nil
	}
	return &raw
}

func optionalPathInt64(r *http.Request, name string) (int64, bool) {
	raw := chi.URLParam(r, name)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}
