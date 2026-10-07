package sped

import (
	"fmt"
	"strings"
	"time"
)

// CodVer é a versão do leiaute da EFD ICMS/IPI pelo ano do período (Ato
// COTEPE/ICMS 44/2018 e atualizações). Confira no PVA a versão vigente.
func CodVer(dtIni time.Time) string {
	switch y := dtIni.Year(); {
	case y >= 2026:
		return "020"
	case y == 2025:
		return "019"
	case y == 2024:
		return "018"
	case y == 2023:
		return "017"
	default:
		return "016"
	}
}

// Generate monta o arquivo da EFD ICMS/IPI no leiaute do Guia Prático: blocos
// 0, B, C, D, E, G, H, K, 1 e 9 (os sem dados abertos com indicador 1), cada
// registro com a sua quantidade exata de campos, e as contagens de linhas
// (x990, 9900, 9999) calculadas sobre o que foi escrito.
func Generate(p EFDParams) string {
	var b strings.Builder
	counts := map[string]int{}
	ordem := []string{}
	total := 0
	line := func(reg string, fields ...string) {
		for i, f := range fields {
			// O separador do arquivo não pode aparecer dentro de um campo.
			fields[i] = strings.ReplaceAll(strings.ReplaceAll(f, "|", " "), "\n", " ")
		}
		b.WriteString("|" + reg + "|" + strings.Join(fields, "|") + "|\r\n")
		if counts[reg] == 0 {
			ordem = append(ordem, reg)
		}
		counts[reg]++
		total++
	}
	bloco := func(prefixo string) int {
		n := 0
		for reg, c := range counts {
			if strings.HasPrefix(reg, prefixo) {
				n += c
			}
		}
		return n
	}

	dtIni, dtFin := fmtDate(p.Periodo.DataInicial), fmtDate(p.Periodo.DataFinal)
	perfil := p.Empresa.RegimeTributario
	switch perfil {
	case "A", "B", "C":
	default:
		perfil = "A"
	}
	// IND_ATIV: 0 industrial ou equiparado, 1 outros.
	indAtiv := p.Empresa.IndAtividade
	if indAtiv != "1" {
		indAtiv = "0"
	}

	// ── Bloco 0 ──────────────────────────────────────────────────────────────
	line("0000", CodVer(p.Periodo.DataInicial), firstNonBlank(p.Periodo.IndicadorSituacaoEspecial, "0"), dtIni, dtFin,
		p.Empresa.Nome, p.Empresa.CNPJ, "", p.Empresa.UF, p.Empresa.IE, p.Empresa.CodigoMunicipio, p.Empresa.IM,
		p.Empresa.SUFRAMA, perfil, indAtiv)
	line("0001", "0")
	line("0005", p.Empresa.Fantasia, p.Empresa.CEP, p.Empresa.Endereco, p.Empresa.Numero, p.Empresa.Complemento,
		p.Empresa.Bairro, p.Empresa.Fone, "", p.Empresa.Email)
	if p.Empresa.ContabilistaNome != "" {
		line("0100", p.Empresa.ContabilistaNome, p.Empresa.ContabilistaCPF, p.Empresa.ContabilistaCRC, p.Empresa.ContabilistaCNPJ,
			"", "", "", "", "", "", "", "", "")
	}
	for _, pt := range p.Participantes {
		line("0150", pt.CodPart, pt.Nome, firstNonBlank(pt.CodigoPais, "01058"), pt.CNPJ, pt.CPF, pt.IE, pt.CodigoMunicipio,
			pt.SUFRAMA, pt.Endereco, pt.Num, pt.Complemento, pt.Bairro)
	}
	for _, u := range p.Unidades {
		line("0190", u.CodUnd, u.DescUnd)
	}
	for _, it := range p.Itens {
		aliq := ""
		if it.AliqICMS > 0 {
			aliq = fmtAliq(it.AliqICMS)
		}
		line("0200", it.CodItem, it.DescItem, it.CodBarra, it.CodAnt, it.UnCom, it.TipoItem, it.CodNCM, it.ExIPI,
			it.CodGen, it.CodLST, aliq, it.CEST)
		for _, cv := range it.Conversoes {
			line("0220", cv.UnidConv, fmtFator(cv.FatConv), "")
		}
	}
	line("0990", fmt.Sprint(bloco("0")+1))

	// ── Bloco B (ISS do DF): sem dados ───────────────────────────────────────
	line("B001", "1")
	line("B990", "2")

	// ── Bloco C ──────────────────────────────────────────────────────────────
	if len(p.DocumentosFiscais) == 0 {
		line("C001", "1")
	} else {
		line("C001", "0")
	}
	for _, doc := range p.DocumentosFiscais {
		if doc.CodSit == "02" || doc.CodSit == "03" || doc.CodSit == "04" || doc.CodSit == "05" {
			// Cancelado/denegado/inutilizado: só a identificação do documento.
			line("C100", doc.IndOper, doc.IndEmit, "", doc.CodMod, doc.CodSit, doc.SerDoc, doc.NumDoc, doc.ChvNfe,
				"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "")
			continue
		}
		line("C100", doc.IndOper, doc.IndEmit, doc.CodPart, doc.CodMod, doc.CodSit, doc.SerDoc, doc.NumDoc, doc.ChvNfe,
			fmtDate(doc.DtDoc), fmtDate(doc.DtES), fmtVal(doc.VlDoc), doc.IndPgto, fmtVal(doc.VlDesc), fmtVal(doc.VlAbatNt),
			fmtVal(doc.VlMerc), firstNonBlank(doc.IndFrt, "9"), fmtVal(doc.VlFrt), fmtVal(doc.VlSeg), fmtVal(doc.VlOutDa),
			fmtVal(doc.VlBcIcms), fmtVal(doc.VlIcms), fmtVal(doc.VlBcIcmsSt), fmtVal(doc.VlIcmsSt), fmtVal(doc.VlIpi),
			fmtVal(doc.VlPis), fmtVal(doc.VlCofins), fmtVal(doc.VlPisSt), fmtVal(doc.VlCofinsSt))
		for _, it := range doc.Itens {
			line("C170", fmt.Sprint(it.NumItem), it.CodItem, it.DescCompl, fmtQtd(it.Qtd), it.UnCom, fmtVal(it.VlUnt),
				fmtVal(it.VlDesc), firstNonBlank(it.IndMov, "0"), it.CstIcms, it.CfopC170, it.CodNat, fmtVal(it.VlBcIcms),
				fmtAliq(it.AliqIcms), fmtVal(it.VlIcms), fmtVal(it.VlBcIcmsSt), fmtAliq(it.AliqSt), fmtVal(it.VlIcmsSt),
				firstNonBlank(it.IndApur, "0"), it.CstIpi, it.CodEnq, fmtVal(it.VlBcIpi), fmtAliq(it.AliqIpi), fmtVal(it.VlIpi),
				it.CstPis, fmtVal(it.VlBcPis), fmtAliq(it.AliqPis), fmtQtdOpt(it.QtdBcPis), fmtAliqOpt(it.AliqPisQ), fmtVal(it.VlPis),
				it.CstCofins, fmtVal(it.VlBcCofins), fmtAliq(it.AliqCofins), fmtQtdOpt(it.QtdBcCofins), fmtAliqOpt(it.AliqCofinsQ),
				fmtVal(it.VlCofins), it.CodCta, fmtVal(it.VlAbatNt))
		}
		for _, an := range doc.AnaliticosICMS {
			line("C190", an.CstIcms, an.Cfop, fmtAliq(an.AliqIcms), fmtVal(an.VlOpr), fmtVal(an.VlBcIcms), fmtVal(an.VlIcms),
				fmtVal(an.VlBcIcmsSt), fmtVal(an.VlIcmsSt), fmtVal(an.VlRedBc), fmtVal(an.VlIpi), an.CodObs)
		}
	}
	line("C990", fmt.Sprint(bloco("C")+1))

	// ── Bloco D ──────────────────────────────────────────────────────────────
	if len(p.Conhecimentos) == 0 {
		line("D001", "1")
	} else {
		line("D001", "0")
	}
	for _, d := range p.Conhecimentos {
		line("D100", d.IndOper, d.IndEmit, d.CodPart, firstNonBlank(d.CodMod, "57"), firstNonBlank(d.CodSit, "00"), d.Ser, "", d.NumDoc,
			d.ChvCTe, fmtDate(d.DtDoc), fmtDate(d.DtAP), firstNonBlank(d.TpCTe, "0"), "", fmtVal(d.VlDoc), fmtVal(0),
			firstNonBlank(d.IndFrt, "1"), fmtVal(d.VlServ), fmtVal(d.VlBcIcms), fmtVal(d.VlIcms), fmtVal(d.VlNt), "", "",
			d.CodMunOrig, d.CodMunDest)
		for _, a := range d.Analiticos {
			line("D190", a.CstIcms, a.Cfop, fmtAliq(a.AliqIcms), fmtVal(a.VlOpr), fmtVal(a.VlBcIcms), fmtVal(a.VlIcms), fmtVal(a.VlRedBc), "")
		}
	}
	line("D990", fmt.Sprint(bloco("D")+1))

	// ── Bloco E ──────────────────────────────────────────────────────────────
	line("E001", "0")
	line("E100", dtIni, dtFin)
	a := p.ApuracaoICMS
	if a == nil {
		a = &EFDApuracaoICMS{}
	}
	line("E110", fmtVal(a.VlTotDebitos), fmtVal(a.VlAjDebitos), fmtVal(a.VlTotAjDebitos), fmtVal(a.VlEstornosCreditos),
		fmtVal(a.VlTotCreditos), fmtVal(a.VlAjCreditos), fmtVal(a.VlTotAjCreditos), fmtVal(a.VlEstornosDebitos),
		fmtVal(a.VlSaldoCredorAnt), fmtVal(a.VlApuracao), fmtVal(a.VlTotDed), fmtVal(a.VlIcmsRecolher),
		fmtVal(a.VlSaldoCredorTransp), fmtVal(a.DebEspeciais))
	for _, aj := range a.Ajustes {
		line("E111", aj.CodAjApur, aj.DescCompl, fmtVal(aj.VlAjApur))
	}
	for _, o := range a.Obrigacoes {
		line("E116", firstNonBlank(o.CodOr, "000"), fmtVal(o.VlOr), fmtDate(o.DtVcto), o.CodRec, "", "", "", o.TxtCompl, o.MesRef)
	}
	if ipi := p.ApuracaoIPI; ipi != nil {
		line("E500", firstNonBlank(ipi.IndApur, "0"), dtIni, dtFin)
		for _, l := range ipi.Linhas {
			line("E510", l.Cfop, l.CstIpi, fmtVal(l.VlCont), fmtVal(l.VlBcIpi), fmtVal(l.VlIpi))
		}
		saldo := ipi.SdAnt + ipi.Cred + ipi.Oc - ipi.Deb - ipi.Od
		sc, sd := 0.0, 0.0
		if saldo >= 0 {
			sc = saldo
		} else {
			sd = -saldo
		}
		line("E520", fmtVal(ipi.SdAnt), fmtVal(ipi.Deb), fmtVal(ipi.Cred), fmtVal(ipi.Od), fmtVal(ipi.Oc), fmtVal(sc), fmtVal(sd))
	}
	line("E990", fmt.Sprint(bloco("E")+1))

	// ── Bloco G (CIAP): sem dados ────────────────────────────────────────────
	line("G001", "1")
	line("G990", "2")

	// ── Bloco H ──────────────────────────────────────────────────────────────
	if len(p.Inventario) == 0 {
		line("H001", "1")
	} else {
		line("H001", "0")
		var vlInv float64
		for _, inv := range p.Inventario {
			vlInv += inv.VlItem
		}
		line("H005", fmtDate(p.Inventario[0].DtInv), fmtVal(vlInv), firstNonBlank(p.MotivoInventario, "01"))
		for _, inv := range p.Inventario {
			line("H010", inv.CodItem, inv.Unid, fmtQtd(inv.Qtd), fmtUnit(inv.VlUnit), fmtVal(inv.VlItem),
				firstNonBlank(inv.IndProp, "0"), inv.CodPart, inv.TxtCompl, inv.CodCta, fmtVal(inv.VlItemIr))
		}
	}
	line("H990", fmt.Sprint(bloco("H")+1))

	// ── Bloco K (produção e estoque): sem dados ──────────────────────────────
	line("K001", "1")
	line("K990", "2")

	// ── Bloco 1 ──────────────────────────────────────────────────────────────
	line("1001", "0")
	line("1010", "N", "N", "N", "N", "N", "N", "N", "N", "N", "N", "N", "N", "N")
	line("1990", fmt.Sprint(bloco("1")+1))

	// ── Bloco 9 ──────────────────────────────────────────────────────────────
	line("9001", "0")
	regs := append([]string(nil), ordem...)
	regs = append(regs, "9900", "9990", "9999")
	// 9900 lista cada registro do arquivo, inclusive 9900, 9990 e 9999.
	n9900 := len(regs)
	for _, reg := range regs {
		qtd := counts[reg]
		switch reg {
		case "9900":
			qtd = n9900
		case "9990", "9999":
			qtd = 1
		}
		line("9900", reg, fmt.Sprint(qtd))
	}
	line("9990", fmt.Sprint(bloco("9")+2)) // 9001 + 9900s + 9990 + 9999
	line("9999", fmt.Sprint(total+1))
	return b.String()
}

func firstNonBlank(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func fmtDate(t timeVal) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02012006")
}

// timeVal is the minimal interface satisfied by time.Time.
type timeVal interface {
	IsZero() bool
	Format(string) string
}

// Números no padrão do leiaute: vírgula decimal, sem separador de milhar.
func fmtNum(v float64, casas int) string {
	return strings.Replace(fmt.Sprintf("%.*f", casas, v), ".", ",", 1)
}

func fmtVal(v float64) string   { return fmtNum(v, 2) }
func fmtAliq(v float64) string  { return fmtNum(v, 2) }
func fmtQtd(v float64) string   { return fmtNum(v, 5) }
func fmtUnit(v float64) string  { return fmtNum(v, 6) }
func fmtFator(v float64) string { return fmtNum(v, 6) }

func fmtQtdOpt(v float64) string {
	if v == 0 {
		return ""
	}
	return fmtNum(v, 3)
}

func fmtAliqOpt(v float64) string {
	if v == 0 {
		return ""
	}
	return fmtNum(v, 4)
}
