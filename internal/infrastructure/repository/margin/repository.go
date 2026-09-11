// Package margin persiste os parâmetros e o resultado da margem de contribuição.
package margin

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/margin/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const colunasParametros = `ir_pct, admin_pct, freight_pct, financial_rate_monthly,
	avg_sales_term_days, avg_purchase_term_days, production_cycle_days,
	material_payment_days, labor_payment_days, ipi_payment_days,
	icms_payment_days, pis_payment_days, cofins_payment_days`

// UpsertParametros grava os parâmetros do mês.
func (r *Repository) UpsertParametros(ctx context.Context, p entity.Parametros) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO margin_parameters (enterprise_id, year, month, `+colunasParametros+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (enterprise_id, year, month) DO UPDATE SET
			ir_pct=EXCLUDED.ir_pct, admin_pct=EXCLUDED.admin_pct,
			freight_pct=EXCLUDED.freight_pct, financial_rate_monthly=EXCLUDED.financial_rate_monthly,
			avg_sales_term_days=EXCLUDED.avg_sales_term_days,
			avg_purchase_term_days=EXCLUDED.avg_purchase_term_days,
			production_cycle_days=EXCLUDED.production_cycle_days,
			material_payment_days=EXCLUDED.material_payment_days,
			labor_payment_days=EXCLUDED.labor_payment_days,
			ipi_payment_days=EXCLUDED.ipi_payment_days,
			icms_payment_days=EXCLUDED.icms_payment_days,
			pis_payment_days=EXCLUDED.pis_payment_days,
			cofins_payment_days=EXCLUDED.cofins_payment_days,
			updated_at=NOW()`,
		empresa, p.Ano, p.Mes, p.IRPct, p.AdminPct, p.FreightPct, p.FinancialRateMonthly,
		p.AvgSalesTermDays, p.AvgPurchaseTermDays, p.ProductionCycleDays,
		p.MaterialPaymentDays, p.LaborPaymentDays, p.IPIPaymentDays,
		p.ICMSPaymentDays, p.PISPaymentDays, p.COFINSPaymentDays)
	if err != nil {
		return fmt.Errorf("gravando parâmetros de margem: %w", err)
	}
	return nil
}

// GetParametros devolve os parâmetros do mês; nil quando não cadastrados.
func (r *Repository) GetParametros(ctx context.Context, ano, mes int) (*entity.Parametros, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	p := entity.Parametros{Ano: ano, Mes: mes}
	err = r.pool.QueryRow(ctx, `
		SELECT `+colunasParametros+`
		FROM margin_parameters WHERE enterprise_id=$1 AND year=$2 AND month=$3`,
		empresa, ano, mes).Scan(
		&p.IRPct, &p.AdminPct, &p.FreightPct, &p.FinancialRateMonthly,
		&p.AvgSalesTermDays, &p.AvgPurchaseTermDays, &p.ProductionCycleDays,
		&p.MaterialPaymentDays, &p.LaborPaymentDays, &p.IPIPaymentDays,
		&p.ICMSPaymentDays, &p.PISPaymentDays, &p.COFINSPaymentDays)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo parâmetros de margem: %w", err)
	}
	return &p, nil
}

