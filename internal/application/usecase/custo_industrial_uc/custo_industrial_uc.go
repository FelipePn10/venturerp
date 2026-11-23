// Package custo_industrial_uc: o que faltava do custo industrial (melhoria 5) —
// a variação padrão × real consolidada, a margem do pedido de venda e o
// custo-padrão apurado de todos os itens de uma vez. Taxas por centro, rateio
// de indiretos e roll-up por item já existiam (cost_uc).
package custo_industrial_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// VariacaoDaOF é a apuração de uma OF (production_order_costs).
type VariacaoDaOF struct {
	ProductionOrderID int64           `json:"production_order_id"`
	OrderNumber       int64           `json:"order_number"`
	ItemCode          int64           `json:"item_code"`
	ItemName          string          `json:"item_name"`
	ProducedQty       decimal.Decimal `json:"produced_qty"`
	Padrao            decimal.Decimal `json:"standard_cost"`
	Real              decimal.Decimal `json:"actual_cost"`
	Material          decimal.Decimal `json:"material_variance"`
	MaoDeObra         decimal.Decimal `json:"labor_variance"`
	Maquina           decimal.Decimal `json:"machine_variance"`
	Indiretos         decimal.Decimal `json:"overhead_variance"`
	Terceiros         decimal.Decimal `json:"subcontract_variance"`
	Total             decimal.Decimal `json:"total_variance"`
	Percentual        decimal.Decimal `json:"variance_pct"`
	ApuradaEm         time.Time       `json:"settled_at"`
}

// Variacoes é o consolidado do período.
type Variacoes struct {
	De, Ate time.Time       `json:"-"`
	Ordens  []VariacaoDaOF  `json:"orders"`
	Padrao  decimal.Decimal `json:"standard_cost"`
	Real    decimal.Decimal `json:"actual_cost"`
	Total   decimal.Decimal `json:"total_variance"`
	// Desfavoravel: soma das OFs que custaram mais que o padrão.
	Desfavoravel decimal.Decimal `json:"unfavorable_variance"`
	Percentual   decimal.Decimal `json:"variance_pct"`
}

// LinhaDaMargem é uma linha do pedido.
type LinhaDaMargem struct {
	Linha        int64           `json:"line_code"`
	ItemCode     int64           `json:"item_code"`
	ItemName     string          `json:"item_name"`
	Quantidade   decimal.Decimal `json:"quantity"`
	Receita      decimal.Decimal `json:"revenue"`
	CustoPadrao  decimal.Decimal `json:"standard_cost"`
	CustoReal    decimal.Decimal `json:"actual_cost"`
	Faturado     decimal.Decimal `json:"invoiced_qty"`
	MargemPadrao decimal.Decimal `json:"standard_margin"`
	MargemReal   decimal.Decimal `json:"actual_margin"`
	PctPadrao    decimal.Decimal `json:"standard_margin_pct"`
	PctReal      decimal.Decimal `json:"actual_margin_pct"`
	Aviso        string          `json:"warning,omitempty"`
}

// MargemDoPedido: receita (líquida, sem IPI/ST) × custo-padrão do que foi
// pedido e × custo real (CMV) do que a NF-e autorizada já baixou.
type MargemDoPedido struct {
	SalesOrderCode int64           `json:"sales_order_code"`
	Linhas         []LinhaDaMargem `json:"lines"`
	Receita        decimal.Decimal `json:"revenue"`
	CustoPadrao    decimal.Decimal `json:"standard_cost"`
	CustoReal      decimal.Decimal `json:"actual_cost"`
	MargemPadrao   decimal.Decimal `json:"standard_margin"`
	PctPadrao      decimal.Decimal `json:"standard_margin_pct"`
	// MargemReal e PctReal valem para a parte faturada (receita proporcional).
	MargemReal decimal.Decimal `json:"actual_margin"`
	PctReal    decimal.Decimal `json:"actual_margin_pct"`
}

// LinhaDoPedidoCusto é o que o repositório lê de cada linha.
type LinhaDoPedidoCusto struct {
	Linha, ItemCode        int64
	ItemName               string
	Quantidade, Receita    decimal.Decimal
	CustoPadraoUnit        decimal.Decimal
	TemPadrao              bool
	FaturadoQtd, CustoReal decimal.Decimal
}

type Repo interface {
	Variacoes(ctx context.Context, de, ate time.Time) ([]VariacaoDaOF, error)
	LinhasDoPedido(ctx context.Context, pedido int64) ([]LinhaDoPedidoCusto, error)
	ItensParaApurar(ctx context.Context) ([]int64, error)
}

// Apurador é o roll-up do custo-padrão de um item (cost_uc).
type Apurador interface {
	RollUp(ctx context.Context, dto request.CostRollupDTO) (*response.CostRollupResponse, error)
}

