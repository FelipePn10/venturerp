package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

func diasPtr(v int) *int { return &v }

func TestResolverPoliticasHerancaCampoACampo(t *testing.T) {
	empresa := entity.PoliticaEfetiva{Estrategia: entity.EstrategiaConsumo, ConsumoDiasAntes: 30, HorizonteFirmeDias: 0, TempoSegurancaDias: 0,
		Origem: entity.OrigemPolitica{Estrategia: entity.OrigemEmpresa, Janela: entity.OrigemEmpresa, HorizonteFirme: entity.OrigemEmpresa, TempoSeguranca: entity.OrigemEmpresa}}
	sob := entity.EstrategiaSobEncomenda
	cons := entity.EstrategiaConsumo
	raiz := &entity.PoliticaPlanejamento{ID: 1, ClassificationLevel: 1, ClassificationCode: "10", Estrategia: &sob, HorizonteFirmeDias: diasPtr(5)}
	folha := &entity.PoliticaPlanejamento{ID: 2, ClassificationLevel: 2, ClassificationCode: "10.100", HorizonteFirmeDias: diasPtr(10)}
	doItem := map[int64]*entity.PoliticaPlanejamento{7: {ID: 3, Estrategia: &cons, TempoSegurancaDias: diasPtr(2)}}
	porClass := map[int64][]*entity.PoliticaPlanejamento{7: {folha, raiz}, 8: {raiz}}
	got := entity.ResolverPoliticas(empresa, doItem, porClass)

	p7 := got[7]
	// Estratégia do item vence a da raiz; horizonte da folha vence o da raiz;
	// janela fica da empresa; segurança vem do item.
	if p7.Estrategia != entity.EstrategiaConsumo || p7.Origem.Estrategia != entity.OrigemItem ||
		p7.HorizonteFirmeDias != 10 || p7.Origem.HorizonteFirme != "CLASSIFICACAO:10.100" ||
		p7.ConsumoDiasAntes != 30 || p7.Origem.Janela != entity.OrigemEmpresa || p7.TempoSegurancaDias != 2 {
		t.Fatalf("item 7: %+v", p7)
	}
	p8 := got[8]
	if !p8.SobEncomenda() || p8.HorizonteFirmeDias != 5 || p8.Origem.Estrategia != "CLASSIFICACAO:10" {
		t.Fatalf("item 8 herda a raiz: %+v", p8)
	}
	if _, ok := got[9]; ok {
		t.Fatal("item sem regra fica fora do mapa (segue a empresa)")
	}
	params := entity.DefaultTypedPlanningParams()
	params.Politicas = got
	if params.Politica(9).ConsumoDiasAntes != 30 || params.Politica(9).Origem.Janela != entity.OrigemEmpresa {
		t.Fatalf("item sem regra: %+v", params.Politica(9))
	}
}

func TestResolverPoliticasMesmoNivelMenorIDVence(t *testing.T) {
	a := &entity.PoliticaPlanejamento{ID: 4, ClassificationLevel: 2, ClassificationCode: "A", HorizonteFirmeDias: diasPtr(3)}
	b := &entity.PoliticaPlanejamento{ID: 9, ClassificationLevel: 2, ClassificationCode: "B", HorizonteFirmeDias: diasPtr(8)}
	for _, ordem := range [][]*entity.PoliticaPlanejamento{{a, b}, {b, a}} {
		got := entity.ResolverPoliticas(entity.PoliticaEfetiva{}, nil, map[int64][]*entity.PoliticaPlanejamento{1: ordem})
		if got[1].HorizonteFirmeDias != 3 {
			t.Fatalf("empate de nível: a de menor id vence, em qualquer ordem: %+v", got[1])
		}
	}
}

func TestValidarPolitica(t *testing.T) {
	item := int64(1)
	classe := int64(2)
	x := "X"
	casos := map[string]entity.PoliticaPlanejamento{
		"nenhum escopo":  {HorizonteFirmeDias: diasPtr(1)},
		"dois escopos":   {ItemCode: &item, ClassificationID: &classe, HorizonteFirmeDias: diasPtr(1)},
		"estratégia":     {ItemCode: &item, Estrategia: &x},
		"dias negativos": {ItemCode: &item, ConsumoDiasAntes: diasPtr(-1)},
		"dias demais":    {ItemCode: &item, TempoSegurancaDias: diasPtr(366)},
		"sem campo":      {ItemCode: &item},
	}
	for nome, p := range casos {
		if p.Validar() == "" {
			t.Errorf("%s: deveria recusar", nome)
		}
	}
	ok := entity.PoliticaPlanejamento{ItemCode: &item, ConsumoDiasDepois: diasPtr(0)}
	if msg := ok.Validar(); msg != "" {
		t.Fatalf("zero é valor válido: %s", msg)
	}
}

func TestAplicarHorizonteFirme(t *testing.T) {
	hoje := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	ini, rec := hoje.AddDate(0, 0, 2), hoje.AddDate(0, 0, 6)
	i, r, pedido := aplicarHorizonteFirme(ini, rec, 10, hoje)
	if pedido == nil || !pedido.Equal(ini) || !i.Equal(hoje.AddDate(0, 0, 10)) || !r.Equal(hoje.AddDate(0, 0, 14)) {
		t.Fatalf("começa no fim do horizonte e mantém o prazo: %v %v %v", i, r, pedido)
	}
	if i, r, pedido = aplicarHorizonteFirme(ini, rec, 0, hoje); pedido != nil || !i.Equal(ini) || !r.Equal(rec) {
		t.Fatal("horizonte 0 = desligado")
	}
	if _, _, pedido = aplicarHorizonteFirme(hoje.AddDate(0, 0, 10), rec, 10, hoje); pedido != nil {
		t.Fatal("começar no último dia do horizonte (o primeiro fora dele) não mexe")
	}
}

