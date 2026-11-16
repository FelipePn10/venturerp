//go:build integration

package fiscal_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// A nota guarda a foto da transportadora e do local de entrega, e a foto da
// transportadora sai do fornecedor dela, só na empresa dona.
func TestIntegration_FiscalExit_TransporteEEntrega(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	repo := fiscalrepo.NewFiscalRepositoryPG(pool)
	cadastro := fiscalrepo.NewCadastroTributarioPG(pool)

	empresa := func(nome string) (int64, context.Context) {
		var id int64
		code := int64(1_700_000_000 + testutil.UniqueCode()%100_000_000)
		if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,$2) RETURNING id`, code, nome).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM enterprise WHERE id=$1", id) })
		return id, context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{EnterpriseID: id})
	}
	a, ctxA := empresa("Transporte A")
	_, ctxB := empresa("Transporte B")
	user := uuid.New()
	if _, err := pool.Exec(bg, `INSERT INTO users(id,name,email,password) VALUES($1,'Transp',$2,'x')`, user, user.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM users WHERE id=$1", user) })

	fornecedor := testutil.UniqueCode()
	var supplierID, carrierID int64
	if err := pool.QueryRow(bg, `INSERT INTO suppliers(code,name,document_number,state_registration,created_by,enterprise_id)
		VALUES($1,'TRANSPORTES RAPIDO LTDA','12345678000195','9087654321',$2,$3) RETURNING id`, fornecedor, user, a).Scan(&supplierID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO supplier_addresses(supplier_id,address_type,street,number,neighborhood,city,uf,is_default)
		VALUES($1,'COMERCIAL','RUA DO PORTO','50','CENTRO','CURITIBA','PR',TRUE)`, supplierID)
	if err := pool.QueryRow(bg, `INSERT INTO shipping_carriers(enterprise_id,supplier_code) VALUES($1,$2) RETURNING id`, a, fornecedor).Scan(&carrierID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, "DELETE FROM shipping_carriers WHERE id=$1", carrierID)
		testutil.Exec(t, pool, "DELETE FROM supplier_addresses WHERE supplier_id=$1", supplierID)
		testutil.Exec(t, pool, "DELETE FROM suppliers WHERE id=$1", supplierID)
	})

	tr, err := cadastro.TransportadorDaNota(ctxA, carrierID)
	if err != nil {
		t.Fatal(err)
	}
	if tr.CNPJCPF != "12345678000195" || tr.Nome != "TRANSPORTES RAPIDO LTDA" || tr.IE != "9087654321" || tr.Endereco != "RUA DO PORTO, 50, CENTRO" || tr.Municipio != "CURITIBA" || tr.UF != "PR" {
		t.Fatalf("transportador: %+v", tr)
	}
	if _, err := cadastro.TransportadorDaNota(ctxB, carrierID); err == nil {
		t.Fatal("empresa B leu a transportadora da A")
	}

	frete := "FOB"
	exit := &entity.FiscalExit{NumeroNF: testutil.UniqueCode(), Serie: "TST", DataEmissao: time.Now(), Cfop: "5101", NaturezaOperacao: "VENDA",
		ValorProdutos: decimal.NewFromInt(100), ValorTotal: decimal.NewFromInt(100), Status: entity.ExitStatusDraft, CreatedBy: user,
		TipoFrete: &frete, ShippingCarrierID: &carrierID, Transportador: tr,
		LocalEntrega: &entity.LocalEntrega{CNPJCPF: "11222333000181", Nome: "CLIENTE", Logradouro: "RUA DA FABRICA", Numero: "100", Bairro: "DISTRITO", Municipio: "CAMPINAS", UF: "SP"}}
	created, err := repo.CreateExit(ctxA, exit)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM fiscal_exits WHERE id=$1", created.ID) })
	got, err := repo.GetExitByID(ctxA, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.TipoFrete == nil || *got.TipoFrete != "FOB" || got.ShippingCarrierID == nil || *got.ShippingCarrierID != carrierID {
		t.Fatalf("frete/transportadora: %v %v", got.TipoFrete, got.ShippingCarrierID)
	}
	if got.Transportador == nil || *got.Transportador != *tr || got.LocalEntrega == nil || got.LocalEntrega.Logradouro != "RUA DA FABRICA" {
		t.Fatalf("fotos: %+v %+v", got.Transportador, got.LocalEntrega)
	}

	// Nota sem nada disso continua lendo com os campos vazios.
	sem := &entity.FiscalExit{NumeroNF: testutil.UniqueCode(), Serie: "TST", DataEmissao: time.Now(), Cfop: "5101", NaturezaOperacao: "VENDA",
		ValorProdutos: decimal.NewFromInt(1), ValorTotal: decimal.NewFromInt(1), Status: entity.ExitStatusDraft, CreatedBy: user}
	c2, err := repo.CreateExit(ctxA, sem)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.Exec(t, pool, "DELETE FROM fiscal_exits WHERE id=$1", c2.ID) })
	g2, err := repo.GetExitByID(ctxA, c2.ID)
	if err != nil || g2.Transportador != nil || g2.LocalEntrega != nil || g2.TipoFrete != nil {
		t.Fatalf("nota sem transporte: %+v %v", g2, err)
	}
}
