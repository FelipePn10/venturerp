// Package documento grava documentos controlados e suas revisões.
package documento

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/documento"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var _ dom.Repository = (*PG)(nil)

const colunasRev = `id, document_id, revisao, status, descricao_mudanca, conteudo, arquivo_nome, arquivo_tipo, arquivo IS NOT NULL,
	elaborado_por, elaborado_em, aprovado_por, vigente_desde, obsoleta_em`

func scanRev(row pgx.Row) (*dom.Revisao, error) {
	var r dom.Revisao
	if err := row.Scan(&r.ID, &r.DocumentoID, &r.Revisao, &r.Status, &r.DescricaoMudanca, &r.Conteudo, &r.ArquivoNome, &r.ArquivoTipo,
		&r.TemArquivo, &r.ElaboradoPor, &r.ElaboradoEm, &r.AprovadoPor, &r.VigenteDesde, &r.ObsoletaEm); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *PG) Salvar(ctx context.Context, d *dom.Documento, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if d.ItemCode != nil {
		var ok bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2)`, *d.ItemCode, empresa).Scan(&ok); err != nil || !ok {
			return errorsuc.NewValidationError("o item do documento não existe nesta empresa")
		}
	}
	if d.ID == 0 {
		err = r.pool.QueryRow(ctx, `INSERT INTO quality_documents (enterprise_id, codigo, titulo, tipo, item_code, route_operation_id, area, ativo, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at`, empresa, d.Codigo, d.Titulo, d.Tipo, d.ItemCode, d.RouteOperationID, d.Area, d.Ativo, ator).
			Scan(&d.ID, &d.CreatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = r.pool.Exec(ctx, `UPDATE quality_documents SET codigo=$3, titulo=$4, tipo=$5, item_code=$6, route_operation_id=$7, area=$8, ativo=$9
			 WHERE enterprise_id=$1 AND id=$2`, empresa, d.ID, d.Codigo, d.Titulo, d.Tipo, d.ItemCode, d.RouteOperationID, d.Area, d.Ativo)
		if err == nil && tag.RowsAffected() == 0 {
			return errorsuc.NewNotFoundError(fmt.Sprintf("documento %d", d.ID))
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewValidationError(fmt.Sprintf("já existe um documento com o código %s", d.Codigo))
	}
	if err != nil {
		return fmt.Errorf("gravando o documento: %w", err)
	}
	return nil
}

func (r *PG) cabecalho(row pgx.Row) (*dom.Documento, error) {
	var d dom.Documento
	if err := row.Scan(&d.ID, &d.Codigo, &d.Titulo, &d.Tipo, &d.ItemCode, &d.RouteOperationID, &d.Area, &d.Ativo, &d.CreatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.Documento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	d, err := r.cabecalho(r.pool.QueryRow(ctx, `SELECT id, codigo, titulo, tipo, item_code, route_operation_id, area, ativo, created_at
		  FROM quality_documents WHERE enterprise_id=$1 AND id=$2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("documento %d", id))
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasRev+` FROM quality_document_revisions WHERE document_id=$1 AND enterprise_id=$2 ORDER BY revisao DESC`, id, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		rev, err := scanRev(rows)
		if err != nil {
			return nil, err
		}
		d.Revisoes = append(d.Revisoes, *rev)
		switch rev.Status {
		case "VIGENTE":
			v := *rev
			d.Vigente = &v
		case "RASCUNHO":
			v := *rev
			d.Rascunho = &v
		}
	}
	return d, rows.Err()
}

