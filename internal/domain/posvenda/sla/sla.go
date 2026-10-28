// Package sla: prazo de primeira resposta e de solução do chamado de
// assistência técnica (pós-venda, Fase 3.4), em horas corridas ou úteis
// (expediente da empresa menos feriados), com o relógio parado enquanto o
// chamado aguarda o cliente.
package sla

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

const (
	// Situações de um prazo.
	SemPrazo  = "SEM_PRAZO"
	NoPrazo   = "NO_PRAZO"
	EmRisco   = "EM_RISCO"
	Vencido   = "VENCIDO"
	Cumprido  = "CUMPRIDO"
	Estourado = "ESTOURADO" // concluído depois do prazo
	Pausado   = "PAUSADO"   // relógio parado: o chamado aguarda o cliente

	AlertaPadrao = 80
)

var Prioridades = []string{"LOW", "NORMAL", "HIGH", "CRITICAL"}

// StatusDePausa: situações do chamado em que o relógio do SLA para — a
// pendência é do cliente (retorno ou pedido/aprovação dele).
var StatusDePausa = map[string]bool{"WAITING_RETURN": true, "WAITING_ORDER": true}

// Regra: prazos de uma prioridade, gerais ou de um cliente.
type Regra struct {
	ID            int64           `json:"id"`
	Prioridade    string          `json:"prioridade"`
	CustomerCode  *int64          `json:"customer_code,omitempty"`
	ClienteNome   string          `json:"cliente_nome,omitempty"`
	HorasResposta decimal.Decimal `json:"horas_resposta"`
	HorasSolucao  decimal.Decimal `json:"horas_solucao"`
	AlertaPct     int             `json:"alerta_pct"`
	// HorasUteis: o prazo corre só no expediente da empresa (sem feriados).
	HorasUteis bool      `json:"horas_uteis"`
	Ativo      bool      `json:"ativo"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func prioridadeValida(p string) bool {
	for _, x := range Prioridades {
		if x == p {
			return true
		}
	}
	return false
}

// Conferir valida a regra antes de gravar.
func (r Regra) Conferir() error {
	if !prioridadeValida(r.Prioridade) {
		return domainerr.NewValidation("prioridade inválida (LOW, NORMAL, HIGH ou CRITICAL)")
	}
	if !r.HorasResposta.IsPositive() || !r.HorasSolucao.IsPositive() {
		return domainerr.NewValidation("informe as horas de resposta e de solução")
	}
	if r.HorasSolucao.LessThan(r.HorasResposta) {
		return domainerr.NewValidation("o prazo de solução não pode ser menor que o de resposta")
	}
	if r.AlertaPct < 1 || r.AlertaPct > 99 {
		return domainerr.NewValidation("o alerta de risco é um percentual do prazo entre 1 e 99")
	}
	return nil
}

func duracao(horas decimal.Decimal) time.Duration {
	return time.Duration(horas.Mul(decimal.NewFromInt(int64(time.Hour))).IntPart())
}

// Relogio soma e mede o tempo do SLA: corrido, ou útil pelo calendário.
type Relogio struct {
	Uteis bool
	Cal   *Calendario
}

func (r Relogio) Somar(t time.Time, d time.Duration) time.Time {
	if r.Uteis && r.Cal != nil {
		return r.Cal.Somar(t, d)
	}
	return t.Add(d)
}

func (r Relogio) Entre(a, b time.Time) time.Duration {
	if r.Uteis && r.Cal != nil {
		return r.Cal.Entre(a, b)
	}
	if b.After(a) {
		return b.Sub(a)
	}
	return 0
}

// Prazos: limites de resposta e de solução a partir da abertura, e o momento
// em que cada um entra em risco (percentual de alerta do prazo, no mesmo
// relógio). cal é usado quando a regra é em horas úteis.
func (r Regra) Prazos(abertura time.Time, cal *Calendario) Prazo {
	rel := Relogio{Uteis: r.HorasUteis, Cal: cal}
	pct := r.AlertaPct
	if pct <= 0 {
		pct = AlertaPadrao
	}
	risco := func(h decimal.Decimal) time.Duration {
		return duracao(h.Mul(decimal.NewFromInt(int64(pct))).Div(decimal.NewFromInt(100)))
	}
	resp, sol := rel.Somar(abertura, duracao(r.HorasResposta)), rel.Somar(abertura, duracao(r.HorasSolucao))
	rr, sr := rel.Somar(abertura, risco(r.HorasResposta)), rel.Somar(abertura, risco(r.HorasSolucao))
	id, uteis := r.ID, r.HorasUteis
	return Prazo{RegraID: &id, RespostaAte: &resp, SolucaoAte: &sol, AlertaPct: &pct, RespostaRisco: &rr, SolucaoRisco: &sr, HorasUteis: uteis}
}

// Estender empurra um limite pelo tempo em que o relógio ficou parado
// (de inicioPausa até agora), medido e somado no mesmo relógio.
func Estender(limite *time.Time, inicioPausa, agora time.Time, rel Relogio) *time.Time {
	if limite == nil {
		return nil
	}
	n := rel.Somar(*limite, rel.Entre(inicioPausa, agora))
	return &n
}

// Escolher: a regra do cliente vence a geral da mesma prioridade; inativa não vale.
func Escolher(regras []Regra, prioridade string, cliente int64) *Regra {
	var geral *Regra
	for i := range regras {
		r := &regras[i]
		if !r.Ativo || r.Prioridade != prioridade {
			continue
		}
		if r.CustomerCode != nil && *r.CustomerCode == cliente {
			return r
		}
		if r.CustomerCode == nil {
			geral = r
		}
	}
	return geral
}

// Situacao de um prazo com limite ate e momento de risco riscoEm, concluído em
// concluidoEm (nil enquanto aberto). Com o relógio parado (pausadoDesde), o
// prazo aberto não vence nem entra em risco. Sem riscoEm (chamado anterior ao
// cálculo gravado), o risco é o percentual do intervalo corrido.
func Situacao(inicio time.Time, ate, riscoEm, concluidoEm, pausadoDesde *time.Time, alertaPct int, agora time.Time) string {
	if ate == nil {
		return SemPrazo
	}
	if concluidoEm != nil {
		if concluidoEm.After(*ate) {
			return Estourado
		}
		return Cumprido
	}
	if pausadoDesde != nil {
		return Pausado
	}
	if !agora.Before(*ate) {
		return Vencido
	}
	if riscoEm != nil {
		if !agora.Before(*riscoEm) {
			return EmRisco
		}
		return NoPrazo
	}
	if alertaPct <= 0 {
		alertaPct = AlertaPadrao
	}
	total := ate.Sub(inicio)
	if total > 0 && agora.Sub(inicio)*100 >= total*time.Duration(alertaPct) {
		return EmRisco
	}
	return NoPrazo
}

// Chamado: o que o painel do SLA mostra de cada chamado.
type Chamado struct {
	CallCode      int64      `json:"call_code"`
	CallNumber    int64      `json:"call_number"`
	CustomerCode  int64      `json:"customer_code"`
	ClienteNome   string     `json:"cliente_nome,omitempty"`
	Assunto       string     `json:"assunto"`
	Prioridade    string     `json:"prioridade"`
	Status        string     `json:"status"`
	AbertoEm      time.Time  `json:"aberto_em"`
	RespostaAte   *time.Time `json:"resposta_ate,omitempty"`
	RespondidoEm  *time.Time `json:"respondido_em,omitempty"`
	SolucaoAte    *time.Time `json:"solucao_ate,omitempty"`
	SolucionadoEm *time.Time `json:"solucionado_em,omitempty"`
	AlertaPct     int        `json:"alerta_pct"`
	RespostaRisco *time.Time `json:"resposta_risco_em,omitempty"`
	SolucaoRisco  *time.Time `json:"solucao_risco_em,omitempty"`
	HorasUteis    bool       `json:"horas_uteis"`
	PausadoDesde  *time.Time `json:"pausado_desde,omitempty"`
	// TempoPausado: horas de relógio parado já encerradas (pausas anteriores).
	HorasPausadas decimal.Decimal `json:"horas_pausadas"`
	SituacaoResp  string          `json:"situacao_resposta"`
	SituacaoSoluc string          `json:"situacao_solucao"`
	Cancelado     bool            `json:"cancelado"`
}

// Classificar preenche as situações de resposta e de solução.
func (c *Chamado) Classificar(agora time.Time) {
	if c.Cancelado {
		c.SituacaoResp, c.SituacaoSoluc = SemPrazo, SemPrazo
		return
	}
	c.SituacaoResp = Situacao(c.AbertoEm, c.RespostaAte, c.RespostaRisco, c.RespondidoEm, c.PausadoDesde, c.AlertaPct, agora)
	c.SituacaoSoluc = Situacao(c.AbertoEm, c.SolucaoAte, c.SolucaoRisco, c.SolucionadoEm, c.PausadoDesde, c.AlertaPct, agora)
}

// Prazo aplicado ao chamado.
type Prazo struct {
	RegraID       *int64
	RespostaAte   *time.Time
	SolucaoAte    *time.Time
	AlertaPct     *int
	RespostaRisco *time.Time
	SolucaoRisco  *time.Time
	HorasUteis    bool
}

// EstadoDoRelogio: o que a retomada precisa do chamado pausado.
type EstadoDoRelogio struct {
	PausadoDesde  *time.Time
	HorasUteis    bool
	Respondido    bool
	RespostaAte   *time.Time
	RespostaRisco *time.Time
	SolucaoAte    *time.Time
	SolucaoRisco  *time.Time
}

// Retomada: os limites empurrados pelo tempo parado. O de resposta só se
// ainda não respondido.
func (e EstadoDoRelogio) Retomada(agora time.Time, cal *Calendario) (Prazo, time.Duration) {
	rel := Relogio{Uteis: e.HorasUteis, Cal: cal}
	p := Prazo{RespostaAte: e.RespostaAte, RespostaRisco: e.RespostaRisco, HorasUteis: e.HorasUteis}
	if e.PausadoDesde == nil {
		p.SolucaoAte, p.SolucaoRisco = e.SolucaoAte, e.SolucaoRisco
		return p, 0
	}
	parado := rel.Entre(*e.PausadoDesde, agora)
	if !e.Respondido {
		p.RespostaAte, p.RespostaRisco = Estender(e.RespostaAte, *e.PausadoDesde, agora, rel), Estender(e.RespostaRisco, *e.PausadoDesde, agora, rel)
	}
	p.SolucaoAte, p.SolucaoRisco = Estender(e.SolucaoAte, *e.PausadoDesde, agora, rel), Estender(e.SolucaoRisco, *e.PausadoDesde, agora, rel)
	return p, parado
}

type Filtro struct {
	Situacao string // EM_RISCO, VENCIDO, ABERTOS (padrão), TODOS
	Cliente  *int64
	Desde    *time.Time
}

type Repository interface {
	Regras(ctx context.Context) ([]Regra, error)
	SalvarRegra(ctx context.Context, r *Regra, por *uuid.UUID) error
	ExcluirRegra(ctx context.Context, id int64) error
	// DadosDoChamado: prioridade, cliente e abertura do chamado da empresa.
	DadosDoChamado(ctx context.Context, callCode int64) (prioridade string, cliente int64, abertura time.Time, err error)
	AplicarPrazo(ctx context.Context, callCode int64, p Prazo) error
	// Calendario: expediente cadastrado (vazio = padrão), feriados do
	// calendário industrial e fuso da empresa.
	Calendario(ctx context.Context) (*Calendario, error)
	Expediente(ctx context.Context) ([]Intervalo, error)
	SalvarExpediente(ctx context.Context, iv []Intervalo, por *uuid.UUID) error
	// Pausar para o relógio do chamado com SLA (não faz nada se já parado).
	Pausar(ctx context.Context, callCode int64, em time.Time, motivo string) error
	EstadoDoRelogio(ctx context.Context, callCode int64) (*EstadoDoRelogio, error)
	// Retomar grava os novos limites e encerra a pausa aberta.
	Retomar(ctx context.Context, callCode int64, p Prazo, em time.Time, parado time.Duration) error
	// MarcarResposta grava a primeira resposta (não sobrescreve a anterior).
	MarcarResposta(ctx context.Context, callCode int64, em time.Time) error
	Chamados(ctx context.Context, f Filtro) ([]Chamado, error)
}
