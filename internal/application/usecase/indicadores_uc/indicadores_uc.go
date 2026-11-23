// Package indicadores_uc: indicadores prontos (melhoria 12) — margem por produto
// e por cliente, OEE por máquina, giro de estoque, pontualidade de fornecedor e
// acerto da promessa de entrega. Cada bloco aparece para o perfil que lê o
// módulo dele.
package indicadores_uc

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
)

// Margem é a receita × CMV de um produto ou cliente nas NF-e autorizadas.
type Margem struct {
	Code       int64           `json:"code"`
	Nome       string          `json:"name"`
	Receita    decimal.Decimal `json:"revenue"`
	CMV        decimal.Decimal `json:"cogs"`
	Margem     decimal.Decimal `json:"margin"`
	Percentual decimal.Decimal `json:"margin_pct"`
}

// OEE de uma máquina: disponibilidade × desempenho × qualidade. Desempenho vem
// da hora padrão das etapas apontadas; sem etapa/tempo previsto fica nulo e o
// OEE é disponibilidade × qualidade.
type OEE struct {
	MachineID        int64            `json:"machine_id"`
	Codigo           int64            `json:"code"`
	Nome             string           `json:"name"`
	HorasApontadas   decimal.Decimal  `json:"worked_hours"`
	HorasDisponiveis decimal.Decimal  `json:"available_hours"`
	Boas             decimal.Decimal  `json:"good_qty"`
	Refugo           decimal.Decimal  `json:"scrap_qty"`
	Disponibilidade  decimal.Decimal  `json:"availability_pct"`
	Qualidade        decimal.Decimal  `json:"quality_pct"`
	Desempenho       *decimal.Decimal `json:"performance_pct,omitempty"`
	OEE              decimal.Decimal  `json:"oee_pct"`
}

// Giro do estoque no período.
type Giro struct {
	CMV            decimal.Decimal `json:"cogs"`
	ValorEstoque   decimal.Decimal `json:"stock_value"`
	GiroAnualizado decimal.Decimal `json:"annual_turnover"`
	CoberturaDias  decimal.Decimal `json:"coverage_days"`
}

// Pontualidade de um fornecedor (ou o acerto da promessa ao cliente).
type Pontualidade struct {
	Code        int64           `json:"code"`
	Nome        string          `json:"name"`
	Linhas      int             `json:"lines"`
	NoPrazo     int             `json:"on_time"`
	Percentual  decimal.Decimal `json:"on_time_pct"`
	AtrasoMedio decimal.Decimal `json:"avg_delay_days"`
}

// Indicadores do período.
type Indicadores struct {
	De  string `json:"from"`
	Ate string `json:"to"`
	// Vendas: receita e CMV das notas do período, no total (o detalhe por item e
	// cliente é a apuração da margem de contribuição, VCUS0200).
	Vendas          *Margem        `json:"sales,omitempty"`
	OEE             []OEE          `json:"oee,omitempty"`
	Giro            *Giro          `json:"inventory_turnover,omitempty"`
	Fornecedores    []Pontualidade `json:"supplier_on_time,omitempty"`
	PromessaEntrega *Pontualidade  `json:"delivery_promise,omitempty"`
}

type Repo interface {
	Vendas(ctx context.Context, de, ate time.Time) (*Margem, error)
	OEE(ctx context.Context, de, ate time.Time) ([]OEE, error)
	Giro(ctx context.Context, de, ate time.Time) (*Giro, error)
	PontualidadeFornecedores(ctx context.Context, de, ate time.Time) ([]Pontualidade, error)
	AcertoDaPromessa(ctx context.Context, de, ate time.Time) (*Pontualidade, error)
}

type Acesso interface {
	PodeNoModulo(ctx context.Context, m acesso.Modulo, a acesso.Acao) (bool, error)
}

type UseCase struct {
	Repo   Repo
	Acesso Acesso // opcional
}

var cem = decimal.NewFromInt(100)

// Pct é parte/total em %, com 2 casas (0 quando o total é zero).
func Pct(parte, total decimal.Decimal) decimal.Decimal {
	if total.IsZero() {
		return decimal.Zero
	}
	return parte.Div(total).Mul(cem).Round(2)
}

func (uc *UseCase) pode(ctx context.Context, m acesso.Modulo) (bool, error) {
	if uc.Acesso == nil {
		return true, nil
	}
	return uc.Acesso.PodeNoModulo(ctx, m, acesso.Ler)
}

// Calcular devolve os indicadores do período que o perfil pode ver.
func (uc *UseCase) Calcular(ctx context.Context, de, ate time.Time) (*Indicadores, error) {
	if ate.Before(de) {
		return nil, errorsuc.NewValidationError("o fim do período é anterior ao início")
	}
	out := &Indicadores{De: de.Format("2006-01-02"), Ate: ate.Format("2006-01-02")}
	blocos := []struct {
		modulo acesso.Modulo
		rodar  func() error
	}{
		{acesso.Comercial, func() (err error) {
			if out.Vendas, err = uc.Repo.Vendas(ctx, de, ate); err != nil {
				return err
			}
			out.PromessaEntrega, err = uc.Repo.AcertoDaPromessa(ctx, de, ate)
			return err
		}},
		{acesso.Producao, func() (err error) { out.OEE, err = uc.Repo.OEE(ctx, de, ate); return err }},
		{acesso.Estoque, func() (err error) { out.Giro, err = uc.Repo.Giro(ctx, de, ate); return err }},
		{acesso.Suprimentos, func() (err error) { out.Fornecedores, err = uc.Repo.PontualidadeFornecedores(ctx, de, ate); return err }},
	}
	for _, b := range blocos {
		ok, err := uc.pode(ctx, b.modulo)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if err := b.rodar(); err != nil {
			return nil, err
		}
	}
	if out.Vendas != nil {
		completarMargem(out.Vendas)
	}
	for i := range out.OEE {
		completarOEE(&out.OEE[i])
	}
	return out, nil
}

func completarMargem(m *Margem) {
	m.Margem = m.Receita.Sub(m.CMV)
	m.Percentual = Pct(m.Margem, m.Receita)
}

func completarOEE(o *OEE) {
	o.Disponibilidade = decimal.Min(Pct(o.HorasApontadas, o.HorasDisponiveis), cem)
	o.Qualidade = Pct(o.Boas, o.Boas.Add(o.Refugo))
	oee := o.Disponibilidade.Mul(o.Qualidade).Div(cem)
	if o.Desempenho != nil {
		oee = oee.Mul(*o.Desempenho).Div(cem)
	}
	o.OEE = oee.Round(2)
}
