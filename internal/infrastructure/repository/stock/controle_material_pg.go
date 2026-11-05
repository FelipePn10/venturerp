package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// loteSemSaidaSQL: o lote da linha b (stock_lot_balances) não pode sair — está
// numa situação sem saída ou vencido na empresa que bloqueia vencido.
const loteSemSaidaSQL = `EXISTS (SELECT 1 FROM public.stock_lots rl
	 LEFT JOIN public.stock_lot_statuses rs ON rs.enterprise_id = rl.enterprise_id AND rs.code = rl.status_code
	 LEFT JOIN public.stock_parameters rp ON rp.enterprise_id = rl.enterprise_id
	WHERE rl.enterprise_id = b.enterprise_id AND rl.item_code = b.item_code AND rl.mask = b.mask AND rl.lot = b.lot
	  AND ((rs.code IS NOT NULL AND NOT rs.allows_issue)
	       OR (rl.expires_at < CURRENT_DATE AND COALESCE(rp.expired_lot_action, 'BLOQUEAR') = 'BLOQUEAR')))`

// loteForaDoDisponivelSQL: o lote da linha b não conta para promessa (ATP).
const loteForaDoDisponivelSQL = `EXISTS (SELECT 1 FROM public.stock_lots rl
	 LEFT JOIN public.stock_lot_statuses rs ON rs.enterprise_id = rl.enterprise_id AND rs.code = rl.status_code
	 LEFT JOIN public.stock_parameters rp ON rp.enterprise_id = rl.enterprise_id
	WHERE rl.enterprise_id = b.enterprise_id AND rl.item_code = b.item_code AND rl.mask = b.mask AND rl.lot = b.lot
	  AND ((rs.code IS NOT NULL AND NOT rs.counts_available)
	       OR (rl.expires_at < CURRENT_DATE AND COALESCE(rp.expired_lot_action, 'BLOQUEAR') = 'BLOQUEAR')))`

// LoteForaDoDisponivelSQL é a mesma regra para quem planeja: lote que a situação
// tira do disponível ou que venceu não abastece a necessidade.
const LoteForaDoDisponivelSQL = loteForaDoDisponivelSQL

// saidaLiberada: movimentos que tiram material sem olhar a situação do lote. O
// inventário e o estorno acertam o que já aconteceu, a decisão de uso é o que
// resolve o lote, desfazer uma entrada devolve o que entrou, e o ajuste é o
// caminho do descarte.
func saidaLiberada(m *entity.StockMovement) bool {
	if m.SaidaPeloPreco || m.PermiteSaldoNegativo || m.MovementType == entity.MovementTypeAdjustment {
		return true
	}
	if m.ReferenceType == nil {
		return false
	}
	switch *m.ReferenceType {
	case entity.ReferenceTypeInventory, entity.ReferenceTypeCycleCount, entity.ReferenceTypeReversal,
		entity.ReferenceTypeNFEntry, entity.ReferenceTypePurchaseOrder, "QUALITY_LOT":
		return true
	}
	return false
}

