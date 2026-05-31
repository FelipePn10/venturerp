package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_params_uc"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/go-chi/chi/v5"
)

type ICMSApuracaoHandler struct {
	dapiUC         *fiscal_params_uc.DAPITransferReasonUseCase
	apuracaoAdjUC  *fiscal_params_uc.ICMSApuracaoAdjCodeUseCase
	adjCodeUC      *fiscal_params_uc.ICMSAdjustmentCodeUseCase
	apuracaoLineUC *fiscal_params_uc.ICMSApuracaoLineUseCase
	summaryUC      *fiscal_params_uc.ICMSSummaryEntryUseCase
	simplesUC      *fiscal_params_uc.SimplesNacionalUseCase
}

func NewICMSApuracaoHandler(
	dapiUC *fiscal_params_uc.DAPITransferReasonUseCase,
	apuracaoAdjUC *fiscal_params_uc.ICMSApuracaoAdjCodeUseCase,
	adjCodeUC *fiscal_params_uc.ICMSAdjustmentCodeUseCase,
	apuracaoLineUC *fiscal_params_uc.ICMSApuracaoLineUseCase,
	summaryUC *fiscal_params_uc.ICMSSummaryEntryUseCase,
	simplesUC *fiscal_params_uc.SimplesNacionalUseCase,
) *ICMSApuracaoHandler {
	return &ICMSApuracaoHandler{
		dapiUC:         dapiUC,
		apuracaoAdjUC:  apuracaoAdjUC,
		adjCodeUC:      adjCodeUC,
		apuracaoLineUC: apuracaoLineUC,
		summaryUC:      summaryUC,
		simplesUC:      simplesUC,
	}
}

// ─── DAPI Transfer Reasons ────────────────────────────────────────────────────

func (h *ICMSApuracaoHandler) CreateDAPIReason(w http.ResponseWriter, r *http.Request) {
	var d fiscalEntity.DAPITransferReason
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.dapiUC.Create(r.Context(), &d)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, toDAPIResponse(result))
}

func (h *ICMSApuracaoHandler) UpdateDAPIReason(w http.ResponseWriter, r *http.Request) {
	var d fiscalEntity.DAPITransferReason
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.dapiUC.Update(r.Context(), &d)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toDAPIResponse(result))
}

func (h *ICMSApuracaoHandler) GetDAPIReason(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.dapiUC.GetByCode(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toDAPIResponse(result))
}

func (h *ICMSApuracaoHandler) ListDAPIReasons(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.dapiUC.List(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]*response.DAPITransferReasonResponse, 0, len(result))
	for _, d := range result {
		out = append(out, toDAPIResponse(d))
	}
	jsonResponse(w, http.StatusOK, out)
}

func toDAPIResponse(d *fiscalEntity.DAPITransferReason) *response.DAPITransferReasonResponse {
	return &response.DAPITransferReasonResponse{
		ID: d.ID, Code: d.Code, Reason: d.Reason, Destination: d.Destination,
		ValidFrom: d.ValidFrom, ValidTo: d.ValidTo, IsActive: d.IsActive, CreatedAt: d.CreatedAt,
	}
}

// ─── ICMS Apuração Adjustment Codes ──────────────────────────────────────────

func (h *ICMSApuracaoHandler) CreateApuracaoAdjCode(w http.ResponseWriter, r *http.Request) {
	var c fiscalEntity.ICMSApuracaoAdjustmentCode
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.apuracaoAdjUC.Create(r.Context(), &c)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, toApuracaoAdjResponse(result))
}

func (h *ICMSApuracaoHandler) UpdateApuracaoAdjCode(w http.ResponseWriter, r *http.Request) {
	var c fiscalEntity.ICMSApuracaoAdjustmentCode
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.apuracaoAdjUC.Update(r.Context(), &c)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toApuracaoAdjResponse(result))
}

func (h *ICMSApuracaoHandler) GetApuracaoAdjCode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.apuracaoAdjUC.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toApuracaoAdjResponse(result))
}

func (h *ICMSApuracaoHandler) ListApuracaoAdjCodes(w http.ResponseWriter, r *http.Request) {
	uf := r.URL.Query().Get("uf")
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.apuracaoAdjUC.List(r.Context(), uf, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]*response.ICMSApuracaoAdjCodeResponse, 0, len(result))
	for _, c := range result {
		out = append(out, toApuracaoAdjResponse(c))
	}
	jsonResponse(w, http.StatusOK, out)
}

