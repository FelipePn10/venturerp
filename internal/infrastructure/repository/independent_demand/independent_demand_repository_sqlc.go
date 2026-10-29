package independent_demand

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *IndependentDemandRepositorySQLC) Create(
	ctx context.Context,
	d *entity.IndependentDemand,
) (*entity.IndependentDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.CreateIndependentDemand(ctx, sqlc.CreateIndependentDemandParams{
		Code:           d.CodeDemand,
		ItemCode:       d.ItemCode,
		Mask:           pgutil.ToPgTextFromPtr(d.Mask),
		CostCenterCode: pgutil.ToPgInt8Ptr(d.CostCenterCode),
		Quantity:       pgutil.ToPgNumericFromDecimal(d.Quantity),
		DemandDate:     pgutil.ToPgDate(d.DemandDate),
		CreatedBy:      pgutil.ToPgUUID(d.CreatedBy),
		EnterpriseID:   enterpriseID,
		Motivo:         motivoOuPadrao(d.Motivo),
		WarehouseID:    d.WarehouseID,
		CustomerCode:   d.CustomerCode,
		Projeto:        pgutil.ToPgTextFromPtr(d.Projeto),
		Prioridade:     int16(prioridadeOuPadrao(d.Prioridade)),
		Observacao:     pgutil.ToPgTextFromPtr(d.Observacao),
		Programacao:    uuidPtr(d.Programacao),
	})
	if err != nil {
		if strings.Contains(err.Error(), "uq_independent_demands_empresa_codigo") {
			return nil, fmt.Errorf("%w: %d", repository.ErrCodigoEmUso, d.CodeDemand)
		}
		return nil, fmt.Errorf("creating independent demand: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *IndependentDemandRepositorySQLC) Update(
	ctx context.Context,
	d *entity.IndependentDemand,
) (*entity.IndependentDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	var por pgtype.UUID
	if d.UpdatedBy != nil {
		por = pgutil.ToPgUUID(*d.UpdatedBy)
	}
	row, err := r.q.UpdateIndependentDemand(ctx, sqlc.UpdateIndependentDemandParams{
		ItemCode:       d.ItemCode,
		Mask:           pgutil.ToPgTextFromPtr(d.Mask),
		CostCenterCode: pgutil.ToPgInt8Ptr(d.CostCenterCode),
		Quantity:       pgutil.ToPgNumericFromDecimal(d.Quantity),
		DemandDate:     pgutil.ToPgDate(d.DemandDate),
		Code:           d.CodeDemand,
		Motivo:         motivoOuPadrao(d.Motivo),
		WarehouseID:    d.WarehouseID,
		CustomerCode:   d.CustomerCode,
		Projeto:        pgutil.ToPgTextFromPtr(d.Projeto),
		Prioridade:     int16(prioridadeOuPadrao(d.Prioridade)),
		Observacao:     pgutil.ToPgTextFromPtr(d.Observacao),
		UpdatedBy:      por,
		EnterpriseID:   enterpriseID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a demanda %d não está aberta: encerrada ou cancelada não se altera", d.CodeDemand))
		}
		return nil, fmt.Errorf("updating independent demand: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *IndependentDemandRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.IndependentDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	row, err := r.q.GetIndependentDemandByCode(ctx, sqlc.GetIndependentDemandByCodeParams{Code: code, EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("demanda independente %d não encontrada", code))
		}

		return nil, fmt.Errorf("fetching independent demand: %w", err)
	}

	out, err := r.comAtendido(ctx, []*entity.IndependentDemand{rowToEntity(row)})
	if err != nil {
		return nil, err
	}
	return out[0], nil
}

func (r *IndependentDemandRepositorySQLC) ListFromDate(
	ctx context.Context,
	date time.Time,
) ([]*entity.IndependentDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.q.ListDemandsFromDate(ctx, sqlc.ListDemandsFromDateParams{DemandDate: pgutil.ToPgDate(date), EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("listing demands from date: %w", err)
	}

	return r.comAtendido(ctx, rowsToEntities(rows))
}

func (r *IndependentDemandRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.IndependentDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.q.ListIndependentDemands(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing independent demands: %w", err)
	}

	return r.comAtendido(ctx, rowsToEntities(rows))
}

func (r *IndependentDemandRepositorySQLC) ListByItem(
	ctx context.Context,
	itemCode int64,
) ([]*entity.IndependentDemand, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.q.ListDemandsByItem(ctx, sqlc.ListDemandsByItemParams{ItemCode: itemCode, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("listing independent demands by item: %w", err)
	}

	return r.comAtendido(ctx, rowsToEntities(rows))
}

func (r *IndependentDemandRepositorySQLC) Delete(
	ctx context.Context,
	code int64,
) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}

	err = r.q.DeleteIndependentDemand(ctx, sqlc.DeleteIndependentDemandParams{Code: code, EnterpriseID: enterpriseID})
	if err != nil {
		return fmt.Errorf("deleting independent demand %d: %w", code, err)
	}

	return nil
}

// ── Acompanhamento ───────────────────────────────────────────────────────────

func (r *IndependentDemandRepositorySQLC) NextCode(ctx context.Context) (int64, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	code, err := r.q.NextIndependentDemandCode(ctx, enterpriseID)
	if err != nil {
		return 0, fmt.Errorf("next independent demand code: %w", err)
	}
	return code, nil
}

func (r *IndependentDemandRepositorySQLC) TravarNumeracao(ctx context.Context) error {
	if r.db == nil {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if _, err := r.db.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('independent_demands.code'), $1::int)`, enterpriseID); err != nil {
		return fmt.Errorf("locking demand numbering: %w", err)
	}
	return nil
}

func (r *IndependentDemandRepositorySQLC) Encerrar(ctx context.Context, code int64, situacao, motivo string, por uuid.UUID) (bool, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return false, err
	}
	n, err := r.q.CloseIndependentDemand(ctx, sqlc.CloseIndependentDemandParams{
		Situacao: situacao, EncerradaPor: pgutil.ToPgUUID(por), MotivoEncerramento: pgutil.ToPgTextFromString(motivo),
		Code: code, EnterpriseID: enterpriseID,
	})
	if err != nil {
		return false, fmt.Errorf("closing independent demand %d: %w", code, err)
	}
	return n == 1, nil
}

const colunasDemanda = `id, code, item_code, mask, cost_center_code, quantity, demand_date, is_active, created_at, updated_at, created_by,
	motivo, warehouse_id, customer_code, projeto, prioridade, observacao, programacao, situacao, encerrada_em, encerrada_por,
	motivo_encerramento, updated_by`

func (r *IndependentDemandRepositorySQLC) Buscar(ctx context.Context, f entity.Filtro) ([]*entity.IndependentDemand, error) {
	if r.db == nil {
		return nil, errors.New("acompanhamento da demanda indisponível nesta montagem")
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	conds := []string{"enterprise_id = $1"}
	args := []any{enterpriseID}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.ItemCode != nil {
		add("item_code = $%d", *f.ItemCode)
	}
	if f.Motivo != "" {
		add("motivo = $%d", f.Motivo)
	}
	if f.Situacao != "" {
		add("situacao = $%d", f.Situacao)
	}
	if f.De != nil {
		add("demand_date >= $%d", *f.De)
	}
	if f.Ate != nil {
		add("demand_date <= $%d", *f.Ate)
	}
	limite := f.Limite
	if limite <= 0 || limite > 1000 {
		limite = 500
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s FROM independent_demands WHERE %s
		ORDER BY (situacao = 'ABERTA') DESC, demand_date, code LIMIT %d`, colunasDemanda, strings.Join(conds, " AND "), limite), args...)
	if err != nil {
		return nil, fmt.Errorf("searching independent demands: %w", err)
	}
	defer rows.Close()
	var out []*entity.IndependentDemand
	for rows.Next() {
		var row sqlc.IndependentDemand
		if err := rows.Scan(&row.ID, &row.Code, &row.ItemCode, &row.Mask, &row.CostCenterCode, &row.Quantity, &row.DemandDate,
			&row.IsActive, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy, &row.Motivo, &row.WarehouseID, &row.CustomerCode,
			&row.Projeto, &row.Prioridade, &row.Observacao, &row.Programacao, &row.Situacao, &row.EncerradaEm, &row.EncerradaPor,
			&row.MotivoEncerramento, &row.UpdatedBy); err != nil {
			return nil, fmt.Errorf("scanning independent demand: %w", err)
		}
		out = append(out, rowToEntity(row))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return r.comAtendido(ctx, out)
}

func (r *IndependentDemandRepositorySQLC) Ordens(ctx context.Context, demandID int64) ([]entity.OrdemDaDemanda, error) {
	porDemanda, err := r.ordensDe(ctx, []int64{demandID})
	if err != nil {
		return nil, err
	}
	return porDemanda[demandID], nil
}

// Cobertura soma, por demanda, o que as ordens ligadas cobrem (a cancelada
// não cobre nada).
func (r *IndependentDemandRepositorySQLC) Cobertura(ctx context.Context, demandIDs []int64) (map[int64]decimal.Decimal, error) {
	porDemanda, err := r.ordensDe(ctx, demandIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]decimal.Decimal, len(porDemanda))
	for id, ordens := range porDemanda {
		out[id] = entity.Coberta(ordens)
	}
	return out, nil
}

