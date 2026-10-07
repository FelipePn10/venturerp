package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type CreateFiscalEntryUseCase struct {
	Repo repository.FiscalRepository
	// Docs, quando presente, grava a nota completa numa transação (itens,
	// parcelas e a distribuição por plano de contas).
	Docs           repository.FiscalEntryDocumentRepository
	Auth           ports.AuthService
	PurchaseOrders porepo.PurchaseOrderRepository
	Tolerances     ports.PurchaseToleranceEvaluator
	SupplierItems  ports.ItemSupplierResolver
}

func (uc *CreateFiscalEntryUseCase) Execute(ctx context.Context, dto request.CreateFiscalEntryDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanCreateFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	enterpriseID, err := uc.Auth.EnterpriseID(ctx)
	if err != nil {
		return nil, err
	}

	dataEmissao, err := time.Parse("2006-01-02", strings.TrimSpace(dto.DataEmissao))
	if err != nil {
		return nil, errorsuc.NewValidationError("data de emissão inválida: use AAAA-MM-DD")
	}
	dataEntrada, err := time.Parse("2006-01-02", strings.TrimSpace(dto.DataEntrada))
	if err != nil {
		dataEntrada = dataEmissao
	}
	dto.CnpjEmitente = soDigitos(dto.CnpjEmitente)

	entry := &entity.FiscalEntry{
		EnterpriseID:        enterpriseID,
		ChaveAcesso:         dto.ChaveAcesso,
		NumeroNF:            dto.NumeroNF,
		Serie:               dto.Serie,
		Modelo:              dto.Modelo,
		DataEmissao:         dataEmissao,
		DataEntrada:         dataEntrada,
		CnpjEmitente:        dto.CnpjEmitente,
		RazaoSocialEmitente: dto.RazaoSocialEmitente,
		IEEmitente:          dto.IEEmitente,
		UFEmitente:          dto.UFEmitente,
		ValorProdutos:       dto.ValorProdutos,
		ValorFrete:          dto.ValorFrete,
		ValorSeguro:         dto.ValorSeguro,
		ValorDesconto:       dto.ValorDesconto,
		ValorIPI:            dto.ValorIPI,
		ValorICMS:           dto.ValorICMS,
		ValorPIS:            dto.ValorPIS,
		ValorCOFINS:         dto.ValorCOFINS,
		ValorTotal:          dto.ValorTotal,
		TipoDocumento:       dto.TipoDocumento,
		PurchaseOrderCode:   dto.PurchaseOrderCode,
		CteCode:             dto.CteCode,
		Status:              entity.EntryStatusPending,
		Notes:               dto.Notes,
		CreatedBy:           userID,
	}
	pendingItems := make([]*entity.FiscalEntryItem, 0, len(dto.Itens))
	for _, itemDTO := range dto.Itens {
		pendingItems = append(pendingItems, entryItemFromDTO(itemDTO))
	}
	if uc.Docs != nil {
		return uc.criarDocumento(ctx, dto, entry, pendingItems)
	}
	entry.SupplierCode, entry.Warnings, err = validatePurchaseEntryTolerances(ctx, uc.PurchaseOrders, uc.Tolerances, dto.PurchaseOrderCode, pendingItems, dto.ValorProdutos)
	if err != nil {
		return nil, err
	}
	if err = resolveSupplierItems(ctx, uc.SupplierItems, entry.SupplierCode, pendingItems); err != nil {
		return nil, err
	}

	created, err := uc.Repo.CreateEntry(ctx, entry)
	if err != nil {
		return nil, err
	}

	for _, item := range pendingItems {
		item.FiscalEntryID = created.ID
		if _, err := uc.Repo.CreateEntryItem(ctx, item); err != nil {
			return nil, err
		}
	}

	items, _ := uc.Repo.GetEntryItems(ctx, created.ID)
	created.Itens = items

	return toFiscalEntryResponse(created), nil
}

