package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_classification_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
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
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.CreateMask(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ItemClassificationHandler) UpdateMask(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateClassificationMaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.UpdateMask(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) GetMask(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código inválido")
		return
	}
	result, err := h.uc.GetMaskByCode(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) ListMasks(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListMasks(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "não foi possível listar as máscaras de classificação")
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ─── Classifications ──────────────────────────────────────────────────────────

func (h *ItemClassificationHandler) CreateClassification(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateItemClassificationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.CreateClassification(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

func (h *ItemClassificationHandler) UpdateClassification(w http.ResponseWriter, r *http.Request) {
	var dto request.UpdateItemClassificationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.UpdateClassification(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) GetClassification(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	maskCode, err := strconv.ParseInt(chi.URLParam(r, "maskCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código de máscara inválido")
		return
	}
	result, err := h.uc.GetByCode(r.Context(), code, maskCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ListByMask lista as classificações de uma máscara pelo seu código de negócio —
// é o código que a tela conhece e envia.
func (h *ItemClassificationHandler) ListByMask(w http.ResponseWriter, r *http.Request) {
	maskCode, err := strconv.ParseInt(chi.URLParam(r, "maskCode"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "código de máscara inválido")
		return
	}
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListByMaskCode(r.Context(), maskCode, onlyActive)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) ListChildren(w http.ResponseWriter, r *http.Request) {
	parentID, err := strconv.ParseInt(chi.URLParam(r, "parentID"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador da classificação pai inválido")
		return
	}
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListChildren(r.Context(), parentID, onlyActive)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *ItemClassificationHandler) ListCatalog(w http.ResponseWriter, r *http.Request) {
	onlyActive := r.URL.Query().Get("only_active") != "false"
	result, err := h.uc.ListCatalog(r.Context(), onlyActive)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "não foi possível listar as classificações de item")
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
