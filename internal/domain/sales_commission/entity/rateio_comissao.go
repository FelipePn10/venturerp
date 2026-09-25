// Package entity guarda o rateio de comissão de um documento de venda.
//
// A capa do pedido e do orçamento tem UM representante e UM percentual. Na
// prática a venda é dividida: o representante da região e o parceiro que trouxe
// o cliente, cada um com a sua taxa, e a taxa muda de pedido para pedido
// (campanha, venda casada, cliente novo). É o que os ERPs grandes fazem — no
// Protheus pelos pares SC5_VEND1..5/SC5_COMIS1..5, no SAP por funções de
// parceiro, no FoccoERP pelo rateio de comissão.
package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// MaxRepresentantesPorDocumento é o mesmo limite do Protheus: cinco pares
// vendedor/comissão. Não é uma trava técnica, é o que o relatório de comissão
// consegue apresentar sem virar planilha.
const MaxRepresentantesPorDocumento = 5

type Papel string

const (
	PapelPrincipal Papel = "PRINCIPAL"
	PapelParceiro  Papel = "PARCEIRO"
)

func (p Papel) Valido() bool {
	return p == PapelPrincipal || p == PapelParceiro
}

func (p Papel) Rotulo() string {
	switch p {
	case PapelPrincipal:
		return "Representante principal"
	case PapelParceiro:
		return "Parceiro"
	}
	return string(p)
}

// Base é sobre o que a comissão incide. Comissão sobre o total COM IPI paga o
// representante por imposto que a empresa só repassa; ter a base gravada no
// documento é o que permite auditar a conta depois.
type Base string

const (
	BaseTotalProdutos Base = "TOTAL_PRODUTOS"
	BaseTotalLiquido  Base = "TOTAL_LIQUIDO"
)

func (b Base) Valido() bool {
	return b == BaseTotalProdutos || b == BaseTotalLiquido
}

func (b Base) Rotulo() string {
	switch b {
	case BaseTotalProdutos:
		return "Total dos produtos"
	case BaseTotalLiquido:
		return "Total líquido do documento"
	}
	return string(b)
}

// Documento diz de qual tabela o rateio é lido: o pedido e o orçamento têm o
// mesmo desenho, então o mesmo código serve os dois.
type Documento string

const (
	DocumentoPedido    Documento = "PEDIDO"
	DocumentoOrcamento Documento = "ORCAMENTO"
)

func (d Documento) Valido() bool {
	return d == DocumentoPedido || d == DocumentoOrcamento
}

type Rateio struct {
	ID                 int64
	EnterpriseCode     int64
	DocumentCode       int64
	RepresentativeCode int64
	// Nome vem da junção com representatives: é leitura, nunca é gravado.
	RepresentativeName string
	Role               Papel
	CommissionPct      decimal.Decimal
	CommissionBase     Base
	Notes              *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (r *Rateio) Normalizar() {
	if strings.TrimSpace(string(r.Role)) == "" {
		r.Role = PapelPrincipal
	}
	if strings.TrimSpace(string(r.CommissionBase)) == "" {
		r.CommissionBase = BaseTotalProdutos
	}
	r.Role = Papel(strings.ToUpper(strings.TrimSpace(string(r.Role))))
	r.CommissionBase = Base(strings.ToUpper(strings.TrimSpace(string(r.CommissionBase))))
	if r.Notes != nil {
		limpo := strings.TrimSpace(*r.Notes)
		if limpo == "" {
			r.Notes = nil
		} else {
			r.Notes = &limpo
		}
	}
}

func (r *Rateio) Validar() error {
	if r.RepresentativeCode <= 0 {
		return errors.New("informe o representante da comissão")
	}
	if !r.Role.Valido() {
		return fmt.Errorf("papel de comissão inválido: %s", r.Role)
	}
	if !r.CommissionBase.Valido() {
		return fmt.Errorf("base de comissão inválida: %s", r.CommissionBase)
	}
	if r.CommissionPct.IsNegative() || r.CommissionPct.GreaterThan(decimal.NewFromInt(100)) {
		return errors.New("o percentual de comissão deve estar entre 0 e 100")
	}
	return nil
}

// ValidarRateio olha o conjunto: é aqui que o erro do usuário aparece.
// Exatamente um principal, sem representante repetido, e a soma dos percentuais
// não pode passar de 100 — comissão somando mais que a venda é sempre digitação.
func ValidarRateio(linhas []*Rateio) error {
	if len(linhas) == 0 {
		return nil
	}
	if len(linhas) > MaxRepresentantesPorDocumento {
		return fmt.Errorf("são permitidos no máximo %d representantes por documento", MaxRepresentantesPorDocumento)
	}
	principais := 0
	vistos := map[int64]bool{}
	soma := decimal.Zero
	for _, l := range linhas {
		if err := l.Validar(); err != nil {
			return err
		}
		if vistos[l.RepresentativeCode] {
			return fmt.Errorf("o representante %d aparece mais de uma vez no rateio", l.RepresentativeCode)
		}
		vistos[l.RepresentativeCode] = true
		if l.Role == PapelPrincipal {
			principais++
		}
		soma = soma.Add(l.CommissionPct)
	}
	if principais == 0 {
		return errors.New("marque um representante como principal")
	}
	if principais > 1 {
		return errors.New("só pode haver um representante principal no documento")
	}
	if soma.GreaterThan(decimal.NewFromInt(100)) {
		return fmt.Errorf("a soma das comissões (%s%%) passa de 100%%", soma.StringFixed(4))
	}
	return nil
}

// Valor aplica o percentual sobre a base gravada na própria linha.
func (r *Rateio) Valor(totalProdutos, totalLiquido decimal.Decimal) decimal.Decimal {
	base := totalProdutos
	if r.CommissionBase == BaseTotalLiquido {
		base = totalLiquido
	}
	return base.Mul(r.CommissionPct).Div(decimal.NewFromInt(100)).Round(2)
}

// Principal devolve a linha principal do rateio: é ela que continua espelhada
// na capa do documento, para nenhum relatório antigo mudar de resposta.
func Principal(linhas []*Rateio) *Rateio {
	for _, l := range linhas {
		if l.Role == PapelPrincipal {
			return l
		}
	}
	return nil
}
