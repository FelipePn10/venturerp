package fiscal_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type sugestaoDocs struct {
	repository.FiscalEntryDocumentRepository
	vinculos []repository.ItemCandidate
	itens    map[int64]repository.ItemCandidate
	linhas   []repository.PurchaseOrderLine
	ultimo   *repository.PrecoDeCompra
	ipi      map[string]decimal.Decimal
}

func (d sugestaoDocs) SupplierLinksForEntry(context.Context, int64) ([]repository.ItemCandidate, error) {
	return d.vinculos, nil
}
func (d sugestaoDocs) ItemsByCode(context.Context, []int64) (map[int64]repository.ItemCandidate, error) {
	return d.itens, nil
}
func (d sugestaoDocs) OpenPurchaseOrderLines(context.Context, int64, int64) ([]repository.PurchaseOrderLine, error) {
	return d.linhas, nil
}
func (d sugestaoDocs) UltimoPrecoDeCompra(context.Context, *int64, int64) (*repository.PrecoDeCompra, error) {
	return d.ultimo, nil
}
func (d sugestaoDocs) NCMIPIRates(context.Context, []string) (map[string]decimal.Decimal, error) {
	return d.ipi, nil
}

type sugestaoFiscal struct {
	repository.FiscalRepository
	uf string
}

func (f sugestaoFiscal) GetFiscalConfig(context.Context) (*entity.FiscalConfig, error) {
	return &entity.FiscalConfig{UFEmpresa: f.uf}, nil
}
func (f sugestaoFiscal) GetICMSInternal(context.Context, string) (*decimal.Decimal, *decimal.Decimal, error) {
	v := decimal.RequireFromString("0.18")
	return &v, nil, nil
}
func (f sugestaoFiscal) GetICMSInterstate(context.Context, string, string) (*decimal.Decimal, error) {
	v := decimal.RequireFromString("0.12")
	return &v, nil
}

type sugestaoAuth struct{ ports.AuthService }

func (sugestaoAuth) CanCreateFiscalEntry(context.Context) bool { return true }

func casoSugestao() *SugestaoItemEntradaUseCase {
	docs := sugestaoDocs{
		vinculos: []repository.ItemCandidate{{Code: 7, SupplierItemCode: "F1-7", SupplierDescription: "CHAPA 3MM"}},
		itens:    map[int64]repository.ItemCandidate{7: {Code: 7, Name: "Chapa", UOM: "UN", NCM: "7208.51.00", Origem: "0"}, 8: {Code: 8, Name: "Óleo", UOM: "LT", TipoUso: 1, Origem: "1"}},
		linhas:   []repository.PurchaseOrderLine{{OrderNumber: 55, Sequence: 2, UnitPrice: decimal.NewFromInt(90)}},
		ipi:      map[string]decimal.Decimal{"72085100": decimal.NewFromFloat(0.05)},
	}
	return &SugestaoItemEntradaUseCase{Docs: docs, Fiscal: sugestaoFiscal{uf: "SP"}, Auth: sugestaoAuth{}}
}

func TestSugestao_CodigoDoFornecedorChegaAoItemDoCadastro(t *testing.T) {
	forn := int64(1)
	out, err := casoSugestao().Execute(context.Background(), ConsultaSugestaoItem{SupplierCode: &forn, CodigoFornecedor: " f1-7 ", UFEmitente: "SP"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Encontrado || out.ItemCode != 7 || out.NCM != "72085100" || out.CodigoFornecedor != "F1-7" {
		t.Fatalf("vínculo não resolveu: %+v", out)
	}
	if out.PrecoUnitario == nil || *out.PrecoUnitario != 90 || out.OrigemPreco != "Pedido de compra 55, linha 2" {
		t.Fatalf("preço do pedido: %+v", out)
	}
	if out.AliqICMS == nil || *out.AliqICMS != 18 || out.AliqIPI == nil || *out.AliqIPI != 5 || out.CFOP != "1101" {
		t.Fatalf("alíquotas: icms=%v ipi=%v cfop=%s", out.AliqICMS, out.AliqIPI, out.CFOP)
	}
}

func TestSugestao_CodigoDesconhecidoNaoInventaItem(t *testing.T) {
	forn := int64(1)
	out, err := casoSugestao().Execute(context.Background(), ConsultaSugestaoItem{SupplierCode: &forn, CodigoFornecedor: "XYZ"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Encontrado {
		t.Fatalf("não devia achar: %+v", out)
	}
}

func TestSugestao_InterestadualImportadoEFinalidadeDeConsumo(t *testing.T) {
	item, forn := int64(8), int64(1)
	uc := casoSugestao()
	docs := uc.Docs.(sugestaoDocs)
	docs.linhas = nil
	uc.Docs = docs
	out, err := uc.Execute(context.Background(), ConsultaSugestaoItem{SupplierCode: &forn, ItemCode: &item, UFEmitente: "PR"})
	if err != nil {
		t.Fatal(err)
	}
	if out.AliqICMS == nil || *out.AliqICMS != 4 {
		t.Fatalf("importado interestadual é 4%%: %v", out.AliqICMS)
	}
	if out.CFOP != "2556" {
		t.Fatalf("consumo de fora do estado: %s", out.CFOP)
	}
	if out.PrecoUnitario != nil {
		t.Fatalf("sem histórico não há preço: %v", *out.PrecoUnitario)
	}
}

func TestConciliacao_ItemEscolhidoUsaConversaoDoVinculo(t *testing.T) {
	fator, link := decimal.NewFromInt(12), int64(99)
	code, codigo, uom := int64(7), "f1-7", "CX"
	it := &entity.FiscalEntryItem{ItemCode: &code, SupplierItemCode: &codigo, UOM: &uom, Quantity: decimal.NewFromFloat(2)}
	usarVinculoDoItemEscolhido(it, map[string][]repository.ItemCandidate{"F1-7": {{Code: 7, ItemSupplierID: &link, ConversionFactor: &fator, XMLUOM: &uom}}})
	if it.ItemSupplierID == nil || *it.ItemSupplierID != 99 || !it.QuantidadeEstoque.Equal(decimal.NewFromInt(24)) {
		t.Fatalf("vínculo/conversão: %+v", it)
	}
}
