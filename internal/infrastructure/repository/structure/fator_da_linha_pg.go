package structure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// FatoresDaEstruturaPG: a linha de estrutura é da empresa do item PAI (a
// tabela não tem coluna de empresa — AGENTS.md, "dado pendurado no item").
type FatoresDaEstruturaPG struct{ pool *pgxpool.Pool }

func NewFatoresDaEstrutura(pool *pgxpool.Pool) *FatoresDaEstruturaPG {
	return &FatoresDaEstruturaPG{pool: pool}
}

func (r *FatoresDaEstruturaPG) LinhasParaConferir(ctx context.Context, filtro entity.FiltroDeFatores) ([]entity.LinhaDeFator, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var filhos []int64
	if len(filtro.Filhos) > 0 {
		filhos = filtro.Filhos
	}
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.parent_code, COALESCE(NULLIF(p.business_code, ''), p.code::text), s.child_code, COALESCE(NULLIF(f.business_code, ''), f.code::text),
		       s.quantity, COALESCE(s.unit_of_measurement, ''), COALESCE(f.warehouse_unit_of_measurement, ''),
		       COALESCE(s.quantity_stock_uom, 0), COALESCE(s.conversion_factor, 0)
		  FROM item_structures s
		  JOIN items p ON p.code = s.parent_code AND p.enterprise_id = $1
		  JOIN items f ON f.code = s.child_code
		 WHERE s.is_active
		   AND ($2::bigint[] IS NULL OR s.child_code = ANY($2))
		   AND ($3::bigint IS NULL OR s.parent_code = $3)
		 ORDER BY s.parent_code, s.sequence, s.id`, enterpriseID, filhos, filtro.Pai)
	if err != nil {
		return nil, fmt.Errorf("lendo as linhas de estrutura para conferir o fator: %w", err)
	}
	defer rows.Close()
	var out []entity.LinhaDeFator
	for rows.Next() {
		var l entity.LinhaDeFator
		if err := rows.Scan(&l.ID, &l.ParentCode, &l.ParentLabel, &l.ChildCode, &l.ChildLabel, &l.Quantidade, &l.Unidade,
			&l.UnidadeEstoque, &l.QtdGravada, &l.FatorGravado); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *FatoresDaEstruturaPG) GravarFator(ctx context.Context, l entity.LinhaDeFator, qtd, fator decimal.Decimal, motivo string) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	// Trava otimista: a conta foi feita com a linha e a unidade de estoque
	// lidas antes. Se o usuário alterou a linha, ou a unidade do componente
	// mudou (e outro recálculo já gravou), gravar agora poria o número velho
	// por cima do novo. A comparação usa o mesmo COALESCE da leitura: linha
	// antiga com as colunas nulas também é corrigida.
	tag, err := tx.Exec(ctx, `
		UPDATE item_structures s SET quantity_stock_uom = $2, conversion_factor = $3, updated_at = NOW()
		 WHERE s.id = $1 AND s.is_active
		   AND s.quantity = $5
		   AND COALESCE(s.unit_of_measurement, '') = $6
		   AND COALESCE(s.quantity_stock_uom, 0) = $7
		   AND COALESCE(s.conversion_factor, 0) = $8
		   AND EXISTS (SELECT 1 FROM items p WHERE p.code = s.parent_code AND p.enterprise_id = $4)
		   AND EXISTS (SELECT 1 FROM items f WHERE f.code = s.child_code AND COALESCE(f.warehouse_unit_of_measurement, '') = $9)`,
		l.ID, qtd, fator, enterpriseID, l.Quantidade, l.Unidade, l.QtdGravada, l.FatorGravado, l.UnidadeEstoque)
	if err != nil {
		return false, fmt.Errorf("regravando o fator da linha de estrutura %d: %w", l.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil // outra empresa, ou a linha mudou depois da leitura
	}
	estado := func(q, f decimal.Decimal, u string) []byte {
		raw, _ := json.Marshal(map[string]any{
			"quantidade": l.Quantidade.String(), "unidade": l.Unidade, "unidade_estoque": u,
			"quantidade_estoque": q.String(), "fator": f.String(), "motivo": motivo,
		})
		return raw
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO item_structure_history (structure_id, parent_code, child_code, action, changed_by, before_state, after_state)
		VALUES ($1, $2, $3, 'ALTERACAO', $4, $5, $6)`,
		l.ID, l.ParentCode, l.ChildCode, atorDoContexto(ctx),
		estado(l.QtdGravada, l.FatorGravado, ""), estado(qtd, fator, l.UnidadeEstoque)); err != nil {
		return false, fmt.Errorf("registrando no histórico o novo fator da linha %d: %w", l.ID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
