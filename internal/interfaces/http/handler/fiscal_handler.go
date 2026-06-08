package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type FiscalHandler struct {
	createEntryUC        *fiscal_uc.CreateFiscalEntryUseCase
	uploadNFEUC          *fiscal_uc.UploadNFEEntryUseCase
	approveEntryUC       *fiscal_uc.ApproveFiscalEntryUseCase
	listEntriesUC        *fiscal_uc.ListFiscalEntriesUseCase
	getEntryUC           *fiscal_uc.GetFiscalEntryUseCase
	createExitUC         *fiscal_uc.CreateFiscalExitUseCase
	authorizeExitUC      *fiscal_uc.AuthorizeFiscalExitUseCase
	cancelExitUC         *fiscal_uc.CancelFiscalExitUseCase
	listExitsUC          *fiscal_uc.ListFiscalExitsUseCase
	getExitUC            *fiscal_uc.GetFiscalExitUseCase
	getConfigUC          *fiscal_uc.GetFiscalConfigUseCase
	updateConfigUC       *fiscal_uc.UpdateFiscalConfigUseCase
	emitirCCeUC          *fiscal_uc.EmitirCCeUseCase
	createCTeUC          *fiscal_uc.CreateCTeUseCase
	listCTeUC            *fiscal_uc.ListCTeUseCase
	getCTeUC             *fiscal_uc.GetCTeUseCase
	upsertNcmUC          *fiscal_uc.UpsertNcmTaxUseCase
	listNcmUC            *fiscal_uc.ListNcmTaxesUseCase
	deleteNcmUC          *fiscal_uc.DeleteNcmTaxUseCase
	upsertInterstateUC   *fiscal_uc.UpsertICMSInterstateUseCase
	listInterstateUC     *fiscal_uc.ListICMSInterstateUseCase
	upsertInternalUC     *fiscal_uc.UpsertICMSInternalUseCase
	listInternalUC       *fiscal_uc.ListICMSInternalUseCase
	consultarNFeUC       *fiscal_uc.ConsultarNFeUseCase
	listCartasCorrecaoUC *fiscal_uc.ListCartasCorrecaoUseCase
}

func NewFiscalHandler(
	createEntryUC *fiscal_uc.CreateFiscalEntryUseCase,
	uploadNFEUC *fiscal_uc.UploadNFEEntryUseCase,
	approveEntryUC *fiscal_uc.ApproveFiscalEntryUseCase,
	listEntriesUC *fiscal_uc.ListFiscalEntriesUseCase,
	getEntryUC *fiscal_uc.GetFiscalEntryUseCase,
	createExitUC *fiscal_uc.CreateFiscalExitUseCase,
	authorizeExitUC *fiscal_uc.AuthorizeFiscalExitUseCase,
	cancelExitUC *fiscal_uc.CancelFiscalExitUseCase,
	listExitsUC *fiscal_uc.ListFiscalExitsUseCase,
	getExitUC *fiscal_uc.GetFiscalExitUseCase,
	getConfigUC *fiscal_uc.GetFiscalConfigUseCase,
	updateConfigUC *fiscal_uc.UpdateFiscalConfigUseCase,
	emitirCCeUC *fiscal_uc.EmitirCCeUseCase,
	createCTeUC *fiscal_uc.CreateCTeUseCase,
	listCTeUC *fiscal_uc.ListCTeUseCase,
	getCTeUC *fiscal_uc.GetCTeUseCase,
	upsertNcmUC *fiscal_uc.UpsertNcmTaxUseCase,
	listNcmUC *fiscal_uc.ListNcmTaxesUseCase,
	deleteNcmUC *fiscal_uc.DeleteNcmTaxUseCase,
	upsertInterstateUC *fiscal_uc.UpsertICMSInterstateUseCase,
	listInterstateUC *fiscal_uc.ListICMSInterstateUseCase,
	upsertInternalUC *fiscal_uc.UpsertICMSInternalUseCase,
	listInternalUC *fiscal_uc.ListICMSInternalUseCase,
	consultarNFeUC *fiscal_uc.ConsultarNFeUseCase,
	listCartasCorrecaoUC *fiscal_uc.ListCartasCorrecaoUseCase,
) *FiscalHandler {
	return &FiscalHandler{
		createEntryUC:        createEntryUC,
		uploadNFEUC:          uploadNFEUC,
		approveEntryUC:       approveEntryUC,
		listEntriesUC:        listEntriesUC,
		getEntryUC:           getEntryUC,
		createExitUC:         createExitUC,
		authorizeExitUC:      authorizeExitUC,
		cancelExitUC:         cancelExitUC,
		listExitsUC:          listExitsUC,
		getExitUC:            getExitUC,
		getConfigUC:          getConfigUC,
		updateConfigUC:       updateConfigUC,
		emitirCCeUC:          emitirCCeUC,
		createCTeUC:          createCTeUC,
		listCTeUC:            listCTeUC,
		getCTeUC:             getCTeUC,
		upsertNcmUC:          upsertNcmUC,
		listNcmUC:            listNcmUC,
		deleteNcmUC:          deleteNcmUC,
		upsertInterstateUC:   upsertInterstateUC,
		listInterstateUC:     listInterstateUC,
		upsertInternalUC:     upsertInternalUC,
		listInternalUC:       listInternalUC,
		consultarNFeUC:       consultarNFeUC,
		listCartasCorrecaoUC: listCartasCorrecaoUC,
	}
}

