package shipment_uc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepository "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/google/uuid"
)

// StockReserver is the narrow slice of the stock repository the romaneio needs:
// it reserves stock on separation (a soft allocation), and releases those
// reservations when the romaneio ships (consumed — the NF-e does the real
// write-down) or is cancelled. Kept narrow so the shared StockRepository
// interface is untouched.
type StockReserver interface {
	CreateReservation(ctx context.Context, r *stockentity.StockReservation) (*stockentity.StockReservation, error)
	ListActiveReservations(ctx context.Context) ([]*stockentity.StockReservation, error)
	CancelReservation(ctx context.Context, id int64) error
	ConsumeReservation(ctx context.Context, id int64) error
}

const reservationRefShipment = "SHIPMENT"

// ShipmentUseCase handles outbound logistics: creating a dispatch note (romaneio),
// adding lines, separating (with stock reservation), conferência, packing into
// volumes and dispatch — a state machine mirroring SAP's outbound delivery.
type ShipmentUseCase struct {
	Repo  repository.ShipmentRepository
	Stock StockReserver // optional; when nil, separation skips stock reservation
	// Transacao roda status + reservas do romaneio numa transação só: separar,
	// despachar ou cancelar grava tudo ou nada. Sem ela (testes), roda direto.
	Transacao EmTransacaoRomaneio
	// Regras traz o que o cadastro do item exige do despacho. Opcional.
	Regras RegrasDoRomaneio
}

// RegrasDoRomaneio lê o cadastro dos itens da carga.
type RegrasDoRomaneio interface {
	// ItensQueExigemVolume devolve os códigos que precisam estar em volume.
	ItensQueExigemVolume(ctx context.Context, itens []int64) ([]string, error)
	// VolumesPorUnidade devolve quantos volumes cada unidade do item ocupa.
	VolumesPorUnidade(ctx context.Context, itens []int64) (map[int64]decimal.Decimal, error)
	// Embalagens devolve o item de embalagem e o almoxarifado dele, por item.
	Embalagens(ctx context.Context, itens []int64) (map[int64]EmbalagemDoItem, error)
	// SaldoDaEmbalagem devolve o que há disponível da embalagem no almoxarifado.
	SaldoDaEmbalagem(ctx context.Context, e EmbalagemDoItem) (decimal.Decimal, error)
}

// EmbalagemDoItem é a embalagem que sai junto com o item.
type EmbalagemDoItem struct {
	ItemCode    int64
	WarehouseID int64
}

// BaixaDeEmbalagem é o estoque que o despacho consome. Opcional.
type BaixaDeEmbalagem interface {
	CreateMovement(ctx context.Context, m *stockentity.StockMovement) (*stockentity.StockMovement, error)
}

// RomaneioRepos são os repositórios ligados à transação do romaneio.
type RomaneioRepos struct {
	Romaneios repository.ShipmentRepository
	Estoque   StockReserver
}

// EmTransacaoRomaneio roda fn numa transação; erro devolvido desfaz tudo.
type EmTransacaoRomaneio func(ctx context.Context, fn func(RomaneioRepos) error) error

func (uc *ShipmentUseCase) emTransacao(ctx context.Context, fn func(RomaneioRepos) error) error {
	if uc.Transacao == nil {
		return fn(RomaneioRepos{Romaneios: uc.Repo, Estoque: uc.Stock})
	}
	return uc.Transacao(ctx, fn)
}

type CreateShipmentInput struct {
	ReferenceType       *entity.ShipmentReferenceType
	SalesOrderCode      *int64
	PurchaseOrderCode   *int64
	ProductionOrderCode *int64
	CarrierCode         *int64
	ShippingCarrierID   *int64
	TotalVolumes        int
	TotalNetWeight      decimal.Decimal
	TotalGrossWeight    decimal.Decimal
	TotalCubageM3       decimal.Decimal
	Notes               *string
	CreatedBy           uuid.UUID
}

