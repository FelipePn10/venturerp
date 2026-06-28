package aps_uc

import (
	"context"
	"fmt"
	"sort"
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
	orders, err := uc.repo.GetOpenProductionOrders(ctx)
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

	scheduledCount := 0
	for _, order := range orders {
		ops, err := uc.repo.GetOrderOperations(ctx, order.ID)
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

			// Start when both the order's previous op finished AND the CT is free.
			earliest := maxTime(opEndTime, wcNextAvailable[wcID])
			// Skip weekends.
			earliest = skipToWorkday(earliest)

			totalHours := op.SetupHours + op.PlannedHours
			end := advanceByWorkHours(earliest, totalHours, avail)

			seq := &entity.ProductionSequence{
				ProductionOrderID: order.ID,
				OperationID:       &op.ID,
				WorkCenterID:      wcID,
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
			SequencePosition:  s.SequencePosition,
			ScheduledStart:    s.ScheduledStart,
			ScheduledEnd:      s.ScheduledEnd,
			Status:            string(s.Status),
			DurationHours:     dur,
		})
	}
	return out
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
