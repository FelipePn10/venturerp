package delivery_promise_uc

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/representativevalidation"
	dpentity "github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity"
	dprepo "github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/repository"
	rescheduleentity "github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	reschedulerepo "github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository"
	calendarrepo "github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/repository"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DeliveryPromiseUseCase struct {
	Reservations    dprepo.TankReservationRepository
	Reschedules     reschedulerepo.DeliveryRescheduleRepository
	Orders          orderrepo.SalesOrderRepository
	Items           itemrepo.ItemRepository
	Stock           stockrepo.StockRepository
	Calendar        calendarrepo.ItemCalendarPromiseRepository
	Auth            ports.AuthService
	Representatives representativevalidation.Repository
	// Linhas guarda a capacidade de cada tanque / linha. Sem ela (ou sem a linha
	// do tanque), vale a capacidade informada na consulta.
	Linhas dprepo.LinhaDeCapacidadeRepository
}

// capacidades devolve a capacidade diária de cada tanque, com a da consulta como
// reserva para tanque sem cadastro.
func (uc *DeliveryPromiseUseCase) capacidades(ctx context.Context, padrao decimal.Decimal) (func(int64) decimal.Decimal, error) {
	porTanque := map[int64]decimal.Decimal{}
	if uc.Linhas != nil {
		linhas, err := uc.Linhas.Linhas(ctx)
		if err != nil {
			return nil, err
		}
		for _, l := range linhas {
			if l.Ativa {
				porTanque[l.Codigo] = l.CapacidadeDiaria
			}
		}
	}
	return func(tanque int64) decimal.Decimal {
		if c, ok := porTanque[tanque]; ok {
			return c
		}
		return padrao
	}, nil
}

// Linhas lista os tanques / linhas de capacidade.
func (uc *DeliveryPromiseUseCase) ListarLinhas(ctx context.Context) ([]dpentity.LinhaDeCapacidade, error) {
	if uc.Linhas == nil {
		return []dpentity.LinhaDeCapacidade{}, nil
	}
	return uc.Linhas.Linhas(ctx)
}

