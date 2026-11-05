package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ResolverLeitura descobre o que o coletor leu. A ordem importa: endereço do
// almoxarifado em uso, código do item, código de barras (do fornecedor ou o EAN
// que veio nas notas de entrada) e lote. O item não tem código de barras próprio:
// a etiqueta interna é o próprio código do item.
func (r *StockRepositorySQLC) ResolverLeitura(ctx context.Context, codigo string, almox *int64) (*entity.LeituraResolvida, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	texto := strings.TrimSpace(strings.Trim(codigo, "\r\n"))
	if texto == "" {
		return nil, errorsuc.NewValidationError("nada foi lido")
	}
	if almox != nil {
		if l, err := r.lerEndereco(ctx, empresa, texto, almox); l != nil || err != nil {
			return l, err
		}
	}
	if l, err := r.lerItem(ctx, empresa, `i.business_code=$2`, texto); l != nil || err != nil {
		return l, err
	}
	barras := []string{texto}
	gs1, ehGS1 := entity.LerGS1(texto)
	if ehGS1 {
		// GTIN-14 de EAN-13 é o mesmo número com zero à esquerda.
		barras = []string{gs1.GTIN, strings.TrimLeft(gs1.GTIN, "0")}
	}
	l, err := r.lerItem(ctx, empresa,
		`i.code IN (SELECT p.item_code FROM item_preferred_suppliers p
		             WHERE p.enterprise_id=$1 AND p.is_active AND p.barcode = ANY($2)
		            UNION
		            SELECT x.item_code FROM fiscal_entry_items x JOIN fiscal_entries f ON f.id=x.fiscal_entry_id
		             WHERE f.enterprise_id=$1 AND x.ean = ANY($2))`, barras)
	if err != nil {
		return nil, err
	}
	if l != nil {
		if ehGS1 {
			l.Lote, l.Validade, l.Quantidade = gs1.Lote, gs1.Validade, gs1.Quantidade
		}
		return l, nil
	}
	if l, err := r.lerItem(ctx, empresa, `i.code IN (SELECT s.item_code FROM stock_lots s WHERE s.enterprise_id=$1 AND s.lot=$2)`, texto); l != nil || err != nil {
		if l != nil {
			l.Lote = texto
		}
		return l, err
	}
	if almox == nil {
		if l, err := r.lerEndereco(ctx, empresa, texto, nil); l != nil || err != nil {
			return l, err
		}
	}
	return nil, errorsuc.NewNotFoundError(fmt.Sprintf("o código %s não é de item, lote nem endereço", texto))
}

func (r *StockRepositorySQLC) lerItem(ctx context.Context, empresa int64, filtro string, valor any) (*entity.LeituraResolvida, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT i.code, COALESCE(i.name,''), COALESCE(i.warehouse_unit_of_measurement::text,'')
		   FROM items i WHERE i.enterprise_id=$1 AND `+filtro+` LIMIT 2`, empresa, valor)
	if err != nil {
		return nil, fmt.Errorf("reading scanned item: %w", err)
	}
	defer rows.Close()
	var achados []*entity.LeituraResolvida
	for rows.Next() {
		l := &entity.LeituraResolvida{Tipo: entity.LeituraItem}
		var item int64
		if err := rows.Scan(&item, &l.Nome, &l.Unidade); err != nil {
			return nil, fmt.Errorf("reading scanned item: %w", err)
		}
		l.ItemCode = &item
		achados = append(achados, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading scanned item: %w", err)
	}
	switch len(achados) {
	case 0:
		return nil, nil
	case 1:
		return achados[0], nil
	}
	return nil, errorsuc.NewValidationError("este código está ligado a mais de um item: escolha o item pela lupa")
}

func (r *StockRepositorySQLC) lerEndereco(ctx context.Context, empresa int64, texto string, almox *int64) (*entity.LeituraResolvida, error) {
	l := &entity.LeituraResolvida{Tipo: entity.LeituraEndereco}
	var id int64
	err := r.pool.QueryRow(ctx,
		`SELECT a.address, a.warehouse_id, a.is_blocked FROM manufacturing_warehouse_addresses a
		  WHERE a.enterprise_id=$1 AND a.address=$2 AND a.is_active AND ($3::bigint IS NULL OR a.warehouse_id=$3)
		  ORDER BY a.warehouse_id LIMIT 1`, empresa, texto, almox).Scan(&l.Endereco, &id, &l.Bloqueado)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading scanned address: %w", err)
	}
	l.WarehouseID = &id
	return l, nil
}
