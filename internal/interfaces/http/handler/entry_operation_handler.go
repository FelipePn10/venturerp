package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/entry_operation_uc"
	"github.com/go-chi/chi/v5"
)

type EntryOperationHandler struct {
	uc *entry_operation_uc.EntryOperationUseCase
}

func NewEntryOperationHandler(uc *entry_operation_uc.EntryOperationUseCase) *EntryOperationHandler {
	return &EntryOperationHandler{uc: uc}
}

// ─── State Groups ─────────────────────────────────────────────────────────────

func (h *EntryOperationHandler) CreateStateGroup(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateStateGroupDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.CreateStateGroup(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *EntryOperationHandler) GetStateGroup(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	res, err := h.uc.GetStateGroup(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *EntryOperationHandler) ListStateGroups(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListStateGroups(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *EntryOperationHandler) AddStateGroupUF(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.AddStateGroupUFDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.StateGroupCode = code
	if err := h.uc.AddStateGroupUF(r.Context(), dto); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Entry Operation Types ────────────────────────────────────────────────────

func (h *EntryOperationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateEntryOperationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.CreateEntryOperation(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *EntryOperationHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateEntryOperationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.UpdateEntryOperation(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *EntryOperationHandler) Get(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	res, err := h.uc.GetEntryOperation(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *EntryOperationHandler) List(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	res, err := h.uc.ListEntryOperations(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

// Validate applies the UF × state-group rule: GET /{code}/validate?uf=XX
func (h *EntryOperationHandler) Validate(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	uf := r.URL.Query().Get("uf")
	if uf == "" {
		jsonError(w, http.StatusBadRequest, "uf is required")
		return
	}
	if verr := h.uc.ValidateUF(r.Context(), code, uf); verr != nil {
		jsonResponse(w, http.StatusOK, map[string]any{"valid": false, "reason": verr.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"valid": true})
}
