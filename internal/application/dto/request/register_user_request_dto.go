package request

type RegisterUserDTO struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	EnterpriseCode int64  `json:"enterprise_code"`
}
