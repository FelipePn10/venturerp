// Package recuperacao grava as reclamações de custo de garantia ao fornecedor
// e a ligação do chamado com a CAPA da qualidade, sempre da empresa da sessão.
package recuperacao

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/recuperacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

const colunas = `r.id, r.numero, r.supplier_code, COALESCE(s.trade_name, s.name, ''), r.os_id, o.numero, r.call_code, c.call_number, r.item_code, COALESCE(i.name,''),
	r.capa_id, r.motivo, r.valor_reclamado, r.valor_recuperado, r.situacao, r.forma, r.documento, r.resposta, r.recuperado_em, r.created_at, r.updated_at`

const de = ` FROM recuperacoes_fornecedor r
	LEFT JOIN suppliers s ON s.code = r.supplier_code AND s.enterprise_id = r.enterprise_id
	LEFT JOIN ordens_servico o ON o.id = r.os_id AND o.enterprise_id = r.enterprise_id
	LEFT JOIN technical_assistance_calls c ON c.code = r.call_code
	LEFT JOIN items i ON i.code = r.item_code AND i.enterprise_id = r.enterprise_id`

func scan(row pgx.Row) (*dom.Reclamacao, error) {
	var r dom.Reclamacao
	err := row.Scan(&r.ID, &r.Numero, &r.SupplierCode, &r.FornecedorNome, &r.OSID, &r.OSNumero, &r.CallCode, &r.CallNumero, &r.ItemCode, &r.ItemNome,
		&r.CapaID, &r.Motivo, &r.ValorReclamado, &r.ValorRecuperado, &r.Situacao, &r.Forma, &r.Documento, &r.Resposta, &r.RecuperadoEm, &r.CreatedAt, &r.UpdatedAt)
	return &r, err
}

func existe(ctx context.Context, tx pgx.Tx, sql string, args ...any) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(`+sql+`)`, args...).Scan(&ok)
	return ok, err
}

func (p *PG) Abrir(ctx context.Context, r *dom.Reclamacao, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if ok, err := existe(ctx, tx, `SELECT 1 FROM suppliers WHERE code=$1 AND enterprise_id=$2`, r.SupplierCode, empresa); err != nil {
		return err
	} else if !ok {
		return errorsuc.NewNotFoundError(fmt.Sprintf("fornecedor %d", r.SupplierCode))
	}
	if r.OSID != nil {
		// A OS dá o chamado, o item (se não veio) e a CAPA do chamado.
		var call, item *int64
		err := tx.QueryRow(ctx, `SELECT o.call_code, o.item_code FROM ordens_servico o WHERE o.enterprise_id=$1 AND o.id=$2`, empresa, *r.OSID).Scan(&call, &item)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("ordem de serviço %d", *r.OSID))
		}
		if err != nil {
			return err
		}
		r.CallCode = call
		if r.ItemCode == nil {
			r.ItemCode = item
		}
	}
	if r.CallCode != nil {
		err := tx.QueryRow(ctx, `SELECT c.capa_id FROM technical_assistance_calls c WHERE c.code=$2 AND c.enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
			empresa, *r.CallCode).Scan(&r.CapaID)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("chamado %d", *r.CallCode))
		}
		if err != nil {
			return err
		}
	}
	if r.ItemCode != nil {
		if ok, err := existe(ctx, tx, `SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2`, *r.ItemCode, empresa); err != nil {
			return err
		} else if !ok {
			return errorsuc.NewNotFoundError(fmt.Sprintf("item %d", *r.ItemCode))
		}
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('recuperacoes_fornecedor'), $1::int)`, empresa); err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `INSERT INTO recuperacoes_fornecedor (enterprise_id, numero, supplier_code, os_id, call_code, item_code, capa_id, motivo, valor_reclamado, created_by, updated_by)
		VALUES ($1, (SELECT COALESCE(MAX(numero),0)+1 FROM recuperacoes_fornecedor WHERE enterprise_id=$1), $2,$3,$4,$5,$6,$7,$8,$9,$9)
		RETURNING id, numero`, empresa, r.SupplierCode, r.OSID, r.CallCode, r.ItemCode, r.CapaID, r.Motivo, r.ValorReclamado.Round(2), por).Scan(&r.ID, &r.Numero)
	if err != nil {
		return fmt.Errorf("gravando a reclamação ao fornecedor: %w", err)
	}
	return tx.Commit(ctx)
}

func (p *PG) Obter(ctx context.Context, id int64) (*dom.Reclamacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	r, err := scan(p.pool.QueryRow(ctx, `SELECT `+colunas+de+` WHERE r.enterprise_id=$1 AND r.id=$2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("reclamação ao fornecedor %d", id))
	}
	return r, err
}

