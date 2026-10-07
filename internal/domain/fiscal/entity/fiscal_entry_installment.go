package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Origem da parcela da nota de entrada.
const (
	ParcelaOrigemXML      = "XML"      // duplicata (<cobr><dup>) do próprio XML
	ParcelaOrigemCondicao = "CONDICAO" // condição de pagamento
	ParcelaOrigemManual   = "MANUAL"   // informada/ajustada pelo usuário
	ParcelaOrigemPadrao   = "PADRAO"   // nota sem duplicata: uma parcela só
)

// FiscalEntryInstallment é uma parcela (duplicata) da nota de entrada. Na
// aprovação cada parcela vira um título do contas a pagar — o boleto que o
// banco paga inteiro — com a distribuição por plano de contas como rateio.
type FiscalEntryInstallment struct {
	ID             int64
	FiscalEntryID  int64
	Numero         int
	Documento      *string
	DataVencimento time.Time
	Valor          decimal.Decimal
	FormaPagamento *string
	Origem         string
	ContaPagarID   *int64
	Distribuicao   []InstallmentAllocation
}

// InstallmentAllocation é quanto da parcela vai para um plano de contas (e,
// opcionalmente, um centro de custo).
type InstallmentAllocation struct {
	ID            int64
	PlanoContasID int64
	CentroCustoID *int64
	Valor         decimal.Decimal
}

// ChaveConta identifica o destino financeiro de um valor: plano de contas e
// centro de custo (0 = sem centro de custo).
type ChaveConta struct {
	PlanoContasID int64
	CentroCustoID int64
}

func (a InstallmentAllocation) Chave() ChaveConta {
	cc := int64(0)
	if a.CentroCustoID != nil {
		cc = *a.CentroCustoID
	}
	return ChaveConta{PlanoContasID: a.PlanoContasID, CentroCustoID: cc}
}

func (k ChaveConta) CentroCusto() *int64 {
	if k.CentroCustoID == 0 {
		return nil
	}
	v := k.CentroCustoID
	return &v
}
