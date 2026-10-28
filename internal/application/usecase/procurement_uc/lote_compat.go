package procurement_uc

// Rotas antigas da inspeção de recebimento, mantidas para o desktop que ainda
// não usa /api/quality/lotes. Desde a migração 000418 elas leem e gravam o
// lote de inspeção do módulo de qualidade: o id da ordem de recebimento é o id
// do lote; o registro de procurement do tipo RECEIVING_INSPECTION é resolvido
// pelo vínculo legado.
//
// Diferenças deliberadas em relação ao comportamento antigo:
//   - a análise/destinação é a decisão de uso: precisa distribuir o lote
//     inteiro e sempre move o estoque, na mesma transação (antes gravava os
//     números e movia depois, e aceitava analisar duas vezes);
//   - a conformidade da medição é decidida pelos limites do plano, não pelo
//     is_approved enviado;
//   - o roteiro é procurado na empresa da sessão (antes, empresa 1 fixa).

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/lote_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/procurement/entity"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
)

const (
	legadoOrdem    = "RECEIVING_ORDER"
	legadoRegistro = "PROCUREMENT_RECORD"
)

func (uc *UseCase) lotes() (*lote_uc.UseCase, error) {
	if uc.Lotes == nil {
		return nil, errorsuc.NewValidationError("o lote de inspeção não está configurado")
	}
	return uc.Lotes, nil
}

func decimalDe(f float64) decimal.Decimal { return decimal.NewFromFloat(f).Round(6) }

func floatPtr(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	v := d.InexactFloat64()
	return &v
}

// loteDeRecebimento: o lote pelo id da ordem — que é o próprio id do lote.
func (uc *UseCase) loteDeRecebimento(ctx context.Context, id int64) (*lote_uc.LoteResponse, error) {
	lu, err := uc.lotes()
	if err != nil {
		return nil, err
	}
	r, err := lu.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.Origem == dom.OrigemProcesso || r.Origem == dom.OrigemFinal {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("ordem de inspeção %d não encontrada", id))
	}
	return r, nil
}

func statusLegado(l *dom.Lote) string {
	switch l.Status {
	case dom.Cancelado:
		return "CANCELLED"
	case dom.Aberto:
		if len(l.Resultados) > 0 {
			return "PENDING_ANALYSIS"
		}
		return "PENDING_INSPECTION"
	}
	if l.Pulado {
		return "SKIPPED"
	}
	if l.Decisao == nil {
		return "APPROVED"
	}
	switch l.Decisao.Tipo {
	case dom.DecisaoRejeitado:
		return "REJECTED"
	case dom.DecisaoParcial:
		return "PARTIAL"
	}
	return "APPROVED"
}

func origemLegada(l *dom.Lote) string {
	switch {
	case l.Origem == dom.OrigemManual:
		return "MANUAL"
	case l.ReceivingNoticeCode != nil:
		return "RECEIVING_NOTICE"
	case l.FiscalEntryCode != nil:
		return "FISCAL_ENTRY"
	}
	return "PURCHASE_RECEIPT"
}

func ordemLegada(l *dom.Lote) *response.ReceivingInspectionOrderResponse {
	out := &response.ReceivingInspectionOrderResponse{
		ID: l.ID, OrderNumber: l.Numero, RouteID: l.ReceivingRouteID, Source: origemLegada(l), SupplierCode: l.SupplierCode,
		PurchaseOrderCode: l.PurchaseOrderCode, PurchaseOrderItemCode: l.PurchaseOrderItemCode, FiscalEntryCode: l.FiscalEntryCode,
		ReceivingNoticeCode: l.ReceivingNoticeCode, ItemCode: l.ItemCode, Mask: l.Mask, Lot: l.Lote, SerialNumber: l.Serie,
		Quantity: l.Quantidade.InexactFloat64(), Status: statusLegado(l), Certificate: l.Certificado, SupplierNote: l.NotaFornecedor,
		Model: l.Modelo, Notes: l.Observacoes, CreatedAt: l.CreatedAt, CreatedBy: l.CreatedBy,
	}
	if l.WarehouseID != nil {
		out.WarehouseID = *l.WarehouseID
	}
	unidades := map[int]bool{}
	for _, r := range l.Resultados {
		unidades[r.AmostraIndice] = true
	}
	out.InspectedQty = float64(len(unidades))
	if d := l.Decisao; d != nil {
		out.ApprovedQty = d.Aceita.InexactFloat64()
		out.RestrictedQty = d.Restrita.InexactFloat64()
		out.ReworkQty = d.Retrabalho.InexactFloat64()
		out.RejectedQty = d.Rejeitada.Add(d.Sucata).InexactFloat64()
	}
	return out
}

