package aps_uc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
	calendarrepo "github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository"
)

type APSUseCase struct {
	repo repository.APSRepository
	cal  calendarrepo.IndustrialCalendarRepository
}

func New(repo repository.APSRepository) *APSUseCase {
	return &APSUseCase{repo: repo}
}

// WithCalendar injects the industrial calendar so the monthly board can shade
// non-working days from the company's real calendar instead of guessing
// weekends. Optional: without it, the board falls back to Saturday/Sunday.
func (uc *APSUseCase) WithCalendar(cal calendarrepo.IndustrialCalendarRepository) *APSUseCase {
	uc.cal = cal
	return uc
}

// SequenceOrders performs finite-capacity scheduling for all open production orders.
//
// Algorithm (simplified EDD + finite capacity):
//  1. Sort orders by priority ASC, then planned_date ASC (EDD).
//  2. For each order, fetch its route operations in sequence order.
//  3. For each operation: find the earliest available slot at the work center
//     (tracked via a per-work-center clock map), assign it, and advance the clock.
//  4. Upsert all production_sequences.
func (uc *APSUseCase) SequenceOrders(ctx context.Context, dto request.SequenceOrdersDTO) (*response.APSSummaryResponse, error) {
	filter := repository.SequenceFilter{OrderIDs: dto.OrderIDs, MachineIDs: dto.MachineIDs, WorkCenterIDs: dto.WorkCenterIDs, OperationIDs: dto.OperationIDs}
	var orders []repository.OrderRow
	var err error
	selection, selected := uc.repo.(repository.SelectionRepository)
	if selected {
		orders, err = selection.GetSelectedProductionOrders(ctx, filter)
	} else {
		orders, err = uc.repo.GetOpenProductionOrders(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("fetching open orders: %w", err)
	}

	// Sort by priority then planned date.
	sort.Slice(orders, func(i, j int) bool {
		if orders[i].Priority != orders[j].Priority {
			return orders[i].Priority < orders[j].Priority
		}
		return orders[i].PlannedDate.Before(orders[j].PlannedDate)
	})

	// wcNextAvailable tracks when each work center is next free.
	startFrom := dto.StartFrom
	if startFrom.IsZero() {
		startFrom = time.Now().UTC()
	}
	wcNextAvailable := make(map[int64]time.Time)
	machineNextAvailable := make(map[int64]time.Time)

	scheduledCount := 0
	for _, order := range orders {
		var ops []repository.OpRow
		if selected {
			ops, err = selection.GetSelectedOrderOperations(ctx, order.ID, filter)
		} else {
			ops, err = uc.repo.GetOrderOperations(ctx, order.ID)
		}
		if err != nil {
			continue
		}
		if len(ops) == 0 {
			continue
		}

		// Clear previous sequences for this order.
		_ = uc.repo.DeleteByOrder(ctx, order.ID)

		opEndTime := startFrom
		for _, op := range ops {
			if op.WorkCenterID == nil {
				continue
			}
			wcID := *op.WorkCenterID
			avail, _ := uc.repo.GetWorkCenterCapacity(ctx, wcID)
			if avail <= 0 {
				avail = 8
			}

			// Start when both the order's previous op finished AND the selected resource is free.
			earliest := maxTime(opEndTime, wcNextAvailable[wcID])
			// Skip weekends.
			earliest = skipToWorkday(earliest)

			totalHours := op.SetupHours + op.PlannedHours
			end := time.Time{}
			var selectedMachineID *int64
			if selected {
				candidates, candidateErr := selection.ListCandidateMachines(ctx, wcID, dto.MachineIDs)
				if candidateErr != nil {
					return nil, fmt.Errorf("loading machines for work center %d: %w", wcID, candidateErr)
				}
				bestStart, bestEnd := time.Time{}, time.Time{}
				for _, candidate := range candidates {
					candidateStart := maxTime(opEndTime, machineNextAvailable[candidate.ID])
					windows, windowErr := selection.ListAvailabilityWindows(ctx, wcID, []int64{candidate.ID}, candidateStart, candidateStart.AddDate(1, 0, 0))
					if windowErr != nil {
						return nil, fmt.Errorf("loading availability for machine %d: %w", candidate.ID, windowErr)
					}
					downtimes, downtimeErr := selection.ListMachineDowntimeWindows(ctx, candidate.ID, candidateStart, candidateStart.AddDate(1, 0, 0))
					if downtimeErr != nil {
						return nil, fmt.Errorf("loading downtime for machine %d: %w", candidate.ID, downtimeErr)
					}
					windows = subtractDowntimes(windows, downtimes)
					var candidateEnd time.Time
					if len(windows) > 0 {
						candidateStart, candidateEnd = allocateInWindows(candidateStart, totalHours, windows)
					} else {
						capacity := candidate.CapacityHours
						if capacity <= 0 {
							capacity = avail
						}
						candidateEnd = advanceByWorkHours(candidateStart, totalHours, capacity)
					}
					if !candidateEnd.IsZero() && (bestEnd.IsZero() || candidateEnd.Before(bestEnd)) {
						bestStart, bestEnd = candidateStart, candidateEnd
						id := candidate.ID
						selectedMachineID = &id
					}
				}
				if !bestEnd.IsZero() {
					earliest, end = bestStart, bestEnd
					machineNextAvailable[*selectedMachineID] = end
				}
			}
			if end.IsZero() {
				end = advanceByWorkHours(earliest, totalHours, avail)
			}

			seq := &entity.ProductionSequence{
				ProductionOrderID: order.ID,
				OperationID:       &op.ID,
				WorkCenterID:      wcID,
				MachineID:         selectedMachineID,
				SequencePosition:  op.Sequence,
				ScheduledStart:    earliest,
				ScheduledEnd:      end,
				Status:            entity.StatusScheduled,
			}
			if _, err := uc.repo.UpsertSequence(ctx, seq); err != nil {
				return nil, fmt.Errorf("upserting sequence for order %d op %d: %w", order.ID, op.ID, err)
			}
			wcNextAvailable[wcID] = end
			opEndTime = end
			scheduledCount++
		}
	}

	return &response.APSSummaryResponse{
		ScheduledOperations: scheduledCount,
		OrdersProcessed:     len(orders),
	}, nil
}

func (uc *APSUseCase) ExportSequencingEvents(ctx context.Context, dto request.SequenceOrdersDTO) ([]response.SequencingExportRowResponse, error) {
	repo, ok := uc.repo.(repository.SelectionRepository)
	if !ok {
		return nil, fmt.Errorf("sequencing event export is not supported")
	}
	rows, err := repo.ListSequencingEvents(ctx, repository.SequenceFilter{OrderIDs: dto.OrderIDs, MachineIDs: dto.MachineIDs, WorkCenterIDs: dto.WorkCenterIDs, OperationIDs: dto.OperationIDs})
	if err != nil {
		return nil, err
	}
	out := make([]response.SequencingExportRowResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.SequencingExportRowResponse{EventType: row.EventType, ProductionOrderID: row.ProductionOrderID, OrderNumber: row.OrderNumber, MachineID: row.MachineID, WorkCenterID: row.WorkCenterID, OperationID: row.OperationID, EventAt: row.EventAt, Quantity: row.Quantity, Reason: row.Reason})
	}
	return out, nil
}

