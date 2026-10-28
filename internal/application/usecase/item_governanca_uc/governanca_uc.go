// Package item_governanca_uc: governança do cadastro de item (Fase 4).
package item_governanca_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
)

// PermissaoDeArea: o usuário pode aprovar no módulo da área (perfis de acesso).
type PermissaoDeArea interface {
	PodeNoModulo(ctx context.Context, m acesso.Modulo, a acesso.Acao) (bool, error)
}

type UseCase struct {
	Repo  dom.Repository
	Auth  ports.AuthService
	Items any
	Areas PermissaoDeArea
}

var moduloDaVisao = map[string]acesso.Modulo{
	dom.Engenharia: acesso.Engenharia, dom.Fiscal: acesso.Fiscal, dom.Suprimentos: acesso.Suprimentos,
	dom.Planejamento: acesso.Planejamento, dom.Comercial: acesso.Comercial, dom.Custos: acesso.Custos,
}

func (uc *UseCase) ator(ctx context.Context) *uuid.UUID {
	a, err := uc.Auth.UserID(ctx)
	if err != nil || a == uuid.Nil {
		return nil
	}
	return &a
}

func (uc *UseCase) item(ctx context.Context, codigo string) (int64, error) {
	it, err := itemresolution.Resolve(ctx, uc.Items, request.TextCode(codigo))
	if err != nil {
		return 0, err
	}
	return int64(it.Code), nil
}

// ─── Parâmetros e regras ─────────────────────────────────────────────────────

func (uc *UseCase) Parametros(ctx context.Context) (dom.Parametros, error) {
	return uc.Repo.Parametros(ctx)
}

func (uc *UseCase) SalvarParametros(ctx context.Context, p dom.Parametros) (dom.Parametros, error) {
	visto := map[string]bool{}
	vs := []string{}
	for _, v := range p.Visoes {
		v = strings.ToUpper(strings.TrimSpace(v))
		if !dom.VisaoValida(v) {
			return p, errorsuc.NewValidationError(fmt.Sprintf("visão desconhecida: %s", v))
		}
		if !visto[v] {
			visto[v] = true
			vs = append(vs, v)
		}
	}
	if p.ExigeLiberacao && len(vs) == 0 {
		return p, errorsuc.NewValidationError("escolha ao menos uma visão que libera o cadastro")
	}
	p.Visoes = vs
	if err := uc.Repo.SalvarParametros(ctx, p, uc.ator(ctx)); err != nil {
		return p, err
	}
	return uc.Repo.Parametros(ctx)
}

type CampoRegra struct {
	dom.Campo
	Obrigatorio bool `json:"obrigatorio"`
	Ajustado    bool `json:"ajustado"`
}

func (uc *UseCase) Regras(ctx context.Context) ([]CampoRegra, error) {
	aj, err := uc.Repo.Ajustes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CampoRegra, 0, len(dom.Catalogo))
	for _, c := range dom.Catalogo {
		r := CampoRegra{Campo: c, Obrigatorio: c.Padrao}
		if v, ok := aj[c.Chave]; ok {
			r.Obrigatorio, r.Ajustado = v, true
		}
		out = append(out, r)
	}
	return out, nil
}

// SalvarRegra: obrigatorio nil volta ao padrão.
func (uc *UseCase) SalvarRegra(ctx context.Context, campo string, obrigatorio *bool) ([]CampoRegra, error) {
	if _, ok := dom.CampoDoCatalogo(campo); !ok {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("campo desconhecido: %s", campo))
	}
	if err := uc.Repo.SalvarAjuste(ctx, campo, obrigatorio, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.Regras(ctx)
}

// ─── Painel do item ──────────────────────────────────────────────────────────

type VisaoDoItem struct {
	Visao      string          `json:"visao"`
	Exigida    bool            `json:"exigida"`
	Liberada   *dom.Marca      `json:"liberada,omitempty"`
	Pendencias []dom.Pendencia `json:"pendencias"`
}

type Painel struct {
	*dom.Situacao
	TipoNome       string        `json:"tipo_nome"`
	ExigeLiberacao bool          `json:"exige_liberacao"`
	Visoes         []VisaoDoItem `json:"visoes"`
	Completo       bool          `json:"completo"`
}

