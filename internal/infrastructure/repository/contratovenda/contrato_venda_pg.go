// Package contratovenda grava o contrato de venda, a programação e os pedidos
// de chamada (pedido, itens e vínculo numa transação), sempre da empresa da
// sessão.
package contratovenda

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/contratovenda"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	salesorder "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type db interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

func (r *PG) Criar(ctx context.Context, c *dom.Contrato, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE code=$1 AND enterprise_id=$2)`, c.CustomerCode, empresa).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d", c.CustomerCode))
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('contratos_venda'), $1::int)`, empresa); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO contratos_venda (enterprise_id, numero, customer_code, descricao, vigencia_inicio, vigencia_fim, valor_contratado,
			alerta_saldo_pct, alerta_dias, payment_term_code, observacao, created_by, updated_by)
		VALUES ($1, (SELECT COALESCE(MAX(numero),0)+1 FROM contratos_venda WHERE enterprise_id=$1), $2,$3,$4::date,$5::date,$6,$7,$8,$9,$10,$11,$11)
		RETURNING id, numero, situacao, created_at`, empresa, c.CustomerCode, c.Descricao, c.VigenciaInicio, c.VigenciaFim, c.ValorContratado,
		c.AlertaSaldoPct, c.AlertaDias, c.PaymentTermCode, c.Observacao, por).Scan(&c.ID, &c.Numero, &c.Situacao, &c.CreatedAt); err != nil {
		return fmt.Errorf("gravando o contrato: %w", err)
	}
	for i := range c.Itens {
		it := &c.Itens[i]
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE code=$1 AND enterprise_id=$2)`, it.ItemCode, empresa).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewNotFoundError(fmt.Sprintf("item %d", it.ItemCode))
		}
		if err := tx.QueryRow(ctx, `INSERT INTO contratos_venda_itens (enterprise_id, contrato_id, item_code, quantidade, preco_unitario) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			empresa, c.ID, it.ItemCode, it.Quantidade, it.PrecoUnitario).Scan(&it.ID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

const colunas = `c.id, c.numero, c.customer_code, COALESCE(cu.name,''), c.descricao, c.vigencia_inicio, c.vigencia_fim, c.valor_contratado, c.situacao,
	c.alerta_saldo_pct, c.alerta_dias, c.payment_term_code, c.observacao, c.motivo_encerramento, c.created_at,
	COALESCE((SELECT SUM((si.requested_qty - si.cancelled_qty) * si.unit_price) FROM contratos_venda_chamadas ch
	           JOIN sales_order_items si ON si.code=ch.sales_order_item_code
	          WHERE ch.contrato_id=c.id AND ch.enterprise_id=c.enterprise_id AND si.status <> 'CANCELLED'), 0)`

const de = ` FROM contratos_venda c LEFT JOIN customers cu ON cu.code=c.customer_code AND cu.enterprise_id=c.enterprise_id`

func scan(row pgx.Row) (*dom.Contrato, error) {
	var c dom.Contrato
	err := row.Scan(&c.ID, &c.Numero, &c.CustomerCode, &c.ClienteNome, &c.Descricao, &c.VigenciaInicio, &c.VigenciaFim, &c.ValorContratado, &c.Situacao,
		&c.AlertaSaldoPct, &c.AlertaDias, &c.PaymentTermCode, &c.Observacao, &c.MotivoEncerra, &c.CreatedAt, &c.ValorConsumido)
	return &c, err
}

func obter(ctx context.Context, q db, empresa, id int64, travar bool) (*dom.Contrato, error) {
	trava := ""
	if travar {
		trava = " FOR UPDATE OF c"
	}
	c, err := scan(q.QueryRow(ctx, `SELECT `+colunas+de+` WHERE c.enterprise_id=$1 AND c.id=$2`+trava, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("contrato de venda %d", id))
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT ci.id, ci.item_code, COALESCE(i.name,''), ci.quantidade, ci.preco_unitario,
			COALESCE((SELECT SUM(si.requested_qty - si.cancelled_qty) FROM contratos_venda_chamadas ch JOIN sales_order_items si ON si.code=ch.sales_order_item_code
			           WHERE ch.contrato_item_id=ci.id AND si.status <> 'CANCELLED'), 0)
		  FROM contratos_venda_itens ci LEFT JOIN items i ON i.code=ci.item_code AND i.enterprise_id=ci.enterprise_id
		 WHERE ci.enterprise_id=$1 AND ci.contrato_id=$2 ORDER BY ci.id`, empresa, id)
	if err != nil {
		return nil, err
	}
	c.Itens = []dom.Item{}
	for rows.Next() {
		var it dom.Item
		if err := rows.Scan(&it.ID, &it.ItemCode, &it.ItemNome, &it.Quantidade, &it.PrecoUnitario, &it.Consumido); err != nil {
			rows.Close()
			return nil, err
		}
		it.Saldo = it.Quantidade.Sub(it.Consumido)
		if it.Saldo.IsNegative() {
			it.Saldo = decimal.Zero
		}
		c.Itens = append(c.Itens, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	prows, err := q.Query(ctx, `SELECT p.id, p.contrato_item_id, ci.item_code, p.data_prevista, p.quantidade, p.sales_order_code
		  FROM contratos_venda_programacao p JOIN contratos_venda_itens ci ON ci.id=p.contrato_item_id
		 WHERE p.enterprise_id=$1 AND ci.contrato_id=$2 ORDER BY p.data_prevista, p.id`, empresa, id)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	c.Programacao = []dom.Parcela{}
	for prows.Next() {
		var p dom.Parcela
		if err := prows.Scan(&p.ID, &p.ContratoItemID, &p.ItemCode, &p.DataPrevista, &p.Quantidade, &p.SalesOrderCode); err != nil {
			return nil, err
		}
		c.Programacao = append(c.Programacao, p)
	}
	return c, prows.Err()
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.Contrato, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return obter(ctx, r.pool, empresa, id, false)
}

func (r *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.Contrato, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+de+` WHERE c.enterprise_id=$1 AND ($2::bigint IS NULL OR c.customer_code=$2) AND ($3='' OR c.situacao=$3)
		 ORDER BY c.numero DESC LIMIT 500`, empresa, f.Cliente, f.Situacao)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Contrato{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		c.Itens, c.Programacao = []dom.Item{}, []dom.Parcela{}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *PG) TemChamada(ctx context.Context, id int64) (bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contratos_venda_chamadas WHERE enterprise_id=$1 AND contrato_id=$2)`, empresa, id).Scan(&ok)
	return ok, err
}

func (r *PG) Mudar(ctx context.Context, id int64, de, para string, motivo *string, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE contratos_venda SET situacao=$4, motivo_encerramento=COALESCE($5, motivo_encerramento), updated_by=$6, updated_at=NOW()
		 WHERE enterprise_id=$1 AND id=$2 AND situacao=$3`, empresa, id, de, para, motivo, por)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewConflictError("a situação do contrato mudou enquanto você editava: atualize a tela")
	}
	return nil
}

