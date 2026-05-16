package fiscal_uc

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type UploadNFEEntryUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

type nfeXML struct {
	XMLName xml.Name `xml:"NFe"`
	InfNFe  struct {
		Ide struct {
			NUF      string `xml:"cUF"`
			NF       string `xml:"nNF"`
			Serie    string `xml:"serie"`
			Mod      string `xml:"mod"`
			DhEmi    string `xml:"dhEmi"`
		} `xml:"ide"`
		Emit struct {
			CNPJ  string `xml:"CNPJ"`
			XNome string `xml:"xNome"`
			IE    string `xml:"IE"`
			UF    string `xml:"UF"`
		} `xml:"emit"`
		Total struct {
			ICMSTot struct {
				VProd   string `xml:"vProd"`
				VFrete  string `xml:"vFrete"`
				VSeg    string `xml:"vSeg"`
				VDesc   string `xml:"vDesc"`
				VIPI    string `xml:"vIPI"`
				VICMS   string `xml:"vICMS"`
				VPIS    string `xml:"vPIS"`
				VCOFINS string `xml:"vCOFINS"`
				VNF     string `xml:"vNF"`
			} `xml:"ICMSTot"`
		} `xml:"total"`
		Det []struct {
			NItem string `xml:"nItem,attr"`
			Prod  struct {
				CProd string `xml:"cProd"`
				NCM   string `xml:"NCM"`
				CFOP  string `xml:"CFOP"`
				QCom  string `xml:"qCom"`
				VUnCom string `xml:"vUnCom"`
				VProd string `xml:"vProd"`
			} `xml:"prod"`
			Imposto struct {
				ICMS struct {
					Orig  string `xml:"orig"`
					CST   string `xml:"CST"`
					VBC   string `xml:"vBC"`
					PICMS string `xml:"pICMS"`
					VICMS string `xml:"vICMS"`
				} `xml:"ICMS"`
				IPI struct {
					CST  string `xml:"CST"`
					VBC  string `xml:"vBC"`
					PIPI string `xml:"pIPI"`
					VIPI string `xml:"vIPI"`
				} `xml:"IPI"`
				PIS struct {
					CST  string `xml:"CST"`
					VBC  string `xml:"vBC"`
					PPIS string `xml:"pPIS"`
					VPIS string `xml:"vPIS"`
				} `xml:"PIS"`
				COFINS struct {
					CST      string `xml:"CST"`
					VBC      string `xml:"vBC"`
					PCOFINS  string `xml:"pCOFINS"`
					VCOFINS  string `xml:"vCOFINS"`
				} `xml:"COFINS"`
			} `xml:"imposto"`
		} `xml:"det"`
	} `xml:"NFe>infNFe"`
}

func (uc *UploadNFEEntryUseCase) Execute(ctx context.Context, dto request.UploadNFEDTO) (*entity.FiscalEntry, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	var nfe nfeXML
	if err := xml.Unmarshal([]byte(dto.XmlContent), &nfe); err != nil {
		return nil, fmt.Errorf("parsing NFe XML: %w", err)
	}

	inf := nfe.InfNFe

	nfNum, _ := strconv.ParseInt(inf.Ide.NF, 10, 64)
	dataEmissao := parseNFEDate(inf.Ide.DhEmi)

	entry := &entity.FiscalEntry{
		NumeroNF:            nfNum,
		Serie:               inf.Ide.Serie,
		Modelo:              inf.Ide.Mod,
		DataEmissao:         dataEmissao,
		DataEntrada:         dataEmissao,
		CnpjEmitente:        strings.TrimSpace(inf.Emit.CNPJ),
		RazaoSocialEmitente: inf.Emit.XNome,
		IEEmitente:          strPtr(inf.Emit.IE),
		UFEmitente:          strPtr(inf.Emit.UF),
		ValorProdutos:       parseFloat(inf.Total.ICMSTot.VProd),
		ValorFrete:          parseFloat(inf.Total.ICMSTot.VFrete),
		ValorSeguro:         parseFloat(inf.Total.ICMSTot.VSeg),
		ValorDesconto:       parseFloat(inf.Total.ICMSTot.VDesc),
		ValorIPI:            parseFloat(inf.Total.ICMSTot.VIPI),
		ValorICMS:           parseFloat(inf.Total.ICMSTot.VICMS),
		ValorPIS:            parseFloat(inf.Total.ICMSTot.VPIS),
		ValorCOFINS:         parseFloat(inf.Total.ICMSTot.VCOFINS),
		ValorTotal:          parseFloat(inf.Total.ICMSTot.VNF),
		TipoDocumento:       "NFE",
		Status:              entity.EntryStatusPending,
		CreatedBy:           userID,
	}

	created, err := uc.Repo.CreateEntry(ctx, entry)
	if err != nil {
		return nil, err
	}

	for i, det := range inf.Det {
		itemCode := int64PtrFromStr(det.Prod.CProd)
		item := &entity.FiscalEntryItem{
			FiscalEntryID: created.ID,
			Sequence:      i + 1,
			ItemCode:      itemCode,
			Ncm:           strPtr(det.Prod.NCM),
			Cfop:          det.Prod.CFOP,
			Quantity:      parseFloat(det.Prod.QCom),
			UnitPrice:     parseFloat(det.Prod.VUnCom),
			TotalPrice:    parseFloat(det.Prod.VProd),
			BaseICMS:      parseFloat(det.Imposto.ICMS.VBC),
			AliqICMS:      parseFloat(det.Imposto.ICMS.PICMS) / 100,
			ValorICMS:     parseFloat(det.Imposto.ICMS.VICMS),
			BaseIPI:       parseFloat(det.Imposto.IPI.VBC),
			AliqIPI:       parseFloat(det.Imposto.IPI.PIPI) / 100,
			ValorIPI:      parseFloat(det.Imposto.IPI.VIPI),
			ValorPIS:      parseFloat(det.Imposto.PIS.VPIS),
			ValorCOFINS:   parseFloat(det.Imposto.COFINS.VCOFINS),
			CstICMS:       strPtr(det.Imposto.ICMS.CST),
			CstIPI:        strPtr(det.Imposto.IPI.CST),
			CstPIS:        strPtr(det.Imposto.PIS.CST),
			CstCOFINS:     strPtr(det.Imposto.COFINS.CST),
			GeraCreditoICMS:   true,
			GeraCreditoIPI:    true,
			GeraCreditoPIS:    true,
			GeraCreditoCOFINS: true,
		}
		if _, err := uc.Repo.CreateEntryItem(ctx, item); err != nil {
			return nil, err
		}
	}

	items, _ := uc.Repo.GetEntryItems(ctx, created.ID)
	created.Itens = items

	return created, nil
}

func parseNFEDate(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	t, err := time.Parse("2006-01-02T15:04:05", s[:19])
	if err != nil {
		t, err = time.Parse("2006-01-02", s[:10])
		if err != nil {
			return time.Now()
		}
	}
	return t
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func int64PtrFromStr(s string) *int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