func entryItemFromDTO(itemDTO request.CreateFiscalEntryItemDTO) *entity.FiscalEntryItem {
	return &entity.FiscalEntryItem{
		Sequence:              itemDTO.Sequence,
		ItemCode:              itemDTO.ItemCode,
		SupplierItemCode:      itemDTO.SupplierItemCode,
		UOM:                   itemDTO.UOM,
		Ncm:                   itemDTO.Ncm,
		Cfop:                  itemDTO.Cfop,
		Quantity:              itemDTO.Quantity,
		UnitPrice:             itemDTO.UnitPrice,
		TotalPrice:            itemDTO.TotalPrice,
		BaseICMS:              itemDTO.BaseICMS,
		AliqICMS:              itemDTO.AliqICMS,
		ValorICMS:             itemDTO.ValorICMS,
		BaseIPI:               itemDTO.BaseIPI,
		AliqIPI:               itemDTO.AliqIPI,
		ValorIPI:              itemDTO.ValorIPI,
		ValorPIS:              itemDTO.ValorPIS,
		ValorCOFINS:           itemDTO.ValorCOFINS,
		CstICMS:               itemDTO.CstICMS,
		CstIPI:                itemDTO.CstIPI,
		CstPIS:                itemDTO.CstPIS,
		CstCOFINS:             itemDTO.CstCOFINS,
		GeraCreditoICMS:       itemDTO.GeraCreditoICMS,
		GeraCreditoIPI:        itemDTO.GeraCreditoIPI,
		GeraCreditoPIS:        itemDTO.GeraCreditoPIS,
		GeraCreditoCOFINS:     itemDTO.GeraCreditoCOFINS,
		Description:           itemDTO.Description,
		Notes:                 itemDTO.Notes,
		PlanoContasID:         itemDTO.PlanoContasID,
		CentroCustoID:         itemDTO.CentroCustoID,
		EntryOperationCode:    itemDTO.EntryOperationCode,
		WarehouseID:           itemDTO.WarehouseID,
		PurchaseOrderItemCode: itemDTO.PurchaseOrderItemCode,
		MovimentaEstoque:      true,
		GeraFinanceiro:        true,
	}
}

