package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/lot_mask_uc"
	"github.com/go-chi/chi/v5"
)

// LotMaskHandler serves the Lot/Serial Mask register + lot-code generation.
type LotMaskHandler struct {
	uc *lot_mask_uc.LotMaskUseCase
}

func NewLotMaskHandler(uc *lot_mask_uc.LotMaskUseCase) *LotMaskHandler {
	return &LotMaskHandler{uc: uc}
}

func (h *LotMaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.LotMaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.CreatedBy = actingUser(r)
	res, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *LotMaskHandler) List(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.List(r.Context(), r.URL.Query().Get("only_active") == "true")
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *LotMaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid lot mask id")
		return
	}
	res, err := h.uc.Get(r.Context(), id)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *LotMaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid lot mask id")
		return
	}
	var dto request.LotMaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	dto.ID = id
	res, err := h.uc.Update(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *LotMaskHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid lot mask id")
		return
	}
	if err := h.uc.Deactivate(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *LotMaskHandler) AddPart(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid lot mask id")
		return
	}
	var dto request.LotMaskPartDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.AddPart(r.Context(), id, dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, res)
}

func (h *LotMaskHandler) UpdatePart(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "partId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid part id")
		return
	}
	var dto request.LotMaskPartDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.UpdatePart(r.Context(), id, dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}

func (h *LotMaskHandler) DeletePart(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "partId"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid part id")
		return
	}
	if err := h.uc.DeletePart(r.Context(), id); err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Generate produces a lot code from the resolved mask.
func (h *LotMaskHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var dto request.GenerateLotDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	res, err := h.uc.Generate(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, res)
}
