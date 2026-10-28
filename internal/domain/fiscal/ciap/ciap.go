// Package ciap: o Controle de Crédito do ICMS do Ativo Permanente (LC 87/1996,
// art. 20, § 5º; EFD ICMS/IPI, Bloco G, Guia Prático). O ICMS do bem do ativo
// imobilizado é apropriado em parcelas mensais (48, salvo legislação da UF), a
// partir do mês da entrada; cada parcela é multiplicada pela participação das
// saídas tributadas e de exportação no total de saídas do mês. O bem que sai
// antes do fim (alienação, perecimento, outras) deixa de apropriar o restante.
package ciap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Tipos de movimentação do G125 que o sistema escritura.
const (
	MovSaldoInicial = "SI" // bem escriturado em mês anterior, ainda apropriando
	MovImobilizacao = "IM" // entrada do bem no mês
	MovFimPeriodo   = "BA" // baixa pelo fim do período de apropriação
	MovAlienacao    = "AT" // alienação ou transferência
	MovPerecimento  = "PE" // perecimento, extravio ou deterioração
	MovOutras       = "OT" // outras saídas do imobilizado
)

// TipoBaixaValido: as saídas do CIAP antes do fim do período.
func TipoBaixaValido(t string) bool {
	return t == MovAlienacao || t == MovPerecimento || t == MovOutras
}

// Documento da entrada do bem (G130/G140 no mês da imobilização).
type Documento struct {
	Modelo, Serie, Numero, Chave string
	Emissao                      time.Time
	CodPart                      string // participante (0150)
	NumItem                      int
	CodItem, Unidade             string // item (0200) e unidade do documento
	Quantidade                   decimal.Decimal
}

// Bem no CIAP.
type Bem struct {
	ID                                   int64
	Codigo                               string // COD_IND_BEM (0300)
	Descricao                            string
	Funcao                               string
	VidaUtilMeses                        int
	ContaCodigo, ContaNome               string // 0300/COD_CTA e 0500
	ContaNivel                           int
	CentroCodigo, CentroNome             string // 0305/COD_CCUS e 0600
	DataEntrada                          time.Time
	ICMSOp, ICMSST, ICMSFrete, ICMSDifal decimal.Decimal
	NumeroParcelas                       int
	Baixado                              bool
	DataBaixa                            *time.Time
	TipoBaixa                            string
	Documento                            *Documento
}

// Total: o crédito do bem (campos 05 a 08 do G125).
func (b Bem) Total() decimal.Decimal {
	return b.ICMSOp.Add(b.ICMSST).Add(b.ICMSFrete).Add(b.ICMSDifal)
}

// ValorParcela: o total dividido pelas parcelas, truncado em 2 casas — o PVA
// recusa parcela maior que o total ÷ NR_PARC.
func (b Bem) ValorParcela() decimal.Decimal {
	n := b.NumeroParcelas
	if n <= 0 {
		n = 48
	}
	return b.Total().Div(decimal.NewFromInt(int64(n))).Truncate(2)
}

// Conferir: código, descrição, entrada, crédito e parcelas.
func (b *Bem) Conferir() error {
	b.Codigo, b.Descricao = strings.TrimSpace(b.Codigo), strings.TrimSpace(b.Descricao)
	if b.Codigo == "" || len(b.Codigo) > 60 {
		return domainerr.NewValidation("informe o código do bem (até 60 caracteres), o do controle patrimonial")
	}
	if b.Descricao == "" {
		return domainerr.NewValidation("informe a descrição do bem")
	}
	if b.DataEntrada.IsZero() {
		return domainerr.NewValidation("informe a data de entrada do bem")
	}
	for _, v := range []decimal.Decimal{b.ICMSOp, b.ICMSST, b.ICMSFrete, b.ICMSDifal} {
		if v.IsNegative() {
			return domainerr.NewValidation("o ICMS do bem não pode ser negativo")
		}
	}
	if !b.Total().IsPositive() {
		return domainerr.NewValidation("o bem precisa de ICMS a apropriar (operação própria, ST, frete ou diferencial)")
	}
	if b.NumeroParcelas < 1 || b.NumeroParcelas > 999 {
		return domainerr.NewValidation("número de parcelas de 1 a 999 (48 pela LC 87/1996)")
	}
	return nil
}

func mesIndex(t time.Time) int { return t.Year()*12 + int(t.Month()) - 1 }

// Parcela: o número da parcela que o bem apropria no mês (0 = não apropria).
func (b Bem) Parcela(ano, mes int) int {
	n := ano*12 + mes - 1 - mesIndex(b.DataEntrada) + 1
	if n < 1 || n > b.NumeroParcelas {
		return 0
	}
	return n
}

// Movimento: uma linha do G125.
type Movimento struct {
	Bem           Bem
	TipoMov       string
	Data          time.Time
	NumeroParcela int
	ValorParcela  decimal.Decimal
	// ComValores: os campos 05 a 08 (ICMS do bem) são informados — nas linhas
	// de entrada (SI, IM); nas de saída (BA, AT, PE, OT), não.
	ComValores bool
}

// Apuracao do mês (G110) com os movimentos (G125).
type Apuracao struct {
	Ano, Mes       int
	Inicio, Fim    time.Time
	SaldoInicial   decimal.Decimal // SALDO_IN_ICMS
	SomaParcelas   decimal.Decimal // SOM_PARC
	SaidasTribExp  decimal.Decimal // VL_TRIB_EXP
	SaidasTotal    decimal.Decimal // VL_TOTAL
	Indice         decimal.Decimal // IND_PER_SAI (8 casas)
	ICMSApropriado decimal.Decimal // ICMS_APROP
	Movimentos     []Movimento
}

