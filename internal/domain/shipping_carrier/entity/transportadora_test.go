package entity

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func texto(s string) *string { return &s }

func transportadoraBase() *Transportadora {
	rntrc := "12345678"
	prazo := int16(5)
	seguradora := "Seguradora Alfa"
	return &Transportadora{
		ID:                1,
		SupplierCode:      900,
		SupplierName:      "Transportes Boa Entrega",
		ANTTRNTRC:         &rntrc,
		Modal:             ModalRodoviario,
		IssuesCTe:         true,
		FreightMinValue:   dec("80"),
		FreightKgRate:     dec("0.90"),
		FreightPctValue:   dec("0.5"),
		GrisPct:           dec("0.2"),
		TollPer100Kg:      dec("6"),
		InsuranceCompany:  &seguradora,
		InsuranceCoverage: dec("100000"),
		AverageLeadDays:   &prazo,
		IsActive:          true,
	}
}

func TestRNTRCPrecisaDeOitoDigitos(t *testing.T) {
	c := transportadoraBase()
	curto := "1234"
	c.ANTTRNTRC = &curto
	if err := c.Validar(); err == nil {
		t.Fatal("RNTRC curto deveria ser recusado")
	}
	// O usuário digita com máscara; normalizar tira a máscara e o número passa.
	comMascara := "123.456-78"
	c.ANTTRNTRC = &comMascara
	c.Normalizar()
	if *c.ANTTRNTRC != "12345678" {
		t.Fatalf("RNTRC não normalizado: %s", *c.ANTTRNTRC)
	}
	if err := c.Validar(); err != nil {
		t.Fatalf("RNTRC válido recusado: %v", err)
	}
}

func TestRodoviariaComCTeExigeRNTRC(t *testing.T) {
	c := transportadoraBase()
	c.ANTTRNTRC = nil
	if err := c.Validar(); err == nil {
		t.Fatal("rodoviária emitindo CT-e sem RNTRC deveria ser recusada")
	}
	c.IssuesCTe = false
	if err := c.Validar(); err != nil {
		t.Fatalf("sem emissão de CT-e o RNTRC não é obrigatório: %v", err)
	}
}

func TestPlacaAceitaOsDoisPadroes(t *testing.T) {
	for _, placa := range []string{"abc1234", "ABC-1234", "abc1d23"} {
		v := &Veiculo{Plate: placa, IsActive: true}
		v.Normalizar()
		if err := v.Validar(); err != nil {
			t.Fatalf("placa %s recusada: %v", placa, err)
		}
	}
	v := &Veiculo{Plate: "AB1234"}
	v.Normalizar()
	if err := v.Validar(); err == nil {
		t.Fatal("placa inválida deveria ser recusada")
	}
}

func TestRegiaoAtendePorFaixaDeCEPEPorUF(t *testing.T) {
	uf := "SP"
	porUF := &RegiaoAtendida{State: &uf, IsActive: true}
	if !porUF.Atende("sp", "") || porUF.Atende("RJ", "") {
		t.Fatal("região por UF errada")
	}
	de, ate := "01000000", "01999999"
	porCEP := &RegiaoAtendida{PostalCodeFrom: &de, PostalCodeTo: &ate, IsActive: true}
	if !porCEP.Atende("SP", "01310-100") {
		t.Fatal("CEP dentro da faixa deveria ser atendido")
	}
	if porCEP.Atende("SP", "02000000") {
		t.Fatal("CEP fora da faixa não deveria ser atendido")
	}
	inativa := &RegiaoAtendida{State: &uf}
	if inativa.Atende("SP", "") {
		t.Fatal("região inativa não atende")
	}
}

