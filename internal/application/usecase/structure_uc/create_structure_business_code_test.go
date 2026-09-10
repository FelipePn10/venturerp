package structure_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	itemvo "github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structrepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
	"github.com/google/uuid"
)

type structureBusinessCodeAuth struct{ ports.AuthService }

func (structureBusinessCodeAuth) CanCreateStructure(context.Context) bool { return true }
func (structureBusinessCodeAuth) UserID(context.Context) (uuid.UUID, error) {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111"), nil
}

type structureBusinessCodeItems struct {
	items map[itemvo.BusinessCode]*itementity.Item
}

func (f structureBusinessCodeItems) FindItemByBusinessCode(_ context.Context, code itemvo.BusinessCode) (*itementity.Item, error) {
	item := f.items[code]
	if item == nil {
		return nil, itemrepo.ErrNotFound
	}
	return item, nil
}

type structureBusinessCodeRepo struct {
	structrepo.ItemStructureRepository
	created     *entity.ItemStructure
	hasCycle    bool
	cycleStart  int64
	cycleTarget int64
}

func (r *structureBusinessCodeRepo) ItemExists(context.Context, int64) (bool, error) {
	return true, nil
}
func (r *structureBusinessCodeRepo) HasCyclicReference(_ context.Context, startCode, targetCode int64) (bool, error) {
	r.cycleStart = startCode
	r.cycleTarget = targetCode
	return r.hasCycle, nil
}
func (r *structureBusinessCodeRepo) SequenceExists(context.Context, int64, int) (bool, error) {
	return false, nil
}
func (r *structureBusinessCodeRepo) Create(_ context.Context, component *entity.ItemStructure) (*entity.ItemStructure, error) {
	r.created = component
	component.ID = 1
	return component, nil
}

func TestCreateStructureAcceptsAlphanumericBusinessCodes(t *testing.T) {
	items := structureBusinessCodeItems{items: map[itemvo.BusinessCode]*itementity.Item{
		"RN-01001":   {Code: 101, BusinessCode: "RN-01001"},
		"TP-01001-A": {Code: 202, BusinessCode: "TP-01001-A"},
	}}
	repo := &structureBusinessCodeRepo{}
	uc := NewCreateStructureComponentUseCase(repo, structureBusinessCodeAuth{}, items)

	component, err := uc.Execute(context.Background(), request.CreateStructureComponentDTO{
		ParentCode:        "RN-01001",
		ChildCode:         "TP-01001-A",
		Quantity:          2,
		UnitOfMeasurement: "UN",
		Health:            "ATIVO",
		LossPercentage:    21,
		QuantityRounding:  "NONE",
		QuantityScale:     4,
		Sequence:          1,
		IsActive:          true,
	})
	if err != nil {
		t.Fatalf("criação com códigos comerciais falhou: %v", err)
	}
	if component.ParentCode != 101 || component.ChildCode != 202 {
		t.Fatalf("códigos internos resolvidos incorretamente: pai=%d filho=%d", component.ParentCode, component.ChildCode)
	}
	if repo.cycleStart != 202 || repo.cycleTarget != 101 {
		t.Fatalf("busca de ciclo invertida: início=%d alvo=%d", repo.cycleStart, repo.cycleTarget)
	}
}

func TestCreateStructureBlocksActualCycle(t *testing.T) {
	items := structureBusinessCodeItems{items: map[itemvo.BusinessCode]*itementity.Item{
		"RN-01001":   {Code: 101, BusinessCode: "RN-01001"},
		"TP-01001-A": {Code: 202, BusinessCode: "TP-01001-A"},
	}}
	repo := &structureBusinessCodeRepo{hasCycle: true}
	uc := NewCreateStructureComponentUseCase(repo, structureBusinessCodeAuth{}, items)

	_, err := uc.Execute(context.Background(), request.CreateStructureComponentDTO{
		ParentCode: "RN-01001", ChildCode: "TP-01001-A", Sequence: 1,
	})
	if err == nil || err.Error() != "o componente criaria um ciclo na estrutura" {
		t.Fatalf("ciclo real deveria ser bloqueado, erro=%v", err)
	}
	if repo.created != nil {
		t.Fatal("componente com ciclo não deveria ser criado")
	}
}

func TestCreateStructureBlocksSelfReference(t *testing.T) {
	items := structureBusinessCodeItems{items: map[itemvo.BusinessCode]*itementity.Item{
		"RN-01001": {Code: 101, BusinessCode: "RN-01001"},
	}}
	repo := &structureBusinessCodeRepo{}
	uc := NewCreateStructureComponentUseCase(repo, structureBusinessCodeAuth{}, items)

	_, err := uc.Execute(context.Background(), request.CreateStructureComponentDTO{
		ParentCode: "RN-01001", ChildCode: "RN-01001", Sequence: 1,
	})
	if err == nil || err.Error() != "o componente criaria um ciclo na estrutura" {
		t.Fatalf("autorreferência deveria ser bloqueada, erro=%v", err)
	}
	if repo.cycleStart != 0 || repo.cycleTarget != 0 {
		t.Fatal("autorreferência deveria ser bloqueada antes de consultar a árvore")
	}
}
