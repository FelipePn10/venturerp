package entity

import "fmt"

// ValidateOperationTransition keeps execution changes explicit and terminal states immutable.
func ValidateOperationTransition(from, to, reason string) error {
	allowed := map[string]map[string]bool{
		"PENDING":     {"IN_PROGRESS": true, "SKIPPED": true},
		"IN_PROGRESS": {"PAUSED": true, "INTERRUPTED": true, "DONE": true},
		"PAUSED":      {"IN_PROGRESS": true},
		"INTERRUPTED": {"IN_PROGRESS": true},
	}
	if !allowed[from][to] {
		return fmt.Errorf("transição de etapa não permitida: %s → %s", from, to)
	}
	if (to == "PAUSED" || to == "INTERRUPTED" || to == "SKIPPED") && reason == "" {
		return fmt.Errorf("informe o motivo da pausa, interrupção ou dispensa")
	}
	return nil
}
