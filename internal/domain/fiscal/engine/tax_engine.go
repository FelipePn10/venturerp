package engine

import (
	"fmt"
	"strings"

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
	CalcFomentar         bool
	FomentarRetentionPct float64 // fraction of ICMS retained by state (e.g. 0.70 = 70%)

	// IPITransferPrice: when > 0, overrides ValorUnitario as IPI base (used for
	// transfer invoices where the IPI base comes from a sales table).
	// Set this per-item in TaxItem.IPITransferPrice instead.

	// Indicadores do tipo de nota (VFIS0110) escolhido na saída. Valor zero =
	// comportamento padrão, para quem não informa tipo de nota nada mudar.
	// SemDiferimento: o tipo de nota não calcula diferimento de ICMS.
	SemDiferimento bool
	// SemST: o tipo de nota não calcula ST pela regra do cadastro (MVA
	// informada à mão na linha continua valendo).
	SemST bool
	// SemDIFAL: o tipo de nota não calcula o diferencial de alíquota.
	SemDIFAL bool
	// CSTICMSOperacao: CST imposto pela operação (40 isenta, 41 não tributada)
	// quando o tipo de nota é de ICMS isento/outros. Vazio = o do cálculo.
	CSTICMSOperacao string
}

// Regime tributário do emitente, como o motor o entende.
type Regime string

const (
	RegimeSimples   Regime = "SIMPLES"
	RegimePresumido Regime = "PRESUMIDO"
	RegimeReal      Regime = "REAL"
)

// RegimeDoCadastro traduz o regime da configuração fiscal ("1" Simples, "2"
// Lucro Presumido, "3" Lucro Real — ou os textos antigos, como "lucro_real").
// Desconhecido vale Lucro Real, o cálculo que o sistema sempre fez.
func RegimeDoCadastro(v string) Regime {
	r := strings.ToLower(strings.TrimSpace(v))
	switch {
	case r == "1" || strings.Contains(r, "simples"):
		return RegimeSimples
	case r == "2" || strings.Contains(r, "presumido"):
		return RegimePresumido
	default:
		return RegimeReal
	}
}

// Finalidade da compra pelo destinatário.
const (
	FinalidadeRevenda          = "REVENDA"
	FinalidadeIndustrializacao = "INDUSTRIALIZACAO"
	FinalidadeUsoConsumo       = "USO_CONSUMO"
)

// FinalidadeValida diz se o texto é uma finalidade conhecida (vazio não é).
func FinalidadeValida(f string) bool {
	switch f {
	case FinalidadeRevenda, FinalidadeIndustrializacao, FinalidadeUsoConsumo:
		return true
	}
	return false
}

// MvaAjustada aplica a fórmula do Convênio ICMS 142/18 (cláusula 13ª) à MVA
// original na operação interestadual. Frações (0,40 = 40%).
func MvaAjustada(mva, aliqInter, aliqIntra decimal.Decimal) decimal.Decimal {
	um := decimal.NewFromInt(1)
	if !um.Sub(aliqIntra).IsPositive() {
		return mva
	}
	return um.Add(mva).Mul(um.Sub(aliqInter)).Div(um.Sub(aliqIntra)).Sub(um).Round(6)
}

// CSOSNDoCST deduz o CSOSN do Simples Nacional a partir do CST que o cálculo
// chegou: tributada com crédito (101) ou sem (102); com ST (201/202); sem
// tributação (400); ST cobrada antes (500).
func CSOSNDoCST(cst string, comST, comCredito bool) string {
	switch {
	case cst == "60":
		return "500"
	case comST && (cst == "30" || cstICMSSemImposto[cst]):
		return "202"
	case comST && comCredito:
		return "201"
	case comST:
		return "202"
	case cstICMSSemImposto[cst]:
		return "400"
	case comCredito:
		return "101"
	default:
		return "102"
	}
}

