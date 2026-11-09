package item_conversion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/uomgraph"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ItemConversionRepositorySQLC mantém o nome por compatibilidade com o wiring;
// desde a migração 474 fala pgx direto (versões, transação e histórico não
// cabem bem no gerador).
type ItemConversionRepositorySQLC struct {
	pool *pgxpool.Pool
}

// New devolve o repositório completo. O parâmetro sqlc ficou por compatibilidade.
func New(_ *sqlc.Queries, pool *pgxpool.Pool) domainrepo.ManagedItemConversionRepository {
	return &ItemConversionRepositorySQLC{pool: pool}
}

const colunas = `c.id, c.item_code, c.mask, c.from_uom, c.to_uom, c.from_quantity, c.to_quantity, c.factor,
	c.rounding_percent, c.tolerance_value, c.tolerance_type, c.valid_from, c.valid_to, c.origin, c.notes,
	c.is_active, c.created_at, c.created_by, c.updated_at, c.updated_by`

func scan(row pgx.Row) (*entity.ItemUnitConversion, error) {
	var c entity.ItemUnitConversion
	err := row.Scan(&c.ID, &c.ItemCode, &c.Mask, &c.FromUOM, &c.ToUOM, &c.FromQty, &c.ToQty, &c.Factor,
		&c.RoundingPercent, &c.ToleranceValue, &c.ToleranceType, &c.ValidFrom, &c.ValidTo, &c.Origin, &c.Notes,
		&c.IsActive, &c.CreatedAt, &c.CreatedBy, &c.UpdatedAt, &c.UpdatedBy)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanAll(rows pgx.Rows) ([]*entity.ItemUnitConversion, error) {
	defer rows.Close()
	out := []*entity.ItemUnitConversion{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type execer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// traduzir transforma as violações do banco em mensagem para o usuário.
func traduzir(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23P01":
			return errorsuc.NewConflictError("já existe uma conversão deste par de unidades (ou a inversa dela) vigente no mesmo período: altere a existente ou encerre-a antes")
		case "23505":
			return errorsuc.NewConflictError("já existe uma conversão deste par de unidades começando nesta data")
		}
	}
	return err
}

func (r *ItemConversionRepositorySQLC) insert(ctx context.Context, db execer, empresa int64, c *entity.ItemUnitConversion) (*entity.ItemUnitConversion, error) {
	row := db.QueryRow(ctx, `
		INSERT INTO item_unit_conversions AS c (item_code, mask, from_uom, to_uom, factor, from_quantity, to_quantity,
		       rounding_percent, tolerance_value, tolerance_type, valid_from, valid_to, origin, notes, created_by, updated_by)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$15
		 WHERE EXISTS (SELECT 1 FROM items i WHERE i.code = $1 AND i.enterprise_id = $16)
		RETURNING `+colunas,
		c.ItemCode, c.Mask, c.FromUOM, c.ToUOM, c.Factor, c.FromQty, c.ToQty,
		c.RoundingPercent, c.ToleranceValue, c.ToleranceType, c.ValidFrom, c.ValidTo, c.Origin, c.Notes, c.CreatedBy, empresa)
	out, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado nesta empresa", c.ItemCode))
	}
	if err != nil {
		return nil, traduzir(err)
	}
	return out, nil
}