// conferirRastreioDoItem exige o lote e o endereço de quem digita o movimento
// de um item que controla. Item sem o controle ligado segue como antes.
func conferirRastreioDoItem(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if !m.ExigirRastreio {
		return nil
	}
	var lote, endereco bool
	err := tx.QueryRow(ctx,
		`SELECT controls_lot, controls_address FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`,
		enterpriseID, m.ItemCode).Scan(&lote, &endereco)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lendo o controle do item: %w", err)
	}
	if lote && !loteInformado(m) {
		return errorsuc.NewValidationError("este item controla lote: informe o lote. Se o saldo entrou sem lote, implante o lote em Estoque › Lotes")
	}
	if !endereco || enderecoDoSaldo(m) != "" {
		return nil
	}
	var usa bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1 AND warehouse_id=$2)`,
		enterpriseID, m.WarehouseID).Scan(&usa); err != nil {
		return fmt.Errorf("conferindo os endereços do almoxarifado: %w", err)
	}
	if usa {
		return errorsuc.NewValidationError("este item controla endereço: informe o endereço")
	}
	return nil
}

// conferirLoteLiberado recusa a saída de lote numa situação que não permite ou
// vencido na empresa que bloqueia. Transferir o lote só depende da situação:
// levar o vencido para a quarentena é o que se quer que aconteça.
func conferirLoteLiberado(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if !loteInformado(m) || saidaLiberada(m) || !signedDecimalQuantity(m.MovementType, movementQuantity(m)).IsNegative() {
		return nil
	}
	lote := strings.TrimSpace(*m.Lot)
	var descricao *string
	var saida, transfere, vencido, bloqueiaVencido bool
	err := tx.QueryRow(ctx,
		`SELECT s.description, COALESCE(s.allows_issue, TRUE), COALESCE(s.allows_transfer, TRUE),
		        COALESCE(l.expires_at < CURRENT_DATE, FALSE), COALESCE(p.expired_lot_action, 'BLOQUEAR') = 'BLOQUEAR'
		   FROM public.stock_lots l
		   LEFT JOIN public.stock_lot_statuses s ON s.enterprise_id=l.enterprise_id AND s.code=l.status_code
		   LEFT JOIN public.stock_parameters p ON p.enterprise_id=l.enterprise_id
		  WHERE l.enterprise_id=$1 AND l.item_code=$2 AND l.mask=$3 AND l.lot=$4`,
		enterpriseID, m.ItemCode, m.Mask, lote).Scan(&descricao, &saida, &transfere, &vencido, &bloqueiaVencido)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("conferindo a situação do lote: %w", err)
	}
	situacao := ""
	if descricao != nil {
		situacao = *descricao
	}
	if m.MovementType == entity.MovementTypeTransferOut {
		if !transfere {
			return errorsuc.NewValidationError(fmt.Sprintf("o lote %s está como %s, que não permite transferência", lote, situacao))
		}
		return nil
	}
	if !saida {
		return errorsuc.NewValidationError(fmt.Sprintf("o lote %s está como %s, que não permite saída: troque a situação do lote ou use outro", lote, situacao))
	}
	if vencido && bloqueiaVencido {
		return errorsuc.NewValidationError(fmt.Sprintf("o lote %s está vencido: use outro lote, ou faça a baixa por ajuste para descartá-lo", lote))
	}
	return nil
}

// quantidadesRetidas: quanto de cada endereço está em lote que não pode sair.
func quantidadesRetidas(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) (map[string]decimal.Decimal, error) {
	rows, err := tx.Query(ctx,
		`SELECT b.address, SUM(b.quantity) FROM public.stock_lot_balances b
		  WHERE b.enterprise_id=$1 AND b.item_code=$2 AND b.mask=$3 AND b.warehouse_id=$4 AND b.quantity > 0 AND `+loteSemSaidaSQL+`
		  GROUP BY b.address`, enterpriseID, m.ItemCode, m.Mask, m.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("lendo os lotes retidos: %w", err)
	}
	defer rows.Close()
	out := map[string]decimal.Decimal{}
	for rows.Next() {
		var e string
		var q decimal.Decimal
		if err := rows.Scan(&e, &q); err != nil {
			return nil, fmt.Errorf("lendo os lotes retidos: %w", err)
		}
		out[e] = q
	}
	return out, rows.Err()
}

// registrarLoteDaEntrada garante o cadastro do lote que entrou, com a validade do
// movimento. Sem isto a validade digitada no recebimento ficava só no movimento:
// o FEFO e os alertas leem o cadastro e tratavam o lote como sem validade.
func registrarLoteDaEntrada(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement, lote string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO public.stock_lots (item_code, mask, lot, received_at, expires_at, created_by, enterprise_id)
		 VALUES ($1,$2,$3,CURRENT_DATE,$4,$5,$6)
		 ON CONFLICT (enterprise_id, item_code, mask, lot) DO UPDATE
		 SET expires_at = COALESCE(public.stock_lots.expires_at, EXCLUDED.expires_at),
		     received_at = COALESCE(public.stock_lots.received_at, EXCLUDED.received_at)`,
		m.ItemCode, m.Mask, lote, m.ExpirationDate, m.CreatedBy, enterpriseID)
	if err != nil {
		return fmt.Errorf("registrando o lote da entrada: %w", err)
	}
	return nil
}

