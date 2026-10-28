package fiscal_uc

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// Convivência dos dois modelos na NF-e de saída, contra o formato da Focus.
//
// A reforma só ACRESCENTA o grupo IBS/CBS no item: todo campo do modelo atual
// (ICMS, ICMS-ST, IPI, PIS, COFINS, cBenef, rateios, totais, partes, pagamento)
// tem de sair idêntico ao da nota sem reforma, em qualquer cenário. E os campos
// novos só podem ter nomes que existem na referência oficial da Focus
// (campos.focusnfe.com.br/nfe/NotaFiscalXML.html), no formato dela.

// camposFocusIBSCBS: os nomes de campo de IBS/CBS do item aceitos pela Focus.
var camposFocusIBSCBS = map[string]bool{}

func init() {
	for _, c := range strings.Fields(`cbs_aliquota cbs_aliquota_compra_governamental cbs_aliquota_efetiva cbs_aliquota_efetiva_regular
		cbs_credito_percentual cbs_percentual_devolvido cbs_percentual_diferimento cbs_percentual_reducao_aliquota cbs_valor
		cbs_valor_ajuste_competencia cbs_valor_condicao_suspensiva cbs_valor_credito_presumido cbs_valor_diferido
		cbs_valor_estorno_credito cbs_valor_transferido cbs_valor_tributo_calculado cbs_valor_tributo_devolvido
		cbs_valor_tributo_regular ibs_aliquota_competencia_municipio ibs_aliquota_competencia_uf
		ibs_aliquota_efetiva_regular_municipio ibs_aliquota_efetiva_regular_uf ibs_cbs_base_calculo
		ibs_cbs_classificacao_tributaria ibs_cbs_classificacao_tributaria_regular ibs_cbs_indicacao_doacao ibs_cbs_municipio
		ibs_cbs_situacao_tributaria ibs_cbs_situacao_tributaria_regular ibs_mun_aliquota ibs_mun_aliquota_efetiva
		ibs_mun_percentual_devolvido ibs_mun_percentual_diferimento ibs_mun_percentual_reducao_aliquota ibs_mun_valor
		ibs_mun_valor_diferido ibs_mun_valor_tributo_devolvido ibs_percentual_credito_presumido ibs_uf_aliquota
		ibs_uf_aliquota_efetiva ibs_uf_percentual_devolvido ibs_uf_percentual_diferimento ibs_uf_percentual_reducao_aliquota
		ibs_uf_valor ibs_uf_valor_diferido ibs_uf_valor_tributo_devolvido ibs_valor_ajuste_competencia
		ibs_valor_condicao_suspensiva ibs_valor_credito_presumido ibs_valor_estorno_credito ibs_valor_total
		ibs_valor_transferido ibs_valor_tributo_calculado_municipio ibs_valor_tributo_calculado_uf
		ibs_valor_tributo_regular_municipio ibs_valor_tributo_regular_uf`) {
		camposFocusIBSCBS[c] = true
	}
}

func f64p(v float64) *float64 { return &v }

