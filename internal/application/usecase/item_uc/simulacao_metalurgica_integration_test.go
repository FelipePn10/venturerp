//go:build integration

package item_uc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/group_uc"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc"
	fiscaldom "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	grouprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/group"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	porepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order"
	shiprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment"
	stockrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Catálogo de uma metalúrgica com linha de móveis passando pelo que cada
// indicador promete. Cada item é real no banco; só o despacho e o MRP ficam nos
// testes próprios deles.
func TestSimulacaoMetalurgicaEMoveleira(t *testing.T) {
	pool := testutil.Pool(t)
	bg := context.Background()
	actor := testutil.Actor(t, pool)
	u := testutil.UniqueCode() % 800_000
	a, b := testutil.DuasEmpresas(t, pool, func(a, b testutil.Empresa) {
		for _, e := range []testutil.Empresa{a, b} {
			for _, q := range []string{`DELETE FROM stock_movements WHERE enterprise_id=$1`, `DELETE FROM stock_lot_balances WHERE enterprise_id=$1`,
				`DELETE FROM stock_lots WHERE enterprise_id=$1`, `DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1`,
				`DELETE FROM manufacturing_stock_item_controls WHERE enterprise_id=$1`,
				`DELETE FROM items WHERE enterprise_id=$1`, `DELETE FROM groups WHERE enterprise_id=$1`} {
				if _, err := pool.Exec(bg, q, e.ID); err != nil {
					t.Logf("limpeza %s: %v", q, err)
				}
			}
		}
	})
	almox := testutil.NovoAlmoxarifado(t, pool, a.ID)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(bg, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	repo := itemrepo.NewRepositoryItemSQLC(sqlc.New(pool)).ComPool(pool)
	auth := permissaoTotal{ator: actor}
	uc := item_uc.NewUpdateItemUseCase(repo, auth)

	novo := func(n int64, nome string, tipo int, corpo string) (int64, string) {
		t.Helper()
		code, cod := 96_000_000+u+n*1_000_000, fmt.Sprintf("SIM%d-%d", n, u)
		exec(`INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name,engineering_type,engineering_weight) VALUES($1,$2,$3,$4,$5,$6,$7,'{"gross":1,"net":1,"unit":"KG"}'::jsonb)`, code, cod, almox, actor, a.ID, nome, tipo)
		if strings.Contains(corpo, "shelf_life_days") {
			// Validade é do lote: o item controla lote antes de recebê-la.
			exec(`INSERT INTO manufacturing_stock_item_controls(enterprise_id,item_code,stock_uom,controls_lot,controls_address,lot_issue_rule) VALUES($1,$2,'UN',TRUE,FALSE,'FEFO')`, a.ID, code)
		}
		var dto request.UpdateItemDTO
		if err := json.Unmarshal([]byte(corpo), &dto); err != nil {
			t.Fatal(err)
		}
		if _, err := uc.ExecuteBusinessCode(a.Ctx, cod, dto); err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		return code, cod
	}
	ler := func(cod string) (decimalItem struct {
		venda func(q string, cliente *int64) error
	}) {
		t.Helper()
		it, err := repo.FindItemByBusinessCode(a.Ctx, valueobject.BusinessCode(cod))
		if err != nil {
			t.Fatal(err)
		}
		decimalItem.venda = func(q string, c *int64) error { return it.ConferirVenda(decimal.RequireFromString(q), c) }
		return
	}

	// Parafuso em caixa de 100: múltiplo, mínimo e sem fração.
	_, parafuso := novo(1, "PARAFUSO SEXTAVADO M8", 1, `{"warehouse":{"accepts_fractional_quantity":false},"commercial":{"sale_multiple":100,"minimum_sale_quantity":100}}`)
	v := ler(parafuso)
	if err := v.venda("150", nil); err == nil || !strings.Contains(err.Error(), "múltiplos") {
		t.Fatalf("150 não é múltiplo de 100: %v", err)
	}
	if err := v.venda("200", nil); err != nil {
		t.Fatal(err)
	}

	// Peça de desenho do cliente.
	var cliente int64 = 890_000 + u%90_000
	exec(`INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES($1,'MONTADORA','11444777000161',$2,$3)`, cliente, actor, a.ID)
	_, exclusiva := novo(2, "SUPORTE DESENHO 4471", 0, fmt.Sprintf(`{"engineering":{"oem":true},"commercial":{"exclusive_customer_code":%d},"planning":{"exclusive":true}}`, cliente))
	outro := cliente + 1
	if err := ler(exclusiva).venda("1", &outro); err == nil {
		t.Fatal("outro cliente não compra peça exclusiva")
	}
	if err := ler(exclusiva).venda("1", &cliente); err != nil {
		t.Fatal(err)
	}
	// Cliente de outra empresa não vira dono do item.
	if _, err := uc.ExecuteBusinessCode(a.Ctx, exclusiva, atualizacao(t, `{"commercial":{"exclusive_customer_code":1}}`)); err == nil {
		t.Fatal("cliente inexistente na empresa deve ser recusado")
	}

	// Tinta inflamável: validade preenche o lote, ONU obrigatório.
	if _, err := uc.ExecuteBusinessCode(a.Ctx, parafuso, atualizacao(t, `{"warehouse":{"hazardous":true}}`)); err == nil {
		t.Fatal("perigoso sem ONU")
	}
	tinta, _ := novo(3, "TINTA ESMALTE PRETA", 1, `{"warehouse":{"shelf_life_days":180,"min_remaining_life_days":60,"hazardous":true,"un_number":"1263","hazard_class":"3"}}`)
	estoque := stockrepo.NewStockRepositorySQLC(pool)
	lote := "TINTA-1"
	mov := &entity.StockMovement{ItemCode: tinta, WarehouseID: almox, MovementType: entity.MovementTypeIn, Quantity: decimal.NewFromInt(5), UnitPrice: decimal.NewFromInt(10), TotalPrice: decimal.NewFromInt(50), Lot: &lote, CreatedBy: actor}
	if _, err := estoque.CreateMovement(a.Ctx, mov); err != nil {
		t.Fatal(err)
	}
	var dias int
	if err := pool.QueryRow(bg, `SELECT expires_at - CURRENT_DATE FROM stock_lots WHERE enterprise_id=$1 AND item_code=$2 AND lot=$3`, a.ID, tinta, lote).Scan(&dias); err != nil {
		t.Fatal(err)
	}
	if dias != 180 {
		t.Fatalf("o lote sem data assume a validade do item: %d dias", dias)
	}

	// Chapa com corrida: a regra de recebimento vem do cadastro e o certificado grava.
	chapa, _ := novo(4, "CHAPA ACO 3MM", 1, `{"supplies":{"requires_certificate":true,"receiving_checklist":true,"receipt_routing":"DIRETO","over_delivery_tolerance_pct":5},"engineering":{"cutting_grain":"LENGTH"}}`)
	regra, err := porepo.NewPurchaseOrderRepositorySQLC(pool).RegraDeRecebimento(a.Ctx, chapa)
	if err != nil {
		t.Fatal(err)
	}
	if !regra.ExigeCertificado || !regra.ExigeChecklist || regra.Roteiro != "DIRETO" || regra.ToleranciaExcessoPct == nil {
		t.Fatalf("regra do recebimento: %+v", regra)
	}
	corrida, certificado := "C-77", "CERT-9"
	if err := estoque.RegistrarCertificadoDoLote(a.Ctx, chapa, "", "L-1", &corrida, &certificado); err != nil {
		t.Fatal(err)
	}
	var guardada string
	if err := pool.QueryRow(bg, `SELECT heat_number FROM stock_lots WHERE enterprise_id=$1 AND item_code=$2 AND lot='L-1'`, a.ID, chapa).Scan(&guardada); err != nil || guardada != corrida {
		t.Fatalf("corrida gravada: %q %v", guardada, err)
	}

	// Guarda-roupa: volumes e exigência de montar antes de despachar.
	guarda, gcod := novo(5, "GUARDA ROUPA 3P", 0, `{"commercial":{"volumes_per_unit":3,"assemble_shipping_volumes":true,"requires_special_packaging":true}}`)
	regras := shiprepo.NewRegrasDoRomaneioPG(pool)
	exigem, err := regras.ItensQueExigemVolume(a.Ctx, []int64{guarda, chapa})
	if err != nil || len(exigem) != 1 || exigem[0] != gcod {
		t.Fatalf("exigem volume: %v %v", exigem, err)
	}
	porUnidade, err := regras.VolumesPorUnidade(a.Ctx, []int64{guarda})
	if err != nil || !porUnidade[guarda].Equal(decimal.NewFromInt(3)) {
		t.Fatalf("volumes: %v %v", porUnidade, err)
	}

	// NF-e e EFD: descrição travada, GTIN, tipo 05 da sucata.
	sucata, _ := novo(6, "SUCATA DE ACO", 1, `{"is_scrap":true,"commercial":{"description":"SUCATA FERROSA","allow_billing_description_change":false},"accounting":{"gtin":"7891000100103","tax_unit":"KG","tax_unit_factor":1}}`)
	fiscal := fiscalrepo.NewFiscalRepositoryPG(pool).(interface {
		IdentificacaoFiscal(context.Context, []int64) (map[int64]fiscaldom.IdentificacaoFiscal, error)
	})
	nota, err := fiscal.IdentificacaoFiscal(a.Ctx, []int64{sucata})
	if err != nil {
		t.Fatal(err)
	}
	if c := nota[sucata]; c.DescricaoFixa != "SUCATA FERROSA" || c.GTIN != "7891000100103" || c.UnidadeTributavel != "KG" {
		t.Fatalf("identificação fiscal: %+v", c)
	}

	// Grupo: mesmo código em duas empresas, nome repetido recusado, inativo some.
	grupos := grouprepo.NewRepositoryGroupSQLC(sqlc.New(pool))
	criar := group_uc.NewCreateGroupUseCase(grupos, auth)
	atualizar := group_uc.NewUpdateGroupUseCase(grupos, auth)
	codigo := int(40_000 + u%20_000)
	if _, err := criar.Execute(a.Ctx, request.CreateGroupDTO{Code: codigo, Description: "CHAPAS"}); err != nil {
		t.Fatal(err)
	}
	if _, err := criar.Execute(b.Ctx, request.CreateGroupDTO{Code: codigo, Description: "CHAPAS"}); err != nil {
		t.Fatalf("cada empresa usa os seus códigos: %v", err)
	}
	if _, err := criar.Execute(a.Ctx, request.CreateGroupDTO{Code: codigo + 1, Description: " chapas "}); err == nil {
		t.Fatal("nome repetido com outra grafia")
	}
	if _, err := criar.Execute(a.Ctx, request.CreateGroupDTO{Code: codigo, Description: "OUTRO"}); err == nil || !strings.Contains(err.Error(), "código") {
		t.Fatalf("código repetido com mensagem clara: %v", err)
	}
	nao := false
	if _, err := atualizar.Execute(a.Ctx, codigo, "CHAPAS", &nao); err != nil {
		t.Fatal(err)
	}
	if err := repo.ValidatePDMReferences(a.Ctx, a.ID, codigo, 0); err == nil {
		t.Fatal("grupo inativo não aceita item novo")
	}
	if err := repo.ValidatePDMReferences(b.Ctx, b.ID, codigo, 0); err != nil {
		t.Fatalf("inativar numa empresa não mexe na outra: %v", err)
	}
	if _, err := atualizar.Execute(a.Ctx, codigo+5, "X", nil); err == nil {
		t.Fatal("grupo inexistente deve dar erro")
	}
}
