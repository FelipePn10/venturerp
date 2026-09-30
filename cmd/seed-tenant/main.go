// Command seed-tenant dá o primeiro habitante a uma base recém-criada de empresa
// cliente: a empresa, os usuários que vão entrar no sistema e, opcionalmente, a
// configuração fiscal inicial.
//
// Existe porque POST /users/register exige um token ADMIN, e numa base nova não
// há ninguém para emiti-lo. Este comando resolve esse ovo-e-galinha e nada além.
//
//	go run ./cmd/seed-tenant \
//	  -database-url "postgres://venturerp_usimac:...@127.0.0.1:5432/venturerp_usimac?sslmode=disable" \
//	  -enterprise-name "USIMAC USINAGEM TATUI" \
//	  -user "JHONATA OLIVEIRA|compras@usimacusinagem.com.br|ADMIN" \
//	  -user "ADRIANA LOPES|financeiro@usimacusinagem.com.br|ADMIN"
//
// É idempotente: rodar de novo não recria ninguém nem troca senha de quem já
// existe. As senhas temporárias são mostradas UMA vez, na saída — não ficam
// gravadas em lugar algum além do hash bcrypt.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type userSpec struct {
	name  string
	email string
	role  string
}

type userList []userSpec

func (u *userList) String() string { return fmt.Sprintf("%d usuário(s)", len(*u)) }

func (u *userList) Set(value string) error {
	parts := strings.Split(value, "|")
	if len(parts) != 3 {
		return errors.New(`use o formato "Nome Completo|email@empresa.com.br|ADMIN"`)
	}
	name := strings.TrimSpace(parts[0])
	email := strings.ToLower(strings.TrimSpace(parts[1]))
	role := strings.ToUpper(strings.TrimSpace(parts[2]))
	if name == "" || email == "" {
		return errors.New("nome e e-mail são obrigatórios")
	}
	if !strings.Contains(email, "@") {
		return fmt.Errorf("e-mail inválido: %s", email)
	}
	switch role {
	case "ADMIN", "USER", "OPERATOR", "VIEWER":
	default:
		return fmt.Errorf("perfil inválido: %s (use ADMIN, USER, OPERATOR ou VIEWER)", role)
	}
	*u = append(*u, userSpec{name: name, email: email, role: role})
	return nil
}

func main() {
	var users userList
	databaseURL := flag.String("database-url", "", "URL da base da empresa (obrigatório)")
	enterpriseCode := flag.Int("enterprise-code", 1, "código da empresa dentro da base")
	enterpriseName := flag.String("enterprise-name", "", "razão social abreviada, até 35 caracteres (obrigatório)")
	flag.Var(&users, "user", `usuário no formato "Nome|email|PERFIL"; pode repetir`)
	fiscalCNPJ := flag.String("fiscal-cnpj", "", "CNPJ só com dígitos; habilita a configuração fiscal inicial")
	fiscalRazao := flag.String("fiscal-razao-social", "", "razão social completa")
	fiscalIE := flag.String("fiscal-ie", "", "inscrição estadual")
	fiscalUF := flag.String("fiscal-uf", "", "UF da empresa")
	fiscalRegime := flag.String("fiscal-regime", "lucro_presumido", "regime tributário")
	fiscalLogradouro := flag.String("fiscal-logradouro", "", "logradouro")
	fiscalNumero := flag.String("fiscal-numero", "", "número")
	fiscalBairro := flag.String("fiscal-bairro", "", "bairro")
	fiscalMunicipio := flag.String("fiscal-municipio", "", "município")
	fiscalCodigoMunicipio := flag.String("fiscal-codigo-municipio", "", "código IBGE do município")
	fiscalCEP := flag.String("fiscal-cep", "", "CEP só com dígitos")
	dryRun := flag.Bool("dry-run", false, "mostra o que faria, sem gravar")
	flag.Parse()

	if *databaseURL == "" || *enterpriseName == "" || len(users) == 0 {
		fmt.Fprintln(os.Stderr, "seed-tenant: -database-url, -enterprise-name e ao menos um -user são obrigatórios")
		flag.Usage()
		os.Exit(2)
	}
	if len(*enterpriseName) > 35 {
		fmt.Fprintf(os.Stderr, "seed-tenant: -enterprise-name tem %d caracteres; o limite da coluna é 35\n", len(*enterpriseName))
		os.Exit(2)
	}
	// Uma base de empresa sem ADMIN não consegue cadastrar mais ninguém depois.
	hasAdmin := false
	for _, u := range users {
		if u.role == "ADMIN" {
			hasAdmin = true
		}
	}
	if !hasAdmin {
		fmt.Fprintln(os.Stderr, "seed-tenant: pelo menos um usuário precisa ser ADMIN, senão ninguém poderá cadastrar os demais")
		os.Exit(2)
	}

	if err := run(*databaseURL, *enterpriseCode, *enterpriseName, users, fiscalConfig{
		cnpj:            digitsOnly(*fiscalCNPJ),
		razaoSocial:     *fiscalRazao,
		ie:              digitsOnly(*fiscalIE),
		uf:              strings.ToUpper(*fiscalUF),
		regime:          *fiscalRegime,
		logradouro:      *fiscalLogradouro,
		numero:          *fiscalNumero,
		bairro:          *fiscalBairro,
		municipio:       *fiscalMunicipio,
		codigoMunicipio: digitsOnly(*fiscalCodigoMunicipio),
		cep:             digitsOnly(*fiscalCEP),
	}, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "seed-tenant: %v\n", err)
		os.Exit(1)
	}
}