// notaCompleta: o que a indústria emite no dia a dia — linha tributada com
// IPI e PIS/COFINS, linha isenta de ICMS com cBenef e linha com ICMS-ST — e
// frete, seguro e desconto para ratear.
func notaCompleta() (*fiscalentity.FiscalExit, []*fiscalentity.FiscalExitItem) {
	nota := notaBase()
	nota.ValorProdutos, nota.ValorFrete, nota.ValorSeguro, nota.ValorDesconto = 3000, 90, 15, 60
	nota.ValorIPI, nota.ValorICMS, nota.ValorICMSST = 150, 288, 45
	nota.ValorTotal = 3000 + 150 + 45 + 90 + 15 - 60
	cst := func(s string) *string { return &s }
	itens := []*fiscalentity.FiscalExitItem{
		{ID: 21, Sequence: 1, ItemCode: i64p(5), Ncm: strp("73269000"), Cfop: "6101", Quantity: 10, UnitPrice: 150, TotalPrice: 1500,
			BaseICMS: 1500, AliqICMS: 12, ValorICMS: 180, BaseIPI: 1500, AliqIPI: 5, ValorIPI: 75, AliqPIS: 1.65, ValorPIS: 24.75,
			AliqCOFINS: 7.6, ValorCOFINS: 114, CstICMS: cst("00"), CstIPI: cst("50"), CstPIS: cst("01"), CstCOFINS: cst("01"),
			OrigemMercadoria: "0", Description: strp("BALANCA RN-01001"), UnidadeComercial: strp("UN")},
		{ID: 22, Sequence: 2, ItemCode: i64p(6), Ncm: strp("84382000"), Cfop: "6101", Quantity: 2.5, UnitPrice: 400, TotalPrice: 1000,
			BaseIPI: 1000, AliqIPI: 7.5, ValorIPI: 75, CstICMS: cst("40"), CstIPI: cst("50"), CstPIS: cst("01"), CstCOFINS: cst("01"),
			AliqPIS: 1.65, ValorPIS: 16.5, AliqCOFINS: 7.6, ValorCOFINS: 76, OrigemMercadoria: "0", Description: strp("PICADOR"),
			UnidadeComercial: strp("KG"), CodigoBeneficioFiscal: strp("PR800001")},
		{ID: 23, Sequence: 3, ItemCode: i64p(7), Ncm: strp("40169300"), Cfop: "6401", Quantity: 4, UnitPrice: 125, TotalPrice: 500,
			BaseICMS: 500, AliqICMS: 12, ValorICMS: 60, BaseICMSST: 650, AliqICMSST: 18, ValorICMSST: 45, MVA: 30,
			CstICMS: cst("10"), CstIPI: cst("53"), CstPIS: cst("01"), CstCOFINS: cst("01"), AliqPIS: 1.65, ValorPIS: 8.25,
			AliqCOFINS: 7.6, ValorCOFINS: 38, OrigemMercadoria: "0", Description: strp("VEDACAO"), UnidadeComercial: strp("PC")},
	}
	return nota, itens
}

func payloadMapa(t *testing.T, p focusnfe.NFEPayload) map[string]any {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func ehCampoReforma(k string) bool {
	return strings.HasPrefix(k, "ibs_") || strings.HasPrefix(k, "cbs_")
}

// semReforma tira os campos do grupo IBS/CBS dos itens, deixando só o modelo atual.
func semReforma(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		if k != "items" {
			out[k] = v
			continue
		}
		var itens []any
		for _, it := range v.([]any) {
			limpo := map[string]any{}
			for ik, iv := range it.(map[string]any) {
				if !ehCampoReforma(ik) {
					limpo[ik] = iv
				}
			}
			itens = append(itens, limpo)
		}
		out[k] = itens
	}
	return out
}

func camposReforma(it map[string]any) []string {
	var ks []string
	for k := range it {
		if ehCampoReforma(k) {
			ks = append(ks, k)
		}
	}
	sort.Strings(ks)
	return ks
}

func casas(v float64) int {
	for c := 0; c <= 8; c++ {
		if math.Abs(v*math.Pow10(c)-math.Round(v*math.Pow10(c))) < 1e-6 {
			return c
		}
	}
	return 9
}

type cenarioReforma struct {
	nome    string
	cfg     func(*fiscalentity.FiscalConfig)
	emissao time.Time
	repo    *reformaFake
	// grupo esperado por linha: "" (sem grupo), "SEM_VALORES", "INTEGRAL", "REDUZIDO"
	grupos [3]string
}