func (uc *UseCase) painel(ctx context.Context, item int64) (*Painel, error) {
	s, err := uc.Repo.Situacao(ctx, item)
	if err != nil {
		return nil, err
	}
	f, err := uc.Repo.Ficha(ctx, item)
	if err != nil {
		return nil, err
	}
	p, err := uc.Repo.Parametros(ctx)
	if err != nil {
		return nil, err
	}
	aj, err := uc.Repo.Ajustes(ctx)
	if err != nil {
		return nil, err
	}
	exigida := map[string]bool{}
	for _, v := range p.Visoes {
		exigida[v] = true
	}
	out := &Painel{Situacao: s, TipoNome: dom.RotuloTipo(s.Tipo), ExigeLiberacao: p.ExigeLiberacao, Completo: true}
	for _, v := range dom.Visoes {
		vi := VisaoDoItem{Visao: v, Exigida: exigida[v], Pendencias: dom.Pendencias(*f, []string{v}, aj)}
		if m, ok := s.Liberadas[v]; ok {
			m := m
			vi.Liberada = &m
		}
		if len(vi.Pendencias) > 0 {
			out.Completo = false
		}
		out.Visoes = append(out.Visoes, vi)
	}
	return out, nil
}

func (uc *UseCase) Painel(ctx context.Context, codigo string) (*Painel, error) {
	item, err := uc.item(ctx, codigo)
	if err != nil {
		return nil, err
	}
	return uc.painel(ctx, item)
}

// AoCriarItem: com a liberação exigida, o item novo nasce EM_CADASTRO.
func (uc *UseCase) AoCriarItem(ctx context.Context, itemCode int64) error {
	p, err := uc.Repo.Parametros(ctx)
	if err != nil {
		return err
	}
	if !p.ExigeLiberacao {
		return nil
	}
	return uc.Repo.IniciarCadastro(ctx, itemCode, uc.ator(ctx))
}

func (uc *UseCase) conferirArea(ctx context.Context, visao string) error {
	if uc.Areas == nil {
		return nil
	}
	ok, err := uc.Areas.PodeNoModulo(ctx, moduloDaVisao[visao], acesso.Aprovar)
	if err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewValidationError(fmt.Sprintf("liberar a visão %s exige permissão de aprovar no módulo %s", strings.ToLower(visao), moduloDaVisao[visao]))
	}
	return nil
}

// Liberar a visão: só completa e por quem aprova na área. Com a última visão
// exigida, o item em cadastro fica liberado.
func (uc *UseCase) Liberar(ctx context.Context, codigo, visao string) (*Painel, error) {
	visao = strings.ToUpper(strings.TrimSpace(visao))
	if !dom.VisaoValida(visao) {
		return nil, errorsuc.NewValidationError("visão inválida")
	}
	if err := uc.conferirArea(ctx, visao); err != nil {
		return nil, err
	}
	item, err := uc.item(ctx, codigo)
	if err != nil {
		return nil, err
	}
	pn, err := uc.painel(ctx, item)
	if err != nil {
		return nil, err
	}
	for _, v := range pn.Visoes {
		if v.Visao == visao && len(v.Pendencias) > 0 {
			nomes := make([]string, 0, len(v.Pendencias))
			for _, x := range v.Pendencias {
				nomes = append(nomes, strings.ToLower(x.Rotulo))
			}
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a visão %s tem pendências: %s", strings.ToLower(visao), strings.Join(nomes, ", ")))
		}
	}
	liberadas := map[string]bool{visao: true}
	for v := range pn.Liberadas {
		liberadas[v] = true
	}
	exigidas := []string{}
	for _, v := range pn.Visoes {
		if v.Exigida {
			exigidas = append(exigidas, v.Visao)
		}
	}
	novo := dom.StatusAposLiberacao(pn.Status, exigidas, liberadas)
	if err := uc.Repo.Liberar(ctx, item, visao, novo, uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.painel(ctx, item)
}

func (uc *UseCase) EstornarLiberacao(ctx context.Context, codigo, visao, motivo string) (*Painel, error) {
	visao = strings.ToUpper(strings.TrimSpace(visao))
	if !dom.VisaoValida(visao) {
		return nil, errorsuc.NewValidationError("visão inválida")
	}
	if strings.TrimSpace(motivo) == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do estorno")
	}
	if err := uc.conferirArea(ctx, visao); err != nil {
		return nil, err
	}
	item, err := uc.item(ctx, codigo)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.EstornarLiberacao(ctx, item, visao, strings.TrimSpace(motivo), uc.ator(ctx)); err != nil {
		return nil, err
	}
	return uc.painel(ctx, item)
}

