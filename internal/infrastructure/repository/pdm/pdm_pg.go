// Package pdm grava o padrão descritivo de materiais, sempre da empresa da sessão.
package pdm

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/pdm"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

func (r *PG) Modelo(ctx context.Context, group, modifier int32) (*dom.Modelo, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	m := dom.Modelo{GroupCode: group, ModifierID: modifier, Atributos: []dom.Atributo{}}
	err = r.pool.QueryRow(ctx, `SELECT g.description, md.description, md.abbreviation
		  FROM groups g JOIN modifier md ON md.enterprise_id = g.enterprise_id AND md.id = $3
		 WHERE g.enterprise_id = $1 AND g.code = $2`, empresa, group, modifier).
		Scan(&m.GroupDescription, &m.ModifierDescription, &m.ModifierAbbreviation)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("grupo ou modificador do PDM nesta empresa")
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o modelo do PDM: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT id, group_code, modifier_id, name, position, value_type, unit, prefix, decimal_places,
			is_required, in_description, in_signature, is_active
		  FROM pdm_atributos WHERE enterprise_id = $1 AND group_code = $2 AND modifier_id = $3
		 ORDER BY position, id`, empresa, group, modifier)
	if err != nil {
		return nil, fmt.Errorf("lendo os atributos do PDM: %w", err)
	}
	defer rows.Close()
	indice := map[int64]int{}
	ids := []int64{}
	for rows.Next() {
		var a dom.Atributo
		var casas int16
		var modificador int64
		if err := rows.Scan(&a.ID, &a.GroupCode, &modificador, &a.Nome, &a.Ordem, &a.Tipo, &a.Unidade, &a.Prefixo, &casas,
			&a.Obrigatorio, &a.NaDescricao, &a.NaAssinatura, &a.Ativo); err != nil {
			return nil, err
		}
		a.ModifierID, a.Casas, a.Valores = int32(modificador), int(casas), []dom.Valor{}
		indice[a.ID] = len(m.Atributos)
		ids = append(ids, a.ID)
		m.Atributos = append(m.Atributos, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	vals, err := r.pool.Query(ctx, `SELECT attribute_id, id, value, abbreviation, position, is_active
		  FROM pdm_atributo_valores WHERE attribute_id = ANY($1) ORDER BY position, id`, ids)
	if err != nil {
		return nil, fmt.Errorf("lendo os valores do PDM: %w", err)
	}
	defer vals.Close()
	for vals.Next() {
		var atributo int64
		var v dom.Valor
		if err := vals.Scan(&atributo, &v.ID, &v.Valor, &v.Abreviacao, &v.Ordem, &v.Ativo); err != nil {
			return nil, err
		}
		i := indice[atributo]
		m.Atributos[i].Valores = append(m.Atributos[i].Valores, v)
	}
	return &m, vals.Err()
}

func (r *PG) ModificadoresDoGrupo(ctx context.Context, group int32) ([]int32, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT modifier_id FROM pdm_grupo_modificadores WHERE enterprise_id = $1 AND group_code = $2 ORDER BY modifier_id`, empresa, group)
	if err != nil {
		return nil, fmt.Errorf("lendo os modificadores do grupo: %w", err)
	}
	defer rows.Close()
	out := []int32{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, int32(id))
	}
	return out, rows.Err()
}

func (r *PG) SalvarModificadoresDoGrupo(ctx context.Context, group int32, modificadores []int32) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var grupoExiste bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM groups WHERE enterprise_id = $1 AND code = $2)`, empresa, group).Scan(&grupoExiste); err != nil {
		return err
	}
	if !grupoExiste {
		return errorsuc.NewNotFoundError("grupo do PDM nesta empresa")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM pdm_grupo_modificadores WHERE enterprise_id = $1 AND group_code = $2`, empresa, group); err != nil {
		return fmt.Errorf("limpando os modificadores do grupo: %w", err)
	}
	for _, id := range modificadores {
		tag, err := tx.Exec(ctx, `INSERT INTO pdm_grupo_modificadores (enterprise_id, group_code, modifier_id)
			SELECT $1, $2, m.id FROM modifier m WHERE m.id = $3 AND m.enterprise_id = $1`, empresa, group, id)
		if err != nil {
			return fmt.Errorf("vinculando o modificador %d: %w", id, err)
		}
		if tag.RowsAffected() == 0 {
			return errorsuc.NewValidationError(fmt.Sprintf("o modificador %d não existe nesta empresa", id))
		}
	}
	return tx.Commit(ctx)
}