type UseCase struct {
	Repo     Repo
	Apurador Apurador
	Auth     ports.AuthService
}

var cem = decimal.NewFromInt(100)

func pct(parte, total decimal.Decimal) decimal.Decimal {
	if total.IsZero() {
		return decimal.Zero
	}
	return parte.Div(total).Mul(cem).Round(2)
}

// VariacoesDoPeriodo consolida a variação das OFs apuradas entre de e ate.
func (uc *UseCase) VariacoesDoPeriodo(ctx context.Context, de, ate time.Time) (*Variacoes, error) {
	if ate.Before(de) {
		return nil, errorsuc.NewValidationError("o fim do período é anterior ao início")
	}
	lista, err := uc.Repo.Variacoes(ctx, de, ate)
	if err != nil {
		return nil, err
	}
	out := &Variacoes{De: de, Ate: ate, Ordens: lista}
	if out.Ordens == nil {
		out.Ordens = []VariacaoDaOF{}
	}
	for i := range out.Ordens {
		o := &out.Ordens[i]
		o.Percentual = pct(o.Total, o.Padrao)
		out.Padrao, out.Real, out.Total = out.Padrao.Add(o.Padrao), out.Real.Add(o.Real), out.Total.Add(o.Total)
		if o.Total.IsPositive() {
			out.Desfavoravel = out.Desfavoravel.Add(o.Total)
		}
	}
	out.Percentual = pct(out.Total, out.Padrao)
	return out, nil
}

// Margem calcula a margem do pedido de venda.
func (uc *UseCase) Margem(ctx context.Context, pedido int64) (*MargemDoPedido, error) {
	linhas, err := uc.Repo.LinhasDoPedido(ctx, pedido)
	if err != nil {
		return nil, err
	}
	if len(linhas) == 0 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("pedido %d sem linhas nesta empresa", pedido))
	}
	out := &MargemDoPedido{SalesOrderCode: pedido}
	receitaFaturada := decimal.Zero
	for _, l := range linhas {
		m := LinhaDaMargem{Linha: l.Linha, ItemCode: l.ItemCode, ItemName: l.ItemName, Quantidade: l.Quantidade, Receita: l.Receita.Round(2),
			CustoPadrao: l.CustoPadraoUnit.Mul(l.Quantidade).Round(2), CustoReal: l.CustoReal.Round(2), Faturado: l.FaturadoQtd}
		if !l.TemPadrao {
			m.Aviso = "item sem custo-padrão: apure em VCUS0100 (aba Apuração do custo)"
		}
		m.MargemPadrao = m.Receita.Sub(m.CustoPadrao)
		m.PctPadrao = pct(m.MargemPadrao, m.Receita)
		if l.FaturadoQtd.IsPositive() && l.Quantidade.IsPositive() {
			parte := m.Receita.Mul(decimal.Min(l.FaturadoQtd, l.Quantidade)).Div(l.Quantidade).Round(2)
			m.MargemReal = parte.Sub(m.CustoReal)
			m.PctReal = pct(m.MargemReal, parte)
			receitaFaturada = receitaFaturada.Add(parte)
		}
		out.Receita, out.CustoPadrao, out.CustoReal = out.Receita.Add(m.Receita), out.CustoPadrao.Add(m.CustoPadrao), out.CustoReal.Add(m.CustoReal)
		out.Linhas = append(out.Linhas, m)
	}
	out.MargemPadrao = out.Receita.Sub(out.CustoPadrao)
	out.PctPadrao = pct(out.MargemPadrao, out.Receita)
	if receitaFaturada.IsPositive() {
		out.MargemReal = receitaFaturada.Sub(out.CustoReal)
		out.PctReal = pct(out.MargemReal, receitaFaturada)
	}
	return out, nil
}

// ResultadoApuracao resume o roll-up em lote.
type ResultadoApuracao struct {
	Apurados int      `json:"calculated"`
	Falhas   []string `json:"failures"`
}

// ApurarTodos roda o roll-up do custo-padrão de todos os itens fabricados e
// comprados da empresa. Falha de um item não para os outros: volta na lista.
func (uc *UseCase) ApurarTodos(ctx context.Context) (*ResultadoApuracao, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil || ator == uuid.Nil {
		return nil, errorsuc.ErrUnauthorized
	}
	itens, err := uc.Repo.ItensParaApurar(ctx)
	if err != nil {
		return nil, err
	}
	out := &ResultadoApuracao{Falhas: []string{}}
	for _, it := range itens {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if _, err := uc.Apurador.RollUp(ctx, request.CostRollupDTO{ItemCode: it, CalculatedBy: ator.String()}); err != nil {
			out.Falhas = append(out.Falhas, fmt.Sprintf("item %d: %v", it, err))
			continue
		}
		out.Apurados++
	}
	return out, nil
}