// criarDocumento é o lançamento manual completo: fornecedor pelo cadastro (ou
// pelo CNPJ), itens com plano de contas, parcelas e distribuição, tudo numa
// transação.
func (uc *CreateFiscalEntryUseCase) criarDocumento(ctx context.Context, dto request.CreateFiscalEntryDTO, entry *entity.FiscalEntry, itens []*entity.FiscalEntryItem) (*response.FiscalEntryResponse, error) {
	if dto.SupplierCode != nil {
		entry.SupplierCode = dto.SupplierCode
	}
	for _, v := range []decimal.Decimal{dto.ValorRetPIS, dto.ValorRetCOFINS, dto.ValorRetCSLL, dto.ValorIRRF, dto.ValorRetPrev, dto.ValorISSRet} {
		if v.IsNegative() {
			return nil, errorsuc.NewValidationError("retenção não pode ser negativa")
		}
	}
	entry.EntryOperationCode = dto.EntryOperationCode
	entry.ValorRetPIS, entry.ValorRetCOFINS, entry.ValorRetCSLL = dto.ValorRetPIS.Round(2), dto.ValorRetCOFINS.Round(2), dto.ValorRetCSLL.Round(2)
	entry.ValorIRRF, entry.ValorRetPrev, entry.ValorISSRet = dto.ValorIRRF.Round(2), dto.ValorRetPrev.Round(2), dto.ValorISSRet.Round(2)
	if entry.TotalRetencoes().GreaterThan(decimal.NewFromFloat(dto.ValorTotal)) {
		return nil, errorsuc.NewValidationError("as retenções passam do total da nota")
	}
	entry.StockStatus = entity.StockStatusPendente
	if entry.SupplierCode == nil && dto.PurchaseOrderCode != nil && uc.PurchaseOrders != nil {
		po, err := uc.PurchaseOrders.GetByCode(ctx, *dto.PurchaseOrderCode)
		if err != nil {
			return nil, err
		}
		entry.SupplierCode = po.SupplierCode
	}
	var contaFornecedor *string
	if f, err := uc.Docs.FindSupplierByDocument(ctx, dto.CnpjEmitente); err != nil {
		return nil, err
	} else if f != nil {
		if entry.SupplierCode == nil {
			code := f.Code
			entry.SupplierCode = &code
		}
		contaFornecedor = f.FinancialAccount
	}
	if entry.ChaveAcesso != nil {
		chave := soDigitos(*entry.ChaveAcesso)
		entry.ChaveAcesso = strPtr(chave)
		if chave != "" {
			existente, err := uc.Docs.FindEntryByChave(ctx, chave)
			if err != nil {
				return nil, err
			}
			if existente != nil {
				return nil, errorsuc.NewConflictError(fmt.Sprintf("a NF-e de chave %s já foi lançada como entrada %d", chave, existente.ID))
			}
		}
	}

	// Valor contábil do item: produto + IPI + a parte do frete/seguro/desconto
	// do cabeçalho proporcional ao valor do item.
	somaProdutos := decimal.Zero
	for _, it := range itens {
		somaProdutos = somaProdutos.Add(decimal.NewFromFloat(it.TotalPrice))
	}
	extras := decimal.NewFromFloat(dto.ValorFrete).Add(decimal.NewFromFloat(dto.ValorSeguro)).Sub(decimal.NewFromFloat(dto.ValorDesconto))
	for i, it := range itens {
		if it.Sequence <= 0 {
			it.Sequence = i + 1
		}
		total := decimal.NewFromFloat(it.TotalPrice)
		it.ValorContabil = total.Add(decimal.NewFromFloat(it.ValorIPI))
		if somaProdutos.IsPositive() {
			it.ValorContabil = it.ValorContabil.Add(extras.Mul(total).Div(somaProdutos))
		}
		if it.ItemCode != nil {
			s := "MANUAL"
			agora := time.Now()
			it.ResolutionStrategy, it.ResolvedAt = &s, &agora
			aplicarConversao(it, nil, nil)
		}
	}
	entrada.AjustarValorContabil(itens, decimal.NewFromFloat(dto.ValorTotal))
	if err := conciliacaoAutomatica(ctx, uc.Docs, entry.SupplierCode, itens); err != nil {
		return nil, err
	}
	entry.Itens = itens
	servico := &EntradaServico{Docs: uc.Docs, Fiscal: uc.Repo, Tolerancias: uc.Tolerances}
	if err := servico.Preparar(ctx, entry); err != nil {
		return nil, err
	}
	if err := sugerirPlanosDeContas(ctx, uc.Docs, contaFornecedor, itens); err != nil {
		return nil, err
	}

	if len(dto.Parcelas) > 0 {
		parcelas, err := parcelasDoDTO(dto.Parcelas)
		if err != nil {
			return nil, err
		}
		entry.Parcelas = parcelas
	} else if _, aPagar, _, _ := entrada.PlanoFinanceiro(entry); aPagar.IsPositive() {
		entry.Parcelas = []*entity.FiscalEntryInstallment{{
			Numero:         1,
			DataVencimento: entry.DataEmissao.AddDate(0, 0, 30),
			Valor:          aPagar,
			Origem:         entity.ParcelaOrigemPadrao,
		}}
	}
	semDistribuicao := true
	for _, p := range entry.Parcelas {
		if len(p.Distribuicao) > 0 {
			semDistribuicao = false
		}
	}
	if semDistribuicao {
		if err := Distribuir(entry); err != nil {
			return nil, errorsuc.NewValidationError(err.Error())
		}
	}
	status, err := servico.Status(ctx, entry)
	if err != nil {
		return nil, err
	}
	entry.Status = status

	if _, err := uc.Docs.CreateEntryDocument(ctx, entry); err != nil {
		return nil, err
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, entry.ID)
	if err != nil {
		return nil, err
	}
	doc.Warnings = entry.Warnings
	return servico.Responder(ctx, doc)
}

func resolveSupplierItems(ctx context.Context, resolver ports.ItemSupplierResolver, supplier *int64, items []*entity.FiscalEntryItem) error {
	if resolver == nil || supplier == nil {
		return nil
	}
	for _, item := range items {
		if item.ItemCode != nil {
			strategy := "MANUAL"
			item.ResolutionStrategy = &strategy
			now := time.Now()
			item.ResolvedAt = &now
			continue
		}
		code := ""
		if item.SupplierItemCode != nil {
			code = *item.SupplierItemCode
		}
		description := ""
		if item.Description != nil {
			description = *item.Description
		}
		resolved, err := resolver.ResolveExternal(ctx, *supplier, code, description)
		if err != nil {
			return err
		}
		if resolved == nil {
			strategy := "NAO_RESOLVIDO"
			item.ResolutionStrategy = &strategy
			continue
		}
		item.ItemCode = &resolved.ItemCode
		item.ItemSupplierID = &resolved.LinkID
		item.ResolutionStrategy = &resolved.Strategy
		now := time.Now()
		item.ResolvedAt = &now
	}
	return nil
}
