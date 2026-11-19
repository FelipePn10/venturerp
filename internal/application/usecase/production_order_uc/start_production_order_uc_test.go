package production_order_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
)

type inicioRepo struct {
	repository.ProductionOrderRepository
	modo      string
	pend      entity.PendenciasDoInicio
	iniciada  bool
	conferida bool
}

func (*inicioRepo) HasProductionOperations(context.Context, int64) (bool, error) { return true, nil }
func (r *inicioRepo) ModoConferenciaInicio(context.Context) (string, error)      { return r.modo, nil }
func (r *inicioRepo) PendenciasDoInicio(context.Context, int64) (entity.PendenciasDoInicio, error) {
	r.conferida = true
	return r.pend, nil
}
func (r *inicioRepo) Start(_ context.Context, id int64, _ time.Time) (*entity.ProductionOrder, error) {
	r.iniciada = true
	return &entity.ProductionOrder{ID: id, Status: entity.StatusInProgress}, nil
}

// O parâmetro 52 decide: BLOQUEAR recusa com a OF filha aberta, AVISAR inicia e
// devolve o aviso, NAO nem confere.
func TestInicioDaOFConfereOsComponentes(t *testing.T) {
	pend := entity.PendenciasDoInicio{Filhas: []entity.OrdemFilhaPendente{{
		PlannedOrderCode: 7, ProductionOrderID: 31, OrderNumber: 1201, ItemCode: 99, BusinessCode: "BU-120/50",
		Planejada: dq(20), Produzida: dq(5),
	}}}
	for _, c := range []struct {
		modo                 string
		recusa, inicia, avis bool
	}{
		{entity.ConferenciaInicioBloquear, true, false, false},
		{entity.ConferenciaInicioAvisar, false, true, true},
		{entity.ConferenciaInicioNao, false, true, false},
	} {
		repo := &inicioRepo{modo: c.modo, pend: pend}
		uc := &StartProductionOrderUseCase{Repo: repo, Auth: scannerAuth{}}
		order, err := uc.Execute(context.Background(), request.StartProductionOrderDTO{ID: 5})
		if (err != nil) != c.recusa {
			t.Fatalf("%s: erro = %v, recusa esperada = %v", c.modo, err, c.recusa)
		}
		if c.recusa && !strings.Contains(err.Error(), "OF 31") {
			t.Fatalf("%s: a recusa não diz qual OF filha falta: %v", c.modo, err)
		}
		if repo.iniciada != c.inicia {
			t.Fatalf("%s: iniciada = %v, esperado %v", c.modo, repo.iniciada, c.inicia)
		}
		if c.modo == entity.ConferenciaInicioNao && repo.conferida {
			t.Fatal("NAO não deveria conferir as pendências")
		}
		if order != nil && (len(order.AvisosDoInicio) > 0) != c.avis {
			t.Fatalf("%s: avisos = %v", c.modo, order.AvisosDoInicio)
		}
	}
}

// A peça reservada para a OF conta como dela; o comprometido das OFs da frente
// sai só do livre.
func TestDisponivelSomaAReservaDaOF(t *testing.T) {
	l := entity.MaterialAvailabilityLine{Required: dq(20), Issued: dq(2), FreeStock: dq(3), CommittedEarlier: dq(5), ReservedForOrder: dq(12)}
	if got := l.Disponivel(); !got.Equal(dq(12)) {
		t.Fatalf("disponível = %s, esperado 12 (livre 3 − 5 da frente = 0, + 12 reservados)", got)
	}
	if got := l.Falta(); !got.Equal(dq(6)) {
		t.Fatalf("falta = %s, esperado 6 (18 a baixar − 12)", got)
	}
	if entity.ModoConferenciaDoParametro(" bloquear ") != entity.ConferenciaInicioBloquear ||
		entity.ModoConferenciaDoParametro("talvez") != entity.ConferenciaInicioAvisar {
		t.Fatal("leitura do parâmetro 52 errada")
	}
}
