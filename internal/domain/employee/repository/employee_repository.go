package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
)

type EmployeeRepository interface {
	Create(ctx context.Context, e *entity.Employee) (*entity.Employee, error)
	Update(ctx context.Context, e *entity.Employee) (*entity.Employee, error)
	GetByCode(ctx context.Context, code int64) (*entity.Employee, error)
	List(ctx context.Context) ([]*entity.Employee, error)
	ListByRole(ctx context.Context, role string) ([]*entity.Employee, error)
	Deactivate(ctx context.Context, code int64) error
}