func TestConvivenciaDosDoisModelosNoPayloadFocus(t *testing.T) {
	d := decimal.RequireFromString
	dia2026 := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	dia2027 := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	cbs2027 := map[int]*reforma.Aliquotas{2027: {IBSUF: d("0.05"), IBSMun: d("0.05"), CBS: d("8.7")}}
	cenarios := []cenarioReforma{
		{nome: "simples nacional: só o modelo atual", cfg: func(c *fiscalentity.FiscalConfig) { c.RegimeTributario = "1" }, emissao: dia2026,
			repo: &reformaFake{tabela: tabelaTeste()}, grupos: [3]string{"", "", ""}},
		{nome: "empresa em SOMENTE_ATUAL", emissao: dia2026,
			repo: &reformaFake{tabela: tabelaTeste(), modo: reforma.ModoSomenteAtual}, grupos: [3]string{"", "", ""}},
		{nome: "regime normal 2026: tributação integral", emissao: dia2026,
			repo: &reformaFake{tabela: tabelaTeste()}, grupos: [3]string{"INTEGRAL", "INTEGRAL", "INTEGRAL"}},
		{nome: "2026 com redução, bonificação pelo CFOP e integral", emissao: dia2026,
			repo:   &reformaFake{tabela: tabelaTeste(), candidatos: map[int64]reforma.Candidatos{21: {DaClassificacao: "200030"}, 22: {DoCFOP: "410001"}}},
			grupos: [3]string{"REDUZIDO", "SEM_VALORES", "INTEGRAL"}},
		{nome: "2027 com a CBS da empresa", emissao: dia2027,
			repo:   &reformaFake{tabela: tabelaTeste(), aliquotas: cbs2027, candidatos: map[int64]reforma.Candidatos{23: {DoItem: "200030"}}},
			grupos: [3]string{"INTEGRAL", "INTEGRAL", "REDUZIDO"}},
	}

	notaRef, itensRef := notaCompleta()
	base, _ := montarPayloadNFe(notaRef, itensRef, configBase(), PlanoDaNota{}, TributacaoReforma{})
	referencia := payloadMapa(t, base)
	for _, it := range referencia["items"].([]any) {
		if ks := camposReforma(it.(map[string]any)); len(ks) != 0 {
			t.Fatalf("a nota de referência não pode ter campos da reforma: %v", ks)
		}
	}

	for _, c := range cenarios {
		t.Run(c.nome, func(t *testing.T) {
			nota, itens := notaCompleta()
			nota.DataEmissao = c.emissao
			cfg := configBase()
			if c.cfg != nil {
				c.cfg(cfg)
			}
			rt, err := resolverReformaDaNota(context.Background(), c.repo, nota, itens, cfg)
			if err != nil {
				t.Fatal(err)
			}
			p, retrato := montarPayloadNFe(nota, itens, cfg, PlanoDaNota{}, rt)
			m := payloadMapa(t, p)

			// 1. O modelo atual sai idêntico ao da mesma nota sem a reforma.
			refNota, refItens := notaCompleta()
			refNota.DataEmissao = c.emissao
			semGrupo, _ := montarPayloadNFe(refNota, refItens, cfg, PlanoDaNota{}, TributacaoReforma{})
			ref := semReforma(payloadMapa(t, semGrupo))
			if !reflect.DeepEqual(semReforma(m), ref) {
				t.Fatalf("a reforma alterou campos do modelo atual:\ncom:  %v\nsem:  %v", semReforma(m), ref)
			}
			// 2. Nenhum campo de reforma fora do item (totais a Focus calcula).
			for k := range m {
				if ehCampoReforma(k) {
					t.Fatalf("campo de reforma no cabeçalho: %s", k)
				}
			}

			gruposComValores := 0
			for i, raw := range m["items"].([]any) {
				it := raw.(map[string]any)
				ks := camposReforma(it)
				// 3. Só nomes da referência da Focus.
				for _, k := range ks {
					if !camposFocusIBSCBS[k] {
						t.Fatalf("linha %d: campo %q não existe na referência da Focus", i+1, k)
					}
				}
				integral := []string{"cbs_aliquota", "cbs_valor", "ibs_cbs_base_calculo", "ibs_cbs_classificacao_tributaria", "ibs_cbs_situacao_tributaria",
					"ibs_mun_aliquota", "ibs_mun_valor", "ibs_uf_aliquota", "ibs_uf_valor", "ibs_valor_total"}
				esperado := map[string][]string{
					"":            nil,
					"SEM_VALORES": {"ibs_cbs_classificacao_tributaria", "ibs_cbs_situacao_tributaria"},
					"INTEGRAL":    integral,
					"REDUZIDO": append(append([]string{}, integral...), "cbs_aliquota_efetiva", "cbs_percentual_reducao_aliquota",
						"ibs_mun_aliquota_efetiva", "ibs_mun_percentual_reducao_aliquota", "ibs_uf_aliquota_efetiva", "ibs_uf_percentual_reducao_aliquota"),
				}[c.grupos[i]]
				sort.Strings(esperado)
				if !reflect.DeepEqual(ks, esperado) && !(len(ks) == 0 && len(esperado) == 0) {
					t.Fatalf("linha %d (%s): campos %v, esperado %v", i+1, c.grupos[i], ks, esperado)
				}
				if len(ks) == 0 {
					continue
				}
				// 4. Formato da Focus: CST String[3], cClassTrib String[6] com o CST no início.
				cst, _ := it["ibs_cbs_situacao_tributaria"].(string)
				cls, _ := it["ibs_cbs_classificacao_tributaria"].(string)
				if len(cst) != 3 || len(cls) != 6 || !strings.HasPrefix(cls, cst) {
					t.Fatalf("linha %d: CST %q / cClassTrib %q fora do formato", i+1, cst, cls)
				}
				for k, v := range it {
					if !ehCampoReforma(k) || k == "ibs_cbs_situacao_tributaria" || k == "ibs_cbs_classificacao_tributaria" {
						continue
					}
					n, ok := v.(float64)
					if !ok || n < 0 {
						t.Fatalf("linha %d: %s = %v não é número não negativo", i+1, k, v)
					}
					limite := 2 // Decimal[13.2]: base e valores
					if strings.Contains(k, "aliquota") || strings.Contains(k, "percentual") {
						limite = 4 // Decimal[3.2-4]
					}
					if casas(n) > limite {
						t.Fatalf("linha %d: %s = %v passa de %d casas", i+1, k, n, limite)
					}
				}
				if c.grupos[i] == "INTEGRAL" || c.grupos[i] == "REDUZIDO" {
					gruposComValores++
					// IBS total = UF + município.
					if math.Abs(it["ibs_valor_total"].(float64)-it["ibs_uf_valor"].(float64)-it["ibs_mun_valor"].(float64)) > 0.001 {
						t.Fatalf("linha %d: ibs_valor_total diferente de UF + município", i+1)
					}
				}
				if c.grupos[i] == "REDUZIDO" {
					aliq, red, efet := it["cbs_aliquota"].(float64), it["cbs_percentual_reducao_aliquota"].(float64), it["cbs_aliquota_efetiva"].(float64)
					if math.Abs(aliq*(100-red)/100-efet) > 0.0001 {
						t.Fatalf("linha %d: alíquota efetiva %v ≠ %v × (1 − %v%%)", i+1, efet, aliq, red)
					}
				}
			}
			// 5. Retrato = o que foi para o payload.
			if len(retrato) != 0 && len(retrato) != len(itens) {
				t.Fatalf("retrato com %d linhas para %d itens", len(retrato), len(itens))
			}
			if (gruposComValores > 0) != (len(retrato) > 0) {
				t.Fatalf("retrato e payload divergem: %d grupos com valores, retrato %d", gruposComValores, len(retrato))
			}
		})
	}
}

