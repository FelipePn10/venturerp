// Package capa_uc conduz a CAPA/8D: abertura (de NC, reclamação, fornecedor,
// auditoria ou interna), análise etapa a etapa, ações com responsável e prazo,
// avanço com conferência do que a etapa exige e encerramento pela eficácia.
package capa_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/capa"
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

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type MembroDTO struct {
	UserID string `json:"user_id"`
	Papel  string `json:"papel"`
}

type AberturaDTO struct {
	Metodo       string      `json:"metodo"`
	Tipo         string      `json:"tipo"`
	Origem       string      `json:"origem"`
	Titulo       string      `json:"titulo"`
	ItemCode     *int64      `json:"item_code,omitempty"`
	SupplierCode *int64      `json:"supplier_code,omitempty"`
	CustomerCode *int64      `json:"customer_code,omitempty"`
	Referencia   *string     `json:"referencia,omitempty"`
	Responsavel  string      `json:"responsavel,omitempty"`
	Prazo        string      `json:"prazo,omitempty"`
	NCs          []int64     `json:"ncs,omitempty"`
	Equipe       []MembroDTO `json:"equipe,omitempty"`
}

// AnaliseDTO altera só o que vier preenchido.
type AnaliseDTO struct {
	Titulo              *string              `json:"titulo,omitempty"`
	Responsavel         *string              `json:"responsavel,omitempty"`
	Prazo               *string              `json:"prazo,omitempty"`
	Problema            *dom.Problema        `json:"problema,omitempty"`
	ContencaoDispensada *string              `json:"contencao_dispensada,omitempty"`
	MetodoCausa         *string              `json:"metodo_causa,omitempty"`
	CincoPorques        *[]string            `json:"cinco_porques,omitempty"`
	Ishikawa            *map[string][]string `json:"ishikawa,omitempty"`
	CausaRaiz           *string              `json:"causa_raiz,omitempty"`
	PontoDeEscape       *string              `json:"ponto_de_escape,omitempty"`
	Prevencao           *string              `json:"prevencao,omitempty"`
	Equipe              *[]MembroDTO         `json:"equipe,omitempty"`
}

type EncerramentoDTO struct {
	Eficaz      bool    `json:"eficaz"`
	Verificacao string  `json:"verificacao"`
	Licoes      *string `json:"licoes,omitempty"`
}

type AcaoDTO struct {
	Tipo        string `json:"tipo"`
	Descricao   string `json:"descricao"`
	Responsavel string `json:"responsavel"`
	Prazo       string `json:"prazo"`
}

type AndamentoDTO struct {
	Status      string  `json:"status"`
	Evidencia   *string `json:"evidencia,omitempty"`
	Prazo       *string `json:"prazo,omitempty"`
	Responsavel *string `json:"responsavel,omitempty"`
}

// CapaResponse: a CAPA com o que falta para avançar.
type CapaResponse struct {
	*dom.Capa
	NomeEtapa  string   `json:"nome_etapa"`
	Pendencias []string `json:"pendencias"`
}

// ─── auxiliares ───────────────────────────────────────────────────────────────

func (uc *UseCase) ator(ctx context.Context) (uuid.UUID, error) {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return uuid.Nil, errorsuc.ErrUnauthorized
	}
	return a, nil
}

func data(s, campo string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s inválido (use AAAA-MM-DD)", campo))
	}
	return &t, nil
}

func usuario(s, campo string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return uuid.Nil, errorsuc.NewValidationError(fmt.Sprintf("%s inválido", campo))
	}
	return id, nil
}

// daEmpresa confere que os usuários são da empresa (ativos) e devolve os nomes.
func (uc *UseCase) daEmpresa(ctx context.Context, ids ...uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	nomes, err := uc.Repo.UsuariosDaEmpresa(ctx, ids)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, ok := nomes[id]; !ok {
			return errorsuc.NewValidationError("o usuário indicado não é desta empresa (ou está inativo)")
		}
	}
	return nil
}

var papeis = map[string]bool{"LIDER": true, "MEMBRO": true, "PATROCINADOR": true}

