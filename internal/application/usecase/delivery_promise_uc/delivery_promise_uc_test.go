package delivery_promise_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	calendarentity "github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity"
)

type fakePromiseCalendar struct {
	days map[string]bool
}

func (f fakePromiseCalendar) UpsertDay(ctx context.Context, c *calendarentity.ItemCalendarPromise) (*calendarentity.ItemCalendarPromise, error) {
	return c, nil
}

func (f fakePromiseCalendar) GetDay(ctx context.Context, itemCode int64, mask string, year, month, day int) (*calendarentity.ItemCalendarPromise, error) {
	if f.days == nil {
		return nil, errors.New("not configured")
	}
	key := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
	workday, ok := f.days[key]
	if !ok {
		return nil, errors.New("not configured")
	}
	return &calendarentity.ItemCalendarPromise{
		ItemCode:  itemCode,
		Mask:      mask,
		Year:      year,
		Month:     month,
		Day:       day,
		IsWorkday: workday,
	}, nil
}

func (f fakePromiseCalendar) GetWorkdaysInMonth(ctx context.Context, itemCode int64, mask string, year, month int) ([]*calendarentity.ItemCalendarPromise, error) {
	return nil, nil
}

func (f fakePromiseCalendar) ListMonth(ctx context.Context, itemCode int64, mask string, year, month int) ([]*calendarentity.ItemCalendarPromise, error) {
	return nil, nil
}

func (f fakePromiseCalendar) DeleteDay(ctx context.Context, itemCode int64, mask string, year, month, day int) error {
	return nil
}

func TestAllocateBackwardsSplitsCapacityAndSkipsNonWorkdays(t *testing.T) {
	uc := &DeliveryPromiseUseCase{
		Calendar: fakePromiseCalendar{days: map[string]bool{
			"2026-07-01": true,
			"2026-07-02": false,
			"2026-07-03": true,
		}},
	}

	allocations, err := uc.allocateBackwards(context.Background(), allocationRequest{
		itemCode:  1001,
		mask:      "A",
		tankCode:  7,
		quantity:  12,
		unitPrice: 10,
	}, time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), 5)
	if err != nil {
		t.Fatalf("allocateBackwards returned error: %v", err)
	}

	if len(allocations) != 3 {
		t.Fatalf("expected 3 allocations, got %d", len(allocations))
	}
	wantDates := []string{"2026-06-30", "2026-07-01", "2026-07-03"}
	wantQty := []float64{2, 5, 5}
	for i := range allocations {
		if got := allocations[i].AllocationDate.Format(time.DateOnly); got != wantDates[i] {
			t.Fatalf("allocation %d date = %s, want %s", i, got, wantDates[i])
		}
		if allocations[i].Quantity != wantQty[i] {
			t.Fatalf("allocation %d qty = %.2f, want %.2f", i, allocations[i].Quantity, wantQty[i])
		}
	}
}

func TestIsWorkdayFallsBackToWeekdaysWhenCalendarIsMissing(t *testing.T) {
	uc := &DeliveryPromiseUseCase{Calendar: fakePromiseCalendar{}}

	friday, err := uc.isWorkday(context.Background(), 1001, "", time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("isWorkday friday returned error: %v", err)
	}
	if !friday {
		t.Fatal("friday should be workday by fallback")
	}

	saturday, err := uc.isWorkday(context.Background(), 1001, "", time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("isWorkday saturday returned error: %v", err)
	}
	if saturday {
		t.Fatal("saturday should not be workday by fallback")
	}
}
