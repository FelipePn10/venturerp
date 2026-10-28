// Package ordemservico grava a ordem de serviço da assistência técnica, as
// horas, as peças (saída do estoque na mesma transação) e as despesas, sempre
// da empresa da sessão.
package ordemservico

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/ordem"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	serierepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/serie"
	stockrepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ReferenciaEstoque: tipo de referência dos movimentos de peça da OS.
const ReferenciaEstoque = "ASSISTENCIA_OS"

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

// ─── Parâmetros ──────────────────────────────────────────────────────────────

func (r *PG) Parametros(ctx context.Context) (*dom.Parametros, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var p dom.Parametros
	err = r.pool.QueryRow(ctx, `SELECT p.custo_hora, p.preco_hora, p.custo_km, p.preco_km, p.warehouse_id, p.item_servico_code,
			COALESCE(i.business_code,''), COALESCE(i.name,'')
		  FROM os_parametros p LEFT JOIN items i ON i.code = p.item_servico_code AND i.enterprise_id = p.enterprise_id
		 WHERE p.enterprise_id=$1`, empresa).
		Scan(&p.CustoHora, &p.PrecoHora, &p.CustoKM, &p.PrecoKM, &p.WarehouseID, &p.ItemServicoCode, &p.ItemServico, &p.ItemServicoNome)
	if errors.Is(err, pgx.ErrNoRows) {
		return &p, nil
	}
	return &p, err
}

