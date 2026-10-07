package fiscal_uc

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// NFeLida é a NF-e de entrada como veio no XML do fornecedor, já com os
// valores em decimal. É dela que nascem o cabeçalho, os itens e as parcelas da
// entrada — o usuário não redigita nada que a nota já diz.
type NFeLida struct {
	ChaveAcesso      string
	Protocolo        string
	Numero           string
	Serie            string
	Modelo           string
	NaturezaOperacao string
	DataEmissao      time.Time
	DataSaidaEntrada *time.Time

	EmitenteCNPJ      string
	EmitenteNome      string
	EmitenteFantasia  string
	EmitenteIE        string
	EmitenteUF        string
	EmitenteMunicipio string
	// Endereço do emitente (o destinatário de uma devolução de compra).
	EmitenteLogradouro, EmitenteNumero, EmitenteComplemento, EmitenteBairro string
	EmitenteCodigoMunicipio, EmitenteCEP, EmitenteFone                      string

	DestinatarioCNPJ string

	ValorProdutos decimal.Decimal
	ValorFrete    decimal.Decimal
	ValorSeguro   decimal.Decimal
	ValorDesconto decimal.Decimal
	ValorOutras   decimal.Decimal
	ValorIPI      decimal.Decimal
	ValorICMS     decimal.Decimal
	ValorICMSST   decimal.Decimal
	ValorPIS      decimal.Decimal
	ValorCOFINS   decimal.Decimal
	ValorTotal    decimal.Decimal

	ModalidadeFrete           string
	InformacoesComplementares string

	// Reforma Tributária (NT 2025.002): totais de IBS, CBS e Imposto Seletivo.
	BaseIBSCBS decimal.Decimal
	ValorIBS   decimal.Decimal
	ValorCBS   decimal.Decimal
	ValorIS    decimal.Decimal

	// Retenções na fonte declaradas na nota (<retTrib>, <ISSQNtot>): o
	// fornecedor recebe o líquido e a empresa recolhe a diferença.
	ValorRetPIS    decimal.Decimal
	ValorRetCOFINS decimal.Decimal
	ValorRetCSLL   decimal.Decimal
	BaseIRRF       decimal.Decimal
	ValorIRRF      decimal.Decimal
	BaseRetPrev    decimal.Decimal
	ValorRetPrev   decimal.Decimal
	ValorISSRet    decimal.Decimal

	Itens      []NFeItemLido
	Duplicatas []NFeDuplicata
	Pagamentos []NFePagamento
}

type NFeItemLido struct {
	Numero        int
	CodigoProduto string
	EAN           string
	Descricao     string
	NCM           string
	CEST          string
	CFOP          string
	Unidade       string
	Quantidade    decimal.Decimal
	ValorUnitario decimal.Decimal
	ValorProduto  decimal.Decimal
	ValorFrete    decimal.Decimal
	ValorSeguro   decimal.Decimal
	ValorDesconto decimal.Decimal
	ValorOutras   decimal.Decimal
	PedidoCompra  string
	ItemPedido    string

	Origem    string
	CSTICMS   string // CST (regime normal) ou CSOSN (Simples Nacional)
	BaseICMS  decimal.Decimal
	AliqICMS  decimal.Decimal // percentual, como no XML (18.00)
	ValorICMS decimal.Decimal
	BaseST    decimal.Decimal
	ValorST   decimal.Decimal

	CSTIPI   string
	BaseIPI  decimal.Decimal
	AliqIPI  decimal.Decimal
	ValorIPI decimal.Decimal

	CSTPIS      string
	ValorPIS    decimal.Decimal
	CSTCOFINS   string
	ValorCOFINS decimal.Decimal

	CSTIBSCBS   string
	ClassTrib   string
	BaseIBSCBS  decimal.Decimal
	AliqIBSUF   decimal.Decimal
	ValorIBSUF  decimal.Decimal
	AliqIBSMun  decimal.Decimal
	ValorIBSMun decimal.Decimal
	ValorIBS    decimal.Decimal
	AliqCBS     decimal.Decimal
	ValorCBS    decimal.Decimal
	ValorIS     decimal.Decimal
}

