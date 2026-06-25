package fiscal

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ = time.Now
var _ = uuid.UUID{}

type FiscalRepositoryPG struct {
	pool *pgxpool.Pool
}

var _ repository.FiscalRepository = (*FiscalRepositoryPG)(nil)

func NewFiscalRepositoryPG(pool *pgxpool.Pool) repository.FiscalRepository {
	return &FiscalRepositoryPG{pool: pool}
}

// ---------- Fiscal Entries ----------

func (r *FiscalRepositoryPG) CreateEntry(ctx context.Context, e *entity.FiscalEntry) (*entity.FiscalEntry, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.fiscal_entries
			(chave_acesso, numero_nf, serie, modelo, data_emissao, data_entrada,
			 cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
			 valor_produtos, valor_frete, valor_seguro, valor_desconto,
			 valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
			 tipo_documento, purchase_order_code, cte_code, status, xml_path, notes, created_by, supplier_code)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
		 RETURNING id, is_active, created_at, updated_at`,
		e.ChaveAcesso, e.NumeroNF, e.Serie, e.Modelo, e.DataEmissao, e.DataEntrada,
		e.CnpjEmitente, e.RazaoSocialEmitente, e.IEEmitente, e.UFEmitente,
		e.ValorProdutos, e.ValorFrete, e.ValorSeguro, e.ValorDesconto,
		e.ValorIPI, e.ValorICMS, e.ValorPIS, e.ValorCOFINS, e.ValorTotal,
		e.TipoDocumento, e.PurchaseOrderCode, e.CteCode, e.Status, e.XmlPath, e.Notes, e.CreatedBy, e.SupplierCode,
	).Scan(&e.ID, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating fiscal entry: %w", err)
	}
	return e, nil
}

func (r *FiscalRepositoryPG) CreateEntryItem(ctx context.Context, item *entity.FiscalEntryItem) (*entity.FiscalEntryItem, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.fiscal_entry_items
			(fiscal_entry_id, sequence, item_code, ncm, cfop, quantity, unit_price, total_price,
			 base_icms, aliq_icms, valor_icms, base_ipi, aliq_ipi, valor_ipi, valor_pis, valor_cofins,
			 cst_icms, cst_ipi, cst_pis, cst_cofins,
			 gera_credito_icms, gera_credito_ipi, gera_credito_pis, gera_credito_cofins,
			 description, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)
		 RETURNING id, created_at`,
		item.FiscalEntryID, item.Sequence, item.ItemCode, item.Ncm, item.Cfop, item.Quantity, item.UnitPrice, item.TotalPrice,
		item.BaseICMS, item.AliqICMS, item.ValorICMS, item.BaseIPI, item.AliqIPI, item.ValorIPI, item.ValorPIS, item.ValorCOFINS,
		item.CstICMS, item.CstIPI, item.CstPIS, item.CstCOFINS,
		item.GeraCreditoICMS, item.GeraCreditoIPI, item.GeraCreditoPIS, item.GeraCreditoCOFINS,
		item.Description, item.Notes,
	).Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating fiscal entry item: %w", err)
	}
	return item, nil
}

func (r *FiscalRepositoryPG) GetEntryByID(ctx context.Context, id int64) (*entity.FiscalEntry, error) {
	var e entity.FiscalEntry
	err := r.pool.QueryRow(ctx,
		`SELECT id, chave_acesso, numero_nf, serie, modelo, data_emissao, data_entrada,
		        cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
		        valor_produtos, valor_frete, valor_seguro, valor_desconto,
		        valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		        tipo_documento, purchase_order_code, cte_code, status, xml_path, notes,
		        is_active, created_at, updated_at, created_by, supplier_code
		 FROM public.fiscal_entries WHERE id = $1`, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.Modelo, &e.DataEmissao, &e.DataEntrada,
		&e.CnpjEmitente, &e.RazaoSocialEmitente, &e.IEEmitente, &e.UFEmitente,
		&e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.TipoDocumento, &e.PurchaseOrderCode, &e.CteCode, &e.Status, &e.XmlPath, &e.Notes,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.SupplierCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("fiscal entry %d not found", id)
		}
		return nil, fmt.Errorf("getting fiscal entry: %w", err)
	}
	return &e, nil
}

