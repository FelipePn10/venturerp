package response

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRouteLeadTimeUsesStableSnakeCaseContract(t *testing.T) {
	payload, err := json.Marshal(RouteLeadTimeResponse{RouteID: 7, TotalHours: 3.5, CriticalPath: []int64{10, 20}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, field := range []string{`"route_id"`, `"lead_time_hours"`, `"critical_path"`} {
		if !strings.Contains(text, field) {
			t.Fatalf("response %s does not contain %s", text, field)
		}
	}
	if strings.Contains(text, `"total_hours"`) {
		t.Fatalf("legacy lead-time field leaked in %s", text)
	}
}
