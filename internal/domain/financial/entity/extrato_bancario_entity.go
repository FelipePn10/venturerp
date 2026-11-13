package entity

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// ExtratoTipo é o sentido do lançamento como o BANCO o informa no OFX.
type ExtratoTipo string

const (
	ExtratoTipoCredito ExtratoTipo = "CREDIT"
	ExtratoTipoDebito  ExtratoTipo = "DEBIT"
)

// ExtratoBancario é uma linha do extrato importado (OFX) de uma conta bancária.
//
// O extrato é o que o banco diz que aconteceu; o fluxo de caixa é o que o sistema
// registrou. Conciliar é casar uma linha com um movimento — e a linha só fica
// conciliada quando esse movimento existe e confere.
type ExtratoBancario struct {
	ID              int64
	ContaBancariaID int64
	Data            time.Time
	Valor           decimal.Decimal // sempre positivo; o sentido está em Tipo
	Tipo            ExtratoTipo
	Descricao       string
	FitID           string
	Hash            string
	Conciliado      bool
	FluxoCaixaID    *int64
}

// TipoNoCaixa traduz o sentido do banco para o do fluxo de caixa: crédito na
// conta é entrada, débito é saída.
func (e ExtratoBancario) TipoNoCaixa() FluxoCaixaTipo {
	if e.Tipo == ExtratoTipoDebito {
		return FluxoCaixaTipoSaida
	}
	return FluxoCaixaTipoEntrada
}

// MotivoParaNaoConciliar devolve vazio quando a linha do extrato pode ser casada
// com o movimento, ou o motivo pronto para a tela quando não pode.
//
// As regras são as que fazem a conciliação significar alguma coisa: o movimento
// é da MESMA conta (casar o extrato de uma conta com o pagamento de outra é o erro
// mais caro desta rotina), tem o mesmo sentido e o mesmo valor, e nenhum dos dois
// já está casado com outra coisa. A data não é regra: boleto e TED compensam dias
// depois do registro, e quem confere a diferença é a pessoa, que vê as duas datas.
func (e ExtratoBancario) MotivoParaNaoConciliar(f FluxoCaixa) string {
	if e.Conciliado {
		return "este lançamento do extrato já está conciliado; desfaça a conciliação antes de casá-lo com outro movimento"
	}
	if f.Conciliado {
		return "o movimento escolhido já está conciliado com outro lançamento do extrato"
	}
	if f.ContaBancariaID == nil || *f.ContaBancariaID != e.ContaBancariaID {
		return "o movimento escolhido é de outra conta bancária"
	}
	if f.Tipo != e.TipoNoCaixa() {
		if e.Tipo == ExtratoTipoDebito {
			return "o lançamento do extrato é uma saída (débito) e o movimento escolhido é uma entrada"
		}
		return "o lançamento do extrato é uma entrada (crédito) e o movimento escolhido é uma saída"
	}
	if !e.Valor.Round(2).Equal(f.Valor.Round(2)) {
		return fmt.Sprintf("o valor do extrato (%s) é diferente do valor do movimento (%s)",
			e.Valor.StringFixed(2), f.Valor.StringFixed(2))
	}
	return ""
}

// MotivoParaNaoLancar devolve vazio quando a linha pode virar um movimento novo no
// caixa (tarifa, IOF, aplicação, resgate — o que não tem título por trás).
func (e ExtratoBancario) MotivoParaNaoLancar() string {
	if e.Conciliado {
		return "este lançamento do extrato já está conciliado"
	}
	if !e.Valor.IsPositive() {
		return "lançamento sem valor não movimenta o caixa"
	}
	return ""
}

// MovimentoLancadoPeloExtrato diz se o movimento do caixa nasceu desta linha do
// extrato (pela ação "lançar no caixa"), e não de uma baixa de título. Desfazer a
// conciliação de um movimento assim desfaz também o movimento — ele não existiria
// sem a linha.
func (e ExtratoBancario) MovimentoLancadoPeloExtrato(f FluxoCaixa) bool {
	return f.ExtratoHash != nil && *f.ExtratoHash == e.Hash &&
		f.ContasPagarID == nil && f.ContasReceberID == nil
}
