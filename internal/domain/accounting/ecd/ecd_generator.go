package ecd

import (
	"fmt"
	"strings"
	"time"
)

func Generate(p ECDParams) string {
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
		"S-001",                    // LEIAUTE (versão)
		p.Empresa.IndSitEsp,        // IND_SIT_ESP
		p.Empresa.IndSitEsp,        // NUM_ORD (reused field placeholder)
		p.Empresa.NumOrd,           // NUM_ORD
		p.Empresa.NomeAudi,         // NOME_AUDI
		dtIni,                      // DT_INI
		dtFin,                      // DT_FIN
		p.Empresa.Nome,             // NOME
		p.Empresa.CNPJ,             // CNPJ
		p.Empresa.CPF,              // CPF
		p.Empresa.UF,               // UF
		p.Empresa.Email,            // EMAIL
		p.Empresa.IE,               // IE
		p.Empresa.CodigoMunicipio,  // COD_MUN
		p.Empresa.NIRE,             // NIRE
		p.Empresa.IndSitAtiv,       // IND_SIT_ATIV
		p.Empresa.IndNireCert,      // IND_NIRE_CERT
		p.Empresa.IndGrandePorte,   // IND_GRANDE_PORTE
		p.Empresa.HashECDSub,       // HASH_ECD_SUB
		p.Empresa.IndEscCons,       // IND_ESC_CONS
		p.Empresa.TipoECD,          // TIPO_ECD
	)

	line("0001", "0")

	for _, pt := range p.Participantes {
		line("0150",
			pt.CodPart,
			pt.Nome,
			pt.CodPais,
			pt.CNPJ,
			pt.CPF,
			pt.TipoPart,
		)
	}

	line("0990", fmt.Sprint(
		counts["0000"]+counts["0001"]+counts["0150"]+1,
	))

	// ── Bloco I ──────────────────────────────────────────────────────────────
	line("I001", "0")

	line("I010",
		dtIni,
		dtFin,
		p.Empresa.Nome,
		p.Empresa.CNPJ+p.Empresa.CPF,
		p.Empresa.Email,
		p.Empresa.UF,
		p.Empresa.CodigoMunicipio,
		p.Empresa.CEP,
		p.Empresa.Endereco,
		p.Empresa.Numero,
		p.Empresa.Complemento,
		p.Empresa.Bairro,
		p.Empresa.Fone,
		"0",
		"",
	)

	for _, cc := range p.CostCenters {
		line("I020", cc.CodCCus, cc.CCus)
	}

	for _, liv := range p.Livros {
		line("I030",
			liv.NumOrd,
			liv.NatLivro,
			liv.NumLiv,
			liv.DescLiv,
			liv.CodHash,
			liv.NumHash,
			fmtDate(liv.PerIni),
			fmtDate(liv.PerFin),
			liv.CodHashAnt,
			liv.NumHashAnt,
		)
	}

	for _, cta := range p.Contas {
		line("I050",
			cta.CodCta,
			cta.CodECD,
			cta.TipoCta,
			fmt.Sprint(cta.Nivel),
			cta.CodCtaSup,
			cta.CtaRef,
			cta.IndCtaCons,
			cta.DescCta,
			cta.Codigo,
			cta.NIF,
		)
	}

	line("I100",
		dtIni,
		dtFin,
		"BRL",
		"1",
		"",
		"",
		"",
	)

	for _, pt := range p.Participantes {
		line("I150",
			pt.CodPart,
			pt.Nome,
			pt.CodPais,
			pt.CNPJ,
			pt.CPF,
			pt.TipoPart,
		)
	}

	for _, lcto := range p.Lancamentos {
		line("I200",
			lcto.NumLcto,
			fmtDate(lcto.DtLcto),
			lcto.CodHist,
			lcto.DescHist,
		)
		for _, pt := range lcto.Partidas {
			line("I250",
				pt.CodCta,
				pt.CodCCus,
				lcto.NumLcto,
				fmtDate(lcto.DtLcto),
				fmtVal(pt.VlLcto),
				pt.IndDC,
				pt.DescHist,
				pt.CodHist,
				pt.NumDoc,
			)
		}
	}

	iTotal := counts["I010"] + counts["I020"] + counts["I030"] + counts["I050"] +
		counts["I100"] + counts["I150"] + counts["I200"] + counts["I250"]
	line("I990", fmt.Sprint(iTotal+2))

	// ── Bloco J ──────────────────────────────────────────────────────────────
	line("J001", "0")

	if len(p.Balancetes) > 0 || len(p.DRE) > 0 {
		line("J005",
			dtIni,
			dtFin,
			"",
			"BRL",
			"",
			fmtDate(p.Periodo.DataFinal),
		)

		for _, bal := range p.Balancetes {
			line("J100",
				bal.CodCtaSup,
				bal.CodCta,
				bal.DescCta,
				fmtVal(bal.VlIni),
				bal.IndDCIni,
				fmtVal(bal.VlFin),
				bal.IndDCFin,
			)
		}

		for _, dre := range p.DRE {
			line("J150",
				dre.CodCtaSup,
				dre.CodCta,
				dre.DescCta,
				dre.TipoDem,
				fmtDate(dre.DtIni),
				fmtDate(dre.DtFin),
				fmtVal(dre.VlCta),
				dre.IndDC,
			)
		}

		line("J930", dtIni, dtFin, "", "", "")
	}

	jTotal := counts["J005"] + counts["J100"] + counts["J150"] + counts["J930"]
	line("J990", fmt.Sprint(jTotal+2))

	// ── Bloco 9 ──────────────────────────────────────────────────────────────
	line("9001", "0")

	for reg, cnt := range counts {
		line("9900", reg, fmt.Sprint(cnt))
	}

	line("9990", fmt.Sprint(counts["9900"]+4))
	line("9999", fmt.Sprint(b.Len()))

	return b.String()
}

func fmtDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02012006")
}

func fmtVal(v float64) string { return fmt.Sprintf("%.2f", v) }