func (h *FiscalHandler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateFiscalEntryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createEntryUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FiscalHandler) UploadNFE(w http.ResponseWriter, r *http.Request) {
	var dto request.UploadNFEDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uploadNFEUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FiscalHandler) ApproveEntry(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.approveEntryUC.Execute(r.Context(), request.ApproveFiscalEntryDTO{ID: code})
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) ListEntries(w http.ResponseWriter, r *http.Request) {
	results, err := h.listEntriesUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *FiscalHandler) GetEntry(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getEntryUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) CreateExit(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateFiscalExitDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createExitUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FiscalHandler) AuthorizeExit(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.authorizeExitUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) CancelExit(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var body struct {
		Motivo string `json:"motivo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.cancelExitUC.Execute(r.Context(), fiscal_uc.CancelFiscalExitParams{
		ID:     code,
		Motivo: body.Motivo,
	})
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) ListExits(w http.ResponseWriter, r *http.Request) {
	results, err := h.listExitsUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *FiscalHandler) GetExit(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getExitUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	result, err := h.getConfigUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateFiscalConfigDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.updateConfigUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// CC-e

func (h *FiscalHandler) EmitirCCe(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var body struct {
		TextoCorrecao string `json:"texto_correcao"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.emitirCCeUC.Execute(r.Context(), fiscal_uc.EmitirCCeParams{
		FiscalExitID:  code,
		TextoCorrecao: body.TextoCorrecao,
	})
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// CT-e

func (h *FiscalHandler) CreateCTe(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCTeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createCTeUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *FiscalHandler) ListCTe(w http.ResponseWriter, r *http.Request) {
	results, err := h.listCTeUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *FiscalHandler) GetCTe(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getCTeUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---------- NCM Tax Table ----------

func (h *FiscalHandler) UpsertNcmTax(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertNcmTaxDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.upsertNcmUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) ListNcmTaxes(w http.ResponseWriter, r *http.Request) {
	results, err := h.listNcmUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *FiscalHandler) DeleteNcmTax(w http.ResponseWriter, r *http.Request) {
	ncm := chi.URLParam(r, "ncm")
	if err := h.deleteNcmUC.Execute(r.Context(), ncm); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "NCM desativado"})
}

// ---------- ICMS Tables ----------

func (h *FiscalHandler) UpsertICMSInterstate(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertICMSInterstateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.upsertInterstateUC.Execute(r.Context(), dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "ICMS interestadual atualizado"})
}

func (h *FiscalHandler) ListICMSInterstate(w http.ResponseWriter, r *http.Request) {
	result, err := h.listInterstateUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) UpsertICMSInternal(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertICMSInternalDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.upsertInternalUC.Execute(r.Context(), dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"message": "ICMS interno atualizado"})
}

func (h *FiscalHandler) ListICMSInternal(w http.ResponseWriter, r *http.Request) {
	result, err := h.listInternalUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// ---------- NF-e consultation & CC-e list ----------

func (h *FiscalHandler) ConsultarNFe(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	result, err := h.consultarNFeUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *FiscalHandler) ListCartasCorrecao(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	results, err := h.listCartasCorrecaoUC.Execute(r.Context(), id)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}