func (uc *ShipmentUseCase) Create(ctx context.Context, in CreateShipmentInput) (*response.ShipmentResponse, error) {
	code, err := uc.Repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	s := &entity.Shipment{
		Code:                code,
		ReferenceType:       in.ReferenceType,
		SalesOrderCode:      in.SalesOrderCode,
		PurchaseOrderCode:   in.PurchaseOrderCode,
		ProductionOrderCode: in.ProductionOrderCode,
		CarrierCode:         in.CarrierCode,
		ShippingCarrierID:   in.ShippingCarrierID,
		Status:              entity.ShipmentStatusOpen,
		TotalVolumes:        in.TotalVolumes,
		TotalNetWeight:      in.TotalNetWeight,
		TotalGrossWeight:    in.TotalGrossWeight,
		TotalCubageM3:       in.TotalCubageM3,
		Notes:               in.Notes,
		CreatedBy:           in.CreatedBy,
	}
	created, err := uc.Repo.Create(ctx, s)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(created), nil
}

type AddShipmentItemInput struct {
	ShipmentCode       int64
	Sequence           int
	ItemCode           int64
	SalesOrderItemCode *int64
	WarehouseID        *int64
	Quantity           decimal.Decimal
	UnitNetWeight      decimal.Decimal
	UnitGrossWeight    decimal.Decimal
	Notes              *string
}

func (uc *ShipmentUseCase) AddItem(ctx context.Context, in AddShipmentItemInput) (*response.ShipmentItemResponse, error) {
	ship, err := uc.Repo.GetByCode(ctx, in.ShipmentCode)
	if err != nil {
		return nil, err
	}
	if ship.Status != entity.ShipmentStatusOpen && ship.Status != entity.ShipmentStatusSeparated {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("romaneio %d não aceita itens no status %s", in.ShipmentCode, ship.Status))
	}
	item := &entity.ShipmentItem{
		ShipmentID:         ship.ID,
		Sequence:           in.Sequence,
		ItemCode:           in.ItemCode,
		SalesOrderItemCode: in.SalesOrderItemCode,
		WarehouseID:        in.WarehouseID,
		Quantity:           in.Quantity,
		UnitNetWeight:      in.UnitNetWeight,
		UnitGrossWeight:    in.UnitGrossWeight,
		Notes:              in.Notes,
	}
	created, err := uc.Repo.AddItem(ctx, item)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.RecalcTotals(ctx, in.ShipmentCode); err != nil {
		return nil, fmt.Errorf("recalculando os totais do romaneio: %w", err)
	}
	return toShipmentItemResponse(created), nil
}

func (uc *ShipmentUseCase) Get(ctx context.Context, code int64) (*response.ShipmentResponse, error) {
	s, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(s), nil
}

func (uc *ShipmentUseCase) List(ctx context.Context, f repository.ShipmentFilter) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.ListFiltered(ctx, f)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

