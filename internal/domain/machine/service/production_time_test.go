package service

import (
	"math"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
)

func TestCalculateProductionTime_BatchAndSetup(t *testing.T) {
	imt := &entity.ItemMachineTime{
		ProductionTime:     5, // 5 minutes per cycle
		ProductionTimeUnit: types.Minute,
		ProductionBaseQty:  10, // 10 items per cycle
		SetupTime:          30,
	}
	machine := &entity.Machine{
		Capacity:       100,
		CapacityPeriod: types.Minute,
		EfficiencyRate: 1.0,
	}

	// demand 73 → ceil(73/10) = 8 cycles → 8*5 = 40 machining + 30 setup = 70 min
	res := CalculateProductionTime(imt, machine, 73, 1.0, 480)
	if res.BatchCount != 8 {
		t.Errorf("BatchCount = %v, want 8", res.BatchCount)
	}
	if res.MachiningMinutes != 40 {
		t.Errorf("MachiningMinutes = %v, want 40", res.MachiningMinutes)
	}
	if res.SetupMinutes != 30 {
		t.Errorf("SetupMinutes = %v, want 30", res.SetupMinutes)
	}
	if res.TotalMinutes != 70 {
		t.Errorf("TotalMinutes = %v, want 70", res.TotalMinutes)
	}
	if math.Abs(res.TotalHours-70.0/60.0) > 1e-9 {
		t.Errorf("TotalHours = %v", res.TotalHours)
	}
}

func TestCalculateProductionTime_HourUnitScaling(t *testing.T) {
	// production time expressed in hours should scale by 60.
	imt := &entity.ItemMachineTime{
		ProductionTime:     1, // 1 hour per cycle
		ProductionTimeUnit: types.Hour,
		ProductionBaseQty:  1,
		SetupTime:          0,
	}
	machine := &entity.Machine{Capacity: 1, CapacityPeriod: types.Hour, EfficiencyRate: 1}
	res := CalculateProductionTime(imt, machine, 3, 1.0, 480)
	// 3 cycles × 60 min = 180
	if res.TotalMinutes != 180 {
		t.Errorf("TotalMinutes = %v, want 180", res.TotalMinutes)
	}
}

func TestCalculateProductionTime_Bottleneck(t *testing.T) {
	imt := &entity.ItemMachineTime{ProductionTime: 1, ProductionTimeUnit: types.Minute, ProductionBaseQty: 1, SetupTime: 0}
	// machine capacity 1 unit/min, efficiency 1 → 1 unit/min.
	machine := &entity.Machine{Capacity: 1, CapacityPeriod: types.Minute, EfficiencyRate: 1}

	// demand 100 units, conversion 1 → required throughput 100/100min = 1/min, not a bottleneck.
	res := CalculateProductionTime(imt, machine, 100, 1.0, 480)
	if res.MachineIsBottleneck {
		t.Errorf("expected not bottleneck, got bottleneck (capPerMin=%v)", res.MachineCapacityPerMinute)
	}

	// conversion 5 → demand in machine units 500 over 100 min → 5/min > 1/min capacity → bottleneck.
	res = CalculateProductionTime(imt, machine, 100, 5.0, 480)
	if !res.MachineIsBottleneck {
		t.Errorf("expected bottleneck with high conversion factor")
	}
}

func TestCalculateProductionTime_DefaultWorkingMinutes(t *testing.T) {
	imt := &entity.ItemMachineTime{ProductionTime: 1, ProductionTimeUnit: types.Day, ProductionBaseQty: 1, SetupTime: 0}
	machine := &entity.Machine{Capacity: 1, CapacityPeriod: types.Day, EfficiencyRate: 1}
	// workingMins <= 0 → defaults to 480; 1 day cycle = 480 min.
	res := CalculateProductionTime(imt, machine, 1, 1.0, 0)
	if res.TotalMinutes != 480 {
		t.Errorf("TotalMinutes = %v, want 480 (default day)", res.TotalMinutes)
	}
}

func TestCalculateProductionTime_ExplainsEfficiencyAndCapacity(t *testing.T) {
	imt := &entity.ItemMachineTime{ProductionTime: 10, ProductionTimeUnit: types.Minute, ProductionBaseQty: 5}
	machine := &entity.Machine{Capacity: 10, CapacityPeriod: types.Hour, EfficiencyRate: 0.5}
	res := CalculateProductionTime(imt, machine, 10, 1, 480)
	if res.StandardCycleMinutes != 10 || res.EffectiveCycleMinutes != 20 || res.TotalMinutes != 40 {
		t.Fatalf("componentes inesperados: %+v", res)
	}
	if res.MachineEfficiencyRate != 0.5 || res.ResourceTimeFactor != 1 || len(res.CalculationFactors) == 0 {
		t.Fatalf("fatores não explicados: %+v", res)
	}
}

