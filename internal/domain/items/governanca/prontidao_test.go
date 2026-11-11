package governanca

import (
	"encoding/json"
	"strings"
	"testing"
)

func fichaDe(tipo int, dados map[string]any) Ficha {
	f := Ficha{Tipo: tipo, Dados: map[string]json.RawMessage{}}
	for k, v := range dados {
		b, _ := json.Marshal(v)
		f.Dados[k] = b
	}
	return f
}

func processo(t *testing.T, ps []Prontidao, nome string) Prontidao {
	t.Helper()
	for _, p := range ps {
		if p.Processo == nome {
			return p
		}
	}
	t.Fatalf("processo %s não veio", nome)
	return Prontidao{}
}

func TestProntidaoDoItemComprado(t *testing.T) {
	f := fichaDe(Comprado, map[string]any{"warehouse_code": 1, "supplies_purchase_uom": "KG", "supplies_warehouse_code": 2, "accounting_origin": 0, "health": "ATIVO"})
	f.TemFornecedor, f.Fiscal.NCM = true, "72085100"
	ps := AvaliarProntidao(f, Liberado, nil)
	if !processo(t, ps, "COMPRAR").Pronto || !processo(t, ps, "FATURAR").Pronto {
		t.Fatalf("item completo deveria estar pronto: %+v", ps)
	}
	if processo(t, ps, "PRODUZIR").Aplica {
		t.Fatal("item comprado não se produz")
	}
	f.TemFornecedor, f.CompraEmOutraUM = false, true
	c := processo(t, AvaliarProntidao(f, Liberado, nil), "COMPRAR")
	if c.Pronto || len(c.Pendencias) < 2 {
		t.Fatalf("sem fornecedor e sem conversão não compra: %+v", c)
	}
}

func TestProntidaoExigeRotaQuandoORoteiroEInspecao(t *testing.T) {
	f := fichaDe(Comprado, map[string]any{"supplies_purchase_uom": "UN", "supplies_receipt_routing": "INSPECAO"})
	f.TemFornecedor = true
	if processo(t, AvaliarProntidao(f, Liberado, nil), "COMPRAR").Pronto {
		t.Fatal("exigir inspeção sem rota ativa deveria impedir")
	}
	f.TemRotaInspecao = true
	if !processo(t, AvaliarProntidao(f, Liberado, nil), "COMPRAR").Pronto {
		t.Fatal("com rota ativa o item compra")
	}
}

func TestPoliticaDoStatusEntraNaProntidao(t *testing.T) {
	f := fichaDe(Comprado, map[string]any{"commercial_description": "X", "health": "ATIVO"})
	matriz := MatrizDePolitica([]RegraPolitica{{Status: Descontinuado, Processo: ProcVenda, Politica: PoliticaAviso}})
	ps := AvaliarProntidao(f, Descontinuado, matriz)
	v := processo(t, ps, "VENDER")
	if v.Politica != PoliticaAviso || !v.Pronto {
		t.Fatalf("venda com aviso continua pronta: %+v", v)
	}
	if c := processo(t, ps, "COMPRAR"); c.Politica != PoliticaBloqueado || c.Pronto {
		t.Fatalf("sem ajuste, descontinuado bloqueia a compra: %+v", c)
	}
}

func TestMatrizDePoliticaCompletaComBloqueio(t *testing.T) {
	m := MatrizDePolitica(nil)
	if len(m) != len(StatusComPolitica)*len(Processos) {
		t.Fatalf("matriz com %d células", len(m))
	}
	for _, r := range m {
		if r.Politica != PoliticaBloqueado {
			t.Fatalf("o padrão preserva o comportamento antigo: %+v", r)
		}
	}
	if PoliticaDe(Liberado, ProcCompra, m) != PoliticaLiberado {
		t.Fatal("item liberado não sofre política")
	}
}

func TestConferirRegraPolitica(t *testing.T) {
	if (RegraPolitica{Status: Liberado, Processo: ProcCompra, Politica: PoliticaAviso}).Conferir() == nil {
		t.Fatal("LIBERADO não tem política ajustável")
	}
	if (RegraPolitica{Status: Bloqueado, Processo: "X", Politica: PoliticaAviso}).Conferir() == nil {
		t.Fatal("processo desconhecido")
	}
	if err := (RegraPolitica{Status: Bloqueado, Processo: ProcVenda, Politica: PoliticaAviso}).Conferir(); err != nil {
		t.Fatal(err)
	}
}

