package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/frete"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/journal"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var _ repository.FreightRepository = (*FiscalRepositoryPG)(nil)

// NewFreightRepositoryPG expõe o repositório fiscal pelo contrato do frete.
func NewFreightRepositoryPG(r repository.FiscalRepository) repository.FreightRepository {
	return r.(*FiscalRepositoryPG)
}

const freteCols = `f.id, f.chave_cte, f.numero, f.serie, f.data_emissao, f.cnpj_transportadora, f.nome_transportadora, f.uf_transportadora,
	f.supplier_code, s.name, f.cfop, f.valor_frete, f.base_icms, f.aliq_icms, f.valor_icms, f.credita_icms, f.tipo_rateio,
	f.data_vencimento, f.status, f.conta_pagar_id, f.observacao, f.created_by, f.created_at, f.lancado_em, f.cancelado_em, f.cancel_reason`

func scanFrete(row pgx.Row) (*repository.FreightDocument, error) {
	var f repository.FreightDocument
	err := row.Scan(&f.ID, &f.ChaveCTe, &f.Numero, &f.Serie, &f.DataEmissao, &f.CNPJTransportadora, &f.NomeTransportadora, &f.UFTransportadora,
		&f.SupplierCode, &f.SupplierName, &f.CFOP, &f.ValorFrete, &f.BaseICMS, &f.AliqICMS, &f.ValorICMS, &f.CreditaICMS, &f.TipoRateio,
		&f.DataVencimento, &f.Status, &f.ContaPagarID, &f.Observacao, &f.CreatedBy, &f.CreatedAt, &f.LancadoEm, &f.CanceladoEm, &f.CancelReason)
	return &f, err
}

func (r *FiscalRepositoryPG) CreateFreight(ctx context.Context, f *repository.FreightDocument, entryIDs []int64) (*repository.FreightDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if f.ChaveCTe != nil && *f.ChaveCTe != "" {
		var obtido bool
		if err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("frete:%d:%s", empresa, *f.ChaveCTe)).Scan(&obtido); err != nil {
			return nil, err
		}
		if !obtido {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("o CT-e %s está sendo lançado por outro usuário neste momento", *f.ChaveCTe))
		}
		var existente int64
		err = tx.QueryRow(ctx, `SELECT id FROM fiscal_freight_documents WHERE enterprise_id=$1 AND chave_cte=$2 AND status <> 'CANCELADO'`, empresa, *f.ChaveCTe).Scan(&existente)
		if err == nil {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("este CT-e (chave %s) já foi registrado como frete %d", *f.ChaveCTe, existente))
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO fiscal_freight_documents
			(enterprise_id, chave_cte, numero, serie, data_emissao, cnpj_transportadora, nome_transportadora, uf_transportadora,
			 supplier_code, cfop, valor_frete, base_icms, aliq_icms, valor_icms, credita_icms, tipo_rateio, data_vencimento,
			 observacao, xml_content, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20) RETURNING id`,
		empresa, f.ChaveCTe, f.Numero, f.Serie, f.DataEmissao, f.CNPJTransportadora, truncarStr(f.NomeTransportadora, 200), f.UFTransportadora,
		f.SupplierCode, f.CFOP, f.ValorFrete, f.BaseICMS, f.AliqICMS, f.ValorICMS, f.CreditaICMS, f.TipoRateio, f.DataVencimento,
		f.Observacao, f.XMLContent, f.CreatedBy).Scan(&f.ID)
	if err != nil {
		return nil, fmt.Errorf("gravando o frete: %w", err)
	}
	if err := vincularNotasFrete(ctx, tx, empresa, f.ID, entryIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetFreight(ctx, f.ID)
}

// vincularNotasFrete troca as notas do frete, conferindo que são da empresa.
func vincularNotasFrete(ctx context.Context, tx pgx.Tx, empresa, freightID int64, entryIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM fiscal_freight_document_entries WHERE freight_id=$1 AND enterprise_id=$2`, freightID, empresa); err != nil {
		return err
	}
	if len(entryIDs) == 0 {
		return nil
	}
	tag, err := tx.Exec(ctx,
		`INSERT INTO fiscal_freight_document_entries (freight_id, fiscal_entry_id, enterprise_id)
		 SELECT $1, e.id, $2 FROM fiscal_entries e WHERE e.enterprise_id=$2 AND e.id = ANY($3) AND e.status <> 'CANCELLED'
		 ON CONFLICT DO NOTHING`, freightID, empresa, entryIDs)
	if err != nil {
		return fmt.Errorf("vinculando as notas ao frete: %w", err)
	}
	if int(tag.RowsAffected()) != len(unicosIDs(entryIDs)) {
		return errorsuc.NewValidationError("alguma nota informada não existe nesta empresa ou está cancelada")
	}
	return nil
}