func (r *PG) SalvarParametros(ctx context.Context, p *dom.Parametros, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if p.WarehouseID != nil {
		var ok bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouse WHERE id=$1 AND enterprise_id=$2)`, *p.WarehouseID, empresa).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewNotFoundError(fmt.Sprintf("almoxarifado %d", *p.WarehouseID))
		}
	}
	if p.ItemServicoCode != nil {
		var ok bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2)`, *p.ItemServicoCode, empresa).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewNotFoundError(fmt.Sprintf("item %d", *p.ItemServicoCode))
		}
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO os_parametros (enterprise_id, custo_hora, preco_hora, custo_km, preco_km, warehouse_id, item_servico_code, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (enterprise_id) DO UPDATE SET custo_hora=EXCLUDED.custo_hora, preco_hora=EXCLUDED.preco_hora, custo_km=EXCLUDED.custo_km,
			preco_km=EXCLUDED.preco_km, warehouse_id=EXCLUDED.warehouse_id, item_servico_code=EXCLUDED.item_servico_code,
			updated_by=EXCLUDED.updated_by, updated_at=NOW()`,
		empresa, p.CustoHora, p.PrecoHora, p.CustoKM, p.PrecoKM, p.WarehouseID, p.ItemServicoCode, por)
	return err
}

// ─── dom.OS ──────────────────────────────────────────────────────────────────────

const colunas = `o.id, o.numero, o.call_code, c.call_number, o.call_item_code, o.serie_id, s.numero_serie, o.item_code, COALESCE(i.name,''),
	o.customer_code, COALESCE(cu.name,''), o.local, o.situacao, o.tecnico_code, COALESCE(e.name,''), o.agendada_para, o.iniciada_em, o.concluida_em,
	o.em_garantia, o.cobre_pecas, o.cobre_mao_obra, o.cobre_deslocamento, o.defeito_constatado, o.solucao, o.observacoes,
	o.motivo_cancelamento, o.sales_order_code, o.created_at, o.updated_at`

const de = ` FROM ordens_servico o
	LEFT JOIN technical_assistance_calls c ON c.code = o.call_code
	LEFT JOIN numeros_serie s ON s.id = o.serie_id AND s.enterprise_id = o.enterprise_id
	LEFT JOIN items i ON i.code = o.item_code AND i.enterprise_id = o.enterprise_id
	LEFT JOIN customers cu ON cu.code = o.customer_code AND cu.enterprise_id = o.enterprise_id
	LEFT JOIN employees e ON e.code = o.tecnico_code AND e.enterprise_id = o.enterprise_id`

func scan(row pgx.Row) (*dom.OS, error) {
	var o dom.OS
	err := row.Scan(&o.ID, &o.Numero, &o.CallCode, &o.CallNumero, &o.CallItemCode, &o.SerieID, &o.NumeroSerie, &o.ItemCode, &o.ItemNome,
		&o.CustomerCode, &o.ClienteNome, &o.Local, &o.Situacao, &o.TecnicoCode, &o.TecnicoNome, &o.AgendadaPara, &o.IniciadaEm, &o.ConcluidaEm,
		&o.EmGarantia, &o.CobrePecas, &o.CobreMaoObra, &o.CobreDeslocamento, &o.DefeitoConstatado, &o.Solucao, &o.Observacoes,
		&o.MotivoCancelamento, &o.SalesOrderCode, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// DoChamado lê o chamado da empresa e, se informado, o item dele.
func (r *PG) DoChamado(ctx context.Context, callCode int64, itemCode *int64) (*dom.Chamado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	c := dom.Chamado{CallCode: callCode}
	err = r.pool.QueryRow(ctx, `SELECT c.customer_code FROM technical_assistance_calls c JOIN enterprise e ON e.code = c.enterprise_code
		 WHERE c.code=$1 AND e.id=$2`, callCode, empresa).Scan(&c.CustomerCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("chamado %d", callCode))
	}
	if err != nil {
		return nil, err
	}
	if itemCode != nil {
		var em bool
		var pecas, mao, desloc *bool
		var item int64
		err = r.pool.QueryRow(ctx, `SELECT item_code, serie_id, in_warranty, cobre_pecas, cobre_mao_obra, cobre_deslocamento
			  FROM technical_assistance_call_items WHERE code=$1 AND call_code=$2`, *itemCode, callCode).Scan(&item, &c.SerieID, &em, &pecas, &mao, &desloc)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d do chamado %d", *itemCode, callCode))
		}
		if err != nil {
			return nil, err
		}
		c.CallItemCode, c.ItemCode = itemCode, &item
		// Sem cobertura gravada (item antigo), em garantia cobre peças e mão de obra.
		c.Cobertura = dom.Cobertura{EmGarantia: em, CobrePecas: valor(pecas, em), CobreMaoObra: valor(mao, em), CobreDeslocamento: valor(desloc, false)}
	}
	return &c, nil
}

func valor(b *bool, padrao bool) bool {
	if b == nil {
		return padrao
	}
	return *b
}

func (r *PG) Abrir(ctx context.Context, o *dom.OS, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('ordens_servico'), $1::int)`, empresa); err != nil {
		return err
	}
	if o.TecnicoCode != nil {
		if err := conferirTecnico(ctx, tx, empresa, *o.TecnicoCode); err != nil {
			return err
		}
	}
	err = tx.QueryRow(ctx, `INSERT INTO ordens_servico (enterprise_id, numero, call_code, call_item_code, serie_id, item_code, customer_code, local,
			situacao, tecnico_code, agendada_para, em_garantia, cobre_pecas, cobre_mao_obra, cobre_deslocamento, observacoes, created_by, updated_by)
		VALUES ($1, (SELECT COALESCE(MAX(numero),0)+1 FROM ordens_servico WHERE enterprise_id=$1), $2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$16)
		RETURNING id, numero`, empresa, o.CallCode, o.CallItemCode, o.SerieID, o.ItemCode, o.CustomerCode, o.Local, o.Situacao, o.TecnicoCode,
		o.AgendadaPara, o.EmGarantia, o.CobrePecas, o.CobreMaoObra, o.CobreDeslocamento, o.Observacoes, por).Scan(&o.ID, &o.Numero)
	if err != nil {
		return fmt.Errorf("abrindo a ordem de serviço: %w", err)
	}
	if err := evento(ctx, tx, empresa, o.ID, "ABERTURA", nil, o.Situacao, nil, por); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func conferirTecnico(ctx context.Context, q pgx.Tx, empresa, tecnico int64) error {
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employees WHERE code=$1 AND enterprise_id=$2)`, tecnico, empresa).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewNotFoundError(fmt.Sprintf("funcionário %d", tecnico))
	}
	return nil
}

func evento(ctx context.Context, tx pgx.Tx, empresa, osID int64, ev string, de *string, para string, detalhe *string, por *uuid.UUID) error {
	_, err := tx.Exec(ctx, `INSERT INTO os_eventos (enterprise_id, os_id, evento, de, para, detalhe, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		empresa, osID, ev, de, para, detalhe, por)
	if err != nil {
		return fmt.Errorf("gravando o histórico da OS: %w", err)
	}
	return nil
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.OS, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	o, err := scan(r.pool.QueryRow(ctx, `SELECT `+colunas+de+` WHERE o.enterprise_id=$1 AND o.id=$2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("ordem de serviço %d", id))
	}
	return o, err
}

func (r *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.OS, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+de+`
		 WHERE o.enterprise_id=$1 AND ($2='' OR o.situacao=$2) AND ($3::bigint IS NULL OR o.customer_code=$3)
		   AND ($4::bigint IS NULL OR o.call_code=$4) AND ($5::bigint IS NULL OR o.tecnico_code=$5)
		 ORDER BY o.numero DESC LIMIT 300`, empresa, f.Situacao, f.Cliente, f.Chamado, f.Tecnico)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.OS{}
	for rows.Next() {
		o, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (r *PG) Mudar(ctx context.Context, id int64, m dom.Mudanca) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := mudarTx(ctx, tx, empresa, id, m); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// situacaoTravada lê a situação da OS com a linha travada.
func situacaoTravada(ctx context.Context, tx pgx.Tx, empresa, id int64) (string, error) {
	var de string
	if err := tx.QueryRow(ctx, `SELECT situacao FROM ordens_servico WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, id).Scan(&de); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errorsuc.NewNotFoundError(fmt.Sprintf("ordem de serviço %d", id))
		}
		return "", err
	}
	return de, nil
}