func (r *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.Documento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	busca := "%" + strings.ToUpper(strings.TrimSpace(f.Busca)) + "%"
	rows, err := r.pool.Query(ctx, `SELECT d.id, d.codigo, d.titulo, d.tipo, d.item_code, d.route_operation_id, d.area, d.ativo, d.created_at,
			v.revisao, v.vigente_desde, (SELECT COUNT(*) FROM quality_document_revisions x WHERE x.document_id = d.id AND x.status = 'RASCUNHO') > 0
		  FROM quality_documents d
		  LEFT JOIN quality_document_revisions v ON v.document_id = d.id AND v.status = 'VIGENTE'
		 WHERE d.enterprise_id=$1 AND ($2 = '' OR d.tipo=$2) AND ($3 = 0 OR d.item_code=$3) AND ($5 OR d.ativo)
		   AND (UPPER(d.codigo) LIKE $4 OR UPPER(d.titulo) LIKE $4)
		 ORDER BY d.codigo LIMIT 500`, empresa, strings.ToUpper(f.Tipo), f.ItemCode, busca, f.Todos)
	if err != nil {
		return nil, fmt.Errorf("listando os documentos: %w", err)
	}
	defer rows.Close()
	out := []dom.Documento{}
	for rows.Next() {
		var d dom.Documento
		var rev *int
		var desde *time.Time
		var rascunho bool
		if err := rows.Scan(&d.ID, &d.Codigo, &d.Titulo, &d.Tipo, &d.ItemCode, &d.RouteOperationID, &d.Area, &d.Ativo, &d.CreatedAt, &rev, &desde, &rascunho); err != nil {
			return nil, err
		}
		if rev != nil {
			d.Vigente = &dom.Revisao{Revisao: *rev, Status: "VIGENTE"}
			if desde != nil {
				d.Vigente.VigenteDesde = desde
			}
		}
		if rascunho {
			d.Rascunho = &dom.Revisao{Status: "RASCUNHO"}
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PG) GravarRascunho(ctx context.Context, docID int64, rev *dom.Revisao) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ativo bool
	err = tx.QueryRow(ctx, `SELECT ativo FROM quality_documents WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, docID).Scan(&ativo)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("documento %d", docID))
	}
	if err != nil {
		return err
	}
	if !ativo {
		return errorsuc.NewValidationError("documento inativo não recebe revisão")
	}
	var atual *int64
	var proxima int
	if err := tx.QueryRow(ctx, `SELECT (SELECT id FROM quality_document_revisions WHERE document_id=$1 AND status='RASCUNHO'),
			COALESCE((SELECT MAX(revisao) + 1 FROM quality_document_revisions WHERE document_id=$1 AND status <> 'RASCUNHO'), 0)`, docID).Scan(&atual, &proxima); err != nil {
		return err
	}
	var arquivo []byte
	if len(rev.Arquivo) > 0 {
		arquivo = rev.Arquivo
	}
	if atual == nil {
		err = tx.QueryRow(ctx, `INSERT INTO quality_document_revisions (document_id, enterprise_id, revisao, descricao_mudanca, conteudo, arquivo_nome, arquivo_tipo, arquivo, elaborado_por)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, revisao, status, elaborado_em`,
			docID, empresa, proxima, rev.DescricaoMudanca, rev.Conteudo, rev.ArquivoNome, rev.ArquivoTipo, arquivo, rev.ElaboradoPor).
			Scan(&rev.ID, &rev.Revisao, &rev.Status, &rev.ElaboradoEm)
	} else {
		// Arquivo novo substitui; sem arquivo novo, mantém o anterior.
		err = tx.QueryRow(ctx, `UPDATE quality_document_revisions SET descricao_mudanca=$2, conteudo=$3,
				arquivo_nome=CASE WHEN $6::bytea IS NULL THEN arquivo_nome ELSE $4 END,
				arquivo_tipo=CASE WHEN $6::bytea IS NULL THEN arquivo_tipo ELSE $5 END,
				arquivo=COALESCE($6, arquivo), elaborado_por=$7, elaborado_em=NOW()
			 WHERE id=$1 RETURNING id, revisao, status, elaborado_em`,
			*atual, rev.DescricaoMudanca, rev.Conteudo, rev.ArquivoNome, rev.ArquivoTipo, arquivo, rev.ElaboradoPor).
			Scan(&rev.ID, &rev.Revisao, &rev.Status, &rev.ElaboradoEm)
	}
	if err != nil {
		return fmt.Errorf("gravando o rascunho: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *PG) Aprovar(ctx context.Context, docID int64, aprovador uuid.UUID) (*dom.Revisao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rev, err := scanRev(tx.QueryRow(ctx, `SELECT `+colunasRev+` FROM quality_document_revisions
		 WHERE document_id=$1 AND enterprise_id=$2 AND status='RASCUNHO' FOR UPDATE`, docID, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewValidationError("o documento não tem revisão em rascunho para aprovar")
	}
	if err != nil {
		return nil, err
	}
	if err := rev.PodeAprovar(aprovador); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE quality_document_revisions SET status='OBSOLETA', obsoleta_em=NOW()
		 WHERE document_id=$1 AND enterprise_id=$2 AND status='VIGENTE'`, docID, empresa); err != nil {
		return nil, fmt.Errorf("tornando a revisão anterior obsoleta: %w", err)
	}
	if err := tx.QueryRow(ctx, `UPDATE quality_document_revisions SET status='VIGENTE', aprovado_por=$2, vigente_desde=NOW()
		 WHERE id=$1 RETURNING status, aprovado_por, vigente_desde`, rev.ID, aprovador).Scan(&rev.Status, &rev.AprovadoPor, &rev.VigenteDesde); err != nil {
		return nil, fmt.Errorf("aprovando a revisão: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return rev, nil
}

func (r *PG) DescartarRascunho(ctx context.Context, docID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM quality_document_revisions WHERE document_id=$1 AND enterprise_id=$2 AND status='RASCUNHO'`, docID, empresa)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("não há rascunho para descartar")
	}
	return nil
}

func (r *PG) Arquivo(ctx context.Context, revID int64) (*dom.Revisao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var rev dom.Revisao
	err = r.pool.QueryRow(ctx, `SELECT id, document_id, revisao, status, arquivo_nome, arquivo_tipo, arquivo FROM quality_document_revisions
		 WHERE id=$1 AND enterprise_id=$2 AND arquivo IS NOT NULL`, revID, empresa).
		Scan(&rev.ID, &rev.DocumentoID, &rev.Revisao, &rev.Status, &rev.ArquivoNome, &rev.ArquivoTipo, &rev.Arquivo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("arquivo da revisão")
	}
	return &rev, err
}