func (uc *ShipmentUseCase) ListBySalesOrder(ctx context.Context, code int64) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.ListBySalesOrder(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

func (uc *ShipmentUseCase) ListByPurchaseOrder(ctx context.Context, code int64) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.ListByPurchaseOrder(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

func (uc *ShipmentUseCase) ListByProductionOrder(ctx context.Context, code int64) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.ListByProductionOrder(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

// ConferItem records the checked (conferred) quantity of a line. A divergence
// from the planned quantity is preserved (not silently accepted) and surfaces
// when shipping.
func (uc *ShipmentUseCase) ConferItem(ctx context.Context, shipmentCode, itemID int64, conferredQty decimal.Decimal) error {
	ship, err := uc.Repo.GetByCode(ctx, shipmentCode)
	if err != nil {
		return err
	}
	if ship.Status == entity.ShipmentStatusShipped || ship.Status == entity.ShipmentStatusCancelled {
		return errorsuc.NewValidationError(fmt.Sprintf("romaneio %d não pode ser conferido no status %s", shipmentCode, ship.Status))
	}
	if conferredQty.IsNegative() {
		return errorsuc.NewValidationError("quantidade conferida não pode ser negativa")
	}
	return uc.Repo.ConferItem(ctx, itemID, conferredQty)
}

// Separate moves OPEN → SEPARATED, reserving the picked stock (soft allocation).
func (uc *ShipmentUseCase) Separate(ctx context.Context, code int64, actor uuid.UUID) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := guardTransition(ship, entity.ShipmentStatusSeparated); err != nil {
		return err
	}
	if len(ship.Items) == 0 {
		return errorsuc.NewValidationError(fmt.Sprintf("romaneio %d não possui itens para separar", code))
	}
	return uc.emTransacao(ctx, func(r RomaneioRepos) error {
		baixa := repository.BaixaPelaNFe{}
		if b, ok := r.Romaneios.(baixaPelaNFe); ok {
			var err error
			if baixa, err = b.BaixaPelaNFe(ctx, code); err != nil {
				return err
			}
		}
		mascaras, err := uc.mascarasDasLinhas(ctx, ship)
		if err != nil {
			return err
		}
		if err := uc.completarDepositos(ctx, ship); err != nil {
			return err
		}
		jaBaixado, err := reserveStock(ctx, r.Estoque, ship, actor, baixa, mascaras)
		if err != nil {
			return err
		}
		nota := "separação"
		if jaBaixado > 0 {
			nota = fmt.Sprintf("separação — %d linha(s) já baixada(s) pela NF-e autorizada, sem reserva de estoque", jaBaixado)
		}
		return r.Romaneios.UpdateStatus(ctx, code, entity.ShipmentStatusSeparated, &actor, nota)
	})
}

// baixaPelaNFe lê o que a NF-e já tirou do estoque (faturar antes de separar).
type baixaPelaNFe interface {
	BaixaPelaNFe(ctx context.Context, shipmentCode int64) (repository.BaixaPelaNFe, error)
}

// Confer moves SEPARATED → CONFERRED. Every line must already be conferred.
func (uc *ShipmentUseCase) Confer(ctx context.Context, code int64, actor uuid.UUID) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := guardTransition(ship, entity.ShipmentStatusConferred); err != nil {
		return err
	}
	for _, it := range ship.Items {
		if !it.IsConferred {
			return errorsuc.NewValidationError(fmt.Sprintf("item %d (seq %d) ainda não conferido", it.ItemCode, it.Sequence))
		}
	}
	return uc.Repo.UpdateStatus(ctx, code, entity.ShipmentStatusConferred, &actor, "conferência")
}

// Ship moves CONFERRED → SHIPPED. It blocks on quantity divergences unless they
// are explicitly accepted, and releases (consumes) the stock reservations — the
// fiscal NF-e remains responsible for the actual stock write-down.
func (uc *ShipmentUseCase) Ship(ctx context.Context, code int64, actor uuid.UUID, acceptDivergences bool) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := guardTransition(ship, entity.ShipmentStatusShipped); err != nil {
		return err
	}
	if err := ship.ValidateForShipping(acceptDivergences); err != nil {
		return err
	}
	if err := uc.conferirVolumes(ctx, ship); err != nil {
		return err
	}
	baixas, faltaEmbalagem, err := uc.baixasDeEmbalagem(ctx, ship, actor)
	if err != nil {
		return err
	}
	nota := "despacho"
	if faltaEmbalagem != "" {
		nota += "; " + faltaEmbalagem
	}
	return uc.emTransacao(ctx, func(r RomaneioRepos) error {
		if err := releaseReservations(ctx, r.Estoque, ship.Code, true); err != nil {
			return err
		}
		if len(baixas) > 0 {
			estoque, ok := r.Estoque.(BaixaDeEmbalagem)
			if !ok {
				return fmt.Errorf("baixa da embalagem indisponível")
			}
			for _, b := range baixas {
				if _, err := estoque.CreateMovement(ctx, b); err != nil {
					return fmt.Errorf("baixando a embalagem do despacho: %w", err)
				}
			}
		}
		return r.Romaneios.UpdateStatus(ctx, code, entity.ShipmentStatusShipped, &actor, nota)
	})
}