func gravarEvento(ctx context.Context, db execer, ev *entity.Event) error {
	if ev == nil {
		return nil
	}
	antes, _ := json.Marshal(ev.Before)
	depois, _ := json.Marshal(ev.After)
	if ev.Before == nil {
		antes = nil
	}
	if ev.After == nil {
		depois = nil
	}
	_, err := db.Exec(ctx, `INSERT INTO item_unit_conversion_events
		(item_code, conversion_id, lot_conversion_id, action, before_data, after_data, reason, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		ev.ItemCode, ev.ConversionID, ev.LotConversionID, ev.Action, antes, depois, ev.Reason, ev.CreatedBy)
	if err != nil {
		return fmt.Errorf("gravando o histórico da conversão: %w", err)
	}
	return nil
}

func (r *ItemConversionRepositorySQLC) Create(ctx context.Context, c *entity.ItemUnitConversion) (*entity.ItemUnitConversion, error) {
	return r.CreateWithEvent(ctx, c, nil)
}

func (r *ItemConversionRepositorySQLC) CreateWithEvent(ctx context.Context, c *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var out *entity.ItemUnitConversion
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var e error
		if out, e = r.insert(ctx, tx, empresa, c); e != nil {
			return e
		}
		if ev != nil {
			ev.ConversionID, ev.After = &out.ID, out.Snapshot()
		}
		return gravarEvento(ctx, tx, ev)
	})
	return out, err
}

func (r *ItemConversionRepositorySQLC) ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+` FROM item_unit_conversions c
		WHERE c.item_code = $1 AND c.is_active AND (c.valid_to IS NULL OR c.valid_to >= CURRENT_DATE)
		  AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $2)
		ORDER BY c.mask, c.from_uom, c.to_uom, c.valid_from`, itemCode, empresa)
	if err != nil {
		return nil, err
	}
	return scanAll(rows)
}

func (r *ItemConversionRepositorySQLC) ListVersions(ctx context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+` FROM item_unit_conversions c
		WHERE c.item_code = $1 AND c.is_active
		  AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $2)
		ORDER BY c.mask, c.from_uom, c.to_uom, c.valid_from`, itemCode, empresa)
	if err != nil {
		return nil, err
	}
	return scanAll(rows)
}

func (r *ItemConversionRepositorySQLC) Get(ctx context.Context, itemCode int64, fromUOM, toUOM string) (*entity.ItemUnitConversion, error) {
	return r.GetConfigured(ctx, itemCode, "", fromUOM, toUOM)
}

// GetConfigured: conversão direta vigente hoje para item/máscara/par.
func (r *ItemConversionRepositorySQLC) GetConfigured(ctx context.Context, itemCode int64, mask, fromUOM, toUOM string) (*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return scan(r.pool.QueryRow(ctx, `SELECT `+colunas+` FROM item_unit_conversions c
		WHERE c.item_code = $1 AND c.mask = $2 AND c.from_uom = $3 AND c.to_uom = $4 AND c.is_active
		  AND c.valid_from <= CURRENT_DATE AND (c.valid_to IS NULL OR c.valid_to >= CURRENT_DATE)
		  AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $5)
		ORDER BY c.valid_from DESC LIMIT 1`, itemCode, mask, strings.ToUpper(fromUOM), strings.ToUpper(toUOM), empresa))
}

func (r *ItemConversionRepositorySQLC) GetByID(ctx context.Context, id int64) (*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := scan(r.pool.QueryRow(ctx, `SELECT `+colunas+` FROM item_unit_conversions c
		WHERE c.id = $1 AND c.is_active
		  AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $2)`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("conversão %d não encontrada", id))
	}
	return c, err
}

func (r *ItemConversionRepositorySQLC) Update(ctx context.Context, c *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var out *entity.ItemUnitConversion
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `UPDATE item_unit_conversions c SET
			factor=$2, from_quantity=$3, to_quantity=$4, rounding_percent=$5, tolerance_value=$6, tolerance_type=$7,
			valid_from=$8, valid_to=$9, notes=$10, updated_at=NOW(), updated_by=$11
			WHERE c.id=$1 AND c.is_active
			  AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $12)
			RETURNING `+colunas,
			c.ID, c.Factor, c.FromQty, c.ToQty, c.RoundingPercent, c.ToleranceValue, c.ToleranceType,
			c.ValidFrom, c.ValidTo, c.Notes, c.UpdatedBy, empresa)
		var e error
		out, e = scan(row)
		if errors.Is(e, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("conversão %d não encontrada", c.ID))
		}
		if e != nil {
			return traduzir(e)
		}
		if ev != nil {
			ev.ConversionID, ev.After = &out.ID, out.Snapshot()
		}
		return gravarEvento(ctx, tx, ev)
	})
	return out, err
}

func (r *ItemConversionRepositorySQLC) NewVersion(ctx context.Context, atual *entity.ItemUnitConversion, closeAt time.Time, nova *entity.ItemUnitConversion, ev *entity.Event) (*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var out *entity.ItemUnitConversion
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		tag, e := tx.Exec(ctx, `UPDATE item_unit_conversions c SET valid_to=$2, updated_at=NOW(), updated_by=$3
			WHERE c.id=$1 AND c.is_active
			  AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $4)`,
			atual.ID, closeAt, nova.CreatedBy, empresa)
		if e != nil {
			return traduzir(e)
		}
		if tag.RowsAffected() == 0 {
			return errorsuc.NewNotFoundError(fmt.Sprintf("conversão %d não encontrada", atual.ID))
		}
		if out, e = r.insert(ctx, tx, empresa, nova); e != nil {
			return e
		}
		if ev != nil {
			ev.ConversionID, ev.After = &out.ID, out.Snapshot()
		}
		return gravarEvento(ctx, tx, ev)
	})
	return out, err
}

func (r *ItemConversionRepositorySQLC) Deactivate(ctx context.Context, id int64, ev *entity.Event) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var ator any
		if ev != nil && ev.CreatedBy != uuid.Nil {
			ator = ev.CreatedBy
		}
		tag, e := tx.Exec(ctx, `UPDATE item_unit_conversions c SET is_active = FALSE, updated_at = NOW(),
			       updated_by = COALESCE($3, c.updated_by)
			WHERE c.id = $1 AND c.is_active
			  AND EXISTS (SELECT 1 FROM items i WHERE i.code = c.item_code AND i.enterprise_id = $2)`, id, empresa, ator)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return errorsuc.NewNotFoundError(fmt.Sprintf("conversão %d não encontrada", id))
		}
		return gravarEvento(ctx, tx, ev)
	})
}

func (r *ItemConversionRepositorySQLC) Delete(ctx context.Context, id int64) error {
	return r.Deactivate(ctx, id, nil)
}

func (r *ItemConversionRepositorySQLC) AcceptsFractional(ctx context.Context, itemCode int64) (bool, error) {
	var value bool
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	err = r.pool.QueryRow(ctx, `SELECT accepts_fractional_quantity FROM items WHERE code=$1 AND enterprise_id=$2`, itemCode, empresa).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado", itemCode))
	}
	return value, err
}

func (r *ItemConversionRepositorySQLC) Graph(ctx context.Context, itemCode int64, mask, lot string, at time.Time) ([]service.Aresta, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return uomgraph.Carregar(ctx, r.pool, empresa, itemCode, strings.TrimSpace(mask), strings.TrimSpace(lot), at)
}

func (r *ItemConversionRepositorySQLC) MissingUnits(ctx context.Context, itemCode int64, units ...string) ([]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	pedidas := []string{}
	for _, u := range units {
		if u = strings.ToUpper(strings.TrimSpace(u)); u != "" {
			pedidas = append(pedidas, u)
		}
	}
	rows, err := r.pool.Query(ctx, `SELECT x FROM unnest($1::text[]) x
		WHERE NOT EXISTS (SELECT 1 FROM units_of_measure u WHERE u.enterprise_id = $2 AND u.code = x AND u.is_active)`, pedidas, empresa)
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

// ─── Lote ────────────────────────────────────────────────────────────────────

const colunasLote = `l.id, l.item_code, l.lot, l.from_uom, l.to_uom, l.from_quantity, l.to_quantity, l.factor,
	l.notes, l.is_active, l.created_by, l.created_at, l.updated_at`

func scanLote(row pgx.Row) (*entity.LotConversion, error) {
	var l entity.LotConversion
	if err := row.Scan(&l.ID, &l.ItemCode, &l.Lot, &l.FromUOM, &l.ToUOM, &l.FromQty, &l.ToQty, &l.Factor,
		&l.Notes, &l.IsActive, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *ItemConversionRepositorySQLC) ListLotConversions(ctx context.Context, itemCode int64) ([]*entity.LotConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasLote+` FROM item_lot_unit_conversions l
		WHERE l.item_code = $1 AND l.is_active
		  AND EXISTS (SELECT 1 FROM items i WHERE i.code = l.item_code AND i.enterprise_id = $2)
		ORDER BY l.lot, l.from_uom, l.to_uom`, itemCode, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.LotConversion{}
	for rows.Next() {
		l, err := scanLote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *ItemConversionRepositorySQLC) GetLotConversion(ctx context.Context, id int64) (*entity.LotConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	l, err := scanLote(r.pool.QueryRow(ctx, `SELECT `+colunasLote+` FROM item_lot_unit_conversions l
		WHERE l.id = $1 AND l.is_active
		  AND EXISTS (SELECT 1 FROM items i WHERE i.code = l.item_code AND i.enterprise_id = $2)`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("conversão do lote %d não encontrada", id))
	}
	return l, err
}

func (r *ItemConversionRepositorySQLC) UpsertLotConversion(ctx context.Context, l *entity.LotConversion, ev *entity.Event) (*entity.LotConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var out *entity.LotConversion
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO item_lot_unit_conversions AS l (item_code, lot, from_uom, to_uom, from_quantity, to_quantity, factor, notes, created_by)
			SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9
			 WHERE EXISTS (SELECT 1 FROM items i WHERE i.code = $1 AND i.enterprise_id = $10)
			ON CONFLICT (item_code, lot, from_uom, to_uom) WHERE is_active
			DO UPDATE SET from_quantity = EXCLUDED.from_quantity, to_quantity = EXCLUDED.to_quantity,
			              factor = EXCLUDED.factor, notes = EXCLUDED.notes, updated_at = NOW()
			RETURNING `+colunasLote,
			l.ItemCode, l.Lot, l.FromUOM, l.ToUOM, l.FromQty, l.ToQty, l.Factor, l.Notes, l.CreatedBy, empresa)
		var e error
		out, e = scanLote(row)
		if errors.Is(e, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado nesta empresa", l.ItemCode))
		}
		if e != nil {
			return e
		}
		if ev != nil {
			ev.LotConversionID, ev.After = &out.ID, out.Snapshot()
		}
		return gravarEvento(ctx, tx, ev)
	})
	return out, err
}

