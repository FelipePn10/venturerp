package sped

import (
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Dados do período lidos das notas do sistema, para montar a EFD sem
// digitação. Valores chegam em decimal e somam em decimal; só viram float64,
// já arredondados em 2 casas, ao preencher os registros do leiaute.

type Participante struct {
	Cod, Nome, CNPJ, CPF, IE, CodMun, End, Num, Compl, Bairro string
}

type ItemCadastro struct {
	Cod, Desc, UnidInv, Tipo, NCM, CEST, CodBarra string
}

type ItemEntrada struct {
	Item       ItemCadastro
	Seq        int
	Desc, Unid string
	Qtd, Fator decimal.Decimal
	// Total é o valor bruto do item (VL_ITEM); ValorContabil o que ele custou
	// na nota (total − desconto + frete + seguro + outras + IPI + ST).
	Total, Desconto, ValorContabil decimal.Decimal
	CST, CFOP                      string // CST do ICMS com a origem (3 dígitos); CFOP de entrada
	BaseICMS, AliqICMS, ICMS       decimal.Decimal
	CreditaICMS                    bool
	BaseST, AliqST, ST             decimal.Decimal
	BaseIPI, AliqIPI, IPI          decimal.Decimal
	CreditaIPI                     bool
	BasePIS, AliqPIS, PIS          decimal.Decimal
	BaseCOFINS, AliqCOFINS, COFINS decimal.Decimal
	CreditaPISCOFINS               bool
	MovEstoque                     bool
}

type NotaEntrada struct {
	Part                                          Participante
	Modelo, Serie, Numero, Chave, IndPgto, IndFrt string
	Emissao, Entrada                              time.Time
	Total, Produtos, Frete, Seguro, Desconto      decimal.Decimal
	Outras, IPI, ST, BaseST, PIS, COFINS          decimal.Decimal
	Itens                                         []ItemEntrada
}

type ItemSaida struct {
	CST, CFOP, CSTIPI        string
	AliqICMS, BaseICMS, ICMS decimal.Decimal
	BaseST, ST, BaseIPI, IPI decimal.Decimal
	Total                    decimal.Decimal
}

type NotaSaida struct {
	Part                                          Participante
	Modelo, Serie, Numero, Chave, IndPgto, IndFrt string
	Emissao, Saida                                time.Time
	Cancelada                                     bool
	Total, Produtos, Frete, Seguro, Desconto      decimal.Decimal
	IPI, ST, BaseST, PIS, COFINS                  decimal.Decimal
	Itens                                         []ItemSaida
}

type Frete struct {
	Part                            Participante
	Serie, Numero, Chave, CST, CFOP string
	Emissao, Lancamento             time.Time
	Valor, Base, Aliq, ICMS         decimal.Decimal
	CreditaICMS                     bool
	MunOrig, MunDest                string
}

// DadosEFD é o período inteiro, como o sistema registrou.
type DadosEFD struct {
	Empresa                 EFDEmpresa
	Periodo                 EFDPeriodo
	Entradas                []NotaEntrada
	Saidas                  []NotaSaida
	Fretes                  []Frete
	SaldoCredorAnteriorICMS decimal.Decimal
	SaldoCredorAnteriorIPI  decimal.Decimal
	CodReceitaICMS          string
	VencimentoICMS          time.Time
	ContribuinteIPI         bool
	Inventario              []EFDInventarioItem
	MotivoInventario        string
	// Itens das saídas (a NF-e própria não tem C170, mas o 0200 é exigido
	// pelos itens do inventário e é útil ao fisco).
	Itens []ItemCadastro
	// AjustesApuracao (E111): as notas especiais de ajuste emitidas no mês.
	AjustesApuracao []AjusteApuracao
	// InventarioItens (bloco H): o estoque próprio numa data (em geral 31/12,
	// entregue na EFD de fevereiro). Vazio = bloco H sem movimento.
	InventarioItens []ItemInventario
	DataInventario  time.Time
	// ContaEstoque (H010/COD_CTA): a conta contábil analítica do estoque.
	ContaEstoque string
}

// ItemInventario é o saldo de um item numa data, com o valor pelo custo.
type ItemInventario struct {
	Item       ItemCadastro
	Quantidade decimal.Decimal
	Valor      decimal.Decimal
}

// AjusteApuracao é um lançamento de ajuste da apuração do ICMS próprio, com o
// código da tabela 5.1.1 da UF (8 caracteres: UF + 0 + tipo + sequência).
type AjusteApuracao struct {
	Codigo, Descricao string
	Valor             decimal.Decimal
}

// TipoAjuste lê o tipo do código 5.1.1: o 3º caractere é a apuração (0 = ICMS
// próprio) e o 4º o tipo (0 outros débitos, 1 estorno de créditos, 2 outros
// créditos, 3 estorno de débitos, 4 deduções, 5 débitos especiais). ok=false
// para código fora do formato ou de outra apuração (ST/DIFAL vão no E220/E311).
func TipoAjuste(codigo string) (tipo byte, ok bool) {
	c := strings.TrimSpace(codigo)
	if len(c) != 8 || c[2] != '0' || c[3] < '0' || c[3] > '5' {
		return 0, false
	}
	return c[3], true
}

func v2(d decimal.Decimal) float64 { return d.Round(2).InexactFloat64() }
func vn(d decimal.Decimal) float64 { return d.InexactFloat64() }

func cfopSaida(c string) bool { return c != "" && (c[0] == '5' || c[0] == '6' || c[0] == '7') }

// grupoC190 acumula em decimal um registro analítico.
type grupoC190 struct {
	cst, cfop               string
	aliq                    decimal.Decimal
	opr, bc, icms, bcST, st decimal.Decimal
	ipi                     decimal.Decimal
}

type analitico struct {
	grupos map[string]*grupoC190
	ordem  []string
}

func novoAnalitico() *analitico { return &analitico{grupos: map[string]*grupoC190{}} }

func (a *analitico) somar(cst, cfop string, aliq, opr, bc, icms, bcST, st, ipi decimal.Decimal) {
	k := cst + "|" + cfop + "|" + aliq.StringFixed(2)
	g := a.grupos[k]
	if g == nil {
		g = &grupoC190{cst: cst, cfop: cfop, aliq: aliq}
		a.grupos[k] = g
		a.ordem = append(a.ordem, k)
	}
	g.opr = g.opr.Add(opr)
	g.bc = g.bc.Add(bc)
	g.icms = g.icms.Add(icms)
	g.bcST = g.bcST.Add(bcST)
	g.st = g.st.Add(st)
	g.ipi = g.ipi.Add(ipi)
}

// fechar arredonda os grupos e faz a soma do VL_OPR fechar no VL_DOC: a
// sobra de arredondamento do rateio (centavos) vai para o maior grupo.
func (a *analitico) fechar(vlDoc decimal.Decimal) []EFDC190 {
	soma := decimal.Zero
	maior := ""
	for _, k := range a.ordem {
		g := a.grupos[k]
		g.opr = g.opr.Round(2)
		soma = soma.Add(g.opr)
		if maior == "" || g.opr.GreaterThan(a.grupos[maior].opr) {
			maior = k
		}
	}
	if maior != "" {
		dif := vlDoc.Round(2).Sub(soma)
		tolerancia := decimal.NewFromFloat(0.01).Mul(decimal.NewFromInt(int64(len(a.ordem))))
		if !dif.IsZero() && dif.Abs().LessThanOrEqual(tolerancia) {
			a.grupos[maior].opr = a.grupos[maior].opr.Add(dif)
		}
	}
	out := make([]EFDC190, 0, len(a.ordem))
	for _, k := range a.ordem {
		g := a.grupos[k]
		out = append(out, EFDC190{CstIcms: g.cst, Cfop: g.cfop, AliqIcms: v2(g.aliq), VlOpr: v2(g.opr), VlBcIcms: v2(g.bc),
			VlIcms: v2(g.icms), VlBcIcmsSt: v2(g.bcST), VlIcmsSt: v2(g.st), VlIpi: v2(g.ipi)})
	}
	return out
}

// Montar transforma o período nos registros da EFD.
//   - Entradas (terceiros): C100 + C170 + C190, com o CFOP de entrada e só o
//     ICMS que a empresa credita (sem crédito, base e imposto zerados).
//   - Saídas (emissão própria): C100 + C190 — NF-e própria não leva C170; a
//     cancelada vai só com a identificação (COD_SIT 02).
//   - Fretes (CT-e): D100 + D190.
//   - E110 a partir dos próprios C190/D190; E116 quando há ICMS a recolher;
//     E500/E510/E520 para o contribuinte do IPI.
func Montar(d DadosEFD) EFDParams {
	p := EFDParams{Empresa: d.Empresa, Periodo: d.Periodo, Inventario: d.Inventario, MotivoInventario: d.MotivoInventario}
	cad := newCadastros()

	debitos, creditos := decimal.Zero, decimal.Zero
	ipi := newApuracaoIPI(d.ContribuinteIPI)

	for _, n := range d.Entradas {
		doc := EFDDocumentoFiscal{IndOper: "0", IndEmit: "1", CodPart: cad.participante(n.Part), CodMod: firstNonBlank(n.Modelo, "55"),
			CodSit: "00", SerDoc: n.Serie, NumDoc: n.Numero, ChvNfe: n.Chave, DtDoc: n.Emissao, DtES: n.Entrada, VlDoc: v2(n.Total),
			IndPgto: firstNonBlank(n.IndPgto, "1"), VlDesc: v2(n.Desconto), VlMerc: v2(n.Produtos), IndFrt: firstNonBlank(n.IndFrt, "9"),
			VlFrt: v2(n.Frete), VlSeg: v2(n.Seguro), VlOutDa: v2(n.Outras), VlBcIcmsSt: v2(n.BaseST), VlIcmsSt: v2(n.ST), VlIpi: v2(n.IPI),
			VlPis: v2(n.PIS), VlCofins: v2(n.COFINS)}
		an := novoAnalitico()
		bcDoc, icmsDoc := decimal.Zero, decimal.Zero
		for i, it := range n.Itens {
			unid := strings.ToUpper(firstNonBlank(it.Unid, it.Item.UnidInv, "UN"))
			cad.item(it.Item, unid, it.Fator)
			bc, aliq, icms := decimal.Zero, decimal.Zero, decimal.Zero
			if it.CreditaICMS {
				bc, aliq, icms = it.BaseICMS, it.AliqICMS, it.ICMS
			}
			// CST de entrada: IPI 00 (com crédito) / 49 (outras); PIS/COFINS 50
			// (com crédito) / 70 (sem crédito).
			cstIPI, cstPC := "49", "70"
			if it.CreditaIPI {
				cstIPI = "00"
			}
			if it.CreditaPISCOFINS {
				cstPC = "50"
			}
			mov := "0"
			if !it.MovEstoque {
				mov = "1"
			}
			doc.Itens = append(doc.Itens, EFDItemDoc{NumItem: primeiroPositivo(it.Seq, i+1), CodItem: it.Item.Cod, DescCompl: it.Desc,
				Qtd: vn(it.Qtd), UnCom: unid, VlUnt: v2(it.Total), VlDesc: v2(it.Desconto), IndMov: mov, CstIcms: it.CST, CfopC170: it.CFOP,
				VlBcIcms: v2(bc), AliqIcms: v2(aliq), VlIcms: v2(icms), VlBcIcmsSt: v2(it.BaseST), AliqSt: v2(it.AliqST), VlIcmsSt: v2(it.ST),
				IndApur: "0", CstIpi: cstIPI, VlBcIpi: v2(it.BaseIPI), AliqIpi: v2(it.AliqIPI), VlIpi: v2(it.IPI),
				CstPis: cstPC, VlBcPis: v2(it.BasePIS), AliqPis: vn(it.AliqPIS), VlPis: v2(it.PIS),
				CstCofins: cstPC, VlBcCofins: v2(it.BaseCOFINS), AliqCofins: vn(it.AliqCOFINS), VlCofins: v2(it.COFINS)})
			an.somar(it.CST, it.CFOP, aliq, it.ValorContabil, bc, icms, it.BaseST, it.ST, it.IPI)
			bcDoc, icmsDoc = bcDoc.Add(bc), icmsDoc.Add(icms)
			ipiCredito := decimal.Zero
			if it.CreditaIPI {
				ipiCredito = it.IPI
			}
			ipi.somar(it.CFOP, cstIPI, it.ValorContabil, it.BaseIPI, ipiCredito, false)
		}
		doc.VlBcIcms, doc.VlIcms = v2(bcDoc), v2(icmsDoc)
		doc.AnaliticosICMS = an.fechar(n.Total)
		for _, g := range an.ordem {
			creditos = creditos.Add(an.grupos[g].icms.Round(2))
		}
		p.DocumentosFiscais = append(p.DocumentosFiscais, doc)
	}

	for _, n := range d.Saidas {
		if n.Cancelada {
			p.DocumentosFiscais = append(p.DocumentosFiscais, EFDDocumentoFiscal{IndOper: "1", IndEmit: "0",
				CodMod: firstNonBlank(n.Modelo, "55"), CodSit: "02", SerDoc: n.Serie, NumDoc: n.Numero, ChvNfe: n.Chave})
			continue
		}
		doc := EFDDocumentoFiscal{IndOper: "1", IndEmit: "0", CodPart: cad.participante(n.Part), CodMod: firstNonBlank(n.Modelo, "55"),
			CodSit: "00", SerDoc: n.Serie, NumDoc: n.Numero, ChvNfe: n.Chave, DtDoc: n.Emissao, DtES: n.Saida, VlDoc: v2(n.Total),
			IndPgto: firstNonBlank(n.IndPgto, "1"), VlDesc: v2(n.Desconto), VlMerc: v2(n.Produtos), IndFrt: firstNonBlank(n.IndFrt, "9"),
			VlFrt: v2(n.Frete), VlSeg: v2(n.Seguro), VlBcIcmsSt: v2(n.BaseST), VlIcmsSt: v2(n.ST), VlIpi: v2(n.IPI),
			VlPis: v2(n.PIS), VlCofins: v2(n.COFINS)}
		// O VL_OPR de cada item leva a sua parte do frete, seguro e desconto da
		// nota (rateados pelo valor do item), para o C190 fechar no VL_DOC.
		somaItens := decimal.Zero
		for _, it := range n.Itens {
			somaItens = somaItens.Add(it.Total)
		}
		acessorias := n.Frete.Add(n.Seguro).Sub(n.Desconto)
		an := novoAnalitico()
		bcDoc, icmsDoc := decimal.Zero, decimal.Zero
		for _, it := range n.Itens {
			parte := decimal.Zero
			if somaItens.IsPositive() {
				parte = acessorias.Mul(it.Total).Div(somaItens)
			}
			opr := it.Total.Add(parte).Add(it.IPI).Add(it.ST)
			an.somar(it.CST, it.CFOP, it.AliqICMS, opr, it.BaseICMS, it.ICMS, it.BaseST, it.ST, it.IPI)
			bcDoc, icmsDoc = bcDoc.Add(it.BaseICMS), icmsDoc.Add(it.ICMS)
			ipi.somar(it.CFOP, firstNonBlank(it.CSTIPI, "50"), opr, it.BaseIPI, it.IPI, cfopSaida(it.CFOP))
		}
		doc.VlBcIcms, doc.VlIcms = v2(bcDoc), v2(icmsDoc)
		doc.AnaliticosICMS = an.fechar(n.Total)
		for _, k := range an.ordem {
			g := an.grupos[k]
			if cfopSaida(g.cfop) {
				debitos = debitos.Add(g.icms.Round(2))
			} else {
				creditos = creditos.Add(g.icms.Round(2))
			}
		}
		p.DocumentosFiscais = append(p.DocumentosFiscais, doc)
	}

	for _, f := range d.Fretes {
		bc, aliq, icms := decimal.Zero, decimal.Zero, decimal.Zero
		if f.CreditaICMS {
			bc, aliq, icms = f.Base, f.Aliq, f.ICMS
		}
		p.Conhecimentos = append(p.Conhecimentos, EFDConhecimento{IndOper: "0", IndEmit: "1", CodPart: cad.participante(f.Part),
			CodMod: "57", CodSit: "00", Ser: f.Serie, NumDoc: f.Numero, ChvCTe: f.Chave, DtDoc: f.Emissao, DtAP: f.Lancamento, TpCTe: "0",
			VlDoc: v2(f.Valor), IndFrt: "1", VlServ: v2(f.Valor), VlBcIcms: v2(bc), VlIcms: v2(icms), VlNt: v2(f.Valor.Sub(bc)),
			CodMunOrig: f.MunOrig, CodMunDest: f.MunDest,
			Analiticos: []EFDD190{{CstIcms: firstNonBlank(f.CST, "000"), Cfop: f.CFOP, AliqIcms: v2(aliq), VlOpr: v2(f.Valor),
				VlBcIcms: v2(bc), VlIcms: v2(icms)}}})
		creditos = creditos.Add(icms.Round(2))
	}

	p.ApuracaoICMS = apurarICMS(d, debitos, creditos)

	p.ApuracaoIPI = ipi.fechar(d.SaldoCredorAnteriorIPI)

	for _, it := range d.Itens {
		cad.item(it, strings.ToUpper(firstNonBlank(it.UnidInv, "UN")), decimal.NewFromInt(1))
	}
	// Bloco H: cada item do inventário precisa do seu 0200.
	for _, inv := range d.InventarioItens {
		if !inv.Quantidade.IsPositive() {
			continue
		}
		unid := strings.ToUpper(firstNonBlank(inv.Item.UnidInv, "UN"))
		cad.item(inv.Item, unid, decimal.NewFromInt(1))
		valor := inv.Valor.Round(2)
		p.Inventario = append(p.Inventario, EFDInventarioItem{DtInv: d.DataInventario, CodItem: inv.Item.Cod, Unid: unid,
			Qtd: vn(inv.Quantidade), VlUnit: valor.Div(inv.Quantidade).Round(6).InexactFloat64(), VlItem: v2(valor), IndProp: "0",
			CodCta: d.ContaEstoque})
	}
	if len(p.Inventario) > 0 && p.MotivoInventario == "" {
		p.MotivoInventario = "01"
	}
	for _, inv := range d.Inventario {
		if inv.Unid != "" {
			cad.unidades[strings.ToUpper(inv.Unid)] = true
		}
	}
	p.Participantes, p.Itens, p.Unidades = cad.listar()
	return p
}

// apurarICMS — E110/E111/E116, na fórmula do Guia Prático:
//
//	saldo = débitos + ajustes a débito + estornos de crédito
//	      − (créditos + ajustes a crédito + estornos de débito + saldo credor anterior)
//
// Saldo devedor menos as deduções é o ICMS a recolher (E116 000); os débitos
// especiais são obrigação à parte (E116 090). Saldo credor é transportado.
func apurarICMS(d DadosEFD, debitos, creditos decimal.Decimal) *EFDApuracaoICMS {
	var ajDeb, estCred, ajCred, estDeb, deducoes, debEsp decimal.Decimal
	var ajustes []EFDApuracaoAjuste
	for _, aj := range d.AjustesApuracao {
		tipo, ok := TipoAjuste(aj.Codigo)
		if !ok || !aj.Valor.IsPositive() {
			continue
		}
		v := aj.Valor.Round(2)
		switch tipo {
		case '0':
			ajDeb = ajDeb.Add(v)
		case '1':
			estCred = estCred.Add(v)
		case '2':
			ajCred = ajCred.Add(v)
		case '3':
			estDeb = estDeb.Add(v)
		case '4':
			deducoes = deducoes.Add(v)
		case '5':
			debEsp = debEsp.Add(v)
		}
		ajustes = append(ajustes, EFDApuracaoAjuste{CodAjApur: strings.TrimSpace(aj.Codigo), DescCompl: aj.Descricao, VlAjApur: v2(v)})
	}
	saldo := debitos.Add(ajDeb).Add(estCred).Sub(creditos.Add(ajCred).Add(estDeb).Add(d.SaldoCredorAnteriorICMS)).Round(2)
	a := &EFDApuracaoICMS{VlTotDebitos: v2(debitos), VlTotAjDebitos: v2(ajDeb), VlEstornosCreditos: v2(estCred),
		VlTotCreditos: v2(creditos), VlTotAjCreditos: v2(ajCred), VlEstornosDebitos: v2(estDeb),
		VlSaldoCredorAnt: v2(d.SaldoCredorAnteriorICMS), DebEspeciais: v2(debEsp), Ajustes: ajustes}
	mesRef := d.Periodo.DataInicial.Format("012006")
	if saldo.IsPositive() {
		ded := decimal.Min(deducoes, saldo)
		recolher := saldo.Sub(ded)
		a.VlApuracao, a.VlTotDed, a.VlIcmsRecolher = v2(saldo), v2(ded), v2(recolher)
		if recolher.IsPositive() {
			a.Obrigacoes = append(a.Obrigacoes, EFDObrigacao{CodOr: "000", VlOr: v2(recolher), DtVcto: d.VencimentoICMS,
				CodRec: d.CodReceitaICMS, MesRef: mesRef})
		}
	} else {
		a.VlSaldoCredorTransp = v2(saldo.Neg())
	}
	if debEsp.IsPositive() {
		a.Obrigacoes = append(a.Obrigacoes, EFDObrigacao{CodOr: "090", VlOr: v2(debEsp), DtVcto: d.VencimentoICMS,
			CodRec: d.CodReceitaICMS, MesRef: mesRef})
	}
	return a
}

type apuracaoIPI struct {
	ativo     bool
	linhas    map[string]*EFDE510
	acum      map[string][3]decimal.Decimal
	deb, cred decimal.Decimal
}

func newApuracaoIPI(ativo bool) *apuracaoIPI {
	return &apuracaoIPI{ativo: ativo, linhas: map[string]*EFDE510{}, acum: map[string][3]decimal.Decimal{}}
}

func (a *apuracaoIPI) somar(cfop, cst string, cont, base, valor decimal.Decimal, debito bool) {
	if !a.ativo {
		return
	}
	k := cfop + "|" + cst
	if _, ok := a.linhas[k]; !ok {
		a.linhas[k] = &EFDE510{Cfop: cfop, CstIpi: cst}
	}
	s := a.acum[k]
	a.acum[k] = [3]decimal.Decimal{s[0].Add(cont), s[1].Add(base), s[2].Add(valor)}
	if debito {
		a.deb = a.deb.Add(valor)
	} else {
		a.cred = a.cred.Add(valor)
	}
}

func (a *apuracaoIPI) fechar(saldoAnt decimal.Decimal) *EFDApuracaoIPI {
	if !a.ativo {
		return nil
	}
	out := &EFDApuracaoIPI{IndApur: "0", SdAnt: v2(saldoAnt), Deb: v2(a.deb), Cred: v2(a.cred)}
	chaves := make([]string, 0, len(a.linhas))
	for k := range a.linhas {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	for _, k := range chaves {
		l := *a.linhas[k]
		s := a.acum[k]
		l.VlCont, l.VlBcIpi, l.VlIpi = v2(s[0]), v2(s[1]), v2(s[2])
		out.Linhas = append(out.Linhas, l)
	}
	return out
}

// cadastros reúne o que os documentos referenciam (0150, 0190, 0200, 0220).
type cadastros struct {
	partes   map[string]EFDParticipante
	unidades map[string]bool
	itens    map[string]*EFDItem
}

func newCadastros() *cadastros {
	return &cadastros{partes: map[string]EFDParticipante{}, unidades: map[string]bool{}, itens: map[string]*EFDItem{}}
}

func (c *cadastros) participante(pt Participante) string {
	if pt.Cod == "" {
		return ""
	}
	if _, ok := c.partes[pt.Cod]; !ok {
		c.partes[pt.Cod] = EFDParticipante{CodPart: pt.Cod, Nome: pt.Nome, CodigoPais: "01058", CNPJ: pt.CNPJ, CPF: pt.CPF, IE: pt.IE,
			CodigoMunicipio: pt.CodMun, Endereco: pt.End, Num: pt.Num, Complemento: pt.Compl, Bairro: pt.Bairro}
	}
	return pt.Cod
}

// item registra o 0200 do item e, quando a unidade do documento difere da de
// estoque, o 0220 com o fator (quantas unidades de estoque a do documento vale).
func (c *cadastros) item(it ItemCadastro, unidDoc string, fator decimal.Decimal) {
	if it.Cod == "" {
		return
	}
	e := c.itens[it.Cod]
	if e == nil {
		unidInv := strings.ToUpper(firstNonBlank(it.UnidInv, unidDoc, "UN"))
		c.unidades[unidInv] = true
		e = &EFDItem{CodItem: it.Cod, DescItem: it.Desc, CodBarra: it.CodBarra, UnCom: unidInv, TipoItem: firstNonBlank(it.Tipo, "99"),
			CodNCM: it.NCM, CEST: it.CEST}
		c.itens[it.Cod] = e
	}
	if unidDoc == "" || unidDoc == e.UnCom {
		return
	}
	c.unidades[unidDoc] = true
	for _, cv := range e.Conversoes {
		if cv.UnidConv == unidDoc {
			return
		}
	}
	if !fator.IsPositive() {
		fator = decimal.NewFromInt(1)
	}
	e.Conversoes = append(e.Conversoes, EFDConversao{UnidConv: unidDoc, FatConv: vn(fator)})
}

func (c *cadastros) listar() ([]EFDParticipante, []EFDItem, []EFDUnidade) {
	var ps []EFDParticipante
	for _, k := range chavesOrdenadas(c.partes) {
		ps = append(ps, c.partes[k])
	}
	var its []EFDItem
	for _, k := range chavesOrdenadas(c.itens) {
		its = append(its, *c.itens[k])
	}
	var us []EFDUnidade
	for _, k := range chavesOrdenadas(c.unidades) {
		us = append(us, EFDUnidade{CodUnd: k, DescUnd: k})
	}
	return ps, its, us
}

func chavesOrdenadas[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func primeiroPositivo(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}
