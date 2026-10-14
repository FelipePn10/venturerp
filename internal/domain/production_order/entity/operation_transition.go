package entity

import (
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// ValidateOperationTransition keeps execution changes explicit and terminal states immutable.
func ValidateOperationTransition(from, to, reason string) error {
	allowed := map[string]map[string]bool{
		"PENDING":     {"IN_PROGRESS": true, "SKIPPED": true},
		"IN_PROGRESS": {"PAUSED": true, "INTERRUPTED": true, "DONE": true},
		"PAUSED":      {"IN_PROGRESS": true},
		"INTERRUPTED": {"IN_PROGRESS": true},
	}
	if !allowed[from][to] {
		return domainerr.NewValidation(fmt.Sprintf("transição de etapa não permitida: %s → %s", from, to))
	}
	if (to == "PAUSED" || to == "INTERRUPTED" || to == "SKIPPED") && reason == "" {
		return domainerr.NewValidation("informe o motivo da pausa, interrupção ou dispensa")
	}
	return nil
}
