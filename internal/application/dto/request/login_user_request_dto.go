package request

type LoginUserDTO struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	EnterpriseCode *int64 `json:"enterprise_code,omitempty"`
}
