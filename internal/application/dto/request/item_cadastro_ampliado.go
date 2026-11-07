package request

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/shopspring/decimal"
)

// Opcional distingue o campo ausente (preserva o gravado) do null (limpa). Um
// `**T` não faz isso: o encoding/json zera o ponteiro de fora no null, e o caso
// de uso não enxerga a diferença — o valor antigo continuava gravado.
type Opcional[T any] struct {
	Set   bool
	Value *T
}

func (o *Opcional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// Aplicar copia para o destino quando o campo veio na requisição.
func (o Opcional[T]) Aplicar(destino **T) {
	if o.Set {
		*destino = o.Value
	}
}

// AplicarValor copia para um campo obrigatório; null não limpa o que não pode
// ficar vazio.
func (o Opcional[T]) AplicarValor(destino *T) {
	if o.Set && o.Value != nil {
		*destino = *o.Value
	}
}

func aplicarTexto(o Opcional[string], destino **string) {
	if !o.Set {
		return
	}
	*destino = nil
	if o.Value != nil {
		if s := strings.TrimSpace(*o.Value); s != "" {
			*destino = &s
		}
	}
}

// Os blocos abaixo entram embutidos nos DTOs de criar e de alterar, e a mesma
// função grava nos dois: campo novo não fica para trás em um dos caminhos.

type IdentificacaoAmpliadaDTO struct {
	LongDescription        Opcional[string] `json:"long_description"`
	GeneralNotes           Opcional[string] `json:"general_notes"`
	Revision               Opcional[string] `json:"revision"`
	PreviousCode           Opcional[string] `json:"previous_code"`
	ManufacturerName       Opcional[string] `json:"manufacturer_name"`
	ManufacturerPartNumber Opcional[string] `json:"manufacturer_part_number"`
	IsScrap                Opcional[bool]   `json:"is_scrap"`
	DescriptionEN          Opcional[string] `json:"description_en"`
}

func (d IdentificacaoAmpliadaDTO) AplicarEm(i *entity.Item) {
	aplicarTexto(d.LongDescription, &i.LongDescription)
	aplicarTexto(d.GeneralNotes, &i.GeneralNotes)
	aplicarTexto(d.Revision, &i.Revision)
	aplicarTexto(d.PreviousCode, &i.PreviousCode)
	aplicarTexto(d.ManufacturerName, &i.ManufacturerName)
	aplicarTexto(d.ManufacturerPartNumber, &i.ManufacturerPartNumber)
	d.IsScrap.AplicarValor(&i.IsScrap)
	aplicarTexto(d.DescriptionEN, &i.DescriptionEN)
}

type EstoqueAmpliadoDTO struct {
	ShelfLifeDays             Opcional[int]             `json:"shelf_life_days"`
	MinRemainingLifeDays      Opcional[int]             `json:"min_remaining_life_days"`
	StorageConditions         Opcional[string]          `json:"storage_conditions"`
	Hazardous                 Opcional[bool]            `json:"hazardous"`
	UNNumber                  Opcional[string]          `json:"un_number"`
	HazardClass               Opcional[string]          `json:"hazard_class"`
	AcceptsFractionalQuantity Opcional[bool]            `json:"accepts_fractional_quantity"`
	SecondaryUOM              Opcional[string]          `json:"secondary_uom"`
	SecondaryFactor           Opcional[decimal.Decimal] `json:"secondary_factor"`
}

func (d EstoqueAmpliadoDTO) AplicarEm(w *entity.Warehouse) {
	d.ShelfLifeDays.Aplicar(&w.ShelfLifeDays)
	d.MinRemainingLifeDays.Aplicar(&w.MinRemainingLifeDays)
	aplicarTexto(d.StorageConditions, &w.StorageConditions)
	d.Hazardous.AplicarValor(&w.Hazardous)
	aplicarTexto(d.UNNumber, &w.UNNumber)
	aplicarTexto(d.HazardClass, &w.HazardClass)
	d.AcceptsFractionalQuantity.AplicarValor(&w.AcceptsFractionalQuantity)
	aplicarTexto(d.SecondaryUOM, &w.SecondaryUOM)
	d.SecondaryFactor.Aplicar(&w.SecondaryFactor)
}

type EngenhariaAmpliadaDTO struct {
	Volume            Opcional[decimal.Decimal] `json:"volume"`
	RequiresDrawing   Opcional[bool]            `json:"requires_drawing"`
	CuttingGrain      Opcional[string]          `json:"cutting_grain"`
	CuttingNoRotation Opcional[bool]            `json:"cutting_no_rotation"`
}

func (d EngenhariaAmpliadaDTO) AplicarEm(e *entity.Engineering) {
	d.Volume.Aplicar(&e.Volume)
	d.RequiresDrawing.AplicarValor(&e.RequiresDrawing)
	d.CuttingGrain.AplicarValor(&e.CuttingGrain)
	d.CuttingNoRotation.AplicarValor(&e.CuttingNoRotation)
}

// ComercialAmpliadoDTO: cliente dono do item e volumes por unidade.
type ComercialAmpliadoDTO struct {
	ExclusiveCustomerCode Opcional[int64] `json:"exclusive_customer_code"`
	VolumesPerUnit        Opcional[int]   `json:"volumes_per_unit"`
}

func (d ComercialAmpliadoDTO) AplicarEm(c *entity.Commercial) {
	d.ExclusiveCustomerCode.Aplicar(&c.ExclusiveCustomerCode)
	d.VolumesPerUnit.Aplicar(&c.VolumesPerUnit)
}

type PlanejamentoAmpliadoDTO struct {
	PlannerEmployeeCode       Opcional[int64]           `json:"planner_employee_code"`
	BuyerEmployeeCode         Opcional[int64]           `json:"buyer_employee_code"`
	PurchaseLeadTimeDays      Opcional[int]             `json:"purchase_lead_time_days"`
	ManufacturingLeadTimeDays Opcional[int]             `json:"manufacturing_lead_time_days"`
	Notes                     Opcional[string]          `json:"notes"`
	MaterialIssueTiming       Opcional[string]          `json:"material_issue_timing"`
	ProductionReportingType   Opcional[string]          `json:"production_reporting_type"`
	MaximumStock              Opcional[decimal.Decimal] `json:"maximum_stock"`
	Coverage                  Opcional[int]             `json:"coverage"`
	GroupingKey               Opcional[string]          `json:"grouping_key"`
	SafetyTimeDays            Opcional[int]             `json:"safety_time_days"`
	MaximumLot                Opcional[decimal.Decimal] `json:"maximum_lot"`
}

func (d PlanejamentoAmpliadoDTO) AplicarEm(p *entity.Planning) {
	d.PlannerEmployeeCode.Aplicar(&p.PlannerEmployeeCode)
	d.BuyerEmployeeCode.Aplicar(&p.BuyerEmployeeCode)
	d.PurchaseLeadTimeDays.Aplicar(&p.PurchaseLeadTimeDays)
	d.ManufacturingLeadTimeDays.Aplicar(&p.ManufacturingLeadTimeDays)
	aplicarTexto(d.Notes, &p.Notes)
	d.MaterialIssueTiming.AplicarValor(&p.MaterialIssueTiming)
	d.ProductionReportingType.AplicarValor(&p.ProductionReportingType)
	if d.MaximumStock.Set {
		p.Extras.MaximumStock = decimal.Zero
		d.MaximumStock.AplicarValor(&p.Extras.MaximumStock)
	}
	if d.Coverage.Set {
		p.Extras.Coverage = 0
		d.Coverage.AplicarValor(&p.Extras.Coverage)
	}
	aplicarTexto(d.GroupingKey, &p.Extras.GroupingKey)
	d.SafetyTimeDays.Aplicar(&p.Extras.SafetyTimeDays)
	d.MaximumLot.Aplicar(&p.Extras.MaximumLot)
}

type SuprimentosAmpliadoDTO struct {
	UnderDeliveryTolerancePct Opcional[decimal.Decimal] `json:"under_delivery_tolerance_pct"`
	OverDeliveryTolerancePct  Opcional[decimal.Decimal] `json:"over_delivery_tolerance_pct"`
	ReceiptRouting            Opcional[string]          `json:"receipt_routing"`
	ClassificationCode        Opcional[string]          `json:"classification_code"`
	RequiresCertificate       Opcional[bool]            `json:"requires_certificate"`
}

func (d SuprimentosAmpliadoDTO) AplicarEm(s *entity.Supplies) {
	d.UnderDeliveryTolerancePct.Aplicar(&s.UnderDeliveryTolerancePct)
	d.OverDeliveryTolerancePct.Aplicar(&s.OverDeliveryTolerancePct)
	d.ReceiptRouting.AplicarValor(&s.ReceiptRouting)
	aplicarTexto(d.ClassificationCode, &s.ClassificationCode)
	d.RequiresCertificate.AplicarValor(&s.RequiresCertificate)
}

type FiscalAmpliadoDTO struct {
	GTIN             Opcional[string]          `json:"gtin"`
	GTINTax          Opcional[string]          `json:"gtin_tax"`
	EFDItemType      Opcional[string]          `json:"efd_item_type"`
	TaxUnit          Opcional[string]          `json:"tax_unit"`
	TaxUnitFactor    Opcional[decimal.Decimal] `json:"tax_unit_factor"`
	ExTIPI           Opcional[string]          `json:"ex_tipi"`
	RelevantScale    Opcional[bool]            `json:"relevant_scale"`
	ManufacturerCNPJ Opcional[string]          `json:"manufacturer_cnpj"`
	FCINumber        Opcional[string]          `json:"fci_number"`
}

func (d FiscalAmpliadoDTO) AplicarEm(a *entity.Accounting) {
	aplicarTexto(d.GTIN, &a.GTIN)
	aplicarTexto(d.GTINTax, &a.GTINTax)
	aplicarTexto(d.EFDItemType, &a.EFDItemType)
	aplicarTexto(d.TaxUnit, &a.TaxUnit)
	if a.TaxUnit != nil {
		u := strings.ToUpper(*a.TaxUnit)
		a.TaxUnit = &u
	}
	d.TaxUnitFactor.Aplicar(&a.TaxUnitFactor)
	aplicarTexto(d.ExTIPI, &a.ExTIPI)
	d.RelevantScale.Aplicar(&a.RelevantScale)
	aplicarTexto(d.ManufacturerCNPJ, &a.ManufacturerCNPJ)
	aplicarTexto(d.FCINumber, &a.FCINumber)
}