func (r *FiscalRepositoryPG) GetEntryItems(ctx context.Context, fiscalEntryID int64) ([]*entity.FiscalEntryItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, fiscal_entry_id, sequence, item_code, ncm, cfop, quantity, unit_price, total_price,
		        base_icms, aliq_icms, valor_icms, base_ipi, aliq_ipi, valor_ipi, valor_pis, valor_cofins,
		        cst_icms, cst_ipi, cst_pis, cst_cofins,
		        gera_credito_icms, gera_credito_ipi, gera_credito_pis, gera_credito_cofins,
		        description, notes, created_at
		 FROM public.fiscal_entry_items WHERE fiscal_entry_id = $1 ORDER BY sequence`, fiscalEntryID)
	if err != nil {
		return nil, fmt.Errorf("listing fiscal entry items: %w", err)
	}
	defer rows.Close()
	return scanEntryItems(rows)
}

func (r *FiscalRepositoryPG) ListEntries(ctx context.Context) ([]*entity.FiscalEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, chave_acesso, numero_nf, serie, modelo, data_emissao, data_entrada,
		        cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
		        valor_produtos, valor_frete, valor_seguro, valor_desconto,
		        valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		        tipo_documento, purchase_order_code, cte_code, status, xml_path, notes,
		        is_active, created_at, updated_at, created_by, supplier_code
		 FROM public.fiscal_entries ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing fiscal entries: %w", err)
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (r *FiscalRepositoryPG) ListEntriesByStatus(ctx context.Context, status entity.FiscalEntryStatus) ([]*entity.FiscalEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, chave_acesso, numero_nf, serie, modelo, data_emissao, data_entrada,
		        cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
		        valor_produtos, valor_frete, valor_seguro, valor_desconto,
		        valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		        tipo_documento, purchase_order_code, cte_code, status, xml_path, notes,
		        is_active, created_at, updated_at, created_by, supplier_code
		 FROM public.fiscal_entries WHERE status = $1 ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("listing fiscal entries by status: %w", err)
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (r *FiscalRepositoryPG) UpdateEntryStatus(ctx context.Context, id int64, status entity.FiscalEntryStatus) (*entity.FiscalEntry, error) {
	var e entity.FiscalEntry
	err := r.pool.QueryRow(ctx,
		`UPDATE public.fiscal_entries SET status = $1, updated_at = NOW() WHERE id = $2
		 RETURNING id, chave_acesso, numero_nf, serie, modelo, data_emissao, data_entrada,
		           cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
		           valor_produtos, valor_frete, valor_seguro, valor_desconto,
		           valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		           tipo_documento, purchase_order_code, cte_code, status, xml_path, notes,
		           is_active, created_at, updated_at, created_by, supplier_code`,
		status, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.Modelo, &e.DataEmissao, &e.DataEntrada,
		&e.CnpjEmitente, &e.RazaoSocialEmitente, &e.IEEmitente, &e.UFEmitente,
		&e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.TipoDocumento, &e.PurchaseOrderCode, &e.CteCode, &e.Status, &e.XmlPath, &e.Notes,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.SupplierCode)
	if err != nil {
		return nil, fmt.Errorf("updating fiscal entry status: %w", err)
	}
	return &e, nil
}

func (r *FiscalRepositoryPG) GetNextNFNumber(ctx context.Context) (int64, error) {
	var next int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(numero_nf), 0) + 1 FROM public.fiscal_exits`).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("getting next NF number: %w", err)
	}
	return next, nil
}

func scanEntries(rows pgx.Rows) ([]*entity.FiscalEntry, error) {
	var result []*entity.FiscalEntry
	for rows.Next() {
		var e entity.FiscalEntry
		if err := rows.Scan(
			&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.Modelo, &e.DataEmissao, &e.DataEntrada,
			&e.CnpjEmitente, &e.RazaoSocialEmitente, &e.IEEmitente, &e.UFEmitente,
			&e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
			&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
			&e.TipoDocumento, &e.PurchaseOrderCode, &e.CteCode, &e.Status, &e.XmlPath, &e.Notes,
			&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.SupplierCode,
		); err != nil {
			return nil, fmt.Errorf("scanning fiscal entry: %w", err)
		}
		result = append(result, &e)
	}
	return result, rows.Err()
}

func scanEntryItems(rows pgx.Rows) ([]*entity.FiscalEntryItem, error) {
	var result []*entity.FiscalEntryItem
	for rows.Next() {
		var it entity.FiscalEntryItem
		if err := rows.Scan(
			&it.ID, &it.FiscalEntryID, &it.Sequence, &it.ItemCode, &it.Ncm, &it.Cfop, &it.Quantity, &it.UnitPrice, &it.TotalPrice,
			&it.BaseICMS, &it.AliqICMS, &it.ValorICMS, &it.BaseIPI, &it.AliqIPI, &it.ValorIPI, &it.ValorPIS, &it.ValorCOFINS,
			&it.CstICMS, &it.CstIPI, &it.CstPIS, &it.CstCOFINS,
			&it.GeraCreditoICMS, &it.GeraCreditoIPI, &it.GeraCreditoPIS, &it.GeraCreditoCOFINS,
			&it.Description, &it.Notes, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning fiscal entry item: %w", err)
		}
		result = append(result, &it)
	}
	return result, rows.Err()
}

// ---------- Fiscal Exits ----------

func (r *FiscalRepositoryPG) CreateExit(ctx context.Context, e *entity.FiscalExit) (*entity.FiscalExit, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.fiscal_exits
			(chave_acesso, numero_nf, serie, data_emissao, data_saida,
			 cnpj_destinatario, razao_social_destinatario, ie_destinatario, uf_destinatario,
			 cfop, natureza_operacao, valor_produtos, valor_frete, valor_seguro, valor_desconto,
			 valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
			 sales_order_code, status, created_by, base_icms_st, valor_icms_st)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		 RETURNING id, is_active, created_at, updated_at`,
		e.ChaveAcesso, e.NumeroNF, e.Serie, e.DataEmissao, e.DataSaida,
		e.CnpjDestinatario, e.RazaoSocialDestinatario, e.IEDestinatario, e.UFDestinatario,
		e.Cfop, e.NaturezaOperacao, e.ValorProdutos, e.ValorFrete, e.ValorSeguro, e.ValorDesconto,
		e.ValorIPI, e.ValorICMS, e.ValorPIS, e.ValorCOFINS, e.ValorTotal,
		e.SalesOrderCode, e.Status, e.CreatedBy, e.BaseICMSST, e.ValorICMSST,
	).Scan(&e.ID, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating fiscal exit: %w", err)
	}
	return e, nil
}

func (r *FiscalRepositoryPG) CreateExitItem(ctx context.Context, item *entity.FiscalExitItem) (*entity.FiscalExitItem, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.fiscal_exit_items
			(fiscal_exit_id, sequence, item_code, ncm, cfop, quantity, unit_price, total_price,
			 base_icms, aliq_icms, valor_icms, valor_icms_diferido,
			 base_ipi, aliq_ipi, valor_ipi, aliq_pis, valor_pis, aliq_cofins, valor_cofins,
			 cst_icms, cst_ipi, cst_pis, cst_cofins, origem_mercadoria, description,
			 base_icms_st, aliq_icms_st, valor_icms_st, mva)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)
		 RETURNING id, created_at`,
		item.FiscalExitID, item.Sequence, item.ItemCode, item.Ncm, item.Cfop, item.Quantity, item.UnitPrice, item.TotalPrice,
		item.BaseICMS, item.AliqICMS, item.ValorICMS, item.ValorICMSDiferido,
		item.BaseIPI, item.AliqIPI, item.ValorIPI, item.AliqPIS, item.ValorPIS, item.AliqCOFINS, item.ValorCOFINS,
		item.CstICMS, item.CstIPI, item.CstPIS, item.CstCOFINS, item.OrigemMercadoria, item.Description,
		item.BaseICMSST, item.AliqICMSST, item.ValorICMSST, item.MVA,
	).Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating fiscal exit item: %w", err)
	}
	return item, nil
}