func unicosIDs(ids []int64) map[int64]bool {
	m := map[int64]bool{}
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func (r *FiscalRepositoryPG) UpdateFreight(ctx context.Context, f *repository.FreightDocument, entryIDs []int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM fiscal_freight_documents WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, f.ID, empresa).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("frete %d não encontrado", f.ID))
	}
	if err != nil {
		return err
	}
	if status != repository.FreteStatusPendente {
		return errorsuc.NewValidationError("só o frete pendente pode ser alterado")
	}
	if _, err = tx.Exec(ctx,
		`UPDATE fiscal_freight_documents SET supplier_code=$3, valor_frete=$4, base_icms=$5, aliq_icms=$6, valor_icms=$7, credita_icms=$8,
		     tipo_rateio=$9, data_vencimento=$10, observacao=$11
		  WHERE id=$1 AND enterprise_id=$2`,
		f.ID, empresa, f.SupplierCode, f.ValorFrete, f.BaseICMS, f.AliqICMS, f.ValorICMS, f.CreditaICMS, f.TipoRateio, f.DataVencimento, f.Observacao); err != nil {
		return fmt.Errorf("alterando o frete: %w", err)
	}
	if err := vincularNotasFrete(ctx, tx, empresa, f.ID, entryIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *FiscalRepositoryPG) GetFreight(ctx context.Context, id int64) (*repository.FreightDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	f, err := scanFrete(r.pool.QueryRow(ctx, `SELECT `+freteCols+`
		   FROM fiscal_freight_documents f
		   LEFT JOIN suppliers s ON s.code = f.supplier_code AND s.enterprise_id = f.enterprise_id
		  WHERE f.id=$1 AND f.enterprise_id=$2`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("frete %d não encontrado", id))
	}
	if err != nil {
		return nil, err
	}
	if err := r.carregarNotasDosFretes(ctx, empresa, []*repository.FreightDocument{f}); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT a.fiscal_entry_id, a.fiscal_entry_item_id, a.item_code, a.warehouse_id, a.plano_contas_id, a.centro_custo_id,
		        COALESCE(i.description,''), a.valor, a.valor_estoque, a.valor_despesa, a.stock_movement_id
		   FROM fiscal_freight_allocations a LEFT JOIN fiscal_entry_items i ON i.id = a.fiscal_entry_item_id
		  WHERE a.freight_id=$1 AND a.enterprise_id=$2 ORDER BY a.id`, id, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a repository.FreightAllocation
		if err := rows.Scan(&a.FiscalEntryID, &a.FiscalEntryItemID, &a.ItemCode, &a.WarehouseID, &a.PlanoContasID, &a.CentroCustoID,
			&a.Descricao, &a.Valor, &a.ValorEstoque, &a.ValorDespesa, &a.StockMovementID); err != nil {
			return nil, err
		}
		f.Alocacoes = append(f.Alocacoes, a)
	}
	return f, rows.Err()
}

func (r *FiscalRepositoryPG) ListFreights(ctx context.Context, status string) ([]*repository.FreightDocument, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+freteCols+`
		   FROM fiscal_freight_documents f
		   LEFT JOIN suppliers s ON s.code = f.supplier_code AND s.enterprise_id = f.enterprise_id
		  WHERE f.enterprise_id=$1 AND ($2 = '' OR f.status = $2)
		  ORDER BY f.data_emissao DESC, f.id DESC LIMIT 500`, empresa, strings.ToUpper(strings.TrimSpace(status)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*repository.FreightDocument
	for rows.Next() {
		f, err := scanFrete(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err := r.carregarNotasDosFretes(ctx, empresa, out); err != nil {
		return nil, err
	}
	return out, nil
}

// carregarNotasDosFretes preenche as notas transportadas de cada frete numa
// única consulta (a listagem mostra os números das notas).
func (r *FiscalRepositoryPG) carregarNotasDosFretes(ctx context.Context, empresa int64, fretes []*repository.FreightDocument) error {
	if len(fretes) == 0 {
		return nil
	}
	ids := make([]int64, len(fretes))
	porID := make(map[int64]*repository.FreightDocument, len(fretes))
	for i, f := range fretes {
		ids[i] = f.ID
		porID[f.ID] = f
	}
	rows, err := r.pool.Query(ctx,
		`SELECT fe.freight_id, e.id, e.numero_nf, e.serie, e.razao_social_emitente, e.valor_total, e.status, e.chave_acesso
		   FROM fiscal_freight_document_entries fe JOIN fiscal_entries e ON e.id = fe.fiscal_entry_id AND e.enterprise_id = fe.enterprise_id
		  WHERE fe.freight_id = ANY($1) AND fe.enterprise_id=$2 ORDER BY fe.freight_id, e.id`, ids, empresa)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var freightID int64
		var n repository.FreightEntryRef
		if err := rows.Scan(&freightID, &n.FiscalEntryID, &n.NumeroNF, &n.Serie, &n.Emitente, &n.ValorTotal, &n.Status, &n.ChaveAcesso); err != nil {
			return err
		}
		if f := porID[freightID]; f != nil {
			f.Notas = append(f.Notas, n)
		}
	}
	return rows.Err()
}

func (r *FiscalRepositoryPG) EntriesByChaves(ctx context.Context, chaves []string) ([]int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if len(chaves) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id FROM fiscal_entries WHERE enterprise_id=$1 AND chave_acesso = ANY($2) AND is_active AND status <> 'CANCELLED' ORDER BY id`, empresa, chaves)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// itensFreteSQL: os itens das notas, com o peso do cadastro e o saldo atual do
// item no almoxarifado da nota.
const itensFreteSQL = `SELECT i.id, i.fiscal_entry_id, i.item_code, i.warehouse_id, i.plano_contas_id, i.centro_custo_id,
	        COALESCE(i.description,''), i.valor_contabil, COALESCE(i.quantidade_estoque, i.quantity), i.movimenta_estoque,
	        it.engineering_weight->>'gross', COALESCE(it.engineering_weight->>'unit','KG'),
	        COALESCE((SELECT SUM(sb.quantity) FROM stock_balances sb
	                   WHERE sb.enterprise_id=$1 AND sb.item_code=i.item_code AND sb.mask='' AND sb.warehouse_id=i.warehouse_id),0)
	   FROM fiscal_entry_items i
	   JOIN fiscal_entries e ON e.id = i.fiscal_entry_id AND e.enterprise_id = $1
	   LEFT JOIN items it ON it.code = i.item_code AND it.enterprise_id = $1
	  WHERE i.fiscal_entry_id = ANY($2)
	  ORDER BY i.fiscal_entry_id, i.sequence, i.id`

