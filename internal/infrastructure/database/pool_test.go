package database

import (
	"strings"
	"testing"
	"time"
)

// TestPoolLimitsAplicamSobreOPadraoDoDriver cobre o motivo de esta configuração
// existir: o padrão do pgxpool é max(4, NumCPU), e numa VPS de 3 vCPU isso dá
// quatro conexões para a empresa inteira. O teste prova que o valor configurado
// realmente chega ao pool, em vez de continuar valendo o padrão.
func TestPoolLimitsAplicamSobreOPadraoDoDriver(t *testing.T) {
	// Sem servidor: só o caminho de configuração até o Ping interessa aqui, e
	// NewDBURLWithLimits falha no Ping — por isso a configuração é montada pelo
	// mesmo caminho, mas inspecionada antes de conectar.
	const url = "postgres://u:p@127.0.0.1:1/d?sslmode=disable"

	cfg, err := buildPoolConfig(url, PoolLimits{
		MaxConns:        25,
		MinConns:        3,
		MaxConnLifetime: 45 * time.Minute,
		MaxConnIdleTime: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("montando configuração: %v", err)
	}
	if cfg.MaxConns != 25 {
		t.Fatalf("MaxConns = %d, esperado 25 (o padrão do driver venceu)", cfg.MaxConns)
	}
	if cfg.MinConns != 3 {
		t.Fatalf("MinConns = %d, esperado 3", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != 45*time.Minute {
		t.Fatalf("MaxConnLifetime = %v, esperado 45m", cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime != 10*time.Minute {
		t.Fatalf("MaxConnIdleTime = %v, esperado 10m", cfg.MaxConnIdleTime)
	}
}

// TestPoolLimitsZeroPreservaOPadrao: quem não configura nada continua com o
// comportamento do driver, para que ferramentas e testes não mudem de perfil.
func TestPoolLimitsZeroPreservaOPadrao(t *testing.T) {
	const url = "postgres://u:p@127.0.0.1:1/d?sslmode=disable"

	padrao, err := buildPoolConfig(url, PoolLimits{})
	if err != nil {
		t.Fatalf("montando configuração: %v", err)
	}
	if padrao.MaxConns < 4 {
		t.Fatalf("MaxConns = %d, o padrão do pgxpool é no mínimo 4", padrao.MaxConns)
	}
}

// TestURLVenceSobreOPadraoGlobal: um `pool_max_conns` escrito na URL é escolha
// explícita daquele ambiente e não pode ser apagada pela configuração global.
func TestURLVenceSobreOPadraoGlobal(t *testing.T) {
	url := "postgres://u:p@127.0.0.1:1/d?sslmode=disable&pool_max_conns=7"

	cfg, err := buildPoolConfig(url, PoolLimits{MaxConns: 25})
	if err != nil {
		t.Fatalf("montando configuração: %v", err)
	}
	if cfg.MaxConns != 7 {
		t.Fatalf("MaxConns = %d, esperado 7 — a URL deveria vencer", cfg.MaxConns)
	}
}

// TestMinNuncaPassaDoMax: o pgxpool recusa abrir o pool nessa situação, e um erro
// do driver na subida da API é pior que um ajuste silencioso e coerente.
func TestMinNuncaPassaDoMax(t *testing.T) {
	const url = "postgres://u:p@127.0.0.1:1/d?sslmode=disable"

	cfg, err := buildPoolConfig(url, PoolLimits{MaxConns: 5, MinConns: 9})
	if err != nil {
		t.Fatalf("montando configuração: %v", err)
	}
	if cfg.MinConns > cfg.MaxConns {
		t.Fatalf("MinConns (%d) passou de MaxConns (%d)", cfg.MinConns, cfg.MaxConns)
	}
}

// TestURLInvalidaFalhaComMensagemUtil: URL quebrada tem de parar aqui, e não
// virar pânico mais adiante.
func TestURLInvalidaFalhaComMensagemUtil(t *testing.T) {
	if _, err := buildPoolConfig("nao-e-uma-url://", PoolLimits{}); err == nil {
		t.Fatal("URL inválida foi aceita")
	} else if !strings.Contains(err.Error(), "DB config") {
		t.Fatalf("mensagem não identifica o problema: %v", err)
	}
}
