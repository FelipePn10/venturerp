package focusnfe

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// DadosDaPrevia completa o payload com o que a Focus tira do cadastro do
// emitente na conta dela (IE, município, nome fantasia) e com a numeração.
type DadosDaPrevia struct {
	Serie           string
	Numero          int64
	IE              string
	CodigoMunicipio string
	NomeFantasia    string
	Complemento     string
}

// AvisoPrevia vai no topo e nas informações complementares do XML da prévia.
const AvisoPrevia = "PREVIA SEM VALIDADE FISCAL - gerada pelo VentureERP antes da transmissao; nao foi assinada nem autorizada pela SEFAZ"

// XMLDaPrevia monta, no layout 4.00 da NF-e, o XML que corresponde ao payload
// que o ERP enviaria à Focus. É para conferência: sai em homologação
// (tpAmb 2), sem chave, assinatura nem protocolo, e sem o que a própria Focus
// acrescenta na transmissão (cNF/cDV, autXML, infRespTec).
func XMLDaPrevia(p NFEPayload, d DadosDaPrevia) []byte {
	var b xmlb
	b.raw(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.raw("<!-- " + AvisoPrevia + " -->")
	b.raw(`<NFe xmlns="http://www.portalfiscal.inf.br/nfe"><infNFe versao="4.00">`)

	emit := p.Emitente
	b.open("ide")
	b.leaf("cUF", codigoUF[strings.ToUpper(emit.UF)])
	b.leaf("natOp", p.NaturezaOperacao)
	b.leaf("mod", "55")
	b.leaf("serie", d.Serie)
	b.leaf("nNF", strconv.FormatInt(d.Numero, 10))
	b.leaf("dhEmi", dataHoraPrevia(p.DataEmissao))
	b.leaf("tpNF", strconv.Itoa(p.TipoDocumento))
	b.leaf("idDest", strconv.Itoa(p.LocalDestino))
	b.leaf("cMunFG", soDigitos(d.CodigoMunicipio))
	b.leaf("tpImp", "1")
	b.leaf("tpEmis", "1")
	b.leaf("tpAmb", "2")
	b.leaf("finNFe", strconv.Itoa(p.FinalidadeEmissao))
	b.leaf("indFinal", strconv.Itoa(p.ConsumidorFinal))
	b.leaf("indPres", strconv.Itoa(p.PresencaComprador))
	b.leaf("indIntermed", "0")
	b.leaf("procEmi", "0")
	b.leaf("verProc", "VentureERP-previa")
	if !refPorItem(p) {
		for _, r := range p.NotasReferenciadas {
			b.open("NFref")
			b.leaf("refNFe", soDigitos(r.ChaveNFe))
			b.close("NFref")
		}
	}
	b.close("ide")

	b.open("emit")
	b.leaf("CNPJ", soDigitos(emit.CNPJ))
	b.leaf("xNome", emit.Nome)
	b.leaf("xFant", d.NomeFantasia)
	b.open("enderEmit")
	b.leaf("xLgr", emit.Logradouro)
	b.leaf("nro", emit.Numero)
	b.leaf("xCpl", d.Complemento)
	b.leaf("xBairro", emit.Bairro)
	b.leaf("cMun", soDigitos(d.CodigoMunicipio))
	b.leaf("xMun", emit.Municipio)
	b.leaf("UF", emit.UF)
	b.leaf("CEP", soDigitos(emit.CEP))
	b.leaf("cPais", "1058")
	b.leaf("xPais", "Brasil")
	b.leaf("fone", soDigitos(emit.Telefone))
	b.close("enderEmit")
	b.leaf("IE", soDigitos(d.IE))
	b.leaf("CRT", strconv.Itoa(emit.RegimeTributario))
	b.close("emit")

	dest := p.Destinatario
	b.open("dest")
	if doc := soDigitos(dest.CNPJCPF); len(doc) == 11 {
		b.leaf("CPF", doc)
	} else {
		b.leaf("CNPJ", doc)
	}
	b.leaf("xNome", dest.Nome)
	b.open("enderDest")
	b.leaf("xLgr", dest.Logradouro)
	b.leaf("nro", dest.Numero)
	b.leaf("xCpl", dest.Complemento)
	b.leaf("xBairro", dest.Bairro)
	b.leaf("cMun", soDigitos(dest.CodigoMunicipio))
	b.leaf("xMun", dest.Municipio)
	b.leaf("UF", dest.UF)
	b.leaf("CEP", soDigitos(dest.CEP))
	b.leaf("fone", soDigitos(dest.Telefone))
	b.close("enderDest")
	b.leaf("indIEDest", strconv.Itoa(dest.IndicadorIE))
	if dest.IndicadorIE == 1 && dest.IE != nil {
		b.leaf("IE", soDigitos(*dest.IE))
	}
	b.leaf("ISUF", soDigitos(dest.Suframa))
	b.leaf("email", dest.Email)
	b.close("dest")

	if e := p.Entrega; e != nil {
		b.open("entrega")
		if doc := soDigitos(e.CNPJCPF); len(doc) == 11 {
			b.leaf("CPF", doc)
		} else {
			b.leaf("CNPJ", doc)
		}
		b.leaf("xNome", e.Nome)
		b.leaf("xLgr", e.Logradouro)
		b.leaf("nro", e.Numero)
		b.leaf("xCpl", e.Complemento)
		b.leaf("xBairro", e.Bairro)
		b.leaf("xMun", e.Municipio)
		b.leaf("UF", e.UF)
		b.leaf("CEP", soDigitos(e.CEP))
		b.close("entrega")
	}

	var t totaisPrevia
	for _, it := range p.Items {
		b.detPrevia(it, &t)
	}

	b.open("total")
	b.open("ICMSTot")
	b.leaf("vBC", vd(t.vBC))
	b.leaf("vICMS", vd(t.vICMS))
	b.leaf("vICMSDeson", vd(t.vDeson))
	if t.temUFDest {
		b.leaf("vFCPUFDest", vd(t.vFCPUFDest))
		b.leaf("vICMSUFDest", vd(t.vICMSUFDest))
		b.leaf("vICMSUFRemet", v2(0))
	}
	b.leaf("vFCP", v2(0))
	b.leaf("vBCST", vd(t.vBCST))
	b.leaf("vST", vd(t.vST))
	b.leaf("vFCPST", v2(0))
	b.leaf("vFCPSTRet", v2(0))
	b.leaf("vProd", vd(t.vProd))
	b.leaf("vFrete", vd(t.vFrete))
	b.leaf("vSeg", vd(t.vSeg))
	b.leaf("vDesc", vd(t.vDesc))
	b.leaf("vII", v2(0))
	b.leaf("vIPI", vd(t.vIPI))
	b.leaf("vIPIDevol", v2(0))
	b.leaf("vPIS", vd(t.vPIS))
	b.leaf("vCOFINS", vd(t.vCOFINS))
	b.leaf("vOutro", v2(0))
	vNF := decimal.NewFromFloat(p.ValorTotal).Round(2)
	if !vNF.IsPositive() {
		vNF = t.vProd.Sub(t.vDesc).Add(t.vST).Add(t.vFrete).Add(t.vSeg).Add(t.vIPI).Sub(t.vDeson)
	}
	b.leaf("vNF", vd(vNF))
	b.close("ICMSTot")
	if t.temIBSCBS {
		b.open("IBSCBSTot")
		b.leaf("vBCIBSCBS", vd(t.vBCIBSCBS))
		b.open("gIBS")
		b.open("gIBSUF")
		b.leaf("vDif", v2(0))
		b.leaf("vDevTrib", v2(0))
		b.leaf("vIBSUF", vd(t.vIBSUF))
		b.close("gIBSUF")
		b.open("gIBSMun")
		b.leaf("vDif", v2(0))
		b.leaf("vDevTrib", v2(0))
		b.leaf("vIBSMun", vd(t.vIBSMun))
		b.close("gIBSMun")
		b.leaf("vIBS", vd(t.vIBS))
		b.leaf("vCredPres", v2(0))
		b.leaf("vCredPresCondSus", v2(0))
		b.close("gIBS")
		b.open("gCBS")
		b.leaf("vDif", v2(0))
		b.leaf("vDevTrib", v2(0))
		b.leaf("vCBS", vd(t.vCBS))
		b.leaf("vCredPres", v2(0))
		b.leaf("vCredPresCondSus", v2(0))
		b.close("gCBS")
		b.close("IBSCBSTot")
	}
	b.close("total")

	b.open("transp")
	modFrete := p.ModalidadeFrete
	if modFrete == 0 && p.ValorFrete == 0 {
		modFrete = 9
	}
	b.leaf("modFrete", strconv.Itoa(modFrete))
	if tr := p.Transportador; tr != nil {
		b.open("transporta")
		if doc := soDigitos(tr.CNPJCPF); len(doc) == 11 {
			b.leaf("CPF", doc)
		} else {
			b.leaf("CNPJ", doc)
		}
		b.leaf("xNome", truncar(tr.Nome, 60))
		b.leaf("IE", soDigitos(tr.IE))
		b.leaf("xEnder", truncar(tr.Endereco, 60))
		b.leaf("xMun", truncar(tr.Municipio, 60))
		b.leaf("UF", tr.UF)
		b.close("transporta")
	}
	b.close("transp")

	if len(p.Duplicatas) > 0 {
		b.open("cobr")
		for _, dup := range p.Duplicatas {
			b.open("dup")
			b.leaf("nDup", dup.Numero)
			b.leaf("dVenc", dup.DataVencimento)
			b.leaf("vDup", v2(dup.Valor))
			b.close("dup")
		}
		b.close("cobr")
	}

	b.open("pag")
	for _, f := range p.FormaPagamento {
		b.open("detPag")
		b.leaf("tPag", f.FormaPagamento)
		b.leaf("vPag", v2(f.Valor))
		b.close("detPag")
	}
	b.close("pag")

	b.open("infAdic")
	b.leaf("infCpl", strings.TrimSpace(AvisoPrevia+". "+strings.TrimSpace(p.InformacoesAdicionais)))
	b.close("infAdic")
	if po := strings.TrimSpace(p.PedidoCompra); po != "" {
		b.open("compra")
		b.leaf("xPed", po)
		b.close("compra")
	}
	b.raw("</infNFe></NFe>")
	return []byte(b.String())
}

// totaisPrevia soma em decimal (centavos exatos): float perde centavo em nota
// grande ou com muitas linhas.
type totaisPrevia struct {
	vBC, vICMS, vDeson, vBCST, vST, vProd, vFrete, vSeg, vDesc, vIPI, vPIS, vCOFINS decimal.Decimal
	temUFDest                                                                       bool
	vFCPUFDest, vICMSUFDest                                                         decimal.Decimal
	temIBSCBS                                                                       bool
	vBCIBSCBS, vIBSUF, vIBSMun, vIBS, vCBS                                          decimal.Decimal
}

// soma acumula o valor da linha já arredondado em centavos, como sai no XML.
func soma(d *decimal.Decimal, f float64) { *d = d.Add(decimal.NewFromFloat(f).Round(2)) }

// vd: total em decimal, no mesmo formato de v2.
func vd(d decimal.Decimal) string {
	if d.Round(2).IsZero() {
		return "0"
	}
	return d.StringFixed(2)
}

func (b *xmlb) detPrevia(it NFEItem, t *totaisPrevia) {
	b.raw(fmt.Sprintf(`<det nItem="%d">`, it.NumeroItem))
	b.open("prod")
	b.leaf("cProd", it.CodigoProduto)
	b.leaf("cEAN", gtin(it.CodigoBarrasComercial))
	b.leaf("xProd", it.Descricao)
	b.leaf("NCM", soDigitos(it.CodigoNCM))
	b.leaf("CEST", soDigitos(it.CEST))
	if it.EscalaRelevante != nil {
		b.leaf("indEscala", map[bool]string{true: "S", false: "N"}[*it.EscalaRelevante])
	}
	b.leaf("CNPJFab", soDigitos(it.CNPJFabricante))
	b.leaf("cBenef", strings.TrimSpace(it.CodigoBeneficioFiscal))
	b.leaf("EXTIPI", it.CodigoExTIPI)
	b.leaf("CFOP", it.CFOP)
	b.leaf("uCom", it.UnidadeComercial)
	b.leaf("qCom", v4(it.QuantidadeComercial))
	b.leaf("vUnCom", vUnit(it.ValorUnitarioComercial))
	b.leaf("vProd", v2(it.ValorBruto))
	b.leaf("cEANTrib", gtin(it.CodigoBarrasTributavel))
	uTrib, qTrib, vUnTrib := it.UnidadeComercial, it.QuantidadeComercial, it.ValorUnitarioComercial
	if it.UnidadeTributavel != "" {
		uTrib, qTrib, vUnTrib = it.UnidadeTributavel, it.QuantidadeTributavel, it.ValorUnitarioTributavel
	}
	b.leaf("uTrib", uTrib)
	b.leaf("qTrib", v4(qTrib))
	b.leaf("vUnTrib", vUnit(vUnTrib))
	b.leafPos("vFrete", it.ValorFrete)
	b.leafPos("vSeg", it.ValorSeguro)
	b.leafPos("vDesc", it.ValorDesconto)
	b.leaf("indTot", "1")
	b.leaf("xPed", strings.TrimSpace(it.PedidoCompra))
	b.leaf("nFCI", it.NumeroFCI)
	b.close("prod")

	b.open("imposto")
	b.icmsPrevia(it, t)
	if it.CodigoSituacaoTributariaIPI != "" {
		b.open("IPI")
		enq := soDigitos(it.EnquadramentoIPI)
		if enq == "" {
			enq = "999"
		}
		b.leaf("cEnq", enq)
		switch it.CodigoSituacaoTributariaIPI {
		case "00", "49", "50", "99":
			base := it.BaseIPI
			if base == 0 && it.ValorIPI > 0 {
				base = it.ValorBruto
			}
			b.open("IPITrib")
			b.leaf("CST", it.CodigoSituacaoTributariaIPI)
			b.leaf("vBC", v2(base))
			b.leaf("pIPI", v4(it.AliquotaIPI))
			b.leaf("vIPI", v2(it.ValorIPI))
			b.close("IPITrib")
		default:
			b.open("IPINT")
			b.leaf("CST", it.CodigoSituacaoTributariaIPI)
			b.close("IPINT")
		}
		b.close("IPI")
		soma(&t.vIPI, it.ValorIPI)
	}
	base := basePISCOFINS(it)
	b.contribPrevia("PIS", it.CodigoSituacaoTributariaPIS, base, it.AliquotaPIS, it.ValorPIS)
	b.contribPrevia("COFINS", it.CodigoSituacaoTributariaCOFINS, base, it.AliquotaCOFINS, it.ValorCOFINS)
	soma(&t.vPIS, it.ValorPIS)
	soma(&t.vCOFINS, it.ValorCOFINS)
	if g := it.UFDest; g != nil {
		b.open("ICMSUFDest")
		b.leaf("vBCUFDest", v2(g.Base))
		baseFCP := 0.0
		if g.PercentualFCP > 0 {
			baseFCP = g.Base
		}
		b.leaf("vBCFCPUFDest", v2(baseFCP))
		b.leaf("pFCPUFDest", v4(g.PercentualFCP))
		b.leaf("pICMSUFDest", v4(g.AliquotaInterna))
		b.leaf("pICMSInter", v2(g.AliquotaInterestadual))
		b.leaf("pICMSInterPart", v4(100))
		b.leaf("vFCPUFDest", v2(g.ValorFCP))
		b.leaf("vICMSUFDest", v2(g.ValorDestino))
		b.leaf("vICMSUFRemet", v2(0))
		b.close("ICMSUFDest")
		t.temUFDest = true
		soma(&t.vFCPUFDest, g.ValorFCP)
		soma(&t.vICMSUFDest, g.ValorDestino)
	}
	if g := it.IBSCBS; g != nil {
		t.temIBSCBS = true
		b.open("IBSCBS")
		b.leaf("CST", g.CST)
		b.leaf("cClassTrib", g.ClassTrib)
		if !g.SemValores {
			b.open("gIBSCBS")
			b.leaf("vBC", v2(g.Base))
			b.open("gIBSUF")
			b.leaf("pIBSUF", v4(g.AliqIBSUF))
			b.redPrevia(g.ComReducao, g.RedIBSUF, g.EfetIBSUF)
			b.leaf("vIBSUF", v2(g.ValorIBSUF))
			b.close("gIBSUF")
			b.open("gIBSMun")
			b.leaf("pIBSMun", v4(g.AliqIBSMun))
			b.redPrevia(g.ComReducao, g.RedIBSMun, g.EfetIBSMun)
			b.leaf("vIBSMun", v2(g.ValorIBSMun))
			b.close("gIBSMun")
			b.leaf("vIBS", v2(g.ValorIBS))
			b.open("gCBS")
			b.leaf("pCBS", v4(g.AliqCBS))
			b.redPrevia(g.ComReducao, g.RedCBS, g.EfetCBS)
			b.leaf("vCBS", v2(g.ValorCBS))
			b.close("gCBS")
			b.close("gIBSCBS")
			soma(&t.vBCIBSCBS, g.Base)
			soma(&t.vIBSUF, g.ValorIBSUF)
			soma(&t.vIBSMun, g.ValorIBSMun)
			soma(&t.vIBS, g.ValorIBS)
			soma(&t.vCBS, g.ValorCBS)
		}
		b.close("IBSCBS")
	}
	b.close("imposto")
	b.leaf("infAdProd", strings.TrimSpace(it.InformacoesAdicionaisItem))
	if it.DFeRefChave != "" && it.DFeRefItem > 0 {
		b.open("DFeReferenciado")
		b.leaf("chaveAcesso", soDigitos(it.DFeRefChave))
		b.leaf("nItem", strconv.Itoa(it.DFeRefItem))
		b.close("DFeReferenciado")
	}
	b.raw("</det>")

	soma(&t.vProd, it.ValorBruto)
	soma(&t.vFrete, it.ValorFrete)
	soma(&t.vSeg, it.ValorSeguro)
	soma(&t.vDesc, it.ValorDesconto)
}

// icmsPrevia escreve o grupo do ICMS pelo CST/CSOSN, na ordem do layout, com
// os mesmos valores que o payload manda à Focus.
func (b *xmlb) icmsPrevia(it NFEItem, t *totaisPrevia) {
	cst := it.CodigoSituacaoTributariaICMS
	simples := len(cst) == 3
	grupo := "ICMS" + cst
	if simples {
		grupo = "ICMSSN" + cst
		switch cst {
		case "102", "103", "300", "400":
			grupo = "ICMSSN102"
		case "202", "203":
			grupo = "ICMSSN202"
		}
	} else if cst == "41" || cst == "50" {
		grupo = "ICMS40"
	}
	vICMS := it.ValorICMS
	if cst == "51" && it.ValorICMSDiferido != nil && *it.ValorICMSDiferido > 0 {
		vICMS = math.Round((it.ValorICMS-*it.ValorICMSDiferido)*100) / 100
	}
	proprio := func(comRed bool) {
		b.leaf("modBC", strconv.Itoa(it.ModalidadeBaseCalculoICMS))
		if comRed {
			b.leafPtrPos("pRedBC", it.PercentualReducaoBC)
		}
		b.leaf("vBC", v2(it.ValorBaseCalculoICMS))
		b.leaf("pICMS", v4(it.AliquotaICMS))
	}
	st := func() {
		if it.ValorICMSST == nil {
			return
		}
		if it.ModalidadeBaseCalculoICMSST != nil {
			b.leaf("modBCST", strconv.Itoa(*it.ModalidadeBaseCalculoICMSST))
		} else {
			b.leaf("modBCST", "4")
		}
		b.leafPtrPos("pMVAST", it.PercentualMVAICMSST)
		b.leafPtrPos("pRedBCST", it.PercentualReducaoBCST)
		b.leaf("vBCST", v2(deref(it.BaseCalculoICMSST)))
		b.leaf("pICMSST", v4(deref(it.AliquotaICMSST)))
		b.leaf("vICMSST", v2(*it.ValorICMSST))
		soma(&t.vBCST, deref(it.BaseCalculoICMSST))
		soma(&t.vST, *it.ValorICMSST)
	}
	deson := func() {
		if it.MotivoDesoneracao != "" && it.ValorICMSDesonerado > 0 {
			b.leaf("vICMSDeson", v2(it.ValorICMSDesonerado))
			b.leaf("motDesICMS", it.MotivoDesoneracao)
			b.leaf("indDeduzDeson", "1")
			soma(&t.vDeson, it.ValorICMSDesonerado)
		}
	}
	credito := func() {
		if it.AliquotaCreditoSimples != nil && it.ValorCreditoSimples != nil {
			b.leaf("pCredSN", v4(*it.AliquotaCreditoSimples))
			b.leaf("vCredICMSSN", v2(*it.ValorCreditoSimples))
		}
	}

	b.open("ICMS")
	b.open(grupo)
	b.leaf("orig", strconv.Itoa(it.OrigemMercadoria))
	if simples {
		b.leaf("CSOSN", cst)
	} else {
		b.leaf("CST", cst)
	}
	switch grupo {
	case "ICMS00":
		proprio(false)
		b.leaf("vICMS", v2(vICMS))
	case "ICMS10":
		proprio(false)
		b.leaf("vICMS", v2(vICMS))
		st()
	case "ICMS20":
		proprio(true)
		b.leaf("vICMS", v2(vICMS))
		deson()
	case "ICMS30":
		st()
		deson()
	case "ICMS40":
		deson()
	case "ICMS51":
		proprio(true)
		if it.ValorICMSDiferido != nil && *it.ValorICMSDiferido > 0 {
			b.leaf("vICMSOp", v2(it.ValorICMS))
			b.leafPtrPos("pDif", it.PercentualDiferimento)
			b.leaf("vICMSDif", v2(*it.ValorICMSDiferido))
		}
		b.leaf("vICMS", v2(vICMS))
	case "ICMS70", "ICMS90":
		proprio(true)
		b.leaf("vICMS", v2(vICMS))
		st()
		deson()
	case "ICMSSN101":
		credito()
	case "ICMSSN201":
		st()
		credito()
	case "ICMSSN202":
		st()
	case "ICMSSN900":
		proprio(true)
		b.leaf("vICMS", v2(vICMS))
		st()
		credito()
	}
	b.close(grupo)
	b.close("ICMS")
	if cstComICMS[cst] {
		soma(&t.vBC, it.ValorBaseCalculoICMS)
		soma(&t.vICMS, vICMS)
	}
}

func (b *xmlb) contribPrevia(tributo, cst string, base, aliq, valor float64) {
	b.open(tributo)
	switch cst {
	case "04", "05", "06", "07", "08", "09":
		b.open(tributo + "NT")
		b.leaf("CST", cst)
		b.close(tributo + "NT")
	default:
		grupo := tributo + "Outr"
		if cst == "01" || cst == "02" {
			grupo = tributo + "Aliq"
		}
		b.open(grupo)
		b.leaf("CST", cst)
		b.leaf("vBC", v2(base))
		b.leaf("p"+tributo, v4(aliq))
		b.leaf("v"+tributo, v2(valor))
		b.close(grupo)
	}
	b.close(tributo)
}

func (b *xmlb) redPrevia(com bool, red, efet float64) {
	if !com {
		return
	}
	b.open("gRed")
	b.leaf("pRedAliq", v4(red))
	b.leaf("pAliqEfet", v4(efet))
	b.close("gRed")
}

func refPorItem(p NFEPayload) bool {
	for _, it := range p.Items {
		if it.DFeRefChave != "" {
			return true
		}
	}
	return false
}

// dataHoraPrevia: a data do payload, com hora e fuso de Brasília se vier só a data.
func dataHoraPrevia(s string) string {
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return s
	}
	if d, err := time.Parse("2006-01-02", s); err == nil {
		return d.Format("2006-01-02") + "T00:00:00-03:00"
	}
	return s
}

