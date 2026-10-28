//go:build integration

package item_governanca_uc_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_governanca_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	item_ "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	govpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/itemgovernanca"
	notificationrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/notification"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// areas: a permissão de aprovar por módulo, decidida pelo teste.
type areas map[acesso.Modulo]bool

func (a areas) PodeNoModulo(_ context.Context, m acesso.Modulo, _ acesso.Acao) (bool, error) {
	return a[m], nil
}

// Item novo nasce em cadastro quando a empresa exige liberação, cada área só
// libera a sua visão completa, o item só entra em documento depois de
// liberado, descontinuado aponta o substituto, e tudo fica no histórico.
// Outra empresa não vê nada.
func TestGovernancaDoCadastroDeItem(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	fab, comp, pai := 92_000_000+u%1_000_000, 92_500_000+u%1_000_000, 93_500_000+u%1_000_000
	cliente := 870_000 + u%90_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM notification_alerts WHERE enterprise_id=$1`, `DELETE FROM notification_outbox WHERE enterprise_id=$1`,
				`DELETE FROM enterprise_notification_subscriptions WHERE enterprise_id=$1`, `DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1`,
				`DELETE FROM suppliers WHERE enterprise_id=$1`,
				`DELETE FROM fiscal_exits WHERE enterprise_id=$1`, `DELETE FROM item_structures WHERE parent_code IN (SELECT code FROM items WHERE enterprise_id=$1)`,
				`DELETE FROM item_governanca_parametros WHERE enterprise_id=$1`, `DELETE FROM item_regras_campo WHERE enterprise_id=$1`,
				`DELETE FROM customers WHERE enterprise_id=$1`, `DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM fiscal_classifications WHERE enterprise_id=$1`, `DELETE FROM warehouse WHERE enterprise_id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	var almox int64
	if err := pool.QueryRow(bg, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Geral',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("GOV-%d", u), actor, a.ID).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	cod := func(c int64) string { return fmt.Sprintf("G%d", c) }
	for _, it := range []struct {
		code int64
		nome string
		tipo int
	}{{fab, "SUPORTE DE FIXAÇÃO 3MM", dom.Fabricado}, {comp, "SUPORTE FIXACAO 3 MM", dom.Comprado}, {pai, "CONJUNTO BASE", dom.Fabricado}} {
		testutil.Exec(t, pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,engineering_type) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			it.code, cod(it.code), almox, actor, a.ID, it.nome, it.tipo)
	}
	testutil.Exec(t, pool, `INSERT INTO item_structures(parent_code,child_code,quantity,unit_of_measurement,created_by,is_active) VALUES($1,$2,1,'UN',$3,TRUE)`, pai, fab, actor)
	testutil.Exec(t, pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE GOV','11444777000161',$2,$3)`, cliente, actor, a.ID)

	permissoes := areas{acesso.Engenharia: true, acesso.Fiscal: false}
	uc := &item_governanca_uc.UseCase{Repo: govpg.New(pool), Auth: &auth.AuthService{}, Items: item_.NewRepositoryItemSQLC(sqlc.New(pool)), Areas: permissoes}

	// Sem exigir liberação, o item novo continua liberado (como antes).
	if err := uc.AoCriarItem(a.Ctx, fab); err != nil {
		t.Fatal(err)
	}
	if p, _ := uc.Painel(a.Ctx, cod(fab)); p.Status != dom.Liberado {
		t.Fatalf("sem exigir liberação: %s", p.Status)
	}
	if _, err := uc.SalvarParametros(a.Ctx, dom.Parametros{ExigeLiberacao: true, Visoes: []string{"engenharia", "FISCAL"}}); err != nil {
		t.Fatal(err)
	}
	if err := uc.AoCriarItem(a.Ctx, fab); err != nil {
		t.Fatal(err)
	}
	p, err := uc.Painel(a.Ctx, cod(fab))
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != dom.EmCadastro || p.Completo {
		t.Fatalf("item em cadastro com pendências: %+v", p.Situacao)
	}

	// Em cadastro, não entra em NF nova.
	var nota int64
	if err := pool.QueryRow(bg, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,customer_code,status)
		VALUES($1,$2::date,'5102','VENDA',$3,$4,$5,'DRAFT') RETURNING id`, 9000+u%1000, time.Now(), actor, a.ID, cliente).Scan(&nota); err != nil {
		t.Fatal(err)
	}
	linha := func(item int64) error {
		_, err := pool.Exec(bg, `INSERT INTO fiscal_exit_items(fiscal_exit_id,item_code,cfop,quantity,unit_price,total_price) VALUES($1,$2,'5102',1,10,10)`, nota, item)
		return err
	}
	if err := linha(fab); err == nil || !strings.Contains(err.Error(), "ainda está em cadastro") {
		t.Fatalf("item em cadastro não entra na NF: %v", err)
	}

	// A engenharia só libera completa; o ajuste da empresa tira estrutura e roteiro.
	if _, err := uc.Liberar(a.Ctx, cod(fab), "ENGENHARIA"); err == nil || !strings.Contains(err.Error(), "pendências") {
		t.Fatalf("engenharia com pendência: %v", err)
	}
	f := false
	for _, campo := range []string{"estrutura", "roteiro"} {
		if _, err := uc.SalvarRegra(a.Ctx, campo, &f); err != nil {
			t.Fatal(err)
		}
	}
	classif := u % 1_000_000_000
	testutil.Exec(t, pool, `INSERT INTO fiscal_classifications(code,description,ncm,enterprise_id,created_by) VALUES($1,'Peças','73269090',$2,$3)`, classif, a.ID, actor)
	testutil.Exec(t, pool, `UPDATE items SET pdm_group_code=1, pdm_description_technique='SUPORTE ACO 3MM', accounting_sale_fiscal_classification_code=$2, accounting_origin=0 WHERE code=$1`, fab, fmt.Sprint(classif))
	if _, err := uc.Liberar(a.Ctx, cod(fab), "ENGENHARIA"); err != nil {
		t.Fatal(err)
	}
	// O fiscal sem permissão de aprovar no módulo fiscal não libera.
	if _, err := uc.Liberar(a.Ctx, cod(fab), "FISCAL"); err == nil || !strings.Contains(err.Error(), "módulo fiscal") {
		t.Fatalf("liberar a visão fiscal exige o módulo fiscal: %v", err)
	}
	permissoes[acesso.Fiscal] = true
	p, err = uc.Liberar(a.Ctx, cod(fab), "FISCAL")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != dom.Liberado {
		t.Fatalf("última visão exigida liberada: item liberado — %s", p.Status)
	}
	if err := linha(fab); err != nil {
		t.Fatalf("item liberado entra na NF: %v", err)
	}

	// Descontinuado com substituto: o documento novo recusa e aponta o substituto.
	sub := request.TextCode(cod(comp))
	if _, err := uc.MudarStatus(a.Ctx, cod(fab), item_governanca_uc.StatusDTO{Status: "DESCONTINUADO", Motivo: "fora de linha", Substituto: &sub, De: "EM_CADASTRO"}); err == nil {
		t.Fatal("status esperado desatualizado: conflito")
	}
	p, err = uc.MudarStatus(a.Ctx, cod(fab), item_governanca_uc.StatusDTO{Status: "DESCONTINUADO", Motivo: "fora de linha", Substituto: &sub})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != dom.Descontinuado || p.SubstitutoCd == nil || *p.SubstitutoCd != cod(comp) {
		t.Fatalf("descontinuado com substituto: %+v", p.Situacao)
	}
	if err := linha(fab); err == nil || !strings.Contains(err.Error(), "use o item "+cod(comp)) {
		t.Fatalf("descontinuado aponta o substituto: %v", err)
	}
	// Documento já aberto segue: alterar a quantidade da linha existente passa.
	if _, err := pool.Exec(bg, `UPDATE fiscal_exit_items SET quantity=2 WHERE fiscal_exit_id=$1 AND item_code=$2`, nota, fab); err != nil {
		t.Fatalf("linha já lançada continua editável: %v", err)
	}

	// Uso do item: é componente do conjunto.
	uso, err := uc.Uso(a.Ctx, cod(fab))
	if err != nil || len(uso.EstruturasPai) != 1 || uso.EstruturasPai[0] != cod(pai) {
		t.Fatalf("onde é usado: %+v %v", uso, err)
	}

	// Histórico: criação, alteração campo a campo (com autor da transação), liberação e status.
	tx, err := pool.Begin(bg)
	if err != nil {
		t.Fatal(err)
	}
	quem := uuid.New()
	testutil.Exec(t, pool, `INSERT INTO users(id,name,email,password) VALUES($1,'Cadastrista',$2,'x')`, quem, quem.String()+"@example.test")
	if _, err := tx.Exec(bg, `SELECT set_config('venture.item_actor',$1,true)`, quem.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(bg, `UPDATE items SET name='SUPORTE DE FIXACAO 3MM ZINCADO' WHERE code=$1`, fab); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(bg); err != nil {
		t.Fatal(err)
	}
	h, err := uc.Historico(a.Ctx, cod(fab), 0)
	if err != nil {
		t.Fatal(err)
	}
	tem := map[string]bool{}
	var autorDoNome *uuid.UUID
	for _, e := range h {
		chave := e.Evento
		if e.Campo != nil {
			chave += ":" + *e.Campo
		}
		tem[chave] = true
		if chave == "ALTERACAO:name" {
			autorDoNome = e.Por
		}
	}
	for _, quer := range []string{"CRIACAO", "ALTERACAO:pdm_group_code", "ALTERACAO:name", "LIBERACAO:ENGENHARIA", "LIBERACAO:FISCAL", "STATUS:status"} {
		if !tem[quer] {
			t.Errorf("histórico sem %s: %v", quer, tem)
		}
	}
	if autorDoNome == nil || *autorDoNome != quem {
		t.Fatalf("autor da alteração vem da transação: %v", autorDoNome)
	}
	// O caminho da tela: o repositório com pool grava o autor da sessão.
	repo := item_.NewRepositoryItemSQLC(sqlc.New(pool)).ComPool(pool)
	it, err := repo.FindItemByCode(a.Ctx, valueobject.ItemCode(comp))
	if err != nil {
		t.Fatal(err)
	}
	it.Name = "SUPORTE FIXACAO 3 MM PRETO"
	if _, err := repo.UpdateFolders(a.Ctx, it); err != nil {
		t.Fatal(err)
	}
	var porTela *uuid.UUID
	if err := pool.QueryRow(bg, `SELECT por FROM item_historico WHERE item_code=$1 AND campo='name' ORDER BY id DESC LIMIT 1`, comp).Scan(&porTela); err != nil {
		t.Fatal(err)
	}
	if porTela == nil || *porTela != actor {
		t.Fatalf("alteração pela tela registra o usuário da sessão: %v (sessão %v)", porTela, actor)
	}
	if _, err := pool.Exec(bg, `DELETE FROM item_historico WHERE item_code=$1`, fab); err == nil {
		t.Fatal("histórico é imutável")
	}

	// Duplicidade: o nome parecido do item comprado aparece.
	d, err := uc.Duplicados(a.Ctx, "Suporte de fixação 3mm", "", "")
	if err != nil {
		t.Fatal(err)
	}
	achou := false
	for _, c := range d {
		if c.ItemCode == comp {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("duplicado parecido: %+v", d)
	}

	// Alerta de cadastro: o comprado sem fornecedor gera pendência e ela se resolve
	// quando o fornecedor é cadastrado.
	testutil.Exec(t, pool, `INSERT INTO enterprise_notification_subscriptions(enterprise_id,event_key,cadence,created_by) VALUES($1,'CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR','RESUMO_DIARIO',$2)`, a.ID, actor)
	notif := notificationrepo.New(pool)
	if err := notif.ScheduleOperationalAlerts(bg); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM notification_outbox WHERE enterprise_id=$1 AND event_key='CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR' AND aggregate_internal_id=$2`, a.ID, fmt.Sprint(comp)).Scan(&n)
	if n != 1 {
		t.Fatalf("comprado sem fornecedor gera o alerta: %d", n)
	}
	testutil.Exec(t, pool, `INSERT INTO notification_alerts(enterprise_id,event_key,aggregate_type,aggregate_internal_id,deduplication_key,severity,summary)
		VALUES($1,'CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR','ITEM',$2,$3,'ATENCAO','sem fornecedor')`, a.ID, fmt.Sprint(comp), fmt.Sprintf("item:%d:CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR", comp))
	var forn int64
	if err := pool.QueryRow(bg, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORN GOV','11222333000181',$2,$3) RETURNING code`, 860_000+u%90_000, actor, a.ID).Scan(&forn); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, pool, `INSERT INTO item_preferred_suppliers(item_code,supplier_code,ranking,is_active,created_by,enterprise_id) VALUES($1,$2,1,TRUE,$3,$4)`, comp, forn, actor, a.ID)
	if err := notif.ScheduleOperationalAlerts(bg); err != nil {
		t.Fatal(err)
	}
	var estado string
	_ = pool.QueryRow(bg, `SELECT state FROM notification_alerts WHERE enterprise_id=$1 AND event_key='CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR' AND aggregate_internal_id=$2`, a.ID, fmt.Sprint(comp)).Scan(&estado)
	if estado != "RESOLVIDO" {
		t.Fatalf("alerta resolvido com o fornecedor cadastrado: %s", estado)
	}

	// Isolamento.
	if _, err := uc.Painel(b.Ctx, cod(fab)); err == nil {
		t.Fatal("a outra empresa não vê a governança do item")
	}
	if dd, _ := uc.Duplicados(b.Ctx, "Suporte de fixação 3mm", "", ""); len(dd) != 0 {
		t.Fatalf("a outra empresa não vê os itens desta: %+v", dd)
	}
	if pe, _ := uc.Pendentes(b.Ctx); len(pe) != 0 {
		t.Fatal("pendências da outra empresa")
	}
}
