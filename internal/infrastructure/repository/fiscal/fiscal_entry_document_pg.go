package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var _ repository.FiscalEntryDocumentRepository = (*FiscalRepositoryPG)(nil)

// NewFiscalEntryDocumentRepositoryPG expõe o mesmo repositório pelo contrato da
// nota de entrada completa.
func NewFiscalEntryDocumentRepositoryPG(r repository.FiscalRepository) repository.FiscalEntryDocumentRepository {
	return r.(*FiscalRepositoryPG)
}

// ncmDoItemSQL resolve o NCM do item pela classificação fiscal de compra (ou,
// na falta dela, a de venda) da própria empresa.
const ncmDoItemSQL = `COALESCE((SELECT fc.ncm FROM fiscal_classifications fc
        WHERE fc.enterprise_id = it.enterprise_id
          AND fc.code::text = COALESCE(NULLIF(it.accounting_purchase_fiscal_classification_code,''), it.accounting_sale_fiscal_classification_code)
        LIMIT 1), '')`

const entryHeaderCols = `e.id, e.chave_acesso, e.numero_nf, e.serie, e.modelo, e.data_emissao, e.data_entrada,
        e.cnpj_emitente, e.razao_social_emitente, e.ie_emitente, e.uf_emitente,
        e.valor_produtos, e.valor_frete, e.valor_seguro, e.valor_desconto,
        e.valor_ipi, e.valor_icms, e.valor_pis, e.valor_cofins, e.valor_total,
        e.tipo_documento, e.purchase_order_code, e.cte_code, e.status, e.xml_path, e.notes,
        e.is_active, e.created_at, e.updated_at, e.created_by, e.supplier_code,
        e.natureza_operacao, e.cnpj_destinatario, e.protocolo, e.valor_icms_st, e.valor_outras,
        e.modalidade_frete, e.informacoes_complementares, e.approved_at, e.approved_by, e.sem_pagamento,
        e.entry_operation_code, e.base_ibscbs, e.valor_ibs, e.valor_cbs, e.valor_is,
        e.valor_ret_pis, e.valor_ret_cofins, e.valor_ret_csll, e.base_irrf, e.valor_irrf, e.base_ret_prev, e.valor_ret_prev, e.valor_iss_ret,
        e.stock_status, e.cancelled_at, e.cancel_reason,
        (SELECT s.name FROM suppliers s WHERE s.code = e.supplier_code AND s.enterprise_id = e.enterprise_id LIMIT 1)`

func scanEntryHeader(row pgx.Row) (*entity.FiscalEntry, error) {
	var e entity.FiscalEntry
	err := row.Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.Modelo, &e.DataEmissao, &e.DataEntrada,
		&e.CnpjEmitente, &e.RazaoSocialEmitente, &e.IEEmitente, &e.UFEmitente,
		&e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.TipoDocumento, &e.PurchaseOrderCode, &e.CteCode, &e.Status, &e.XmlPath, &e.Notes,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.SupplierCode,
		&e.NaturezaOperacao, &e.CnpjDestinatario, &e.Protocolo, &e.ValorICMSST, &e.ValorOutras,
		&e.ModalidadeFrete, &e.InformacoesComplementares, &e.ApprovedAt, &e.ApprovedBy, &e.SemPagamento,
		&e.EntryOperationCode, &e.BaseIBSCBS, &e.ValorIBS, &e.ValorCBS, &e.ValorIS,
		&e.ValorRetPIS, &e.ValorRetCOFINS, &e.ValorRetCSLL, &e.BaseIRRF, &e.ValorIRRF, &e.BaseRetPrev, &e.ValorRetPrev, &e.ValorISSRet,
		&e.StockStatus, &e.CancelledAt, &e.CancelReason,
		&e.SupplierName)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *FiscalRepositoryPG) FindEntryByChave(ctx context.Context, chave string) (*entity.FiscalEntry, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	chave = strings.TrimSpace(chave)
	if chave == "" {
		return nil, nil
	}
	e, err := scanEntryHeader(r.pool.QueryRow(ctx,
		`SELECT `+entryHeaderCols+` FROM fiscal_entries e
		  WHERE e.enterprise_id = $1 AND e.chave_acesso = $2 AND e.is_active AND e.status <> 'CANCELLED'
		  ORDER BY e.id LIMIT 1`, empresa, chave))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("procurando nota de entrada pela chave: %w", err)
	}
	return e, nil
}

