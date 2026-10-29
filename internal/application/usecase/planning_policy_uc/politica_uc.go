// Package planning_policy_uc é a política de planejamento por item ou por
// classificação (VPLA0310). Ver .ai/tasks/politica-planejamento-enterprise.md.
package planning_policy_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
)

// Parametros lê os parâmetros do planejamento (46 a 49), que a política herda.
type Parametros interface {
	LoadTypedPlanningParams(ctx context.Context) (*entity.TypedPlanningParams, error)
}

type UseCase struct {
	Repo   repository.PoliticaRepository
	Params Parametros
	Auth   ports.AuthService
	// Items resolve o código de negócio do item.
	Items any
}

func (uc *UseCase) Listar(ctx context.Context, escopo, busca string) ([]*response.PoliticaPlanejamentoResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	escopo = strings.ToUpper(strings.TrimSpace(escopo))
	if escopo != "" && escopo != "ITEM" && escopo != "CLASSIFICACAO" {
		return nil, errorsuc.NewValidationError("escopo inválido: use ITEM ou CLASSIFICACAO")
	}
	ps, err := uc.Repo.Listar(ctx, entity.FiltroPolitica{Escopo: escopo, Busca: busca})
	if err != nil {
		return nil, err
	}
	out := make([]*response.PoliticaPlanejamentoResponse, 0, len(ps))
	for _, p := range ps {
		out = append(out, resposta(p))
	}
	return out, nil
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*response.PoliticaPlanejamentoResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	return resposta(p), nil
}

func (uc *UseCase) Criar(ctx context.Context, dto request.PoliticaPlanejamentoDTO) (*response.PoliticaPlanejamentoResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	autor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	p := campos(dto)
	p.CreatedBy = autor
	temItem := strings.TrimSpace(dto.ItemCode.String()) != ""
	switch {
	case temItem && dto.ClassificationID != nil:
		return nil, errorsuc.NewValidationError("a política é de um item ou de uma classificação, não dos dois")
	case temItem:
		it, err := itemresolution.ResolveActive(ctx, uc.Items, dto.ItemCode)
		if err != nil {
			return nil, err
		}
		code := int64(it.Code)
		p.ItemCode = &code
	case dto.ClassificationID != nil:
		c, err := uc.Repo.Classificacao(ctx, *dto.ClassificationID)
		if err != nil {
			return nil, err
		}
		p.ClassificationID = &c.ID
	}
	if msg := p.Validar(); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	criada, err := uc.Repo.Criar(ctx, p)
	if err != nil {
		return nil, err
	}
	return resposta(criada), nil
}

// Alterar troca os campos; o escopo (item ou classificação) não muda.
func (uc *UseCase) Alterar(ctx context.Context, id int64, dto request.PoliticaPlanejamentoDTO) (*response.PoliticaPlanejamentoResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	autor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	atual, err := uc.Repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	p := campos(dto)
	p.ID, p.ItemCode, p.ClassificationID, p.UpdatedBy = id, atual.ItemCode, atual.ClassificationID, &autor
	if msg := p.Validar(); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	alterada, err := uc.Repo.Alterar(ctx, p)
	if err != nil {
		return nil, err
	}
	return resposta(alterada), nil
}

func (uc *UseCase) Excluir(ctx context.Context, id int64) error {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Excluir(ctx, id)
}

// Efetiva é a política que vale para o item hoje, com a origem de cada valor.
func (uc *UseCase) Efetiva(ctx context.Context, item request.TextCode) (*response.PoliticaEfetivaResponse, error) {
	if !uc.Auth.CanManagePlanningParams(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	it, err := itemresolution.Resolve(ctx, uc.Items, item)
	if err != nil {
		return nil, err
	}
	params, err := uc.Params.LoadTypedPlanningParams(ctx)
	if err != nil {
		return nil, err
	}
	ef, err := uc.Repo.PoliticaDoItem(ctx, int64(it.Code), params.PoliticaDaEmpresa())
	if err != nil {
		return nil, err
	}
	return &response.PoliticaEfetivaResponse{
		ItemBusinessCode: string(it.BusinessCode), ItemName: it.Name,
		Estrategia: ef.Estrategia, EstrategiaDescricao: DescricaoDaEstrategia(ef.Estrategia),
		ConsumoDiasAntes: ef.ConsumoDiasAntes, ConsumoDiasDepois: ef.ConsumoDiasDepois,
		HorizonteFirmeDias: ef.HorizonteFirmeDias, TempoSegurancaDias: ef.TempoSegurancaDias,
		Origem: response.OrigemPoliticaResponse{
			Estrategia: entity.DescreverOrigem(ef.Origem.Estrategia), Janela: entity.DescreverOrigem(ef.Origem.Janela),
			HorizonteFirme: entity.DescreverOrigem(ef.Origem.HorizonteFirme), TempoSeguranca: entity.DescreverOrigem(ef.Origem.TempoSeguranca),
		},
	}, nil
}

// DescricaoDaEstrategia é a estratégia como o usuário lê.
func DescricaoDaEstrategia(e string) string {
	if e == entity.EstrategiaSobEncomenda {
		return "Sob encomenda: só o pedido de venda gera necessidade"
	}
	return "Consumo: o pedido de venda consome a previsão e a demanda de estoque"
}

// campos copia os campos da política; estratégia vazia herda.
func campos(dto request.PoliticaPlanejamentoDTO) *entity.PoliticaPlanejamento {
	p := &entity.PoliticaPlanejamento{
		ConsumoDiasAntes: dto.ConsumoDiasAntes, ConsumoDiasDepois: dto.ConsumoDiasDepois,
		HorizonteFirmeDias: dto.HorizonteFirmeDias, TempoSegurancaDias: dto.TempoSegurancaDias,
	}
	if dto.Estrategia != nil {
		if e := strings.ToUpper(strings.TrimSpace(*dto.Estrategia)); e != "" {
			p.Estrategia = &e
		}
	}
	if dto.Observacao != nil {
		if o := strings.TrimSpace(*dto.Observacao); o != "" {
			p.Observacao = &o
		}
	}
	return p
}

func resposta(p *entity.PoliticaPlanejamento) *response.PoliticaPlanejamentoResponse {
	escopo := "CLASSIFICACAO"
	if p.ItemCode != nil {
		escopo = "ITEM"
	}
	return &response.PoliticaPlanejamentoResponse{
		ID: p.ID, Escopo: escopo, ItemCode: p.ItemCode, ItemBusinessCode: p.ItemBusinessCode, ItemName: p.ItemName,
		ClassificationID: p.ClassificationID, ClassificationCode: p.ClassificationCode, ClassificationLabel: p.ClassificationLabel,
		Estrategia: p.Estrategia, ConsumoDiasAntes: p.ConsumoDiasAntes, ConsumoDiasDepois: p.ConsumoDiasDepois,
		HorizonteFirmeDias: p.HorizonteFirmeDias, TempoSegurancaDias: p.TempoSegurancaDias, Observacao: p.Observacao,
		UpdatedAt: p.UpdatedAt,
	}
}
