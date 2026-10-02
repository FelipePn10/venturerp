package fiscal_uc

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type UploadNFEEntryUseCase struct {
	Repo           repository.FiscalRepository
	Auth           ports.AuthService
	PurchaseOrders porepo.PurchaseOrderRepository
	Tolerances     ports.PurchaseToleranceEvaluator
	SupplierItems  ports.ItemSupplierResolver
}

// nfeProcXML é o envelope que a SEFAZ devolve quando autoriza a nota: o arquivo
// que o contador manda e que o usuário tem no disco é quase sempre
// <nfeProc><NFe>…</NFe><protNFe>…</protNFe></nfeProc>, não <NFe> na raiz.
type nfeProcXML struct {
	XMLName xml.Name `xml:"nfeProc"`
	NFe     nfeXML   `xml:"NFe"`
}

// lerXMLdeNFe aceita a nota com <NFe> na raiz e também dentro do envelope
// <nfeProc>, recusando qualquer outra coisa.
//
// ⚠️ Aceitar só <NFe> na raiz recusava justamente o arquivo mais comum. Como
// `nfeXML` declara `XMLName xml.Name "NFe"`, o decodificador do Go exige a raiz:
// um nfeProc legítimo falhava com "expected element type <NFe> but have <nfeProc>"
// e o usuário recebia "não é um XML de NF-e válido" olhando para o XML correto da
// nota dele. (A mesma checagem de raiz é o que FAZ `<foo/>` ser recusado — por isso
// não basta trocar a mensagem: o envelope precisa ser entendido.)
func lerXMLdeNFe(conteudo string) (nfeXML, error) {
	recusa := errorsuc.NewValidationError(
		"o arquivo enviado não é um XML de NF-e: a raiz precisa ser <NFe> ou <nfeProc>. " +
			"Confira se é o XML da nota (não o DANFE em PDF) e se o conteúdo não está truncado")

	bruto := strings.TrimSpace(conteudo)
	if bruto == "" {
		return nfeXML{}, errorsuc.NewValidationError("envie o conteúdo do XML da nota")
	}

	var direta nfeXML
	if err := xml.Unmarshal([]byte(bruto), &direta); err == nil {
		return direta, nil
	}
	var envelope nfeProcXML
	if err := xml.Unmarshal([]byte(bruto), &envelope); err == nil {
		// Envelope sem a nota dentro não serve: seguiria com a estrutura zerada e
		// gravaria uma entrada fiscal vazia, que é o defeito que a conferência da
		// chave de acesso abaixo impede.
		if strings.TrimSpace(envelope.NFe.InfNFe.Ide.NF) == "" {
			return nfeXML{}, errorsuc.NewValidationError(
				"o XML tem o envelope <nfeProc> mas não contém a nota (<NFe>) dentro dele")
		}
		return envelope.NFe, nil
	}
	return nfeXML{}, recusa
}

type nfeXML struct {
	XMLName xml.Name `xml:"NFe"`
	InfNFe  struct {
		Ide struct {
			NUF   string `xml:"cUF"`
			NF    string `xml:"nNF"`
			Serie string `xml:"serie"`
			Mod   string `xml:"mod"`
			DhEmi string `xml:"dhEmi"`
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
				CProd  string `xml:"cProd"`
				XProd  string `xml:"xProd"`
				NCM    string `xml:"NCM"`
				CFOP   string `xml:"CFOP"`
				QCom   string `xml:"qCom"`
				VUnCom string `xml:"vUnCom"`
				UCom   string `xml:"uCom"`
				VProd  string `xml:"vProd"`
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
					CST     string `xml:"CST"`
					VBC     string `xml:"vBC"`
					PCOFINS string `xml:"pCOFINS"`
					VCOFINS string `xml:"vCOFINS"`
				} `xml:"COFINS"`
			} `xml:"imposto"`
		} `xml:"det"`
		// ⚠️ Era `xml:"NFe>infNFe"` COM a raiz já declarada como NFe acima, ou seja,
		// o decodificador procurava <NFe><NFe><infNFe>, que não existe em nota
		// nenhuma. O arquivo era ACEITO (a raiz casava) e a estrutura ficava toda
		// zerada: a entrada fiscal era gravada com número 0, CNPJ vazio e totais
		// zero, sem erro. A nota real é <NFe><infNFe>…</infNFe></NFe>.
	} `xml:"infNFe"`
}

