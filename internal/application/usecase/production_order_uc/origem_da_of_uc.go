package production_order_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	projentity "github.com/FelipePn10/panossoerp/internal/domain/project/entity"
)

// identificadorDeOFs lê, numa consulta, a identificação de várias OFs.
type identificadorDeOFs interface {
	IdentificacaoDasOFs(ctx context.Context, ids []int64) (map[int64]entity.IdentificacaoOF, error)
}

type origemDaOFRepo interface {
	DefinirOrigem(ctx context.Context, ordem int64, cliente, projeto *int64) error
	ClienteDaEmpresa(ctx context.Context, cliente int64) (string, bool, error)
}

// LeitorDeProjetos acha o projeto da empresa pelo código.
type LeitorDeProjetos interface {
	GetByCode(ctx context.Context, code string) (*projentity.Project, error)
}

// identificar completa as respostas com a origem de cada OF. Repositório sem
// a consulta (dublê de teste) devolve as OFs como estão.
func identificar(ctx context.Context, repo repository.ProductionOrderRepository, lista []*response.ProductionOrderResponse) error {
	ident, ok := repo.(identificadorDeOFs)
	if !ok || len(lista) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(lista))
	for _, o := range lista {
		if o != nil {
			ids = append(ids, o.ID)
		}
	}
	mapa, err := ident.IdentificacaoDasOFs(ctx, ids)
	if err != nil {
		return err
	}
	for _, o := range lista {
		if o == nil {
			continue
		}
		if x, ok := mapa[o.ID]; ok {
			o.Origin = toOrigin(x)
		}
	}
	return nil
}

func toOrigin(x entity.IdentificacaoOF) *response.ProductionOrderOrigin {
	o := &response.ProductionOrderOrigin{
		ItemBusinessCode: x.ItemBusinessCode, ItemName: x.ItemName,
		CustomerCode: x.CustomerCode, CustomerName: x.CustomerName,
		ProjectID: x.ProjectID, ProjectCode: x.ProjectCode, ProjectName: x.ProjectName,
		SalesOrders: x.SalesOrders, Demands: x.Demands,
		ParentOrderID: x.ParentOrderID, ParentOrderNumber: x.ParentOrderNumber, ParentPlannedOrder: x.ParentPlannedCode,
		ParentItem: x.ParentItemBusiness, Level: x.Level, Children: x.Children,
	}
	if o.SalesOrders == nil {
		o.SalesOrders = []int64{}
	}
	if o.Demands == nil {
		o.Demands = []int64{}
	}
	// A raiz só interessa quando não é a própria OF.
	if x.Level > 0 {
		o.RootOrderID, o.RootOrderNumber, o.RootItem = x.RootOrderID, x.RootOrderNumber, x.RootItemBusiness
	}
	return o
}

// DefinirOrigemDaOFUseCase grava cliente e projeto de uma OF. É classificação
// (não muda quantidade nem material), por isso vale também com a OF em
// andamento; OF encerrada ou cancelada fica como está.
type DefinirOrigemDaOFUseCase struct {
	Repo     repository.ProductionOrderRepository
	Auth     ports.AuthService
	Projetos LeitorDeProjetos
}

func (uc *DefinirOrigemDaOFUseCase) Execute(ctx context.Context, id int64, dto request.ProductionOrderOriginDTO) (*response.ProductionOrderResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	repo, ok := uc.Repo.(origemDaOFRepo)
	if !ok {
		return nil, errorsuc.NewValidationError("cliente e projeto da OF não estão disponíveis")
	}
	ordem, err := uc.Repo.GetByCode(ctx, id)
	if err != nil {
		return nil, err
	}
	if ordem.Status == entity.StatusClosed || ordem.Status == entity.StatusCancelled {
		return nil, errorsuc.NewValidationError("OF encerrada ou cancelada não muda de cliente nem de projeto")
	}
	cliente, projeto, err := resolverOrigem(ctx, repo, uc.Projetos, dto.CustomerCode, dto.ProjectCode)
	if err != nil {
		return nil, err
	}
	if err := repo.DefinirOrigem(ctx, id, cliente, projeto); err != nil {
		return nil, err
	}
	out := toProductionOrderResponse(ordem)
	if err := identificar(ctx, uc.Repo, []*response.ProductionOrderResponse{out}); err != nil {
		return nil, err
	}
	return out, nil
}

// resolverOrigem confere o cliente (da empresa) e acha o projeto pelo código
// (ativo). Cliente 0/vazio e projeto "" = sem.
func resolverOrigem(ctx context.Context, repo origemDaOFRepo, projetos LeitorDeProjetos, clienteIn *int64, projetoIn *string) (*int64, *int64, error) {
	var cliente, projeto *int64
	if clienteIn != nil && *clienteIn > 0 {
		_, existe, err := repo.ClienteDaEmpresa(ctx, *clienteIn)
		if err != nil {
			return nil, nil, err
		}
		if !existe {
			return nil, nil, errorsuc.NewValidationError("cliente não encontrado nesta empresa")
		}
		c := *clienteIn
		cliente = &c
	}
	if projetoIn != nil && strings.TrimSpace(*projetoIn) != "" {
		if projetos == nil {
			return nil, nil, errorsuc.NewValidationError("cadastro de projetos indisponível")
		}
		p, err := projetos.GetByCode(ctx, *projetoIn)
		if err != nil {
			if errorsuc.EhNaoEncontrado(err) {
				return nil, nil, errorsuc.NewValidationError("projeto " + strings.TrimSpace(*projetoIn) + " não está cadastrado (VPLA0120)")
			}
			return nil, nil, err
		}
		if p.Status != projentity.StatusAtivo {
			return nil, nil, errorsuc.NewValidationError("o projeto " + p.Code + " está " + strings.ToLower(p.Status) + " e não recebe OF")
		}
		// Projeto com cliente: a OF sem cliente leva o do projeto; com outro
		// cliente é engano de digitação.
		if p.CustomerCode != nil {
			if cliente == nil {
				c := *p.CustomerCode
				cliente = &c
			} else if *cliente != *p.CustomerCode {
				return nil, nil, errorsuc.NewValidationError("o projeto " + p.Code + " é de outro cliente (" + p.CustomerName + ")")
			}
		}
		id := p.ID
		projeto = &id
	}
	return cliente, projeto, nil
}