func (uc *APSUseCase) ListSequencingResources(ctx context.Context) ([]response.SequencingResourceResponse, error) {
	repo, ok := uc.repo.(repository.SelectionRepository)
	if !ok {
		return nil, fmt.Errorf("sequencing resources are not supported")
	}
	rows, err := repo.ListSequencingResources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]response.SequencingResourceResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.SequencingResourceResponse{ID: row.ID, Code: row.Code, Name: row.Name, WorkCenterID: row.WorkCenterID, ResourceGroupID: row.ResourceGroupID, IsActive: row.IsActive})
	}
	return out, nil
}

func (uc *APSUseCase) ViewSequencing(ctx context.Context, dto request.SequencingViewDTO) ([]*response.GanttTaskResponse, error) {
	if dto.ResourceGroupID <= 0 {
		return nil, fmt.Errorf("resource_group_id is required")
	}
	if dto.From.IsZero() || dto.To.IsZero() || !dto.To.After(dto.From) {
		return nil, fmt.Errorf("from and to must define a valid interval")
	}
	unit := strings.ToUpper(strings.TrimSpace(dto.TimeUnit))
	if unit != "" && unit != "HOUR" && unit != "MINUTE" && unit != "HORA" && unit != "MINUTO" {
		return nil, fmt.Errorf("time_unit must be HOUR or MINUTE")
	}
	if dto.RefreshValue < 0 {
		return nil, fmt.Errorf("refresh_value cannot be negative")
	}
	repo, ok := uc.repo.(repository.SelectionRepository)
	if !ok {
		return nil, fmt.Errorf("sequencing view is not supported")
	}
	rows, err := repo.ListSequencingView(ctx, repository.SequencingViewFilter{From: dto.From, To: dto.To, ResourceGroupID: dto.ResourceGroupID, FromOrder: dto.FromOrder, ToOrder: dto.ToOrder, FromMachine: dto.FromMachine, ToMachine: dto.ToMachine, FromWorkCenter: dto.FromWorkCenter, ToWorkCenter: dto.ToWorkCenter, FromPlanner: dto.FromPlanner, ToPlanner: dto.ToPlanner})
	if err != nil {
		return nil, err
	}
	return toGanttSlice(rows), nil
}