func (r *FiscalRepositoryPG) GetExitByID(ctx context.Context, id int64) (*entity.FiscalExit, error) {
	var e entity.FiscalExit
	err := r.pool.QueryRow(ctx,
		`SELECT id, chave_acesso, numero_nf, serie, data_emissao, data_saida,
		        cnpj_destinatario, razao_social_destinatario, ie_destinatario, uf_destinatario,
		        cfop, natureza_operacao, valor_produtos, valor_frete, valor_seguro, valor_desconto,
		        valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		        sales_order_code, status, protocolo, xml_path, danfe_path, focus_ref,
		        is_active, created_at, updated_at, created_by, base_icms_st, valor_icms_st
		 FROM public.fiscal_exits WHERE id = $1`, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.DataEmissao, &e.DataSaida,
		&e.CnpjDestinatario, &e.RazaoSocialDestinatario, &e.IEDestinatario, &e.UFDestinatario,
		&e.Cfop, &e.NaturezaOperacao, &e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.SalesOrderCode, &e.Status, &e.Protocolo, &e.XmlPath, &e.DanfePath, &e.FocusRef,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.BaseICMSST, &e.ValorICMSST)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("fiscal exit %d not found", id)
		}
		return nil, fmt.Errorf("getting fiscal exit: %w", err)
	}
	return &e, nil
}