func (uc *UseCase) GenerateReceivingInspectionOrder(ctx context.Context, dto request.GenerateReceivingInspectionOrderDTO) (*response.ReceivingInspectionOrderResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ItemCode <= 0 || dto.WarehouseID <= 0 || dto.Quantity <= 0 {
		return nil, errorsuc.NewValidationError("código do item, depósito e quantidade são obrigatórios")
	}
	if dto.Source == "" {
		dto.Source = "MANUAL"
	}
	if !validInspectionSource(dto.Source) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("origem %q inválida", dto.Source))
	}
	lu, err := uc.lotes()
	if err != nil {
		return nil, err
	}
	origem := string(dom.OrigemRecebimento)
	if dto.Source == "MANUAL" {
		origem = string(dom.OrigemManual)
	}
	wh := dto.WarehouseID
	r, err := lu.Abrir(ctx, lote_uc.AberturaDTO{
		Origem: origem, ItemCode: dto.ItemCode, Mask: dto.Mask, Lote: dto.Lot, Serie: dto.SerialNumber, Quantidade: decimalDe(dto.Quantity),
		WarehouseID: &wh, SupplierCode: dto.SupplierCode, PurchaseOrderCode: dto.PurchaseOrderCode, PurchaseOrderItemCode: dto.PurchaseOrderItemCode,
		FiscalEntryCode: dto.FiscalEntryCode, ReceivingNoticeCode: dto.ReceivingNoticeCode, Certificado: dto.Certificate,
		NotaFornecedor: dto.SupplierNote, Modelo: dto.Model, Observacoes: dto.Notes,
	})
	if err != nil {
		return nil, err
	}
	return ordemLegada(r.Lote), nil
}

// GetReceivingInspectionOrder abre uma ordem de inspeção da empresa da sessão (VINS0400).
func (uc *UseCase) GetReceivingInspectionOrder(ctx context.Context, id int64) (*response.ReceivingInspectionOrderResponse, error) {
	r, err := uc.loteDeRecebimento(ctx, id)
	if err != nil {
		return nil, err
	}
	return ordemLegada(r.Lote), nil
}

