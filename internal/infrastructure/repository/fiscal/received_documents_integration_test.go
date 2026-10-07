//go:build integration

package fiscal_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// A reserva da sincronização agendada vale entre instâncias: duas rodadas
// simultâneas nunca pegam a mesma empresa, e a empresa sincronizada há menos
// de uma hora não volta. Os prazos de manifestação usam a data do processo.
func TestIntegration_DFeReservaEPrazos(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	u := testutil.UniqueCode()
	actor := testutil.Actor(t, pool)
	var empresa int64
	code := 1_500_000_000 + u%90_000_000
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'DF-e reserva') RETURNING id`, code).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id,focus_nfe_token,focus_nfe_ambiente)
		VALUES('52454668000102','DFE TESTE','PR',$1,$2,'token-teste','homologacao')`, actor, empresa)
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_received_documents WHERE enterprise_id=$1`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, empresa)
		if _, err := pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, empresa); err != nil {
			t.Errorf("limpeza: %v", err)
		}
	})
	ctx := context.WithValue(bg, contextkey.UserKey, &security.AuthUser{ID: uuid.NewString(), Role: "ADMIN", EnterpriseID: empresa, EnterpriseCode: code})
	repo := fiscalpg.NewReceivedDocumentsRepositoryPG(fiscalpg.NewFiscalRepositoryPG(pool))

	// Dez rodadas ao mesmo tempo: a empresa sai em uma só.
	var wg sync.WaitGroup
	var mu sync.Mutex
	vezes := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es, err := repo.ReservarSincronizacaoDFe(bg, time.Hour)
			if err != nil {
				t.Error(err)
				return
			}
			for _, e := range es {
				if e.EnterpriseID == empresa {
					mu.Lock()
					vezes++
					mu.Unlock()
					if e.EnterpriseCode != code || e.Ator != actor {
						t.Errorf("reserva sem o código da empresa ou o ator: %+v", e)
					}
				}
			}
		}()
	}
	wg.Wait()
	if vezes != 1 {
		t.Fatalf("empresa reservada %d vezes, esperado 1", vezes)
	}
	es, _ := repo.ReservarSincronizacaoDFe(bg, time.Hour)
	for _, e := range es {
		if e.EnterpriseID == empresa {
			t.Fatal("empresa sincronizada agora voltou antes do intervalo")
		}
	}
	if err := repo.SetDFeAutomatico(ctx, false); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `UPDATE fiscal_configs SET dfe_ultima_tentativa = NOW() - interval '2 hours' WHERE enterprise_id=$1`, empresa)
	es, _ = repo.ReservarSincronizacaoDFe(bg, time.Hour)
	for _, e := range es {
		if e.EnterpriseID == empresa {
			t.Fatal("sincronização automática desligada e a empresa foi reservada")
		}
	}

	// Resultado registrado.
	msg := "CNPJ do emitente não autorizado"
	if err := repo.RegistrarResultadoDFe(ctx, &msg); err != nil {
		t.Fatal(err)
	}

	// Prazos: hoje fixo, datas como parâmetro.
	hoje := time.Date(2026, 10, 6, 23, 30, 0, 0, time.FixedZone("BRT", -3*3600))
	nota := func(i int, emissao string, manifestacao *string, situacao string) {
		testutil.Exec(t, pool, `INSERT INTO fiscal_received_documents(enterprise_id,chave_acesso,data_emissao,manifestacao,situacao) VALUES($1,$2,$3::date,$4,$5)`,
			empresa, fmt.Sprintf("%044d", u*10+int64(i)), emissao, manifestacao, situacao)
	}
	ciencia, confirmacao := "ciencia", "confirmacao"
	nota(1, "2026-04-20", &ciencia, "autorizada")     // vence em 11 dias → próximo
	nota(2, "2026-09-01", nil, "autorizada")          // longe
	nota(3, "2026-03-01", nil, "autorizada")          // vencido
	nota(4, "2026-04-20", &confirmacao, "autorizada") // encerrado
	nota(5, "2026-03-01", nil, "cancelada")           // cancelada
	st, err := repo.DFeStatus(ctx, hoje)
	if err != nil {
		t.Fatal(err)
	}
	if st.Automatico || st.UltimoErro == nil || *st.UltimoErro != msg || st.PrazoProximo != 1 || st.PrazoVencido != 1 {
		t.Fatalf("status: automático %v erro %v próximo %d vencido %d", st.Automatico, st.UltimoErro, st.PrazoProximo, st.PrazoVencido)
	}
	lista, err := repo.ListReceivedDocuments(ctx, repository.ReceivedDocumentsFilter{SomentePrazo: true, Hoje: hoje})
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 2 {
		t.Fatalf("filtro de prazo devolveu %d notas, esperado 2 (próxima e vencida)", len(lista))
	}
}
