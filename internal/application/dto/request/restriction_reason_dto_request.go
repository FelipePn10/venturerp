package request

type CreateRestrictionReasonDTO struct {
	Description string `json:"description"`
	Situation   string `json:"situation"`
}

type UpdateRestrictionReasonDTO struct {
	Code        int64  `json:"code"`
	Description string `json:"description"`
	Situation   string `json:"situation"`
}