func (uc *UseCase) equipe(ctx context.Context, dtos []MembroDTO) ([]dom.Membro, error) {
	out := make([]dom.Membro, 0, len(dtos))
	visto := map[uuid.UUID]bool{}
	ids := []uuid.UUID{}
	lideres := 0
	for _, m := range dtos {
		id, err := usuario(m.UserID, "membro da equipe")
		if err != nil {
			return nil, err
		}
		p := strings.ToUpper(strings.TrimSpace(m.Papel))
		if p == "" {
			p = "MEMBRO"
		}
		if !papeis[p] {
			return nil, errorsuc.NewValidationError("papel inválido (LIDER, MEMBRO ou PATROCINADOR)")
		}
		if visto[id] {
			return nil, errorsuc.NewValidationError("a mesma pessoa aparece duas vezes na equipe")
		}
		if p == "LIDER" {
			lideres++
		}
		visto[id] = true
		ids = append(ids, id)
		out = append(out, dom.Membro{UserID: id, Papel: p})
	}
	if lideres > 1 {
		return nil, errorsuc.NewValidationError("a equipe tem um líder só")
	}
	return out, uc.daEmpresa(ctx, ids...)
}

func resposta(c *dom.Capa) *CapaResponse {
	p := []string{}
	if c.Status == dom.Aberta && c.Etapa != c.Metodo.EtapaFinal() {
		p = c.Pendencias()
		if p == nil {
			p = []string{}
		}
	}
	return &CapaResponse{Capa: c, NomeEtapa: dom.NomeDaEtapa[c.Etapa], Pendencias: p}
}

func limpo(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

// ─── CAPA ─────────────────────────────────────────────────────────────────────

func (uc *UseCase) Abrir(ctx context.Context, dto AberturaDTO) (*CapaResponse, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	resp := ator
	if strings.TrimSpace(dto.Responsavel) != "" {
		if resp, err = usuario(dto.Responsavel, "responsável"); err != nil {
			return nil, err
		}
	}
	prazo, err := data(dto.Prazo, "prazo")
	if err != nil {
		return nil, err
	}
	eq, err := uc.equipe(ctx, dto.Equipe)
	if err != nil {
		return nil, err
	}
	if err := uc.daEmpresa(ctx, resp); err != nil {
		return nil, err
	}
	c := &dom.Capa{
		Metodo: dom.Metodo(strings.ToUpper(strings.TrimSpace(dto.Metodo))), Tipo: strings.ToUpper(strings.TrimSpace(dto.Tipo)),
		Origem: strings.ToUpper(strings.TrimSpace(dto.Origem)), Titulo: dto.Titulo, ItemCode: dto.ItemCode, SupplierCode: dto.SupplierCode,
		CustomerCode: dto.CustomerCode, Referencia: limpo(dto.Referencia), Responsavel: resp, Prazo: prazo, NCs: dto.NCs, Equipe: eq,
	}
	if c.Metodo == "" {
		c.Metodo = dom.Metodo8D
	}
	if err := c.ValidarAbertura(); err != nil {
		return nil, err
	}
	if err := uc.Repo.Abrir(ctx, c, ator); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, c.ID)
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*CapaResponse, error) {
	c, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	return resposta(c), nil
}

// Listar: minhas = responsável ou na equipe; atrasadas = prazo da CAPA ou de
// alguma ação vencido.
func (uc *UseCase) Listar(ctx context.Context, status string, minhas, atrasadas bool) ([]dom.Capa, error) {
	f := dom.Filtro{Status: strings.ToUpper(status), Atrasadas: atrasadas, Hoje: uc.agora()}
	if minhas {
		a, err := uc.ator(ctx)
		if err != nil {
			return nil, err
		}
		f.Responsavel = &a
	}
	return uc.Repo.Listar(ctx, f)
}

