//go:build integration

// Testes dos filtros da carteira (contas a pagar e a receber).
//
// Existem porque o filtro estava quebrado de um jeito que nenhum teste pegava: o
// handler lia os campos do CORPO de uma rota GET, então nada chegava e a consulta
// devolvia a carteira inteira — sem erro, com o filtro marcado na tela. O defeito
// mais perigoso deste módulo não é o que falha, é o que responde a pergunta
// errada com cara de certo.
//
// Cada teste tem controle: monta títulos que DEVEM sair e títulos que DEVEM ficar,
// e cobra os dois lados. Um filtro que devolve tudo e um que devolve nada passam
// por metade dos testes ingênuos.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run Filtro ./internal/infrastructure/repository/financial/
package financial_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	financialrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type carteira struct {
	repo domrepo.FinancialRepository
	ctx  context.Context
	t    *testing.T
}

func montarCarteira(t *testing.T) *carteira {
	t.Helper()
	pool := testutil.Pool(t)
	repo := financialrepo.NewFinancialRepositoryPG(pool)
	ctx := testutil.TenantContext(t, pool)
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM contas_pagar WHERE numero_documento LIKE 'FLT-%'")
		testutil.Exec(t, pool, "DELETE FROM contas_receber WHERE numero_documento LIKE 'FLT-%'")
	})
	return &carteira{repo: repo, ctx: ctx, t: t}
}

