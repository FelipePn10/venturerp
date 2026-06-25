package cutting_plan_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
)

// ExportMap renders the plan's cutting map in the given vector format (svg/dxf/pdf).
func (uc *CuttingPlanUseCase) ExportMap(ctx context.Context, planID int64, format string) ([]byte, string, error) {
	plan, err := uc.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, "", err
	}
	patterns, err := uc.repo.ListPatterns(ctx, planID)
	if err != nil {
		return nil, "", err
	}
	if len(patterns) == 0 {
		return nil, "", fmt.Errorf("plan has no patterns; optimise it first")
	}
	b := service.MapBranding{GeneratedAt: time.Now()}
	if uc.branding != nil {
		if cfg, err := uc.branding.GetFiscalConfig(ctx); err == nil && cfg != nil {
			b.CompanyName = cfg.RazaoSocial
			if cfg.BrandColor != nil {
				b.BrandColorHex = *cfg.BrandColor
			}
		}
	}
	return service.RenderCutMap(plan.Code, patterns, service.MapFormat(format), b)
}

// GetProgram returns the ordered cut program (the shop-floor cut sequence) per
// pattern, ready to drive a saw/seccionadora.
func (uc *CuttingPlanUseCase) GetProgram(ctx context.Context, planID int64) (*response.CutProgramResponse, error) {
	plan, err := uc.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	patterns, err := uc.repo.ListPatterns(ctx, planID)
	if err != nil {
		return nil, err
	}
	out := &response.CutProgramResponse{PlanID: plan.ID, PlanCode: plan.Code, CutType: string(plan.CutType)}
	for _, p := range patterns {
		pat := response.CutProgramPatternResponse{
			Sequence: p.Sequence, RepeatCount: p.RepeatCount,
			StockLengthMM: p.StockLengthMM, StockWidthMM: p.StockWidthMM, StockHeightMM: p.StockHeightMM,
		}
		for _, pl := range p.Placements {
			pat.Steps = append(pat.Steps, response.CutProgramStepResponse{
				Sequence: pl.Sequence, Label: pl.Label,
				OffsetMM: pl.OffsetMM, LengthMM: pl.LengthMM,
				PosXMM: pl.PosXMM, PosYMM: pl.PosYMM, WidthMM: pl.WidthMM, HeightMM: pl.HeightMM, RotationDeg: pl.RotationDeg,
			})
		}
		out.Patterns = append(out.Patterns, pat)
	}
	return out, nil
}

// ScheduleOnMachine books the plan on its machine (a MachineSchedule entry),
// sequencing the cut on the shop-floor calendar. Requires a machine on the plan
// and an injected machine repository.
func (uc *CuttingPlanUseCase) ScheduleOnMachine(ctx context.Context, planID int64) (*response.CutScheduleResponse, error) {
	if uc.machines == nil {
		return nil, fmt.Errorf("machine scheduling is not configured")
	}
	plan, err := uc.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.MachineCode == nil {
		return nil, fmt.Errorf("plan has no machine set")
	}
	patterns, err := uc.repo.ListPatterns(ctx, planID)
	if err != nil {
		return nil, err
	}
	pieces := 0
	for _, p := range patterns {
		pieces += len(p.Placements) * p.RepeatCount
	}

	// A cutting plan is not a planned order, so order_code stays null; the plan is
	// referenced via the machine + notes.
	sched := &machineentity.MachineSchedule{
		MachineCode:  *plan.MachineCode,
		OrderCode:    nil,
		ScheduleDate: time.Now().Truncate(24 * time.Hour),
		PlannedQty:   float64(pieces),
		Sequence:     1,
		Notes:        strPtr(fmt.Sprintf("Plano de corte #%d", plan.Code)),
	}
	created, err := uc.machines.CreateSchedule(ctx, sched)
	if err != nil {
		return nil, fmt.Errorf("scheduling cut on machine: %w", err)
	}
	return &response.CutScheduleResponse{
		PlanID:        plan.ID,
		PlanCode:      plan.Code,
		ScheduleCode:  created.Code,
		MachineCode:   created.MachineCode,
		PlannedPieces: pieces,
		ScheduleDate:  created.ScheduleDate,
	}, nil
}
