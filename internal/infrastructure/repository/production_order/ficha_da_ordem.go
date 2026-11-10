package production_order

import (
	"context"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FichaDaOrdem lê a ordem como ela vai impressa para a fábrica. Tudo filtrado
// pela empresa da sessão: a ordem, o item, o centro, o fornecedor e as OS.
func (r *ProductionOrderRepositoryPGX) FichaDaOrdem(ctx context.Context, id int64) (*entity.FichaDaOrdem, error) {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	f := &entity.FichaDaOrdem{}
	var routeID *int64
	err = r.pool.QueryRow(ctx, `SELECT po.id, po.order_number, po.item_code, COALESCE(i.business_code,''), COALESCE(i.name,''), po.mask,
			COALESCE(i.warehouse_unit_of_measurement,'UN'), COALESCE(po.item_revision, i.revision, ''),
			po.status, po.planned_qty, po.produced_qty, po.scrapped_qty, po.start_date::timestamptz, po.end_date::timestamptz,
			COALESCE(po.priority,''), COALESCE(po.notes,''), COALESCE(po.origin_type,''), po.sales_order_code, po.route_id, po.warehouse_id
		FROM production_orders po
		LEFT JOIN items i ON i.code = po.item_code AND i.enterprise_id = po.enterprise_id
		WHERE po.id = $1 AND po.enterprise_id = $2`, id, eid).Scan(
		&f.ID, &f.Numero, &f.ItemCode, &f.ItemCodigo, &f.ItemNome, &f.Mascara, &f.Unidade, &f.Revisao,
		&f.Status, &f.QtdPlanejada, &f.QtdProduzida, &f.QtdRefugada, &f.Inicio, &f.Fim,
		&f.Prioridade, &f.Observacoes, &f.Origem, &f.PedidoVenda, &routeID, &f.Deposito)
	if err == pgx.ErrNoRows {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("ordem %d não encontrada", id))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a ordem: %w", err)
	}

	if routeID != nil {
		var rt entity.RoteiroDaFicha
		err := r.pool.QueryRow(ctx, `SELECT code, alternative, COALESCE(description,'')
			FROM manufacturing_routes WHERE id = $1 AND enterprise_id = $2`, *routeID, eid).Scan(&rt.Codigo, &rt.Alternativa, &rt.Descricao)
		if err != nil && err != pgx.ErrNoRows {
			return nil, fmt.Errorf("lendo o roteiro da ordem: %w", err)
		}
		if err == nil {
			f.Roteiro = &rt
		}
	}

	// Etapas: o snapshot da ordem (production_order_operations) manda; o roteiro
	// só completa o que o snapshot não guarda — origem, inspeção e fornecedor.
	rows, err := r.pool.Query(ctx, `SELECT op.sequence, op.operation_name,
			COALESCE(mt.code::text || ' — ' || mt.name, ''),
			COALESCE(o.origin::text IN ('EXTERNA','TERCEIROS'), FALSE),
			COALESCE((SELECT s.code::text || ' — ' || COALESCE(NULLIF(s.name,''), s.trade_name)
				FROM third_party_service_orders so
				JOIN suppliers s ON s.code = so.supplier_code AND s.enterprise_id = so.enterprise_id
				WHERE so.production_order_id = op.production_order_id AND so.route_operation_id = op.route_operation_id
				  AND so.enterprise_id = $2
				ORDER BY so.id DESC LIMIT 1),
			  (SELECT s.code::text || ' — ' || COALESCE(NULLIF(s.name,''), s.trade_name) FROM suppliers s
				WHERE s.enterprise_id = $2 AND (s.code = COALESCE(ro.supplier_id, o.supplier_id) OR s.id = COALESCE(ro.supplier_id, o.supplier_id))
				ORDER BY (s.code = COALESCE(ro.supplier_id, o.supplier_id)) DESC LIMIT 1), ''),
			op.setup_hours, op.planned_hours, op.actual_hours, op.planned_qty,
			COALESCE(ro.inspection_required, FALSE), op.status, COALESCE(op.notes,''), op.started_at, op.completed_at
		FROM production_order_operations op
		LEFT JOIN route_operations ro ON ro.id = op.route_operation_id
		LEFT JOIN operations o ON o.id = ro.operation_id AND o.enterprise_id = op.enterprise_id
		LEFT JOIN machine_types mt ON mt.id = op.work_center_id AND mt.enterprise_id = op.enterprise_id
		WHERE op.production_order_id = $1 AND op.enterprise_id = $2
		ORDER BY op.sequence, op.id`, id, eid)
	if err != nil {
		return nil, fmt.Errorf("lendo as etapas da ordem: %w", err)
	}
	for rows.Next() {
		var e entity.EtapaDaFicha
		if err := rows.Scan(&e.Sequencia, &e.Operacao, &e.Centro, &e.Externa, &e.Terceiro,
			&e.HorasPrep, &e.HorasPrevista, &e.HorasReais, &e.QtdEntra, &e.Inspecao, &e.Status, &e.Observacao, &e.Inicio, &e.Fim); err != nil {
			rows.Close()
			return nil, err
		}
		if !e.Externa {
			e.Terceiro = ""
		}
		f.Etapas = append(f.Etapas, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// O atendido da lista não é atualizado pelo consumo lançado na OP; o que
	// saiu de fato do estoque para a ordem está em production_consumptions.
	rows, err = r.pool.Query(ctx, `SELECT m.item_code, COALESCE(i.business_code,''), COALESCE(i.name,''), m.mask, m.quantity,
			GREATEST(m.attended_quantity, COALESCE((SELECT SUM(c.consumed_qty) FROM production_consumptions c
				WHERE c.production_order_id = m.production_order_id AND c.item_code = COALESCE(m.substituted_item_code, m.item_code)), 0)),
			m.uom, m.route_operation_sequence, m.warehouse_id, m.material_kind
		FROM production_order_materials m
		LEFT JOIN items i ON i.code = COALESCE(m.substituted_item_code, m.item_code) AND i.enterprise_id = m.enterprise_id
		WHERE m.production_order_id = $1 AND m.enterprise_id = $2
		ORDER BY m.route_operation_sequence NULLS FIRST, m.item_code, m.id`, id, eid)
	if err != nil {
		return nil, fmt.Errorf("lendo os materiais da ordem: %w", err)
	}
	for rows.Next() {
		var m entity.MaterialDaFicha
		if err := rows.Scan(&m.ItemCode, &m.ItemCodigo, &m.ItemNome, &m.Mascara, &m.Quantidade, &m.Atendida,
			&m.Unidade, &m.Etapa, &m.Deposito, &m.Tipo); err != nil {
			rows.Close()
			return nil, err
		}
		f.Materiais = append(f.Materiais, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = r.pool.Query(ctx, `SELECT op.sequence, t.code, t.name, rt.qty_required
		FROM production_order_operations op
		JOIN route_operation_tools rt ON rt.route_operation_id = op.route_operation_id
		JOIN tools t ON t.id = rt.tool_id AND t.enterprise_id = op.enterprise_id
		WHERE op.production_order_id = $1 AND op.enterprise_id = $2 AND op.status <> 'SKIPPED'
		ORDER BY op.sequence, t.code`, id, eid)
	if err != nil {
		return nil, fmt.Errorf("lendo as ferramentas da ordem: %w", err)
	}
	for rows.Next() {
		var t entity.FerramentaDaFicha
		if err := rows.Scan(&t.Etapa, &t.Codigo, &t.Nome, &t.Quantidade); err != nil {
			rows.Close()
			return nil, err
		}
		f.Ferramentas = append(f.Ferramentas, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = r.pool.Query(ctx, `SELECT so.code, ro.sequence,
			COALESCE(s.code::text || ' — ' || COALESCE(NULLIF(s.name,''), s.trade_name), so.supplier_code::text),
			so.quantity, so.fulfilled_quantity, so.due_date::timestamptz, so.status
		FROM third_party_service_orders so
		LEFT JOIN route_operations ro ON ro.id = so.route_operation_id
		LEFT JOIN suppliers s ON s.code = so.supplier_code AND s.enterprise_id = so.enterprise_id
		WHERE so.production_order_id = $1 AND so.enterprise_id = $2
		ORDER BY ro.sequence NULLS LAST, so.code`, id, eid)
	if err != nil {
		return nil, fmt.Errorf("lendo as ordens de serviço de terceiro: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var o entity.OrdemTerceiroDaFicha
		if err := rows.Scan(&o.Codigo, &o.Etapa, &o.Fornecedor, &o.Quantidade, &o.Recebida, &o.Prazo, &o.Status); err != nil {
			return nil, err
		}
		f.OrdensTerceiro = append(f.OrdensTerceiro, o)
	}
	return f, rows.Err()
}

// NomeDoUsuario dá o nome de quem emite a ordem impressa.
func (r *ProductionOrderRepositoryPGX) NomeDoUsuario(ctx context.Context, id uuid.UUID) (string, error) {
	var nome string
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(name, email, '') FROM users WHERE id = $1`, id).Scan(&nome)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return nome, err
}

// OrdensComCustoApurado: OFs da empresa da sessão que já têm apuração de custo.
func (r *ProductionOrderRepositoryPGX) OrdensComCustoApurado(ctx context.Context) ([]int64, error) {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT c.production_order_id FROM production_order_costs c
		JOIN production_orders o ON o.id = c.production_order_id AND o.enterprise_id = $1 ORDER BY 1`, eid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
