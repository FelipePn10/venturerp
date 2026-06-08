package sped

import (
	"fmt"
	"strings"
)

// Generate builds the SPED EFD ICMS/IPI text and returns it as a string.
// The caller is responsible for writing it to a file or HTTP response.
func Generate(p EFDParams) string {
	var b strings.Builder
	counts := make(map[string]int)

	line := func(reg string, fields ...string) {
		row := "|" + reg + "|" + strings.Join(fields, "|") + "|\n"
		b.WriteString(row)
		counts[reg]++
	}

	dtIni := fmtDate(p.Periodo.DataInicial)
	dtFin := fmtDate(p.Periodo.DataFinal)

	// ── Bloco 0 ──────────────────────────────────────────────────────────────
	line("0000",
		"010",                               // COD_VER
		p.Periodo.IndicadorSituacaoEspecial, // COD_FIN
		dtIni,                               // DT_INI
		dtFin,                               // DT_FIN
		p.Empresa.Nome,                      // NOME
		p.Empresa.CNPJ,                      // CNPJ
		"",                                  // CPF
		p.Empresa.UF,                        // UF
		p.Empresa.IE,                        // IE
		p.Empresa.CodigoMunicipio,           // COD_MUN
		p.Empresa.IM,                        // IM
		p.Empresa.SUFRAMA,                   // SUFRAMA
		p.Empresa.RegimeTributario,          // IND_PERFIL (A=completo, B=simplificado, C=micro)
		p.Empresa.CodigoFinalizacao,         // IND_ATIV
	)

	line("0001", "0") // abertura bloco 0

	line("0005",
		"SPED EFD",     // DTEC
		"",             // NIRE
		p.Empresa.CNPJ, // CNPJ
		p.Empresa.Nome, // NAME
		"",             // END
		"",             // NUM
		"",             // COMPL
		"",             // BAIRRO
		p.Empresa.CodigoMunicipio,
		p.Empresa.UF,
		"", // CEP
		"", // TEL
		"", // FAX
		"", // EMAIL
	)

	if p.Empresa.ContabilistaNome != "" {
		line("0100",
			p.Empresa.ContabilistaNome,
			p.Empresa.ContabilistaCPF,
			p.Empresa.ContabilistaCRC,
			"",
			"",
			"",
			"",
			"",
			p.Empresa.ContabilistaCNPJ,
			"",
		)
	}

	for _, pt := range p.Participantes {
		line("0150",
			pt.CodPart,
			pt.Nome,
			pt.CodigoPais,
			pt.CNPJ,
			pt.CPF,
			pt.IE,
			pt.CodigoMunicipio,
			pt.SUFRAMA,
			pt.Endereco,
			pt.Num,
			pt.Complemento,
			pt.Bairro,
			pt.CEP,
			pt.Telefone,
		)
	}

	for _, u := range p.Unidades {
		line("0190", u.CodUnd, u.DescUnd)
	}

	for _, it := range p.Itens {
		line("0200",
			it.CodItem,
			it.DescItem,
			it.CodBarra,
			it.CodAnt,
			it.UnCom,
			it.TipoItem,
			it.CodNCM,
			it.ExIPI,
			it.CodGen,
			it.CodLST,
			fmtAliq(it.AliqICMS),
		)
	}

	line("0990", fmt.Sprint(counts["0000"]+counts["0001"]+counts["0005"]+
		counts["0100"]+counts["0150"]+counts["0190"]+counts["0200"]+1))

	// ── Bloco C ──────────────────────────────────────────────────────────────
	line("C001", "0")

	for _, doc := range p.DocumentosFiscais {
		line("C100",
			doc.IndOper,
			doc.IndEmit,
			doc.CodPart,
			doc.CodMod,
			doc.CodSit,
			doc.SerDoc,
			doc.NumDoc,
			doc.ChvNfe,
			fmtDate(doc.DtDoc),
			fmtDate(doc.DtES),
			fmtVal(doc.VlDoc),
			doc.IndPgto,
			fmtVal(doc.VlDesc),
			fmtVal(doc.VlAbatNt),
			fmtVal(doc.VlMerc),
			doc.IndFrt,
			fmtVal(doc.VlFrt),
			fmtVal(doc.VlSeg),
			fmtVal(doc.VlOutDa),
			fmtVal(doc.VlBcIcms),
			fmtVal(doc.VlIcms),
			fmtVal(doc.VlBcIcmsSt),
			fmtVal(doc.VlIcmsSt),
			fmtVal(doc.VlIpi),
			fmtVal(doc.VlPis),
			fmtVal(doc.VlCofins),
			fmtVal(doc.VlPisSt),
			fmtVal(doc.VlCofinsSt),
		)
		for _, it := range doc.Itens {
			line("C170",
				fmt.Sprint(it.NumItem),
				it.CodItem,
				it.DescCompl,
				fmtQtd(it.Qtd),
				it.UnCom,
				fmtVal(it.VlUnt),
				fmtVal(it.VlDesc),
				it.IndMov,
				it.CstIcms,
				it.CfopC170,
				it.CodNat,
				fmtVal(it.VlBcIcms),
				fmtAliq(it.AliqIcms),
				fmtVal(it.VlIcms),
				fmtVal(it.VlBcIcmsSt),
				fmtAliq(it.AliqSt),
				fmtVal(it.VlIcmsSt),
				it.IndApur,
				it.CstIpi,
				it.CodEnq,
				fmtVal(it.VlBcIpi),
				fmtAliq(it.AliqIpi),
				fmtVal(it.VlIpi),
				it.CstPis,
				fmtVal(it.VlBcPis),
				fmtAliq(it.AliqPis),
				fmtQtd(it.QtdBcPis),
				fmtAliq(it.AliqPisQ),
				fmtVal(it.VlPis),
				it.CstCofins,
				fmtVal(it.VlBcCofins),
				fmtAliq(it.AliqCofins),
				fmtQtd(it.QtdBcCofins),
				fmtAliq(it.AliqCofinsQ),
				fmtVal(it.VlCofins),
				it.CodCta,
				fmtVal(it.VlAbatNt),
			)
		}
		for _, an := range doc.AnaliticosICMS {
			line("C190",
				an.CstIcms,
				an.Cfop,
				fmtAliq(an.AliqIcms),
				fmtVal(an.VlOpr),
				fmtVal(an.VlBcIcms),
				fmtVal(an.VlIcms),
				fmtVal(an.VlBcIcmsSt),
				fmtVal(an.VlIcmsSt),
				fmtVal(an.VlRedBc),
				fmtVal(an.VlIpi),
				an.CodObs,
			)
		}
	}

	cTotal := counts["C100"] + counts["C170"] + counts["C190"]
	line("C990", fmt.Sprint(cTotal+2)) // +2 for C001 and C990

	// ── Bloco E ──────────────────────────────────────────────────────────────
	line("E001", "0")

	if a := p.ApuracaoICMS; a != nil {
		line("E110",
			fmtVal(a.VlTotDebitos),
			fmtVal(a.VlAjDebitos),
			fmtVal(a.VlTotAjDebitos),
			fmtVal(a.VlEstornosCreditos),
			fmtVal(a.VlTotCreditos),
			fmtVal(a.VlAjCreditos),
			fmtVal(a.VlTotAjCreditos),
			fmtVal(a.VlEstornosDebitos),
			fmtVal(a.VlSaldoCredorAnt),
			fmtVal(a.VlApuracao),
			fmtVal(a.VlTotDed),
			fmtVal(a.VlIcmsRecolher),
			fmtVal(a.VlSaldoCredorTransp),
			fmtVal(a.DebEspeciais),
		)
		for _, aj := range a.Ajustes {
			line("E111", aj.CodAjApur, aj.DescCompl, fmtVal(aj.VlAjApur))
		}
	}

	eTotal := counts["E110"] + counts["E111"]
	line("E990", fmt.Sprint(eTotal+2))

	// ── Bloco H ──────────────────────────────────────────────────────────────
	line("H001", "0")

	for _, inv := range p.Inventario {
		line("H010",
			fmtDate(inv.DtInv),
			inv.CodItem,
			inv.Unid,
			fmtQtd(inv.Qtd),
			fmtVal(inv.VlUnit),
			fmtVal(inv.VlItem),
			inv.IndProp,
			inv.CodPart,
			inv.TxtCompl,
			inv.CodCta,
			fmtVal(inv.VlItemIr),
		)
	}

	line("H990", fmt.Sprint(counts["H010"]+2))

	// ── Bloco 9 ──────────────────────────────────────────────────────────────
	line("9001", "0")

	// 9900: one line per registro type with total count
	for reg, cnt := range counts {
		line("9900", reg, fmt.Sprint(cnt))
	}
	// account for 9001, 9900 lines themselves, and 9990+9999
	line("9990", fmt.Sprint(counts["9900"]+4))
	line("9999", fmt.Sprint(b.Len())) // approximate — field is total lines

	return b.String()
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

func fmtVal(v float64) string  { return fmt.Sprintf("%.2f", v) }
func fmtAliq(v float64) string { return fmt.Sprintf("%.2f", v) }
func fmtQtd(v float64) string  { return fmt.Sprintf("%.4f", v) }
