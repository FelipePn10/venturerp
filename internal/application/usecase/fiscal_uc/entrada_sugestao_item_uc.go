package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// SugestaoItemEntradaUseCase dá ao lançamento manual o que o ERP sabe sobre o
// item comprado. Preço e impostos da nota são os que o FORNECEDOR imprimiu: aqui
// eles são só um ponto de partida para conferir com o papel, nunca a verdade.
// O caminho inverso também vale: do código que o fornecedor usa chega-se ao nosso
// item, pelo mesmo vínculo que concilia o XML.
type SugestaoItemEntradaUseCase struct {
	Docs   repository.FiscalEntryDocumentRepository
	Fiscal repository.FiscalRepository
	Auth   ports.AuthService
}

type ConsultaSugestaoItem struct {
	SupplierCode     *int64
	ItemCode         *int64
	CodigoFornecedor string
	UFEmitente       string
}

func (uc *SugestaoItemEntradaUseCase) Execute(ctx context.Context, q ConsultaSugestaoItem) (*response.SugestaoItemEntradaResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if q.ItemCode == nil && q.CodigoFornecedor == "" {
		return nil, errorsuc.NewValidationError("informe o item ou o código do fornecedor")
	}

	var vinculos []repository.ItemCandidate
	if q.SupplierCode != nil {
		var err error
		if vinculos, err = uc.Docs.SupplierLinksForEntry(ctx, *q.SupplierCode); err != nil {
			return nil, err
		}
	}
	vinculo := escolherVinculo(vinculos, q.ItemCode, q.CodigoFornecedor)
	itemCode := q.ItemCode
	if itemCode == nil {
		if vinculo == nil {
			return &response.SugestaoItemEntradaResponse{Encontrado: false}, nil
		}
		itemCode = &vinculo.Code
	}

	cadastro, err := uc.Docs.ItemsByCode(ctx, []int64{*itemCode})
	if err != nil {
		return nil, err
	}
	item, ok := cadastro[*itemCode]
	if !ok {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("o item %d não foi encontrado", *itemCode))
	}

	out := &response.SugestaoItemEntradaResponse{
		Encontrado: true, ItemCode: item.Code, Nome: item.Name, UOM: item.UOM, NCM: soDigitos(item.NCM),
	}
	if vinculo != nil {
		out.CodigoFornecedor = vinculo.SupplierItemCode
		out.DescricaoFornecedor = vinculo.SupplierDescription
		if vinculo.XMLUOM != nil {
			out.UOMFornecedor = *vinculo.XMLUOM
		}
		if vinculo.ConversionFactor != nil {
			f := vinculo.ConversionFactor.InexactFloat64()
			out.FatorConversao = &f
		}
	}
	if err := uc.preencherPreco(ctx, q.SupplierCode, item.Code, out); err != nil {
		return nil, err
	}
	if err := uc.preencherAliquotas(ctx, q.UFEmitente, item, out); err != nil {
		return nil, err
	}
	return out, nil
}

// escolherVinculo procura pelo código do fornecedor e, sem ele, pelo item. Código
// repetido entre itens fica sem vínculo: o que é ambíguo não concilia sozinho.
func escolherVinculo(vinculos []repository.ItemCandidate, itemCode *int64, codigo string) *repository.ItemCandidate {
	var achado *repository.ItemCandidate
	if c := normCodigo(codigo); c != "" {
		qtd := 0
		for i := range vinculos {
			if normCodigo(vinculos[i].SupplierItemCode) != c {
				continue
			}
			if itemCode != nil && vinculos[i].Code != *itemCode {
				continue
			}
			achado = &vinculos[i]
			qtd++
		}
		if qtd == 1 {
			return achado
		}
		return nil
	}
	if itemCode == nil {
		return nil
	}
	for i := range vinculos {
		if vinculos[i].Code == *itemCode && normCodigo(vinculos[i].SupplierItemCode) != "" {
			return &vinculos[i]
		}
	}
	return nil
}

// preencherPreco: o pedido em aberto é o preço combinado; sem ele, a última compra.
func (uc *SugestaoItemEntradaUseCase) preencherPreco(ctx context.Context, supplier *int64, item int64, out *response.SugestaoItemEntradaResponse) error {
	if supplier != nil {
		linhas, err := uc.Docs.OpenPurchaseOrderLines(ctx, *supplier, item)
		if err != nil {
			return err
		}
		if len(linhas) > 0 {
			l := linhas[0]
			p := l.UnitPrice.InexactFloat64()
			out.PrecoUnitario = &p
			out.OrigemPreco = fmt.Sprintf("Pedido de compra %d, linha %d", l.OrderNumber, l.Sequence)
			return nil
		}
	}
	ultimo, err := uc.Docs.UltimoPrecoDeCompra(ctx, supplier, item)
	if err != nil || ultimo == nil {
		return err
	}
	p := ultimo.Preco.InexactFloat64()
	out.PrecoUnitario = &p
	origem := "Última compra, NF %d, em %s"
	if ultimo.DoPedido {
		origem = "Último pedido de compra, nº %d, em %s"
	}
	out.OrigemPreco = fmt.Sprintf(origem, ultimo.Numero, ultimo.Data.Format("02/01/2006"))
	return nil
}

// preencherAliquotas consulta as tabelas fiscais: IPI pelo NCM e ICMS pela rota
// entre as UFs (mercadoria importada interestadual é 4%, Res. SF 13/2012).
func (uc *SugestaoItemEntradaUseCase) preencherAliquotas(ctx context.Context, ufEmitente string, item repository.ItemCandidate, out *response.SugestaoItemEntradaResponse) error {
	cem := decimal.NewFromInt(100)
	if out.NCM != "" {
		ipi, err := uc.Docs.NCMIPIRates(ctx, []string{out.NCM})
		if err != nil {
			return err
		}
		if a, ok := ipi[out.NCM]; ok && a.IsPositive() {
			v := a.Mul(cem).Round(4).InexactFloat64()
			out.AliqIPI = &v
		}
	}
	ufEmpresa, err := (&EntradaServico{Fiscal: uc.Fiscal}).ufEmpresa(ctx)
	if err != nil {
		return err
	}
	finalidade := entrada.FinalidadeDoTipoDeUso(item.TipoUso)
	if ufEmitente != "" {
		out.CFOP = entrada.CFOPEntradaComFinalidade("5101", "", finalidade, ufEmitente, ufEmpresa)
	}
	if ufEmitente == "" || ufEmitente == "EX" || ufEmpresa == "" {
		return nil
	}
	var aliq *float64
	switch {
	case ufEmitente == ufEmpresa:
		if aliq, _, err = uc.Fiscal.GetICMSInternal(ctx, ufEmitente); err != nil {
			return err
		}
	case origemImportada(item.Origem):
		quatro := 0.04
		aliq = &quatro
	default:
		if aliq, err = uc.Fiscal.GetICMSInterstate(ctx, ufEmitente, ufEmpresa); err != nil {
			return err
		}
	}
	if aliq != nil && *aliq > 0 {
		v := decimal.NewFromFloat(*aliq).Mul(cem).Round(4).InexactFloat64()
		out.AliqICMS = &v
	}
	return nil
}

func origemImportada(o string) bool {
	switch strings.TrimSpace(o) {
	case "1", "2", "3", "8":
		return true
	}
	return false
}
