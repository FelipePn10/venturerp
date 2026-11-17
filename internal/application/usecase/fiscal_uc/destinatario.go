package fiscal_uc

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
	"strings"
	"time"

	"fmt"
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
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
// O endereço do destinatário é o cadastral: COMERCIAL, depois COBRANÇA, depois
// OUTRO; o de ENTREGA só quando o cliente não tem outro. O destinatário é a
// pessoa jurídica (a IE dela é desse endereço e a UF decide CFOP e DIFAL); se a
// mercadoria vai para outro endereço, ele vai no grupo de local de entrega.
//
// O que o usuário digitou na nota nunca é sobrescrito: só campo vazio é
// preenchido. Uma entrega pontual num endereço diferente continua valendo.
func resolverDestinatario(
	ctx context.Context,
	dto *request.CreateFiscalExitDTO,
	customers customerrepo.CustomerRepository,
	orders salesrepo.SalesOrderRepository,
) error {
	if dto == nil || customers == nil {
		return nil
	}

	// Pedido ou cliente inexistente só deixa os campos em branco (a nota diz o
	// que falta); falha de leitura é erro — antes sumia e a nota saía sem
	// destinatário sem ninguém saber por quê.
	customerCode := dto.CustomerCode
	if dto.SalesOrderCode != nil && orders != nil {
		order, err := orders.GetByCode(ctx, *dto.SalesOrderCode)
		if err != nil && !errorsuc.EhNaoEncontrado(err) {
			return fmt.Errorf("lendo o pedido da nota: %w", err)
		}
		if err == nil && order != nil {
			if customerCode == nil && order.CustomerCode != nil {
				c := *order.CustomerCode
				customerCode = &c
			}
			// Tipo comercial sem equivalente na nota (CONVENIO) não trava o
			// faturamento: fica sem tipo e o modFrete sai do valor do frete.
			if order.FreightType != nil {
				if t, ok := fiscalentity.TipoFreteDaNota(*order.FreightType); ok {
					preencher(&dto.TipoFrete, t)
				}
			}
		}
	}
	// Nota avulsa com o CNPJ/CPF de um cliente cadastrado fica ligada a ele:
	// sem o vínculo, o faturamento dele (Visão 360, crédito) não vê a nota.
	if customerCode == nil && dto.CnpjDestinatario != nil {
		if doc := somenteDigitos(*dto.CnpjDestinatario); len(doc) == 11 || len(doc) == 14 {
			c, err := customers.GetCustomerByDocument(ctx, doc)
			if err != nil && !errorsuc.EhNaoEncontrado(err) {
				return fmt.Errorf("procurando o cliente pelo documento da nota: %w", err)
			}
			if err == nil && c != nil {
				code := c.Code
				customerCode = &code
			}
		}
	}
	if customerCode == nil {
		return nil
	}

	cliente, err := customers.GetCustomerByCode(ctx, *customerCode)
	if err != nil && !errorsuc.EhNaoEncontrado(err) {
		return fmt.Errorf("lendo o cliente da nota: %w", err)
	}
	if err != nil || cliente == nil {
		return nil
	}
	dto.CustomerCode = customerCode

	preencher(&dto.RazaoSocialDestinatario, cliente.Name)
	preencher(&dto.CnpjDestinatario, cliente.DocumentNumber)
	if cliente.StateRegistration != nil {
		preencher(&dto.IEDestinatario, *cliente.StateRegistration)
	}
	if isuf := SuframaValida(cliente.SuframaCode, cliente.SuframaExpiry, time.Now()); isuf != "" {
		preencher(&dto.SuframaDestinatario, isuf)
	}
	if dto.TipoPessoa == nil && cliente.DocumentType == customerentity.DocumentCPF {
		f := "F"
		dto.TipoPessoa = &f
	}
	preencher(&dto.DestEmail, primeiroEmailDoCliente(cliente))
	if cliente.FreightType != nil {
		preencher(&dto.TipoFrete, *cliente.FreightType)
	}
	if dto.ShippingCarrierID == nil && cliente.ShippingCarrierID != nil {
		id := *cliente.ShippingCarrierID
		dto.ShippingCarrierID = &id
	}

	end, err := enderecoDaNota(ctx, customers, cliente)
	if err != nil {
		return err
	}
	if end == nil {
		return nil
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
	return nil
}

// enderecoDaNota escolhe o endereço cadastral do destinatário: COMERCIAL →
// COBRANÇA → OUTRO → ENTREGA; no mesmo tipo, o padrão e depois o primeiro.
func enderecoDaNota(ctx context.Context, customers customerrepo.CustomerRepository, cliente *customerentity.Customer) (*customerentity.CustomerAddress, error) {
	enderecos, err := enderecosDoCliente(ctx, customers, cliente)
	if err != nil {
		return nil, err
	}
	return melhorEndereco(enderecos, map[customerentity.AddressType]int{
		customerentity.AddressComercial: 0, customerentity.AddressCobranca: 1, customerentity.AddressOutro: 2, customerentity.AddressEntrega: 3,
	}), nil
}

func enderecosDoCliente(ctx context.Context, customers customerrepo.CustomerRepository, cliente *customerentity.Customer) ([]*customerentity.CustomerAddress, error) {
	if len(cliente.Addresses) > 0 {
		return cliente.Addresses, nil
	}
	enderecos, err := customers.ListAddresses(ctx, cliente.ID)
	if err != nil {
		return nil, fmt.Errorf("lendo os endereços do cliente: %w", err)
	}
	return enderecos, nil
}

// melhorEndereco: o de menor posição pelo tipo; empate, o padrão; depois o primeiro.
func melhorEndereco(enderecos []*customerentity.CustomerAddress, posicao map[customerentity.AddressType]int) *customerentity.CustomerAddress {
	var melhor *customerentity.CustomerAddress
	rank := func(e *customerentity.CustomerAddress) int {
		p, ok := posicao[e.AddressType]
		if !ok {
			p = len(posicao)
		}
		p *= 2
		if !e.IsDefault {
			p++
		}
		return p
	}
	for _, e := range enderecos {
		if e != nil && (melhor == nil || rank(e) < rank(melhor)) {
			melhor = e
		}
	}
	return melhor
}

// localDeEntrega: o grupo de entrega da NF-e quando a mercadoria vai para um
// endereço de ENTREGA do cliente diferente do endereço do destinatário. O
// endereço de entrega do pedido (shipping_address_code) tem preferência.
func localDeEntrega(ctx context.Context, dto *request.CreateFiscalExitDTO, customers customerrepo.CustomerRepository, orders salesrepo.SalesOrderRepository) (*fiscalentity.LocalEntrega, error) {
	if dto == nil || customers == nil || dto.CustomerCode == nil {
		return nil, nil
	}
	cliente, err := customers.GetCustomerByCode(ctx, *dto.CustomerCode)
	if err != nil {
		if errorsuc.EhNaoEncontrado(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("lendo o cliente da nota: %w", err)
	}
	enderecos, err := enderecosDoCliente(ctx, customers, cliente)
	if err != nil {
		return nil, err
	}
	var entrega *customerentity.CustomerAddress
	if dto.SalesOrderCode != nil && orders != nil {
		if order, err := orders.GetByCode(ctx, *dto.SalesOrderCode); err == nil && order != nil && order.ShippingAddressCode != nil {
			for _, e := range enderecos {
				if e != nil && e.ID == *order.ShippingAddressCode {
					entrega = e
				}
			}
		}
	}
	if entrega == nil {
		var soEntrega []*customerentity.CustomerAddress
		for _, e := range enderecos {
			if e != nil && e.AddressType == customerentity.AddressEntrega {
				soEntrega = append(soEntrega, e)
			}
		}
		entrega = melhorEndereco(soEntrega, nil)
	}
	if entrega == nil || mesmoEndereco(entrega, dto) {
		return nil, nil
	}
	return &fiscalentity.LocalEntrega{
		CNPJCPF: somenteDigitos(cliente.DocumentNumber), Nome: cliente.Name,
		Logradouro: valorOuVazio(entrega.Street), Numero: valorOuVazio(entrega.Number), Complemento: valorOuVazio(entrega.Complement),
		Bairro: valorOuVazio(entrega.Neighborhood), Municipio: valorOuVazio(entrega.City), UF: valorOuVazio(entrega.UF),
		CEP: somenteDigitos(valorOuVazio(entrega.ZipCode)),
	}, nil
}

// mesmoEndereco compara o endereço de entrega com o do destinatário da nota.
func mesmoEndereco(e *customerentity.CustomerAddress, dto *request.CreateFiscalExitDTO) bool {
	norm := func(s string) string { return strings.ToUpper(strings.Join(strings.Fields(s), " ")) }
	d := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	return norm(valorOuVazio(e.Street)) == norm(d(dto.DestLogradouro)) &&
		norm(valorOuVazio(e.Number)) == norm(d(dto.DestNumero)) &&
		norm(valorOuVazio(e.City)) == norm(d(dto.DestMunicipio)) &&
		norm(valorOuVazio(e.UF)) == norm(d(dto.UFDestinatario))
}

// primeiroEmailDoCliente: o e-mail de NF-e do cadastro (o primeiro, se houver
// vários separados por ";"); sem ele, o do primeiro contato que tiver.
func primeiroEmailDoCliente(cliente *customerentity.Customer) string {
	if cliente.NFeEmail != nil {
		if e := strings.TrimSpace(strings.Split(*cliente.NFeEmail, ";")[0]); e != "" {
			return e
		}
	}
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
// Existe para a regra de qual endereço vale (comercial, cobrança, outro e só
// então entrega) morar num lugar só. Duplicá-la faria a nota de
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
	end, err := enderecoDaNota(ctx, r.Customers, cliente)
	if err != nil {
		return nil, err
	}
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

// SuframaValida: a inscrição SUFRAMA que vai para a NF-e (só dígitos), ou ""
// quando o cliente não tem, o dígito não confere (cadastro antigo) ou ela já venceu na data — vencida, a SEFAZ recusa
// (rejeição 235) e o benefício da Zona Franca não vale.
func SuframaValida(codigo *string, validade *time.Time, hoje time.Time) string {
	if codigo == nil {
		return ""
	}
	d := soDigitos(*codigo)
	if d == "" || !validation.SuframaValida(d) {
		return ""
	}
	if validade != nil && validade.Before(time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, validade.Location())) {
		return ""
	}
	return d
}