func (r *FiscalRepositoryPG) CreateEntryDocument(ctx context.Context, e *entity.FiscalEntry) (*entity.FiscalEntry, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Duas pessoas importando o mesmo XML ao mesmo tempo: a trava pela chave
	// não espera — quem chega com a importação em andamento recebe conflito na
	// hora. Esperar segurando a conexão esgota o pool quando dezenas importam o
	// mesmo arquivo, e o dono da trava não termina (o mesmo do faturamento).
	if e.ChaveAcesso != nil && *e.ChaveAcesso != "" {
		var obtido bool
		if err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("nfe-entrada:%d:%s", empresa, *e.ChaveAcesso)).Scan(&obtido); err != nil {
			return nil, err
		}
		if !obtido {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("esta NF-e (chave %s) está sendo importada por outro usuário neste momento", *e.ChaveAcesso))
		}
		var existente int64
		err = tx.QueryRow(ctx, `SELECT id FROM fiscal_entries WHERE enterprise_id=$1 AND chave_acesso=$2 AND is_active AND status <> 'CANCELLED' LIMIT 1`,
			empresa, *e.ChaveAcesso).Scan(&existente)
		if err == nil {
			return nil, errorsuc.NewConflictError(fmt.Sprintf("esta NF-e (chave %s) já foi importada como entrada %d", *e.ChaveAcesso, existente))
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	if e.StockStatus == "" {
		e.StockStatus = entity.StockStatusPendente
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO fiscal_entries
			(chave_acesso, numero_nf, serie, modelo, data_emissao, data_entrada,
			 cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
			 valor_produtos, valor_frete, valor_seguro, valor_desconto,
			 valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
			 tipo_documento, purchase_order_code, cte_code, status, xml_path, notes, created_by, supplier_code, enterprise_id,
			 natureza_operacao, cnpj_destinatario, protocolo, valor_icms_st, valor_outras,
			 modalidade_frete, informacoes_complementares, xml_content, sem_pagamento,
			 entry_operation_code, base_ibscbs, valor_ibs, valor_cbs, valor_is,
			 valor_ret_pis, valor_ret_cofins, valor_ret_csll, base_irrf, valor_irrf, base_ret_prev, valor_ret_prev, valor_iss_ret, stock_status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,
		         $29,$30,$31,$32,$33,$34,$35,$36,$37,
		         $38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51)
		 RETURNING id, is_active, created_at, updated_at`,
		e.ChaveAcesso, e.NumeroNF, e.Serie, e.Modelo, e.DataEmissao, e.DataEntrada,
		e.CnpjEmitente, e.RazaoSocialEmitente, e.IEEmitente, e.UFEmitente,
		e.ValorProdutos, e.ValorFrete, e.ValorSeguro, e.ValorDesconto,
		e.ValorIPI, e.ValorICMS, e.ValorPIS, e.ValorCOFINS, e.ValorTotal,
		e.TipoDocumento, e.PurchaseOrderCode, e.CteCode, e.Status, e.XmlPath, e.Notes, e.CreatedBy, e.SupplierCode, empresa,
		e.NaturezaOperacao, e.CnpjDestinatario, e.Protocolo, e.ValorICMSST, e.ValorOutras,
		e.ModalidadeFrete, e.InformacoesComplementares, e.XMLContent, e.SemPagamento,
		e.EntryOperationCode, e.BaseIBSCBS, e.ValorIBS, e.ValorCBS, e.ValorIS,
		e.ValorRetPIS, e.ValorRetCOFINS, e.ValorRetCSLL, e.BaseIRRF, e.ValorIRRF, e.BaseRetPrev, e.ValorRetPrev, e.ValorISSRet, e.StockStatus,
	).Scan(&e.ID, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("gravando nota de entrada: %w", err)
	}
	e.EnterpriseID = empresa

	for _, it := range e.Itens {
		it.FiscalEntryID = e.ID
		err = tx.QueryRow(ctx,
			`INSERT INTO fiscal_entry_items
				(fiscal_entry_id, sequence, item_code, ncm, cfop, quantity, unit_price, total_price,
				 base_icms, aliq_icms, valor_icms, base_ipi, aliq_ipi, valor_ipi, valor_pis, valor_cofins,
				 cst_icms, cst_ipi, cst_pis, cst_cofins,
				 gera_credito_icms, gera_credito_ipi, gera_credito_pis, gera_credito_cofins,
				 description, notes, uom, item_supplier_id, supplier_item_identifier, resolution_strategy, resolved_at,
				 ean, cest, origem_mercadoria, valor_frete, valor_seguro, valor_desconto, valor_outras,
				 base_icms_st, valor_icms_st, valor_contabil, fator_conversao, quantidade_estoque,
				 pedido_compra_xml, item_pedido_xml, plano_contas_id, centro_custo_id,
				 cfop_entrada, entry_operation_code, movimenta_estoque, gera_financeiro, warehouse_id, purchase_order_code, purchase_order_item_code,
				 cst_ibscbs, cclass_trib, base_ibscbs, aliq_ibs_uf, valor_ibs_uf, aliq_ibs_mun, valor_ibs_mun, valor_ibs, aliq_cbs, valor_cbs, valor_is, gera_credito_ibscbs)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,
			         $32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,
			         $48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59,$60,$61,$62,$63,$64,$65,$66)
			 RETURNING id, created_at`,
			it.FiscalEntryID, it.Sequence, it.ItemCode, it.Ncm, it.Cfop, it.Quantity, it.UnitPrice, it.TotalPrice,
			it.BaseICMS, it.AliqICMS, it.ValorICMS, it.BaseIPI, it.AliqIPI, it.ValorIPI, it.ValorPIS, it.ValorCOFINS,
			it.CstICMS, it.CstIPI, it.CstPIS, it.CstCOFINS,
			it.GeraCreditoICMS, it.GeraCreditoIPI, it.GeraCreditoPIS, it.GeraCreditoCOFINS,
			truncar(it.Description, 300), it.Notes, truncar(it.UOM, 10), it.ItemSupplierID, it.SupplierItemCode, it.ResolutionStrategy, it.ResolvedAt,
			it.EAN, it.CEST, it.Origem, it.ValorFrete, it.ValorSeguro, it.ValorDesconto, it.ValorOutras,
			it.BaseICMSST, it.ValorICMSST, it.ValorContabil, it.FatorConversao, it.QuantidadeEstoque,
			truncar(it.PedidoCompraXML, 15), truncar(it.ItemPedidoXML, 6), it.PlanoContasID, it.CentroCustoID,
			truncar(it.CfopEntrada, 4), it.EntryOperationCode, it.MovimentaEstoque, it.GeraFinanceiro, it.WarehouseID, it.PurchaseOrderCode, it.PurchaseOrderItemCode,
			truncar(it.CSTIBSCBS, 3), truncar(it.ClassTrib, 6), it.BaseIBSCBS, it.AliqIBSUF, it.ValorIBSUF, it.AliqIBSMun, it.ValorIBSMun, it.ValorIBS, it.AliqCBS, it.ValorCBS, it.ValorIS, it.GeraCreditoIBSCBS,
		).Scan(&it.ID, &it.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("gravando item %d da nota de entrada: %w", it.Sequence, err)
		}
	}

	if err = insertParcelas(ctx, tx, empresa, e.ID, e.Parcelas); err != nil {
		return nil, err
	}
	// Liga o documento recebido pela distribuição DF-e à nota importada.
	if e.ChaveAcesso != nil && *e.ChaveAcesso != "" {
		if _, err = tx.Exec(ctx, `UPDATE fiscal_received_documents SET fiscal_entry_id=$3 WHERE enterprise_id=$1 AND chave_acesso=$2`,
			empresa, *e.ChaveAcesso, e.ID); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return e, nil
}

func insertParcelas(ctx context.Context, tx pgx.Tx, empresa, entryID int64, parcelas []*entity.FiscalEntryInstallment) error {
	for _, p := range parcelas {
		p.FiscalEntryID = entryID
		if err := tx.QueryRow(ctx,
			`INSERT INTO fiscal_entry_installments
				(enterprise_id, fiscal_entry_id, numero, documento, data_vencimento, valor, forma_pagamento, origem)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
			empresa, entryID, p.Numero, truncar(p.Documento, 60), p.DataVencimento, p.Valor, truncar(p.FormaPagamento, 20), p.Origem,
		).Scan(&p.ID); err != nil {
			return fmt.Errorf("gravando parcela %d: %w", p.Numero, err)
		}
		for i := range p.Distribuicao {
			a := &p.Distribuicao[i]
			if err := tx.QueryRow(ctx,
				`INSERT INTO fiscal_entry_installment_allocations (installment_id, plano_contas_id, centro_custo_id, valor)
				 VALUES ($1,$2,$3,$4) RETURNING id`,
				p.ID, a.PlanoContasID, a.CentroCustoID, a.Valor,
			).Scan(&a.ID); err != nil {
				return fmt.Errorf("gravando distribuição da parcela %d: %w", p.Numero, err)
			}
		}
	}
	return nil
}

