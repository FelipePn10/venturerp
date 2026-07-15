// Package version exposes build metadata injected by the release pipeline.
package version

import "strings"

// Version and MinClient are replaced at build time with:
//
//	-ldflags "-X github.com/FelipePn10/panossoerp/internal/version.Version=1.2.3 ..."
//
// Development builds intentionally identify themselves as dev.
var (
	Version   = "dev"
	MinClient = "dev"
)

// Info is the stable public API contract consumed by desktop clients.
type Info struct {
	Version   string `json:"version"`
	MinClient string `json:"min_client"`
}

// Current returns normalized build metadata without a leading v.
func Current() Info {
	return Info{
		Version:   normalize(Version),
		MinClient: normalize(MinClient),
	}
}

func normalize(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	if value == "" {
		return "dev"
	}
	return value
}
