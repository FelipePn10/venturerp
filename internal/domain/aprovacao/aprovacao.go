// Package aprovacao é o motor de aprovação por alçada: política por empresa e
// tipo de documento, níveis em escalada pelo valor e segregação de funções.
package aprovacao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Tipo é o documento que passa pela aprovação.
type Tipo string

const (
	LiberacaoCredito           Tipo = "LIBERACAO_CREDITO"
	OrcamentoPoliticaComercial Tipo = "ORCAMENTO_POLITICA_COMERCIAL"
	// PedidoPoliticaComercial: pedido de venda com política comercial que
	// exige aprovação ou desconto acima da alçada (Fase 5.3).
	PedidoPoliticaComercial Tipo = "PEDIDO_POLITICA_COMERCIAL"
)

// Tipos disponíveis, com o nome para a tela.
var Tipos = []struct {
	Tipo Tipo
	Nome string
}{
	{LiberacaoCredito, "Liberação de crédito do pedido de venda"},
	{OrcamentoPoliticaComercial, "Orçamento bloqueado por política comercial"},
	{PedidoPoliticaComercial, "Pedido bloqueado por política comercial ou desconto acima da alçada"},
}

// TipoValido diz se o tipo existe.
func TipoValido(t Tipo) bool {
	for _, x := range Tipos {
		if x.Tipo == t {
			return true
		}
	}
	return false
}

// Status da solicitação.
type Status string

const (
	Pendente  Status = "PENDENTE"
	Aprovada  Status = "APROVADA"
	Reprovada Status = "REPROVADA"
	Cancelada Status = "CANCELADA"
)

// Nivel é um degrau da alçada. ValorAte nil = sem teto. O aprovador é um perfil
// de acesso da empresa OU um usuário.
type Nivel struct {
	Ordem    int
	ValorAte *decimal.Decimal
	PerfilID *int64
	UserID   *uuid.UUID
}

// Politica é a alçada da empresa para um tipo de documento.
type Politica struct {
	ID         int64
	Tipo       Tipo
	Descricao  *string
	Ativa      bool
	Segregacao bool
	Niveis     []Nivel
	UpdatedAt  time.Time
	UpdatedBy  uuid.UUID
}

// Validar exige níveis 1..n em sequência, com teto crescente e só o último
// sem teto (senão haveria valor que nenhum nível encerra).
func (p *Politica) Validar() error {
	if !TipoValido(p.Tipo) {
		return domainerr.NewValidation(fmt.Sprintf("tipo de aprovação %q não existe", p.Tipo))
	}
	if len(p.Niveis) == 0 {
		return domainerr.NewValidation("informe ao menos um nível de aprovação")
	}
	if len(p.Niveis) > 9 {
		return domainerr.NewValidation("a alçada tem no máximo 9 níveis")
	}
	var anterior *decimal.Decimal
	for i := range p.Niveis {
		n := &p.Niveis[i]
		n.Ordem = i + 1
		if (n.PerfilID == nil) == (n.UserID == nil) {
			return domainerr.NewValidation(fmt.Sprintf("nível %d: informe o perfil OU o usuário aprovador", n.Ordem))
		}
		ultimo := i == len(p.Niveis)-1
		if n.ValorAte == nil && !ultimo {
			return domainerr.NewValidation(fmt.Sprintf("nível %d: só o último nível pode ficar sem valor máximo", n.Ordem))
		}
		if n.ValorAte != nil {
			if !n.ValorAte.IsPositive() {
				return domainerr.NewValidation(fmt.Sprintf("nível %d: o valor máximo deve ser maior que zero", n.Ordem))
			}
			if anterior != nil && !n.ValorAte.GreaterThan(*anterior) {
				return domainerr.NewValidation(fmt.Sprintf("nível %d: o valor máximo deve ser maior que o do nível anterior", n.Ordem))
			}
			anterior = n.ValorAte
		}
	}
	if p.Descricao != nil {
		d := strings.TrimSpace(*p.Descricao)
		if d == "" {
			p.Descricao = nil
		} else {
			p.Descricao = &d
		}
	}
	return nil
}

// NiveisExigidos: os níveis até o primeiro cujo teto cobre o valor. Valor acima
// de todos os tetos exige todos os níveis.
func (p Politica) NiveisExigidos(valor decimal.Decimal) int {
	for _, n := range p.Niveis {
		if n.ValorAte == nil || !valor.GreaterThan(*n.ValorAte) {
			return n.Ordem
		}
	}
	return len(p.Niveis)
}

