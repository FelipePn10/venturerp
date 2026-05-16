package engine

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type TaxCalculationParams struct {
	Itens       []TaxItem
	EmitenteUF  string
	DestinoUF   string
	DestinoTipo string
	Cfop        string
}

type TaxItem struct {
	Ncm              string
	ValorUnitario    float64
	Quantidade       float64
	ValorFrete       float64
	ValorDesconto    float64
	OrigemMercadoria string
}

type TaxCalculationResult struct {
	Itens   []TaxItemResult
	Totais  TaxTotals
	Cenario string
}

type TaxItemResult struct {
	BaseICMS          float64
	AliquotaICMS      float64
	ValorICMS         float64
	ValorICMSDiferido float64
	CSTICMS           string
	BaseIPI           float64
	AliquotaIPI       float64
	ValorIPI          float64
	CSTIPI            string
	BasePIS           float64
	ValorPIS          float64
	CSTPIS            string
	BaseCOFINS        float64
	ValorCOFINS       float64
	CSTCOFINS         string
}

type TaxTotals struct {
	BaseICMS       float64
	ValorICMS      float64
	BaseIPI        float64
	ValorIPI       float64
	BasePIS        float64
	ValorPIS       float64
	BaseCOFINS     float64
	ValorCOFINS    float64
	DifalValorICMS float64
}

type TaxScenarioConfig struct {
	AliqICMS   float64
	DifICMSPct float64
	CstICMS    string
	CalcDifal  bool
}

func CalcularImpostos(
	params TaxCalculationParams,
	ncmTable map[string]*NcmTaxConfig,
	interstateTable map[string]float64,
	internalTable map[string]ICMSInternalConfig,
	scenario TaxScenarioConfig,
	fiscalConfig FiscalConfig,
) (*TaxCalculationResult, error) {
	result := &TaxCalculationResult{
		Itens: make([]TaxItemResult, 0, len(params.Itens)),
	}

	isInterno := params.EmitenteUF == params.DestinoUF

	if isInterno {
		if params.DestinoTipo == "contribuinte" {
			result.Cenario = "INTERNA_CONTRIBUINTE"
		} else {
			result.Cenario = "INTERNA_NAO_CONTRIBUINTE"
		}
	} else {
		result.Cenario = "INTERESTADUAL"
	}

	for _, item := range params.Itens {
		itemResult := calculateItemTax(item, params, ncmTable, interstateTable, internalTable, scenario, fiscalConfig, isInterno)
		result.Itens = append(result.Itens, itemResult)

		result.Totais.BaseICMS += itemResult.BaseICMS
		result.Totais.ValorICMS += itemResult.ValorICMS
		result.Totais.BaseIPI += itemResult.BaseIPI
		result.Totais.ValorIPI += itemResult.ValorIPI
		result.Totais.BasePIS += itemResult.BasePIS
		result.Totais.ValorPIS += itemResult.ValorPIS
		result.Totais.BaseCOFINS += itemResult.BaseCOFINS
		result.Totais.ValorCOFINS += itemResult.ValorCOFINS
		result.Totais.DifalValorICMS += (itemResult.ValorICMSDiferido)
	}

	return result, nil
}

