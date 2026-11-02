package fiscal_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// LeitorConfigFiscal é o pedaço da configuração fiscal que a sugestão de CFOP usa.
type LeitorConfigFiscal interface {
	GetFiscalConfig(ctx context.Context) (*entity.FiscalConfig, error)
}

// DestinatarioDaNotaUseCase entrega à tela de nota avulsa o que o cadastro já
// sabe do cliente, para ninguém digitar duas vezes. Usa a mesma regra de
// endereço e de CFOP da criação da nota e do faturamento do pedido: o que a tela
// mostra é o que a nota vai usar.
type DestinatarioDaNotaUseCase struct {
	Customers customerrepo.CustomerRepository
	Fiscal    LeitorConfigFiscal
	Auth      ports.AuthService
}

// CfopSugerido é o CFOP de venda da operação e a natureza que o acompanha.
type CfopSugerido struct {
	Cfop     string `json:"cfop"`
	Natureza string `json:"natureza"`
}

// EnderecoDoDestinatario é um endereço do cadastro com o CFOP que ele implica: a
// UF do endereço decide se a operação é interna ou interestadual.
type EnderecoDoDestinatario struct {
	Tipo        string `json:"tipo"`
	Logradouro  string `json:"logradouro"`
	Numero      string `json:"numero"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Municipio   string `json:"municipio"`
	UF          string `json:"uf"`
	CEP         string `json:"cep"`
	// Sugerido marca o endereço que a nota usa se ninguém escolher outro.
	Sugerido bool `json:"sugerido"`
	// Faltando lista, em português, o que a SEFAZ exige e não está preenchido.
	Faltando []string `json:"faltando"`
	CfopSugerido
}

// DestinatarioDaNota são os dados do cliente prontos para a nota.
type DestinatarioDaNota struct {
	CustomerCode     int64                    `json:"customer_code"`
	Nome             string                   `json:"nome"`
	Documento        string                   `json:"documento"`
	TipoPessoa       string                   `json:"tipo_pessoa"`
	IE               string                   `json:"ie"`
	Email            string                   `json:"email"`
	Telefone         string                   `json:"telefone"`
	FinalidadeCompra string                   `json:"finalidade_compra"`
	InvoiceTypeID    *int64                   `json:"invoice_type_id,omitempty"`
	Bloqueado        bool                     `json:"bloqueado"`
	Enderecos        []EnderecoDoDestinatario `json:"enderecos"`
	// CfopSemEndereco vale enquanto o cliente não tem endereço: sem UF, a
	// operação é tratada como interna.
	CfopSemEndereco CfopSugerido `json:"cfop_sem_endereco"`
}

func (uc *DestinatarioDaNotaUseCase) DoCliente(ctx context.Context, code int64) (*DestinatarioDaNota, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	cliente, err := uc.Customers.GetCustomerByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if cliente == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d não encontrado", code))
	}
	ufEmpresa, err := uc.ufDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}

	if len(cliente.Contacts) == 0 {
		if cliente.Contacts, err = uc.Customers.ListContacts(ctx, cliente.ID); err != nil {
			return nil, fmt.Errorf("lendo os contatos do cliente: %w", err)
		}
	}
	tipoPessoa := "J"
	if cliente.DocumentType == customerentity.DocumentCPF {
		tipoPessoa = "F"
	}
	out := &DestinatarioDaNota{
		CustomerCode:  cliente.Code,
		Nome:          strings.TrimSpace(cliente.Name),
		Documento:     somenteDigitos(cliente.DocumentNumber),
		TipoPessoa:    tipoPessoa,
		IE:            valorOuVazio(cliente.StateRegistration),
		Email:         primeiroEmailDoCliente(cliente),
		Telefone:      primeiroTelefoneDoCliente(cliente),
		InvoiceTypeID: cliente.InvoiceTypeID,
		Bloqueado:     cliente.Blocked,
		Enderecos:     []EnderecoDoDestinatario{},
	}
	out.FinalidadeCompra = valorOuVazio(cliente.FinalidadeCompra)
	out.CfopSemEndereco = cfopDaVenda(ufEmpresa, "", out.IE, tipoPessoa)

	sugerido, err := enderecoDaNota(ctx, uc.Customers, cliente)
	if err != nil {
		return nil, err
	}
	enderecos := cliente.Addresses
	if len(enderecos) == 0 {
		if enderecos, err = uc.Customers.ListAddresses(ctx, cliente.ID); err != nil {
			return nil, fmt.Errorf("lendo os endereços do cliente: %w", err)
		}
	}
	for _, e := range enderecos {
		if e == nil {
			continue
		}
		uf := strings.ToUpper(valorOuVazio(e.UF))
		end := EnderecoDoDestinatario{
			Tipo: string(e.AddressType), Logradouro: valorOuVazio(e.Street), Numero: valorOuVazio(e.Number),
			Complemento: valorOuVazio(e.Complement), Bairro: valorOuVazio(e.Neighborhood), Municipio: valorOuVazio(e.City),
			UF: uf, CEP: somenteDigitos(valorOuVazio(e.ZipCode)), Sugerido: sugerido != nil && e.ID == sugerido.ID,
			CfopSugerido: cfopDaVenda(ufEmpresa, uf, out.IE, tipoPessoa),
		}
		end.Faltando = faltandoNoEndereco(end)
		out.Enderecos = append(out.Enderecos, end)
	}
	return out, nil
}

// SugerirCfop atende o destinatário sem cadastro, em que a UF e a IE são digitadas.
func (uc *DestinatarioDaNotaUseCase) SugerirCfop(ctx context.Context, uf, ie, tipoPessoa string) (*CfopSugerido, error) {
	if !uc.Auth.CanCreateFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	ufEmpresa, err := uc.ufDaEmpresa(ctx)
	if err != nil {
		return nil, err
	}
	s := cfopDaVenda(ufEmpresa, strings.ToUpper(strings.TrimSpace(uf)), ie, tipoPessoa)
	return &s, nil
}

// ufDaEmpresa devolve vazio quando a empresa ainda não tem configuração fiscal: a
// sugestão vira operação interna, e a conferência da nota aponta o que falta.
func (uc *DestinatarioDaNotaUseCase) ufDaEmpresa(ctx context.Context) (string, error) {
	cfg, err := uc.Fiscal.GetFiscalConfig(ctx)
	var semConfig *errorsuc.NotFoundError
	if err != nil && !errors.As(err, &semConfig) {
		return "", err
	}
	if cfg == nil {
		return "", nil
	}
	return strings.ToUpper(strings.TrimSpace(cfg.UFEmpresa)), nil
}

// cfopDaVenda escolhe o CFOP de venda de produção pela UF de destino. O
// faturamento do pedido usa esta mesma regra; o ajuste para revenda (item
// comprado) é feito linha a linha por cfopDaLinha.
func cfopDaVenda(ufEmpresa, ufDestino, ie, tipoPessoa string) CfopSugerido {
	if ufEmpresa == "" || ufDestino == "" || strings.EqualFold(ufEmpresa, ufDestino) {
		return CfopSugerido{Cfop: "5101", Natureza: "VENDA DE PRODUCAO DO ESTABELECIMENTO"}
	}
	// Venda a consumidor final não contribuinte de outro estado tem CFOP
	// próprio (6107): é a operação com DIFAL (EC 87/2015).
	if tipoDestinatario(&ie, &tipoPessoa) != "contribuinte" {
		return CfopSugerido{Cfop: "6107", Natureza: "VENDA DE PRODUCAO A NAO CONTRIBUINTE"}
	}
	return CfopSugerido{Cfop: "6101", Natureza: "VENDA DE PRODUCAO DO ESTABELECIMENTO"}
}

func faltandoNoEndereco(e EnderecoDoDestinatario) []string {
	faltando := []string{}
	for _, c := range []struct{ nome, valor string }{
		{"logradouro", e.Logradouro}, {"número", e.Numero}, {"bairro", e.Bairro},
		{"município", e.Municipio}, {"UF", e.UF}, {"CEP", e.CEP},
	} {
		if c.valor == "" {
			faltando = append(faltando, c.nome)
		}
	}
	return faltando
}

func primeiroTelefoneDoCliente(cliente *customerentity.Customer) string {
	for _, c := range cliente.Contacts {
		if c == nil {
			continue
		}
		if t := valorOuVazio(c.Phone); t != "" {
			return t
		}
		if t := valorOuVazio(c.Mobile); t != "" {
			return t
		}
	}
	return ""
}