// Apurar monta o mês: cada bem ativo dentro das parcelas apropria a dele; o da
// última parcela sai com BA; o baixado no mês sai com SI (sem parcela — o
// tratamento conservador; a UF pode admitir a parcela do mês da saída) e o tipo
// da baixa. Saídas totais zero = índice zero (sem saída, sem crédito).
func Apurar(bens []Bem, ano, mes int, tribExp, total decimal.Decimal) (*Apuracao, error) {
	if mes < 1 || mes > 12 || ano < 2000 || ano > 2100 {
		return nil, domainerr.NewValidation("informe o mês e o ano da apuração")
	}
	if tribExp.IsNegative() || total.IsNegative() {
		return nil, domainerr.NewValidation("as saídas não podem ser negativas")
	}
	if tribExp.GreaterThan(total) {
		return nil, domainerr.NewValidation(fmt.Sprintf("as saídas tributadas e de exportação (%s) não podem passar do total de saídas (%s)", tribExp.StringFixed(2), total.StringFixed(2)))
	}
	inicio := time.Date(ano, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
	fim := inicio.AddDate(0, 1, -1)
	a := &Apuracao{Ano: ano, Mes: mes, Inicio: inicio, Fim: fim, SaidasTribExp: tribExp.Round(2), SaidasTotal: total.Round(2),
		SaldoInicial: decimal.Zero, SomaParcelas: decimal.Zero, Indice: decimal.Zero}
	if total.IsPositive() {
		a.Indice = tribExp.Div(total).Round(8)
	}
	mesAtual := ano*12 + mes - 1
	for _, b := range bens {
		entrada := mesIndex(b.DataEntrada)
		if entrada > mesAtual {
			continue // entra depois
		}
		anterior := entrada < mesAtual
		if b.Baixado && b.DataBaixa != nil {
			baixa := mesIndex(*b.DataBaixa)
			if baixa < mesAtual {
				continue // já saiu
			}
			if baixa == mesAtual {
				a.Movimentos = append(a.Movimentos,
					Movimento{Bem: b, TipoMov: MovSaldoInicial, Data: inicio, ComValores: true},
					Movimento{Bem: b, TipoMov: b.TipoBaixa, Data: *b.DataBaixa})
				if anterior {
					a.SaldoInicial = a.SaldoInicial.Add(b.Total())
				}
				continue
			}
		}
		n := b.Parcela(ano, mes)
		if n == 0 {
			continue // passou das parcelas (já saiu com BA)
		}
		v := b.ValorParcela()
		tipo, data := MovSaldoInicial, inicio
		if !anterior {
			tipo, data = MovImobilizacao, b.DataEntrada
		} else {
			a.SaldoInicial = a.SaldoInicial.Add(b.Total())
		}
		a.Movimentos = append(a.Movimentos, Movimento{Bem: b, TipoMov: tipo, Data: data, NumeroParcela: n, ValorParcela: v, ComValores: true})
		a.SomaParcelas = a.SomaParcelas.Add(v)
		if n == b.NumeroParcelas {
			a.Movimentos = append(a.Movimentos, Movimento{Bem: b, TipoMov: MovFimPeriodo, Data: fim})
		}
	}
	a.ICMSApropriado = a.SomaParcelas.Mul(a.Indice).Round(2)
	return a, nil
}

// Parametros do CIAP da empresa: o código de ajuste (tabela 5.1.1 da UF) com
// que o crédito do mês entra no E111, a conta do ativo padrão e as parcelas.
type Parametros struct {
	CodigoAjuste   string
	ContaAtivoID   *int64
	NumeroParcelas int
}

// BemCadastrado: o bem com o que a tela mostra além do domínio.
type BemCadastrado struct {
	Bem
	Origem         string // NF_ENTRADA ou MANUAL
	ContaAtivoID   *int64
	CostCenterID   *int64
	ParcelasUsadas int
}

// BemManual: o que a tela grava no bem.
type BemManual struct {
	Bem
	ContaAtivoID *int64
	CostCenterID *int64
}

// ApuracaoGravada do mês.
type ApuracaoGravada struct {
	Apuracao
	ID               int64
	SaidasInformadas bool
	Fechada          bool
	ApuradoEm        time.Time
}

// Repository do CIAP, na empresa da sessão.
type Repository interface {
	Parametros(ctx context.Context) (Parametros, error)
	SalvarParametros(ctx context.Context, p Parametros, por uuid.UUID) error
	// Sincronizar traz os bens das notas de entrada aprovadas e tira os das
	// canceladas (baixa, se já apropriaram).
	Sincronizar(ctx context.Context) error
	Bens(ctx context.Context) ([]BemCadastrado, error)
	Bem(ctx context.Context, id int64) (*BemCadastrado, error)
	CriarBem(ctx context.Context, b BemManual, por uuid.UUID) (int64, error)
	AtualizarBem(ctx context.Context, b BemManual, valores bool) error
	Baixar(ctx context.Context, id int64, tipo string, dia time.Time, motivo string) error
	ExcluirBem(ctx context.Context, id int64) error
	Saidas(ctx context.Context, ano, mes int) (tribExp, total decimal.Decimal, err error)
	Apuracao(ctx context.Context, ano, mes int) (*ApuracaoGravada, error)
	GravarApuracao(ctx context.Context, a *Apuracao, informadas bool, por uuid.UUID) error
	Fechar(ctx context.Context, ano, mes int, fechar bool) error
	ApuracoesDoAno(ctx context.Context, ano int) ([]ApuracaoGravada, error)
}