func lerItensFrete(ctx context.Context, q journal.Executor, empresa int64, entryIDs []int64) ([]frete.Item, error) {
	rows, err := q.Query(ctx, itensFreteSQL, empresa, entryIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []frete.Item
	for rows.Next() {
		var it frete.Item
		var peso *string
		var unidade string
		if err := rows.Scan(&it.ItemID, &it.EntryID, &it.ItemCode, &it.WarehouseID, &it.PlanoContasID, &it.CentroCustoID,
			&it.Descricao, &it.ValorContabil, &it.Quantidade, &it.MovimentaEstoque, &peso, &unidade, &it.EmEstoque); err != nil {
			return nil, err
		}
		if peso != nil {
			if p, err := decimal.NewFromString(strings.TrimSpace(*peso)); err == nil && p.IsPositive() {
				kg := frete.PesoEmKg(p, unidade)
				it.PesoUnitario = &kg
			}
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) FreteItens(ctx context.Context, entryIDs []int64) ([]frete.Item, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return lerItensFrete(ctx, r.pool, empresa, entryIDs)
}

// LancarFrete efetiva o frete numa transação: trava o frete e os saldos dos
// itens, rateia sobre o saldo travado (a parte que ainda está no estoque vira
// custo; a consumida, despesa), complementa o custo médio, grava o rateio, o
// título da transportadora com o rateio por plano e a contabilização.
func (r *FiscalRepositoryPG) LancarFrete(ctx context.Context, l repository.LancamentoFrete) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status, tipo string
	var numero int64
	var serie string
	err = tx.QueryRow(ctx, `SELECT status, tipo_rateio, numero, serie FROM fiscal_freight_documents WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`,
		l.FreightID, empresa).Scan(&status, &tipo, &numero, &serie)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("frete %d não encontrado", l.FreightID))
	}
	if err != nil {
		return err
	}
	if status != repository.FreteStatusPendente {
		return errorsuc.NewConflictError(fmt.Sprintf("o frete %d já está %s", l.FreightID, strings.ToLower(status)))
	}
	var entryIDs []int64
	var naoAprovadas []string
	rows, err := tx.Query(ctx,
		`SELECT e.id, e.status, e.numero_nf FROM fiscal_freight_document_entries fe
		   JOIN fiscal_entries e ON e.id = fe.fiscal_entry_id AND e.enterprise_id = fe.enterprise_id
		  WHERE fe.freight_id=$1 AND fe.enterprise_id=$2 FOR SHARE OF e`, l.FreightID, empresa)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, nf int64
		var st string
		if err := rows.Scan(&id, &st, &nf); err != nil {
			rows.Close()
			return err
		}
		entryIDs = append(entryIDs, id)
		if st != "APPROVED" {
			naoAprovadas = append(naoAprovadas, fmt.Sprintf("NF %d", nf))
		}
	}
	rows.Close()
	if len(entryIDs) == 0 {
		return errorsuc.NewValidationError("vincule ao frete as notas de entrada que ele transportou")
	}
	if len(naoAprovadas) > 0 {
		return errorsuc.NewValidationError("aprove antes as notas de entrada do frete (o custo complementa o estoque já recebido): " + strings.Join(naoAprovadas, ", "))
	}
	// Trava os saldos dos itens: o rateio entre estoque e despesa vale para o
	// saldo do momento do lançamento.
	if _, err = tx.Exec(ctx,
		`SELECT sb.id FROM stock_balances sb
		  WHERE sb.enterprise_id=$1 AND sb.mask='' AND EXISTS (
		        SELECT 1 FROM fiscal_entry_items i WHERE i.fiscal_entry_id = ANY($2) AND i.item_code = sb.item_code AND i.warehouse_id = sb.warehouse_id)
		  FOR UPDATE`, empresa, entryIDs); err != nil {
		return err
	}
	itens, err := lerItensFrete(ctx, tx, empresa, entryIDs)
	if err != nil {
		return err
	}
	partes, err := frete.Ratear(l.Custo, tipo, itens)
	if err != nil {
		return errorsuc.NewValidationError(err.Error())
	}
	titulo, lote, err := l.Montar(partes)
	if err != nil {
		return err
	}

	notaCusto := fmt.Sprintf("frete CT-e %d/%s", numero, serie)
	for _, p := range partes {
		var movID *int64
		if p.ValorEstoque.IsPositive() {
			id, aplicado, err := stockrepo.AjustarCustoTx(ctx, tx, empresa, *p.Item.ItemCode, "", *p.Item.WarehouseID, p.ValorEstoque,
				repository.OrigemFrete, l.FreightID, notaCusto, l.UserID)
			if err != nil {
				return err
			}
			if !aplicado.Round(2).Equal(p.ValorEstoque) {
				return fmt.Errorf("o ajuste de custo do item %q aplicou %s de %s", p.Item.Descricao, aplicado, p.ValorEstoque)
			}
			movID = &id
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO fiscal_freight_allocations (enterprise_id, freight_id, fiscal_entry_id, fiscal_entry_item_id, item_code, warehouse_id,
			     plano_contas_id, centro_custo_id, valor, valor_estoque, valor_despesa, stock_movement_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			empresa, l.FreightID, p.Item.EntryID, p.Item.ItemID, p.Item.ItemCode, p.Item.WarehouseID, p.Item.PlanoContasID, p.Item.CentroCustoID,
			p.Valor, p.ValorEstoque, p.ValorDespesa, movID); err != nil {
			return fmt.Errorf("gravando o rateio do frete: %w", err)
		}
	}

	var cpID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO contas_pagar
			(numero_documento, tipo_documento, fornecedor_id, fornecedor_cnpj, freight_document_id,
			 data_lancamento, data_emissao, data_vencimento, valor_bruto, desconto, juros, multa, valor_pago,
			 parcela_numero, parcela_total, plano_contas_id, centro_custo_id, status_aprovacao, status, observacao, is_active, criado_por, enterprise_id)
		 VALUES ($1,'CTE',$2,$3,$4, $5::date,$6,$7,$8,0,0,0,0, 1,1,$9,$10,'PENDENTE','PENDENTE',$11,TRUE,$12,$13) RETURNING id`,
		truncarStr(titulo.NumeroDocumento, 60), titulo.FornecedorCode, nuloSeVazio(titulo.FornecedorCNPJ), l.FreightID,
		titulo.DataEmissao, titulo.DataEmissao, titulo.DataVencimento, titulo.Valor, titulo.PlanoContasID, titulo.CentroCustoID,
		titulo.Observacao, l.UserID, empresa).Scan(&cpID); err != nil {
		return fmt.Errorf("gerando o título da transportadora: %w", err)
	}
	for _, al := range titulo.Rateio {
		if !al.Valor.IsPositive() {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO contas_pagar_rateios (enterprise_id, conta_pagar_id, plano_contas_id, centro_custo_id, valor) VALUES ($1,$2,$3,$4,$5)`,
			empresa, cpID, al.PlanoContasID, al.CentroCustoID, al.Valor); err != nil {
			return fmt.Errorf("gravando o rateio do título do frete: %w", err)
		}
	}
	if lote != nil {
		if err := journal.Gravar(ctx, tx, empresa, *lote); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE fiscal_freight_documents SET status='LANCADO', conta_pagar_id=$3, lancado_em=NOW(), lancado_por=$4 WHERE id=$1 AND enterprise_id=$2`,
		l.FreightID, empresa, cpID, l.UserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CancelarFrete desfaz o frete lançado: recusa se o título já foi pago; cancela
// o título, estorna o complemento de custo (sem levar o custo abaixo de zero) e
// a contabilização. Frete pendente só deixa de valer.
func (r *FiscalRepositoryPG) CancelarFrete(ctx context.Context, c repository.CancelamentoFrete) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var cpID *int64
	err = tx.QueryRow(ctx, `SELECT status, conta_pagar_id FROM fiscal_freight_documents WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`,
		c.FreightID, empresa).Scan(&status, &cpID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("frete %d não encontrado", c.FreightID))
	}
	if err != nil {
		return err
	}
	if status == repository.FreteStatusCancelado {
		return errorsuc.NewConflictError("o frete já está cancelado")
	}
	if status == repository.FreteStatusLancado {
		if cpID != nil {
			var cpStatus string
			var pago, abatido decimal.Decimal
			if err := tx.QueryRow(ctx, `SELECT status, COALESCE(valor_pago,0), COALESCE(valor_adiantamento_abatido,0) FROM contas_pagar WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`,
				*cpID, empresa).Scan(&cpStatus, &pago, &abatido); err != nil {
				return err
			}
			if cpStatus == "PAGO" || pago.IsPositive() || abatido.IsPositive() {
				return errorsuc.NewValidationError("o título da transportadora já foi pago (ou abatido): estorne o pagamento antes de cancelar o frete")
			}
			if _, err := tx.Exec(ctx, `UPDATE contas_pagar SET status='CANCELADO', is_active=FALSE, updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, *cpID, empresa); err != nil {
				return err
			}
		}
		rows, err := tx.Query(ctx, `SELECT item_code, warehouse_id, valor_estoque FROM fiscal_freight_allocations
		                            WHERE freight_id=$1 AND enterprise_id=$2 AND valor_estoque > 0 ORDER BY id`, c.FreightID, empresa)
		if err != nil {
			return err
		}
		type ajuste struct {
			item, wh int64
			valor    decimal.Decimal
		}
		var ajustes []ajuste
		for rows.Next() {
			var a ajuste
			if err := rows.Scan(&a.item, &a.wh, &a.valor); err != nil {
				rows.Close()
				return err
			}
			ajustes = append(ajustes, a)
		}
		rows.Close()
		nota := "estorno do frete: " + c.Motivo
		for _, a := range ajustes {
			if _, _, err := stockrepo.AjustarCustoTx(ctx, tx, empresa, a.item, "", a.wh, a.valor.Neg(), repository.OrigemFreteEstorno, c.FreightID, nota, c.UserID); err != nil {
				return err
			}
		}
		if _, err := journal.Estornar(ctx, tx, empresa, repository.OrigemFrete, c.FreightID, repository.OrigemFreteEstorno, "FRE", c.DataEstorno); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE fiscal_freight_documents SET status='CANCELADO', cancelado_em=NOW(), cancel_reason=$3 WHERE id=$1 AND enterprise_id=$2`,
		c.FreightID, empresa, c.Motivo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *FiscalRepositoryPG) FreteDoTitulo(ctx context.Context, contaPagarID int64) (*int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var id *int64
	err = r.pool.QueryRow(ctx, `SELECT freight_document_id FROM contas_pagar WHERE id=$1 AND enterprise_id=$2`, contaPagarID, empresa).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return id, err
}
