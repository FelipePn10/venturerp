package fiscal_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

type leitorDaRemessaFake struct {
	uf      string
	ligado  int64
	recusar bool
}

func (l *leitorDaRemessaFake) RemessaDaOS(context.Context, int64, *int64) (*RemessaParaIndustrializar, error) {
	return &RemessaParaIndustrializar{OSID: 5, OSCode: 12, ItemCode: 601, Descricao: "Eixo", NCM: "84839000", Unidade: "UN",
		Fornecedor: 90, CustoMedio: decimal.RequireFromString("12.5"), PrevisaoRetorno: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)}, nil
}
func (l *leitorDaRemessaFake) FornecedorDestinatario(context.Context, int64) (*DestinatarioDaRemessa, error) {
	return &DestinatarioDaRemessa{CNPJ: "11222333000181", RazaoSocial: "TRATE", UF: l.uf, Logradouro: "Rua A", Numero: "1", Bairro: "Centro", Municipio: "Cidade", CEP: "01001000"}, nil
}
func (l *leitorDaRemessaFake) LigarNotaAoMovimento(_ context.Context, _, _, exitID int64) error {
	l.ligado = exitID
	return nil
}

func (r *loadFiscalRepo) GetExitByID(_ context.Context, _ int64) (*fiscalentity.FiscalExit, error) {
	return r.created, nil
}

// Bug I: a remessa para o terceiro sai com NF-e 5901 (6901 fora da UF), sem
// cobrança, com a peça ao custo médio e ICMS/IPI suspensos.
func TestNotaDeRemessaParaIndustrializacao(t *testing.T) {
	for _, c := range []struct{ uf, cfop string }{{"SP", "5901"}, {"PR", "6901"}} {
		repo := &loadFiscalRepo{nextNF: 10}
		leitor := &leitorDaRemessaFake{uf: c.uf}
		uc := &RemessaIndustrializacaoUseCase{Repo: repo, Auth: loadFiscalAuth{uid: uuid.New()}, Leitor: leitor}
		mov := int64(3)
		if _, err := uc.Criar(context.Background(), 5, RemessaIndustrializacaoDTO{MovementID: &mov, Quantity: "4", DataEmissao: "2026-11-21"}); err != nil {
			t.Fatal(err)
		}
		x, it := repo.created, repo.items[0]
		if x.Cfop != c.cfop || it.Cfop != c.cfop || !it.TotalPrice.Equal(decimal.NewFromInt(50)) || *it.CstICMS != "50" || *it.CstIPI != "55" {
			t.Fatalf("UF %s: cfop=%s/%s total=%s cst=%s/%s", c.uf, x.Cfop, it.Cfop, it.TotalPrice, *it.CstICMS, *it.CstIPI)
		}
		if x.SourceType == nil || *x.SourceType != SourceRemessaIndustrializacao || x.SupplierCode == nil || *x.SupplierCode != 90 {
			t.Fatalf("origem/fornecedor da nota: %+v", x)
		}
		if !fiscalentity.ValorCobrado(x, repo.items).IsZero() {
			t.Fatal("a remessa não é cobrança: sem duplicata")
		}
		if leitor.ligado != 77 {
			t.Fatal("a nota não foi ligada ao movimento de remessa")
		}
	}
}

// A tributação da remessa segue o que a contadora definir; código inválido é
// recusado ANTES de criar a nota (sem rascunho órfão).
func TestRemessaComTributacaoInformada(t *testing.T) {
	repo := &loadFiscalRepo{nextNF: 10}
	uc := &RemessaIndustrializacaoUseCase{Repo: repo, Auth: loadFiscalAuth{uid: uuid.New()}, Leitor: &leitorDaRemessaFake{uf: "SP"}}
	if _, err := uc.Criar(context.Background(), 5, RemessaIndustrializacaoDTO{Quantity: "1", DataEmissao: "2026-11-21", CSTICMS: "5x"}); err == nil || repo.created != nil {
		t.Fatalf("CST inválido: err=%v nota criada=%v", err, repo.created != nil)
	}
	if _, err := uc.Criar(context.Background(), 5, RemessaIndustrializacaoDTO{Quantity: "1", DataEmissao: "2026-11-21", CSTICMS: "41", CBenef: "SP000001", TextoLegal: "Suspensão art. 402 RICMS"}); err != nil {
		t.Fatal(err)
	}
	it := repo.items[0]
	if *it.CstICMS != "41" || *it.CstIPI != "55" || it.CodigoBeneficioFiscal == nil || *it.CodigoBeneficioFiscal != "SP000001" ||
		!strings.Contains(*repo.created.InformacoesComplementares, "art. 402") {
		t.Fatalf("linha: cst=%s ipi=%s cbenef=%v info=%s", *it.CstICMS, *it.CstIPI, it.CodigoBeneficioFiscal, *repo.created.InformacoesComplementares)
	}
}