// SalvarLinha cria, altera ou desativa um tanque / linha de capacidade.
func (uc *DeliveryPromiseUseCase) SalvarLinha(ctx context.Context, l dpentity.LinhaDeCapacidade) error {
	if !uc.Auth.CanManageDeliveryPromise(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if err := l.Conferir(); err != nil {
		return err
	}
	return uc.Linhas.SalvarLinha(ctx, l)
}

type allocationRequest struct {
	itemCode  int64
	mask      string
	tankCode  int64
	quantity  decimal.Decimal
	unitPrice decimal.Decimal
}

func (uc *DeliveryPromiseUseCase) Occupation(ctx context.Context, dto request.DeliveryPromiseOccupationDTO) ([]response.DeliveryPromiseOccupationDayResponse, error) {
	if !uc.Auth.CanManageDeliveryPromise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	from, err := parseDate(dto.FromDate)
	if err != nil {
		return nil, err
	}
	to, err := parseDate(dto.ToDate)
	if err != nil {
		return nil, err
	}
	if to.Before(from) {
		return nil, errorsuc.NewValidationError("a data final deve ser maior ou igual à data inicial")
	}

	capacidade, err := uc.capacidades(ctx, dto.DailyCapacity)
	if err != nil {
		return nil, err
	}
	days := map[string]*dpentity.TankOccupationDay{}
	orders, err := uc.Orders.ListAdvanced(ctx, orderrepo.SalesOrderFilter{DeliveryFrom: &from, DeliveryTo: &to})
	if err != nil {
		return nil, err
	}
	for _, order := range orders {
		if order.Status == orderentity.SalesOrderStatusCancelled || order.Status == orderentity.SalesOrderStatusInvoiced {
			continue
		}
		items, err := uc.Orders.ListItems(ctx, order.Code)
		if err != nil {
			return nil, err
		}
		for _, line := range items {
			lineDate := effectiveLineDate(order, line)
			if lineDate == nil || lineDate.Before(from) || lineDate.After(to) {
				continue
			}
			tankCode, warn := uc.tankCode(ctx, line.ItemCode)
			if !tankAllowed(tankCode, dto.TankCodes) {
				continue
			}
			alloc := dpentity.TankAllocation{
				TankCode:       tankCode,
				ItemCode:       line.ItemCode,
				Mask:           line.Mask,
				AllocationDate: truncateDate(*lineDate),
				Quantity:       decimal.Max(decimal.Zero, line.RequestedQty.Sub(line.AttendedQty).Sub(line.CancelledQty)),
				UnitPrice:      line.UnitPrice,
				Source:         "SALES_ORDER",
				ReferenceCode:  &order.Code,
			}
			day := ensureDay(days, tankCode, alloc.AllocationDate, capacidade(tankCode))
			if warn != "" {
				day.Warnings = append(day.Warnings, warn)
			}
			addAllocation(day, alloc)
		}
	}

	reservations, err := uc.Reservations.ListActiveReservations(ctx, from, to, dto.TankCodes)
	if err != nil {
		return nil, err
	}
	for _, reservation := range reservations {
		alloc := dpentity.TankAllocation{
			TankCode:       reservation.TankCode,
			ItemCode:       reservation.ItemCode,
			Mask:           reservation.Mask,
			AllocationDate: truncateDate(reservation.AllocationDate),
			Quantity:       reservation.ReservedQty,
			Source:         "TANK_RESERVATION",
			ReferenceCode:  &reservation.Code,
		}
		addAllocation(ensureDay(days, alloc.TankCode, alloc.AllocationDate, capacidade(alloc.TankCode)), alloc)
	}

	return occupationMapToResponse(days), nil
}

func (uc *DeliveryPromiseUseCase) ReserveTank(ctx context.Context, dto request.DeliveryTankReservationDTO) (*response.DeliveryTankReservationResponse, error) {
	if !uc.Auth.CanManageDeliveryPromise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	capacidade, err := uc.capacidades(ctx, dto.DailyCapacity)
	if err != nil {
		return nil, err
	}
	// A capacidade vem do tanque cadastrado; a da requisição só cobre tanque sem cadastro.
	for _, line := range dto.Lines {
		if tanque, _ := uc.tankCode(ctx, line.ItemCode); !capacidade(tanque).IsPositive() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o tanque %d do item %d não tem capacidade cadastrada: cadastre a linha nos Parâmetros da VPME0102ITE ou informe a capacidade diária", tanque, line.ItemCode))
		}
	}
	createdBy, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	requestedDate, err := parseDate(dto.RequestedDeliveryDate)
	if err != nil {
		return nil, err
	}
	expiresAt := truncateDate(time.Now().AddDate(0, 0, maxInt(dto.FirmDays, 1)))
	warnings := []string{}
	planned := []dpentity.TankAllocation{}

	for _, line := range dto.Lines {
		if !line.Quantity.IsPositive() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a quantidade do item %d deve ser maior que zero", line.ItemCode))
		}
		tankCode, warn := uc.tankCode(ctx, line.ItemCode)
		if warn != "" {
			warnings = append(warnings, warn)
		}
		qty := line.Quantity
		if dto.VerifyStock {
			available, err := uc.availableStock(ctx, line.ItemCode, line.Mask)
			if err != nil {
				return nil, err
			}
			covered := decimal.Min(qty, available)
			qty = qty.Sub(covered)
			if covered.IsPositive() {
				warnings = append(warnings, fmt.Sprintf("item %d: %s covered by ATP stock", line.ItemCode, covered.StringFixed(4)))
			}
		}
		if !qty.IsPositive() {
			continue
		}
		allocs, err := uc.allocateBackwards(ctx, allocationRequest{
			itemCode:  line.ItemCode,
			mask:      line.Mask,
			tankCode:  tankCode,
			quantity:  qty,
			unitPrice: line.UnitPrice,
		}, requestedDate, capacidade(tankCode))
		if err != nil {
			return nil, err
		}
		planned = append(planned, allocs...)
	}

	if dto.Commit {
		for i := range planned {
			code, err := uc.Reservations.NextReservationCode(ctx)
			if err != nil {
				return nil, err
			}
			_, err = uc.Reservations.CreateReservation(ctx, &dpentity.TankReservation{
				Code:           code,
				CustomerCode:   dto.CustomerCode,
				ItemCode:       planned[i].ItemCode,
				Mask:           planned[i].Mask,
				TankCode:       planned[i].TankCode,
				RequestedQty:   planned[i].Quantity,
				ReservedQty:    planned[i].Quantity,
				AllocationDate: planned[i].AllocationDate,
				ExpiresAt:      expiresAt,
				Status:         dpentity.TankReservationActive,
				Notes:          dto.Notes,
				CreatedBy:      createdBy,
			})
			if err != nil {
				return nil, err
			}
			planned[i].Source = "TANK_RESERVATION"
			planned[i].ReferenceCode = &code
		}
	}

	return &response.DeliveryTankReservationResponse{
		RequestedDeliveryDate: requestedDate,
		ExpiresAt:             expiresAt,
		Committed:             dto.Commit,
		Allocations:           allocationsToResponse(planned),
		Warnings:              warnings,
	}, nil
}

