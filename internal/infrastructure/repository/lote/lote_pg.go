// Package lote grava o lote de inspeção da empresa da sessão: abertura com as
// características, resultados, decisão de uso (com o estoque, a amostragem e a
// NC na mesma transação) e cancelamento.
package lote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/amostragem"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	amostragempg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/amostragem"
	stockpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ReferenciaEstoque é o reference_type dos movimentos da decisão de uso — o
// único (com o estorno da entrada e o inventário) que tira material de um
// almoxarifado de inspeção.
const ReferenciaEstoque = "QUALITY_LOT"

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var _ dom.Repository = (*PG)(nil)

const colunasLote = `l.id, l.numero, l.origem, l.status, l.item_code, l.mask, l.lote, l.serie, l.quantidade, l.warehouse_id,
	l.supplier_code, l.purchase_order_code, l.purchase_order_item_code, l.fiscal_entry_code, l.receiving_notice_code,
	l.production_order_id, l.route_operation_id, l.receiving_route_id, l.inspection_plan_id, l.sampling_procedure_id,
	l.chave_amostragem, l.plano, l.pulado, l.certificado, l.nota_fornecedor, l.modelo, l.observacoes, l.motivo_cancelamento,
	l.legacy_kind, l.legacy_id, l.created_at, l.created_by, l.updated_at`

func scanLote(row pgx.Row) (*dom.Lote, error) {
	var l dom.Lote
	var origem, status string
	var plano []byte
	if err := row.Scan(&l.ID, &l.Numero, &origem, &status, &l.ItemCode, &l.Mask, &l.Lote, &l.Serie, &l.Quantidade, &l.WarehouseID,
		&l.SupplierCode, &l.PurchaseOrderCode, &l.PurchaseOrderItemCode, &l.FiscalEntryCode, &l.ReceivingNoticeCode,
		&l.ProductionOrderID, &l.RouteOperationID, &l.ReceivingRouteID, &l.InspectionPlanID, &l.ProcedimentoID,
		&l.ChaveAmostragem, &plano, &l.Pulado, &l.Certificado, &l.NotaFornecedor, &l.Modelo, &l.Observacoes, &l.MotivoCancelamento,
		&l.LegacyKind, &l.LegacyID, &l.CreatedAt, &l.CreatedBy, &l.UpdatedAt); err != nil {
		return nil, err
	}
	l.Origem, l.Status = dom.Origem(origem), dom.Status(status)
	if len(plano) > 0 {
		var p amostragem.Plano
		if err := json.Unmarshal(plano, &p); err != nil {
			return nil, fmt.Errorf("lendo o plano de amostragem do lote: %w", err)
		}
		l.Plano = &p
	}
	l.Caracteristicas, l.Resultados = []dom.Caracteristica{}, []dom.Resultado{}
	return &l, nil
}