func (r *ItemConversionRepositorySQLC) DeactivateLotConversion(ctx context.Context, id int64, ev *entity.Event) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		tag, e := tx.Exec(ctx, `UPDATE item_lot_unit_conversions l SET is_active = FALSE, updated_at = NOW()
			WHERE l.id = $1 AND l.is_active
			  AND EXISTS (SELECT 1 FROM items i WHERE i.code = l.item_code AND i.enterprise_id = $2)`, id, empresa)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return errorsuc.NewNotFoundError(fmt.Sprintf("conversão do lote %d não encontrada", id))
		}
		return gravarEvento(ctx, tx, ev)
	})
}

func (r *ItemConversionRepositorySQLC) LotExists(ctx context.Context, itemCode int64, lot string) (bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM stock_lots s
		WHERE s.item_code = $1 AND s.lot = $2 AND s.enterprise_id = $3)`, itemCode, lot, empresa).Scan(&ok)
	return ok, err
}

// ─── Histórico ───────────────────────────────────────────────────────────────

func (r *ItemConversionRepositorySQLC) Events(ctx context.Context, itemCode int64, limit int) ([]*entity.Event, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `SELECT e.id, e.item_code, e.conversion_id, e.lot_conversion_id, e.action,
			e.before_data, e.after_data, e.reason, e.created_by, COALESCE(u.name, ''), e.created_at
		  FROM item_unit_conversion_events e
		  LEFT JOIN users u ON u.id = e.created_by
		 WHERE e.item_code = $1
		   AND EXISTS (SELECT 1 FROM items i WHERE i.code = e.item_code AND i.enterprise_id = $2)
		 ORDER BY e.created_at DESC, e.id DESC LIMIT $3`, itemCode, empresa, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.Event{}
	for rows.Next() {
		var ev entity.Event
		var antes, depois []byte
		if err := rows.Scan(&ev.ID, &ev.ItemCode, &ev.ConversionID, &ev.LotConversionID, &ev.Action,
			&antes, &depois, &ev.Reason, &ev.CreatedBy, &ev.CreatedByName, &ev.CreatedAt); err != nil {
			return nil, err
		}
		if len(antes) > 0 {
			_ = json.Unmarshal(antes, &ev.Before)
		}
		if len(depois) > 0 {
			_ = json.Unmarshal(depois, &ev.After)
		}
		out = append(out, &ev)
	}
	return out, rows.Err()
}

// ─── Cobertura e impacto ─────────────────────────────────────────────────────

func (r *ItemConversionRepositorySQLC) UnitsOfItem(ctx context.Context, itemCode int64) (*domainrepo.UnidadesDoItem, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	u := &domainrepo.UnidadesDoItem{ItemCode: itemCode}
	err = r.pool.QueryRow(ctx, `SELECT upper(i.warehouse_unit_of_measurement),
			COALESCE(upper(trim(i.supplies_purchase_uom)), ''),
			COALESCE(upper(trim(i.accounting_purchase_unit_of_measurement)), ''),
			COALESCE(upper(trim(i.accounting_sale_unit_of_measurement)), ''),
			COALESCE(upper(trim(i.warehouse_secondary_uom)), ''),
			i.accepts_fractional_quantity
		  FROM items i WHERE i.code = $1 AND i.enterprise_id = $2`, itemCode, empresa).
		Scan(&u.Estoque, &u.Compra, &u.CompraContabil, &u.VendaContabil, &u.Secundaria, &u.AceitaFracao)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado", itemCode))
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT upper(s.unit_of_measurement), 'ESTRUTURA', count(*) FROM item_structures s
		 WHERE s.child_code = $1 AND s.is_active GROUP BY 1
		UNION ALL
		SELECT upper(trim(p.xml_uom)), 'FORNECEDOR', count(*) FROM item_preferred_suppliers p
		 WHERE p.item_code = $1 AND p.enterprise_id = $2 AND p.is_active AND COALESCE(trim(p.xml_uom), '') <> '' GROUP BY 1
		UNION ALL
		SELECT upper(trim(g.uom)), 'EMBALAGEM', count(*) FROM item_gtins g
		 WHERE g.item_code = $1 AND g.enterprise_id = $2 AND COALESCE(trim(g.uom), '') <> '' GROUP BY 1
		UNION ALL
		SELECT upper(trim(poi.purchase_uom)), 'PEDIDO', count(*) FROM purchase_order_items poi
		  JOIN purchase_orders po ON po.code = poi.purchase_order_code
		 WHERE poi.item_code = $1 AND po.enterprise_code = $3 AND po.status <> 'CANCELLED' AND poi.is_active AND poi.status IN ('OPEN','PARTIAL')
		   AND COALESCE(trim(poi.purchase_uom), '') <> '' GROUP BY 1
		UNION ALL
		SELECT upper(COALESCE(NULLIF(trim(fi.unidade_normalizada), ''), trim(fi.uom))), 'NOTA', count(*) FROM fiscal_entry_items fi
		  JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id
		 WHERE fi.item_code = $1 AND fe.enterprise_id = $2 AND fe.data_entrada >= (CURRENT_DATE - 365)
		   AND COALESCE(trim(fi.uom), '') <> '' GROUP BY 1`, itemCode, empresa, empresaCode(ctx))
	if err != nil {
		return nil, fmt.Errorf("lendo onde o item aparece em outras unidades: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uso domainrepo.UsoDeUnidade
		if err := rows.Scan(&uso.Unidade, &uso.Onde, &uso.Quantidade); err != nil {
			return nil, err
		}
		u.Usos = append(u.Usos, uso)
	}
	return u, rows.Err()
}