func (r *FiscalRepositoryPG) GetExitItems(ctx context.Context, fiscalExitID int64) ([]*entity.FiscalExitItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, fiscal_exit_id, sequence, item_code, ncm, cfop, quantity, unit_price, total_price,
		        base_icms, aliq_icms, valor_icms, valor_icms_diferido,
		        base_ipi, aliq_ipi, valor_ipi, aliq_pis, valor_pis, aliq_cofins, valor_cofins,
		        cst_icms, cst_ipi, cst_pis, cst_cofins, origem_mercadoria, description,
		        base_icms_st, aliq_icms_st, valor_icms_st, mva, created_at
		 FROM public.fiscal_exit_items WHERE fiscal_exit_id = $1 ORDER BY sequence`, fiscalExitID)
	if err != nil {
		return nil, fmt.Errorf("listing fiscal exit items: %w", err)
	}
	defer rows.Close()
	return scanExitItems(rows)
}

func (r *FiscalRepositoryPG) ListExits(ctx context.Context) ([]*entity.FiscalExit, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, chave_acesso, numero_nf, serie, data_emissao, data_saida,
		        cnpj_destinatario, razao_social_destinatario, ie_destinatario, uf_destinatario,
		        cfop, natureza_operacao, valor_produtos, valor_frete, valor_seguro, valor_desconto,
		        valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		        sales_order_code, status, protocolo, xml_path, danfe_path, focus_ref,
		        is_active, created_at, updated_at, created_by, base_icms_st, valor_icms_st
		 FROM public.fiscal_exits ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing fiscal exits: %w", err)
	}
	defer rows.Close()
	return scanExits(rows)
}

func (r *FiscalRepositoryPG) ListExitsByStatus(ctx context.Context, status entity.FiscalExitStatus) ([]*entity.FiscalExit, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, chave_acesso, numero_nf, serie, data_emissao, data_saida,
		        cnpj_destinatario, razao_social_destinatario, ie_destinatario, uf_destinatario,
		        cfop, natureza_operacao, valor_produtos, valor_frete, valor_seguro, valor_desconto,
		        valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		        sales_order_code, status, protocolo, xml_path, danfe_path, focus_ref,
		        is_active, created_at, updated_at, created_by, base_icms_st, valor_icms_st
		 FROM public.fiscal_exits WHERE status = $1 ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("listing fiscal exits by status: %w", err)
	}
	defer rows.Close()
	return scanExits(rows)
}

func (r *FiscalRepositoryPG) UpdateExitStatus(ctx context.Context, id int64, status entity.FiscalExitStatus) (*entity.FiscalExit, error) {
	var e entity.FiscalExit
	err := r.pool.QueryRow(ctx,
		`UPDATE public.fiscal_exits SET status = $1, updated_at = NOW() WHERE id = $2
		 RETURNING id, chave_acesso, numero_nf, serie, data_emissao, data_saida,
		           cnpj_destinatario, razao_social_destinatario, ie_destinatario, uf_destinatario,
		           cfop, natureza_operacao, valor_produtos, valor_frete, valor_seguro, valor_desconto,
		           valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		           sales_order_code, status, protocolo, xml_path, danfe_path, focus_ref,
		           is_active, created_at, updated_at, created_by, base_icms_st, valor_icms_st`,
		status, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.DataEmissao, &e.DataSaida,
		&e.CnpjDestinatario, &e.RazaoSocialDestinatario, &e.IEDestinatario, &e.UFDestinatario,
		&e.Cfop, &e.NaturezaOperacao, &e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.SalesOrderCode, &e.Status, &e.Protocolo, &e.XmlPath, &e.DanfePath, &e.FocusRef,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.BaseICMSST, &e.ValorICMSST)
	if err != nil {
		return nil, fmt.Errorf("updating fiscal exit status: %w", err)
	}
	return &e, nil
}

func (r *FiscalRepositoryPG) UpdateExitAuthorization(ctx context.Context, id int64, chaveAcesso, protocolo, focusRef, xmlPath, danfePath string) (*entity.FiscalExit, error) {
	var e entity.FiscalExit
	err := r.pool.QueryRow(ctx,
		`UPDATE public.fiscal_exits SET
		     chave_acesso = $1, protocolo = $2, focus_ref = $3,
		     xml_path    = NULLIF($5, ''),
		     danfe_path  = NULLIF($6, ''),
		     status = 'AUTHORIZED', updated_at = NOW()
		 WHERE id = $4
		 RETURNING id, chave_acesso, numero_nf, serie, data_emissao, data_saida,
		           cnpj_destinatario, razao_social_destinatario, ie_destinatario, uf_destinatario,
		           cfop, natureza_operacao, valor_produtos, valor_frete, valor_seguro, valor_desconto,
		           valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		           sales_order_code, status, protocolo, xml_path, danfe_path, focus_ref,
		           is_active, created_at, updated_at, created_by, base_icms_st, valor_icms_st`,
		chaveAcesso, protocolo, focusRef, id, xmlPath, danfePath,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.DataEmissao, &e.DataSaida,
		&e.CnpjDestinatario, &e.RazaoSocialDestinatario, &e.IEDestinatario, &e.UFDestinatario,
		&e.Cfop, &e.NaturezaOperacao, &e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.SalesOrderCode, &e.Status, &e.Protocolo, &e.XmlPath, &e.DanfePath, &e.FocusRef,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.BaseICMSST, &e.ValorICMSST)
	if err != nil {
		return nil, fmt.Errorf("updating fiscal exit authorization: %w", err)
	}
	return &e, nil
}

func scanExits(rows pgx.Rows) ([]*entity.FiscalExit, error) {
	var result []*entity.FiscalExit
	for rows.Next() {
		var e entity.FiscalExit
		if err := rows.Scan(
			&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.DataEmissao, &e.DataSaida,
			&e.CnpjDestinatario, &e.RazaoSocialDestinatario, &e.IEDestinatario, &e.UFDestinatario,
			&e.Cfop, &e.NaturezaOperacao, &e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
			&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
			&e.SalesOrderCode, &e.Status, &e.Protocolo, &e.XmlPath, &e.DanfePath, &e.FocusRef,
			&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.BaseICMSST, &e.ValorICMSST,
		); err != nil {
			return nil, fmt.Errorf("scanning fiscal exit: %w", err)
		}
		result = append(result, &e)
	}
	return result, rows.Err()
}

func scanExitItems(rows pgx.Rows) ([]*entity.FiscalExitItem, error) {
	var result []*entity.FiscalExitItem
	for rows.Next() {
		var it entity.FiscalExitItem
		if err := rows.Scan(
			&it.ID, &it.FiscalExitID, &it.Sequence, &it.ItemCode, &it.Ncm, &it.Cfop, &it.Quantity, &it.UnitPrice, &it.TotalPrice,
			&it.BaseICMS, &it.AliqICMS, &it.ValorICMS, &it.ValorICMSDiferido,
			&it.BaseIPI, &it.AliqIPI, &it.ValorIPI, &it.AliqPIS, &it.ValorPIS, &it.AliqCOFINS, &it.ValorCOFINS,
			&it.CstICMS, &it.CstIPI, &it.CstPIS, &it.CstCOFINS, &it.OrigemMercadoria, &it.Description,
			&it.BaseICMSST, &it.AliqICMSST, &it.ValorICMSST, &it.MVA, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning fiscal exit item: %w", err)
		}
		result = append(result, &it)
	}
	return result, rows.Err()
}

// ---------- Fiscal Config ----------

func (r *FiscalRepositoryPG) GetFiscalConfig(ctx context.Context) (*entity.FiscalConfig, error) {
	var cfg entity.FiscalConfig
	err := r.pool.QueryRow(ctx,
		`SELECT id, cnpj_empresa, razao_social, ie_empresa, regime_tributario, uf_empresa,
		        icms_interno_aliquota, icms_diferimento_percentual,
		        focus_nfe_token, focus_nfe_ambiente, juros_mes, multa_atraso,
		        vencimento_icms_dia, vencimento_ipi_dia, vencimento_pis_cofins_dia,
		        COALESCE(logradouro,''), COALESCE(numero,''), complemento,
		        COALESCE(bairro,''), COALESCE(municipio,''), COALESCE(codigo_municipio,''),
		        COALESCE(cep,''), telefone,
		        logo, logo_mime, brand_color,
		        created_at, updated_at, updated_by
		 FROM public.fiscal_configs ORDER BY id LIMIT 1`,
	).Scan(&cfg.ID, &cfg.CnpjEmpresa, &cfg.RazaoSocial, &cfg.IEEmpresa, &cfg.RegimeTributario, &cfg.UFEmpresa,
		&cfg.IcmsInternoAliquota, &cfg.IcmsDiferimentoPercentual,
		&cfg.FocusNfeToken, &cfg.FocusNfeAmbiente, &cfg.JurosMes, &cfg.MultaAtraso,
		&cfg.VencimentoIcmsDia, &cfg.VencimentoIPIDia, &cfg.VencimentoPisCofinsDia,
		&cfg.Logradouro, &cfg.Numero, &cfg.Complemento,
		&cfg.Bairro, &cfg.Municipio, &cfg.CodigoMunicipio,
		&cfg.CEP, &cfg.Telefone,
		&cfg.Logo, &cfg.LogoMime, &cfg.BrandColor,
		&cfg.CreatedAt, &cfg.UpdatedAt, &cfg.UpdatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("fiscal config not found")
		}
		return nil, fmt.Errorf("getting fiscal config: %w", err)
	}
	return &cfg, nil
}

func (r *FiscalRepositoryPG) UpdateFiscalConfig(ctx context.Context, cfg *entity.FiscalConfig) (*entity.FiscalConfig, error) {
	// Upsert: singleton row with id=1. Works on first call (no prior row) and subsequent updates.
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.fiscal_configs
		     (id, cnpj_empresa, razao_social, ie_empresa, regime_tributario, uf_empresa,
		      icms_interno_aliquota, icms_diferimento_percentual,
		      focus_nfe_token, focus_nfe_ambiente, juros_mes, multa_atraso,
		      vencimento_icms_dia, vencimento_ipi_dia, vencimento_pis_cofins_dia,
		      logradouro, numero, complemento, bairro, municipio, codigo_municipio, cep, telefone,
		      updated_at, updated_by)
		 VALUES (1,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,NOW(),$23)
		 ON CONFLICT (id) DO UPDATE SET
		     cnpj_empresa = EXCLUDED.cnpj_empresa, razao_social = EXCLUDED.razao_social,
		     ie_empresa = EXCLUDED.ie_empresa, regime_tributario = EXCLUDED.regime_tributario,
		     uf_empresa = EXCLUDED.uf_empresa,
		     icms_interno_aliquota = EXCLUDED.icms_interno_aliquota,
		     icms_diferimento_percentual = EXCLUDED.icms_diferimento_percentual,
		     focus_nfe_token = EXCLUDED.focus_nfe_token, focus_nfe_ambiente = EXCLUDED.focus_nfe_ambiente,
		     juros_mes = EXCLUDED.juros_mes, multa_atraso = EXCLUDED.multa_atraso,
		     vencimento_icms_dia = EXCLUDED.vencimento_icms_dia,
		     vencimento_ipi_dia = EXCLUDED.vencimento_ipi_dia,
		     vencimento_pis_cofins_dia = EXCLUDED.vencimento_pis_cofins_dia,
		     logradouro = EXCLUDED.logradouro, numero = EXCLUDED.numero,
		     complemento = EXCLUDED.complemento, bairro = EXCLUDED.bairro,
		     municipio = EXCLUDED.municipio, codigo_municipio = EXCLUDED.codigo_municipio,
		     cep = EXCLUDED.cep, telefone = EXCLUDED.telefone,
		     updated_at = NOW(), updated_by = EXCLUDED.updated_by
		 RETURNING id, cnpj_empresa, razao_social, ie_empresa, regime_tributario, uf_empresa,
		           icms_interno_aliquota, icms_diferimento_percentual,
		           focus_nfe_token, focus_nfe_ambiente, juros_mes, multa_atraso,
		           vencimento_icms_dia, vencimento_ipi_dia, vencimento_pis_cofins_dia,
		           COALESCE(logradouro,''), COALESCE(numero,''), complemento,
		           COALESCE(bairro,''), COALESCE(municipio,''), COALESCE(codigo_municipio,''),
		           COALESCE(cep,''), telefone,
		           created_at, updated_at, updated_by`,
		cfg.CnpjEmpresa, cfg.RazaoSocial, cfg.IEEmpresa, cfg.RegimeTributario, cfg.UFEmpresa,
		cfg.IcmsInternoAliquota, cfg.IcmsDiferimentoPercentual,
		cfg.FocusNfeToken, cfg.FocusNfeAmbiente, cfg.JurosMes, cfg.MultaAtraso,
		cfg.VencimentoIcmsDia, cfg.VencimentoIPIDia, cfg.VencimentoPisCofinsDia,
		cfg.Logradouro, cfg.Numero, cfg.Complemento, cfg.Bairro,
		cfg.Municipio, cfg.CodigoMunicipio, cfg.CEP, cfg.Telefone,
		cfg.UpdatedBy,
	).Scan(&cfg.ID, &cfg.CnpjEmpresa, &cfg.RazaoSocial, &cfg.IEEmpresa, &cfg.RegimeTributario, &cfg.UFEmpresa,
		&cfg.IcmsInternoAliquota, &cfg.IcmsDiferimentoPercentual,
		&cfg.FocusNfeToken, &cfg.FocusNfeAmbiente, &cfg.JurosMes, &cfg.MultaAtraso,
		&cfg.VencimentoIcmsDia, &cfg.VencimentoIPIDia, &cfg.VencimentoPisCofinsDia,
		&cfg.Logradouro, &cfg.Numero, &cfg.Complemento, &cfg.Bairro,
		&cfg.Municipio, &cfg.CodigoMunicipio, &cfg.CEP, &cfg.Telefone,
		&cfg.CreatedAt, &cfg.UpdatedAt, &cfg.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("upserting fiscal config: %w", err)
	}
	return cfg, nil
}

