package fiscal_uc

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// CTeLido é o que o frete de compra aproveita do XML do CT-e.
type CTeLido struct {
	Chave          string
	Numero         int64
	Serie          string
	CFOP           string
	DataEmissao    time.Time
	EmitenteCNPJ   string
	EmitenteNome   string
	EmitenteUF     string
	TomadorCNPJ    string
	ValorPrestacao decimal.Decimal
	BaseICMS       decimal.Decimal
	AliqICMS       decimal.Decimal
	ValorICMS      decimal.Decimal
	ChavesNFe      []string // notas transportadas
	ConteudoXML    string
}

type xmlPessoaCTe struct {
	CNPJ string `xml:"CNPJ"`
	CPF  string `xml:"CPF"`
}

type xmlGrupoICMSCTe struct {
	XMLName      xml.Name
	VBC          string `xml:"vBC"`
	PICMS        string `xml:"pICMS"`
	VICMS        string `xml:"vICMS"`
	VBCOutraUF   string `xml:"vBCOutraUF"`
	PICMSOutraUF string `xml:"pICMSOutraUF"`
	VICMSOutraUF string `xml:"vICMSOutraUF"`
}

type xmlInfCTe struct {
	ID  string `xml:"Id,attr"`
	Ide struct {
		CFOP  string `xml:"CFOP"`
		Serie string `xml:"serie"`
		NCT   string `xml:"nCT"`
		DhEmi string `xml:"dhEmi"`
		Toma3 struct {
			Toma string `xml:"toma"`
		} `xml:"toma3"`
		Toma4 struct {
			Toma string `xml:"toma"`
			CNPJ string `xml:"CNPJ"`
			CPF  string `xml:"CPF"`
		} `xml:"toma4"`
	} `xml:"ide"`
	Emit struct {
		CNPJ      string `xml:"CNPJ"`
		XNome     string `xml:"xNome"`
		EnderEmit struct {
			UF string `xml:"UF"`
		} `xml:"enderEmit"`
	} `xml:"emit"`
	Rem    xmlPessoaCTe `xml:"rem"`
	Exped  xmlPessoaCTe `xml:"exped"`
	Receb  xmlPessoaCTe `xml:"receb"`
	Dest   xmlPessoaCTe `xml:"dest"`
	VPrest struct {
		VTPrest string `xml:"vTPrest"`
	} `xml:"vPrest"`
	Imp struct {
		ICMS struct {
			Grupos []xmlGrupoICMSCTe `xml:",any"`
		} `xml:"ICMS"`
	} `xml:"imp"`
	Norm struct {
		InfDoc struct {
			InfNFe []struct {
				Chave string `xml:"chave"`
			} `xml:"infNFe"`
		} `xml:"infDoc"`
	} `xml:"infCTeNorm"`
}

type xmlCTeProc struct {
	XMLName xml.Name `xml:"cteProc"`
	CTe     struct {
		Inf xmlInfCTe `xml:"infCte"`
	} `xml:"CTe"`
	Prot struct {
		Inf struct {
			ChCTe string `xml:"chCTe"`
		} `xml:"infProt"`
	} `xml:"protCTe"`
}

type xmlCTe struct {
	XMLName xml.Name  `xml:"CTe"`
	Inf     xmlInfCTe `xml:"infCte"`
}

// LerCTe interpreta o XML do CT-e (com ou sem o envelope cteProc).
func LerCTe(conteudo []byte) (*CTeLido, error) {
	bruto := strings.TrimSpace(strings.TrimPrefix(string(conteudo), "\uFEFF"))
	if bruto == "" {
		return nil, errorsuc.NewValidationError("envie o arquivo XML do CT-e")
	}
	var inf xmlInfCTe
	chaveProt := ""
	var proc xmlCTeProc
	if err := xml.Unmarshal([]byte(bruto), &proc); err == nil {
		inf, chaveProt = proc.CTe.Inf, strings.TrimSpace(proc.Prot.Inf.ChCTe)
	} else {
		var direto xmlCTe
		if err := xml.Unmarshal([]byte(bruto), &direto); err != nil {
			return nil, errorsuc.NewValidationError("o arquivo não é um XML de CT-e: a raiz precisa ser <CTe> ou <cteProc>")
		}
		inf = direto.Inf
	}
	numero := strings.TrimSpace(inf.Ide.NCT)
	cnpj := soDigitos(inf.Emit.CNPJ)
	if numero == "" || cnpj == "" {
		return nil, errorsuc.NewValidationError("o XML foi lido mas não traz o número do CT-e e o CNPJ da transportadora")
	}
	c := &CTeLido{
		Chave:          firstNonEmpty(soDigitos(strings.TrimPrefix(strings.TrimSpace(inf.ID), "CTe")), chaveProt),
		Serie:          firstNonEmpty(strings.TrimSpace(inf.Ide.Serie), "1"),
		CFOP:           soDigitos(inf.Ide.CFOP),
		DataEmissao:    lerDataNFe(inf.Ide.DhEmi),
		EmitenteCNPJ:   cnpj,
		EmitenteNome:   strings.TrimSpace(inf.Emit.XNome),
		EmitenteUF:     strings.ToUpper(strings.TrimSpace(inf.Emit.EnderEmit.UF)),
		ValorPrestacao: dec(inf.VPrest.VTPrest),
		ConteudoXML:    bruto,
	}
	if n, err := decimal.NewFromString(numero); err == nil {
		c.Numero = n.IntPart()
	}
	// Tomador: quem paga o frete (toma3: 0 remetente, 1 expedidor, 2 recebedor,
	// 3 destinatário; toma4: outro, com o CNPJ).
	switch strings.TrimSpace(inf.Ide.Toma3.Toma) {
	case "0":
		c.TomadorCNPJ = soDigitos(firstNonEmpty(inf.Rem.CNPJ, inf.Rem.CPF))
	case "1":
		c.TomadorCNPJ = soDigitos(firstNonEmpty(inf.Exped.CNPJ, inf.Exped.CPF))
	case "2":
		c.TomadorCNPJ = soDigitos(firstNonEmpty(inf.Receb.CNPJ, inf.Receb.CPF))
	case "3":
		c.TomadorCNPJ = soDigitos(firstNonEmpty(inf.Dest.CNPJ, inf.Dest.CPF))
	}
	if c.TomadorCNPJ == "" {
		c.TomadorCNPJ = soDigitos(firstNonEmpty(inf.Ide.Toma4.CNPJ, inf.Ide.Toma4.CPF))
	}
	for _, g := range inf.Imp.ICMS.Grupos {
		v := dec(firstNonEmpty(g.VICMS, g.VICMSOutraUF))
		if v.IsPositive() {
			c.BaseICMS = dec(firstNonEmpty(g.VBC, g.VBCOutraUF))
			c.AliqICMS = dec(firstNonEmpty(g.PICMS, g.PICMSOutraUF))
			c.ValorICMS = v
			break
		}
	}
	for _, n := range inf.Norm.InfDoc.InfNFe {
		if ch := soDigitos(n.Chave); len(ch) == 44 {
			c.ChavesNFe = append(c.ChavesNFe, ch)
		}
	}
	if !c.ValorPrestacao.IsPositive() {
		return nil, errorsuc.NewValidationError("o CT-e não tem valor da prestação (vTPrest)")
	}
	return c, nil
}
