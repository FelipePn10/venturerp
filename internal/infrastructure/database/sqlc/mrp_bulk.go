package sqlc

import "context"

// listLatestStockSnapshots returns the most recent snapshot for every item
// that has any snapshot, using DISTINCT ON to get one row per item_code.
const listLatestStockSnapshots = `
SELECT DISTINCT ON (item_code)
    id, item_code, warehouse_code, quantity, reserved_qty, safety_stock, snapshot_date, created_at
FROM stock_snapshots
ORDER BY item_code, snapshot_date DESC
`

func (q *Queries) ListLatestStockSnapshots(ctx context.Context) ([]StockSnapshot, error) {
	rows, err := q.db.Query(ctx, listLatestStockSnapshots)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []StockSnapshot
	for rows.Next() {
		var s StockSnapshot
		if err := rows.Scan(
			&s.ID,
			&s.ItemCode,
			&s.WarehouseCode,
			&s.Quantity,
			&s.ReservedQty,
			&s.SafetyStock,
			&s.SnapshotDate,
			&s.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

const listAllActiveConfiguredRules = `
SELECT id, item_code, table_type, field_name, rule_type, rule_value, sequence, is_active,
       created_at, updated_at, created_by, code
FROM configured_item_rules
WHERE is_active = TRUE
ORDER BY item_code, sequence
`

func (q *Queries) ListAllActiveConfiguredRules(ctx context.Context) ([]ConfiguredItemRule, error) {
	rows, err := q.db.Query(ctx, listAllActiveConfiguredRules)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ConfiguredItemRule
	for rows.Next() {
		var r ConfiguredItemRule
		if err := rows.Scan(
			&r.ID,
			&r.ItemCode,
			&r.TableType,
			&r.FieldName,
			&r.RuleType,
			&r.RuleValue,
			&r.Sequence,
			&r.IsActive,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.CreatedBy,
			&r.Code,
		); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}
