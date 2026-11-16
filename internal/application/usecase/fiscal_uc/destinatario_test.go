package fiscal_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
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

// O destinatário é o endereço cadastral (o da IE); a fábrica onde a mercadoria
// é entregue vai no grupo de local de entrega — antes a nota apontava o
// destinatário para a entrega, e uma entrega em outra UF mudaria CFOP e DIFAL.
func TestDestinatarioCadastralEEntregaNoGrupoProprio(t *testing.T) {
	clientes := clienteComEnderecos(
		endereco(customerentity.AddressEntrega, "RUA DA FABRICA", "CAMPINAS", "SP"),
		endereco(customerentity.AddressCobranca, "RUA DO ESCRITORIO", "SAO PAULO", "SP"),
	)
	dto := request.CreateFiscalExitDTO{CustomerCode: i64p(7)}
	if err := resolverDestinatario(context.Background(), &dto, clientes, nil); err != nil {
		t.Fatal(err)
	}
	if dto.DestLogradouro == nil || *dto.DestLogradouro != "RUA DO ESCRITORIO" {
		t.Fatalf("o destinatário deveria ser o endereço cadastral, veio %v", dto.DestLogradouro)
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
	entrega, err := localDeEntrega(context.Background(), &dto, clientes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if entrega == nil || entrega.Logradouro != "RUA DA FABRICA" || entrega.Municipio != "CAMPINAS" || entrega.CNPJCPF != "11222333000181" || entrega.CEP != "13050000" {
		t.Fatalf("local de entrega: %+v", entrega)
	}
}

// Comercial vem antes de cobrança; no mesmo tipo, o padrão.
func TestOrdemDoEnderecoCadastral(t *testing.T) {
	outraCobranca := endereco(customerentity.AddressCobranca, "RUA COBRANCA PADRAO", "SAO PAULO", "SP")
	outraCobranca.IsDefault = true
	clientes := clienteComEnderecos(
		endereco(customerentity.AddressCobranca, "RUA COBRANCA", "SAO PAULO", "SP"), outraCobranca,
		endereco(customerentity.AddressComercial, "RUA COMERCIAL", "SAO PAULO", "SP"),
	)
	end, _ := enderecoDaNota(context.Background(), clientes, clientes.cliente)
	if end == nil || *end.Street != "RUA COMERCIAL" {
		t.Fatalf("comercial deveria vencer: %v", end)
	}
	clientes.enderecos = clientes.enderecos[:2]
	if end, _ := enderecoDaNota(context.Background(), clientes, clientes.cliente); end == nil || *end.Street != "RUA COBRANCA PADRAO" {
		t.Fatalf("no mesmo tipo, o padrão: %v", end)
	}
}

// Cadastro só com endereço de entrega (o legado da maioria) continua emitindo
// igual: ele é o destinatário e não há grupo de entrega repetindo o endereço.
func TestSoEntregaViraDestinatarioSemGrupo(t *testing.T) {
	clientes := clienteComEnderecos(endereco(customerentity.AddressEntrega, "RUA DA FABRICA", "CAMPINAS", "SP"))
	dto := request.CreateFiscalExitDTO{CustomerCode: i64p(7)}
	if err := resolverDestinatario(context.Background(), &dto, clientes, nil); err != nil {
		t.Fatal(err)
	}
	if dto.DestLogradouro == nil || *dto.DestLogradouro != "RUA DA FABRICA" {
		t.Fatalf("sem outro endereço, o de entrega é o do destinatário: %v", dto.DestLogradouro)
	}
	if e, _ := localDeEntrega(context.Background(), &dto, clientes, nil); e != nil {
		t.Fatalf("entrega igual ao destinatário não vai na nota: %+v", e)
	}
}

// Frete e transportadora vêm do cadastro do cliente quando a nota não diz.
func TestFreteETransportadoraDoCliente(t *testing.T) {
	clientes := clienteComEnderecos(endereco(customerentity.AddressCobranca, "RUA DO ESCRITORIO", "SAO PAULO", "SP"))
	clientes.cliente.FreightType = strp("FOB")
	clientes.cliente.ShippingCarrierID = i64p(3)
	dto := request.CreateFiscalExitDTO{CustomerCode: i64p(7)}
	_ = resolverDestinatario(context.Background(), &dto, clientes, nil)
	if dto.TipoFrete == nil || *dto.TipoFrete != "FOB" || dto.ShippingCarrierID == nil || *dto.ShippingCarrierID != 3 {
		t.Fatalf("frete %v transportadora %v", dto.TipoFrete, dto.ShippingCarrierID)
	}
	dto2 := request.CreateFiscalExitDTO{CustomerCode: i64p(7), TipoFrete: strp("CIF"), ShippingCarrierID: i64p(9)}
	_ = resolverDestinatario(context.Background(), &dto2, clientes, nil)
	if *dto2.TipoFrete != "CIF" || *dto2.ShippingCarrierID != 9 {
		t.Fatal("o informado na nota prevalece sobre o cadastro")
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

func (c *clientesFake) GetCustomerByDocument(_ context.Context, doc string) (*customerentity.Customer, error) {
	if c.cliente == nil || c.cliente.DocumentNumber != doc {
		return nil, errNaoAchouDoc
	}
	return c.cliente, nil
}

var errNaoAchouDoc = errorsuc.NewNotFoundError("cliente não encontrado para o documento informado")

// Nota avulsa digitada com o CNPJ de um cliente cadastrado fica ligada a ele;
// documento desconhecido segue sem cliente.
func TestNotaAvulsaPeloCNPJLigaAoCliente(t *testing.T) {
	clientes := clienteComEnderecos(endereco(customerentity.AddressCobranca, "RUA DO ESCRITORIO", "SAO PAULO", "SP"))
	dto := request.CreateFiscalExitDTO{CnpjDestinatario: strp("11.222.333/0001-81")}
	if err := resolverDestinatario(context.Background(), &dto, clientes, nil); err != nil {
		t.Fatal(err)
	}
	if dto.CustomerCode == nil || *dto.CustomerCode != 7 || dto.DestLogradouro == nil {
		t.Fatalf("nota não foi ligada ao cliente: %v %v", dto.CustomerCode, dto.DestLogradouro)
	}
	outro := request.CreateFiscalExitDTO{CnpjDestinatario: strp("99.999.999/0001-91")}
	if err := resolverDestinatario(context.Background(), &outro, clientes, nil); err != nil || outro.CustomerCode != nil {
		t.Fatalf("documento desconhecido: %v %v", outro.CustomerCode, err)
	}
}

type pedidoComFrete struct {
	salesrepo.SalesOrderRepository
	frete string
}

func (p pedidoComFrete) GetByCode(_ context.Context, code int64) (*salesentity.SalesOrder, error) {
	c := int64(7)
	return &salesentity.SalesOrder{Code: code, CustomerCode: &c, FreightType: &p.frete}, nil
}

// O pedido convertido do orçamento traz o tipo comercial ("FOB CONTRAT."): a
// nota o traduz, e um tipo sem equivalente não trava o faturamento.
func TestFreteComercialDoPedido(t *testing.T) {
	clientes := clienteComEnderecos(endereco(customerentity.AddressCobranca, "RUA DO ESCRITORIO", "SAO PAULO", "SP"))
	for frete, quer := range map[string]string{"FOB CONTRAT.": "FOB", "RETIRA": "FOB", "CONVENIO": ""} {
		dto := request.CreateFiscalExitDTO{SalesOrderCode: i64p(1)}
		if err := resolverDestinatario(context.Background(), &dto, clientes, pedidoComFrete{frete: frete}); err != nil {
			t.Fatal(err)
		}
		if got := valorOuVazio(dto.TipoFrete); got != quer {
			t.Errorf("%s: %q, quer %q", frete, got, quer)
		}
	}
}
