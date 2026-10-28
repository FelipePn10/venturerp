// Package aprovacao grava políticas de alçada e solicitações de aprovação da
// empresa da sessão.
package aprovacao

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var _ dom.Repository = (*PG)(nil)

// ─── políticas ────────────────────────────────────────────────────────────────

func (r *PG) ListarPoliticas(ctx context.Context) ([]dom.Politica, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tipo, descricao, ativa, segregacao, updated_at, updated_by
		  FROM aprovacao_politicas WHERE enterprise_id = $1 ORDER BY tipo`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando as políticas de aprovação: %w", err)
	}
	var out []dom.Politica
	for rows.Next() {
		var p dom.Politica
		if err := rows.Scan(&p.ID, &p.Tipo, &p.Descricao, &p.Ativa, &p.Segregacao, &p.UpdatedAt, &p.UpdatedBy); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Niveis, err = r.niveis(ctx, empresa, out[i].ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *PG) niveis(ctx context.Context, empresa, politica int64) ([]dom.Nivel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ordem, valor_ate, perfil_id, user_id FROM aprovacao_niveis
		 WHERE politica_id = $1 AND enterprise_id = $2 ORDER BY ordem`, politica, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo os níveis da política: %w", err)
	}
	defer rows.Close()
	out := []dom.Nivel{}
	for rows.Next() {
		var n dom.Nivel
		var ate *decimal.Decimal
		if err := rows.Scan(&n.Ordem, &ate, &n.PerfilID, &n.UserID); err != nil {
			return nil, err
		}
		n.ValorAte = ate
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PG) politica(ctx context.Context, where string, arg any) (*dom.Politica, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var p dom.Politica
	err = r.pool.QueryRow(ctx, `SELECT id, tipo, descricao, ativa, segregacao, updated_at, updated_by
		  FROM aprovacao_politicas WHERE enterprise_id = $1 AND `+where, empresa, arg).
		Scan(&p.ID, &p.Tipo, &p.Descricao, &p.Ativa, &p.Segregacao, &p.UpdatedAt, &p.UpdatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a política de aprovação: %w", err)
	}
	if p.Niveis, err = r.niveis(ctx, empresa, p.ID); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PG) PoliticaAtiva(ctx context.Context, t dom.Tipo) (*dom.Politica, error) {
	return r.politica(ctx, "tipo = $2 AND ativa", string(t))
}

func (r *PG) PoliticaPorID(ctx context.Context, id int64) (*dom.Politica, error) {
	return r.politica(ctx, "id = $2", id)
}

