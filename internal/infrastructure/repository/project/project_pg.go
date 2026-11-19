// Package project grava e lê o cadastro de Projetos, sempre da empresa da
// sessão. O cliente é da mesma empresa (FK composta no banco).
package project

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/project/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/project/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var _ repository.ProjectRepository = (*PG)(nil)

const colunas = `p.id, p.enterprise_id, p.code, p.name, p.customer_code, COALESCE(c.name, ''), p.status, p.start_date, p.end_date,
	COALESCE(p.notes, ''), p.created_by, p.created_at, p.updated_at,
	(SELECT COUNT(*) FROM production_orders o WHERE o.enterprise_id = p.enterprise_id AND o.project_id = p.id AND o.status IN ('OPEN','IN_PROGRESS')),
	(SELECT COUNT(*) FROM production_orders o WHERE o.enterprise_id = p.enterprise_id AND o.project_id = p.id)`

const origem = ` FROM projects p LEFT JOIN customers c ON c.enterprise_id = p.enterprise_id AND c.code = p.customer_code `

func scan(row pgx.Row) (*entity.Project, error) {
	p := &entity.Project{}
	err := row.Scan(&p.ID, &p.EnterpriseID, &p.Code, &p.Name, &p.CustomerCode, &p.CustomerName, &p.Status, &p.StartDate, &p.EndDate,
		&p.Notes, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.OpenOrders, &p.TotalOrders)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainerr.NewNotFound("projeto não encontrado")
	}
	return p, err
}

func traduzir(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return domainerr.NewConflict("já existe um projeto com esse código nesta empresa")
		case "23503":
			return domainerr.NewValidation("cliente não encontrado nesta empresa")
		}
	}
	return err
}

func (r *PG) List(ctx context.Context, f repository.Filtro) ([]*entity.Project, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	busca := "%" + strings.ToUpper(strings.TrimSpace(f.Busca)) + "%"
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+origem+`
		 WHERE p.enterprise_id = $1 AND ($2 = '' OR p.status = $2)
		   AND ($3 = '%%' OR upper(p.code) LIKE $3 OR upper(p.name) LIKE $3 OR upper(COALESCE(c.name, '')) LIKE $3 OR p.customer_code::text LIKE $3)
		 ORDER BY (p.status = 'ATIVO') DESC, p.code`, empresa, strings.ToUpper(strings.TrimSpace(f.Status)), busca)
	if err != nil {
		return nil, fmt.Errorf("listando os projetos: %w", err)
	}
	defer rows.Close()
	out := []*entity.Project{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PG) GetByCode(ctx context.Context, code string) (*entity.Project, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scan(r.pool.QueryRow(ctx, `SELECT `+colunas+origem+` WHERE p.enterprise_id = $1 AND upper(btrim(p.code)) = upper(btrim($2))`, empresa, code))
}

func (r *PG) GetByID(ctx context.Context, id int64) (*entity.Project, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scan(r.pool.QueryRow(ctx, `SELECT `+colunas+origem+` WHERE p.enterprise_id = $1 AND p.id = $2`, empresa, id))
}

func (r *PG) Create(ctx context.Context, p *entity.Project) (*entity.Project, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO projects(enterprise_id, code, name, customer_code, status, start_date, end_date, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9) RETURNING id`,
		empresa, p.Code, p.Name, p.CustomerCode, p.Status, p.StartDate, p.EndDate, p.Notes, p.CreatedBy).Scan(&id); err != nil {
		return nil, traduzir(err)
	}
	return r.GetByID(ctx, id)
}

// Update altera pelo id, só na empresa da sessão.
func (r *PG) Update(ctx context.Context, p *entity.Project) (*entity.Project, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE projects SET code = $3, name = $4, customer_code = $5, status = $6, start_date = $7, end_date = $8,
		notes = NULLIF($9, ''), updated_at = NOW() WHERE enterprise_id = $1 AND id = $2`,
		empresa, p.ID, p.Code, p.Name, p.CustomerCode, p.Status, p.StartDate, p.EndDate, p.Notes)
	if err != nil {
		return nil, traduzir(err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domainerr.NewNotFound("projeto não encontrado")
	}
	return r.GetByID(ctx, p.ID)
}
