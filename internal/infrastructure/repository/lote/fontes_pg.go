package lote

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// Fontes lê roteiros de inspeção de recebimento e planos de inspeção da
// produção. O roteiro de recebimento usa enterprise_code (é do comercial/
// compras); o plano é da empresa pelo item.
type Fontes struct{ pool *pgxpool.Pool }

func NewFontes(pool *pgxpool.Pool) *Fontes { return &Fontes{pool: pool} }

var _ dom.Fontes = (*Fontes)(nil)

func (f *Fontes) RoteiroDeRecebimento(ctx context.Context, item int64, mask string) (*dom.Fonte, error) {
	codigo, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	var fonte dom.Fonte
	var rota, almox int64
	err = f.pool.QueryRow(ctx, `SELECT id, inspection_warehouse_id, sampling_procedure_id FROM receiving_inspection_routes
		 WHERE enterprise_code = $1 AND is_active AND basis = 'ITEM' AND item_code = $2 AND (mask = $3 OR mask = '')
		   AND valid_from <= CURRENT_DATE AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)
		 ORDER BY CASE WHEN mask = $3 THEN 0 ELSE 1 END, valid_from DESC, id DESC LIMIT 1`, codigo, item, mask).
		Scan(&rota, &almox, &fonte.ProcedimentoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("procurando o roteiro de inspeção do item: %w", err)
	}
	fonte.RouteID, fonte.AlmoxInspecao = &rota, &almox
	rows, err := f.pool.Query(ctx, `SELECT id, sequence, inspection_name, kind::text, is_required, nominal_value, min_value, max_value,
			sample_unit, instrument_group, norm, reference
		  FROM receiving_inspection_route_steps WHERE route_id = $1 AND (valid_to IS NULL OR valid_to >= CURRENT_DATE)
		 ORDER BY sequence, id`, rota)
	if err != nil {
		return nil, fmt.Errorf("lendo as etapas do roteiro: %w", err)
	}
	defer rows.Close()
	visto := map[int]bool{}
	for rows.Next() {
		var c dom.Caracteristica
		var step int64
		var tipo string
		if err := rows.Scan(&step, &c.Sequencia, &c.Nome, &tipo, &c.Obrigatoria, &c.Nominal, &c.Minimo, &c.Maximo,
			&c.Unidade, &c.Instrumento, &c.Norma, &c.Referencia); err != nil {
			return nil, err
		}
		if visto[c.Sequencia] {
			continue
		}
		visto[c.Sequencia] = true
		c.OrigemStep = &step
		switch tipo {
		case "VALUE":
			c.Tipo = dom.Variavel
		case "ATTRIBUTE":
			c.Tipo = dom.Atributo
		default:
			c.Tipo = dom.Estrutura
		}
		if err := limites(c); err != nil {
			return nil, err
		}
		fonte.Caracteristicas = append(fonte.Caracteristicas, c)
	}
	return &fonte, rows.Err()
}

func limites(c dom.Caracteristica) error {
	if c.Minimo != nil && c.Maximo != nil && c.Minimo.GreaterThan(*c.Maximo) {
		return errorsuc.NewValidationError(fmt.Sprintf("a característica %s tem mínimo maior que o máximo no plano: corrija o plano antes de abrir o lote", c.Nome))
	}
	return nil
}

func (f *Fontes) PlanoDeInspecao(ctx context.Context, planoID int64) (*dom.Fonte, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var fonte dom.Fonte
	var plano int64
	err = f.pool.QueryRow(ctx, `SELECT p.id, p.route_operation_id, p.sampling_procedure_id FROM inspection_plans p
		 JOIN items it ON it.code = p.item_code AND it.enterprise_id = $2
		 WHERE p.id = $1 AND p.is_active`, planoID, empresa).Scan(&plano, &fonte.RouteOperation, &fonte.ProcedimentoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("plano de inspeção %d", planoID))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o plano de inspeção: %w", err)
	}
	fonte.PlanID = &plano
	rows, err := f.pool.Query(ctx, `SELECT id, name, nominal, tolerance_lower, tolerance_upper, unit, is_critical
		  FROM inspection_plan_characteristics WHERE plan_id = $1 ORDER BY id`, plano)
	if err != nil {
		return nil, fmt.Errorf("lendo as características do plano: %w", err)
	}
	defer rows.Close()
	seq := 0
	for rows.Next() {
		var c dom.Caracteristica
		var id int64
		var nominal, minimo, maximo *decimal.Decimal
		if err := rows.Scan(&id, &c.Nome, &nominal, &minimo, &maximo, &c.Unidade, &c.Critica); err != nil {
			return nil, err
		}
		seq++
		c.Sequencia, c.OrigemCarac, c.Obrigatoria = seq, &id, true
		c.Nominal, c.Minimo, c.Maximo = nominal, minimo, maximo
		c.Tipo = dom.Atributo
		if nominal != nil || minimo != nil || maximo != nil {
			c.Tipo = dom.Variavel
		}
		if err := limites(c); err != nil {
			return nil, err
		}
		fonte.Caracteristicas = append(fonte.Caracteristicas, c)
	}
	return &fonte, rows.Err()
}
