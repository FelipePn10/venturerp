//go:build integration

package serie_uc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/serie_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	item_ "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	seriepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/serie"
	tapg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// A garantia sai do perfil e da base instalada: série pela instalação, peça
// sem série pela NF de saída do cliente, garantia estendida, e o item do
// chamado gravando o que decidiu.
func TestGarantiaPelaSeriePelaNotaENoChamado(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	equip, peca := 96_000_000+u%1_000_000, 97_000_000+u%1_000_000
	cliente := 800_000 + u%90_000
	hoje := time.Now()
	a, _ := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM technical_assistance_calls WHERE enterprise_code=(SELECT code FROM enterprise WHERE id=$1)`,
				`DELETE FROM fiscal_exits WHERE enterprise_id=$1`, `DELETE FROM numeros_serie WHERE enterprise_id=$1`,
				`DELETE FROM item_pos_venda WHERE enterprise_id=$1`, `DELETE FROM customers WHERE enterprise_id=$1`,
				`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Acabado',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("GAR-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	for _, it := range []struct {
		code int64
		nome string
	}{{equip, "BALANÇA"}, {peca, "ENGRENAGEM SOB ENCOMENDA"}} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,commercial_warranty_days) VALUES($1,($1::bigint)::text,$5,$2,$3,$4,0)`, it.code, actor, a.ID, it.nome, almox)
	}
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE GARANTIA','11444777000161',$2,$3)`, cliente, actor, a.ID)

	repo := seriepg.New(pool)
	uc := &serie_uc.UseCase{Repo: repo, Auth: &auth.AuthService{}, Items: item_.NewRepositoryItemSQLC(sqlc.New(pool))}
	txt := func(c int64) string { return fmt.Sprint(c) }

	// Equipamento: 12 meses a partir da instalação, cobre peças e mão de obra.
	if _, err := uc.SalvarPerfil(a.Ctx, txt(equip), serie_uc.PerfilDTO{ControlaSerie: true, GarantiaMeses: 12, GarantiaInicio: "INSTALACAO", CobrePecas: true, CobreMaoObra: true}); err != nil {
		t.Fatal(err)
	}
	venda := hoje.AddDate(0, 0, -400).Format("2006-01-02")
	inst := hoje.AddDate(0, 0, -200).Format("2006-01-02")
	ss, err := uc.Cadastrar(a.Ctx, serie_uc.CadastroDTO{ItemCode: request.TextCode(txt(equip)), Series: []string{"G-1"}, Situacao: "NO_CLIENTE",
		CustomerCode: &cliente, DataVenda: &venda, DataInstalacao: &inst})
	if err != nil {
		t.Fatal(err)
	}
	g, err := uc.ConsultarGarantia(a.Ctx, serie_uc.ConsultaGarantiaDTO{Item: txt(equip), Serie: "g-1", Cliente: &cliente})
	if err != nil || !g.EmGarantia || g.Origem != "SERIE" || !g.CobrePecas || g.CobreDeslocamento {
		t.Fatalf("série instalada há 200 dias, 12 meses da instalação: em garantia — %+v %v", g, err)
	}

	// Peça sem série: 6 meses da venda, pela NF autorizada do cliente.
	if _, err := uc.SalvarPerfil(a.Ctx, txt(peca), serie_uc.PerfilDTO{GarantiaMeses: 6, CobrePecas: true}); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,customer_code,status)
		VALUES($1,$2::date,'5102','VENDA',$3,$4,$5,'AUTHORIZED')`, 7000+u%1000, hoje.AddDate(0, 0, -100), actor, a.ID, cliente)
	nf := 7000 + u%1000
	g, err = uc.ConsultarGarantia(a.Ctx, serie_uc.ConsultaGarantiaDTO{Item: txt(peca), Cliente: &cliente, NumeroNF: &nf})
	if err != nil || !g.EmGarantia || g.Origem != "NF" || g.Nota == nil {
		t.Fatalf("NF de 100 dias, 6 meses: em garantia — %+v %v", g, err)
	}
	outra := nf + 1
	if g, _ = uc.ConsultarGarantia(a.Ctx, serie_uc.ConsultaGarantiaDTO{Item: txt(peca), Cliente: &cliente, NumeroNF: &outra}); g.Ate != nil || len(g.Avisos) == 0 {
		t.Fatalf("NF que não existe não dá garantia e avisa: %+v", g)
	}

	// Série vencida fica coberta pela garantia estendida.
	velha := hoje.AddDate(-3, 0, 0).Format("2006-01-02")
	sv, err := uc.Cadastrar(a.Ctx, serie_uc.CadastroDTO{ItemCode: request.TextCode(txt(equip)), Series: []string{"G-OLD"}, Situacao: "NO_CLIENTE", CustomerCode: &cliente, DataVenda: &velha})
	if err != nil {
		t.Fatal(err)
	}
	if g, _ = uc.ConsultarGarantia(a.Ctx, serie_uc.ConsultaGarantiaDTO{Item: txt(equip), Serie: "G-OLD"}); g.EmGarantia {
		t.Fatal("venda há 3 anos, sem instalação: vencida")
	}
	if _, err := uc.EstenderGarantia(a.Ctx, sv[0].ID, serie_uc.GarantiaEstendidaDTO{Ate: hoje.AddDate(0, 6, 0).Format("2006-01-02"), Motivo: "contrato de manutenção"}); err != nil {
		t.Fatal(err)
	}
	if g, _ = uc.ConsultarGarantia(a.Ctx, serie_uc.ConsultaGarantiaDTO{Item: txt(equip), Serie: "G-OLD"}); !g.EmGarantia || !g.Estendida {
		t.Fatalf("com a garantia estendida: em garantia — %+v", g)
	}

	// Chamado: o item pela série grava a decisão, sem digitar dias nem NF.
	ta := &technical_assistance_uc.UseCase{Repo: tapg.New(pool), Auth: &auth.AuthService{}, Garantia: uc}
	serie := "G-1"
	call, err := ta.CreateCall(a.Ctx, request.CreateTechnicalAssistanceCallDTO{EnterpriseCode: a.Code, CustomerCode: cliente, Subject: "Balança sem leitura", Priority: "NORMAL",
		Items: []request.CreateTechnicalAssistanceCallItemDTO{{ItemCode: equip, SerialNumber: &serie, Quantity: decimal.NewFromInt(1)}}})
	if err != nil {
		t.Fatal(err)
	}
	var origem *string
	var serieID *int64
	var emGarantia bool
	if err := pool.QueryRow(bg, `SELECT garantia_origem, serie_id, in_warranty FROM technical_assistance_call_items WHERE call_code=$1`, call.Code).Scan(&origem, &serieID, &emGarantia); err != nil {
		t.Fatal(err)
	}
	if origem == nil || *origem != "SERIE" || serieID == nil || *serieID != ss[0].ID || !emGarantia {
		t.Fatalf("item do chamado: origem %v série %v garantia %v", origem, serieID, emGarantia)
	}
}
