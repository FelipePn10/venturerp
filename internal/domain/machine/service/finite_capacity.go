package service

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type CapacityWindow struct{ Start, End time.Time }

// AllocateMachineTime consumes only productive windows, never nights or downtime.
// Occupied windows are removed before allocation; the returned slices are the
// exact reservations, so a later order may use gaps between shifts safely.
func AllocateMachineTime(earliest time.Time, minutes float64, windows, occupied []CapacityWindow) ([]CapacityWindow, error) {
	if minutes > float64(math.MaxInt64)/float64(time.Minute) || minutes <= 0 || math.IsNaN(minutes) || math.IsInf(minutes, 0) {
		return nil, fmt.Errorf("duração de produção inválida")
	}
	merged := freeMachineWindows(windows, occupied)
	remaining := time.Duration(math.Ceil(minutes * float64(time.Minute)))
	slots := []CapacityWindow{}
	for _, w := range merged {
		if w.Start.Before(earliest) {
			w.Start = earliest
		}
		if !w.End.After(w.Start) {
			continue
		}
		length := w.End.Sub(w.Start)
		if length > remaining {
			length = remaining
		}
		slots = append(slots, CapacityWindow{w.Start, w.Start.Add(length)})
		remaining -= length
		if remaining <= 0 {
			return slots, nil
		}
	}
	return nil, fmt.Errorf("não há capacidade disponível no horizonte de um ano; revise turnos, paradas e produtividade")
}

func freeMachineWindows(windows, occupied []CapacityWindow) []CapacityWindow {
	free := append([]CapacityWindow(nil), windows...)
	sort.Slice(free, func(i, j int) bool { return free[i].Start.Before(free[j].Start) })
	merged := []CapacityWindow{}
	for _, w := range free {
		if !w.End.After(w.Start) {
			continue
		}
		n := len(merged)
		if n > 0 && !w.Start.After(merged[n-1].End) {
			if w.End.After(merged[n-1].End) {
				merged[n-1].End = w.End
			}
		} else {
			merged = append(merged, w)
		}
	}
	for _, busy := range occupied {
		next := []CapacityWindow{}
		for _, w := range merged {
			if !busy.End.After(w.Start) || !busy.Start.Before(w.End) {
				next = append(next, w)
				continue
			}
			if busy.Start.After(w.Start) {
				next = append(next, CapacityWindow{w.Start, busy.Start})
			}
			if busy.End.Before(w.End) {
				next = append(next, CapacityWindow{busy.End, w.End})
			}
		}
		merged = next
	}

	return merged
}

// A closed cycle cannot cross downtime or a shift boundary. Several complete
// cycles may share a window; setup is charged once, before the first cycle.
func AllocateMachineCycles(earliest time.Time, cycles, cycleMinutes, setupMinutes float64, windows, occupied []CapacityWindow) ([]CapacityWindow, error) {
	if cycles < 1 || cycles != math.Trunc(cycles) || cycleMinutes <= 0 || setupMinutes < 0 || math.IsNaN(cycles+cycleMinutes+setupMinutes) || math.IsInf(cycles+cycleMinutes+setupMinutes, 0) || cycleMinutes+setupMinutes >= float64(math.MaxInt64)/float64(time.Minute) {
		return nil, fmt.Errorf("ciclo de produção inválido")
	}
	cycle := time.Duration(math.Ceil(cycleMinutes * float64(time.Minute)))
	setup := time.Duration(math.Ceil(setupMinutes * float64(time.Minute)))
	slots := []CapacityWindow{}
	for _, w := range freeMachineWindows(windows, occupied) {
		if w.Start.Before(earliest) {
			w.Start = earliest
		}
		if !w.End.After(w.Start) {
			continue
		}
		usable := w.End.Sub(w.Start) - setup
		if usable < cycle {
			continue
		}
		count := math.Min(cycles, float64(usable/cycle))
		duration := setup + time.Duration(count)*cycle
		slots = append(slots, CapacityWindow{w.Start, w.Start.Add(duration)})
		setup = 0
		cycles -= count
		if cycles == 0 {
			return slots, nil
		}
	}
	return nil, fmt.Errorf("não há janela suficiente para os ciclos completos; revise turnos, paradas e duração do ciclo")
}
