package ecd

import "time"

type ECDParams struct {
	Empresa       ECDEmpresa
	Periodo       ECDPeriodo
	Plano         ECDPlano
	Contas        []ECDConta
	CostCenters   []ECDCentroCusto
	Livros        []ECDLivro
	Participantes []ECDParticipante
	Lancamentos   []ECDLancamento
	Balancetes    []ECDBalancete
	DRE           []ECDDREItem
}

type ECDEmpresa struct {
	CNPJ            string
	CPF             string
	Nome            string
	UF              string
	Email           string
	IE              string
	CodigoMunicipio string
	CEP             string
	Endereco        string
	Numero          string
	Complemento     string
	Bairro          string
	Fone            string
	NIRE            string
	IndSitAtiv      string
	IndNireCert     string
	IndGrandePorte  string
	IndEscCons      string
	TipoECD         string
	HashECDSub      string
	NumOrd          string
	NomeAudi        string
	IndSitEsp       string
}

type ECDPeriodo struct {
	DataInicial time.Time
	DataFinal   time.Time
}

type ECDPlano struct {
	Numero    int
	Descricao string
}

type ECDConta struct {
	CodCta     string
	CodECD     string
	TipoCta    string
	Nivel      int
	CodCtaSup  string
	CtaRef     string
	IndCtaCons string
	DescCta    string
	Codigo     string
	NIF        string
}

type ECDCentroCusto struct {
	CodCCus string
	CCus    string
}

type ECDLivro struct {
	NumOrd     string
	NatLivro   string
	NumLiv     string
	DescLiv    string
	CodHash    string
	NumHash    string
	PerIni     time.Time
	PerFin     time.Time
	CodHashAnt string
	NumHashAnt string
}

type ECDParticipante struct {
	CodPart  string
	Nome     string
	CodPais  string
	CNPJ     string
	CPF      string
	TipoPart string
}

type ECDLancamento struct {
	NumLcto  string
	DtLcto   time.Time
	CodHist  string
	DescHist string
	Partidas []ECDPartida
}

type ECDPartida struct {
	CodCta   string
	CodCCus  string
	VlLcto   float64
	IndDC    string
	DescHist string
	CodHist  string
	NumDoc   string
}

type ECDBalancete struct {
	CodCtaSup string
	CodCta    string
	DescCta   string
	VlIni     float64
	IndDCIni  string
	VlFin     float64
	IndDCFin  string
}

type ECDDREItem struct {
	CodCtaSup string
	CodCta    string
	DescCta   string
	TipoDem   string
	DtIni     time.Time
	DtFin     time.Time
	VlCta     float64
	IndDC     string
}
