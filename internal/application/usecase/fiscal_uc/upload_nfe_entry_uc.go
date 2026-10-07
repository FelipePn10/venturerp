package fiscal_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

// UploadNFEEntryUseCase importa a NF-e de entrada a partir do XML do
// fornecedor (o arquivo, não o texto colado). Lê do XML tudo o que a nota
// diz — emitente, itens, impostos por item, frete, duplicatas, pagamento — e
// já devolve a nota conciliada até onde o vínculo produto × fornecedor
// permite, com plano de contas sugerido e as parcelas distribuídas.
type UploadNFEEntryUseCase struct {
	Repo           repository.FiscalRepository
	Docs           repository.FiscalEntryDocumentRepository
	Auth           ports.AuthService
	PurchaseOrders porepo.PurchaseOrderRepository
	Tolerances     ports.PurchaseToleranceEvaluator
	SupplierItems  ports.ItemSupplierResolver
}

func (uc *UploadNFEEntryUseCase) Execute(ctx context.Context, dto request.UploadNFEDTO) (*response.FiscalEntryResponse, error) {
	return uc.ExecuteFile(ctx, []byte(dto.XmlContent), dto)
}

// ExecuteFile é a importação a partir do arquivo enviado.
func (uc *UploadNFEEntryUseCase) ExecuteFile(ctx context.Context, conteudo []byte, dto request.UploadNFEDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if uc.Docs == nil {
		return nil, fmt.Errorf("repositório da nota de entrada não configurado")
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}

	nfe, err := LerNFe(conteudo)
	if err != nil {
		return nil, err
	}

	// A mesma nota não entra duas vezes: crédito de imposto e título a pagar
	// em dobro é o erro mais caro de uma entrada fiscal.
	if nfe.ChaveAcesso != "" {
		existente, err := uc.Docs.FindEntryByChave(ctx, nfe.ChaveAcesso)
		if err != nil {
			return nil, err
		}
		if existente != nil {
			return nil, errorsuc.NewConflictError(fmt.Sprintf(
				"esta NF-e (chave %s) já foi importada como entrada %d, nota %d de %s",
				nfe.ChaveAcesso, existente.ID, existente.NumeroNF, existente.RazaoSocialEmitente))
		}
	}

	dataEntrada := nfe.DataEmissao
	if nfe.DataSaidaEntrada != nil {
		dataEntrada = *nfe.DataSaidaEntrada
	}
	if dto.DataEntrada != nil && strings.TrimSpace(*dto.DataEntrada) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*dto.DataEntrada))
		if err != nil {
			return nil, errorsuc.NewValidationError("data de entrada inválida: use AAAA-MM-DD")
		}
		dataEntrada = t
	}

	entry, err := montarEntradaDoXML(nfe, strings.TrimSpace(string(conteudo)), dataEntrada)
	if err != nil {
		return nil, err
	}
	entry.EnterpriseID = enterpriseID
	entry.CreatedBy = userID
	entry.PurchaseOrderCode = dto.PurchaseOrderCode

	var warnings []string
	cfg, err := uc.Repo.GetFiscalConfig(ctx)
	var semConfig *errorsuc.NotFoundError
	if err != nil && !errors.As(err, &semConfig) {
		return nil, err
	}
	if err == nil && cfg != nil && cfg.CnpjEmpresa != "" && nfe.DestinatarioCNPJ != "" &&
		soDigitos(cfg.CnpjEmpresa) != nfe.DestinatarioCNPJ {
		warnings = append(warnings, fmt.Sprintf(
			"o destinatário da nota (%s) não é o CNPJ desta empresa (%s): confira se a nota é mesmo para esta empresa",
			nfe.DestinatarioCNPJ, soDigitos(cfg.CnpjEmpresa)))
	}

	// Fornecedor pelo CNPJ do emitente.
	fornecedor, err := uc.Docs.FindSupplierByDocument(ctx, nfe.EmitenteCNPJ)
	if err != nil {
		return nil, err
	}
	var contaFornecedor *string
	if fornecedor != nil {
		code := fornecedor.Code
		entry.SupplierCode = &code
		contaFornecedor = fornecedor.FinancialAccount
		if !fornecedor.IsActive || fornecedor.Blocked {
			warnings = append(warnings, fmt.Sprintf("o fornecedor %d (%s) está inativo ou bloqueado", fornecedor.Code, fornecedor.Name))
		}
	} else {
		warnings = append(warnings, fmt.Sprintf(
			"o emitente %s (%s) não está cadastrado como fornecedor: cadastre-o para memorizar os vínculos dos itens e o contas a pagar nascer com o fornecedor",
			nfe.EmitenteCNPJ, nfe.EmitenteNome))
	}

	// Pedido informado na importação: o fornecedor dele vale quando o CNPJ não
	// casou com o cadastro.
	if entry.SupplierCode == nil && dto.PurchaseOrderCode != nil && uc.PurchaseOrders != nil {
		po, err := uc.PurchaseOrders.GetByCode(ctx, *dto.PurchaseOrderCode)
		if err != nil {
			return nil, err
		}
		entry.SupplierCode = po.SupplierCode
	}

	if err = conciliacaoAutomatica(ctx, uc.Docs, entry.SupplierCode, entry.Itens); err != nil {
		return nil, err
	}
	servico := uc.servico()
	// Operação, pedido de compra (3-way), almoxarifado e CFOP de entrada.
	if err = servico.Preparar(ctx, entry); err != nil {
		return nil, err
	}
	if err = sugerirPlanosDeContas(ctx, uc.Docs, contaFornecedor, entry.Itens); err != nil {
		return nil, err
	}
	if err = Distribuir(entry); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	if entry.Status, err = servico.Status(ctx, entry); err != nil {
		return nil, err
	}

	if _, err = uc.Docs.CreateEntryDocument(ctx, entry); err != nil {
		return nil, err
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entry.ID)
	if err != nil {
		return nil, err
	}
	doc.Warnings = warnings
	return servico.Responder(ctx, doc)
}

func (uc *UploadNFEEntryUseCase) servico() *EntradaServico {
	return &EntradaServico{Docs: uc.Docs, Fiscal: uc.Repo, Tolerancias: uc.Tolerances}
}
