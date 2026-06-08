package fiscal_uc

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	purchaserepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

// ImportNFePurchaseUseCase downloads a purchase NF-e from Focus, creates the
// fiscal entry and automatically records ENTRADA movements in stock for each
// line item whose product code can be parsed as a numeric item_code.
type ImportNFePurchaseUseCase struct {
	FiscalRepo fiscalrepo.FiscalRepository
	StockRepo  stockrepo.StockRepository
	Auth       ports.AuthService
	// SupplierDefaults is optional. When set, the emitter CNPJ is matched to a
	// registered supplier and linked on the fiscal entry. Nil disables it.
	SupplierDefaults ports.SupplierPurchasingDefaultsProvider
	// PurchaseOrderRepo is optional. When set together with a purchase order code,
	// the received quantities are written down against the purchase order lines.
	PurchaseOrderRepo purchaserepo.PurchaseOrderRepository
}

type ImportNFePurchaseDTO struct {
	ChaveAcesso       string `json:"chave_acesso"`
	PurchaseOrderCode *int64 `json:"purchase_order_code,omitempty"`
	WarehouseID       int64  `json:"warehouse_id"`
}

type ImportNFePurchaseResult struct {
	Entry                  *entity.FiscalEntry `json:"entry"`
	MovementsCreated       int                 `json:"movements_created"`
	Skipped                []string            `json:"skipped,omitempty"`
	SupplierMatched        bool                `json:"supplier_matched"`
	PurchaseOrderLinesDown int                 `json:"purchase_order_lines_written_down"`
}

func (uc *ImportNFePurchaseUseCase) Execute(ctx context.Context, dto ImportNFePurchaseDTO) (*ImportNFePurchaseResult, error) {
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	cfg, err := uc.FiscalRepo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading fiscal config: %w", err)
	}
	if cfg.FocusNfeToken == nil || *cfg.FocusNfeToken == "" {
		return nil, fmt.Errorf("token Focus NF-e não configurado")
	}
	focusCli := focusnfe.NewClient(*cfg.FocusNfeToken, cfg.FocusNfeAmbiente)

	nfe, err := focusCli.ConsultarNFePorChave(ctx, dto.ChaveAcesso)
	if err != nil {
		return nil, fmt.Errorf("downloading NF-e: %w", err)
	}

	dataEmissaoStr := nfe.DataEmissao
	if len(dataEmissaoStr) > 10 {
		dataEmissaoStr = dataEmissaoStr[:10]
	}
	dataEmissao, _ := time.Parse("2006-01-02", dataEmissaoStr)
	dataEntrada := time.Now()

	// Parse NF-e number as int64 (Focus returns as string).
	var numNF int64
	if n, parseErr := strconv.ParseInt(nfe.NumeroNF, 10, 64); parseErr == nil {
		numNF = n
	}

	// Best-effort: link the emitter to a registered supplier by CNPJ/CPF.
	var supplierCode *int64
	supplierMatched := false
	if uc.SupplierDefaults != nil && nfe.CnpjEmitente != "" {
		if code, found, _ := uc.SupplierDefaults.FindSupplierCodeByDocument(ctx, nfe.CnpjEmitente); found {
			c := code
			supplierCode = &c
			supplierMatched = true
		}
	}

	entry := &entity.FiscalEntry{
		ChaveAcesso:         &dto.ChaveAcesso,
		NumeroNF:            numNF,
		Serie:               nfe.Serie,
		DataEmissao:         dataEmissao,
		DataEntrada:         dataEntrada,
		CnpjEmitente:        nfe.CnpjEmitente,
		RazaoSocialEmitente: nfe.NomeEmitente,
		ValorTotal:          nfe.ValorTotal,
		TipoDocumento:       "NFE",
		PurchaseOrderCode:   dto.PurchaseOrderCode,
		SupplierCode:        supplierCode,
		Status:              entity.EntryStatusPending,
		CreatedBy:           userID,
	}
	created, err := uc.FiscalRepo.CreateEntry(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("creating fiscal entry: %w", err)
	}

	movementsCreated := 0
	var skipped []string
	receivedByItem := make(map[int64]float64)

	for _, item := range nfe.Items {
		itemCode, parseErr := strconv.ParseInt(item.CodigoProduto, 10, 64)
		if parseErr != nil || itemCode <= 0 {
			skipped = append(skipped, fmt.Sprintf("item %d: product_code '%s' is not a numeric item_code",
				item.NumeroItem, item.CodigoProduto))
			continue
		}
		receivedByItem[itemCode] += item.QuantidadeComercial

		ic := itemCode
		entryItem := &entity.FiscalEntryItem{
			FiscalEntryID: created.ID,
			Sequence:      item.NumeroItem,
			ItemCode:      &ic,
			Cfop:          item.CFOP,
			Quantity:      item.QuantidadeComercial,
			UnitPrice:     item.ValorUnitario,
			TotalPrice:    item.ValorTotal,
		}
		_, _ = uc.FiscalRepo.CreateEntryItem(ctx, entryItem)

		refType := stockentity.ReferenceTypeNFEntry
		mov := &stockentity.StockMovement{
			ItemCode:      itemCode,
			WarehouseID:   dto.WarehouseID,
			MovementType:  stockentity.MovementTypeIn,
			Quantity:      item.QuantidadeComercial,
			UnitPrice:     item.ValorUnitario,
			TotalPrice:    item.ValorTotal,
			ReferenceType: &refType,
			ReferenceCode: &created.ID,
			CreatedBy:     userID,
		}
		if _, moveErr := uc.StockRepo.CreateMovement(ctx, mov); moveErr == nil {
			movementsCreated++
		} else {
			skipped = append(skipped, fmt.Sprintf("item %d: stock movement failed: %v", item.NumeroItem, moveErr))
		}
	}

	if movementsCreated > 0 {
		_, _ = uc.FiscalRepo.UpdateEntryStatus(ctx, created.ID, entity.EntryStatusApproved)
	}

	// Write the received quantities down against the purchase order lines so the
	// order reflects what has actually arrived (status PARTIAL/RECEIVED).
	linesDown := 0
	if uc.PurchaseOrderRepo != nil && dto.PurchaseOrderCode != nil && len(receivedByItem) > 0 {
		if n, recErr := uc.PurchaseOrderRepo.RegisterReceipts(ctx, *dto.PurchaseOrderCode, receivedByItem); recErr == nil {
			linesDown = n
		} else {
			skipped = append(skipped, fmt.Sprintf("purchase order write-down failed: %v", recErr))
		}
	}

	return &ImportNFePurchaseResult{
		Entry:                  created,
		MovementsCreated:       movementsCreated,
		Skipped:                skipped,
		SupplierMatched:        supplierMatched,
		PurchaseOrderLinesDown: linesDown,
	}, nil
}
