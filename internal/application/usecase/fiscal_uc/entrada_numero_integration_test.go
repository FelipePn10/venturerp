//go:build integration

package fiscal_uc_test

import (
	"fmt"
	"testing"
	"time"

	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Nota sem chave só se reconhece por emitente, série e número: é isso que impede
// o lançamento manual em dobro, com título a pagar e crédito de imposto repetidos.
func TestIntegration_NotaDeEntradaPorNumero(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := testutil.TenantContext(t, pool)
	actor := testutil.Actor(t, pool)
	empresa := testutil.EnterpriseID(t, ctx)
	docs := fiscalpg.NewFiscalEntryDocumentRepositoryPG(fiscalpg.NewFiscalRepositoryPG(pool))

	u := testutil.UniqueCode()
	cnpj := fmt.Sprintf("%014d", u%100_000_000_000_000)
	numero := u%1_000_000 + 1
	hoje := time.Now()

	var id int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO fiscal_entries (enterprise_id, numero_nf, serie, modelo, data_emissao, data_entrada, cnpj_emitente, razao_social_emitente, status, created_by)
		VALUES ($1, $2, '001', '55', $3::date, $3::date, $4, 'EMITENTE DO TESTE', 'PENDING', $5)
		RETURNING id`, empresa, numero, hoje, cnpj, actor).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM fiscal_entries WHERE id = $1`, id) })

	achada, err := docs.FindEntryByNumero(ctx, cnpj, "1", numero)
	if err != nil {
		t.Fatal(err)
	}
	if achada == nil || achada.ID != id {
		t.Fatalf("série 1 e 001 são a mesma nota: esperava a entrada %d, veio %+v", id, achada)
	}

	for nome, busca := range map[string]func() (bool, error){
		"outra série":  func() (bool, error) { e, err := docs.FindEntryByNumero(ctx, cnpj, "2", numero); return e != nil, err },
		"outro número": func() (bool, error) { e, err := docs.FindEntryByNumero(ctx, cnpj, "1", numero+1); return e != nil, err },
		"outro emitente": func() (bool, error) {
			e, err := docs.FindEntryByNumero(ctx, "00000000000191", "1", numero)
			return e != nil, err
		},
	} {
		if achou, err := busca(); err != nil || achou {
			t.Errorf("%s não é a mesma nota (achou=%v, err=%v)", nome, achou, err)
		}
	}

	// Cancelada pode ser lançada de novo.
	if _, err := pool.Exec(ctx, `UPDATE fiscal_entries SET status = 'CANCELLED' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if e, err := docs.FindEntryByNumero(ctx, cnpj, "1", numero); err != nil || e != nil {
		t.Fatalf("nota cancelada não bloqueia novo lançamento (achada=%+v, err=%v)", e, err)
	}

	// Outra empresa não enxerga a nota.
	if _, err := pool.Exec(ctx, `UPDATE fiscal_entries SET status = 'PENDING' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE fiscal_entries SET enterprise_id = NULL WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if e, err := docs.FindEntryByNumero(ctx, cnpj, "1", numero); err != nil || e != nil {
		t.Fatalf("nota fora da empresa não pode aparecer (achada=%+v, err=%v)", e, err)
	}
}
