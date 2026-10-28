// Package amostragem grava procedimentos de amostragem, o estado de severidade
// de cada unidade de controle e o histórico de trocas, na empresa da sessão.
package amostragem

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var _ dom.Repository = (*PG)(nil)

const colunasProcedimento = `id, codigo, descricao, tipo, COALESCE(nivel, ''), COALESCE(nqa, ''), permite_atenuada,
	amostra_fixa, ac_fixo, re_fixo, COALESCE(skip_apos, 0), COALESCE(skip_frequencia, 0), ativo, updated_at, updated_by`

func scanProcedimento(row pgx.Row) (*dom.Procedimento, error) {
	var p dom.Procedimento
	var tipo, nivel string
	var amostra *int64
	var ac, re *int32
	var apos, freq int32
	if err := row.Scan(&p.ID, &p.Codigo, &p.Descricao, &tipo, &nivel, &p.NQA, &p.PermiteAtenuada,
		&amostra, &ac, &re, &apos, &freq, &p.Ativo, &p.UpdatedAt, &p.UpdatedBy); err != nil {
		return nil, err
	}
	p.Tipo, p.Nivel, p.AmostraFixa = dom.TipoProcedimento(tipo), dom.Nivel(nivel), amostra
	if ac != nil {
		v := int(*ac)
		p.AcFixo = &v
	}
	if re != nil {
		v := int(*re)
		p.ReFixo = &v
	}
	p.Skip = dom.SkipLot{AposAceitos: int(apos), Frequencia: int(freq)}
	return &p, nil
}