// pisCofinsDoRegime: alíquotas e CST de PIS/COFINS da linha. Lucro Real é o
// não cumulativo (1,65% e 7,6%); Lucro Presumido, o cumulativo (0,65% e 3%);
// o Simples não destaca (CST 49). A tabela de NCM traz as exceções do produto
// (monofásico, alíquota zero, suspensão...): no Real ela vale inteira; no
// Presumido, só o que não é a alíquota básica (CST 01), que é do regime.
func pisCofinsDoRegime(regime Regime, cfg *NcmTaxConfig, hasNcm bool) (aliqPIS, aliqCOFINS decimal.Decimal, cstPIS, cstCOFINS string) {
	switch regime {
	case RegimeSimples:
		return decimal.Zero, decimal.Zero, "49", "49"
	case RegimePresumido:
		aliqPIS, cstPIS = decimal.NewFromFloat(0.0065), "01"
		aliqCOFINS, cstCOFINS = decimal.NewFromFloat(0.03), "01"
		if hasNcm {
			if c := strings.TrimSpace(cfg.CstPis); c != "" && c != "01" {
				aliqPIS, cstPIS = decimal.NewFromFloat(cfg.AliqPis), c
			}
			if c := strings.TrimSpace(cfg.CstCofins); c != "" && c != "01" {
				aliqCOFINS, cstCOFINS = decimal.NewFromFloat(cfg.AliqCofins), c
			}
		}
		return
	default:
		if hasNcm {
			return decimal.NewFromFloat(cfg.AliqPis), decimal.NewFromFloat(cfg.AliqCofins), cfg.CstPis, cfg.CstCofins
		}
		return decimal.NewFromFloat(0.0165), decimal.NewFromFloat(0.076), "01", "01"
	}
}

type TaxCalculationParams struct {
	Itens       []TaxItem
	EmitenteUF  string
	DestinoUF   string
	DestinoTipo string
	Cfop        string
	Flags       InvoiceTypeFlags
	// Finalidade da compra (REVENDA, INDUSTRIALIZACAO, USO_CONSUMO); vazio =
	// deduzir do destinatário. Ver ConsumidorFinal.
	Finalidade string
}

