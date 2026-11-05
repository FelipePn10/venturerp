package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

const cartaoSQL = `SELECT k.id, k.item_code, COALESCE(i.name,''), k.warehouse_id, k.card_count, k.quantity_per_card, k.reorder_point,
	       k.container_type, k.status,
	       COALESCE((SELECT SUM(b.available_qty) FROM stock_balances b WHERE b.enterprise_id=k.enterprise_id AND b.item_code=k.item_code
	                  AND (k.warehouse_id IS NULL OR b.warehouse_id=k.warehouse_id)), 0),
	       COALESCE(i.engineering_type, 1) = 0
	  FROM kanban_cards k JOIN items i ON i.code=k.item_code AND i.enterprise_id=k.enterprise_id`

func scanCartao(row pgx.Row) (*entity.CartaoKanban, error) {
	var c entity.CartaoKanban
	if err := row.Scan(&c.ID, &c.ItemCode, &c.Nome, &c.WarehouseID, &c.CardCount, &c.QuantityPerCard, &c.ReorderPoint,
		&c.ContainerType, &c.Status, &c.Saldo, &c.Fabricado); err != nil {
		return nil, err
	}
	return &c, nil
}

// ListarCartoesKanban lista os cartões da empresa.
func (r *StockRepositorySQLC) ListarCartoesKanban(ctx context.Context) ([]entity.CartaoKanban, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, cartaoSQL+` WHERE k.enterprise_id=$1 ORDER BY i.name, k.id`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os cartões kanban: %w", err)
	}
	defer rows.Close()
	out := []entity.CartaoKanban{}
	for rows.Next() {
		c, err := scanCartao(rows)
		if err != nil {
			return nil, fmt.Errorf("lendo o cartão kanban: %w", err)
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ObterCartaoKanban lê um cartão da empresa.
func (r *StockRepositorySQLC) ObterCartaoKanban(ctx context.Context, id int64) (*entity.CartaoKanban, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := scanCartao(r.pool.QueryRow(ctx, cartaoSQL+` WHERE k.enterprise_id=$1 AND k.id=$2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("cartão kanban não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o cartão kanban: %w", err)
	}
	return c, nil
}

// SalvarCartaoKanban cria ou altera um cartão.
func (r *StockRepositorySQLC) SalvarCartaoKanban(ctx context.Context, c entity.CartaoKanban, ator uuid.UUID) (*entity.CartaoKanban, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var ok bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM items WHERE code=$2 AND enterprise_id=$1)
		    AND ($3::bigint IS NULL OR EXISTS (SELECT 1 FROM warehouse WHERE id=$3 AND enterprise_id=$1))`,
		empresa, c.ItemCode, c.WarehouseID).Scan(&ok); err != nil {
		return nil, fmt.Errorf("conferindo o cartão kanban: %w", err)
	}
	if !ok {
		return nil, errorsuc.NewValidationError("o item ou o almoxarifado do cartão não existe nesta empresa")
	}
	if c.ID == 0 {
		err = r.pool.QueryRow(ctx,
			`INSERT INTO kanban_cards(item_code,card_count,quantity_per_card,reorder_point,container_type,status,created_by,enterprise_id,warehouse_id)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
			c.ItemCode, c.CardCount, c.QuantityPerCard, c.ReorderPoint, c.ContainerType, c.Status, ator, empresa, c.WarehouseID).Scan(&c.ID)
		if err != nil {
			return nil, fmt.Errorf("gravando o cartão kanban: %w", err)
		}
		return r.ObterCartaoKanban(ctx, c.ID)
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE kanban_cards SET card_count=$3, quantity_per_card=$4, reorder_point=$5, container_type=$6, status=$7, warehouse_id=$8, updated_at=NOW()
		  WHERE id=$1 AND enterprise_id=$2`,
		c.ID, empresa, c.CardCount, c.QuantityPerCard, c.ReorderPoint, c.ContainerType, c.Status, c.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("alterando o cartão kanban: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errorsuc.NewNotFoundError("cartão kanban não encontrado")
	}
	return r.ObterCartaoKanban(ctx, c.ID)
}