// empresaPropria cria uma empresa nova e devolve a sessão dela. testutil.TenantContext
// devolve sempre a PRIMEIRA empresa do banco, então dois contextos dele são a mesma
// empresa — e um teste de vazamento com eles passa sem provar nada.
func empresaPropria(t *testing.T, pool *pgxpool.Pool) context.Context {
	t.Helper()
	ator := testutil.Actor(t, pool)
	// enterprise.code é INTEGER: a faixa cabe em int32 e não colide com cadastro real.
	code := 880_000_000 + (testutil.UniqueCode() % 100_000_000)
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO enterprise (code, name, created_by) VALUES ($1, 'FILTRO VIZINHA', $2) RETURNING id`,
		code, ator).Scan(&id); err != nil {
		t.Fatalf("criando empresa vizinha: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, `DELETE FROM enterprise WHERE id = $1`, id) })
	return context.WithValue(context.Background(), contextkey.UserKey,
		&security.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: id, EnterpriseCode: code})
}

func dia(ano int, mes time.Month, d int) time.Time {
	return time.Date(ano, mes, d, 0, 0, 0, 0, time.UTC)
}

// pagar grava um título a pagar com as dimensões que os filtros atacam.
func (c *carteira) pagar(doc string, emissao, vencimento time.Time, valor int64, status string, fornecedor int64) *entity.ContaPagar {
	c.t.Helper()
	user := testutil.Actor(c.t, testutil.Pool(c.t))
	cp, err := c.repo.CreateContaPagar(c.ctx, &entity.ContaPagar{
		NumeroDocumento: doc,
		TipoDocumento:   "NF-e",
		FornecedorID:    &fornecedor,
		DataLancamento:  emissao,
		DataEmissao:     emissao,
		DataVencimento:  vencimento,
		ValorBruto:      decimal.NewFromInt(valor),
		Desconto:        decimal.Zero,
		Juros:           decimal.Zero,
		Multa:           decimal.Zero,
		ValorPago:       decimal.Zero,
		ParcelaNumero:   1,
		ParcelaTotal:    1,
		Status:          entity.ContaPagarStatus(status),
		StatusAprovacao: entity.AprovacaoPendente,
		IsActive:        true,
		CriadoPor:       user,
	})
	if err != nil {
		c.t.Fatalf("criando conta a pagar %s: %v", doc, err)
	}
	return cp
}

// docs devolve os números de documento devolvidos pela consulta, para a asserção
// dizer QUAL título veio errado, não só quantos.
func docsPagar(t *testing.T, rows []*entity.ContaPagar) []string {
	t.Helper()
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.NumeroDocumento)
	}
	return out
}

func contem(lista []string, valor string) bool {
	for _, v := range lista {
		if v == valor {
			return true
		}
	}
	return false
}

// exige que a consulta traga exatamente os documentos esperados entre os FLT-*.
func (c *carteira) exigePagar(rotulo string, f domrepo.CPFilter, esperados ...string) {
	c.t.Helper()
	rows, err := c.repo.ListContasPagar(c.ctx, f)
	if err != nil {
		c.t.Fatalf("%s: %v", rotulo, err)
	}
	vieram := make([]string, 0)
	for _, d := range docsPagar(c.t, rows) {
		if len(d) > 4 && d[:4] == "FLT-" {
			vieram = append(vieram, d)
		}
	}
	for _, e := range esperados {
		if !contem(vieram, e) {
			c.t.Fatalf("%s: faltou %s no resultado (veio %v)", rotulo, e, vieram)
		}
	}
	if len(vieram) != len(esperados) {
		c.t.Fatalf("%s: veio %v, esperado exatamente %v", rotulo, vieram, esperados)
	}
}

func TestFiltroDeContasPagarRecortaPorCadaDimensao(t *testing.T) {
	c := montarCarteira(t)
	hoje := time.Now()

	// Datas relativas a hoje: com datas fixas o teste envelhece e "a vencer" vira
	// "vencido" sem ninguém mexer no código.
	emissaoA, vencA := hoje.AddDate(0, 0, -40), hoje.AddDate(0, 0, 30)
	emissaoB, vencB := hoje.AddDate(0, 0, -10), hoje.AddDate(0, 0, 60)
	c.pagar("FLT-A", emissaoA, vencA, 100, "PENDENTE", 4001)
	c.pagar("FLT-B", emissaoB, vencB, 5000, "PENDENTE", 4002)
	c.pagar("FLT-C", emissaoB, hoje.AddDate(0, 0, 90), 900, "PAGO", 4001)
	// Vencido de verdade: em aberto, com vencimento no passado.
	c.pagar("FLT-VENCIDO", hoje.AddDate(0, -3, 0), hoje.AddDate(0, 0, -20), 300, "PENDENTE", 4003)
	// Vencimento no passado mas já pago: NÃO é vencido. É o par de controle que
	// impede o filtro de "vencidos" virar "tudo que é antigo".
	c.pagar("FLT-PAGO-ANTIGO", hoje.AddDate(0, -3, 0), hoje.AddDate(0, 0, -20), 300, "PAGO", 4003)

	todos := []string{"FLT-A", "FLT-B", "FLT-C", "FLT-VENCIDO", "FLT-PAGO-ANTIGO"}
	c.exigePagar("sem filtro", domrepo.CPFilter{}, todos...)

	// Minúsculas de propósito: a coluna grava MAIÚSCULAS e a tela manda minúsculas.
	// Sem normalizar caixa, este filtro não casa com nada e ninguém percebe.
	c.exigePagar("status pendente (minúsculo)", domrepo.CPFilter{Status: ptr("pendente")}, "FLT-A", "FLT-B", "FLT-VENCIDO")
	c.exigePagar("status PENDENTE (maiúsculo)", domrepo.CPFilter{Status: ptr("PENDENTE")}, "FLT-A", "FLT-B", "FLT-VENCIDO")
	c.exigePagar("status pago", domrepo.CPFilter{Status: ptr("pago")}, "FLT-C", "FLT-PAGO-ANTIGO")

	fornecedor := int64(4001)
	c.exigePagar("fornecedor 4001", domrepo.CPFilter{FornecedorID: &fornecedor}, "FLT-A", "FLT-C")

	// O MESMO período por EMISSÃO e por VENCIMENTO devolve conjuntos diferentes —
	// é isso que prova que date_field faz algo, e não que a consulta ignora o campo.
	ini, fim := hoje.AddDate(0, 0, -15), hoje.AddDate(0, 0, -5)
	c.exigePagar("emitidos nos últimos 15 dias", domrepo.CPFilter{StartDate: &ini, EndDate: &fim, DateField: domrepo.DateFieldEmissao}, "FLT-B", "FLT-C")
	c.exigePagar("vencendo nos últimos 15 dias", domrepo.CPFilter{StartDate: &ini, EndDate: &fim})

	c.exigePagar("documento por trecho, sem caixa", domrepo.CPFilter{Documento: ptr("flt-a")}, "FLT-A")

	minimo, maximo := 200.0, 1000.0
	c.exigePagar("valor entre 200 e 1000", domrepo.CPFilter{ValorMinimo: &minimo, ValorMaximo: &maximo}, "FLT-C", "FLT-VENCIDO", "FLT-PAGO-ANTIGO")
	fornecedor2 := int64(4001)
	_ = fornecedor2

	c.exigePagar("somente vencidos", domrepo.CPFilter{SomenteVencidos: true}, "FLT-VENCIDO")

	// Dimensões combinadas: um filtro que ignora o segundo campo passa nos testes
	// de cima e falha aqui.
	c.exigePagar("vencidos do fornecedor 4003", domrepo.CPFilter{SomenteVencidos: true, FornecedorID: ptr(int64(4003))}, "FLT-VENCIDO")
	c.exigePagar("vencidos do fornecedor 4001", domrepo.CPFilter{SomenteVencidos: true, FornecedorID: &fornecedor})
}

func TestFiltroDeContasReceberRecortaPorCadaDimensao(t *testing.T) {
	c := montarCarteira(t)
	pool := testutil.Pool(t)
	user := testutil.Actor(t, pool)
	hoje := time.Now()

	criar := func(doc string, emissao, vencimento time.Time, valor int64, status string, cliente int64) {
		t.Helper()
		if _, err := c.repo.CreateContaReceber(c.ctx, &entity.ContaReceber{
			NumeroDocumento: &doc,
			ClienteID:       &cliente,
			DataLancamento:  emissao,
			DataEmissao:     emissao,
			DataVencimento:  vencimento,
			ValorBruto:      decimal.NewFromInt(valor),
			Desconto:        decimal.Zero,
			Juros:           decimal.Zero,
			Multa:           decimal.Zero,
			ValorRecebido:   decimal.Zero,
			ParcelaNumero:   1,
			ParcelaTotal:    1,
			Status:          entity.ContaReceberStatus(status),
			IsActive:        true,
			CriadoPor:       user,
		}); err != nil {
			t.Fatalf("criando conta a receber %s: %v", doc, err)
		}
	}
	criar("FLT-R1", hoje.AddDate(0, 0, -30), hoje.AddDate(0, 0, 20), 700, "PENDENTE", 9001)
	criar("FLT-R2", hoje.AddDate(0, 0, -8), hoje.AddDate(0, 0, 45), 2500, "PENDENTE", 9002)
	criar("FLT-R3", hoje.AddDate(0, -2, 0), hoje.AddDate(0, 0, -15), 400, "PENDENTE", 9001)
	// Vencimento no passado, já recebido: NÃO é vencido. Par de controle.
	criar("FLT-R4", hoje.AddDate(0, -2, 0), hoje.AddDate(0, 0, -15), 400, "RECEBIDO", 9001)

	exige := func(rotulo string, f domrepo.CRFilter, esperados ...string) {
		t.Helper()
		rows, err := c.repo.ListContasReceber(c.ctx, f)
		if err != nil {
			t.Fatalf("%s: %v", rotulo, err)
		}
		vieram := make([]string, 0)
		for _, r := range rows {
			doc := ""
			if r.NumeroDocumento != nil {
				doc = *r.NumeroDocumento
			}
			if len(doc) > 4 && doc[:4] == "FLT-" {
				vieram = append(vieram, doc)
			}
		}
		for _, e := range esperados {
			if !contem(vieram, e) {
				t.Fatalf("%s: faltou %s (veio %v)", rotulo, e, vieram)
			}
		}
		if len(vieram) != len(esperados) {
			t.Fatalf("%s: veio %v, esperado exatamente %v", rotulo, vieram, esperados)
		}
	}

	exige("sem filtro", domrepo.CRFilter{}, "FLT-R1", "FLT-R2", "FLT-R3", "FLT-R4")
	exige("cliente 9001", domrepo.CRFilter{ClienteID: ptr(int64(9001))}, "FLT-R1", "FLT-R3", "FLT-R4")
	exige("somente vencidos", domrepo.CRFilter{SomenteVencidos: true}, "FLT-R3")
	exige("documento por trecho", domrepo.CRFilter{Documento: ptr("r2")}, "FLT-R2")
	exige("valor acima de 1000", domrepo.CRFilter{ValorMinimo: ptrF(1000)}, "FLT-R2")

	ini, fim := hoje.AddDate(0, 0, -12), hoje.AddDate(0, 0, -2)
	exige("emitidos nos últimos 12 dias", domrepo.CRFilter{StartDate: &ini, EndDate: &fim, DateField: domrepo.DateFieldEmissao}, "FLT-R2")
	exige("vencendo nos últimos 12 dias", domrepo.CRFilter{StartDate: &ini, EndDate: &fim})
	exige("status recebido (minúsculo)", domrepo.CRFilter{Status: ptr("recebido")}, "FLT-R4")
}

// TestFiltroNaoVazaEntreEmpresas: o filtro é WHERE adicional; se ele substituísse
// a cláusula de empresa em vez de somar, a carteira de uma empresa apareceria na
// outra — e o teste de contagem simples não veria.
func TestFiltroNaoVazaEntreEmpresas(t *testing.T) {
	pool := testutil.Pool(t)
	repo := financialrepo.NewFinancialRepositoryPG(pool)
	dona := testutil.TenantContext(t, pool)
	vizinha := empresaPropria(t, pool)
	user := testutil.Actor(t, pool)
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM contas_pagar WHERE numero_documento LIKE 'VAZA-%'") })

	if _, err := repo.CreateContaPagar(dona, &entity.ContaPagar{
		NumeroDocumento: "VAZA-1", TipoDocumento: "NF-e", FornecedorID: ptr(int64(777)),
		DataLancamento: time.Now(), DataEmissao: time.Now(), DataVencimento: time.Now().AddDate(0, 0, -5),
		ValorBruto: decimal.NewFromInt(1234), Desconto: decimal.Zero, Juros: decimal.Zero,
		Multa: decimal.Zero, ValorPago: decimal.Zero, ParcelaNumero: 1, ParcelaTotal: 1,
		Status: "pendente", StatusAprovacao: "pendente", IsActive: true, CriadoPor: user,
	}); err != nil {
		t.Fatalf("criando título da dona: %v", err)
	}

	// Cada recorte é tentado pela vizinha: nenhum pode revelar o título alheio.
	recortes := map[string]domrepo.CPFilter{
		"sem filtro":       {},
		"por fornecedor":   {FornecedorID: ptr(int64(777))},
		"por documento":    {Documento: ptr("VAZA")},
		"somente vencidos": {SomenteVencidos: true},
		"por valor":        {ValorMinimo: ptrF(1000), ValorMaximo: ptrF(2000)},
	}
	for rotulo, f := range recortes {
		rows, err := repo.ListContasPagar(vizinha, f)
		if err != nil {
			t.Fatalf("vizinha consultando %s: %v", rotulo, err)
		}
		for _, r := range rows {
			if r.NumeroDocumento == "VAZA-1" {
				t.Fatalf("%s: a empresa vizinha viu o título da outra", rotulo)
			}
		}
	}
	// Controle positivo: a dona encontra o próprio título por cada recorte, senão
	// o teste acima passaria com a consulta simplesmente não achando nada.
	for rotulo, f := range recortes {
		rows, err := repo.ListContasPagar(dona, f)
		if err != nil {
			t.Fatalf("dona consultando %s: %v", rotulo, err)
		}
		achou := false
		for _, r := range rows {
			if r.NumeroDocumento == "VAZA-1" {
				achou = true
			}
		}
		if !achou {
			t.Fatalf("%s: a dona não achou o próprio título", rotulo)
		}
	}
}

func ptr[T any](v T) *T       { return &v }
func ptrF(v float64) *float64 { return &v }