func (r *FiscalRepositoryPG) GetEntryXML(ctx context.Context, id int64) (int64, string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, "", err
	}
	var numero int64
	var conteudo *string
	err = r.pool.QueryRow(ctx, `SELECT numero_nf, xml_content FROM fiscal_entries WHERE id=$1 AND enterprise_id=$2`, id, empresa).Scan(&numero, &conteudo)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", errorsuc.NewNotFoundError(fmt.Sprintf("nota fiscal de entrada %d não encontrada", id))
	}
	if err != nil {
		return 0, "", err
	}
	if conteudo == nil {
		return numero, "", nil
	}
	return numero, *conteudo, nil
}

func (r *FiscalRepositoryPG) GetEntryDocument(ctx context.Context, id int64) (*entity.FiscalEntry, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	e, err := scanEntryHeader(r.pool.QueryRow(ctx,
		`SELECT `+entryHeaderCols+` FROM fiscal_entries e WHERE e.id = $1 AND e.enterprise_id = $2`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("nota fiscal de entrada %d não encontrada", id))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo nota de entrada: %w", err)
	}
	e.EnterpriseID = empresa
	// Pedido de compra usa a convenção enterprise_code (compras).
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT i.id, i.fiscal_entry_id, i.sequence, i.item_code, i.ncm, i.cfop, i.quantity, i.unit_price, i.total_price,
		        i.base_icms, i.aliq_icms, i.valor_icms, i.base_ipi, i.aliq_ipi, i.valor_ipi, i.valor_pis, i.valor_cofins,
		        i.cst_icms, i.cst_ipi, i.cst_pis, i.cst_cofins,
		        i.gera_credito_icms, i.gera_credito_ipi, i.gera_credito_pis, i.gera_credito_cofins,
		        i.description, i.notes, i.created_at, i.uom, i.item_supplier_id, i.supplier_item_identifier, i.resolution_strategy, i.resolved_at,
		        i.ean, i.cest, i.origem_mercadoria, i.valor_frete, i.valor_seguro, i.valor_desconto, i.valor_outras,
		        i.base_icms_st, i.valor_icms_st, i.valor_contabil, i.fator_conversao, i.quantidade_estoque,
		        i.pedido_compra_xml, i.item_pedido_xml, i.plano_contas_id, i.centro_custo_id,
		        i.cfop_entrada, i.entry_operation_code, i.movimenta_estoque, i.gera_financeiro, i.warehouse_id,
		        i.purchase_order_code, i.purchase_order_item_code, i.qtd_recebida_antes, i.stock_movement_id, i.custo_aquisicao,
		        i.cst_ibscbs, i.cclass_trib, i.base_ibscbs, i.aliq_ibs_uf, i.valor_ibs_uf, i.aliq_ibs_mun, i.valor_ibs_mun,
		        i.valor_ibs, i.aliq_cbs, i.valor_cbs, i.valor_is, i.gera_credito_ibscbs,
		        it.name, it.warehouse_unit_of_measurement::text, pc.codigo, pc.descricao, cc.descricao,
		        w.code || ' — ' || w.description, op.description,
		        CASE WHEN it.code IS NULL THEN NULL ELSE `+ncmDoItemSQL+` END,
		        po.order_number, poi.sequence
		   FROM fiscal_entry_items i
		   LEFT JOIN items it ON it.code = i.item_code AND it.enterprise_id = $2
		   LEFT JOIN plano_contas pc ON pc.id = i.plano_contas_id AND pc.enterprise_id = $2
		   LEFT JOIN centros_custo cc ON cc.id = i.centro_custo_id AND cc.enterprise_id = $2
		   LEFT JOIN warehouse w ON w.id = i.warehouse_id AND w.enterprise_id = $2
		   LEFT JOIN entry_operation_types op ON op.code = i.entry_operation_code
		   LEFT JOIN purchase_order_items poi ON poi.code = i.purchase_order_item_code
		   LEFT JOIN purchase_orders po ON po.code = COALESCE(poi.purchase_order_code, i.purchase_order_code) AND po.enterprise_code = $3
		  WHERE i.fiscal_entry_id = $1
		  ORDER BY i.sequence, i.id`, id, empresa, empresaCode)
	if err != nil {
		return nil, fmt.Errorf("lendo itens da nota de entrada: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var it entity.FiscalEntryItem
		if err := rows.Scan(&it.ID, &it.FiscalEntryID, &it.Sequence, &it.ItemCode, &it.Ncm, &it.Cfop, &it.Quantity, &it.UnitPrice, &it.TotalPrice,
			&it.BaseICMS, &it.AliqICMS, &it.ValorICMS, &it.BaseIPI, &it.AliqIPI, &it.ValorIPI, &it.ValorPIS, &it.ValorCOFINS,
			&it.CstICMS, &it.CstIPI, &it.CstPIS, &it.CstCOFINS,
			&it.GeraCreditoICMS, &it.GeraCreditoIPI, &it.GeraCreditoPIS, &it.GeraCreditoCOFINS,
			&it.Description, &it.Notes, &it.CreatedAt, &it.UOM, &it.ItemSupplierID, &it.SupplierItemCode, &it.ResolutionStrategy, &it.ResolvedAt,
			&it.EAN, &it.CEST, &it.Origem, &it.ValorFrete, &it.ValorSeguro, &it.ValorDesconto, &it.ValorOutras,
			&it.BaseICMSST, &it.ValorICMSST, &it.ValorContabil, &it.FatorConversao, &it.QuantidadeEstoque,
			&it.PedidoCompraXML, &it.ItemPedidoXML, &it.PlanoContasID, &it.CentroCustoID,
			&it.CfopEntrada, &it.EntryOperationCode, &it.MovimentaEstoque, &it.GeraFinanceiro, &it.WarehouseID,
			&it.PurchaseOrderCode, &it.PurchaseOrderItemCode, &it.QtdRecebidaAntes, &it.StockMovementID, &it.CustoAquisicao,
			&it.CSTIBSCBS, &it.ClassTrib, &it.BaseIBSCBS, &it.AliqIBSUF, &it.ValorIBSUF, &it.AliqIBSMun, &it.ValorIBSMun,
			&it.ValorIBS, &it.AliqCBS, &it.ValorCBS, &it.ValorIS, &it.GeraCreditoIBSCBS,
			&it.ItemName, &it.ItemUOM, &it.PlanoContasCodigo, &it.PlanoContasNome, &it.CentroCustoNome,
			&it.WarehouseName, &it.EntryOperationName, &it.ItemNCM,
			&it.PurchaseOrderNumber, &it.PurchaseOrderSequence,
		); err != nil {
			return nil, fmt.Errorf("lendo item da nota de entrada: %w", err)
		}
		e.Itens = append(e.Itens, &it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if e.Parcelas, err = r.loadParcelas(ctx, empresa, id); err != nil {
		return nil, err
	}
	return e, nil
}

func (r *FiscalRepositoryPG) loadParcelas(ctx context.Context, empresa, entryID int64) ([]*entity.FiscalEntryInstallment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.fiscal_entry_id, p.numero, p.documento, p.data_vencimento, p.valor, p.forma_pagamento, p.origem, p.conta_pagar_id,
		        a.id, a.plano_contas_id, a.centro_custo_id, a.valor
		   FROM fiscal_entry_installments p
		   LEFT JOIN fiscal_entry_installment_allocations a ON a.installment_id = p.id
		  WHERE p.fiscal_entry_id = $1 AND p.enterprise_id = $2
		  ORDER BY p.numero, a.plano_contas_id, a.centro_custo_id NULLS FIRST, a.id`, entryID, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo parcelas da nota de entrada: %w", err)
	}
	defer rows.Close()
	var out []*entity.FiscalEntryInstallment
	porID := map[int64]*entity.FiscalEntryInstallment{}
	for rows.Next() {
		var p entity.FiscalEntryInstallment
		var aID, aPlano, aCC *int64
		var aValor *decimal.Decimal
		if err := rows.Scan(&p.ID, &p.FiscalEntryID, &p.Numero, &p.Documento, &p.DataVencimento, &p.Valor, &p.FormaPagamento, &p.Origem, &p.ContaPagarID,
			&aID, &aPlano, &aCC, &aValor); err != nil {
			return nil, fmt.Errorf("lendo parcela: %w", err)
		}
		cur, ok := porID[p.ID]
		if !ok {
			cp := p
			cur = &cp
			porID[p.ID] = cur
			out = append(out, cur)
		}
		if aID != nil && aPlano != nil && aValor != nil {
			cur.Distribuicao = append(cur.Distribuicao, entity.InstallmentAllocation{ID: *aID, PlanoContasID: *aPlano, CentroCustoID: aCC, Valor: *aValor})
		}
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) SaveConciliation(ctx context.Context, entryID int64, cab repository.HeaderConciliation, itens []repository.ItemConciliation, vinculos []repository.SupplierItemLink, parcelas []*entity.FiscalEntryInstallment, status entity.FiscalEntryStatus) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var atual string
	if err = tx.QueryRow(ctx, `SELECT status FROM fiscal_entries WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, entryID, empresa).Scan(&atual); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errorsuc.NewNotFoundError(fmt.Sprintf("nota fiscal de entrada %d não encontrada", entryID))
		}
		return err
	}
	if atual != string(entity.EntryStatusPending) && atual != string(entity.EntryStatusConferred) {
		return errorsuc.NewValidationError(fmt.Sprintf("a nota está %s e não pode mais ser alterada", atual))
	}

	for _, c := range itens {
		tag, err := tx.Exec(ctx,
			`UPDATE fiscal_entry_items SET item_code=$3, item_supplier_id=$4, resolution_strategy=$5,
			        resolved_at = CASE WHEN $3::bigint IS NULL THEN NULL ELSE COALESCE(resolved_at, NOW()) END,
			        plano_contas_id=$6, centro_custo_id=$7, fator_conversao=$8, quantidade_estoque=$9,
			        cfop_entrada=$10, entry_operation_code=$11, movimenta_estoque=$12, gera_financeiro=$13, warehouse_id=$14,
			        purchase_order_code=$15, purchase_order_item_code=$16,
			        gera_credito_icms=$17, gera_credito_ipi=$18, gera_credito_pis=$19, gera_credito_cofins=$20, gera_credito_ibscbs=$21
			  WHERE id=$1 AND fiscal_entry_id=$2`,
			c.ItemID, entryID, c.ItemCode, c.ItemSupplierID, c.ResolutionStrategy, c.PlanoContasID, c.CentroCustoID, c.FatorConversao, c.QuantidadeEstoque,
			truncar(c.CfopEntrada, 4), c.EntryOperationCode, c.MovimentaEstoque, c.GeraFinanceiro, c.WarehouseID,
			c.PurchaseOrderCode, c.PurchaseOrderItemCode,
			c.GeraCreditoICMS, c.GeraCreditoIPI, c.GeraCreditoPIS, c.GeraCreditoCOFINS, c.GeraCreditoIBSCBS)
		if err != nil {
			return fmt.Errorf("gravando conciliação do item %d: %w", c.ItemID, err)
		}
		if tag.RowsAffected() == 0 {
			return errorsuc.NewValidationError(fmt.Sprintf("o item %d não pertence a esta nota", c.ItemID))
		}
	}

	for _, v := range vinculos {
		linkID, err := upsertVinculo(ctx, tx, empresa, v)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx,
			`UPDATE fiscal_entry_items SET item_supplier_id=$3
			  WHERE fiscal_entry_id=$1 AND item_code=$2 AND upper(btrim(COALESCE(supplier_item_identifier,'')))=upper(btrim($4))`,
			entryID, v.ItemCode, linkID, v.SupplierItemCode); err != nil {
			return err
		}
	}

	if parcelas != nil {
		var travada bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fiscal_entry_installments WHERE fiscal_entry_id=$1 AND conta_pagar_id IS NOT NULL)`, entryID).Scan(&travada); err != nil {
			return err
		}
		if travada {
			return errorsuc.NewValidationError("as parcelas desta nota já geraram contas a pagar e não podem ser alteradas")
		}
		if _, err = tx.Exec(ctx, `DELETE FROM fiscal_entry_installments WHERE fiscal_entry_id=$1 AND enterprise_id=$2`, entryID, empresa); err != nil {
			return err
		}
		if err = insertParcelas(ctx, tx, empresa, entryID, parcelas); err != nil {
			return err
		}
	}

	if _, err = tx.Exec(ctx, `UPDATE fiscal_entries SET status=$3, entry_operation_code=$4, purchase_order_code=COALESCE($5, purchase_order_code), updated_at=NOW()
		WHERE id=$1 AND enterprise_id=$2`, entryID, empresa, string(status), cab.EntryOperationCode, cab.PurchaseOrderCode); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// upsertVinculo memoriza o "de/para" do fornecedor. Se o código do fornecedor
