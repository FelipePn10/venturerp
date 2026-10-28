// Package gestao: o que faz do centro de custo um instrumento de gestão — o
// código da empresa, quem responde por ele, quem aprova as requisições, o
// orçamento mensal (orçado × realizado) e a árvore consolidada.
package gestao

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Gestao do centro (cost_center_gestao).
type Gestao struct {
	CostCenterID      int64
	Codigo            string
	ResponsavelID     *uuid.UUID
	AprovadorID       *uuid.UUID
	AprovaRequisicoes bool
	ContaPadraoID     *int64
}

// Conferir: código da empresa obrigatório; aprovar requisições exige aprovador.
func (g *Gestao) Conferir() error {
	g.Codigo = strings.TrimSpace(g.Codigo)
	if g.Codigo == "" {
		return domainerr.NewValidation("informe o código do centro de custo")
	}
	if len(g.Codigo) > 50 {
		return domainerr.NewValidation("o código do centro de custo tem no máximo 50 caracteres")
	}
	if g.AprovaRequisicoes && g.AprovadorID == nil {
		return domainerr.NewValidation("para aprovar as requisições do centro, informe o aprovador")
	}
	return nil
}

// Centro: o centro com a gestão, como a tela e a árvore usam.
type Centro struct {
	ID          int64
	Code        int32
	Codigo      string
	Descricao   string
	ParentCode  *int32
	Tipo        string
	IsRatio     bool
	StartDate   time.Time
	EndDate     *time.Time
	IsActive    bool
	Gestao      Gestao
	Responsavel string
	Aprovador   string
}

// RecebeLancamentoEm: ativo e dentro da vigência no dia.
func (c *Centro) RecebeLancamentoEm(dia time.Time) error {
	if !c.IsActive {
		return domainerr.NewValidation(fmt.Sprintf("o centro de custo %s está inativo", c.Codigo))
	}
	d := dia.Format(time.DateOnly)
	if d < c.StartDate.Format(time.DateOnly) || (c.EndDate != nil && d > c.EndDate.Format(time.DateOnly)) {
		return domainerr.NewValidation(fmt.Sprintf("o centro de custo %s está fora da vigência em %s", c.Codigo, dia.Format("02/01/2006")))
	}
	return nil
}

// Orcamento: o orçado de um centro num mês.
type Orcamento struct {
	CostCenterID int64
	Ano, Mes     int
	Valor        decimal.Decimal
}

// ConferirOrcamento: mês 1 a 12, ano plausível, valor não negativo.
func ConferirOrcamento(o Orcamento) error {
	if o.Mes < 1 || o.Mes > 12 {
		return domainerr.NewValidation("mês do orçamento de 1 a 12")
	}
	if o.Ano < 2000 || o.Ano > 2100 {
		return domainerr.NewValidation("ano do orçamento inválido")
	}
	if o.Valor.IsNegative() {
		return domainerr.NewValidation("o orçamento não pode ser negativo")
	}
	return nil
}

// No da árvore: os valores do próprio centro e os consolidados (ele + filhos).
type No struct {
	Centro
	Orcado, Realizado                       decimal.Decimal
	OrcadoConsolidado, RealizadoConsolidado decimal.Decimal
	Filhos                                  []*No
}

// Estourado: realizado consolidado acima do orçado consolidado (com orçamento).
func (n *No) Estourado() bool {
	return n.OrcadoConsolidado.IsPositive() && n.RealizadoConsolidado.GreaterThan(n.OrcadoConsolidado)
}

// MontarArvore liga cada centro ao pai (pelo code) e soma os filhos no pai.
// Centro cujo pai não está na lista vira raiz.
func MontarArvore(centros []Centro, orcado, realizado map[int64]decimal.Decimal) []*No {
	nos := make(map[int32]*No, len(centros))
	for _, c := range centros {
		nos[c.Code] = &No{Centro: c, Orcado: orcado[c.ID], Realizado: realizado[c.ID]}
	}
	var raizes []*No
	for _, c := range centros {
		n := nos[c.Code]
		if c.ParentCode != nil {
			if p, ok := nos[*c.ParentCode]; ok && p != n {
				p.Filhos = append(p.Filhos, n)
				continue
			}
		}
		raizes = append(raizes, n)
	}
	var somar func(n *No, profundidade int)
	somar = func(n *No, profundidade int) {
		n.OrcadoConsolidado, n.RealizadoConsolidado = n.Orcado, n.Realizado
		if profundidade > 50 { // árvore com ciclo: não soma além
			return
		}
		sort.Slice(n.Filhos, func(i, j int) bool { return n.Filhos[i].Codigo < n.Filhos[j].Codigo })
		for _, f := range n.Filhos {
			somar(f, profundidade+1)
			n.OrcadoConsolidado = n.OrcadoConsolidado.Add(f.OrcadoConsolidado)
			n.RealizadoConsolidado = n.RealizadoConsolidado.Add(f.RealizadoConsolidado)
		}
	}
	sort.Slice(raizes, func(i, j int) bool { return raizes[i].Codigo < raizes[j].Codigo })
	for _, r := range raizes {
		somar(r, 0)
	}
	return raizes
}

