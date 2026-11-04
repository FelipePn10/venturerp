package sped

import (
	"time"

	"github.com/shopspring/decimal"
)

// EFDParams holds all data needed to generate a SPED EFD ICMS/IPI file.
type EFDParams struct {
	Empresa           EFDEmpresa
	Periodo           EFDPeriodo
	Participantes     []EFDParticipante
	Unidades          []EFDUnidade
	Itens             []EFDItem
	DocumentosFiscais []EFDDocumentoFiscal
	// Conhecimentos de transporte (D100/D190): o CT-e do frete de compra.
	Conhecimentos []EFDConhecimento
	ApuracaoICMS  *EFDApuracaoICMS
	// CIAP (Bloco G, 0300/0305/0500/0600): o crédito do ativo imobilizado.
	CIAP *EFDCIAP
	// ApuracoesST (E200/E210/E250): o ICMS-ST retido, por UF de destino.
	ApuracoesST []EFDApuracaoST
	// ApuracoesDIFAL (E300/E310/E316): DIFAL e FCP da venda a consumidor final
	// não contribuinte de outra UF (EC 87/2015), por UF de destino.
	ApuracoesDIFAL []EFDApuracaoDIFAL
	// ApuracaoIPI (E500/E510/E520): contribuinte do IPI (indústria).
	ApuracaoIPI *EFDApuracaoIPI
	Inventario  []EFDInventarioItem
	// MotivoInventario (H005): 01 final do período, 02 mudança de tributação...
	MotivoInventario string
}

type EFDEmpresa struct {
	CNPJ              string
	Nome              string
	UF                string
	IE                string
	IM                string
	SUFRAMA           string
	CodigoMunicipio   string
	RegimeTributario  string // 1=Simples, 2=Normal, 3=MEI
	CodigoFinalizacao string // 0=regular, 1=extinção, 2=fusão, 3=cisão, 4=transformação
	// Dados do contabilista
	ContabilistaNome string
	ContabilistaCPF  string
	ContabilistaCRC  string
	ContabilistaCNPJ string
	// Registro 0005 (dados complementares).
	Fantasia, CEP, Endereco, Numero, Complemento, Bairro, Fone, Email string
	// IndAtividade (0000/IND_ATIV): 0 industrial ou equiparado, 1 outros.
	IndAtividade string
}

type EFDPeriodo struct {
	DataInicial time.Time
	DataFinal   time.Time
	// 0 = regular, 1 = retificadora
	IndicadorSituacaoEspecial string
}

type EFDParticipante struct {
	CodPart         string // internal code
	Nome            string
	CodigoPais      string // 1058 = Brasil
	CNPJ            string
	CPF             string
	IE              string
	CodigoMunicipio string
	SUFRAMA         string
	Endereco        string
	Num             string
	Complemento     string
	Bairro          string
	CEP             string
	Telefone        string
}

type EFDUnidade struct {
	CodUnd  string
	DescUnd string
}

type EFDItem struct {
	CodItem  string
	DescItem string
	CodBarra string
	CodAnt   string
	UnCom    string
	TipoItem string // 00=mercadoria para revenda, 01=materia-prima, etc.
	CodNCM   string
	ExIPI    string
	CodGen   string
	CodLST   string
	AliqICMS decimal.Decimal
	CEST     string
	// Conversoes (0220): unidades dos documentos diferentes da de estoque.
	Conversoes []EFDConversao
}

// EFDConversao — 0220: a unidade do documento e quantas unidades de estoque ela vale.
type EFDConversao struct {
	UnidConv string
	FatConv  decimal.Decimal
}

// EFDConhecimento — D100 (CT-e) e os seus D190.
type EFDConhecimento struct {
	IndOper    string // 0 entrada (frete contratado pela empresa)
	IndEmit    string // 1 terceiros
	CodPart    string
	CodMod     string // 57
	CodSit     string
	Ser        string
	NumDoc     string
	ChvCTe     string
	DtDoc      time.Time
	DtAP       time.Time
	TpCTe      string // 0 normal
	VlDoc      decimal.Decimal
	IndFrt     string // 0 por conta do emitente, 1 do destinatário, 2 de terceiros, 9 sem frete
	VlServ     decimal.Decimal
	VlBcIcms   decimal.Decimal
	VlIcms     decimal.Decimal
	VlNt       decimal.Decimal
	CodMunOrig string
	CodMunDest string
	Analiticos []EFDD190
}

// EFDD190 — registro analítico do CT-e.
type EFDD190 struct {
	CstIcms  string
	Cfop     string
	AliqIcms decimal.Decimal
	VlOpr    decimal.Decimal
	VlBcIcms decimal.Decimal
	VlIcms   decimal.Decimal
	VlRedBc  decimal.Decimal
}

type EFDDocumentoFiscal struct {
	// Registro C100
	IndOper    string // 0=entrada, 1=saída
	IndEmit    string // 0=emissão própria, 1=terceiros
	CodPart    string
	CodMod     string // 55=NF-e, 65=NFC-e
	CodSit     string // 00=regular, 02=cancelado, etc.
	SerDoc     string
	NumDoc     string
	ChvNfe     string
	DtDoc      time.Time
	DtES       time.Time // data entrada/saída
	VlDoc      decimal.Decimal
	IndPgto    string // 0=à vista, 1=a prazo, 9=outros
	VlDesc     decimal.Decimal
	VlAbatNt   decimal.Decimal
	VlMerc     decimal.Decimal
	IndFrt     string // 0=CIF, 1=FOB, 2=terceiros, 3=próprio remetente, 4=próprio destinatário, 9=sem frete
	VlFrt      decimal.Decimal
	VlSeg      decimal.Decimal
	VlOutDa    decimal.Decimal
	VlBcIcms   decimal.Decimal
	VlIcms     decimal.Decimal
	VlBcIcmsSt decimal.Decimal
	VlIcmsSt   decimal.Decimal
	VlIpi      decimal.Decimal
	VlPis      decimal.Decimal
	VlCofins   decimal.Decimal
	VlPisSt    decimal.Decimal
	VlCofinsSt decimal.Decimal
	Itens      []EFDItemDoc
	// C101: DIFAL e FCP da UF de destino (venda a não contribuinte de outra UF).
	C101 *EFDC101
	// C190 analítico (one per CFOP/CST/aliquota combination)
	AnaliticosICMS []EFDC190
}