// Cancel cancels a non-shipped romaneio and releases its stock reservations.
func (uc *ShipmentUseCase) Cancel(ctx context.Context, code int64, actor uuid.UUID, reason string) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := guardTransition(ship, entity.ShipmentStatusCancelled); err != nil {
		return err
	}
	// Cancelar o romaneio deixava a NF-e dele viva e pronta para transmitir.
	if ship.FiscalExitID != nil {
		return errorsuc.NewValidationError(fmt.Sprintf(
			"o romaneio %d tem %s: descarte a nota (rascunho) ou cancele-a (autorizada) antes de cancelar o romaneio",
			code, ship.NotaDoRomaneio()))
	}
	return uc.emTransacao(ctx, func(r RomaneioRepos) error {
		if err := releaseReservations(ctx, r.Estoque, ship.Code, false); err != nil {
			return err
		}
		return r.Romaneios.UpdateStatus(ctx, code, entity.ShipmentStatusCancelled, &actor, reason)
	})
}

// UpdateTransport sets the trip/transport data (carrier, freight, vehicle,
// driver, seals, estimated delivery).
func (uc *ShipmentUseCase) UpdateTransport(ctx context.Context, code int64, t repository.TransportInput, actor uuid.UUID) (*response.ShipmentResponse, error) {
	if err := uc.Repo.UpdateTransport(ctx, code, t, &actor); err != nil {
		return nil, err
	}
	return uc.Get(ctx, code)
}

// LinkFiscalExit attaches the NF-e (saída fiscal) of the load to the romaneio.
func (uc *ShipmentUseCase) LinkFiscalExit(ctx context.Context, code int64, fiscalExitID, nfeNumber *int64, nfeKey *string, actor uuid.UUID) error {
	return uc.Repo.SetFiscalExit(ctx, code, fiscalExitID, nfeNumber, nfeKey, &actor)
}

type AddVolumeInput struct {
	ShipmentCode int64
	VolumeNumber int
	PackageType  string
	NetWeight    decimal.Decimal
	GrossWeight  decimal.Decimal
	LengthCm     decimal.Decimal
	WidthCm      decimal.Decimal
	HeightCm     decimal.Decimal
	Marking      *string
	Contents     *string
}

func (uc *ShipmentUseCase) AddVolume(ctx context.Context, in AddVolumeInput) (*response.ShipmentVolumeResponse, error) {
	ship, err := uc.Repo.GetByCode(ctx, in.ShipmentCode)
	if err != nil {
		return nil, err
	}
	if in.PackageType == "" {
		in.PackageType = "CAIXA"
	}
	v := &entity.ShipmentVolume{
		ShipmentID:   ship.ID,
		VolumeNumber: in.VolumeNumber,
		PackageType:  in.PackageType,
		NetWeight:    in.NetWeight,
		GrossWeight:  in.GrossWeight,
		LengthCm:     in.LengthCm,
		WidthCm:      in.WidthCm,
		HeightCm:     in.HeightCm,
		Marking:      in.Marking,
		Contents:     in.Contents,
	}
	v.CubageM3 = v.Cubage()
	created, err := uc.Repo.AddVolume(ctx, v)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.RecalcTotals(ctx, in.ShipmentCode); err != nil {
		return nil, fmt.Errorf("recalculando os totais do romaneio: %w", err)
	}
	return toShipmentVolumeResponse(created), nil
}

func (uc *ShipmentUseCase) ListVolumes(ctx context.Context, shipmentCode int64) ([]*response.ShipmentVolumeResponse, error) {
	ship, err := uc.Repo.GetByCode(ctx, shipmentCode)
	if err != nil {
		return nil, err
	}
	return toShipmentVolumeResponses(ship.Volumes), nil
}

func (uc *ShipmentUseCase) DeleteVolume(ctx context.Context, shipmentCode, volumeID int64) error {
	if err := uc.Repo.DeleteVolume(ctx, volumeID); err != nil {
		return err
	}
	return uc.Repo.RecalcTotals(ctx, shipmentCode)
}

func (uc *ShipmentUseCase) ListEvents(ctx context.Context, shipmentCode int64) ([]*response.ShipmentEventResponse, error) {
	ship, err := uc.Repo.GetByCode(ctx, shipmentCode)
	if err != nil {
		return nil, err
	}
	events, err := uc.Repo.ListEvents(ctx, ship.ID)
	if err != nil {
		return nil, err
	}
	return toShipmentEventResponses(events), nil
}

