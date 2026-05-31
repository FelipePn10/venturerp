package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_classification_uc"
	"github.com/go-chi/chi/v5"
)

type ItemClassificationHandler struct {
	uc *item_classification_uc.ItemClassificationUseCase
}

func NewItemClassificationHandler(uc *item_classification_uc.ItemClassificationUseCase) *ItemClassificationHandler {
	return &ItemClassificationHandler{uc: uc}
}

// ─── Masks ────────────────────────────────────────────────────────────────────

func (h *ItemClassificationHandler) CreateMask(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateClassificationMaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateMask(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ItemClassificationHandler) UpdateMask(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateClassificationMaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateMask(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) GetMask(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.uc.GetMaskByCode(r.Context(), code)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) ListMasks(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListMasks(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Classifications ──────────────────────────────────────────────────────────

func (h *ItemClassificationHandler) CreateClassification(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateItemClassificationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.CreateClassification(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ItemClassificationHandler) UpdateClassification(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateItemClassificationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.UpdateClassification(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) GetClassification(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	maskCode, err := strconv.ParseInt(chi.URLParam(r, "maskCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid mask code")
		return
	}
	result, err := h.uc.GetByCode(r.Context(), code, maskCode)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) ListByMask(w http.ResponseWriter, r *http.Request) {
	maskID, err := strconv.ParseInt(chi.URLParam(r, "maskID"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid mask id")
		return
	}
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListByMask(r.Context(), maskID, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) ListChildren(w http.ResponseWriter, r *http.Request) {
	parentID, err := strconv.ParseInt(chi.URLParam(r, "parentID"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid parent id")
		return
	}
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListChildren(r.Context(), parentID, onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
