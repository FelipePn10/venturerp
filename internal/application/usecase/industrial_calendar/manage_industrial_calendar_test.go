package industrial_calendar_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	calendarrepo "github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository"
)

type calendarAuth struct{ ports.AuthService }

func (calendarAuth) CanManageIndustrialCalendar(context.Context) bool { return true }

type calendarRangeRepo struct {
	calendarrepo.IndustrialCalendarRepository
	year               int
	month              *int
	weekdays           []int
	created, preserved int
}

func (r *calendarRangeRepo) GenerateRange(_ context.Context, year int, month *int, weekdays []int) (int, int, error) {
	r.year = year
	r.month = month
	r.weekdays = weekdays
	return r.created, r.preserved, nil
}

func TestGenerateYearUsesMondayToFridayAndIsIdempotent(t *testing.T) {
	r := &calendarRangeRepo{created: 360, preserved: 5}
	uc := &ManageCalendarUseCase{Repo: r, Auth: calendarAuth{}}
	got, err := uc.Generate(context.Background(), request.GenerateIndustrialCalendarDTO{Year: 2028})
	if err != nil {
		t.Fatal(err)
	}
	if got.Created != 360 || got.Preserved != 5 || got.Ignored != 1 || got.Month != nil {
		t.Fatalf("resultado inesperado: %+v", got)
	}
	want := []int{1, 2, 3, 4, 5}
	for i, v := range want {
		if r.weekdays[i] != v {
			t.Fatalf("weekdays=%v", r.weekdays)
		}
	}
}

func TestGenerateRejectsInvalidOrRepeatedWeekdays(t *testing.T) {
	uc := &ManageCalendarUseCase{Repo: &calendarRangeRepo{}, Auth: calendarAuth{}}
	for _, days := range [][]int{{-1}, {7}, {1, 1}} {
		if _, err := uc.Generate(context.Background(), request.GenerateIndustrialCalendarDTO{Year: 2026, Weekdays: days}); err == nil {
			t.Fatalf("aceitou weekdays=%v", days)
		}
	}
}
