//go:build integration

package fiscal_uc_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	infraauth "github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	fiscalpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// A nota de saída tributa cada linha pela regra do cadastro de parâmetros de
// ICMS/IPI (VFIS0320): o mesmo NCM sai com uma alíquota para o revendedor
// (contribuinte) e outra para o consumidor final; o item com isenção sai com
// CST 40 e o cBenef da regra; o NCM sem regra segue a regra geral da empresa
// (19,5% com diferimento de 38,46%, CST 51). A linha guarda a regra aplicada.
func TestIntegration_NotaDeSaidaUsaParametroICMSIPI(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	ator := testutil.Actor(t, pool)
	code := 1_975_000_000 + testutil.UniqueCode()%10_000_000
	var emp int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Parâmetro ICMS/IPI') RETURNING id`, code).Scan(&emp); err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(bg, contextkey.UserKey, &appsecurity.AuthUser{ID: ator.String(), Role: "ADMIN", EnterpriseID: emp, EnterpriseCode: code})
	testutil.Exec(t, pool, `INSERT INTO fiscal_configs(cnpj_empresa,razao_social,uf_empresa,updated_by,enterprise_id) VALUES('11222333000181','EMPRESA TESTE','PR',$1,$2)`, ator, emp)
	u := testutil.UniqueCode()
	comum, isento, geral := u, u+1, u+2
	for _, it := range []int64{comum, isento, geral} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id) VALUES($1,($1::bigint)::text,1,$2,$3)`, it, ator, emp)
	}
	t.Cleanup(func() {
		testutil.Exec(t, pool, `DELETE FROM fiscal_exit_items WHERE fiscal_exit_id IN (SELECT id FROM fiscal_exits WHERE enterprise_id=$1)`, emp)
		testutil.Exec(t, pool, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM icms_ipi_tax_params WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM items WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, emp)
		testutil.Exec(t, pool, `DELETE FROM enterprise WHERE id=$1`, emp)
	})
	var regraNCM, regraItem int64
	if err := pool.QueryRow(bg, `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,ncm_code,icms_pct_contrib,cst_icms_contrib,icms_pct_non_contrib,is_active)
		VALUES($1,'PR','SAIDA','7326.90.90',12,'00',18,true) RETURNING id`, emp).Scan(&regraNCM); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO icms_ipi_tax_params(enterprise_id,uf,operation_type,item_code,cst_icms_contrib,cst_icms_non_contrib,cod_benef_contrib,is_active)
		VALUES($1,'PR','AMBAS',$2,'40','40','PR810001',true) RETURNING id`, emp, isento).Scan(&regraItem); err != nil {
		t.Fatal(err)
	}

	fiscalRepo := fiscalpg.NewFiscalRepositoryPG(pool)
	uc := &fiscal_uc.CreateFiscalExitUseCase{Repo: fiscalRepo, Auth: &infraauth.AuthService{},
		Beneficios: fiscalpg.NewBeneficioFiscalPG(pool), Parametros: fiscalpg.NewParametroICMSIPIPG(pool)}
	linha := func(seq int, item int64, ncm string) request.CreateFiscalExitItemDTO {
		return request.CreateFiscalExitItemDTO{Sequence: seq, ItemCode: &item, Ncm: &ncm, Cfop: "5102", Quantity: 1, UnitPrice: 1000, TotalPrice: 1000, OrigemMercadoria: "0"}
	}
	uf := "PR"
	emitir := func(ie *string) []*entity.FiscalExitItem {
		t.Helper()
		res, err := uc.Execute(ctx, request.CreateFiscalExitDTO{
			Serie: "1", DataEmissao: time.Now().Format("2006-01-02"), UFDestinatario: &uf, IEDestinatario: ie,
			Cfop: "5102", NaturezaOperacao: "Venda", ValorProdutos: 3000,
			Itens: []request.CreateFiscalExitItemDTO{linha(1, comum, "73269090"), linha(2, isento, "73082000"), linha(3, geral, "73089010")},
		})
		if err != nil {
			t.Fatal(err)
		}
		itens, err := fiscalRepo.GetExitItems(ctx, res.ID)
		if err != nil || len(itens) != 3 {
			t.Fatalf("itens: %d %v", len(itens), err)
		}
		return itens
	}
	confere := func(nome string, it *entity.FiscalExitItem, cst string, aliq, base, icms, diferido float64, regra int64) {
		t.Helper()
		got := ""
		if it.CstICMS != nil {
			got = *it.CstICMS
		}
		var r int64
		if it.RegraICMSIPIID != nil {
			r = *it.RegraICMSIPIID
		}
		perto := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
		if got != cst || !perto(it.AliqICMS, aliq) || !perto(it.BaseICMS, base) || !perto(it.ValorICMS, icms) || !perto(it.ValorICMSDiferido, diferido) || r != regra {
			t.Fatalf("%s: CST %s aliq %.4f base %.2f ICMS %.2f diferido %.2f regra %d; quer CST %s aliq %.4f base %.2f ICMS %.2f diferido %.2f regra %d",
				nome, got, it.AliqICMS, it.BaseICMS, it.ValorICMS, it.ValorICMSDiferido, r, cst, aliq, base, icms, diferido, regra)
		}
	}

	ie := "9012345678"
	rev := emitir(&ie)
	confere("revenda, NCM com regra", rev[0], "00", 0.12, 1000, 120, 0, regraNCM)
	confere("revenda, item isento", rev[1], "40", 0, 0, 0, 0, regraItem)
	if rev[1].CodigoBeneficioFiscal == nil || *rev[1].CodigoBeneficioFiscal != "PR810001" {
		t.Fatalf("cBenef do item isento: %v, quer PR810001", rev[1].CodigoBeneficioFiscal)
	}
	confere("revenda, NCM sem regra (geral)", rev[2], "51", 0.195, 1000, 195, 75, 0)

	consumidor := emitir(nil)
	// Consumidor final: 18% da regra sobre a base com IPI (NCM com IPI de 5%).
	confere("consumidor final, NCM com regra", consumidor[0], "00", 0.18, 1050, 189, 0, regraNCM)
	confere("consumidor final, item isento", consumidor[1], "40", 0, 0, 0, 0, regraItem)
	confere("consumidor final, NCM sem regra", consumidor[2], "00", 0.195, 1000, 195, 0, 0)
}