// ListParametros devolve os meses já cadastrados de um ano.
func (r *Repository) ListParametros(ctx context.Context, ano int) ([]entity.Parametros, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT year, month, `+colunasParametros+`
		FROM margin_parameters WHERE enterprise_id=$1 AND year=$2 ORDER BY month`, empresa, ano)
	if err != nil {
		return nil, fmt.Errorf("listando parâmetros de margem: %w", err)
	}
	defer rows.Close()
	out := []entity.Parametros{}
	for rows.Next() {
		var p entity.Parametros
		if err := rows.Scan(&p.Ano, &p.Mes, &p.IRPct, &p.AdminPct, &p.FreightPct, &p.FinancialRateMonthly,
			&p.AvgSalesTermDays, &p.AvgPurchaseTermDays, &p.ProductionCycleDays,
			&p.MaterialPaymentDays, &p.LaborPaymentDays, &p.IPIPaymentDays,
			&p.ICMSPaymentDays, &p.PISPaymentDays, &p.COFINSPaymentDays); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// VendaApurada é uma linha de nota de saída com os custos já resolvidos.
type VendaApurada struct {
	SourceID   int64
	SourceItem int
	IssueDate  string
	ItemCode   *int64
	Venda      entity.Venda
}

// ListarVendasDoPeriodo traz as linhas das notas de saída emitidas no período,
// com o custo do item resolvido pela base escolhida.
//
// `MEDIO` usa o custo médio do estoque (o que a mercadoria custou de fato);
// `PADRAO` usa o custo padrão calculado pelo rollup. É a mesma escolha do
// parâmetro 91 do FoccoERP — e ela muda o número, por isso fica gravada junto
// com o resultado.
func (r *Repository) ListarVendasDoPeriodo(ctx context.Context, de, ate, base string) ([]VendaApurada, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}

	custo := `COALESCE(sc.material_cost, 0)`
	transformacao := `COALESCE(sc.labor_cost, 0) + COALESCE(sc.overhead_cost, 0)`
	if base == "MEDIO" {
		// Custo médio do estoque cobre a mercadoria inteira; não há separação
		// entre material e transformação, então tudo entra como material — é o
		// que o custo médio representa.
		custo = `COALESCE(sb.average_cost, sc.material_cost, 0)`
		transformacao = `CASE WHEN sb.average_cost IS NOT NULL THEN 0
		                      ELSE COALESCE(sc.labor_cost,0) + COALESCE(sc.overhead_cost,0) END`
	}

	rows, err := r.pool.Query(ctx, `
		SELECT fe.id, fei.sequence, fe.data_emissao::date::text, fei.item_code,
		       fei.quantity, fei.unit_price,
		       COALESCE(fei.valor_ipi,0), COALESCE(fei.valor_icms,0),
		       COALESCE(fei.valor_pis,0), COALESCE(fei.valor_cofins,0),
		       (`+custo+`) * fei.quantity,
		       (`+transformacao+`) * fei.quantity
		FROM fiscal_exits fe
		JOIN fiscal_exit_items fei ON fei.fiscal_exit_id = fe.id
		LEFT JOIN item_standard_costs sc ON sc.item_code = fei.item_code
		LEFT JOIN LATERAL (
		    SELECT AVG(NULLIF(b.avg_cost,0)) AS average_cost
		    FROM stock_balances b
		    WHERE b.item_code = fei.item_code AND b.enterprise_id = fe.enterprise_id
		) sb ON TRUE
		WHERE fe.enterprise_id = $1
		  AND fe.data_emissao::date BETWEEN $2::date AND $3::date
		ORDER BY fe.data_emissao, fe.id, fei.sequence`, empresa, de, ate)
	if err != nil {
		return nil, fmt.Errorf("listando vendas do período: %w", err)
	}
	defer rows.Close()

	out := []VendaApurada{}
	for rows.Next() {
		var v VendaApurada
		var seq int32
		if err := rows.Scan(&v.SourceID, &seq, &v.IssueDate, &v.ItemCode,
			&v.Venda.Quantidade, &v.Venda.ValorLiquido,
			&v.Venda.IPI, &v.Venda.ICMS, &v.Venda.PIS, &v.Venda.COFINS,
			&v.Venda.CustoMateriaPrima, &v.Venda.CustoTransformacao); err != nil {
			return nil, err
		}
		v.SourceItem = int(seq)
		out = append(out, v)
	}
	return out, rows.Err()
}

// GravarResultado grava a cascata de uma linha; recalcular sobrescreve.
func (r *Repository) GravarResultado(ctx context.Context, v VendaApurada, res entity.Resultado, base string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO contribution_margin (enterprise_id, source, source_id, source_item, issue_date,
			item_code, quantity, gross_revenue, ipi, merchandise_revenue, icms, pis_cofins,
			material_cost, conversion_cost, gross_profit, admin_expense, commission, freight,
			other_expense, financial_expense, income_tax_provision, margin, margin_pct, cost_basis)
		VALUES ($1,'NOTA',$2,$3,$4::date,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		ON CONFLICT (enterprise_id, source, source_id, source_item) DO UPDATE SET
			issue_date=EXCLUDED.issue_date, item_code=EXCLUDED.item_code, quantity=EXCLUDED.quantity,
			gross_revenue=EXCLUDED.gross_revenue, ipi=EXCLUDED.ipi,
			merchandise_revenue=EXCLUDED.merchandise_revenue, icms=EXCLUDED.icms,
			pis_cofins=EXCLUDED.pis_cofins, material_cost=EXCLUDED.material_cost,
			conversion_cost=EXCLUDED.conversion_cost, gross_profit=EXCLUDED.gross_profit,
			admin_expense=EXCLUDED.admin_expense, commission=EXCLUDED.commission,
			freight=EXCLUDED.freight, other_expense=EXCLUDED.other_expense,
			financial_expense=EXCLUDED.financial_expense,
			income_tax_provision=EXCLUDED.income_tax_provision, margin=EXCLUDED.margin,
			margin_pct=EXCLUDED.margin_pct, cost_basis=EXCLUDED.cost_basis, calculated_at=NOW()`,
		empresa, v.SourceID, v.SourceItem, v.IssueDate, v.ItemCode, v.Venda.Quantidade,
		res.FaturamentoBruto, res.IPI, res.FaturamentoMercadoria, res.ICMS, res.PISCOFINS,
		res.CustoMateriaPrima, res.CustoTransformacao, res.LucroBruto, res.DespesaAdministrativa,
		res.Comissao, res.Frete, res.Outros, res.DespesaFinanceira, res.ProvisaoIR,
		res.Margem, res.MargemPct, base)
	if err != nil {
		return fmt.Errorf("gravando margem da nota %d: %w", v.SourceID, err)
	}
	return nil
}