func (r *PG) ListarProcedimentos(ctx context.Context, somenteAtivos bool) ([]dom.Procedimento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasProcedimento+` FROM quality_sampling_procedures
		 WHERE enterprise_id = $1 AND (ativo OR NOT $2) ORDER BY codigo`, empresa, somenteAtivos)
	if err != nil {
		return nil, fmt.Errorf("listando os procedimentos de amostragem: %w", err)
	}
	defer rows.Close()
	out := []dom.Procedimento{}
	for rows.Next() {
		p, err := scanProcedimento(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PG) ObterProcedimento(ctx context.Context, id int64) (*dom.Procedimento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	p, err := scanProcedimento(r.pool.QueryRow(ctx, `SELECT `+colunasProcedimento+` FROM quality_sampling_procedures
		 WHERE enterprise_id = $1 AND id = $2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("procedimento de amostragem %d", id))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o procedimento de amostragem: %w", err)
	}
	return p, nil
}

func nulo(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func (r *PG) SalvarProcedimento(ctx context.Context, p *dom.Procedimento) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var nivel, nqa *string
	if p.Nivel != "" {
		s := string(p.Nivel)
		nivel = &s
	}
	if p.NQA != "" {
		nqa = &p.NQA
	}
	args := []any{empresa, p.Codigo, p.Descricao, string(p.Tipo), nivel, nqa, p.PermiteAtenuada,
		p.AmostraFixa, p.AcFixo, p.ReFixo, nulo(p.Skip.AposAceitos), nulo(p.Skip.Frequencia), p.Ativo, p.UpdatedBy}
	if p.ID == 0 {
		err = r.pool.QueryRow(ctx, `
			INSERT INTO quality_sampling_procedures (enterprise_id, codigo, descricao, tipo, nivel, nqa, permite_atenuada,
				amostra_fixa, ac_fixo, re_fixo, skip_apos, skip_frequencia, ativo, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id, updated_at`, args...).Scan(&p.ID, &p.UpdatedAt)
	} else {
		err = r.pool.QueryRow(ctx, `
			UPDATE quality_sampling_procedures SET codigo=$2, descricao=$3, tipo=$4, nivel=$5, nqa=$6, permite_atenuada=$7,
				amostra_fixa=$8, ac_fixo=$9, re_fixo=$10, skip_apos=$11, skip_frequencia=$12, ativo=$13, updated_by=$14, updated_at=NOW()
			 WHERE enterprise_id=$1 AND id=$15 RETURNING updated_at`, append(args, p.ID)...).Scan(&p.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("procedimento de amostragem %d", p.ID))
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewValidationError(fmt.Sprintf("já existe um procedimento de amostragem com o código %s", p.Codigo))
	}
	if err != nil {
		return fmt.Errorf("gravando o procedimento de amostragem: %w", err)
	}
	return nil
}

const colunasEstado = `severidade, pontuacao, ultimos, recusas_em_apertada, suspensa, aceitos_seguidos, lotes_desde_inspecao`

func scanEstado(row pgx.Row, extra ...any) (dom.Estado, error) {
	var e dom.Estado
	var sev string
	dest := append([]any{&sev, &e.Pontuacao, &e.Ultimos, &e.RecusasEmApertada, &e.Suspensa, &e.AceitosSeguidos, &e.LotesDesdeInspecao}, extra...)
	if err := row.Scan(dest...); err != nil {
		return e, err
	}
	e.Severidade = dom.Severidade(sev)
	return e, nil
}

func (r *PG) Estado(ctx context.Context, procedimentoID, item int64, chave string) (dom.Estado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return dom.Estado{}, err
	}
	e, err := scanEstado(r.pool.QueryRow(ctx, `SELECT `+colunasEstado+` FROM quality_sampling_states
		 WHERE enterprise_id=$1 AND procedure_id=$2 AND item_code=$3 AND chave=$4`, empresa, procedimentoID, item, chave))
	if errors.Is(err, pgx.ErrNoRows) {
		return dom.Novo(), nil
	}
	if err != nil {
		return e, fmt.Errorf("lendo o estado da amostragem: %w", err)
	}
	return e, nil
}

func (r *PG) AtualizarEstado(ctx context.Context, procedimentoID, item int64, chave string, referencia *string, ator uuid.UUID, fn dom.Atualizacao) (dom.Estado, *dom.Transicao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return dom.Estado{}, nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return dom.Estado{}, nil, err
	}
	defer tx.Rollback(ctx)
	novo, tr, err := AtualizarEstadoTx(ctx, tx, empresa, procedimentoID, item, chave, referencia, ator, fn)
	if err != nil {
		return dom.Estado{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dom.Estado{}, nil, err
	}
	return novo, tr, nil
}

// AtualizarEstadoTx é AtualizarEstado dentro de uma transação já aberta (a
// decisão de uso do lote grava a troca de severidade junto com a decisão).
func AtualizarEstadoTx(ctx context.Context, tx pgx.Tx, empresa, procedimentoID, item int64, chave string, referencia *string, ator uuid.UUID, fn dom.Atualizacao) (dom.Estado, *dom.Transicao, error) {
	// Garante a linha antes de travar: o primeiro lote cria o estado normal.
	if _, err := tx.Exec(ctx, `INSERT INTO quality_sampling_states (enterprise_id, procedure_id, item_code, chave)
		SELECT $1, $2, $3, $4 WHERE EXISTS (SELECT 1 FROM quality_sampling_procedures WHERE enterprise_id=$1 AND id=$2)
		ON CONFLICT DO NOTHING`, empresa, procedimentoID, item, chave); err != nil {
		return dom.Estado{}, nil, fmt.Errorf("criando o estado da amostragem: %w", err)
	}
	atual, err := scanEstado(tx.QueryRow(ctx, `SELECT `+colunasEstado+` FROM quality_sampling_states
		 WHERE enterprise_id=$1 AND procedure_id=$2 AND item_code=$3 AND chave=$4 FOR UPDATE`, empresa, procedimentoID, item, chave))
	if errors.Is(err, pgx.ErrNoRows) {
		return dom.Estado{}, nil, errorsuc.NewNotFoundError(fmt.Sprintf("procedimento de amostragem %d", procedimentoID))
	}
	if err != nil {
		return dom.Estado{}, nil, fmt.Errorf("travando o estado da amostragem: %w", err)
	}
	novo, tr, err := fn(atual)
	if err != nil {
		return dom.Estado{}, nil, err
	}
	ultimos := novo.Ultimos
	if ultimos == nil {
		ultimos = []bool{}
	}
	if _, err := tx.Exec(ctx, `UPDATE quality_sampling_states SET severidade=$5, pontuacao=$6, ultimos=$7, recusas_em_apertada=$8,
			suspensa=$9, aceitos_seguidos=$10, lotes_desde_inspecao=$11, updated_at=NOW()
		 WHERE enterprise_id=$1 AND procedure_id=$2 AND item_code=$3 AND chave=$4`,
		empresa, procedimentoID, item, chave, string(novo.Severidade), novo.Pontuacao, ultimos, novo.RecusasEmApertada,
		novo.Suspensa, novo.AceitosSeguidos, novo.LotesDesdeInspecao); err != nil {
		return dom.Estado{}, nil, fmt.Errorf("gravando o estado da amostragem: %w", err)
	}
	if tr != nil {
		evento := "TROCA"
		switch {
		case tr.Suspensao:
			evento = "SUSPENSAO"
		case tr.Liberacao:
			evento = "LIBERACAO"
		}
		var por *uuid.UUID
		if ator != uuid.Nil {
			por = &ator
		}
		if _, err := tx.Exec(ctx, `INSERT INTO quality_sampling_events
			(enterprise_id, procedure_id, item_code, chave, de, para, evento, motivo, referencia, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, empresa, procedimentoID, item, chave,
			string(tr.De), string(tr.Para), evento, tr.Motivo, referencia, por); err != nil {
			return dom.Estado{}, nil, fmt.Errorf("gravando o histórico da amostragem: %w", err)
		}
	}
	return novo, tr, nil
}

func (r *PG) EstadosDoItem(ctx context.Context, item int64) ([]dom.EstadoDoItem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT s.severidade, s.pontuacao, s.ultimos, s.recusas_em_apertada, s.suspensa, s.aceitos_seguidos,
			s.lotes_desde_inspecao, s.procedure_id, p.codigo, s.item_code, s.chave, s.updated_at
		  FROM quality_sampling_states s
		  JOIN quality_sampling_procedures p ON p.enterprise_id = s.enterprise_id AND p.id = s.procedure_id
		 WHERE s.enterprise_id = $1 AND ($2 = 0 OR s.item_code = $2)
		 ORDER BY s.suspensa DESC, s.item_code, p.codigo, s.chave LIMIT 500`, empresa, item)
	if err != nil {
		return nil, fmt.Errorf("listando os estados da amostragem: %w", err)
	}
	defer rows.Close()
	out := []dom.EstadoDoItem{}
	for rows.Next() {
		var x dom.EstadoDoItem
		e, err := scanEstado(rows, &x.ProcedimentoID, &x.Procedimento, &x.ItemCode, &x.Chave, &x.UpdatedAt)
		if err != nil {
			return nil, err
		}
		x.Estado = e
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *PG) Eventos(ctx context.Context, item int64, limite int) ([]dom.Evento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if limite <= 0 || limite > 500 {
		limite = 200
	}
	rows, err := r.pool.Query(ctx, `SELECT id, procedure_id, item_code, chave, de, para, evento, motivo, referencia, created_at, created_by
		  FROM quality_sampling_events WHERE enterprise_id = $1 AND ($2 = 0 OR item_code = $2)
		 ORDER BY created_at DESC, id DESC LIMIT $3`, empresa, item, limite)
	if err != nil {
		return nil, fmt.Errorf("listando o histórico da amostragem: %w", err)
	}
	defer rows.Close()
	out := []dom.Evento{}
	for rows.Next() {
		var ev dom.Evento
		var de, para string
		if err := rows.Scan(&ev.ID, &ev.ProcedimentoID, &ev.ItemCode, &ev.Chave, &de, &para, &ev.Tipo, &ev.Motivo, &ev.Referencia, &ev.CreatedAt, &ev.CreatedBy); err != nil {
			return nil, err
		}
		ev.De, ev.Para = dom.Severidade(de), dom.Severidade(para)
		out = append(out, ev)
	}
	return out, rows.Err()
}
