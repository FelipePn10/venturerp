package aps_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	apsentity "github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
)

func mkSeq(id, order int64, pos int, wc int64, start, end time.Time) *apsentity.ProductionSequence {
	return &apsentity.ProductionSequence{
		ID: id, ProductionOrderID: order, SequencePosition: pos, WorkCenterID: wc,
		ScheduledStart: start, ScheduledEnd: end, Status: apsentity.StatusScheduled,
	}
}

func boolPtr(b bool) *bool { return &b }

// Moving op-1 forward must push op-2 (its implicit successor) so it never starts
// before op-1 finishes, preserving each op's wall-clock duration.
func TestReschedule_CascadeShiftsSuccessor(t *testing.T) {
	repo := &fakeAPSRepo{seqs: map[int64]*apsentity.ProductionSequence{
		1: mkSeq(1, 101, 1, 7, dt(2026, 6, 3, 8), dt(2026, 6, 3, 12)),  // 4h
		2: mkSeq(2, 101, 2, 8, dt(2026, 6, 3, 13), dt(2026, 6, 3, 17)), // 4h
	}}
	res, err := New(repo).RescheduleSequence(context.Background(), request.RescheduleSequenceDTO{
		SequenceID: 1, NewStart: dt(2026, 6, 5, 8),
	})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if !res.Moved.ScheduledStart.Equal(dt(2026, 6, 5, 8)) || !res.Moved.ScheduledEnd.Equal(dt(2026, 6, 5, 12)) {
		t.Errorf("moved bar = %+v, want 05/06 08:00–12:00", res.Moved)
	}
	if !res.CascadeApplied {
		t.Error("cascade should be applied by default")
	}
	if len(res.Shifted) != 1 || res.Shifted[0].SequenceID != 2 {
		t.Fatalf("want op-2 shifted, got %+v", res.Shifted)
	}
	got := res.Shifted[0]
	if !got.ScheduledStart.Equal(dt(2026, 6, 5, 12)) || !got.ScheduledEnd.Equal(dt(2026, 6, 5, 16)) {
		t.Errorf("op-2 = %+v, want 05/06 12:00–16:00 (duration preserved)", got)
	}
}

// With cascade disabled the dragged bar moves alone.
func TestReschedule_NoCascade(t *testing.T) {
	repo := &fakeAPSRepo{seqs: map[int64]*apsentity.ProductionSequence{
		1: mkSeq(1, 101, 1, 7, dt(2026, 6, 3, 8), dt(2026, 6, 3, 12)),
		2: mkSeq(2, 101, 2, 8, dt(2026, 6, 3, 13), dt(2026, 6, 3, 17)),
	}}
	res, err := New(repo).RescheduleSequence(context.Background(), request.RescheduleSequenceDTO{
		SequenceID: 1, NewStart: dt(2026, 6, 5, 8), Cascade: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if res.CascadeApplied {
		t.Error("cascade must be off")
	}
	if len(res.Shifted) != 0 {
		t.Errorf("no successor should move, got %+v", res.Shifted)
	}
}

// Moving a bar onto a work center that is already busy that day must raise a
// non-blocking capacity warning (the move still succeeds).
func TestReschedule_CapacityWarning(t *testing.T) {
	repo := &fakeAPSRepo{
		capacity: map[int64]float64{7: 8},
		seqs: map[int64]*apsentity.ProductionSequence{
			// op-1 of order 101 starts elsewhere in the month, 6h long.
			1: mkSeq(1, 101, 1, 7, dt(2026, 6, 3, 8), dt(2026, 6, 3, 14)),
			// op of another order already booked on CT 7 on 05/06, 6h.
			9: mkSeq(9, 999, 1, 7, dt(2026, 6, 5, 8), dt(2026, 6, 5, 14)),
		},
	}
	res, err := New(repo).RescheduleSequence(context.Background(), request.RescheduleSequenceDTO{
		SequenceID: 1, NewStart: dt(2026, 6, 5, 8), // now two 6h jobs share CT 7 on 05/06
	})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("want 1 capacity warning, got %d: %+v", len(res.Warnings), res.Warnings)
	}
	w := res.Warnings[0]
	if w.WorkCenterID != 7 || w.AvailableHours != 8 || w.ScheduledHours != 12 || w.OverByHours != 4 {
		t.Errorf("warning = %+v, want CT7 12h/8h over 4h on 05/06", w)
	}
	if !w.Date.Equal(dt(2026, 6, 5, 0)) {
		t.Errorf("warning date = %s, want 05/06", w.Date)
	}
}

func TestReschedule_Validation(t *testing.T) {
	uc := New(&fakeAPSRepo{seqs: map[int64]*apsentity.ProductionSequence{}})
	if _, err := uc.RescheduleSequence(context.Background(), request.RescheduleSequenceDTO{NewStart: dt(2026, 6, 5, 8)}); err == nil {
		t.Error("missing sequence_id must be rejected")
	}
	if _, err := uc.RescheduleSequence(context.Background(), request.RescheduleSequenceDTO{SequenceID: 1}); err == nil {
		t.Error("missing new_start must be rejected")
	}
}