// ListarSituacoesLote lista o cadastro de situações da empresa.
func (r *StockRepositorySQLC) ListarSituacoesLote(ctx context.Context) ([]entity.SituacaoLote, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT code, description, allows_issue, allows_transfer, counts_available, is_active
		   FROM stock_lot_statuses WHERE enterprise_id=$1 ORDER BY code`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando as situações de lote: %w", err)
	}
	defer rows.Close()
	out := []entity.SituacaoLote{}
	for rows.Next() {
		var s entity.SituacaoLote
		if err := rows.Scan(&s.Code, &s.Description, &s.AllowsIssue, &s.AllowsTransfer, &s.CountsAvailable, &s.IsActive); err != nil {
			return nil, fmt.Errorf("lendo a situação de lote: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SalvarSituacaoLote cria ou altera uma situação.
func (r *StockRepositorySQLC) SalvarSituacaoLote(ctx context.Context, s entity.SituacaoLote) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO stock_lot_statuses(enterprise_id,code,description,allows_issue,allows_transfer,counts_available,is_active)
		 VALUES($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (enterprise_id,code) DO UPDATE SET description=EXCLUDED.description, allows_issue=EXCLUDED.allows_issue,
		     allows_transfer=EXCLUDED.allows_transfer, counts_available=EXCLUDED.counts_available, is_active=EXCLUDED.is_active, updated_at=NOW()`,
		empresa, s.Code, s.Description, s.AllowsIssue, s.AllowsTransfer, s.CountsAvailable, s.IsActive)
	if err != nil {
		return fmt.Errorf("gravando a situação de lote: %w", err)
	}
	return nil
}

// ParametrosDeEstoque lê as regras da empresa; sem registro vale o padrão.
func (r *StockRepositorySQLC) ParametrosDeEstoque(ctx context.Context) (entity.ParametrosEstoque, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return entity.ParametrosEstoque{}, err
	}
	return lerParametrosDeEstoque(ctx, r.pool, empresa)
}

// lerParametrosDeEstoque aceita o pool ou a transação de quem chamou.
func lerParametrosDeEstoque(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, empresa int64) (entity.ParametrosEstoque, error) {
	p := entity.ParametrosEstoque{ExpiredLotAction: entity.LoteVencidoBloqueia}
	err := q.QueryRow(ctx,
		`SELECT expired_lot_action, retroactive_months, tolerance_a, tolerance_b, tolerance_c, obsolete_days, obsolete_provision_pct
		   FROM stock_parameters WHERE enterprise_id=$1`, empresa).
		Scan(&p.ExpiredLotAction, &p.RetroactiveMonths, &p.ToleranceA, &p.ToleranceB, &p.ToleranceC, &p.ObsoleteDays, &p.ObsoleteProvisionPct)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return p, fmt.Errorf("lendo os parâmetros de estoque: %w", err)
	}
	return p, nil
}

// ListarMotivosDivergencia lê o cadastro de motivos da empresa.
func (r *StockRepositorySQLC) ListarMotivosDivergencia(ctx context.Context) ([]entity.MotivoDivergencia, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT code, description, is_active FROM stock_discrepancy_reasons WHERE enterprise_id=$1 ORDER BY code`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os motivos: %w", err)
	}
	defer rows.Close()
	out := []entity.MotivoDivergencia{}
	for rows.Next() {
		var m entity.MotivoDivergencia
		if err := rows.Scan(&m.Code, &m.Description, &m.IsActive); err != nil {
			return nil, fmt.Errorf("lendo o motivo: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SalvarMotivoDivergencia cria ou altera um motivo.
func (r *StockRepositorySQLC) SalvarMotivoDivergencia(ctx context.Context, m entity.MotivoDivergencia) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO stock_discrepancy_reasons(enterprise_id,code,description,is_active) VALUES($1,$2,$3,$4)
		 ON CONFLICT (enterprise_id,code) DO UPDATE SET description=EXCLUDED.description, is_active=EXCLUDED.is_active, updated_at=NOW()`,
		empresa, m.Code, m.Description, m.IsActive)
	if err != nil {
		return fmt.Errorf("gravando o motivo: %w", err)
	}
	return nil
}