// SetBranding stores (or clears) the company logo and/or brand colour on the
// singleton fiscal config. Nil/empty arguments leave the corresponding column
// untouched, so callers can update the logo and the colour independently.
func (r *FiscalRepositoryPG) SetBranding(ctx context.Context, logo []byte, logoMime, brandColor string, by uuid.UUID) error {
	// Ensure the singleton row exists before patching individual columns.
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.fiscal_configs (id, cnpj_empresa, razao_social, updated_by)
		 VALUES (1, '00000000000000', 'Empresa', $1)
		 ON CONFLICT (id) DO NOTHING`, by)
	if err != nil {
		return fmt.Errorf("ensuring fiscal config row: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE public.fiscal_configs SET
		     logo        = COALESCE($1, logo),
		     logo_mime   = COALESCE($2, logo_mime),
		     brand_color = COALESCE($3, brand_color),
		     updated_at  = NOW(),
		     updated_by  = $4
		 WHERE id = 1`,
		nullBytes(logo), nullStr(logoMime), nullStr(brandColor), by)
	if err != nil {
		return fmt.Errorf("setting branding: %w", err)
	}
	return nil
}

func nullBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ---------- NCM Tax Table ----------

func (r *FiscalRepositoryPG) GetNcmTax(ctx context.Context, ncm string) (*entity.NcmTaxTable, error) {
	var n entity.NcmTaxTable
	err := r.pool.QueryRow(ctx,
		`SELECT id, ncm, aliq_ipi, aliq_pis, aliq_cofins, cst_pis, cst_cofins, cst_ipi, description, is_active, created_at
		 FROM public.ncm_tax_table WHERE ncm = $1`, ncm,
	).Scan(&n.ID, &n.Ncm, &n.AliqIPI, &n.AliqPis, &n.AliqCofins, &n.CstPis, &n.CstCofins, &n.CstIPI, &n.Description, &n.IsActive, &n.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("NCM tax %s not found", ncm)
		}
		return nil, fmt.Errorf("getting NCM tax: %w", err)
	}
	return &n, nil
}