func (uc *UploadNFEEntryUseCase) Execute(ctx context.Context, dto request.UploadNFEDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}

	nfe, err := lerXMLdeNFe(dto.XmlContent)
	if err != nil {
		return nil, err
	}

	inf := nfe.InfNFe

	// Trava de conteúdo, não só de formato: nota sem número e sem CNPJ do emitente
	// não é nota. Sem isto, qualquer mudança no caminho do XML volta a gravar
	// entrada fiscal zerada em silêncio — foi exatamente o que aconteceu enquanto o
	// campo apontava para <NFe><NFe><infNFe>. A mesma lição do importador de OFX.
	numeroTexto := strings.TrimSpace(inf.Ide.NF)
	cnpjEmitente := strings.TrimSpace(inf.Emit.CNPJ)
	if numeroTexto == "" || cnpjEmitente == "" {
		return nil, errorsuc.NewValidationError(
			"o XML foi lido mas não traz os dados da nota (número e CNPJ do emitente). " +
				"Confira se o arquivo é o XML completo da NF-e e não está truncado")
	}

	nfNum, err := strconv.ParseInt(numeroTexto, 10, 64)
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"o número da nota no XML (%q) não é um número", numeroTexto))
	}
	dataEmissao := parseNFEDate(inf.Ide.DhEmi)

	entry := &entity.FiscalEntry{
		EnterpriseID:        enterpriseID,
		NumeroNF:            nfNum,
		Serie:               inf.Ide.Serie,
		Modelo:              inf.Ide.Mod,
		DataEmissao:         dataEmissao,
		DataEntrada:         dataEmissao,
		CnpjEmitente:        cnpjEmitente,
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
		PurchaseOrderCode:   dto.PurchaseOrderCode,
		Status:              entity.EntryStatusPending,
		CreatedBy:           userID,
	}

	pendingItems := make([]*entity.FiscalEntryItem, 0, len(inf.Det))
	for i, det := range inf.Det {
		itemCode := int64PtrFromStr(det.Prod.CProd)
		item := &entity.FiscalEntryItem{
			Sequence:          i + 1,
			ItemCode:          itemCode,
			SupplierItemCode:  strPtr(det.Prod.CProd),
			Description:       strPtr(det.Prod.XProd),
			UOM:               strPtr(det.Prod.UCom),
			Ncm:               strPtr(det.Prod.NCM),
			Cfop:              det.Prod.CFOP,
			Quantity:          parseFloat(det.Prod.QCom),
			UnitPrice:         parseFloat(det.Prod.VUnCom),
			TotalPrice:        parseFloat(det.Prod.VProd),
			BaseICMS:          parseFloat(det.Imposto.ICMS.VBC),
			AliqICMS:          parseFloat(det.Imposto.ICMS.PICMS) / 100,
			ValorICMS:         parseFloat(det.Imposto.ICMS.VICMS),
			BaseIPI:           parseFloat(det.Imposto.IPI.VBC),
			AliqIPI:           parseFloat(det.Imposto.IPI.PIPI) / 100,
			ValorIPI:          parseFloat(det.Imposto.IPI.VIPI),
			ValorPIS:          parseFloat(det.Imposto.PIS.VPIS),
			ValorCOFINS:       parseFloat(det.Imposto.COFINS.VCOFINS),
			CstICMS:           strPtr(det.Imposto.ICMS.CST),
			CstIPI:            strPtr(det.Imposto.IPI.CST),
			CstPIS:            strPtr(det.Imposto.PIS.CST),
			CstCOFINS:         strPtr(det.Imposto.COFINS.CST),
			GeraCreditoICMS:   true,
			GeraCreditoIPI:    true,
			GeraCreditoPIS:    true,
			GeraCreditoCOFINS: true,
		}
		pendingItems = append(pendingItems, item)
	}
	entry.SupplierCode, entry.Warnings, err = validatePurchaseEntryTolerances(ctx, uc.PurchaseOrders, uc.Tolerances, dto.PurchaseOrderCode, pendingItems, entry.ValorProdutos)
	if err != nil {
		return nil, err
	}
	if err = resolveSupplierItems(ctx, uc.SupplierItems, entry.SupplierCode, pendingItems); err != nil {
		return nil, err
	}
	created, err := uc.Repo.CreateEntry(ctx, entry)
	if err != nil {
		return nil, err
	}
	for _, item := range pendingItems {
		item.FiscalEntryID = created.ID
		if _, err = uc.Repo.CreateEntryItem(ctx, item); err != nil {
			return nil, err
		}
	}

	items, _ := uc.Repo.GetEntryItems(ctx, created.ID)
	created.Itens = items

	return toFiscalEntryResponse(created), nil
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
