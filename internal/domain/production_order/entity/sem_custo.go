package entity

import "github.com/google/uuid"

// OFSemCusto é a OF que já teve movimento (apontamento, horas de etapa, saída
// ou entrada de estoque por ela) e nunca teve o custo real apurado — movida
// antes de o custo ser automático, ou por carga direta no banco. A rotina que
// apura a trata na sessão da empresa dela.
type OFSemCusto struct {
	ID             int64
	EnterpriseID   int64
	EnterpriseCode int64
	CreatedBy      uuid.UUID
}
