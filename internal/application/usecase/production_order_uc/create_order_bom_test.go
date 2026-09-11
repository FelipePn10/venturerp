package production_order_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// estruturaPorMascara é o dublê que oferece a leitura por máscara, como o
// repositório real. Guarda a máscara pedida para o teste conferir.
type estruturaPorMascara struct {
	filhos        []*structentity.ItemStructure
	mascaraPedida string
}

func (f *estruturaPorMascara) GetAllDirectChildren(context.Context, int64) ([]*structentity.ItemStructure, error) {
	panic("a OF não pode mais ler a estrutura ignorando a máscara")
}

func (f *estruturaPorMascara) GetDirectChildrenForMask(_ context.Context, _ int64, mask string) ([]*structentity.ItemStructure, error) {
	f.mascaraPedida = mask
	return f.filhos, nil
}

type variaveisFixas map[string]float64

func (v variaveisFixas) GetMaskAnswersWithNames(context.Context, int64, string) (map[string]float64, error) {
	return v, nil
}

// TestCriarOFRespeitaMascaraVigenciaEFormula trava as três falhas da lista de
// materiais da OF: ela lia a estrutura sem máscara (item configurado recebia os
// componentes de todas as variantes), não olhava vigência (componente vencido
// entrava) e usava a quantidade fixa em vez da fórmula da configuração.
func TestCriarOFRespeitaMascaraVigenciaEFormula(t *testing.T) {
	d := func(s string) *time.Time { v, _ := time.Parse("2006-01-02", s); return &v }
	formula := "COMPRIMENTO/1000"
	estrutura := &estruturaPorMascara{filhos: []*structentity.ItemStructure{
		{ChildCode: 20, Quantity: 1},                             // vigente
		{ChildCode: 30, Quantity: 1, EndDate: d("2026-08-12")},   // vencido
		{ChildCode: 40, Quantity: 1, StartDate: d("2026-12-01")}, // ainda não vale
		{ChildCode: 50, Quantity: 9, QuantityFormula: &formula},  // paramétrico
	}}
	repo := &fakePORepo{automatic: map[int64]int64{}}
	uc := CreateProductionOrderUseCase{
		Repo: repo, Auth: fakeAuth{canPlanned: true}, Structure: estrutura,
		MaskVars: variaveisFixas{"COMPRIMENTO": 2000},
		Items:    &fakeOrderItems{item: &itementity.Item{Code: 10, BusinessCode: "MOVEL-10"}},
	}
	inicio := "2026-09-15"
	if _, err := uc.Execute(context.Background(), request.CreateProductionOrderDTO{
		ItemCode: "MOVEL-10", Mask: "2000#600", PlannedQty: 5, StartDate: &inicio, CreatedBy: uuid.New(),
	}); err != nil {
		t.Fatal(err)
	}

	if estrutura.mascaraPedida != "2000#600" {
		t.Errorf("a estrutura foi lida com a máscara %q, esperado %q", estrutura.mascaraPedida, "2000#600")
	}
	qtd := map[int64]decimal.Decimal{}
	for _, m := range repo.createdMaterials {
		qtd[m.ItemCode] = m.Quantity
	}
	if _, ok := qtd[30]; ok {
		t.Error("componente vencido entrou na lista de materiais")
	}
	if _, ok := qtd[40]; ok {
		t.Error("componente que ainda não entrou em vigor entrou na lista de materiais")
	}
	if !qtd[20].Equal(decimal.NewFromInt(5)) {
		t.Errorf("componente vigente: esperado 5, obtido %s", qtd[20])
	}
	// COMPRIMENTO/1000 = 2 por unidade × 5 unidades = 10 — e não 9 × 5 = 45.
	if !qtd[50].Equal(decimal.NewFromInt(10)) {
		t.Errorf("componente paramétrico: esperado 10 (fórmula), obtido %s", qtd[50])
	}
}