func TestImpactoBloqueiaUnidadeComSaldo(t *testing.T) {
	antes, depois := Retrato{Unidade: "UN"}, Retrato{Unidade: "KG"}
	im := AvaliarImpacto(antes, depois, Uso{Saldo: "12.5"})
	if len(im) != 1 || im[0].Severidade != ImpactoBloqueia {
		t.Fatalf("impactos: %+v", im)
	}
	if err := TemBloqueio(im); err == nil || !strings.Contains(err.Error(), "Zere o saldo") {
		t.Fatalf("erro: %v", err)
	}
	if im := AvaliarImpacto(antes, depois, Uso{Saldo: "0.000000"}); len(im) != 0 {
		t.Fatalf("sem saldo nem uso a unidade muda: %+v", im)
	}
	if im := AvaliarImpacto(antes, depois, Uso{Saldo: "0", PedidosVenda: 1}); TemBloqueio(im) == nil {
		t.Fatal("pedido aberto na unidade antiga bloqueia")
	}
	if im := AvaliarImpacto(antes, depois, Uso{Saldo: "0", EstruturasPai: []string{"A"}}); len(im) != 1 || im[0].Severidade != ImpactoAtencao {
		t.Fatalf("estrutura é atenção: %+v", im)
	}
}

func TestImpactoDoTipoEDosCamposDeAviso(t *testing.T) {
	if TemBloqueio(AvaliarImpacto(Retrato{Tipo: "COMPRADO"}, Retrato{Tipo: "FABRICADO"}, Uso{Saldo: "0", PedidosCompra: 1})) == nil {
		t.Fatal("pedido de compra aberto bloqueia a troca do tipo")
	}
	im := AvaliarImpacto(Retrato{GTIN: "1", TipoEFD: "04", Validade: "10", UnidadeTributavel: "KG", Fator: "1"},
		Retrato{GTIN: "2", TipoEFD: "03", Validade: "20", UnidadeTributavel: "KG", Fator: "2"}, Uso{Saldo: "0"})
	if len(im) != 4 || TemBloqueio(im) != nil {
		t.Fatalf("avisos: %+v", im)
	}
	if im := AvaliarImpacto(Retrato{}, Retrato{GTIN: "789"}, Uso{Saldo: "0"}); len(im) != 0 {
		t.Fatalf("informar o primeiro GTIN não tem impacto: %+v", im)
	}
}

func TestPreviaFiscalConfereOQueASefazRecusaria(t *testing.T) {
	f := fichaDe(Comprado, map[string]any{"accounting_gtin": "7891000100104", "warehouse_unit_of_measurement": "UN", "accounting_cest": "123"})
	f.Fiscal.ClassTrib = "12345"
	p := AvaliarFiscal(f, Uso{Saldo: "0"})
	if p.PodeFaturar {
		t.Fatalf("NCM, origem, GTIN, CEST e cClassTrib estão errados: %+v", p.Checks)
	}
	if len(p.Checks) != 5 {
		t.Fatalf("checks: %+v", p.Checks)
	}
	if p.UnidadeTributavel != "UN" || p.FatorTributavel != "1" {
		t.Fatalf("sem unidade própria, repete a comercial: %+v", p)
	}
}

func TestPreviaFiscalAvisaEscalaSemCEST(t *testing.T) {
	f := fichaDe(Comprado, map[string]any{"warehouse_unit_of_measurement": "UN", "accounting_relevant_scale": true})
	avisou := func(p PreviaFiscal) bool {
		for _, c := range p.Checks {
			if strings.Contains(c.Mensagem, "junto do CEST") {
				return true
			}
		}
		return false
	}
	if !avisou(AvaliarFiscal(f, Uso{Saldo: "0"})) {
		t.Fatal("sem CEST, a escala fica fora da nota: a tela precisa dizer")
	}
	f = fichaDe(Comprado, map[string]any{"warehouse_unit_of_measurement": "UN", "accounting_relevant_scale": true, "accounting_cest": "0107500"})
	if avisou(AvaliarFiscal(f, Uso{Saldo: "0"})) {
		t.Fatal("com CEST não há o que avisar")
	}
}

func TestGTINBrasileiro(t *testing.T) {
	for gtin, esperado := range map[string]bool{"7891000100103": true, "7901000100109": true, "17891000100100": true, "4006381333931": false, "14006381333938": false, "96385074": false} {
		if gtinBrasileiro(gtin) != esperado {
			t.Fatalf("%s: esperava %v", gtin, esperado)
		}
	}
}

