package shipment_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	romaneioExport "github.com/FelipePn10/panossoerp/internal/infrastructure/export/romaneio"
)

type ShipmentExportUseCase struct {
	ShipmentRepo repository.ShipmentRepository
	Enricher     RomaneioEnricher
}

type RomaneioEnricher interface {
	GetEnterprise(ctx context.Context) (romaneioExport.CompanyInfo, error)
	GetBranding(ctx context.Context) (logo []byte, brandColorHex string, err error)
	GetCustomer(ctx context.Context, customerCode int64) (romaneioExport.CompanyInfo, error)
	GetSupplier(ctx context.Context, supplierCode int64) (romaneioExport.CompanyInfo, error)
	GetCarrier(ctx context.Context, carrierCode int64) (romaneioExport.CarrierInfo, error)
	GetItemDetails(ctx context.Context, itemCode int64) (romaneioExport.RomaneioItem, error)
	GetSalesOrder(ctx context.Context, code int64) (*SalesOrderHeader, error)
	GetPurchaseOrder(ctx context.Context, code int64) (*PurchaseOrderHeader, error)
	GetProductionOrder(ctx context.Context, code int64) (*ProductionOrderHeader, error)
}

func (uc *ShipmentExportUseCase) GeneratePDF(ctx context.Context, shipmentCode int64) ([]byte, error) {
	data, err := uc.buildRomaneioData(ctx, shipmentCode)
	if err != nil {
		return nil, err
	}
	return romaneioExport.GenerateRomaneioPDF(data)
}

func (uc *ShipmentExportUseCase) GenerateXLSX(ctx context.Context, shipmentCode int64) ([]byte, error) {
	data, err := uc.buildRomaneioData(ctx, shipmentCode)
	if err != nil {
		return nil, err
	}
	return romaneioExport.GenerateRomaneioXLSX(data)
}

func (uc *ShipmentExportUseCase) buildRomaneioData(ctx context.Context, shipmentCode int64) (*romaneioExport.RomaneioData, error) {
	s, err := uc.ShipmentRepo.GetByCode(ctx, shipmentCode)
	if err != nil {
		return nil, err
	}

	d := &romaneioExport.RomaneioData{
		Title:        "ROMANEIO DE EXPEDICAO",
		Code:         s.Code,
		Date:         s.CreatedAt,
		Status:       string(s.Status),
		Notes:        strOrEmpty(s.Notes),
		TotalVolumes: s.TotalVolumes,
		TotalWeight:  s.TotalGrossWeight,
		GeneratedAt:  time.Now(),
		Seals:        strOrEmpty(s.Seals),
		Volumes:      mapRomaneioVolumes(s.Volumes),
	}
	if s.NFeNumber != nil {
		d.NFeNumber = *s.NFeNumber
	}
	d.NFeKey = strOrEmpty(s.NFeKey)

	if s.ReferenceType != nil {
		d.ReferenceType = string(*s.ReferenceType)
		switch *s.ReferenceType {
		case entity.ShipmentRefSalesOrder:
			if s.SalesOrderCode != nil {
				d.ReferenceCode = *s.SalesOrderCode
			}
		case entity.ShipmentRefPurchaseOrder:
			if s.PurchaseOrderCode != nil {
				d.ReferenceCode = *s.PurchaseOrderCode
			}
		case entity.ShipmentRefProductionOrder:
			if s.ProductionOrderCode != nil {
				d.ReferenceCode = *s.ProductionOrderCode
			}
		}
	}

	if uc.Enricher != nil {
		ent, err := uc.Enricher.GetEnterprise(ctx)
		if err == nil {
			d.Enterprise = ent
		}

		if logo, color, err := uc.Enricher.GetBranding(ctx); err == nil {
			d.Logo = logo
			d.BrandColorHex = color
		}

		if s.CarrierCode != nil {
			carrier, err := uc.Enricher.GetCarrier(ctx, *s.CarrierCode)
			if err == nil {
				d.Carrier = carrier
			}
		}

		d.TransportInfo.VolumeQuantity = float64(s.TotalVolumes)
		d.TransportInfo.NetWeight = s.TotalNetWeight
		d.TransportInfo.GrossWeight = s.TotalGrossWeight
		d.TransportInfo.VolumeType = "UN"

		for _, it := range s.Items {
			rit, err := uc.Enricher.GetItemDetails(ctx, it.ItemCode)
			if err != nil {
				rit = romaneioExport.RomaneioItem{
					Sequence:    it.Sequence,
					ItemCode:    it.ItemCode,
					Description: fmt.Sprintf("Item %d", it.ItemCode),
					Quantity:    it.Quantity,
					Unit:        "UN",
				}
			} else {
				rit.Sequence = it.Sequence
				rit.Quantity = it.Quantity
			}
			d.Items = append(d.Items, rit)
		}

		if s.ReferenceType != nil {
			switch *s.ReferenceType {
			case entity.ShipmentRefSalesOrder:
				if s.SalesOrderCode != nil {
					so, err := uc.Enricher.GetSalesOrder(ctx, *s.SalesOrderCode)
					if err == nil {
						if so.CustomerCode != nil {
							if cust, err := uc.Enricher.GetCustomer(ctx, *so.CustomerCode); err == nil {
								d.Destinatario = cust
							}
						}
						d.TotalGross = so.TotalGross
						d.TotalNet = so.TotalNet
						for i := range d.Items {
							if i < len(so.Items) {
								si := so.Items[i]
								d.Items[i].UnitPrice = si.UnitPrice
								d.Items[i].TotalPrice = si.TotalGross
								d.Items[i].ICMSPct = si.ICMSPct
								d.Items[i].IPIPct = si.IPIPct
								d.Items[i].PISPct = si.PISPct
								d.Items[i].COFINSPct = si.COFINSPct
								d.Items[i].STPct = si.STPct
								d.Items[i].WeightNet = si.UnitWeightNet * si.RequestedQty
								d.Items[i].WeightGross = si.UnitWeightGross * si.RequestedQty
							}
						}
					}
				}
			case entity.ShipmentRefPurchaseOrder:
				if s.PurchaseOrderCode != nil {
					po, err := uc.Enricher.GetPurchaseOrder(ctx, *s.PurchaseOrderCode)
					if err == nil {
						if po.SupplierCode != nil {
							if sup, err := uc.Enricher.GetSupplier(ctx, *po.SupplierCode); err == nil {
								d.Destinatario = sup
							}
						}
						for i := range d.Items {
							if i < len(po.Items) {
								pi := po.Items[i]
								d.Items[i].UnitPrice = pi.UnitPrice
								d.Items[i].TotalPrice = pi.TotalPrice
								d.Items[i].ICMSPct = pi.ICMSPct
								d.Items[i].IPIPct = pi.IPIPct
								d.Items[i].STPct = pi.ICMSSTPct
							}
						}
					}
				}
			case entity.ShipmentRefProductionOrder:
				if s.ProductionOrderCode != nil {
					po, err := uc.Enricher.GetProductionOrder(ctx, *s.ProductionOrderCode)
					if err == nil {
						d.ReferenceCode = po.Code
					}
				}
			}
		}
	}

	// Persisted trip/transport data is authoritative over master-data defaults.
	applyShipmentTransport(d, s)

	if d.Enterprise.Name == "" {
		d.Enterprise = romaneioExport.CompanyInfo{
			Name:    "Empresa",
			CNPJCPF: "00.000.000/0000-00",
		}
	}

	d.Subtitle = fmt.Sprintf("Romaneio No %d", d.Code)

	return d, nil
}