// Status da aprovação de requisição pelo aprovador do centro.
const (
	AprovacaoPendente  = "PENDENTE"
	AprovacaoAprovada  = "APROVADA"
	AprovacaoReprovada = "REPROVADA"
)

// AprovacaoRequisicao: a requisição de compra com item do centro espera o
// aprovador dele antes de virar pedido.
type AprovacaoRequisicao struct {
	ID              int64
	RequisitionCode int64
	CostCenterID    int64
	CentroCodigo    string
	CentroDescricao string
	Status          string
	AprovadorID     uuid.UUID
	Solicitante     *uuid.UUID
	DecididoPor     *uuid.UUID
	DecididoEm      *time.Time
	Justificativa   *string
	CriadoEm        time.Time
}

// Decidir: só a pendente, só o aprovador do centro, e quem pediu não aprova a
// própria requisição (segregação de funções). Reprovar exige justificativa.
func (a *AprovacaoRequisicao) Decidir(ator uuid.UUID, aprovar bool, justificativa string) error {
	if a.Status != AprovacaoPendente {
		return domainerr.NewValidation("esta aprovação já foi decidida")
	}
	if ator != a.AprovadorID {
		return domainerr.NewValidation(fmt.Sprintf("só o aprovador do centro de custo %s decide esta requisição", a.CentroCodigo))
	}
	if a.Solicitante != nil && *a.Solicitante == ator {
		return domainerr.NewValidation("segregação de funções: quem pediu a requisição não pode aprová-la")
	}
	justificativa = strings.TrimSpace(justificativa)
	if !aprovar && justificativa == "" {
		return domainerr.NewValidation("informe o motivo da reprovação")
	}
	a.Status = AprovacaoAprovada
	if !aprovar {
		a.Status = AprovacaoReprovada
	}
	agora := time.Now()
	a.DecididoPor, a.DecididoEm = &ator, &agora
	if justificativa != "" {
		a.Justificativa = &justificativa
	}
	return nil
}

// Usuario da empresa (para escolher responsável e aprovador).
type Usuario struct {
	ID   uuid.UUID `json:"id"`
	Nome string    `json:"nome"`
}

// Repository da gestão do centro, na empresa da sessão.
type Repository interface {
	Centros(ctx context.Context) ([]Centro, error)
	Centro(ctx context.Context, code int32) (*Centro, error)
	SalvarGestao(ctx context.Context, g Gestao, por uuid.UUID) error

	Orcamentos(ctx context.Context, ano int) ([]Orcamento, error)
	SalvarOrcamentos(ctx context.Context, os []Orcamento, por uuid.UUID) error
	// Realizado: o contas a pagar por centro (rateio do título, ou o centro do
	// título sem rateio), pela data de emissão, sem os cancelados. mes 0 = ano todo.
	Realizado(ctx context.Context, ano, mes int) (map[int64]decimal.Decimal, error)

	// Aprovações de requisição pelo centro.
	// AbrirAprovacoes: o solicitante é o criador da requisição.
	AbrirAprovacoes(ctx context.Context, requisitionCode int64) error
	Aprovacoes(ctx context.Context, status string, aprovador *uuid.UUID, requisitionCode *int64) ([]AprovacaoRequisicao, error)
	Aprovacao(ctx context.Context, id int64) (*AprovacaoRequisicao, error)
	GravarDecisao(ctx context.Context, a *AprovacaoRequisicao) error
	// Usuarios: os usuários da empresa, para responsável e aprovador.
	Usuarios(ctx context.Context) ([]Usuario, error)
	// RequisicoesBloqueadas: das requisições, as que têm aprovação de centro
	// pendente ou reprovada (com o motivo).
	RequisicoesBloqueadas(ctx context.Context, codes []int64) (map[int64]string, error)
}
