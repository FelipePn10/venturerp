package item

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// BuscaPorDescricaoPG acha itens da empresa pela descrição (nome, descrição
// técnica ou comercial). O que bate exatamente vem primeiro; ativos antes.
type BuscaPorDescricaoPG struct {
	pool *pgxpool.Pool
}

func NewBuscaPorDescricaoPG(pool *pgxpool.Pool) *BuscaPorDescricaoPG {
	return &BuscaPorDescricaoPG{pool: pool}
}

func (b *BuscaPorDescricaoPG) ItensPorDescricao(ctx context.Context, texto string, limite int) ([]item_uc.ItemPorDescricao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	texto = strings.Join(strings.Fields(texto), " ")
	if texto == "" {
		return nil, nil
	}
	rows, err := b.pool.Query(ctx, `
		SELECT COALESCE(business_code, code::text), COALESCE(NULLIF(name, ''), pdm_description_technique)
		  FROM items
		 WHERE enterprise_id = $1
		   AND (name ILIKE '%' || $2 || '%' OR pdm_description_technique ILIKE '%' || $2 || '%'
		        OR COALESCE(commercial_description, '') ILIKE '%' || $2 || '%')
		 ORDER BY (upper(btrim(name)) = upper($2) OR upper(btrim(pdm_description_technique)) = upper($2)) DESC,
		          (health::text = 'ATIVO') DESC, code
		 LIMIT $3`, empresa, texto, limite)
	if err != nil {
		return nil, fmt.Errorf("buscando item pela descrição: %w", err)
	}
	defer rows.Close()
	var out []item_uc.ItemPorDescricao
	for rows.Next() {
		var it item_uc.ItemPorDescricao
		if err := rows.Scan(&it.Codigo, &it.Descricao); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
