package aps_uc

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	apsentity "github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	apsrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
)

// cadeiaAPSRepo: o RN (OF 1) consome a bucha (OF 2). O RN tem prioridade
// maior e entrega antes — sozinho, o sequenciamento o poria primeiro — e cada
// um roda num centro diferente, então a capacidade não os separa.
type cadeiaAPSRepo struct {
	*fakeAPSRepo
	ordens  []apsrepo.OrderRow
	filhas  map[int64][]int64
	fins    map[int64]time.Time
	upserts map[int64][]*apsentity.ProductionSequence
}

func (f *cadeiaAPSRepo) GetOpenProductionOrders(context.Context) ([]apsrepo.OrderRow, error) {
	return f.ordens, nil
}
func (f *cadeiaAPSRepo) GetOrderOperations(_ context.Context, orderID int64) ([]apsrepo.OpRow, error) {
	wc := 7 + orderID
	return []apsrepo.OpRow{{ID: orderID * 10, Sequence: 10, WorkCenterID: &wc, PlannedHours: 16}}, nil
}
func (f *cadeiaAPSRepo) UpsertSequence(_ context.Context, seq *apsentity.ProductionSequence) (*apsentity.ProductionSequence, error) {
	f.upserts[seq.ProductionOrderID] = append(f.upserts[seq.ProductionOrderID], seq)
	return seq, nil
}
func (f *cadeiaAPSRepo) OrdensFilhas(context.Context, []int64) (map[int64][]int64, error) {
	return f.filhas, nil
}
func (f *cadeiaAPSRepo) FimProgramado(_ context.Context, ids []int64) (map[int64]time.Time, error) {
	out := map[int64]time.Time{}
	for _, id := range ids {
		if fim, ok := f.fins[id]; ok {
			out[id] = fim
		}
	}
	return out, nil
}

func novoCadeiaRepo(ordens ...apsrepo.OrderRow) *cadeiaAPSRepo {
	return &cadeiaAPSRepo{fakeAPSRepo: &fakeAPSRepo{capacity: map[int64]float64{8: 8, 9: 8}}, ordens: ordens,
		filhas: map[int64][]int64{1: {2}}, fins: map[int64]time.Time{}, upserts: map[int64][]*apsentity.ProductionSequence{}}
}

func TestAPSProgramaAFilhaAntesDaPai(t *testing.T) {
	repo := novoCadeiaRepo(
		apsrepo.OrderRow{ID: 1, Priority: 1, PlannedDate: mon, DueDate: mon.AddDate(0, 0, 10)},
		apsrepo.OrderRow{ID: 2, Priority: 2, PlannedDate: mon, DueDate: mon.AddDate(0, 0, 5)},
	)
	if _, err := New(repo).SequenceOrders(context.Background(), request.SequenceOrdersDTO{StartFrom: mon}); err != nil {
		t.Fatal(err)
	}
	rn, bu := repo.upserts[1], repo.upserts[2]
	if len(rn) != 1 || len(bu) != 1 {
		t.Fatalf("programadas: RN %d, bucha %d", len(rn), len(bu))
	}
	if rn[0].ScheduledStart.Before(bu[0].ScheduledEnd) {
		t.Fatalf("o RN começa %s, antes de a bucha terminar %s", rn[0].ScheduledStart, bu[0].ScheduledEnd)
	}
}

// Só o RN na rodada: a bucha, já programada antes, segura o início dele.
func TestAPSPaiEsperaFilhaJaProgramada(t *testing.T) {
	repo := novoCadeiaRepo(apsrepo.OrderRow{ID: 1, Priority: 1, PlannedDate: mon})
	pronta := mon.AddDate(0, 0, 3).Add(10 * time.Hour) // quinta, 10h
	repo.fins[2] = pronta
	if _, err := New(repo).SequenceOrders(context.Background(), request.SequenceOrdersDTO{StartFrom: mon}); err != nil {
		t.Fatal(err)
	}
	if rn := repo.upserts[1]; len(rn) != 1 || rn[0].ScheduledStart.Before(pronta) {
		t.Fatalf("o RN não esperou a bucha programada para %s: %+v", pronta, rn)
	}
}

// Sem cadeia, nada muda: prioridade e entrega decidem.
func TestOrdenaPorCadeiaMantemOrdemSemVinculo(t *testing.T) {
	ordens := []apsrepo.OrderRow{{ID: 3}, {ID: 1}, {ID: 2}}
	got := ordenaPorCadeia(ordens, map[int64][]int64{1: {2}, 2: {1}}) // ciclo não trava
	if len(got) != 3 || got[0].ID != 3 {
		t.Fatalf("ordem = %+v", got)
	}
	got = ordenaPorCadeia(ordens, map[int64][]int64{3: {2}})
	if got[0].ID != 2 || got[1].ID != 3 || got[2].ID != 1 {
		t.Fatalf("a filha 2 tinha de vir antes da pai 3: %+v", got)
	}
}

// Regressiva: as duas entregam no mesmo dia, mas a bucha tem de terminar até o
// RN começar — sem a cadeia ela terminaria junto com a entrega do RN.
func TestAPSRegressivoTerminaAFilhaAteOInicioDaPai(t *testing.T) {
	entrega := mon.AddDate(0, 0, 10)
	repo := novoCadeiaRepo(
		apsrepo.OrderRow{ID: 2, Priority: 1, PlannedDate: mon, DueDate: entrega},
		apsrepo.OrderRow{ID: 1, Priority: 2, PlannedDate: mon, DueDate: entrega},
	)
	if _, err := New(repo).SequenceOrders(context.Background(), request.SequenceOrdersDTO{StartFrom: mon, Direction: "BACKWARD"}); err != nil {
		t.Fatal(err)
	}
	rn, bu := repo.upserts[1], repo.upserts[2]
	if len(rn) != 1 || len(bu) != 1 {
		t.Fatalf("programadas: RN %d, bucha %d", len(rn), len(bu))
	}
	if bu[0].ScheduledEnd.After(rn[0].ScheduledStart) {
		t.Fatalf("a bucha termina %s, depois de o RN começar %s", bu[0].ScheduledEnd, rn[0].ScheduledStart)
	}
}