type EFDItemDoc struct {
	// Registro C170
	NumItem     int
	CodItem     string
	DescCompl   string
	Qtd         decimal.Decimal
	UnCom       string
	VlUnt       decimal.Decimal
	VlDesc      decimal.Decimal
	IndMov      string // 0=sim, 1=não
	CstIcms     string
	CfopC170    string
	CodNat      string
	VlBcIcms    decimal.Decimal
	AliqIcms    decimal.Decimal
	VlIcms      decimal.Decimal
	VlBcIcmsSt  decimal.Decimal
	AliqSt      decimal.Decimal
	VlIcmsSt    decimal.Decimal
	IndApur     string
	CstIpi      string
	CodEnq      string
	VlBcIpi     decimal.Decimal
	AliqIpi     decimal.Decimal
	VlIpi       decimal.Decimal
	CstPis      string
	VlBcPis     decimal.Decimal
	AliqPis     decimal.Decimal
	QtdBcPis    decimal.Decimal
	AliqPisQ    decimal.Decimal
	VlPis       decimal.Decimal
	CstCofins   string
	VlBcCofins  decimal.Decimal
	AliqCofins  decimal.Decimal
	QtdBcCofins decimal.Decimal
	AliqCofinsQ decimal.Decimal
	VlCofins    decimal.Decimal
	CodCta      string
	VlAbatNt    decimal.Decimal
}

// EFDC190 — Registro analítico do documento (C190).
type EFDC190 struct {
	CstIcms    string
	Cfop       string
	AliqIcms   decimal.Decimal
	VlOpr      decimal.Decimal
	VlBcIcms   decimal.Decimal
	VlIcms     decimal.Decimal
	VlBcIcmsSt decimal.Decimal
	VlIcmsSt   decimal.Decimal
	VlRedBc    decimal.Decimal
	VlIpi      decimal.Decimal
	CodObs     string
}

// EFDApuracaoICMS — Registros E110/E111/E116.
type EFDApuracaoICMS struct {
	VlTotDebitos        decimal.Decimal
	VlAjDebitos         decimal.Decimal
	VlTotAjDebitos      decimal.Decimal
	VlEstornosCreditos  decimal.Decimal
	VlTotCreditos       decimal.Decimal
	VlAjCreditos        decimal.Decimal
	VlTotAjCreditos     decimal.Decimal
	VlEstornosDebitos   decimal.Decimal
	VlSaldoCredorAnt    decimal.Decimal
	VlApuracao          decimal.Decimal
	VlTotDed            decimal.Decimal
	VlIcmsRecolher      decimal.Decimal
	VlSaldoCredorTransp decimal.Decimal
	DebEspeciais        decimal.Decimal
	Ajustes             []EFDApuracaoAjuste
	// Obrigacoes (E116): o ICMS a recolher.
	Obrigacoes []EFDObrigacao
}

// EFDObrigacao — E116.
type EFDObrigacao struct {
	CodOr    string // 000 ICMS a recolher
	VlOr     decimal.Decimal
	DtVcto   time.Time
	CodRec   string // código de receita da UF
	TxtCompl string
	MesRef   string // MMAAAA
}

// EFDApuracaoIPI — E500/E510/E520.
type EFDApuracaoIPI struct {
	IndApur string // 0 mensal
	Linhas  []EFDE510
	SdAnt   decimal.Decimal
	Deb     decimal.Decimal
	Cred    decimal.Decimal
	Od      decimal.Decimal
	Oc      decimal.Decimal
}

// EFDE510 — consolidação por CFOP e CST do IPI.
type EFDE510 struct {
	Cfop    string
	CstIpi  string
	VlCont  decimal.Decimal
	VlBcIpi decimal.Decimal
	VlIpi   decimal.Decimal
}

// EFDApuracaoAjuste — Registro E111 (ajustes de apuração).
type EFDApuracaoAjuste struct {
	CodAjApur string
	DescCompl string
	VlAjApur  decimal.Decimal
}

type EFDInventarioItem struct {
	DtInv    time.Time
	CodItem  string
	Unid     string
	Qtd      decimal.Decimal
	VlUnit   decimal.Decimal
	VlItem   decimal.Decimal
	IndProp  string // 0=posse própria, 1=posse de terceiros, 2=posse de terceiros em nosso poder
	CodPart  string
	TxtCompl string
	CodCta   string
	VlItemIr decimal.Decimal
}

// EFDC101 — informação complementar do C100 na venda interestadual a
// consumidor final não contribuinte (EC 87/2015).
type EFDC101 struct {
	VlFcpUfDest, VlIcmsUfDest, VlIcmsUfRem decimal.Decimal
}

// EFDApuracaoST — E200 (período), E210 (apuração) e E250 (obrigação) de uma UF.
type EFDApuracaoST struct {
	UF                 string
	Retencao, Recolher decimal.Decimal
	Obrigacao          EFDObrigacao
}

// EFDApuracaoDIFAL — E300 (período), E310 (apuração) e E316 (obrigações) de uma UF.
type EFDApuracaoDIFAL struct {
	UF         string
	Difal, FCP decimal.Decimal
	Obrigacoes []EFDObrigacao
}