func toApuracaoAdjResponse(c *fiscalEntity.ICMSApuracaoAdjustmentCode) *response.ICMSApuracaoAdjCodeResponse {
	return &response.ICMSApuracaoAdjCodeResponse{
		ID: c.ID, Code: c.Code, UF: c.UF, Description: c.Description,
		ValidFrom: c.ValidFrom, ValidTo: c.ValidTo, IsActive: c.IsActive, CreatedAt: c.CreatedAt,
	}
}

// ─── ICMS Adjustment Codes ────────────────────────────────────────────────────

func (h *ICMSApuracaoHandler) CreateAdjCode(w http.ResponseWriter, r *http.Request) {
	var c fiscalEntity.ICMSAdjustmentCode
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.adjCodeUC.Create(r.Context(), &c)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, toAdjCodeResponse(result))
}

func (h *ICMSApuracaoHandler) UpdateAdjCode(w http.ResponseWriter, r *http.Request) {
	var c fiscalEntity.ICMSAdjustmentCode
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.adjCodeUC.Update(r.Context(), &c)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toAdjCodeResponse(result))
}

func (h *ICMSApuracaoHandler) GetAdjCode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.adjCodeUC.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toAdjCodeResponse(result))
}

func (h *ICMSApuracaoHandler) ListAdjCodes(w http.ResponseWriter, r *http.Request) {
	uf := r.URL.Query().Get("uf")
	tableRef := r.URL.Query().Get("table_ref")
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.adjCodeUC.List(r.Context(), uf, tableRef, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]*response.ICMSAdjustmentCodeResponse, 0, len(result))
	for _, c := range result {
		out = append(out, toAdjCodeResponse(c))
	}
	jsonResponse(w, http.StatusOK, out)
}

func toAdjCodeResponse(c *fiscalEntity.ICMSAdjustmentCode) *response.ICMSAdjustmentCodeResponse {
	return &response.ICMSAdjustmentCodeResponse{
		ID: c.ID, UF: c.UF, Code: c.Code, Description: c.Description, TableRef: string(c.TableRef),
		ValidFrom: c.ValidFrom, ValidTo: c.ValidTo, IsActive: c.IsActive, CreatedAt: c.CreatedAt,
	}
}

// ─── ICMS Apuração Lines ──────────────────────────────────────────────────────

func (h *ICMSApuracaoHandler) CreateApuracaoLine(w http.ResponseWriter, r *http.Request) {
	var l fiscalEntity.ICMSApuracaoLine
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.apuracaoLineUC.Create(r.Context(), &l)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, toApuracaoLineResponse(result))
}

func (h *ICMSApuracaoHandler) UpdateApuracaoLine(w http.ResponseWriter, r *http.Request) {
	var l fiscalEntity.ICMSApuracaoLine
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.apuracaoLineUC.Update(r.Context(), &l)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toApuracaoLineResponse(result))
}

func (h *ICMSApuracaoHandler) GetApuracaoLine(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	result, err := h.apuracaoLineUC.GetByCode(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toApuracaoLineResponse(result))
}

func (h *ICMSApuracaoHandler) ListApuracaoLines(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.apuracaoLineUC.List(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]*response.ICMSApuracaoLineResponse, 0, len(result))
	for _, l := range result {
		out = append(out, toApuracaoLineResponse(l))
	}
	jsonResponse(w, http.StatusOK, out)
}

func toApuracaoLineResponse(l *fiscalEntity.ICMSApuracaoLine) *response.ICMSApuracaoLineResponse {
	return &response.ICMSApuracaoLineResponse{
		ID: l.ID, Code: l.Code, Description: l.Description, LineType: string(l.LineType),
		AcceptsEntries: l.AcceptsEntries, Nature: l.Nature, ApuracaoAdjCodeID: l.ApuracaoAdjCodeID,
		IsActive: l.IsActive, CreatedAt: l.CreatedAt,
	}
}

// ─── ICMS Summary Entries ─────────────────────────────────────────────────────

