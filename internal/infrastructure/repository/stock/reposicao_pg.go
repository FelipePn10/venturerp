package stock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// itemLiberadoSQL: só item liberado (ou sem governança) é reposto.
const itemLiberadoSQL = `COALESCE((SELECT g.status FROM item_governanca g
	  WHERE g.enterprise_id = i.enterprise_id AND g.item_code = i.code), 'LIBERADO') = 'LIBERADO'`

// itensPorMascara é o conjunto do cálculo: o item e, ao lado, cada máscara com
// política própria. Saldo e chegada de máscara com política vão só para ela; os
// demais ficam no item, então nada conta duas vezes.
type itensPorMascara struct {
	itens    map[int64]*entity.ItemReposicao
	mascaras map[int64]map[string]*entity.ItemReposicao
}

func (m itensPorMascara) de(item int64, mask string) *entity.ItemReposicao {
	if it := m.mascaras[item][mask]; it != nil {
		return it
	}
	return m.itens[item]
}

// SugestoesDeReposicao lê a situação de cada item e deixa o cálculo para o
// domínio. Item bloqueado, descontinuado, em cadastro, fantasma ou fora do
// planejamento não entra.
func (r *StockRepositorySQLC) SugestoesDeReposicao(ctx context.Context) ([]entity.SugestaoReposicao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	itens, ordem, err := r.itensDeReposicao(ctx, empresa)
	if err != nil {
		return nil, err
	}
	out := []entity.SugestaoReposicao{}
	if len(itens) == 0 {
		return out, nil
	}
	conjunto := itensPorMascara{itens: itens, mascaras: map[int64]map[string]*entity.ItemReposicao{}}
	if err := r.politicasDosItens(ctx, empresa, conjunto); err != nil {
		return nil, err
	}
	if err := r.saldosDeReposicao(ctx, empresa, conjunto); err != nil {
		return nil, err
	}
	if err := r.chegadasDeReposicao(ctx, empresa, conjunto); err != nil {
		return nil, err
	}
	for _, code := range ordem {
		// Item só com política de máscara é controlado por elas: o mínimo do cadastro
		// pediria o item sem máscara, que ninguém usa, com o saldo todo nas máscaras.
		if len(itens[code].Politicas) > 0 || len(conjunto.mascaras[code]) == 0 {
			out = append(out, itens[code].Sugestoes()...)
		}
		mascaras := make([]string, 0, len(conjunto.mascaras[code]))
		for m := range conjunto.mascaras[code] {
			mascaras = append(mascaras, m)
		}
		sort.Strings(mascaras)
		for _, m := range mascaras {
			out = append(out, conjunto.mascaras[code][m].Sugestoes()...)
		}
	}
	return out, nil
}

