package production_order

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// CadastrosDoPosto devolve o mínimo que o posto precisa para apontar: máquinas,
// operadores e depósitos da empresa, só código e nome. O perfil OPERATOR não lê
// o cadastro completo (funcionário tem dado pessoal), e sem estas listas a
// aba Apontamento da VPRO0900 não deixava escolher máquina nem operador.
func (r *ProductionOrderRepositoryPGX) CadastrosDoPosto(ctx context.Context) (*entity.CadastrosDoPosto, error) {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := &entity.CadastrosDoPosto{}
	rows, err := r.pool.Query(ctx, `SELECT id, code::text, COALESCE(name, ''), is_active FROM machines WHERE enterprise_id = $1 ORDER BY code`, eid)
	if err != nil {
		return nil, fmt.Errorf("listando máquinas do posto: %w", err)
	}
	for rows.Next() {
		var v entity.CadastroDoPosto
		if err := rows.Scan(&v.ID, &v.Codigo, &v.Nome, &v.Ativo); err != nil {
			rows.Close()
			return nil, err
		}
		out.Maquinas = append(out.Maquinas, v)
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT id, code::text, COALESCE(name, ''), COALESCE(situation::text, '') <> 'INACTIVE' FROM employees WHERE enterprise_id = $1 ORDER BY name`, eid)
	if err != nil {
		return nil, fmt.Errorf("listando operadores do posto: %w", err)
	}
	for rows.Next() {
		var v entity.CadastroDoPosto
		if err := rows.Scan(&v.ID, &v.Codigo, &v.Nome, &v.Ativo); err != nil {
			rows.Close()
			return nil, err
		}
		out.Operadores = append(out.Operadores, v)
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT id, code::text, COALESCE(description, ''), TRUE FROM warehouse WHERE enterprise_id = $1 ORDER BY code`, eid)
	if err != nil {
		return nil, fmt.Errorf("listando depósitos do posto: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v entity.CadastroDoPosto
		if err := rows.Scan(&v.ID, &v.Codigo, &v.Nome, &v.Ativo); err != nil {
			return nil, err
		}
		out.Depositos = append(out.Depositos, v)
	}
	return out, rows.Err()
}
