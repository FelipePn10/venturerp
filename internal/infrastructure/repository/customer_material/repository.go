// Package customer_material grava o material do cliente em poder da empresa
// (beneficiamento / estoque de terceiros).
//
// Empresa por `enterprise_id`, como o resto do domínio operacional: toda consulta
// filtra pela empresa da sessão. O razão é separado de `stock_balances` de
// propósito — material que não é da empresa não pode entrar em valoração, custeio
// nem no líquido do MRP. Ver a migration 000370 para o raciocínio completo.
package customer_material

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// definirAtorDaAuditoria publica quem assina a operação para os triggers de
// auditoria da migração 000371. As três tabelas só têm autor no insert
// (`created_by`); numa alteração o banco não tem como saber quem foi, e é o
// parâmetro de sessão que conta.
//
// Tem de ser dentro de uma transação aberta: `set_config(...,true)` vale até o
// fim da transação corrente, e num Exec solto sobre o pool o valor morre antes do
// UPDATE seguinte — e ainda vazaria para a próxima requisição que pegasse aquela
// conexão se fosse gravado como sessão.
func definirAtorDaAuditoria(ctx context.Context, tx pgx.Tx, usuario string, motivo *string) error {
	texto := ""
	if motivo != nil {
		texto = *motivo
	}
	if _, err := tx.Exec(ctx,
		`SELECT set_config('venture.cm_actor',$1,true), set_config('venture.cm_reason',$2,true)`,
		usuario, texto); err != nil {
		return fmt.Errorf("registrar o autor da auditoria: %w", err)
	}
	return nil
}

// Os enums entram como $n::text::<enum>, nunca $n::<enum>. O pgx pergunta ao
// Postgres o tipo de cada parâmetro; com o cast direto o tipo inferido é o ENUM,
// e o driver não sabe codificar uma string Go num OID de enum que ele não
// conhece — a gravação falha com erro de domínio em vez de funcionar.
const colunasRemessa = `r.id, r.enterprise_id, r.customer_code, r.nfe_number, r.nfe_series, r.nfe_key,
r.cfop, r.issue_date, r.received_at, r.fiscal_return_deadline, r.total_value, r.status::text,
r.sales_order_code, r.blocked, r.block_reason, r.closed_at, r.closed_by::text, r.close_reason,
r.notes, r.created_by::text, r.created_at, r.updated_at`

const colunasItem = `i.id, i.enterprise_id, i.remittance_id, i.line_number, i.customer_item_code,
i.item_code, i.description, i.ncm, i.cst, i.uom, i.qty_invoiced, i.qty_received, i.unit_value,
i.qty_returned, i.qty_leftover, i.qty_scrapped, i.balance_qty, i.divergence_qty,
i.divergence_reason, i.divergence_settled_by::text, i.divergence_settled_at,
i.warehouse_id, i.address, i.created_at, i.updated_at`

const colunasMovimento = `m.id, m.enterprise_id, m.remittance_item_id, m.movement_type::text,
m.quantity, m.unit_value, m.cfop, m.production_order_id, m.fiscal_exit_id,
m.scrap_destination::text, m.reason, m.idempotency_key, m.created_by::text, m.created_at,
m.reversed_at, m.reversed_by::text, m.reversal_reason`