func empresaCode(ctx context.Context) int64 {
	c, err := tenant.Code(ctx)
	if err != nil {
		return -1
	}
	return c
}

func (r *ItemConversionRepositorySQLC) Impact(ctx context.Context, c *entity.ItemUnitConversion) (*domainrepo.Impacto, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	par := []string{c.FromUOM, c.ToUOM}
	imp := &domainrepo.Impacto{}
	err = r.pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM item_structures s JOIN items i ON i.code = s.child_code AND i.enterprise_id = $2
		    WHERE s.child_code = $1 AND s.is_active AND upper(s.unit_of_measurement) = ANY($3)
		      AND upper(s.unit_of_measurement) <> upper(i.warehouse_unit_of_measurement)),
		  (SELECT count(*) FROM purchase_order_items poi JOIN purchase_orders po ON po.code = poi.purchase_order_code
		    WHERE poi.item_code = $1 AND po.enterprise_code = $4 AND po.status <> 'CANCELLED' AND poi.is_active AND poi.status IN ('OPEN','PARTIAL')
		      AND upper(trim(poi.purchase_uom)) = ANY($3)),
		  (SELECT count(*) FROM fiscal_entry_items fi JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id
		    WHERE fi.item_code = $1 AND fe.enterprise_id = $2 AND fe.status IN ('PENDING','CONFERRED')
		      AND upper(COALESCE(NULLIF(trim(fi.unidade_normalizada), ''), trim(fi.uom))) = ANY($3)),
		  (SELECT count(*) FROM item_preferred_suppliers p
		    WHERE p.item_code = $1 AND p.enterprise_id = $2 AND p.is_active AND upper(trim(p.xml_uom)) = ANY($3)),
		  (SELECT count(DISTINCT fe.id) FROM fiscal_entry_items fi JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id
		    WHERE fi.item_code = $1 AND fe.enterprise_id = $2 AND fe.status IN ('APPROVED','WRITTEN_OFF')
		      AND date_trunc('month', fe.data_entrada) = date_trunc('month', CURRENT_DATE))`,
		c.ItemCode, empresa, par, empresaCode(ctx)).
		Scan(&imp.EstruturasNaUnidade, &imp.PedidosAbertos, &imp.NotasPendentes, &imp.VinculosFornecedor, &imp.NotasAprovadasNoMes)
	if err != nil {
		return nil, fmt.Errorf("medindo o impacto da conversão: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT upper(s.unit_of_measurement), COALESCE(s.conversion_factor, 1), count(*)
		  FROM item_structures s JOIN items i ON i.code = s.child_code AND i.enterprise_id = $2
		 WHERE s.child_code = $1 AND s.is_active AND upper(s.unit_of_measurement) <> upper(i.warehouse_unit_of_measurement)
		 GROUP BY 1, 2`, c.ItemCode, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo os fatores congelados nas estruturas: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var f domainrepo.FatorCongelado
		if err := rows.Scan(&f.Unidade, &f.Fator, &f.Linhas); err != nil {
			return nil, err
		}
		imp.FatoresCongelados = append(imp.FatoresCongelados, f)
	}
	return imp, rows.Err()
}

