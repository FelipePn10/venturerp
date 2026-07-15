package version

import "testing"

func TestCurrentNormalizesInjectedVersions(t *testing.T) {
	originalVersion, originalMinClient := Version, MinClient
	t.Cleanup(func() { Version, MinClient = originalVersion, originalMinClient })

	Version = " v1.4.0 "
	MinClient = "v1.2.0"

	got := Current()
	if got.Version != "1.4.0" || got.MinClient != "1.2.0" {
		t.Fatalf("Current() = %#v, want normalized semantic versions", got)
	}
}

func TestCurrentUsesDevelopmentMarkerForEmptyBuildValues(t *testing.T) {
	originalVersion, originalMinClient := Version, MinClient
	t.Cleanup(func() { Version, MinClient = originalVersion, originalMinClient })

	Version, MinClient = "", " "
	got := Current()
	if got.Version != "dev" || got.MinClient != "dev" {
		t.Fatalf("Current() = %#v, want development markers", got)
	}
}
