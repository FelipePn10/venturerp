package financial

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// Títulos a receber da NFS-e (contas_receber.nfse_id, migration 000486).

// TitulosDaNFSe: os títulos ativos da NFS-e, para conferir recebimento antes
// de cancelar a nota.
func (r *FinancialRepositoryPG) TitulosDaNFSe(ctx context.Context, nfseID int64) ([]*entity.ContaReceber, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, numero_documento, valor_bruto, valor_recebido, status
		   FROM contas_receber
		  WHERE nfse_id = $1 AND enterprise_id = $2 AND is_active`, nfseID, empresa)
	if err != nil {
		return nil, fmt.Errorf("lendo os títulos da NFS-e %d: %w", nfseID, err)
	}
	defer rows.Close()
	var out []*entity.ContaReceber
	for rows.Next() {
		var t entity.ContaReceber
		var status string
		if err := rows.Scan(&t.ID, &t.NumeroDocumento, &t.ValorBruto, &t.ValorRecebido, &status); err != nil {
			return nil, err
		}
		t.Status = entity.ContaReceberStatus(status)
		out = append(out, &t)
	}
	return out, rows.Err()
}

// GerarTituloDaNFSe grava o título da NFS-e uma vez só: com um título ativo
// da nota já gravado, não faz nada (devolve false).
func (r *FinancialRepositoryPG) GerarTituloDaNFSe(ctx context.Context, nfseID int64, c *entity.ContaReceber) (bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO contas_receber
			(numero_documento, cliente_id, nfse_id, sales_order_id,
			 data_lancamento, data_emissao, data_vencimento,
			 valor_bruto, desconto, juros, multa, valor_recebido,
			 parcela_numero, parcela_total, forma_pagamento,
			 status, is_active, criado_por, enterprise_id)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,0,0,0,0,1,1,$9,$10,true,$11,$12
		  WHERE NOT EXISTS (SELECT 1 FROM contas_receber
		                     WHERE nfse_id = $3 AND enterprise_id = $12 AND is_active)`,
		c.NumeroDocumento, c.ClienteID, nfseID, c.SalesOrderID,
		c.DataLancamento, c.DataEmissao, c.DataVencimento,
		c.ValorBruto, c.FormaPagamento,
		string(entity.ContaReceberStatusPendente), c.CriadoPor, empresa)
	if err != nil {
		return false, fmt.Errorf("gerando o título da NFS-e %d: %w", nfseID, err)
	}
	return tag.RowsAffected() > 0, nil
}

// CancelarTitulosDaNFSe cancela os títulos em aberto da NFS-e cancelada.
func (r *FinancialRepositoryPG) CancelarTitulosDaNFSe(ctx context.Context, nfseID int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx,
		`UPDATE contas_receber SET status = 'CANCELADO', is_active = false, updated_at = NOW()
		  WHERE nfse_id = $1 AND status IN ('PENDENTE', 'APROVADO', 'VENCIDO') AND enterprise_id = $2`,
		nfseID, empresa); err != nil {
		return fmt.Errorf("cancelando os títulos da NFS-e %d: %w", nfseID, err)
	}
	return nil
}
