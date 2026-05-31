package entity

import (
	"testing"

	"github.com/google/uuid"
)

func baseInput() SupplierInput {
	ie := "1234567890"
	return SupplierInput{
		Name:              "Fornecedor Teste Ltda",
		PersonType:        PersonJuridica,
		DocumentType:      DocumentCNPJ,
		DocumentNumber:    "11222333000181",
		TypeKind:          KindNormal,
		StateRegistration: &ie,
	}
}

func TestNewSupplier_ValidCNPJ(t *testing.T) {
	s, err := NewSupplier(1, baseInput(), uuid.New())
	if err != nil {
		t.Fatalf("expected valid supplier, got error: %v", err)
	}
	if !s.IsActive || s.FreightType != FreightSemFrete || s.ICMSContributor != ICMSContribuinte {
		t.Errorf("unexpected defaults: active=%v freight=%s icms=%s", s.IsActive, s.FreightType, s.ICMSContributor)
	}
}

func TestNewSupplier_InvalidCNPJ(t *testing.T) {
	in := baseInput()
	in.DocumentNumber = "11222333000182" // wrong check digit
	if _, err := NewSupplier(1, in, uuid.New()); err == nil {
		t.Fatal("expected error for invalid CNPJ, got nil")
	}
}

func TestNewSupplier_ValidCPF(t *testing.T) {
	in := baseInput()
	in.PersonType = PersonFisica
	in.DocumentType = DocumentCPF
	in.DocumentNumber = "52998224725"
	if _, err := NewSupplier(1, in, uuid.New()); err != nil {
		t.Fatalf("expected valid CPF supplier, got error: %v", err)
	}
}

func TestNewSupplier_MEINotAllowedForFisica(t *testing.T) {
	in := baseInput()
	in.PersonType = PersonFisica
	in.DocumentType = DocumentCPF
	in.DocumentNumber = "52998224725"
	in.IsMEI = true
	if _, err := NewSupplier(1, in, uuid.New()); err == nil {
		t.Fatal("expected error: MEI cannot be a Pessoa Física")
	}
}

func TestNewSupplier_StateRegistrationRequiredForNormal(t *testing.T) {
	in := baseInput()
	in.StateRegistration = nil
	if _, err := NewSupplier(1, in, uuid.New()); err == nil {
		t.Fatal("expected error: IE required for NORMAL supplier")
	}
}

func TestNewSupplier_StateRegistrationOptionalForCarrier(t *testing.T) {
	in := baseInput()
	in.StateRegistration = nil
	in.TypeKind = KindTransportadora
	if _, err := NewSupplier(1, in, uuid.New()); err != nil {
		t.Fatalf("IE should be optional for carriers, got error: %v", err)
	}
}

func TestNewSupplier_AgricultureRegistrationFormat(t *testing.T) {
	in := baseInput()
	bad := "RS123456"
	in.AgricultureMinistryRegistration = &bad
	if _, err := NewSupplier(1, in, uuid.New()); err == nil {
		t.Fatal("expected error for malformed Registro M.A.")
	}

	good := "RS-12345-6"
	in.AgricultureMinistryRegistration = &good
	if _, err := NewSupplier(1, in, uuid.New()); err != nil {
		t.Fatalf("expected valid Registro M.A. format, got error: %v", err)
	}
}

func TestSupplierKind_RequiresStateRegistration(t *testing.T) {
	cases := map[SupplierKind]bool{
		KindNormal:         true,
		KindTransportadora: false,
		KindTranspRedesp:   false,
		KindRedespacho:     false,
	}
	for k, want := range cases {
		if got := k.RequiresStateRegistration(); got != want {
			t.Errorf("%s.RequiresStateRegistration() = %v, want %v", k, got, want)
		}
	}
}