// TotalRetencoes é o que a empresa retém e recolhe no lugar do fornecedor.
func (n *NFeLida) TotalRetencoes() decimal.Decimal {
	return n.ValorRetPIS.Add(n.ValorRetCOFINS).Add(n.ValorRetCSLL).Add(n.ValorIRRF).Add(n.ValorRetPrev).Add(n.ValorISSRet)
}

// ValorContabil é quanto o item custa de fato na nota: o que vai para o plano
// de contas e o que o contas a pagar rateia. ICMS próprio, PIS e COFINS não
// somam — já estão dentro do preço; IPI e ICMS-ST são cobrados por fora.
func (i NFeItemLido) ValorContabil() decimal.Decimal {
	return i.ValorProduto.Add(i.ValorFrete).Add(i.ValorSeguro).Add(i.ValorOutras).
		Sub(i.ValorDesconto).Add(i.ValorIPI).Add(i.ValorST)
}

type NFeDuplicata struct {
	Numero     string
	Vencimento time.Time
	Valor      decimal.Decimal
}

type NFePagamento struct {
	Indicador string // 0 à vista, 1 a prazo
	Meio      string // tPag: 01 dinheiro, 15 boleto, 90 sem pagamento…
	Valor     decimal.Decimal
}

// ---- estrutura do XML (layout 4.00) ----

type xmlNFeProc struct {
	XMLName xml.Name `xml:"nfeProc"`
	NFe     xmlNFe   `xml:"NFe"`
	Prot    struct {
		Inf struct {
			ChNFe string `xml:"chNFe"`
			NProt string `xml:"nProt"`
			CStat string `xml:"cStat"`
		} `xml:"infProt"`
	} `xml:"protNFe"`
}

type xmlNFe struct {
	XMLName xml.Name  `xml:"NFe"`
	Inf     xmlInfNFe `xml:"infNFe"`
}

type xmlInfNFe struct {
	ID  string `xml:"Id,attr"`
	Ide struct {
		NatOp    string `xml:"natOp"`
		Mod      string `xml:"mod"`
		Serie    string `xml:"serie"`
		NNF      string `xml:"nNF"`
		DhEmi    string `xml:"dhEmi"`
		DEmi     string `xml:"dEmi"`
		DhSaiEnt string `xml:"dhSaiEnt"`
	} `xml:"ide"`
	Emit struct {
		CNPJ      string `xml:"CNPJ"`
		CPF       string `xml:"CPF"`
		XNome     string `xml:"xNome"`
		XFant     string `xml:"xFant"`
		IE        string `xml:"IE"`
		EnderEmit struct {
			XLgr    string `xml:"xLgr"`
			Nro     string `xml:"nro"`
			XCpl    string `xml:"xCpl"`
			XBairro string `xml:"xBairro"`
			CMun    string `xml:"cMun"`
			XMun    string `xml:"xMun"`
			UF      string `xml:"UF"`
			CEP     string `xml:"CEP"`
			Fone    string `xml:"fone"`
		} `xml:"enderEmit"`
	} `xml:"emit"`
	Dest struct {
		CNPJ string `xml:"CNPJ"`
		CPF  string `xml:"CPF"`
	} `xml:"dest"`
	Det   []xmlDet `xml:"det"`
	Total struct {
		ICMSTot struct {
			VBC     string `xml:"vBC"`
			VICMS   string `xml:"vICMS"`
			VST     string `xml:"vST"`
			VProd   string `xml:"vProd"`
			VFrete  string `xml:"vFrete"`
			VSeg    string `xml:"vSeg"`
			VDesc   string `xml:"vDesc"`
			VIPI    string `xml:"vIPI"`
			VPIS    string `xml:"vPIS"`
			VCOFINS string `xml:"vCOFINS"`
			VOutro  string `xml:"vOutro"`
			VNF     string `xml:"vNF"`
		} `xml:"ICMSTot"`
		ISSQNtot struct {
			VISSRet string `xml:"vISSRet"`
		} `xml:"ISSQNtot"`
		RetTrib struct {
			VRetPIS    string `xml:"vRetPIS"`
			VRetCOFINS string `xml:"vRetCOFINS"`
			VRetCSLL   string `xml:"vRetCSLL"`
			VBCIRRF    string `xml:"vBCIRRF"`
			VIRRF      string `xml:"vIRRF"`
			VBCRetPrev string `xml:"vBCRetPrev"`
			VRetPrev   string `xml:"vRetPrev"`
		} `xml:"retTrib"`
		IBSCBSTot struct {
			VBCIBSCBS string `xml:"vBCIBSCBS"`
			GIBS      struct {
				VIBS string `xml:"vIBS"`
			} `xml:"gIBS"`
			GCBS struct {
				VCBS string `xml:"vCBS"`
			} `xml:"gCBS"`
		} `xml:"IBSCBSTot"`
		ISTot struct {
			VIS string `xml:"vIS"`
		} `xml:"ISTot"`
	} `xml:"total"`
	Transp struct {
		ModFrete string `xml:"modFrete"`
	} `xml:"transp"`
	Cobr struct {
		Dup []struct {
			NDup  string `xml:"nDup"`
			DVenc string `xml:"dVenc"`
			VDup  string `xml:"vDup"`
		} `xml:"dup"`
	} `xml:"cobr"`
	Pag struct {
		DetPag []struct {
			IndPag string `xml:"indPag"`
			TPag   string `xml:"tPag"`
			VPag   string `xml:"vPag"`
		} `xml:"detPag"`
	} `xml:"pag"`
	InfAdic struct {
		InfCpl string `xml:"infCpl"`
	} `xml:"infAdic"`
}

