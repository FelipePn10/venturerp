package quality_uc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/repository"
	"github.com/shopspring/decimal"
)

type fakeRepo struct {
	repository.QualityRepository
	gravada *entity.InspectionCharacteristic
}

func (f *fakeRepo) GetPlanByID(context.Context, int64) (*entity.InspectionPlan, error) {
	return &entity.InspectionPlan{ID: 1}, nil
}

func (f *fakeRepo) AddCharacteristic(_ context.Context, c *entity.InspectionCharacteristic) (*entity.InspectionCharacteristic, error) {
	f.gravada = c
	return c, nil
}

func dec(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}

func TestCaracteristicaGuardaToleranciaExata(t *testing.T) {
	repo := &fakeRepo{}
	uc := New(repo)
	// Em float64, 0,1 + 0,2 passa de 0,3; aqui o limite inferior igual ao
	// superior tem de ser aceito e gravado sem resíduo.
	inferior := decimal.RequireFromString("0.1").Add(decimal.RequireFromString("0.2"))
	resp, err := uc.AddCharacteristic(context.Background(), request.AddCharacteristicDTO{
		PlanID: 1, Name: "Diâmetro", Nominal: dec("25.4"), ToleranceLower: &inferior, ToleranceUpper: dec("0.3"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.gravada.ToleranceLower.Equal(decimal.RequireFromString("0.3")) {
		t.Fatalf("tolerância gravada %s", repo.gravada.ToleranceLower)
	}
	if resp.Nominal == nil || *resp.Nominal != 25.4 {
		t.Fatalf("a resposta deveria seguir numérica: %v", resp.Nominal)
	}
}

func TestCaracteristicaRecusaInferiorMaiorQueSuperior(t *testing.T) {
	uc := New(&fakeRepo{})
	_, err := uc.AddCharacteristic(context.Background(), request.AddCharacteristicDTO{
		PlanID: 1, Name: "Espessura", ToleranceLower: dec("0.0002"), ToleranceUpper: dec("0.0001"),
	})
	if err == nil {
		t.Fatal("deveria recusar tolerância inferior maior que a superior")
	}
}

// O registro virou lote de inspeção (lote_compat.go): as validações de entrada
// continuam antes de tocar o lote; a gravação exata da fração está no teste de
// integração do lote.
func TestRegistroValidaQuantidadesAntesDoLote(t *testing.T) {
	uc := New(&fakeRepo{})
	if _, err := uc.CreateRecord(context.Background(), request.CreateQualityRecordDTO{PlanID: 1, ItemCode: 1}); err == nil {
		t.Fatal("deveria recusar quantidade inspecionada zerada")
	}
	_, err := uc.CreateRecord(context.Background(), request.CreateQualityRecordDTO{
		PlanID: 1, ItemCode: 1, InspectedQty: decimal.RequireFromString("15.530965"), ApprovedQty: decimal.RequireFromString("15"), Result: "APROVADO",
	})
	if err == nil || err.Error() != "a quantidade aprovada mais a rejeitada deve somar a inspecionada" {
		t.Fatalf("aprovada + rejeitada tem de fechar a inspecionada: %v", err)
	}
	if _, err := uc.CreateRecord(context.Background(), request.CreateQualityRecordDTO{
		PlanID: 1, ItemCode: 1, InspectedQty: decimal.RequireFromString("1"), Result: "TALVEZ"}); err == nil {
		t.Fatal("resultado inválido deveria ser recusado")
	}
}

func TestDTOAceitaNumeroDoClienteAtual(t *testing.T) {
	// O desktop envia número JSON; o decimal tem de aceitar sem exigir string.
	var dto request.CreateQualityRecordDTO
	if err := json.Unmarshal([]byte(`{"plan_id":1,"item_code":1,"inspected_qty":15.530965,"approved_qty":15,"rejected_qty":0.530965}`), &dto); err != nil {
		t.Fatal(err)
	}
	if !dto.InspectedQty.Equal(dto.ApprovedQty.Add(dto.RejectedQty)) {
		t.Fatalf("aprovada + rejeitada deveria fechar a inspecionada: %s %s %s", dto.InspectedQty, dto.ApprovedQty, dto.RejectedQty)
	}
}