// applyShipmentTransport overlays the trip data persisted on the shipment (plate,
// driver, ANTT, freight modality/value, insurance, estimated delivery) onto the
// romaneio document.
func applyShipmentTransport(d *romaneioExport.RomaneioData, s *entity.Shipment) {
	if s.VehiclePlate != nil {
		d.Carrier.Plate = *s.VehiclePlate
	}
	if s.DriverName != nil {
		d.Carrier.Driver = *s.DriverName
	}
	if s.ANTTCode != nil {
		d.Carrier.ANTT = *s.ANTTCode
	}
	if s.FreightModality != nil {
		d.Carrier.FreightType = *s.FreightModality
		d.TransportInfo.FreightType = *s.FreightModality
	}
	if s.FreightValue > 0 {
		d.TransportInfo.FreightValue = s.FreightValue
	}
	if s.InsuranceValue > 0 {
		d.TransportInfo.InsuranceValue = s.InsuranceValue
	}
	if s.EstimatedDelivery != nil {
		d.TransportInfo.EstimatedDelivery = s.EstimatedDelivery.Format("02/01/2006")
	}
}

func mapRomaneioVolumes(vols []*entity.ShipmentVolume) []romaneioExport.RomaneioVolume {
	out := make([]romaneioExport.RomaneioVolume, 0, len(vols))
	for _, v := range vols {
		out = append(out, romaneioExport.RomaneioVolume{
			Number:      v.VolumeNumber,
			PackageType: v.PackageType,
			NetWeight:   v.NetWeight,
			GrossWeight: v.GrossWeight,
			LengthCm:    v.LengthCm,
			WidthCm:     v.WidthCm,
			HeightCm:    v.HeightCm,
			CubageM3:    v.CubageM3,
			Marking:     strOrEmpty(v.Marking),
			Contents:    strOrEmpty(v.Contents),
		})
	}
	return out
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
