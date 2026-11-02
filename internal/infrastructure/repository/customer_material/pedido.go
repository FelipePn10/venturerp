package customer_material

import (
	"context"
	"errors"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VincularPedido grava o pedido da remessa. A liberação do bloqueio, quando
// pedida, vai na MESMA transação: vincular e continuar bloqueado por falta de
// pedido seria estado contraditório. O autor e o motivo ficam na trilha.
func (r *Repository) VincularPedido(ctx context.Context, id, salesOrderCode int64, desbloquear bool, usuario string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrir transação do vínculo do pedido: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	motivo := "pedido de beneficiamento vinculado"
	if err := definirAtorDaAuditoria(ctx, tx, usuario, &motivo); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
UPDATE public.customer_material_remittances
SET sales_order_code = $1,
    blocked      = CASE WHEN $2 THEN FALSE ELSE blocked END,
    block_reason = CASE WHEN $2 THEN NULL ELSE block_reason END,
    updated_at   = NOW()
WHERE id = $3 AND enterprise_id = $4`, salesOrderCode, desbloquear, id, enterpriseID)
	if err != nil {
		return traduzirErro(err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("remessa de beneficiamento não encontrada")
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar o vínculo do pedido: %w", err)
	}
	return nil
}

// SituacaoDoCliente lê o cadastro do dono do material. Cliente usa enterprise_id.
func (r *Repository) SituacaoDoCliente(ctx context.Context, code int64) (bool, bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, false, err
	}
	var ativo bool
	err = r.pool.QueryRow(ctx, `
SELECT is_active FROM public.customers WHERE code = $1 AND enterprise_id = $2`, code, enterpriseID).Scan(&ativo)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("ler o cadastro do cliente da remessa: %w", err)
	}
	return true, ativo, nil
}

// Pedidos lê o pedido de venda para o beneficiamento.
//
// ⚠️ Pedido de venda é do COMERCIAL e usa `enterprise_code` (tenant.Code), não
// `enterprise_id` como o razão de terceiros. Comparar com tenant.ID passaria na
// empresa 1 e esconderia o pedido na segunda empresa.
type Pedidos struct {
	pool *pgxpool.Pool
}

func NewPedidos(pool *pgxpool.Pool) *Pedidos { return &Pedidos{pool: pool} }

// VincularPedidoDaOF grava o pedido da OF. Só OF de beneficiamento: a de item comum
// não aparece no relatório de horas, e o vínculo ficaria sem efeito nenhum.
func (p *Pedidos) VincularPedidoDaOF(ctx context.Context, ofID int64, salesOrderCode *int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := p.pool.Exec(ctx, `
UPDATE public.production_orders o SET sales_order_code = $1
 WHERE o.id = $2 AND o.enterprise_id = $3 AND o.is_active
   AND EXISTS (SELECT 1 FROM public.item_structures s
                WHERE s.parent_code = o.item_code AND s.is_active AND s.customer_supplied)`,
		salesOrderCode, ofID, enterpriseID)
	if err != nil {
		return fmt.Errorf("vincular o pedido à OF %d: %w", ofID, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("ordem de fabricação de beneficiamento não encontrada")
	}
	return nil
}

// PedidoDeBeneficiamento devolve o pedido com o saldo de cada linha. O que está
// em nota ainda não autorizada conta como comprometido — exceto o rascunho DESTA
// remessa, que o faturamento retoma em vez de duplicar.
func (p *Pedidos) PedidoDeBeneficiamento(ctx context.Context, code int64, remessaID int64) (*entity.PedidoDeBeneficiamento, error) {
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	empresaID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	pedido := &entity.PedidoDeBeneficiamento{}
	err = p.pool.QueryRow(ctx, `
SELECT so.code, so.order_number, so.customer_code, so.status, so.is_active
FROM sales_orders so
WHERE so.code = $1 AND so.enterprise_code = $2`, code, empresaCode).
		Scan(&pedido.Code, &pedido.Numero, &pedido.CustomerCode, &pedido.Status, &pedido.Ativo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("pedido de venda %d não encontrado", code))
		}
		return nil, fmt.Errorf("ler o pedido de beneficiamento: %w", err)
	}

	rows, err := p.pool.Query(ctx, `
SELECT soi.code, soi.sequence, soi.item_code, COALESCE(i.business_code, soi.item_code::text),
       COALESCE(NULLIF(i.commercial_description, ''), NULLIF(i.pdm_description_technique, ''), i.name, ''),
       COALESCE(soi.sales_uom, ''), soi.requested_qty, soi.attended_qty, soi.cancelled_qty,
       COALESCE((
         SELECT SUM(fi.quantity)
           FROM fiscal_exit_items fi
           JOIN fiscal_exits fe ON fe.id = fi.fiscal_exit_id
          WHERE fi.sales_order_item_code = soi.code
            AND fe.enterprise_id = $2 AND fe.is_active
            AND fe.status IN ('DRAFT','AGUARDANDO_AUTORIZACAO')
            AND fe.customer_material_remittance_id IS DISTINCT FROM $3
       ), 0),
       soi.unit_price, COALESCE(soi.status, '')
FROM sales_order_items soi
LEFT JOIN items i ON i.code = soi.item_code AND i.enterprise_id = $2
WHERE soi.sales_order_code = $1 AND soi.is_active
ORDER BY soi.sequence, soi.code`, pedido.Code, empresaID, nuloSeZero(remessaID))
	if err != nil {
		return nil, fmt.Errorf("ler as linhas do pedido de beneficiamento: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var l entity.LinhaDoPedido
		if err := rows.Scan(&l.Code, &l.Sequencia, &l.ItemCode, &l.ItemBusinessCode, &l.Descricao, &l.Unidade,
			&l.Quantidade, &l.Atendida, &l.Cancelada, &l.EmNota, &l.PrecoUnitario, &l.Status); err != nil {
			return nil, err
		}
		pedido.Linhas = append(pedido.Linhas, l)
	}
	return pedido, rows.Err()
}

func nuloSeZero(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}