// A base do IBS/CBS sai do valor da operação rateado (com frete e seguro, menos
// desconto) sem ICMS, PIS e COFINS — e o rateio é o MESMO do modelo atual.
func TestBaseIBSCBSUsaORateioDoModeloAtual(t *testing.T) {
	nota, itens := notaCompleta()
	nota.DataEmissao = time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	rt, err := resolverReformaDaNota(context.Background(), &reformaFake{tabela: tabelaTeste()}, nota, itens, configBase())
	if err != nil {
		t.Fatal(err)
	}
	p, _ := montarPayloadNFe(nota, itens, configBase(), PlanoDaNota{}, rt)
	for i, it := range p.Items {
		op := it.ValorBruto + it.ValorFrete + it.ValorSeguro - it.ValorDesconto
		esperado := math.Round((op-it.ValorICMS-it.ValorPIS-it.ValorCOFINS)*100) / 100
		if math.Abs(it.IBSCBS.Base-esperado) > 0.005 {
			t.Fatalf("linha %d: base %v, esperado %v", i+1, it.IBSCBS.Base, esperado)
		}
	}
}

// CST cujo grupo a emissão não monta (diferimento, monofásico…) é recusado antes
// da SEFAZ, com a linha e a origem.
func TestCSTNaoSuportadoERecusado(t *testing.T) {
	tab := tabelaTeste()
	dif := oficial("510001", true, "0")
	tab["510001"] = dif
	repo := &reformaFake{tabela: tab, candidatos: map[int64]reforma.Candidatos{21: {DaClassificacao: "510001"}}}
	nota, itens := notaCompleta()
	nota.DataEmissao = time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	_, err := resolverReformaDaNota(context.Background(), repo, nota, itens, configBase())
	if err == nil || !strings.Contains(err.Error(), "linha 1") || !strings.Contains(err.Error(), "CST 510") {
		t.Fatalf("CST 510 deveria ser recusado: %v", err)
	}
	for _, cst := range []string{"000", "010", "011", "200", "400", "410", "550"} {
		if !reforma.CSTSuportadoNaEmissao(cst) {
			t.Errorf("CST %s deveria ser suportado", cst)
		}
	}
	for _, cst := range []string{"220", "221", "222", "510", "515", "620", "800", "810", "811", "820", "830"} {
		if reforma.CSTSuportadoNaEmissao(cst) {
			t.Errorf("CST %s não é montado pela emissão", cst)
		}
	}
}

