package security

import "time"

type AuthUser struct {
	ID             string
	Role           string
	EnterpriseID   int64
	EnterpriseCode int64
}

// SessionInfo é o que o token diz sobre a própria sessão: quando ela começou
// (para o teto do "manter conectado" contar do login de verdade) e se o usuário
// pediu para continuar conectado (para a renovação não rebaixar o prazo).
type SessionInfo struct {
	Start    time.Time
	Remember bool
}
