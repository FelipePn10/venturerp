package purchase_order_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/shopspring/decimal"
)

// ReceivingInspectionGate hooks receiving inspection into physical receipt
// (FINS0212). When a received line matches an active inspection route the material
// is routed into the inspection warehouse and an inspection order is opened.
// Implemented by the procurement use case; kept as a port to avoid coupling.
type ReceivingInspectionGate interface {
	// ResolveInspectionRoute returns the inspection warehouse for an active route
	// matching the item, and whether a route matched.
	ResolveInspectionRoute(ctx context.Context, itemCode int64, mask string) (inspectionWarehouseID int64, matched bool)
	// OpenInspectionOrderFromReceipt opens an inspection order for material routed
	// into the inspection warehouse; returns the created order id and number.
	// AbrirDoRecebimento abre o lote de inspeção (id e número).
	AbrirDoRecebimento(ctx context.Context, item int64, mask string, quantidade decimal.Decimal, almoxInspecao int64,
		fornecedor, pedido, linha *int64, lote *string, notaEntrada *int64) (int64, int64, error)
}

// RecebimentoRepos são os repositórios ligados à transação do recebimento.
type RecebimentoRepos struct {
	Pedidos porepo.PurchaseOrderRepository
	Estoque stockrepo.StockRepository
}

// EmTransacaoRecebimento abre a transação do recebimento: as entradas e a baixa
// nas linhas do pedido valem juntas, ou nada vale.
type EmTransacaoRecebimento func(ctx context.Context, fn func(RecebimentoRepos) error) error

type ReceivePurchaseOrderUseCase struct {
	Repo      porepo.PurchaseOrderRepository
	StockRepo stockrepo.StockRepository
	// Transacao é opcional na montagem: sem ela cada gravação vale sozinha.
	Transacao EmTransacaoRecebimento
	Auth      ports.AuthService
	// Inspection is optional; nil disables auto-inspection on receipt.
	Inspection ReceivingInspectionGate
	Tolerances ports.PurchaseToleranceEvaluator
	// Unidade converte linhas antigas, gravadas sem quantidade interna.
	Unidade *UnidadeInterna
	// Regras traz o que o cadastro do item define para o recebimento. Opcional.
	Regras RegrasDeRecebimento
}

// RegraDoItem é o que o cadastro do item define para o recebimento.
type RegraDoItem struct {
	Roteiro              string
	ToleranciaExcessoPct *decimal.Decimal
	ToleranciaFaltaPct   *decimal.Decimal
	ValidadeDias         *int
	VidaMinimaDias       *int
	ExigeChecklist       bool
	ExigeCertificado     bool
}

// EncerramentoPorTolerancia cancela o saldo que ficou dentro da tolerância de falta.
type EncerramentoPorTolerancia interface {
	EncerrarSaldoPorTolerancia(ctx context.Context, pedido int64, porLinha map[int64]decimal.Decimal) error
}

// CertificadoDoLote grava corrida e certificado no lote recebido.
type CertificadoDoLote interface {
	RegistrarCertificadoDoLote(ctx context.Context, itemCode int64, mask, lote string, corrida, certificado *string) error
}

type certificadoRecebido struct {
	item                 int64
	mask, lote           string
	corrida, certificado *string
}

// conferirDocumentosDoRecebimento cobra o que o cadastro do item exige na doca.
func conferirDocumentosDoRecebimento(regra RegraDoItem, item request.ReceivePurchaseOrderItemDTO) error {
	if regra.ExigeChecklist && !item.ChecklistConfirmed {
		return errorsuc.NewValidationError(fmt.Sprintf("confirme o check list de recebimento do item %d", item.PurchaseOrderItemCode))
	}
	if !regra.ExigeCertificado {
		return nil
	}
	vazio := func(s *string) bool { return s == nil || strings.TrimSpace(*s) == "" }
	if vazio(item.Lot) || vazio(item.HeatNumber) || vazio(item.Certificate) {
		return errorsuc.NewValidationError(fmt.Sprintf("o item %d exige lote, corrida e certificado do material no recebimento", item.PurchaseOrderItemCode))
	}
	return nil
}

