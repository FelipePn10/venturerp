//go:build integration

package financial_test

import (
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	financialrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// Os relatórios fiscais e gerenciais liam as notas de todas as empresas. Uma
// nota de entrada aprovada noutra empresa não pode aparecer no livro, na
// apuração de créditos nem nas compras do período desta — e todo relatório
// tem de rodar (a ficha técnica consultava tabelas que não existem).
func TestIntegration_RelatoriosIsolamPorEmpresa(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	repo := financialrepo.NewFinancialRepositoryPG(pool)
	minha := testutil.TenantContext(t, pool)
	actor := testutil.Actor(t, pool)

	u := testutil.UniqueCode()
	var outra int64
	if err := pool.QueryRow(bg, `INSERT INTO enterprise(code,name) VALUES($1,'Outra empresa') RETURNING id`, 1_700_000_000+u%90_000_000).Scan(&outra); err != nil {
		t.Fatal(err)
	}
	ctxOutra := context.WithValue(bg, contextkey.UserKey, &security.AuthUser{ID: uuid.NewString(), Role: "ADMIN", EnterpriseID: outra, EnterpriseCode: 1_700_000_000 + u%90_000_000})
	hoje := time.Now()
	numero := u % 1_000_000_000
	testutil.Exec(t, pool, `INSERT INTO fiscal_entries(numero_nf,serie,modelo,data_emissao,data_entrada,cnpj_emitente,razao_social_emitente,valor_produtos,valor_icms,valor_total,tipo_documento,status,created_by,enterprise_id)
		VALUES($1,'1','55',$2::date,$2::date,'11111111000191','FORNECEDOR DA OUTRA',1000,180,1000,'NFE','APPROVED',$3,$4)`, numero, hoje, actor, outra)
	// Crédito por item: 180 de ICMS com crédito e 50 de um item sem crédito (uso e consumo).
	testutil.Exec(t, pool, `INSERT INTO fiscal_entry_items(fiscal_entry_id,sequence,cfop,quantity,unit_price,total_price,valor_icms,gera_credito_icms)
		SELECT id,1,'5101',1,800,800,180,TRUE FROM fiscal_entries WHERE enterprise_id=$1 AND numero_nf=$2
		UNION ALL SELECT id,2,'5102',1,200,200,50,FALSE FROM fiscal_entries WHERE enterprise_id=$1 AND numero_nf=$2`, outra, numero)
	t.Cleanup(func() {
		_, _ = pool.Exec(bg, `DELETE FROM notification_outbox WHERE enterprise_id=$1`, outra)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_entries WHERE enterprise_id=$1`, outra)
		_, _ = pool.Exec(bg, `DELETE FROM fiscal_freight_documents WHERE enterprise_id=$1`, outra)
		if _, err := pool.Exec(bg, `DELETE FROM enterprise WHERE id=$1`, outra); err != nil {
			t.Errorf("a limpeza deixou dados da empresa de teste: %v", err)
		}
	})

	inicio, fim := hoje.AddDate(0, 0, -3), hoje.AddDate(0, 0, 3)
	contem := func(linhas []map[string]interface{}) bool {
		for _, l := range linhas {
			for _, v := range l {
				if fmt.Sprint(v) == "FORNECEDOR DA OUTRA" || fmt.Sprint(v) == fmt.Sprint(numero) {
					return true
				}
			}
		}
		return false
	}
	livro, err := repo.GetLivroEntradas(minha, inicio, fim)
	if err != nil {
		t.Fatal(err)
	}
	if contem(livro) {
		t.Fatal("o livro de entradas mostra a nota de outra empresa")
	}
	if l, err := repo.GetLivroEntradas(ctxOutra, inicio, fim); err != nil || !contem(l) {
		t.Fatalf("controle: a dona da nota deveria vê-la no livro (%v)", err)
	}
	for nome, f := range map[string]func(context.Context, time.Time, time.Time) ([]map[string]interface{}, error){
		"impostos de entrada": repo.GetImpostosEntradas, "compras do período": repo.GetComprasPeriodo,
	} {
		l, err := f(minha, inicio, fim)
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		if contem(l) {
			t.Fatalf("%s mostra a nota de outra empresa", nome)
		}
	}
	creditos, err := repo.GetFiscalCredits(minha, hoje.Format("01/2006"))
	if err != nil {
		t.Fatal(err)
	}
	if outros, _ := repo.GetFiscalCredits(ctxOutra, hoje.Format("01/2006")); !outros["ICMS"].Equal(decimal.NewFromInt(180)) {
		t.Fatalf("controle: crédito de ICMS da outra empresa = %v", outros["ICMS"])
	}
	// O ICMS do CT-e de frete lançado com crédito entra na apuração (como na EFD).
	testutil.Exec(t, pool, `INSERT INTO fiscal_freight_documents(enterprise_id,numero,serie,data_emissao,cnpj_transportadora,valor_frete,base_icms,aliq_icms,valor_icms,credita_icms,data_vencimento,status,created_by,lancado_em)
		VALUES($1,1,'1',$2::date,'77888999000100',50,50,12,6,TRUE,$2::date,'LANCADO',$3,$4)`, outra, hoje, actor,
		time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 12, 0, 0, 0, time.UTC))
	if outros, _ := repo.GetFiscalCredits(ctxOutra, hoje.Format("01/2006")); !outros["ICMS"].Equal(decimal.NewFromInt(186)) {
		t.Fatalf("crédito de ICMS com o frete = %v, want 186", outros["ICMS"])
	}
	_ = creditos

	// Todos os relatórios executam (SQL válido) no contexto da empresa.
	for nome, f := range map[string]func(context.Context, time.Time, time.Time) ([]map[string]interface{}, error){
		"livro de saídas": repo.GetLivroSaidas, "impostos de saída": repo.GetImpostosSaidas,
		"produtos vendidos": repo.GetProdutosVendidos, "produtos produzidos": repo.GetProdutosProduzidos,
		"histórico de custos": repo.GetHistoricoCustos, "curva ABC clientes": repo.GetCurvaABCClientes,
		"curva ABC produtos": repo.GetCurvaABCProdutos,
	} {
		if _, err := f(minha, inicio, fim); err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
	}
	if _, err := repo.GetDRE(minha, inicio, fim); err != nil {
		t.Fatalf("DRE: %v", err)
	}
	if _, err := repo.GetDREComCMV(minha, inicio, fim); err != nil {
		t.Fatalf("DRE com CMV: %v", err)
	}
	if _, err := repo.GetFiscalDebits(minha, hoje.Format("01/2006")); err != nil {
		t.Fatalf("débitos fiscais: %v", err)
	}
	if _, err := repo.GetFichaTecnicaCusto(minha, u); err != nil {
		t.Fatalf("ficha técnica: %v", err)
	}
}