// Faturar: pedido de venda e OS FATURADA juntos — antes eram passos soltos, e
// uma linha que falhasse deixava pedido pela metade (e dois cliques podiam
// faturar a mesma OS duas vezes).
func (r *PG) Faturar(ctx context.Context, id int64, f dom.Faturamento) (*orderentity.SalesOrder, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	de, err := situacaoTravada(ctx, tx, empresa, id)
	if err != nil {
		return nil, err
	}
	if err := dom.Transicao(de, dom.Faturada); err != nil {
		return nil, err
	}
	pedidos := salesorder.NewSalesOrderRepositorySQLC(sqlc.New(tx))
	numero, err := pedidos.NextOrderNumber(ctx, f.Pedido.EnterpriseCode)
	if err != nil {
		return nil, err
	}
	f.Pedido.OrderNumber = numero
	pedido, err := pedidos.Create(ctx, f.Pedido)
	if err != nil {
		return nil, fmt.Errorf("criando o pedido de faturamento da OS: %w", err)
	}
	for i, it := range f.Itens {
		it.SalesOrderCode = pedido.Code
		it.RecalcularTotais()
		if _, err := pedidos.CreateItem(ctx, it); err != nil {
			return nil, fmt.Errorf("criando a linha %d do pedido de faturamento: %w", i+1, err)
		}
	}
	if err := pedidos.RecalculateTotals(ctx, pedido.Code); err != nil {
		return nil, fmt.Errorf("totalizando o pedido de faturamento: %w", err)
	}
	m := f.Mudanca
	m.Para, m.SalesOrderCode = dom.Faturada, &pedido.Code
	if m.Detalhe == nil {
		d := fmt.Sprintf("pedido de venda %d", numero)
		m.Detalhe = &d
	}
	if err := mudarTx(ctx, tx, empresa, id, m); err != nil {
		return nil, err
	}
	return pedido, tx.Commit(ctx)
}

