package request

import "github.com/google/uuid"

type UpdatePlanningParamDTO struct {
	ParamNumber int    `json:"param_number"`
	Value       string `json:"value"`
	// UpdatedBy vem da sessão (JWT), nunca do corpo: o autor da alteração não
	// pode ser escolhido por quem chama.
	UpdatedBy uuid.UUID `json:"-"`
}
