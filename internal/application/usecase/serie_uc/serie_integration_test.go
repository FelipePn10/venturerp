//go:build integration

package serie_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/serie_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/serie"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	item_ "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	seriepg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/serie"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

func ip(v int64) *int64   { return &v }
func sp(v string) *string { return &v }

func TestSerieDaProducaoAteOClienteENaoMisturaEmpresas(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	item := 95_000_000 + u%1_000_000
	cliente, outroCliente := 900_000+u%90_000, 990_000+u%9_000
	itemTxt := request.TextCode(fmt.Sprint(item))
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM fiscal_exits WHERE enterprise_id=$1`,
				`DELETE FROM numeros_serie WHERE enterprise_id=$1`,
				`DELETE FROM item_pos_venda WHERE enterprise_id=$1`,
				`DELETE FROM production_orders WHERE enterprise_id=$1`,
				`DELETE FROM lot_masks WHERE enterprise_id=$1`,
				`DELETE FROM customers WHERE enterprise_id=$1`,
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
		VALUES($1,'Acabado',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("SER-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name) VALUES($1,($1::bigint)::text,$2,$3,$4,'BALANÇA RODOVIÁRIA')`, item, almox, actor, a.ID)
	for _, c := range []struct {
		code int64
		emp  int64
	}{{cliente, a.ID}, {outroCliente, a.ID}, {cliente, b.ID}} {
		testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,$2,'11444777000161',$3,$4)`,
			c.code, fmt.Sprintf("CLIENTE %d", c.code), actor, c.emp)
	}

	repo := seriepg.New(pool)
	uc := &serie_uc.UseCase{Repo: repo, Auth: &auth.AuthService{}, Items: item_.NewRepositoryItemSQLC(sqlc.New(pool))}

	// 1. Sem perfil o item não aceita série; o perfil de outra empresa não acha o item.
	if _, err := uc.Cadastrar(a.Ctx, serie_uc.CadastroDTO{ItemCode: itemTxt, Series: []string{"X1"}, Situacao: "EM_ESTOQUE", WarehouseID: &almox}); err == nil {
		t.Fatal("item sem controle de série não pode receber série")
	}
	if _, err := uc.SalvarPerfil(b.Ctx, itemTxt.String(), serie_uc.PerfilDTO{ControlaSerie: true}); err == nil {
		t.Fatal("a empresa B não pode ligar o perfil de um item da A")
	}

	// Máscara EQ-0001… na empresa A.
	var mascara int64
	if err := pool.QueryRow(bg, `INSERT INTO lot_masks(application,description,is_active,created_by,enterprise_id) VALUES('PRODUCAO','Série balança',TRUE,$1,$2) RETURNING id`, actor, a.ID).Scan(&mascara); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO lot_mask_parts(lot_mask_id,sequence,part_type,value,size) VALUES($1,1,'CARACTER','EQ-',0),($1,2,'SEQ_NUMERICA','1',4)`, mascara)
	if _, err := uc.SalvarPerfil(a.Ctx, itemTxt.String(), serie_uc.PerfilDTO{ControlaSerie: true, MascaraSerieID: &mascara}); err != nil {
		t.Fatal(err)
	}

	// 2. Cadastro manual e série repetida.
	manuais, err := uc.Cadastrar(a.Ctx, serie_uc.CadastroDTO{ItemCode: itemTxt, Series: []string{"LEG-1", "LEG-2"}, Situacao: "EM_ESTOQUE", WarehouseID: &almox})
	if err != nil || len(manuais) != 2 {
		t.Fatalf("cadastro manual: %v %v", manuais, err)
	}
	if _, err := uc.Cadastrar(a.Ctx, serie_uc.CadastroDTO{ItemCode: itemTxt, Series: []string{"LEG-1"}, Situacao: "EM_ESTOQUE", WarehouseID: &almox}); err == nil || !strings.Contains(err.Error(), "já existe") {
		t.Fatalf("série repetida deve ser recusada: %v", err)
	}
	if lista, _ := uc.Listar(b.Ctx, "", dom.Filtro{}); len(lista) != 0 {
		t.Fatalf("a empresa B não pode ver as séries da A: %d", len(lista))
	}
	if _, err := uc.Obter(b.Ctx, manuais[0].ID); err == nil {
		t.Fatal("a empresa B não pode abrir a série da A")
	}

	// 3. Entrega da OF: 3 séries pela máscara, na transação.
	var of int64
	if err := pool.QueryRow(bg, `INSERT INTO production_orders(order_number,item_code,planned_qty,created_by,enterprise_id) VALUES($1,$2,3,$3,$4) RETURNING id`, u%1_000_000, item, actor, a.ID).Scan(&of); err != nil {
		t.Fatal(err)
	}
	gerar := func(qtd int64, informadas []string) ([]string, error) {
		tx, err := pool.Begin(a.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(a.Ctx)
		s, err := seriepg.EntradaPelaProducaoTx(a.Ctx, tx, a.ID, of, item, nil, almox, sp("L1"), decimal.NewFromInt(qtd), informadas, &actor, time.Now())
		if err == nil {
			err = tx.Commit(a.Ctx)
		}
		return s, err
	}
	geradas, err := gerar(3, nil)
	if err != nil || strings.Join(geradas, ",") != "EQ-0001,EQ-0002,EQ-0003" {
		t.Fatalf("séries pela máscara: %v %v", geradas, err)
	}
	if mais, err := gerar(1, nil); err != nil || mais[0] != "EQ-0004" {
		t.Fatalf("a sequência continua: %v %v", mais, err)
	}
	if _, err := gerar(2, []string{"Z1"}); err == nil {
		t.Fatal("menos séries que unidades deve recusar a entrega")
	}

	// 4. NF de saída: escolher 2 séries numa linha de 2 unidades.
	novaNota := func(qtd int) (int64, int64) {
		var nf, linha int64
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,customer_code,status)
			VALUES($1,$2::date,'5102','VENDA',$3,$4,$5,'DRAFT') RETURNING id`, u%100_000+int64(qtd)*7, time.Now(), actor, a.ID, cliente).Scan(&nf); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(bg, `INSERT INTO fiscal_exit_items(fiscal_exit_id,sequence,item_code,cfop,quantity,unit_price,total_price) VALUES($1,1,$2,'5102',$3,1000,$4) RETURNING id`,
			nf, item, qtd, qtd*1000).Scan(&linha); err != nil {
			t.Fatal(err)
		}
		return nf, linha
	}
	porNumero := map[string]int64{}
	for _, s := range mustListar(t, uc, a.Ctx, item) {
		porNumero[s.NumeroSerie] = s.ID
	}
	nf, linha := novaNota(2)
	linhas, err := uc.LinhasDaNota(a.Ctx, nf)
	if err != nil || len(linhas) != 1 || len(dom.PendenciasDaNota(linhas)) != 1 {
		t.Fatalf("sem série escolhida a prévia impede: %+v %v", linhas, err)
	}
	if _, err := uc.DefinirSeriesDaLinha(a.Ctx, nf, linha, serie_uc.SeriesDaLinhaDTO{Series: []int64{porNumero["EQ-0001"], porNumero["EQ-0002"]}}); err != nil {
		t.Fatal(err)
	}
	linhas, _ = uc.LinhasDaNota(a.Ctx, nf)
	if p := dom.PendenciasDaNota(linhas); len(p) != 0 {
		t.Fatalf("com as séries escolhidas não há pendência: %v", p)
	}
	nf2, linha2 := novaNota(1)
	if _, err := uc.DefinirSeriesDaLinha(a.Ctx, nf2, linha2, serie_uc.SeriesDaLinhaDTO{Series: []int64{porNumero["EQ-0001"]}}); err == nil || !strings.Contains(err.Error(), "já está na nota") {
		t.Fatalf("a mesma série não pode estar em duas notas vivas: %v", err)
	}
	if _, err := uc.DefinirSeriesDaLinha(b.Ctx, nf, linha, serie_uc.SeriesDaLinhaDTO{Series: []int64{porNumero["EQ-0003"]}}); err == nil {
		t.Fatal("a empresa B não pode mexer na nota da A")
	}

	// 5. Autorização: a série vai ao cliente; instalação; transferência.
	if err := repo.SaidaPelaNota(a.Ctx, nf, ip(cliente), time.Now(), &actor); err != nil {
		t.Fatal(err)
	}
	s1, _ := uc.Obter(a.Ctx, porNumero["EQ-0001"])
	if s1.Situacao != dom.NoCliente || s1.CustomerCode == nil || *s1.CustomerCode != cliente || s1.FiscalExitID == nil || *s1.FiscalExitID != nf {
		t.Fatalf("série depois da venda: %+v", s1.Serie)
	}
	if _, err := uc.Instalar(a.Ctx, s1.ID, serie_uc.InstalacaoDTO{Data: time.Now().Format("2006-01-02"), Local: sp("Pátio de pesagem")}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Transferir(a.Ctx, s1.ID, serie_uc.TransferenciaDTO{CustomerCode: outroCliente, Motivo: "revenda do equipamento"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Transferir(a.Ctx, porNumero["EQ-0003"], serie_uc.TransferenciaDTO{CustomerCode: outroCliente, Motivo: "x"}); err == nil {
		t.Fatal("série em estoque não pode ser transferida de cliente")
	}

	// 6. Cancelamento da nota: as séries voltam ao estoque e soltam a nota.
	if err := repo.EstornoDaNota(a.Ctx, nf, &actor); err != nil {
		t.Fatal(err)
	}
	s1, _ = uc.Obter(a.Ctx, porNumero["EQ-0001"])
	if s1.Situacao != dom.EmEstoque || s1.CustomerCode != nil || s1.FiscalExitID != nil {
		t.Fatalf("série depois do estorno: %+v", s1.Serie)
	}
	eventos := []string{}
	for _, e := range s1.Historico {
		eventos = append(eventos, e.Evento)
	}
	if strings.Join(eventos, ",") != "PRODUCAO,VENDA,INSTALACAO,TRANSFERENCIA,ESTORNO_VENDA" {
		t.Fatalf("histórico: %v", eventos)
	}
	if _, err := uc.DefinirSeriesDaLinha(a.Ctx, nf2, linha2, serie_uc.SeriesDaLinhaDTO{Series: []int64{porNumero["EQ-0001"]}}); err != nil {
		t.Fatalf("depois do cancelamento a série pode ir em outra nota: %v", err)
	}

	// 7. Histórico imutável e baixa.
	if _, err := pool.Exec(bg, `UPDATE numeros_serie_eventos SET motivo='x' WHERE serie_id=$1`, s1.ID); err == nil {
		t.Fatal("o histórico da série não pode ser alterado")
	}
	if _, err := uc.Baixar(a.Ctx, porNumero["LEG-2"], "sucateada no teste"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Baixar(a.Ctx, porNumero["LEG-2"], "de novo"); err == nil {
		t.Fatal("série baixada não baixa de novo")
	}
}

func mustListar(t *testing.T, uc *serie_uc.UseCase, ctx context.Context, item int64) []dom.Serie {
	t.Helper()
	l, err := uc.Listar(ctx, fmt.Sprint(item), dom.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	return l
}