func (r *PG) SalvarAbreviacao(ctx context.Context, modifier int32, abreviacao *string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE modifier SET abbreviation = $3 WHERE id = $1 AND enterprise_id = $2`, modifier, empresa, abreviacao)
	if err != nil {
		return fmt.Errorf("gravando a abreviação do modificador: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("modificador do PDM nesta empresa")
	}
	return nil
}

func (r *PG) SalvarAtributo(ctx context.Context, a *dom.Atributo, por string) (*dom.Atributo, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var existe bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM groups g JOIN modifier m ON m.enterprise_id = g.enterprise_id
		   WHERE g.enterprise_id = $1 AND g.code = $2 AND m.id = $3)`, empresa, a.GroupCode, a.ModifierID).Scan(&existe); err != nil {
		return nil, err
	}
	if !existe {
		return nil, errorsuc.NewValidationError("grupo ou modificador do PDM não existe nesta empresa")
	}
	if a.ID == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO pdm_atributos (enterprise_id, group_code, modifier_id, name, position, value_type, unit, prefix,
				decimal_places, is_required, in_description, in_signature, is_active, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::uuid) RETURNING id`,
			empresa, a.GroupCode, a.ModifierID, a.Nome, a.Ordem, a.Tipo, a.Unidade, a.Prefixo, a.Casas,
			a.Obrigatorio, a.NaDescricao, a.NaAssinatura, a.Ativo, por).Scan(&a.ID)
	} else {
		var tag int64
		cmd, e := tx.Exec(ctx, `UPDATE pdm_atributos SET name=$3, position=$4, value_type=$5, unit=$6, prefix=$7, decimal_places=$8,
				is_required=$9, in_description=$10, in_signature=$11, is_active=$12, updated_at=NOW()
			WHERE id=$1 AND enterprise_id=$2`, a.ID, empresa, a.Nome, a.Ordem, a.Tipo, a.Unidade, a.Prefixo, a.Casas,
			a.Obrigatorio, a.NaDescricao, a.NaAssinatura, a.Ativo)
		err = e
		if err == nil {
			tag = cmd.RowsAffected()
			if tag == 0 {
				return nil, errorsuc.NewNotFoundError("atributo do PDM nesta empresa")
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("gravando o atributo do PDM: %w", err)
	}
	// Valor que saiu da lista é inativado, não apagado: item antigo pode usá-lo.
	if _, err := tx.Exec(ctx, `UPDATE pdm_atributo_valores SET is_active = FALSE WHERE attribute_id = $1`, a.ID); err != nil {
		return nil, fmt.Errorf("revendo os valores do atributo: %w", err)
	}
	for i, v := range a.Valores {
		if _, err := tx.Exec(ctx, `INSERT INTO pdm_atributo_valores (attribute_id, value, abbreviation, position, is_active)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (attribute_id, value) DO UPDATE
			SET abbreviation = EXCLUDED.abbreviation, position = EXCLUDED.position, is_active = EXCLUDED.is_active`,
			a.ID, v.Valor, v.Abreviacao, i, v.Ativo); err != nil {
			return nil, fmt.Errorf("gravando o valor %s: %w", v.Valor, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

func (r *PG) InativarAtributo(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE pdm_atributos SET is_active = FALSE, updated_at = NOW() WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return fmt.Errorf("inativando o atributo do PDM: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("atributo do PDM nesta empresa")
	}
	return nil
}

func (r *PG) ItensComAssinatura(ctx context.Context, assinatura, excluir string) ([]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT business_code FROM items WHERE enterprise_id = $1 AND pdm_signature = $2 AND business_code <> $3 ORDER BY business_code LIMIT 10`, empresa, assinatura, excluir)
	if err != nil {
		return nil, fmt.Errorf("procurando itens com o mesmo padrão: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PG) Uso(ctx context.Context) (*dom.Uso, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := &dom.Uso{Grupos: map[int32]int{}, Modificadores: map[int32]int{}}
	rows, err := r.pool.Query(ctx, `SELECT pdm_group_code, pdm_modifier_code, COUNT(*) FROM items WHERE enterprise_id = $1 GROUP BY 1, 2`, empresa)
	if err != nil {
		return nil, fmt.Errorf("contando o uso do PDM: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var grupo, modificador int64
		var n int
		if err := rows.Scan(&grupo, &modificador, &n); err != nil {
			return nil, err
		}
		out.Grupos[int32(grupo)] += n
		out.Modificadores[int32(modificador)] += n
	}
	return out, rows.Err()
}
