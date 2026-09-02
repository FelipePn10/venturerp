package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/bom_header_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type BomHeaderHandler struct {
	uc *bom_header_uc.BomHeaderUseCase
}

func NewBomHeaderHandler(uc *bom_header_uc.BomHeaderUseCase) *BomHeaderHandler {
	return &BomHeaderHandler{uc: uc}
}

func (h *BomHeaderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateBomHeaderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusCreated, result)
}

// ListByItem recebe o código de negócio do item (texto), como a tela o conhece.
func (h *BomHeaderHandler) ListByItem(w http.ResponseWriter, r *http.Request) {
	itemCode := strings.TrimSpace(chi.URLParam(r, "itemCode"))
	if itemCode == "" {
		jsonError(w, http.StatusBadRequest, "informe o código do item")
		return
	}
	result, err := h.uc.ListByItem(r.Context(), request.TextCode(itemCode))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *BomHeaderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador inválido")
		return
	}
	result, err := h.uc.Get(r.Context(), id)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *BomHeaderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "identificador inválido")
		return
	}
	var dto request.UpdateBomHeaderStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "conteúdo da requisição inválido: "+err.Error())
		return
	}
	dto.ID = id
	result, err := h.uc.UpdateStatus(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, result)
}