// SalvarParametrosDeEstoque grava as regras da empresa.
func (r *StockRepositorySQLC) SalvarParametrosDeEstoque(ctx context.Context, p entity.ParametrosEstoque, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO stock_parameters(enterprise_id,expired_lot_action,updated_by,retroactive_months,tolerance_a,tolerance_b,tolerance_c,obsolete_days,obsolete_provision_pct)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (enterprise_id) DO UPDATE SET expired_lot_action=EXCLUDED.expired_lot_action, updated_by=EXCLUDED.updated_by,
		     retroactive_months=EXCLUDED.retroactive_months, tolerance_a=EXCLUDED.tolerance_a, tolerance_b=EXCLUDED.tolerance_b, tolerance_c=EXCLUDED.tolerance_c,
		     obsolete_days=EXCLUDED.obsolete_days, obsolete_provision_pct=EXCLUDED.obsolete_provision_pct, updated_at=NOW()`,
		empresa, p.ExpiredLotAction, ator, p.RetroactiveMonths, p.ToleranceA, p.ToleranceB, p.ToleranceC, p.ObsoleteDays, p.ObsoleteProvisionPct)
	if err != nil {
		return fmt.Errorf("gravando os parâmetros de estoque: %w", err)
	}
	return nil
}

// ControleDoItem lê como o estoque do item é controlado.
func (r *StockRepositorySQLC) ControleDoItem(ctx context.Context, itemCode int64) (entity.ControleDoItem, error) {
	c := entity.ControleDoItem{ItemCode: itemCode, LotIssueRule: entity.BaixaFEFO}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return c, err
	}
	err = r.pool.QueryRow(ctx,
		`SELECT controls_lot, controls_address, lot_issue_rule FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`,
		empresa, itemCode).Scan(&c.ControlsLot, &c.ControlsAddress, &c.LotIssueRule)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return c, fmt.Errorf("lendo o controle do item: %w", err)
	}
	return c, nil
}

// SalvarControleDoItem liga ou desliga o controle sem mexer no resto da
// configuração de manufatura do item.
func (r *StockRepositorySQLC) SalvarControleDoItem(ctx context.Context, c entity.ControleDoItem) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var uom string
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(NULLIF(warehouse_unit_of_measurement::text,''),'UN') FROM items WHERE code=$1 AND enterprise_id=$2`, c.ItemCode, empresa).Scan(&uom)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError("item não encontrado nesta empresa")
	}
	if err != nil {
		return fmt.Errorf("lendo o item: %w", err)
	}
	if !c.ControlsLot {
		var temLote bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2 AND quantity > 0)`, empresa, c.ItemCode).Scan(&temLote); err != nil {
			return fmt.Errorf("conferindo o saldo em lote: %w", err)
		}
		var controlava bool
		if err := r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT controls_lot FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2), FALSE)`, empresa, c.ItemCode).Scan(&controlava); err != nil {
			return fmt.Errorf("lendo o controle do item: %w", err)
		}
		if temLote && controlava {
			return errorsuc.NewValidationError("este item tem saldo em lote: baixe os lotes antes de desligar o controle de lote")
		}
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO manufacturing_stock_item_controls(enterprise_id,item_code,stock_uom,controls_lot,controls_address,lot_issue_rule)
		 VALUES($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (enterprise_id,item_code) DO UPDATE SET controls_lot=EXCLUDED.controls_lot, controls_address=EXCLUDED.controls_address,
		     lot_issue_rule=EXCLUDED.lot_issue_rule, updated_at=NOW()`,
		empresa, c.ItemCode, uom, c.ControlsLot, c.ControlsAddress, c.LotIssueRule)
	if err != nil {
		return fmt.Errorf("gravando o controle do item: %w", err)
	}
	return nil
}

// MudarSituacaoDoLote troca a situação e o grau do lote e registra a mudança.
func (r *StockRepositorySQLC) MudarSituacaoDoLote(ctx context.Context, m entity.MudancaSituacaoLote, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrindo a mudança do lote: %w", err)
	}
	defer tx.Rollback(ctx)
	if m.Status != nil {
		var ativa bool
		err := tx.QueryRow(ctx, `SELECT is_active FROM stock_lot_statuses WHERE enterprise_id=$1 AND code=$2`, empresa, *m.Status).Scan(&ativa)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && !ativa) {
			return errorsuc.NewValidationError("situação de lote não cadastrada ou inativa")
		}
		if err != nil {
			return fmt.Errorf("lendo a situação: %w", err)
		}
	}
	var antes *string
	err = tx.QueryRow(ctx, `SELECT status_code FROM stock_lots WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND lot=$4 FOR UPDATE`,
		empresa, m.ItemCode, m.Mask, m.Lot).Scan(&antes)
	if errors.Is(err, pgx.ErrNoRows) {
		// Lote com saldo mas sem cadastro (entrou antes do registro automático).
		var tem bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND lot=$4)`,
			empresa, m.ItemCode, m.Mask, m.Lot).Scan(&tem); err != nil {
			return fmt.Errorf("conferindo o lote: %w", err)
		}
		if !tem {
			return errorsuc.NewNotFoundError(fmt.Sprintf("lote %s não encontrado neste item", m.Lot))
		}
		if _, err := tx.Exec(ctx, `INSERT INTO stock_lots(item_code,mask,lot,created_by,enterprise_id) VALUES($1,$2,$3,$4,$5)`,
			m.ItemCode, m.Mask, m.Lot, ator, empresa); err != nil {
			return fmt.Errorf("cadastrando o lote: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("lendo o lote: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_lots SET status_code=$5, grade=$6 WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND lot=$4`,
		empresa, m.ItemCode, m.Mask, m.Lot, m.Status, m.Grade); err != nil {
		return fmt.Errorf("mudando a situação do lote: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO stock_lot_status_events(enterprise_id,item_code,mask,lot,from_status,to_status,reason,changed_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, empresa, m.ItemCode, m.Mask, m.Lot, antes, m.Status, m.Reason, ator); err != nil {
		return fmt.Errorf("registrando a mudança do lote: %w", err)
	}
	return tx.Commit(ctx)
}

// ImplantarLote dá lote a um saldo que entrou sem lote, no mesmo endereço. A
// quantidade do almoxarifado não muda; o movimento fica no extrato como registro.
func (r *StockRepositorySQLC) ImplantarLote(ctx context.Context, i entity.ImplantacaoLote, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrindo a implantação do lote: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := TravarItensTx(ctx, tx, empresa, i.ItemCode); err != nil {
		return err
	}
	if err := conferirPeriodoFechado(ctx, tx, empresa, nil); err != nil {
		return err
	}
	if err := conferirBloqueioDeInventario(ctx, tx, empresa, i.WarehouseID, i.ItemCode, nil); err != nil {
		return err
	}
	var semLote, custo decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT s.quantity - COALESCE((SELECT SUM(l.quantity) FROM stock_lot_balances l
		          WHERE l.enterprise_id=s.enterprise_id AND l.item_code=s.item_code AND l.mask=s.mask
		            AND l.warehouse_id=s.warehouse_id AND l.address=s.address), 0), s.avg_cost
		   FROM stock_balances s
		  WHERE s.enterprise_id=$1 AND s.item_code=$2 AND s.mask=$3 AND s.warehouse_id=$4 AND s.address=$5 FOR UPDATE OF s`,
		empresa, i.ItemCode, i.Mask, i.WarehouseID, i.Address).Scan(&semLote, &custo)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lendo o saldo sem lote: %w", err)
	}
	if semLote.LessThan(i.Quantity) {
		return errorsuc.NewValidationError(fmt.Sprintf("há %s sem lote neste endereço e a implantação pede %s", entity.QuantidadeBR(semLote), entity.QuantidadeBR(i.Quantity)))
	}
	m := &entity.StockMovement{ItemCode: i.ItemCode, Mask: i.Mask, WarehouseID: i.WarehouseID, CreatedBy: ator}
	if err := registrarLoteDaEntrada(ctx, tx, empresa, m, i.Lot); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO stock_lot_balances(item_code,mask,warehouse_id,lot,quantity,last_cost,last_movement_at,enterprise_id,address)
		 VALUES($1,$2,$3,$4,$5,$6,NOW(),$7,$8)
		 ON CONFLICT (enterprise_id,item_code,mask,warehouse_id,lot,address) WHERE enterprise_id IS NOT NULL DO UPDATE
		 SET quantity = stock_lot_balances.quantity + EXCLUDED.quantity, last_movement_at = NOW(), updated_at = NOW()`,
		i.ItemCode, i.Mask, i.WarehouseID, i.Lot, i.Quantity, custo, empresa, i.Address); err != nil {
		return fmt.Errorf("implantando o lote: %w", err)
	}
	nota := "Implantação do lote " + i.Lot
	if _, err := tx.Exec(ctx,
		`INSERT INTO stock_movements(item_code,mask,warehouse_id,movement_type,quantity,unit_price,total_price,reference_type,lot,notes,created_by,enterprise_id,address,address_to)
		 VALUES($1,$2,$3,$4,$5,$6,0,$7,$8,$9,$10,$11,$12,$12)`,
		i.ItemCode, i.Mask, i.WarehouseID, entity.MovementTypeAddressTransfer, i.Quantity, custo, entity.ReferenceTypeLotImplant, i.Lot, nota, ator, empresa, i.Address); err != nil {
		return fmt.Errorf("registrando a implantação do lote: %w", err)
	}
	return tx.Commit(ctx)
}