func (uc *APSUseCase) GetGanttByOrder(ctx context.Context, orderID int64) ([]*response.GanttTaskResponse, error) {
	seqs, err := uc.repo.ListByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return toGanttSlice(seqs), nil
}

func (uc *APSUseCase) GetGanttByWorkCenter(ctx context.Context, dto request.GanttByWorkCenterDTO) ([]*response.GanttTaskResponse, error) {
	seqs, err := uc.repo.ListByWorkCenter(ctx, dto.WorkCenterID, dto.From, dto.To)
	if err != nil {
		return nil, err
	}
	return toGanttSlice(seqs), nil
}

func toGanttSlice(seqs []*entity.ProductionSequence) []*response.GanttTaskResponse {
	out := make([]*response.GanttTaskResponse, 0, len(seqs))
	for _, s := range seqs {
		dur := s.ScheduledEnd.Sub(s.ScheduledStart).Hours()
		out = append(out, &response.GanttTaskResponse{
			SequenceID:        s.ID,
			ProductionOrderID: s.ProductionOrderID,
			WorkCenterID:      s.WorkCenterID,
			MachineID:         s.MachineID,
			SequencePosition:  s.SequencePosition,
			ScheduledStart:    s.ScheduledStart,
			ScheduledEnd:      s.ScheduledEnd,
			Status:            string(s.Status),
			DurationHours:     dur,
		})
	}
	return out
}