func (r *IndependentDemandRepositorySQLC) ordensDe(ctx context.Context, demandIDs []int64) (map[int64][]entity.OrdemDaDemanda, error) {
	out := map[int64][]entity.OrdemDaDemanda{}
	if r.db == nil || len(demandIDs) == 0 {
		return out, nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// A OF vem direto (Gerar OF agora) ou pela ordem planejada firmada. A
	// ordem planejada excluída (is_active falso, status intocado) sem OF viva
	// não cobre mais nada: sai como cancelada.
	rows, err := r.db.Query(ctx, `
		SELECT v.demand_id, po.code, COALESCE(dof.id, pof.id), COALESCE(dof.order_number, pof.order_number, po.order_number),
		       COALESCE(dof.status, pof.status, CASE WHEN po.is_active THEN po.status::text ELSE 'CANCELLED' END, ''), v.quantity,
		       COALESCE(dof.planned_qty, pof.planned_qty, po.quantity, 0), COALESCE(dof.produced_qty, pof.produced_qty, 0)
		  FROM independent_demand_orders v
		  LEFT JOIN planned_orders po ON po.id = v.planned_order_id AND po.enterprise_id = v.enterprise_id
		  LEFT JOIN production_orders dof ON dof.id = v.production_order_id AND dof.enterprise_id = v.enterprise_id
		  LEFT JOIN LATERAL (SELECT p.* FROM production_orders p
		                      WHERE p.planned_order_id = v.planned_order_id AND p.enterprise_id = v.enterprise_id AND p.is_active
		                      ORDER BY p.id DESC LIMIT 1) pof ON v.production_order_id IS NULL
		 WHERE v.demand_id = ANY($1) AND v.enterprise_id = $2
		 ORDER BY v.id`, demandIDs, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing orders of demands: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var demandID int64
		var o entity.OrdemDaDemanda
		var qtd, plan, prod pgtype.Numeric
		if err := rows.Scan(&demandID, &o.PlannedOrderCode, &o.ProductionOrderID, &o.OrderNumber, &o.Status, &qtd, &plan, &prod); err != nil {
			return nil, fmt.Errorf("scanning demand order: %w", err)
		}
		o.Quantidade = pgutil.FromPgNumericToDecimal(qtd)
		o.Planejada = pgutil.FromPgNumericToDecimal(plan)
		o.Produzida = pgutil.FromPgNumericToDecimal(prod)
		out[demandID] = append(out[demandID], o)
	}
	return out, rows.Err()
}

// TravarDemanda trava a demanda até o fim da transação: duas ações ao mesmo
// tempo (dois "Atender") não passam do saldo.
func (r *IndependentDemandRepositorySQLC) TravarDemanda(ctx context.Context, code int64) error {
	if r.db == nil {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if _, err := r.db.Exec(ctx, `SELECT 1 FROM independent_demands WHERE code = $1 AND enterprise_id = $2 FOR UPDATE`, code, enterpriseID); err != nil {
		return fmt.Errorf("locking demand %d: %w", code, err)
	}
	return nil
}

// AlmoxarifadoDaEmpresa diz se o almoxarifado é da empresa da sessão.
func (r *IndependentDemandRepositorySQLC) AlmoxarifadoDaEmpresa(ctx context.Context, id int64) (bool, error) {
	if r.db == nil {
		return true, nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM warehouse WHERE id = $1 AND enterprise_id = $2)`, id, enterpriseID).Scan(&ok); err != nil {
		return false, fmt.Errorf("checking warehouse %d: %w", id, err)
	}
	return ok, nil
}

// VincularSugestao grava, pelo pegging da sugestão firmada, a ordem planejada
// de cada demanda independente que ela cobre. Idempotente: a demanda já ligada
// a essa ordem não ganha outro vínculo.
func (r *IndependentDemandRepositorySQLC) VincularSugestao(ctx context.Context, suggestionCode, plannedOrderCode int64, por uuid.UUID) error {
	if r.db == nil {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO independent_demand_orders (enterprise_id, demand_id, planned_order_id, quantity, created_by)
		SELECT $1, d.id, po.id, SUM(p.quantity), $4
		  FROM mrp_pegging p
		  JOIN independent_demands d ON d.code = p.demand_code AND d.enterprise_id = p.enterprise_id
		  JOIN planned_orders po ON po.code = $3 AND po.enterprise_id = $1
		 WHERE p.enterprise_id = $1 AND p.supply_kind = 'SUGESTAO' AND p.supply_code = $2
		   AND p.demand_kind = 'DEMANDA_INDEPENDENTE' AND p.quantity > 0
		   AND NOT EXISTS (SELECT 1 FROM independent_demand_orders x
		                    WHERE x.enterprise_id = $1 AND x.demand_id = d.id AND x.planned_order_id = po.id)
		 GROUP BY d.id, po.id`, enterpriseID, suggestionCode, plannedOrderCode, pgutil.ToPgUUID(por))
	if err != nil {
		return fmt.Errorf("linking demands of suggestion %d: %w", suggestionCode, err)
	}
	return nil
}

// Vendas: linhas de pedido de venda da empresa (abertas ou já atendidas) com a
// quantidade vendida sem a cancelada. Só é venda o pedido (P), o pedido em
// análise (A) e o faturado (F): rascunho e orçamento não consomem nada, e o
// bloqueado fica de fora, como na demanda do MRP. Data = entrega da linha ou do
// pedido; sem ela, a linha aberta vale hoje (a mesma data que o MRP dá à
// necessidade dela) e a atendida, a emissão.
func (r *IndependentDemandRepositorySQLC) Vendas(ctx context.Context, itens []int64) ([]entity.Venda, error) {
	if r.db == nil || len(itens) == 0 {
		return nil, nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT soi.code, so.order_number, soi.sequence, COALESCE(NULLIF(cu.trade_name, ''), cu.name, ''),
		       soi.item_code, COALESCE(soi.mask, ''),
		       COALESCE(soi.delivery_date, so.delivery_date,
		                CASE WHEN so.status IN ('P', 'A') AND soi.status IN ('OPEN', 'PARTIAL')
		                          AND soi.requested_qty > soi.attended_qty + soi.cancelled_qty
		                     THEN $3::date ELSE so.emission_date::date END)::date,
		       soi.requested_qty - soi.cancelled_qty
		  FROM sales_order_items soi
		  JOIN sales_orders so ON so.code = soi.sales_order_code
		  JOIN enterprise e ON e.code = so.enterprise_code AND e.id = $1
		  LEFT JOIN customers cu ON cu.code = so.customer_code AND cu.enterprise_id = e.id
		 WHERE soi.item_code = ANY($2)
		   AND so.is_active AND NOT so.is_blocked AND so.status IN ('P', 'A', 'F')
		   AND soi.is_active AND soi.status <> 'CANCELLED'
		   AND soi.requested_qty > soi.cancelled_qty
		 ORDER BY 7, 2, 3`, enterpriseID, itens, mrpservice.Hoje())
	if err != nil {
		return nil, fmt.Errorf("listing sales for consumption: %w", err)
	}
	defer rows.Close()
	var out []entity.Venda
	for rows.Next() {
		var linha, pedido, item int64
		var sequencia int
		var cliente, mask string
		var data time.Time
		var qtd pgtype.Numeric
		if err := rows.Scan(&linha, &pedido, &sequencia, &cliente, &item, &mask, &data, &qtd); err != nil {
			return nil, fmt.Errorf("scanning sale for consumption: %w", err)
		}
		out = append(out, entity.Venda{Chave: entity.ChaveItem(item, mask), Data: data, Quantidade: pgutil.FromPgNumericToDecimal(qtd), Linha: &linha,
			Pedido: pedido, Sequencia: sequencia, Cliente: cliente})
	}
	return out, rows.Err()
}

func (r *IndependentDemandRepositorySQLC) ClienteDaEmpresa(ctx context.Context, code int64) (bool, error) {
	if r.db == nil {
		return true, nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM customers WHERE code = $1 AND enterprise_id = $2)`, code, enterpriseID).Scan(&ok); err != nil {
		return false, fmt.Errorf("checking customer %d: %w", code, err)
	}
	return ok, nil
}

func (r *IndependentDemandRepositorySQLC) IDDoCentro(ctx context.Context, code int64) (*int64, error) {
	if r.db == nil {
		return nil, nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	err = r.db.QueryRow(ctx, `SELECT id FROM cost_centers WHERE code = $1 AND enterprise_id = $2`, code, enterpriseID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("centro de custo %d não encontrado nesta empresa", code))
	}
	if err != nil {
		return nil, fmt.Errorf("resolving cost center %d: %w", code, err)
	}
	return &id, nil
}

// comAtendido preenche o atendido e o rótulo do item; sem db a leitura sai
// sem eles.
func (r *IndependentDemandRepositorySQLC) comAtendido(ctx context.Context, ds []*entity.IndependentDemand) ([]*entity.IndependentDemand, error) {
	if err := r.preencherAtendido(ctx, ds); err != nil {
		return nil, err
	}
	if err := r.preencherItens(ctx, ds); err != nil {
		return nil, err
	}
	if err := r.preencherDestinos(ctx, ds); err != nil {
		return nil, err
	}
	return ds, nil
}

// preencherDestinos dá nome ao almoxarifado e ao cliente: a tela não mostra
// o id interno.
func (r *IndependentDemandRepositorySQLC) preencherDestinos(ctx context.Context, ds []*entity.IndependentDemand) error {
	if r.db == nil || len(ds) == 0 {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var almox, clientes []int64
	for _, d := range ds {
		if d.WarehouseID != nil {
			almox = append(almox, *d.WarehouseID)
		}
		if d.CustomerCode != nil {
			clientes = append(clientes, *d.CustomerCode)
		}
	}
	nomes := func(sql string, codes []int64) (map[int64]string, error) {
		out := map[int64]string{}
		if len(codes) == 0 {
			return out, nil
		}
		rows, err := r.db.Query(ctx, sql, enterpriseID, codes)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var code int64
			var nome string
			if err := rows.Scan(&code, &nome); err != nil {
				return nil, err
			}
			out[code] = nome
		}
		return out, rows.Err()
	}
	nomesAlmox, err := nomes(`SELECT id, code::text || ' — ' || COALESCE(description, '') FROM warehouse
		WHERE enterprise_id = $1 AND id = ANY($2)`, almox)
	if err != nil {
		return fmt.Errorf("reading warehouses of demands: %w", err)
	}
	nomesClientes, err := nomes(`SELECT code, COALESCE(NULLIF(trade_name, ''), name, '') FROM customers
		WHERE enterprise_id = $1 AND code = ANY($2)`, clientes)
	if err != nil {
		return fmt.Errorf("reading customers of demands: %w", err)
	}
	for _, d := range ds {
		if d.WarehouseID != nil {
			d.WarehouseNome = nomesAlmox[*d.WarehouseID]
		}
		if d.CustomerCode != nil {
			d.CustomerNome = nomesClientes[*d.CustomerCode]
		}
	}
	return nil
}

func (r *IndependentDemandRepositorySQLC) preencherItens(ctx context.Context, ds []*entity.IndependentDemand) error {
	if r.db == nil || len(ds) == 0 {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	codes := make([]int64, 0, len(ds))
	for _, d := range ds {
		codes = append(codes, d.ItemCode)
	}
	rows, err := r.db.Query(ctx, `SELECT code, COALESCE(business_code, ''), COALESCE(name, '') FROM items
		WHERE enterprise_id = $1 AND code = ANY($2)`, enterpriseID, codes)
	if err != nil {
		return fmt.Errorf("reading items of demands: %w", err)
	}
	defer rows.Close()
	type rotulo struct{ codigo, nome string }
	itens := map[int64]rotulo{}
	for rows.Next() {
		var code int64
		var r rotulo
		if err := rows.Scan(&code, &r.codigo, &r.nome); err != nil {
			return fmt.Errorf("scanning item of demand: %w", err)
		}
		itens[code] = r
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, d := range ds {
		d.ItemBusinessCode, d.ItemName = itens[d.ItemCode].codigo, itens[d.ItemCode].nome
	}
	return nil
}

// preencherAtendido: saídas de estoque com a referência da demanda, sem as
// estornadas.
func (r *IndependentDemandRepositorySQLC) preencherAtendido(ctx context.Context, ds []*entity.IndependentDemand) error {
	if r.db == nil || len(ds) == 0 {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	codes := make([]int64, 0, len(ds))
	for _, d := range ds {
		codes = append(codes, d.CodeDemand)
	}
	rows, err := r.db.Query(ctx, `
		SELECT reference_code, SUM(ABS(quantity)) FROM stock_movements
		 WHERE enterprise_id = $1 AND reference_type = $2 AND reference_code = ANY($3)
		   AND movement_type = 'OUT' AND reversed_by_movement_id IS NULL
		 GROUP BY reference_code`, enterpriseID, entity.ReferenciaAtendimento, codes)
	if err != nil {
		return fmt.Errorf("reading attended quantity of demands: %w", err)
	}
	defer rows.Close()
	atendido := map[int64]decimal.Decimal{}
	for rows.Next() {
		var code int64
		var qtd pgtype.Numeric
		if err := rows.Scan(&code, &qtd); err != nil {
			return fmt.Errorf("scanning attended quantity: %w", err)
		}
		atendido[code] = pgutil.FromPgNumericToDecimal(qtd)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, d := range ds {
		d.Atendida = atendido[d.CodeDemand]
	}
	return nil
}

func rowToEntity(row sqlc.IndependentDemand) *entity.IndependentDemand {
	e := &entity.IndependentDemand{
		ID:           row.ID,
		CodeDemand:   row.Code,
		ItemCode:     row.ItemCode,
		Quantity:     pgutil.FromPgNumericToDecimal(row.Quantity),
		DemandDate:   pgutil.FromPgDate(row.DemandDate),
		IsActive:     row.IsActive,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:    pgutil.FromPgUUID(row.CreatedBy),
		Motivo:       row.Motivo,
		WarehouseID:  row.WarehouseID,
		CustomerCode: row.CustomerCode,
		Projeto:      pgutil.FromPgTextPtr(row.Projeto),
		Prioridade:   int(row.Prioridade),
		Observacao:   pgutil.FromPgTextPtr(row.Observacao),
		Situacao:     row.Situacao,
	}
	if row.EncerradaEm.Valid {
		t := row.EncerradaEm.Time
		e.EncerradaEm = &t
	}

	if row.Mask.Valid {
		v := row.Mask.String
		e.Mask = &v
	}
	if row.CostCenterCode.Valid {
		v := row.CostCenterCode.Int64
		e.CostCenterCode = &v
	}
	if row.Programacao.Valid {
		v := uuid.UUID(row.Programacao.Bytes)
		e.Programacao = &v
	}
	if row.EncerradaPor.Valid {
		v := uuid.UUID(row.EncerradaPor.Bytes)
		e.EncerradaPor = &v
	}
	if row.UpdatedBy.Valid {
		v := uuid.UUID(row.UpdatedBy.Bytes)
		e.UpdatedBy = &v
	}
	e.MotivoEncerramento = pgutil.FromPgTextPtr(row.MotivoEncerramento)

	return e
}

func rowsToEntities(
	rows []sqlc.IndependentDemand,
) []*entity.IndependentDemand {

	out := make([]*entity.IndependentDemand, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}

func motivoOuPadrao(m string) string {
	if m == "" {
		return entity.MotivoEstoque
	}
	return m
}

func prioridadeOuPadrao(p int) int {
	if p == 0 {
		return 3
	}
	return p
}

func uuidPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}