func scanRemessa(row pgx.Row) (*entity.Remessa, error) {
	var r entity.Remessa
	var status string
	if err := row.Scan(&r.ID, &r.EnterpriseID, &r.CustomerCode, &r.NFeNumber, &r.NFeSeries, &r.NFeKey,
		&r.CFOP, &r.IssueDate, &r.ReceivedAt, &r.FiscalReturnDeadline, &r.TotalValue, &status,
		&r.SalesOrderCode, &r.Blocked, &r.BlockReason, &r.ClosedAt, &r.ClosedBy, &r.CloseReason,
		&r.Notes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	r.Status = entity.StatusRemessa(status)
	return &r, nil
}

func scanItem(row pgx.Row) (*entity.ItemRemessa, error) {
	var i entity.ItemRemessa
	if err := row.Scan(&i.ID, &i.EnterpriseID, &i.RemittanceID, &i.LineNumber, &i.CustomerItemCode,
		&i.ItemCode, &i.Description, &i.NCM, &i.CST, &i.UOM, &i.QtyInvoiced, &i.QtyReceived, &i.UnitValue,
		&i.QtyReturned, &i.QtyLeftover, &i.QtyScrapped, &i.BalanceQty, &i.DivergenceQty,
		&i.DivergenceReason, &i.DivergenceSettledBy, &i.DivergenceSettledAt,
		&i.WarehouseID, &i.Address, &i.CreatedAt, &i.UpdatedAt); err != nil {
		return nil, err
	}
	return &i, nil
}

func scanMovimento(row pgx.Row) (*entity.Movimento, error) {
	var m entity.Movimento
	var tipo string
	var destino *string
	if err := row.Scan(&m.ID, &m.EnterpriseID, &m.RemittanceItemID, &tipo,
		&m.Quantity, &m.UnitValue, &m.CFOP, &m.ProductionOrderID, &m.FiscalExitID,
		&destino, &m.Reason, &m.IdempotencyKey, &m.CreatedBy, &m.CreatedAt,
		&m.ReversedAt, &m.ReversedBy, &m.ReversalReason); err != nil {
		return nil, err
	}
	m.MovementType = entity.TipoMovimento(tipo)
	if destino != nil && *destino != "" {
		d := entity.DestinoSucata(*destino)
		m.ScrapDestination = &d
	}
	return &m, nil
}

// Registrar grava a capa e as linhas na mesma transação: uma remessa sem item
// deixaria material físico na empresa sem saldo para controlar.
func (r *Repository) Registrar(ctx context.Context, remessa *entity.Remessa) (*entity.Remessa, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if len(remessa.Itens) == 0 {
		return nil, errorsuc.NewValidationError("a remessa precisa de pelo menos um item")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir transação da remessa: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := definirAtorDaAuditoria(ctx, tx, remessa.CreatedBy, nil); err != nil {
		return nil, err
	}

	var id int64
	err = tx.QueryRow(ctx, `
INSERT INTO public.customer_material_remittances
  (enterprise_id, customer_code, nfe_number, nfe_series, nfe_key, cfop, issue_date, received_at,
   fiscal_return_deadline, total_value, status, sales_order_code, blocked, block_reason, notes, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::text::customer_material_remittance_status_enum,$12,$13,$14,$15,$16::uuid)
RETURNING id`,
		enterpriseID, remessa.CustomerCode, remessa.NFeNumber, remessa.NFeSeries, remessa.NFeKey,
		remessa.CFOP, remessa.IssueDate, remessa.ReceivedAt, remessa.FiscalReturnDeadline,
		remessa.TotalValue, string(remessa.Status), remessa.SalesOrderCode, remessa.Blocked,
		remessa.BlockReason, remessa.Notes, remessa.CreatedBy).Scan(&id)
	if err != nil {
		return nil, traduzirErro(err)
	}

	for _, item := range remessa.Itens {
		var itemID int64
		err := tx.QueryRow(ctx, `
INSERT INTO public.customer_material_items
  (enterprise_id, remittance_id, line_number, customer_item_code, item_code, description, ncm, cst, uom,
   qty_invoiced, qty_received, unit_value, divergence_reason, warehouse_id, address)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
RETURNING id`,
			enterpriseID, id, item.LineNumber, item.CustomerItemCode, item.ItemCode, item.Description,
			item.NCM, item.CST, item.UOM, item.QtyInvoiced, item.QtyReceived, item.UnitValue,
			item.DivergenceReason, item.WarehouseID, item.Address).Scan(&itemID)
		if err != nil {
			return nil, traduzirErro(err)
		}
		item.ID = itemID
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmar a remessa: %w", err)
	}
	return r.BuscarPorID(ctx, id)
}

func (r *Repository) BuscarPorID(ctx context.Context, id int64) (*entity.Remessa, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	remessa, err := scanRemessa(r.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT %s FROM public.customer_material_remittances r
WHERE r.id = $1 AND r.enterprise_id = $2`, colunasRemessa), id, enterpriseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("remessa de beneficiamento não encontrada")
	}
	if err != nil {
		return nil, fmt.Errorf("buscar remessa: %w", err)
	}
	if err := r.carregarItens(ctx, enterpriseID, []*entity.Remessa{remessa}); err != nil {
		return nil, err
	}
	return remessa, nil
}

func (r *Repository) Listar(ctx context.Context, f domrepo.FiltroRemessa) ([]*entity.Remessa, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	condicoes := []string{"r.enterprise_id = $1"}
	args := []any{enterpriseID}
	adicionar := func(cond string, valor any) {
		args = append(args, valor)
		condicoes = append(condicoes, fmt.Sprintf(cond, len(args)))
	}
	if f.CustomerCode != nil {
		adicionar("r.customer_code = $%d", *f.CustomerCode)
	}
	if f.NFeNumber != nil {
		adicionar("r.nfe_number = $%d", *f.NFeNumber)
	}
	if f.SalesOrderCode != nil {
		adicionar("r.sales_order_code = $%d", *f.SalesOrderCode)
	}
	if len(f.Status) > 0 {
		situacoes := make([]string, 0, len(f.Status))
		for _, s := range f.Status {
			situacoes = append(situacoes, string(s))
		}
		adicionar("r.status::text = ANY($%d)", situacoes)
	}
	if f.SomenteBloqueada {
		condicoes = append(condicoes, "r.blocked = TRUE")
	}
	if f.VencendoAte != nil {
		adicionar("r.fiscal_return_deadline <= $%d", *f.VencendoAte)
		// Prazo sem situação informada significa "o que ainda precisa voltar": uma
		// remessa encerrada ou cancelada não tem retorno pendente, e listá-la
		// poluiria a fila de cobrança.
		//
		// Também é o que permite usar o índice: idx_..._prazo é PARCIAL em
		// ABERTA/PARCIAL, e sem esta condição o plano cai em varredura da tabela
		// inteira — medido, 8.040 linhas varridas para devolver 50.
		if len(f.Status) == 0 {
			condicoes = append(condicoes, "r.status IN ('ABERTA','PARCIAL')")
		}
	}
	if busca := strings.TrimSpace(f.Busca); busca != "" {
		args = append(args, "%"+strings.ToLower(busca)+"%")
		condicoes = append(condicoes, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM public.customer_material_items b
			WHERE b.remittance_id = r.id
			  AND (LOWER(b.description) LIKE $%d OR LOWER(b.customer_item_code) LIKE $%d))`, len(args), len(args)))
	}
	// Saldo só existe nas linhas, então o filtro vive num EXISTS: assim uma
	// remessa totalmente devolvida não aparece na fila de quem tem material aqui.
	if f.SomenteComSaldo {
		condicoes = append(condicoes, `EXISTS (
			SELECT 1 FROM public.customer_material_items b
			WHERE b.remittance_id = r.id AND b.balance_qty > 0)`)
	}

	limite := ""
	if f.Limit > 0 {
		args = append(args, f.Limit)
		limite = fmt.Sprintf(" LIMIT $%d", len(args))
		if f.Offset > 0 {
			args = append(args, f.Offset)
			limite += fmt.Sprintf(" OFFSET $%d", len(args))
		}
	}

	// Prazo primeiro: a fila que importa é a de quem está perto de vencer os 30
	// dias, não a de quem chegou por último.
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT %s FROM public.customer_material_remittances r
WHERE %s
ORDER BY r.fiscal_return_deadline, r.nfe_number%s`,
		colunasRemessa, strings.Join(condicoes, " AND "), limite), args...)
	if err != nil {
		return nil, fmt.Errorf("listar remessas: %w", err)
	}
	defer rows.Close()

	var out []*entity.Remessa
	for rows.Next() {
		remessa, err := scanRemessa(rows)
		if err != nil {
			return nil, fmt.Errorf("ler remessa: %w", err)
		}
		out = append(out, remessa)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.carregarItens(ctx, enterpriseID, out); err != nil {
		return nil, err
	}
	return out, nil
}

// carregarItens busca as linhas de todas as remessas numa consulta só: sem isso a
// tela pagaria N+1 requisições para mostrar o saldo de cada remessa.
func (r *Repository) carregarItens(ctx context.Context, enterpriseID int64, remessas []*entity.Remessa) error {
	if len(remessas) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(remessas))
	porID := make(map[int64]*entity.Remessa, len(remessas))
	for _, remessa := range remessas {
		ids = append(ids, remessa.ID)
		porID[remessa.ID] = remessa
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT %s FROM public.customer_material_items i
WHERE i.remittance_id = ANY($1) AND i.enterprise_id = $2
ORDER BY i.remittance_id, i.line_number`, colunasItem), ids, enterpriseID)
	if err != nil {
		return fmt.Errorf("carregar itens da remessa: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return fmt.Errorf("ler item da remessa: %w", err)
		}
		if remessa, ok := porID[item.RemittanceID]; ok {
			remessa.Itens = append(remessa.Itens, item)
		}
	}
	return rows.Err()
}

