package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/recurring_sales_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
	rsrepo "github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type RecurringSalesHandler struct {
	uc *recurring_sales_uc.UseCase
}

func NewRecurringSalesHandler(uc *recurring_sales_uc.UseCase) *RecurringSalesHandler {
	return &RecurringSalesHandler{uc: uc}
}

func (h *RecurringSalesHandler) UpsertParameters(w http.ResponseWriter, r *http.Request) {
	var dto request.UpsertRecurringSalesParametersDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.UpsertParameters(r.Context(), dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) GetParameters(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "enterpriseCode")
	if !ok {
		return
	}
	result, err := h.uc.GetParameters(r.Context(), code)
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) CreateAdjustmentDate(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRecurringSalesAdjustmentDateDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CreateAdjustmentDate(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *RecurringSalesHandler) ListAdjustmentDates(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListAdjustmentDates(r.Context(), h.filter(r))
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRecurringSaleDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.Create(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *RecurringSalesHandler) List(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.List(r.Context(), h.filter(r))
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) Get(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.Get(r.Context(), code)
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) Update(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	var dto request.UpdateRecurringSaleDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.Update(r.Context(), code, dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) AddRepresentative(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateRecurringSaleRepresentativeDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if code, ok := optionalPathInt64(r, "code"); ok {
		dto.RecurringSaleCode = code
	}
	result, err := h.uc.AddRepresentative(r.Context(), dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *RecurringSalesHandler) MarkOrderGenerated(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	var dto request.MarkRecurringSaleOrderDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.GenerateSalesOrder(r.Context(), code, dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) ClearGeneratedOrder(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.ClearGeneratedOrder(r.Context(), code)
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	var dto request.CancelRecurringSaleDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.Cancel(r.Context(), code, dto)
	h.respond(w, result, err, http.StatusCreated)
}

func (h *RecurringSalesHandler) CalculateAdjustment(w http.ResponseWriter, r *http.Request) {
	var dto request.CalculateRecurringSalesAdjustmentDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.CalculateAdjustment(r.Context(), dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) RecalculateAdjustment(w http.ResponseWriter, r *http.Request) {
	code, ok := pathInt64(w, r, "code")
	if !ok {
		return
	}
	var dto request.RecalculateRecurringSalesAdjustmentDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	result, err := h.uc.RecalculateAdjustment(r.Context(), code, dto)
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) RevenueProjection(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.RevenueProjection(r.Context(), h.projectionFilter(r))
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) CommissionProjection(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.CommissionProjection(r.Context(), h.projectionFilter(r))
	h.respond(w, result, err, http.StatusOK)
}

func (h *RecurringSalesHandler) filter(r *http.Request) rsrepo.Filter {
	var movement *entity.MovementType
	if raw := strings.TrimSpace(r.URL.Query().Get("movement_type")); raw != "" {
		v := entity.MovementType(strings.ToUpper(raw))
		movement = &v
	}
	return rsrepo.Filter{
		EnterpriseCode: int64Query(r, "enterprise_code"), CustomerCode: int64Query(r, "customer_code"),
		EstablishmentCode: int64Query(r, "establishment_code"), ItemCode: int64Query(r, "item_code"),
		RepresentativeCode: int64Query(r, "representative_code"), MovementType: movement,
		OnlyActive: r.URL.Query().Get("active") != "false",
	}
}

func (h *RecurringSalesHandler) projectionFilter(r *http.Request) rsrepo.ProjectionFilter {
	pct := 0.0
	if raw := r.URL.Query().Get("adjustment_percent"); raw != "" {
		pct, _ = strconv.ParseFloat(raw, 64)
	}
	return rsrepo.ProjectionFilter{
		From: parseHandlerDateValue(r.URL.Query().Get("from")), To: parseHandlerDateValue(r.URL.Query().Get("to")),
		EnterpriseCode: int64Query(r, "enterprise_code"), CustomerCode: int64Query(r, "customer_code"),
		ItemCode: int64Query(r, "item_code"), RepresentativeCode: int64Query(r, "representative_code"),
		AdjustmentPercent: pct,
	}
}

func parseHandlerDateValue(s string) time.Time {
	t, _ := datetime.ParseDate(s)
	return t
}

func (h *RecurringSalesHandler) respond(w http.ResponseWriter, result any, err error, status int) {
	if err != nil {
		switch {
		case errors.Is(err, errorsuc.ErrUnauthorized):
			security.RespondError(w, http.StatusUnauthorized, err.Error())
		default:
			security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		}
		return
	}
	security.RespondJSON(w, status, result)
}
