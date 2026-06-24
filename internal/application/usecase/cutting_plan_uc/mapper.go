package cutting_plan_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service"
)

func toPlanResponse(p *entity.CuttingPlan) *response.CuttingPlanResponse {
	return &response.CuttingPlanResponse{
		ID:               p.ID,
		Code:             p.Code,
		Description:      p.Description,
		CutType:          string(p.CutType),
		Source:           string(p.Source),
		Status:           string(p.Status),
		MaterialItemCode: p.MaterialItemCode,
		MachineCode:      p.MachineCode,
		StockUoM:         string(p.StockUoM),
		UoMFactor:        p.UoMFactor,
		KerfMM:           p.KerfMM,
		TrimMM:           p.TrimMM,
		MinRemnantMM:     p.MinRemnantMM,
		UtilizationPct:   p.UtilizationPct,
		ScrapPct:         p.ScrapPct,
		StockUsedCount:   p.StockUsedCount,
		CutCount:         p.CutCount,
		TotalDemand:      p.TotalDemand,
		TotalStock:       p.TotalStock,
		CreatedAt:        p.CreatedAt,
	}
}

func toPartResponse(p *entity.CuttingPlanPart) *response.CuttingPlanPartResponse {
	grain := ""
	if p.Grain != "" && p.Grain != entity.GrainNone {
		grain = string(p.Grain)
	}
	return &response.CuttingPlanPartResponse{
		ID:              p.ID,
		PlanID:          p.PlanID,
		ItemCode:        p.ItemCode,
		Label:           p.Label,
		LengthMM:        p.LengthMM,
		WidthMM:         p.WidthMM,
		HeightMM:        p.HeightMM,
		Grain:           grain,
		AllowRotation:   p.AllowRotation,
		Geometry:        p.Geometry,
		EdgeTop:         p.EdgeTop,
		EdgeBottom:      p.EdgeBottom,
		EdgeLeft:        p.EdgeLeft,
		EdgeRight:       p.EdgeRight,
		BandItemCode:    p.BandItemCode,
		BandingLengthMM: p.BandingLengthMM(),
		Quantity:        p.Quantity,
		SourceRef:       p.SourceRef,
	}
}

func toSettingsResponse(s *entity.CuttingSettings) *response.CuttingSettingsResponse {
	return &response.CuttingSettingsResponse{
		DefaultConsumptionMode: string(s.DefaultConsumptionMode),
		DefaultMinRemnantMM:    s.DefaultMinRemnantMM,
		DefaultWarehouseID:     s.DefaultWarehouseID,
	}
}

func toRemnantResponse(r *entity.StockRemnant) *response.StockRemnantResponse {
	return &response.StockRemnantResponse{
		ID:          r.ID,
		ItemCode:    r.ItemCode,
		WarehouseID: r.WarehouseID,
		LengthMM:    r.LengthMM,
		WidthMM:     r.WidthMM,
		HeightMM:    r.HeightMM,
		Lot:         r.Lot,
		HeatNumber:  r.HeatNumber,
		Certificate: r.Certificate,
		Status:      string(r.Status),
		UnitCost:    r.UnitCost,
	}
}

func toStockResponse(s *entity.CuttingStockPiece) *response.CuttingStockPieceResponse {
	return &response.CuttingStockPieceResponse{
		ID:        s.ID,
		PlanID:    s.PlanID,
		LengthMM:  s.LengthMM,
		WidthMM:   s.WidthMM,
		HeightMM:  s.HeightMM,
		Quantity:  s.Quantity,
		Lot:       s.Lot,
		IsRemnant: s.IsRemnant,
	}
}

func buildDetail(
	plan *entity.CuttingPlan,
	parts []*entity.CuttingPlanPart,
	stock []*entity.CuttingStockPiece,
	patterns []*entity.CuttingPattern,
	unplaced []service.DemandPiece,
) *response.CuttingPlanDetailResponse {
	det := &response.CuttingPlanDetailResponse{Plan: *toPlanResponse(plan)}

	var bandLen, bandCost float64
	for _, p := range parts {
		det.Parts = append(det.Parts, *toPartResponse(p))
		l := p.BandingLengthMM()
		bandLen += l
		bandCost += (l / 1000.0) * p.BandCostPerM // cost per metre × length in metres
	}
	if bandLen > 0 {
		det.Banding = &response.BandingSummaryResponse{TotalLengthMM: bandLen, TotalCost: bandCost}
	}
	for _, s := range stock {
		det.StockPieces = append(det.StockPieces, *toStockResponse(s))
	}
	for _, pat := range patterns {
		reusable := false
		if plan.MinRemnantMM > 0 {
			if pat.StockWidthMM > 0 { // 2D
				reusable = pat.RemnantWidthMM >= plan.MinRemnantMM && pat.RemnantHeightMM >= plan.MinRemnantMM
			} else {
				reusable = pat.RemnantMM >= plan.MinRemnantMM
			}
		}
		pr := response.CuttingPatternResponse{
			Sequence:        pat.Sequence,
			StockLengthMM:   pat.StockLengthMM,
			StockWidthMM:    pat.StockWidthMM,
			StockHeightMM:   pat.StockHeightMM,
			RepeatCount:     pat.RepeatCount,
			UsedMM:          pat.UsedMM,
			UsedAreaMM2:     pat.UsedAreaMM2,
			KerfLossMM:      pat.KerfLossMM,
			RemnantMM:       pat.RemnantMM,
			RemnantAreaMM2:  pat.RemnantAreaMM2,
			RemnantWidthMM:  pat.RemnantWidthMM,
			RemnantHeightMM: pat.RemnantHeightMM,
			UtilizationPct:  pat.UtilizationPct,
			IsRemnant:       pat.IsRemnant,
			ReusableScrap:   reusable,
		}
		for _, pl := range pat.Placements {
			pr.Placements = append(pr.Placements, response.CuttingPatternPlacementResponse{
				Sequence:    int64(pl.Sequence),
				PartID:      pl.PartID,
				Label:       pl.Label,
				LengthMM:    pl.LengthMM,
				OffsetMM:    pl.OffsetMM,
				PosXMM:      pl.PosXMM,
				PosYMM:      pl.PosYMM,
				WidthMM:     pl.WidthMM,
				HeightMM:    pl.HeightMM,
				Rotated:     pl.Rotated,
				RotationDeg: pl.RotationDeg,
			})
		}
		det.Patterns = append(det.Patterns, pr)
	}
	for _, u := range unplaced {
		det.Unplaced = append(det.Unplaced, response.UnplacedPieceResponse{
			Label: u.Label, LengthMM: u.Length, WidthMM: u.Width, HeightMM: u.Height, Quantity: u.Qty,
		})
	}
	return det
}