func (uc *UseCase) Analisar(ctx context.Context, id int64, dto AnaliseDTO) (*CapaResponse, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	var eq *[]dom.Membro
	if dto.Equipe != nil {
		m, err := uc.equipe(ctx, *dto.Equipe)
		if err != nil {
			return nil, err
		}
		eq = &m
	}
	var resp *uuid.UUID
	if dto.Responsavel != nil {
		r, err := usuario(*dto.Responsavel, "responsável")
		if err != nil {
			return nil, err
		}
		if err := uc.daEmpresa(ctx, r); err != nil {
			return nil, err
		}
		resp = &r
	}
	if dto.MetodoCausa != nil && *dto.MetodoCausa != "5PORQUES" && *dto.MetodoCausa != "ISHIKAWA" {
		return nil, errorsuc.NewValidationError("método da causa raiz inválido (5PORQUES ou ISHIKAWA)")
	}
	if dto.Ishikawa != nil {
		validas := map[string]bool{}
		for _, k := range dom.CategoriasIshikawa {
			validas[k] = true
		}
		for k := range *dto.Ishikawa {
			if !validas[k] {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("categoria do Ishikawa inválida: %s (use %s)", k, strings.Join(dom.CategoriasIshikawa, ", ")))
			}
		}
	}
	var prazo *time.Time
	if dto.Prazo != nil {
		if prazo, err = data(*dto.Prazo, "prazo"); err != nil {
			return nil, err
		}
	}
	c, err := uc.Repo.Atualizar(ctx, id, ator, func(c *dom.Capa) (*dom.Evento, error) {
		if c.Status != dom.Aberta {
			return nil, errorsuc.NewValidationError("a CAPA não está aberta")
		}
		if dto.Titulo != nil {
			if t := strings.TrimSpace(*dto.Titulo); t != "" {
				c.Titulo = t
			}
		}
		if resp != nil {
			c.Responsavel = *resp
		}
		if dto.Prazo != nil {
			c.Prazo = prazo
		}
		if dto.Problema != nil {
			c.Problema = *dto.Problema
		}
		if dto.ContencaoDispensada != nil {
			c.ContencaoDispensada = limpo(dto.ContencaoDispensada)
		}
		if dto.MetodoCausa != nil {
			c.MetodoCausa = dto.MetodoCausa
		}
		if dto.CincoPorques != nil {
			c.CincoPorques = *dto.CincoPorques
		}
		if dto.Ishikawa != nil {
			c.Ishikawa = *dto.Ishikawa
		}
		if dto.CausaRaiz != nil {
			c.CausaRaiz = limpo(dto.CausaRaiz)
		}
		if dto.PontoDeEscape != nil {
			c.PontoDeEscape = limpo(dto.PontoDeEscape)
		}
		if dto.Prevencao != nil {
			c.Prevencao = limpo(dto.Prevencao)
		}
		if eq != nil {
			c.Equipe = *eq
		}
		etapa := c.Etapa
		return &dom.Evento{Evento: "ATUALIZADA", De: &etapa}, nil
	})
	if err != nil {
		return nil, err
	}
	return resposta(c), nil
}

func (uc *UseCase) Avancar(ctx context.Context, id int64) (*CapaResponse, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	c, err := uc.Repo.Atualizar(ctx, id, ator, func(c *dom.Capa) (*dom.Evento, error) {
		de, para, err := c.Avancar()
		if err != nil {
			return nil, err
		}
		return &dom.Evento{Evento: "ETAPA", De: &de, Para: &para}, nil
	})
	if err != nil {
		return nil, err
	}
	return resposta(c), nil
}

// Encerrar registra a eficácia (rota APROVAR). Ineficaz volta à causa raiz.
func (uc *UseCase) Encerrar(ctx context.Context, id int64, dto EncerramentoDTO) (*CapaResponse, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	agora := uc.agora()
	c, err := uc.Repo.Atualizar(ctx, id, ator, func(c *dom.Capa) (*dom.Evento, error) {
		de := c.Etapa
		encerrou, err := c.Encerrar(dto.Eficaz, dto.Verificacao, limpo(dto.Licoes), ator, agora)
		if err != nil {
			return nil, err
		}
		v := strings.TrimSpace(dto.Verificacao)
		if encerrou {
			return &dom.Evento{Evento: "ENCERRADA", De: &de, Detalhe: &v}, nil
		}
		para := c.Etapa
		d := fmt.Sprintf("ineficaz — ciclo %d: %s", c.Ciclo, v)
		return &dom.Evento{Evento: "INEFICAZ", De: &de, Para: &para, Detalhe: &d}, nil
	})
	if err != nil {
		return nil, err
	}
	return resposta(c), nil
}

func (uc *UseCase) Cancelar(ctx context.Context, id int64, motivo string) (*CapaResponse, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do cancelamento")
	}
	agora := uc.agora()
	c, err := uc.Repo.Atualizar(ctx, id, ator, func(c *dom.Capa) (*dom.Evento, error) {
		if c.Status != dom.Aberta {
			return nil, errorsuc.NewValidationError("a CAPA não está aberta")
		}
		de := c.Etapa
		c.Status, c.ClosedAt, c.ClosedBy = dom.Cancelada, &agora, &ator
		return &dom.Evento{Evento: "CANCELADA", De: &de, Detalhe: &motivo}, nil
	})
	if err != nil {
		return nil, err
	}
	return resposta(c), nil
}

// ─── ações ────────────────────────────────────────────────────────────────────