func (uc *UseCase) ListReceivingInspectionOrders(ctx context.Context, status string) ([]*response.ReceivingInspectionOrderResponse, error) {
	lu, err := uc.lotes()
	if err != nil {
		return nil, err
	}
	filtro := dom.Filtro{Limite: 500}
	switch strings.ToUpper(status) {
	case "PENDING_INSPECTION", "PENDING_ANALYSIS":
		filtro.Status = string(dom.Aberto)
	case "CANCELLED":
		filtro.Status = string(dom.Cancelado)
	case "APPROVED", "REJECTED", "PARTIAL", "SKIPPED":
		filtro.Status = string(dom.Decidido)
	}
	ls, err := lu.Listar(ctx, filtro)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ReceivingInspectionOrderResponse, 0, len(ls))
	for i := range ls {
		l := &ls[i]
		if l.Origem == dom.OrigemProcesso || l.Origem == dom.OrigemFinal {
			continue
		}
		o := ordemLegada(l)
		// A lista não carrega resultados: PENDING_ANALYSIS só no detalhe.
		if status != "" && o.Status != strings.ToUpper(status) && !(l.Status == dom.Aberto && filtro.Status == string(dom.Aberto)) {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

func (uc *UseCase) RecordReceivingInspectionResult(ctx context.Context, orderID int64, dto request.RecordReceivingInspectionResultDTO) (*response.ReceivingInspectionResultResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if orderID <= 0 || dto.Sequence <= 0 {
		return nil, errorsuc.NewValidationError("informe a ordem e a sequência")
	}
	if dto.SampleIndex <= 0 {
		dto.SampleIndex = 1
	}
	r, err := uc.loteDeRecebimento(ctx, orderID)
	if err != nil {
		return nil, err
	}
	var carac *dom.Caracteristica
	for i, c := range r.Caracteristicas {
		if (dto.StepID != nil && c.OrigemStep != nil && *c.OrigemStep == *dto.StepID) || (dto.StepID == nil && c.Sequencia == dto.Sequence) {
			carac = &r.Caracteristicas[i]
			break
		}
	}
	if carac == nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a etapa %d não existe no roteiro copiado para a ordem %d", dto.Sequence, r.Numero))
	}
	var valor *decimal.Decimal
	if dto.MeasuredValue != nil {
		v := decimalDe(*dto.MeasuredValue)
		valor = &v
	}
	aprovado := dto.IsApproved
	res, err := uc.Lotes.GravarResultados(ctx, orderID, []lote_uc.ResultadoDTO{{
		CaracteristicaID: carac.ID, AmostraIndice: dto.SampleIndex, Valor: valor, Conforme: &aprovado,
		Atributo: dto.AttributeDescription, Observacao: dto.Notes,
	}})
	if err != nil {
		return nil, err
	}
	for _, x := range res.Resultados {
		if x.CaracteristicaID == carac.ID && x.AmostraIndice == dto.SampleIndex {
			return &response.ReceivingInspectionResultResponse{
				ID: x.ID, OrderID: orderID, StepID: carac.OrigemStep, Sequence: carac.Sequencia, SampleIndex: x.AmostraIndice,
				MeasuredValue: floatPtr(x.Valor), MinValue: floatPtr(carac.Minimo), MaxValue: floatPtr(carac.Maximo),
				AttributeDescription: x.Atributo, IsApproved: x.Conforme, Notes: x.Observacao, CreatedAt: x.CreatedAt, CreatedBy: x.CreatedBy,
			}, nil
		}
	}
	return nil, fmt.Errorf("resultado gravado não encontrado no lote %d", orderID)
}

var tratamentoLegado = map[string]string{
	"ACCEPT_WITH_RESTRICTION": "", "RETURN_TO_SUPPLIER": "DEVOLVER", "SCRAP": "SUCATEAR",
	"REWORK": "RETRABALHAR", "SORTING": "SELECIONAR", "CONCESSION": "CONCESSAO",
}

// AnalyzeReceivingInspectionOrder é a decisão de uso do lote.
func (uc *UseCase) AnalyzeReceivingInspectionOrder(ctx context.Context, orderID int64, dto request.AnalyzeReceivingInspectionOrderDTO) (*response.ReceivingInspectionAnalysisResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) || !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !validInspectionTreatment(dto.Treatment) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("tratamento %q inválido", dto.Treatment))
	}
	if _, err := uc.loteDeRecebimento(ctx, orderID); err != nil {
		return nil, err
	}
	restrito := dto.RestrictedWarehouseID
	if restrito == nil {
		restrito = dto.DestinationWarehouseID
	}
	trat := tratamentoLegado[dto.Treatment]
	var tratamento *string
	if trat != "" {
		tratamento = &trat
	}
	// Sem retrabalho/rejeição explícitos, a sucata do tratamento SCRAP é a rejeitada.
	dec := lote_uc.DecisaoDTO{
		Aceita: decimalDe(dto.ConformQty), Restrita: decimalDe(dto.RestrictedQty), Retrabalho: decimalDe(dto.ReworkQty), Rejeitada: decimalDe(dto.RejectedQty),
		DestinoAceito: dto.DestinationWarehouseID, DestinoRestrito: restrito, DestinoRetrabalho: dto.ReworkWarehouseID, DestinoRejeitado: dto.RejectionWarehouseID,
		Tratamento: tratamento, AfetaIQF: &dto.AffectsSupplierScore, Motivo: dto.Notes,
	}
	r, err := uc.Lotes.Decidir(ctx, orderID, dec)
	if err != nil {
		return nil, err
	}
	d := r.Decisao
	return &response.ReceivingInspectionAnalysisResponse{
		ID: r.ID, OrderID: r.ID, ConformQty: d.Aceita.InexactFloat64(), RejectedQty: d.Rejeitada.Add(d.Sucata).InexactFloat64(),
		ReworkQty: d.Retrabalho.InexactFloat64(), RestrictedQty: d.Restrita.InexactFloat64(), Treatment: dto.Treatment,
		AffectsSupplierScore: d.AfetaIQF, Notes: d.Motivo, AnalyzedAt: d.DecididoEm, AnalyzedBy: d.DecididoPor, Order: ordemLegada(r.Lote),
	}, nil
}

