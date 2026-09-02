package handler

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc"
	taentity "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity"
	tarepo "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

func (h *TechnicalAssistanceHandler) CreateRMA(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	var dto request.CreateTechnicalAssistanceRMADTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	dto.CallCode = code
	dto.IdempotencyKey = r.Header.Get("Idempotency-Key")
	result, err := h.uc.CreateRMA(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *TechnicalAssistanceHandler) ListRMAs(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.ListRMAsByCall(r.Context(), code)
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) GetRMA(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "rmaCode")
	if !ok {
		return
	}
	result, err := h.uc.GetRMA(r.Context(), code)
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) TransitionRMA(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "rmaCode")
	if !ok {
		return
	}
	var dto request.TransitionTechnicalAssistanceRMADTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.TransitionRMA(r.Context(), code, dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) AddRMAEvidence(w http.ResponseWriter, r *http.Request) {
	rmaCode, ok := pathInt64(w, r, "rmaCode")
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
	if fileName == "." || fileName == "" || fileName != header.Filename || strings.ContainsAny(header.Filename, `/\`) {
		security.RespondError(w, http.StatusUnprocessableEntity, "nome de arquivo inválido")
		return
	}
	result, err := h.uc.AddRMAEvidence(r.Context(), rmaCode, fileName, http.DetectContentType(content), content)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *TechnicalAssistanceHandler) ListRMAEvidences(w http.ResponseWriter, r *http.Request) {
	rmaCode, ok := pathInt64(w, r, "rmaCode")
	if !ok {
		return
	}
	result, err := h.uc.ListRMAEvidences(r.Context(), rmaCode)
	h.respond(w, result, err, http.StatusOK)
}

func (h *TechnicalAssistanceHandler) DownloadRMAEvidence(w http.ResponseWriter, r *http.Request) {
	rmaCode, ok := pathInt64(w, r, "rmaCode")
	if !ok {
		return
	}
	evidenceID, err := uuid.Parse(chi.URLParam(r, "evidenceID"))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "identificador da evidência inválido")
		return
	}
	evidence, err := h.uc.GetRMAEvidence(r.Context(), rmaCode, evidenceID)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.Header().Set("Content-Type", evidence.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(evidence.SizeBytes, 10))
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": evidence.FileName}))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(evidence.Content)
}

func (h *TechnicalAssistanceHandler) respond(w http.ResponseWriter, result any, err error, status int) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, status, result)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		security.RespondError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return false
	}
	return true
}

func pathInt64(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "parâmetro "+name+" inválido")
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
