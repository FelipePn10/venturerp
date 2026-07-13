package aps_uc

import (
	"testing"
	"time"

	apsrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
)

// 2024-01-01 is a Monday — used as a deterministic anchor.
var (
	mon = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	fri = time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	sat = time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)
	sun = time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)
)

func TestSkipToWorkday(t *testing.T) {
	if got := skipToWorkday(sat); got.Weekday() != time.Monday {
		t.Errorf("Saturday should skip to Monday, got %v", got.Weekday())
	}
	if got := skipToWorkday(sun); got.Weekday() != time.Monday {
		t.Errorf("Sunday should skip to Monday, got %v", got.Weekday())
	}
	if got := skipToWorkday(mon); !got.Equal(mon) {
		t.Errorf("Monday should stay, got %v", got)
	}
}

func TestAllocateInWindowsSplitsAcrossCalendarIntervals(t *testing.T) {
	windows := []apsrepo.AvailabilityWindow{{Start: mon.Add(7 * time.Hour), End: mon.Add(11 * time.Hour)}, {Start: mon.Add(13 * time.Hour), End: mon.Add(17 * time.Hour)}}
	start, end := allocateInWindows(mon, 6, windows)
	if !start.Equal(mon.Add(7*time.Hour)) || !end.Equal(mon.Add(15*time.Hour)) {
		t.Fatalf("start=%v end=%v", start, end)
	}
}

func TestAllocateInWindowsMergesParallelResourceCalendars(t *testing.T) {
	windows := []apsrepo.AvailabilityWindow{{Start: mon.Add(7 * time.Hour), End: mon.Add(12 * time.Hour)}, {Start: mon.Add(8 * time.Hour), End: mon.Add(17 * time.Hour)}}
	start, end := allocateInWindows(mon, 8, windows)
	if !start.Equal(mon.Add(7*time.Hour)) || !end.Equal(mon.Add(15*time.Hour)) {
		t.Fatalf("start=%v end=%v", start, end)
	}
}
func TestSubtractDowntimesSplitsAvailability(t *testing.T) {
	windows := []apsrepo.AvailabilityWindow{{Start: mon.Add(7 * time.Hour), End: mon.Add(17 * time.Hour)}}
	downs := []apsrepo.AvailabilityWindow{{Start: mon.Add(10 * time.Hour), End: mon.Add(12 * time.Hour)}}
	got := subtractDowntimes(windows, downs)
	if len(got) != 2 || !got[0].End.Equal(mon.Add(10*time.Hour)) || !got[1].Start.Equal(mon.Add(12*time.Hour)) {
		t.Fatalf("windows=%+v", got)
	}
}

func TestAdvanceByWorkHours_PartialDay(t *testing.T) {
	// 4h of 8h/day from Monday 00:00 → +0.5 day = Monday 12:00.
	got := advanceByWorkHours(mon, 4, 8)
	want := mon.Add(12 * time.Hour)
	if !got.Equal(want) {
		t.Errorf("advanceByWorkHours(4h) = %v, want %v", got, want)
	}
}

func TestAdvanceByWorkHours_FullDay(t *testing.T) {
	// 8h of 8h/day from Monday → next day (Tuesday 00:00).
	got := advanceByWorkHours(mon, 8, 8)
	want := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("advanceByWorkHours(8h) = %v, want %v", got, want)
	}
}

func TestAdvanceByWorkHours_SkipsWeekend(t *testing.T) {
	// 16h of 8h/day starting Friday: Fri consumes 8h → Sat; skip to Mon; 8h → Tue.
	got := advanceByWorkHours(fri, 16, 8)
	want := time.Date(2024, 1, 9, 0, 0, 0, 0, time.UTC) // Tuesday
	if !got.Equal(want) {
		t.Errorf("advanceByWorkHours(16h from Fri) = %v, want %v (weekend skipped)", got, want)
	}
}

func TestAdvanceByWorkHours_DefaultAvailable(t *testing.T) {
	// availablePerDay <= 0 defaults to 8 → 8h = one full day.
	got := advanceByWorkHours(mon, 8, 0)
	want := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("default availablePerDay: got %v, want %v", got, want)
	}
}

func TestMaxTime(t *testing.T) {
	if got := maxTime(mon, fri); !got.Equal(fri) {
		t.Errorf("maxTime should return the later time")
	}
	if got := maxTime(fri, mon); !got.Equal(fri) {
		t.Errorf("maxTime should be order-independent")
	}
}
