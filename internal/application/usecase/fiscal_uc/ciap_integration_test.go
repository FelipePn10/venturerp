//go:build integration

package fiscal_uc_test

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// CIAP: a nota de entrada aprovada com CFOP de ativo põe o bem no CIAP; a
// apuração do mês calcula a parcela pelo índice das saídas; a nota cancelada
// tira o bem; o mês fechado não se refaz.
func TestCIAPDaNotaDeEntradaAApuracao(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	code := int(1_900_000_000 + testutil.UniqueCode()%90_000_000)
	repo := enterprise.NewRepositoryEnterpriseSQLC(sqlc.New(pool)).ComPool(pool)
	e, err := mapper.ToEnterpriseEntity(request.CreateEnterpriseDTO{Code: code, CreatedBy: actor, CNPJ: "11222333000181",
		RazaoSocial: "CIAP TESTE LTDA", RegimeTributario: "3", UF: "PR", CodigoMunicipio: "4106902"})
	if err != nil {
		t.Fatal(err)
	}
	criada, err := repo.Create(bg, e)
	if err != nil {
		t.Fatal(err)
	}
	emp := int64(criada.ID)
	t.Cleanup(func() { testutil.ApagarEmpresas(t, pool, emp) })
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM ciap_apuracoes WHERE enterprise_id=$1`, `DELETE FROM ciap_bens WHERE enterprise_id=$1`, `DELETE FROM ciap_parametros WHERE enterprise_id=$1`,
			`DELETE FROM accounting_accounts WHERE plan_id IN (SELECT id FROM accounting_plans WHERE enterprise_id=$1) AND parent_id IS NOT NULL`,
			`DELETE FROM accounting_accounts WHERE plan_id IN (SELECT id FROM accounting_plans WHERE enterprise_id=$1)`, `DELETE FROM accounting_plans WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_entry_items WHERE fiscal_entry_id IN (SELECT id FROM fiscal_entries WHERE enterprise_id=$1)`, `DELETE FROM fiscal_entries WHERE enterprise_id=$1`,
			`DELETE FROM fiscal_configs WHERE enterprise_id=$1`, `DELETE FROM user_enterprises WHERE enterprise_id=$1`, `DELETE FROM enterprise WHERE id=$1`,
		} {
			_, _ = pool.Exec(bg, q, emp)
		}
	})
	ctx := context.WithValue(bg, contextkey.UserKey, &security.AuthUser{ID: actor.String(), Role: "ADMIN", EnterpriseID: emp, EnterpriseCode: int64(code)})

	// Nota de entrada aprovada em 03/2026: um torno (1551, ICMS 4.800) e um
	// material de consumo (1556), que não entra.
	var nota int64
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_entries (numero_nf, serie, modelo, data_emissao, data_entrada, cnpj_emitente, razao_social_emitente, status, created_by, enterprise_id)
		VALUES (5501, '1', '55', '2026-03-02', '2026-03-03', '44555666000190', 'FORNECEDOR DO TORNO', 'APPROVED', $1, $2) RETURNING id`, actor, emp).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_entry_items (fiscal_entry_id, sequence, cfop, cfop_entrada, quantity, unit_price, total_price, valor_icms, description, uom)
		VALUES ($1, 1, '5101', '1551', 1, 40000, 40000, 4800, 'TORNO CNC', 'UN'), ($1, 2, '5102', '1556', 10, 10, 100, 12, 'LUVA', 'PC')`, nota)
	// Saídas de 04/2026: 75.000 tributadas, 25.000 sem ICMS.
	var saida int64
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_exits (numero_nf, data_emissao, cfop, natureza_operacao, status, tipo_documento, created_by, enterprise_id)
		VALUES (900, '2026-04-10', '5101', 'VENDA', 'AUTHORIZED', 1, $1, $2) RETURNING id`, actor, emp).Scan(&saida); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_exit_items (fiscal_exit_id, cfop, quantity, unit_price, total_price, valor_icms)
		VALUES ($1, '5101', 1, 75000, 75000, 9000), ($1, '5949', 1, 25000, 25000, 0)`, saida)

	uc := &fiscal_uc.CIAPUseCase{Repo: fiscal.NewCIAPPG(pool), Fiscal: fiscal.NewFiscalRepositoryPG(pool), Auth: &auth.AuthService{}}
	if _, err := uc.SalvarParametros(ctx, fiscal_uc.CIAPParametrosDTO{CodigoAjuste: "SP020001"}); err == nil || !strings.Contains(err.Error(), "outra UF") {
		t.Fatalf("código de ajuste de outra UF: %v", err)
	}
	if _, err := uc.SalvarParametros(ctx, fiscal_uc.CIAPParametrosDTO{CodigoAjuste: "PR000001"}); err == nil {
		t.Fatal("código de outros débitos não serve para o crédito do CIAP")
	}
	// Conta do ativo: analítica, filha de um grupo (nível 2 no 0500).
	var plano, grupo, conta int64
	if err := pool.QueryRow(bg, `INSERT INTO accounting_plans (plan_number, description, valid_from, status, enterprise_id) VALUES (1, 'PLANO', '2020-01-01', 'A', $1) RETURNING id`, emp).Scan(&plano); err != nil {
		t.Fatal(err)
	}
	_ = pool.QueryRow(bg, `INSERT INTO accounting_accounts (plan_id, account_number, description, nature_code, valid_from, is_analytic) VALUES ($1,'1.2','IMOBILIZADO','01','2020-01-01',FALSE) RETURNING id`, plano).Scan(&grupo)
	_ = pool.QueryRow(bg, `INSERT INTO accounting_accounts (plan_id, parent_id, account_number, description, nature_code, valid_from, is_analytic) VALUES ($1,$2,'1.2.01','MAQUINAS','01','2020-01-01',TRUE) RETURNING id`, plano, grupo).Scan(&conta)
	if _, err := uc.SalvarParametros(ctx, fiscal_uc.CIAPParametrosDTO{CodigoAjuste: "PR020001", ContaAtivoID: &grupo}); err == nil {
		t.Fatal("conta sintética não serve")
	}
	if _, err := uc.SalvarParametros(ctx, fiscal_uc.CIAPParametrosDTO{CodigoAjuste: "PR020001", NumeroParcelas: 48, ContaAtivoID: &conta}); err != nil {
		t.Fatal(err)
	}

	bens, err := uc.Bens(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(bens) != 1 || bens[0].Descricao != "TORNO CNC" || !bens[0].Total.Equal(decimal.NewFromInt(4800)) || bens[0].Origem != "NF_ENTRADA" {
		t.Fatalf("o torno entra no CIAP, a luva não: %+v", bens)
	}
	if b2, _ := uc.Bens(ctx); len(b2) != 1 {
		t.Fatal("a sincronização não duplica")
	}
	if bens[0].Conta != "1.2.01 — MAQUINAS" {
		t.Fatalf("o bem da nota leva a conta padrão: %q", bens[0].Conta)
	}
	cad, _ := fiscal.NewCIAPPG(pool).Bem(ctx, bens[0].ID)
	if cad.ContaNivel != 2 || cad.Documento == nil || cad.Documento.CodPart != "44555666000190" {
		t.Fatalf("nível da conta e documento do bem: %+v", cad)
	}

	// Abril: parcela 2 (100,00) × índice 0,75 = 75,00.
	prev, err := uc.Previa(ctx, fiscal_uc.CIAPApurarDTO{Ano: 2026, Mes: 4})
	if err != nil {
		t.Fatal(err)
	}
	if !prev.Indice.Equal(decimal.RequireFromString("0.75")) || !prev.ICMSApropriado.Equal(decimal.NewFromInt(75)) || prev.Gravada {
		t.Fatalf("prévia de abril: %+v", prev)
	}
	a, err := uc.Apurar(ctx, fiscal_uc.CIAPApurarDTO{Ano: 2026, Mes: 4})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Gravada || len(a.Movimentos) != 1 || a.Movimentos[0].TipoMov != "SI" || a.Movimentos[0].NumeroParcela != 2 {
		t.Fatalf("apuração gravada: %+v", a)
	}
	// Saídas informadas (a UF exclui operações): o índice segue o informado.
	a, err = uc.Apurar(ctx, fiscal_uc.CIAPApurarDTO{Ano: 2026, Mes: 4, SaidasTribExp: "75.000,00", SaidasTotal: "80.000,00"})
	if err != nil || !a.SaidasInformadas || !a.ICMSApropriado.Equal(decimal.RequireFromString("93.75")) {
		t.Fatalf("com as saídas informadas: %v %+v", err, a)
	}
	if err := uc.Fechar(ctx, 2026, 4, true); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Apurar(ctx, fiscal_uc.CIAPApurarDTO{Ano: 2026, Mes: 4}); err == nil {
		t.Fatal("o mês fechado não se refaz")
	}

	// A nota cancelada depois: com parcela apropriada, o bem fica baixado (OT).
	testutil.Exec(t, pool, `UPDATE fiscal_entries SET status='CANCELLED' WHERE id=$1`, nota)
	bens, _ = uc.Bens(ctx)
	if len(bens) != 1 || bens[0].Situacao != "BAIXADO" || bens[0].TipoBaixa != "OT" {
		t.Fatalf("bem da nota cancelada: %+v", bens)
	}

	// Bem manual (anterior ao sistema): entra e, sem parcela, sai.
	m, err := uc.CriarBem(ctx, fiscal_uc.CIAPBemDTO{Codigo: "PRENSA-01", Descricao: "Prensa", DataEntrada: "2024-01-15", ICMSOp: "960,00"})
	if err != nil {
		t.Fatal(err)
	}
	if !m.ValorParcela.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("parcela da prensa: %+v", m)
	}
	if err := uc.ExcluirBem(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
}
