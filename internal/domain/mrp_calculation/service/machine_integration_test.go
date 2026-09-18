package service

import (
	"errors"
	machinesvc "github.com/FelipePn10/panossoerp/internal/domain/machine/service"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"math"
	"strings"
	"testing"
	"time"
)

func TestMachineMinutesUsesItemRateOnce(t *testing.T) {
	eff := 0.8
	mt := &entity.MachineTimeInfo{ProductionTime: 1, ProductionTimeUnit: "HORA", ProductionBaseQty: 120, SetupTime: 15, EfficiencyRate: &eff, MachineEfficiencyRate: 0.5, WorkingHoursPerDay: 8, TimeBasis: "PROPORTIONAL"}
	got, err := machineMinutes(mt, 60)
	if err != nil || math.Abs(got-52.5) > 1e-9 {
		t.Fatalf("got %v,%v; want 52.5 min", got, err)
	}

	eff = 1
	got, err = machineMinutes(mt, 60)
	if err != nil || got != 45 {
		t.Fatalf("real expected rate discounted twice: %v %v", got, err)
	}
	eff = 0.8
	mt.TimeBasis = "CYCLE"
	got, err = machineMinutes(mt, 60)
	if err != nil || got != 90 {
		t.Fatalf("cycle got %v,%v; want 90", got, err)
	}
	mt.ProductionTime = 60
	mt.ProductionTimeUnit = "MINUTO"
	got, err = machineMinutes(mt, 60)
	if err != nil || got != 90 {
		t.Fatalf("minutes got %v,%v", got, err)
	}
	mt.ProductionBaseQty = 0
	if _, err = machineMinutes(mt, 60); err == nil {
		t.Fatal("accepted zero base quantity")
	}
}
func TestMachineSelectionPrefersExactMaskAndStablePriority(t *testing.T) {
	base := &entity.MachineTimeInfo{MachineID: 1, Priority: 0}
	other := &entity.MachineTimeInfo{MachineID: 2, Mask: "OTHER", Priority: 0}
	exact := &entity.MachineTimeInfo{MachineID: 3, Mask: "A", Priority: 9}
	if got := selectMachineTime([]*entity.MachineTimeInfo{base, other, exact}, "A"); got != exact {
		t.Fatal("exact mask must win")
	}
	if got := selectMachineTime([]*entity.MachineTimeInfo{base, other, exact}, "B"); got != base {
		t.Fatal("default mask expected")
	}
	if got := selectMachineTime([]*entity.MachineTimeInfo{other, exact}, "B"); got != nil {
		t.Fatal("different mask selected")
	}
}

func TestMachineMaterialDependencies(t *testing.T) {
	parent := int64(10)
	in := []*entity.PlannedOrderSuggestion{{Code: 1, ItemCode: 10}, {Code: 2, ItemCode: 20, ParentItemCode: &parent}}
	out, err := orderMachineSuggestions(in)
	if err != nil || out[0].Code != 2 {
		t.Fatalf("component must precede parent: %v %v", out, err)
	}
	child := int64(20)
	in[0].ParentItemCode = &child
	if _, err = orderMachineSuggestions(in); err == nil {
		t.Fatal("material cycle accepted")
	}
}

// Três máquinas idênticas são três recursos que fazem o mesmo, não uma preferida
// com duas reservas. Empilhar a fila na primeira estica o prazo com capacidade
// sobrando ao lado — que é o pior jeito de errar, porque ninguém vê.
func TestEquivalentesAgrupaMesmaPrioridadeEMascara(t *testing.T) {
	perfil := func(id int64, prio int, mascara string) *entity.MachineTimeInfo {
		return &entity.MachineTimeInfo{MachineID: id, MachineCode: id, Priority: prio, Mask: mascara}
	}
	lista := []*entity.MachineTimeInfo{
		perfil(10, 1, ""), perfil(11, 1, ""), perfil(12, 1, ""), // as três iguais
		perfil(20, 2, ""),       // alternativa de segunda escolha
		perfil(30, 1, "GRANDE"), // outra máscara
	}
	eq := equivalentes(lista, lista[0])
	if len(eq) != 3 {
		ids := []int64{}
		for _, m := range eq {
			ids = append(ids, m.MachineID)
		}
		t.Fatalf("esperava as três de prioridade 1 sem máscara, vieram %v", ids)
	}
	// A de prioridade 2 não entra: "prefira a serra 1" continua valendo.
	for _, m := range eq {
		if m.Priority != 1 || m.Mask != "" {
			t.Fatalf("máquina %d entrou indevidamente (prioridade %d, máscara %q)", m.MachineID, m.Priority, m.Mask)
		}
	}
	// A própria escolhida vem primeiro, para o desempate ser estável.
	if eq[0].MachineID != 10 {
		t.Fatalf("a escolhida deveria abrir a lista, veio %d", eq[0].MachineID)
	}
}

