package nfse_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	financialentity "github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	nfseentity "github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
)

type clientesFake map[int64]*customerentity.Customer

func (c clientesFake) GetCustomerByCode(_ context.Context, code int64) (*customerentity.Customer, error) {
	if cli, ok := c[code]; ok {
		return cli, nil
	}
	return nil, errorsuc.NewNotFoundError("cliente não encontrado")
}

type titulosFake struct {
	titulos    []*financialentity.ContaReceber
	gerado     *financialentity.ContaReceber
	cancelados int
}

func (t *titulosFake) TitulosDaNFSe(context.Context, int64) ([]*financialentity.ContaReceber, error) {
	return t.titulos, nil
}
func (t *titulosFake) GerarTituloDaNFSe(_ context.Context, _ int64, c *financialentity.ContaReceber) (bool, error) {
	t.gerado = c
	return true, nil
}
func (t *titulosFake) CancelarTitulosDaNFSe(context.Context, int64) error {
	t.cancelados++
	return nil
}

type authCancelamento struct{ fakeNFSeAuth }

func (authCancelamento) CanAuthorizeFiscalExit(context.Context) bool { return true }

func codigo(v int64) *int64 { return &v }

// Com cliente, a NFS-e guarda o cliente e o vencimento (padrão emissão + 30)
// e preenche o tomador em branco com o cadastro.
func TestNFSeComClienteConfereEPreencheOTomador(t *testing.T) {
	repo := &fakeNFSeRepo{}
	uc := &CreateNFSeUseCase{Repo: repo, Auth: &fakeNFSeAuth{canCreate: true}, Clientes: clientesFake{
		7: {Code: 7, Name: "Cliente Sete Ltda", DocumentNumber: "11.444.777/0001-61"},
	}}
	dto := validCreateDTO()
	dto.CustomerCode = codigo(7)
	if _, err := uc.Execute(context.Background(), dto); err != nil {
		t.Fatal(err)
	}
	n := repo.created
	if *n.CustomerCode != 7 || *n.TomadorCnpjCpf != "11444777000161" || *n.TomadorRazaoSocial != "Cliente Sete Ltda" {
		t.Fatalf("cliente/tomador = %v %v %v", *n.CustomerCode, *n.TomadorCnpjCpf, *n.TomadorRazaoSocial)
	}
	if !n.DataVencimento.Equal(time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("vencimento = %v, esperado emissão + 30 dias", n.DataVencimento)
	}
}

func TestNFSeRecusaClienteDeOutraEmpresaEVencimentoAntesDaEmissao(t *testing.T) {
	uc := &CreateNFSeUseCase{Repo: &fakeNFSeRepo{}, Auth: &fakeNFSeAuth{canCreate: true}, Clientes: clientesFake{}}
	dto := validCreateDTO()
	dto.CustomerCode = codigo(99)
	if _, err := uc.Execute(context.Background(), dto); err == nil || !strings.Contains(err.Error(), "cliente 99") {
		t.Fatalf("cliente inexistente: %v", err)
	}
	dto = validCreateDTO()
	antes := "2026-06-01"
	dto.DataVencimento = &antes
	if _, err := uc.Execute(context.Background(), dto); err == nil || !strings.Contains(err.Error(), "vencimento") {
		t.Fatalf("vencimento antes da emissão: %v", err)
	}
}

// O título é o valor líquido (o ISS retido o tomador recolhe), com o número
// da nota e o vencimento gravado.
func TestTituloDaNFSePeloValorLiquido(t *testing.T) {
	num := "6"
	venc := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)
	n := &nfseentity.NFSe{ID: 3, NumeroNFSe: &num, CustomerCode: codigo(7), DataEmissao: time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC),
		DataVencimento: &venc, ValorServicos: decimal.NewFromInt(1000), ValorLiquido: decimal.NewFromInt(970)}
	c := tituloDaNFSe(n, [16]byte{}, time.Now())
	if *c.NumeroDocumento != "NFS-e 6" || *c.ClienteID != 7 || !c.ValorBruto.Equal(decimal.NewFromInt(970)) || !c.DataVencimento.Equal(venc) {
		t.Fatalf("título = %s cliente %d valor %s venc %v", *c.NumeroDocumento, *c.ClienteID, c.ValorBruto, c.DataVencimento)
	}
}

// Título recebido impede o cancelamento antes de chamar a prefeitura.
func TestCancelarNFSeComTituloRecebidoERecusado(t *testing.T) {
	ref := "nfse1"
	repo := &fakeNFSeRepo{created: &nfseentity.NFSe{ID: 1, Status: nfseentity.NFSeStatusAutorizada, FocusRef: &ref}}
	doc := "NFS-e 6"
	titulos := &titulosFake{titulos: []*financialentity.ContaReceber{{NumeroDocumento: &doc, ValorRecebido: decimal.NewFromInt(100)}}}
	uc := &CancelNFSeUseCase{Repo: repo, Auth: &authCancelamento{}, Titulos: titulos}
	_, err := uc.Execute(context.Background(), 1, request.CancelNFSeDTO{Justificativa: "serviço não foi prestado ao cliente"})
	if err == nil || !strings.Contains(err.Error(), "estorne o recebimento") {
		t.Fatalf("erro = %v", err)
	}
	if repo.created.Status != nfseentity.NFSeStatusAutorizada || titulos.cancelados != 0 {
		t.Fatalf("a nota ou o título mudaram: %s, %d", repo.created.Status, titulos.cancelados)
	}
}
