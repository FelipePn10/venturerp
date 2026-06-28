package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/gantt"
	"github.com/go-chi/chi/v5"
)

func (h *APSHandler) SequenceOrders(w http.ResponseWriter, r *http.Request) {
	var dto request.SequenceOrdersDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.SequenceOrders(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *APSHandler) GetGanttByOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "orderID"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid orderID")
		return
	}
	result, err := h.uc.GetGanttByOrder(r.Context(), orderID)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (h *APSHandler) GetGanttByWorkCenter(w http.ResponseWriter, r *http.Request) {
	var dto request.GanttByWorkCenterDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.GetGanttByWorkCenter(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// GetMonthGantt serves the monthly production-schedule board as JSON.
// GET /api/aps/gantt/month/{year}/{month}?group_by=work_center|order
func (h *APSHandler) GetMonthGantt(w http.ResponseWriter, r *http.Request) {
	year, month, ok := parseYearMonth(w, r)
	if !ok {
		return
	}
	groupBy := aps_uc.ParseGroupBy(r.URL.Query().Get("group_by"))
	result, err := h.uc.GetMonthSchedule(r.Context(), year, month, groupBy)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// GetGanttBoard serves the schedule board for an arbitrary range and scale as JSON.
// GET /api/aps/gantt/board?from=YYYY-MM-DD&to=YYYY-MM-DD&scale=day|week&group_by=...
func (h *APSHandler) GetGanttBoard(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseRange(w, r)
	if !ok {
		return
	}
	scale := aps_uc.ParseScale(r.URL.Query().Get("scale"))
	groupBy := aps_uc.ParseGroupBy(r.URL.Query().Get("group_by"))
	result, err := h.uc.GetBoard(r.Context(), from, to, scale, groupBy)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ExportGanttBoard renders an arbitrary-range board as a downloadable SVG or PDF.
// GET /api/aps/gantt/board/export?from=...&to=...&scale=...&group_by=...&format=svg|pdf
func (h *APSHandler) ExportGanttBoard(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseRange(w, r)
	if !ok {
		return
	}
	scale := aps_uc.ParseScale(r.URL.Query().Get("scale"))
	groupBy := aps_uc.ParseGroupBy(r.URL.Query().Get("group_by"))
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "svg"
	}
	board, err := h.uc.BuildBoard(r.Context(), from, to, scale, groupBy)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	data, contentType, err := gantt.Render(board, format, h.ganttBranding(r.Context()))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	filename := "programacao-" + from.Format("20060102") + "-" + to.Format("20060102") + "." + format
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// RescheduleSequence applies a manual board move (drag-drop) and returns the move,
// any cascaded shifts and capacity warnings.
// POST /api/aps/gantt/reschedule
func (h *APSHandler) RescheduleSequence(w http.ResponseWriter, r *http.Request) {
	var dto request.RescheduleSequenceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	result, err := h.uc.RescheduleSequence(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

// ExportMonthGantt renders the monthly board as a downloadable SVG or PDF.
// GET /api/aps/gantt/month/{year}/{month}/export?format=svg|pdf&group_by=...
func (h *APSHandler) ExportMonthGantt(w http.ResponseWriter, r *http.Request) {
	year, month, ok := parseYearMonth(w, r)
	if !ok {
		return
	}
	groupBy := aps_uc.ParseGroupBy(r.URL.Query().Get("group_by"))
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "svg"
	}

	board, err := h.uc.BuildMonthSchedule(r.Context(), year, month, groupBy)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	data, contentType, err := gantt.Render(board, format, h.ganttBranding(r.Context()))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	filename := "programacao-" + strconv.Itoa(year) + "-" + leftPad2(month) + "." + format
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *APSHandler) ganttBranding(ctx context.Context) gantt.Branding {
	b := gantt.Branding{GeneratedAt: time.Now()}
	if h.fiscal != nil {
		if cfg, err := h.fiscal.GetFiscalConfig(ctx); err == nil && cfg != nil {
			b.CompanyName = cfg.RazaoSocial
			if cfg.BrandColor != nil {
				b.BrandColorHex = *cfg.BrandColor
			}
		}
	}
	return b
}

func parseYearMonth(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid year")
		return 0, 0, false
	}
	month, err := strconv.Atoi(chi.URLParam(r, "month"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid month")
		return 0, 0, false
	}
	return year, month, true
}

// parseRange reads from/to (YYYY-MM-DD) query params. 'to' is inclusive in the URL
// for an intuitive API ("from 1st to 30th") and converted to the half-open end the
// board uses. A missing 'to' defaults to a 30-day window from 'from'.
func parseRange(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	q := r.URL.Query()
	from, err := time.ParseInLocation("2006-01-02", q.Get("from"), time.Local)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid or missing 'from' (want YYYY-MM-DD)")
		return time.Time{}, time.Time{}, false
	}
	var to time.Time
	if raw := q.Get("to"); raw != "" {
		to, err = time.ParseInLocation("2006-01-02", raw, time.Local)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid 'to' (want YYYY-MM-DD)")
			return time.Time{}, time.Time{}, false
		}
		to = to.AddDate(0, 0, 1) // make the inclusive end day half-open
	} else {
		to = from.AddDate(0, 0, 30)
	}
	return from, to, true
}

func leftPad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
