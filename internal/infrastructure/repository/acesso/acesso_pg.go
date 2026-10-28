// Package acesso grava perfis de acesso, permissões e vínculos da empresa.
package acesso

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/acesso"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var _ dom.Repository = (*PG)(nil)

func traduzir(err error, oQue string) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewConflictError("já existe um perfil com esse nome nesta empresa")
	}
	return fmt.Errorf("%s: %w", oQue, err)
}

func (r *PG) ListarPerfis(ctx context.Context) ([]*dom.Perfil, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.enterprise_id, p.nome, p.descricao, p.ativo, p.created_at, p.updated_at, p.updated_by,
		       (SELECT count(*) FROM acesso_usuario_perfis u WHERE u.perfil_id = p.id)
		  FROM acesso_perfis p WHERE p.enterprise_id = $1 ORDER BY lower(p.nome)`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os perfis de acesso: %w", err)
	}
	defer rows.Close()
	out := []*dom.Perfil{}
	porID := map[int64]*dom.Perfil{}
	for rows.Next() {
		p := &dom.Perfil{Permissoes: dom.Permissoes{}}
		if err := rows.Scan(&p.ID, &p.EnterpriseID, &p.Nome, &p.Descricao, &p.Ativo, &p.CreatedAt, &p.UpdatedAt, &p.UpdatedBy, &p.Usuarios); err != nil {
			return nil, err
		}
		out = append(out, p)
		porID[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.carregarPermissoes(ctx, empresa, porID); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PG) carregarPermissoes(ctx context.Context, empresa int64, porID map[int64]*dom.Perfil) error {
	if len(porID) == 0 {
		return nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT pp.perfil_id, pp.modulo, pp.acao FROM acesso_perfil_permissoes pp
		  JOIN acesso_perfis p ON p.id = pp.perfil_id WHERE p.enterprise_id = $1`, empresa)
	if err != nil {
		return fmt.Errorf("lendo as permissões dos perfis: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var m, a string
		if err := rows.Scan(&id, &m, &a); err != nil {
			return err
		}
		if p, ok := porID[id]; ok {
			p.Permissoes.Unir(dom.Permissoes{dom.Modulo(m): {dom.Acao(a): true}})
		}
	}
	return rows.Err()
}

