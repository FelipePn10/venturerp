package fiscal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// CadastroTributarioPG lê do cadastro o que muda a tributação de uma venda: a
// finalidade da compra (do pedido ou do cliente, ou a da tributação deles
// quando ela é de consumidor) e os indicadores do tipo de nota.
//
// Clientes, tributações e tipos de nota são por empresa (enterprise_id); o
// pedido de venda é da convenção comercial (enterprise_code) — cada consulta
// usa a coluna da sua tabela.
type CadastroTributarioPG struct {
	pool *pgxpool.Pool
}

func NewCadastroTributarioPG(pool *pgxpool.Pool) *CadastroTributarioPG {
	return &CadastroTributarioPG{pool: pool}
}

func (r *CadastroTributarioPG) FinalidadeDoCliente(ctx context.Context, customerCode int64) (*string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var f *string
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(c.finalidade_compra, CASE WHEN t.is_consumer THEN 'USO_CONSUMO' END)
		  FROM customers c
		  LEFT JOIN tax_types t ON t.id = c.tax_type_id AND t.enterprise_id = c.enterprise_id
		 WHERE c.code = $1 AND c.enterprise_id = $2`, customerCode, empresa).Scan(&f)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a finalidade de compra do cliente: %w", err)
	}
	return f, nil
}

func (r *CadastroTributarioPG) FinalidadeDoPedido(ctx context.Context, salesOrderCode int64) (*string, error) {
	codigoEmpresa, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	var f *string
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(o.finalidade_compra, CASE WHEN t.is_consumer THEN 'USO_CONSUMO' END)
		  FROM sales_orders o
		  JOIN enterprise e ON e.code = o.enterprise_code
		  LEFT JOIN tax_types t ON t.code = o.tax_type_code AND t.enterprise_id = e.id
		 WHERE o.code = $1 AND o.enterprise_code = $2`, salesOrderCode, codigoEmpresa).Scan(&f)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a finalidade de compra do pedido: %w", err)
	}
	return f, nil
}

func (r *CadastroTributarioPG) TipoNota(ctx context.Context, id int64) (*entity.TipoNotaFiscal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var tn entity.TipoNotaFiscal
	var ativo bool
	err = r.pool.QueryRow(ctx, `
		SELECT id, code, description, type::text, icms_type::text, icms_reduction_pct::float8,
		       calc_substitution_tax, calc_icms_deferral, calc_pis_cofins, calc_difal, is_active
		  FROM invoice_types WHERE id = $1 AND enterprise_id = $2`, id, empresa).Scan(
		&tn.ID, &tn.Code, &tn.Descricao, &tn.Tipo, &tn.TipoICMS, &tn.ReducaoICMSPct,
		&tn.CalculaST, &tn.CalculaDiferimento, &tn.CalculaPISCOFINS, &tn.CalculaDIFAL, &ativo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o tipo de nota %d não existe nesta empresa", id))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo o tipo de nota: %w", err)
	}
	if !ativo {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o tipo de nota %d (%s) está inativo", tn.Code, tn.Descricao))
	}
	return &tn, nil
}
