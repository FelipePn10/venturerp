package representative_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/representative/repository"
)

func TestRepresentativeFromCreateBuildsAddressAndNormalizesState(t *testing.T) {
	state := "rs"
	street := "Rua A"
	number := "120"
	rep, err := representativeFromCreate(request.CreateRepresentativeDTO{
		Name:           "Representante Sul",
		DocumentNumber: "12345678901",
		State:          &state,
		Street:         &street,
		StreetNumber:   &number,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.State == nil || *rep.State != "RS" {
		t.Fatalf("state = %v, want RS", rep.State)
	}
	if rep.FullAddress == nil || *rep.FullAddress != "Rua A, 120" {
		t.Fatalf("full address = %v, want Rua A, 120", rep.FullAddress)
	}
}

func TestRepresentativeFromCreateRequiresDocument(t *testing.T) {
	_, err := representativeFromCreate(request.CreateRepresentativeDTO{Name: "Sem documento"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNormalizeFilterDefaults(t *testing.T) {
	filter := normalizeFilter(repository.RepresentativeFilter{SortBy: "bad"})
	if filter.SortBy != "CODE" || filter.ActiveStatus != "ACTIVE" {
		t.Fatalf("unexpected defaults: %#v", filter)
	}
}