func gtin(s string) string {
	if s = soDigitos(s); s == "" {
		return "SEM GTIN"
	}
	return s
}

func deref(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// v2 e v4: zero sai "0", como a Focus escreve.
func v2(f float64) string {
	if math.Round(f*100) == 0 {
		return "0"
	}
	return strconv.FormatFloat(math.Round(f*100)/100, 'f', 2, 64)
}

func v4(f float64) string {
	if math.Round(f*10000) == 0 {
		return "0"
	}
	return strconv.FormatFloat(f, 'f', 4, 64)
}

// vUnit: até 10 casas, sem zeros à direita além da segunda.
func vUnit(f float64) string {
	s := strconv.FormatFloat(f, 'f', 10, 64)
	s = strings.TrimRight(s, "0")
	if i := strings.IndexByte(s, '.'); len(s)-i-1 < 2 {
		s += strings.Repeat("0", 2-(len(s)-i-1))
	}
	return s
}

var codigoUF = map[string]string{
	"RO": "11", "AC": "12", "AM": "13", "RR": "14", "PA": "15", "AP": "16", "TO": "17",
	"MA": "21", "PI": "22", "CE": "23", "RN": "24", "PB": "25", "PE": "26", "AL": "27", "SE": "28", "BA": "29",
	"MG": "31", "ES": "32", "RJ": "33", "SP": "35", "PR": "41", "SC": "42", "RS": "43",
	"MS": "50", "MT": "51", "GO": "52", "DF": "53",
}

// xmlb escreve o XML na ordem do layout; folha vazia não sai (o schema não
// aceita tag vazia).
type xmlb struct{ strings.Builder }

func (b *xmlb) raw(s string)     { b.WriteString(s) }
func (b *xmlb) open(tag string)  { b.WriteString("<" + tag + ">") }
func (b *xmlb) close(tag string) { b.WriteString("</" + tag + ">") }
func (b *xmlb) leaf(tag, v string) {
	if v = strings.TrimSpace(v); v == "" {
		return
	}
	b.WriteString("<" + tag + ">" + escaparXML(v) + "</" + tag + ">")
}
func (b *xmlb) leafPos(tag string, f float64) {
	if f > 0 {
		b.leaf(tag, v2(f))
	}
}
func (b *xmlb) leafPtrPos(tag string, f *float64) {
	if f != nil && *f > 0 {
		b.leaf(tag, v4(*f))
	}
}

func escaparXML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(s)
}