// MaskExists: a máscara foi gerada para o item (configurador) nesta empresa.
func (r *ItemConversionRepositorySQLC) MaskExists(ctx context.Context, itemCode int64, mask string) (bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM item_masks m
		 WHERE m.item_code = $1 AND m.mask = $2
		   AND EXISTS (SELECT 1 FROM items i WHERE i.code = m.item_code AND i.enterprise_id = $3))`, itemCode, mask, empresa).Scan(&ok)
	return ok, err
}

// SincronizarSegundaUnidade grava no cadastro do item (segunda unidade) o
// fator da conversão do par estoque ↔ segunda unidade. Devolve o fator
// anterior e se mudou; não faz nada quando o par não é esse.
//
// O gatilho de stock_movements congela o peso nominal com este fator a cada
// movimento: sem a sincronização a VSUP0110 dizia 24 KG por chapa e o saldo em
// KG seguia somando 25.
func (r *ItemConversionRepositorySQLC) SincronizarSegundaUnidade(ctx context.Context, itemCode int64, de, para string, fator decimal.Decimal) (*decimal.Decimal, bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, false, err
	}
	if !fator.IsPositive() {
		return nil, false, nil
	}
	var antes *decimal.Decimal
	var novo decimal.Decimal
	err = r.pool.QueryRow(ctx, `WITH alvo AS (
			SELECT i.code, i.warehouse_secondary_factor AS antes,
			       CASE WHEN upper(i.warehouse_unit_of_measurement::text) = $3 THEN round($5::numeric, 6)
			            ELSE round(1 / $5::numeric, 6) END AS novo
			  FROM items i
			 WHERE i.code = $1 AND i.enterprise_id = $2 AND i.warehouse_secondary_factor IS NOT NULL
			   AND ((upper(i.warehouse_unit_of_measurement::text) = $3 AND upper(trim(i.warehouse_secondary_uom)) = $4)
			     OR (upper(i.warehouse_unit_of_measurement::text) = $4 AND upper(trim(i.warehouse_secondary_uom)) = $3))
			   FOR UPDATE)
		UPDATE items i SET warehouse_secondary_factor = alvo.novo
		  FROM alvo WHERE i.code = alvo.code AND alvo.novo > 0
		RETURNING alvo.antes, alvo.novo`, itemCode, empresa, strings.ToUpper(de), strings.ToUpper(para), fator).Scan(&antes, &novo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("atualizando a segunda unidade do cadastro do item: %w", err)
	}
	return antes, antes == nil || !antes.Equal(novo), nil
}