// LinhaApurada é uma linha do relatório de margem (FoccoERP FCST0320).
// As tags são o contrato da API: esta struct é devolvida direto pelo handler e,
// sem elas, nota, item e data chegavam em PascalCase — a tela mostrava "0/0" e
// data vazia porque procurava snake_case.
type LinhaApurada struct {
	SourceID   int64   `json:"source_id"`
	SourceItem int     `json:"source_item"`
	IssueDate  string  `json:"issue_date"`
	ItemCode   *int64  `json:"item_code,omitempty"`
	Quantity   float64 `json:"quantity"`
	CostBasis  string  `json:"cost_basis"`
	entity.Resultado
}

// ListarApuracao devolve a margem já calculada do período, ordenada por
// faturamento ou por margem — as duas ordenações que o FCST0320 oferece,
// porque respondem perguntas diferentes: "onde está o volume" e "onde está o
// dinheiro".
func (r *Repository) ListarApuracao(ctx context.Context, de, ate, ordem string, itemCode, customerCode *int64) ([]LinhaApurada, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ordenacao := "merchandise_revenue DESC"
	if ordem == "MARGEM" {
		ordenacao = "margin_pct ASC" // pior margem primeiro: é o que precisa de ação
	}
	rows, err := r.pool.Query(ctx, `
		SELECT source_id, source_item, issue_date::text, item_code, quantity, cost_basis,
		       gross_revenue, ipi, merchandise_revenue, icms, pis_cofins,
		       material_cost, conversion_cost, gross_profit, admin_expense, commission,
		       freight, other_expense, financial_expense, income_tax_provision, margin, margin_pct
		FROM contribution_margin
		WHERE enterprise_id=$1 AND issue_date BETWEEN $2::date AND $3::date
		  AND ($4::bigint IS NULL OR item_code = $4)
		  AND ($5::bigint IS NULL OR customer_code = $5)
		ORDER BY `+ordenacao, empresa, de, ate, itemCode, customerCode)
	if err != nil {
		return nil, fmt.Errorf("listando apuração de margem: %w", err)
	}
	defer rows.Close()
	out := []LinhaApurada{}
	for rows.Next() {
		var l LinhaApurada
		var seq int32
		if err := rows.Scan(&l.SourceID, &seq, &l.IssueDate, &l.ItemCode, &l.Quantity, &l.CostBasis,
			&l.FaturamentoBruto, &l.IPI, &l.FaturamentoMercadoria, &l.ICMS, &l.PISCOFINS,
			&l.CustoMateriaPrima, &l.CustoTransformacao, &l.LucroBruto, &l.DespesaAdministrativa,
			&l.Comissao, &l.Frete, &l.Outros, &l.DespesaFinanceira, &l.ProvisaoIR,
			&l.Margem, &l.MargemPct); err != nil {
			return nil, err
		}
		l.SourceItem = int(seq)
		out = append(out, l)
	}
	return out, rows.Err()
}
