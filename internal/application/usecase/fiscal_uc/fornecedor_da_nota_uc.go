package fiscal_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// CadastroDeFornecedor é o cadastro de fornecedores (VSUP0500) com as mesmas
// regras da tela: documento único, condição de ICMS × inscrição estadual.
type CadastroDeFornecedor interface {
	CreateSupplier(ctx context.Context, dto request.CreateSupplierDTO) (*response.SupplierResponse, error)
	AddAddress(ctx context.Context, dto request.AddSupplierAddressDTO) (*response.SupplierAddressResponse, error)
}

// FornecedorDaNotaUseCase cadastra o emitente de uma nota de entrada como
// fornecedor com os dados que o próprio XML declara (razão social, fantasia,
// CNPJ/CPF, IE e endereço) e liga o fornecedor à nota e às outras notas
// pendentes do mesmo CNPJ.
type FornecedorDaNotaUseCase struct {
	Docs     repository.FiscalEntryDocumentRepository
	Fiscal   repository.FiscalRepository
	Cadastro CadastroDeFornecedor
	Auth     ports.AuthService
	Servico  *EntradaServico
}

// FornecedorDaNotaDTO: tudo opcional; o que vier sobrepõe o que o XML diz.
type FornecedorDaNotaDTO struct {
	SupplierTypeCode   *int64  `json:"supplier_type_code,omitempty"`
	PaymentConditionID *int64  `json:"payment_condition_id,omitempty"`
	ICMSContributor    string  `json:"icms_contributor,omitempty"`
	StateRegistration  *string `json:"state_registration,omitempty"`
	TradeName          *string `json:"trade_name,omitempty"`
}

func (uc *FornecedorDaNotaUseCase) Execute(ctx context.Context, entryID int64, dto FornecedorDaNotaDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if doc.Status != entity.EntryStatusPending && doc.Status != entity.EntryStatusConferred {
		return nil, errorsuc.NewValidationError("só nota pendente ou conferida recebe fornecedor")
	}
	if doc.SupplierCode != nil {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("a nota já está ligada ao fornecedor %d", *doc.SupplierCode))
	}
	if f, err := uc.Docs.FindSupplierByDocument(ctx, doc.CnpjEmitente); err != nil {
		return nil, err
	} else if f != nil {
		// Cadastrado depois da importação: só liga.
		if _, err := uc.Docs.LinkSupplierToPendingEntries(ctx, doc.CnpjEmitente, f.Code); err != nil {
			return nil, err
		}
		return uc.responder(ctx, entryID)
	}

	in := DadosFornecedorDaNota(doc, uc.lerXML(ctx, doc.ID))
	if dto.TradeName != nil && strings.TrimSpace(*dto.TradeName) != "" {
		in.Supplier.TradeName = dto.TradeName
	}
	if dto.StateRegistration != nil {
		in.Supplier.StateRegistration = dto.StateRegistration
		in.Supplier.ICMSContributor = condicaoICMS(*dto.StateRegistration)
	}
	if dto.ICMSContributor != "" {
		in.Supplier.ICMSContributor = dto.ICMSContributor
	}
	in.Supplier.SupplierTypeCode, in.Supplier.PaymentConditionID = dto.SupplierTypeCode, dto.PaymentConditionID

	criado, err := uc.Cadastro.CreateSupplier(ctx, in.Supplier)
	if err != nil {
		return nil, err
	}
	if in.Endereco != nil {
		in.Endereco.SupplierCode = criado.Code
		// O endereço é complemento: o fornecedor já existe e a nota já pode
		// seguir; uma falha aqui não desfaz o cadastro, vira aviso.
		if _, err := uc.Cadastro.AddAddress(ctx, *in.Endereco); err != nil {
			r, rerr := uc.ligarEResponder(ctx, doc.CnpjEmitente, criado.Code, entryID)
			if rerr != nil {
				return nil, rerr
			}
			r.Warnings = append(r.Warnings, "fornecedor cadastrado, mas o endereço não foi gravado: "+err.Error())
			return r, nil
		}
	}
	return uc.ligarEResponder(ctx, doc.CnpjEmitente, criado.Code, entryID)
}

