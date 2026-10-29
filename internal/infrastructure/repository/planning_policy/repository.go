// Package planning_policy guarda a política de planejamento por item ou por
// classificação (VPLA0310) e monta a política efetiva que o MRP usa.
package planning_policy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// db é o pool ou uma transação.
type db interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db db }

func New(d db) *Repository { return &Repository{db: d} }

var _ repository.PoliticaRepository = (*Repository)(nil)

const colunas = `p.id, p.item_code, p.classification_id, p.estrategia, p.consumo_dias_antes, p.consumo_dias_depois,
	p.horizonte_firme_dias, p.tempo_seguranca_dias, p.observacao, p.created_at, p.created_by, p.updated_at, p.updated_by,
	COALESCE(i.business_code, ''), COALESCE(i.name, ''), COALESCE(c.code, ''),
	COALESCE(c.description, '') || COALESCE(' (' || m.description || ')', ''), COALESCE(c.level, 0)`

const juncoes = `FROM politicas_planejamento p
	LEFT JOIN items i ON i.code = p.item_code AND i.enterprise_id = p.enterprise_id
	LEFT JOIN item_classifications c ON c.id = p.classification_id
	LEFT JOIN item_classification_masks m ON m.id = c.mask_id`

func scan(row pgx.Row) (*entity.PoliticaPlanejamento, error) {
	p := &entity.PoliticaPlanejamento{}
	var por *uuid.UUID
	if err := row.Scan(&p.ID, &p.ItemCode, &p.ClassificationID, &p.Estrategia, &p.ConsumoDiasAntes, &p.ConsumoDiasDepois,
		&p.HorizonteFirmeDias, &p.TempoSegurancaDias, &p.Observacao, &p.CreatedAt, &p.CreatedBy, &p.UpdatedAt, &por,
		&p.ItemBusinessCode, &p.ItemName, &p.ClassificationCode, &p.ClassificationLabel, &p.ClassificationLevel); err != nil {
		return nil, err
	}
	p.UpdatedBy = por
	return p, nil
}

func (r *Repository) Listar(ctx context.Context, f entity.FiltroPolitica) ([]*entity.PoliticaPlanejamento, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	busca := "%" + strings.ToLower(strings.TrimSpace(f.Busca)) + "%"
	rows, err := r.db.Query(ctx, `SELECT `+colunas+` `+juncoes+`
		WHERE p.enterprise_id = $1
		  AND ($2 = '' OR ($2 = 'ITEM' AND p.item_code IS NOT NULL) OR ($2 = 'CLASSIFICACAO' AND p.classification_id IS NOT NULL))
		  AND ($3 = '%%' OR LOWER(COALESCE(i.business_code, '') || ' ' || COALESCE(i.name, '') || ' ' || COALESCE(c.code, '') || ' '
		       || COALESCE(c.description, '') || ' ' || COALESCE(p.observacao, '')) LIKE $3)
		ORDER BY (p.item_code IS NULL), c.code, i.business_code, p.id
		LIMIT 1000`, e, f.Escopo, busca)
	if err != nil {
		return nil, fmt.Errorf("listando as políticas de planejamento: %w", err)
	}
	defer rows.Close()
	var out []*entity.PoliticaPlanejamento
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("lendo a política de planejamento: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) Obter(ctx context.Context, id int64) (*entity.PoliticaPlanejamento, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	p, err := scan(r.db.QueryRow(ctx, `SELECT `+colunas+` `+juncoes+` WHERE p.id = $1 AND p.enterprise_id = $2`, id, e))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("política de planejamento %d não encontrada nesta empresa", id))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a política de planejamento %d: %w", id, err)
	}
	return p, nil
}

// emUso traduz a violação de unicidade (uma política por item ou por
// classificação) para a mensagem da tela.
func emUso(err error, p *entity.PoliticaPlanejamento) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if p.ItemCode != nil {
			return errorsuc.NewConflictError("este item já tem política: altere a que existe")
		}
		return errorsuc.NewConflictError("esta classificação já tem política: altere a que existe")
	}
	return err
}

func (r *Repository) Criar(ctx context.Context, p *entity.PoliticaPlanejamento) (*entity.PoliticaPlanejamento, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	err = r.db.QueryRow(ctx, `INSERT INTO politicas_planejamento (enterprise_id, item_code, classification_id, estrategia,
		consumo_dias_antes, consumo_dias_depois, horizonte_firme_dias, tempo_seguranca_dias, observacao, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10) RETURNING id`,
		e, p.ItemCode, p.ClassificationID, p.Estrategia, p.ConsumoDiasAntes, p.ConsumoDiasDepois, p.HorizonteFirmeDias,
		p.TempoSegurancaDias, p.Observacao, p.CreatedBy).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("gravando a política de planejamento: %w", emUso(err, p))
	}
	return r.Obter(ctx, id)
}

