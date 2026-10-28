package fiscal_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type reformaFake struct {
	repository.ReformaTributariaRepository
	aliquotas  map[int]*reforma.Aliquotas
	candidatos map[int64]reforma.Candidatos
	tabela     map[string]reforma.ClassificacaoOficial
	modo       reforma.Modo
}

func (f *reformaFake) Parametros(context.Context) (reforma.Parametros, error) {
	if f.modo == "" {
		return reforma.Parametros{Modo: reforma.ModoAutomatico}, nil
	}
	return reforma.Parametros{Modo: f.modo}, nil
}

func (f *reformaFake) AliquotasDoAno(_ context.Context, ano int) (*reforma.Aliquotas, error) {
	return f.aliquotas[ano], nil
}
func (f *reformaFake) CandidatosDaNota(context.Context, int64) (map[int64]reforma.Candidatos, error) {
	return f.candidatos, nil
}
func (f *reformaFake) BuscarClassificacoes(_ context.Context, codigos []string) (map[string]reforma.ClassificacaoOficial, error) {
	out := map[string]reforma.ClassificacaoOficial{}
	for _, c := range codigos {
		if v, ok := f.tabela[c]; ok {
			out[c] = v
		}
	}
	return out, nil
}

func oficial(codigo string, exige bool, red string) reforma.ClassificacaoOficial {
	r := decimal.RequireFromString(red)
	return reforma.ClassificacaoOficial{
		Classificacao: reforma.Classificacao{ClassTrib: codigo, CST: codigo[:3], ExigeTributacao: exige, PercReducaoIBS: r, PercReducaoCBS: r},
		AceitaNFe:     true, VigenciaInicio: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), Descricao: "teste " + codigo,
	}
}

func tabelaTeste() map[string]reforma.ClassificacaoOficial {
	return map[string]reforma.ClassificacaoOficial{
		"000001": oficial("000001", true, "0"), "200030": oficial("200030", true, "60"), "410001": oficial("410001", false, "0"),
	}
}

func duasLinhas() []*fiscalentity.FiscalExitItem {
	return []*fiscalentity.FiscalExitItem{
		{ID: 11, Sequence: 1, Cfop: "5101", Quantity: 1, UnitPrice: 1000, TotalPrice: 1000, Description: strp("A")},
		{ID: 12, Sequence: 2, Cfop: "5910", Quantity: 1, UnitPrice: 500, TotalPrice: 500, Description: strp("B")},
	}
}

func TestReformaDaNotaAplicaPrecedenciaEReducao(t *testing.T) {
	repo := &reformaFake{
		candidatos: map[int64]reforma.Candidatos{11: {DaClassificacao: "200030"}, 12: {DoCFOP: "410001", DaClassificacao: "200030"}},
		tabela:     tabelaTeste(),
	}
	nota, itens := notaBase(), duasLinhas()
	rt, err := resolverReformaDaNota(context.Background(), repo, nota, itens, configBase())
	if err != nil {
		t.Fatal(err)
	}
	if rt.PorItem[11].Origem != reforma.OrigemClassificacao || rt.PorItem[12].Origem != reforma.OrigemCFOP {
		t.Fatalf("origens: %+v", rt.PorItem)
	}
	p, retrato := montarPayloadNFe(nota, itens, configBase(), PlanoDaNota{}, rt)
	g1, g2 := p.Items[0].IBSCBS, p.Items[1].IBSCBS
	// 2026 pela lei: CBS 0,9% com 60% de redução = 0,36%.
	if g1.CST != "200" || !g1.ComReducao || g1.EfetCBS != 0.36 || g1.RedCBS != 60 {
		t.Fatalf("linha 1 reduzida: %+v", g1)
	}
	if g2.CST != "410" || !g2.SemValores {
		t.Fatalf("linha 2 (bonificação pelo CFOP) não leva valores: %+v", g2)
	}
	ibs, cbs := totaisDoRetrato(retrato)
	if len(retrato) != 2 || !cbs.Equal(retrato[0].Grupo.CBS.Valor) || !ibs.Equal(retrato[0].Grupo.IBS) {
		t.Fatalf("retrato: %+v ibs %s cbs %s", retrato, ibs, cbs)
	}
}

func TestReformaDaNotaRecusaOQueASEFAZRejeitaria(t *testing.T) {
	nota, itens := notaBase(), duasLinhas()
	cfg := configBase()

	repo := &reformaFake{candidatos: map[int64]reforma.Candidatos{11: {DoItem: "999999"}}, tabela: tabelaTeste()}
	if _, err := resolverReformaDaNota(context.Background(), repo, nota, itens, cfg); err == nil || !strings.Contains(err.Error(), "999999") {
		t.Fatalf("cClassTrib fora da tabela deveria recusar: %v", err)
	}

	tab := tabelaTeste()
	fora := tab["200030"]
	fora.AceitaNFe = false
	tab["200030"] = fora
	repo = &reformaFake{candidatos: map[int64]reforma.Candidatos{11: {DaClassificacao: "200030"}}, tabela: tab}
	if _, err := resolverReformaDaNota(context.Background(), repo, nota, itens, cfg); err == nil || !strings.Contains(err.Error(), "não é admitido na NF-e") {
		t.Fatalf("cClassTrib não admitido na NF-e deveria recusar: %v", err)
	}

	nota2027 := notaBase()
	nota2027.DataEmissao = time.Date(2027, 1, 4, 0, 0, 0, 0, time.UTC)
	repo = &reformaFake{tabela: tabelaTeste()}
	if _, err := resolverReformaDaNota(context.Background(), repo, nota2027, itens, cfg); err == nil || !strings.Contains(err.Error(), "CBS de 2027") {
		t.Fatalf("2027 sem CBS cadastrada deveria recusar: %v", err)
	}
	repo.aliquotas = map[int]*reforma.Aliquotas{2027: {IBSUF: decimal.RequireFromString("0.05"), IBSMun: decimal.RequireFromString("0.05"), CBS: decimal.RequireFromString("8.7")}}
	rt, err := resolverReformaDaNota(context.Background(), repo, nota2027, itens, cfg)
	if err != nil || rt.Aliquotas.Fonte != reforma.FonteEmpresa || !rt.Aliquotas.CBS.Equal(decimal.RequireFromString("8.7")) {
		t.Fatalf("2027 com cadastro: %+v %v", rt.Aliquotas, err)
	}
}

func TestReformaNaoSeAplicaAoSimples(t *testing.T) {
	cfg := configBase()
	cfg.RegimeTributario = "1"
	rt, err := resolverReformaDaNota(context.Background(), &reformaFake{}, notaBase(), duasLinhas(), cfg)
	if err != nil || rt.Aplica {
		t.Fatalf("Simples não leva o grupo: %+v %v", rt, err)
	}
	p, retrato := montarPayloadNFe(notaBase(), duasLinhas(), cfg, PlanoDaNota{}, rt)
	if p.Items[0].IBSCBS != nil || retrato != nil {
		t.Fatal("Simples não leva o grupo IBS/CBS")
	}
}

func TestAvisoPISCOFINSEm2027(t *testing.T) {
	nota := notaBase()
	itens := duasLinhas()
	itens[0].ValorPIS = 16.5
	if avisoPISCOFINSExtintos(nota, itens) != nil {
		t.Fatal("2026 não avisa")
	}
	nota.DataEmissao = time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	if a := avisoPISCOFINSExtintos(nota, itens); len(a) != 1 || a[0].Nivel != NivelAtencao {
		t.Fatalf("2027 com PIS deveria avisar: %+v", a)
	}
}
