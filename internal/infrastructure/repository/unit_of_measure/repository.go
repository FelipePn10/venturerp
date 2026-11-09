// Package unit_of_measure persiste o catálogo de unidades por empresa, os
// sinônimos e as conversões padrão (global_unit_conversions).
package unit_of_measure

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/uomgraph"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) domainrepo.UnitRepository { return &Repository{pool: pool} }

// NewCatalogo devolve o tipo concreto, para quem só confere códigos no catálogo.
func NewCatalogo(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const colunasUnidade = `id, code, description, dimension, fiscal_code, decimals, is_system, is_active, created_by, created_at, updated_at`

func scanUnidade(row pgx.Row) (*entity.Unit, error) {
	var u entity.Unit
	var dim string
	if err := row.Scan(&u.ID, &u.Code, &u.Description, &dim, &u.FiscalCode, &u.Decimals, &u.IsSystem, &u.IsActive, &u.CreatedBy, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	u.Dimension = entity.Dimensao(dim)
	return &u, nil
}

func conflito(err error, msg string) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewConflictError(msg)
	}
	if errors.As(err, &pg) && pg.Code == "23503" {
		return errorsuc.NewValidationError("a unidade informada não existe no catálogo desta empresa")
	}
	return err
}

func (r *Repository) ListUnits(ctx context.Context, includeInactive bool) ([]*entity.Unit, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasUnidade+` FROM units_of_measure
		WHERE enterprise_id = $1 AND ($2 OR is_active) ORDER BY dimension, code`, empresa, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.Unit{}
	for rows.Next() {
		u, err := scanUnidade(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) GetUnit(ctx context.Context, code string) (*entity.Unit, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	u, err := scanUnidade(r.pool.QueryRow(ctx, `SELECT `+colunasUnidade+` FROM units_of_measure
		WHERE enterprise_id = $1 AND code = $2`, empresa, entity.NormalizarCodigo(code)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("a unidade %s não existe no catálogo desta empresa", entity.NormalizarCodigo(code)))
	}
	return u, err
}

func (r *Repository) CreateUnit(ctx context.Context, u *entity.Unit) (*entity.Unit, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out, err := scanUnidade(r.pool.QueryRow(ctx, `INSERT INTO units_of_measure
		(enterprise_id, code, description, dimension, fiscal_code, decimals, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING `+colunasUnidade,
		empresa, u.Code, u.Description, string(u.Dimension), u.FiscalCode, u.Decimals, u.CreatedBy))
	if err != nil {
		return nil, conflito(err, fmt.Sprintf("a unidade %s já existe no catálogo", u.Code))
	}
	return out, nil
}

// UpdateUnit altera descrição, dimensão, código fiscal e casas — o código é a
// identidade e não muda (itens, estruturas e conversões o referenciam).
func (r *Repository) UpdateUnit(ctx context.Context, u *entity.Unit) (*entity.Unit, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out, err := scanUnidade(r.pool.QueryRow(ctx, `UPDATE units_of_measure
		SET description = $3, dimension = $4, fiscal_code = $5, decimals = $6, updated_at = NOW()
		WHERE enterprise_id = $1 AND code = $2 RETURNING `+colunasUnidade,
		empresa, u.Code, u.Description, string(u.Dimension), u.FiscalCode, u.Decimals))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("a unidade %s não existe no catálogo desta empresa", u.Code))
	}
	return out, err
}

