package mrp_calculation_uc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
)

type excecaoAuth struct {
	ports.AuthService
	ator uuid.UUID
}

func (excecaoAuth) CanRunMRPCalculation(context.Context) bool   { return true }
func (a excecaoAuth) UserID(context.Context) (uuid.UUID, error) { return a.ator, nil }

type excecoesFalsas struct {
	itens    map[int64]*entity.MRPExceptionMessage
	marcadas []string
}

func (f *excecoesFalsas) GetException(_ context.Context, code int64) (*entity.MRPExceptionMessage, error) {
	if e, ok := f.itens[code]; ok {
		return e, nil
	}
	return nil, errorsuc.NewNotFoundError("exceção não encontrada")
}
func (f *excecoesFalsas) MarkExceptionHandled(_ context.Context, code int64, action string, note *string, _ uuid.UUID) (*entity.MRPExceptionMessage, error) {
	e := f.itens[code]
	agora := time.Now()
	e.HandledAt, e.HandlingAction = &agora, action
	if note != nil {
		e.HandlingNote = *note
	}
	f.marcadas = append(f.marcadas, action)
	return e, nil
}

type ordensFalsas struct {
	reprogramadas map[int64]time.Time
	canceladas    []int64
	comOF         map[int64]int64
}

func (o *ordensFalsas) Reschedule(_ context.Context, code int64, d time.Time) (*plannedentity.PlannedOrder, error) {
	o.reprogramadas[code] = d
	return &plannedentity.PlannedOrder{Code: code}, nil
}
func (o *ordensFalsas) Cancel(_ context.Context, code int64) (*plannedentity.PlannedOrder, error) {
	o.canceladas = append(o.canceladas, code)
	return &plannedentity.PlannedOrder{Code: code}, nil
}
func (o *ordensFalsas) ActiveProductionOrderNumber(_ context.Context, code int64) (*int64, error) {
	if n, ok := o.comOF[code]; ok {
		return &n, nil
	}
	return nil, nil
}

func excecao(code int64, tipo entity.ExceptionMessageType, ordem int64, sugerida *time.Time) *entity.MRPExceptionMessage {
	origem := "PLANNED_ORDER"
	return &entity.MRPExceptionMessage{Code: code, MessageType: tipo, SourceCode: &ordem, SourceType: &origem, SuggestedDate: sugerida}
}

// Da exceção, o planejador reprograma a ordem para a data que o cálculo pediu,
// cancela a ordem sem demanda ou registra que analisou — e a mensagem fica
// marcada com o que foi feito.
func TestTratamentoDaExcecaoDoMRP(t *testing.T) {
	d := func(dia int) *time.Time { v := time.Date(2026, 10, dia, 0, 0, 0, 0, time.UTC); return &v }
	exc := &excecoesFalsas{itens: map[int64]*entity.MRPExceptionMessage{
		1: excecao(1, entity.ExceptionRescheduleIn, 100, d(20)),
		2: excecao(2, entity.ExceptionExpedite, 101, d(5)), // sugerida no passado
		3: excecao(3, entity.ExceptionCancel, 102, nil),
		4: excecao(4, entity.ExceptionRescheduleOut, 103, d(25)), // ordem já virou OF
		5: {Code: 5, MessageType: entity.ExceptionExcess},
	}}
	ord := &ordensFalsas{reprogramadas: map[int64]time.Time{}, comOF: map[int64]int64{103: 7001}}
	uc := &HandleMRPExceptionUseCase{Exceptions: exc, Orders: ord, Auth: excecaoAuth{ator: uuid.New()},
		Now: func() time.Time { return time.Date(2026, 10, 12, 18, 0, 0, 0, time.Local) }}
	ctx := context.Background()
	acao := func(a string) request.HandleMRPExceptionDTO { return request.HandleMRPExceptionDTO{Action: a} }

	if r, err := uc.Execute(ctx, 1, acao("reprogramar")); err != nil || r.HandlingAction != entity.ExceptionHandledRescheduled || !ord.reprogramadas[100].Equal(*d(20)) {
		t.Fatalf("reprogramar para a data sugerida: %+v %v %v", r, err, ord.reprogramadas)
	}
	if _, err := uc.Execute(ctx, 1, acao("REPROGRAMAR")); !isConflito(err) {
		t.Fatalf("tratar duas vezes: %v", err)
	}
	if _, err := uc.Execute(ctx, 2, acao("REPROGRAMAR")); err != nil || !ord.reprogramadas[101].Equal(*d(12)) {
		t.Fatalf("sugerida no passado deveria virar hoje: %v %v", err, ord.reprogramadas[101])
	}
	if _, err := uc.Execute(ctx, 3, acao("REPROGRAMAR")); err == nil {
		t.Fatal("reprogramou a partir de exceção de cancelar")
	}
	if _, err := uc.Execute(ctx, 3, acao("CANCELAR")); err != nil || len(ord.canceladas) != 1 || ord.canceladas[0] != 102 {
		t.Fatalf("cancelar: %v %v", err, ord.canceladas)
	}
	if _, err := uc.Execute(ctx, 4, acao("REPROGRAMAR")); err == nil || ord.reprogramadas[103] != (time.Time{}) {
		t.Fatalf("reprogramou ordem que já virou OF: %v", err)
	}
	passado := "2026-10-01"
	if _, err := uc.Execute(ctx, 4, request.HandleMRPExceptionDTO{Action: "REPROGRAMAR", NewDate: &passado}); err == nil {
		t.Fatal("aceitou nova data no passado")
	}
	if _, err := uc.Execute(ctx, 5, acao("TRATAR")); err == nil {
		t.Fatal("tratou sem dizer o que foi feito")
	}
	nota := "excesso absorvido pelo pedido de novembro"
	if r, err := uc.Execute(ctx, 5, request.HandleMRPExceptionDTO{Action: "TRATAR", Note: &nota}); err != nil || r.HandlingNote != nota {
		t.Fatalf("tratar com nota: %+v %v", r, err)
	}
	if _, err := uc.Execute(ctx, 5, acao("APAGAR")); err == nil {
		t.Fatal("aceitou ação desconhecida")
	}
}

func isConflito(err error) bool {
	var c *errorsuc.ConflictError
	return errors.As(err, &c)
}
