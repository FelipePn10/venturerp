package fiscal_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
)

// clientesFake implementa só o que o resolvedor usa; o resto da interface fica
// embutido e nunca é chamado.
type clientesFake struct {
	customerrepo.CustomerRepository
	cliente   *customerentity.Customer
	enderecos []*customerentity.CustomerAddress
}

func (c *clientesFake) GetCustomerByCode(_ context.Context, code int64) (*customerentity.Customer, error) {
	if c.cliente == nil || c.cliente.Code != code {
		return nil, errNaoAchou
	}
	return c.cliente, nil
}

func (c *clientesFake) ListAddresses(_ context.Context, _ int64) ([]*customerentity.CustomerAddress, error) {
	return c.enderecos, nil
}

var errNaoAchou = &naoAchou{}

type naoAchou struct{}

func (*naoAchou) Error() string { return "cliente não encontrado" }

func clienteComEnderecos(enderecos ...*customerentity.CustomerAddress) *clientesFake {
	return &clientesFake{
		cliente: &customerentity.Customer{
			ID: 42, Code: 7, Name: "METALURGICA CLIENTE LTDA",
			DocumentType: customerentity.DocumentCNPJ, DocumentNumber: "11222333000181",
			StateRegistration: strp("9012345678"),
		},
		enderecos: enderecos,
	}
}

func endereco(tipo customerentity.AddressType, rua, cidade, uf string) *customerentity.CustomerAddress {
	return &customerentity.CustomerAddress{
		AddressType: tipo, Street: strp(rua), Number: strp("100"),
		Neighborhood: strp("CENTRO"), City: strp(cidade), UF: strp(uf), ZipCode: strp("13050-000"),
	}
}

// A mercadoria vai para o endereço de ENTREGA — é o que a NF-e pede. Usar o de
// cobrança faria a nota apontar para o escritório em vez da fábrica.
func TestEnderecoDeEntregaTemPreferencia(t *testing.T) {
	clientes := clienteComEnderecos(
		endereco(customerentity.AddressCobranca, "RUA DO ESCRITORIO", "SAO PAULO", "SP"),
		endereco(customerentity.AddressEntrega, "RUA DA FABRICA", "CAMPINAS", "SP"),
	)
	dto := request.CreateFiscalExitDTO{CustomerCode: i64p(7)}
	resolverDestinatario(context.Background(), &dto, clientes, nil)

	if dto.DestLogradouro == nil || *dto.DestLogradouro != "RUA DA FABRICA" {
		t.Fatalf("deveria usar o endereço de entrega, veio %v", dto.DestLogradouro)
	}
	if dto.DestMunicipio == nil || *dto.DestMunicipio != "CAMPINAS" {
		t.Fatalf("município errado: %v", dto.DestMunicipio)
	}
	if dto.DestCEP == nil || *dto.DestCEP != "13050000" {
		t.Fatalf("o CEP tem de ir só com dígitos, veio %v", dto.DestCEP)
	}
	if dto.RazaoSocialDestinatario == nil || *dto.RazaoSocialDestinatario != "METALURGICA CLIENTE LTDA" {
		t.Fatalf("razão social não veio do cadastro: %v", dto.RazaoSocialDestinatario)
	}
	if dto.IEDestinatario == nil || *dto.IEDestinatario != "9012345678" {
		t.Fatalf("inscrição estadual não veio do cadastro: %v", dto.IEDestinatario)
	}
}

// Quase todo cadastro tem só endereço de cobrança. Recusar a nota nesse caso
// deixaria a empresa sem faturar por um cadastro que ela considera completo.
func TestSemEntregaUsaCobranca(t *testing.T) {
	clientes := clienteComEnderecos(endereco(customerentity.AddressCobranca, "RUA DO ESCRITORIO", "SAO PAULO", "SP"))
	dto := request.CreateFiscalExitDTO{CustomerCode: i64p(7)}
	resolverDestinatario(context.Background(), &dto, clientes, nil)
	if dto.DestLogradouro == nil || *dto.DestLogradouro != "RUA DO ESCRITORIO" {
		t.Fatalf("deveria cair no endereço de cobrança, veio %v", dto.DestLogradouro)
	}
}

// Uma entrega pontual num endereço diferente é digitada na nota. O cadastro não
// pode sobrescrever o que o usuário informou.
func TestNaoSobrescreveOQueFoiDigitadoNaNota(t *testing.T) {
	clientes := clienteComEnderecos(endereco(customerentity.AddressEntrega, "RUA DA FABRICA", "CAMPINAS", "SP"))
	dto := request.CreateFiscalExitDTO{
		CustomerCode:   i64p(7),
		DestLogradouro: strp("RUA DA OBRA"),
		DestMunicipio:  strp("JUNDIAI"),
	}
	resolverDestinatario(context.Background(), &dto, clientes, nil)
	if *dto.DestLogradouro != "RUA DA OBRA" || *dto.DestMunicipio != "JUNDIAI" {
		t.Fatalf("o que foi digitado na nota tem de prevalecer: %v / %v", *dto.DestLogradouro, *dto.DestMunicipio)
	}
	// O que ficou vazio continua sendo completado pelo cadastro.
	if dto.DestBairro == nil || *dto.DestBairro != "CENTRO" {
		t.Fatalf("campo vazio deveria ser completado: %v", dto.DestBairro)
	}
}

func TestClienteSemEnderecoNaoQuebra(t *testing.T) {
	clientes := clienteComEnderecos()
	dto := request.CreateFiscalExitDTO{CustomerCode: i64p(7)}
	resolverDestinatario(context.Background(), &dto, clientes, nil)
	if dto.DestLogradouro != nil {
		t.Fatalf("sem endereço cadastrado nada deveria ser preenchido: %v", dto.DestLogradouro)
	}
	if dto.RazaoSocialDestinatario == nil {
		t.Fatal("o nome do cliente deveria vir mesmo sem endereço")
	}
}