// SaldoPorItem é a visão que o inventário mostra ao lado do estoque próprio. Soma
// só linhas com saldo, e traz o prazo mais próximo para a conferência saber o que
// precisa voltar primeiro.
func (r *Repository) SaldoPorItem(ctx context.Context, f domrepo.FiltroSaldo) ([]*entity.SaldoPorItem, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	condicoes := []string{"r.enterprise_id = $1", "i.balance_qty > 0"}
	args := []any{enterpriseID}
	adicionar := func(cond string, valor any) {
		args = append(args, valor)
		condicoes = append(condicoes, fmt.Sprintf(cond, len(args)))
	}
	if f.CustomerCode != nil {
		adicionar("r.customer_code = $%d", *f.CustomerCode)
	}
	if codigo := strings.TrimSpace(f.CustomerItemCode); codigo != "" {
		adicionar("i.customer_item_code = $%d", codigo)
	}
	if f.ItemCode != nil {
		adicionar("i.item_code = $%d", *f.ItemCode)
	}
	if busca := strings.TrimSpace(f.Busca); busca != "" {
		args = append(args, "%"+strings.ToLower(busca)+"%")
		condicoes = append(condicoes, fmt.Sprintf(
			"(LOWER(i.description) LIKE $%d OR LOWER(i.customer_item_code) LIKE $%d)", len(args), len(args)))
	}

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT r.customer_code,
       COALESCE(c.name, ''),
       i.customer_item_code,
       MIN(i.item_code),
       MIN(i.description),
       MIN(i.uom),
       SUM(i.balance_qty),
       COUNT(DISTINCT r.id),
       MIN(r.fiscal_return_deadline)
FROM public.customer_material_items i
JOIN public.customer_material_remittances r ON r.id = i.remittance_id
LEFT JOIN public.customers c ON c.code = r.customer_code
WHERE %s
GROUP BY r.customer_code, c.name, i.customer_item_code
ORDER BY MIN(r.fiscal_return_deadline), r.customer_code, i.customer_item_code`,
		strings.Join(condicoes, " AND ")), args...)
	if err != nil {
		return nil, fmt.Errorf("saldo de terceiros por item: %w", err)
	}
	defer rows.Close()

	var out []*entity.SaldoPorItem
	for rows.Next() {
		var s entity.SaldoPorItem
		var prazo *time.Time
		if err := rows.Scan(&s.CustomerCode, &s.CustomerName, &s.CustomerItemCode, &s.ItemCode,
			&s.Description, &s.UOM, &s.Balance, &s.RemessasAbertas, &prazo); err != nil {
			return nil, fmt.Errorf("ler saldo de terceiro: %w", err)
		}
		s.PrazoMaisProximo = prazo
		out = append(out, &s)
	}
	return out, rows.Err()
}

// RegistrarMovimento é o coração do módulo. Tudo numa transação porque um
// movimento gravado sem o consumo da linha faria o saldo mentir — e o saldo é o
// que diz quanto material do cliente ainda está aqui.
//
// A linha é travada com FOR UPDATE: dois faturamentos simultâneos da mesma linha
// poderiam ler o mesmo saldo e devolver mais do que entrou. O CHECK do banco
// pegaria, mas com erro de constraint em vez de mensagem que a pessoa entenda.
func (r *Repository) RegistrarMovimento(ctx context.Context, mov domrepo.NovoMovimento, usuario string) (*entity.Movimento, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(mov.IdempotencyKey) == "" {
		return nil, errorsuc.NewValidationError("informe a chave de idempotência do movimento")
	}
	quantidade, err := decimal.NewFromString(strings.TrimSpace(mov.Quantity))
	if err != nil {
		return nil, errorsuc.NewValidationError("quantidade do movimento inválida")
	}
	if !quantidade.IsPositive() {
		return nil, errorsuc.NewValidationError("a quantidade do movimento deve ser maior que zero")
	}
	valorUnitario := decimal.Zero
	if bruto := strings.TrimSpace(mov.UnitValue); bruto != "" {
		valorUnitario, err = decimal.NewFromString(bruto)
		if err != nil {
			return nil, errorsuc.NewValidationError("valor unitário do movimento inválido")
		}
	}
	if mov.MovementType == entity.MovimentoSucata && mov.ScrapDestination == nil {
		return nil, errorsuc.NewValidationError("informe a destinação da sucata")
	}
	if mov.MovementType == entity.MovimentoAjuste && (mov.Reason == nil || strings.TrimSpace(*mov.Reason) == "") {
		return nil, errorsuc.NewValidationError("informe a justificativa do ajuste")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir transação do movimento: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := definirAtorDaAuditoria(ctx, tx, usuario, mov.Reason); err != nil {
		return nil, err
	}

	// Idempotência primeiro: repetir a requisição devolve o movimento original em
	// vez de baixar o saldo de novo.
	existente, err := scanMovimento(tx.QueryRow(ctx, fmt.Sprintf(`
SELECT %s FROM public.customer_material_movements m
WHERE m.enterprise_id = $1 AND m.idempotency_key = $2`, colunasMovimento), enterpriseID, mov.IdempotencyKey))
	if err == nil {
		if existente.RemittanceItemID != mov.RemittanceItemID || !existente.Quantity.Equal(quantidade) ||
			existente.MovementType != mov.MovementType {
			return nil, errorsuc.NewConflictError("esta chave de idempotência já foi usada com outro conteúdo")
		}
		return existente, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("conferir idempotência: %w", err)
	}

	var saldo decimal.Decimal
	var bloqueada bool
	var statusRemessa string
	var remessaID int64
	err = tx.QueryRow(ctx, `
SELECT i.balance_qty, r.blocked, r.status::text, r.id
FROM public.customer_material_items i
JOIN public.customer_material_remittances r ON r.id = i.remittance_id
WHERE i.id = $1 AND i.enterprise_id = $2
FOR UPDATE OF i`, mov.RemittanceItemID, enterpriseID).Scan(&saldo, &bloqueada, &statusRemessa, &remessaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("item da remessa não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("travar o item da remessa: %w", err)
	}

	if entity.StatusRemessa(statusRemessa) == entity.StatusCancelada {
		return nil, errorsuc.NewConflictError("a remessa está cancelada e não aceita movimento")
	}
	if mov.MovementType.ConsomeSaldo() {
		if bloqueada {
			return nil, errorsuc.NewConflictError("a remessa está bloqueada; regularize antes de movimentar o material")
		}
		if quantidade.GreaterThan(saldo) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"a quantidade %s passa do saldo de %s em poder da empresa", quantidade.String(), saldo.String()))
		}
	}

	var destino *string
	if mov.ScrapDestination != nil {
		d := string(*mov.ScrapDestination)
		destino = &d
	}
	criado, err := scanMovimento(tx.QueryRow(ctx, fmt.Sprintf(`
INSERT INTO public.customer_material_movements
  (enterprise_id, remittance_item_id, movement_type, quantity, unit_value, cfop, production_order_id,
   fiscal_exit_id, scrap_destination, reason, idempotency_key, created_by)
VALUES ($1,$2,$3::text::customer_material_movement_type_enum,$4,$5,$6,$7,$8,
        $9::text::customer_material_scrap_destination_enum,$10,$11,$12::uuid)
RETURNING %s`, strings.ReplaceAll(colunasMovimento, "m.", "")),
		enterpriseID, mov.RemittanceItemID, string(mov.MovementType), quantidade, valorUnitario,
		mov.CFOP, mov.ProductionOrderID, mov.FiscalExitID, destino, mov.Reason,
		mov.IdempotencyKey, usuario))
	if err != nil {
		return nil, traduzirErro(err)
	}

	// O consumo da linha acompanha o tipo do movimento. Ajuste entra como sobra
	// porque também devolve saldo ao cliente, mas com justificativa registrada.
	coluna := map[entity.TipoMovimento]string{
		entity.MovimentoRetorno: "qty_returned",
		entity.MovimentoSobra:   "qty_leftover",
		entity.MovimentoSucata:  "qty_scrapped",
		entity.MovimentoAjuste:  "qty_leftover",
	}[mov.MovementType]
	if coluna != "" {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
UPDATE public.customer_material_items
SET %s = %s + $1, updated_at = NOW()
WHERE id = $2 AND enterprise_id = $3`, coluna, coluna),
			quantidade, mov.RemittanceItemID, enterpriseID); err != nil {
			return nil, traduzirErro(err)
		}
	}

	// A situação da remessa é derivada do saldo das linhas, na mesma transação:
	// status que contradiz o saldo é o defeito mais confuso possível numa
	// conferência fiscal.
	if _, err := tx.Exec(ctx, `
UPDATE public.customer_material_remittances r
SET status = CASE
      WHEN r.status IN ('ENCERRADA','CANCELADA') THEN r.status
      WHEN NOT EXISTS (SELECT 1 FROM public.customer_material_items i
                       WHERE i.remittance_id = r.id AND i.balance_qty > 0)
        THEN 'ENCERRADA'::customer_material_remittance_status_enum
      WHEN EXISTS (SELECT 1 FROM public.customer_material_items i
                   WHERE i.remittance_id = r.id
                     AND (i.qty_returned + i.qty_leftover + i.qty_scrapped) > 0)
        THEN 'PARCIAL'::customer_material_remittance_status_enum
      ELSE 'ABERTA'::customer_material_remittance_status_enum
    END,
    updated_at = NOW()
WHERE r.id = $1 AND r.enterprise_id = $2`, remessaID, enterpriseID); err != nil {
		return nil, fmt.Errorf("atualizar a situação da remessa: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmar o movimento: %w", err)
	}
	return criado, nil
}

