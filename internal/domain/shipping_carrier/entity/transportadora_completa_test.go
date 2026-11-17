package entity

import (
	"strings"
	"testing"
	"time"
)

// renavamCom completa 10 dígitos com o verificador, pelo mesmo cálculo do DETRAN.
func renavamCom(base string) string {
	pesos := []int{3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	soma := 0
	for i, p := range pesos {
		soma += int(base[i]-'0') * p
	}
	dv := 11 - soma%11
	if dv >= 10 {
		dv = 0
	}
	return base + string(rune('0'+dv))
}

func TestRenavam(t *testing.T) {
	ok := renavamCom("0063988496")
	if !RenavamValido(ok) {
		t.Fatalf("%s deveria ser válido", ok)
	}
	if !RenavamValido(ok[2:]) {
		t.Fatal("RENAVAM antigo de 9 dígitos deveria ser completado com zeros")
	}
	errado := ok[:10] + string(rune('0'+(int(ok[10]-'0')+1)%10))
	if RenavamValido(errado) || RenavamValido("12345") || RenavamValido("abcdefghijk") {
		t.Fatal("RENAVAM com dígito errado ou formato inválido passou")
	}
}

func TestVeiculoValidaDadosDoMDFe(t *testing.T) {
	casos := map[string]func(v *Veiculo){
		"UF":         func(v *Veiculo) { v.PlateState = texto("XX") },
		"RENAVAM":    func(v *Veiculo) { v.Renavam = texto("12345678901") },
		"rodado":     func(v *Veiculo) { v.WheelType = texto("09") },
		"carroceria": func(v *Veiculo) { v.BodyType = texto("07") },
		"TAC":        func(v *Veiculo) { v.OwnerType = texto("0") },
		"CPF":        func(v *Veiculo) { v.DriverDocument = texto("11111111111") },
		"CNH":        func(v *Veiculo) { v.DriverCNHCategory = texto("Z") },
	}
	for nome, estraga := range casos {
		v := &Veiculo{Plate: "ABC1D23", IsActive: true}
		estraga(v)
		v.Normalizar()
		if err := v.Validar(); err == nil {
			t.Errorf("%s inválido passou", nome)
		}
	}
	v := &Veiculo{Plate: "abc-1d23", PlateState: texto("pr"), Renavam: texto(renavamCom("0063988496")),
		WheelType: texto("01"), BodyType: texto("02"), OwnerType: texto("1"), ANTTOwner: texto("1234.567-8"),
		DriverDocument: texto("529.982.247-25"), DriverCNHCategory: texto("e"), IsActive: true}
	v.Normalizar()
	if err := v.Validar(); err != nil {
		t.Fatalf("veículo válido recusado: %v", err)
	}
	if *v.PlateState != "PR" || *v.ANTTOwner != "12345678" || *v.DriverCNHCategory != "E" {
		t.Fatalf("normalização errada: %+v", v)
	}
}

func TestCotacaoUsaPesoCubadoETaxaDeDespacho(t *testing.T) {
	c := transportadoraBase()
	c.FreightMinValue = dec("0")
	c.FreightPctValue = dec("0")
	c.GrisPct = dec("0")
	c.TollPer100Kg = dec("0")
	c.CubageFactor = dec("300")
	c.DispatchFee = dec("25")
	// 100 kg reais em 2 m³ = 600 kg cubados: cobra 600 × 0,90 + 25.
	cot, err := c.Cotar(Carga{PesoKg: dec("100"), VolumeM3: dec("2"), UF: "SP"})
	if err != nil {
		t.Fatal(err)
	}
	if !cot.PesoCubado || !cot.PesoTaxado.Equal(dec("600")) || !cot.Total.Equal(dec("565")) {
		t.Fatalf("peso taxado %s cubado=%v total %s", cot.PesoTaxado, cot.PesoCubado, cot.Total)
	}
	// Carga densa: o peso real manda.
	cot, _ = c.Cotar(Carga{PesoKg: dec("1000"), VolumeM3: dec("1"), UF: "SP"})
	if cot.PesoCubado || !cot.Total.Equal(dec("925")) {
		t.Fatalf("carga densa: cubado=%v total %s", cot.PesoCubado, cot.Total)
	}
}

func TestCotacaoRecusaProdutoPerigoso(t *testing.T) {
	c := transportadoraBase()
	if _, err := c.Cotar(Carga{PesoKg: dec("10"), UF: "SP", ProdutoPerigoso: true}); err == nil {
		t.Fatal("transportadora que não aceita produto perigoso foi cotada")
	}
	c.AcceptsDangerousGoods = true
	if _, err := c.Cotar(Carga{PesoKg: dec("10"), UF: "SP", ProdutoPerigoso: true}); err != nil {
		t.Fatal(err)
	}
}

func TestAlertasAvisamVencimentoProximoECNH(t *testing.T) {
	hoje := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	c := transportadoraBase()
	logo := hoje.AddDate(0, 0, 10)
	longe := hoje.AddDate(0, 0, 90)
	c.ANTTExpiry = &logo
	c.InsuranceExpiry = &longe
	vencida := hoje.AddDate(0, 0, -1)
	c.Vehicles = []*Veiculo{{Plate: "ABC1D23", IsActive: true, DriverName: texto("João"), DriverCNHExpiry: &vencida}}
	alertas := strings.Join(c.Alertas(hoje), "|")
	for _, quer := range []string{"RNTRC vence em 27/11/2026", "CNH do motorista João (ABC1D23) vencida", "sem UF da placa ou tara"} {
		if !strings.Contains(alertas, quer) {
			t.Errorf("faltou %q em %s", quer, alertas)
		}
	}
	if strings.Contains(alertas, "seguro") {
		t.Errorf("seguro com 90 dias não deveria alertar: %s", alertas)
	}
}

func TestCotacaoSeparaImpedimentoDeAviso(t *testing.T) {
	base := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	c := transportadoraBase()
	venceu := base.AddDate(0, 0, -3)
	c.ANTTExpiry = &venceu
	c.InsuranceCoverage = dec("5000")
	c.Vehicles = []*Veiculo{{Plate: "ABC1D23", IsActive: true, CapacityKg: dec("50")}}
	cot, err := c.Cotar(Carga{PesoKg: dec("100"), VolumeM3: dec("1"), ValorMercadoria: dec("9000"), UF: "SP", Base: base})
	if err != nil {
		t.Fatal(err)
	}
	imp := strings.Join(cot.Impedimentos, "|")
	for _, quer := range []string{"RNTRC vencido em 14/11/2026", "cobertura do seguro", "nenhum veículo"} {
		if !strings.Contains(imp, quer) {
			t.Errorf("faltou o impedimento %q em %s", quer, imp)
		}
	}
	avisos := strings.Join(cot.Alertas, "|")
	if strings.Contains(avisos, "RNTRC vencido") {
		t.Errorf("impedimento repetido nos avisos: %s", avisos)
	}
	if !strings.Contains(avisos, "sem fator de cubagem") {
		t.Errorf("faltou o aviso de cubagem para carga volumosa: %s", avisos)
	}
}

func TestCotacaoSemTabelaNaoViraFreteZero(t *testing.T) {
	c := &Transportadora{IsActive: true, Modal: ModalRodoviario}
	if _, err := c.Cotar(Carga{PesoKg: dec("100"), ValorMercadoria: dec("1000"), UF: "PR"}); err == nil {
		t.Fatal("transportadora sem tabela foi cotada com frete zero")
	}
}