// DisposeInspection é a destinação do registro de procurement do tipo
// RECEIVING_INSPECTION: vira a decisão de uso do lote ligado a ele (aberto na
// hora, se o registro é de depois da migração).
func (uc *UseCase) DisposeInspection(ctx context.Context, id int64, dto request.DisposeReceivingInspectionDTO) (*response.ReceivingInspectionDispositionResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) || !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ApprovedQty < 0 || dto.RejectedQty < 0 || dto.ApprovedQty+dto.RejectedQty <= 0 {
		return nil, errorsuc.NewValidationError("informe a quantidade aprovada ou a rejeitada, maior que zero")
	}
	rec, err := uc.Repo.GetRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec.RecordType != entity.RecordReceivingInspection {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o registro %d não é uma inspeção de recebimento", id))
	}
	if rec.ItemCode == nil || rec.WarehouseID == nil {
		return nil, errorsuc.NewValidationError("inspeção exige código do item e depósito")
	}
	lu, err := uc.lotes()
	if err != nil {
		return nil, err
	}
	l, err := lu.ObterPorLegado(ctx, legadoRegistro, id)
	if err != nil {
		if !errorsuc.EhNaoEncontrado(err) {
			return nil, err
		}
		kind, legado := legadoRegistro, id
		aberto, errAbrir := lu.Abrir(ctx, lote_uc.AberturaDTO{
			Origem: string(dom.OrigemRecebimento), ItemCode: *rec.ItemCode, Mask: rec.Mask, Quantidade: decimalDe(rec.Quantity),
			WarehouseID: rec.WarehouseID, SupplierCode: rec.SupplierCode, PurchaseOrderCode: rec.PurchaseOrderCode,
			PurchaseOrderItemCode: rec.PurchaseOrderItemCode, Observacoes: rec.Reference, SemRoteiro: true,
			LegacyKind: &kind, LegacyID: &legado,
		})
		if errAbrir != nil {
			return nil, errAbrir
		}
		l = aberto.Lote
	}
	trat := "DEVOLVER"
	r, err := lu.Decidir(ctx, l.ID, lote_uc.DecisaoDTO{
		Aceita: decimalDe(dto.ApprovedQty), Rejeitada: decimalDe(dto.RejectedQty),
		DestinoAceito: dto.DestinationWarehouseID, DestinoRejeitado: dto.QuarantineWarehouseID, Tratamento: &trat, Motivo: dto.Reason,
	})
	if err != nil {
		return nil, err
	}
	next := entity.StatusApproved
	if dto.ApprovedQty == 0 {
		next = entity.StatusRejected
	} else if dto.RejectedQty > 0 {
		next = entity.StatusPartial
	}
	updated, err := uc.Repo.UpdateRecordStatus(ctx, id, next)
	if err != nil {
		return nil, err
	}
	d := r.Decisao
	return &response.ReceivingInspectionDispositionResponse{
		ID: r.ID, RecordID: id, ApprovedQty: d.Aceita.InexactFloat64(), RejectedQty: d.Rejeitada.InexactFloat64(),
		QuarantineWarehouseID: d.DestinoRejeitado, DestinationWarehouseID: d.DestinoAceito, Reason: d.Motivo,
		DisposedAt: d.DecididoEm, DisposedBy: d.DecididoPor, Inspection: toRecordResponse(updated), Movements: []response.StockMovementResponse{},
	}, nil
}