func TestNettingComTempoDeSegurancaEHorizonte(t *testing.T) {
	hoje := today()
	service := &MRPServiceImpl{}
	params := entity.DefaultTypedPlanningParams()
	params.Politicas = map[int64]entity.PoliticaEfetiva{100: {Estrategia: entity.EstrategiaConsumo, TempoSegurancaDias: 3}}
	lead := []*entity.ConfiguredItemRule{{ItemCode: 100, FieldName: "lead_time", RuleValue: "5", IsActive: true}}
	need := hoje.AddDate(0, 0, 40)
	input := &entity.MRPInput{PlanCode: 1, ItemCode: 100, Quantity: decimal.NewFromInt(5), NeedDate: need}
	out, err := service.calcNetReqFast(context.Background(), input, map[int64]*entity.StockSnapshot{}, map[int64][]*entity.ConfiguredItemRule{100: lead}, nil, map[int64]*cachedItemMRP{100: {}}, params)
	if err != nil {
		t.Fatal(err)
	}
	s := out.PlannedOrders[0]
	if !s.NeedDate.Equal(need.AddDate(0, 0, -3)) || !s.StartDate.Equal(need.AddDate(0, 0, -8)) || s.InicioPedido != nil {
		t.Fatalf("segurança de 3 dias: chega em %v, começa em %v", s.NeedDate, s.StartDate)
	}

	// Horizonte de 60 dias: a sugestão começaria em +32, vai para +60.
	params.Politicas[100] = entity.PoliticaEfetiva{Estrategia: entity.EstrategiaConsumo, TempoSegurancaDias: 3, HorizonteFirmeDias: 60}
	out, _ = service.calcNetReqFast(context.Background(), input, map[int64]*entity.StockSnapshot{}, map[int64][]*entity.ConfiguredItemRule{100: lead}, nil, map[int64]*cachedItemMRP{100: {}}, params)
	s = out.PlannedOrders[0]
	if s.InicioPedido == nil || !s.InicioPedido.Equal(need.AddDate(0, 0, -8)) || !s.StartDate.Equal(hoje.AddDate(0, 0, 60)) || !s.NeedDate.Equal(hoje.AddDate(0, 0, 65)) {
		t.Fatalf("horizonte: começa %v chega %v (pedido %v)", s.StartDate, s.NeedDate, s.InicioPedido)
	}
	s.Code = 9
	b := newExceptionBasis(nil, nil)
	b.addFirmFence(s)
	msgs := b.firmFenceMessages()
	if len(msgs) != 1 || msgs[0].MessageType != entity.ExceptionFirmFence || msgs[0].SourceCode == nil || *msgs[0].SourceCode != 9 {
		t.Fatalf("exceção do horizonte: %+v", msgs)
	}
}

func TestConsumoComPoliticaPorItem(t *testing.T) {
	hoje := today()
	params := entity.DefaultTypedPlanningParams()
	// Item 7: janela só para a frente (0 antes, 10 depois). Item 8: sob encomenda.
	params.Politicas = map[int64]entity.PoliticaEfetiva{
		7: {Estrategia: entity.EstrategiaConsumo, ConsumoDiasAntes: 0, ConsumoDiasDepois: 10},
		8: {Estrategia: entity.EstrategiaSobEncomenda, ConsumoDiasAntes: 30},
	}
	demands := []*inddemandentity.IndependentDemand{
		{CodeDemand: 1, ItemCode: 7, Quantity: decimal.NewFromInt(10), DemandDate: hoje.AddDate(0, 0, 5), Motivo: inddemandentity.MotivoEstoque, Situacao: inddemandentity.SituacaoAberta},
		{CodeDemand: 2, ItemCode: 8, Quantity: decimal.NewFromInt(10), DemandDate: hoje.AddDate(0, 0, 5), Motivo: inddemandentity.MotivoEstoque, Situacao: inddemandentity.SituacaoAberta},
		{CodeDemand: 3, ItemCode: 8, Quantity: decimal.NewFromInt(2), DemandDate: hoje.AddDate(0, 0, 5), Motivo: inddemandentity.MotivoFeira, Situacao: inddemandentity.SituacaoAberta},
	}
	previsao := []*entity.MRPInput{{ItemCode: 8, Quantity: decimal.NewFromInt(50), NeedDate: hoje.AddDate(0, 0, 7), DemandType: "FORECAST"}}
	linha := int64(1)
	s := &MRPServiceImpl{Vendas: vendasFixas{{Chave: inddemandentity.ChaveItem(7, ""), Data: hoje, Quantidade: decimal.NewFromInt(4), Linha: &linha}}}
	d, p, err := s.consumirPelaVenda(context.Background(), params, demands, previsao, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Item 7: a venda de hoje consome para a frente (demanda em +5): planeja 6.
	// Item 8 sob encomenda: a de estoque e a previsão saem; a de feira fica.
	if len(d) != 2 || d[0].CodeDemand != 1 || !d[0].Quantity.Equal(decimal.NewFromInt(6)) || d[1].CodeDemand != 3 {
		t.Fatalf("demandas: %+v", d)
	}
	if len(p) != 0 {
		t.Fatalf("previsão de item sob encomenda não entra: %+v", p)
	}
}