func TestPreviaFiscalMostraTipoEFD(t *testing.T) {
	f := fichaDe(Fabricado, map[string]any{"accounting_origin": 0, "warehouse_unit_of_measurement": "UN"})
	f.Fiscal.NCM, f.Fiscal.ClassTrib = "84137090", "000001"
	p := AvaliarFiscal(f, Uso{Saldo: "0", EstruturasPai: []string{"PAI"}})
	if p.TipoEFD != "04" || p.TipoEFDOrigem != TipoEFDDeduzido || !p.PodeFaturar {
		t.Fatalf("prévia: %+v", p)
	}
	if len(p.Checks) != 1 {
		t.Fatalf("fabricado que é componente deve avisar do tipo: %+v", p.Checks)
	}
	f = fichaDe(Fabricado, map[string]any{"accounting_origin": 0, "accounting_efd_item_type": "03"})
	f.Fiscal.NCM = "84137090"
	if p := AvaliarFiscal(f, Uso{Saldo: "0"}); p.TipoEFD != "03" || p.TipoEFDOrigem != TipoEFDInformado {
		t.Fatalf("o tipo informado vence: %+v", p)
	}
}

func TestProntidaoDosIndicadores(t *testing.T) {
	f := fichaDe(Fabricado, map[string]any{"commercial_sale_type": "VENDA", "engineering_requires_drawing": true, "is_tool": true, "warehouse_code": 1, "planning_active": false, "engineering_oem": true})
	f.TemEstrutura, f.TemRoteiro, f.TemCustoPadrao = true, true, true
	ps := AvaliarProntidao(f, Liberado, nil)
	produzir := processo(t, ps, "PRODUZIR")
	if produzir.Pronto {
		t.Fatalf("exige desenho e não tem: %+v", produzir)
	}
	textos := func(p Prontidao) string {
		out := ""
		for _, x := range p.Pendencias {
			out += x.Mensagem + ";"
		}
		return out
	}
	if !strings.Contains(textos(produzir), "Fora do MRP") {
		t.Fatalf("item desligado avisa: %s", textos(produzir))
	}
	if !strings.Contains(textos(processo(t, ps, "ESTOCAR")), "ficha de vida útil") {
		t.Fatal("ferramenta sem ficha avisa")
	}
	if !strings.Contains(textos(processo(t, ps, "VENDER")), "sem o cliente dono") {
		t.Fatal("OEM sem cliente avisa")
	}
	f.TemDesenho, f.TemFichaFerramenta = true, true
	if !processo(t, AvaliarProntidao(f, Liberado, nil), "PRODUZIR").Pronto {
		t.Fatal("com desenho produz")
	}
}

func TestSucataSaiComoSubprodutoNaPrevia(t *testing.T) {
	f := fichaDe(Comprado, map[string]any{"is_scrap": true, "accounting_origin": 0})
	f.Fiscal.NCM = "72044900"
	if p := AvaliarFiscal(f, Uso{Saldo: "0"}); p.TipoEFD != "05" {
		t.Fatalf("sucata é 05: %+v", p)
	}
}

func TestVersaoDaEstruturaGovernaAProducao(t *testing.T) {
	base := func() Ficha {
		f := fichaDe(Fabricado, map[string]any{"warehouse_code": 1, "health": "ATIVO"})
		f.TemEstrutura, f.TemRoteiro, f.TemCustoPadrao = true, true, true
		return f
	}
	temPendencia := func(p Prontidao, trecho string) bool {
		for _, x := range p.Pendencias {
			if strings.Contains(x.Mensagem, trecho) {
				return true
			}
		}
		return false
	}
	// Sem versão nenhuma: comportamento de antes (não exige aprovação).
	if p := processo(t, AvaliarProntidao(base(), Liberado, nil), "PRODUZIR"); temPendencia(p, "versão") {
		t.Fatalf("item sem controle de versão não pode ser cobrado: %+v", p)
	}
	// Com versão, mas nenhuma aprovada: impede.
	f := base()
	f.ControlaVersaoEstrutura = true
	if p := processo(t, AvaliarProntidao(f, Liberado, nil), "PRODUZIR"); p.Pronto || !temPendencia(p, "aprovada") {
		t.Fatalf("sem MBOM aprovada deveria impedir: %+v", p)
	}
	// Aprovada, mas a estrutura mudou depois: só atenção.
	f.TemMBOMAprovada, f.EstruturaMudouAposAprov = true, true
	if p := processo(t, AvaliarProntidao(f, Liberado, nil), "PRODUZIR"); !p.Pronto || !temPendencia(p, "mudou") {
		t.Fatalf("mudança após aprovação é atenção, não impedimento: %+v", p)
	}
}