func (r *Repository) MovimentosDoItem(ctx context.Context, itemID int64) ([]*entity.Movimento, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT %s FROM public.customer_material_movements m
WHERE m.remittance_item_id = $1 AND m.enterprise_id = $2
ORDER BY m.created_at, m.id`, colunasMovimento), itemID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listar movimentos do item: %w", err)
	}
	defer rows.Close()
	var out []*entity.Movimento
	for rows.Next() {
		m, err := scanMovimento(rows)
		if err != nil {
			return nil, fmt.Errorf("ler movimento: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) Bloquear(ctx context.Context, id int64, motivo string, usuario string) error {
	if strings.TrimSpace(motivo) == "" {
		return errorsuc.NewValidationError("informe o motivo do bloqueio")
	}
	return r.atualizarBloqueio(ctx, id, true, &motivo, usuario)
}

func (r *Repository) Desbloquear(ctx context.Context, id int64, usuario string) error {
	return r.atualizarBloqueio(ctx, id, false, nil, usuario)
}

// A alteração é de uma linha só, mas roda em transação: é dentro dela que o
// autor fica visível para o trigger de auditoria.
func (r *Repository) atualizarBloqueio(ctx context.Context, id int64, bloqueada bool, motivo *string, usuario string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrir transação do bloqueio: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := definirAtorDaAuditoria(ctx, tx, usuario, motivo); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
UPDATE public.customer_material_remittances
SET blocked = $1, block_reason = $2, updated_at = NOW()
WHERE id = $3 AND enterprise_id = $4`, bloqueada, motivo, id, enterpriseID)
	if err != nil {
		return traduzirErro(err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("remessa de beneficiamento não encontrada")
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar o bloqueio: %w", err)
	}
	return nil
}

