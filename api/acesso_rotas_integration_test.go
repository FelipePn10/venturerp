//go:build integration

package main

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/config"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	httpmw "github.com/FelipePn10/panossoerp/internal/interfaces/middleware"
)

// Toda rota autenticada tem módulo de permissão (ou é livre). Rota nova sem
// classificação faria o usuário com perfil receber "operação sem módulo" — o
// teste falha antes, no CI, apontando a rota.
func TestToRotaTemModuloDePermissao(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	db, err := database.NewDBURL(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{JWTSecret: "teste-0123456789-abcdefghijklmnopqrstuv", RateLimitRPS: 10, RateLimitBurst: 10, AuthRateLimitRPM: 10, AuthRateLimitBurst: 10}
	app := &application{config: cfg, logger: applogger.New("error"), db: db, authDB: db, metrics: httpmw.NewMetrics()}
	parametro := regexp.MustCompile(`\{[^}]+\}`)
	total, porModulo := 0, map[acesso.Modulo]int{}
	_ = chi.Walk(app.mount(), func(metodo, rota string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(rota, "/api/") {
			return nil // /health e /users (login) ficam fora do grupo autenticado
		}
		caminho := parametro.ReplaceAllString(rota, "1")
		c := httpmw.ClassificarRota(metodo, caminho)
		if !c.Conhecida {
			t.Errorf("rota sem módulo de permissão: %s %s (classifique em catalogo_rotas.go)", metodo, rota)
		}
		total++
		porModulo[c.Modulo]++
		return nil
	})
	if total < 1000 {
		t.Fatalf("o roteador deveria ter mais de mil rotas /api, veio %d", total)
	}
	for _, m := range acesso.Catalogo {
		if porModulo[m.Modulo] == 0 {
			t.Errorf("o módulo %s não tem nenhuma rota", m.Modulo)
		}
	}
}
