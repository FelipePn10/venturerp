// Package certificado grava os certificados de qualidade e lê, da empresa da
// sessão, o que eles imprimem (empresa, cliente, item, NF, corrida).
package certificado

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/certificado"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var (
	_ dom.Repository = (*PG)(nil)
	_ dom.Fontes     = (*PG)(nil)
)

const colunas = `id, numero, customer_code, fiscal_exit_id, sales_order_code, item_code, mask, lote, quantidade, corrida, certificado_usina,
	lotes_inspecao, conteudo, observacoes, status, motivo_cancelamento, emitido_em, emitido_por, cancelado_em`

func scan(row pgx.Row) (*dom.Certificado, error) {
	var c dom.Certificado
	var conteudo []byte
	if err := row.Scan(&c.ID, &c.Numero, &c.CustomerCode, &c.FiscalExitID, &c.SalesOrderCode, &c.ItemCode, &c.Mask, &c.Lote, &c.Quantidade,
		&c.Corrida, &c.CertificadoUsina, &c.LotesInspecao, &conteudo, &c.Observacoes, &c.Status, &c.MotivoCancelamento, &c.EmitidoEm,
		&c.EmitidoPor, &c.CanceladoEm); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(conteudo, &c.Conteudo); err != nil {
		return nil, fmt.Errorf("lendo o conteúdo do certificado: %w", err)
	}
	return &c, nil
}

func (r *PG) Emitir(ctx context.Context, c *dom.Certificado, limite decimal.Decimal) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('quality_certificate:' || $1::bigint, 0))`, empresa); err != nil {
		return err
	}
	var ja decimal.Decimal
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(quantidade), 0) FROM quality_certificates
		 WHERE enterprise_id=$1 AND status='EMITIDO' AND lotes_inspecao && $2`, empresa, c.LotesInspecao).Scan(&ja); err != nil {
		return err
	}
	if c.Quantidade.Add(ja).GreaterThan(limite) {
		return errorsuc.NewConflictError(fmt.Sprintf("outro certificado foi emitido sobre esses lotes: já há %s certificado de %s aprovado", ja, limite))
	}
	conteudo, err := json.Marshal(c.Conteudo)
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `INSERT INTO quality_certificates (enterprise_id, numero, customer_code, fiscal_exit_id, sales_order_code, item_code, mask, lote,
			quantidade, corrida, certificado_usina, lotes_inspecao, conteudo, observacoes, emitido_por)
		SELECT $1, COALESCE((SELECT MAX(numero) FROM quality_certificates WHERE enterprise_id = $1), 0) + 1, $2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
		 WHERE EXISTS (SELECT 1 FROM items WHERE code = $5 AND enterprise_id = $1)
		RETURNING id, numero, status, emitido_em`,
		empresa, c.CustomerCode, c.FiscalExitID, c.SalesOrderCode, c.ItemCode, c.Mask, c.Lote, c.Quantidade, c.Corrida, c.CertificadoUsina,
		c.LotesInspecao, conteudo, c.Observacoes, c.EmitidoPor).Scan(&c.ID, &c.Numero, &c.Status, &c.EmitidoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewValidationError("o item não existe nesta empresa")
	}
	if err != nil {
		return fmt.Errorf("emitindo o certificado: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.Certificado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := scan(r.pool.QueryRow(ctx, `SELECT `+colunas+` FROM quality_certificates WHERE enterprise_id=$1 AND id=$2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("certificado %d", id))
	}
	return c, err
}

func (r *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.Certificado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+` FROM quality_certificates
		 WHERE enterprise_id=$1 AND ($2 = 0 OR item_code=$2) AND ($3 = 0 OR customer_code=$3) AND ($4 = '' OR status=$4)
		 ORDER BY numero DESC LIMIT 300`, empresa, f.ItemCode, f.CustomerCode, strings.ToUpper(f.Status))
	if err != nil {
		return nil, fmt.Errorf("listando os certificados: %w", err)
	}
	defer rows.Close()
	out := []dom.Certificado{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *PG) Cancelar(ctx context.Context, id int64, motivo string, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE quality_certificates SET status='CANCELADO', motivo_cancelamento=$3, cancelado_em=NOW(), cancelado_por=$4
		 WHERE enterprise_id=$1 AND id=$2 AND status='EMITIDO'`, empresa, id, motivo, ator)
	if err != nil {
		return fmt.Errorf("cancelando o certificado: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("certificado não encontrado ou já cancelado")
	}
	return nil
}

// ─── fontes ───────────────────────────────────────────────────────────────────

func (r *PG) Empresa(ctx context.Context) (dom.Parte, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return dom.Parte{}, err
	}
	var p dom.Parte
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(NULLIF(fc.razao_social, ''), e.name), COALESCE(fc.cnpj_empresa, '')
		  FROM enterprise e LEFT JOIN fiscal_configs fc ON fc.enterprise_id = e.id WHERE e.id = $1`, empresa).Scan(&p.Nome, &p.CNPJ)
	return p, err
}

func (r *PG) Cliente(ctx context.Context, codigo int64) (*dom.Parte, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	p := dom.Parte{Codigo: codigo}
	err = r.pool.QueryRow(ctx, `SELECT name, COALESCE(document_number, '') FROM customers WHERE code=$1 AND enterprise_id=$2`, codigo, empresa).Scan(&p.Nome, &p.CNPJ)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("cliente %d não encontrado nesta empresa", codigo))
	}
	return &p, err
}

func (r *PG) Item(ctx context.Context, codigo int64) (string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return "", err
	}
	var nome string
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(name, '') FROM items WHERE code=$1 AND enterprise_id=$2`, codigo, empresa).Scan(&nome)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errorsuc.NewValidationError(fmt.Sprintf("item %d não encontrado nesta empresa", codigo))
	}
	return nome, err
}

func (r *PG) NotaFiscal(ctx context.Context, id int64) (*dom.NotaFiscal, *int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, nil, err
	}
	var nf dom.NotaFiscal
	var cliente *int64
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(numero_nf, 0), COALESCE(serie, ''), data_emissao, customer_code FROM fiscal_exits
		 WHERE id=$1 AND enterprise_id=$2`, id, empresa).Scan(&nf.Numero, &nf.Serie, &nf.Data, &cliente)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, errorsuc.NewValidationError(fmt.Sprintf("NF de saída %d não encontrada nesta empresa", id))
	}
	return &nf, cliente, err
}

func (r *PG) Corrida(ctx context.Context, item int64, lote string) (string, string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return "", "", err
	}
	var corrida, cert string
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(heat_number, ''), COALESCE(certificate, '') FROM stock_lots
		 WHERE enterprise_id=$1 AND item_code=$2 AND lot=$3 ORDER BY id DESC LIMIT 1`, empresa, item, lote).Scan(&corrida, &cert)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	return corrida, cert, err
}

func (r *PG) Instrumentos(ctx context.Context, ids []int64) (map[int64]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id, codigo FROM quality_instruments WHERE enterprise_id=$1 AND id = ANY($2)`, empresa, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var cod string
		if err := rows.Scan(&id, &cod); err != nil {
			return nil, err
		}
		out[id] = cod
	}
	return out, rows.Err()
}

func (r *PG) JaCertificado(ctx context.Context, lotes []int64) (decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	var total decimal.Decimal
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(quantidade), 0) FROM quality_certificates
		 WHERE enterprise_id=$1 AND status='EMITIDO' AND lotes_inspecao && $2`, empresa, lotes).Scan(&total)
	return total, err
}
