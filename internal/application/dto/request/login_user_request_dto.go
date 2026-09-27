package request

type LoginUserDTO struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	EnterpriseCode *int64 `json:"enterprise_code,omitempty"`
	// RememberMe é o "manter conectado" da tela de login. Com ele o token vale
	// uma semana e é renovado a cada abertura do app (até o teto de 30 dias);
	// sem ele vale o dia de trabalho e o cliente guarda a sessão só enquanto o
	// app está aberto.
	RememberMe bool `json:"remember_me,omitempty"`
}