func (uc *APSUseCase) configurationRepo() (repository.ConfigurationRepository, error) {
	repo, ok := uc.repo.(repository.ConfigurationRepository)
	if !ok {
		return nil, fmt.Errorf("sequencing configuration is not supported")
	}
	return repo, nil
}
func (uc *APSUseCase) UpsertResourceGroup(ctx context.Context, dto request.ResourceGroupDTO) (response.ResourceGroupResponse, error) {
	repo, err := uc.configurationRepo()
	if err != nil {
		return response.ResourceGroupResponse{}, err
	}
	dto.Code = strings.TrimSpace(dto.Code)
	dto.Description = strings.TrimSpace(dto.Description)
	if dto.Code == "" || dto.Description == "" {
		return response.ResourceGroupResponse{}, fmt.Errorf("code and description are required")
	}
	v, err := repo.UpsertResourceGroup(ctx, dto.Code, dto.Description)
	return response.ResourceGroupResponse{ID: v.ID, Code: v.Code, Description: v.Description}, err
}
func (uc *APSUseCase) ListResourceGroups(ctx context.Context) ([]response.ResourceGroupResponse, error) {
	repo, err := uc.configurationRepo()
	if err != nil {
		return nil, err
	}
	rows, err := repo.ListResourceGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]response.ResourceGroupResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, response.ResourceGroupResponse{ID: v.ID, Code: v.Code, Description: v.Description})
	}
	return out, nil
}
func (uc *APSUseCase) UpsertMachineCalendar(ctx context.Context, dto request.MachineCalendarDTO) (response.MachineCalendarResponse, error) {
	repo, err := uc.configurationRepo()
	if err != nil {
		return response.MachineCalendarResponse{}, err
	}
	if dto.Code <= 0 || strings.TrimSpace(dto.Description) == "" {
		return response.MachineCalendarResponse{}, fmt.Errorf("code and description are required")
	}
	intervals := make([]repository.MachineCalendarInterval, 0, len(dto.Intervals))
	for _, v := range dto.Intervals {
		if v.Weekday < 0 || v.Weekday > 6 || v.Start == "" || v.End == "" {
			return response.MachineCalendarResponse{}, fmt.Errorf("invalid calendar interval")
		}
		intervals = append(intervals, repository.MachineCalendarInterval{Weekday: v.Weekday, Start: v.Start, End: v.End})
	}
	v, err := repo.UpsertMachineCalendar(ctx, dto.Code, strings.TrimSpace(dto.Description), intervals)
	return calendarResponse(v), err
}
func (uc *APSUseCase) ListMachineCalendars(ctx context.Context) ([]response.MachineCalendarResponse, error) {
	repo, err := uc.configurationRepo()
	if err != nil {
		return nil, err
	}
	rows, err := repo.ListMachineCalendars(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]response.MachineCalendarResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, calendarResponse(v))
	}
	return out, nil
}
func calendarResponse(v repository.MachineCalendar) response.MachineCalendarResponse {
	out := response.MachineCalendarResponse{ID: v.ID, Code: v.Code, Description: v.Description, Intervals: make([]response.MachineCalendarIntervalResponse, 0, len(v.Intervals))}
	for _, i := range v.Intervals {
		out.Intervals = append(out.Intervals, response.MachineCalendarIntervalResponse{Weekday: i.Weekday, Start: i.Start, End: i.End})
	}
	return out
}
func (uc *APSUseCase) UpdateSequencingSettings(ctx context.Context, dto request.SequencingSettingsDTO) error {
	repo, err := uc.configurationRepo()
	if err != nil {
		return err
	}
	return repo.UpdateSequencingSettings(ctx, dto.ListOnlyActiveResources)
}
func (uc *APSUseCase) UpdateWorkCenterSequencing(ctx context.Context, id int64, dto request.WorkCenterSequencingDTO) error {
	repo, err := uc.configurationRepo()
	if err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("invalid work center")
	}
	if dto.MachineCostCenterID != nil && dto.LaborCostCenterID != nil && *dto.MachineCostCenterID == *dto.LaborCostCenterID {
		return fmt.Errorf("labor cost center must differ from machine cost center")
	}
	if strings.TrimSpace(dto.CapacityHours) == "" {
		return fmt.Errorf("capacity_hours is required")
	}
	return repo.UpdateWorkCenterSequencing(ctx, id, dto.MachineCostCenterID, dto.LaborCostCenterID, dto.CapacityHours)
}
func (uc *APSUseCase) UpdateResourceSequencing(ctx context.Context, id int64, dto request.ResourceSequencingDTO) error {
	repo, err := uc.configurationRepo()
	if err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("invalid resource")
	}
	return repo.UpdateResourceSequencing(ctx, id, dto.ResourceGroupID, dto.CalendarID, dto.Location, dto.IsCritical, dto.IsActive)
}
func (uc *APSUseCase) DeleteResourceGroup(ctx context.Context, id int64) error {
	repo, err := uc.configurationRepo()
	if err != nil {
		return err
	}
	return repo.DeleteResourceGroup(ctx, id)
}
func (uc *APSUseCase) DeleteMachineCalendar(ctx context.Context, id int64) error {
	repo, err := uc.configurationRepo()
	if err != nil {
		return err
	}
	return repo.DeleteMachineCalendar(ctx, id)
}
func (uc *APSUseCase) CreateMachineDowntime(ctx context.Context, dto request.MachineDowntimeDTO) (response.MachineDowntimeResponse, error) {
	repo, err := uc.configurationRepo()
	if err != nil {
		return response.MachineDowntimeResponse{}, err
	}
	kind := strings.ToUpper(strings.TrimSpace(dto.DowntimeType))
	if dto.MachineID <= 0 || dto.StartsAt.IsZero() || !dto.EndsAt.After(dto.StartsAt) || strings.TrimSpace(dto.Reason) == "" {
		return response.MachineDowntimeResponse{}, fmt.Errorf("machine, valid interval and reason are required")
	}
	if kind != "PLANNED" && kind != "UNPLANNED" && kind != "MAINTENANCE" {
		return response.MachineDowntimeResponse{}, fmt.Errorf("invalid downtime_type")
	}
	v, err := repo.CreateMachineDowntime(ctx, repository.MachineDowntime{MachineID: dto.MachineID, StartsAt: dto.StartsAt, EndsAt: dto.EndsAt, DowntimeType: kind, Reason: strings.TrimSpace(dto.Reason), MaintenanceOrderID: dto.MaintenanceOrderID})
	return downtimeResponse(v), err
}
func (uc *APSUseCase) ListMachineDowntimes(ctx context.Context, machineID int64, from, to time.Time) ([]response.MachineDowntimeResponse, error) {
	repo, err := uc.configurationRepo()
	if err != nil {
		return nil, err
	}
	if from.IsZero() || !to.After(from) {
		return nil, fmt.Errorf("valid from/to are required")
	}
	rows, err := repo.ListMachineDowntimes(ctx, machineID, from, to)
	if err != nil {
		return nil, err
	}
	out := make([]response.MachineDowntimeResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, downtimeResponse(v))
	}
	return out, nil
}
func (uc *APSUseCase) DeleteMachineDowntime(ctx context.Context, id int64) error {
	repo, err := uc.configurationRepo()
	if err != nil {
		return err
	}
	return repo.DeleteMachineDowntime(ctx, id)
}
func downtimeResponse(v repository.MachineDowntime) response.MachineDowntimeResponse {
	return response.MachineDowntimeResponse{ID: v.ID, MachineID: v.MachineID, StartsAt: v.StartsAt, EndsAt: v.EndsAt, DowntimeType: v.DowntimeType, Reason: v.Reason, MaintenanceOrderID: v.MaintenanceOrderID}
}
func (uc *APSUseCase) UpsertEmployeeSequencingProfile(ctx context.Context, id int64, dto request.EmployeeSequencingProfileDTO) error {
	repo, err := uc.configurationRepo()
	if err != nil {
		return err
	}
	if id <= 0 || strings.TrimSpace(dto.CreditLimit) == "" {
		return fmt.Errorf("employee and credit_limit are required")
	}
	p := repository.EmployeeSequencingProfile{CreditLimit: dto.CreditLimit, ValidUntil: dto.ValidUntil}
	for _, v := range dto.Contacts {
		kind := strings.ToUpper(strings.TrimSpace(v.ContactType))
		if kind != "PHONE" && kind != "EMAIL" {
			return fmt.Errorf("invalid contact_type")
		}
		if strings.TrimSpace(v.Value) == "" {
			return fmt.Errorf("contact value is required")
		}
		p.Contacts = append(p.Contacts, repository.EmployeeContact{ContactType: kind, Value: strings.TrimSpace(v.Value), IsPrimary: v.IsPrimary})
	}
	for _, v := range dto.Functions {
		if strings.TrimSpace(v.FunctionName) == "" {
			return fmt.Errorf("function_name is required")
		}
		p.Functions = append(p.Functions, repository.EmployeeFunction{FunctionName: strings.TrimSpace(v.FunctionName), CostCenterID: v.CostCenterID, IsSupervisor: v.IsSupervisor, IsManager: v.IsManager})
	}
	return repo.UpsertEmployeeSequencingProfile(ctx, id, p)
}
func (uc *APSUseCase) UpsertMachineIndustrialProfile(ctx context.Context, id int64, dto request.MachineIndustrialProfileDTO) error {
	repo, err := uc.configurationRepo()
	if err != nil {
		return err
	}
	unit := strings.ToUpper(strings.TrimSpace(dto.PreparationTimeUnit))
	if id <= 0 || strings.TrimSpace(dto.PreparationTime) == "" || (unit != "MINUTE" && unit != "HOUR") {
		return fmt.Errorf("machine, preparation_time and valid unit are required")
	}
	p := repository.MachineIndustrialProfile{UsageDescription: dto.UsageDescription, AcquiredOn: dto.AcquiredOn, PreparationTime: dto.PreparationTime, PreparationTimeUnit: unit, SupplierCode: dto.SupplierCode, Brand: dto.Brand, IsPreferred: dto.IsPreferred, MaintenanceResponsibleEmployeeID: dto.MaintenanceResponsibleEmployeeID}
	for _, s := range dto.Services {
		stype := strings.ToUpper(s.ServiceType)
		funit := strings.ToUpper(s.FrequencyUnit)
		if s.ServiceCode == "" || s.Description == "" || s.FrequencyValue <= 0 {
			return fmt.Errorf("invalid preventive service")
		}
		service := repository.MachineService{ServiceCode: s.ServiceCode, Description: s.Description, ServiceType: stype, FrequencyValue: s.FrequencyValue, FrequencyUnit: funit, MaxTolerance: s.MaxTolerance, SupplierCode: s.SupplierCode, ImplementedOn: s.ImplementedOn, LastExecutedOn: s.LastExecutedOn, Notes: s.Notes, ResponsibleEmployeeIDs: s.ResponsibleEmployeeIDs}
		for _, i := range s.Items {
			if i.ItemCode <= 0 || i.Quantity == "" {
				return fmt.Errorf("invalid service item")
			}
			service.Items = append(service.Items, repository.ServiceItem{ItemCode: i.ItemCode, Quantity: i.Quantity, Notes: i.Notes})
		}
		p.Services = append(p.Services, service)
	}
	for _, v := range dto.SpecialValues {
		p.SpecialValues = append(p.SpecialValues, repository.SpecialValue{Name: v.Name, ValueType: strings.ToUpper(v.ValueType), TextValue: v.TextValue, NumericValue: v.NumericValue, MaxLength: v.MaxLength})
	}
	return repo.UpsertMachineIndustrialProfile(ctx, id, p)
}
func (uc *APSUseCase) GetEmployeeSequencingProfile(ctx context.Context, id int64) (response.EmployeeSequencingProfileResponse, error) {
	repo, err := uc.configurationRepo()
	if err != nil {
		return response.EmployeeSequencingProfileResponse{}, err
	}
	p, err := repo.GetEmployeeSequencingProfile(ctx, id)
	out := response.EmployeeSequencingProfileResponse{CreditLimit: p.CreditLimit, ValidUntil: p.ValidUntil, Contacts: []response.EmployeeContactResponse{}, Functions: []response.EmployeeFunctionResponse{}}
	for _, v := range p.Contacts {
		out.Contacts = append(out.Contacts, response.EmployeeContactResponse{ID: v.ID, ContactType: v.ContactType, Value: v.Value, IsPrimary: v.IsPrimary})
	}
	for _, v := range p.Functions {
		out.Functions = append(out.Functions, response.EmployeeFunctionResponse{ID: v.ID, FunctionName: v.FunctionName, CostCenterID: v.CostCenterID, IsSupervisor: v.IsSupervisor, IsManager: v.IsManager})
	}
	return out, err
}
func (uc *APSUseCase) GetMachineIndustrialProfile(ctx context.Context, id int64) (response.MachineIndustrialProfileResponse, error) {
	repo, err := uc.configurationRepo()
	if err != nil {
		return response.MachineIndustrialProfileResponse{}, err
	}
	p, err := repo.GetMachineIndustrialProfile(ctx, id)
	out := response.MachineIndustrialProfileResponse{UsageDescription: p.UsageDescription, AcquiredOn: p.AcquiredOn, PreparationTime: p.PreparationTime, PreparationTimeUnit: p.PreparationTimeUnit, SupplierCode: p.SupplierCode, Brand: p.Brand, IsPreferred: p.IsPreferred, MaintenanceResponsibleEmployeeID: p.MaintenanceResponsibleEmployeeID, Services: []response.MachineServiceResponse{}, SpecialValues: []response.SpecialValueResponse{}}
	for _, s := range p.Services {
		sr := response.MachineServiceResponse{ID: s.ID, ServiceCode: s.ServiceCode, Description: s.Description, ServiceType: s.ServiceType, FrequencyValue: s.FrequencyValue, FrequencyUnit: s.FrequencyUnit, MaxTolerance: s.MaxTolerance, SupplierCode: s.SupplierCode, ImplementedOn: s.ImplementedOn, LastExecutedOn: s.LastExecutedOn, Notes: s.Notes, ResponsibleEmployeeIDs: s.ResponsibleEmployeeIDs, Items: []response.ServiceItemResponse{}}
		for _, i := range s.Items {
			sr.Items = append(sr.Items, response.ServiceItemResponse{ID: i.ID, ItemCode: i.ItemCode, Quantity: i.Quantity, Notes: i.Notes})
		}
		out.Services = append(out.Services, sr)
	}
	for _, v := range p.SpecialValues {
		out.SpecialValues = append(out.SpecialValues, response.SpecialValueResponse{FieldID: v.FieldID, Name: v.Name, ValueType: v.ValueType, TextValue: v.TextValue, NumericValue: v.NumericValue, MaxLength: v.MaxLength})
	}
	return out, err
}

