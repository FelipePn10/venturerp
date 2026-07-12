package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateTokenCarriesSelectedEnterprise(t *testing.T) {
	const secret = "test-secret"
	tokenString, err := GenerateToken("user-id", "USER", 73, secret)
	if err != nil {
		t.Fatal(err)
	}
	claims := &UserClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (interface{}, error) { return []byte(secret), nil })
	if err != nil || !token.Valid {
		t.Fatalf("invalid token: %v", err)
	}
	if claims.EnterpriseID != 73 {
		t.Fatalf("expected enterprise 73, got %d", claims.EnterpriseID)
	}
}
