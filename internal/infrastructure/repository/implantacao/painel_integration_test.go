//go:build integration

package implantacao_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/implantacao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// O placar é da empresa da sessão: o fornecedor sem endereço e o inventário
// aberto da A não pesam no painel da B.
func TestPainelDeImplantacaoPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	base := context.Background()
	ator := testutil.Actor(t, pool)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		pool.Exec(base, `DELETE FROM physical_inventories WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
		pool.Exec(base, `DELETE FROM suppliers WHERE enterprise_id = ANY($1)`, []int64{a.ID, b.ID})
	})
	cod := testutil.UniqueCode()
	testutil.Exec(t, pool, `INSERT INTO suppliers(code,name,document_type,document_number,created_by,enterprise_id,is_active) VALUES($1,'Sem endereço','CNPJ',$2,$3,$4,TRUE)`,
		cod, fmt.Sprint(cod), ator, a.ID)
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	testutil.Exec(t, pool, `INSERT INTO physical_inventories(code,description,warehouse_id,start_date,status,created_by,enterprise_id) VALUES($1,'esquecido',$2,CURRENT_DATE,'OPEN',$3,$4)`,
		testutil.UniqueCode()%1_000_000, almox, ator, a.ID)
	repo := implantacao.New(pool)
	ca, err := repo.Contagens(a.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ca["fornecedores_endereco"].Pendentes != 1 || ca["inventarios_abertos"].Pendentes != 1 {
		t.Fatalf("empresa A: %+v", ca)
	}
	cb, err := repo.Contagens(b.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cb["fornecedores_endereco"].Total != 0 || cb["inventarios_abertos"].Pendentes != 0 {
		t.Fatalf("a empresa B enxergou cadastros da A: %+v", cb)
	}
}
