package stock_uc

import (
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

func TestAplicarTipoDeMovimento(t *testing.T) {
	base := entity.TipoMovimentoConfig{Sigla: "CI", Descricao: "Consumo interno", Direcao: "SAIDA", AtualizaCusto: true, PermiteManual: true, Ativo: true, ExigeCentroCusto: true}
	cc := int32(10)

	dto := request.CreateStockMovementDTO{MovementType: "IN", CostCenterCode: &cc}
	if err := aplicarTipoDeMovimento(&base, &dto); err != nil || dto.MovementType != "OUT" {
		t.Fatalf("tipo de saída força OUT: %v %s", err, dto.MovementType)
	}
	dto = request.CreateStockMovementDTO{}
	if err := aplicarTipoDeMovimento(&base, &dto); err == nil || !strings.Contains(err.Error(), "centro de custo") {
		t.Fatalf("tipo que exige centro de custo: %v", err)
	}
	t2 := base
	t2.PermiteManual = false
	if err := aplicarTipoDeMovimento(&t2, &request.CreateStockMovementDTO{CostCenterCode: &cc}); err == nil {
		t.Fatal("tipo sem lançamento manual deveria recusar")
	}
	t3 := base
	t3.Ativo = false
	if err := aplicarTipoDeMovimento(&t3, &request.CreateStockMovementDTO{CostCenterCode: &cc}); err == nil {
		t.Fatal("tipo inativo deveria recusar")
	}
	t4 := base
	t4.Direcao, t4.ExigeCentroCusto = "AMBOS", false
	if err := aplicarTipoDeMovimento(&t4, &request.CreateStockMovementDTO{}); err == nil {
		t.Fatal("tipo AMBOS sem direção deveria pedir IN ou OUT")
	}
	if err := aplicarTipoDeMovimento(&t4, &request.CreateStockMovementDTO{MovementType: "IN"}); err != nil {
		t.Fatalf("AMBOS com IN: %v", err)
	}
	t5 := base
	t5.Direcao = "TRANSFERENCIA"
	if err := aplicarTipoDeMovimento(&t5, &request.CreateStockMovementDTO{CostCenterCode: &cc}); err == nil {
		t.Fatal("tipo de transferência vai para a rotina de transferência")
	}
}
