package mrp_calculation_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
)

type motorFake struct {
	mrpservice.MRPService
	calculou bool
}

func (m *motorFake) Calculate(context.Context, int64, int64, bool) (*entity.MRPCalculationLog, error) {
	m.calculou = true
	return &entity.MRPCalculationLog{Status: "COMPLETED"}, nil
}

type recalculoMRPFake struct {
	avisos []string
	err    error
	rodou  bool
}

func (r *recalculoMRPFake) RecalcularDoComponente(context.Context, int64, string) (int, []string, error) {
	return 0, nil, nil
}
func (r *recalculoMRPFake) RecalcularDaEmpresa(context.Context, string) (int, []string, error) {
	r.rodou = true
	return 1, r.avisos, r.err
}

// O MRP recalcula as estruturas antes de calcular, põe no log a linha que não
// converte e não calcula com fator que não conseguiu conferir.
func TestMRPConfereAsUnidadesDaEstruturaAntes(t *testing.T) {
	motor := &motorFake{}
	rec := &recalculoMRPFake{avisos: []string{"estrutura do item 1: sem conversão"}}
	log, err := (&MRPComEstruturaEmDia{MRPService: motor, Estruturas: rec}).Calculate(context.Background(), 1, 1, false)
	if err != nil || !rec.rodou || !motor.calculou {
		t.Fatalf("err=%v recalculou=%v calculou=%v", err, rec.rodou, motor.calculou)
	}
	if av, _ := log.Errors[ChaveEstruturaSemConversao].([]string); len(av) != 1 {
		t.Fatalf("aviso fora do log: %v", log.Errors)
	}

	motor2 := &motorFake{}
	_, err = (&MRPComEstruturaEmDia{MRPService: motor2, Estruturas: &recalculoMRPFake{err: errors.New("banco fora")}}).Calculate(context.Background(), 1, 1, false)
	if err == nil || motor2.calculou {
		t.Fatal("calculou o MRP sem conseguir conferir as estruturas")
	}

	motor3 := &motorFake{}
	log, _ = (&MRPComEstruturaEmDia{MRPService: motor3, Estruturas: &recalculoMRPFake{}}).Calculate(context.Background(), 1, 1, false)
	if _, tem := log.Errors[ChaveEstruturaSemConversao]; tem {
		t.Fatal("aviso sem linha sem conversão")
	}
}

type anotadorFake struct {
	code  int64
	chave string
	err   error
}

func (a *anotadorFake) AnotarNoLog(_ context.Context, code int64, chave string, _ any) error {
	a.code, a.chave = code, chave
	return a.err
}

// O aviso vai também para o log guardado; se a gravação falha, o cálculo vale
// e a falha aparece na resposta, sem ser engolida.
func TestMRPGravaOAvisoNoLog(t *testing.T) {
	motor := &motorComCodigo{}
	rec := &recalculoMRPFake{avisos: []string{"estrutura do item 1: sem conversão"}}
	an := &anotadorFake{}
	log, err := (&MRPComEstruturaEmDia{MRPService: motor, Estruturas: rec, Log: an}).Calculate(context.Background(), 1, 1, false)
	if err != nil || an.code != 77 || an.chave != ChaveEstruturaSemConversao {
		t.Fatalf("err=%v anotou=%d/%q", err, an.code, an.chave)
	}
	if _, tem := log.Errors[ChaveLogNaoAnotado]; tem {
		t.Fatal("acusou falha sem falha")
	}

	an2 := &anotadorFake{err: errors.New("banco fora")}
	log, err = (&MRPComEstruturaEmDia{MRPService: &motorComCodigo{}, Estruturas: rec, Log: an2}).Calculate(context.Background(), 1, 1, false)
	if err != nil {
		t.Fatalf("o cálculo já valeu e foi desfeito: %v", err)
	}
	if log.Errors[ChaveLogNaoAnotado] == nil {
		t.Fatal("falha da gravação engolida")
	}
}

type motorComCodigo struct{ mrpservice.MRPService }

func (motorComCodigo) Calculate(context.Context, int64, int64, bool) (*entity.MRPCalculationLog, error) {
	return &entity.MRPCalculationLog{Code: 77, Status: "COMPLETED"}, nil
}