// Alterar troca os campos da política; o escopo (item ou classificação) não
// muda.
func (r *Repository) Alterar(ctx context.Context, p *entity.PoliticaPlanejamento) (*entity.PoliticaPlanejamento, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := r.db.Exec(ctx, `UPDATE politicas_planejamento
		SET estrategia = $3, consumo_dias_antes = $4, consumo_dias_depois = $5, horizonte_firme_dias = $6,
		    tempo_seguranca_dias = $7, observacao = $8, updated_by = $9, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2`,
		p.ID, e, p.Estrategia, p.ConsumoDiasAntes, p.ConsumoDiasDepois, p.HorizonteFirmeDias, p.TempoSegurancaDias,
		p.Observacao, p.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("alterando a política de planejamento %d: %w", p.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("política de planejamento %d não encontrada nesta empresa", p.ID))
	}
	return r.Obter(ctx, p.ID)
}

func (r *Repository) Excluir(ctx context.Context, id int64) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM politicas_planejamento WHERE id = $1 AND enterprise_id = $2`, id, e)
	if err != nil {
		return fmt.Errorf("excluindo a política de planejamento %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("política de planejamento %d não encontrada nesta empresa", id))
	}
	return nil
}

func (r *Repository) Classificacao(ctx context.Context, id int64) (*repository.ClassificacaoRef, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	c := &repository.ClassificacaoRef{}
	err = r.db.QueryRow(ctx, `SELECT c.id, c.code, c.description, c.level FROM item_classifications c
		JOIN item_classification_masks m ON m.id = c.mask_id
		WHERE c.id = $1 AND m.enterprise_id = $2`, id, e).Scan(&c.ID, &c.Code, &c.Descricao, &c.Level)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("classificação %d não encontrada nesta empresa", id))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a classificação %d: %w", id, err)
	}
	return c, nil
}

func (r *Repository) PoliticasDePlanejamento(ctx context.Context, empresa entity.PoliticaEfetiva) (map[int64]entity.PoliticaEfetiva, error) {
	return r.resolver(ctx, nil, empresa)
}

func (r *Repository) PoliticaDoItem(ctx context.Context, item int64, empresa entity.PoliticaEfetiva) (entity.PoliticaEfetiva, error) {
	pols, err := r.resolver(ctx, &item, empresa)
	if err != nil {
		return entity.PoliticaEfetiva{}, err
	}
	if p, ok := pols[item]; ok {
		return p, nil
	}
	return empresa, nil
}

// resolver lê as regras de item e as de classificação que alcançam cada item
// (a classificação atribuída a ele e as ancestrais dela) e monta a efetiva.
func (r *Repository) resolver(ctx context.Context, item *int64, empresa entity.PoliticaEfetiva) (map[int64]entity.PoliticaEfetiva, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	doItem := map[int64]*entity.PoliticaPlanejamento{}
	rows, err := r.db.Query(ctx, `SELECT id, item_code, estrategia, consumo_dias_antes, consumo_dias_depois,
		horizonte_firme_dias, tempo_seguranca_dias
		FROM politicas_planejamento WHERE enterprise_id = $1 AND item_code IS NOT NULL AND ($2::bigint IS NULL OR item_code = $2)`, e, item)
	if err != nil {
		return nil, fmt.Errorf("lendo as políticas dos itens: %w", err)
	}
	for rows.Next() {
		p := &entity.PoliticaPlanejamento{}
		var code int64
		if err := rows.Scan(&p.ID, &code, &p.Estrategia, &p.ConsumoDiasAntes, &p.ConsumoDiasDepois, &p.HorizonteFirmeDias, &p.TempoSegurancaDias); err != nil {
			rows.Close()
			return nil, fmt.Errorf("lendo a política do item: %w", err)
		}
		p.ItemCode = &code
		doItem[code] = p
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	porClassificacao := map[int64][]*entity.PoliticaPlanejamento{}
	rows, err = r.db.Query(ctx, `
		WITH RECURSIVE anc AS (
			SELECT a.item_code, c.id, c.parent_id
			  FROM item_classification_assignments a
			  JOIN item_classifications c ON c.id = a.classification_id
			 WHERE a.enterprise_id = $1 AND ($2::bigint IS NULL OR a.item_code = $2)
			UNION
			SELECT anc.item_code, p.id, p.parent_id
			  FROM anc JOIN item_classifications p ON p.id = anc.parent_id
		)
		SELECT DISTINCT anc.item_code, pp.id, c.code, c.level, pp.estrategia, pp.consumo_dias_antes, pp.consumo_dias_depois,
		       pp.horizonte_firme_dias, pp.tempo_seguranca_dias
		  FROM anc
		  JOIN politicas_planejamento pp ON pp.classification_id = anc.id AND pp.enterprise_id = $1
		  JOIN item_classifications c ON c.id = pp.classification_id AND c.is_active`, e, item)
	if err != nil {
		return nil, fmt.Errorf("lendo as políticas das classificações: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		p := &entity.PoliticaPlanejamento{}
		var code int64
		if err := rows.Scan(&code, &p.ID, &p.ClassificationCode, &p.ClassificationLevel, &p.Estrategia, &p.ConsumoDiasAntes,
			&p.ConsumoDiasDepois, &p.HorizonteFirmeDias, &p.TempoSegurancaDias); err != nil {
			return nil, fmt.Errorf("lendo a política da classificação: %w", err)
		}
		porClassificacao[code] = append(porClassificacao[code], p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entity.ResolverPoliticas(empresa, doItem, porClassificacao), nil
}
