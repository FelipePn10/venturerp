package fiscal_uc

import (
	"context"
	"strings"

	"fmt"
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

// resolverDestinatario completa os dados do destinatário da NF-e a partir do
// cadastro do cliente.
//
// A nota é digitada (ou gerada por carga) com CNPJ, razão social, IE e UF, mas o
// layout da NF-e exige o endereço completo do destinatário. Ninguém digita
// logradouro/bairro/município/CEP a cada nota, e a nota tirada do pedido já sabe
// quem é o cliente: o endereço vem do cadastro.
//
// A preferência é o endereço de ENTREGA — é para lá que a mercadoria vai e é o
// que a NF-e pede. Sem ele, cai no de COBRANÇA (o único que a maioria dos
// cadastros tem), e depois em qualquer um marcado como padrão.
//
// O que o usuário digitou na nota nunca é sobrescrito: só campo vazio é
// preenchido. Uma entrega pontual num endereço diferente continua valendo.
func resolverDestinatario(
	ctx context.Context,
	dto *request.CreateFiscalExitDTO,
	customers customerrepo.CustomerRepository,
	orders salesrepo.SalesOrderRepository,
) {
	if dto == nil || customers == nil {
		return
	}

	customerCode := dto.CustomerCode
	if customerCode == nil && dto.SalesOrderCode != nil && orders != nil {
		if order, err := orders.GetByCode(ctx, *dto.SalesOrderCode); err == nil && order != nil && order.CustomerCode != nil {
			c := *order.CustomerCode
			customerCode = &c
		}
	}
	if customerCode == nil {
		return
	}

	cliente, err := customers.GetCustomerByCode(ctx, *customerCode)
	if err != nil || cliente == nil {
		return
	}
	dto.CustomerCode = customerCode

	preencher(&dto.RazaoSocialDestinatario, cliente.Name)
	preencher(&dto.CnpjDestinatario, cliente.DocumentNumber)
	if cliente.StateRegistration != nil {
		preencher(&dto.IEDestinatario, *cliente.StateRegistration)
	}
	if dto.TipoPessoa == nil && cliente.DocumentType == customerentity.DocumentCPF {
		f := "F"
		dto.TipoPessoa = &f
	}
	preencher(&dto.DestEmail, primeiroEmailDoCliente(cliente))

	end := enderecoDaNota(ctx, customers, cliente)
	if end == nil {
		return
	}
	if end.Street != nil {
		preencher(&dto.DestLogradouro, *end.Street)
	}
	if end.Number != nil {
		preencher(&dto.DestNumero, *end.Number)
	}
	if end.Complement != nil {
		preencher(&dto.DestComplemento, *end.Complement)
	}
	if end.Neighborhood != nil {
		preencher(&dto.DestBairro, *end.Neighborhood)
	}
	if end.City != nil {
		preencher(&dto.DestMunicipio, *end.City)
	}
	if end.ZipCode != nil {
		preencher(&dto.DestCEP, somenteDigitos(*end.ZipCode))
	}
	if end.UF != nil {
		preencher(&dto.UFDestinatario, *end.UF)
	}
}

// enderecoDaNota escolhe o endereço na ordem ENTREGA → COBRANÇA → padrão → primeiro.
func enderecoDaNota(ctx context.Context, customers customerrepo.CustomerRepository, cliente *customerentity.Customer) *customerentity.CustomerAddress {
	enderecos := cliente.Addresses
	if len(enderecos) == 0 {
		enderecos, _ = customers.ListAddresses(ctx, cliente.ID)
	}
	if len(enderecos) == 0 {
		return nil
	}
	var cobranca, padrao, primeiro *customerentity.CustomerAddress
	for _, e := range enderecos {
		if e == nil {
			continue
		}
		if primeiro == nil {
			primeiro = e
		}
		switch e.AddressType {
		case customerentity.AddressEntrega:
			return e
		case customerentity.AddressCobranca:
			if cobranca == nil {
				cobranca = e
			}
		}
		if e.IsDefault && padrao == nil {
			padrao = e
		}
	}
	if cobranca != nil {
		return cobranca
	}
	if padrao != nil {
		return padrao
	}
	return primeiro
}

func primeiroEmailDoCliente(cliente *customerentity.Customer) string {
	for _, c := range cliente.Contacts {
		if c != nil && c.Email != nil && strings.TrimSpace(*c.Email) != "" {
			return strings.TrimSpace(*c.Email)
		}
	}
	return ""
}

// preencher só escreve em campo vazio: o que o usuário digitou na nota manda.
func preencher(destino **string, valor string) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return
	}
	if *destino != nil && strings.TrimSpace(**destino) != "" {
		return
	}
	v := valor
	*destino = &v
}

func somenteDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// DadosFiscaisDoDestinatario são os campos do destinatário que a NF-e exige, já
// resolvidos do cadastro do cliente.
type DadosFiscaisDoDestinatario struct {
	CNPJ            string
	RazaoSocial     string
	IE              string
	UF              string
	Logradouro      string
	Numero          string
	Complemento     string
	Bairro          string
	Municipio       string
	CodigoMunicipio string
	CEP             string
	Email           string
	Telefone        string
}

// ResolvedorDeDestinatario expõe a resolução do destinatário para quem emite nota
// fora do módulo fiscal — hoje, o faturamento do beneficiamento.
//
// Existe para a regra de qual endereço vale (entrega, depois cobrança, depois
// padrão, depois o primeiro) morar num lugar só. Duplicá-la faria a nota de
// beneficiamento sair com endereço diferente da nota de venda do mesmo cliente.
type ResolvedorDeDestinatario struct {
	Customers customerrepo.CustomerRepository
}

func (r *ResolvedorDeDestinatario) DadosFiscaisDoCliente(ctx context.Context, code int64) (*DadosFiscaisDoDestinatario, error) {
	if r.Customers == nil {
		return nil, errorsuc.NewValidationError("cadastro de clientes indisponível para resolver o destinatário")
	}
	cliente, err := r.Customers.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if cliente == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d não encontrado", code))
	}

	dados := &DadosFiscaisDoDestinatario{
		CNPJ:        cliente.DocumentNumber,
		RazaoSocial: cliente.Name,
		Email:       primeiroEmailDoCliente(cliente),
	}
	if cliente.StateRegistration != nil {
		dados.IE = *cliente.StateRegistration
	}

	// Sem endereço a SEFAZ rejeita por campo obrigatório ausente. Recusar aqui é
	// melhor que descobrir na transmissão, com a nota já criada.
	end := enderecoDaNota(ctx, r.Customers, cliente)
	if end == nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"o cliente %d não tem endereço cadastrado; a NF-e exige logradouro, número, bairro, município, código IBGE e CEP", code))
	}
	dados.Logradouro = valorOuVazio(end.Street)
	dados.Numero = valorOuVazio(end.Number)
	dados.Complemento = valorOuVazio(end.Complement)
	dados.Bairro = valorOuVazio(end.Neighborhood)
	dados.Municipio = valorOuVazio(end.City)
	dados.UF = valorOuVazio(end.UF)
	dados.CEP = somenteDigitos(valorOuVazio(end.ZipCode))

	// O código IBGE do município NÃO vem do endereço do cliente: a tabela não tem a
	// coluna. Fica vazio aqui, igual ao que a criação de nota de venda já faz, e a
	// prévia da NF-e sinaliza a ausência antes da transmissão.
	return dados, nil
}

func valorOuVazio(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
