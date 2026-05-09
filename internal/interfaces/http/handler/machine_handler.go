package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

type MachineHandler struct {
	createMachineUC *machine_uc.CreateMachineUseCase
	listMachinesUC  *machine_uc.ListMachinesUseCase
	getMachineUC    *machine_uc.GetMachineUseCase

	createTypeUC     *machine_uc.CreateMachineTypeUseCase
	listTypesUC      *machine_uc.ListMachineTypesUseCase
	getMachineTypeUC *machine_uc.GetMachineTypeUseCase

	createItemTimeUC *machine_uc.CreateItemMachineTimeUseCase
	listItemTimesUC  *machine_uc.ListItemMachineTimesUseCase
	//getItemTimeUC    *machine_uc.GetItemMachineTimeUseCase

	scheduleUC *machine_uc.ScheduleMachineUseCase
}

func (h *MachineHandler) CreateType(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.CreateMachineTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	result, err := h.createTypeUC.Execute(r.Context(), dto, "system")
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MachineHandler) ListTypes(w http.ResponseWriter, r *http.Request) {
	results, err := h.listTypesUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineHandler) CreateMachine(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.CreateMachineDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	result, err := h.createMachineUC.Execute(r.Context(), dto, "system")
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MachineHandler) ListMachines(w http.ResponseWriter, r *http.Request) {
	results, err := h.listMachinesUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineHandler) CreateItemTime(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.CreateItemMachineTimeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	result, err := h.createItemTimeUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MachineHandler) ListItemTimes(w http.ResponseWriter, r *http.Request) {
	itemCodeStr := chi.URLParam(r, "item_code")

	itemCode, err := strconv.ParseInt(itemCodeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item_code")
		return
	}

	results, err := h.listItemTimesUC.Execute(r.Context(), itemCode)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.CreateMachineScheduleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	result, err := h.scheduleUC.CreateSchedule(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MachineHandler) ReorderSchedule(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.ReorderScheduleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if err := h.scheduleUC.ReorderSchedule(r.Context(), dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "schedule reordered",
	})
}

func (h *MachineHandler) GetTypeByCode(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	result, err := h.listTypesUC.GetByCodeType(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) GetMachineByCode(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	result, err := h.listMachinesUC.GetByCodeMachine(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) GetItemTime(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	result, err := h.createItemTimeUC.GetByCodeTime(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	result, err := h.scheduleUC.GetSchedule(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	machineCode, err := strconv.ParseInt(
		chi.URLParam(r, "machine_code"),
		10,
		64,
	)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid machine_code")
		return
	}

	dateStr := r.URL.Query().Get("date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "invalid date format", http.StatusBadRequest)
		return
	}

	results, err := h.scheduleUC.ListSchedules(
		r.Context(),
		machineCode,
		date,
	)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineHandler) UpdateScheduleStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	var dto request.UpdateScheduleStatusDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	result, err := h.scheduleUC.UpdateStatus(
		r.Context(),
		code,
		dto,
	)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) UpdateScheduleTimes(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	var dto request.UpdateScheduleTimesDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	result, err := h.scheduleUC.UpdateTimes(
		r.Context(),
		code,
		dto,
	)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}

	if err := h.scheduleUC.DeleteSchedule(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	security.RespondJSON(w, http.StatusOK, map[string]any{
		"status": "success",
	})
}
