package routing_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
)

type fakeRouteOps struct {
	repository.RoutingRepository
	ops    []*entity.RouteOperation
	gravou *entity.RouteOperation
}

func (f *fakeRouteOps) RouteIDOfOperation(context.Context, int64) (int64, error) { return 1, nil }
func (f *fakeRouteOps) GetRouteOperations(context.Context, int64) ([]*entity.RouteOperation, error) {
	return f.ops, nil
}
func (f *fakeRouteOps) GetNetworkEdges(context.Context, int64) ([]*entity.NetworkEdge, error) {
	return nil, nil
}
func (f *fakeRouteOps) UpdateRouteOperation(_ context.Context, op *entity.RouteOperation) (*entity.RouteOperation, error) {
	f.gravou = op
	return op, nil
}

// Alterar a etapa sem informar a situação mantém a atual — antes ia vazio para
// o banco, e um padrão APROVADA reativaria em silêncio uma etapa inativada.
func TestAlterarEtapaMantemSituacao(t *testing.T) {
	repo := &fakeRouteOps{ops: []*entity.RouteOperation{{ID: 7, Situation: entity.RouteOpInactive}}}
	uc := NewRouteUseCase(repo)
	setup := 0.3
	if _, err := uc.UpdateOperation(context.Background(), request.UpdateRouteOperationDTO{ID: 7, SetupTime: &setup}); err != nil {
		t.Fatal(err)
	}
	if repo.gravou == nil || repo.gravou.Situation != entity.RouteOpInactive {
		t.Fatalf("situação gravada = %+v; esperado manter INATIVA", repo.gravou)
	}

	repo.gravou = nil
	_, err := uc.UpdateOperation(context.Background(), request.UpdateRouteOperationDTO{ID: 7, Situation: "XPTO"})
	if err == nil || !strings.Contains(err.Error(), "APROVADA") || repo.gravou != nil {
		t.Fatalf("situação inválida: err=%v gravou=%+v", err, repo.gravou)
	}

	// Etapa que não está no roteiro: não encontrada, nada gravado.
	_, err = uc.UpdateOperation(context.Background(), request.UpdateRouteOperationDTO{ID: 99})
	if err == nil || repo.gravou != nil {
		t.Fatalf("etapa inexistente: err=%v gravou=%+v", err, repo.gravou)
	}
}
