package engine

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// InvoiceTypeFlags carries the behavior flags from an InvoiceType record into the engine.
type InvoiceTypeFlags struct {
	// CalcReducao applies ICMSReductionPct to reduce the ICMS calculation base.
	CalcReducao      bool
	ICMSReductionPct float64 // e.g. 0.30 means base reduced by 30%

	// SkipPISCOFINS — when true, PIS and COFINS are zeroed out (e.g. isento, não tributado).
	// Default false means PIS/COFINS are calculated normally.
	SkipPISCOFINS bool

	// CalcFomentar applies the FOMENTAR/PRODUZIR incentive (e.g. Goiás):
	// ICMS effectively paid = total ICMS × (1 - FomentarRetentionPct).
	CalcFomentar          bool
	FomentarRetentionPct  float64 // fraction of ICMS retained by state (e.g. 0.70 = 70%)

	// IPITransferPrice: when > 0, overrides ValorUnitario as IPI base (used for
	// transfer invoices where the IPI base comes from a sales table).
	// Set this per-item in TaxItem.IPITransferPrice instead.
}

type TaxCalculationParams struct {
	Itens       []TaxItem
	EmitenteUF  string
	DestinoUF   string
	DestinoTipo string
	Cfop        string
	Flags       InvoiceTypeFlags
}

type TaxItem struct {
	Ncm              string
	ValorUnitario    float64
	Quantidade       float64
	ValorFrete       float64
	ValorDesconto    float64
	OrigemMercadoria string
	// IPITransferPrice, when > 0, replaces ValorUnitario as the IPI base.
	// Populated by the use case when InvoiceType.IPITransferSalesTableId is set.
	IPITransferPrice float64
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
	ValorICMSDiferido float64 // cenário 2: ICMS diferido (CST 51)
	ValorDIFAL        float64 // cenário 1: DIFAL para não-contribuinte interestadual
	ValorFCP          float64 // Fundo de Combate à Pobreza (DIFAL)
	ValorFomentar     float64 // ICMS retido via benefício FOMENTAR/PRODUZIR
	CSTICMS           string
	BaseIPI           float64
	AliquotaIPI       float64
	ValorIPI          float64
	CSTIPI            string
	BasePIS           float64
	AliquotaPIS       float64
	ValorPIS          float64
	CSTPIS            string
	BaseCOFINS        float64
	AliquotaCOFINS    float64
	ValorCOFINS       float64
	CSTCOFINS         string
}

