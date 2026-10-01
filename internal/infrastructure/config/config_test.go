package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadRequiresJWTSecretInProduction(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "at least 32") {
		t.Fatalf("Load() error = %v, want missing JWT_SECRET error", err)
	}
}

func TestLoadAcceptsJWTSecretInProduction(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "test-only-production-secret-32-chars")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.JWTSecret != "test-only-production-secret-32-chars" {
		t.Fatalf("JWTSecret = %q, want environment value", cfg.JWTSecret)
	}
}

func TestLoadRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "development")
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,not-a-network")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXY_CIDRS") {
		t.Fatalf("Load() error = %v, want trusted proxy validation error", err)
	}
}

func TestLoadTrainingRequiresSeparateIdentityDatabase(t *testing.T) {
	for name, identityURL := range map[string]string{
		"missing":       "",
		"same database": "postgres://localhost/training",
		"same database with other credentials and options": "postgresql://other:secret@localhost:5432/training?application_name=identity",
	} {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			t.Setenv("ENV", "development")
			t.Setenv("DATA_ENVIRONMENT", "training")
			t.Setenv("DATABASE_URL", "postgres://localhost/training")
			t.Setenv("IDENTITY_DATABASE_URL", identityURL)
			if _, err := Load(); err == nil {
				t.Fatal("Load() expected unsafe training configuration to fail")
			}
		})
	}
}

func TestLoadAcceptsSeparatedTrainingDatabases(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("ENV", "development")
	t.Setenv("DATA_ENVIRONMENT", "training")
	t.Setenv("DATABASE_URL", "postgres://localhost/training")
	t.Setenv("IDENTITY_DATABASE_URL", "postgres://localhost/production")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsTraining() {
		t.Fatal("expected training configuration")
	}
}

// TestPoolDePadraoNaoDependeDaCPU: o padrão do pgxpool é max(4, NumCPU), o que na
// VPS de 3 vCPU daria quatro conexões para a empresa inteira. O padrão explícito
// existe para não depender do tamanho da máquina.
func TestPoolDePadraoNaoDependeDaCPU(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("JWT_SECRET", "segredo-de-teste-suficientemente-longo-para-passar")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBMaxConns != 20 {
		t.Fatalf("DB_MAX_CONNS padrão = %d, esperado 20", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 2 {
		t.Fatalf("DB_MIN_CONNS padrão = %d, esperado 2", cfg.DBMinConns)
	}
	if cfg.DBMaxConnIdleTime >= 30 {
		t.Fatalf("DB_MAX_CONN_IDLE_MIN = %d; deve ser menor que os 30 do driver numa máquina pequena", cfg.DBMaxConnIdleTime)
	}
}

// TestPoolInvalidoFalhaNaConfiguracao: errar o teto tem de parar aqui, com o nome
// da variável, e não virar erro de driver na abertura do pool.
func TestPoolInvalidoFalhaNaConfiguracao(t *testing.T) {
	for _, caso := range []struct {
		nome, variavel, valor, esperado string
	}{
		{"teto zero", "DB_MAX_CONNS", "0", "DB_MAX_CONNS"},
		{"teto negativo", "DB_MAX_CONNS", "-3", "DB_MAX_CONNS"},
		{"mínimo negativo", "DB_MIN_CONNS", "-1", "DB_MIN_CONNS"},
		{"vida útil zero", "DB_MAX_CONN_LIFETIME_MIN", "0", "DB_MAX_CONN_LIFETIME_MIN"},
		{"ociosidade zero", "DB_MAX_CONN_IDLE_MIN", "0", "DB_MAX_CONN_IDLE_MIN"},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			t.Setenv("JWT_SECRET", "segredo-de-teste-suficientemente-longo-para-passar")
			t.Setenv(caso.variavel, caso.valor)
			_, err := Load()
			if err == nil {
				t.Fatalf("%s=%s foi aceito", caso.variavel, caso.valor)
			}
			if !strings.Contains(err.Error(), caso.esperado) {
				t.Fatalf("erro não cita %s: %v", caso.esperado, err)
			}
		})
	}
}

// TestMinimoAcimaDoTetoEhRecusado: aceitar isso faria o pgxpool recusar a abertura
// do pool, com a API já tentando subir.
func TestMinimoAcimaDoTetoEhRecusado(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("JWT_SECRET", "segredo-de-teste-suficientemente-longo-para-passar")
	t.Setenv("DB_MAX_CONNS", "5")
	t.Setenv("DB_MIN_CONNS", "9")
	_, err := Load()
	if err == nil {
		t.Fatal("DB_MIN_CONNS acima de DB_MAX_CONNS foi aceito")
	}
	if !strings.Contains(err.Error(), "DB_MIN_CONNS") {
		t.Fatalf("erro não explica o conflito: %v", err)
	}
}