func (r *PG) Programar(ctx context.Context, contratoItemID int64, data time.Time, qtd decimal.Decimal) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `INSERT INTO contratos_venda_programacao (enterprise_id, contrato_item_id, data_prevista, quantidade)
		SELECT $1, ci.id, $3::date, $4 FROM contratos_venda_itens ci JOIN contratos_venda c ON c.id=ci.contrato_id
		 WHERE ci.enterprise_id=$1 AND ci.id=$2 AND c.situacao IN ('RASCUNHO','ATIVO')`, empresa, contratoItemID, data, qtd)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("item de contrato não encontrado ou contrato encerrado")
	}
	return nil
}

func (r *PG) RemoverParcela(ctx context.Context, parcelaID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM contratos_venda_programacao WHERE enterprise_id=$1 AND id=$2 AND sales_order_code IS NULL`, empresa, parcelaID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("parcela não encontrada ou já atendida por pedido")
	}
	return nil
}

func (r *PG) Chamar(ctx context.Context, ch dom.Chamada) (*orderentity.SalesOrder, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if len(ch.Itens) != len(ch.Linhas) {
		return nil, fmt.Errorf("chamada com %d linhas e %d itens de pedido", len(ch.Linhas), len(ch.Itens))
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// O saldo é conferido de novo com o contrato travado: duas chamadas ao mesmo
	// tempo não passam do contratado.
	c, err := obter(ctx, tx, empresa, ch.ContratoID, true)
	if err != nil {
		return nil, err
	}
	if _, err := dom.ConferirChamada(c, ch.Linhas, ch.Data); err != nil {
		return nil, err
	}
	pedidos := salesorder.NewSalesOrderRepositorySQLC(sqlc.New(tx))
	numero, err := pedidos.NextOrderNumber(ctx, ch.Pedido.EnterpriseCode)
	if err != nil {
		return nil, err
	}
	ch.Pedido.OrderNumber = numero
	pedido, err := pedidos.Create(ctx, ch.Pedido)
	if err != nil {
		return nil, fmt.Errorf("criando o pedido de chamada: %w", err)
	}
	for i, it := range ch.Itens {
		it.SalesOrderCode = pedido.Code
		it.RecalcularTotais() // sem isto a linha nascia com total zero
		criado, err := pedidos.CreateItem(ctx, it)
		if err != nil {
			return nil, fmt.Errorf("criando o item %d do pedido de chamada: %w", i+1, err)
		}
		l := ch.Linhas[i]
		if _, err := tx.Exec(ctx, `INSERT INTO contratos_venda_chamadas (enterprise_id, contrato_id, contrato_item_id, sales_order_code, sales_order_item_code, created_by)
			VALUES ($1,$2,$3,$4,$5,$6)`, empresa, c.ID, l.ContratoItemID, pedido.Code, criado.Code, ch.Por); err != nil {
			return nil, err
		}
		if l.ParcelaID != nil {
			tag, err := tx.Exec(ctx, `UPDATE contratos_venda_programacao p SET sales_order_code=$3
				  FROM contratos_venda_itens ci WHERE ci.id=p.contrato_item_id AND p.enterprise_id=$1 AND p.id=$2
				   AND ci.id=$4 AND p.sales_order_code IS NULL`, empresa, *l.ParcelaID, pedido.Code, l.ContratoItemID)
			if err != nil {
				return nil, err
			}
			if tag.RowsAffected() == 0 {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("a parcela %d não é do item ou já foi atendida", *l.ParcelaID))
			}
		}
	}
	// A capa (crédito, comissão, faturamento) lê os totais do pedido.
	if err := pedidos.RecalculateTotals(ctx, pedido.Code); err != nil {
		return nil, fmt.Errorf("totalizando o pedido de chamada: %w", err)
	}
	return pedido, tx.Commit(ctx)
}

// PrecoContratado: o preço do contrato da linha de pedido de chamada (nil quando
// a linha não veio de contrato). A alteração da linha mantém esse preço em vez
// de repreçar pela tabela.
func (r *PG) PrecoContratado(ctx context.Context, salesOrderItemCode int64) (*decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var p decimal.Decimal
	err = r.pool.QueryRow(ctx, `SELECT i.preco_unitario FROM contratos_venda_chamadas ch
		JOIN contratos_venda_itens i ON i.enterprise_id=ch.enterprise_id AND i.id=ch.contrato_item_id
		WHERE ch.enterprise_id=$1 AND ch.sales_order_item_code=$2`, empresa, salesOrderItemCode).Scan(&p)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
