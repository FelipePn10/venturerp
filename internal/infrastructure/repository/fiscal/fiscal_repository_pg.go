package fiscal

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
			 tipo_documento, purchase_order_code, cte_code, status, xml_path, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)
		 RETURNING id, is_active, created_at, updated_at`,
		e.ChaveAcesso, e.NumeroNF, e.Serie, e.Modelo, e.DataEmissao, e.DataEntrada,
		e.CnpjEmitente, e.RazaoSocialEmitente, e.IEEmitente, e.UFEmitente,
		e.ValorProdutos, e.ValorFrete, e.ValorSeguro, e.ValorDesconto,
		e.ValorIPI, e.ValorICMS, e.ValorPIS, e.ValorCOFINS, e.ValorTotal,
		e.TipoDocumento, e.PurchaseOrderCode, e.CteCode, e.Status, e.XmlPath, e.Notes, e.CreatedBy,
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
		        is_active, created_at, updated_at, created_by
		 FROM public.fiscal_entries WHERE id = $1`, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.Modelo, &e.DataEmissao, &e.DataEntrada,
		&e.CnpjEmitente, &e.RazaoSocialEmitente, &e.IEEmitente, &e.UFEmitente,
		&e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.TipoDocumento, &e.PurchaseOrderCode, &e.CteCode, &e.Status, &e.XmlPath, &e.Notes,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy)
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
		        is_active, created_at, updated_at, created_by
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
		        is_active, created_at, updated_at, created_by
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
		           is_active, created_at, updated_at, created_by`,
		status, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.Modelo, &e.DataEmissao, &e.DataEntrada,
		&e.CnpjEmitente, &e.RazaoSocialEmitente, &e.IEEmitente, &e.UFEmitente,
		&e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.TipoDocumento, &e.PurchaseOrderCode, &e.CteCode, &e.Status, &e.XmlPath, &e.Notes,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy)
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
			&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy,
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
			 sales_order_code, status, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		 RETURNING id, is_active, created_at, updated_at`,
		e.ChaveAcesso, e.NumeroNF, e.Serie, e.DataEmissao, e.DataSaida,
		e.CnpjDestinatario, e.RazaoSocialDestinatario, e.IEDestinatario, e.UFDestinatario,
		e.Cfop, e.NaturezaOperacao, e.ValorProdutos, e.ValorFrete, e.ValorSeguro, e.ValorDesconto,
		e.ValorIPI, e.ValorICMS, e.ValorPIS, e.ValorCOFINS, e.ValorTotal,
		e.SalesOrderCode, e.Status, e.CreatedBy,
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
			 base_ipi, aliq_ipi, valor_ipi, valor_pis, valor_cofins,
			 cst_icms, cst_ipi, cst_pis, cst_cofins, origem_mercadoria, description)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		 RETURNING id, created_at`,
		item.FiscalExitID, item.Sequence, item.ItemCode, item.Ncm, item.Cfop, item.Quantity, item.UnitPrice, item.TotalPrice,
		item.BaseICMS, item.AliqICMS, item.ValorICMS, item.ValorICMSDiferido,
		item.BaseIPI, item.AliqIPI, item.ValorIPI, item.ValorPIS, item.ValorCOFINS,
		item.CstICMS, item.CstIPI, item.CstPIS, item.CstCOFINS, item.OrigemMercadoria, item.Description,
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
		        is_active, created_at, updated_at, created_by
		 FROM public.fiscal_exits WHERE id = $1`, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.DataEmissao, &e.DataSaida,
		&e.CnpjDestinatario, &e.RazaoSocialDestinatario, &e.IEDestinatario, &e.UFDestinatario,
		&e.Cfop, &e.NaturezaOperacao, &e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.SalesOrderCode, &e.Status, &e.Protocolo, &e.XmlPath, &e.DanfePath, &e.FocusRef,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy)
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
		        base_ipi, aliq_ipi, valor_ipi, valor_pis, valor_cofins,
		        cst_icms, cst_ipi, cst_pis, cst_cofins, origem_mercadoria, description, created_at
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
		        is_active, created_at, updated_at, created_by
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
		        is_active, created_at, updated_at, created_by
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
		           is_active, created_at, updated_at, created_by`,
		status, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.DataEmissao, &e.DataSaida,
		&e.CnpjDestinatario, &e.RazaoSocialDestinatario, &e.IEDestinatario, &e.UFDestinatario,
		&e.Cfop, &e.NaturezaOperacao, &e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.SalesOrderCode, &e.Status, &e.Protocolo, &e.XmlPath, &e.DanfePath, &e.FocusRef,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("updating fiscal exit status: %w", err)
	}
	return &e, nil
}

func (r *FiscalRepositoryPG) UpdateExitAuthorization(ctx context.Context, id int64, chaveAcesso, protocolo, focusRef string) (*entity.FiscalExit, error) {
	var e entity.FiscalExit
	err := r.pool.QueryRow(ctx,
		`UPDATE public.fiscal_exits SET
		     chave_acesso = $1, protocolo = $2, focus_ref = $3, status = 'AUTHORIZED', updated_at = NOW()
		 WHERE id = $4
		 RETURNING id, chave_acesso, numero_nf, serie, data_emissao, data_saida,
		           cnpj_destinatario, razao_social_destinatario, ie_destinatario, uf_destinatario,
		           cfop, natureza_operacao, valor_produtos, valor_frete, valor_seguro, valor_desconto,
		           valor_ipi, valor_icms, valor_pis, valor_cofins, valor_total,
		           sales_order_code, status, protocolo, xml_path, danfe_path, focus_ref,
		           is_active, created_at, updated_at, created_by`,
		chaveAcesso, protocolo, focusRef, id,
	).Scan(&e.ID, &e.ChaveAcesso, &e.NumeroNF, &e.Serie, &e.DataEmissao, &e.DataSaida,
		&e.CnpjDestinatario, &e.RazaoSocialDestinatario, &e.IEDestinatario, &e.UFDestinatario,
		&e.Cfop, &e.NaturezaOperacao, &e.ValorProdutos, &e.ValorFrete, &e.ValorSeguro, &e.ValorDesconto,
		&e.ValorIPI, &e.ValorICMS, &e.ValorPIS, &e.ValorCOFINS, &e.ValorTotal,
		&e.SalesOrderCode, &e.Status, &e.Protocolo, &e.XmlPath, &e.DanfePath, &e.FocusRef,
		&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy)
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
			&e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy,
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
			&it.BaseIPI, &it.AliqIPI, &it.ValorIPI, &it.ValorPIS, &it.ValorCOFINS,
			&it.CstICMS, &it.CstIPI, &it.CstPIS, &it.CstCOFINS, &it.OrigemMercadoria, &it.Description, &it.CreatedAt,
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
		        created_at, updated_at, updated_by
		 FROM public.fiscal_configs ORDER BY id LIMIT 1`,
	).Scan(&cfg.ID, &cfg.CnpjEmpresa, &cfg.RazaoSocial, &cfg.IEEmpresa, &cfg.RegimeTributario, &cfg.UFEmpresa,
		&cfg.IcmsInternoAliquota, &cfg.IcmsDiferimentoPercentual,
		&cfg.FocusNfeToken, &cfg.FocusNfeAmbiente, &cfg.JurosMes, &cfg.MultaAtraso,
		&cfg.VencimentoIcmsDia, &cfg.VencimentoIPIDia, &cfg.VencimentoPisCofinsDia,
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
	err := r.pool.QueryRow(ctx,
		`UPDATE public.fiscal_configs SET
		     cnpj_empresa = $1, razao_social = $2, ie_empresa = $3, regime_tributario = $4, uf_empresa = $5,
		     icms_interno_aliquota = $6, icms_diferimento_percentual = $7,
		     focus_nfe_token = $8, focus_nfe_ambiente = $9, juros_mes = $10, multa_atraso = $11,
		     vencimento_icms_dia = $12, vencimento_ipi_dia = $13, vencimento_pis_cofins_dia = $14,
		     updated_at = NOW(), updated_by = $15
		 WHERE id = $16
		 RETURNING id, cnpj_empresa, razao_social, ie_empresa, regime_tributario, uf_empresa,
		           icms_interno_aliquota, icms_diferimento_percentual,
		           focus_nfe_token, focus_nfe_ambiente, juros_mes, multa_atraso,
		           vencimento_icms_dia, vencimento_ipi_dia, vencimento_pis_cofins_dia,
		           created_at, updated_at, updated_by`,
		cfg.CnpjEmpresa, cfg.RazaoSocial, cfg.IEEmpresa, cfg.RegimeTributario, cfg.UFEmpresa,
		cfg.IcmsInternoAliquota, cfg.IcmsDiferimentoPercentual,
		cfg.FocusNfeToken, cfg.FocusNfeAmbiente, cfg.JurosMes, cfg.MultaAtraso,
		cfg.VencimentoIcmsDia, cfg.VencimentoIPIDia, cfg.VencimentoPisCofinsDia,
		cfg.UpdatedBy, cfg.ID,
	).Scan(&cfg.ID, &cfg.CnpjEmpresa, &cfg.RazaoSocial, &cfg.IEEmpresa, &cfg.RegimeTributario, &cfg.UFEmpresa,
		&cfg.IcmsInternoAliquota, &cfg.IcmsDiferimentoPercentual,
		&cfg.FocusNfeToken, &cfg.FocusNfeAmbiente, &cfg.JurosMes, &cfg.MultaAtraso,
		&cfg.VencimentoIcmsDia, &cfg.VencimentoIPIDia, &cfg.VencimentoPisCofinsDia,
		&cfg.CreatedAt, &cfg.UpdatedAt, &cfg.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("updating fiscal config: %w", err)
	}
	return cfg, nil
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
