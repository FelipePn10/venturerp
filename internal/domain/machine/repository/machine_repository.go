package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
)

type MachineRepository interface {
	CreateType(ctx context.Context, mt *entity.MachineType) (*entity.MachineType, error)
	UpdateType(ctx context.Context, mt *entity.MachineType) (*entity.MachineType, error)
	GetTypeByCode(ctx context.Context, code int64) (*entity.MachineType, error)
	ListTypes(ctx context.Context) ([]*entity.MachineType, error)
	DeleteType(ctx context.Context, code int64) error

	Create(ctx context.Context, m *entity.Machine) (*entity.Machine, error)
	Update(ctx context.Context, m *entity.Machine) (*entity.Machine, error)
	GetByCode(ctx context.Context, code int64) (*entity.Machine, error)
	List(ctx context.Context) ([]*entity.Machine, error)
	ListByType(ctx context.Context, typeID int64) ([]*entity.Machine, error)
	Delete(ctx context.Context, code int64) error

	CreateItemMachineTime(ctx context.Context, imt *entity.ItemMachineTime) (*entity.ItemMachineTime, error)
	//GetItemMachineTime(ctx context.Context, code int64) (*entity.ItemMachineTime, error)
	ListItemMachineTimes(ctx context.Context, itemCode int64) ([]*entity.ItemMachineTime, error)
	ListItemsByMachine(ctx context.Context, machineCone int64) ([]*entity.ItemMachineTime, error)
	//DeleteItemMachineTime(ctx context.Context, code int64) error

	CreateSchedule(ctx context.Context, s *entity.MachineSchedule) (*entity.MachineSchedule, error)
	GetSchedule(ctx context.Context, code int64) (*entity.MachineSchedule, error)
	ListSchedules(ctx context.Context, machineID int64, date time.Time) ([]*entity.MachineSchedule, error)
	ListSchedulesByRange(ctx context.Context, machineID int64, start, end time.Time) ([]*entity.MachineSchedule, error)
	UpdateScheduleSequence(ctx context.Context, code int64, sequence int, priorityOverride *int) (*entity.MachineSchedule, error)
	UpdateScheduleStatus(ctx context.Context, code int64, status string, producedQty float64) (*entity.MachineSchedule, error)
	UpdateScheduleTimes(ctx context.Context, code int64, startTime, endTime *time.Time) (*entity.MachineSchedule, error)
	DeleteSchedule(ctx context.Context, code int64) error
}