func validateContact(v request.EmployeeContactDTO) (repository.EmployeeContact, error) {
	kind := strings.ToUpper(strings.TrimSpace(v.ContactType))
	if (kind != "PHONE" && kind != "EMAIL") || strings.TrimSpace(v.Value) == "" {
		return repository.EmployeeContact{}, fmt.Errorf("valid contact_type and value are required")
	}
	return repository.EmployeeContact{ContactType: kind, Value: strings.TrimSpace(v.Value), IsPrimary: v.IsPrimary}, nil
}
func (uc *APSUseCase) UpdateEmployeeContact(ctx context.Context, employeeID, contactID int64, dto request.EmployeeContactDTO) error {
	v, err := validateContact(dto)
	if err != nil || employeeID <= 0 || contactID <= 0 {
		if err != nil {
			return err
		}
		return fmt.Errorf("valid employee and contact ids are required")
	}
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.UpdateEmployeeContact(ctx, employeeID, contactID, v)
}
func (uc *APSUseCase) DeleteEmployeeContact(ctx context.Context, employeeID, contactID int64) error {
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.DeleteEmployeeContact(ctx, employeeID, contactID)
}
func validateFunction(v request.EmployeeFunctionDTO) (repository.EmployeeFunction, error) {
	name := strings.TrimSpace(v.FunctionName)
	if name == "" {
		return repository.EmployeeFunction{}, fmt.Errorf("function_name is required")
	}
	return repository.EmployeeFunction{FunctionName: name, CostCenterID: v.CostCenterID, IsSupervisor: v.IsSupervisor, IsManager: v.IsManager}, nil
}
func (uc *APSUseCase) UpdateEmployeeFunction(ctx context.Context, employeeID, functionID int64, dto request.EmployeeFunctionDTO) error {
	v, e := validateFunction(dto)
	if e != nil {
		return e
	}
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.UpdateEmployeeFunction(ctx, employeeID, functionID, v)
}
func (uc *APSUseCase) DeleteEmployeeFunction(ctx context.Context, employeeID, functionID int64) error {
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.DeleteEmployeeFunction(ctx, employeeID, functionID)
}
func validateService(s request.MachineServiceDTO) (repository.MachineService, error) {
	st := strings.ToUpper(strings.TrimSpace(s.ServiceType))
	fu := strings.ToUpper(strings.TrimSpace(s.FrequencyUnit))
	if strings.TrimSpace(s.ServiceCode) == "" || strings.TrimSpace(s.Description) == "" || (st != "ELECTRICAL" && st != "MECHANICAL" && st != "BOTH") || s.FrequencyValue <= 0 || (fu != "DAY" && fu != "WEEK" && fu != "MONTH" && fu != "YEAR" && fu != "UNIT") || s.ImplementedOn.IsZero() {
		return repository.MachineService{}, fmt.Errorf("invalid preventive service")
	}
	return repository.MachineService{ServiceCode: strings.TrimSpace(s.ServiceCode), Description: strings.TrimSpace(s.Description), ServiceType: st, FrequencyValue: s.FrequencyValue, FrequencyUnit: fu, MaxTolerance: s.MaxTolerance, SupplierCode: s.SupplierCode, ImplementedOn: s.ImplementedOn, LastExecutedOn: s.LastExecutedOn, Notes: s.Notes, ResponsibleEmployeeIDs: s.ResponsibleEmployeeIDs}, nil
}
func (uc *APSUseCase) UpdateMachineService(ctx context.Context, machineID, serviceID int64, dto request.MachineServiceDTO) error {
	v, e := validateService(dto)
	if e != nil {
		return e
	}
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.UpdateMachineService(ctx, machineID, serviceID, v)
}
func (uc *APSUseCase) DeleteMachineService(ctx context.Context, machineID, serviceID int64) error {
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.DeleteMachineService(ctx, machineID, serviceID)
}
func validateServiceItem(i request.ServiceItemDTO) (repository.ServiceItem, error) {
	if i.ItemCode <= 0 || strings.TrimSpace(i.Quantity) == "" {
		return repository.ServiceItem{}, fmt.Errorf("item_code and quantity are required")
	}
	return repository.ServiceItem{ItemCode: i.ItemCode, Quantity: i.Quantity, Notes: i.Notes}, nil
}
func (uc *APSUseCase) UpdateMachineServiceItem(ctx context.Context, machineID, serviceID, itemID int64, dto request.ServiceItemDTO) error {
	v, e := validateServiceItem(dto)
	if e != nil {
		return e
	}
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.UpdateMachineServiceItem(ctx, machineID, serviceID, itemID, v)
}
func (uc *APSUseCase) DeleteMachineServiceItem(ctx context.Context, machineID, serviceID, itemID int64) error {
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.DeleteMachineServiceItem(ctx, machineID, serviceID, itemID)
}
func validateSpecialValue(v request.SpecialValueDTO) (repository.SpecialValue, error) {
	typ := strings.ToUpper(strings.TrimSpace(v.ValueType))
	name := strings.TrimSpace(v.Name)
	if name == "" || (typ != "TEXT" && typ != "NUMBER") || (typ == "TEXT" && v.TextValue == "") || (typ == "NUMBER" && v.NumericValue == "") || (typ == "TEXT" && v.NumericValue != "") || (typ == "NUMBER" && v.TextValue != "") {
		return repository.SpecialValue{}, fmt.Errorf("special field requires name, valid type and exactly one matching value")
	}
	return repository.SpecialValue{Name: name, ValueType: typ, TextValue: v.TextValue, NumericValue: v.NumericValue, MaxLength: v.MaxLength}, nil
}
func (uc *APSUseCase) UpdateMachineSpecialValue(ctx context.Context, machineID, fieldID int64, dto request.SpecialValueDTO) error {
	v, e := validateSpecialValue(dto)
	if e != nil {
		return e
	}
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.UpdateMachineSpecialValue(ctx, machineID, fieldID, v)
}
func (uc *APSUseCase) DeleteMachineSpecialValue(ctx context.Context, machineID, fieldID int64) error {
	repo, e := uc.configurationRepo()
	if e != nil {
		return e
	}
	return repo.DeleteMachineSpecialValue(ctx, machineID, fieldID)
}