func calculateItemTax(
	item TaxItem,
	params TaxCalculationParams,
	ncmTable map[string]*NcmTaxConfig,
	interstateTable map[string]float64,
	internalTable map[string]ICMSInternalConfig,
	scenario TaxScenarioConfig,
	fiscalConfig FiscalConfig,
	isInterno bool,
) TaxItemResult {
	r := TaxItemResult{}

	itemTotal := decimal.NewFromFloat(item.ValorUnitario).Mul(decimal.NewFromFloat(item.Quantidade))

	// IPI
	ncmCfg, hasNcm := ncmTable[item.Ncm]
	aliqIPI := decimal.NewFromFloat(0)
	cstIPI := "50"
	if hasNcm {
		aliqIPI = decimal.NewFromFloat(ncmCfg.AliqIPI)
		cstIPI = ncmCfg.CstIPI
	}
	r.AliquotaIPI, _ = aliqIPI.Float64()
	r.CSTIPI = cstIPI

	baseIPI := itemTotal
	r.BaseIPI, _ = baseIPI.Round(2).Float64()
	r.ValorIPI, _ = baseIPI.Mul(aliqIPI).Round(2).Float64()

	totalComIPI := itemTotal.Add(baseIPI.Mul(aliqIPI))

	// Frete / desconto rateio
	freteItem := decimal.Zero
	descontoItem := decimal.Zero

	if len(params.Itens) > 0 {
		totalAll := decimal.Zero
		for _, it := range params.Itens {
			totalAll = totalAll.Add(decimal.NewFromFloat(it.ValorUnitario).Mul(decimal.NewFromFloat(it.Quantidade)))
		}
		if !totalAll.IsZero() {
			proporcao := itemTotal.Div(totalAll)
			freteItem = decimal.NewFromFloat(item.ValorFrete).Mul(proporcao)
			descontoItem = decimal.NewFromFloat(item.ValorDesconto).Mul(proporcao)
		}
	}

	// PIS / COFINS
	basePisCofins := itemTotal.Add(freteItem).Sub(descontoItem)

	aliqPIS := decimal.NewFromFloat(0.0165)
	cstPIS := "01"
	aliqCOFINS := decimal.NewFromFloat(0.076)
	cstCOFINS := "01"
	if hasNcm {
		if ncmCfg.AliqPis > 0 {
			aliqPIS = decimal.NewFromFloat(ncmCfg.AliqPis)
		}
		cstPIS = ncmCfg.CstPis
		if ncmCfg.AliqCofins > 0 {
			aliqCOFINS = decimal.NewFromFloat(ncmCfg.AliqCofins)
		}
		cstCOFINS = ncmCfg.CstCofins
	}

	r.BasePIS, _ = basePisCofins.Round(2).Float64()
	r.ValorPIS, _ = basePisCofins.Mul(aliqPIS).Round(2).Float64()
	r.CSTPIS = cstPIS

	r.BaseCOFINS, _ = basePisCofins.Round(2).Float64()
	r.ValorCOFINS, _ = basePisCofins.Mul(aliqCOFINS).Round(2).Float64()
	r.CSTCOFINS = cstCOFINS

	// ICMS
	if isInterno {
		// Cenario interno PR
		aliqICMS := decimal.NewFromFloat(fiscalConfig.IcmsInternoAliquota)
		baseICMS := totalComIPI.Add(freteItem).Sub(descontoItem)

		r.AliquotaICMS, _ = aliqICMS.Float64()
		r.BaseICMS, _ = baseICMS.Round(2).Float64()
		valorICMS := baseICMS.Mul(aliqICMS)
		r.ValorICMS, _ = valorICMS.Round(2).Float64()

		if params.DestinoTipo == "contribuinte" {
			// Diferimento 38.46%
			difPct := decimal.NewFromFloat(fiscalConfig.IcmsDiferimentoPercentual).Div(decimal.NewFromInt(100))
			diferido := valorICMS.Mul(difPct)
			r.ValorICMSDiferido, _ = diferido.Round(2).Float64()
			r.CSTICMS = "51"
		} else {
			r.ValorICMSDiferido = 0
			r.CSTICMS = "00"
		}
	} else {
		// Cenario interestadual
		interstateKey := params.EmitenteUF + params.DestinoUF
		var aliqInter decimal.Decimal
		if item.OrigemMercadoria == "1" || item.OrigemMercadoria == "2" {
			aliqInter = decimal.NewFromFloat(0.04)
		} else {
			aliqInter = decimal.NewFromFloat(interstateTable[interstateKey])
		}

		baseICMS := totalComIPI.Add(freteItem).Sub(descontoItem)
		r.AliquotaICMS, _ = aliqInter.Float64()
		r.BaseICMS, _ = baseICMS.Round(2).Float64()
		r.ValorICMS, _ = baseICMS.Mul(aliqInter).Round(2).Float64()

		// DIFAL for non-contributor
		if params.DestinoTipo == "nao_contribuinte" || params.DestinoTipo == "pessoa_fisica" {
			icmsInternal := decimal.NewFromFloat(internalTable[params.DestinoUF].ICMS)
			difal := icmsInternal.Sub(aliqInter)
			if difal.IsPositive() {
				difalValor := baseICMS.Mul(difal)
				r.ValorICMSDiferido, _ = difalValor.Round(2).Float64()
			}
		}
		r.CSTICMS = "00"
	}

	roundResult(&r)
	return r
}

func roundResult(r *TaxItemResult) {
	r.BaseICMS, _ = decimal.NewFromFloat(r.BaseICMS).Round(2).Float64()
	r.ValorICMS, _ = decimal.NewFromFloat(r.ValorICMS).Round(2).Float64()
	r.ValorICMSDiferido, _ = decimal.NewFromFloat(r.ValorICMSDiferido).Round(2).Float64()
	r.BaseIPI, _ = decimal.NewFromFloat(r.BaseIPI).Round(2).Float64()
	r.ValorIPI, _ = decimal.NewFromFloat(r.ValorIPI).Round(2).Float64()
	r.BasePIS, _ = decimal.NewFromFloat(r.BasePIS).Round(2).Float64()
	r.ValorPIS, _ = decimal.NewFromFloat(r.ValorPIS).Round(2).Float64()
	r.BaseCOFINS, _ = decimal.NewFromFloat(r.BaseCOFINS).Round(2).Float64()
	r.ValorCOFINS, _ = decimal.NewFromFloat(r.ValorCOFINS).Round(2).Float64()
}

type NcmTaxConfig struct {
	AliqIPI    float64
	AliqPis    float64
	AliqCofins float64
	CstPis     string
	CstCofins  string
	CstIPI     string
}

type ICMSInternalConfig struct {
	ICMS float64
	FCP  float64
}

type FiscalConfig struct {
	UFEmpresa                string
	IcmsInternoAliquota      float64
	IcmsDiferimentoPercentual float64
}

var _ = fmt.Println