func (r *FiscalRepositoryPG) ListNcmTaxes(ctx context.Context) ([]*entity.NcmTaxTable, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, ncm, aliq_ipi, aliq_pis, aliq_cofins, cst_pis, cst_cofins, cst_ipi, description, is_active, created_at
		 FROM public.ncm_tax_table ORDER BY ncm`)
	if err != nil {
		return nil, fmt.Errorf("listing NCM taxes: %w", err)
	}
	defer rows.Close()

	var result []*entity.NcmTaxTable
	for rows.Next() {
		var n entity.NcmTaxTable
		if err := rows.Scan(&n.ID, &n.Ncm, &n.AliqIPI, &n.AliqPis, &n.AliqCofins, &n.CstPis, &n.CstCofins, &n.CstIPI, &n.Description, &n.IsActive, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning NCM tax: %w", err)
		}
		result = append(result, &n)
	}
	return result, rows.Err()
}

// ---------- Tax Scenarios ----------

func (r *FiscalRepositoryPG) ListTaxScenarios(ctx context.Context) ([]*entity.TaxScenario, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, scenario_name, destination_uf, destination_type, aliq_icms, dif_icms_pct,
		        cst_icms, calc_difal, aliq_fcp, is_active, created_at, updated_at
		 FROM public.tax_scenarios ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listing tax scenarios: %w", err)
	}
	defer rows.Close()

	var result []*entity.TaxScenario
	for rows.Next() {
		var s entity.TaxScenario
		if err := rows.Scan(&s.ID, &s.ScenarioName, &s.DestinationUF, &s.DestinationType,
			&s.AliqICMS, &s.DifICMSPct, &s.CstICMS, &s.CalcDifal, &s.AliqFCP,
			&s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning tax scenario: %w", err)
		}
		result = append(result, &s)
	}
	return result, rows.Err()
}

// ---------- ICMS Tables ----------

func (r *FiscalRepositoryPG) GetICMSInterstate(ctx context.Context, originUF, destUF string) (*float64, error) {
	var aliq float64
	err := r.pool.QueryRow(ctx,
		`SELECT aliq_icms FROM public.icms_interstate WHERE origin_uf = $1 AND destination_uf = $2 AND is_active = TRUE`,
		originUF, destUF,
	).Scan(&aliq)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("ICMS interstate not found for %s -> %s", originUF, destUF)
		}
		return nil, fmt.Errorf("getting ICMS interstate: %w", err)
	}
	return &aliq, nil
}

func (r *FiscalRepositoryPG) GetICMSInternal(ctx context.Context, uf string) (*float64, *float64, error) {
	var aliqICMS, aliqFCP float64
	err := r.pool.QueryRow(ctx,
		`SELECT aliq_icms, aliq_fcp FROM public.icms_internal WHERE uf = $1 AND is_active = TRUE`, uf,
	).Scan(&aliqICMS, &aliqFCP)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, fmt.Errorf("ICMS internal not found for %s", uf)
		}
		return nil, nil, fmt.Errorf("getting ICMS internal: %w", err)
	}
	return &aliqICMS, &aliqFCP, nil
}

func (r *FiscalRepositoryPG) ListICMSInterstate(ctx context.Context) (map[string]float64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT origin_uf, destination_uf, aliq_icms FROM public.icms_interstate WHERE is_active = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("listing ICMS interstate: %w", err)
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var origin, dest string
		var aliq float64
		if err := rows.Scan(&origin, &dest, &aliq); err != nil {
			return nil, fmt.Errorf("scanning ICMS interstate: %w", err)
		}
		result[origin+dest] = aliq
	}
	return result, rows.Err()
}

func (r *FiscalRepositoryPG) ListICMSInternal(ctx context.Context) (map[string]struct{ ICMS, FCP float64 }, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT uf, aliq_icms, aliq_fcp FROM public.icms_internal WHERE is_active = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("listing ICMS internal: %w", err)
	}
	defer rows.Close()

	result := make(map[string]struct{ ICMS, FCP float64 })
	for rows.Next() {
		var uf string
		var icms, fcp float64
		if err := rows.Scan(&uf, &icms, &fcp); err != nil {
			return nil, fmt.Errorf("scanning ICMS internal: %w", err)
		}
		result[uf] = struct{ ICMS, FCP float64 }{ICMS: icms, FCP: fcp}
	}
	return result, rows.Err()
}

// ---------- Cancel with motivo ----------

func (r *FiscalRepositoryPG) CancelExitWithMotivo(ctx context.Context, id int64, motivo string, userID uuid.UUID) (*entity.FiscalExit, error) {
	var e entity.FiscalExit
	err := r.pool.QueryRow(ctx,
		`UPDATE public.fiscal_exits SET
		     status = 'CANCELLED',
		     motivo_cancelamento = $1,
		     data_cancelamento = NOW(),
		     cancelado_por = $2,
		     updated_at = NOW()
		 WHERE id = $3
		 RETURNING id, chave_acesso, numero_nf, serie, data_emissao, data_saida,
		           cnpj_destinatario, razao_social_destinatario, ie_destinatario, uf_destinatario,
		           cfop, natureza_operacao, valor_produtos, valor_frete, valor_seguro, valor_desconto,
		           valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		           sales_order_code, status, protocolo, xml_path, danfe_path, focus_ref,
		           is_active, created_at, updated_at, created_by`,
		motivo, userID, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.DataEmissao, &e.DataSaida,
		&e.CnpjDestinatario, &e.RazaoSocialDestinatario, &e.IEDestinatario, &e.UFDestinatario,
		&e.Cfop, &e.NaturezaOperacao, &e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.SalesOrderCode, &e.Status, &e.Protocolo, &e.XmlPath, &e.DanfePath, &e.FocusRef,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("cancelling fiscal exit with motivo: %w", err)
	}
	return &e, nil
}

// ---------- Focus NF-e Logs ----------

func (r *FiscalRepositoryPG) SaveFocusLog(ctx context.Context, fiscalExitID int64, endpoint, method, reqBody, respBody string, statusCode, durationMs int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.focus_nfe_logs (fiscal_exit_id, endpoint, method, request_body, response_body, status_code, duration_ms)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		fiscalExitID, endpoint, method, reqBody, respBody, statusCode, durationMs)
	return err
}

// ---------- Carta de Correção ----------

func (r *FiscalRepositoryPG) SaveCartaCorrecao(ctx context.Context, fiscalExitID int64, texto, focusRef string, userID uuid.UUID) (int, error) {
	var seq int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(numero_seq), 0) + 1 FROM public.carta_correcao WHERE fiscal_exit_id = $1`,
		fiscalExitID).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("getting CC-e sequence: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO public.carta_correcao (fiscal_exit_id, numero_seq, texto_correcao, focus_ref, status, created_by)
		 VALUES ($1,$2,$3,$4,'ENVIADA',$5)`,
		fiscalExitID, seq, texto, focusRef, userID)
	if err != nil {
		return 0, fmt.Errorf("saving carta correcao: %w", err)
	}
	return seq, nil
}

func (r *FiscalRepositoryPG) ListCartasCorrecao(ctx context.Context, fiscalExitID int64) ([]*entity.CartaCorrecao, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, fiscal_exit_id, numero_seq, texto_correcao, focus_ref, status, protocolo, chave_evento, created_at, created_by
		 FROM public.carta_correcao WHERE fiscal_exit_id = $1 ORDER BY numero_seq`, fiscalExitID)
	if err != nil {
		return nil, fmt.Errorf("listing cartas correcao: %w", err)
	}
	defer rows.Close()
	var result []*entity.CartaCorrecao
	for rows.Next() {
		var c entity.CartaCorrecao
		if err := rows.Scan(&c.ID, &c.FiscalExitID, &c.NumeroSeq, &c.TextoCorrecao,
			&c.FocusRef, &c.Status, &c.Protocolo, &c.ChaveEvento, &c.CreatedAt, &c.CreatedBy); err != nil {
			return nil, fmt.Errorf("scanning carta correcao: %w", err)
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

// ---------- CT-e ----------

func (r *FiscalRepositoryPG) CreateCTe(ctx context.Context, c *entity.FiscalCTe) (*entity.FiscalCTe, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.fiscal_cte
		     (chave_acesso, numero_cte, serie, data_emissao, data_entrada,
		      cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
		      cfop, valor_frete, valor_seguro, valor_outros, valor_total,
		      valor_icms, base_icms, aliq_icms, cst_icms, tipo_rateio,
		      fiscal_entry_id, status, xml_path, notes, created_by, emission_data)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		 RETURNING id, is_active, created_at, updated_at`,
		c.ChaveAcesso, c.NumeroCTe, c.Serie, c.DataEmissao, c.DataEntrada,
		c.CnpjEmitente, c.RazaoSocialEmitente, c.IEEmitente, c.UFEmitente,
		c.Cfop, c.ValorFrete, c.ValorSeguro, c.ValorOutros, c.ValorTotal,
		c.ValorICMS, c.BaseICMS, c.AliqICMS, c.CstICMS, c.TipoRateio,
		c.FiscalEntryID, c.Status, c.XmlPath, c.Notes, c.CreatedBy, c.EmissionData,
	).Scan(&c.ID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating CT-e: %w", err)
	}
	return c, nil
}

func (r *FiscalRepositoryPG) GetCTeByID(ctx context.Context, id int64) (*entity.FiscalCTe, error) {
	var c entity.FiscalCTe
	err := r.pool.QueryRow(ctx,
		`SELECT id, chave_acesso, numero_cte, serie, data_emissao, data_entrada,
		        cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
		        cfop, valor_frete, valor_seguro, valor_outros, valor_total,
		        valor_icms, base_icms, aliq_icms, cst_icms, tipo_rateio,
		        fiscal_entry_id, status, xml_path, notes, is_active, created_at, updated_at, created_by,
		        focus_ref, protocolo, emission_data
		 FROM public.fiscal_cte WHERE id = $1`, id,
	).Scan(&c.ID, &c.ChaveAcesso, &c.NumeroCTe, &c.Serie, &c.DataEmissao, &c.DataEntrada,
		&c.CnpjEmitente, &c.RazaoSocialEmitente, &c.IEEmitente, &c.UFEmitente,
		&c.Cfop, &c.ValorFrete, &c.ValorSeguro, &c.ValorOutros, &c.ValorTotal,
		&c.ValorICMS, &c.BaseICMS, &c.AliqICMS, &c.CstICMS, &c.TipoRateio,
		&c.FiscalEntryID, &c.Status, &c.XmlPath, &c.Notes, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.CreatedBy,
		&c.FocusRef, &c.Protocolo, &c.EmissionData)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("CT-e %d not found", id)
		}
		return nil, fmt.Errorf("getting CT-e: %w", err)
	}
	return &c, nil
}

func (r *FiscalRepositoryPG) ListCTe(ctx context.Context) ([]*entity.FiscalCTe, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, chave_acesso, numero_cte, serie, data_emissao, data_entrada,
		        cnpj_emitente, razao_social_emitente, ie_emitente, uf_emitente,
		        cfop, valor_frete, valor_seguro, valor_outros, valor_total,
		        valor_icms, base_icms, aliq_icms, cst_icms, tipo_rateio,
		        fiscal_entry_id, status, xml_path, notes, is_active, created_at, updated_at, created_by,
		        focus_ref, protocolo, emission_data
		 FROM public.fiscal_cte WHERE is_active = true ORDER BY data_emissao DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing CT-e: %w", err)
	}
	defer rows.Close()
	var result []*entity.FiscalCTe
	for rows.Next() {
		var c entity.FiscalCTe
		if err := rows.Scan(&c.ID, &c.ChaveAcesso, &c.NumeroCTe, &c.Serie, &c.DataEmissao, &c.DataEntrada,
			&c.CnpjEmitente, &c.RazaoSocialEmitente, &c.IEEmitente, &c.UFEmitente,
			&c.Cfop, &c.ValorFrete, &c.ValorSeguro, &c.ValorOutros, &c.ValorTotal,
			&c.ValorICMS, &c.BaseICMS, &c.AliqICMS, &c.CstICMS, &c.TipoRateio,
			&c.FiscalEntryID, &c.Status, &c.XmlPath, &c.Notes, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.CreatedBy,
			&c.FocusRef, &c.Protocolo, &c.EmissionData); err != nil {
			return nil, fmt.Errorf("scanning CT-e: %w", err)
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (r *FiscalRepositoryPG) UpdateCTeStatus(ctx context.Context, id int64, status string) (*entity.FiscalCTe, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.fiscal_cte SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return nil, fmt.Errorf("updating CT-e status: %w", err)
	}
	return r.GetCTeByID(ctx, id)
}

func (r *FiscalRepositoryPG) UpdateCTeAuthorization(ctx context.Context, id int64, chaveAcesso, protocolo, focusRef string) (*entity.FiscalCTe, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.fiscal_cte SET chave_acesso = $1, protocolo = $2, focus_ref = $3,
		     status = 'AUTORIZADO', updated_at = NOW() WHERE id = $4`,
		chaveAcesso, protocolo, focusRef, id)
	if err != nil {
		return nil, fmt.Errorf("updating CT-e authorization: %w", err)
	}
	return r.GetCTeByID(ctx, id)
}

// ---------- NCM Tax Table write operations ----------

func (r *FiscalRepositoryPG) UpsertNcmTax(ctx context.Context, n *entity.NcmTaxTable) (*entity.NcmTaxTable, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.ncm_tax_table (ncm, aliq_ipi, aliq_pis, aliq_cofins, cst_pis, cst_cofins, cst_ipi, description, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
		 ON CONFLICT (ncm) DO UPDATE SET
		     aliq_ipi    = EXCLUDED.aliq_ipi,
		     aliq_pis    = EXCLUDED.aliq_pis,
		     aliq_cofins = EXCLUDED.aliq_cofins,
		     cst_pis     = EXCLUDED.cst_pis,
		     cst_cofins  = EXCLUDED.cst_cofins,
		     cst_ipi     = EXCLUDED.cst_ipi,
		     description = EXCLUDED.description,
		     is_active   = true
		 RETURNING id, ncm, aliq_ipi, aliq_pis, aliq_cofins, cst_pis, cst_cofins, cst_ipi, description, is_active, created_at`,
		n.Ncm, n.AliqIPI, n.AliqPis, n.AliqCofins, n.CstPis, n.CstCofins, n.CstIPI, n.Description,
	).Scan(&n.ID, &n.Ncm, &n.AliqIPI, &n.AliqPis, &n.AliqCofins, &n.CstPis, &n.CstCofins, &n.CstIPI, &n.Description, &n.IsActive, &n.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting NCM tax: %w", err)
	}
	return n, nil
}

func (r *FiscalRepositoryPG) DeleteNcmTax(ctx context.Context, ncm string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.ncm_tax_table SET is_active = false WHERE ncm = $1`, ncm)
	if err != nil {
		return fmt.Errorf("deleting NCM tax: %w", err)
	}
	return nil
}

// ---------- ICMS table write operations ----------

func (r *FiscalRepositoryPG) UpsertICMSInterstate(ctx context.Context, originUF, destUF string, aliq float64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.icms_interstate (origin_uf, destination_uf, aliq_icms, is_active)
		 VALUES ($1, $2, $3, true)
		 ON CONFLICT (origin_uf, destination_uf) DO UPDATE SET aliq_icms = EXCLUDED.aliq_icms, is_active = true`,
		originUF, destUF, aliq)
	if err != nil {
		return fmt.Errorf("upserting ICMS interstate: %w", err)
	}
	return nil
}

func (r *FiscalRepositoryPG) UpsertICMSInternal(ctx context.Context, uf string, aliqICMS, aliqFCP float64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.icms_internal (uf, aliq_icms, aliq_fcp, is_active)
		 VALUES ($1, $2, $3, true)
		 ON CONFLICT (uf) DO UPDATE SET aliq_icms = EXCLUDED.aliq_icms, aliq_fcp = EXCLUDED.aliq_fcp, is_active = true`,
		uf, aliqICMS, aliqFCP)
	if err != nil {
		return fmt.Errorf("upserting ICMS internal: %w", err)
	}
	return nil
}
