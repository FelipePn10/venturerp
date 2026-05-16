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
	createEntryUC   *fiscal_uc.CreateFiscalEntryUseCase
	uploadNFEUC     *fiscal_uc.UploadNFEEntryUseCase
	approveEntryUC  *fiscal_uc.ApproveFiscalEntryUseCase
	listEntriesUC   *fiscal_uc.ListFiscalEntriesUseCase
	getEntryUC      *fiscal_uc.GetFiscalEntryUseCase
	createExitUC    *fiscal_uc.CreateFiscalExitUseCase
	authorizeExitUC *fiscal_uc.AuthorizeFiscalExitUseCase
	cancelExitUC    *fiscal_uc.CancelFiscalExitUseCase
	listExitsUC     *fiscal_uc.ListFiscalExitsUseCase
	getExitUC       *fiscal_uc.GetFiscalExitUseCase
	getConfigUC     *fiscal_uc.GetFiscalConfigUseCase
	updateConfigUC  *fiscal_uc.UpdateFiscalConfigUseCase
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
) *FiscalHandler {
	return &FiscalHandler{
		createEntryUC:   createEntryUC,
		uploadNFEUC:     uploadNFEUC,
		approveEntryUC:  approveEntryUC,
		listEntriesUC:   listEntriesUC,
		getEntryUC:      getEntryUC,
		createExitUC:    createExitUC,
		authorizeExitUC: authorizeExitUC,
		cancelExitUC:    cancelExitUC,
		listExitsUC:     listExitsUC,
		getExitUC:       getExitUC,
		getConfigUC:     getConfigUC,
		updateConfigUC:  updateConfigUC,
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
	result, err := h.cancelExitUC.Execute(r.Context(), code)
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