func (r *Repository) SetUnitActive(ctx context.Context, code string, active bool) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE units_of_measure SET is_active = $3, updated_at = NOW()
		WHERE enterprise_id = $1 AND code = $2`, empresa, entity.NormalizarCodigo(code), active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("a unidade %s não existe no catálogo desta empresa", code))
	}
	return nil
}

func (r *Repository) UnitUsage(ctx context.Context, code string) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var n int64
	err = r.pool.QueryRow(ctx, `SELECT
		  (SELECT count(*) FROM items i WHERE i.enterprise_id = $1
		     AND $2 IN (upper(i.warehouse_unit_of_measurement), upper(trim(COALESCE(i.supplies_purchase_uom,''))),
		                upper(trim(COALESCE(i.accounting_purchase_unit_of_measurement,''))),
		                upper(trim(COALESCE(i.accounting_sale_unit_of_measurement,''))), upper(trim(COALESCE(i.warehouse_secondary_uom,'')))))
		+ (SELECT count(*) FROM item_structures s JOIN items i ON i.code = s.child_code AND i.enterprise_id = $1
		     WHERE s.is_active AND upper(s.unit_of_measurement) = $2)
		+ (SELECT count(*) FROM item_unit_conversions c JOIN items i ON i.code = c.item_code AND i.enterprise_id = $1
		     WHERE c.is_active AND $2 IN (c.from_uom, c.to_uom))
		+ (SELECT count(*) FROM item_lot_unit_conversions l JOIN items i ON i.code = l.item_code AND i.enterprise_id = $1
		     WHERE l.is_active AND $2 IN (l.from_uom, l.to_uom))
		+ (SELECT count(*) FROM global_unit_conversions g
		     WHERE g.enterprise_id = $1 AND g.is_active AND $2 IN (g.from_uom, g.to_uom))`, empresa, entity.NormalizarCodigo(code)).Scan(&n)
	return n, err
}

// ─── Sinônimos ───────────────────────────────────────────────────────────────

func (r *Repository) ListAliases(ctx context.Context) ([]*entity.Alias, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT a.id, a.alias, a.unit_code, a.supplier_code, COALESCE(s.name, ''), a.is_system, a.created_by, a.created_at
		  FROM unit_of_measure_aliases a
		  LEFT JOIN suppliers s ON s.code = a.supplier_code AND s.enterprise_id = a.enterprise_id
		 WHERE a.enterprise_id = $1
		 ORDER BY a.unit_code, a.supplier_code NULLS FIRST, a.alias`, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.Alias{}
	for rows.Next() {
		var a entity.Alias
		if err := rows.Scan(&a.ID, &a.Alias, &a.UnitCode, &a.SupplierCode, &a.SupplierName, &a.IsSystem, &a.CreatedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *Repository) CreateAlias(ctx context.Context, a *entity.Alias) (*entity.Alias, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := *a
	err = r.pool.QueryRow(ctx, `INSERT INTO unit_of_measure_aliases (enterprise_id, alias, unit_code, supplier_code, created_by)
		VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`, empresa, a.Alias, a.UnitCode, a.SupplierCode, a.CreatedBy).
		Scan(&out.ID, &out.CreatedAt)
	if err != nil {
		return nil, conflito(err, fmt.Sprintf("o sinônimo %s já está cadastrado (para este fornecedor ou para todos)", a.Alias))
	}
	return &out, nil
}

func (r *Repository) DeleteAlias(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM unit_of_measure_aliases WHERE id = $1 AND enterprise_id = $2`, id, empresa)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("sinônimo %d não encontrado", id))
	}
	return nil
}

func (r *Repository) SupplierExists(ctx context.Context, supplierCode int64) (bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM suppliers WHERE code = $1 AND enterprise_id = $2)`, supplierCode, empresa).Scan(&ok)
	return ok, err
}

// ─── Conversões padrão ───────────────────────────────────────────────────────

func (r *Repository) ListStandardConversions(ctx context.Context) ([]*entity.StandardConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT g.id, g.from_uom, g.to_uom, g.from_quantity, g.to_quantity, g.factor, g.is_system, g.is_active,
			g.created_at, g.updated_at, COALESCE(uf.dimension, 'OUTRA'), COALESCE(ut.dimension, 'OUTRA')
		  FROM global_unit_conversions g
		  LEFT JOIN units_of_measure uf ON uf.enterprise_id = g.enterprise_id AND uf.code = g.from_uom
		  LEFT JOIN units_of_measure ut ON ut.enterprise_id = g.enterprise_id AND ut.code = g.to_uom
		 WHERE g.enterprise_id = $1 AND g.is_active
		 ORDER BY uf.dimension, g.from_uom, g.to_uom`, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.StandardConversion{}
	for rows.Next() {
		var c entity.StandardConversion
		var df, dt string
		if err := rows.Scan(&c.ID, &c.FromUOM, &c.ToUOM, &c.FromQty, &c.ToQty, &c.Factor, &c.IsSystem, &c.IsActive,
			&c.CreatedAt, &c.UpdatedAt, &df, &dt); err != nil {
			return nil, err
		}
		c.FromDim, c.ToDim = entity.Dimensao(df), entity.Dimensao(dt)
		out = append(out, &c)
	}
	return out, rows.Err()
}

// UpsertStandardConversion grava pelo par (em qualquer sentido): cadastrar
// MM→M quando já existe M→MM substitui a existente em vez de criar a inversa.
func (r *Repository) UpsertStandardConversion(ctx context.Context, c *entity.StandardConversion) (*entity.StandardConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ator := tenant.Ator(ctx)
	out := *c
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, `DELETE FROM global_unit_conversions WHERE enterprise_id = $1
			AND ((from_uom = $2 AND to_uom = $3) OR (from_uom = $3 AND to_uom = $2))`, empresa, c.FromUOM, c.ToUOM); e != nil {
			return e
		}
		return tx.QueryRow(ctx, `INSERT INTO global_unit_conversions
			(enterprise_id, from_uom, to_uom, factor, from_quantity, to_quantity, created_by)
			VALUES ($1,$2,$3,$4,$5,$6, COALESCE(NULLIF($7,'')::uuid, '00000000-0000-0000-0000-000000000000'::uuid))
			RETURNING id, created_at, updated_at`,
			empresa, c.FromUOM, c.ToUOM, c.Factor, c.FromQty, c.ToQty, ator).Scan(&out.ID, &out.CreatedAt, &out.UpdatedAt)
	})
	if err != nil {
		return nil, conflito(err, "já existe conversão padrão para este par")
	}
	return &out, nil
}

func (r *Repository) DeleteStandardConversion(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE global_unit_conversions SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2 AND is_active`, id, empresa)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("conversão padrão %d não encontrada", id))
	}
	return nil
}