type TaxTotals struct {
	BaseICMS      float64
	ValorICMS     float64
	ValorDiferido float64
	BaseIPI       float64
	ValorIPI      float64
	BasePIS       float64
	ValorPIS      float64
	BaseCOFINS    float64
	ValorCOFINS   float64
	ValorDIFAL    float64
	ValorFCP      float64
	ValorFomentar float64
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
		result.Totais.ValorDiferido += itemResult.ValorICMSDiferido
		result.Totais.BaseIPI += itemResult.BaseIPI
		result.Totais.ValorIPI += itemResult.ValorIPI
		result.Totais.BasePIS += itemResult.BasePIS
		result.Totais.ValorPIS += itemResult.ValorPIS
		result.Totais.BaseCOFINS += itemResult.BaseCOFINS
		result.Totais.ValorCOFINS += itemResult.ValorCOFINS
		result.Totais.ValorDIFAL += itemResult.ValorDIFAL
		result.Totais.ValorFCP += itemResult.ValorFCP
		result.Totais.ValorFomentar += itemResult.ValorFomentar
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

	// IPI — use transfer price when provided (IPITransferSalesTableId lookup resolved upstream)
	ipiBase := itemTotal
	if item.IPITransferPrice > 0 {
		ipiBase = decimal.NewFromFloat(item.IPITransferPrice).Mul(decimal.NewFromFloat(item.Quantidade))
	}

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

	r.BaseIPI, _ = ipiBase.Round(2).Float64()
	r.ValorIPI, _ = ipiBase.Mul(aliqIPI).Round(2).Float64()

	totalComIPI := itemTotal.Add(ipiBase.Mul(aliqIPI))

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
		aliqPIS = decimal.NewFromFloat(ncmCfg.AliqPis)
		cstPIS = ncmCfg.CstPis
		aliqCOFINS = decimal.NewFromFloat(ncmCfg.AliqCofins)
		cstCOFINS = ncmCfg.CstCofins
	}

	r.BasePIS, _ = basePisCofins.Round(2).Float64()
	r.AliquotaPIS, _ = aliqPIS.Float64()
	r.CSTPIS = cstPIS
	r.BaseCOFINS, _ = basePisCofins.Round(2).Float64()
	r.AliquotaCOFINS, _ = aliqCOFINS.Float64()
	r.CSTCOFINS = cstCOFINS
	if !params.Flags.SkipPISCOFINS {
		r.ValorPIS, _ = basePisCofins.Mul(aliqPIS).Round(2).Float64()
		r.ValorCOFINS, _ = basePisCofins.Mul(aliqCOFINS).Round(2).Float64()
	}

	// ICMS
	if isInterno {
		// Cenário 2 (PR contribuinte) ou 3 (PR não contribuinte)
		// Base ICMS: para não contribuinte inclui IPI; para contribuinte não inclui IPI
		var baseICMS decimal.Decimal
		if params.DestinoTipo == "contribuinte" {
			baseICMS = itemTotal.Add(freteItem).Sub(descontoItem)
		} else {
			baseICMS = totalComIPI.Add(freteItem).Sub(descontoItem)
		}

		aliqICMS := decimal.NewFromFloat(fiscalConfig.IcmsInternoAliquota)
		r.AliquotaICMS, _ = aliqICMS.Float64()
		r.BaseICMS, _ = baseICMS.Round(2).Float64()
		valorICMS := baseICMS.Mul(aliqICMS)
		r.ValorICMS, _ = valorICMS.Round(2).Float64()

		if params.DestinoTipo == "contribuinte" {
			// Diferimento parcial 38.46% → CST 51
			difPct := decimal.NewFromFloat(fiscalConfig.IcmsDiferimentoPercentual).Div(decimal.NewFromInt(100))
			diferido := valorICMS.Mul(difPct)
			r.ValorICMSDiferido, _ = diferido.Round(2).Float64()
			r.CSTICMS = "51"
		} else {
			r.ValorICMSDiferido = 0
			r.CSTICMS = "00"
		}
	} else {
		// Cenário 1 – Interestadual
		interstateKey := params.EmitenteUF + params.DestinoUF

		// Resolução SF 13/2012: mercadoria importada (origens 3,4,5,8) → 4%
		importadaOrigens := map[string]bool{"3": true, "4": true, "5": true, "8": true}
		var aliqInter decimal.Decimal
		if importadaOrigens[item.OrigemMercadoria] {
			aliqInter = decimal.NewFromFloat(0.04)
		} else {
			aliq, ok := interstateTable[interstateKey]
			if !ok || aliq == 0 {
				aliq = 0.12 // default Sul/Sudeste quando não mapeado
			}
			aliqInter = decimal.NewFromFloat(aliq)
		}

		// Base ICMS interestadual para não contribuinte inclui IPI
		var baseICMS decimal.Decimal
		if params.DestinoTipo != "contribuinte" {
			baseICMS = totalComIPI.Add(freteItem).Sub(descontoItem)
		} else {
			baseICMS = itemTotal.Add(freteItem).Sub(descontoItem)
		}

		r.AliquotaICMS, _ = aliqInter.Float64()
		r.BaseICMS, _ = baseICMS.Round(2).Float64()
		r.ValorICMS, _ = baseICMS.Mul(aliqInter).Round(2).Float64()
		r.CSTICMS = "00"

		// DIFAL (EC 87/2015) para venda a não contribuinte interestadual
		if params.DestinoTipo == "nao_contribuinte" || params.DestinoTipo == "pessoa_fisica" {
			internalCfg, hasInternal := internalTable[params.DestinoUF]
			if hasInternal && internalCfg.ICMS > 0 {
				aliqInterna := decimal.NewFromFloat(internalCfg.ICMS)
				difal := aliqInterna.Sub(aliqInter)
				if difal.IsPositive() {
					r.ValorDIFAL, _ = baseICMS.Mul(difal).Round(2).Float64()
				}
				if internalCfg.FCP > 0 {
					r.ValorFCP, _ = baseICMS.Mul(decimal.NewFromFloat(internalCfg.FCP)).Round(2).Float64()
				}
			}
		}
	}

	// CalcReducao: apply ICMSReductionPct to reduce the effective ICMS base and recalculate.
	// Example: ICMSReductionPct=0.30 → only 70% of the computed base is taxed.
	if params.Flags.CalcReducao && params.Flags.ICMSReductionPct > 0 {
		reductionFactor := decimal.NewFromFloat(1 - params.Flags.ICMSReductionPct)
		reducedBase := decimal.NewFromFloat(r.BaseICMS).Mul(reductionFactor)
		r.BaseICMS, _ = reducedBase.Round(2).Float64()
		newValorICMS := reducedBase.Mul(decimal.NewFromFloat(r.AliquotaICMS))
		r.ValorICMS, _ = newValorICMS.Round(2).Float64()
		// Recalculate diferimento over the reduced amount
		if r.ValorICMSDiferido > 0 {
			difPct := decimal.NewFromFloat(fiscalConfig.IcmsDiferimentoPercentual).Div(decimal.NewFromInt(100))
			r.ValorICMSDiferido, _ = newValorICMS.Mul(difPct).Round(2).Float64()
		}
	}

	// CalcFomentar: portion of ICMS retained by state via FOMENTAR/PRODUZIR incentive.
	// ValorFomentar = ValorICMS × FomentarRetentionPct (ICMS that is credited back to the company).
	if params.Flags.CalcFomentar && params.Flags.FomentarRetentionPct > 0 {
		r.ValorFomentar, _ = decimal.NewFromFloat(r.ValorICMS).
			Mul(decimal.NewFromFloat(params.Flags.FomentarRetentionPct)).Round(2).Float64()
	}

	roundResult(&r)
	return r
}

func roundResult(r *TaxItemResult) {
	r.BaseICMS, _ = decimal.NewFromFloat(r.BaseICMS).Round(2).Float64()
	r.ValorICMS, _ = decimal.NewFromFloat(r.ValorICMS).Round(2).Float64()
	r.ValorICMSDiferido, _ = decimal.NewFromFloat(r.ValorICMSDiferido).Round(2).Float64()
	r.ValorDIFAL, _ = decimal.NewFromFloat(r.ValorDIFAL).Round(2).Float64()
	r.ValorFCP, _ = decimal.NewFromFloat(r.ValorFCP).Round(2).Float64()
	r.ValorFomentar, _ = decimal.NewFromFloat(r.ValorFomentar).Round(2).Float64()
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
