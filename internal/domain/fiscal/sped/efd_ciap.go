package sped

import (
	"fmt"
	"sort"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/ciap"
)

// CIAP na EFD (Guia Prático, Bloco G e registros 0300, 0305, 0500 e 0600).

// EFDCIAP: o Bloco G do período e os cadastros que ele referencia.
type EFDCIAP struct {
	// G110
	SaldoInicial, SomaParcelas, TribExp, Total, ICMSApropriado float64
	Indice                                                     float64 // 8 casas
	Movimentos                                                 []EFDG125
	Bens                                                       []EFD0300
	Contas                                                     []EFD0500
	Centros                                                    []EFD0600
}

type EFDG125 struct {
	CodBem, TipoMov  string
	DtMov            time.Time
	Op, ST, Frt, Dif float64
	ComValores       bool
	NumParc          int
	VlParc           float64
	Documento        *EFDG130
}

type EFDG130 struct {
	IndEmit, CodPart, CodMod, Serie, NumDoc, Chave string
	DtDoc                                          time.Time
	Itens                                          []EFDG140
}

type EFDG140 struct {
	NumItem          int
	CodItem, Unid    string
	Qtde             float64
	Op, ST, Frt, Dif float64
}

type EFD0300 struct {
	CodBem, IdentMerc, Descr, CodPrnc, CodCta string
	NrParc                                    int
	CodCcus, Func                             string
	VidaUtil                                  int
}

type EFD0500 struct {
	DtAlt         time.Time
	NatCC, IndCta string
	Nivel         int
	CodCta, Nome  string
}

type EFD0600 struct {
	DtAlt         time.Time
	CodCcus, Nome string
}

// DadosCIAP: a apuração do CIAP gravada para o mês e o código de ajuste (5.1.1)
// com que o crédito entra no E111.
type DadosCIAP struct {
	Apuracao     ciap.Apuracao
	CodigoAjuste string
}

// centroPadrao: sem centro de custo no bem, o código que o Guia manda usar
// (industrial: 3 área produtiva; comercial ou serviços: 1 área operacional).
func centroPadrao(indAtiv string) EFD0600 {
	if indAtiv == "1" {
		return EFD0600{CodCcus: "1", Nome: "AREA OPERACIONAL"}
	}
	return EFD0600{CodCcus: "3", Nome: "AREA PRODUTIVA"}
}

// montarCIAP converte a apuração do CIAP em G110/G125/G130/G140 e nos
// cadastros 0300/0305/0500/0600; devolve o ajuste do E111 com o crédito.
func montarCIAP(d DadosEFD, indAtiv string) (*EFDCIAP, *AjusteApuracao) {
	if d.CIAP == nil || len(d.CIAP.Apuracao.Movimentos) == 0 {
		return nil, nil
	}
	a := d.CIAP.Apuracao
	g := &EFDCIAP{SaldoInicial: v2(a.SaldoInicial), SomaParcelas: v2(a.SomaParcelas), TribExp: v2(a.SaidasTribExp), Total: v2(a.SaidasTotal),
		Indice: a.Indice.Round(8).InexactFloat64(), ICMSApropriado: v2(a.ICMSApropriado)}
	bens, contas, centros := map[string]EFD0300{}, map[string]EFD0500{}, map[string]EFD0600{}
	for _, m := range a.Movimentos {
		b := m.Bem
		mov := EFDG125{CodBem: b.Codigo, TipoMov: m.TipoMov, DtMov: m.Data, ComValores: m.ComValores, NumParc: m.NumeroParcela, VlParc: v2(m.ValorParcela)}
		if m.ComValores {
			mov.Op, mov.ST, mov.Frt, mov.Dif = v2(b.ICMSOp), v2(b.ICMSST), v2(b.ICMSFrete), v2(b.ICMSDifal)
		}
		// G130/G140: obrigatórios na imobilização (IM) e na alienação (AT).
		if doc := b.Documento; doc != nil && (m.TipoMov == ciap.MovImobilizacao) {
			mov.Documento = &EFDG130{IndEmit: "1", CodPart: doc.CodPart, CodMod: firstNonBlank(doc.Modelo, "55"), Serie: doc.Serie, NumDoc: doc.Numero,
				Chave: doc.Chave, DtDoc: doc.Emissao,
				Itens: []EFDG140{{NumItem: primeiroPositivo(doc.NumItem, 1), CodItem: doc.CodItem, Unid: firstNonBlank(doc.Unidade, "UN"),
					Qtde: doc.Quantidade.InexactFloat64(), Op: v2(b.ICMSOp), ST: v2(b.ICMSST), Frt: v2(b.ICMSFrete), Dif: v2(b.ICMSDifal)}}}
		}
		g.Movimentos = append(g.Movimentos, mov)
		centro := EFD0600{CodCcus: b.CentroCodigo, Nome: b.CentroNome}
		if centro.CodCcus == "" {
			centro = centroPadrao(indAtiv)
		}
		centro.DtAlt = d.Periodo.DataInicial
		centros[centro.CodCcus] = centro
		if b.ContaCodigo != "" {
			contas[b.ContaCodigo] = EFD0500{DtAlt: d.Periodo.DataInicial, NatCC: "01", IndCta: "A", Nivel: primeiroPositivo(b.ContaNivel, 1),
				CodCta: b.ContaCodigo, Nome: b.ContaNome}
		}
		bens[b.Codigo] = EFD0300{CodBem: b.Codigo, IdentMerc: "1", Descr: b.Descricao, CodCta: b.ContaCodigo, NrParc: b.NumeroParcelas,
			CodCcus: centro.CodCcus, Func: firstNonBlank(b.Funcao, b.Descricao), VidaUtil: b.VidaUtilMeses}
	}
	for _, k := range chavesOrdenadas(bens) {
		g.Bens = append(g.Bens, bens[k])
	}
	for _, k := range chavesOrdenadas(contas) {
		g.Contas = append(g.Contas, contas[k])
	}
	for _, k := range chavesOrdenadas(centros) {
		g.Centros = append(g.Centros, centros[k])
	}
	// G125 na ordem do bem, a entrada antes da saída.
	ordemTipo := map[string]int{ciap.MovSaldoInicial: 0, ciap.MovImobilizacao: 0}
	sort.SliceStable(g.Movimentos, func(i, j int) bool {
		if g.Movimentos[i].CodBem != g.Movimentos[j].CodBem {
			return g.Movimentos[i].CodBem < g.Movimentos[j].CodBem
		}
		oi, oki := ordemTipo[g.Movimentos[i].TipoMov]
		oj, okj := ordemTipo[g.Movimentos[j].TipoMov]
		if !oki {
			oi = 1
		}
		if !okj {
			oj = 1
		}
		return oi < oj
	})
	var aj *AjusteApuracao
	if d.CIAP.CodigoAjuste != "" && a.ICMSApropriado.IsPositive() {
		aj = &AjusteApuracao{Codigo: d.CIAP.CodigoAjuste, Valor: a.ICMSApropriado,
			Descricao: fmt.Sprintf("CREDITO CIAP %02d/%d - SOMA DAS PARCELAS %s X INDICE %s", a.Mes, a.Ano, fmtVal(v2(a.SomaParcelas)), fmtNum(a.Indice.InexactFloat64(), 8))}
	}
	return g, aj
}