func (r *PG) Abrir(ctx context.Context, l *dom.Lote) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Numeração por empresa sem buraco nem duplicata sob concorrência.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('quality_lot:' || $1::bigint, 0))`, empresa); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(numero), 0) + 1 FROM quality_inspection_lots WHERE enterprise_id = $1`, empresa).Scan(&l.Numero); err != nil {
		return fmt.Errorf("numerando o lote de inspeção: %w", err)
	}
	var plano []byte
	if l.Plano != nil {
		if plano, err = json.Marshal(l.Plano); err != nil {
			return err
		}
	}
	if l.Status == "" {
		l.Status = dom.Aberto
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO quality_inspection_lots (enterprise_id, numero, origem, status, item_code, mask, lote, serie, quantidade, warehouse_id,
			supplier_code, purchase_order_code, purchase_order_item_code, fiscal_entry_code, receiving_notice_code,
			production_order_id, route_operation_id, receiving_route_id, inspection_plan_id, sampling_procedure_id,
			chave_amostragem, plano, pulado, certificado, nota_fornecedor, modelo, observacoes, created_by, legacy_kind, legacy_id)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30
		 WHERE EXISTS (SELECT 1 FROM items WHERE code = $5 AND enterprise_id = $1)
		   AND ($10::bigint IS NULL OR EXISTS (SELECT 1 FROM warehouse WHERE id = $10 AND enterprise_id = $1))
		RETURNING id, created_at, updated_at`,
		empresa, l.Numero, string(l.Origem), string(l.Status), l.ItemCode, l.Mask, l.Lote, l.Serie, l.Quantidade, l.WarehouseID,
		l.SupplierCode, l.PurchaseOrderCode, l.PurchaseOrderItemCode, l.FiscalEntryCode, l.ReceivingNoticeCode,
		l.ProductionOrderID, l.RouteOperationID, l.ReceivingRouteID, l.InspectionPlanID, l.ProcedimentoID,
		l.ChaveAmostragem, plano, l.Pulado, l.Certificado, l.NotaFornecedor, l.Modelo, l.Observacoes, l.CreatedBy, l.LegacyKind, l.LegacyID,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewValidationError("o item ou o almoxarifado do lote de inspeção não existe nesta empresa")
	}
	if err != nil {
		return fmt.Errorf("abrindo o lote de inspeção: %w", err)
	}
	for i := range l.Caracteristicas {
		c := &l.Caracteristicas[i]
		if err := tx.QueryRow(ctx, `
			INSERT INTO quality_lot_characteristics (lot_id, enterprise_id, sequencia, nome, tipo, obrigatoria, critica, nominal, minimo, maximo,
				unidade, instrumento, norma, referencia, origem_step_id, origem_characteristic_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id`,
			l.ID, empresa, c.Sequencia, c.Nome, string(c.Tipo), c.Obrigatoria, c.Critica, c.Nominal, c.Minimo, c.Maximo,
			c.Unidade, c.Instrumento, c.Norma, c.Referencia, c.OrigemStep, c.OrigemCarac).Scan(&c.ID); err != nil {
			return fmt.Errorf("gravando a característica %s do lote: %w", c.Nome, err)
		}
	}
	return tx.Commit(ctx)
}

func (r *PG) carregar(ctx context.Context, q pgx.Tx, empresa int64, where string, args ...any) (*dom.Lote, error) {
	var row pgx.Row
	sql := `SELECT ` + colunasLote + ` FROM quality_inspection_lots l WHERE l.enterprise_id = $1 AND ` + where
	all := append([]any{empresa}, args...)
	if q != nil {
		row = q.QueryRow(ctx, sql+` FOR UPDATE`, all...)
	} else {
		row = r.pool.QueryRow(ctx, sql, all...)
	}
	l, err := scanLote(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("lote de inspeção não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o lote de inspeção: %w", err)
	}
	if err := r.detalhes(ctx, q, empresa, l); err != nil {
		return nil, err
	}
	return l, nil
}

type consultor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *PG) detalhes(ctx context.Context, tx pgx.Tx, empresa int64, l *dom.Lote) error {
	var q consultor = r.pool
	if tx != nil {
		q = tx
	}
	rows, err := q.Query(ctx, `SELECT id, sequencia, nome, tipo, obrigatoria, critica, nominal, minimo, maximo, unidade, instrumento, norma,
			referencia, origem_step_id, origem_characteristic_id
		  FROM quality_lot_characteristics WHERE lot_id = $1 AND enterprise_id = $2 ORDER BY sequencia`, l.ID, empresa)
	if err != nil {
		return fmt.Errorf("lendo as características do lote: %w", err)
	}
	for rows.Next() {
		var c dom.Caracteristica
		var tipo string
		if err := rows.Scan(&c.ID, &c.Sequencia, &c.Nome, &tipo, &c.Obrigatoria, &c.Critica, &c.Nominal, &c.Minimo, &c.Maximo,
			&c.Unidade, &c.Instrumento, &c.Norma, &c.Referencia, &c.OrigemStep, &c.OrigemCarac); err != nil {
			rows.Close()
			return err
		}
		c.Tipo = dom.TipoCaracteristica(tipo)
		l.Caracteristicas = append(l.Caracteristicas, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	rows, err = q.Query(ctx, `SELECT id, characteristic_id, amostra_indice, valor, atributo, conforme, observacao, created_at, created_by, instrument_id
		  FROM quality_lot_results WHERE lot_id = $1 AND enterprise_id = $2 ORDER BY characteristic_id, amostra_indice`, l.ID, empresa)
	if err != nil {
		return fmt.Errorf("lendo os resultados do lote: %w", err)
	}
	for rows.Next() {
		var x dom.Resultado
		if err := rows.Scan(&x.ID, &x.CaracteristicaID, &x.AmostraIndice, &x.Valor, &x.Atributo, &x.Conforme, &x.Observacao, &x.CreatedAt, &x.CreatedBy, &x.InstrumentoID); err != nil {
			rows.Close()
			return err
		}
		l.Resultados = append(l.Resultados, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var d dom.Decisao
	err = q.QueryRow(ctx, `SELECT decisao, qtd_aceita, qtd_restrita, qtd_retrabalho, qtd_rejeitada, qtd_sucata, destino_aceito, destino_restrito,
			destino_retrabalho, destino_rejeitado, destino_sucata, tratamento, nao_conformes, aceito_amostragem, afeta_iqf, motivo, decided_at, decided_by
		  FROM quality_lot_decisions WHERE lot_id = $1 AND enterprise_id = $2`, l.ID, empresa).
		Scan(&d.Tipo, &d.Aceita, &d.Restrita, &d.Retrabalho, &d.Rejeitada, &d.Sucata, &d.DestinoAceito, &d.DestinoRestrito,
			&d.DestinoRetrabalho, &d.DestinoRejeitado, &d.DestinoSucata, &d.Tratamento, &d.NaoConformes, &d.AceitoAmostragem, &d.AfetaIQF,
			&d.Motivo, &d.DecididoEm, &d.DecididoPor)
	if err == nil {
		l.Decisao = &d
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lendo a decisão do lote: %w", err)
	}
	return nil
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.Lote, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return r.carregar(ctx, nil, empresa, `l.id = $2`, id)
}

func (r *PG) ObterPorLegado(ctx context.Context, kind string, id int64) (*dom.Lote, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return r.carregar(ctx, nil, empresa, `l.legacy_kind = $2 AND l.legacy_id = $3`, kind, id)
}

func (r *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.Lote, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if f.Limite <= 0 || f.Limite > 500 {
		f.Limite = 200
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasLote+` FROM quality_inspection_lots l
		 WHERE l.enterprise_id = $1
		   AND ($2 = '' OR l.status = $2) AND ($3 = '' OR l.origem = $3)
		   AND ($4 = 0 OR l.item_code = $4) AND ($5 = 0 OR l.supplier_code = $5) AND ($6 = 0 OR l.production_order_id = $6)
		 ORDER BY l.status = 'ABERTO' DESC, l.created_at DESC, l.id DESC LIMIT $7`,
		empresa, strings.ToUpper(f.Status), strings.ToUpper(f.Origem), f.ItemCode, f.SupplierCode, f.ProductionOrderID, f.Limite)
	if err != nil {
		return nil, fmt.Errorf("listando os lotes de inspeção: %w", err)
	}
	var out []dom.Lote
	for rows.Next() {
		l, err := scanLote(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, *l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Decisão de cada lote decidido (a lista mostra o resultado).
	for i := range out {
		if out[i].Status != dom.Decidido {
			continue
		}
		var d dom.Decisao
		if err := r.pool.QueryRow(ctx, `SELECT decisao, qtd_aceita, qtd_restrita, qtd_retrabalho, qtd_rejeitada, qtd_sucata, aceito_amostragem, decided_at
			  FROM quality_lot_decisions WHERE lot_id = $1 AND enterprise_id = $2`, out[i].ID, empresa).
			Scan(&d.Tipo, &d.Aceita, &d.Restrita, &d.Retrabalho, &d.Rejeitada, &d.Sucata, &d.AceitoAmostragem, &d.DecididoEm); err == nil {
			out[i].Decisao = &d
		}
	}
	if out == nil {
		out = []dom.Lote{}
	}
	return out, nil
}

func (r *PG) GravarResultados(ctx context.Context, loteID int64, rs []dom.Resultado, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM quality_inspection_lots WHERE id = $1 AND enterprise_id = $2 FOR UPDATE`, loteID, empresa).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError("lote de inspeção não encontrado")
	}
	if err != nil {
		return err
	}
	if status != string(dom.Aberto) {
		return errorsuc.NewValidationError("o lote não está aberto: os resultados não podem mais ser alterados")
	}
	var por *uuid.UUID
	if ator != uuid.Nil {
		por = &ator
	}
	for _, x := range rs {
		tag, err := tx.Exec(ctx, `
			INSERT INTO quality_lot_results (lot_id, enterprise_id, characteristic_id, amostra_indice, valor, atributo, conforme, observacao, created_by, instrument_id)
			SELECT $1, $2, c.id, $4, $5, $6, $7, $8, $9, $10 FROM quality_lot_characteristics c
			 WHERE c.id = $3 AND c.lot_id = $1 AND c.enterprise_id = $2
			ON CONFLICT (characteristic_id, amostra_indice) DO UPDATE
			   SET valor = EXCLUDED.valor, atributo = EXCLUDED.atributo, conforme = EXCLUDED.conforme, observacao = EXCLUDED.observacao,
			       created_at = NOW(), created_by = EXCLUDED.created_by, instrument_id = EXCLUDED.instrument_id`,
			loteID, empresa, x.CaracteristicaID, x.AmostraIndice, x.Valor, x.Atributo, x.Conforme, x.Observacao, por, x.InstrumentoID)
		if err != nil {
			return fmt.Errorf("gravando o resultado do lote: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return errorsuc.NewValidationError(fmt.Sprintf("a característica %d não pertence a este lote", x.CaracteristicaID))
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE quality_inspection_lots SET updated_at = NOW() WHERE id = $1 AND enterprise_id = $2`, loteID, empresa); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) Almoxarifados(ctx context.Context, ids []int64) (map[int64]dom.Almoxarifado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id, location::text, disposition FROM warehouse WHERE enterprise_id = $1 AND id = ANY($2)`, empresa, ids)
	if err != nil {
		return nil, fmt.Errorf("lendo os almoxarifados: %w", err)
	}
	defer rows.Close()
	out := map[int64]dom.Almoxarifado{}
	for rows.Next() {
		var a dom.Almoxarifado
		if err := rows.Scan(&a.ID, &a.Localizacao, &a.Disposicao); err != nil {
			return nil, err
		}
		out[a.ID] = a
	}
	return out, rows.Err()
}

func saldoNoAlmox(ctx context.Context, tx pgx.Tx, empresa int64, l *dom.Lote) (decimal.Decimal, error) {
	var saldo decimal.Decimal
	err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(quantity), 0) FROM stock_balances
		 WHERE enterprise_id = $1 AND item_code = $2 AND mask = $3 AND warehouse_id = $4`, empresa, l.ItemCode, l.Mask, *l.WarehouseID).Scan(&saldo)
	return saldo, err
}

func (r *PG) Decidir(ctx context.Context, loteID int64, d *dom.Decisao, ator uuid.UUID, preparar dom.Preparacao) (*dom.Lote, error) {
	l, err := r.decidir(ctx, loteID, d, ator, preparar)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (r *PG) decidir(ctx context.Context, loteID int64, d *dom.Decisao, ator uuid.UUID, preparar dom.Preparacao) (*dom.Lote, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// Trava e relê: dois inspetores decidindo o mesmo lote não movem o
	// estoque duas vezes, e resultado gravado no meio não escapa da validação.
	l, err := r.carregar(ctx, tx, empresa, `l.id = $2`, loteID)
	if err != nil {
		return nil, err
	}
	if l.Status != dom.Aberto {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("o lote %d já está %s", l.Numero, strings.ToLower(string(l.Status))))
	}
	ids := []int64{}
	for _, p := range []*int64{l.WarehouseID, d.DestinoAceito, d.DestinoRestrito, d.DestinoRetrabalho, d.DestinoRejeitado, d.DestinoSucata} {
		if p != nil {
			ids = append(ids, *p)
		}
	}
	almox := map[int64]dom.Almoxarifado{}
	rows, err := tx.Query(ctx, `SELECT id, location::text, disposition FROM warehouse WHERE enterprise_id = $1 AND id = ANY($2)`, empresa, ids)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var a dom.Almoxarifado
		if err := rows.Scan(&a.ID, &a.Localizacao, &a.Disposicao); err != nil {
			rows.Close()
			return nil, err
		}
		almox[a.ID] = a
	}
	rows.Close()
	amostra, nc, err := preparar(l, almox)
	if err != nil {
		return nil, err
	}
	var por *uuid.UUID
	if ator != uuid.Nil {
		por = &ator
	}
	if l.RetemEstoque() {
		saldo, err := saldoNoAlmox(ctx, tx, empresa, l)
		if err != nil {
			return nil, fmt.Errorf("conferindo o saldo do almoxarifado de inspeção: %w", err)
		}
		if saldo.LessThan(l.Quantidade) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o almoxarifado de inspeção tem %s do item e o lote tem %s: confira se a entrada foi estornada", saldo.String(), l.Quantidade.String()))
		}
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO quality_lot_decisions (lot_id, enterprise_id, decisao, qtd_aceita, qtd_restrita, qtd_retrabalho, qtd_rejeitada, qtd_sucata,
			destino_aceito, destino_restrito, destino_retrabalho, destino_rejeitado, destino_sucata, tratamento, nao_conformes,
			aceito_amostragem, afeta_iqf, motivo, decided_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19) RETURNING decided_at`,
		l.ID, empresa, d.Tipo, d.Aceita, d.Restrita, d.Retrabalho, d.Rejeitada, d.Sucata,
		d.DestinoAceito, d.DestinoRestrito, d.DestinoRetrabalho, d.DestinoRejeitado, d.DestinoSucata, d.Tratamento, d.NaoConformes,
		d.AceitoAmostragem, d.AfetaIQF, d.Motivo, por).Scan(&d.DecididoEm); err != nil {
		return nil, fmt.Errorf("gravando a decisão de uso: %w", err)
	}
	d.DecididoPor = por
	if l.RetemEstoque() {
		ref := ReferenciaEstoque
		codigo := l.ID
		for _, p := range d.Pernas() {
			nota := fmt.Sprintf("Lote de inspeção %d — %s", l.Numero, strings.ToLower(p.Parte))
			saida := &stockentity.StockMovement{
				ItemCode: l.ItemCode, Mask: l.Mask, WarehouseID: *l.WarehouseID, MovementType: stockentity.MovementTypeTransferOut,
				Quantity: p.Quantidade, Lot: l.Lote, ReferenceType: &ref, ReferenceCode: &codigo, Notes: &nota, CreatedBy: ator,
			}
			if p.Destino == nil {
				saida.MovementType = stockentity.MovementTypeOut
			}
			if err := stockpg.CreateMovementTx(ctx, tx, empresa, saida); err != nil {
				return nil, fmt.Errorf("movendo a parte %s do lote: %w", strings.ToLower(p.Parte), err)
			}
			if p.Destino == nil {
				continue
			}
			entrada := &stockentity.StockMovement{
				ItemCode: l.ItemCode, Mask: l.Mask, WarehouseID: *p.Destino, MovementType: stockentity.MovementTypeTransferIn,
				Quantity: p.Quantidade, UnitPrice: saida.UnitPrice, TotalPrice: saida.TotalPrice, Lot: l.Lote,
				ReferenceType: &ref, ReferenceCode: &codigo, Notes: &nota, CreatedBy: ator,
			}
			if err := stockpg.CreateMovementTx(ctx, tx, empresa, entrada); err != nil {
				return nil, fmt.Errorf("movendo a parte %s do lote: %w", strings.ToLower(p.Parte), err)
			}
		}
	}
	if amostra != nil {
		ref := amostra.Referencia
		if _, _, err := amostragempg.AtualizarEstadoTx(ctx, tx, empresa, amostra.ProcedimentoID, amostra.ItemCode, amostra.Chave, &ref, ator, amostra.Fn); err != nil {
			return nil, err
		}
	}
	if nc != nil {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('non_conformance_code', 0))`); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO non_conformances (code, lot_id, production_order_id, item_code, lot, nonconform_qty, description, severity, created_by)
			SELECT COALESCE((SELECT MAX(code) FROM non_conformances), 0) + 1, $1, $2, $3, $4, $5::numeric, $6, $7::nc_severity_enum, $8`,
			l.ID, l.ProductionOrderID, l.ItemCode, l.Lote, nc.Quantidade, nc.Descricao, nc.Severidade, ator); err != nil {
			return nil, fmt.Errorf("abrindo a não conformidade do lote: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE quality_inspection_lots SET status = 'DECIDIDO', updated_at = NOW() WHERE id = $1 AND enterprise_id = $2`, l.ID, empresa); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	l.Status, l.Decisao = dom.Decidido, d
	return l, nil
}

func (r *PG) Cancelar(ctx context.Context, id int64, motivo string, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	l, err := r.carregar(ctx, tx, empresa, `l.id = $2`, id)
	if err != nil {
		return err
	}
	if l.Status != dom.Aberto {
		return errorsuc.NewValidationError(fmt.Sprintf("o lote %d já está %s", l.Numero, strings.ToLower(string(l.Status))))
	}
	if l.RetemEstoque() {
		saldo, err := saldoNoAlmox(ctx, tx, empresa, l)
		if err != nil {
			return err
		}
		if saldo.GreaterThanOrEqual(l.Quantidade) {
			return errorsuc.NewValidationError("o material do lote ainda está no almoxarifado de inspeção: tome a decisão de uso (aceitar, rejeitar, devolver) em vez de cancelar. O cancelamento é para lote cuja entrada foi estornada")
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE quality_inspection_lots SET status = 'CANCELADO', motivo_cancelamento = $3, updated_at = NOW()
		 WHERE id = $1 AND enterprise_id = $2`, id, empresa, motivo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) VincularRelatorio(ctx context.Context, loteID, relatorioID int64, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO quality_lot_supplier_reports (lot_id, enterprise_id, quality_report_id, linked_by)
		SELECT l.id, $1, q.id, $4
		  FROM quality_inspection_lots l
		  JOIN item_supplier_quality_reports q ON q.id = $3 AND q.enterprise_id = $1
		  JOIN item_preferred_suppliers s ON s.id = q.item_supplier_id AND s.enterprise_id = $1 AND s.item_code = l.item_code
		   AND (l.supplier_code IS NULL OR s.supplier_code = l.supplier_code)
		 WHERE l.id = $2 AND l.enterprise_id = $1
		ON CONFLICT (lot_id, quality_report_id) DO NOTHING`, empresa, loteID, relatorioID, ator)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23503" {
			return errorsuc.NewValidationError("laudo incompatível com a empresa, o item ou o fornecedor do lote")
		}
		return fmt.Errorf("vinculando o laudo ao lote: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var existe bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM quality_lot_supplier_reports WHERE lot_id = $1 AND quality_report_id = $2 AND enterprise_id = $3)`,
			loteID, relatorioID, empresa).Scan(&existe)
		if !existe {
			return errorsuc.NewNotFoundError("laudo incompatível com a empresa, o item ou o fornecedor do lote")
		}
	}
	return nil
}

func (r *PG) DesvincularRelatorio(ctx context.Context, loteID, relatorioID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM quality_lot_supplier_reports WHERE enterprise_id = $1 AND lot_id = $2 AND quality_report_id = $3`, empresa, loteID, relatorioID)
	if err != nil {
		return fmt.Errorf("desvinculando o laudo do lote: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("vínculo de laudo não encontrado")
	}
	return nil
}

func (r *PG) Relatorios(ctx context.Context, loteID int64) ([]int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT quality_report_id FROM quality_lot_supplier_reports WHERE enterprise_id = $1 AND lot_id = $2 ORDER BY linked_at`, empresa, loteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
