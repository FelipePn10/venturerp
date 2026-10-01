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

// Fornecedor não contribuinte de ICMS legitimamente não tem inscrição estadual:
// prestador de serviço, pessoa física e boa parte dos MEI. Antes, a exigência
// olhava só o tipo e só transportadora escapava — para cadastrar um prestador era
// preciso inventar um número (que entraria na apuração de ICMS das notas dele) ou
// declará-lo transportadora. Foi o que barrou os 12 não contribuintes da Usimac.
func TestInscricaoEstadualSegueACondicaoDeICMS(t *testing.T) {
	ator := uuid.New()
	base := func(icms ICMSContributor, ie *string) SupplierInput {
		return SupplierInput{
			Name:              "FORNECEDOR DE TESTE LTDA",
			PersonType:        PersonJuridica,
			DocumentType:      DocumentCNPJ,
			DocumentNumber:    "23208854000163",
			TypeKind:          KindNormal,
			ICMSContributor:   icms,
			StateRegistration: ie,
		}
	}
	numero := "653082415113"
	vazio := ""

	casos := []struct {
		nome   string
		icms   ICMSContributor
		ie     *string
		aceita bool
	}{
		{"contribuinte com inscrição", ICMSContribuinte, &numero, true},
		{"contribuinte sem inscrição", ICMSContribuinte, nil, false},
		{"contribuinte com inscrição em branco", ICMSContribuinte, &vazio, false},
		{"contribuinte com inscrição só de espaços", ICMSContribuinte, ptr("   "), false},
		{"não contribuinte sem inscrição", ICMSNaoContribuinte, nil, true},
		{"isento sem inscrição", ICMSIsento, nil, true},
		{"não contribuinte que tem inscrição", ICMSNaoContribuinte, &numero, true},
		// Controle: silêncio NÃO dispensa. O padrão da coluna é CONTRIBUINTE, e a
		// dispensa precisa de alguém declarando a condição.
		{"condição em branco sem inscrição", "", nil, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			_, err := NewSupplier(1, base(c.icms, c.ie), ator)
			if c.aceita && err != nil {
				t.Fatalf("esperava aceitar, recusou: %v", err)
			}
			if !c.aceita && err == nil {
				t.Fatal("esperava recusar, aceitou")
			}
		})
	}
}

// Condição inexistente tem de ser recusada com nome, não gravada para o CHECK do
// banco barrar depois com erro cru. "Contribuinte" (como vem da planilha do
// cliente) não é "CONTRIBUINTE".
func TestCondicaoDeICMSInexistenteERecusada(t *testing.T) {
	_, err := NewSupplier(1, SupplierInput{
		Name: "X LTDA", PersonType: PersonJuridica, DocumentType: DocumentCNPJ,
		DocumentNumber: "23208854000163", TypeKind: KindNormal,
		ICMSContributor: "Contribuinte", StateRegistration: ptr("653082415113"),
	}, uuid.New())
	if err == nil {
		t.Fatal("esperava recusar a condição \"Contribuinte\" (minúsculas), aceitou")
	}
}

func ptr(s string) *string { return &s }

// Achado da revisão do PR #178: o construtor validava com a condição recebida e
// devolvia a entidade cravada em CONTRIBUINTE. Um fornecedor aceito como não
// contribuinte (logo, sem inscrição estadual) saía declarado contribuinte — estado
// que a própria validação existe para impedir. No caso de uso a atribuição
// posterior corrigia por acidente; qualquer outro chamador gravaria errado.
func TestConstrutorDevolveACondicaoQueValidou(t *testing.T) {
	casos := map[ICMSContributor]ICMSContributor{
		ICMSNaoContribuinte: ICMSNaoContribuinte,
		ICMSIsento:          ICMSIsento,
		ICMSContribuinte:    ICMSContribuinte,
		"":                  ICMSContribuinte, // vazio = padrão da coluna
	}
	for entrada, esperado := range casos {
		in := SupplierInput{
			Name: "X LTDA", PersonType: PersonJuridica, DocumentType: DocumentCNPJ,
			DocumentNumber: "23208854000163", TypeKind: KindNormal,
			ICMSContributor: entrada,
		}
		if entrada.RequiresStateRegistration() {
			in.StateRegistration = ptr("653082415113")
		}
		s, err := NewSupplier(1, in, uuid.New())
		if err != nil {
			t.Fatalf("entrada %q: %v", entrada, err)
		}
		if s.ICMSContributor != esperado {
			t.Errorf("entrada %q devolveu %q, esperado %q", entrada, s.ICMSContributor, esperado)
		}
		// O estado incoerente que o achado descreve: sem inscrição E declarado
		// contribuinte não pode sair do construtor.
		semIE := s.StateRegistration == nil || *s.StateRegistration == ""
		if semIE && s.ICMSContributor == ICMSContribuinte {
			t.Errorf("entrada %q: fornecedor sem inscrição saiu declarado contribuinte", entrada)
		}
	}
}