func (r *Repository) Resolver(ctx context.Context, supplierCode *int64, textos []string) (map[string]domainrepo.Resolucao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return uomgraph.Normalizar(ctx, r.pool, empresa, supplierCode, textos)
}

// UnidadesForaDoCatalogo devolve, dos códigos pedidos, os que não existem (ou
// estão inativos) no catálogo da empresa.
func (r *Repository) UnidadesForaDoCatalogo(ctx context.Context, codigos ...string) ([]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	pedidos := make([]string, 0, len(codigos))
	for _, c := range codigos {
		if c = entity.NormalizarCodigo(c); c != "" {
			pedidos = append(pedidos, c)
		}
	}
	if len(pedidos) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT x FROM unnest($1::text[]) x
		WHERE NOT EXISTS (SELECT 1 FROM units_of_measure u WHERE u.enterprise_id = $2 AND u.code = x AND u.is_active)`, pedidos, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// FatorDaConversaoDoItem: fator de → para da conversão do item (sem máscara)
// vigente hoje, cadastrada na VSUP0110 em qualquer sentido. Nil quando não há.
// Não olha a segunda unidade do próprio cadastro do item: é ela que está sendo
// conferida contra esta.
func (r *Repository) FatorDaConversaoDoItem(ctx context.Context, item int64, de, para string) (*decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	de, para = entity.NormalizarCodigo(de), entity.NormalizarCodigo(para)
	var qDe, qPara decimal.Decimal
	var direto bool
	err = r.pool.QueryRow(ctx, `SELECT c.from_quantity, c.to_quantity, c.from_uom = $3
		  FROM item_unit_conversions c
		 WHERE c.item_code = $1 AND c.mask = '' AND c.is_active
		   AND ((c.from_uom = $3 AND c.to_uom = $4) OR (c.from_uom = $4 AND c.to_uom = $3))
		   AND c.valid_from <= CURRENT_DATE AND (c.valid_to IS NULL OR c.valid_to >= CURRENT_DATE)
		   AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $2)
		 ORDER BY c.valid_from DESC LIMIT 1`, item, empresa, de, para).Scan(&qDe, &qPara, &direto)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if qDe.IsZero() || qPara.IsZero() {
		return nil, nil
	}
	f := qPara.DivRound(qDe, 10)
	if !direto {
		f = qDe.DivRound(qPara, 10)
	}
	return &f, nil
}

// CodigosFiscais: código do catálogo → código com que sai na NF-e/SPED
// (fiscal_code). Inclui unidades desativadas: documento antigo pode usá-las.
func (r *Repository) CodigosFiscais(ctx context.Context, codigos ...string) (map[string]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	pedidos := make([]string, 0, len(codigos))
	for _, c := range codigos {
		if c = entity.NormalizarCodigo(c); c != "" {
			pedidos = append(pedidos, c)
		}
	}
	out := map[string]string{}
	if len(pedidos) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT code, fiscal_code FROM units_of_measure WHERE enterprise_id = $1 AND code = ANY($2::text[])`, empresa, pedidos)
	if err != nil {
		return nil, fmt.Errorf("lendo o código fiscal das unidades: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var codigo, fiscal string
		if err := rows.Scan(&codigo, &fiscal); err != nil {
			return nil, err
		}
		out[codigo] = fiscal
	}
	return out, rows.Err()
}
