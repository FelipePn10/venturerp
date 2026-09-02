package planning_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
)

func TestPlanningPipelineValidatesReferencesBeforeEngines(t *testing.T) {
	uc := &RunPlanningPipelineUseCase{}
	tests := []struct {
		name string
		dto  request.RunPlanningPipelineDTO
		want string
	}{
		{"plano", request.RunPlanningPipelineDTO{InitialOrderNumber: 1, StartFrom: time.Now()}, "plan_code"},
		{"ordem", request.RunPlanningPipelineDTO{PlanCode: 1, StartFrom: time.Now()}, "initial_order_number"},
		{"início", request.RunPlanningPipelineDTO{PlanCode: 1, InitialOrderNumber: 1}, "start_from"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), tc.dto)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("erro=%v; esperado campo %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "violates foreign key") || strings.Contains(err.Error(), "no rows") {
				t.Fatalf("erro de infraestrutura vazado: %v", err)
			}
		})
	}
}
