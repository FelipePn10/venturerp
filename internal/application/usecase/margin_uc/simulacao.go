package margin_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/margin/entity"
)

// SimulacaoEntrada é a pergunta "e se eu vender assim?".
//
// Os percentuais de imposto e comissão são informados porque variam por
// operação — a mesma peça sai com ICMS diferente dentro e fora do estado. O
// resto (incidência administrativa, frete médio, taxa financeira, ciclo de
// caixa) vem dos parâmetros do mês, os mesmos que a apuração usa.
type SimulacaoEntrada struct {
	Ano               int             `json:"ano"`
	Mes               int             `json:"mes"`
	Quantidade        decimal.Decimal `json:"quantidade"`
	PrecoUnitario     decimal.Decimal `json:"preco_unitario"`
	CustoUnitario     decimal.Decimal `json:"custo_unitario"`
	CustoTransfUnit   decimal.Decimal `json:"custo_transformacao_unitario"`
	IPIPct            decimal.Decimal `json:"ipi_pct"`
	ICMSPct           decimal.Decimal `json:"icms_pct"`
	PISCOFINSPct      decimal.Decimal `json:"pis_cofins_pct"`
	ComissaoPct       decimal.Decimal `json:"comissao_pct"`
	OutrosValor       decimal.Decimal `json:"outros_valor"`
	MargemDesejadaPct decimal.Decimal `json:"margem_desejada_pct"`
}

// ResultadoJSON é a cascata na resposta: números, como sempre foram — a conta
// é feita em decimal (entity.Resultado) e só a saída vira float.
type ResultadoJSON struct {
	FaturamentoBruto      float64 `json:"faturamento_bruto"`
	IPI                   float64 `json:"ipi"`
	FaturamentoMercadoria float64 `json:"faturamento_mercadoria"`
	ICMS                  float64 `json:"icms"`
	PISCOFINS             float64 `json:"pis_cofins"`
	CustoMateriaPrima     float64 `json:"custo_materia_prima"`
	CustoTransformacao    float64 `json:"custo_transformacao"`
	LucroBruto            float64 `json:"lucro_bruto"`
	DespesaAdministrativa float64 `json:"despesa_administrativa"`
	Comissao              float64 `json:"comissao"`
	Frete                 float64 `json:"frete"`
	Outros                float64 `json:"outros"`
	DespesaFinanceira     float64 `json:"despesa_financeira"`
	ProvisaoIR            float64 `json:"provisao_ir"`
	Margem                float64 `json:"margem"`
	MargemPct             float64 `json:"margem_pct"`
}

func resultadoJSON(r entity.Resultado) ResultadoJSON {
	return ResultadoJSON{
		FaturamentoBruto: r.FaturamentoBruto.InexactFloat64(), IPI: r.IPI.InexactFloat64(),
		FaturamentoMercadoria: r.FaturamentoMercadoria.InexactFloat64(), ICMS: r.ICMS.InexactFloat64(),
		PISCOFINS: r.PISCOFINS.InexactFloat64(), CustoMateriaPrima: r.CustoMateriaPrima.InexactFloat64(),
		CustoTransformacao: r.CustoTransformacao.InexactFloat64(), LucroBruto: r.LucroBruto.InexactFloat64(),
		DespesaAdministrativa: r.DespesaAdministrativa.InexactFloat64(), Comissao: r.Comissao.InexactFloat64(),
		Frete: r.Frete.InexactFloat64(), Outros: r.Outros.InexactFloat64(),
		DespesaFinanceira: r.DespesaFinanceira.InexactFloat64(), ProvisaoIR: r.ProvisaoIR.InexactFloat64(),
		Margem: r.Margem.InexactFloat64(), MargemPct: r.MargemPct.InexactFloat64(),
	}
}

// SimulacaoSaida devolve a cascata e o preço que atingiria a margem desejada.
type SimulacaoSaida struct {
	ResultadoJSON
	PrecoUnitario     float64  `json:"preco_unitario"`
	CicloDeCaixaDias  int      `json:"ciclo_caixa_dias"`
	TaxaFinanceiraPct float64  `json:"taxa_financeira_real_pct"`
	MargemDesejadaPct float64  `json:"margem_desejada_pct"`
	PrecoMinimo       *float64 `json:"preco_minimo,omitempty"`
	PrecoMinimoNota   string   `json:"preco_minimo_nota,omitempty"`
}

var cem = decimal.NewFromInt(100)

// vendaDe monta a venda a partir da entrada. Os impostos entram em valor, como
// a apuração espera — é a conversão que mantém as duas contas iguais.
func vendaDe(in SimulacaoEntrada, preco decimal.Decimal) entity.Venda {
	bruto := preco.Mul(in.Quantidade)
	ipi := bruto.Mul(in.IPIPct).Div(cem)
	mercadoria := bruto
	return entity.Venda{
		Quantidade:         in.Quantidade,
		ValorLiquido:       preco,
		IPI:                ipi,
		ICMS:               mercadoria.Mul(in.ICMSPct).Div(cem),
		PIS:                mercadoria.Mul(in.PISCOFINSPct).Div(cem),
		COFINS:             decimal.Zero,
		CustoMateriaPrima:  in.CustoUnitario.Mul(in.Quantidade),
		CustoTransformacao: in.CustoTransfUnit.Mul(in.Quantidade),
		ComissaoValor:      mercadoria.Mul(in.ComissaoPct).Div(cem),
		OutrosValor:        in.OutrosValor,
	}
}