func (uc *UseCase) AdicionarAcao(ctx context.Context, capaID int64, dto AcaoDTO) (*CapaResponse, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := usuario(dto.Responsavel, "responsável pela ação")
	if err != nil {
		return nil, err
	}
	if err := uc.daEmpresa(ctx, resp); err != nil {
		return nil, err
	}
	prazo, err := data(dto.Prazo, "prazo da ação")
	if err != nil {
		return nil, err
	}
	a := &dom.Acao{Tipo: strings.ToUpper(strings.TrimSpace(dto.Tipo)), Descricao: dto.Descricao, Responsavel: resp}
	if prazo != nil {
		a.Prazo = *prazo
	}
	if err := dom.ValidarAcao(a); err != nil {
		return nil, err
	}
	if err := uc.Repo.AdicionarAcao(ctx, capaID, a, ator); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, capaID)
}

var andamentos = map[string]bool{"PENDENTE": true, "EM_ANDAMENTO": true, "CONCLUIDA": true, "CANCELADA": true}

// AtualizarAcao muda o andamento da ação. Concluir exige evidência; ação de
// ciclo anterior (CAPA reaberta por ineficácia) não muda mais.
func (uc *UseCase) AtualizarAcao(ctx context.Context, capaID, acaoID int64, dto AndamentoDTO) (*CapaResponse, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	status := strings.ToUpper(strings.TrimSpace(dto.Status))
	if status != "" && !andamentos[status] {
		return nil, errorsuc.NewValidationError("situação da ação inválida (PENDENTE, EM_ANDAMENTO, CONCLUIDA ou CANCELADA)")
	}
	var resp *uuid.UUID
	if dto.Responsavel != nil {
		r, err := usuario(*dto.Responsavel, "responsável pela ação")
		if err != nil {
			return nil, err
		}
		if err := uc.daEmpresa(ctx, r); err != nil {
			return nil, err
		}
		resp = &r
	}
	var prazo *time.Time
	if dto.Prazo != nil {
		if prazo, err = data(*dto.Prazo, "prazo da ação"); err != nil {
			return nil, err
		}
	}
	agora := uc.agora()
	c, err := uc.Repo.AtualizarAcao(ctx, capaID, acaoID, ator, func(c *dom.Capa, a *dom.Acao) (*dom.Evento, error) {
		if c.Status != dom.Aberta {
			return nil, errorsuc.NewValidationError("a CAPA não está aberta")
		}
		if a.Ciclo != c.Ciclo {
			return nil, errorsuc.NewValidationError("esta ação é de um ciclo anterior da CAPA e não muda mais")
		}
		if a.Status == "CONCLUIDA" || a.Status == "CANCELADA" {
			return nil, errorsuc.NewValidationError("a ação já está " + strings.ToLower(a.Status))
		}
		if resp != nil {
			a.Responsavel = *resp
		}
		if prazo != nil {
			a.Prazo = *prazo
		}
		if status == "" {
			status = a.Status
		}
		if status == "CONCLUIDA" {
			ev := limpo(dto.Evidencia)
			if ev == nil {
				return nil, errorsuc.NewValidationError("informe a evidência da conclusão (o que foi feito e onde está o registro)")
			}
			a.Evidencia, a.ConcluidaEm, a.ConcluidaPor = ev, &agora, &ator
		}
		a.Status = status
		d := fmt.Sprintf("%s \"%s\": %s", strings.ToLower(a.Tipo), a.Descricao, strings.ToLower(status))
		return &dom.Evento{Evento: "ACAO", Detalhe: &d}, nil
	})
	if err != nil {
		return nil, err
	}
	return resposta(c), nil
}

// AcoesPendentes: as do usuário (minhas) ou todas.
func (uc *UseCase) AcoesPendentes(ctx context.Context, minhas bool) ([]dom.AcaoPendente, error) {
	if !minhas {
		return uc.Repo.AcoesPendentes(ctx, nil)
	}
	a, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Repo.AcoesPendentes(ctx, &a)
}

func (uc *UseCase) VincularNC(ctx context.Context, capaID, ncID int64) (*CapaResponse, error) {
	ator, err := uc.ator(ctx)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.VincularNC(ctx, capaID, ncID, ator); err != nil {
		return nil, err
	}
	return uc.Obter(ctx, capaID)
}

func (uc *UseCase) NCs(ctx context.Context, f dom.FiltroNC) ([]dom.NC, error) {
	return uc.Repo.NCs(ctx, f)
}

func (uc *UseCase) Usuarios(ctx context.Context) ([]dom.Membro, error) { return uc.Repo.Usuarios(ctx) }
