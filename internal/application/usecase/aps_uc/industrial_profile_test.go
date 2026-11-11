package aps_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	apsrepo "github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
)

// Só a validação roda: um repositório que chega a ser chamado entra em pânico.
type perfilSemBanco struct {
	*fakeAPSRepo
	apsrepo.ConfigurationRepository
}

func TestPerfilIndustrialRecusaCampoEspecialRepetido(t *testing.T) {
	uc := &APSUseCase{repo: perfilSemBanco{fakeAPSRepo: &fakeAPSRepo{}}}
	dto := request.MachineIndustrialProfileDTO{PreparationTime: "10", PreparationTimeUnit: "MINUTE"}
	dto.SpecialValues = append(dto.SpecialValues, request.SpecialValueDTO{Name: "Potência kW", ValueType: "NUMBER", NumericValue: "7.5"}, request.SpecialValueDTO{Name: " potência KW ", ValueType: "NUMBER", NumericValue: "8"})
	err := uc.UpsertMachineIndustrialProfile(context.Background(), 1, dto)
	if err == nil || !strings.Contains(err.Error(), "mais de uma vez") {
		t.Fatalf("esperava recusa do campo repetido, veio %v", err)
	}
}
