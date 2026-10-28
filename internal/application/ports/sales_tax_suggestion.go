package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

// SugestorTributosVenda sugere os impostos de uma linha de orçamento ou pedido
// com o mesmo cálculo da nota fiscal (regra de ICMS/IPI, regime, finalidade).
// Implementado por fiscal_uc.SugestorTributos.
type SugestorTributosVenda interface {
	SugerirTributos(ctx context.Context, q ConsultaTributosVenda) (*TributosSugeridos, error)
}

// ConsultaTributosVenda: o cliente (ou o pedido, de onde vem o cliente e a
// finalidade), o item e o valor da linha.
type ConsultaTributosVenda struct {
	CustomerCode   *int64
	SalesOrderCode *int64
	ItemCode       int64
	Quantidade     decimal.Decimal
	ValorUnitario  decimal.Decimal
	// Finalidade informada; nil = a do pedido, do cliente ou deduzida.
	Finalidade *string
}

// TributosSugeridos: percentuais (12 = 12%). STPct é o ICMS-ST efetivo sobre o
// valor dos produtos — é como o orçamento e o pedido guardam a ST.
type TributosSugeridos struct {
	IPIPct, ICMSPct, PISPct, COFINSPct, STPct decimal.Decimal
	// RegraICMSIPI: a regra do cadastro que valeu (nil = regra geral).
	RegraICMSIPI *int64
}