// xmlGrupoTributo lê qualquer subgrupo de tributo: ICMS00, ICMS10 … ICMS90,
// ICMSSN101 … ICMSSN900, IPITrib/IPINT, PISAliq/PISOutr/PISNT… Os campos têm o
// mesmo nome em todos eles, e cada nota traz só um subgrupo por imposto.
type xmlGrupoTributo struct {
	XMLName xml.Name
	Orig    string `xml:"orig"`
	CST     string `xml:"CST"`
	CSOSN   string `xml:"CSOSN"`
	VBC     string `xml:"vBC"`
	PICMS   string `xml:"pICMS"`
	VICMS   string `xml:"vICMS"`
	VBCST   string `xml:"vBCST"`
	VICMSST string `xml:"vICMSST"`
	PIPI    string `xml:"pIPI"`
	VIPI    string `xml:"vIPI"`
	VPIS    string `xml:"vPIS"`
	VCOFINS string `xml:"vCOFINS"`
}

type xmlDet struct {
	NItem string `xml:"nItem,attr"`
	Prod  struct {
		CProd   string `xml:"cProd"`
		CEAN    string `xml:"cEAN"`
		XProd   string `xml:"xProd"`
		NCM     string `xml:"NCM"`
		CEST    string `xml:"CEST"`
		CFOP    string `xml:"CFOP"`
		UCom    string `xml:"uCom"`
		QCom    string `xml:"qCom"`
		VUnCom  string `xml:"vUnCom"`
		VProd   string `xml:"vProd"`
		VFrete  string `xml:"vFrete"`
		VSeg    string `xml:"vSeg"`
		VDesc   string `xml:"vDesc"`
		VOutro  string `xml:"vOutro"`
		XPed    string `xml:"xPed"`
		NItemPd string `xml:"nItemPed"`
	} `xml:"prod"`
	Imposto struct {
		ICMS struct {
			Grupos []xmlGrupoTributo `xml:",any"`
		} `xml:"ICMS"`
		IPI struct {
			Grupos []xmlGrupoTributo `xml:",any"`
		} `xml:"IPI"`
		PIS struct {
			Grupos []xmlGrupoTributo `xml:",any"`
		} `xml:"PIS"`
		COFINS struct {
			Grupos []xmlGrupoTributo `xml:",any"`
		} `xml:"COFINS"`
		IBSCBS struct {
			CST        string `xml:"CST"`
			CClassTrib string `xml:"cClassTrib"`
			G          struct {
				VBC    string `xml:"vBC"`
				GIBSUF struct {
					PIBSUF string `xml:"pIBSUF"`
					VIBSUF string `xml:"vIBSUF"`
				} `xml:"gIBSUF"`
				GIBSMun struct {
					PIBSMun string `xml:"pIBSMun"`
					VIBSMun string `xml:"vIBSMun"`
				} `xml:"gIBSMun"`
				VIBS string `xml:"vIBS"`
				GCBS struct {
					PCBS string `xml:"pCBS"`
					VCBS string `xml:"vCBS"`
				} `xml:"gCBS"`
			} `xml:"gIBSCBS"`
		} `xml:"IBSCBS"`
		IS struct {
			VIS string `xml:"vIS"`
		} `xml:"IS"`
	} `xml:"imposto"`
}