// ---- helpers ----

// baixasDeEmbalagem monta a saída do item de embalagem: uma por volume, somando os
// itens que usam a mesma embalagem e contando o que foi conferido. Falta de caixa no
// sistema não segura a carga que já está pronta: baixa o saldo que há e devolve a
// falta para ficar registrada no despacho.
func (uc *ShipmentUseCase) baixasDeEmbalagem(ctx context.Context, ship *entity.Shipment, actor uuid.UUID) ([]*stockentity.StockMovement, string, error) {
	if uc.Regras == nil {
		return nil, "", nil
	}
	itens := make([]int64, 0, len(ship.Items))
	for _, it := range ship.Items {
		itens = append(itens, it.ItemCode)
	}
	embalagens, err := uc.Regras.Embalagens(ctx, itens)
	if err != nil || len(embalagens) == 0 {
		return nil, "", err
	}
	porUnidade, err := uc.Regras.VolumesPorUnidade(ctx, itens)
	if err != nil {
		return nil, "", err
	}
	total := map[EmbalagemDoItem]decimal.Decimal{}
	for _, it := range ship.Items {
		e, ok := embalagens[it.ItemCode]
		if !ok {
			continue
		}
		fator := decimal.NewFromInt(1)
		if n := porUnidade[it.ItemCode]; n.IsPositive() {
			fator = n
		}
		quantidade := it.Quantity
		if it.IsConferred {
			quantidade = it.ConferredQty
		}
		total[e] = total[e].Add(quantidade.Mul(fator).Ceil())
	}
	referencia, codigo := "SHIPMENT", ship.Code
	baixas := make([]*stockentity.StockMovement, 0, len(total))
	faltas := []string{}
	for e, q := range total {
		saldo, err := uc.Regras.SaldoDaEmbalagem(ctx, e)
		if err != nil {
			return nil, "", err
		}
		if saldo.LessThan(q) {
			faltas = append(faltas, fmt.Sprintf("embalagem %d sem saldo: faltaram %s", e.ItemCode, q.Sub(decimal.Max(saldo, decimal.Zero)).String()))
			q = decimal.Max(saldo, decimal.Zero)
		}
		if !q.IsPositive() {
			continue
		}
		baixas = append(baixas, &stockentity.StockMovement{ItemCode: e.ItemCode, WarehouseID: e.WarehouseID, MovementType: stockentity.MovementTypeOut,
			Quantity: q, ReferenceType: &referencia, ReferenceCode: &codigo, CreatedBy: actor})
	}
	sort.Strings(faltas)
	return baixas, strings.Join(faltas, "; "), nil
}

// conferirVolumes recusa o despacho sem volumes quando algum item exige a montagem.
func (uc *ShipmentUseCase) conferirVolumes(ctx context.Context, ship *entity.Shipment) error {
	if uc.Regras == nil {
		return nil
	}
	volumes, err := uc.Repo.ListVolumes(ctx, ship.ID)
	if err != nil {
		return err
	}
	if len(volumes) > 0 {
		return nil
	}
	itens := make([]int64, 0, len(ship.Items))
	for _, it := range ship.Items {
		itens = append(itens, it.ItemCode)
	}
	exigem, err := uc.Regras.ItensQueExigemVolume(ctx, itens)
	if err != nil {
		return err
	}
	if len(exigem) > 0 {
		return errorsuc.NewValidationError(fmt.Sprintf("monte os volumes antes de despachar: o cadastro exige para %s", strings.Join(exigem, ", ")))
	}
	return nil
}

func guardTransition(ship *entity.Shipment, next entity.ShipmentStatus) error {
	if !ship.Status.CanTransitionTo(next) {
		return errorsuc.NewValidationError(fmt.Sprintf("transição inválida: romaneio %d está %s e não pode ir para %s",
			ship.Code, ship.Status, next))
	}
	return nil
}