func (uc *FornecedorDaNotaUseCase) ligarEResponder(ctx context.Context, cnpj string, code, entryID int64) (*response.FiscalEntryResponse, error) {
	if _, err := uc.Docs.LinkSupplierToPendingEntries(ctx, cnpj, code); err != nil {
		return nil, err
	}
	return uc.responder(ctx, entryID)
}

func (uc *FornecedorDaNotaUseCase) responder(ctx context.Context, entryID int64) (*response.FiscalEntryResponse, error) {
	doc, err := uc.Docs.GetEntryDocument(ctx, entryID)
	if err != nil {
		return nil, err
	}
	s := uc.Servico
	if s == nil {
		s = &EntradaServico{Docs: uc.Docs, Fiscal: uc.Fiscal}
	}
	return s.Responder(ctx, doc)
}

func (uc *FornecedorDaNotaUseCase) lerXML(ctx context.Context, id int64) *NFeLida {
	_, xmlTxt, err := uc.Docs.GetEntryXML(ctx, id)
	if err != nil || strings.TrimSpace(xmlTxt) == "" {
		return nil
	}
	n, err := LerNFe([]byte(xmlTxt))
	if err != nil {
		return nil
	}
	return n
}

// CadastroDaNota é o fornecedor (e o endereço) que a nota permite cadastrar.
type CadastroDaNota struct {
	Supplier request.CreateSupplierDTO
	Endereco *request.AddSupplierAddressDTO
}

// DadosFornecedorDaNota monta o cadastro do emitente: do XML quando há, do
// cabeçalho da nota (lançamento manual) quando não.
func DadosFornecedorDaNota(doc *entity.FiscalEntry, n *NFeLida) CadastroDaNota {
	documento := soDigitos(doc.CnpjEmitente)
	nome := strings.TrimSpace(doc.RazaoSocialEmitente)
	ie := strings.TrimSpace(deref(doc.IEEmitente))
	uf := strings.ToUpper(strings.TrimSpace(deref(doc.UFEmitente)))
	var fantasia string
	if n != nil {
		nome = firstNonEmpty(strings.TrimSpace(n.EmitenteNome), nome)
		ie = firstNonEmpty(strings.TrimSpace(n.EmitenteIE), ie)
		uf = firstNonEmpty(strings.ToUpper(strings.TrimSpace(n.EmitenteUF)), uf)
		fantasia = strings.TrimSpace(n.EmitenteFantasia)
	}
	s := request.CreateSupplierDTO{Name: nome, PersonType: "JURIDICA", DocumentType: "CNPJ", DocumentNumber: documento}
	if len(documento) == 11 {
		s.PersonType, s.DocumentType = "FISICA", "CPF"
	}
	if fantasia != "" {
		s.TradeName = &fantasia
	}
	if ie != "" && !strings.EqualFold(ie, "ISENTO") {
		v := soDigitos(ie)
		s.StateRegistration = &v
	}
	s.ICMSContributor = condicaoICMS(ie)

	out := CadastroDaNota{Supplier: s}
	if n != nil && (n.EmitenteLogradouro != "" || n.EmitenteMunicipio != "") {
		opt := func(v string) *string {
			if v = strings.TrimSpace(v); v == "" {
				return nil
			}
			return &v
		}
		out.Endereco = &request.AddSupplierAddressDTO{
			AddressType: "COMERCIAL", ZipCode: opt(n.EmitenteCEP), Street: opt(n.EmitenteLogradouro), Number: opt(n.EmitenteNumero),
			Complement: opt(n.EmitenteComplemento), Neighborhood: opt(n.EmitenteBairro), City: opt(n.EmitenteMunicipio),
			UF: opt(uf), IsDefault: true,
		}
	}
	return out
}

// condicaoICMS: inscrição estadual informada é contribuinte; "ISENTO" é isento;
// sem inscrição, não contribuinte.
func condicaoICMS(ie string) string {
	ie = strings.TrimSpace(ie)
	switch {
	case strings.EqualFold(ie, "ISENTO"):
		return "ISENTO"
	case soDigitos(ie) != "":
		return "CONTRIBUINTE"
	}
	return "NAO_CONTRIBUINTE"
}
