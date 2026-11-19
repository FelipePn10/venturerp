// Package project_uc mantém o cadastro de Projetos (VPLA0120).
package project_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/project/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/project/repository"
)

type UseCase struct {
	Repo repository.ProjectRepository
	Auth ports.AuthService
}

func (uc *UseCase) List(ctx context.Context, busca, status string) ([]*response.ProjectResponse, error) {
	lista, err := uc.Repo.List(ctx, repository.Filtro{Busca: busca, Status: status})
	if err != nil {
		return nil, err
	}
	out := make([]*response.ProjectResponse, 0, len(lista))
	for _, p := range lista {
		out = append(out, toResponse(p))
	}
	return out, nil
}

func (uc *UseCase) Get(ctx context.Context, code string) (*response.ProjectResponse, error) {
	p, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toResponse(p), nil
}

func (uc *UseCase) Create(ctx context.Context, dto request.ProjectDTO) (*response.ProjectResponse, error) {
	p, err := fromDTO(dto)
	if err != nil {
		return nil, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	p.CreatedBy = &ator
	salvo, err := uc.Repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	return toResponse(salvo), nil
}

// Update altera o projeto achado pelo código atual (o código pode mudar).
func (uc *UseCase) Update(ctx context.Context, code string, dto request.ProjectDTO) (*response.ProjectResponse, error) {
	atual, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	p, err := fromDTO(dto)
	if err != nil {
		return nil, err
	}
	p.ID = atual.ID
	salvo, err := uc.Repo.Update(ctx, p)
	if err != nil {
		return nil, err
	}
	return toResponse(salvo), nil
}

func fromDTO(dto request.ProjectDTO) (*entity.Project, error) {
	p := &entity.Project{Code: dto.Code, Name: dto.Name, CustomerCode: dto.CustomerCode, Status: dto.Status, Notes: dto.Notes}
	var err error
	if p.StartDate, err = data(dto.StartDate, "início"); err != nil {
		return nil, err
	}
	if p.EndDate, err = data(dto.EndDate, "término"); err != nil {
		return nil, err
	}
	if err := p.Normalizar(); err != nil {
		return nil, err
	}
	return p, nil
}

func data(s, rotulo string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if len(s) > 10 {
		s = s[:10]
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, errorsuc.NewValidationError("data de " + rotulo + " do projeto inválida; use AAAA-MM-DD")
	}
	return &t, nil
}

func toResponse(p *entity.Project) *response.ProjectResponse {
	r := &response.ProjectResponse{ID: p.ID, Code: p.Code, Name: p.Name, CustomerCode: p.CustomerCode, CustomerName: p.CustomerName,
		Status: p.Status, Notes: p.Notes, OpenOrders: p.OpenOrders, TotalOrders: p.TotalOrders}
	if p.StartDate != nil {
		r.StartDate = p.StartDate.Format("2006-01-02")
	}
	if p.EndDate != nil {
		r.EndDate = p.EndDate.Format("2006-01-02")
	}
	return r
}