func (r *StockRepositorySQLC) itensDeReposicao(ctx context.Context, empresa int64) (map[int64]*entity.ItemReposicao, []int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT i.code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''),
		        COALESCE(i.warehouse_unit_of_measurement::text,''), COALESCE(i.engineering_type,0), i.planning_reorder_point,
		        COALESCE(i.warehouse_minimum_stock,0), COALESCE(i.planning_safety_stock,0),
		        COALESCE(i.planning_minimum_lot,0), COALESCE(i.planning_multiple_lot,0),
		        COALESCE((SELECT x.maximum_stock FROM item_planning_extras x
		                   WHERE x.enterprise_id=i.enterprise_id AND x.item_code=i.code), 0),
		        COALESCE((SELECT a.avg_monthly_consumption FROM item_consumption_averages a
		                   WHERE a.enterprise_id=i.enterprise_id AND a.item_code=i.code ORDER BY a.calculated_at DESC LIMIT 1), 0)
		   FROM items i
		  WHERE i.enterprise_id=$1 AND COALESCE(i.planning_active, TRUE) AND NOT COALESCE(i.planning_ghost, FALSE)
		    AND `+itemLiberadoSQL+`
		    AND (COALESCE(i.warehouse_minimum_stock,0) > 0
		         OR (i.planning_reorder_point IS NOT NULL AND i.planning_reorder_point <> 'null'::jsonb)
		         OR EXISTS (SELECT 1 FROM stock_replenishment_policies p
		                     WHERE p.enterprise_id=i.enterprise_id AND p.item_code=i.code AND p.is_active))
		  ORDER BY 2`, empresa)
	if err != nil {
		return nil, nil, fmt.Errorf("listing replenishment candidates: %w", err)
	}
	defer rows.Close()
	itens := map[int64]*entity.ItemReposicao{}
	var ordem []int64
	for rows.Next() {
		it := &entity.ItemReposicao{PorAlmox: map[int64]entity.SaldoReposicao{}}
		var ppRaw []byte
		var minimo decimal.Decimal
		if err := rows.Scan(&it.ItemCode, &it.Codigo, &it.Nome, &it.Unidade, &it.TipoItem, &ppRaw, &minimo, &it.Seguranca,
			&it.LoteMinimo, &it.LoteMultiplo, &it.Maximo, &it.ConsumoMedio); err != nil {
			return nil, nil, fmt.Errorf("scanning replenishment candidate: %w", err)
		}
		it.Ponto, it.Motivo = calcularPontoDePedido(ppRaw, minimo, it.Seguranca, it.ConsumoMedio)
		itens[it.ItemCode] = it
		ordem = append(ordem, it.ItemCode)
	}
	return itens, ordem, rows.Err()
}

func (r *StockRepositorySQLC) politicasDosItens(ctx context.Context, empresa int64, c itensPorMascara) error {
	lista, err := r.lerPoliticas(ctx, empresa, true)
	if err != nil {
		return err
	}
	for _, p := range lista {
		it := c.itens[p.ItemCode]
		if it == nil {
			continue
		}
		if p.Mask != "" {
			if c.mascaras[p.ItemCode] == nil {
				c.mascaras[p.ItemCode] = map[string]*entity.ItemReposicao{}
			}
			if c.mascaras[p.ItemCode][p.Mask] == nil {
				c.mascaras[p.ItemCode][p.Mask] = it.DaMascara(p.Mask)
			}
			it = c.mascaras[p.ItemCode][p.Mask]
		}
		it.Politicas = append(it.Politicas, p)
	}
	return nil
}

// saldosDeReposicao: saldo por almoxarifado, retido em lote e requisição aberta
// além do que conseguiu reservar. Total só soma o que conta para promessa.
func (r *StockRepositorySQLC) saldosDeReposicao(ctx context.Context, empresa int64, c itensPorMascara) error {
	rows, err := r.pool.Query(ctx,
		`WITH saldo AS (
		   SELECT b.item_code, b.mask, b.warehouse_id, SUM(b.quantity) fisico, SUM(b.reserved_qty) reservado
		     FROM stock_balances b WHERE b.enterprise_id=$1 GROUP BY 1, 2, 3),
		 retido AS (
		   SELECT b.item_code, b.mask, b.warehouse_id, SUM(b.quantity) qtd
		     FROM stock_lot_balances b WHERE b.enterprise_id=$1 AND b.quantity > 0 AND `+loteSemSaidaSQL+` GROUP BY 1, 2, 3),
		 requisitado AS (
		   SELECT l.item_code, COALESCE(l.mask,'') mask, q.warehouse_id,
		          SUM(GREATEST(l.quantity - l.issued_qty
		              - COALESCE((SELECT SUM(s.quantity) FROM stock_reservations s
		                           WHERE s.reference_type=$2 AND s.reference_item_code=l.id AND s.status='ACTIVE'), 0), 0)) qtd
		     FROM material_requisitions q JOIN material_requisition_items l ON l.requisition_id=q.id
		    WHERE q.enterprise_id=$1 AND q.status IN ('PENDENTE_APROVACAO','ABERTA','PARCIAL') GROUP BY 1, 2, 3)
		 SELECT w.id, `+almoxPrometivelSQL+`, x.item_code, x.mask, COALESCE(s.fisico,0), COALESCE(s.reservado,0), COALESCE(t.qtd,0), COALESCE(q.qtd,0)
		   FROM (SELECT item_code, mask, warehouse_id FROM saldo UNION SELECT item_code, mask, warehouse_id FROM requisitado) x
		   JOIN warehouse w ON w.id=x.warehouse_id AND w.enterprise_id=$1
		   LEFT JOIN saldo s ON s.item_code=x.item_code AND s.mask=x.mask AND s.warehouse_id=x.warehouse_id
		   LEFT JOIN retido t ON t.item_code=x.item_code AND t.mask=x.mask AND t.warehouse_id=x.warehouse_id
		   LEFT JOIN requisitado q ON q.item_code=x.item_code AND q.mask=x.mask AND q.warehouse_id=x.warehouse_id`,
		empresa, entity.ReferenceTypeRequisition)
	if err != nil {
		return fmt.Errorf("reading replenishment balances: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var almox, item int64
		var mask string
		var conta bool
		var s entity.SaldoReposicao
		if err := rows.Scan(&almox, &conta, &item, &mask, &s.Fisico, &s.Reservado, &s.Retido, &s.Requisitado); err != nil {
			return fmt.Errorf("scanning replenishment balance: %w", err)
		}
		it := c.de(item, mask)
		if it == nil {
			continue
		}
		it.PorAlmox[almox] = it.PorAlmox[almox].Somar(s)
		if conta {
			it.Total = it.Total.Somar(s)
		}
	}
	return rows.Err()
}

// chegadasDeReposicao soma o que já está a caminho. Sem almoxarifado (requisição
// de compra e pedido sem destino) fica em SemDestino, e o cálculo decide quem
// aproveita. Envio em trânsito conta no destino. Ordem planejada ainda não firme conta: firmada, ela vira a ordem de
// produção ou a requisição de compra que entra aqui pelo outro lado.
func (r *StockRepositorySQLC) chegadasDeReposicao(ctx context.Context, empresa int64, c itensPorMascara) error {
	rows, err := r.pool.Query(ctx,
		`SELECT p.item_code, COALESCE(p.mask,''), p.warehouse_id,
		        SUM(GREATEST(p.requested_qty - p.received_qty - p.cancelled_qty, 0)
		            * CASE WHEN p.internal_qty > 0 AND p.requested_qty > 0 THEN p.internal_qty / p.requested_qty ELSE 1 END)
		   FROM purchase_order_items p JOIN purchase_orders o ON o.code = p.purchase_order_code
		   JOIN enterprise e ON e.code = o.enterprise_code
		  WHERE e.id=$1 AND p.is_active AND o.is_active AND p.status IN ('OPEN','PARTIAL') AND o.status IN ('REQUESTED','APPROVED','PARTIAL')
		  GROUP BY 1, 2, 3
		 UNION ALL
		 SELECT ri.item_code, ri.mask, ri.warehouse_id, SUM(GREATEST(ri.quantity - ri.attended_qty - ri.cancelled_qty, 0))
		   FROM purchase_requisition_items ri JOIN purchase_requisitions rq ON rq.code = ri.requisition_code
		   JOIN enterprise e ON e.code = rq.enterprise_code
		  WHERE e.id=$1 AND ri.is_active AND rq.is_active AND rq.status IN ('OPEN','PARTIAL')
		  GROUP BY 1, 2, 3
		 UNION ALL
		 SELECT o.item_code, COALESCE(o.mask,''), o.warehouse_id, SUM(GREATEST(o.planned_qty - o.produced_qty - o.scrapped_qty, 0))
		   FROM production_orders o
		  WHERE o.enterprise_id=$1 AND o.status::text IN ('OPEN','PLANNED','RELEASED','IN_PROGRESS')
		  GROUP BY 1, 2, 3
		 UNION ALL
		 SELECT o.item_code, COALESCE(o.mask,''), o.warehouse_code, SUM(o.quantity)
		   FROM planned_orders o
		  WHERE o.enterprise_id=$1 AND o.is_active AND NOT o.is_firm AND o.status::text='PLANNED'
		  GROUP BY 1, 2, 3
		 UNION ALL
		 SELECT l.item_code, COALESCE(l.mask,''), q.destination_warehouse_id, SUM(GREATEST(l.quantity - l.issued_qty, 0))
		   FROM material_requisitions q JOIN material_requisition_items l ON l.requisition_id=q.id
		  WHERE q.enterprise_id=$1 AND q.kind='TRANSFERENCIA' AND q.status IN ('PENDENTE_APROVACAO','ABERTA','PARCIAL')
		  GROUP BY 1, 2, 3
		 UNION ALL
		 SELECT t.item_code, COALESCE(t.mask,''), t.to_warehouse_id, SUM(t.quantity - t.received_qty - t.returned_qty - t.lost_qty)
		   FROM stock_transit_shipments t
		  WHERE t.enterprise_id=$1 AND t.status='EM_TRANSITO'
		  GROUP BY 1, 2, 3`, empresa)
	if err != nil {
		return fmt.Errorf("reading replenishment receipts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item int64
		var mask string
		var almox *int64
		var q decimal.Decimal
		if err := rows.Scan(&item, &mask, &almox, &q); err != nil {
			return fmt.Errorf("scanning replenishment receipt: %w", err)
		}
		it := c.de(item, mask)
		if it == nil {
			continue
		}
		if almox == nil {
			it.SemDestino = it.SemDestino.Add(q)
			continue
		}
		s := it.PorAlmox[*almox]
		s.Chegando = s.Chegando.Add(q)
		it.PorAlmox[*almox] = s
		it.Total.Chegando = it.Total.Chegando.Add(q)
	}
	return rows.Err()
}

// calcularPontoDePedido: TR × CM / CR + ES do cadastro; CM zerado usa o consumo
// médio apurado (mensal, levado ao ciclo CR em dias); sem fórmula, o mínimo. O
// resultado já contém a segurança: a fórmula a soma, e no mínimo ela é o piso.
func calcularPontoDePedido(raw []byte, minimo, seguranca, consumoMensal decimal.Decimal) (decimal.Decimal, string) {
	var pp valueobject.ReorderPoint
	if len(raw) > 0 && string(raw) != "null" && json.Unmarshal(raw, &pp) == nil && pp.CR > 0 && pp.TR > 0 {
		cm := pp.CM
		origem := "ponto de pedido do cadastro"
		if !cm.IsPositive() && consumoMensal.IsPositive() {
			cm = consumoMensal.Mul(decimal.NewFromInt(int64(pp.CR))).Div(decimal.NewFromInt(30))
			origem = "ponto de pedido com o consumo médio apurado"
		}
		if cm.IsPositive() {
			es := pp.ES
			if !es.IsPositive() {
				es = seguranca
			}
			return valueobject.PontoDePedido(pp.TR, cm, pp.CR, es), origem
		}
	}
	if minimo.IsPositive() {
		return decimal.Max(minimo, seguranca), "estoque mínimo do cadastro"
	}
	return decimal.Zero, ""
}

// HojeDaEmpresa: perto da meia-noite o relógio do servidor já está em outro dia.
func (r *StockRepositorySQLC) HojeDaEmpresa(ctx context.Context) (time.Time, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return time.Time{}, err
	}
	fuso, err := r.fusoDaEmpresa(ctx, empresa)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().In(fuso), nil
}

// ---------- Política por almoxarifado ----------

func (r *StockRepositorySQLC) lerPoliticas(ctx context.Context, empresa int64, soAtivas bool) ([]entity.PoliticaReposicao, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, minimum_qty, maximum_qty, safety_qty, source_type, source_warehouse_id, is_active
		   FROM stock_replenishment_policies
		  WHERE enterprise_id=$1 AND (NOT $2 OR is_active) ORDER BY item_code, mask, warehouse_id`, empresa, soAtivas)
	if err != nil {
		return nil, fmt.Errorf("listing replenishment policies: %w", err)
	}
	defer rows.Close()
	out := []entity.PoliticaReposicao{}
	for rows.Next() {
		var p entity.PoliticaReposicao
		if err := rows.Scan(&p.ID, &p.ItemCode, &p.Mask, &p.WarehouseID, &p.Minimo, &p.Maximo, &p.Seguranca, &p.Origem, &p.AlmoxOrigem, &p.Ativa); err != nil {
			return nil, fmt.Errorf("scanning replenishment policy: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListarPoliticasDeReposicao devolve as políticas da empresa, ativas ou não.
func (r *StockRepositorySQLC) ListarPoliticasDeReposicao(ctx context.Context) ([]entity.PoliticaReposicao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return r.lerPoliticas(ctx, empresa, false)
}

// SalvarPoliticaDeReposicao cria ou troca a política do item no almoxarifado.
func (r *StockRepositorySQLC) SalvarPoliticaDeReposicao(ctx context.Context, p entity.PoliticaReposicao) (*entity.PoliticaReposicao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var tipoItem int
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(engineering_type,0) FROM items WHERE enterprise_id=$1 AND code=$2`, empresa, p.ItemCode).Scan(&tipoItem)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewValidationError("item da política não existe nesta empresa")
	}
	if err != nil {
		return nil, fmt.Errorf("reading policy item: %w", err)
	}
	if err := entity.OrigemAceitaParaItem(p.Origem, tipoItem); err != nil {
		return nil, err
	}
	var conta bool
	err = r.pool.QueryRow(ctx, `SELECT `+almoxPrometivelSQL+` FROM warehouse w WHERE w.id=$1 AND w.enterprise_id=$2`, p.WarehouseID, empresa).Scan(&conta)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewValidationError("almoxarifado da política não existe nesta empresa")
	}
	if err != nil {
		return nil, fmt.Errorf("reading policy warehouse: %w", err)
	}
	if !conta {
		return nil, errorsuc.NewValidationError("almoxarifado de inspeção, rejeição ou sem disponibilidade não tem reposição")
	}
	if p.AlmoxOrigem != nil {
		var ok bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM warehouse WHERE id=$1 AND enterprise_id=$2)`, *p.AlmoxOrigem, empresa).Scan(&ok); err != nil {
			return nil, fmt.Errorf("reading policy source warehouse: %w", err)
		}
		if !ok {
			return nil, errorsuc.NewValidationError("almoxarifado de origem não existe nesta empresa")
		}
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO stock_replenishment_policies (enterprise_id, item_code, warehouse_id, minimum_qty, maximum_qty, safety_qty,
		     source_type, source_warehouse_id, is_active, created_by, mask)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id) DO UPDATE
		   SET minimum_qty=EXCLUDED.minimum_qty, maximum_qty=EXCLUDED.maximum_qty, safety_qty=EXCLUDED.safety_qty,
		       source_type=EXCLUDED.source_type, source_warehouse_id=EXCLUDED.source_warehouse_id,
		       is_active=EXCLUDED.is_active, updated_at=NOW()
		 RETURNING id`,
		empresa, p.ItemCode, p.WarehouseID, p.Minimo, p.Maximo, p.Seguranca, p.Origem, p.AlmoxOrigem, p.Ativa, p.CreatedBy, p.Mask).Scan(&p.ID)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23514" {
			return nil, errorsuc.NewValidationError("confira a faixa de mínimo e máximo e a origem da reposição")
		}
		return nil, fmt.Errorf("saving replenishment policy: %w", err)
	}
	return &p, nil
}

// ExcluirPoliticaDeReposicao apaga a política: o item volta ao cadastro.
func (r *StockRepositorySQLC) ExcluirPoliticaDeReposicao(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM stock_replenishment_policies WHERE id=$1 AND enterprise_id=$2`, id, empresa)
	if err != nil {
		return fmt.Errorf("deleting replenishment policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("política de reposição não encontrada")
	}
	return nil
}
