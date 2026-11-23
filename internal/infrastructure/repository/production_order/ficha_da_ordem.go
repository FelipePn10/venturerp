package production_order

import (
	"context"
	"fmt"
	"strings"
	"time"

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
			po.status, po.planned_qty, po.produced_qty, po.scrapped_qty, po.start_date, po.end_date, -- DATE puro: ::timestamptz no fuso do banco (UTC) virava o dia anterior no processo (São Paulo)
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
			COALESCE(ro.inspection_required, FALSE), op.status, COALESCE(op.notes,''), op.started_at, op.completed_at,
			op.id, COALESCE(ro.notes,'')
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
			&e.HorasPrep, &e.HorasPrevista, &e.HorasReais, &e.QtdEntra, &e.Inspecao, &e.Status, &e.Observacao, &e.Inicio, &e.Fim,
			&e.ID, &e.Instrucao); err != nil {
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

	if err := r.eventosDasEtapas(ctx, eid, f); err != nil {
		return nil, err
	}
	if err := r.assinaturasDaFicha(ctx, eid, f); err != nil {
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
			so.quantity, so.fulfilled_quantity, so.due_date, so.status
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

// FusoDaEmpresa: fuso da empresa da sessão (padrão America/Sao_Paulo).
func (r *ProductionOrderRepositoryPGX) FusoDaEmpresa(ctx context.Context) (*time.Location, error) {
	eid, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var tz string
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT NULLIF(timezone,'') FROM enterprise_notification_settings WHERE enterprise_id=$1),'America/Sao_Paulo')`, eid).Scan(&tz); err != nil {
		return nil, fmt.Errorf("lendo o fuso da empresa: %w", err)
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("fuso da empresa %q inválido: %w", tz, err)
	}
	return loc, nil
}

// eventosDasEtapas: as mudanças de situação de cada etapa (trilha imutável), com
// quem fez e o motivo — é o que mostra início, pausas, retomadas e conclusão.
func (r *ProductionOrderRepositoryPGX) eventosDasEtapas(ctx context.Context, eid int64, f *entity.FichaDaOrdem) error {
	rows, err := r.pool.Query(ctx, `SELECT e.operation_id, e.new_status, e.occurred_at, COALESCE(u.name, ''), COALESCE(e.notes, '')
		FROM production_operation_execution_events e
		LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.production_order_id = $1 AND e.enterprise_id = $2 AND e.new_status <> e.old_status
		ORDER BY e.occurred_at, e.id`, f.ID, eid)
	if err != nil {
		return fmt.Errorf("lendo o histórico das etapas: %w", err)
	}
	defer rows.Close()
	porEtapa := map[int64]int{}
	for i, e := range f.Etapas {
		porEtapa[e.ID] = i
	}
	for rows.Next() {
		var op int64
		var ev entity.EventoDaEtapa
		var notas string
		if err := rows.Scan(&op, &ev.Status, &ev.Quando, &ev.Quem, &notas); err != nil {
			return err
		}
		ev.Motivo = motivoDoEvento(notas)
		if i, ok := porEtapa[op]; ok {
			f.Etapas[i].Eventos = append(f.Etapas[i].Eventos, ev)
		}
	}
	return rows.Err()
}

// motivoDoEvento: a última linha da observação da etapa é a deste evento
// ("2026-11-23 10:51:58 -03 · Pausada: almoço"); o motivo vem depois do ": ".
func motivoDoEvento(notas string) string {
	linhas := strings.Split(strings.TrimSpace(notas), "\n")
	ultima := linhas[len(linhas)-1]
	_, depois, ok := strings.Cut(ultima, " · ")
	if !ok {
		return ""
	}
	_, motivo, _ := strings.Cut(depois, ": ")
	return strings.TrimSpace(motivo)
}

// assinaturasDaFicha: quem emitiu, separou, executou e liberou a ordem, pelo
// que o sistema registrou. O que não aconteceu fica vazio (assina-se à mão).
func (r *ProductionOrderRepositoryPGX) assinaturasDaFicha(ctx context.Context, eid int64, f *entity.FichaDaOrdem) error {
	ler := func(dst *entity.AssinaturaDaFicha, sql string) error {
		var quando time.Time
		var nome string
		err := r.pool.QueryRow(ctx, sql, f.ID, eid).Scan(&quando, &nome)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		dst.Nome, dst.Quando = nome, &quando
		return nil
	}
	if err := ler(&f.Emissao, `SELECT po.created_at, COALESCE(u.name, '') FROM production_orders po
		LEFT JOIN users u ON u.id = po.created_by WHERE po.id = $1 AND po.enterprise_id = $2`); err != nil {
		return fmt.Errorf("lendo a emissão da ordem: %w", err)
	}
	// Separação: a primeira baixa de material para a ordem.
	if err := ler(&f.Separacao, `SELECT c.created_at, COALESCE(u.name, '') FROM production_consumptions c
		JOIN production_orders po ON po.id = c.production_order_id AND po.enterprise_id = $2
		LEFT JOIN users u ON u.id = c.created_by
		WHERE c.production_order_id = $1 ORDER BY c.created_at, c.id LIMIT 1`); err != nil {
		return fmt.Errorf("lendo a separação da ordem: %w", err)
	}
	// Execução: só com a ordem concluída — a última etapa concluída ou, sem
	// etapas, o último apontamento.
	if f.Status == "COMPLETED" || f.Status == "CLOSED" {
		if err := ler(&f.Execucao, `SELECT quando, nome FROM (
				SELECT e.occurred_at AS quando, COALESCE(u.name, '') AS nome, 1 AS fonte FROM production_operation_execution_events e
				LEFT JOIN users u ON u.id = e.actor_id
				WHERE e.production_order_id = $1 AND e.enterprise_id = $2 AND e.new_status = 'DONE' AND e.old_status <> 'DONE'
				UNION ALL
				SELECT a.created_at, COALESCE(u.name, ''), 2 FROM production_appointments a
				JOIN production_orders po ON po.id = a.production_order_id AND po.enterprise_id = $2
				LEFT JOIN users u ON u.id = a.created_by
				WHERE a.production_order_id = $1) x
			ORDER BY fonte, quando DESC LIMIT 1`); err != nil {
			return fmt.Errorf("lendo a execução da ordem: %w", err)
		}
	}
	// Qualidade: a última decisão do lote de inspeção FINAL da ordem — a de
	// processo não é liberação e não pode aparecer como inspeção final.
	var insp entity.InspecaoFinalDaFicha
	err := r.pool.QueryRow(ctx, `SELECT d.decisao, COALESCE(d.qtd_aceita, 0), COALESCE(u.name, ''), d.decided_at
		FROM quality_inspection_lots l
		JOIN quality_lot_decisions d ON d.lot_id = l.id AND d.enterprise_id = l.enterprise_id
		LEFT JOIN users u ON u.id = d.decided_by
		WHERE l.production_order_id = $1 AND l.enterprise_id = $2 AND l.status = 'DECIDIDO' AND l.origem = 'FINAL'
		ORDER BY d.decided_at DESC LIMIT 1`, f.ID, eid).Scan(&insp.Decisao, &insp.QtdAceita, &insp.Inspetor, &insp.Quando)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("lendo a liberação da qualidade: %w", err)
	}
	if err == nil {
		f.InspecaoFinal = &insp
		quando := insp.Quando
		f.Qualidade = entity.AssinaturaDaFicha{Nome: insp.Inspetor, Quando: &quando}
	}
	return nil
}
