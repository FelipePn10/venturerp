package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

type MachineHandler struct {
	createMachineUC *machine_uc.CreateMachineUseCase
	listMachinesUC  *machine_uc.ListMachinesUseCase
	getMachineUC    *machine_uc.GetMachineUseCase
	updateMachineUC *machine_uc.UpdateMachineUseCase
	deleteMachineUC *machine_uc.DeleteMachineUseCase
	listByTypeUC    *machine_uc.ListMachinesByTypeUseCase

	createTypeUC     *machine_uc.CreateMachineTypeUseCase
	listTypesUC      *machine_uc.ListMachineTypesUseCase
	getMachineTypeUC *machine_uc.GetMachineTypeUseCase
	updateTypeUC     *machine_uc.UpdateMachineTypeUseCase
	deleteTypeUC     *machine_uc.DeleteMachineTypeUseCase

	createItemTimeUC          *machine_uc.CreateItemMachineTimeUseCase
	listItemTimesUC           *machine_uc.ListItemMachineTimesUseCase
	calculateProductionTimeUC *machine_uc.CalculateProductionTimeUseCase
	//getItemTimeUC    *machine_uc.GetItemMachineTimeUseCase

	scheduleUC *machine_uc.ScheduleMachineUseCase
}

func (h *MachineHandler) CreateType(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.CreateMachineTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}

	result, err := h.createTypeUC.Execute(r.Context(), dto, "system")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MachineHandler) ListTypes(w http.ResponseWriter, r *http.Request) {
	results, err := h.listTypesUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineHandler) CreateMachine(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.CreateMachineDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}

	result, err := h.createMachineUC.Execute(r.Context(), dto, "system")
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MachineHandler) ListMachines(w http.ResponseWriter, r *http.Request) {
	results, err := h.listMachinesUC.Execute(r.Context())
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineHandler) CreateItemTime(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.CreateItemMachineTimeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}

	result, err := h.createItemTimeUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MachineHandler) ListItemTimes(w http.ResponseWriter, r *http.Request) {
	// GET /time/list?item_code=TEA452-0 — the filter is a query-string param, not a
	// path segment, so it must be read from the URL query.
	itemCodeStr := r.URL.Query().Get("item_code")
	if itemCodeStr == "" {
		itemCodeStr = chi.URLParam(r, "item_code")
	}

	itemCode := request.TextCode(itemCodeStr)

	results, err := h.listItemTimesUC.Execute(r.Context(), itemCode)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	machineCodeStr := r.URL.Query().Get("machine_code")
	if machineCodeStr != "" {
		machineCode, parseErr := strconv.ParseInt(machineCodeStr, 10, 64)
		if parseErr != nil || machineCode <= 0 {
			security.RespondError(w, http.StatusBadRequest, "machine_code deve ser um código inteiro positivo")
			return
		}
		filtered := results[:0]
		for _, result := range results {
			if result.MachineCode == machineCode {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}
	page, pageSize := 1, 50
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 {
			security.RespondError(w, http.StatusBadRequest, "page deve ser positivo")
			return
		}
		page = parsed
	}
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > 200 {
			security.RespondError(w, http.StatusBadRequest, "page_size deve estar entre 1 e 200")
			return
		}
		pageSize = parsed
	}
	total := len(results)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Page", strconv.Itoa(page))
	w.Header().Set("X-Page-Size", strconv.Itoa(pageSize))
	results = results[start:end]

	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.CreateMachineScheduleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}

	result, err := h.scheduleUC.CreateSchedule(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *MachineHandler) ReorderSchedule(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dto request.ReorderScheduleDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}

	if err := h.scheduleUC.ReorderSchedule(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "fila da máquina reordenada",
	})
}

func (h *MachineHandler) GetTypeByCode(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}

	result, err := h.listTypesUC.GetByCodeType(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) GetMachineByCode(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}

	result, err := h.listMachinesUC.GetByCodeMachine(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) GetItemTime(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}

	result, err := h.createItemTimeUC.GetByCodeTime(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}

	result, err := h.scheduleUC.GetSchedule(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	// GET /schedule/list?machine_code=123&date=2026-06-30 — both filters are
	// query-string params, not path segments.
	machineCodeStr := r.URL.Query().Get("machine_code")
	if machineCodeStr == "" {
		machineCodeStr = chi.URLParam(r, "machine_code")
	}
	machineCode, err := strconv.ParseInt(machineCodeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "informe a máquina na consulta")
		return
	}

	// date defaults to today when omitted, so the board loads without forcing a
	// filter; a malformed date is still rejected.
	date := time.Now()
	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		parsed, perr := datetime.ParseDate(dateStr)
		if !perr {
			security.RespondError(w, http.StatusBadRequest, "formato de data inválido: use ano-mês-dia")
			return
		}
		date = parsed
	}

	results, err := h.scheduleUC.ListSchedules(
		r.Context(),
		machineCode,
		date,
	)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, results)
}

func (h *MachineHandler) UpdateScheduleStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}

	var dto request.UpdateScheduleStatusDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}

	result, err := h.scheduleUC.UpdateStatus(
		r.Context(),
		code,
		dto,
	)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) UpdateScheduleTimes(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}

	var dto request.UpdateScheduleTimesDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}

	result, err := h.scheduleUC.UpdateTimes(
		r.Context(),
		code,
		dto,
	)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) CalculateProductionTime(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var input machine_uc.ProductionTimeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}

	result, err := h.calculateProductionTimeUC.Execute(r.Context(), input)
	if err != nil {
		if errors.Is(err, machine_uc.ErrProductionTimeNotConfigured) {
			security.RespondErrorCode(w, http.StatusUnprocessableEntity, "TEMPO_PRODUCAO_NAO_CADASTRADO", err.Error())
			return
		}
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MachineHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return
	}

	if err := h.scheduleUC.DeleteSchedule(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}

	security.RespondJSON(w, http.StatusOK, map[string]any{
		"status": "success",
	})
}

// UpdateMachine altera o cadastro da máquina. O caso de uso existia desde
// sempre, mas sem handler nem rota: não havia como corrigir nome, capacidade,
// centro de trabalho ou qualquer outro dado depois de cadastrar.
func (h *MachineHandler) UpdateMachine(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código da máquina inválido")
		return
	}

	var dto request.UpdateMachineDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	dto.Code = code

	result, err := h.updateMachineUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// UpdateType altera o cadastro do tipo de máquina.
func (h *MachineHandler) UpdateType(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código do tipo de máquina inválido")
		return
	}

	var dto request.UpdateMachineTypeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, "conteúdo da requisição inválido")
		return
	}
	dto.Code = code

	result, err := h.updateTypeUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

// DeleteMachine inativa a máquina. O caso de uso existia sem rota: um recurso
// cadastrado por engano ficava para sempre nas listas e nos roteiros.
func (h *MachineHandler) DeleteMachine(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código da máquina inválido")
		return
	}
	if err := h.deleteMachineUC.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// DeleteType inativa o tipo de máquina.
func (h *MachineHandler) DeleteType(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código do tipo de máquina inválido")
		return
	}
	if err := h.deleteTypeUC.Execute(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ListMachinesByType lista as máquinas de um tipo — é o que permite ao roteiro
// pedir "uma serra" e ver quais recursos atendem.
func (h *MachineHandler) ListMachinesByType(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.ParseInt(chi.URLParam(r, "code"), 10, 64)
	if err != nil || code <= 0 {
		security.RespondError(w, http.StatusBadRequest, "código do tipo de máquina inválido")
		return
	}
	result, err := h.listByTypeUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}