// A ordem vai para a máquina equivalente que TERMINA ANTES, não para a primeira
// da lista. Sem isso, três serras iguais viram uma fila e duas ociosas.
func TestEscolheAMaquinaQueTerminaAntes(t *testing.T) {
	base := time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)
	perfil := func(id int64) *entity.MachineTimeInfo {
		return &entity.MachineTimeInfo{MachineID: id, MachineCode: id, Priority: 1}
	}
	m10, m11, m12 := perfil(10), perfil(11), perfil(12)
	step := machineStep{profile: m10, minutes: 60, alternativas: []*entity.MachineTimeInfo{m10, m11, m12}}

	// A 10 está cheia até as 16h; a 11, até as 10h; a 12, até as 12h.
	fimPorMaquina := map[int64]time.Time{
		10: base.Add(8 * time.Hour),
		11: base.Add(2 * time.Hour),
		12: base.Add(4 * time.Hour),
	}
	slots, escolhida, err := escolherMaquinaQueTerminaAntes(step, func(c *entity.MachineTimeInfo) ([]machinesvc.CapacityWindow, error) {
		fim := fimPorMaquina[c.MachineID]
		return []machinesvc.CapacityWindow{{Start: fim.Add(-time.Hour), End: fim}}, nil
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if escolhida.MachineID != 11 {
		t.Fatalf("escolheu a máquina %d; a 11 termina antes", escolhida.MachineID)
	}
	if slots[len(slots)-1].End != base.Add(2*time.Hour) {
		t.Fatalf("devolveu os slots da máquina errada: %v", slots)
	}
}

// Quando nenhuma equivalente tem janela, o motivo da PRIMEIRA recusa é o que
// chega ao usuário — "nenhuma disponível" não diz o que revisar.
func TestNenhumaEquivalenteDisponivelExplicaOMotivo(t *testing.T) {
	m10 := &entity.MachineTimeInfo{MachineID: 10, MachineCode: 777, Priority: 1}
	step := machineStep{profile: m10, minutes: 60, alternativas: []*entity.MachineTimeInfo{m10}}

	_, _, err := escolherMaquinaQueTerminaAntes(step, func(*entity.MachineTimeInfo) ([]machinesvc.CapacityWindow, error) {
		return nil, errors.New("não há capacidade no horizonte de um ano")
	})
	if err == nil {
		t.Fatal("deveria falhar")
	}
	if !strings.Contains(err.Error(), "777") || !strings.Contains(err.Error(), "horizonte") {
		t.Fatalf("a mensagem não diz qual máquina nem por quê: %v", err)
	}
}

// Sem alternativas cadastradas o comportamento antigo é preservado: usa o perfil.
func TestSemAlternativasUsaOProprioPerfil(t *testing.T) {
	m := &entity.MachineTimeInfo{MachineID: 42, MachineCode: 42, Priority: 1}
	step := machineStep{profile: m, minutes: 30}
	base := time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)

	_, escolhida, err := escolherMaquinaQueTerminaAntes(step, func(c *entity.MachineTimeInfo) ([]machinesvc.CapacityWindow, error) {
		return []machinesvc.CapacityWindow{{Start: base, End: base.Add(30 * time.Minute)}}, nil
	})
	if err != nil || escolhida.MachineID != 42 {
		t.Fatalf("deveria usar o próprio perfil: %v %v", escolhida, err)
	}
}