// reserveStock reserva o que o romaneio vai tirar da prateleira. O que a NF-e
// autorizada já baixou não está mais no estoque: reservar de novo pedia um
// saldo que não existe, e o romaneio feito depois da nota não separava.
// Devolve quantas linhas ficaram (total ou parcialmente) sem reserva por isso.
// completarDepositos: linha do romaneio sem depósito separa no depósito padrão
// do item — a mesma regra da baixa pela NF-e. Sem isto a reserva ia para o
// depósito 0, onde nunca há saldo, e a separação sempre falhava.
func (uc *ShipmentUseCase) completarDepositos(ctx context.Context, ship *entity.Shipment) error {
	d, ok := uc.Regras.(interface {
		DepositosPadrao(ctx context.Context, itens []int64) (map[int64]int64, error)
	})
	if !ok || uc.Regras == nil {
		return nil
	}
	var itens []int64
	for _, it := range ship.Items {
		if it.WarehouseID == nil || *it.WarehouseID == 0 {
			itens = append(itens, it.ItemCode)
		}
	}
	if len(itens) == 0 {
		return nil
	}
	padrao, err := d.DepositosPadrao(ctx, itens)
	if err != nil {
		return err
	}
	for _, it := range ship.Items {
		if dep, ok := padrao[it.ItemCode]; ok && (it.WarehouseID == nil || *it.WarehouseID == 0) {
			it.WarehouseID = &dep
		}
	}
	return nil
}

// mascarasDasLinhas lê a máscara das linhas de pedido do romaneio (opcional).
func (uc *ShipmentUseCase) mascarasDasLinhas(ctx context.Context, ship *entity.Shipment) (map[int64]string, error) {
	m, ok := uc.Regras.(interface {
		MascarasDasLinhas(ctx context.Context, linhas []int64) (map[int64]string, error)
	})
	if !ok || uc.Regras == nil {
		return nil, nil
	}
	var linhas []int64
	for _, it := range ship.Items {
		if it.SalesOrderItemCode != nil {
			linhas = append(linhas, *it.SalesOrderItemCode)
		}
	}
	return m.MascarasDasLinhas(ctx, linhas)
}

func reserveStock(ctx context.Context, estoque StockReserver, ship *entity.Shipment, actor uuid.UUID, baixa repository.BaixaPelaNFe, mascaras ...map[int64]string) (int, error) {
	if estoque == nil {
		return 0, nil
	}
	faturado := make(map[int64]decimal.Decimal, len(baixa.PorLinhaDoPedido))
	for linha, qtd := range baixa.PorLinhaDoPedido {
		faturado[linha] = qtd
	}
	doPedido, err := reservasDoPedido(ctx, estoque, ship)
	if err != nil {
		return 0, err
	}
	jaBaixado := 0
	for _, it := range ship.Items {
		wh := int64(0)
		if it.WarehouseID != nil {
			wh = *it.WarehouseID
		}
		qty := it.Quantity
		if !qty.IsPositive() {
			continue
		}
		switch {
		case baixa.NotaVinculadaBaixou:
			qty = decimal.Zero
		case it.SalesOrderItemCode != nil:
			if f := faturado[*it.SalesOrderItemCode]; f.IsPositive() {
				coberto := decimal.Min(qty, f)
				faturado[*it.SalesOrderItemCode] = f.Sub(coberto)
				qty = qty.Sub(coberto)
			}
		}
		if !qty.Equal(it.Quantity) {
			jaBaixado++
		}
		if !qty.IsPositive() {
			continue
		}
		res := &stockentity.StockReservation{
			ItemCode:          it.ItemCode,
			WarehouseID:       wh,
			Quantity:          qty,
			ReferenceType:     reservationRefShipment,
			ReferenceCode:     ship.Code,
			ReferenceItemCode: it.SalesOrderItemCode,
			ReservationDate:   time.Now(),
			Status:            "ACTIVE",
			CreatedBy:         actor,
		}
		// Linha do pedido: a reserva que o pedido já fez vale para a separação;
		// só o que faltar entra como reserva do PEDIDO (a NF-e a consome, e o
		// romaneio cancelado não tira do pedido o que ele ainda vai expedir).
		// Antes reservava tudo de novo sobre o saldo já reservado pelo pedido.
		if it.SalesOrderItemCode != nil && ship.SalesOrderCode != nil {
			r := doPedido[*it.SalesOrderItemCode]
			coberto := decimal.Min(qty, r.qtd)
			r.qtd = r.qtd.Sub(coberto)
			doPedido[*it.SalesOrderItemCode] = r
			if res.Quantity = qty.Sub(coberto); !res.Quantity.IsPositive() {
				continue
			}
			res.ReferenceType, res.ReferenceCode, res.Mask = stockentity.ReferenceTypeSalesOrder, *ship.SalesOrderCode, r.mask
			// Sem reserva anterior do pedido, a máscara vem da linha do pedido.
			if res.Mask == "" && len(mascaras) > 0 {
				res.Mask = mascaras[0][*it.SalesOrderItemCode]
			}
			if it.WarehouseID == nil && r.almox != 0 {
				res.WarehouseID = r.almox
			}
		}
		if _, err := estoque.CreateReservation(ctx, res); err != nil {
			// Depósito que não aceita reservas: o romaneio segue sem reservar.
			if errors.Is(err, stockrepository.ErrAlmoxSemReserva) {
				continue
			}
			// O erro tipado chega ao usuário só pela mensagem dele: sem isto a
			// tela dizia "estoque insuficiente" sem dizer de qual item nem quanto.
			if errors.Is(err, stockrepository.ErrInsufficientStock) {
				// A linha, não o código interno do item: a tela mostra a linha com o
				// código comercial, que quase nunca é igual ao interno.
				return 0, errorsuc.NewValidationError(fmt.Sprintf("a linha %d do romaneio %d não tem saldo livre para separar %s: reduza a quantidade dela (entrega parcial) ou aguarde a entrada do material",
					it.Sequence, ship.Code, res.Quantity.String()))
			}
			return 0, fmt.Errorf("reservando o item %d do romaneio: %w", it.ItemCode, err)
		}
	}
	return jaBaixado, nil
}

