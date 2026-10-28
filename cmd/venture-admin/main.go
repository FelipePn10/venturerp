// Command venture-admin reúne os gestos que só a Venture ERP faz numa base de
// cliente, direto no servidor. Hoje: o token Focus NF-e de produção, que a
// tela da empresa não grava nem troca (ver entity.ConferirTrocaDoFocus).
//
// Roda dentro do contêiner da API, que já tem DATABASE_URL:
//
//	docker exec -i <api> /app/venture-admin focus-status -empresa 1
//	docker exec -i <api> /app/venture-admin focus-producao -empresa 1 \
//	  -por "Felipe (Venture)" -motivo "início da emissão em produção" < token.txt
//	docker exec -i <api> /app/venture-admin focus-homologacao -empresa 1 \
//	  -por "Felipe (Venture)" -motivo "troca de certificado"
//
// O token vem pela entrada padrão, nunca por argumento (que fica no histórico
// do shell e na lista de processos). Cada troca fica em focus_token_eventos,
// que não aceita alteração nem exclusão.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		uso()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	empresa := fs.Int("empresa", 0, "código da empresa (enterprise.code)")
	por := fs.String("por", "", "quem executa, pela Venture (obrigatório para gravar)")
	motivo := fs.String("motivo", "", "motivo da troca (obrigatório para gravar)")
	dbURL := fs.String("database-url", os.Getenv("DATABASE_URL"), "conexão com a base do cliente")
	_ = fs.Parse(args)
	if *empresa <= 0 || *dbURL == "" {
		fmt.Fprintln(os.Stderr, "informe -empresa e a conexão (DATABASE_URL ou -database-url)")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		falhar(err)
	}
	defer pool.Close()

	switch cmd {
	case "focus-status":
		err = status(ctx, pool, *empresa)
	case "focus-producao":
		var token string
		if token, err = lerToken(os.Stdin); err == nil {
			err = gravar(ctx, pool, *empresa, "PRODUCAO_GRAVADO", token, *por, *motivo)
		}
	case "focus-homologacao":
		err = gravar(ctx, pool, *empresa, "VOLTOU_HOMOLOGACAO", "", *por, *motivo)
	default:
		uso()
		os.Exit(2)
	}
	if err != nil {
		falhar(err)
	}
}

func uso() {
	fmt.Fprintln(os.Stderr, `uso: venture-admin <comando> -empresa N [-por "..." -motivo "..."]
  focus-status       mostra o ambiente e o final do token
  focus-producao     grava o token de produção (lido da entrada padrão) e passa para produção
  focus-homologacao  volta para homologação e apaga o token de produção`)
}

func falhar(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}

func lerToken(r io.Reader) (string, error) {
	linha, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	t := strings.TrimSpace(linha)
	if len(t) < 16 || strings.ContainsAny(t, " \t") {
		return "", errors.New("token inválido na entrada padrão (esperado o token da Focus, numa linha)")
	}
	return t, nil
}

func final(t string) string {
	if len(t) <= 4 {
		return t
	}
	return t[len(t)-4:]
}

func status(ctx context.Context, pool *pgxpool.Pool, empresa int) error {
	var nome, ambiente string
	var token *string
	err := pool.QueryRow(ctx, `SELECT e.name, COALESCE(f.focus_nfe_ambiente, ''), f.focus_nfe_token
		  FROM enterprise e LEFT JOIN fiscal_configs f ON f.enterprise_id = e.id WHERE e.code = $1`, empresa).Scan(&nome, &ambiente, &token)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("empresa %d não encontrada", empresa)
	}
	if err != nil {
		return err
	}
	fim := "sem token"
	if token != nil && *token != "" {
		fim = "token …" + final(*token)
	}
	fmt.Printf("empresa %d (%s): ambiente %q, %s\n", empresa, nome, ambiente, fim)
	return nil
}

func gravar(ctx context.Context, pool *pgxpool.Pool, empresa int, acao, token, por, motivo string) error {
	por, motivo = strings.TrimSpace(por), strings.TrimSpace(motivo)
	if por == "" || motivo == "" {
		return errors.New("informe -por (quem executa) e -motivo")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var enterpriseID int64
	err = tx.QueryRow(ctx, `SELECT f.enterprise_id FROM fiscal_configs f JOIN enterprise e ON e.id = f.enterprise_id
		 WHERE e.code = $1 FOR UPDATE OF f`, empresa).Scan(&enterpriseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("a empresa %d não tem configuração fiscal: cadastre a identificação na VEMP0100 antes", empresa)
	}
	if err != nil {
		return err
	}
	if acao == "PRODUCAO_GRAVADO" {
		_, err = tx.Exec(ctx, `UPDATE fiscal_configs SET focus_nfe_token = $2, focus_nfe_ambiente = 'producao', updated_at = NOW() WHERE enterprise_id = $1`, enterpriseID, token)
	} else {
		_, err = tx.Exec(ctx, `UPDATE fiscal_configs SET focus_nfe_token = NULL, focus_nfe_ambiente = 'homologacao', updated_at = NOW() WHERE enterprise_id = $1`, enterpriseID)
	}
	if err != nil {
		return err
	}
	var fim *string
	if token != "" {
		f := final(token)
		fim = &f
	}
	if _, err := tx.Exec(ctx, `INSERT INTO focus_token_eventos (enterprise_id, acao, token_final, executado_por, motivo) VALUES ($1,$2,$3,$4,$5)`,
		enterpriseID, acao, fim, por, motivo); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return status(ctx, pool, empresa)
}