// ConsumidorFinal: a mercadoria sai do ciclo comercial — destinatário não
// contribuinte, ou contribuinte comprando para uso, consumo ou ativo.
func (p TaxCalculationParams) ConsumidorFinal() bool {
	return p.DestinoTipo != "contribuinte" || p.Finalidade == FinalidadeUsoConsumo
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

	// ── Substituição Tributária (ICMS-ST) ──────────────────────────────────
	// MvaPct is the "Margem de Valor Agregado" as a ratio (e.g. 0.40 = 40%).
	// When > 0, the engine computes ICMS-ST for the item. The MVA may already be
	// the "MVA ajustada" for interstate operations — the engine applies it as-is.
	MvaPct float64
	// AliqInternaDestinoST optionally overrides the destination internal ICMS
	// rate used as the ST rate. When 0, the engine resolves it from the internal
	// table (interstate) or from the fiscal config (intra-state).
	AliqInternaDestinoST float64
	// RedBaseSTPct optionally reduces the ST base (ratio, e.g. 0.20 = 20% off).
	RedBaseSTPct float64

	// Parametro é a regra de ICMS/IPI do cadastro (VFIS0320) que casou com a
	// linha; nil = regra geral da empresa. Ver ParametroICMSIPI.
	Parametro *ParametroICMSIPI
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
	BaseICMSST        float64 // base de cálculo da Substituição Tributária
	AliquotaICMSST    float64 // alíquota interna do destino usada na ST
	ValorICMSST       float64 // ICMS-ST a recolher (substituto tributário)
	MVA               float64 // margem de valor agregado aplicada (ratio)
	CSTICMS           string
	// ReducaoBaseICMSPct é o percentual de redução da base do ICMS (pRedBC,
	// 33.33 = 33,33%) aplicado pela regra do cadastro; 0 = sem redução.
	ReducaoBaseICMSPct float64
	// DiferimentoICMSPct é o percentual de diferimento aplicado (pDif, 38.46 =
	// 38,46%) — o da regra do cadastro ou o da configuração; 0 = sem diferimento.
	DiferimentoICMSPct float64
	// Simples Nacional: pCredSN (%) e vCredICMSSN, o crédito de ICMS que o
	// comprador contribuinte aproveita (CSOSN 101 e 201).
	AliqCreditoSimplesPct float64
	ValorCreditoSimples   float64
	// ReducaoBaseSTPct: pRedBCST (%) aplicado na base da ST.
	ReducaoBaseSTPct float64
	// Grupo ICMSUFDest (DIFAL a não contribuinte): base na UF de destino,
	// alíquota interna do destino, alíquota interestadual e FCP — em %.
	// ValorDIFAL e ValorFCP são os valores do grupo.
	BaseUFDest           float64
	AliqInternaUFDestPct float64
	AliqInterestadualPct float64
	FCPUFDestPct         float64
	// Notas explica, em português, de onde veio a tributação da linha (regra,
	// regime, diferimento, DIFAL, MVA...). É o que o simulador e a prévia mostram.
	Notas          []string
	BaseIPI        float64
	AliquotaIPI    float64
	ValorIPI       float64
	CSTIPI         string
	BasePIS        float64
	AliquotaPIS    float64
	ValorPIS       float64
	CSTPIS         string
	BaseCOFINS     float64
	AliquotaCOFINS float64
	ValorCOFINS    float64
	CSTCOFINS      string
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
	BaseICMSST    float64
	ValorICMSST   float64
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

	// A tabela entra reindexada por NCM só-dígitos: a máscara com que cada linha
	// foi cadastrada não pode decidir se o item é tributado. Ver NormalizarNCM.
	ncmTable = NormalizarTabelaNCM(ncmTable)

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
		result.Totais.BaseICMSST += itemResult.BaseICMSST
		result.Totais.ValorICMSST += itemResult.ValorICMSST
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
	nota := func(format string, a ...any) { r.Notas = append(r.Notas, fmt.Sprintf(format, a...)) }

	regime := fiscalConfig.Regime
	if regime == "" {
		regime = RegimeReal
	}
	simples := regime == RegimeSimples
	contribuinte := params.DestinoTipo == "contribuinte"
	consumidorFinal := params.ConsumidorFinal()
	flags := params.Flags
	if item.Parametro != nil {
		nota("regra de ICMS/IPI nº %d do cadastro", item.Parametro.ID)
	} else {
		nota("sem regra de ICMS/IPI no cadastro: vale a regra geral da empresa")
	}

	itemTotal := decimal.NewFromFloat(item.ValorUnitario).Mul(decimal.NewFromFloat(item.Quantidade))

	// IPI — use transfer price when provided (IPITransferSalesTableId lookup resolved upstream)
	ipiBase := itemTotal
	if item.IPITransferPrice > 0 {
		ipiBase = decimal.NewFromFloat(item.IPITransferPrice).Mul(decimal.NewFromFloat(item.Quantidade))
	}

	// IPI
	ncmCfg, hasNcm := ncmTable[NormalizarNCM(item.Ncm)]
	aliqIPI := decimal.NewFromFloat(0)
	cstIPI := "50"
	if hasNcm {
		aliqIPI = decimal.NewFromFloat(ncmCfg.AliqIPI)
		cstIPI = ncmCfg.CstIPI
	}
	ipiBase, aliqIPI, cstIPI = item.Parametro.ajustarIPI(params.DestinoTipo, ipiBase, aliqIPI, cstIPI)
	if simples {
		// O optante do Simples recolhe o IPI dentro do DAS e não o destaca na
		// nota: CST 99 (outras saídas) com valor zero, salvo CST sem imposto da regra.
		if !cstIPISemImposto[cstIPI] {
			cstIPI = "99"
		}
		aliqIPI = decimal.Zero
		nota("IPI não destacado: optante do Simples Nacional recolhe no DAS")
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
	aliqPIS, aliqCOFINS, cstPIS, cstCOFINS := pisCofinsDoRegime(regime, ncmCfg, hasNcm)
	switch regime {
	case RegimeSimples:
		nota("PIS/COFINS não destacados: optante do Simples Nacional recolhe no DAS (CST 49)")
	case RegimePresumido:
		nota("PIS/COFINS cumulativos do lucro presumido (0,65%% e 3%%) salvo exceção da tabela de NCM")
	}
	if flags.SkipPISCOFINS {
		// O tipo de nota não calcula PIS/COFINS: CST 49 (outras operações de
		// saída), e não CST 01 com valor zero, que declara tributação inexistente.
		aliqPIS, aliqCOFINS, cstPIS, cstCOFINS = decimal.Zero, decimal.Zero, "49", "49"
		nota("PIS/COFINS não calculados: o tipo de nota não calcula PIS/COFINS")
	}

	r.BasePIS, _ = basePisCofins.Round(2).Float64()
	r.AliquotaPIS, _ = aliqPIS.Float64()
	r.CSTPIS = cstPIS
	r.BaseCOFINS, _ = basePisCofins.Round(2).Float64()
	r.AliquotaCOFINS, _ = aliqCOFINS.Float64()
	r.CSTCOFINS = cstCOFINS
	r.ValorPIS, _ = basePisCofins.Mul(aliqPIS).Round(2).Float64()
	r.ValorCOFINS, _ = basePisCofins.Mul(aliqCOFINS).Round(2).Float64()

	// ICMS. A base leva o IPI quando a mercadoria vai para consumidor final —
	// não contribuinte, ou contribuinte comprando para uso, consumo ou ativo (CF
	// art. 155, §2º, XI). Entre contribuintes, para revenda ou industrialização,
	// o IPI fica fora.
	baseOperacao := itemTotal.Add(freteItem).Sub(descontoItem)
	if consumidorFinal {
		baseOperacao = totalComIPI.Add(freteItem).Sub(descontoItem)
	}
	redBase := 0.0 // % de redução da base pela regra do cadastro (pRedBC)
	var aliqICMS, aliqInterNominal decimal.Decimal
	baseICMS := baseOperacao
	if isInterno {
		aliqICMS = decimal.NewFromFloat(fiscalConfig.IcmsInternoAliquota)
		baseICMS, aliqICMS, redBase = item.Parametro.ajustarICMS(params.DestinoTipo, baseICMS, aliqICMS, false)
	} else {
		// Resolução SF 13/2012: importada (origens 1 e 2) ou nacional com
		// conteúdo de importação acima de 40% (origens 3 e 8) → 4%. As origens 4
		// e 5 são nacionais e seguem a alíquota interestadual comum; 6 e 7 (sem
		// similar nacional) ficam fora da resolução. Antes a lista era 3,4,5,8.
		importadaOrigens := map[string]bool{"1": true, "2": true, "3": true, "8": true}
		importada := importadaOrigens[item.OrigemMercadoria]
		if importada {
			aliqInterNominal = decimal.NewFromFloat(0.04)
		} else {
			aliq, ok := interstateTable[params.EmitenteUF+params.DestinoUF]
			if !ok || aliq == 0 {
				aliq = AliquotaInterestadualLegal(params.EmitenteUF, params.DestinoUF)
			}
			aliqInterNominal = decimal.NewFromFloat(aliq)
		}
		baseICMS, aliqICMS, redBase = item.Parametro.ajustarICMS(params.DestinoTipo, baseICMS, aliqInterNominal, importada)
		if item.Parametro != nil && importada && item.Parametro.AliqImportadoPct > 0 {
			aliqInterNominal = fracao(item.Parametro.AliqImportadoPct)
		}
	}
	valorICMS := baseICMS.Mul(aliqICMS)
	r.AliquotaICMS, _ = aliqICMS.Float64()
	r.BaseICMS, _ = baseICMS.Round(2).Float64()
	r.ValorICMS, _ = valorICMS.Round(2).Float64()
	r.CSTICMS = "00"

	// Diferimento parcial (CST 51): só na operação interna entre contribuintes
	// em que a mercadoria segue no ciclo comercial. Uso, consumo e ativo
	// encerram o diferimento (SEFA/PR, Consulta 11/2011). O tipo de nota pode
	// desligá-lo. Sem percentual configurado, CST 00: antes saía CST 51 com 0%
	// diferido e a SEFAZ do PR rejeitava pedindo o cBenef (rejeição 930).
	if isInterno && contribuinte && !consumidorFinal && !flags.SemDiferimento {
		difPct := FracaoDiferimento(fiscalConfig.IcmsDiferimentoPercentual)
		if item.Parametro != nil && item.Parametro.DiferimentoPct > 0 {
			difPct = fracao(item.Parametro.DiferimentoPct)
		}
		if difPct.IsPositive() {
			// ValorICMS segue sendo o da operação; o destacado na nota (operação −
			// diferido) é montado no payload.
			r.ValorICMSDiferido, _ = valorICMS.Mul(difPct).Round(2).Float64()
			r.DiferimentoICMSPct, _ = difPct.Mul(cem).Round(4).Float64()
			r.CSTICMS = "51"
			nota("ICMS diferido em %s%% (operação interna entre contribuintes)", pctBR(difPct.Mul(cem)))
		}
	} else if isInterno && contribuinte && consumidorFinal {
		nota("sem diferimento: contribuinte comprando para uso, consumo ou ativo é consumidor final")
	}

	// DIFAL (EC 87/2015): venda interestadual a consumidor final NÃO contribuinte.
	// O remetente recolhe a diferença para a UF de destino e a nota leva o grupo
	// ICMSUFDest. Contribuinte comprando para uso e consumo recolhe o DIFAL ele
	// mesmo — não vai na nota do vendedor. O optante do Simples não recolhe
	// (STF, ADI 5464).
	if !isInterno && !contribuinte {
		difalTipo := ""
		if item.Parametro != nil {
			difalTipo = strings.ToUpper(strings.TrimSpace(item.Parametro.DifalTipo))
		}
		internalCfg, hasInternal := internalTable[params.DestinoUF]
		switch {
		case simples:
			nota("sem DIFAL: optante do Simples Nacional (STF, ADI 5464)")
		case flags.SemDIFAL:
			nota("sem DIFAL: o tipo de nota não calcula o diferencial de alíquota")
		case difalTipo == "NAO_CONSIDERA" || difalTipo == "ISENTO_OUTRAS":
			nota("sem DIFAL: a regra de ICMS/IPI não considera o diferencial de alíquota")
		case !hasInternal || internalCfg.ICMS <= 0:
			nota("DIFAL não calculado: falta a alíquota interna de %s na tabela de ICMS", params.DestinoUF)
		default:
			baseDest := baseOperacao
			if item.Parametro != nil && item.Parametro.DifalRedPct > 0 {
				baseDest = baseDest.Mul(decimal.NewFromInt(1).Sub(fracao(item.Parametro.DifalRedPct)))
			}
			aliqInterna := decimal.NewFromFloat(internalCfg.ICMS)
			r.BaseUFDest, _ = baseDest.Round(2).Float64()
			r.AliqInternaUFDestPct, _ = aliqInterna.Mul(cem).Round(4).Float64()
			r.AliqInterestadualPct, _ = aliqInterNominal.Mul(cem).Round(4).Float64()
			if difal := aliqInterna.Sub(aliqInterNominal); difal.IsPositive() {
				r.ValorDIFAL, _ = baseDest.Mul(difal).Round(2).Float64()
			}
			fcpPct := internalCfg.FCP
			if item.Parametro != nil && item.Parametro.FCPProdutoPct > 0 {
				fcpPct = fracao(item.Parametro.FCPProdutoPct).InexactFloat64()
			}
			if fcpPct > 0 {
				fcp := decimal.NewFromFloat(fcpPct)
				r.FCPUFDestPct, _ = fcp.Mul(cem).Round(4).Float64()
				r.ValorFCP, _ = baseDest.Mul(fcp).Round(2).Float64()
			}
			nota("DIFAL para %s: interna %s%% − interestadual %s%%", params.DestinoUF,
				pctBR(aliqInterna.Mul(cem)), pctBR(aliqInterNominal.Mul(cem)))
		}
	}

	// CalcReducao: apply ICMSReductionPct to reduce the effective ICMS base and recalculate.
	// Example: ICMSReductionPct=0.30 → only 70% of the computed base is taxed.
	if flags.CalcReducao && flags.ICMSReductionPct > 0 {
		reductionFactor := decimal.NewFromFloat(1 - flags.ICMSReductionPct)
		reducedBase := decimal.NewFromFloat(r.BaseICMS).Mul(reductionFactor)
		r.BaseICMS, _ = reducedBase.Round(2).Float64()
		newValorICMS := reducedBase.Mul(decimal.NewFromFloat(r.AliquotaICMS))
		r.ValorICMS, _ = newValorICMS.Round(2).Float64()
		// Recalculate diferimento over the reduced amount
		if r.ValorICMSDiferido > 0 {
			r.ValorICMSDiferido, _ = newValorICMS.Mul(decimal.NewFromFloat(r.DiferimentoICMSPct).Div(cem)).Round(2).Float64()
		}
	}

	// CST da regra do cadastro — ou o da operação, quando o tipo de nota é
	// isento/outros (a operação manda sobre o produto). Sem CST informado,
	// redução de base em operação tributada vira CST 20. CST sem imposto próprio
	// (isenta, não tributada, suspensão, ST anterior) zera o ICMS, o diferido e o DIFAL.
	r.ReducaoBaseICMSPct = redBase
	cstForcado := item.Parametro.cstICMS(params.DestinoTipo)
	if c := strings.TrimSpace(flags.CSTICMSOperacao); c != "" {
		cstForcado = c
		nota("CST %s do ICMS imposto pelo tipo de nota", c)
	}
	if cstForcado != "" {
		r.CSTICMS = cstForcado
		if cstICMSSemImposto[cstForcado] {
			r.BaseICMS, r.AliquotaICMS, r.ValorICMS, r.ReducaoBaseICMSPct = 0, 0, 0, 0
			r.ValorICMSDiferido, r.DiferimentoICMSPct, r.ValorDIFAL, r.ValorFCP = 0, 0, 0, 0
			r.BaseUFDest, r.AliqInternaUFDestPct, r.AliqInterestadualPct, r.FCPUFDestPct = 0, 0, 0, 0
		} else if cstForcado != "51" {
			r.ValorICMSDiferido, r.DiferimentoICMSPct = 0, 0
		}
		if cstForcado != "51" {
			r.semDiferimento("sem diferimento: o CST " + cstForcado + " foi imposto pela regra ou pelo tipo de nota")
		}
	} else if redBase > 0 && r.CSTICMS == "00" {
		r.CSTICMS = "20"
	}

	// ── Substituição Tributária (ICMS-ST) ──────────────────────────────────────
	// A MVA vem da linha (informada à mão, já como fração e possivelmente
	// ajustada) ou da regra do cadastro — esta só para destinatário contribuinte
	// que revende ou industrializa, e quando o tipo de nota calcula ST. Na venda
	// interestadual a MVA da regra é ajustada (Convênio ICMS 142/18, cl. 13ª):
	//   MVA ajustada = [(1 + MVA) × (1 − ALQ inter) / (1 − ALQ intra)] − 1
	// Fórmula da ST:
	//   BaseST  = (BaseICMSpróprio + IPI) × (1 + MVA) × (1 − redBaseST)
	//   ICMS-ST = BaseST × aliqInternaDestino − ICMSpróprio   (mínimo 0)
	mva := decimal.NewFromFloat(item.MvaPct)
	aliqSTInformada := decimal.NewFromFloat(item.AliqInternaDestinoST)
	redST := decimal.NewFromFloat(item.RedBaseSTPct)
	if !mva.IsPositive() && item.Parametro != nil && item.Parametro.MvaPct > 0 && !flags.SemST {
		if contribuinte && !consumidorFinal {
			mva = fracao(item.Parametro.MvaPct)
			aliqSTInformada = fracao(item.Parametro.Lado(params.DestinoTipo).AliqSTPct)
			redST = fracao(item.Parametro.RedBaseSTPct)
			if !isInterno && item.Parametro.MvaAjustar {
				intra := aliqSTInformada
				if !intra.IsPositive() {
					if cfg, ok := internalTable[params.DestinoUF]; ok && cfg.ICMS > 0 {
						intra = decimal.NewFromFloat(cfg.ICMS)
					}
				}
				if intra.GreaterThan(aliqInterNominal) {
					mva = MvaAjustada(mva, aliqInterNominal, intra)
					nota("MVA ajustada para operação interestadual: %s%%", pctBR(mva.Mul(cem)))
				}
			}
		} else {
			nota("sem ST da regra: destinatário é consumidor final")
		}
	}
	if mva.IsPositive() {
		aliqST := aliqSTInformada
		if !aliqST.IsPositive() {
			if isInterno {
				aliqST = decimal.NewFromFloat(fiscalConfig.IcmsInternoAliquota)
			} else if cfg, ok := internalTable[params.DestinoUF]; ok && cfg.ICMS > 0 {
				aliqST = decimal.NewFromFloat(cfg.ICMS)
			} else {
				aliqST = decimal.NewFromFloat(fiscalConfig.IcmsInternoAliquota)
			}
		}

		// O ICMS próprio que a ST desconta é o da operação, mesmo no Simples
		// (LC 123, art. 13, §6º): calculado pela alíquota normal — por isso o
		// Simples só zera o ICMS próprio depois daqui.
		icmsProprio := decimal.NewFromFloat(r.ValorICMS)
		basePropriaComIPI := decimal.NewFromFloat(r.BaseICMS).Add(decimal.NewFromFloat(r.ValorIPI))
		baseST := basePropriaComIPI.Mul(decimal.NewFromInt(1).Add(mva))
		if redST.IsPositive() {
			baseST = baseST.Mul(decimal.NewFromInt(1).Sub(redST))
		}

		valorST := baseST.Mul(aliqST).Sub(icmsProprio)
		if valorST.IsNegative() {
			valorST = decimal.Zero
		}

		r.MVA, _ = mva.Round(6).Float64()
		r.ReducaoBaseSTPct, _ = redST.Mul(cem).Round(4).Float64()
		r.BaseICMSST, _ = baseST.Round(2).Float64()
		r.AliquotaICMSST, _ = aliqST.Float64()
		r.ValorICMSST, _ = valorST.Round(2).Float64()

		// Promote the CST to a substitution variant: 00→10 (tributada+ST),
		// 20→70 (com redução de base + ST). ST and diferimento (51) are mutually
		// exclusive, so ST prevails: 51→10 and the diferido amount is cleared.
		switch r.CSTICMS {
		case "00", "51":
			if r.CSTICMS == "51" {
				r.semDiferimento("sem diferimento: a operação tem ICMS-ST, que prevalece sobre o diferimento")
			}
			r.CSTICMS = "10"
			r.ValorICMSDiferido, r.DiferimentoICMSPct = 0, 0
		case "20":
			r.CSTICMS = "70"
		case "40", "41":
			r.CSTICMS = "30"
		}
	}

	// Simples Nacional: o ICMS próprio é recolhido no DAS e não é destacado. A
	// situação vai como CSOSN (da regra, ou deduzida do CST calculado), com o
	// crédito que o comprador contribuinte pode aproveitar (pCredSN) quando há.
	if simples {
		comST := r.ValorICMSST > 0 || r.BaseICMSST > 0
		creditoPermitido := fiscalConfig.CreditoSimplesPct > 0 && contribuinte && !consumidorFinal
		csosn := ""
		if item.Parametro != nil {
			csosn = strings.TrimSpace(item.Parametro.CSOSN)
		}
		if csosn == "" {
			csosn = CSOSNDoCST(r.CSTICMS, comST, creditoPermitido)
		}
		r.CSTICMS = csosn
		r.BaseICMS, r.AliquotaICMS, r.ValorICMS, r.ReducaoBaseICMSPct = 0, 0, 0, 0
		r.ValorICMSDiferido, r.DiferimentoICMSPct = 0, 0
		r.semDiferimento("sem diferimento: o ICMS do Simples Nacional é recolhido no DAS")
		r.ValorDIFAL, r.ValorFCP, r.BaseUFDest, r.AliqInternaUFDestPct, r.AliqInterestadualPct, r.FCPUFDestPct = 0, 0, 0, 0, 0, 0
		if (csosn == "101" || csosn == "201") && fiscalConfig.CreditoSimplesPct > 0 {
			pct := decimal.NewFromFloat(fiscalConfig.CreditoSimplesPct)
			r.AliqCreditoSimplesPct, _ = pct.Round(4).Float64()
			r.ValorCreditoSimples, _ = baseOperacao.Mul(pct).Div(cem).Round(2).Float64()
			nota("crédito de ICMS do Simples para o comprador: %s%%", pctBR(pct))
		}
		nota("ICMS do Simples Nacional: CSOSN %s", csosn)
	}

	// CalcFomentar: portion of ICMS retained by state via FOMENTAR/PRODUZIR incentive.
	// ValorFomentar = ValorICMS × FomentarRetentionPct (ICMS that is credited back to the company).
	if flags.CalcFomentar && flags.FomentarRetentionPct > 0 {
		r.ValorFomentar, _ = decimal.NewFromFloat(r.ValorICMS).
			Mul(decimal.NewFromFloat(flags.FomentarRetentionPct)).Round(2).Float64()
	}

	roundResult(&r)
	return r
}

// semDiferimento troca a nota explicativa do diferimento quando um passo
// posterior o desfaz (CST imposto, ST, Simples): a linha não pode dizer
// "ICMS diferido" com o diferido zerado. Sem a nota, não faz nada.
func (r *TaxItemResult) semDiferimento(motivo string) {
	for k, n := range r.Notas {
		if strings.HasPrefix(n, "ICMS diferido em ") {
			r.Notas[k] = motivo
		}
	}
}

func roundResult(r *TaxItemResult) {
	r.BaseICMS, _ = decimal.NewFromFloat(r.BaseICMS).Round(2).Float64()
	r.ValorICMS, _ = decimal.NewFromFloat(r.ValorICMS).Round(2).Float64()
	r.ValorICMSDiferido, _ = decimal.NewFromFloat(r.ValorICMSDiferido).Round(2).Float64()
	r.ValorDIFAL, _ = decimal.NewFromFloat(r.ValorDIFAL).Round(2).Float64()
	r.ValorFCP, _ = decimal.NewFromFloat(r.ValorFCP).Round(2).Float64()
	r.ValorFomentar, _ = decimal.NewFromFloat(r.ValorFomentar).Round(2).Float64()
	r.BaseICMSST, _ = decimal.NewFromFloat(r.BaseICMSST).Round(2).Float64()
	r.ValorICMSST, _ = decimal.NewFromFloat(r.ValorICMSST).Round(2).Float64()
	r.BaseIPI, _ = decimal.NewFromFloat(r.BaseIPI).Round(2).Float64()
	r.ValorIPI, _ = decimal.NewFromFloat(r.ValorIPI).Round(2).Float64()
	r.BasePIS, _ = decimal.NewFromFloat(r.BasePIS).Round(2).Float64()
	r.ValorPIS, _ = decimal.NewFromFloat(r.ValorPIS).Round(2).Float64()
	r.BaseCOFINS, _ = decimal.NewFromFloat(r.BaseCOFINS).Round(2).Float64()
	r.ValorCOFINS, _ = decimal.NewFromFloat(r.ValorCOFINS).Round(2).Float64()
	r.ValorCreditoSimples, _ = decimal.NewFromFloat(r.ValorCreditoSimples).Round(2).Float64()
	r.BaseUFDest, _ = decimal.NewFromFloat(r.BaseUFDest).Round(2).Float64()
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
	UFEmpresa                 string
	IcmsInternoAliquota       float64
	IcmsDiferimentoPercentual float64
	// Regime do emitente; vazio = Lucro Real.
	Regime Regime
	// CreditoSimplesPct: pCredSN (%) do optante do Simples.
	CreditoSimplesPct float64
}

var _ = fmt.Println

// FracaoDiferimento normaliza o percentual de diferimento da configuração fiscal
// para fração. A configuração guarda fração (0,3846 = 38,46%, como diz a tela),
// mas o motor dividia por 100 de novo e o ICMS diferido saía cem vezes menor.
// Valor acima de 1 é tratado como porcentagem, para não errar em quem digitou 38,46.
func FracaoDiferimento(v float64) decimal.Decimal {
	d := decimal.NewFromFloat(v)
	if d.GreaterThan(decimal.NewFromInt(1)) {
		return d.Div(decimal.NewFromInt(100))
	}
	if d.IsNegative() {
		return decimal.Zero
	}
	return d
}

// regiaoDaUF: N, NE, CO, SE e S.
var regiaoDaUF = map[string]string{
	"AC": "N", "AM": "N", "AP": "N", "PA": "N", "RO": "N", "RR": "N", "TO": "N",
	"AL": "NE", "BA": "NE", "CE": "NE", "MA": "NE", "PB": "NE", "PE": "NE", "PI": "NE", "RN": "NE", "SE": "NE",
	"DF": "CO", "GO": "CO", "MS": "CO", "MT": "CO",
	"ES": "SE", "MG": "SE", "RJ": "SE", "SP": "SE",
	"PR": "S", "RS": "S", "SC": "S",
}

// AliquotaInterestadualLegal é a alíquota da Resolução SF 22/1989 quando a
// tabela de ICMS interestadual não tem o par: do Sul e do Sudeste (exceto ES)
// para Norte, Nordeste, Centro-Oeste e ES, 7%; nas demais, 12%. Antes a falta
// do par valia 12% sempre — errado para metade das vendas de quem não é do PR.
func AliquotaInterestadualLegal(origem, destino string) float64 {
	o, d := strings.ToUpper(strings.TrimSpace(origem)), strings.ToUpper(strings.TrimSpace(destino))
	ro, rd := regiaoDaUF[o], regiaoDaUF[d]
	if (ro == "S" || ro == "SE") && o != "ES" && (rd == "N" || rd == "NE" || rd == "CO" || d == "ES") {
		return 0.07
	}
	return 0.12
}

// pctBR escreve o número com duas casas e vírgula decimal, para as notas.
func pctBR(d decimal.Decimal) string {
	return strings.Replace(d.StringFixed(2), ".", ",", 1)
}