func (r *PG) BuscarPerfil(ctx context.Context, id int64) (*dom.Perfil, error) {
	perfis, err := r.ListarPerfis(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range perfis {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, errorsuc.NewNotFoundError(fmt.Sprintf("perfil de acesso %d não encontrado", id))
}

func gravarPermissoes(ctx context.Context, tx pgx.Tx, id int64, p dom.Permissoes) error {
	if _, err := tx.Exec(ctx, `DELETE FROM acesso_perfil_permissoes WHERE perfil_id = $1`, id); err != nil {
		return fmt.Errorf("limpando as permissões do perfil: %w", err)
	}
	for _, item := range p.Lista() {
		if _, err := tx.Exec(ctx, `INSERT INTO acesso_perfil_permissoes (perfil_id, modulo, acao) VALUES ($1, $2, $3)`,
			id, string(item.Modulo), string(item.Acao)); err != nil {
			return fmt.Errorf("gravando a permissão %s/%s: %w", item.Modulo, item.Acao, err)
		}
	}
	return nil
}

func (r *PG) CriarPerfil(ctx context.Context, p *dom.Perfil, actor uuid.UUID) (*dom.Perfil, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `
		INSERT INTO acesso_perfis (enterprise_id, nome, descricao, ativo, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $5) RETURNING id`, empresa, p.Nome, p.Descricao, p.Ativo, actor).Scan(&p.ID); err != nil {
		return nil, traduzir(err, "criando o perfil de acesso")
	}
	if err := gravarPermissoes(ctx, tx, p.ID, p.Permissoes); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.BuscarPerfil(ctx, p.ID)
}

func (r *PG) AtualizarPerfil(ctx context.Context, p *dom.Perfil, actor uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE acesso_perfis SET nome = $1, descricao = $2, ativo = $3, updated_by = $4, updated_at = NOW()
		 WHERE id = $5 AND enterprise_id = $6`, p.Nome, p.Descricao, p.Ativo, actor, p.ID, empresa)
	if err != nil {
		return traduzir(err, "alterando o perfil de acesso")
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("perfil de acesso %d não encontrado", p.ID))
	}
	if err := gravarPermissoes(ctx, tx, p.ID, p.Permissoes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) ExcluirPerfil(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var usuarios int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM acesso_usuario_perfis WHERE perfil_id = $1 AND enterprise_id = $2`, id, empresa).Scan(&usuarios); err != nil {
		return err
	}
	if usuarios > 0 {
		return errorsuc.NewConflictError(fmt.Sprintf("o perfil está ligado a %d usuário(s): retire-o dos usuários ou inative-o", usuarios))
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM acesso_perfis WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("excluindo o perfil de acesso: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("perfil de acesso %d não encontrado", id))
	}
	return nil
}

func (r *PG) ListarUsuarios(ctx context.Context) ([]dom.UsuarioDaEmpresa, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, COALESCE(u.name, ''), COALESCE(u.email, ''), COALESCE(ue.role, ''), COALESCE(u.is_active, TRUE),
		       COALESCE(array_agg(up.perfil_id ORDER BY up.perfil_id) FILTER (WHERE up.perfil_id IS NOT NULL), '{}')
		  FROM user_enterprises ue
		  JOIN users u ON u.id = ue.user_id
		  LEFT JOIN acesso_usuario_perfis up ON up.user_id = ue.user_id AND up.enterprise_id = ue.enterprise_id
		 WHERE ue.enterprise_id = $1
		 GROUP BY u.id, u.name, u.email, ue.role, u.is_active
		 ORDER BY lower(COALESCE(u.name, u.email, ''))`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os usuários da empresa: %w", err)
	}
	defer rows.Close()
	out := []dom.UsuarioDaEmpresa{}
	for rows.Next() {
		var u dom.UsuarioDaEmpresa
		if err := rows.Scan(&u.UserID, &u.Nome, &u.Email, &u.Papel, &u.Ativo, &u.Perfis); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PG) DefinirPerfisDoUsuario(ctx context.Context, userID uuid.UUID, perfis []int64, actor uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var existe bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_enterprises WHERE user_id = $1 AND enterprise_id = $2)`, userID, empresa).Scan(&existe); err != nil {
		return err
	}
	if !existe {
		return errorsuc.NewNotFoundError("o usuário não pertence a esta empresa")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM acesso_usuario_perfis WHERE user_id = $1 AND enterprise_id = $2`, userID, empresa); err != nil {
		return fmt.Errorf("limpando os perfis do usuário: %w", err)
	}
	for _, id := range perfis {
		// A chave composta (empresa, perfil) recusa perfil de outra empresa.
		if _, err := tx.Exec(ctx, `INSERT INTO acesso_usuario_perfis (enterprise_id, user_id, perfil_id, created_by) VALUES ($1, $2, $3, $4)`,
			empresa, userID, id, actor); err != nil {
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23503" {
				return errorsuc.NewNotFoundError(fmt.Sprintf("perfil de acesso %d não encontrado nesta empresa", id))
			}
			return fmt.Errorf("ligando o perfil %d ao usuário: %w", id, err)
		}
	}
	return tx.Commit(ctx)
}

func (r *PG) AcessoDoUsuario(ctx context.Context, userID uuid.UUID, enterpriseID int64) (dom.AcessoDoUsuario, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.nome, pp.modulo, pp.acao
		  FROM acesso_usuario_perfis up
		  JOIN acesso_perfis p ON p.id = up.perfil_id AND p.enterprise_id = up.enterprise_id AND p.ativo
		  LEFT JOIN acesso_perfil_permissoes pp ON pp.perfil_id = p.id
		 WHERE up.user_id = $1 AND up.enterprise_id = $2
		 ORDER BY lower(p.nome)`, userID, enterpriseID)
	if err != nil {
		return dom.AcessoDoUsuario{}, fmt.Errorf("lendo os perfis do usuário: %w", err)
	}
	defer rows.Close()
	a := dom.AcessoDoUsuario{Permissoes: dom.Permissoes{}}
	vistos := map[int64]bool{}
	for rows.Next() {
		var id int64
		var nome string
		var m, ac *string
		if err := rows.Scan(&id, &nome, &m, &ac); err != nil {
			return dom.AcessoDoUsuario{}, err
		}
		if !vistos[id] {
			vistos[id] = true
			a.Perfis = append(a.Perfis, nome)
		}
		if m != nil && ac != nil {
			a.Permissoes.Unir(dom.Permissoes{dom.Modulo(*m): {dom.Acao(*ac): true}})
		}
	}
	a.Restrito = len(a.Perfis) > 0
	return a, rows.Err()
}