// RegrasDeRecebimento lê o cadastro do item na empresa da sessão.
type RegrasDeRecebimento interface {
	RegraDeRecebimento(ctx context.Context, itemCode int64) (RegraDoItem, error)
}

func (uc *ReceivePurchaseOrderUseCase) Execute(ctx context.Context, dto request.ReceivePurchaseOrderDTO) (*response.PurchaseOrderReceiptResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) || !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.PurchaseOrderCode <= 0 {
		return nil, errorsuc.NewValidationError("código do pedido de compra é obrigatório")
	}
	if len(dto.Items) == 0 {
		return nil, errorsuc.NewValidationError("ao menos um item de recebimento é obrigatório")
	}

	order, err := uc.Repo.GetByCode(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	if order.Status == poentity.PurchaseOrderStatusCANCELLED || order.Status == poentity.PurchaseOrderStatusRECEIVED {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("pedido de compra %d não pode ser recebido no status %s", order.Code, order.Status))
	}
	if err := order.AceitaRecebimento(); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	lines, err := uc.Repo.ListItems(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	byCode := make(map[int64]*poentity.PurchaseOrderItem, len(lines))
	for _, line := range lines {
		byCode[line.Code] = line
	}

	receivedByLine := make(map[int64]decimal.Decimal, len(dto.Items))
	receiptLines := make([]response.PurchaseReceiptLine, 0, len(dto.Items))
	movements := make([]response.StockMovementResponse, 0, len(dto.Items))
	inspectionOrders := make([]response.ReceiptInspectionOrder, 0)
	var certificados []certificadoRecebido
	encerrar := map[int64]decimal.Decimal{}
	regras := map[int64]RegraDoItem{}
	warnings := make([]string, 0)
	entradas := make([]*stockentity.StockMovement, 0, len(dto.Items))
	type loteAbrir struct {
		linha    *poentity.PurchaseOrderItem
		item     request.ReceivePurchaseOrderItemDTO
		qtd      decimal.Decimal
		almox    int64
		qtdFloat float64
	}
	var lotesAbrir []loteAbrir
	refType := stockentity.ReferenceTypePurchaseOrder
	refCode := dto.PurchaseOrderCode

	for _, item := range dto.Items {
		if !item.Quantity.IsPositive() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("quantidade deve ser positiva para o item %d do pedido de compra", item.PurchaseOrderItemCode))
		}
		if item.WarehouseID <= 0 {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("depósito é obrigatório para o item %d do pedido de compra", item.PurchaseOrderItemCode))
		}
		line := byCode[item.PurchaseOrderItemCode]
		if line == nil {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d não pertence ao pedido de compra %d", item.PurchaseOrderItemCode, dto.PurchaseOrderCode))
		}
		if line.Status == poentity.PurchaseOrderItemStatusCANCELLED {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d do pedido de compra está cancelado", item.PurchaseOrderItemCode))
		}
		folga := decimal.New(1, -4)
		// A mesma linha pode vir em vários lotes: a tolerância olha o total recebido.
		recebidoAntes := line.ReceivedQty.Add(receivedByLine[line.Code])
		remaining := line.RequestedQty.Sub(recebidoAntes).Sub(line.CancelledQty)
		var regra RegraDoItem
		if uc.Regras != nil {
			var errRegra error
			if regra, errRegra = uc.Regras.RegraDeRecebimento(ctx, line.ItemCode); errRegra != nil {
				return nil, errRegra
			}
		}
		toleranceHandled := false
		// A tolerância do próprio item é a mais específica: vence a da empresa,
		// a do fornecedor e a da linha.
		if regra.ToleranciaExcessoPct != nil {
			toleranceHandled = true
			limite := line.RequestedQty.Add(regra.ToleranciaExcessoPct.Div(decimal.NewFromInt(100)).Mul(line.RequestedQty))
			if recebidoAntes.Add(item.Quantity).GreaterThan(limite.Add(folga)) {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d passou da tolerância de excesso do cadastro (%s%%)", item.PurchaseOrderItemCode, regra.ToleranciaExcessoPct.String()))
			}
		}
		if !toleranceHandled && uc.Tolerances != nil && recebidoAntes.Add(item.Quantity).GreaterThan(line.RequestedQty.Add(folga)) {
			action, message, exceeded, toleranceErr := uc.Tolerances.EvaluatePurchaseTolerance(ctx, order.SupplierCode, "QUANTITY", "RECEIVING_NOTICE", line.RequestedQty, recebidoAntes.Add(item.Quantity))
			if toleranceErr != nil {
				return nil, toleranceErr
			}
			if action != "" {
				toleranceHandled = true
			}
			if exceeded && action == "BLOCK" {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d passou da tolerância de recebimento: %s", item.PurchaseOrderItemCode, message))
			}
			if exceeded && action == "WARN" {
				warnings = append(warnings, fmt.Sprintf("item %d: %s", item.PurchaseOrderItemCode, message))
			}
		}
		if !toleranceHandled {
			toleranceRemaining := line.TolerancePct.Div(decimal.NewFromInt(100)).Mul(line.RequestedQty).Add(line.CancelledToleranceQty)
			if item.Quantity.GreaterThan(remaining.Add(toleranceRemaining).Add(folga)) {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("a quantidade recebida %s passa do saldo de %s do item %d do pedido de compra", item.Quantity.StringFixed(4), remaining.StringFixed(4), item.PurchaseOrderItemCode))
			}
		}

		if !line.InternalQty.IsPositive() && uc.Unidade != nil {
			// Linha anterior à conversão obrigatória: converte agora, pela
			// unidade de estoque do cadastro, em vez de receber em unidade de compra.
			copia := *line
			if err := uc.Unidade.Preencher(ctx, &copia); err != nil {
				return nil, err
			}
			line.InternalQty, line.InternalPrice = copia.InternalQty, copia.InternalPrice
		}
		// Conta em decimal: a quantidade convertida para a unidade interna e o
		// valor vão para o estoque, onde a soma de várias entradas não pode derivar.
		stockQtyExata := item.Quantity
		unitPrice := line.UnitPrice
		if line.InternalQty.IsPositive() && line.RequestedQty.IsPositive() {
			stockQtyExata = stockQtyExata.Mul(line.InternalQty).DivRound(line.RequestedQty, 6)
		}
		if line.InternalPrice.IsPositive() {
			unitPrice = line.InternalPrice
		}
		totalPrice := stockQtyExata.Mul(unitPrice).Round(4)
		stockQty := stockQtyExata.InexactFloat64()
		notes := mergeReceiptNotes(dto.Notes, item.Notes)
		expirationDate, err := parseReceiptDate(item.ExpirationDate)
		if err != nil {
			return nil, err
		}
		if err := conferirVidaUtil(regra, expirationDate, time.Now(), item.PurchaseOrderItemCode); err != nil {
			return nil, err
		}
		if err := conferirDocumentosDoRecebimento(regra, item); err != nil {
			return nil, err
		}
		if regra.ExigeCertificado {
			certificados = append(certificados, certificadoRecebido{line.ItemCode, line.Mask, *item.Lot, item.HeatNumber, item.Certificate})
		}
		regras[line.Code] = regra

		// FINS0212: if an active inspection route matches, receive into the
		// inspection warehouse instead of the requested one so nothing reaches
		// available stock without a laudo.
		targetWarehouse := item.WarehouseID
		underInspection := false
		if uc.Inspection != nil && regra.Roteiro != itementity.RecebimentoDireto {
			if inspWH, matched := uc.Inspection.ResolveInspectionRoute(ctx, line.ItemCode, line.Mask); matched && inspWH > 0 {
				targetWarehouse = inspWH
				underInspection = true
			}
		}
		if regra.Roteiro == itementity.RecebimentoInspecao && !underInspection {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("o item %d exige inspeção no recebimento e não há rota de inspeção ativa: cadastre a rota antes de receber", item.PurchaseOrderItemCode))
		}

		mov := &stockentity.StockMovement{
			ItemCode:          line.ItemCode,
			Mask:              line.Mask,
			WarehouseID:       targetWarehouse,
			MovementType:      stockentity.MovementTypeIn,
			Quantity:          stockQtyExata,
			UnitPrice:         unitPrice,
			TotalPrice:        totalPrice,
			ReferenceType:     &refType,
			ReferenceCode:     &refCode,
			Lot:               item.Lot,
			SecondaryQuantity: item.SecondaryQuantity,
			SerialNumber:      item.SerialNumber,
			Batch:             item.Batch,
			ExpirationDate:    expirationDate,
			Notes:             notes,
			CreatedBy:         actor,
		}
		entradas = append(entradas, mov)
		if underInspection {
			lotesAbrir = append(lotesAbrir, loteAbrir{linha: line, item: item, qtd: stockQtyExata, almox: targetWarehouse, qtdFloat: stockQty})
		}

		receivedByLine[line.Code] = receivedByLine[line.Code].Add(item.Quantity)
		receiptLines = append(receiptLines, response.PurchaseReceiptLine{
			PurchaseOrderItemCode: line.Code,
			ItemCode:              line.ItemCode,
			Mask:                  line.Mask,
			WarehouseID:           targetWarehouse,
			Quantity:              item.Quantity.InexactFloat64(),
			StockQuantity:         stockQty,
			RemainingQty:          remaining.Sub(item.Quantity).InexactFloat64(),
			UnderInspection:       underInspection,
		})
	}
	// A falta se apura depois de somar todos os lotes da linha.
	for _, item := range dto.Items {
		line, regra := byCode[item.PurchaseOrderItemCode], regras[item.PurchaseOrderItemCode]
		if _, feito := encerrar[line.Code]; feito || regra.ToleranciaFaltaPct == nil {
			continue
		}
		falta := line.RequestedQty.Sub(line.ReceivedQty).Sub(line.CancelledQty).Sub(receivedByLine[line.Code])
		if falta.IsPositive() && !falta.GreaterThan(regra.ToleranciaFaltaPct.Div(decimal.NewFromInt(100)).Mul(line.RequestedQty)) {
			warnings = append(warnings, fmt.Sprintf("item %d: linha encerrada, faltaram %s dentro da tolerância de falta do cadastro", line.Code, falta.StringFixed(4)))
			encerrar[line.Code] = falta
		}
	}

	// Tudo valido antes de gravar: a falha na terceira linha não deixa as duas
	// primeiras no estoque com o pedido ainda em aberto — o que, ao repetir o
	// recebimento, entrava em dobro.
	emTransacao := uc.Transacao
	if emTransacao == nil {
		emTransacao = func(ctx context.Context, fn func(RecebimentoRepos) error) error {
			return fn(RecebimentoRepos{Pedidos: uc.Repo, Estoque: uc.StockRepo})
		}
	}
	err = emTransacao(ctx, func(r RecebimentoRepos) error {
		for _, mov := range entradas {
			created, err := r.Estoque.CreateMovement(ctx, mov)
			if err != nil {
				return err
			}
			movements = append(movements, stockMovementReceiptResponse(created))
		}
		if len(certificados) > 0 {
			gravador, ok := r.Estoque.(CertificadoDoLote)
			if !ok {
				return fmt.Errorf("gravação do certificado do lote indisponível")
			}
			for _, c := range certificados {
				if err := gravador.RegistrarCertificadoDoLote(ctx, c.item, c.mask, c.lote, c.corrida, c.certificado); err != nil {
					return err
				}
			}
		}
		// A falta dentro da tolerância sai antes do recebimento: assim a linha
		// fecha e o status do pedido é calculado uma vez, já com ela encerrada.
		if len(encerrar) > 0 {
			encerrador, ok := r.Pedidos.(EncerramentoPorTolerancia)
			if !ok {
				return fmt.Errorf("encerramento por tolerância indisponível")
			}
			if err := encerrador.EncerrarSaldoPorTolerancia(ctx, dto.PurchaseOrderCode, encerrar); err != nil {
				return err
			}
		}
		matched, err := r.Pedidos.RegisterItemReceipts(ctx, dto.PurchaseOrderCode, receivedByLine)
		if err != nil {
			return err
		}
		if matched != len(receivedByLine) {
			return errorsuc.NewValidationError(fmt.Sprintf("foram gravadas %d linhas de recebimento, mas eram esperadas %d", matched, len(receivedByLine)))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// O lote de inspeção nasce depois: o material já está retido no almoxarifado
	// de inspeção, então um erro aqui vira aviso, não desfaz o recebimento.
	for _, l := range lotesAbrir {
		orderID, orderNumber, err := uc.Inspection.AbrirDoRecebimento(
			ctx, l.linha.ItemCode, l.linha.Mask, l.qtd, l.almox,
			order.SupplierCode, &dto.PurchaseOrderCode, &l.linha.Code, l.item.Lot, nil)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("item %d: o material entrou no almoxarifado de inspeção, mas o lote de inspeção não foi aberto — abra-o na tela de inspeção", l.linha.ItemCode))
			continue
		}
		inspectionOrders = append(inspectionOrders, response.ReceiptInspectionOrder{
			InspectionOrderID:     orderID,
			OrderNumber:           orderNumber,
			PurchaseOrderItemCode: l.linha.Code,
			ItemCode:              l.linha.ItemCode,
			WarehouseID:           l.almox,
			Quantity:              l.qtdFloat,
		})
	}
	updated, err := uc.Repo.GetByCode(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	updated.Items, err = uc.Repo.ListItems(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	return &response.PurchaseOrderReceiptResponse{
		PurchaseOrder:    toPurchaseOrderResponse(updated),
		ReceivedLines:    receiptLines,
		Movements:        movements,
		InspectionOrders: inspectionOrders,
		Warnings:         warnings,
	}, nil
}

// conferirVidaUtil recusa o lote que já chega com menos vida útil que o cadastro
// exige. Sem data informada não há o que conferir: a validade do cadastro será
// contada a partir da entrada.
func conferirVidaUtil(regra RegraDoItem, validade *time.Time, hoje time.Time, linha int64) error {
	if regra.VidaMinimaDias == nil || validade == nil {
		return nil
	}
	dia := time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, validade.Location())
	restante := int(validade.Sub(dia).Hours() / 24)
	if restante < *regra.VidaMinimaDias {
		return errorsuc.NewValidationError(fmt.Sprintf("o lote do item %d chega com %d dia(s) de vida útil e o cadastro exige ao menos %d", linha, restante, *regra.VidaMinimaDias))
	}
	return nil
}

func parseReceiptDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("data de validade %q inválida: use o formato AAAA-MM-DD", *s))
	}
	return &t, nil
}

func mergeReceiptNotes(header, line *string) *string {
	switch {
	case header != nil && line != nil && *header != "" && *line != "":
		merged := *header + " | " + *line
		return &merged
	case line != nil && *line != "":
		return line
	case header != nil && *header != "":
		return header
	default:
		return nil
	}
}

func stockMovementReceiptResponse(m *stockentity.StockMovement) response.StockMovementResponse {
	return response.StockMovementResponse{
		ID:             m.ID,
		ItemCode:       m.ItemCode,
		Mask:           m.Mask,
		WarehouseID:    m.WarehouseID,
		MovementType:   m.MovementType,
		Quantity:       m.Quantity.InexactFloat64(),
		UnitPrice:      m.UnitPrice.InexactFloat64(),
		TotalPrice:     m.TotalPrice.InexactFloat64(),
		ReferenceType:  m.ReferenceType,
		ReferenceCode:  m.ReferenceCode,
		Lot:            m.Lot,
		SerialNumber:   m.SerialNumber,
		Batch:          m.Batch,
		ExpirationDate: m.ExpirationDate,
		Notes:          m.Notes,
		CreatedAt:      m.CreatedAt,
		CreatedBy:      m.CreatedBy,
	}
}
