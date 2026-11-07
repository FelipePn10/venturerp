//go:build integration

package item_uc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_governanca_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/pdm_uc"
	dom "github.com/FelipePn10/panossoerp/internal/domain/items/governanca"
	"github.com/FelipePn10/panossoerp/internal/domain/items/pdm"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	govpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/itemgovernanca"
	pdmpg "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/pdm"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type permissaoTotal struct {
	ports.AuthService
	ator uuid.UUID
}

func (permissaoTotal) CanCreateItem(context.Context) bool          { return true }
func (a permissaoTotal) UserID(context.Context) (uuid.UUID, error) { return a.ator, nil }

func decimalDe(n int64) decimal.Decimal { return decimal.NewFromInt(n) }

// nota abre uma NF de saída em rascunho, com número próprio a cada chamada.
func nota(t *testing.T, ctx context.Context, pool *pgxpool.Pool, e testutil.Empresa, ator uuid.UUID) int64 {
	t.Helper()
	cliente := 880_000 + testutil.UniqueCode()%90_000
	if _, err := pool.Exec(ctx, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'CLIENTE','11444777000161',$2,$3) ON CONFLICT DO NOTHING`, cliente, ator, e.ID); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := pool.QueryRow(ctx, `INSERT INTO fiscal_exits(numero_nf,data_emissao,cfop,natureza_operacao,created_by,enterprise_id,customer_code,status)
		VALUES($1,$2::date,'5102','VENDA',$3,$4,$5,'DRAFT') RETURNING id`, testutil.UniqueCode()%900_000, time.Now(), ator, e.ID, cliente).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func atualizacao(t *testing.T, corpo string) request.UpdateItemDTO {
	t.Helper()
	var dto request.UpdateItemDTO
	if err := json.Unmarshal([]byte(corpo), &dto); err != nil {
		t.Fatal(err)
	}
	return dto
}

// O cadastro ampliado grava e relê tudo, inclusive o que mora fora de items; o
// null limpa; o GTIN é único só dentro da empresa; o PDM gera a descrição; a
// política por processo vale no gatilho dos documentos; e a alteração crítica com
// saldo é recusada.
func TestCadastroDeItemAmpliado(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode()
	item := 94_000_000 + u%900_000
	rival := 95_000_000 + u%900_000
	var grupo int32 = int32(70_000 + u%20_000)
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{
				`DELETE FROM pdm_atributos WHERE enterprise_id=$1`, `DELETE FROM pdm_grupo_modificadores WHERE enterprise_id=$1`,
				`DELETE FROM item_status_politicas WHERE enterprise_id=$1`, `DELETE FROM item_governanca WHERE enterprise_id=$1`,
				`DELETE FROM item_planning_extras WHERE enterprise_id=$1`, `DELETE FROM politicas_planejamento WHERE enterprise_id=$1`,
				`DELETE FROM stock_balances WHERE enterprise_id=$1`, `DELETE FROM fiscal_exits WHERE enterprise_id=$1`, `DELETE FROM customers WHERE enterprise_id=$1`,
				`DELETE FROM manufacturing_stock_item_controls WHERE enterprise_id=$1`,
				`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM modifier WHERE enterprise_id=$1`, `DELETE FROM groups WHERE enterprise_id=$1`,
			} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	almoxB := testutil.NovoAlmoxarifado(t, pool, b.ID)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(bg, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	codigo := fmt.Sprintf("AMP%d", item)
	exec(`INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,engineering_type,engineering_weight) VALUES($1,$2,$3,$4,$5,'ITEM AMPLIADO',1,'{"gross":1,"net":1,"unit":"KG"}'::jsonb)`, item, codigo, almox, actor, a.ID)
	exec(`INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,engineering_type,engineering_weight) VALUES($1,$2,$3,$4,$5,'RIVAL',1,'{"gross":1,"net":1,"unit":"KG"}'::jsonb)`, rival, codigo, almoxB, actor, b.ID)

	// Validade e vida útil mínima são do lote: o item controla lote antes.
	exec(`INSERT INTO manufacturing_stock_item_controls(enterprise_id,item_code,stock_uom,controls_lot,controls_address,lot_issue_rule) VALUES($1,$2,'KG',TRUE,FALSE,'FEFO')`, a.ID, item)

	repo := itemrepo.NewRepositoryItemSQLC(sqlc.New(pool)).ComPool(pool)
	auth := permissaoTotal{ator: actor}
	gov := &item_governanca_uc.UseCase{Repo: govpg.New(pool), Auth: auth, Items: repo}
	pdmUC := &pdm_uc.UseCase{Repo: pdmpg.New(pool), Auth: auth}
	uc := item_uc.NewUpdateItemUseCase(repo, auth)
	uc.Uso, uc.PDM = gov, pdmUC

	// Tudo grava e volta, e os complementos ficam em suas tabelas.
	corpo := `{"revision":"B","previous_code":"ANT-1","manufacturer_part_number":"MPN-9","general_notes":"obs",
	 "warehouse":{"shelf_life_days":180,"min_remaining_life_days":30,"storage_conditions":"Seco","hazardous":true,"un_number":"1203","hazard_class":"3","accepts_fractional_quantity":false},
	 "engineering":{"volume":0.25},
	 "planning":{"purchase_lead_time_days":12,"maximum_stock":500,"coverage":3,"grouping_key":"LINHA-A","safety_time_days":2,"maximum_lot":100,"material_issue_timing":"REGISTRATION_RELEASE","notes":"plan","reorder_point":{"TR":5,"CM":10,"CR":30,"ES":2}},
	 "supplies":{"under_delivery_tolerance_pct":2,"over_delivery_tolerance_pct":5,"receipt_routing":"DIRETO","classification_code":"MP"},
	 "accounting":{"gtin":"7891000100103","efd_item_type":"03","tax_unit":"KG","tax_unit_factor":2.5}}`
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo, atualizacao(t, corpo)); err != nil {
		t.Fatal(err)
	}
	lido, err := repo.FindItemByBusinessCode(a.Ctx, valueobject.BusinessCode(codigo))
	if err != nil {
		t.Fatal(err)
	}
	if *lido.Revision != "B" || *lido.Warehouse.ShelfLifeDays != 180 || !lido.Warehouse.Hazardous || lido.Warehouse.AcceptsFractionalQuantity ||
		*lido.Accounting.GTIN != "7891000100103" || *lido.Accounting.EFDItemType != "03" || lido.Supplies.ReceiptRouting != "DIRETO" ||
		lido.Planning.MaterialIssueTiming != "REGISTRATION_RELEASE" || lido.Planning.ReorderPoint == nil {
		t.Fatalf("item relido: %+v", lido)
	}
	e := lido.Planning.Extras
	if !e.MaximumStock.Equal(decimalDe(500)) || e.Coverage != 3 || *e.GroupingKey != "LINHA-A" || *e.SafetyTimeDays != 2 || !e.MaximumLot.Equal(decimalDe(100)) {
		t.Fatalf("complementos relidos: %+v", e)
	}

	// Alteração parcial preserva os complementos; null limpa.
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo, atualizacao(t, `{"name":"ITEM RENOMEADO"}`)); err != nil {
		t.Fatal(err)
	}
	lido, _ = repo.FindItemByBusinessCode(a.Ctx, valueobject.BusinessCode(codigo))
	if lido.Planning.Extras.MaximumLot == nil || lido.Planning.ReorderPoint == nil {
		t.Fatalf("alteração parcial apagou: %+v", lido.Planning)
	}
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo, atualizacao(t, `{"planning":{"maximum_lot":null,"safety_time_days":null},"warehouse":{"min_remaining_life_days":null}}`)); err != nil {
		t.Fatal(err)
	}
	lido, _ = repo.FindItemByBusinessCode(a.Ctx, valueobject.BusinessCode(codigo))
	if lido.Planning.Extras.MaximumLot != nil || lido.Warehouse.MinRemainingLifeDays != nil {
		t.Fatalf("null não limpou: %+v", lido)
	}
	var politicas int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM politicas_planejamento WHERE item_code=$1`, item).Scan(&politicas)
	if politicas != 0 {
		t.Fatal("política vazia do item deveria sumir")
	}

	// GTIN: único na empresa, livre entre empresas.
	exec(`INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,engineering_weight) VALUES($1,$2,$3,$4,$5,'OUTRO','{"gross":1,"net":1,"unit":"KG"}'::jsonb)`, item+1, codigo+"X", almox, actor, a.ID)
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo+"X", atualizacao(t, `{"accounting":{"gtin":"7891000100103"}}`)); err == nil || !strings.Contains(err.Error(), codigo) {
		t.Fatalf("GTIN repetido deve citar o item: %v", err)
	}
	if _, err := uc.ExecuteBusinessCode(b.Ctx, codigo, atualizacao(t, `{"accounting":{"gtin":"7891000100103"}}`)); err != nil {
		t.Fatalf("outra empresa pode usar o mesmo GTIN: %v", err)
	}
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo, atualizacao(t, `{"planning":{"buyer_employee_code":999999999}}`)); err == nil || !strings.Contains(err.Error(), "comprador") {
		t.Fatalf("comprador de fora da empresa: %v", err)
	}

	// PDM: modificador fora do grupo é recusado, atributo obrigatório é cobrado e a
	// descrição sai do padrão.
	var modificador, outro int32
	exec(`INSERT INTO groups(code,description,enterprise_id,created_by) VALUES($1,'FIXADORES',$2,$3)`, grupo, a.ID, actor)
	if err := pool.QueryRow(bg, `INSERT INTO modifier(description,enterprise_id,created_by) VALUES('Parafuso sextavado',$1,$2) RETURNING id`, a.ID, actor).Scan(&modificador); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(bg, `INSERT INTO modifier(description,enterprise_id,created_by) VALUES('Porca',$1,$2) RETURNING id`, a.ID, actor).Scan(&outro); err != nil {
		t.Fatal(err)
	}
	if _, err := pdmUC.SalvarModificadoresDoGrupo(a.Ctx, grupo, []int32{modificador}); err != nil {
		t.Fatal(err)
	}
	prefixo := "M"
	if _, err := pdmUC.SalvarAtributo(a.Ctx, pdm.Atributo{GroupCode: grupo, ModifierID: modificador, Nome: "diametro", Tipo: pdm.TipoNumero, Prefixo: &prefixo, Obrigatorio: true, NaDescricao: true, NaAssinatura: true, Ativo: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := pdmUC.SalvarAtributo(a.Ctx, pdm.Atributo{GroupCode: grupo, ModifierID: modificador, Nome: "acabamento", Ordem: 1, Tipo: pdm.TipoLista, NaDescricao: true, NaAssinatura: true, Ativo: true,
		Valores: []pdm.Valor{{Valor: "zincado", Ativo: true}}}); err != nil {
		t.Fatal(err)
	}
	// Outra empresa não enxerga o padrão.
	if _, err := pdmUC.Modelo(b.Ctx, grupo, modificador); err == nil {
		t.Fatal("o padrão de uma empresa vazou para a outra")
	}
	pdmDe := func(mod int32, attrs string) string {
		return fmt.Sprintf(`{"pdm":{"group_code":%d,"modifier_code":%d,"attributes":%s}}`, grupo, mod, attrs)
	}
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo, atualizacao(t, pdmDe(outro, `[]`))); err == nil || !strings.Contains(err.Error(), "não pertence ao grupo") {
		t.Fatalf("modificador fora do grupo: %v", err)
	}
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo, atualizacao(t, pdmDe(modificador, `[{"Name":"acabamento","Value":"zincado"}]`))); err == nil || !strings.Contains(err.Error(), "DIAMETRO") {
		t.Fatalf("obrigatório faltando: %v", err)
	}
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo, atualizacao(t, pdmDe(modificador, `[{"Name":"diametro","Value":"6"},{"Name":"acabamento","Value":"ZINCADO"}]`))); err != nil {
		t.Fatal(err)
	}
	lido, _ = repo.FindItemByBusinessCode(a.Ctx, valueobject.BusinessCode(codigo))
	if lido.PDM.DescriptionTechnique != "PARAFUSO SEXTAVADO M6 ZINCADO" || lido.PDM.Signature == nil || len(lido.PDM.Attributes) != 2 {
		t.Fatalf("PDM gravado: %+v", lido.PDM)
	}
	previa, err := pdmUC.Previa(a.Ctx, grupo, modificador, []valueobject.Attribute{{Name: "DIAMETRO", Value: "6.0"}, {Name: "ACABAMENTO", Value: "zincado"}}, "NOVO")
	if err != nil {
		t.Fatal(err)
	}
	if len(previa.Duplicados) != 1 || previa.Duplicados[0] != codigo {
		t.Fatalf("a mesma peça com outra grafia deve apontar o duplicado: %+v", previa)
	}

	// Prontidão e prévia fiscal contra o banco.
	prontidao, err := gov.Prontidao(a.Ctx, codigo)
	if err != nil {
		t.Fatal(err)
	}
	if len(prontidao) != 5 {
		t.Fatalf("prontidão: %+v", prontidao)
	}
	fiscal, err := gov.PreviaFiscal(a.Ctx, codigo)
	if err != nil {
		t.Fatal(err)
	}
	if fiscal.PodeFaturar || fiscal.GTIN != "7891000100103" || fiscal.TipoEFD != "03" || fiscal.TipoEFDOrigem != dom.TipoEFDInformado || fiscal.UnidadeTributavel != "KG" {
		t.Fatalf("prévia fiscal: %+v", fiscal)
	}

	// Política por processo: o gatilho dos documentos obedece à matriz.
	if _, err := gov.SalvarPoliticas(a.Ctx, []dom.RegraPolitica{}); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO item_governanca(enterprise_id,item_code,status,motivo) VALUES($1,$2,'DESCONTINUADO','fim de linha')`, a.ID, item)
	usaNoDocumento := func() error {
		_, err := pool.Exec(bg, `INSERT INTO fiscal_exit_items(fiscal_exit_id,item_code,cfop,quantity,unit_price,total_price) VALUES($1,$2,'5102',1,10,10)`, nota(t, bg, pool, a, actor), item)
		return err
	}
	if err := usaNoDocumento(); err == nil || !strings.Contains(err.Error(), "descontinuado") {
		t.Fatalf("sem ajuste o descontinuado segue bloqueado: %v", err)
	}
	if _, err := gov.SalvarPoliticas(a.Ctx, []dom.RegraPolitica{{Status: dom.Descontinuado, Processo: dom.ProcFaturamento, Politica: dom.PoliticaAviso}}); err != nil {
		t.Fatal(err)
	}
	if err := usaNoDocumento(); err != nil {
		t.Fatalf("com aviso o documento passa: %v", err)
	}
	var avisos int
	_ = pool.QueryRow(bg, `SELECT COUNT(*) FROM item_historico WHERE item_code=$1 AND evento='USO_COM_AVISO' AND campo='FATURAMENTO'`, item).Scan(&avisos)
	if avisos != 1 {
		t.Fatalf("o uso com aviso deve ficar no histórico, veio %d", avisos)
	}
	if _, err := gov.SalvarPoliticas(a.Ctx, []dom.RegraPolitica{{Status: dom.Descontinuado, Processo: dom.ProcFaturamento, Politica: dom.PoliticaLiberado}}); err != nil {
		t.Fatal(err)
	}
	if err := usaNoDocumento(); err != nil {
		t.Fatalf("liberado passa: %v", err)
	}
	exec(`DELETE FROM item_governanca WHERE item_code=$1`, item)

	// Impacto: unidade com saldo é recusada no salvar e aparece na simulação.
	exec(`INSERT INTO stock_balances(item_code,mask,warehouse_id,quantity,enterprise_id) VALUES($1,'',$2,12,$3)`, item, almox, a.ID)
	simulacao, err := uc.Impacto(a.Ctx, codigo, atualizacao(t, `{"warehouse":{"unit_of_measurement":"KG"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(simulacao) != 1 || simulacao[0].Severidade != dom.ImpactoBloqueia {
		t.Fatalf("simulação: %+v", simulacao)
	}
	if _, err := uc.ExecuteBusinessCode(a.Ctx, codigo, atualizacao(t, `{"warehouse":{"unit_of_measurement":"KG"}}`)); err == nil || !strings.Contains(err.Error(), "Zere o saldo") {
		t.Fatalf("salvar com saldo: %v", err)
	}
	lido, _ = repo.FindItemByBusinessCode(a.Ctx, valueobject.BusinessCode(codigo))
	if lido.Warehouse.UnitOfMeasurement.String() != "UN" {
		t.Fatalf("a unidade não pode ter mudado: %s", lido.Warehouse.UnitOfMeasurement)
	}
}

func (permissaoTotal) CanCreateGroup(context.Context) bool { return true }