// A cotação é a conta que o usuário confere componente por componente.
func TestCotacaoAbreOsComponentesEAplicaOPedagioPorFracao(t *testing.T) {
	c := transportadoraBase()
	uf := "SP"
	c.ServiceAreas = []*RegiaoAtendida{{State: &uf, LeadDays: 3, IsActive: true}}
	c.Vehicles = []*Veiculo{{Plate: "ABC1D23", CapacityKg: dec("5000"), IsActive: true}}

	base := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	cot, err := c.Cotar(Carga{PesoKg: dec("150"), ValorMercadoria: dec("10000"), UF: "SP", Base: base})
	if err != nil {
		t.Fatalf("cotação recusada: %v", err)
	}
	if !cot.ValorPorPeso.Equal(dec("135")) { // 150 kg * 0,90
		t.Fatalf("valor por peso = %s, esperado 135", cot.ValorPorPeso)
	}
	if !cot.ValorAdValorem.Equal(dec("50")) { // 0,5% de 10.000
		t.Fatalf("ad valorem = %s, esperado 50", cot.ValorAdValorem)
	}
	if !cot.ValorGris.Equal(dec("20")) { // 0,2% de 10.000
		t.Fatalf("GRIS = %s, esperado 20", cot.ValorGris)
	}
	if !cot.ValorPedagio.Equal(dec("12")) { // 150 kg = 2 frações de 100 kg
		t.Fatalf("pedágio = %s, esperado 12", cot.ValorPedagio)
	}
	if !cot.Total.Equal(dec("217")) {
		t.Fatalf("total = %s, esperado 217", cot.Total)
	}
	if cot.PisoAplicado {
		t.Fatal("o piso não deveria ter sido aplicado")
	}
	// O prazo da região manda sobre o prazo médio da capa.
	if cot.PrazoDias != 3 || !cot.PrevisaoEntrega.Equal(base.AddDate(0, 0, 3)) {
		t.Fatalf("prazo errado: %d / %s", cot.PrazoDias, cot.PrevisaoEntrega)
	}
}

func TestCotacaoAplicaOPisoSemPerderOsComponentes(t *testing.T) {
	c := transportadoraBase()
	uf := "SP"
	c.ServiceAreas = []*RegiaoAtendida{{State: &uf, LeadDays: 2, IsActive: true}}
	cot, err := c.Cotar(Carga{PesoKg: dec("10"), ValorMercadoria: dec("500"), UF: "SP"})
	if err != nil {
		t.Fatalf("cotação recusada: %v", err)
	}
	if !cot.PisoAplicado || !cot.Total.Equal(dec("80")) {
		t.Fatalf("piso não aplicado: total %s", cot.Total)
	}
	if cot.ValorPorPeso.IsZero() {
		t.Fatal("o piso não deve apagar os componentes")
	}
}

func TestCotacaoRecusaDestinoNaoAtendido(t *testing.T) {
	c := transportadoraBase()
	uf := "SP"
	c.ServiceAreas = []*RegiaoAtendida{{State: &uf, IsActive: true}}
	if _, err := c.Cotar(Carga{PesoKg: dec("10"), ValorMercadoria: dec("100"), UF: "AM"}); !errors.Is(err, ErrRegiaoNaoAtendida) {
		t.Fatalf("esperava região não atendida, veio %v", err)
	}
}

func TestAlertasApontamHabilitacaoESeguroVencidos(t *testing.T) {
	c := transportadoraBase()
	hoje := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	venceu := hoje.AddDate(0, 0, -1)
	c.ANTTExpiry = &venceu
	c.InsuranceExpiry = &venceu
	uf := "SP"
	c.ServiceAreas = []*RegiaoAtendida{{State: &uf, IsActive: true}}
	c.Vehicles = []*Veiculo{{Plate: "ABC1234", IsActive: true}}

	alertas := c.Alertas(hoje)
	achou := map[string]bool{}
	for _, a := range alertas {
		achou[a] = true
	}
	if !achou["RNTRC vencido em 24/09/2026"] || !achou["seguro vencido em 24/09/2026"] {
		t.Fatalf("alertas de vencimento faltando: %v", alertas)
	}
}

func TestVeiculoCabeRespeitaPesoEVolume(t *testing.T) {
	v := &Veiculo{Plate: "ABC1234", CapacityKg: dec("1000"), CapacityM3: dec("10"), IsActive: true}
	if !v.Cabe(dec("900"), dec("9")) {
		t.Fatal("carga dentro da capacidade deveria caber")
	}
	if v.Cabe(dec("1100"), dec("1")) || v.Cabe(dec("10"), dec("11")) {
		t.Fatal("carga acima da capacidade não cabe")
	}
	// Capacidade zero é "não informada": não pode barrar a carga.
	semCapacidade := &Veiculo{Plate: "ABC1234", IsActive: true}
	if !semCapacidade.Cabe(dec("99999"), dec("999")) {
		t.Fatal("capacidade não informada não deveria barrar")
	}
}

func TestFaixaDeCEPInvertidaEhRecusada(t *testing.T) {
	de, ate := "02000000", "01000000"
	a := &RegiaoAtendida{PostalCodeFrom: &de, PostalCodeTo: &ate}
	if err := a.Validar(); err == nil {
		t.Fatal("faixa invertida deveria ser recusada")
	}
	semDestino := &RegiaoAtendida{}
	if err := semDestino.Validar(); err == nil {
		t.Fatal("região sem UF e sem CEP deveria ser recusada")
	}
	_ = texto("")
}
