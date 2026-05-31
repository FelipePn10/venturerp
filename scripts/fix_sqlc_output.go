//go:build ignore

// fix_sqlc_output.go patches models.go after `sqlc generate` to remove
// known conflicts that cannot be resolved through sqlc configuration alone.
//
// Run via: go run scripts/fix_sqlc_output.go
// Or via Makefile target: make sqlc
package main

import (
	"fmt"
	"os"
	"strings"
)

const modelsPath = "internal/infrastructure/database/sqlc/models.go"

func main() {
	content, err := os.ReadFile(modelsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading models.go:", err)
		os.Exit(1)
	}

	patched, changed := removeOldEmployeeLegacy(string(content))

	if !changed {
		fmt.Println("models.go: nothing to patch (old EmployeeLegacy already absent)")
		return
	}

	if err := os.WriteFile(modelsPath, []byte(patched), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing models.go:", err)
		os.Exit(1)
	}

	fmt.Println("models.go: removed duplicate EmployeeLegacy struct (old 'employee' table)")
}

// removeOldEmployeeLegacy finds and removes the `type EmployeeLegacy struct`
// block that contains the `EnterpriseID` field (generated from the legacy
// `employee` table in migration 000033). The other, full struct—generated from
// the `employees` table—is kept intact. The search is order-independent: sqlc
// may emit the two structs in either order, so we scan every EmployeeLegacy
// block and remove the one carrying EnterpriseID.
func removeOldEmployeeLegacy(text string) (string, bool) {
	const marker = "type EmployeeLegacy struct {"

	for searchFrom := 0; ; {
		rel := strings.Index(text[searchFrom:], marker)
		if rel == -1 {
			return text, false
		}
		idx := searchFrom + rel

		// Find the closing brace of this struct.
		closeRel := strings.Index(text[idx:], "\n}\n")
		if closeRel == -1 {
			return text, false
		}
		blockEnd := idx + closeRel + len("\n}\n")
		block := text[idx:blockEnd]

		// Only remove the OLD struct (the one with EnterpriseID).
		if !strings.Contains(block, "EnterpriseID") {
			searchFrom = blockEnd
			continue
		}

		// Include the blank line that precedes the struct declaration.
		blockStart := idx
		if blockStart >= 2 && text[blockStart-1] == '\n' && text[blockStart-2] == '\n' {
			blockStart--
		}

		return text[:blockStart] + text[blockEnd:], true
	}
}