type reservaDaLinha struct {
	qtd   decimal.Decimal
	mask  string
	almox int64
}

// reservasDoPedido soma, por linha do pedido, o que o pedido do romaneio tem
// reservado e ainda ativo.
func reservasDoPedido(ctx context.Context, estoque StockReserver, ship *entity.Shipment) (map[int64]reservaDaLinha, error) {
	out := map[int64]reservaDaLinha{}
	if ship.SalesOrderCode == nil {
		return out, nil
	}
	ativas, err := estoque.ListActiveReservations(ctx)
	if err != nil {
		return nil, fmt.Errorf("lendo as reservas do pedido %d: %w", *ship.SalesOrderCode, err)
	}
	for _, r := range ativas {
		if r.ReferenceType != stockentity.ReferenceTypeSalesOrder || r.ReferenceCode != *ship.SalesOrderCode || r.ReferenceItemCode == nil {
			continue
		}
		l := out[*r.ReferenceItemCode]
		l.qtd, l.mask, l.almox = l.qtd.Add(r.Quantity), r.Mask, r.WarehouseID
		out[*r.ReferenceItemCode] = l
	}
	return out, nil
}

// releaseReservations baixa (no despacho) ou cancela (no cancelamento) as
// reservas ativas do romaneio. Falha desfaz a transição — antes era ignorada e
// o romaneio despachava com o estoque ainda reservado.
func releaseReservations(ctx context.Context, estoque StockReserver, shipmentCode int64, consume bool) error {
	if estoque == nil {
		return nil
	}
	active, err := estoque.ListActiveReservations(ctx)
	if err != nil {
		return fmt.Errorf("lendo as reservas do romaneio: %w", err)
	}
	for _, res := range active {
		if res.ReferenceType != reservationRefShipment || res.ReferenceCode != shipmentCode {
			continue
		}
		if consume {
			if err := estoque.ConsumeReservation(ctx, res.ID); err != nil {
				return fmt.Errorf("baixando a reserva %d: %w", res.ID, err)
			}
		} else if err := estoque.CancelReservation(ctx, res.ID); err != nil {
			return fmt.Errorf("cancelando a reserva %d: %w", res.ID, err)
		}
	}
	return nil
}
