//go:build integration

package item_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Um código digitado aponta para um item só: o código anterior e a referência do
// fabricante não podem ser o código atual de outro item, e o tanque precisa
// estar cadastrado.
func TestCodigoAnteriorNaoPodeSerCodigoAtualDeOutro(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{`DELETE FROM delivery_capacity_lines WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`} {
			testutil.Exec(t, pool, q, a.ID)
		}
	})
	ctx := context.WithValue(a.Ctx, contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: a.ID})
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	u := testutil.UniqueCode()
	alvo, outro := fmt.Sprintf("ALV-%d", u), fmt.Sprintf("OUT-%d", u)
	for i, c := range []string{alvo, outro} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,engineering_weight) VALUES($1,$2,$3,$4,$5,'Conflito','{"gross":1,"net":1,"unit":"KG"}')`, u+int64(i), c, almox, ator, a.ID)
	}
	repo := itemrepo.NewRepositoryItemSQLC(sqlc.New(pool)).ComPool(pool)
	it, err := repo.FindItemByCode(ctx, valueobject.ItemCode(u))
	if err != nil {
		t.Fatal(err)
	}
	it.PreviousCode = &outro
	if _, err := repo.UpdateFolders(ctx, it); err == nil || !strings.Contains(err.Error(), "é o código atual do item "+outro) {
		t.Fatalf("código anterior igual ao atual de outro: %v", err)
	}
	livre := fmt.Sprintf("ANT-%d", u)
	it.PreviousCode = &livre
	if _, err := repo.UpdateFolders(ctx, it); err != nil {
		t.Fatalf("código anterior livre devia passar: %v", err)
	}
	tanque := 7
	it.Planning.TankCode = &tanque
	if _, err := repo.UpdateFolders(ctx, it); err == nil || !strings.Contains(err.Error(), "não está cadastrado") {
		t.Fatalf("tanque sem cadastro: %v", err)
	}
	testutil.Exec(t, pool, `INSERT INTO delivery_capacity_lines(enterprise_id,code,description,daily_capacity) VALUES($1,7,'Pintura',100)`, a.ID)
	if _, err := repo.UpdateFolders(ctx, it); err != nil {
		t.Fatalf("tanque cadastrado devia passar: %v", err)
	}
}

// O código do fornecedor que é o código atual de outro item é recusado; o do
// próprio item, não. Validade só com controle de lote, dos dois lados.
func TestCodigoDoFornecedorEValidadeComLote(t *testing.T) {
	pool := testutil.Pool(t)
	ator := testutil.Actor(t, pool)
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, q := range []string{`DELETE FROM manufacturing_stock_item_controls WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`} {
			testutil.Exec(t, pool, q, a.ID)
		}
	})
	ctx := context.WithValue(a.Ctx, contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: a.ID})
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	u := testutil.UniqueCode()
	nosso, outro := fmt.Sprintf("VAL-%d", u), fmt.Sprintf("601-%d", u)
	for i, c := range []string{nosso, outro} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,engineering_weight) VALUES($1,$2,$3,$4,$5,'Conflito','{"gross":1,"net":1,"unit":"KG"}')`, u+int64(i), c, almox, ator, a.ID)
	}
	if err := itemrepo.ConferirCodigoDoFornecedor(ctx, pool, a.ID, u, outro); err == nil || !strings.Contains(err.Error(), "é o código atual do nosso item "+outro) {
		t.Fatalf("código do fornecedor igual ao de outro item: %v", err)
	}
	if err := itemrepo.ConferirCodigoDoFornecedor(ctx, pool, a.ID, u, nosso); err != nil {
		t.Fatalf("o código do próprio item pode: %v", err)
	}

	repo := itemrepo.NewRepositoryItemSQLC(sqlc.New(pool)).ComPool(pool)
	it, err := repo.FindItemByCode(ctx, valueobject.ItemCode(u))
	if err != nil {
		t.Fatal(err)
	}
	dias := 180
	it.Warehouse.ShelfLifeDays = &dias
	if _, err := repo.UpdateFolders(ctx, it); err == nil || !strings.Contains(err.Error(), "ligue o controle de lote") {
		t.Fatalf("validade sem lote: %v", err)
	}
	testutil.Exec(t, pool, `INSERT INTO manufacturing_stock_item_controls(enterprise_id,item_code,stock_uom,controls_lot,controls_address,lot_issue_rule) VALUES($1,$2,'UN',TRUE,FALSE,'FEFO')`, a.ID, u)
	if _, err := repo.UpdateFolders(ctx, it); err != nil {
		t.Fatalf("com lote a validade passa: %v", err)
	}
}
