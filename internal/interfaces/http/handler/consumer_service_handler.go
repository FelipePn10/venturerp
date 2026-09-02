package handler

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/consumer_service_uc"
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
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "corpo da requisição inválido: "+err.Error())
		return
	}
	if code, ok := optionalPathInt64(r, "code"); ok {
		dto.CallCode = code
	}
	result, err := h.uc.AddCallReturn(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) AddCallAttachment(w http.ResponseWriter, r *http.Request) {
	callCode, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 11*1024*1024)
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, "arquivo inválido ou maior que 10 MiB")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, "campo multipart 'file' é obrigatório")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 10*1024*1024+1))
	if err != nil || len(content) == 0 || len(content) > 10*1024*1024 {
		security.RespondError(w, http.StatusUnprocessableEntity, "arquivo vazio, inválido ou maior que 10 MiB")
		return
	}
	fileName := filepath.Base(header.Filename)
	if fileName == "." || fileName == "" || fileName != header.Filename || strings.ContainsAny(header.Filename, `/\`) || strings.ContainsRune(header.Filename, '\x00') {
		security.RespondError(w, http.StatusUnprocessableEntity, "nome de arquivo inválido")
		return
	}
	var notes *string
	if value := strings.TrimSpace(r.FormValue("notes")); value != "" {
		notes = &value
	}
	dto := request.AddConsumerServiceCallAttachmentDTO{CallCode: callCode, FileName: fileName, ContentType: http.DetectContentType(content), Content: content, Notes: notes}
	result, err := h.uc.AddCallAttachment(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *ConsumerServiceHandler) ListCallAttachments(w http.ResponseWriter, r *http.Request) {
	callCode, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	call, err := h.uc.GetCall(r.Context(), callCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, call.Attachments)
}

func (h *ConsumerServiceHandler) DownloadCallAttachment(w http.ResponseWriter, r *http.Request) {
	callCode, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	attachmentCode, ok := pathInt64(w, r, "attachmentCode")
	if !ok {
		return
	}
	attachment, err := h.uc.GetCallAttachment(r.Context(), callCode, attachmentCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	contentType := "application/octet-stream"
	if attachment.ContentType != nil {
		contentType = *attachment.ContentType
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(attachment.FileSize, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.FileName}))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(attachment.Content)
}

func (h *ConsumerServiceHandler) DeleteCallAttachment(w http.ResponseWriter, r *http.Request) {
	callCode, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	attachmentCode, ok := pathInt64(w, r, "attachmentCode")
	if !ok {
		return
	}
	if err := h.uc.DeleteCallAttachment(r.Context(), callCode, attachmentCode); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
		security.RespondUseCaseError(w, err)
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
