package shipment

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var cem = decimal.NewFromInt(100)

// DadosDoFrete monta a cotação do romaneio: CNPJ/CEP da empresa (configuração
// fiscal), CNPJ/CEP do cliente do pedido (endereço de entrega ou principal), os
// volumes montados (cm → m) e a chave da NF-e (rastreio).
func (r *ShipmentRepositoryPG) DadosDoFrete(ctx context.Context, code int64) (*ports.PedidoDeCotacao, string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, "", err
	}
	codigo, err := tenant.Code(ctx)
	if err != nil {
		return nil, "", err
	}
	var id int64
	var chave *string
	p := &ports.PedidoDeCotacao{}
	err = r.pool.QueryRow(ctx, `
		SELECT s.id, s.nfe_key, COALESCE(fc.cnpj_empresa,''), COALESCE(fc.cep,''),
		       COALESCE(c.document_number,''), COALESCE(a.zip_code,''),
		       COALESCE((SELECT x.valor_total FROM fiscal_exits x WHERE x.id = s.fiscal_exit_id AND x.enterprise_id = s.enterprise_id),
		                (SELECT SUM(l.total_net_with_ipi) FROM sales_order_items l WHERE l.sales_order_code = so.code), 0)
		  FROM shipments s
		  LEFT JOIN fiscal_configs fc ON fc.enterprise_id = s.enterprise_id
		  LEFT JOIN sales_orders so ON so.code = s.sales_order_code AND so.enterprise_code = $3
		  LEFT JOIN customers c ON c.code = so.customer_code AND c.enterprise_id = s.enterprise_id
		  LEFT JOIN LATERAL (SELECT x.zip_code FROM customer_addresses x WHERE x.customer_id = c.id
		                      ORDER BY (x.id = so.shipping_address_code) DESC, x.is_default DESC, x.id LIMIT 1) a ON TRUE
		 WHERE s.code = $1 AND s.enterprise_id = $2`, code, empresa, codigo).
		Scan(&id, &chave, &p.CNPJEmbarcador, &p.CEPOrigem, &p.DocDestinatario, &p.CEPDestino, &p.ValorNota)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", errorsuc.NewNotFoundError(fmt.Sprintf("romaneio %d não encontrado", code))
	}
	if err != nil {
		return nil, "", fmt.Errorf("lendo o romaneio para o frete: %w", err)
	}
	p.DestinatarioPJ = len(p.DocDestinatario) > 0 && len(soDig(p.DocDestinatario)) == 14
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(length_cm,0), COALESCE(width_cm,0), COALESCE(height_cm,0), COALESCE(gross_weight,0)
		FROM shipment_volumes WHERE shipment_id = $1 ORDER BY volume_number`, id)
	if err != nil {
		return nil, "", fmt.Errorf("lendo os volumes do romaneio: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c, l, a, peso decimal.Decimal
		if err := rows.Scan(&c, &l, &a, &peso); err != nil {
			return nil, "", err
		}
		p.Volumes = append(p.Volumes, ports.VolumeDeFrete{Quantidade: 1, Comp: c.Div(cem), Largura: l.Div(cem), Altura: a.Div(cem), PesoKg: peso})
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if n := len(p.Volumes); n > 0 {
		for i := range p.Volumes {
			p.Volumes[i].ValorUnitario = p.ValorNota.Div(decimal.NewFromInt(int64(n))).Round(2)
		}
	}
	k := ""
	if chave != nil {
		k = *chave
	}
	return p, k, nil
}

func soDig(s string) string {
	var out []rune
	for _, r := range s {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return string(out)
}