// A eficiência que dimensiona o ciclo tem de ser a mesma que dimensiona a
// capacidade. Enquanto o ciclo usava a do item e a capacidade a da máquina, o
// resultado reportava "eficiência 50%" com capacidade calculada a 100% — e o
// indicador de gargalo ficava otimista exatamente nos itens que rendem menos.
func TestCalculateProductionTime_CapacidadeUsaEficienciaDoItem(t *testing.T) {
	eficienciaDoItem := 0.5
	imt := &entity.ItemMachineTime{
		ProductionTime: 60, ProductionTimeUnit: types.Minute, ProductionBaseQty: 120,
		EfficiencyRate: &eficienciaDoItem, TimeBasis: "PROPORTIONAL",
	}
	machine := &entity.Machine{Capacity: 120, CapacityPeriod: types.Hour, EfficiencyRate: 1}

	res := CalculateProductionTime(imt, machine, 120, 1, 480)

	if res.MachineEfficiencyRate != eficienciaDoItem || res.EfficiencySource != "ITEM" {
		t.Fatalf("eficiência do item não prevaleceu: %+v", res)
	}
	esperada := machine.Capacity * eficienciaDoItem / 60
	if res.MachineCapacityPerMinute != esperada {
		t.Fatalf("capacidade por minuto = %v, esperada %v (capacidade × eficiência aplicada)",
			res.MachineCapacityPerMinute, esperada)
	}
	// 120 peças em 120 min exigem 1/min; a máquina, a 50%, entrega exatamente 1/min.
	if res.RequiredCapacityRate != res.MachineCapacityPerMinute {
		t.Fatalf("taxa exigida %v deveria igualar a capacidade %v", res.RequiredCapacityRate, res.MachineCapacityPerMinute)
	}
}

// Corte a laser: o cilindro não dura "N horas", dura conforme o que está sendo
// cortado. A taxa vem do item; a autonomia e o tempo de troca, da máquina. A
// parada para trocar ocupa a máquina e precisa entrar no tempo da ordem.
func TestCalculateProductionTime_TrocaDeConsumivelOcupaAMaquina(t *testing.T) {
	imt := &entity.ItemMachineTime{
		ProductionTime: 60, ProductionTimeUnit: types.Minute, ProductionBaseQty: 1,
		TimeBasis: "PROPORTIONAL", SetupTime: 10,
		Consumable: &entity.ConsumableUsage{
			PerHour: 100, CapacityPerRefill: 200, ReplacementMinutes: 15,
			Unit: "m³", Description: "Oxigênio",
		},
	}
	machine := &entity.Machine{Capacity: 1, CapacityPeriod: types.Hour, EfficiencyRate: 1}

	// 5 peças × 60 min = 300 min de usinagem = 5 h → 500 m³.
	// 500 / 200 = 2,5 → 3 cargas → 2 trocas → 30 min parados.
	res := CalculateProductionTime(imt, machine, 5, 1, 480)

	if res.ConsumableUsed != 500 {
		t.Fatalf("consumo = %v, esperado 500", res.ConsumableUsed)
	}
	if res.ConsumableRefills != 2 || res.ConsumableMinutes != 30 {
		t.Fatalf("trocas = %v (%v min), esperado 2 (30 min)", res.ConsumableRefills, res.ConsumableMinutes)
	}
	if res.TotalMinutes != 300+10+30 {
		t.Fatalf("total = %v, esperado 340 (300 usinagem + 10 setup + 30 de troca)", res.TotalMinutes)
	}

	// Consumo exatamente igual a uma carga não obriga troca: a ordem termina com
	// o cilindro zerado e quem troca é a próxima.
	res = CalculateProductionTime(imt, machine, 2, 1, 480)
	if res.ConsumableUsed != 200 || res.ConsumableRefills != 0 || res.ConsumableMinutes != 0 {
		t.Fatalf("carga exata não deveria obrigar troca: %v usado, %v trocas", res.ConsumableUsed, res.ConsumableRefills)
	}

	// Sem consumível cadastrado nada muda — o caminho antigo continua idêntico.
	imt.Consumable = nil
	if res := CalculateProductionTime(imt, machine, 5, 1, 480); res.TotalMinutes != 310 || res.ConsumableRefills != 0 {
		t.Fatalf("sem consumível o total deveria ser 310: %+v", res.TotalMinutes)
	}
}
