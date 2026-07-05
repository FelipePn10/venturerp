package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_forecast_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"
)

type SalesForecastHandler struct {
	createForecastUC      *sales_forecast_uc.CreateSalesForecastUseCase
	createMonthlyUC       *sales_forecast_uc.CreateMonthlySalesForecastUseCase
	listForecastsUC       *sales_forecast_uc.ListSalesForecastsUseCase
	getForecastByItemUC   *sales_forecast_uc.GetForecastByItemUseCase
	generateForecastUC    *sales_forecast_uc.GenerateSalesForecastUseCase
	createBlockUC         *sales_forecast_uc.CreateForecastBlockUseCase
	listBlocksUC          *sales_forecast_uc.ListForecastBlocksUseCase
	createAppropriationUC *sales_forecast_uc.CreateAppropriationTableUseCase
	listAppropriationsUC  *sales_forecast_uc.ListAppropriationTablesUseCase
	setDefaultUC          *sales_forecast_uc.SetDefaultAppropriationUseCase
}

func NewSalesForecastHandler(
	createForecastUC *sales_forecast_uc.CreateSalesForecastUseCase,
	createMonthlyUC *sales_forecast_uc.CreateMonthlySalesForecastUseCase,
	listForecastsUC *sales_forecast_uc.ListSalesForecastsUseCase,
	getForecastByItemUC *sales_forecast_uc.GetForecastByItemUseCase,
	generateForecastUC *sales_forecast_uc.GenerateSalesForecastUseCase,
	createBlockUC *sales_forecast_uc.CreateForecastBlockUseCase,
	listBlocksUC *sales_forecast_uc.ListForecastBlocksUseCase,
	createAppropriationUC *sales_forecast_uc.CreateAppropriationTableUseCase,
	listAppropriationsUC *sales_forecast_uc.ListAppropriationTablesUseCase,
	setDefaultUC *sales_forecast_uc.SetDefaultAppropriationUseCase,
) *SalesForecastHandler {
	return &SalesForecastHandler{
		createForecastUC:      createForecastUC,
		createMonthlyUC:       createMonthlyUC,
		listForecastsUC:       listForecastsUC,
		getForecastByItemUC:   getForecastByItemUC,
		generateForecastUC:    generateForecastUC,
		createBlockUC:         createBlockUC,
		listBlocksUC:          listBlocksUC,
		createAppropriationUC: createAppropriationUC,
		listAppropriationsUC:  listAppropriationsUC,
		setDefaultUC:          setDefaultUC,
	}
}

func (h *SalesForecastHandler) CreateMonthlyForecast(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateMonthlySalesForecastDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createMonthlyUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesForecastHandler) GenerateForecast(w http.ResponseWriter, r *http.Request) {
	var dto request.GenerateSalesForecastDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.generateForecastUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

// ---- Forecasts ----

func (h *SalesForecastHandler) CreateForecast(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesForecastDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createForecastUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesForecastHandler) ListForecasts(w http.ResponseWriter, r *http.Request) {
	yearStr := chi.URLParam(r, "year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid year")
		return
	}
	results, err := h.listForecastsUC.Execute(r.Context(), year)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesForecastHandler) GetForecastByItem(w http.ResponseWriter, r *http.Request) {
	itemCode, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	results, err := h.getForecastByItemUC.Execute(r.Context(), itemCode)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ---- Forecast Blocks ----

func (h *SalesForecastHandler) CreateBlock(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateForecastBlockDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createBlockUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesForecastHandler) ListBlocks(w http.ResponseWriter, r *http.Request) {
	results, err := h.listBlocksUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

// ---- Appropriation Tables ----

func (h *SalesForecastHandler) CreateAppropriation(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateAppropriationTableDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createAppropriationUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesForecastHandler) ListAppropriations(w http.ResponseWriter, r *http.Request) {
	results, err := h.listAppropriationsUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesForecastHandler) SetDefaultAppropriation(w http.ResponseWriter, r *http.Request) {
	var dto request.SetDefaultAppropriationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.setDefaultUC.Execute(r.Context(), dto.ID); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"status": "default appropriation table updated"})
}
