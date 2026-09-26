// Package employeeresolution traduz o CÓDIGO do funcionário — o número que a
// pessoa vê na tela e no crachá — para a chave interna que as tabelas de
// produção usam.
//
// O banco usa as duas convenções ao mesmo tempo:
//
//	planned_orders.employee_code            → employees.code
//	production_orders.employee_id           → employees.id
//	production_appointments.employee_id     → employees.id
//
// A tela manda sempre o CÓDIGO (é o que o cadastro mostra e o que a busca de
// funcionário devolve). Numa base onde código e id coincidem ninguém percebe;
// na base da Tecnofer os códigos vão de 1 a 28 e os ids de 31 a 58, e o
// apontamento de produção respondia "um dos vínculos informados não existe na
// empresa autenticada" para QUALQUER operador — o chão de fábrica não conseguia
// apontar.
package employeeresolution

import (
	"context"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/employee/entity"
)

// Finder é o mínimo que a resolução precisa do repositório de funcionários.
type Finder interface {
	GetByCode(ctx context.Context, code int64) (*entity.Employee, error)
	List(ctx context.Context) ([]*entity.Employee, error)
}

// ResolveID devolve `employees.id` a partir do que veio da tela.
//
// Tenta o CÓDIGO primeiro, porque é o contrato da interface; se não existir
// funcionário com aquele código, aceita o valor como id (cliente antigo ou
// integração que já mandava a chave interna). Sem funcionário nenhum, devolve
// erro de validação com o número que foi tentado — não um erro de chave
// estrangeira do banco.
func ResolveID(ctx context.Context, finder Finder, codeOrID int64) (int64, error) {
	if finder == nil {
		// Sem repositório configurado o valor segue como está: é o
		// comportamento anterior, e quebrar aqui pararia fluxos que já rodam.
		return codeOrID, nil
	}
	if codeOrID <= 0 {
		return 0, errorsuc.NewValidationError("informe o funcionário")
	}
	if employee, err := finder.GetByCode(ctx, codeOrID); err == nil && employee != nil {
		return employee.ID, nil
	}
	todos, err := finder.List(ctx)
	if err != nil {
		return 0, err
	}
	for _, employee := range todos {
		if employee != nil && employee.ID == codeOrID {
			return employee.ID, nil
		}
	}
	return 0, errorsuc.NewValidationError(fmt.Sprintf(
		"o funcionário %d não existe nesta empresa; confira o código no cadastro de funcionários", codeOrID))
}

// ResolveOptionalID aplica ResolveID quando o ponteiro vem preenchido.
func ResolveOptionalID(ctx context.Context, finder Finder, codeOrID *int64) (*int64, error) {
	if codeOrID == nil || *codeOrID == 0 {
		return nil, nil
	}
	id, err := ResolveID(ctx, finder, *codeOrID)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
