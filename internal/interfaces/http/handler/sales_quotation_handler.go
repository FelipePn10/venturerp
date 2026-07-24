package handler

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_quotation_uc"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quoterepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/go-chi/chi/v5"
)

type SalesQuotationHandler struct {
	uc        *sales_quotation_uc.UseCase
	convertUC *sales_quotation_uc.ConvertUseCase
}

func NewSalesQuotationHandler(uc *sales_quotation_uc.UseCase, convertUC *sales_quotation_uc.ConvertUseCase) *SalesQuotationHandler {
	return &SalesQuotationHandler{uc: uc, convertUC: convertUC}
}

func (h *SalesQuotationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesQuotationHandler) Update(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.UpdateSalesQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.uc.Update(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) Get(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.Get(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := parseQuotationFilter(r)
	result, err := h.uc.List(r.Context(), filter)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.CancelSalesQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.uc.Cancel(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) Attend(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.AttendSalesQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.uc.Attend(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) Uncancel(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.UncancelSalesQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.uc.Uncancel(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.ChangeSalesQuotationStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.uc.ChangeStatus(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) ChangeRelease(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.ChangeSalesQuotationReleaseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.uc.ChangeRelease(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *SalesQuotationHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	rows, err := h.uc.ListEvents(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, rows)
}

func (h *SalesQuotationHandler) Report(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.Report(r.Context(), parseQuotationFilter(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) Convert(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.ConvertSalesQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.convertUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesQuotationHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesQuotationItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateItem(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesQuotationHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "itemCode")
	if !ok {
		return
	}
	var dto request.UpdateSalesQuotationItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.uc.UpdateItem(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.ListItems(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) CancelItem(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "itemCode")
	if !ok {
		return
	}
	var dto request.CancelSalesQuotationItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.uc.CancelItem(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) GetParameters(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.GetParameters(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
func (h *SalesQuotationHandler) SaveParameters(w http.ResponseWriter, r *http.Request) {
	var dto request.SaveSalesQuotationParametersDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.SaveParameters(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
func (h *SalesQuotationHandler) SaveCommissionPattern(w http.ResponseWriter, r *http.Request) {
	var dto request.SaveCommissionPatternDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.SaveCommissionPattern(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}
func (h *SalesQuotationHandler) ListCommissionPatterns(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListCommissionPatterns(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
func (h *SalesQuotationHandler) SaveCancellationReason(w http.ResponseWriter, r *http.Request) {
	var dto request.SaveCancellationReasonDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.SaveCancellationReason(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}
func (h *SalesQuotationHandler) ListCancellationReasons(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListCancellationReasons(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
func (h *SalesQuotationHandler) GenerateDAV(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.GenerateDAV(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
func (h *SalesQuotationHandler) CreateAttachment(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, quoteentity.MaxAttachmentSize+(1<<20))
	file, header, err := r.FormFile("file")
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "multipart file field is required")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, quoteentity.MaxAttachmentSize+1))
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto := request.CreateSalesQuotationAttachmentDTO{SalesQuotationCode: code, FileName: header.Filename, ContentType: header.Header.Get("Content-Type"), FileSize: int64(len(content)), Content: content}
	if dto.FileSize > quoteentity.MaxAttachmentSize {
		security.RespondError(w, http.StatusRequestEntityTooLarge, "attachment cannot exceed 10 MB")
		return
	}
	result, err := h.uc.CreateAttachment(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesQuotationHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	id, ok := parseQuotationCode(w, r, "attachmentID")
	if !ok {
		return
	}
	a, err := h.uc.GetAttachment(r.Context(), code, id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.Header().Set("Content-Type", a.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": a.FileName}))
	w.Header().Set("Content-Length", strconv.FormatInt(a.FileSize, 10))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(a.Content)
}
func (h *SalesQuotationHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.ListAttachments(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
func (h *SalesQuotationHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	id, ok := parseQuotationCode(w, r, "attachmentID")
	if !ok {
		return
	}
	if err := h.uc.DeleteAttachment(r.Context(), code, id); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseQuotationCode(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	code, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return 0, false
	}
	return code, true
}

func parseQuotationFilter(r *http.Request) quoterepo.SalesQuotationFilter {
	q := r.URL.Query()
	var filter quoterepo.SalesQuotationFilter
	if raw := q.Get("quotation_number"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.QuotationNumber = &v
		}
	}
	if raw := q.Get("customer_code"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.CustomerCode = &v
		}
	}
	if raw := q.Get("status"); raw != "" {
		status := quoteentity.SalesQuotationStatus(raw)
		filter.Status = &status
	}
	if raw := q.Get("sales_division_code"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.SalesDivisionCode = &v
		}
	}
	if raw := q.Get("quotation_type"); raw != "" {
		v := quoteentity.SalesQuotationType(raw)
		filter.QuotationType = &v
	}
	if from := q.Get("from"); from != "" {
		t := datetime.ParseDatePtr(&from)
		filter.From = t
	}
	if to := q.Get("to"); to != "" {
		t := datetime.ParseDatePtr(&to)
		filter.To = t
	}
	if purchaseOrder := q.Get("purchase_order_number"); purchaseOrder != "" {
		filter.PurchaseOrderNumber = &purchaseOrder
	}
	if freightType := q.Get("freight_type"); freightType != "" {
		filter.FreightType = &freightType
	}
	if raw := q.Get("limit"); raw != "" {
		filter.Limit, _ = strconv.Atoi(raw)
	}
	if raw := q.Get("offset"); raw != "" {
		filter.Offset, _ = strconv.Atoi(raw)
	}
	return filter
}
