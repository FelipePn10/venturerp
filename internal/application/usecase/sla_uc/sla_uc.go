// Package sla_uc: regras de SLA da assistência e prazos dos chamados
// (pós-venda, Fase 3.4).
package sla_uc

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/sla"
)

type UseCase struct {
	Repo  dom.Repository
	Auth  ports.AuthService
	Agora func() time.Time
}

func (uc *UseCase) agora() time.Time {
	if uc.Agora != nil {
		return uc.Agora()
	}
	return time.Now()
}

func (uc *UseCase) ator(ctx context.Context) *uuid.UUID {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return nil
	}
	return &a
}

func (uc *UseCase) Regras(ctx context.Context) ([]dom.Regra, error) { return uc.Repo.Regras(ctx) }

type RegraDTO struct {
	Prioridade    string          `json:"prioridade"`
	CustomerCode  *int64          `json:"customer_code,omitempty"`
	HorasResposta decimal.Decimal `json:"horas_resposta"`
	HorasSolucao  decimal.Decimal `json:"horas_solucao"`
	AlertaPct     int             `json:"alerta_pct"`
	HorasUteis    bool            `json:"horas_uteis"`
	Ativo         *bool           `json:"ativo,omitempty"`
}

// SalvarRegra cria (id 0) ou altera a regra. Vale para os chamados abertos
// depois: o prazo de chamado já aberto não muda.
func (uc *UseCase) SalvarRegra(ctx context.Context, id int64, dto RegraDTO) (*dom.Regra, error) {
	r := dom.Regra{ID: id, Prioridade: strings.ToUpper(strings.TrimSpace(dto.Prioridade)), CustomerCode: dto.CustomerCode,
		HorasResposta: dto.HorasResposta, HorasSolucao: dto.HorasSolucao, AlertaPct: dto.AlertaPct, HorasUteis: dto.HorasUteis, Ativo: true}
	if r.AlertaPct == 0 {
		r.AlertaPct = dom.AlertaPadrao
	}
	if dto.Ativo != nil {
		r.Ativo = *dto.Ativo
	}
	if r.CustomerCode != nil && *r.CustomerCode <= 0 {
		r.CustomerCode = nil
	}
	if err := r.Conferir(); err != nil {
		return nil, err
	}
	if err := uc.Repo.SalvarRegra(ctx, &r, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return &r, nil
}

func (uc *UseCase) ExcluirRegra(ctx context.Context, id int64) error {
	return uc.Repo.ExcluirRegra(ctx, id)
}

// AplicarAoChamado calcula e grava os prazos do chamado pela regra da
// prioridade (a do cliente vence a geral). Sem regra, o chamado fica sem SLA.
func (uc *UseCase) AplicarAoChamado(ctx context.Context, callCode int64) error {
	prioridade, cliente, abertura, err := uc.Repo.DadosDoChamado(ctx, callCode)
	if err != nil {
		return err
	}
	regras, err := uc.Repo.Regras(ctx)
	if err != nil {
		return err
	}
	var p dom.Prazo
	if r := dom.Escolher(regras, prioridade, cliente); r != nil {
		var cal *dom.Calendario
		if r.HorasUteis {
			if cal, err = uc.Repo.Calendario(ctx); err != nil {
				return err
			}
		}
		p = r.Prazos(abertura, cal)
	}
	return uc.Repo.AplicarPrazo(ctx, callCode, p)
}

// MudouStatus para o relógio quando o chamado passa a aguardar o cliente e o
// retoma — empurrando os prazos em aberto pelo tempo parado — quando sai dessa
// espera (para qualquer situação, inclusive atendido: o tempo esperando o
// cliente não conta contra a solução).
func (uc *UseCase) MudouStatus(ctx context.Context, callCode int64, de, para string) error {
	pausaDe, pausaPara := dom.StatusDePausa[de], dom.StatusDePausa[para]
	agora := uc.agora()
	switch {
	case !pausaDe && pausaPara:
		return uc.Repo.Pausar(ctx, callCode, agora, para)
	case pausaDe && !pausaPara:
		e, err := uc.Repo.EstadoDoRelogio(ctx, callCode)
		if err != nil || e.PausadoDesde == nil {
			return err
		}
		var cal *dom.Calendario
		if e.HorasUteis {
			if cal, err = uc.Repo.Calendario(ctx); err != nil {
				return err
			}
		}
		p, parado := e.Retomada(agora, cal)
		return uc.Repo.Retomar(ctx, callCode, p, agora, parado)
	}
	return nil
}

// HorarioDTO: intervalo de expediente em HH:MM.
type HorarioDTO struct {
	DiaSemana int    `json:"dia_semana"`
	Inicio    string `json:"inicio"`
	Fim       string `json:"fim"`
}

type ExpedienteResponse struct {
	// Padrao: a empresa não cadastrou expediente; vale o padrão mostrado.
	Padrao     bool         `json:"padrao"`
	Intervalos []HorarioDTO `json:"intervalos"`
}

func paraDTO(iv []dom.Intervalo) []HorarioDTO {
	out := make([]HorarioDTO, 0, len(iv))
	for _, x := range iv {
		out = append(out, HorarioDTO{DiaSemana: x.DiaSemana, Inicio: dom.Hora(x.Inicio), Fim: dom.Hora(x.Fim)})
	}
	return out
}

func (uc *UseCase) Expediente(ctx context.Context) (*ExpedienteResponse, error) {
	iv, err := uc.Repo.Expediente(ctx)
	if err != nil {
		return nil, err
	}
	if len(iv) == 0 {
		return &ExpedienteResponse{Padrao: true, Intervalos: paraDTO(dom.ExpedientePadrao())}, nil
	}
	return &ExpedienteResponse{Intervalos: paraDTO(iv)}, nil
}

// SalvarExpediente substitui o expediente. Vale para os prazos calculados
// depois (abertura e retomada); os já gravados não mudam.
func (uc *UseCase) SalvarExpediente(ctx context.Context, dto []HorarioDTO) (*ExpedienteResponse, error) {
	iv := make([]dom.Intervalo, 0, len(dto))
	for _, h := range dto {
		ini, err := dom.LerHora(h.Inicio)
		if err != nil {
			return nil, err
		}
		fim, err := dom.LerHora(h.Fim)
		if err != nil {
			return nil, err
		}
		if fim == 0 && strings.TrimSpace(h.Fim) == "00:00" {
			fim = 24 * 60 // "até meia-noite"
		}
		iv = append(iv, dom.Intervalo{DiaSemana: h.DiaSemana, Inicio: ini, Fim: fim})
	}
	if err := dom.ConferirExpediente(iv); err != nil {
		return nil, err
	}
	if err := uc.Repo.SalvarExpediente(ctx, iv, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Expediente(ctx)
}

// Responder grava a primeira resposta do chamado (a seguinte não muda nada).
func (uc *UseCase) Responder(ctx context.Context, callCode int64) error {
	return uc.Repo.MarcarResposta(ctx, callCode, uc.agora())
}

type Resumo struct {
	Abertos   int `json:"abertos"`
	EmRisco   int `json:"em_risco"`
	Vencidos  int `json:"vencidos"`
	Cumpridos int `json:"cumpridos"`
	Estouros  int `json:"estourados"`
	Pausados  int `json:"pausados"`
}

type Painel struct {
	Resumo   Resumo        `json:"resumo"`
	Chamados []dom.Chamado `json:"chamados"`
}

func pior(a, b string) string {
	ordem := map[string]int{dom.Vencido: 6, dom.Estourado: 5, dom.EmRisco: 4, dom.Pausado: 3, dom.NoPrazo: 2, dom.Cumprido: 1, dom.SemPrazo: 0}
	if ordem[b] > ordem[a] {
		return b
	}
	return a
}

// Painel: chamados com SLA e a situação de cada prazo. situacao EM_RISCO ou
// VENCIDO filtra pelo pior dos dois prazos; TODOS inclui os encerrados.
func (uc *UseCase) Painel(ctx context.Context, situacao string, cliente *int64, dias int) (*Painel, error) {
	situacao = strings.ToUpper(strings.TrimSpace(situacao))
	f := dom.Filtro{Situacao: situacao, Cliente: cliente}
	if situacao == "TODOS" {
		if dias <= 0 {
			dias = 90
		}
		d := uc.agora().AddDate(0, 0, -dias)
		f.Desde = &d
	}
	cs, err := uc.Repo.Chamados(ctx, f)
	if err != nil {
		return nil, err
	}
	agora := uc.agora()
	out := &Painel{Chamados: []dom.Chamado{}}
	for _, c := range cs {
		c.Classificar(agora)
		p := pior(c.SituacaoResp, c.SituacaoSoluc)
		switch {
		case c.Cancelado:
		case c.SolucionadoEm == nil:
			out.Resumo.Abertos++
		case c.SituacaoSoluc == dom.Estourado:
			out.Resumo.Estouros++
		default:
			out.Resumo.Cumpridos++
		}
		if c.SolucionadoEm == nil && !c.Cancelado {
			if c.PausadoDesde != nil {
				out.Resumo.Pausados++
			}
			if p == dom.Vencido {
				out.Resumo.Vencidos++
			} else if p == dom.EmRisco {
				out.Resumo.EmRisco++
			}
		}
		if situacao == dom.Pausado {
			if c.PausadoDesde == nil || c.SolucionadoEm != nil || c.Cancelado {
				continue
			}
		} else if (situacao == dom.EmRisco || situacao == dom.Vencido) && p != situacao {
			continue
		}
		out.Chamados = append(out.Chamados, c)
	}
	return out, nil
}
