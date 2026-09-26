package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// O token novo precisa dizer quando vence e quando a sessão começou: é o que o
// "manter conectado" usa para renovar antes de o usuário levar um 401.
func TestTokenNovoCarregaInicioDeSessaoEVencimento(t *testing.T) {
	token, expiraEm, err := GenerateSessionToken("11111111-1111-1111-1111-111111111111", "USER", 7, 3, "production", "segredo", time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(expiraEm); d < TokenTTL-time.Minute || d > TokenTTL+time.Minute {
		t.Fatalf("vencimento fora do TTL: %s", d)
	}
	claims := lerClaims(t, token)
	if claims.SessionStart == 0 {
		t.Fatal("token novo não trouxe o início da sessão")
	}
	if inicio := claims.SessionStartedAt(); time.Since(inicio) > time.Minute {
		t.Fatalf("início da sessão errado: %s", inicio)
	}
}

// Renovar mantém o início da sessão — é ele que limita o "manter conectado".
func TestRenovacaoPreservaOInicioDaSessao(t *testing.T) {
	inicio := time.Now().Add(-10 * 24 * time.Hour)
	token, expiraEm, err := GenerateSessionToken("11111111-1111-1111-1111-111111111111", "USER", 7, 3, "production", "segredo", inicio, true)
	if err != nil {
		t.Fatal(err)
	}
	claims := lerClaims(t, token)
	if claims.SessionStartedAt().Unix() != inicio.Unix() {
		t.Fatalf("o início da sessão mudou na renovação: %s", claims.SessionStartedAt())
	}
	if !claims.Remember {
		t.Fatal("a escolha de manter conectado não sobreviveu à renovação")
	}
	if d := time.Until(expiraEm); d < TokenTTLRemembered-time.Minute {
		t.Fatalf("renovação no meio da janela devia dar prazo cheio de uma semana, deu %s", d)
	}
}

// Perto do teto, o prazo é o que resta — não mais 24 h. Sem isto, renovar no
// último dia estenderia a sessão além dos 30 dias.
func TestRenovacaoNaBordaNaoPassaDoTeto(t *testing.T) {
	inicio := time.Now().Add(-SessionMaxDuration + 2*time.Hour)
	_, expiraEm, err := GenerateSessionToken("11111111-1111-1111-1111-111111111111", "USER", 7, 3, "production", "segredo", inicio, true)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(expiraEm); d > 2*time.Hour+time.Minute {
		t.Fatalf("o token passou do teto da sessão: faltam %s", d)
	}
}

// Token emitido antes desta mudança não tem a claim: o `iat` faz o papel dela.
func TestTokenAntigoUsaIatComoInicioDeSessao(t *testing.T) {
	emitido := time.Now().Add(-3 * time.Hour)
	claims := UserClaims{RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(emitido)}}
	if claims.SessionStartedAt().Unix() != emitido.Unix() {
		t.Fatalf("token sem session_start deveria cair no iat, veio %s", claims.SessionStartedAt())
	}
	vazio := UserClaims{}
	if !vazio.SessionStartedAt().IsZero() {
		t.Fatal("sem iat e sem session_start o início é desconhecido")
	}
}

func lerClaims(t *testing.T, token string) UserClaims {
	t.Helper()
	var claims UserClaims
	if _, _, err := jwt.NewParser().ParseUnverified(token, &claims); err != nil {
		t.Fatal(err)
	}
	return claims
}

// A caixa "manter conectado" é o que muda o prazo — é o comportamento que o
// usuário percebe: sem ela, a sessão é do dia; com ela, da semana.
func TestManterConectadoMudaOPrazoDaSessao(t *testing.T) {
	_, semLembrar, err := GenerateSessionToken("11111111-1111-1111-1111-111111111111", "USER", 7, 3, "production", "segredo", time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	_, lembrando, err := GenerateSessionToken("11111111-1111-1111-1111-111111111111", "USER", 7, 3, "production", "segredo", time.Time{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(semLembrar); d > TokenTTL+time.Minute {
		t.Fatalf("sessão comum passou de %s: %s", TokenTTL, d)
	}
	if d := time.Until(lembrando); d < TokenTTLRemembered-time.Minute {
		t.Fatalf("sessão com manter conectado deveria durar %s, deu %s", TokenTTLRemembered, d)
	}
}