func optVal(v float64, informar bool) string {
	if !informar {
		return ""
	}
	return fmtVal(v)
}

// linhas0CIAP: 0300/0305 (depois do 0200/0220), 0500 e 0600.
func linhas0CIAP(line func(string, ...string), g *EFDCIAP) {
	if g == nil {
		return
	}
	for _, b := range g.Bens {
		nr := ""
		if b.NrParc > 0 {
			nr = fmt.Sprint(b.NrParc)
		}
		line("0300", b.CodBem, b.IdentMerc, b.Descr, b.CodPrnc, b.CodCta, nr)
		vida := ""
		if b.VidaUtil > 0 {
			vida = fmt.Sprint(b.VidaUtil)
		}
		line("0305", b.CodCcus, b.Func, vida)
	}
}

func linhas0500e0600(line func(string, ...string), g *EFDCIAP) {
	if g == nil {
		return
	}
	for _, c := range g.Contas {
		line("0500", fmtDate(c.DtAlt), c.NatCC, c.IndCta, fmt.Sprint(c.Nivel), c.CodCta, c.Nome)
	}
	for _, c := range g.Centros {
		line("0600", fmtDate(c.DtAlt), c.CodCcus, c.Nome)
	}
}

// linhasG escreve o Bloco G (sem CIAP, G001 = 1).
func linhasG(line func(string, ...string), dtIni, dtFin string, g *EFDCIAP) {
	if g == nil || len(g.Movimentos) == 0 {
		line("G001", "1")
		return
	}
	line("G001", "0")
	line("G110", dtIni, dtFin, fmtVal(g.SaldoInicial), fmtVal(g.SomaParcelas), fmtVal(g.TribExp), fmtVal(g.Total),
		fmtNum(g.Indice, 8), fmtVal(g.ICMSApropriado), fmtVal(0))
	for _, m := range g.Movimentos {
		parc, vl := "", ""
		if m.NumParc > 0 {
			parc, vl = fmt.Sprint(m.NumParc), fmtVal(m.VlParc)
		}
		line("G125", m.CodBem, fmtDate(m.DtMov), m.TipoMov, optVal(m.Op, m.ComValores), optVal(m.ST, m.ComValores),
			optVal(m.Frt, m.ComValores), optVal(m.Dif, m.ComValores), parc, vl)
		if d := m.Documento; d != nil {
			line("G130", d.IndEmit, d.CodPart, d.CodMod, d.Serie, d.NumDoc, d.Chave, fmtDate(d.DtDoc), "")
			for _, it := range d.Itens {
				line("G140", fmt.Sprint(it.NumItem), it.CodItem, fmtQtd(it.Qtde), it.Unid, fmtVal(it.Op), fmtVal(it.ST), fmtVal(it.Frt), fmtVal(it.Dif))
			}
		}
	}
}
