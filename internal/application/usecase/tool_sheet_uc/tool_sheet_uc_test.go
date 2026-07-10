package tool_sheet_uc

import "testing"

func TestMapOrderType(t *testing.T) {
	cases := map[string]string{
		"":                     "OF", // manual production order
		"PRODUCTION":           "OF",
		"OUTSOURCING":          "OFC", // excluded from the LOV
		"TECHNICAL_ASSISTANCE": "TECHNICAL_ASSISTANCE",
	}
	for raw, want := range cases {
		if got := mapOrderType(raw); got != want {
			t.Errorf("mapOrderType(%q) = %q, want %q", raw, got, want)
		}
	}
}
