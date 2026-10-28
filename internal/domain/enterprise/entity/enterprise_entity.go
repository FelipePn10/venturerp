package entity

import "github.com/google/uuid"

type Enterprise struct {
	ID        int
	Code      int
	Name      string
	CreatedBy uuid.UUID
	// Fiscal: identidade fiscal informada no cadastro (opcional).
	Fiscal *DadosFiscais
}
