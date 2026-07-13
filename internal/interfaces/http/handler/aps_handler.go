package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/gantt"
	"github.com/go-chi/chi/v5"
)

func (h *APSHandler) ExportSequencingEvents(w http.ResponseWriter, r *http.Request) {
	var dto request.SequenceOrdersDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	rows, err := h.uc.ExportSequencingEvents(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if r.URL.Query().Get("format") != "csv" {
		jsonResponse(w, http.StatusOK, rows)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=sequencing-events.csv")
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"event_type", "production_order_id", "order_number", "machine_id", "work_center_id", "operation_id", "event_at", "quantity", "reason"})
	for _, row := range rows {
		_ = writer.Write([]string{row.EventType, strconv.FormatInt(row.ProductionOrderID, 10), strconv.FormatInt(row.OrderNumber, 10), formatOptionalID(row.MachineID), formatOptionalID(row.WorkCenterID), formatOptionalID(row.OperationID), row.EventAt.Format(time.RFC3339), row.Quantity, row.Reason})
	}
	writer.Flush()
}

func (h *APSHandler) ListSequencingResources(w http.ResponseWriter, r *http.Request) {
	rows, err := h.uc.ListSequencingResources(r.Context())
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, rows)
}

func (h *APSHandler) ViewSequencing(w http.ResponseWriter, r *http.Request) {
	var dto request.SequencingViewDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	rows, err := h.uc.ViewSequencing(r.Context(), dto)
	if err != nil {
		jsonError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, rows)
}

func formatOptionalID(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}

func (h *APSHandler) UpsertResourceGroup(w http.ResponseWriter, r *http.Request) {
	var dto request.ResourceGroupDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, 400, "invalid payload")
		return
	}
	v, err := h.uc.UpsertResourceGroup(r.Context(), dto)
	if err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *APSHandler) ListResourceGroups(w http.ResponseWriter, r *http.Request) {
	v, err := h.uc.ListResourceGroups(r.Context())
	if err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *APSHandler) UpsertMachineCalendar(w http.ResponseWriter, r *http.Request) {
	var dto request.MachineCalendarDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, 400, "invalid payload")
		return
	}
	v, err := h.uc.UpsertMachineCalendar(r.Context(), dto)
	if err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *APSHandler) ListMachineCalendars(w http.ResponseWriter, r *http.Request) {
	v, err := h.uc.ListMachineCalendars(r.Context())
	if err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *APSHandler) UpdateSequencingSettings(w http.ResponseWriter, r *http.Request) {
	var dto request.SequencingSettingsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, 400, "invalid payload")
		return
	}
	if err := h.uc.UpdateSequencingSettings(r.Context(), dto); err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *APSHandler) UpdateWorkCenterSequencing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	var dto request.WorkCenterSequencingDTO
	if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, 400, "invalid payload")
		return
	}
	if err = h.uc.UpdateWorkCenterSequencing(r.Context(), id, dto); err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *APSHandler) UpdateResourceSequencing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	var dto request.ResourceSequencingDTO
	if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, 400, "invalid payload")
		return
	}
	if err = h.uc.UpdateResourceSequencing(r.Context(), id, dto); err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *APSHandler) DeleteResourceGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	if err = h.uc.DeleteResourceGroup(r.Context(), id); err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *APSHandler) DeleteMachineCalendar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	if err = h.uc.DeleteMachineCalendar(r.Context(), id); err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *APSHandler) CreateMachineDowntime(w http.ResponseWriter, r *http.Request) {
	var dto request.MachineDowntimeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, 400, "invalid payload")
		return
	}
	v, err := h.uc.CreateMachineDowntime(r.Context(), dto)
	if err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	jsonResponse(w, 201, v)
}
func (h *APSHandler) ListMachineDowntimes(w http.ResponseWriter, r *http.Request) {
	machineID, _ := strconv.ParseInt(r.URL.Query().Get("machine_id"), 10, 64)
	from, err1 := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, err2 := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if err1 != nil || err2 != nil {
		jsonError(w, 400, "from/to must be RFC3339")
		return
	}
	v, err := h.uc.ListMachineDowntimes(r.Context(), machineID, from, to)
	if err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *APSHandler) DeleteMachineDowntime(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	if err = h.uc.DeleteMachineDowntime(r.Context(), id); err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *APSHandler) UpsertEmployeeSequencingProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	var dto request.EmployeeSequencingProfileDTO
	if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, 400, "invalid payload")
		return
	}
	if err = h.uc.UpsertEmployeeSequencingProfile(r.Context(), id, dto); err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *APSHandler) UpsertMachineIndustrialProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	var dto request.MachineIndustrialProfileDTO
	if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
		jsonError(w, 400, "invalid payload")
		return
	}
	if err = h.uc.UpsertMachineIndustrialProfile(r.Context(), id, dto); err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *APSHandler) GetEmployeeSequencingProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	v, err := h.uc.GetEmployeeSequencingProfile(r.Context(), id)
	if err != nil {
		jsonError(w, 404, err.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *APSHandler) GetMachineIndustrialProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	v, err := h.uc.GetMachineIndustrialProfile(r.Context(), id)
	if err != nil {
		jsonError(w, 404, err.Error())
		return
	}
	jsonResponse(w, 200, v)
}

func apsPathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, name), 10, 64)
}
func decodeAPSBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		jsonError(w, 400, "invalid payload")
		return false
	}
	return true
}
func granularResult(w http.ResponseWriter, err error) {
	if err != nil {
		jsonError(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APSHandler) UpdateEmployeeContact(w http.ResponseWriter, r *http.Request) {
	employeeID, e1 := apsPathID(r, "employeeID")
	id, e2 := apsPathID(r, "contactID")
	var dto request.EmployeeContactDTO
	if e1 != nil || e2 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	if !decodeAPSBody(w, r, &dto) {
		return
	}
	granularResult(w, h.uc.UpdateEmployeeContact(r.Context(), employeeID, id, dto))
}
func (h *APSHandler) DeleteEmployeeContact(w http.ResponseWriter, r *http.Request) {
	employeeID, e1 := apsPathID(r, "employeeID")
	id, e2 := apsPathID(r, "contactID")
	if e1 != nil || e2 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	granularResult(w, h.uc.DeleteEmployeeContact(r.Context(), employeeID, id))
}
func (h *APSHandler) UpdateEmployeeFunction(w http.ResponseWriter, r *http.Request) {
	employeeID, e1 := apsPathID(r, "employeeID")
	id, e2 := apsPathID(r, "functionID")
	var dto request.EmployeeFunctionDTO
	if e1 != nil || e2 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	if !decodeAPSBody(w, r, &dto) {
		return
	}
	granularResult(w, h.uc.UpdateEmployeeFunction(r.Context(), employeeID, id, dto))
}
func (h *APSHandler) DeleteEmployeeFunction(w http.ResponseWriter, r *http.Request) {
	employeeID, e1 := apsPathID(r, "employeeID")
	id, e2 := apsPathID(r, "functionID")
	if e1 != nil || e2 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	granularResult(w, h.uc.DeleteEmployeeFunction(r.Context(), employeeID, id))
}
func (h *APSHandler) UpdateMachineService(w http.ResponseWriter, r *http.Request) {
	machineID, e1 := apsPathID(r, "machineID")
	id, e2 := apsPathID(r, "serviceID")
	var dto request.MachineServiceDTO
	if e1 != nil || e2 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	if !decodeAPSBody(w, r, &dto) {
		return
	}
	granularResult(w, h.uc.UpdateMachineService(r.Context(), machineID, id, dto))
}
func (h *APSHandler) DeleteMachineService(w http.ResponseWriter, r *http.Request) {
	machineID, e1 := apsPathID(r, "machineID")
	id, e2 := apsPathID(r, "serviceID")
	if e1 != nil || e2 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	granularResult(w, h.uc.DeleteMachineService(r.Context(), machineID, id))
}
func (h *APSHandler) UpdateMachineServiceItem(w http.ResponseWriter, r *http.Request) {
	machineID, e1 := apsPathID(r, "machineID")
	serviceID, e2 := apsPathID(r, "serviceID")
	id, e3 := apsPathID(r, "itemID")
	var dto request.ServiceItemDTO
	if e1 != nil || e2 != nil || e3 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	if !decodeAPSBody(w, r, &dto) {
		return
	}
	granularResult(w, h.uc.UpdateMachineServiceItem(r.Context(), machineID, serviceID, id, dto))
}
func (h *APSHandler) DeleteMachineServiceItem(w http.ResponseWriter, r *http.Request) {
	machineID, e1 := apsPathID(r, "machineID")
	serviceID, e2 := apsPathID(r, "serviceID")
	id, e3 := apsPathID(r, "itemID")
	if e1 != nil || e2 != nil || e3 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	granularResult(w, h.uc.DeleteMachineServiceItem(r.Context(), machineID, serviceID, id))
}
func (h *APSHandler) UpdateMachineSpecialValue(w http.ResponseWriter, r *http.Request) {
	machineID, e1 := apsPathID(r, "machineID")
	id, e2 := apsPathID(r, "fieldID")
	var dto request.SpecialValueDTO
	if e1 != nil || e2 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	if !decodeAPSBody(w, r, &dto) {
		return
	}
	granularResult(w, h.uc.UpdateMachineSpecialValue(r.Context(), machineID, id, dto))
}
func (h *APSHandler) DeleteMachineSpecialValue(w http.ResponseWriter, r *http.Request) {
	machineID, e1 := apsPathID(r, "machineID")
	id, e2 := apsPathID(r, "fieldID")
	if e1 != nil || e2 != nil {
		jsonError(w, 400, "invalid id")
		return
	}
	granularResult(w, h.uc.DeleteMachineSpecialValue(r.Context(), machineID, id))
}

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
