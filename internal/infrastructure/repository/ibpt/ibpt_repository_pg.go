package ibpt

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/ibpt/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IBPTRepositoryPG struct {
	pool *pgxpool.Pool
}

func NewIBPTRepositoryPG(pool *pgxpool.Pool) *IBPTRepositoryPG {
	return &IBPTRepositoryPG{pool: pool}
}

var _ repository.IBPTRepository = (*IBPTRepositoryPG)(nil)

func (r *IBPTRepositoryPG) BulkUpsert(ctx context.Context, rates []*entity.IBPTRate) (int, error) {
	if len(rates) == 0 {
		return 0, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin ibpt tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const q = `INSERT INTO public.ibpt_rates
		(ncm, ex, uf, tipo, descricao, nacional_federal, importado_federal, estadual, municipal,
		 vigencia_inicio, vigencia_fim, chave, versao, fonte)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (ncm, ex, uf, versao) DO UPDATE SET
			tipo = EXCLUDED.tipo, descricao = EXCLUDED.descricao,
			nacional_federal = EXCLUDED.nacional_federal, importado_federal = EXCLUDED.importado_federal,
			estadual = EXCLUDED.estadual, municipal = EXCLUDED.municipal,
			vigencia_inicio = EXCLUDED.vigencia_inicio, vigencia_fim = EXCLUDED.vigencia_fim,
			chave = EXCLUDED.chave, fonte = EXCLUDED.fonte`

	n := 0
	for _, rt := range rates {
		ex := rt.Ex
		if ex == "" {
			ex = "0"
		}
		fonte := rt.Fonte
		if fonte == "" {
			fonte = "IBPT"
		}
		if _, err := tx.Exec(ctx, q,
			rt.NCM, ex, rt.UF, rt.Tipo, rt.Descricao, rt.NacionalFederal, rt.ImportadoFederal,
			rt.Estadual, rt.Municipal, rt.VigenciaInicio, rt.VigenciaFim, rt.Chave, rt.Versao, fonte,
		); err != nil {
			return 0, fmt.Errorf("upserting ibpt rate (ncm %s): %w", rt.NCM, err)
		}
		n++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit ibpt tx: %w", err)
	}
	return n, nil
}

func (r *IBPTRepositoryPG) GetByNCM(ctx context.Context, ncm, uf string) (*entity.IBPTRate, error) {
	var e entity.IBPTRate
	err := r.pool.QueryRow(ctx,
		`SELECT id, ncm, ex, uf, tipo, descricao, nacional_federal, importado_federal, estadual, municipal,
		        vigencia_inicio, vigencia_fim, chave, versao, fonte, created_at
		 FROM public.ibpt_rates
		 WHERE ncm = $1 AND uf = $2
		 ORDER BY vigencia_inicio DESC NULLS LAST, id DESC
		 LIMIT 1`, ncm, uf,
	).Scan(&e.ID, &e.NCM, &e.Ex, &e.UF, &e.Tipo, &e.Descricao, &e.NacionalFederal, &e.ImportadoFederal,
		&e.Estadual, &e.Municipal, &e.VigenciaInicio, &e.VigenciaFim, &e.Chave, &e.Versao, &e.Fonte, &e.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("IBPT não encontrado para NCM %s/%s", ncm, uf)
		}
		return nil, fmt.Errorf("getting ibpt rate: %w", err)
	}
	return &e, nil
}