// Simular responde a cascata para o preço informado e, quando pedida uma margem
// alvo, o preço mínimo que a atinge.
func (uc *UseCase) Simular(ctx context.Context, in SimulacaoEntrada) (*SimulacaoSaida, error) {
	if !in.Quantidade.IsPositive() {
		return nil, errorsuc.NewValidationError("informe uma quantidade maior que zero")
	}
	if !in.PrecoUnitario.IsPositive() {
		return nil, errorsuc.NewValidationError("informe o preço unitário de venda")
	}
	for nome, v := range map[string]decimal.Decimal{
		"IPI": in.IPIPct, "ICMS": in.ICMSPct, "PIS/COFINS": in.PISCOFINSPct, "comissão": in.ComissaoPct,
	} {
		if v.IsNegative() || v.GreaterThan(cem) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o percentual de %s deve ficar entre 0 e 100", nome))
		}
	}
	if in.CustoUnitario.IsNegative() || in.CustoTransfUnit.IsNegative() {
		return nil, errorsuc.NewValidationError("o custo não pode ser negativo")
	}

	ano, mes := in.Ano, in.Mes
	if ano == 0 || mes == 0 {
		agora := time.Now()
		ano, mes = agora.Year(), int(agora.Month())
	}
	p, err := uc.Repo.GetParametros(ctx, ano, mes)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"os parâmetros de margem de %02d/%d não estão cadastrados; cadastre-os na aba Parâmetros do mês antes de simular", mes, ano))
	}

	out := &SimulacaoSaida{
		ResultadoJSON:     resultadoJSON(entity.Calcular(vendaDe(in, in.PrecoUnitario), *p)),
		PrecoUnitario:     in.PrecoUnitario.InexactFloat64(),
		CicloDeCaixaDias:  p.CicloDeCaixaDias(),
		TaxaFinanceiraPct: p.TaxaFinanceiraRealPct().InexactFloat64(),
		MargemDesejadaPct: in.MargemDesejadaPct.InexactFloat64(),
	}

	if in.MargemDesejadaPct.IsPositive() {
		preco, nota := precoParaMargem(in, *p)
		if preco != nil {
			f := preco.InexactFloat64()
			out.PrecoMinimo = &f
		}
		out.PrecoMinimoNota = nota
	}
	return out, nil
}

// precoParaMargem procura, por bisseção, o preço que atinge a margem desejada.
//
// Resolver na álgebra seria mais rápido e mais frágil: a provisão de IR só
// incide sobre lucro positivo, então a função tem um joelho, e qualquer mudança
// futura na cascata deixaria a fórmula fechada em silêncio desatualizada. A
// bisseção chama a MESMA `entity.Calcular` da apuração — o preço devolvido
// sempre confere com a conta que a tela mostra.
func precoParaMargem(in SimulacaoEntrada, p entity.Parametros) (*decimal.Decimal, string) {
	alvo := in.MargemDesejadaPct
	margemEm := func(preco decimal.Decimal) decimal.Decimal {
		return entity.Calcular(vendaDe(in, preco), p).MargemPct
	}

	dois := decimal.NewFromInt(2)
	teto := decimal.New(1, 12)
	baixo := decimal.New(1, -2)
	alto := decimal.Max(in.PrecoUnitario, in.CustoUnitario.Add(in.CustoTransfUnit)).Mul(decimal.NewFromInt(4))
	if alto.LessThan(decimal.NewFromInt(1)) {
		alto = decimal.NewFromInt(1000)
	}
	// A margem cresce com o preço, mas SATURA: tributos, comissão, incidência
	// administrativa, frete e a despesa financeira do prazo são todos
	// proporcionais ao preço, e a provisão de IR come um terço do que sobra.
	// Recusar sem dizer qual é o teto deixa quem perguntou sem a informação que
	// foi buscar — o vendedor quer saber quanto dá, não só que não dá.
	for i := 0; i < 40 && margemEm(alto).LessThan(alvo); i++ {
		alto = alto.Mul(dois)
		if alto.GreaterThan(teto) {
			return nil, fmt.Sprintf(
				"esta margem não é alcançável a nenhum preço: com este custo, esta carga tributária e este prazo, o máximo é %s%%",
				margemEm(teto).StringFixed(2))
		}
	}
	if margemEm(baixo).GreaterThanOrEqual(alvo) {
		v := baixo
		return &v, "qualquer preço acima do mínimo já atinge a margem pedida"
	}
	for i := 0; i < 60; i++ {
		meio := baixo.Add(alto).Div(dois)
		if margemEm(meio).LessThan(alvo) {
			baixo = meio
		} else {
			alto = meio
		}
	}
	v := alto.Mul(cem).Ceil().Div(cem) // arredonda para cima: abaixo disso a margem não fecha
	return &v, ""
}