type fiscalConfig struct {
	cnpj, razaoSocial, ie, uf, regime     string
	logradouro, numero, bairro, municipio string
	codigoMunicipio, cep                  string
}

func (f fiscalConfig) requested() bool { return f.cnpj != "" }

func (f fiscalConfig) validate() error {
	if len(f.cnpj) != 14 {
		return fmt.Errorf("CNPJ deve ter 14 dígitos, recebi %d", len(f.cnpj))
	}
	missing := []string{}
	for label, value := range map[string]string{
		"-fiscal-razao-social":     f.razaoSocial,
		"-fiscal-uf":               f.uf,
		"-fiscal-logradouro":       f.logradouro,
		"-fiscal-numero":           f.numero,
		"-fiscal-bairro":           f.bairro,
		"-fiscal-municipio":        f.municipio,
		"-fiscal-codigo-municipio": f.codigoMunicipio,
		"-fiscal-cep":              f.cep,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, label)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("a configuração fiscal exige também: %s", strings.Join(missing, ", "))
	}
	return nil
}

func run(databaseURL string, code int, name string, users userList, fiscal fiscalConfig, dryRun bool) error {
	if fiscal.requested() {
		if err := fiscal.validate(); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("conectar na base: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("abrir a base: %w", err)
	}

	// Tudo numa transação: uma empresa sem usuário, ou um usuário sem vínculo,
	// deixaria a base num estado em que ninguém entra e nada explica o porquê.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrir transação: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	type created struct {
		email    string
		password string
	}
	var newUsers []created
	var existing []string

	for _, spec := range users {
		var id string
		err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE email = $1`, spec.email).Scan(&id)
		if err == nil {
			existing = append(existing, spec.email)
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("consultar usuário %s: %w", spec.email, err)
		}
		password, err := temporaryPassword()
		if err != nil {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("gerar hash de %s: %w", spec.email, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO users (id, name, email, password, role, is_active)
			 VALUES (gen_random_uuid(), $1, $2, $3, $4, true)`,
			spec.name, spec.email, string(hash), spec.role); err != nil {
			return fmt.Errorf("criar usuário %s: %w", spec.email, err)
		}
		newUsers = append(newUsers, created{email: spec.email, password: password})
	}

	var enterpriseID int64
	err = tx.QueryRow(ctx, `SELECT id FROM enterprise WHERE code = $1`, code).Scan(&enterpriseID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// created_by aponta para o primeiro ADMIN: a empresa passa a ter autor.
		var creator *string
		for _, spec := range users {
			if spec.role != "ADMIN" {
				continue
			}
			var id string
			if scanErr := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE email = $1`, spec.email).Scan(&id); scanErr == nil {
				creator = &id
				break
			}
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO enterprise (code, name, created_by) VALUES ($1, $2, $3::uuid) RETURNING id`,
			code, name, creator).Scan(&enterpriseID); err != nil {
			return fmt.Errorf("criar empresa: %w", err)
		}
	case err != nil:
		return fmt.Errorf("consultar empresa: %w", err)
	}

	for _, spec := range users {
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_enterprises (user_id, enterprise_id, role)
			 SELECT u.id, $1, $2 FROM users u WHERE u.email = $3
			 ON CONFLICT (user_id, enterprise_id) DO NOTHING`,
			enterpriseID, spec.role, spec.email); err != nil {
			return fmt.Errorf("vincular %s à empresa: %w", spec.email, err)
		}
	}

	fiscalSeeded := false
	if fiscal.requested() {
		var actor string
		if err := tx.QueryRow(ctx,
			`SELECT u.id::text FROM users u
			 JOIN user_enterprises ue ON ue.user_id = u.id
			 WHERE ue.enterprise_id = $1 AND ue.role = 'ADMIN' LIMIT 1`, enterpriseID).Scan(&actor); err != nil {
			return fmt.Errorf("achar o ADMIN da empresa: %w", err)
		}
		// focus_nfe_ambiente fica em homologação e sem token de propósito: a
		// emissão real só depois do certificado A1 e da homologação do cliente.
		tag, err := tx.Exec(ctx,
			`INSERT INTO fiscal_configs (
			   enterprise_id, cnpj_empresa, razao_social, ie_empresa, regime_tributario, uf_empresa,
			   focus_nfe_ambiente, logradouro, numero, bairro, municipio, codigo_municipio, cep, updated_by)
			 SELECT $1, $2, $3, $4, $5, $6, 'homologacao', $7, $8, $9, $10, $11, $12, $13::uuid
			 WHERE NOT EXISTS (SELECT 1 FROM fiscal_configs WHERE enterprise_id = $1)`,
			enterpriseID, fiscal.cnpj, fiscal.razaoSocial, nullable(fiscal.ie), fiscal.regime, fiscal.uf,
			fiscal.logradouro, fiscal.numero, fiscal.bairro, fiscal.municipio, fiscal.codigoMunicipio,
			fiscal.cep, actor)
		if err != nil {
			return fmt.Errorf("criar configuração fiscal: %w", err)
		}
		fiscalSeeded = tag.RowsAffected() > 0
	}

	if dryRun {
		fmt.Println("seed-tenant: -dry-run pedido; desfazendo tudo")
		return nil
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar transação: %w", err)
	}

	fmt.Printf("seed-tenant: empresa %q (código %d, id %d) pronta\n", name, code, enterpriseID)
	if len(existing) > 0 {
		fmt.Printf("  já existiam, senha intacta: %s\n", strings.Join(existing, ", "))
	}
	if fiscalSeeded {
		fmt.Printf("  configuração fiscal criada para o CNPJ %s, em homologação e sem token\n", fiscal.cnpj)
	} else if fiscal.requested() {
		fmt.Println("  configuração fiscal já existia — preservada")
	}
	if len(newUsers) > 0 {
		fmt.Println()
		fmt.Println("  Senhas temporárias — anote agora, não são recuperáveis depois:")
		for _, u := range newUsers {
			fmt.Printf("    %-45s %s\n", u.email, u.password)
		}
		fmt.Println()
		fmt.Println("  Cada pessoa deve trocar a senha no primeiro acesso.")
	}
	return nil
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func digitsOnly(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// temporaryPassword devolve algo forte e ditável por telefone: base32 sem
// padding não tem caracteres que se confundam ao ler em voz alta.
func temporaryPassword() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("sortear senha temporária: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}
