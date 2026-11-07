package item

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

// executor é o que as gravações do complemento precisam: a transação do item.
type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// gravarComplementos grava o que mora fora de items — estoque máximo, cobertura,
// agrupamento (item_planning_extras) e tempo de segurança e lote máximo
// (politicas_planejamento, que é onde o MRP os lê) — na transação do item.
func gravarComplementos(ctx context.Context, tx executor, empresa, item int64, it *entity.Item) error {
	if it == nil {
		return nil
	}
	e := it.Planning.Extras
	if _, err := tx.Exec(ctx, `
		INSERT INTO item_planning_extras (item_code, enterprise_id, maximum_stock, coverage, grouping_key)
		SELECT $1::bigint, $2::bigint, $3::numeric, $4::integer, $5::varchar
		 WHERE $3::numeric <> 0 OR $4::integer <> 0 OR $5::varchar IS NOT NULL
		    OR EXISTS (SELECT 1 FROM item_planning_extras x WHERE x.item_code = $1::bigint)
		ON CONFLICT (item_code) DO UPDATE SET maximum_stock = EXCLUDED.maximum_stock, coverage = EXCLUDED.coverage,
		    grouping_key = EXCLUDED.grouping_key, updated_at = NOW()`,
		item, empresa, e.MaximumStock, e.Coverage, e.GroupingKey); err != nil {
		return fmt.Errorf("gravando os parâmetros de planejamento do item: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO politicas_planejamento (enterprise_id, item_code, tempo_seguranca_dias, lote_maximo, created_by)
		SELECT $1::bigint, $2::bigint, $3::smallint, $4::numeric, $5::uuid
		 WHERE $3::smallint IS NOT NULL OR $4::numeric IS NOT NULL
		ON CONFLICT (enterprise_id, item_code) WHERE item_code IS NOT NULL DO UPDATE
		   SET tempo_seguranca_dias = EXCLUDED.tempo_seguranca_dias, lote_maximo = EXCLUDED.lote_maximo, updated_at = NOW()`,
		empresa, item, e.SafetyTimeDays, e.MaximumLot, it.CreatedBy); err != nil {
		return fmt.Errorf("gravando a política de planejamento do item: %w", err)
	}
	if e.SafetyTimeDays == nil && e.MaximumLot == nil {
		// Sem os dois: a linha que só tinha isso sai, porque a tabela recusa regra
		// vazia; a que tem outra regra do item fica, sem os dois valores.
		if _, err := tx.Exec(ctx, `DELETE FROM politicas_planejamento WHERE enterprise_id = $1 AND item_code = $2
			   AND estrategia IS NULL AND consumo_dias_antes IS NULL AND consumo_dias_depois IS NULL
			   AND horizonte_firme_dias IS NULL AND lote_periodo IS NULL`, empresa, item); err != nil {
			return fmt.Errorf("removendo a política vazia do item: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE politicas_planejamento SET tempo_seguranca_dias = NULL, lote_maximo = NULL, updated_at = NOW()
			 WHERE enterprise_id = $1 AND item_code = $2`, empresa, item); err != nil {
			return fmt.Errorf("limpando a política de planejamento do item: %w", err)
		}
	}
	return nil
}

// carregarComplementos lê de volta o que gravarComplementos escreve. Sem isso a
// alteração partiria de zero e apagaria o que a requisição não trouxe.
func (r *RepositoryItemSQLC) carregarComplementos(ctx context.Context, it *entity.Item) error {
	if r.pool == nil {
		return nil
	}
	var extras entity.PlanningExtras
	err := r.pool.QueryRow(ctx, `SELECT maximum_stock, coverage, grouping_key FROM item_planning_extras
		 WHERE enterprise_id = $1 AND item_code = $2`, it.EnterpriseID, int64(it.Code)).Scan(&extras.MaximumStock, &extras.Coverage, &extras.GroupingKey)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lendo os parâmetros de planejamento do item: %w", err)
	}
	var seguranca *int16
	var maximo *decimal.Decimal
	err = r.pool.QueryRow(ctx, `SELECT tempo_seguranca_dias, lote_maximo FROM politicas_planejamento
		 WHERE enterprise_id = $1 AND item_code = $2`, it.EnterpriseID, int64(it.Code)).Scan(&seguranca, &maximo)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lendo a política de planejamento do item: %w", err)
	}
	if seguranca != nil {
		dias := int(*seguranca)
		extras.SafetyTimeDays = &dias
	}
	extras.MaximumLot = maximo
	it.Planning.Extras = extras
	return nil
}

