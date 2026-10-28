// Package recuperacao: cobrança do custo da garantia ao fornecedor do
// componente que falhou (pós-venda, Fase 3.5).
package recuperacao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

const (
	Reclamada  = "RECLAMADA"
	Aceita     = "ACEITA"
	Recusada   = "RECUSADA"
	Recuperada = "RECUPERADA"
)

var Formas = []string{"NOTA_CREDITO", "DEBITO", "REPOSICAO", "DESCONTO"}

type Reclamacao struct {
	ID              int64           `json:"id"`
	Numero          int64           `json:"numero"`
	SupplierCode    int64           `json:"supplier_code"`
	FornecedorNome  string          `json:"fornecedor_nome,omitempty"`
	OSID            *int64          `json:"os_id,omitempty"`
	OSNumero        *int64          `json:"os_numero,omitempty"`
	CallCode        *int64          `json:"call_code,omitempty"`
	CallNumero      *int64          `json:"call_number,omitempty"`
	ItemCode        *int64          `json:"item_code,omitempty"`
	ItemNome        string          `json:"item_nome,omitempty"`
	CapaID          *int64          `json:"capa_id,omitempty"`
	Motivo          string          `json:"motivo"`
	ValorReclamado  decimal.Decimal `json:"valor_reclamado"`
	ValorRecuperado decimal.Decimal `json:"valor_recuperado"`
	Situacao        string          `json:"situacao"`
	Forma           *string         `json:"forma,omitempty"`
	Documento       *string         `json:"documento,omitempty"`
	Resposta        *string         `json:"resposta,omitempty"`
	RecuperadoEm    *time.Time      `json:"recuperado_em,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func Rotulo(s string) string {
	switch s {
	case Reclamada:
		return "reclamada"
	case Aceita:
		return "aceita pelo fornecedor"
	case Recusada:
		return "recusada pelo fornecedor"
	case Recuperada:
		return "recuperada"
	}
	return strings.ToLower(s)
}

// ConferirAbertura: fornecedor, motivo e valor reclamado positivo.
func (r Reclamacao) ConferirAbertura() error {
	if r.SupplierCode <= 0 {
		return domainerr.NewValidation("informe o fornecedor")
	}
	if strings.TrimSpace(r.Motivo) == "" {
		return domainerr.NewValidation("descreva a falha que o fornecedor deve ressarcir")
	}
	if !r.ValorReclamado.IsPositive() {
		return domainerr.NewValidation("informe o valor reclamado")
	}
	return nil
}

// Resposta do fornecedor ou registro do ressarcimento.
type Mudanca struct {
	Para            string
	Resposta        *string
	ValorRecuperado decimal.Decimal
	Forma           *string
	Documento       *string
	RecuperadoEm    *time.Time
	Por             *uuid.UUID
}

func formaValida(f string) bool {
	for _, x := range Formas {
		if x == f {
			return true
		}
	}
	return false
}

// Conferir a mudança a partir da situação atual e do valor reclamado.
// RECLAMADA → ACEITA | RECUSADA | RECUPERADA; ACEITA → RECUPERADA | RECUSADA.
// Recusada e recuperada são finais.
func (m Mudanca) Conferir(atual string, reclamado decimal.Decimal) error {
	permitido := map[string][]string{Reclamada: {Aceita, Recusada, Recuperada}, Aceita: {Recuperada, Recusada}}
	ok := false
	for _, p := range permitido[atual] {
		if p == m.Para {
			ok = true
		}
	}
	if !ok {
		return domainerr.NewValidation(fmt.Sprintf("a reclamação está %s e não pode ficar %s", Rotulo(atual), Rotulo(m.Para)))
	}
	switch m.Para {
	case Recusada:
		if m.Resposta == nil || strings.TrimSpace(*m.Resposta) == "" {
			return domainerr.NewValidation("registre a resposta do fornecedor para a recusa")
		}
	case Recuperada:
		if !m.ValorRecuperado.IsPositive() {
			return domainerr.NewValidation("informe o valor recuperado")
		}
		if m.ValorRecuperado.GreaterThan(reclamado) {
			return domainerr.NewValidation("o valor recuperado não pode passar do reclamado")
		}
		if m.Forma == nil || !formaValida(*m.Forma) {
			return domainerr.NewValidation("informe como o fornecedor ressarciu (nota de crédito, débito, reposição ou desconto)")
		}
		if m.RecuperadoEm == nil {
			return domainerr.NewValidation("informe a data do ressarcimento")
		}
	}
	return nil
}

type Filtro struct {
	Situacao   string
	Fornecedor *int64
	OSID       *int64
}

// ChamadoCAPA: o que a CAPA da reclamação do cliente herda do chamado.
type ChamadoCAPA struct {
	CallCode     int64
	CallNumber   int64
	CustomerCode int64
	Assunto      string
	CapaID       *int64
	// Itens do chamado (código do item do chamado → item).
	Itens map[int64]int64
}

type Repository interface {
	Abrir(ctx context.Context, r *Reclamacao, por *uuid.UUID) error
	Obter(ctx context.Context, id int64) (*Reclamacao, error)
	Listar(ctx context.Context, f Filtro) ([]Reclamacao, error)
	// Mudar trava a reclamação, confere a mudança pelo domínio e grava.
	Mudar(ctx context.Context, id int64, m Mudanca) error
	ChamadoParaCAPA(ctx context.Context, callCode int64) (*ChamadoCAPA, error)
	// VincularCAPA liga a CAPA ao chamado que ainda não tem CAPA.
	VincularCAPA(ctx context.Context, callCode, capaID int64) error
}
