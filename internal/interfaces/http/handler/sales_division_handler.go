package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_division_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type SalesDivisionHandler struct {
	createUC *sales_division_uc.CreateSalesDivisionUseCase
	listUC   *sales_division_uc.ListSalesDivisionsUseCase
	getUC    *sales_division_uc.GetSalesDivisionUseCase
	updateUC *sales_division_uc.UpdateSalesDivisionUseCase
	deleteUC *sales_division_uc.DeleteSalesDivisionUseCase
}

func NewSalesDivisionHandler(
	createUC *sales_division_uc.CreateSalesDivisionUseCase,
	listUC *sales_division_uc.ListSalesDivisionsUseCase,
	getUC *sales_division_uc.GetSalesDivisionUseCase,
	updateUC *sales_division_uc.UpdateSalesDivisionUseCase,
	deleteUC *sales_division_uc.DeleteSalesDivisionUseCase,
) *SalesDivisionHandler {
	return &SalesDivisionHandler{
		createUC: createUC,
		listUC:   listUC,
		getUC:    getUC,
		updateUC: updateUC,
		deleteUC: deleteUC,
	}
}

func (h *SalesDivisionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesDivisionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesDivisionHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesDivisionHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesDivisionHandler) Update(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.UpdateSalesDivisionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.updateUC.Execute(r.Context(), code, dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesDivisionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := h.deleteUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