func (r *PG) SalvarPolitica(ctx context.Context, p *dom.Politica, actor uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `
		INSERT INTO aprovacao_politicas (enterprise_id, tipo, descricao, ativa, segregacao, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (enterprise_id, tipo) DO UPDATE
		   SET descricao = EXCLUDED.descricao, ativa = EXCLUDED.ativa, segregacao = EXCLUDED.segregacao,
		       updated_by = EXCLUDED.updated_by, updated_at = NOW()
		RETURNING id`, empresa, string(p.Tipo), p.Descricao, p.Ativa, p.Segregacao, actor).Scan(&p.ID); err != nil {
		return fmt.Errorf("gravando a política de aprovação: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM aprovacao_niveis WHERE politica_id = $1 AND enterprise_id = $2`, p.ID, empresa); err != nil {
		return err
	}
	for _, n := range p.Niveis {
		if _, err := tx.Exec(ctx, `
			INSERT INTO aprovacao_niveis (politica_id, enterprise_id, ordem, valor_ate, perfil_id, user_id)
			VALUES ($1, $2, $3, $4, $5, $6)`, p.ID, empresa, n.Ordem, n.ValorAte, n.PerfilID, n.UserID); err != nil {
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23503" {
				return errorsuc.NewValidationError(fmt.Sprintf("nível %d: o perfil ou usuário aprovador não existe nesta empresa", n.Ordem))
			}
			return fmt.Errorf("gravando o nível %d: %w", n.Ordem, err)
		}
		if n.UserID != nil {
			var membro bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_enterprises WHERE user_id = $1 AND enterprise_id = $2)`, *n.UserID, empresa).Scan(&membro); err != nil {
				return err
			}
			if !membro {
				return errorsuc.NewValidationError(fmt.Sprintf("nível %d: o usuário aprovador não pertence a esta empresa", n.Ordem))
			}
		}
	}
	return tx.Commit(ctx)
}

// ─── solicitações ─────────────────────────────────────────────────────────────

const colunasSolicitacao = `s.id, s.politica_id, s.tipo, s.documento_codigo, s.valor, s.descricao, s.solicitante,
	COALESCE(NULLIF(u.name, ''), u.email, ''), s.status, s.niveis_exigidos, s.nivel_atual, s.created_at, s.concluida_at`

func scanSolicitacao(row pgx.Row) (dom.Solicitacao, error) {
	var s dom.Solicitacao
	err := row.Scan(&s.ID, &s.PoliticaID, &s.Tipo, &s.DocumentoCodigo, &s.Valor, &s.Descricao, &s.Solicitante,
		&s.SolicitanteNome, &s.Status, &s.NiveisExigidos, &s.NivelAtual, &s.CreatedAt, &s.ConcluidaAt)
	return s, err
}

func (r *PG) CriarSolicitacao(ctx context.Context, s *dom.Solicitacao) (*dom.Solicitacao, bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, false, err
	}
	if atual, err := r.PendenteDoDocumento(ctx, s.Tipo, s.DocumentoCodigo); err != nil || atual != nil {
		return atual, false, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `
		INSERT INTO aprovacao_solicitacoes (enterprise_id, politica_id, tipo, documento_codigo, valor, descricao, solicitante, niveis_exigidos)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		empresa, s.PoliticaID, string(s.Tipo), s.DocumentoCodigo, s.Valor, s.Descricao, s.Solicitante, s.NiveisExigidos).Scan(&s.ID); err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			// Outra requisição criou a pendente ao mesmo tempo: vale a dela.
			atual, errLer := r.PendenteDoDocumento(ctx, s.Tipo, s.DocumentoCodigo)
			return atual, false, errLer
		}
		return nil, false, fmt.Errorf("criando a solicitação de aprovação: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO aprovacao_eventos (enterprise_id, solicitacao_id, evento, ator, justificativa)
		VALUES ($1, $2, 'SOLICITADA', $3, $4)`, empresa, s.ID, s.Solicitante, s.Descricao); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	criada, err := r.BuscarSolicitacao(ctx, s.ID)
	return criada, true, err
}

func (r *PG) BuscarSolicitacao(ctx context.Context, id int64) (*dom.Solicitacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := scanSolicitacao(r.pool.QueryRow(ctx, `SELECT `+colunasSolicitacao+`
		  FROM aprovacao_solicitacoes s JOIN users u ON u.id = s.solicitante
		 WHERE s.id = $1 AND s.enterprise_id = $2`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("solicitação de aprovação %d não encontrada", id))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a solicitação de aprovação: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT e.evento, e.nivel, e.ator, COALESCE(NULLIF(u.name, ''), u.email, ''), e.justificativa, e.created_at
		  FROM aprovacao_eventos e JOIN users u ON u.id = e.ator
		 WHERE e.solicitacao_id = $1 AND e.enterprise_id = $2 ORDER BY e.id`, id, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo o histórico da aprovação: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var e dom.Evento
		var nivel *int16
		if err := rows.Scan(&e.Evento, &nivel, &e.Ator, &e.AtorNome, &e.Justificativa, &e.CreatedAt); err != nil {
			return nil, err
		}
		if nivel != nil {
			n := int(*nivel)
			e.Nivel = &n
		}
		s.Eventos = append(s.Eventos, e)
	}
	return &s, rows.Err()
}

// AprovadaDoDocumento: a última solicitação aprovada do documento, se houver.
func (r *PG) AprovadaDoDocumento(ctx context.Context, t dom.Tipo, documentoCodigo int64) (*dom.Solicitacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	err = r.pool.QueryRow(ctx, `SELECT id FROM aprovacao_solicitacoes
		 WHERE enterprise_id = $1 AND tipo = $2 AND documento_codigo = $3 AND status = 'APROVADA'
		 ORDER BY id DESC LIMIT 1`,
		empresa, string(t), documentoCodigo).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("conferindo a aprovação concedida: %w", err)
	}
	return r.BuscarSolicitacao(ctx, id)
}

