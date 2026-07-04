package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/procurement_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type ProcurementHandler struct {
	uc *procurement_uc.UseCase
}

func NewProcurementHandler(uc *procurement_uc.UseCase) *ProcurementHandler {
	return &ProcurementHandler{uc: uc}
}

func (h *ProcurementHandler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateProcurementRecordDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateRecord(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) GetRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetRecord(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListRecords(r.Context(), r.URL.Query().Get("type"), r.URL.Query().Get("status"))
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.UpdateProcurementRecordStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.UpdateStatus(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) DisposeInspection(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.DisposeReceivingInspectionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.DisposeInspection(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) CreateSupplierScorecard(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSupplierScorecardDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateSupplierScorecard(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) ListSupplierScorecards(w http.ResponseWriter, r *http.Request) {
	supplierCode, err := strconv.ParseInt(chi.URLParam(r, "supplierCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid supplier code")
		return
	}
	result, err := h.uc.ListSupplierScorecards(r.Context(), supplierCode)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) CreateReceivingInspectionRoute(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateReceivingInspectionRouteDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateReceivingInspectionRoute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) GetReceivingInspectionRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetReceivingInspectionRoute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) GenerateReceivingInspectionOrder(w http.ResponseWriter, r *http.Request) {
	var dto request.GenerateReceivingInspectionOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.GenerateReceivingInspectionOrder(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) ListReceivingInspectionOrders(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListReceivingInspectionOrders(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) RecordReceivingInspectionResult(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.RecordReceivingInspectionResultDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.RecordReceivingInspectionResult(r.Context(), orderID, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) AnalyzeReceivingInspectionOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.AnalyzeReceivingInspectionOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.AnalyzeReceivingInspectionOrder(r.Context(), orderID, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ComputeSupplierScorecard(w http.ResponseWriter, r *http.Request) {
	var dto request.ComputeSupplierScorecardDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.ComputeSupplierScorecard(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) CreateApprovalLimit(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateApprovalLimitDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateApprovalLimit(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) ListApprovalLimits(w http.ResponseWriter, r *http.Request) {
	enterpriseCode, _ := strconv.ParseInt(r.URL.Query().Get("enterprise_code"), 10, 64)
	result, err := h.uc.ListApprovalLimits(r.Context(), enterpriseCode)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) CreateSupplierContract(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSupplierContractDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateSupplierContract(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) GetSupplierContract(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetSupplierContract(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ListSupplierContracts(w http.ResponseWriter, r *http.Request) {
	supplierCode, _ := strconv.ParseInt(r.URL.Query().Get("supplier_code"), 10, 64)
	result, err := h.uc.ListSupplierContracts(r.Context(), supplierCode, r.URL.Query().Get("status"))
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) UpdateSupplierContractStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.UpdateSupplierContractStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.UpdateSupplierContractStatus(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ConsumeSupplierContract(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.ConsumeSupplierContractDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.ConsumeSupplierContract(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---- Receiving notice + divergences (FAVR) ----

func (h *ProcurementHandler) CreateReceivingNotice(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateReceivingNoticeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateReceivingNotice(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) GetReceivingNotice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetReceivingNotice(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ListReceivingNotices(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListReceivingNotices(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) UpdateReceivingNoticeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.UpdateReceivingNoticeStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.UpdateReceivingNoticeStatus(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) CreateReceivingDivergence(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateReceivingDivergenceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateReceivingDivergence(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) ListReceivingDivergences(w http.ResponseWriter, r *http.Request) {
	var supplierCode *int64
	if v := r.URL.Query().Get("supplier_code"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierCode = &n
		}
	}
	result, err := h.uc.ListReceivingDivergences(r.Context(), supplierCode, r.URL.Query().Get("resolution"))
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ResolveReceivingDivergence(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.ResolveReceivingDivergenceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.ResolveReceivingDivergence(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---- Supplier EDI (FEDS) ----

func (h *ProcurementHandler) CreateEDIMessage(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateEDIMessageDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateEDIMessage(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) GetEDIMessage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetEDIMessage(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ListEDIMessages(w http.ResponseWriter, r *http.Request) {
	var supplierCode *int64
	if v := r.URL.Query().Get("supplier_code"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierCode = &n
		}
	}
	result, err := h.uc.ListEDIMessages(r.Context(), supplierCode)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---- Import landed cost (FREC0203 / FIMP) ----

func (h *ProcurementHandler) CreateImportProcess(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateImportProcessDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateImportProcess(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) GetImportProcess(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.GetImportProcess(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ListImportProcesses(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListImportProcesses(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) RecomputeImportProcess(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.uc.RecomputeImportProcess(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) UpdateImportProcessStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var dto request.UpdateImportProcessStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.UpdateImportProcessStatus(r.Context(), id, dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---- Procurement parameters (FUTL0125) ----

func (h *ProcurementHandler) UpsertParameter(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertProcurementParameterDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.UpsertParameter(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ListParameters(w http.ResponseWriter, r *http.Request) {
	enterpriseCode, _ := strconv.ParseInt(r.URL.Query().Get("enterprise_code"), 10, 64)
	result, err := h.uc.ListParameters(r.Context(), enterpriseCode, r.URL.Query().Get("domain"))
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---- Supplier homologation (FAVF0203) ----

func (h *ProcurementHandler) CreateSupplierHomologation(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSupplierHomologationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateSupplierHomologation(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *ProcurementHandler) GenerateItemSuppliers(w http.ResponseWriter, r *http.Request) {
	supplierCode, err := strconv.ParseInt(chi.URLParam(r, "supplierCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid supplier code")
		return
	}
	created, err := h.uc.GenerateItemSuppliers(r.Context(), supplierCode)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]int{"created": created})
}

func (h *ProcurementHandler) ListSupplierHomologations(w http.ResponseWriter, r *http.Request) {
	supplierCode, err := strconv.ParseInt(chi.URLParam(r, "supplierCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid supplier code")
		return
	}
	result, err := h.uc.ListSupplierHomologations(r.Context(), supplierCode)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *ProcurementHandler) ListPurchaseMovementHistory(w http.ResponseWriter, r *http.Request) {
	var supplierCode, itemCode *int64
	if v := r.URL.Query().Get("supplier_code"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			supplierCode = &n
		}
	}
	if v := r.URL.Query().Get("item_code"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			itemCode = &n
		}
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	result, err := h.uc.ListPurchaseMovementHistory(r.Context(), supplierCode, itemCode, limit)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
