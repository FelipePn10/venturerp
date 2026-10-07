package sped

import "time"

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
	AliqICMS float64
	CEST     string
	// Conversoes (0220): unidades dos documentos diferentes da de estoque.
	Conversoes []EFDConversao
}

// EFDConversao — 0220: a unidade do documento e quantas unidades de estoque ela vale.
type EFDConversao struct {
	UnidConv string
	FatConv  float64
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
	VlDoc      float64
	IndFrt     string // 0 por conta do emitente, 1 do destinatário, 2 de terceiros, 9 sem frete
	VlServ     float64
	VlBcIcms   float64
	VlIcms     float64
	VlNt       float64
	CodMunOrig string
	CodMunDest string
	Analiticos []EFDD190
}

// EFDD190 — registro analítico do CT-e.
type EFDD190 struct {
	CstIcms  string
	Cfop     string
	AliqIcms float64
	VlOpr    float64
	VlBcIcms float64
	VlIcms   float64
	VlRedBc  float64
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
	VlDoc      float64
	IndPgto    string // 0=à vista, 1=a prazo, 9=outros
	VlDesc     float64
	VlAbatNt   float64
	VlMerc     float64
	IndFrt     string // 0=CIF, 1=FOB, 2=terceiros, 3=próprio remetente, 4=próprio destinatário, 9=sem frete
	VlFrt      float64
	VlSeg      float64
	VlOutDa    float64
	VlBcIcms   float64
	VlIcms     float64
	VlBcIcmsSt float64
	VlIcmsSt   float64
	VlIpi      float64
	VlPis      float64
	VlCofins   float64
	VlPisSt    float64
	VlCofinsSt float64
	Itens      []EFDItemDoc
	// C190 analítico (one per CFOP/CST/aliquota combination)
	AnaliticosICMS []EFDC190
}

type EFDItemDoc struct {
	// Registro C170
	NumItem     int
	CodItem     string
	DescCompl   string
	Qtd         float64
	UnCom       string
	VlUnt       float64
	VlDesc      float64
	IndMov      string // 0=sim, 1=não
	CstIcms     string
	CfopC170    string
	CodNat      string
	VlBcIcms    float64
	AliqIcms    float64
	VlIcms      float64
	VlBcIcmsSt  float64
	AliqSt      float64
	VlIcmsSt    float64
	IndApur     string
	CstIpi      string
	CodEnq      string
	VlBcIpi     float64
	AliqIpi     float64
	VlIpi       float64
	CstPis      string
	VlBcPis     float64
	AliqPis     float64
	QtdBcPis    float64
	AliqPisQ    float64
	VlPis       float64
	CstCofins   string
	VlBcCofins  float64
	AliqCofins  float64
	QtdBcCofins float64
	AliqCofinsQ float64
	VlCofins    float64
	CodCta      string
	VlAbatNt    float64
}

// EFDC190 — Registro analítico do documento (C190).
type EFDC190 struct {
	CstIcms    string
	Cfop       string
	AliqIcms   float64
	VlOpr      float64
	VlBcIcms   float64
	VlIcms     float64
	VlBcIcmsSt float64
	VlIcmsSt   float64
	VlRedBc    float64
	VlIpi      float64
	CodObs     string
}

// EFDApuracaoICMS — Registros E110/E111/E116.
type EFDApuracaoICMS struct {
	VlTotDebitos        float64
	VlAjDebitos         float64
	VlTotAjDebitos      float64
	VlEstornosCreditos  float64
	VlTotCreditos       float64
	VlAjCreditos        float64
	VlTotAjCreditos     float64
	VlEstornosDebitos   float64
	VlSaldoCredorAnt    float64
	VlApuracao          float64
	VlTotDed            float64
	VlIcmsRecolher      float64
	VlSaldoCredorTransp float64
	DebEspeciais        float64
	Ajustes             []EFDApuracaoAjuste
	// Obrigacoes (E116): o ICMS a recolher.
	Obrigacoes []EFDObrigacao
}

// EFDObrigacao — E116.
type EFDObrigacao struct {
	CodOr    string // 000 ICMS a recolher
	VlOr     float64
	DtVcto   time.Time
	CodRec   string // código de receita da UF
	TxtCompl string
	MesRef   string // MMAAAA
}

// EFDApuracaoIPI — E500/E510/E520.
type EFDApuracaoIPI struct {
	IndApur string // 0 mensal
	Linhas  []EFDE510
	SdAnt   float64
	Deb     float64
	Cred    float64
	Od      float64
	Oc      float64
}

// EFDE510 — consolidação por CFOP e CST do IPI.
type EFDE510 struct {
	Cfop    string
	CstIpi  string
	VlCont  float64
	VlBcIpi float64
	VlIpi   float64
}

// EFDApuracaoAjuste — Registro E111 (ajustes de apuração).
type EFDApuracaoAjuste struct {
	CodAjApur string
	DescCompl string
	VlAjApur  float64
}

type EFDInventarioItem struct {
	DtInv    time.Time
	CodItem  string
	Unid     string
	Qtd      float64
	VlUnit   float64
	VlItem   float64
	IndProp  string // 0=posse própria, 1=posse de terceiros, 2=posse de terceiros em nosso poder
	CodPart  string
	TxtCompl string
	CodCta   string
	VlItemIr float64
}