func (r *PG) PendenteDoDocumento(ctx context.Context, t dom.Tipo, documentoCodigo int64) (*dom.Solicitacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	err = r.pool.QueryRow(ctx, `SELECT id FROM aprovacao_solicitacoes
		 WHERE enterprise_id = $1 AND tipo = $2 AND documento_codigo = $3 AND status = 'PENDENTE'`,
		empresa, string(t), documentoCodigo).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("conferindo a aprovação pendente: %w", err)
	}
	return r.BuscarSolicitacao(ctx, id)
}

func (r *PG) ListarSolicitacoes(ctx context.Context, status dom.Status, limite int) ([]dom.Solicitacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if limite <= 0 || limite > 500 {
		limite = 200
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasSolicitacao+`
		  FROM aprovacao_solicitacoes s JOIN users u ON u.id = s.solicitante
		 WHERE s.enterprise_id = $1 AND ($2 = '' OR s.status = $2)
		 ORDER BY (s.status = 'PENDENTE') DESC, s.created_at DESC LIMIT $3`, empresa, string(status), limite)
	if err != nil {
		return nil, fmt.Errorf("listando as solicitações de aprovação: %w", err)
	}
	defer rows.Close()
	out := []dom.Solicitacao{}
	for rows.Next() {
		s, err := scanSolicitacao(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PG) UsuarioTemPerfil(ctx context.Context, userID uuid.UUID, perfilID int64) (bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var tem bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM acesso_usuario_perfis up JOIN acesso_perfis p ON p.id = up.perfil_id AND p.ativo
		 WHERE up.user_id = $1 AND up.perfil_id = $2 AND up.enterprise_id = $3)`, userID, perfilID, empresa).Scan(&tem)
	return tem, err
}

func (r *PG) RegistrarAprovacaoDeNivel(ctx context.Context, id int64, nivelEsperado int, concluir bool, ator uuid.UUID, justificativa *string) error {
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
		UPDATE aprovacao_solicitacoes
		   SET nivel_atual = CASE WHEN $4 THEN nivel_atual ELSE nivel_atual + 1 END,
		       status = CASE WHEN $4 THEN 'APROVADA' ELSE status END,
		       concluida_at = CASE WHEN $4 THEN NOW() ELSE NULL END
		 WHERE id = $1 AND enterprise_id = $2 AND status = 'PENDENTE' AND nivel_atual = $3`,
		id, empresa, nivelEsperado, concluir)
	if err != nil {
		return fmt.Errorf("registrando a aprovação: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewConflictError("a solicitação mudou enquanto você aprovava (outro aprovador agiu antes): atualize e confira")
	}
	evento := "APROVADA_NIVEL"
	if concluir {
		evento = "APROVADA"
	}
	if _, err := tx.Exec(ctx, `INSERT INTO aprovacao_eventos (enterprise_id, solicitacao_id, evento, nivel, ator, justificativa)
		VALUES ($1, $2, $3, $4, $5, $6)`, empresa, id, evento, nivelEsperado, ator, justificativa); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) RegistrarReprovacao(ctx context.Context, id int64, nivel int, ator uuid.UUID, justificativa string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE aprovacao_solicitacoes SET status = 'REPROVADA', concluida_at = NOW()
		 WHERE id = $1 AND enterprise_id = $2 AND status = 'PENDENTE' AND nivel_atual = $3`, id, empresa, nivel)
	if err != nil {
		return fmt.Errorf("registrando a reprovação: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewConflictError("a solicitação mudou enquanto você reprovava: atualize e confira")
	}
	if _, err := tx.Exec(ctx, `INSERT INTO aprovacao_eventos (enterprise_id, solicitacao_id, evento, nivel, ator, justificativa)
		VALUES ($1, $2, 'REPROVADA', $3, $4, $5)`, empresa, id, nivel, ator, justificativa); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
