package fiscal_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	salesentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	salesrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	shipentity "github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	shiprepo "github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/google/uuid"
)

type loadFiscalAuth struct {
	ports.AuthService
	uid uuid.UUID
}

func (a loadFiscalAuth) CanCreateFiscalExit(context.Context) bool { return true }
func (a loadFiscalAuth) UserID(context.Context) (uuid.UUID, error) {
	return a.uid, nil
}

type loadFiscalRepo struct {
	fiscalrepo.FiscalRepository
	nextNF  int64
	created *fiscalentity.FiscalExit
	items   []*fiscalentity.FiscalExitItem
}

func (r *loadFiscalRepo) GetNextNFNumber(context.Context) (int64, error) { return r.nextNF, nil }
func (r *loadFiscalRepo) GetFiscalConfig(context.Context) (*fiscalentity.FiscalConfig, error) {
	return &fiscalentity.FiscalConfig{UFEmpresa: "SP", IcmsInternoAliquota: decimal.NewFromFloat(0.18)}, nil
}
func (r *loadFiscalRepo) NFNumberExists(context.Context, int64, string) (bool, error) {
	return false, nil
}
func (r *loadFiscalRepo) ListNcmTaxes(context.Context) ([]*fiscalentity.NcmTaxTable, error) {
	return nil, nil
}
func (r *loadFiscalRepo) ListICMSInterstate(context.Context) (map[string]decimal.Decimal, error) {
	return map[string]decimal.Decimal{}, nil
}
func (r *loadFiscalRepo) ListICMSInternal(context.Context) (map[string]struct{ ICMS, FCP decimal.Decimal }, error) {
	return map[string]struct{ ICMS, FCP decimal.Decimal }{}, nil
}
func (r *loadFiscalRepo) CreateExit(_ context.Context, e *fiscalentity.FiscalExit) (*fiscalentity.FiscalExit, error) {
	e.ID = 77
	e.IsActive = true
	e.CreatedAt = time.Now()
	e.UpdatedAt = e.CreatedAt
	r.created = e
	return e, nil
}
func (r *loadFiscalRepo) CreateExitItem(_ context.Context, it *fiscalentity.FiscalExitItem) (*fiscalentity.FiscalExitItem, error) {
	it.ID = int64(len(r.items) + 1)
	it.CreatedAt = time.Now()
	r.items = append(r.items, it)
	return it, nil
}
func (r *loadFiscalRepo) GetExitItems(context.Context, int64) ([]*fiscalentity.FiscalExitItem, error) {
	return r.items, nil
}

type loadShipmentRepo struct {
	shiprepo.ShipmentRepository
	load             *shipentity.ShipmentLoad
	shipments        map[int64]*shipentity.Shipment
	linkedFiscalExit []int64
	loadNotes        []shiprepo.AddFiscalNoteToLoadInput
}

func (r *loadShipmentRepo) GetLoadByCode(context.Context, int64) (*shipentity.ShipmentLoad, error) {
	return r.load, nil
}
func (r *loadShipmentRepo) GetByCode(_ context.Context, code int64) (*shipentity.Shipment, error) {
	return r.shipments[code], nil
}
func (r *loadShipmentRepo) SetFiscalExit(_ context.Context, code int64, fiscalExitID, nfeNumber *int64, nfeKey *string, by *uuid.UUID) error {
	r.linkedFiscalExit = append(r.linkedFiscalExit, code)
	return nil
}
func (r *loadShipmentRepo) AddFiscalNoteToLoad(_ context.Context, in shiprepo.AddFiscalNoteToLoadInput) (*shipentity.ShipmentLoadFiscalNote, error) {
	r.loadNotes = append(r.loadNotes, in)
	return &shipentity.ShipmentLoadFiscalNote{FiscalExitID: in.FiscalExitID, LoadCode: in.LoadCode}, nil
}
func (r *loadShipmentRepo) RecalcLoadTotals(context.Context, int64) error { return nil }

