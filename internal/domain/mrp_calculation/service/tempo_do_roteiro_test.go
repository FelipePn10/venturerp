package service

import (
	"math"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	routing "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
)

// O roteiro manda no TEMPO; a produtividade da VMAQ0200 manda no RENDIMENTO.
// Antes as duas camadas se excluíam: ter tempo na etapa descartava a eficiência,
// e "Eficiência deste item = 80 %" não mudava um minuto do plano.
func TestMinutosDoRoteiroAplicaEficienciaDoItem(t *testing.T) {
	eff := 0.8
	mt := &entity.MachineTimeInfo{
		ProductionTime: 1, ProductionTimeUnit: "HORA", ProductionBaseQty: 1,
		EfficiencyRate: &eff, MachineEfficiencyRate: 0.5, WorkingHoursPerDay: 8,
	}
	// 0,5 h de máquina por peça, lote de 4 peças, 0,25 h de preparação.
	tempo := routing.OperationTime{Setup: 0.25, Run: 0.5, RunBaseQty: 1}

	got := minutosDoRoteiro(mt, tempo, 4)
	// 4 ciclos × 30 min ÷ 0,8 = 150 min de usinagem + 15 min de preparação.
	if math.Abs(got-165) > 1e-9 {
		t.Fatalf("com eficiência de 80%%: got %v, want 165", got)
	}

	// A eficiência do item sobrepõe a da máquina; sem ela, vale a da máquina.
	mt.EfficiencyRate = nil
	got = minutosDoRoteiro(mt, tempo, 4)
	if math.Abs(got-255) > 1e-9 { // 120 ÷ 0,5 = 240 + 15
		t.Fatalf("herdando a eficiência da máquina: got %v, want 255", got)
	}

	// Eficiência plena devolve exatamente a ocupação nominal do roteiro.
	mt.MachineEfficiencyRate = 1
	got = minutosDoRoteiro(mt, tempo, 4)
	if math.Abs(got-tempo.MachineHours(4)*60) > 1e-9 {
		t.Fatalf("a 100%%: got %v, want %v", got, tempo.MachineHours(4)*60)
	}
}

// Valor fora de (0,1] não pode encolher nem estourar a ordem: 100 % é o seguro.
func TestEficienciaDeIgnoraValorInvalido(t *testing.T) {
	zero, acima := 0.0, 1.4
	for _, caso := range []*float64{&zero, &acima, nil} {
		mt := &entity.MachineTimeInfo{EfficiencyRate: caso, MachineEfficiencyRate: 0}
		if got := eficienciaDe(mt); got != 1 {
			t.Fatalf("eficiência inválida virou %v, esperado 1", got)
		}
	}
}

// A troca de consumível para a máquina — e o roteiro não sabe disso. Contar só
// o tempo de corte é o erro clássico: a ordem cabe no turno no papel e estoura
// no chão.
func TestMinutosDoRoteiroContaTrocaDeConsumivel(t *testing.T) {
	porHora, capacidade, troca := 10.0, 25.0, 6.0
	mt := &entity.MachineTimeInfo{
		MachineEfficiencyRate: 1,
		ConsumptionPerHour:    &porHora,
		ConsumableCapacity:    &capacidade,
		ConsumableSwapMinutes: &troca,
	}
	tempo := routing.OperationTime{Setup: 0, Run: 1, RunBaseQty: 1}

	// 6 h de usinagem × 10/h = 60 de gasto ÷ 25 por carga = 3 cargas = 2 trocas.
	got := minutosDoRoteiro(mt, tempo, 6)
	if math.Abs(got-(360+12)) > 1e-9 {
		t.Fatalf("got %v, want 372 (360 de usinagem + 2 trocas de 6 min)", got)
	}

	// Gasto dentro de uma carga não obriga parada nenhuma.
	if got := minutosDoRoteiro(mt, tempo, 2); math.Abs(got-120) > 1e-9 {
		t.Fatalf("carga única não deve parar a máquina: got %v, want 120", got)
	}
}

// Lote parcial ocupa o ciclo inteiro, como no resto do sistema.
func TestMinutosDoRoteiroArredondaCicloParaCima(t *testing.T) {
	mt := &entity.MachineTimeInfo{MachineEfficiencyRate: 1}
	tempo := routing.OperationTime{Setup: 0, Run: 1, RunBaseQty: 4}
	if got := minutosDoRoteiro(mt, tempo, 5); math.Abs(got-120) > 1e-9 {
		t.Fatalf("5 peças em ciclos de 4 = 2 ciclos: got %v, want 120", got)
	}
}