func (p *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.Reclamacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := p.pool.Query(ctx, `SELECT `+colunas+de+` WHERE r.enterprise_id=$1
		   AND ($2 = '' OR r.situacao = $2) AND ($3::bigint IS NULL OR r.supplier_code = $3) AND ($4::bigint IS NULL OR r.os_id = $4)
		 ORDER BY r.numero DESC LIMIT 500`, empresa, f.Situacao, f.Fornecedor, f.OSID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Reclamacao{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (p *PG) Mudar(ctx context.Context, id int64, m dom.Mudanca) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var atual string
	var reclamado = m.ValorRecuperado
	err = tx.QueryRow(ctx, `SELECT situacao, valor_reclamado FROM recuperacoes_fornecedor WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, id).Scan(&atual, &reclamado)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("reclamação ao fornecedor %d", id))
	}
	if err != nil {
		return err
	}
	if err := m.Conferir(atual, reclamado); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE recuperacoes_fornecedor SET situacao=$3::text, resposta=COALESCE($4, resposta),
			valor_recuperado=CASE WHEN $3::text='RECUPERADA' THEN $5 ELSE valor_recuperado END,
			forma=COALESCE($6, forma), documento=COALESCE($7, documento), recuperado_em=COALESCE($8, recuperado_em),
			updated_by=$9, updated_at=NOW()
		WHERE enterprise_id=$1 AND id=$2`, empresa, id, m.Para, m.Resposta, m.ValorRecuperado.Round(2), m.Forma, m.Documento, m.RecuperadoEm, m.Por)
	if err != nil {
		return fmt.Errorf("atualizando a reclamação ao fornecedor: %w", err)
	}
	return tx.Commit(ctx)
}

func (p *PG) ChamadoParaCAPA(ctx context.Context, callCode int64) (*dom.ChamadoCAPA, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	c := dom.ChamadoCAPA{CallCode: callCode, Itens: map[int64]int64{}}
	err = p.pool.QueryRow(ctx, `SELECT c.call_number, c.customer_code, c.subject, c.capa_id FROM technical_assistance_calls c
		 WHERE c.code=$2 AND c.enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`, empresa, callCode).Scan(&c.CallNumber, &c.CustomerCode, &c.Assunto, &c.CapaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("chamado %d", callCode))
	}
	if err != nil {
		return nil, err
	}
	rows, err := p.pool.Query(ctx, `SELECT code, item_code FROM technical_assistance_call_items WHERE call_code=$1 ORDER BY sequence, code`, callCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var code, item int64
		if err := rows.Scan(&code, &item); err != nil {
			return nil, err
		}
		c.Itens[code] = item
	}
	return &c, rows.Err()
}

func (p *PG) VincularCAPA(ctx context.Context, callCode, capaID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := p.pool.Exec(ctx, `UPDATE technical_assistance_calls c SET capa_id=$3, updated_at=NOW()
		 WHERE c.code=$2 AND c.capa_id IS NULL AND c.enterprise_code=(SELECT code FROM enterprise WHERE id=$1)
		   AND EXISTS (SELECT 1 FROM quality_capas q WHERE q.id=$3 AND q.enterprise_id=$1)`, empresa, callCode, capaID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("o chamado já tem CAPA ligada")
	}
	return nil
}
