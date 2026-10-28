// Package aprovacao_uc é o motor de aprovação: os módulos pedem aprovação de um
// documento (Solicitar), os aprovadores aprovam ou reprovam nível a nível e,
// no último nível, o efetivador do tipo aplica o resultado no documento.
package aprovacao_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
)

type UseCase struct {
	Repo         aprovacao.Repository
	Auth         ports.AuthService
	Efetivadores map[aprovacao.Tipo]aprovacao.Efetivador
}

// Registrar liga o efetivador de um tipo (feito na montagem da aplicação).
func (uc *UseCase) Registrar(t aprovacao.Tipo, e aprovacao.Efetivador) {
	if uc.Efetivadores == nil {
		uc.Efetivadores = map[aprovacao.Tipo]aprovacao.Efetivador{}
	}
	uc.Efetivadores[t] = e
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type NivelDTO struct {
	ValorAte *decimal.Decimal `json:"valor_ate"`
	PerfilID *int64           `json:"perfil_id,omitempty"`
	UserID   *string          `json:"user_id,omitempty"`
}

type PoliticaDTO struct {
	Descricao  *string    `json:"descricao,omitempty"`
	Ativa      bool       `json:"ativa"`
	Segregacao bool       `json:"segregacao"`
	Niveis     []NivelDTO `json:"niveis"`
}

type NivelResponse struct {
	Ordem    int              `json:"ordem"`
	ValorAte *decimal.Decimal `json:"valor_ate"`
	PerfilID *int64           `json:"perfil_id,omitempty"`
	UserID   *string          `json:"user_id,omitempty"`
}

type PoliticaResponse struct {
	Tipo       string          `json:"tipo"`
	Nome       string          `json:"nome"`
	Existe     bool            `json:"existe"`
	Descricao  *string         `json:"descricao,omitempty"`
	Ativa      bool            `json:"ativa"`
	Segregacao bool            `json:"segregacao"`
	Niveis     []NivelResponse `json:"niveis"`
	UpdatedAt  *string         `json:"updated_at,omitempty"`
}

type EventoResponse struct {
	Evento        string  `json:"evento"`
	Nivel         *int    `json:"nivel,omitempty"`
	Ator          string  `json:"ator"`
	AtorNome      string  `json:"ator_nome"`
	Justificativa *string `json:"justificativa,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type SolicitacaoResponse struct {
	ID              int64            `json:"id"`
	Tipo            string           `json:"tipo"`
	TipoNome        string           `json:"tipo_nome"`
	DocumentoCodigo int64            `json:"documento_codigo"`
	Valor           decimal.Decimal  `json:"valor"`
	Descricao       string           `json:"descricao"`
	Solicitante     string           `json:"solicitante"`
	SolicitanteNome string           `json:"solicitante_nome"`
	Status          string           `json:"status"`
	NiveisExigidos  int              `json:"niveis_exigidos"`
	NivelAtual      int              `json:"nivel_atual"`
	PossoAprovar    bool             `json:"posso_aprovar"`
	MotivoBloqueio  string           `json:"motivo_bloqueio,omitempty"`
	CreatedAt       string           `json:"created_at"`
	ConcluidaAt     *string          `json:"concluida_at,omitempty"`
	Eventos         []EventoResponse `json:"eventos,omitempty"`
}

type DecisaoDTO struct {
	Justificativa string `json:"justificativa"`
}

func nomeDoTipo(t aprovacao.Tipo) string {
	for _, x := range aprovacao.Tipos {
		if x.Tipo == t {
			return x.Nome
		}
	}
	return string(t)
}

// ─── solicitar (usado pelos módulos) ──────────────────────────────────────────

// Solicitar abre a aprovação do documento quando a empresa tem política ativa
// para o tipo. Sem política devolve nil — o módulo segue como antes. Documento
// com pendente devolve a pendente (não duplica).
func (uc *UseCase) Solicitar(ctx context.Context, t aprovacao.Tipo, documento int64, valor decimal.Decimal, descricao string) (*aprovacao.Solicitacao, error) {
	p, err := uc.Repo.PoliticaAtiva(ctx, t)
	if err != nil || p == nil {
		return nil, err
	}
	solicitante, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if valor.IsNegative() {
		valor = decimal.Zero
	}
	s := &aprovacao.Solicitacao{PoliticaID: p.ID, Tipo: t, DocumentoCodigo: documento, Valor: valor.Round(2),
		Descricao: strings.TrimSpace(descricao), Solicitante: solicitante, NiveisExigidos: p.NiveisExigidos(valor)}
	if s.Descricao == "" {
		s.Descricao = nomeDoTipo(t)
	}
	criada, _, err := uc.Repo.CriarSolicitacao(ctx, s)
	return criada, err
}

// TemPoliticaAtiva: há política de aprovação ativa para o tipo (sem ela,
// Solicitar não abre nada e ninguém aprovaria).
func (uc *UseCase) TemPoliticaAtiva(ctx context.Context, t aprovacao.Tipo) (bool, error) {
	p, err := uc.Repo.PoliticaAtiva(ctx, t)
	return p != nil, err
}

// AprovadaDoDocumento devolve a última solicitação aprovada do documento.
func (uc *UseCase) AprovadaDoDocumento(ctx context.Context, t aprovacao.Tipo, documento int64) (*aprovacao.Solicitacao, error) {
	return uc.Repo.AprovadaDoDocumento(ctx, t, documento)
}

// PendenteDoDocumento devolve a solicitação pendente do documento, se houver.
func (uc *UseCase) PendenteDoDocumento(ctx context.Context, t aprovacao.Tipo, documento int64) (*aprovacao.Solicitacao, error) {
	return uc.Repo.PendenteDoDocumento(ctx, t, documento)
}

// ─── políticas ────────────────────────────────────────────────────────────────

func (uc *UseCase) ListarPoliticas(ctx context.Context) ([]PoliticaResponse, error) {
	existentes, err := uc.Repo.ListarPoliticas(ctx)
	if err != nil {
		return nil, err
	}
	porTipo := map[aprovacao.Tipo]aprovacao.Politica{}
	for _, p := range existentes {
		porTipo[p.Tipo] = p
	}
	out := []PoliticaResponse{}
	for _, t := range aprovacao.Tipos {
		r := PoliticaResponse{Tipo: string(t.Tipo), Nome: t.Nome, Segregacao: true, Niveis: []NivelResponse{}}
		if p, ok := porTipo[t.Tipo]; ok {
			r.Existe, r.Descricao, r.Ativa, r.Segregacao = true, p.Descricao, p.Ativa, p.Segregacao
			em := p.UpdatedAt.Format(time.RFC3339)
			r.UpdatedAt = &em
			for _, n := range p.Niveis {
				nr := NivelResponse{Ordem: n.Ordem, PerfilID: n.PerfilID}
				if n.ValorAte != nil {
					v := n.ValorAte.Round(2)
					nr.ValorAte = &v
				}
				if n.UserID != nil {
					u := n.UserID.String()
					nr.UserID = &u
				}
				r.Niveis = append(r.Niveis, nr)
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func (uc *UseCase) SalvarPolitica(ctx context.Context, tipo string, dto PoliticaDTO) error {
	p := &aprovacao.Politica{Tipo: aprovacao.Tipo(tipo), Descricao: dto.Descricao, Ativa: dto.Ativa, Segregacao: dto.Segregacao}
	for _, n := range dto.Niveis {
		nivel := aprovacao.Nivel{ValorAte: n.ValorAte, PerfilID: n.PerfilID}
		if n.UserID != nil && strings.TrimSpace(*n.UserID) != "" {
			u, err := uuid.Parse(strings.TrimSpace(*n.UserID))
			if err != nil {
				return errorsuc.NewValidationError("usuário aprovador inválido")
			}
			nivel.UserID = &u
		}
		p.Niveis = append(p.Niveis, nivel)
	}
	if err := p.Validar(); err != nil {
		return err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Repo.SalvarPolitica(ctx, p, actor)
}

// ─── solicitações ─────────────────────────────────────────────────────────────

func (uc *UseCase) resposta(ctx context.Context, s aprovacao.Solicitacao, ator uuid.UUID, politicas map[int64]*aprovacao.Politica) SolicitacaoResponse {
	r := SolicitacaoResponse{ID: s.ID, Tipo: string(s.Tipo), TipoNome: nomeDoTipo(s.Tipo), DocumentoCodigo: s.DocumentoCodigo,
		Valor: s.Valor.Round(2), Descricao: s.Descricao, Solicitante: s.Solicitante.String(), SolicitanteNome: s.SolicitanteNome,
		Status: string(s.Status), NiveisExigidos: s.NiveisExigidos, NivelAtual: s.NivelAtual, CreatedAt: s.CreatedAt.Format(time.RFC3339)}
	if s.ConcluidaAt != nil {
		c := s.ConcluidaAt.Format(time.RFC3339)
		r.ConcluidaAt = &c
	}
	for _, e := range s.Eventos {
		r.Eventos = append(r.Eventos, EventoResponse{Evento: e.Evento, Nivel: e.Nivel, Ator: e.Ator.String(), AtorNome: e.AtorNome,
			Justificativa: e.Justificativa, CreatedAt: e.CreatedAt.Format(time.RFC3339)})
	}
	if s.Status == aprovacao.Pendente {
		if err := uc.conferir(ctx, s, ator, politicas); err != nil {
			r.MotivoBloqueio = err.Error()
		} else {
			r.PossoAprovar = true
		}
	}
	return r
}

func (uc *UseCase) politicaDe(ctx context.Context, id int64, cache map[int64]*aprovacao.Politica) (*aprovacao.Politica, error) {
	if p, ok := cache[id]; ok {
		return p, nil
	}
	p, err := uc.Repo.PoliticaPorID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errorsuc.NewNotFoundError("a política desta solicitação não existe mais")
	}
	cache[id] = p
	return p, nil
}

func (uc *UseCase) conferir(ctx context.Context, s aprovacao.Solicitacao, ator uuid.UUID, cache map[int64]*aprovacao.Politica) error {
	p, err := uc.politicaDe(ctx, s.PoliticaID, cache)
	if err != nil {
		return err
	}
	temPerfil := false
	if n, ok := p.Nivel(s.NivelAtual); ok && n.PerfilID != nil {
		if temPerfil, err = uc.Repo.UsuarioTemPerfil(ctx, ator, *n.PerfilID); err != nil {
			return err
		}
	}
	return aprovacao.ConferirAprovador(*p, s, ator, temPerfil)
}

func (uc *UseCase) ListarSolicitacoes(ctx context.Context, status string) ([]SolicitacaoResponse, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	st := aprovacao.Status(strings.ToUpper(strings.TrimSpace(status)))
	if st != "" && st != aprovacao.Pendente && st != aprovacao.Aprovada && st != aprovacao.Reprovada && st != aprovacao.Cancelada {
		return nil, errorsuc.NewValidationError("situação inválida: use PENDENTE, APROVADA, REPROVADA ou CANCELADA")
	}
	lista, err := uc.Repo.ListarSolicitacoes(ctx, st, 300)
	if err != nil {
		return nil, err
	}
	cache := map[int64]*aprovacao.Politica{}
	out := make([]SolicitacaoResponse, 0, len(lista))
	for _, s := range lista {
		// A lista não traz o histórico; quem já aprovou importa para a segregação.
		if s.Status == aprovacao.Pendente && s.NivelAtual > 1 {
			if completa, err := uc.Repo.BuscarSolicitacao(ctx, s.ID); err == nil {
				s.Eventos = completa.Eventos
			}
		}
		r := uc.resposta(ctx, s, ator, cache)
		r.Eventos = nil
		out = append(out, r)
	}
	return out, nil
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*SolicitacaoResponse, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := uc.Repo.BuscarSolicitacao(ctx, id)
	if err != nil {
		return nil, err
	}
	r := uc.resposta(ctx, *s, ator, map[int64]*aprovacao.Politica{})
	return &r, nil
}

func (uc *UseCase) Aprovar(ctx context.Context, id int64, dto DecisaoDTO) (*SolicitacaoResponse, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := uc.Repo.BuscarSolicitacao(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := uc.conferir(ctx, *s, ator, map[int64]*aprovacao.Politica{}); err != nil {
		return nil, err
	}
	concluir := s.NivelAtual >= s.NiveisExigidos
	if concluir {
		// Efetiva ANTES de concluir: se o documento não puder ser liberado, a
		// aprovação continua pendente e o erro aparece para o aprovador.
		e, ok := uc.Efetivadores[s.Tipo]
		if !ok {
			return nil, fmt.Errorf("tipo de aprovação %s sem efetivador configurado", s.Tipo)
		}
		if err := e.Efetivar(ctx, s.DocumentoCodigo, s.ID); err != nil {
			return nil, err
		}
	}
	var just *string
	if j := strings.TrimSpace(dto.Justificativa); j != "" {
		just = &j
	}
	if err := uc.Repo.RegistrarAprovacaoDeNivel(ctx, id, s.NivelAtual, concluir, ator, just); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, id)
}

func (uc *UseCase) Reprovar(ctx context.Context, id int64, dto DecisaoDTO) (*SolicitacaoResponse, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	just := strings.TrimSpace(dto.Justificativa)
	if len([]rune(just)) < 5 {
		return nil, errorsuc.NewValidationError("informe a justificativa da reprovação")
	}
	s, err := uc.Repo.BuscarSolicitacao(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := uc.conferir(ctx, *s, ator, map[int64]*aprovacao.Politica{}); err != nil {
		return nil, err
	}
	if err := uc.Repo.RegistrarReprovacao(ctx, id, s.NivelAtual, ator, just); err != nil {
		return nil, err
	}
	if e, ok := uc.Efetivadores[s.Tipo]; ok {
		if err := e.Reprovar(ctx, s.DocumentoCodigo, s.ID, just); err != nil {
			return nil, err
		}
	}
	return uc.Obter(ctx, id)
}