// LerNFe interpreta o XML da NF-e. Aceita a nota com <NFe> na raiz e dentro do
// envelope <nfeProc> (que é o arquivo que o fornecedor manda), e recusa
// qualquer outra coisa — inclusive o XML que é lido mas não traz nota.
func LerNFe(conteudo []byte) (*NFeLida, error) {
	bruto := strings.TrimSpace(strings.TrimPrefix(string(conteudo), "\uFEFF"))
	if bruto == "" {
		return nil, errorsuc.NewValidationError("envie o arquivo XML da nota")
	}

	var inf xmlInfNFe
	var chaveProt, protocolo string
	var proc xmlNFeProc
	if err := xml.Unmarshal([]byte(bruto), &proc); err == nil {
		inf = proc.NFe.Inf
		chaveProt = strings.TrimSpace(proc.Prot.Inf.ChNFe)
		protocolo = strings.TrimSpace(proc.Prot.Inf.NProt)
	} else {
		var direta xmlNFe
		if err := xml.Unmarshal([]byte(bruto), &direta); err != nil {
			return nil, errorsuc.NewValidationError(
				"o arquivo enviado não é um XML de NF-e: a raiz precisa ser <NFe> ou <nfeProc>. " +
					"Confira se é o XML da nota (não o DANFE em PDF) e se o conteúdo não está truncado")
		}
		inf = direta.Inf
	}

	numero := strings.TrimSpace(inf.Ide.NNF)
	cnpj := soDigitos(firstNonEmpty(inf.Emit.CNPJ, inf.Emit.CPF))
	if numero == "" || cnpj == "" {
		return nil, errorsuc.NewValidationError(
			"o XML foi lido mas não traz os dados da nota (número e CNPJ do emitente). " +
				"Confira se o arquivo é o XML completo da NF-e e não está truncado")
	}

	chave := soDigitos(strings.TrimPrefix(strings.TrimSpace(inf.ID), "NFe"))
	if chave == "" {
		chave = chaveProt
	}

	n := &NFeLida{
		ChaveAcesso:               chave,
		Protocolo:                 protocolo,
		Numero:                    numero,
		Serie:                     strings.TrimSpace(inf.Ide.Serie),
		Modelo:                    strings.TrimSpace(inf.Ide.Mod),
		NaturezaOperacao:          strings.TrimSpace(inf.Ide.NatOp),
		DataEmissao:               lerDataNFe(firstNonEmpty(inf.Ide.DhEmi, inf.Ide.DEmi)),
		EmitenteCNPJ:              cnpj,
		EmitenteNome:              strings.TrimSpace(inf.Emit.XNome),
		EmitenteFantasia:          strings.TrimSpace(inf.Emit.XFant),
		EmitenteIE:                strings.TrimSpace(inf.Emit.IE),
		EmitenteUF:                strings.TrimSpace(inf.Emit.EnderEmit.UF),
		EmitenteMunicipio:         strings.TrimSpace(inf.Emit.EnderEmit.XMun),
		EmitenteLogradouro:        strings.TrimSpace(inf.Emit.EnderEmit.XLgr),
		EmitenteNumero:            strings.TrimSpace(inf.Emit.EnderEmit.Nro),
		EmitenteComplemento:       strings.TrimSpace(inf.Emit.EnderEmit.XCpl),
		EmitenteBairro:            strings.TrimSpace(inf.Emit.EnderEmit.XBairro),
		EmitenteCodigoMunicipio:   soDigitos(inf.Emit.EnderEmit.CMun),
		EmitenteCEP:               soDigitos(inf.Emit.EnderEmit.CEP),
		EmitenteFone:              soDigitos(inf.Emit.EnderEmit.Fone),
		DestinatarioCNPJ:          soDigitos(firstNonEmpty(inf.Dest.CNPJ, inf.Dest.CPF)),
		ValorProdutos:             dec(inf.Total.ICMSTot.VProd),
		ValorFrete:                dec(inf.Total.ICMSTot.VFrete),
		ValorSeguro:               dec(inf.Total.ICMSTot.VSeg),
		ValorDesconto:             dec(inf.Total.ICMSTot.VDesc),
		ValorOutras:               dec(inf.Total.ICMSTot.VOutro),
		ValorIPI:                  dec(inf.Total.ICMSTot.VIPI),
		ValorICMS:                 dec(inf.Total.ICMSTot.VICMS),
		ValorICMSST:               dec(inf.Total.ICMSTot.VST),
		ValorPIS:                  dec(inf.Total.ICMSTot.VPIS),
		ValorCOFINS:               dec(inf.Total.ICMSTot.VCOFINS),
		ValorTotal:                dec(inf.Total.ICMSTot.VNF),
		ModalidadeFrete:           strings.TrimSpace(inf.Transp.ModFrete),
		BaseIBSCBS:                dec(inf.Total.IBSCBSTot.VBCIBSCBS),
		ValorIBS:                  dec(inf.Total.IBSCBSTot.GIBS.VIBS),
		ValorCBS:                  dec(inf.Total.IBSCBSTot.GCBS.VCBS),
		ValorIS:                   dec(inf.Total.ISTot.VIS),
		ValorRetPIS:               dec(inf.Total.RetTrib.VRetPIS),
		ValorRetCOFINS:            dec(inf.Total.RetTrib.VRetCOFINS),
		ValorRetCSLL:              dec(inf.Total.RetTrib.VRetCSLL),
		BaseIRRF:                  dec(inf.Total.RetTrib.VBCIRRF),
		ValorIRRF:                 dec(inf.Total.RetTrib.VIRRF),
		BaseRetPrev:               dec(inf.Total.RetTrib.VBCRetPrev),
		ValorRetPrev:              dec(inf.Total.RetTrib.VRetPrev),
		ValorISSRet:               dec(inf.Total.ISSQNtot.VISSRet),
		InformacoesComplementares: strings.TrimSpace(inf.InfAdic.InfCpl),
	}
	if n.DataEmissao.IsZero() {
		return nil, errorsuc.NewValidationError("o XML não traz a data de emissão da nota (dhEmi)")
	}
	if s := strings.TrimSpace(inf.Ide.DhSaiEnt); s != "" {
		if t := lerDataNFe(s); !t.IsZero() {
			n.DataSaidaEntrada = &t
		}
	}

	for i, det := range inf.Det {
		it := NFeItemLido{
			Numero:        i + 1,
			CodigoProduto: strings.TrimSpace(det.Prod.CProd),
			EAN:           eanValido(det.Prod.CEAN),
			Descricao:     strings.TrimSpace(det.Prod.XProd),
			NCM:           soDigitos(det.Prod.NCM),
			CEST:          soDigitos(det.Prod.CEST),
			CFOP:          strings.TrimSpace(det.Prod.CFOP),
			Unidade:       strings.ToUpper(strings.TrimSpace(det.Prod.UCom)),
			Quantidade:    dec(det.Prod.QCom),
			ValorUnitario: dec(det.Prod.VUnCom),
			ValorProduto:  dec(det.Prod.VProd),
			ValorFrete:    dec(det.Prod.VFrete),
			ValorSeguro:   dec(det.Prod.VSeg),
			ValorDesconto: dec(det.Prod.VDesc),
			ValorOutras:   dec(det.Prod.VOutro),
			PedidoCompra:  strings.TrimSpace(det.Prod.XPed),
			ItemPedido:    strings.TrimSpace(det.Prod.NItemPd),
		}
		if g := primeiroGrupo(det.Imposto.ICMS.Grupos); g != nil {
			it.Origem = strings.TrimSpace(g.Orig)
			it.CSTICMS = strings.TrimSpace(firstNonEmpty(g.CST, g.CSOSN))
			it.BaseICMS = dec(g.VBC)
			it.AliqICMS = dec(g.PICMS)
			it.ValorICMS = dec(g.VICMS)
			it.BaseST = dec(g.VBCST)
			it.ValorST = dec(g.VICMSST)
		}
		// No IPI o grupo <cEnq> vem antes do <IPITrib>: o grupo que interessa é o
		// que traz CST.
		if g := grupoComCST(det.Imposto.IPI.Grupos); g != nil {
			it.CSTIPI = strings.TrimSpace(g.CST)
			it.BaseIPI = dec(g.VBC)
			it.AliqIPI = dec(g.PIPI)
			it.ValorIPI = dec(g.VIPI)
		}
		if g := grupoComCST(det.Imposto.PIS.Grupos); g != nil {
			it.CSTPIS = strings.TrimSpace(g.CST)
			it.ValorPIS = dec(g.VPIS)
		}
		if g := grupoComCST(det.Imposto.COFINS.Grupos); g != nil {
			it.CSTCOFINS = strings.TrimSpace(g.CST)
			it.ValorCOFINS = dec(g.VCOFINS)
		}
		ibs := det.Imposto.IBSCBS
		it.CSTIBSCBS = strings.TrimSpace(ibs.CST)
		it.ClassTrib = strings.TrimSpace(ibs.CClassTrib)
		it.BaseIBSCBS = dec(ibs.G.VBC)
		it.AliqIBSUF = dec(ibs.G.GIBSUF.PIBSUF)
		it.ValorIBSUF = dec(ibs.G.GIBSUF.VIBSUF)
		it.AliqIBSMun = dec(ibs.G.GIBSMun.PIBSMun)
		it.ValorIBSMun = dec(ibs.G.GIBSMun.VIBSMun)
		it.ValorIBS = dec(ibs.G.VIBS)
		if it.ValorIBS.IsZero() {
			it.ValorIBS = it.ValorIBSUF.Add(it.ValorIBSMun)
		}
		it.AliqCBS = dec(ibs.G.GCBS.PCBS)
		it.ValorCBS = dec(ibs.G.GCBS.VCBS)
		it.ValorIS = dec(det.Imposto.IS.VIS)
		n.Itens = append(n.Itens, it)
	}
	if len(n.Itens) == 0 {
		return nil, errorsuc.NewValidationError("o XML da nota não tem itens (<det>)")
	}
	if n.ValorIBS.IsZero() && n.ValorCBS.IsZero() {
		for _, it := range n.Itens {
			n.ValorIBS = n.ValorIBS.Add(it.ValorIBS)
			n.ValorCBS = n.ValorCBS.Add(it.ValorCBS)
			n.BaseIBSCBS = n.BaseIBSCBS.Add(it.BaseIBSCBS)
		}
	}

	for _, d := range inf.Cobr.Dup {
		venc := lerDataNFe(d.DVenc)
		valor := dec(d.VDup)
		if venc.IsZero() || !valor.IsPositive() {
			continue
		}
		n.Duplicatas = append(n.Duplicatas, NFeDuplicata{Numero: strings.TrimSpace(d.NDup), Vencimento: venc, Valor: valor})
	}
	for _, p := range inf.Pag.DetPag {
		n.Pagamentos = append(n.Pagamentos, NFePagamento{
			Indicador: strings.TrimSpace(p.IndPag),
			Meio:      strings.TrimSpace(p.TPag),
			Valor:     dec(p.VPag),
		})
	}
	return n, nil
}

