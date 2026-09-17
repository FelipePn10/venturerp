package margin_uc

import (
	"context"
	"fmt"
	"math"
	"time"

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
	Ano               int     `json:"ano"`
	Mes               int     `json:"mes"`
	Quantidade        float64 `json:"quantidade"`
	PrecoUnitario     float64 `json:"preco_unitario"`
	CustoUnitario     float64 `json:"custo_unitario"`
	CustoTransfUnit   float64 `json:"custo_transformacao_unitario"`
	IPIPct            float64 `json:"ipi_pct"`
	ICMSPct           float64 `json:"icms_pct"`
	PISCOFINSPct      float64 `json:"pis_cofins_pct"`
	ComissaoPct       float64 `json:"comissao_pct"`
	OutrosValor       float64 `json:"outros_valor"`
	MargemDesejadaPct float64 `json:"margem_desejada_pct"`
}

// SimulacaoSaida devolve a cascata e o preço que atingiria a margem desejada.
type SimulacaoSaida struct {
	entity.Resultado
	PrecoUnitario     float64  `json:"preco_unitario"`
	CicloDeCaixaDias  int      `json:"ciclo_caixa_dias"`
	TaxaFinanceiraPct float64  `json:"taxa_financeira_real_pct"`
	MargemDesejadaPct float64  `json:"margem_desejada_pct"`
	PrecoMinimo       *float64 `json:"preco_minimo,omitempty"`
	PrecoMinimoNota   string   `json:"preco_minimo_nota,omitempty"`
}

// vendaDe monta a venda a partir da entrada. Os impostos entram em valor, como
// a apuração espera — é a conversão que mantém as duas contas iguais.
func vendaDe(in SimulacaoEntrada, preco float64) entity.Venda {
	bruto := preco * in.Quantidade
	ipi := bruto * in.IPIPct / 100
	mercadoria := bruto
	return entity.Venda{
		Quantidade:         in.Quantidade,
		ValorLiquido:       preco,
		IPI:                ipi,
		ICMS:               mercadoria * in.ICMSPct / 100,
		PIS:                mercadoria * in.PISCOFINSPct / 100,
		COFINS:             0,
		CustoMateriaPrima:  in.CustoUnitario * in.Quantidade,
		CustoTransformacao: in.CustoTransfUnit * in.Quantidade,
		ComissaoValor:      mercadoria * in.ComissaoPct / 100,
		OutrosValor:        in.OutrosValor,
	}
}

// Simular responde a cascata para o preço informado e, quando pedida uma margem
// alvo, o preço mínimo que a atinge.
func (uc *UseCase) Simular(ctx context.Context, in SimulacaoEntrada) (*SimulacaoSaida, error) {
	if in.Quantidade <= 0 {
		return nil, errorsuc.NewValidationError("informe uma quantidade maior que zero")
	}
	if in.PrecoUnitario <= 0 {
		return nil, errorsuc.NewValidationError("informe o preço unitário de venda")
	}
	for nome, v := range map[string]float64{
		"IPI": in.IPIPct, "ICMS": in.ICMSPct, "PIS/COFINS": in.PISCOFINSPct, "comissão": in.ComissaoPct,
	} {
		if v < 0 || v > 100 {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o percentual de %s deve ficar entre 0 e 100", nome))
		}
	}
	if in.CustoUnitario < 0 || in.CustoTransfUnit < 0 {
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
		Resultado:         entity.Calcular(vendaDe(in, in.PrecoUnitario), *p),
		PrecoUnitario:     in.PrecoUnitario,
		CicloDeCaixaDias:  p.CicloDeCaixaDias(),
		TaxaFinanceiraPct: p.TaxaFinanceiraRealPct(),
		MargemDesejadaPct: in.MargemDesejadaPct,
	}

	if in.MargemDesejadaPct > 0 {
		preco, nota := precoParaMargem(in, *p)
		out.PrecoMinimo, out.PrecoMinimoNota = preco, nota
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
func precoParaMargem(in SimulacaoEntrada, p entity.Parametros) (*float64, string) {
	alvo := in.MargemDesejadaPct
	margemEm := func(preco float64) float64 {
		return entity.Calcular(vendaDe(in, preco), p).MargemPct
	}

	baixo := 0.01
	alto := math.Max(in.PrecoUnitario, (in.CustoUnitario+in.CustoTransfUnit)) * 4
	if alto < 1 {
		alto = 1000
	}
	// A margem cresce com o preço; se nem no teto o alvo é atingido, não há
	// preço que resolva com este custo e esta carga tributária.
	for i := 0; i < 40 && margemEm(alto) < alvo; i++ {
		alto *= 2
		if math.IsInf(alto, 0) || alto > 1e12 {
			return nil, "com este custo e esta carga tributária não há preço que atinja a margem pedida"
		}
	}
	if margemEm(baixo) >= alvo {
		v := baixo
		return &v, "qualquer preço acima do mínimo já atinge a margem pedida"
	}
	for i := 0; i < 60; i++ {
		meio := (baixo + alto) / 2
		if margemEm(meio) < alvo {
			baixo = meio
		} else {
			alto = meio
		}
	}
	v := math.Ceil(alto*100) / 100 // arredonda para cima: abaixo disso a margem não fecha
	return &v, ""
}