type StatusDTO struct {
	Status     string            `json:"status"`
	De         string            `json:"de"`
	Motivo     string            `json:"motivo"`
	Substituto *request.TextCode `json:"substituto,omitempty"`
}

// MudarStatus: bloquear, desbloquear, descontinuar (com substituto) ou reativar.
func (uc *UseCase) MudarStatus(ctx context.Context, codigo string, dto StatusDTO) (*Painel, error) {
	item, err := uc.item(ctx, codigo)
	if err != nil {
		return nil, err
	}
	m := dom.Mudanca{Para: strings.ToUpper(strings.TrimSpace(dto.Status)), Motivo: strings.TrimSpace(dto.Motivo), Por: uc.ator(ctx)}
	if dto.Substituto != nil && strings.TrimSpace(string(*dto.Substituto)) != "" {
		s, err := itemresolution.Resolve(ctx, uc.Items, *dto.Substituto)
		if err != nil {
			return nil, err
		}
		c := int64(s.Code)
		if c == item {
			return nil, errorsuc.NewValidationError("o item não substitui a si mesmo")
		}
		m.Substituto = &c
	}
	de := strings.ToUpper(strings.TrimSpace(dto.De))
	if de == "" {
		atual, err := uc.Repo.Situacao(ctx, item)
		if err != nil {
			return nil, err
		}
		de = atual.Status
	}
	if err := uc.Repo.MudarStatus(ctx, item, de, m); err != nil {
		return nil, err
	}
	return uc.painel(ctx, item)
}

func (uc *UseCase) Historico(ctx context.Context, codigo string, limite int) ([]dom.Evento, error) {
	item, err := uc.item(ctx, codigo)
	if err != nil {
		return nil, err
	}
	return uc.Repo.Historico(ctx, item, limite)
}

func (uc *UseCase) Uso(ctx context.Context, codigo string) (*dom.Uso, error) {
	item, err := uc.item(ctx, codigo)
	if err != nil {
		return nil, err
	}
	return uc.Repo.Uso(ctx, item)
}

// Pendentes: itens em cadastro com as visões que faltam.
type Pendente struct {
	dom.Situacao
	Faltam []string `json:"faltam"`
}

func (uc *UseCase) Pendentes(ctx context.Context) ([]Pendente, error) {
	p, err := uc.Repo.Parametros(ctx)
	if err != nil {
		return nil, err
	}
	ss, err := uc.Repo.EmCadastro(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Pendente, 0, len(ss))
	for _, s := range ss {
		pe := Pendente{Situacao: s, Faltam: []string{}}
		for _, v := range p.Visoes {
			if _, ok := s.Liberadas[v]; !ok {
				pe.Faltam = append(pe.Faltam, v)
			}
		}
		out = append(out, pe)
	}
	return out, nil
}

// Duplicados: itens parecidos com o nome (e o NCM) informados.
func (uc *UseCase) Duplicados(ctx context.Context, nome, ncm, excluir string) ([]dom.Candidato, error) {
	nome = strings.TrimSpace(nome)
	if len([]rune(nome)) < 3 {
		return []dom.Candidato{}, nil
	}
	var ex *int64
	if strings.TrimSpace(excluir) != "" {
		if c, err := uc.item(ctx, excluir); err == nil {
			ex = &c
		}
	}
	cands, err := uc.Repo.CandidatosParecidos(ctx, dom.Palavras(nome), ncm, ex)
	if err != nil {
		return nil, err
	}
	return dom.Classificar(nome, ncm, cands), nil
}