// SemPagamento diz se a nota declara que não há o que pagar (tPag 90: remessa,
// bonificação, amostra). Essa nota não gera título no contas a pagar.
func (n *NFeLida) SemPagamento() bool {
	if len(n.Duplicatas) > 0 || len(n.Pagamentos) == 0 {
		return false
	}
	for _, p := range n.Pagamentos {
		if p.Meio != "90" {
			return false
		}
	}
	return true
}

func primeiroGrupo(gs []xmlGrupoTributo) *xmlGrupoTributo {
	if len(gs) == 0 {
		return nil
	}
	return &gs[0]
}

func grupoComCST(gs []xmlGrupoTributo) *xmlGrupoTributo {
	for i := range gs {
		if strings.TrimSpace(gs[i].CST) != "" {
			return &gs[i]
		}
	}
	return nil
}

func dec(s string) decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// lerDataNFe aceita "2026-09-30T10:15:00-03:00" (dhEmi) e "2026-09-30" (dEmi,
// dVenc). A data vale como está no documento: converter o fuso mudaria o dia
// de notas emitidas perto da meia-noite.
func lerDataNFe(s string) time.Time {
	s = strings.TrimSpace(s)
	if len(s) < 10 {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", s[:10])
	if err != nil {
		return time.Time{}
	}
	return t
}

func soDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// eanValido descarta o "SEM GTIN" que a nota traz quando o produto não tem
// código de barras.
func eanValido(s string) string {
	d := soDigitos(s)
	if len(d) < 8 {
		return ""
	}
	return d
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