func TestModoSomenteAtualExigeMotivo(t *testing.T) {
	repo := &modoGravado{}
	uc := &ReformaTributariaUseCase{Repo: repo, Auth: authReforma{}}
	if err := uc.SalvarModo(context.Background(), SalvarModoReformaDTO{Modo: "SOMENTE_ATUAL"}); err == nil {
		t.Fatal("desligar o grupo sem motivo deveria ser recusado")
	}
	motivo := "rejeição 1115 da SEFAZ no grupo IBS/CBS"
	if err := uc.SalvarModo(context.Background(), SalvarModoReformaDTO{Modo: "somente_atual", Motivo: &motivo}); err != nil || repo.p.Modo != reforma.ModoSomenteAtual {
		t.Fatalf("modo gravado: %+v %v", repo.p, err)
	}
	if err := uc.SalvarModo(context.Background(), SalvarModoReformaDTO{Modo: "AUTOMATICO"}); err != nil || repo.p.Modo != reforma.ModoAutomatico {
		t.Fatalf("voltar ao automático não exige motivo: %+v %v", repo.p, err)
	}
	if err := uc.SalvarModo(context.Background(), SalvarModoReformaDTO{Modo: "X"}); err == nil {
		t.Fatal("modo inválido")
	}
}

type modoGravado struct {
	reformaFake
	p reforma.Parametros
}

func (m *modoGravado) SalvarParametros(_ context.Context, p reforma.Parametros, _ uuid.UUID) error {
	m.p = p
	return nil
}
