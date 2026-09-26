package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// TokenTTL é a validade de um token emitido. SessionMaxDuration é o teto do
// "manter conectado": o token pode ser renovado enquanto o app é usado, mas
// nunca além deste prazo contado do login de verdade. Sem teto, um token
// perdido valeria para sempre.
const (
	// TokenTTL é a sessão comum: dura o dia de trabalho e morre quando o app
	// fecha (o cliente guarda o token só na sessão do navegador).
	TokenTTL = 24 * time.Hour
	// TokenTTLRemembered é a sessão de quem marcou "manter conectado". Uma
	// semana cobre feriado e fim de semana longo sem pedir a senha de novo; a
	// renovação a cada abertura estende essa janela até o teto.
	TokenTTLRemembered = 7 * 24 * time.Hour
	// SessionMaxDuration é o teto do "manter conectado": renovar não pode virar
	// sessão eterna, e um token perdido tem prazo para deixar de valer.
	SessionMaxDuration = 30 * 24 * time.Hour
)

type UserClaims struct {
	UserID       string `json:"user_id"`
	Role         string `json:"role"`
	EnterpriseID int64  `json:"enterprise_id"`
	AuthVersion  int64  `json:"auth_version"`
	Environment  string `json:"environment"`
	// SessionStart é quando o usuário digitou a senha. Viaja de token em token
	// pela renovação, e é ele que limita o "manter conectado" a
	// SessionMaxDuration. Token antigo não tem a claim: nesse caso vale o
	// `iat`, que é o comportamento de antes desta mudança.
	SessionStart int64 `json:"session_start,omitempty"`
	// Remember diz se o usuário pediu "manter conectado". É a claim que define
	// o prazo na renovação — sem ela, renovar rebaixaria a sessão longa para 24 h.
	Remember bool `json:"remember,omitempty"`
	jwt.RegisteredClaims
}

// SessionStartedAt devolve o início da sessão, caindo no `iat` quando o token
// foi emitido antes de a claim existir.
func (c UserClaims) SessionStartedAt() time.Time {
	if c.SessionStart > 0 {
		return time.Unix(c.SessionStart, 0)
	}
	if c.IssuedAt != nil {
		return c.IssuedAt.Time
	}
	return time.Time{}
}

func GenerateToken(userID, role string, enterpriseID, authVersion int64, secret string) (string, error) {
	return GenerateTokenForEnvironment(userID, role, enterpriseID, authVersion, "production", secret)
}

func GenerateTokenForEnvironment(userID, role string, enterpriseID, authVersion int64, environment, secret string) (string, error) {
	token, _, err := GenerateSessionToken(userID, role, enterpriseID, authVersion, environment, secret, time.Time{}, false)
	return token, err
}

// GenerateSessionToken emite o token e devolve TAMBÉM o vencimento, para o
// cliente saber quando renovar em vez de descobrir com um 401. `sessionStart`
// zerado significa login novo (a sessão começa agora).
func GenerateSessionToken(
	userID, role string, enterpriseID, authVersion int64, environment, secret string,
	sessionStart time.Time, remember bool,
) (string, time.Time, error) {
	agora := time.Now()
	if sessionStart.IsZero() {
		sessionStart = agora
	}
	ttl := TokenTTL
	if remember {
		ttl = TokenTTLRemembered
	}
	expiraEm := agora.Add(ttl)
	// O token nunca vive além do teto da sessão.
	if limite := sessionStart.Add(SessionMaxDuration); expiraEm.After(limite) {
		expiraEm = limite
	}
	claims := UserClaims{
		UserID:       userID,
		Role:         role,
		EnterpriseID: enterpriseID,
		AuthVersion:  authVersion,
		Environment:  environment,
		SessionStart: sessionStart.Unix(),
		Remember:     remember,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "panosso-erp",
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiraEm),
			IssuedAt:  jwt.NewNumericDate(agora),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	assinado, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return assinado, expiraEm, nil
}
