package service

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
)

type MachineService interface {
	ScheduleOrder(ctx context.Context, machineCode int64, orderCode int64, scheduleDate time.Time, plannedQty float64) (*entity.MachineSchedule, error)
	ReorderSchedule(ctx context.Context, scheduleCode int64, newSequence int) error
	CalculateProductionTime(ctx context.Context, itemCode int64, machineCode int64, quantity float64) (float64, error)
	GetMachineAvailability(ctx context.Context, machineCode int64, date time.Time) (float64, error)
	MoveOrderPriority(ctx context.Context, scheduleCode int64, overridePriority int) error
}