// Encerrar fecha a remessa. Com saldo remanescente o motivo é obrigatório: as
// regras do cliente pedem aprovação formal, e o que sobrou fica registrado.
func (r *Repository) Encerrar(ctx context.Context, id int64, motivo string, usuario string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	remessa, err := r.BuscarPorID(ctx, id)
	if err != nil {
		return err
	}
	if remessa.Status == entity.StatusEncerrada {
		return errorsuc.NewConflictError("a remessa já está encerrada")
	}
	temSaldo := remessa.SaldoTotal().IsPositive()
	if temSaldo && strings.TrimSpace(motivo) == "" {
		return errorsuc.NewValidationError(fmt.Sprintf(
			"a remessa ainda tem %s em saldo; informe o motivo da aprovação para encerrar", remessa.SaldoTotal().String()))
	}
	var motivoGravado *string
	if strings.TrimSpace(motivo) != "" {
		motivoGravado = &motivo
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrir transação do encerramento: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := definirAtorDaAuditoria(ctx, tx, usuario, motivoGravado); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
UPDATE public.customer_material_remittances
SET status = 'ENCERRADA'::customer_material_remittance_status_enum,
    closed_at = NOW(), closed_by = $1::uuid, close_reason = $2, updated_at = NOW()
WHERE id = $3 AND enterprise_id = $4`, usuario, motivoGravado, id, enterpriseID)
	if err != nil {
		return traduzirErro(err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("remessa de beneficiamento não encontrada")
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar o encerramento: %w", err)
	}
	return nil
}

// traduzirErro transforma violação de restrição em mensagem que a pessoa entenda.
// Sem isso, um CHECK do banco chega à tela como texto em inglês do PostgreSQL.
func traduzirErro(err error) error {
	if err == nil {
		return nil
	}
	texto := err.Error()
	switch {
	case strings.Contains(texto, "customer_material_remittances_nfe_unica"):
		return errorsuc.NewConflictError("esta NF-e de remessa já foi lançada para este cliente")
	case strings.Contains(texto, "customer_material_movements_idempotencia"):
		return errorsuc.NewConflictError("este movimento já foi registrado")
	case strings.Contains(texto, "customer_material_items_saldo_nao_negativo"):
		return errorsuc.NewValidationError("a movimentação passaria do saldo do cliente em poder da empresa")
	case strings.Contains(texto, "customer_material_items_divergencia_com_motivo"):
		return errorsuc.NewValidationError("a quantidade recebida difere da nota; informe o motivo da divergência")
	case strings.Contains(texto, "customer_material_remittances_bloqueio_com_motivo"):
		return errorsuc.NewValidationError("informe o motivo do bloqueio")
	case strings.Contains(texto, "customer_material_remittances_prazo_apos_emissao"):
		return errorsuc.NewValidationError("o prazo de retorno não pode ser anterior à emissão da nota")
	case strings.Contains(texto, "customer_material_movements_sucata_com_destino"):
		return errorsuc.NewValidationError("informe a destinação da sucata")
	case strings.Contains(texto, "customer_material_movements_ajuste_com_motivo"):
		return errorsuc.NewValidationError("informe a justificativa do ajuste")
	case strings.Contains(texto, "customer_material_items_linha_unica"):
		return errorsuc.NewConflictError("há duas linhas com o mesmo número nesta remessa")
	}
	return err
}

// chaveDaNota compõe a chave de idempotência de uma baixa feita por nota. Deriva
// da nota, do item e do tipo: repetir o faturamento da mesma nota não baixa o
// saldo duas vezes, e é isso que torna recuperável uma falha entre criar a nota e
// registrar a baixa.
func chaveDaNota(fiscalExitID, remittanceItemID int64, tipo entity.TipoMovimento) string {
	return fmt.Sprintf("nota-%d-item-%d-%s", fiscalExitID, remittanceItemID, tipo)
}

// RegistrarMovimentosDaNota baixa todo o material da nota numa transação só.
//
// A nota é criada ANTES desta chamada e nasce sem autorização. A ordem é
// deliberada: uma falha aqui deixa a nota em rascunho sem baixa, o que se resolve
// repetindo — enquanto o inverso (nota autorizada na SEFAZ sem o saldo baixado)
// seria material do cliente que saiu fiscalmente e continua aparecendo como
// presente. A autorização confere esta baixa antes de transmitir.
func (r *Repository) RegistrarMovimentosDaNota(
	ctx context.Context, fiscalExitID int64, linhas []domrepo.MovimentoDaNota, usuario string,
) ([]*entity.Movimento, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if fiscalExitID <= 0 {
		return nil, errorsuc.NewValidationError("informe a nota de saída que originou a baixa")
	}
	if len(linhas) == 0 {
		return nil, errorsuc.NewValidationError("a nota de beneficiamento precisa devolver material")
	}

	// Já baixado por esta nota? Devolve o que existe, sem baixar de novo.
	existentes, err := r.MovimentosDaNota(ctx, fiscalExitID)
	if err != nil {
		return nil, err
	}
	if len(existentes) > 0 {
		return existentes, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir transação da baixa da nota: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := definirAtorDaAuditoria(ctx, tx, usuario, nil); err != nil {
		return nil, err
	}

	// Trava todas as linhas envolvidas de uma vez, na ordem do id, para dois
	// faturamentos simultâneos não se cruzarem em ordens opostas e travarem.
	ids := make([]int64, 0, len(linhas))
	vistos := map[int64]bool{}
	for _, l := range linhas {
		if !vistos[l.RemittanceItemID] {
			ids = append(ids, l.RemittanceItemID)
			vistos[l.RemittanceItemID] = true
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	saldos := map[int64]decimal.Decimal{}
	remessas := map[int64]int64{}
	rows, err := tx.Query(ctx, `
SELECT i.id, i.balance_qty, r.id, r.blocked
FROM public.customer_material_items i
JOIN public.customer_material_remittances r ON r.id = i.remittance_id
WHERE i.id = ANY($1) AND i.enterprise_id = $2
ORDER BY i.id
FOR UPDATE OF i`, ids, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("travar os itens da nota: %w", err)
	}
	for rows.Next() {
		var itemID, remessaID int64
		var saldo decimal.Decimal
		var bloqueada bool
		if err := rows.Scan(&itemID, &saldo, &remessaID, &bloqueada); err != nil {
			rows.Close()
			return nil, fmt.Errorf("ler item da nota: %w", err)
		}
		if bloqueada {
			rows.Close()
			return nil, errorsuc.NewConflictError("a remessa está bloqueada; regularize antes de faturar")
		}
		saldos[itemID] = saldo
		remessas[itemID] = remessaID
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		if _, achou := saldos[id]; !achou {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d da remessa não encontrado", id))
		}
	}

	criados := make([]*entity.Movimento, 0, len(linhas))
	consumido := map[int64]decimal.Decimal{}

	for _, linha := range linhas {
		quantidade, err := decimal.NewFromString(strings.TrimSpace(linha.Quantity))
		if err != nil || !quantidade.IsPositive() {
			return nil, errorsuc.NewValidationError("quantidade inválida na baixa da nota")
		}
		valor := decimal.Zero
		if bruto := strings.TrimSpace(linha.UnitValue); bruto != "" {
			if valor, err = decimal.NewFromString(bruto); err != nil {
				return nil, errorsuc.NewValidationError("valor unitário inválido na baixa da nota")
			}
		}

		// O saldo é conferido acumulando as linhas da própria nota: duas devoluções
		// do mesmo item na mesma nota somam contra o mesmo saldo.
		consumido[linha.RemittanceItemID] = consumido[linha.RemittanceItemID].Add(quantidade)
		if consumido[linha.RemittanceItemID].GreaterThan(saldos[linha.RemittanceItemID]) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf(
				"a nota devolve %s do item %d, acima do saldo de %s em poder da empresa",
				consumido[linha.RemittanceItemID].String(), linha.RemittanceItemID,
				saldos[linha.RemittanceItemID].String()))
		}

		var cfop *string
		if c := strings.TrimSpace(linha.CFOP); c != "" {
			cfop = &c
		}
		criado, err := scanMovimento(tx.QueryRow(ctx, fmt.Sprintf(`
INSERT INTO public.customer_material_movements
  (enterprise_id, remittance_item_id, movement_type, quantity, unit_value, cfop,
   fiscal_exit_id, scrap_destination, reason, idempotency_key, created_by)
VALUES ($1,$2,$3::text::customer_material_movement_type_enum,$4,$5,$6,$7,
        NULL,$8,$9,$10::uuid)
RETURNING %s`, strings.ReplaceAll(colunasMovimento, "m.", "")),
			enterpriseID, linha.RemittanceItemID, string(linha.MovementType), quantidade, valor,
			cfop, fiscalExitID, "baixa pelo faturamento do beneficiamento",
			chaveDaNota(fiscalExitID, linha.RemittanceItemID, linha.MovementType), usuario))
		if err != nil {
			return nil, traduzirErro(err)
		}
		criados = append(criados, criado)

		coluna := map[entity.TipoMovimento]string{
			entity.MovimentoRetorno: "qty_returned",
			entity.MovimentoSobra:   "qty_leftover",
			entity.MovimentoSucata:  "qty_scrapped",
		}[linha.MovementType]
		if coluna == "" {
			return nil, errorsuc.NewValidationError(
				"a nota devolve retorno, sobra ou sucata; outro tipo não vai na nota")
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
UPDATE public.customer_material_items
SET %s = %s + $1, updated_at = NOW()
WHERE id = $2 AND enterprise_id = $3`, coluna, coluna),
			quantidade, linha.RemittanceItemID, enterpriseID); err != nil {
			return nil, traduzirErro(err)
		}
	}

	// A situação de cada remessa tocada é recalculada do saldo, na mesma transação.
	for _, remessaID := range remessas {
		if _, err := tx.Exec(ctx, `
UPDATE public.customer_material_remittances r
SET status = CASE
      WHEN r.status IN ('ENCERRADA','CANCELADA') THEN r.status
      WHEN NOT EXISTS (SELECT 1 FROM public.customer_material_items i
                       WHERE i.remittance_id = r.id AND i.balance_qty > 0)
        THEN 'ENCERRADA'::customer_material_remittance_status_enum
      WHEN EXISTS (SELECT 1 FROM public.customer_material_items i
                   WHERE i.remittance_id = r.id
                     AND (i.qty_returned + i.qty_leftover + i.qty_scrapped) > 0)
        THEN 'PARCIAL'::customer_material_remittance_status_enum
      ELSE 'ABERTA'::customer_material_remittance_status_enum
    END,
    updated_at = NOW()
WHERE r.id = $1 AND r.enterprise_id = $2`, remessaID, enterpriseID); err != nil {
			return nil, fmt.Errorf("atualizar a situação da remessa: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmar a baixa da nota: %w", err)
	}
	return criados, nil
}