// criarComComplementos grava o item e os complementos numa transação só.
func (r *RepositoryItemSQLC) criarComComplementos(ctx context.Context, p sqlc.CreateItemParams, it *entity.Item) (sqlc.Item, error) {
	if err := r.conferirGTINeResponsaveis(ctx, it); err != nil {
		return sqlc.Item{}, err
	}
	if err := r.conferirConflitoDeCodigos(ctx, it, true); err != nil {
		return sqlc.Item{}, err
	}
	if r.pool == nil {
		return r.q.CreateItem(ctx, p)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.Item{}, err
	}
	defer tx.Rollback(ctx)
	criado, err := r.q.WithTx(tx).CreateItem(ctx, p)
	if err != nil {
		return sqlc.Item{}, err
	}
	if err := gravarComplementos(ctx, tx, criado.EnterpriseID, criado.Code, it); err != nil {
		return sqlc.Item{}, err
	}
	// Validade é do lote: o item que nasce com validade nasce controlando lote.
	// (Na alteração o cadastro recusa validade sem lote; na criação o controle
	// ainda não existe, e quem importa itens não teria como ligá-lo antes.)
	if it.Warehouse.ShelfLifeDays != nil || it.Warehouse.MinRemainingLifeDays != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO manufacturing_stock_item_controls (enterprise_id, item_code, stock_uom, controls_lot, controls_address, lot_issue_rule)
			VALUES ($1, $2, COALESCE(NULLIF($3, ''), 'UN'), TRUE, FALSE, 'FEFO')
			ON CONFLICT (enterprise_id, item_code) DO UPDATE SET controls_lot = TRUE, updated_at = NOW()`,
			criado.EnterpriseID, criado.Code, string(it.Warehouse.UnitOfMeasurement)); err != nil {
			return sqlc.Item{}, fmt.Errorf("ligando o controle de lote do item com validade: %w", err)
		}
	}
	return criado, tx.Commit(ctx)
}

// conferirGTINeResponsaveis devolve a regra em português antes do banco: o índice
// único e a falta de chave do funcionário dariam erro genérico ou passariam calado.
func (r *RepositoryItemSQLC) conferirGTINeResponsaveis(ctx context.Context, it *entity.Item) error {
	if it.Accounting.GTIN != nil {
		outro, err := r.q.ItemGTINEmUso(ctx, sqlc.ItemGTINEmUsoParams{EnterpriseID: it.EnterpriseID, Gtin: pgutil.ToPgTextFromPtr(it.Accounting.GTIN), BusinessCode: string(it.BusinessCode)})
		if err == nil {
			return errorsuc.NewValidationError(fmt.Sprintf("o GTIN %s já está no item %s", *it.Accounting.GTIN, outro))
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("conferindo o GTIN do item: %w", err)
		}
	}
	if r.pool == nil {
		return nil
	}
	if it.Accounting.GTIN != nil {
		// O GTIN de embalagem de outro item também conta: um código, um item.
		var dono string
		err := r.pool.QueryRow(ctx, `SELECT i.business_code FROM item_gtins g JOIN items i ON i.code = g.item_code
			 WHERE g.enterprise_id = $1 AND g.gtin = $2 AND i.business_code <> $3 LIMIT 1`, it.EnterpriseID, *it.Accounting.GTIN, string(it.BusinessCode)).Scan(&dono)
		if err == nil {
			return errorsuc.NewValidationError(fmt.Sprintf("o GTIN %s já está em uma embalagem do item %s", *it.Accounting.GTIN, dono))
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("conferindo o GTIN das embalagens: %w", err)
		}
	}
	if c := it.Commercial.ExclusiveCustomerCode; c != nil {
		var existe bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM customers WHERE code = $1 AND enterprise_id = $2)`, *c, it.EnterpriseID).Scan(&existe); err != nil {
			return fmt.Errorf("conferindo o cliente exclusivo do item: %w", err)
		}
		if !existe {
			return errorsuc.NewValidationError(fmt.Sprintf("o cliente %d não é desta empresa", *c))
		}
	}
	for rotulo, codigo := range map[string]*int64{"planejador": it.Planning.PlannerEmployeeCode, "comprador": it.Planning.BuyerEmployeeCode} {
		if codigo == nil {
			continue
		}
		var existe bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM employees WHERE code = $1 AND enterprise_id = $2)`, *codigo, it.EnterpriseID).Scan(&existe); err != nil {
			return fmt.Errorf("conferindo o %s do item: %w", rotulo, err)
		}
		if !existe {
			return errorsuc.NewValidationError(fmt.Sprintf("o %s %d não é funcionário desta empresa", rotulo, *codigo))
		}
	}
	return nil
}