// Nivel devolve o nível pela ordem.
func (p Politica) Nivel(ordem int) (Nivel, bool) {
	for _, n := range p.Niveis {
		if n.Ordem == ordem {
			return n, true
		}
	}
	return Nivel{}, false
}

// Evento é o histórico imutável da solicitação.
type Evento struct {
	Evento        string
	Nivel         *int
	Ator          uuid.UUID
	AtorNome      string
	Justificativa *string
	CreatedAt     time.Time
}

// Solicitacao é um documento esperando aprovação.
type Solicitacao struct {
	ID              int64
	PoliticaID      int64
	Tipo            Tipo
	DocumentoCodigo int64
	Valor           decimal.Decimal
	Descricao       string
	Solicitante     uuid.UUID
	SolicitanteNome string
	Status          Status
	NiveisExigidos  int
	NivelAtual      int
	CreatedAt       time.Time
	ConcluidaAt     *time.Time
	Eventos         []Evento
}

// Aprovadores devolve quem já aprovou algum nível.
func (s Solicitacao) Aprovadores() []uuid.UUID {
	out := []uuid.UUID{}
	for _, e := range s.Eventos {
		if e.Evento == "APROVADA_NIVEL" || e.Evento == "APROVADA" {
			out = append(out, e.Ator)
		}
	}
	return out
}

// ConferirAprovador decide se o ator pode aprovar o nível atual. temPerfil diz
// se o ator tem (ativo) o perfil exigido pelo nível.
func ConferirAprovador(p Politica, s Solicitacao, ator uuid.UUID, temPerfil bool) error {
	if s.Status != Pendente {
		return domainerr.NewValidation("esta solicitação já foi concluída")
	}
	n, ok := p.Nivel(s.NivelAtual)
	if !ok {
		return domainerr.NewValidation("a política mudou e não tem mais o nível desta solicitação: peça ao administrador para revisá-la")
	}
	if p.Segregacao {
		if ator == s.Solicitante {
			return domainerr.NewValidation("segregação de funções: quem pediu a aprovação não pode aprová-la")
		}
		for _, a := range s.Aprovadores() {
			if a == ator {
				return domainerr.NewValidation("segregação de funções: você já aprovou um nível desta solicitação")
			}
		}
	}
	switch {
	case n.UserID != nil && *n.UserID == ator:
		return nil
	case n.PerfilID != nil && temPerfil:
		return nil
	}
	return domainerr.NewValidation(fmt.Sprintf("você não é aprovador do nível %d desta alçada", n.Ordem))
}

// Efetivador aplica no documento o resultado da aprovação. Efetivar precisa ser
// idempotente: pode rodar de novo se a gravação da aprovação falhar depois.
type Efetivador interface {
	Efetivar(ctx context.Context, documentoCodigo, solicitacaoID int64) error
	Reprovar(ctx context.Context, documentoCodigo, solicitacaoID int64, justificativa string) error
}

// Repository guarda políticas e solicitações da empresa da sessão.
type Repository interface {
	ListarPoliticas(ctx context.Context) ([]Politica, error)
	PoliticaAtiva(ctx context.Context, t Tipo) (*Politica, error)
	PoliticaPorID(ctx context.Context, id int64) (*Politica, error)
	SalvarPolitica(ctx context.Context, p *Politica, actor uuid.UUID) error

	// CriarSolicitacao devolve a pendente que já existir para o documento.
	CriarSolicitacao(ctx context.Context, s *Solicitacao) (*Solicitacao, bool, error)
	BuscarSolicitacao(ctx context.Context, id int64) (*Solicitacao, error)
	PendenteDoDocumento(ctx context.Context, t Tipo, documentoCodigo int64) (*Solicitacao, error)
	// AprovadaDoDocumento: a última aprovada do documento (nil se nenhuma).
	AprovadaDoDocumento(ctx context.Context, t Tipo, documentoCodigo int64) (*Solicitacao, error)
	ListarSolicitacoes(ctx context.Context, status Status, limite int) ([]Solicitacao, error)
	UsuarioTemPerfil(ctx context.Context, userID uuid.UUID, perfilID int64) (bool, error)

	// RegistrarAprovacaoDeNivel avança o nível (ou conclui) se a solicitação
	// ainda estiver no nível esperado — duas aprovações simultâneas não pulam
	// nível.
	RegistrarAprovacaoDeNivel(ctx context.Context, id int64, nivelEsperado int, concluir bool, ator uuid.UUID, justificativa *string) error
	RegistrarReprovacao(ctx context.Context, id int64, nivel int, ator uuid.UUID, justificativa string) error
}
