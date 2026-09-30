package entity

import (
	"math"
	"testing"
	"time"
)

func quase(a, b float64) bool { return math.Abs(a-b) < 0.000001 }

func regra(code string, base BaseDeRateio, metodo MetodoDeRateio, taxa float64) *RegraDeRateio {
	return &RegraDeRateio{
		ID: int64(len(code)), Code: code, Description: code, Base: base,
		Method: metodo, Rate: taxa, IsActive: true,
		ValidFrom: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

var base = ComponentesDeCusto{
	Material: 100, Setup: 10, Maquina: 40, MaoDeObra: 20, Subcontratacao: 30,
}

func TestRateioPercentualIncideSobreABaseCerta(t *testing.T) {
	casos := []struct {
		b        BaseDeRateio
		esperado float64
	}{
		{BaseMaterial, 10},      // 10% de 100
		{BaseSetup, 1},          // 10% de 10
		{BaseMaquina, 4},        // 10% de 40
		{BaseMaoDeObra, 2},      // 10% de 20
		{BaseConversao, 7},      // 10% de (10+40+20)
		{BaseSubcontratacao, 3}, // 10% de 30
		{BaseTotal, 20},         // 10% de (100+70+30)
	}
	for _, c := range casos {
		t.Run(string(c.b), func(t *testing.T) {
			total, rastro := AplicarRateios(base, HorasDoRoteiro{}, []*RegraDeRateio{regra("R1", c.b, MetodoPercentual, 0.10)})
			if !quase(total, c.esperado) {
				t.Fatalf("base %s aplicou %.6f, esperado %.6f", c.b, total, c.esperado)
			}
			if len(rastro) != 1 || rastro[0].Code != "R1" {
				t.Fatalf("rastro do rateio ausente ou errado: %+v", rastro)
			}
			// O rastro precisa dizer sobre QUANTO incidiu, senão o indireto é um
			// número no total que ninguém consegue explicar.
			if rastro[0].BaseValue <= 0 {
				t.Fatalf("rastro sem o valor da base: %+v", rastro[0])
			}
		})
	}
}

func TestRateioPorHoraUsaAsHorasDoRoteiro(t *testing.T) {
	horas := HorasDoRoteiro{Maquina: 2, MaoDeObra: 1.5, Setup: 0.5}
	total, _ := AplicarRateios(base, horas, []*RegraDeRateio{regra("E1", BaseMaquina, MetodoValorPorHora, 18)})
	if !quase(total, 36) {
		t.Fatalf("energia por hora-máquina = %.4f, esperado 36 (2h × 18)", total)
	}
	total, _ = AplicarRateios(base, horas, []*RegraDeRateio{regra("S1", BaseConversao, MetodoValorPorHora, 10)})
	if !quase(total, 40) {
		t.Fatalf("supervisão sobre conversão = %.4f, esperado 40 (4h × 10)", total)
	}
}

func TestRateioPorUnidadeNaoDependeDeBase(t *testing.T) {
	total, rastro := AplicarRateios(base, HorasDoRoteiro{}, []*RegraDeRateio{regra("U1", BaseTotal, MetodoValorPorUnidade, 2.5)})
	if !quase(total, 2.5) {
		t.Fatalf("valor por unidade = %.4f, esperado 2,50", total)
	}
	if rastro[0].BaseValue != 1 {
		t.Fatalf("base do valor por unidade = %.4f, esperado 1", rastro[0].BaseValue)
	}
}

// TestOrdemDasRegrasNaoMudaOResultado é o teste que justifica a ordenação fixa:
// com a ordem vindo do cadastro, duas instalações com as MESMAS regras chegariam a
// custos diferentes, e a divergência seria impossível de explicar.
func TestOrdemDasRegrasNaoMudaOResultado(t *testing.T) {
	r1 := regra("A-MATERIAL", BaseMaterial, MetodoPercentual, 0.08)
	r2 := regra("B-CONVERSAO", BaseConversao, MetodoPercentual, 0.15)
	r3 := regra("C-TOTAL", BaseTotal, MetodoPercentual, 0.05)

	totalA, _ := AplicarRateios(base, HorasDoRoteiro{}, []*RegraDeRateio{r1, r2, r3})
	totalB, _ := AplicarRateios(base, HorasDoRoteiro{}, []*RegraDeRateio{r3, r1, r2})
	totalC, _ := AplicarRateios(base, HorasDoRoteiro{}, []*RegraDeRateio{r2, r3, r1})
	if !quase(totalA, totalB) || !quase(totalA, totalC) {
		t.Fatalf("a ordem de cadastro mudou o custo: %.6f, %.6f, %.6f", totalA, totalB, totalC)
	}
	// 8% de 100 + 15% de 70 + 5% de 200 = 8 + 10,5 + 10 = 28,5
	if !quase(totalA, 28.5) {
		t.Fatalf("total dos indiretos = %.4f, esperado 28,50", totalA)
	}
}

// TestIndiretoNaoIncideSobreIndireto: se a base de TOTAL incluísse o overhead já
// aplicado, a ordem voltaria a importar e o custo cresceria a cada regra nova.
func TestIndiretoNaoIncideSobreIndireto(t *testing.T) {
	comOverhead := base
	comOverhead.Overhead = 1000 // valor absurdo de propósito
	total, _ := AplicarRateios(comOverhead, HorasDoRoteiro{}, []*RegraDeRateio{regra("T1", BaseTotal, MetodoPercentual, 0.10)})
	if !quase(total, 20) {
		t.Fatalf("total = %.4f: a base de TOTAL está incluindo o overhead já aplicado", total)
	}
}

func TestBaseZeradaNaoGeraLinhaDeRastro(t *testing.T) {
	semTerceiro := ComponentesDeCusto{Material: 100}
	total, rastro := AplicarRateios(semTerceiro, HorasDoRoteiro{}, []*RegraDeRateio{
		regra("M1", BaseMaterial, MetodoPercentual, 0.10),
		regra("X1", BaseSubcontratacao, MetodoPercentual, 0.50),
	})
	if !quase(total, 10) {
		t.Fatalf("total = %.4f, esperado 10", total)
	}
	if len(rastro) != 1 || rastro[0].Code != "M1" {
		t.Fatalf("rastro deveria ter só a regra que aplicou algo: %+v", rastro)
	}
}

func TestVigenciaDaRegra(t *testing.T) {
	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fim := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	r := &RegraDeRateio{IsActive: true, ValidFrom: inicio, ValidTo: &fim}

	if r.VigenteEm(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("regra vigente antes do início")
	}
	if !r.VigenteEm(inicio) {
		t.Fatal("o primeiro dia de vigência tem de valer")
	}
	if !r.VigenteEm(fim) {
		t.Fatal("o último dia de vigência tem de valer")
	}
	if r.VigenteEm(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("regra vigente depois do fim")
	}
	r.IsActive = false
	if r.VigenteEm(inicio) {
		t.Fatal("regra inativa não pode valer nem dentro da vigência")
	}
	semFim := &RegraDeRateio{IsActive: true, ValidFrom: inicio}
	if !semFim.VigenteEm(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("regra sem data final vale indefinidamente")
	}
}

func TestEscopoDaRegra(t *testing.T) {
	centros := map[int64]bool{7: true, 9: true}
	item := int64(1001)

	geral := &RegraDeRateio{}
	if !geral.AplicaAoItem(item, centros) {
		t.Fatal("regra sem escopo vale para toda a fábrica")
	}

	doItem := &RegraDeRateio{ItemCode: &item}
	if !doItem.AplicaAoItem(item, centros) {
		t.Fatal("regra do item deveria aplicar ao próprio item")
	}
	outro := int64(2002)
	if (&RegraDeRateio{ItemCode: &outro}).AplicaAoItem(item, centros) {
		t.Fatal("regra de outro item não pode aplicar")
	}

	centro := int64(7)
	if !(&RegraDeRateio{WorkCenterID: &centro}).AplicaAoItem(item, centros) {
		t.Fatal("regra do centro 7 deveria aplicar: o roteiro passa por ele")
	}
	fora := int64(99)
	if (&RegraDeRateio{WorkCenterID: &fora}).AplicaAoItem(item, centros) {
		t.Fatal("regra de centro que o roteiro não usa não pode aplicar")
	}
	// Escopo duplo: item certo E centro errado não aplica. Sem a conjunção, uma
	// regra de energia da usinagem entraria no custo de item que não usina.
	if (&RegraDeRateio{ItemCode: &item, WorkCenterID: &fora}).AplicaAoItem(item, centros) {
		t.Fatal("item certo com centro errado não pode aplicar")
	}
}

func TestRegrasVigentesCombinaVigenciaEEscopo(t *testing.T) {
	hoje := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	centro := int64(7)
	fora := int64(99)
	antiga := regra("VELHA", BaseMaterial, MetodoPercentual, 0.1)
	fimAntigo := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	antiga.ValidTo = &fimAntigo
	doCentro := regra("CENTRO", BaseMaquina, MetodoPercentual, 0.1)
	doCentro.WorkCenterID = &centro
	deOutroCentro := regra("OUTRO", BaseMaquina, MetodoPercentual, 0.1)
	deOutroCentro.WorkCenterID = &fora
	geral := regra("GERAL", BaseTotal, MetodoPercentual, 0.02)

	vigentes := RegrasVigentes([]*RegraDeRateio{antiga, doCentro, deOutroCentro, geral},
		1001, map[int64]bool{7: true}, hoje)
	if len(vigentes) != 2 {
		t.Fatalf("vigentes = %d, esperado 2 (CENTRO e GERAL)", len(vigentes))
	}
	nomes := map[string]bool{}
	for _, v := range vigentes {
		nomes[v.Code] = true
	}
	if !nomes["CENTRO"] || !nomes["GERAL"] {
		t.Fatalf("regras selecionadas erradas: %v", nomes)
	}
}

func TestComponentesTotalENaoDobraOsCortes(t *testing.T) {
	c := ComponentesDeCusto{
		Material: 100, Setup: 10, Maquina: 40, MaoDeObra: 20,
		Subcontratacao: 30, Overhead: 15,
		// Os cortes por nível descrevem os MESMOS valores por outro ângulo.
		NivelProprio: 85, NivelInferior: 130,
	}
	if !quase(c.Total(), 215) {
		t.Fatalf("total = %.4f, esperado 215 — os cortes por nível não entram na soma", c.Total())
	}
	if !quase(c.Conversao(), 70) {
		t.Fatalf("conversão = %.4f, esperado 70", c.Conversao())
	}
}