// ─── scheduling helpers ───────────────────────────────────────────────────────

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func skipToWorkday(t time.Time) time.Time {
	for t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		t = t.Add(24 * time.Hour)
	}
	return t
}

// advanceByWorkHours advances time by workHours assuming availableHoursPerDay per workday.
func advanceByWorkHours(start time.Time, workHours, availablePerDay float64) time.Time {
	if availablePerDay <= 0 {
		availablePerDay = 8
	}
	t := start
	remaining := workHours
	for remaining > 0 {
		t = skipToWorkday(t)
		dayRemain := availablePerDay
		if remaining <= dayRemain {
			fraction := remaining / availablePerDay
			t = t.Add(time.Duration(fraction * float64(24*time.Hour)))
			remaining = 0
		} else {
			remaining -= dayRemain
			t = t.Add(24 * time.Hour)
		}
	}
	return t
}

func allocateInWindows(earliest time.Time, hours float64, windows []repository.AvailabilityWindow) (time.Time, time.Time) {
	merged := make([]repository.AvailabilityWindow, 0, len(windows))
	for _, window := range windows {
		if len(merged) > 0 && !window.Start.After(merged[len(merged)-1].End) {
			if window.End.After(merged[len(merged)-1].End) {
				merged[len(merged)-1].End = window.End
			}
			continue
		}
		merged = append(merged, window)
	}
	remaining := time.Duration(hours * float64(time.Hour))
	var actualStart time.Time
	for _, w := range merged {
		start := maxTime(earliest, w.Start)
		if !start.Before(w.End) {
			continue
		}
		if actualStart.IsZero() {
			actualStart = start
		}
		available := w.End.Sub(start)
		if remaining <= available {
			return actualStart, start.Add(remaining)
		}
		remaining -= available
	}
	return time.Time{}, time.Time{}
}

func subtractDowntimes(windows, downtimes []repository.AvailabilityWindow) []repository.AvailabilityWindow {
	result := append([]repository.AvailabilityWindow(nil), windows...)
	for _, down := range downtimes {
		next := make([]repository.AvailabilityWindow, 0, len(result)+1)
		for _, window := range result {
			if !down.Start.Before(window.End) || !down.End.After(window.Start) {
				next = append(next, window)
				continue
			}
			if down.Start.After(window.Start) {
				next = append(next, repository.AvailabilityWindow{Start: window.Start, End: minTime(down.Start, window.End)})
			}
			if down.End.Before(window.End) {
				next = append(next, repository.AvailabilityWindow{Start: maxTime(down.End, window.Start), End: window.End})
			}
		}
		result = next
	}
	return result
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
