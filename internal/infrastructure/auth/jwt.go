package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type UserClaims struct {
	UserID       string `json:"user_id"`
	Role         string `json:"role"`
	EnterpriseID int64  `json:"enterprise_id"`
	AuthVersion  int64  `json:"auth_version"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, role string, enterpriseID, authVersion int64, secret string) (string, error) {
	claims := UserClaims{
		UserID:       userID,
		Role:         role,
		EnterpriseID: enterpriseID,
		AuthVersion:  authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "panosso-erp",
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