func (r *Repository) MovimentosDaNota(ctx context.Context, fiscalExitID int64) ([]*entity.Movimento, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	// Movimento estornado não conta: a pergunta que esta consulta responde é "o que
	// esta nota ainda sustenta em baixa de saldo". Quem responde é a conferência
	// antes de autorizar na SEFAZ e a idempotência do faturamento — para as duas,
	// uma baixa já devolvida ao cliente é o mesmo que baixa que não existe.
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT %s FROM public.customer_material_movements m
WHERE m.fiscal_exit_id = $1 AND m.enterprise_id = $2 AND m.reversed_at IS NULL
ORDER BY m.id`, colunasMovimento), fiscalExitID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listar movimentos da nota: %w", err)
	}
	defer rows.Close()
	var out []*entity.Movimento
	for rows.Next() {
		m, err := scanMovimento(rows)
		if err != nil {
			return nil, fmt.Errorf("ler movimento da nota: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// TrilhaDaRemessa lê a auditoria gravada pelos triggers da migração 000371 — da
// remessa, dos seus itens e dos seus movimentos, numa consulta só, do evento mais
// recente para o mais antigo.
//
// O limite existe porque a trilha de uma remessa movimentada cresce sem teto e a
// tela mostra as últimas alterações; quem precisa do histórico inteiro consulta o
// banco. O nome do autor vem por LEFT JOIN: com ator nulo (correção por script) o
// evento continua aparecendo, sem nome.
func (r *Repository) TrilhaDaRemessa(ctx context.Context, remittanceID int64, limite int) ([]*entity.EventoDeAuditoria, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if limite <= 0 || limite > 500 {
		limite = 200
	}
	rows, err := r.pool.Query(ctx, `
SELECT a.id, a.entity_type, a.entity_id, a.remittance_id, a.action, a.changed_fields,
       a.before_state, a.after_state, a.reason, a.actor_id::text, u.name, u.email, a.occurred_at
FROM public.customer_material_audit a
LEFT JOIN public.users u ON u.id = a.actor_id
WHERE a.enterprise_id = $1 AND a.remittance_id = $2
ORDER BY a.occurred_at DESC, a.id DESC
LIMIT $3`, enterpriseID, remittanceID, limite)
	if err != nil {
		return nil, fmt.Errorf("ler a trilha da remessa: %w", err)
	}
	defer rows.Close()

	eventos := make([]*entity.EventoDeAuditoria, 0, limite)
	for rows.Next() {
		var e entity.EventoDeAuditoria
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.RemittanceID, &e.Acao,
			&e.Alterados, &e.Antes, &e.Depois, &e.Motivo, &e.AtorID, &e.AtorNome,
			&e.AtorEmail, &e.OcorridoEm); err != nil {
			return nil, fmt.Errorf("ler evento da trilha: %w", err)
		}
		eventos = append(eventos, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("percorrer a trilha da remessa: %w", err)
	}
	return eventos, nil
}

// EstornarMovimentosDaNota devolve ao saldo do cliente tudo que uma nota baixou.
// É o que o documento 6 da Usimac chama de estornar as movimentações relacionadas
// quando a nota fiscal é cancelada: sem isso o sistema afirma que o material
// voltou ao cliente enquanto ele continua no pátio.
//
// Idempotente: repetir devolve os movimentos já estornados sem mexer no saldo de
// novo. É o que permite refazer o estorno depois de uma falha no meio do
// cancelamento, sem duplicar a devolução.
func (r *Repository) EstornarMovimentosDaNota(
	ctx context.Context, fiscalExitID int64, usuario, motivo string,
) ([]*entity.Movimento, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(usuario) == "" {
		return nil, errorsuc.NewValidationError("não foi possível identificar quem está estornando")
	}
	if strings.TrimSpace(motivo) == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do estorno")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir transação do estorno: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := definirAtorDaAuditoria(ctx, tx, usuario, &motivo); err != nil {
		return nil, err
	}

	// Trava os movimentos vivos da nota e, com eles, as linhas que serão
	// devolvidas. Ordem pelo id do item, como no faturamento: dois estornos
	// simultâneos que travassem em ordens opostas ficariam presos um no outro.
	rows, err := tx.Query(ctx, `
SELECT m.id, m.remittance_item_id, m.movement_type::text, m.quantity, i.remittance_id
FROM public.customer_material_movements m
JOIN public.customer_material_items i ON i.id = m.remittance_item_id
WHERE m.fiscal_exit_id = $1 AND m.enterprise_id = $2 AND m.reversed_at IS NULL
ORDER BY m.remittance_item_id, m.id
FOR UPDATE OF m, i`, fiscalExitID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("travar os movimentos da nota: %w", err)
	}
	type baixa struct {
		movimentoID int64
		itemID      int64
		tipo        string
		quantidade  decimal.Decimal
		remessaID   int64
	}
	var baixas []baixa
	for rows.Next() {
		var b baixa
		if err := rows.Scan(&b.movimentoID, &b.itemID, &b.tipo, &b.quantidade, &b.remessaID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("ler movimento a estornar: %w", err)
		}
		baixas = append(baixas, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("percorrer os movimentos da nota: %w", err)
	}

	// Nada vivo para estornar: ou a nota nunca baixou saldo, ou o estorno já
	// aconteceu. Nos dois casos o resultado é o estado atual, não um erro — é isso
	// que torna a operação repetível.
	if len(baixas) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("confirmar o estorno: %w", err)
		}
		return r.movimentosEstornadosDaNota(ctx, fiscalExitID)
	}

	remessas := map[int64]bool{}
	for _, b := range baixas {
		coluna := map[string]string{
			string(entity.MovimentoRetorno): "qty_returned",
			string(entity.MovimentoSobra):   "qty_leftover",
			string(entity.MovimentoSucata):  "qty_scrapped",
			string(entity.MovimentoAjuste):  "qty_leftover",
		}[b.tipo]
		if coluna == "" {
			return nil, errorsuc.NewValidationError(
				fmt.Sprintf("movimento %d é de tipo %s e não se sabe qual consumo devolver", b.movimentoID, b.tipo))
		}
		// Sem clamp de propósito. Consumo ficando negativo significa que esta nota foi
		// estornada duas vezes, e o CHECK `qty_returned >= 0` da migração 000370
		// aborta a transação — recusar alto é melhor que absorver em silêncio, que
		// foi justamente o que escondeu a falta de idempotência num teste.
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
UPDATE public.customer_material_items
SET %s = %s - $1, updated_at = NOW()
WHERE id = $2 AND enterprise_id = $3`, coluna, coluna),
			b.quantidade, b.itemID, enterpriseID); err != nil {
			return nil, traduzirErro(err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE public.customer_material_movements
SET reversed_at = NOW(), reversed_by = $1::uuid, reversal_reason = $2
WHERE id = $3 AND enterprise_id = $4`, usuario, motivo, b.movimentoID, enterpriseID); err != nil {
			return nil, traduzirErro(err)
		}
		remessas[b.remessaID] = true
	}

	for remessaID := range remessas {
		// Reabre a remessa que havia encerrado sozinha ao zerar o saldo. Encerramento
		// MANUAL (closed_by preenchido) não é reaberto: foi decisão aprovada de
		// pessoa, e desfazê-la em silêncio por causa de uma nota cancelada trocaria
		// um problema de saldo por um de governança.
		if _, err := tx.Exec(ctx, `
UPDATE public.customer_material_remittances r
SET status = CASE
      WHEN r.status = 'CANCELADA' THEN r.status
      WHEN r.status = 'ENCERRADA' AND r.closed_by IS NOT NULL THEN r.status
      WHEN NOT EXISTS (SELECT 1 FROM public.customer_material_items i
                       WHERE i.remittance_id = r.id AND i.balance_qty > 0)
        THEN 'ENCERRADA'::customer_material_remittance_status_enum
      WHEN EXISTS (SELECT 1 FROM public.customer_material_items i
                   WHERE i.remittance_id = r.id
                     AND (i.qty_returned + i.qty_leftover + i.qty_scrapped) > 0)
        THEN 'PARCIAL'::customer_material_remittance_status_enum
      ELSE 'ABERTA'::customer_material_remittance_status_enum
    END,
    updated_at = NOW()
WHERE r.id = $1 AND r.enterprise_id = $2`, remessaID, enterpriseID); err != nil {
			return nil, fmt.Errorf("recalcular a situação da remessa após o estorno: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmar o estorno: %w", err)
	}
	return r.movimentosEstornadosDaNota(ctx, fiscalExitID)
}

// movimentosEstornadosDaNota devolve o que a nota baixou e já foi estornado. É o
// resultado do estorno, e o que a repetição devolve sem mexer no saldo.
func (r *Repository) movimentosEstornadosDaNota(ctx context.Context, fiscalExitID int64) ([]*entity.Movimento, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT %s FROM public.customer_material_movements m
WHERE m.fiscal_exit_id = $1 AND m.enterprise_id = $2 AND m.reversed_at IS NOT NULL
ORDER BY m.id`, colunasMovimento), fiscalExitID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listar movimentos estornados: %w", err)
	}
	defer rows.Close()
	var out []*entity.Movimento
	for rows.Next() {
		m, err := scanMovimento(rows)
		if err != nil {
			return nil, fmt.Errorf("ler movimento estornado: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