func (uc *DeliveryPromiseUseCase) Reschedule(ctx context.Context, dto request.DeliveryRescheduleBatchDTO) (*response.DeliveryRescheduleBatchResponse, error) {
	if !uc.Auth.CanManageDeliveryPromise(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := representativevalidation.Validate(ctx, uc.Representatives, dto.RepresentativeCode); err != nil {
		return nil, err
	}
	createdBy, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	from, err := parseDate(dto.DeliveryFrom)
	if err != nil {
		return nil, err
	}
	to, err := parseDate(dto.DeliveryTo)
	if err != nil {
		return nil, err
	}
	newDate, err := parseDate(dto.NewDate)
	if err != nil {
		return nil, err
	}

	orders, err := uc.loadRescheduleOrders(ctx, from, to, dto)
	if err != nil {
		return nil, err
	}
	out := &response.DeliveryRescheduleBatchResponse{}
	for _, order := range orders {
		items, err := uc.Orders.ListItems(ctx, order.Code)
		if err != nil {
			return nil, err
		}
		changedItems := false
		maxDate := order.DeliveryDate
		for _, line := range items {
			if !matchesInt64(line.ItemCode, dto.ItemCodes) {
				continue
			}
			if line.DeliveryDateFirm {
				out.Skipped = append(out.Skipped, fmt.Sprintf("order %d item %d has firm delivery date", order.Code, line.ItemCode))
				continue
			}
			oldDate := effectiveLineDate(order, line)
			line.DeliveryDate = &newDate
			line.DeliveryDateFirm = true
			if _, err := uc.Orders.UpdateItem(ctx, line); err != nil {
				return nil, err
			}
			out.UpdatedItems++
			changedItems = true
			if oldDate != nil {
				if err := uc.registerReschedule(ctx, order.Code, line.ItemCode, *oldDate, newDate, dto.Reason, createdBy); err != nil {
					return nil, err
				}
			}
			if maxDate == nil || newDate.After(*maxDate) {
				maxDate = &newDate
			}
		}
		if order.DeliveryDateFirm {
			out.Skipped = append(out.Skipped, fmt.Sprintf("order %d has firm delivery date", order.Code))
			continue
		}
		if changedItems || len(dto.ItemCodes) == 0 {
			order.DeliveryDate = maxDate
			if len(dto.ItemCodes) == 0 {
				order.DeliveryDate = &newDate
				order.DeliveryDateFirm = true
			}
			if _, err := uc.Orders.Update(ctx, order); err != nil {
				return nil, err
			}
			out.UpdatedOrders++
		}
	}
	return out, nil
}

func (uc *DeliveryPromiseUseCase) registerReschedule(ctx context.Context, orderCode, itemCode int64, oldDate, newDate time.Time, reason *string, createdBy uuid.UUID) error {
	if uc.Reschedules == nil {
		return nil
	}
	_, err := uc.Reschedules.Create(ctx, &rescheduleentity.DeliveryReschedule{
		SalesOrderCode: orderCode,
		ItemCode:       valueobject.ItemCode(itemCode),
		OldDate:        oldDate,
		NewDate:        newDate,
		Reason:         reason,
		CreatedBy:      createdBy,
	})
	return err
}

func (uc *DeliveryPromiseUseCase) CancelReservation(ctx context.Context, code int64) error {
	if !uc.Auth.CanManageDeliveryPromise(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Reservations.CancelReservation(ctx, code)
}

func (uc *DeliveryPromiseUseCase) ExpireReservations(ctx context.Context, now time.Time) (int64, error) {
	if !uc.Auth.CanManageDeliveryPromise(ctx) {
		return 0, errorsuc.ErrUnauthorized
	}
	return uc.Reservations.ExpireReservations(ctx, now)
}

func (uc *DeliveryPromiseUseCase) allocateBackwards(ctx context.Context, req allocationRequest, requestedDate time.Time, dailyCapacity decimal.Decimal) ([]dpentity.TankAllocation, error) {
	remaining := req.quantity
	date := truncateDate(requestedDate)
	out := []dpentity.TankAllocation{}
	for remaining.IsPositive() {
		workday, err := uc.isWorkday(ctx, req.itemCode, req.mask, date)
		if err != nil {
			return nil, err
		}
		if workday {
			qty := decimal.Min(remaining, dailyCapacity)
			out = append(out, dpentity.TankAllocation{
				TankCode:       req.tankCode,
				ItemCode:       req.itemCode,
				Mask:           req.mask,
				AllocationDate: date,
				Quantity:       qty,
				UnitPrice:      req.unitPrice,
				Source:         "SIMULATION",
			})
			remaining = remaining.Sub(qty)
		}
		date = date.AddDate(0, 0, -1)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AllocationDate.Before(out[j].AllocationDate) })
	return out, nil
}

func (uc *DeliveryPromiseUseCase) isWorkday(ctx context.Context, itemCode int64, mask string, date time.Time) (bool, error) {
	day, err := uc.Calendar.GetDay(ctx, itemCode, mask, date.Year(), int(date.Month()), date.Day())
	if err == nil {
		return day.IsWorkday, nil
	}
	return date.Weekday() != time.Saturday && date.Weekday() != time.Sunday, nil
}

func (uc *DeliveryPromiseUseCase) tankCode(ctx context.Context, itemCode int64) (int64, string) {
	item, err := uc.Items.FindItemByCode(ctx, valueobject.ItemCode(itemCode))
	if err != nil || item == nil || item.Planning.TankCode == nil {
		return 0, fmt.Sprintf("item %d has no planning tank; using tank 0", itemCode)
	}
	return int64(*item.Planning.TankCode), ""
}

func (uc *DeliveryPromiseUseCase) availableStock(ctx context.Context, itemCode int64, mask string) (decimal.Decimal, error) {
	balances, err := uc.Stock.ListBalancesByItem(ctx, itemCode)
	if err != nil {
		return decimal.Zero, err
	}
	total := decimal.Zero
	for _, balance := range balances {
		if mask != "" && balance.Mask != mask {
			continue
		}
		total = total.Add(balance.AvailableQty)
	}
	return total, nil
}

func (uc *DeliveryPromiseUseCase) loadRescheduleOrders(ctx context.Context, from, to time.Time, dto request.DeliveryRescheduleBatchDTO) ([]*orderentity.SalesOrder, error) {
	if len(dto.SalesOrderCodes) > 0 {
		out := make([]*orderentity.SalesOrder, 0, len(dto.SalesOrderCodes))
		for _, code := range dto.SalesOrderCodes {
			order, err := uc.Orders.GetByCode(ctx, code)
			if err != nil {
				return nil, err
			}
			out = append(out, order)
		}
		return out, nil
	}
	return uc.Orders.ListAdvanced(ctx, orderrepo.SalesOrderFilter{
		CustomerCode:       dto.CustomerCode,
		RepresentativeCode: dto.RepresentativeCode,
		DeliveryFrom:       &from,
		DeliveryTo:         &to,
	})
}

func parseDate(v string) (time.Time, error) {
	t, err := time.Parse(time.DateOnly, v)
	if err != nil {
		return time.Time{}, errorsuc.NewValidationError("data inválida; use o formato AAAA-MM-DD")
	}
	return truncateDate(t), nil
}

func truncateDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func effectiveLineDate(order *orderentity.SalesOrder, line *orderentity.SalesOrderItem) *time.Time {
	if line.DeliveryDate != nil {
		d := truncateDate(*line.DeliveryDate)
		return &d
	}
	if order.DeliveryDate != nil {
		d := truncateDate(*order.DeliveryDate)
		return &d
	}
	return nil
}

func ensureDay(days map[string]*dpentity.TankOccupationDay, tankCode int64, date time.Time, capacity decimal.Decimal) *dpentity.TankOccupationDay {
	key := fmt.Sprintf("%d:%s", tankCode, date.Format(time.DateOnly))
	if day, ok := days[key]; ok {
		return day
	}
	day := &dpentity.TankOccupationDay{TankCode: tankCode, Date: date, Capacity: capacity, Free: capacity}
	days[key] = day
	return day
}

func addAllocation(day *dpentity.TankOccupationDay, alloc dpentity.TankAllocation) {
	day.Allocations = append(day.Allocations, alloc)
	day.Allocated = day.Allocated.Add(alloc.Quantity)
	day.Quantity = day.Quantity.Add(alloc.Quantity)
	day.ForecastRevenue = day.ForecastRevenue.Add(alloc.Quantity.Mul(alloc.UnitPrice))
	day.Free = day.Capacity.Sub(day.Allocated)
	if day.Capacity.IsPositive() {
		day.OccupationPct = day.Allocated.Div(day.Capacity).Mul(decimal.NewFromInt(100))
	}
}

func occupationMapToResponse(days map[string]*dpentity.TankOccupationDay) []response.DeliveryPromiseOccupationDayResponse {
	list := make([]*dpentity.TankOccupationDay, 0, len(days))
	for _, day := range days {
		if !day.Capacity.IsPositive() {
			day.Warnings = append(day.Warnings, "daily capacity was not informed; free capacity and occupation percentage are indicative")
		}
		list = append(list, day)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Date.Equal(list[j].Date) {
			return list[i].TankCode < list[j].TankCode
		}
		return list[i].Date.Before(list[j].Date)
	})
	out := make([]response.DeliveryPromiseOccupationDayResponse, 0, len(list))
	for _, day := range list {
		out = append(out, response.DeliveryPromiseOccupationDayResponse{
			TankCode:        day.TankCode,
			Date:            day.Date,
			Capacity:        day.Capacity.InexactFloat64(),
			Allocated:       day.Allocated.InexactFloat64(),
			Free:            day.Free.InexactFloat64(),
			OccupationPct:   day.OccupationPct.InexactFloat64(),
			Quantity:        day.Quantity.InexactFloat64(),
			ForecastRevenue: day.ForecastRevenue.InexactFloat64(),
			Allocations:     allocationsToResponse(day.Allocations),
			Warnings:        day.Warnings,
		})
	}
	return out
}

func allocationsToResponse(list []dpentity.TankAllocation) []response.DeliveryPromiseAllocationResponse {
	out := make([]response.DeliveryPromiseAllocationResponse, 0, len(list))
	for _, alloc := range list {
		out = append(out, response.DeliveryPromiseAllocationResponse{
			TankCode:       alloc.TankCode,
			ItemCode:       alloc.ItemCode,
			Mask:           alloc.Mask,
			AllocationDate: alloc.AllocationDate,
			Quantity:       alloc.Quantity.InexactFloat64(),
			UnitPrice:      alloc.UnitPrice.InexactFloat64(),
			Source:         alloc.Source,
			ReferenceCode:  alloc.ReferenceCode,
		})
	}
	return out
}

func tankAllowed(tankCode int64, filter []int64) bool {
	if len(filter) == 0 {
		return true
	}
	return matchesInt64(tankCode, filter)
}

func matchesInt64(v int64, filter []int64) bool {
	if len(filter) == 0 {
		return true
	}
	for _, allowed := range filter {
		if v == allowed {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
