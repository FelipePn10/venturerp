package mapper

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
)

func ToEmployeeEntity(d request.CreateEmployeeDTO) (*entity.Employee, error) {
	return entity.NewEmployee(
		d.Code,
		d.Name,
		d.Role,
		d.ParticipatesBudget,
		d.TechnicalAssistant,
		d.CreatedBy,
	)
}