func mudarTx(ctx context.Context, tx pgx.Tx, empresa, id int64, m dom.Mudanca) error {
	var de string
	de, err := situacaoTravada(ctx, tx, empresa, id)
	if err != nil {
		return err
	}
	if err := dom.Transicao(de, m.Para); err != nil {
		return err
	}
	if m.TecnicoCode != nil {
		if err := conferirTecnico(ctx, tx, empresa, *m.TecnicoCode); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE ordens_servico SET situacao=$3::text,
			tecnico_code=COALESCE($4, tecnico_code), agendada_para=COALESCE($5, agendada_para),
			iniciada_em=CASE WHEN $3::text='EM_EXECUCAO' THEN COALESCE(iniciada_em, NOW()) ELSE iniciada_em END,
			concluida_em=CASE WHEN $3::text='CONCLUIDA' THEN NOW() WHEN $3::text='EM_EXECUCAO' THEN NULL ELSE concluida_em END,
			defeito_constatado=COALESCE($6, defeito_constatado), solucao=COALESCE($7, solucao),
			motivo_cancelamento=COALESCE($8, motivo_cancelamento), sales_order_code=COALESCE($9, sales_order_code),
			updated_by=$10, updated_at=NOW()
		WHERE enterprise_id=$1 AND id=$2`, empresa, id, m.Para, m.TecnicoCode, m.AgendadaPara, m.DefeitoConstatado, m.Solucao,
		m.MotivoCancelamento, m.SalesOrderCode, m.Por)
	if err != nil {
		return fmt.Errorf("atualizando a OS: %w", err)
	}
	if m.SerieEmAssistencia != nil {
		var serie *int64
		if err := tx.QueryRow(ctx, `SELECT serie_id FROM ordens_servico WHERE enterprise_id=$1 AND id=$2`, empresa, id).Scan(&serie); err != nil {
			return err
		}
		if serie != nil {
			if err := serierepository.AssistenciaTx(ctx, tx, empresa, *serie, id, *m.SerieEmAssistencia, m.Por); err != nil {
				return err
			}
		}
	}
	return evento(ctx, tx, empresa, id, "SITUACAO", &de, m.Para, m.Detalhe, m.Por)
}

// emLancamento trava a dom.OS e confere que ela aceita lançamentos.
func emLancamento(ctx context.Context, tx pgx.Tx, empresa, id int64) error {
	var situacao string
	if err := tx.QueryRow(ctx, `SELECT situacao FROM ordens_servico WHERE enterprise_id=$1 AND id=$2 FOR UPDATE`, empresa, id).Scan(&situacao); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("ordem de serviço %d", id))
		}
		return err
	}
	return dom.Lancamentos(situacao)
}

func (r *PG) emTx(ctx context.Context, fn func(tx pgx.Tx, empresa int64) error) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(tx, empresa); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) AddHora(ctx context.Context, osID int64, h *dom.Hora, por *uuid.UUID) error {
	return r.emTx(ctx, func(tx pgx.Tx, empresa int64) error {
		if err := emLancamento(ctx, tx, empresa, osID); err != nil {
			return err
		}
		if h.TecnicoCode != nil {
			if err := conferirTecnico(ctx, tx, empresa, *h.TecnicoCode); err != nil {
				return err
			}
		}
		return tx.QueryRow(ctx, `INSERT INTO os_horas (enterprise_id, os_id, tecnico_code, data, tipo, horas, custo_hora, preco_hora, cobrado, observacao, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, empresa, osID, h.TecnicoCode, h.Data, h.Tipo, h.Horas, h.CustoHora, h.PrecoHora,
			h.Cobrado, h.Observacao, por).Scan(&h.ID)
	})
}