func (h *ICMSApuracaoHandler) CreateSummaryEntry(w http.ResponseWriter, r *http.Request) {
	var e fiscalEntity.ICMSSummaryEntry
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.summaryUC.Create(r.Context(), &e)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, toSummaryEntryResponse(result))
}

func (h *ICMSApuracaoHandler) UpdateSummaryEntry(w http.ResponseWriter, r *http.Request) {
	var e fiscalEntity.ICMSSummaryEntry
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.summaryUC.Update(r.Context(), &e)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toSummaryEntryResponse(result))
}

func (h *ICMSApuracaoHandler) GetSummaryEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.summaryUC.GetByID(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toSummaryEntryResponse(result))
}

func (h *ICMSApuracaoHandler) ListSummaryEntries(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	uf := r.URL.Query().Get("uf")
	result, err := h.summaryUC.List(r.Context(), period, uf)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]*response.ICMSSummaryEntryResponse, 0, len(result))
	for _, e := range result {
		out = append(out, toSummaryEntryResponse(e))
	}
	jsonResponse(w, http.StatusOK, out)
}

func (h *ICMSApuracaoHandler) AddSummaryEntryNote(w http.ResponseWriter, r *http.Request) {
	var n fiscalEntity.ICMSSummaryEntryNote
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.summaryUC.AddNote(r.Context(), &n)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ICMSApuracaoHandler) ListSummaryEntryNotes(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.summaryUC.ListNotes(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func toSummaryEntryResponse(e *fiscalEntity.ICMSSummaryEntry) *response.ICMSSummaryEntryResponse {
	return &response.ICMSSummaryEntryResponse{
		ID: e.ID, Period: e.Period, UF: e.UF, CFOPID: e.CFOPID,
		ICMSBase: e.ICMSBase, ICMSValue: e.ICMSValue, ICMSBaseOther: e.ICMSBaseOther,
		ICMSValueOther: e.ICMSValueOther, IsActive: e.IsActive, CreatedAt: e.CreatedAt,
	}
}

// ─── Simples Nacional ─────────────────────────────────────────────────────────

func (h *ICMSApuracaoHandler) CreateSimplesApuracao(w http.ResponseWriter, r *http.Request) {
	var s fiscalEntity.SimplesNacionalApuracao
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.simplesUC.Create(r.Context(), &s)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, toSimplesResponse(result))
}

func (h *ICMSApuracaoHandler) UpdateSimplesApuracao(w http.ResponseWriter, r *http.Request) {
	var s fiscalEntity.SimplesNacionalApuracao
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.simplesUC.Update(r.Context(), &s)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toSimplesResponse(result))
}

func (h *ICMSApuracaoHandler) GetSimplesApuracao(w http.ResponseWriter, r *http.Request) {
	period := chi.URLParam(r, "period")
	annex := chi.URLParam(r, "annex")
	result, err := h.simplesUC.Get(r.Context(), period, fiscalEntity.SimplesNacionalAnnex(annex))
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, toSimplesResponse(result))
}

func (h *ICMSApuracaoHandler) ListSimplesApuracoes(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	result, err := h.simplesUC.List(r.Context(), period)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]*response.SimplesNacionalApuracaoResponse, 0, len(result))
	for _, s := range result {
		out = append(out, toSimplesResponse(s))
	}
	jsonResponse(w, http.StatusOK, out)
}

func toSimplesResponse(s *fiscalEntity.SimplesNacionalApuracao) *response.SimplesNacionalApuracaoResponse {
	return &response.SimplesNacionalApuracaoResponse{
		ID: s.ID, Period: s.Period, Annex: string(s.Annex),
		ReceitaInterna: s.ReceitaInterna, ReceitaExterna: s.ReceitaExterna, FolhaPagamento: s.FolhaPagamento,
		ReceitaBruta12M: s.ReceitaBruta12M, SimplesRecolhido: s.SimplesRecolhido,
		AliquotaNominal: s.AliquotaNominal, AliquotaEfetiva: s.AliquotaEfetiva,
		AliquotaEfetivaICMS: s.AliquotaEfetivaICMS, ParcelaDeduzir: s.ParcelaDeduzir,
		Observation: s.Observation, IsActive: s.IsActive, CreatedAt: s.CreatedAt,
	}
}
