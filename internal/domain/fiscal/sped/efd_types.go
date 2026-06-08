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
	ApuracaoICMS      *EFDApuracaoICMS
	Inventario        []EFDInventarioItem
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
