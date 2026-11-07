package setor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/setor"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

func (r *PG) Setores(ctx context.Context) ([]dom.Setor, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT d.code, d.name, d.active,
			COALESCE((SELECT json_agg(json_build_object('user_id', u.id, 'name', COALESCE(u.name, ''), 'email', COALESCE(u.email, '')) ORDER BY u.name)
			  FROM enterprise_department_users m JOIN users u ON u.id = m.user_id
			 WHERE m.enterprise_id = d.enterprise_id AND m.department_id = d.id), '[]'::json)
		  FROM enterprise_departments d WHERE d.enterprise_id = $1 ORDER BY d.active DESC, d.name`, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo os setores: %w", err)
	}
	defer rows.Close()
	out := []dom.Setor{}
	for rows.Next() {
		var s dom.Setor
		if err := rows.Scan(&s.Codigo, &s.Nome, &s.Ativo, &s.Membros); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PG) SalvarSetor(ctx context.Context, s dom.Setor) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO enterprise_departments (id, enterprise_id, code, name, active, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
		ON CONFLICT (enterprise_id, code) DO UPDATE SET name = EXCLUDED.name, active = EXCLUDED.active`,
		empresa, s.Codigo, s.Nome, s.Ativo); err != nil {
		return fmt.Errorf("gravando o setor %s: %w", s.Codigo, err)
	}
	return nil
}

// DefinirMembros troca as pessoas do setor. Só entra usuário da empresa.
func (r *PG) DefinirMembros(ctx context.Context, codigo string, usuarios []uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM enterprise_departments WHERE enterprise_id = $1 AND code = $2`, empresa, strings.ToUpper(strings.TrimSpace(codigo))).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError("setor " + codigo + " não encontrado")
		}
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM enterprise_department_users WHERE enterprise_id = $1 AND department_id = $2`, empresa, id); err != nil {
		return fmt.Errorf("limpando as pessoas do setor: %w", err)
	}
	for _, u := range usuarios {
		var daEmpresa bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_enterprises WHERE user_id = $1 AND enterprise_id = $2)`, u, empresa).Scan(&daEmpresa); err != nil {
			return err
		}
		if !daEmpresa {
			return errorsuc.NewValidationError("só usuário desta empresa entra no setor")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO enterprise_department_users (enterprise_id, department_id, user_id, created_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT DO NOTHING`, empresa, id, u); err != nil {
			return fmt.Errorf("incluindo a pessoa no setor: %w", err)
		}
	}
	return tx.Commit(ctx)
}

const responsaveisSQL = `SELECT g.code, g.description, pd.code, COALESCE(pd.name, ''), cd.code, COALESCE(cd.name, ''),
		COALESCE((SELECT array_agg(COALESCE(NULLIF(u.name, ''), u.email) ORDER BY u.name) FROM enterprise_department_users m JOIN users u ON u.id = m.user_id
		           WHERE m.enterprise_id = g.enterprise_id AND m.department_id = pd.id), '{}'),
		COALESCE((SELECT array_agg(COALESCE(NULLIF(u.name, ''), u.email) ORDER BY u.name) FROM enterprise_department_users m JOIN users u ON u.id = m.user_id
		           WHERE m.enterprise_id = g.enterprise_id AND m.department_id = cd.id), '{}')
	  FROM groups g
	  LEFT JOIN pdm_group_responsibles r ON r.enterprise_id = g.enterprise_id AND r.group_code = g.code
	  LEFT JOIN enterprise_departments pd ON pd.id = r.planning_department_id AND pd.enterprise_id = g.enterprise_id
	  LEFT JOIN enterprise_departments cd ON cd.id = r.purchasing_department_id AND cd.enterprise_id = g.enterprise_id`

func scanResponsaveis(row pgx.Row) (*dom.Responsaveis, error) {
	var r dom.Responsaveis
	if err := row.Scan(&r.Grupo, &r.GrupoDescricao, &r.Planejamento, &r.PlanejamentoNome, &r.Compras, &r.ComprasNome, &r.PessoasPlanejamento, &r.PessoasCompras); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("grupo não encontrado")
		}
		return nil, err
	}
	return &r, nil
}

func (r *PG) ResponsaveisDoGrupo(ctx context.Context, grupo int) (*dom.Responsaveis, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scanResponsaveis(r.pool.QueryRow(ctx, responsaveisSQL+` WHERE g.enterprise_id = $1 AND g.code = $2`, empresa, grupo))
}

func (r *PG) ResponsaveisDoItem(ctx context.Context, codigoDoItem string) (*dom.Responsaveis, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scanResponsaveis(r.pool.QueryRow(ctx, responsaveisSQL+` JOIN items i ON i.enterprise_id = g.enterprise_id AND i.pdm_group_code = g.code
		 WHERE g.enterprise_id = $1 AND i.business_code = $2`, empresa, codigoDoItem))
}

// SalvarResponsaveis grava os setores do grupo; setor em branco limpa.
func (r *PG) SalvarResponsaveis(ctx context.Context, resp dom.Responsaveis, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	setor := func(codigo *string) (*uuid.UUID, error) {
		if codigo == nil || strings.TrimSpace(*codigo) == "" {
			return nil, nil
		}
		var id uuid.UUID
		var ativo bool
		err := r.pool.QueryRow(ctx, `SELECT id, active FROM enterprise_departments WHERE enterprise_id = $1 AND code = $2`, empresa, strings.ToUpper(strings.TrimSpace(*codigo))).Scan(&id, &ativo)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewValidationError("setor " + *codigo + " não encontrado: cadastre em VNOT0100 › Setores")
		}
		if err != nil {
			return nil, err
		}
		if !ativo {
			return nil, errorsuc.NewValidationError("o setor " + *codigo + " está desativado")
		}
		return &id, nil
	}
	plan, err := setor(resp.Planejamento)
	if err != nil {
		return err
	}
	comp, err := setor(resp.Compras)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `INSERT INTO pdm_group_responsibles (enterprise_id, group_code, planning_department_id, purchasing_department_id, updated_by, updated_at)
		SELECT $1, $2, $3, $4, $5, NOW() WHERE EXISTS (SELECT 1 FROM groups WHERE enterprise_id = $1 AND code = $2)
		ON CONFLICT (enterprise_id, group_code) DO UPDATE SET planning_department_id = EXCLUDED.planning_department_id,
		    purchasing_department_id = EXCLUDED.purchasing_department_id, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		empresa, resp.Grupo, plan, comp, por)
	if err != nil {
		return fmt.Errorf("gravando os responsáveis do grupo %d: %w", resp.Grupo, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("grupo %d não encontrado", resp.Grupo))
	}
	return nil
}
