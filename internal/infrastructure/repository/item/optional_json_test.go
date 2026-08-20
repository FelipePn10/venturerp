package item

import "testing"

func TestHasOptionalJSONValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "SQL NULL representation", want: false},
		{name: "JSON null", raw: " null ", want: false},
		{name: "empty object", raw: "{}", want: false},
		{name: "empty object with whitespace", raw: "{ \n\t }", want: false},
		{name: "non-empty valid object", raw: `{"days_interval":1}`, want: true},
		{name: "non-empty invalid domain object", raw: `{"days_interval":0}`, want: true},
		{name: "malformed value", raw: `{`, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasOptionalJSONValue([]byte(tt.raw)); got != tt.want {
				t.Fatalf("hasOptionalJSONValue(%q) = %t, want %t", tt.raw, got, tt.want)
			}
		})
	}
}