// já apontava para OUTRO item, o vínculo antigo perde o código: quem corrige a
// conciliação está dizendo que o vínculo anterior estava errado.
func upsertVinculo(ctx context.Context, tx pgx.Tx, empresa int64, v repository.SupplierItemLink) (int64, error) {
	code := strings.TrimSpace(v.SupplierItemCode)
	if code == "" {
		return 0, errorsuc.NewValidationError("o item da nota não tem código do fornecedor para memorizar o vínculo")
	}
	if _, err := tx.Exec(ctx,
		`UPDATE item_preferred_suppliers SET supplier_item_code = NULL, updated_at = NOW()
		  WHERE enterprise_id=$1 AND supplier_code=$2 AND is_active AND item_code <> $3
		    AND upper(btrim(COALESCE(supplier_item_code,''))) = upper(btrim($4))`,
		empresa, v.SupplierCode, v.ItemCode, code); err != nil {
		return 0, fmt.Errorf("liberando código do fornecedor: %w", err)
	}
	if v.Barcode != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE item_preferred_suppliers SET barcode = NULL, updated_at = NOW()
			  WHERE enterprise_id=$1 AND supplier_code=$2 AND is_active AND item_code <> $3 AND barcode = $4`,
			empresa, v.SupplierCode, v.ItemCode, *v.Barcode); err != nil {
			return 0, fmt.Errorf("liberando código de barras do fornecedor: %w", err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx,
		`INSERT INTO item_preferred_suppliers
			(enterprise_id, item_code, supplier_code, mask, ranking, supplier_item_code, supplier_description,
			 uom, xml_uom, conversion_factor, barcode, created_by)
		 VALUES ($1,$2,$3,'',
		         COALESCE((SELECT MAX(ranking)+1 FROM item_preferred_suppliers WHERE enterprise_id=$1 AND item_code=$2 AND is_active), 1),
		         $4,$5,$6,$6,$7,$8,$9)
		 ON CONFLICT (enterprise_id, item_code, supplier_code, mask) WHERE enterprise_id IS NOT NULL
		 DO UPDATE SET supplier_item_code = EXCLUDED.supplier_item_code,
		               supplier_description = EXCLUDED.supplier_description,
		               uom = COALESCE(item_preferred_suppliers.uom, EXCLUDED.uom),
		               xml_uom = COALESCE(EXCLUDED.xml_uom, item_preferred_suppliers.xml_uom),
		               conversion_factor = COALESCE(EXCLUDED.conversion_factor, item_preferred_suppliers.conversion_factor),
		               barcode = COALESCE(EXCLUDED.barcode, item_preferred_suppliers.barcode),
		               is_active = TRUE, updated_at = NOW()
		 RETURNING id`,
		empresa, v.ItemCode, v.SupplierCode, code, truncarStr(v.SupplierDescription, 200), truncar(v.XMLUOM, 10), v.ConversionFactor, v.Barcode, v.CreatedBy,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("memorizando vínculo do item %d com o fornecedor %d: %w", v.ItemCode, v.SupplierCode, err)
	}
	return id, nil
}

// ---- apoio à conciliação ----

func (r *FiscalRepositoryPG) SupplierByCode(ctx context.Context, code int64) (*repository.SupplierRef, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var s repository.SupplierRef
	err = r.pool.QueryRow(ctx,
		`SELECT s.code, s.name, s.is_active, s.blocked
		   FROM suppliers s WHERE s.enterprise_id = $1 AND s.code = $2`, empresa, code).Scan(&s.Code, &s.Name, &s.IsActive, &s.Blocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("procurando fornecedor %d: %w", code, err)
	}
	return &s, nil
}

func (r *FiscalRepositoryPG) LinkSupplierToPendingEntries(ctx context.Context, cnpj string, code int64) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	digitos := soDigitosRepo(cnpj)
	if digitos == "" {
		return 0, nil
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE fiscal_entries SET supplier_code = $3, updated_at = NOW()
		  WHERE enterprise_id = $1 AND regexp_replace(cnpj_emitente, '\D', '', 'g') = $2
		    AND supplier_code IS NULL AND status IN ('PENDING','CONFERRED') AND is_active`, empresa, digitos, code)
	if err != nil {
		return 0, fmt.Errorf("ligando o fornecedor às notas pendentes: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *FiscalRepositoryPG) FindSupplierByDocument(ctx context.Context, document string) (*repository.SupplierRef, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	digitos := soDigitosRepo(document)
	if digitos == "" {
		return nil, nil
	}
	// O documento pode ter sido cadastrado com máscara: compara só os dígitos.
	var s repository.SupplierRef
	err = r.pool.QueryRow(ctx,
		`SELECT s.code, s.name, s.is_active, s.blocked,
		        (SELECT se.financial_account FROM supplier_enterprises se
		          WHERE se.supplier_id = s.id AND se.is_active
		            AND se.enterprise_code = (SELECT code FROM enterprise WHERE id = $1) LIMIT 1)
		   FROM suppliers s
		  WHERE s.enterprise_id = $1 AND regexp_replace(s.document_number, '\D', '', 'g') = $2
		  ORDER BY s.is_active DESC, s.blocked, s.code
		  LIMIT 1`, empresa, digitos).Scan(&s.Code, &s.Name, &s.IsActive, &s.Blocked, &s.FinancialAccount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("procurando fornecedor pelo CNPJ: %w", err)
	}
	return &s, nil
}

func (r *FiscalRepositoryPG) ResolveByEAN(ctx context.Context, supplierCode *int64, ean string) (*repository.ItemMatch, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ean = strings.TrimSpace(ean)
	if ean == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT s.item_code, CASE WHEN s.supplier_code = $3 THEN s.id END
		   FROM item_preferred_suppliers s
		   JOIN items it ON it.code = s.item_code AND it.enterprise_id = s.enterprise_id
		  WHERE s.enterprise_id = $1 AND s.is_active AND s.barcode = $2
		    AND ($3::bigint IS NULL OR s.supplier_code = $3 OR NOT EXISTS (
		          SELECT 1 FROM item_preferred_suppliers x WHERE x.enterprise_id=$1 AND x.is_active AND x.barcode=$2 AND x.supplier_code=$3))
		  LIMIT 2`, empresa, ean, supplierCode)
	if err != nil {
		return nil, fmt.Errorf("procurando item pelo código de barras: %w", err)
	}
	defer rows.Close()
	var achados []repository.ItemMatch
	for rows.Next() {
		var m repository.ItemMatch
		if err := rows.Scan(&m.ItemCode, &m.ItemSupplierID); err != nil {
			return nil, err
		}
		achados = append(achados, m)
	}
	// Mais de um item com o mesmo código de barras é ambíguo: quem decide é o
	// usuário, não o sistema.
	if len(achados) != 1 {
		return nil, rows.Err()
	}
	return &achados[0], rows.Err()
}

func (r *FiscalRepositoryPG) LastAccountForItem(ctx context.Context, itemCode int64) (*repository.ItemAccount, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var a repository.ItemAccount
	err = r.pool.QueryRow(ctx,
		`SELECT i.plano_contas_id, i.centro_custo_id
		   FROM fiscal_entry_items i
		   JOIN fiscal_entries e ON e.id = i.fiscal_entry_id AND e.enterprise_id = $1
		   JOIN plano_contas pc ON pc.id = i.plano_contas_id AND pc.is_active
		  WHERE i.item_code = $2 AND i.plano_contas_id IS NOT NULL AND e.status <> 'CANCELLED'
		  ORDER BY e.data_entrada DESC, i.id DESC LIMIT 1`, empresa, itemCode).Scan(&a.PlanoContasID, &a.CentroCustoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *FiscalRepositoryPG) PlanoContasByCodigo(ctx context.Context, codigo string) (*int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return nil, nil
	}
	var id int64
	err = r.pool.QueryRow(ctx, `SELECT id FROM plano_contas WHERE enterprise_id=$1 AND codigo=$2 AND is_active LIMIT 1`, empresa, codigo).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

const candidatoCols = `it.code, it.name, it.warehouse_unit_of_measurement::text, ` + ncmDoItemSQL + `, it.health::text = 'ATIVO', COALESCE(it.accounting_origin, 0)::text,
	COALESCE((SELECT w.id FROM warehouse w WHERE w.id = it.supplies_warehouse_code AND w.enterprise_id = it.enterprise_id),
	         (SELECT w.id FROM warehouse w WHERE w.id = it.warehouse_code AND w.enterprise_id = it.enterprise_id))`

func (r *FiscalRepositoryPG) SearchItems(ctx context.Context, termos []string, limite int) ([]repository.ItemCandidate, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if limite <= 0 {
		limite = 50
	}
	conds := []string{}
	args := []any{empresa}
	for _, t := range termos {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		args = append(args, "%"+t+"%")
		n := len(args)
		conds = append(conds, fmt.Sprintf("(it.name ILIKE $%d OR it.commercial_description ILIKE $%d OR it.code::text = btrim($%d, '%%'))", n, n, n))
	}
	if len(conds) == 0 {
		return nil, nil
	}
	args = append(args, limite)
	q := `SELECT ` + candidatoCols + ` FROM items it WHERE it.enterprise_id = $1 AND (` + strings.Join(conds, " OR ") +
		fmt.Sprintf(`) ORDER BY it.code LIMIT $%d`, len(args))
	return r.queryCandidatos(ctx, q, args...)
}

func (r *FiscalRepositoryPG) ItemsByCode(ctx context.Context, codes []int64) (map[int64]repository.ItemCandidate, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]repository.ItemCandidate{}
	if len(codes) == 0 {
		return out, nil
	}
	cs, err := r.queryCandidatos(ctx, `SELECT `+candidatoCols+` FROM items it WHERE it.enterprise_id=$1 AND it.code = ANY($2)`, empresa, codes)
	if err != nil {
		return nil, err
	}
	for _, c := range cs {
		out[c.Code] = c
	}
	return out, nil
}

func (r *FiscalRepositoryPG) SupplierLinksForEntry(ctx context.Context, supplierCode int64) ([]repository.ItemCandidate, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT it.code, it.name, it.warehouse_unit_of_measurement::text, `+ncmDoItemSQL+`, it.health::text = 'ATIVO', COALESCE(it.accounting_origin, 0)::text,
		        COALESCE(s.supplier_item_code,''), s.id, COALESCE(s.supplier_description,''), s.xml_uom, s.conversion_factor, COALESCE(s.barcode,'')
		   FROM item_preferred_suppliers s
		   JOIN items it ON it.code = s.item_code AND it.enterprise_id = s.enterprise_id
		  WHERE s.enterprise_id = $1 AND s.supplier_code = $2 AND s.is_active
		    AND (s.valid_until IS NULL OR s.valid_until >= CURRENT_DATE)
		  ORDER BY s.is_preferred DESC, s.ranking, it.code LIMIT 500`, empresa, supplierCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []repository.ItemCandidate
	for rows.Next() {
		var c repository.ItemCandidate
		var link int64
		if err := rows.Scan(&c.Code, &c.Name, &c.UOM, &c.NCM, &c.IsActive, &c.Origem, &c.SupplierItemCode, &link,
			&c.SupplierDescription, &c.XMLUOM, &c.ConversionFactor, &c.Barcode); err != nil {
			return nil, err
		}
		c.ItemSupplierID = &link
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) queryCandidatos(ctx context.Context, q string, args ...any) ([]repository.ItemCandidate, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("buscando itens do cadastro: %w", err)
	}
	defer rows.Close()
	var out []repository.ItemCandidate
	for rows.Next() {
		var c repository.ItemCandidate
		if err := rows.Scan(&c.Code, &c.Name, &c.UOM, &c.NCM, &c.IsActive, &c.Origem, &c.DefaultWarehouseID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *FiscalRepositoryPG) ExistingPlanos(ctx context.Context, ids []int64) (map[int64]bool, error) {
	return r.existentes(ctx, `SELECT id FROM plano_contas WHERE enterprise_id=$1 AND is_active AND id = ANY($2)`, ids)
}

func (r *FiscalRepositoryPG) ExistingCentros(ctx context.Context, ids []int64) (map[int64]bool, error) {
	return r.existentes(ctx, `SELECT id FROM centros_custo WHERE enterprise_id=$1 AND is_active AND id = ANY($2)`, ids)
}

func (r *FiscalRepositoryPG) existentes(ctx context.Context, q string, ids []int64) (map[int64]bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, q, empresa, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func truncar(s *string, n int) *string {
	if s == nil {
		return nil
	}
	v := truncarStr(*s, n)
	return &v
}

func truncarStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func nuloSeVazio(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func soDigitosRepo(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
