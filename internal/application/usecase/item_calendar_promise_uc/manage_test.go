package item_calendar_promise_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	calendarrepo "github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/repository"
)

type itemCalendarAuth struct{ ports.AuthService }

func (itemCalendarAuth) CanManageItemCalendarPromise(context.Context) bool { return true }

type itemCalendarRepo struct {
	calendarrepo.ItemCalendarPromiseRepository
}

func TestItemCalendarRejectsInvalidYearAndMonthBeforeRepository(t *testing.T) {
	uc := &ManageItemCalendarPromiseUseCase{Repo: &itemCalendarRepo{}, Auth: itemCalendarAuth{}}
	for _, tc := range []struct{ year, month int }{{-1, 1}, {2026, -1}, {2026, 13}} {
		if _, err := uc.GetWorkdaysInMonth(context.Background(), 10, "PADRAO", tc.year, tc.month); err == nil {
			t.Fatalf("aceitou ano/mês inválido: %+v", tc)
		}
	}
}