func (r *PG) AddDespesa(ctx context.Context, osID int64, d *dom.Despesa, por *uuid.UUID) error {
	return r.emTx(ctx, func(tx pgx.Tx, empresa int64) error {
		if err := emLancamento(ctx, tx, empresa, osID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `INSERT INTO os_despesas (enterprise_id, os_id, tipo, quantidade, custo, preco, cobrado, observacao, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, empresa, osID, d.Tipo, d.Quantidade, d.Custo, d.Preco, d.Cobrado, d.Observacao, por).Scan(&d.ID)
	})
}

// AddPeca: a peça em garantia sai do estoque (custo médio) na transação do
// lançamento. A peça cobrada fica "a faturar" com o custo médio do momento como
// estimativa: ela sai do estoque pela NF do pedido de faturamento — sair aqui e
// de novo na nota baixaria duas vezes.
func (r *PG) AddPeca(ctx context.Context, osID int64, p *dom.Peca, por uuid.UUID) error {
	return r.emTx(ctx, func(tx pgx.Tx, empresa int64) error {
		if err := emLancamento(ctx, tx, empresa, osID); err != nil {
			return err
		}
		if p.Cobrado {
			// Custo estimado pela média: a saída real é a NF do pedido do faturamento.
			if err := tx.QueryRow(ctx, `SELECT COALESCE(CASE WHEN SUM(quantity) > 0 THEN ROUND(SUM(total_cost) / SUM(quantity), 10) ELSE MAX(avg_cost) END, 0)
			  FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`,
				empresa, p.ItemCode, p.WarehouseID).Scan(&p.CustoUnitario); err != nil {
				return fmt.Errorf("lendo o custo médio da peça: %w", err)
			}
			return tx.QueryRow(ctx, `INSERT INTO os_pecas (enterprise_id, os_id, item_code, quantidade, warehouse_id, custo_unitario, preco_unitario, cobrado,
					observacao, created_by)
				VALUES ($1,$2,$3,$4,$5,$6,$7,TRUE,$8,$9) RETURNING id`, empresa, osID, p.ItemCode, p.Quantidade, p.WarehouseID, p.CustoUnitario,
				p.PrecoUnitario, p.Observacao, por).Scan(&p.ID)
		}
		ref := ReferenciaEstoque
		nota := fmt.Sprintf("Peça aplicada na dom.OS (id %d)", osID)
		m := &stockentity.StockMovement{ItemCode: p.ItemCode, WarehouseID: p.WarehouseID, MovementType: stockentity.MovementTypeOut,
			Quantity: p.Quantidade, ReferenceType: &ref, ReferenceCode: &osID, Notes: &nota, CreatedBy: por}
		if err := stockrepository.CreateMovementTx(ctx, tx, empresa, m); err != nil {
			return err
		}
		p.CustoUnitario, p.StockMovementID = m.UnitPrice, &m.ID
		return tx.QueryRow(ctx, `INSERT INTO os_pecas (enterprise_id, os_id, item_code, quantidade, warehouse_id, custo_unitario, preco_unitario, cobrado,
				stock_movement_id, observacao, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, empresa, osID, p.ItemCode, p.Quantidade, p.WarehouseID, p.CustoUnitario,
			p.PrecoUnitario, p.Cobrado, p.StockMovementID, p.Observacao, por).Scan(&p.ID)
	})
}

// RemoverLancamento: hora e despesa somem; a peça volta ao estoque pelo mesmo custo.
func (r *PG) RemoverLancamento(ctx context.Context, osID int64, tipo string, id int64, por uuid.UUID) error {
	return r.emTx(ctx, func(tx pgx.Tx, empresa int64) error {
		if err := emLancamento(ctx, tx, empresa, osID); err != nil {
			return err
		}
		switch tipo {
		case "horas":
			tag, err := tx.Exec(ctx, `DELETE FROM os_horas WHERE enterprise_id=$1 AND os_id=$2 AND id=$3`, empresa, osID, id)
			if err == nil && tag.RowsAffected() == 0 {
				return errorsuc.NewNotFoundError("lançamento de horas")
			}
			return err
		case "despesas":
			tag, err := tx.Exec(ctx, `DELETE FROM os_despesas WHERE enterprise_id=$1 AND os_id=$2 AND id=$3`, empresa, osID, id)
			if err == nil && tag.RowsAffected() == 0 {
				return errorsuc.NewNotFoundError("despesa")
			}
			return err
		case "pecas":
			var p dom.Peca
			err := tx.QueryRow(ctx, `DELETE FROM os_pecas WHERE enterprise_id=$1 AND os_id=$2 AND id=$3
				RETURNING item_code, quantidade, warehouse_id, custo_unitario, stock_movement_id`, empresa, osID, id).
				Scan(&p.ItemCode, &p.Quantidade, &p.WarehouseID, &p.CustoUnitario, &p.StockMovementID)
			if errors.Is(err, pgx.ErrNoRows) {
				return errorsuc.NewNotFoundError("peça")
			}
			if err != nil {
				return err
			}
			if p.StockMovementID == nil {
				return nil // peça a faturar: não saiu do estoque
			}
			ref := ReferenciaEstoque
			nota := fmt.Sprintf("Peça devolvida da dom.OS (id %d)", osID)
			return stockrepository.CreateMovementTx(ctx, tx, empresa, &stockentity.StockMovement{ItemCode: p.ItemCode, WarehouseID: p.WarehouseID,
				MovementType: stockentity.MovementTypeIn, Quantity: p.Quantidade, UnitPrice: p.CustoUnitario, ReferenceType: &ref,
				ReferenceCode: &osID, Notes: &nota, CreatedBy: por})
		}
		return errorsuc.NewValidationError("tipo de lançamento desconhecido")
	})
}

func (r *PG) Lancamentos(ctx context.Context, osID int64) ([]dom.Hora, []dom.Peca, []dom.Despesa, []dom.Evento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	horas, pecas, despesas, eventos := []dom.Hora{}, []dom.Peca{}, []dom.Despesa{}, []dom.Evento{}
	rows, err := r.pool.Query(ctx, `SELECT id, tecnico_code, data, tipo, horas, custo_hora, preco_hora, cobrado, observacao FROM os_horas
		 WHERE enterprise_id=$1 AND os_id=$2 ORDER BY data, id`, empresa, osID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for rows.Next() {
		var h dom.Hora
		if err := rows.Scan(&h.ID, &h.TecnicoCode, &h.Data, &h.Tipo, &h.Horas, &h.CustoHora, &h.PrecoHora, &h.Cobrado, &h.Observacao); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		horas = append(horas, h)
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT p.id, p.item_code, COALESCE(i.name,''), p.quantidade, p.warehouse_id, p.custo_unitario, p.preco_unitario, p.cobrado,
			p.stock_movement_id, p.observacao
		  FROM os_pecas p LEFT JOIN items i ON i.code=p.item_code AND i.enterprise_id=p.enterprise_id
		 WHERE p.enterprise_id=$1 AND p.os_id=$2 ORDER BY p.id`, empresa, osID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for rows.Next() {
		var p dom.Peca
		if err := rows.Scan(&p.ID, &p.ItemCode, &p.ItemNome, &p.Quantidade, &p.WarehouseID, &p.CustoUnitario, &p.PrecoUnitario, &p.Cobrado,
			&p.StockMovementID, &p.Observacao); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		pecas = append(pecas, p)
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT id, tipo, quantidade, custo, preco, cobrado, observacao FROM os_despesas WHERE enterprise_id=$1 AND os_id=$2 ORDER BY id`, empresa, osID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for rows.Next() {
		var d dom.Despesa
		if err := rows.Scan(&d.ID, &d.Tipo, &d.Quantidade, &d.Custo, &d.Preco, &d.Cobrado, &d.Observacao); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		despesas = append(despesas, d)
	}
	rows.Close()
	rows, err = r.pool.Query(ctx, `SELECT id, evento, de, para, detalhe, ocorrido_em, created_by FROM os_eventos WHERE enterprise_id=$1 AND os_id=$2 ORDER BY ocorrido_em, id`, empresa, osID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for rows.Next() {
		var e dom.Evento
		if err := rows.Scan(&e.ID, &e.Evento, &e.De, &e.Para, &e.Detalhe, &e.OcorridoEm, &e.CreatedBy); err != nil {
			rows.Close()
			return nil, nil, nil, nil, err
		}
		eventos = append(eventos, e)
	}
	rows.Close()
	return horas, pecas, despesas, eventos, rows.Err()
}
