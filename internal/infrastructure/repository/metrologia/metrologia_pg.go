// Package metrologia grava instrumentos e calibrações da empresa da sessão.
package metrologia

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/metrologia"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var _ dom.Repository = (*PG)(nil)

const colunas = `id, codigo, descricao, grupo, fabricante, numero_serie, faixa, resolucao, ema, unidade, localizacao, frequencia_dias,
	status, motivo_status, ultima_calibracao, proxima_calibracao, updated_at, updated_by`

func scan(row pgx.Row) (*dom.Instrumento, error) {
	var i dom.Instrumento
	if err := row.Scan(&i.ID, &i.Codigo, &i.Descricao, &i.Grupo, &i.Fabricante, &i.NumeroSerie, &i.Faixa, &i.Resolucao, &i.EMA, &i.Unidade,
		&i.Localizacao, &i.FrequenciaDias, &i.Status, &i.MotivoStatus, &i.UltimaCalibracao, &i.ProximaCalibracao, &i.UpdatedAt, &i.UpdatedBy); err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *PG) Salvar(ctx context.Context, i *dom.Instrumento) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	args := []any{empresa, i.Codigo, i.Descricao, i.Grupo, i.Fabricante, i.NumeroSerie, i.Faixa, i.Resolucao, i.EMA, i.Unidade, i.Localizacao,
		i.FrequenciaDias, i.Status, i.MotivoStatus, i.UpdatedBy}
	if i.ID == 0 {
		err = r.pool.QueryRow(ctx, `INSERT INTO quality_instruments (enterprise_id, codigo, descricao, grupo, fabricante, numero_serie, faixa, resolucao, ema,
				unidade, localizacao, frequencia_dias, status, motivo_status, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id, updated_at`, args...).Scan(&i.ID, &i.UpdatedAt)
	} else {
		// A frequência nova vale para a próxima calibração; a data já
		// calculada não muda aqui (ela vem de uma calibração registrada).
		err = r.pool.QueryRow(ctx, `UPDATE quality_instruments SET codigo=$2, descricao=$3, grupo=$4, fabricante=$5, numero_serie=$6, faixa=$7,
				resolucao=$8, ema=$9, unidade=$10, localizacao=$11, frequencia_dias=$12, status=$13, motivo_status=$14, updated_by=$15, updated_at=NOW()
			 WHERE enterprise_id=$1 AND id=$16 RETURNING updated_at`, append(args, i.ID)...).Scan(&i.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("instrumento %d", i.ID))
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewValidationError(fmt.Sprintf("já existe um instrumento com o código %s", i.Codigo))
	}
	if err != nil {
		return fmt.Errorf("gravando o instrumento: %w", err)
	}
	return nil
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.Instrumento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	i, err := scan(r.pool.QueryRow(ctx, `SELECT `+colunas+` FROM quality_instruments WHERE enterprise_id=$1 AND id=$2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("instrumento %d", id))
	}
	return i, err
}

func (r *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.Instrumento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if f.Hoje.IsZero() {
		f.Hoje = time.Now()
	}
	if f.Dias <= 0 {
		f.Dias = 30
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+` FROM quality_instruments
		 WHERE enterprise_id = $1 AND ($2 = '' OR grupo = $2) AND ($6 OR status <> 'BAIXADO')
		   AND CASE $3
		         WHEN 'VENCIDOS' THEN status = 'ATIVO' AND (proxima_calibracao IS NULL OR proxima_calibracao < $4::date)
		         WHEN 'A_VENCER' THEN status = 'ATIVO' AND proxima_calibracao >= $4::date AND proxima_calibracao <= $4::date + $5::int
		         WHEN 'BLOQUEADOS' THEN status = 'BLOQUEADO'
		         ELSE TRUE END
		 ORDER BY proxima_calibracao NULLS FIRST, codigo`, empresa, dom.NormalizarGrupo(f.Grupo), f.Situacao, f.Hoje, f.Dias, f.Inclusive)
	if err != nil {
		return nil, fmt.Errorf("listando os instrumentos: %w", err)
	}
	defer rows.Close()
	out := []dom.Instrumento{}
	for rows.Next() {
		i, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *i)
	}
	return out, rows.Err()
}

func (r *PG) RegistrarCalibracao(ctx context.Context, id int64, c *dom.Calibracao, hoje time.Time) (*dom.Instrumento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	i, err := scan(tx.QueryRow(ctx, `SELECT `+colunas+` FROM quality_instruments WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("instrumento %d", id))
	}
	if err != nil {
		return nil, err
	}
	if i.Status == "BAIXADO" {
		return nil, errorsuc.NewValidationError("instrumento baixado não recebe calibração")
	}
	if err := c.Avaliar(i, hoje); err != nil {
		return nil, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO quality_calibrations (enterprise_id, instrument_id, data, executor, laboratorio, certificado, erro_encontrado,
			incerteza, ema, resultado, ajustado, proxima, observacoes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id, created_at`,
		empresa, id, c.Data, c.Executor, c.Laboratorio, c.Certificado, c.ErroEncontrado, c.Incerteza, c.EMA, c.Resultado, c.Ajustado,
		c.Proxima, c.Observacoes, c.CreatedBy).Scan(&c.ID, &c.CreatedAt); err != nil {
		return nil, fmt.Errorf("gravando a calibração: %w", err)
	}
	c.InstrumentoID = id
	i.Aplicar(c)
	if _, err := tx.Exec(ctx, `UPDATE quality_instruments SET status=$3, motivo_status=$4, ultima_calibracao=$5, proxima_calibracao=$6, updated_at=NOW(), updated_by=$7
		 WHERE enterprise_id=$1 AND id=$2`, empresa, id, i.Status, i.MotivoStatus, i.UltimaCalibracao, i.ProximaCalibracao, c.CreatedBy); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return i, nil
}

func (r *PG) Calibracoes(ctx context.Context, id int64) ([]dom.Calibracao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id, instrument_id, data, executor, laboratorio, certificado, erro_encontrado, incerteza, ema, resultado, ajustado,
			proxima, observacoes, created_at, created_by
		  FROM quality_calibrations WHERE enterprise_id=$1 AND instrument_id=$2 ORDER BY data DESC, id DESC`, empresa, id)
	if err != nil {
		return nil, fmt.Errorf("listando as calibrações: %w", err)
	}
	defer rows.Close()
	out := []dom.Calibracao{}
	for rows.Next() {
		var c dom.Calibracao
		if err := rows.Scan(&c.ID, &c.InstrumentoID, &c.Data, &c.Executor, &c.Laboratorio, &c.Certificado, &c.ErroEncontrado, &c.Incerteza, &c.EMA,
			&c.Resultado, &c.Ajustado, &c.Proxima, &c.Observacoes, &c.CreatedAt, &c.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PG) Impacto(ctx context.Context, id int64, desde time.Time) ([]dom.Impacto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT l.id, l.numero, l.item_code, l.status, d.decisao, COUNT(*), MAX(r.created_at)
		  FROM quality_lot_results r
		  JOIN quality_inspection_lots l ON l.id = r.lot_id AND l.enterprise_id = r.enterprise_id
		  LEFT JOIN quality_lot_decisions d ON d.lot_id = l.id
		 WHERE r.enterprise_id = $1 AND r.instrument_id = $2 AND r.created_at >= $3
		 GROUP BY l.id, l.numero, l.item_code, l.status, d.decisao
		 ORDER BY MAX(r.created_at) DESC`, empresa, id, desde)
	if err != nil {
		return nil, fmt.Errorf("levantando o impacto: %w", err)
	}
	defer rows.Close()
	out := []dom.Impacto{}
	for rows.Next() {
		var x dom.Impacto
		if err := rows.Scan(&x.LoteID, &x.LoteNumero, &x.ItemCode, &x.Status, &x.Decisao, &x.Medicoes, &x.UltimaEm); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *PG) PorIDs(ctx context.Context, ids []int64) (map[int64]dom.Instrumento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+` FROM quality_instruments WHERE enterprise_id=$1 AND id = ANY($2)`, empresa, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]dom.Instrumento{}
	for rows.Next() {
		i, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out[i.ID] = *i
	}
	return out, rows.Err()
}