type loadSalesRepo struct {
	salesrepo.SalesOrderRepository
	items []*salesentity.SalesOrderItem
}

func (r loadSalesRepo) ListItems(context.Context, int64) ([]*salesentity.SalesOrderItem, error) {
	return r.items, nil
}

func TestCreateFiscalExitFromLoad_UsesSalesOrderPricesAndLinksLoad(t *testing.T) {
	fiscal := &loadFiscalRepo{nextNF: 123}
	shipment := &loadShipmentRepo{
		load: &shipentity.ShipmentLoad{
			Code:   9001,
			Status: shipentity.LoadStatusReleased,
			Shipments: []*shipentity.ShipmentLoadShipment{
				{ShipmentCode: 5001},
			},
		},
		shipments: map[int64]*shipentity.Shipment{
			5001: {
				ID: 1, Code: 5001, Status: shipentity.ShipmentStatusConferred, SalesOrderCode: i64p(7001),
				Items: []*shipentity.ShipmentItem{
					{ItemCode: 1001, Quantity: 2, IsConferred: true, ConferredQty: 2},
				},
			},
		},
	}
	sales := loadSalesRepo{items: []*salesentity.SalesOrderItem{
		{Code: 1, ItemCode: 1001, UnitPrice: decimal.NewFromInt(50)},
	}}
	uc := &CreateFiscalExitFromLoadUseCase{
		CreateUC:       &CreateFiscalExitUseCase{Repo: fiscal, Auth: loadFiscalAuth{uid: uuid.New()}},
		FiscalRepo:     fiscal,
		ShipmentRepo:   shipment,
		SalesOrderRepo: sales,
	}

	resp, err := uc.Execute(context.Background(), request.CreateFiscalExitFromLoadDTO{
		LoadCode:         9001,
		Serie:            "1",
		DataEmissao:      "2026-07-06",
		Cfop:             "5102",
		NaturezaOperacao: "Venda de mercadoria",
		UFDestinatario:   strp("SP"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NumeroNF != 0 || fiscal.created.NumeroNF != 123 || resp.ValorProdutos != 100 || resp.ShipmentLoadCode == nil || *resp.ShipmentLoadCode != 9001 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if fiscal.created == nil || fiscal.created.SourceType == nil || *fiscal.created.SourceType != "LOAD" {
		t.Fatalf("source metadata not persisted: %+v", fiscal.created)
	}
	if len(fiscal.items) != 1 || !fiscal.items[0].UnitPrice.Equal(decimal.NewFromFloat(50)) || !fiscal.items[0].TotalPrice.Equal(decimal.NewFromFloat(100)) {
		t.Fatalf("items not generated from order price: %+v", fiscal.items)
	}
	if len(shipment.linkedFiscalExit) != 1 || shipment.linkedFiscalExit[0] != 5001 {
		t.Fatalf("shipment fiscal link not written: %v", shipment.linkedFiscalExit)
	}
	if len(shipment.loadNotes) != 1 || shipment.loadNotes[0].FiscalExitID != 77 {
		t.Fatalf("load fiscal note not written: %+v", shipment.loadNotes)
	}
}

func TestCreateFiscalExitFromLoad_MissingPriceRequiresOverride(t *testing.T) {
	fiscal := &loadFiscalRepo{nextNF: 123}
	shipment := &loadShipmentRepo{
		load: &shipentity.ShipmentLoad{
			Code:      9001,
			Status:    shipentity.LoadStatusReleased,
			Shipments: []*shipentity.ShipmentLoadShipment{{ShipmentCode: 5001}},
		},
		shipments: map[int64]*shipentity.Shipment{
			5001: {ID: 1, Code: 5001, Status: shipentity.ShipmentStatusConferred, Items: []*shipentity.ShipmentItem{{ItemCode: 1001, Quantity: 2}}},
		},
	}
	uc := &CreateFiscalExitFromLoadUseCase{
		CreateUC:     &CreateFiscalExitUseCase{Repo: fiscal, Auth: loadFiscalAuth{uid: uuid.New()}},
		FiscalRepo:   fiscal,
		ShipmentRepo: shipment,
	}

	_, err := uc.Execute(context.Background(), request.CreateFiscalExitFromLoadDTO{
		LoadCode:         9001,
		Serie:            "1",
		DataEmissao:      "2026-07-06",
		Cfop:             "5102",
		NaturezaOperacao: "Venda de mercadoria",
		UFDestinatario:   strp("SP"),
	})
	if err == nil || !strings.Contains(err.Error(), "sem preço") {
		t.Fatalf("expected missing price error, got %v", err)
	}
}

type cadastroDaCargaFake map[int64]fiscalrepo.ItemCandidate

func (c cadastroDaCargaFake) ItemsByCode(context.Context, []int64) (map[int64]fiscalrepo.ItemCandidate, error) {
	return c, nil
}

func TestCompletarDoCadastro_LeNcmDescricaoEUnidadeDoItem(t *testing.T) {
	code := int64(1001)
	desc := "Item 1001 - romaneio 1"
	items := []request.CreateFiscalExitItemDTO{{ItemCode: &code, Cfop: "5101", Description: &desc, OrigemMercadoria: "0"}}
	uc := &CreateFiscalExitFromLoadUseCase{Itens: cadastroDaCargaFake{
		1001: {Code: 1001, BusinessCode: "CH-10", Name: "Chapa de aço", NCM: "7208.51.00", UOM: "KG", Origem: "1"},
	}}
	if err := uc.completarDoCadastro(context.Background(), items, request.CreateFiscalExitFromLoadDTO{Cfop: "5101"}, nil); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	it := items[0]
	if *it.Description != "Chapa de aço" || *it.Ncm != "7208.51.00" || *it.UnidadeComercial != "KG" || *it.CodigoProduto != "CH-10" || it.OrigemMercadoria != "1" {
		t.Fatalf("o cadastro não chegou à linha da nota: %+v", it)
	}
}

func TestCompletarDoCadastro_ItemSemNcmParaAntesDeCriar(t *testing.T) {
	code := int64(1001)
	items := []request.CreateFiscalExitItemDTO{{ItemCode: &code, Cfop: "5101"}}
	uc := &CreateFiscalExitFromLoadUseCase{Itens: cadastroDaCargaFake{1001: {Code: 1001, Name: "Chapa"}}}
	err := uc.completarDoCadastro(context.Background(), items, request.CreateFiscalExitFromLoadDTO{Cfop: "5101"}, nil)
	if err == nil || !strings.Contains(err.Error(), "sem NCM") {
		t.Fatalf("esperava recusa por falta de NCM, veio %v", err)
	}
}

func TestCompletarDoCadastro_OverrideVenceOCadastro(t *testing.T) {
	code := int64(1001)
	ncm, desc := "99999999", "Texto escolhido"
	items := []request.CreateFiscalExitItemDTO{{ItemCode: &code, Cfop: "5101", Ncm: &ncm, Description: &desc}}
	ov := map[fiscalLoadOverrideKey]request.FiscalExitLoadItemOverride{{itemCode: 1001}: {Ncm: &ncm, Description: &desc}}
	uc := &CreateFiscalExitFromLoadUseCase{Itens: cadastroDaCargaFake{1001: {Code: 1001, Name: "Chapa", NCM: "72085100"}}}
	if err := uc.completarDoCadastro(context.Background(), items, request.CreateFiscalExitFromLoadDTO{Cfop: "5101"}, ov); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if *items[0].Ncm != "99999999" || *items[0].Description != "Texto escolhido" {
		t.Fatalf("o override foi ignorado: %+v", items[0])
	}
}
