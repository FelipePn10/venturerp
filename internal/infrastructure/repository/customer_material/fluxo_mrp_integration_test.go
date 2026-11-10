//go:build integration

// Testes do que liga o beneficiamento ao MRP: a sugestão firmada leva o pedido
// até a OF, a OF encerrada deixa de ser suprimento e a peça pronta sem nota conta
// como estoque do cálculo.
package customer_material_test

import (
	"testing"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	repocm "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer_material"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ordemFirme grava uma ordem planejada firme e devolve id e código.
func (a *ambiente) ordemFirme(t *testing.T, item int64, qtd string) (int64, int64) {
	t.Helper()
	var id, code int64
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,is_firm)
		VALUES($1,$2,$3::numeric,'PRODUCTION','SALES_ORDER',$4::date,$5::uuid,1,$6,TRUE) RETURNING id, code`,
		testutil.UniqueCode()%1_000_000, item, qtd, time.Now(), a.userID, a.enterpriseID).Scan(&id, &code); err != nil {
		t.Fatalf("criando ordem firme: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, a.pool, `DELETE FROM planned_orders WHERE id = $1`, id) })
	return id, code
}

// ordemDeFabricacao grava a OF e devolve o id e o número.
func (a *ambiente) ordemDeFabricacao(t *testing.T, item int64, planejada *int64, status, produzida string) (int64, int64) {
	t.Helper()
	var id, numero int64
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO production_orders(order_number,item_code,planned_qty,produced_qty,status,created_by,enterprise_id,planned_order_id)
		VALUES($1,$2,10,$3::numeric,$4,$5::uuid,$6,$7) RETURNING id, order_number`,
		testutil.UniqueCode()%1_000_000, item, produzida, status, a.userID, a.enterpriseID, planejada).Scan(&id, &numero); err != nil {
		t.Fatalf("criando OF: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, a.pool, `DELETE FROM production_orders WHERE id = $1`, id) })
	return id, numero
}

func (a *ambiente) marcarDoCliente(t *testing.T, pai, filho int64) {
	t.Helper()
	testutil.Exec(t, a.pool, `INSERT INTO item_structures(parent_code,child_code,quantity,created_by,customer_supplied)
		VALUES($1,$2,1,$3::uuid,TRUE)`, pai, filho, a.userID)
	t.Cleanup(func() { testutil.Exec(t, a.pool, `DELETE FROM item_structures WHERE parent_code = $1`, pai) })
}

// TestSuprimentoFirmeEhOQueFaltaChegar: sem isto, a ordem firme de uma OF já
// encerrada cobria para sempre os pedidos seguintes do item.
func TestSuprimentoFirmeEhOQueFaltaChegar(t *testing.T) {
	a := montar(t)
	adapter := planned_order.NewPlannedOrderSupplyAdapter(sqlc.New(a.pool))
	item := a.item(t, "PECA")
	suprimento := func(ctx *ambiente) decimal.Decimal {
		t.Helper()
		m, err := adapter.ListFirmSupplyForItems(ctx.ctx, []int64{item})
		if err != nil {
			t.Fatalf("lendo suprimento: %v", err)
		}
		total := decimal.Zero
		for _, s := range m[item] {
			total = total.Add(s.Quantity)
		}
		return total
	}

	planejada, _ := a.ordemFirme(t, item, "10")
	if got := suprimento(a).String(); got != "10" {
		t.Fatalf("firme sem OF = %s, esperado 10", got)
	}
	of, _ := a.ordemDeFabricacao(t, item, &planejada, "IN_PROGRESS", "4")
	if got := suprimento(a).String(); got != "6" {
		t.Fatalf("OF com entrega parcial = %s, esperado 6", got)
	}
	if b := montar(t); !suprimento(b).IsZero() {
		t.Fatal("suprimento firme vazou para outra empresa")
	}
	testutil.Exec(t, a.pool, `UPDATE production_orders SET status = 'COMPLETED', produced_qty = 7 WHERE id = $1`, of)
	if got := suprimento(a); !got.IsZero() {
		t.Fatalf("OF concluída com 7 de 10 ainda é suprimento: %s", got)
	}

	cancelada, _ := a.ordemFirme(t, item, "3")
	a.ordemDeFabricacao(t, item, &cancelada, "CANCELLED", "0")
	if got := suprimento(a); !got.IsZero() {
		t.Fatalf("OF cancelada ainda é suprimento: %s", got)
	}
	// A ordem de serviço (etapa em terceiro) da mesma peça não é peça a chegar:
	// a peça volta pela OF. Contá-la dobrava o suprimento.
	servico, _ := a.ordemFirme(t, item, "5")
	testutil.Exec(t, a.pool, `UPDATE planned_orders SET order_type = 'OUTSOURCING' WHERE id = $1`, servico)
	if got := suprimento(a); !got.IsZero() {
		t.Fatalf("ordem de serviço em terceiro contou como suprimento da peça: %s", got)
	}
}

// TestProducaoAguardandoRetorno: a peça pronta do cliente conta até a nota levar;
// o atendido é do comercial e filtra por enterprise_code.
func TestProducaoAguardandoRetorno(t *testing.T) {
	a := montar(t)
	b := montar(t)
	empresaCode := a.codigoDaEmpresa(t)
	ler := func(amb *ambiente) map[int64]decimal.Decimal {
		t.Helper()
		m, err := mrp_calculation.NewMRPCalculationRepositorySQLC(sqlc.New(amb.pool), amb.pool).ProducaoAguardandoRetorno(amb.ctx)
		if err != nil {
			t.Fatalf("lendo produção a devolver: %v", err)
		}
		return m
	}

	bruto := a.item(t, "BRUTO DO CLIENTE")
	parcial := a.item(t, "PECA PARCIAL")
	atendida := a.item(t, "PECA ATENDIDA")
	propria := a.item(t, "PECA PROPRIA")
	a.marcarDoCliente(t, parcial, bruto)
	a.marcarDoCliente(t, atendida, bruto)

	a.ordemDeFabricacao(t, parcial, nil, "COMPLETED", "6")
	a.ordemDeFabricacao(t, parcial, nil, "CANCELLED", "9") // cancelada não entrega
	a.ordemDeFabricacao(t, atendida, nil, "COMPLETED", "4")
	a.ordemDeFabricacao(t, propria, nil, "COMPLETED", "8") // sem material do cliente: entrou no estoque

	a.pedido(t, empresaCode, 100, "P", linhaDeTeste{parcial, "6", "1", "2"})
	a.pedido(t, empresaCode, 100, "F", linhaDeTeste{atendida, "4", "1", "4"})
	// Pedidos que NÃO podem abater: de outra empresa, e gravado com o id no lugar do código.
	a.pedido(t, b.codigoDaEmpresa(t), 100, "F", linhaDeTeste{parcial, "1", "1", "1"})
	a.pedido(t, a.enterpriseID, 100, "F", linhaDeTeste{parcial, "1", "1", "1"})

	m := ler(a)
	if got := m[parcial]; got.String() != "4" {
		t.Fatalf("a devolver = %s, esperado 4 (6 entregues − 2 atendidos)", got)
	}
	if _, ok := m[atendida]; ok {
		t.Fatalf("item todo atendido segue a devolver: %s", m[atendida])
	}
	if _, ok := m[propria]; ok {
		t.Fatal("item sem componente do cliente não é produção a devolver")
	}
	if daOutra := ler(b); len(daOutra) != 0 {
		t.Fatalf("produção vazou para outra empresa: %v", daOutra)
	}
}

// TestSugestaoFirmadaLevaOPedidoAteAsHoras: é o caminho real do MRP — a ordem
// planejada não guarda o pedido; quem guarda é o vínculo gravado ao firmar, a
// partir do pegging, que o próximo cálculo apaga.
func TestSugestaoFirmadaLevaOPedidoAteAsHoras(t *testing.T) {
	a := montar(t)
	empresaCode := a.codigoDaEmpresa(t)
	pai := a.item(t, "PECA USINADA")
	a.marcarDoCliente(t, pai, a.item(t, "BRUTO DO CLIENTE"))

	cliente := testutil.UniqueCode() % 1_000_000_000
	testutil.Exec(t, a.pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id)
		VALUES($1,'USIMAC TESTE','12345678000195',$2::uuid,$3)`, cliente, a.userID, a.enterpriseID)
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM customers WHERE code = $1 AND enterprise_id = $2`, cliente, a.enterpriseID)
	})
	pedido, linhas := a.pedido(t, empresaCode, cliente, "P", linhaDeTeste{pai, "6", "23", "0"})

	planejada, planejadaCode := a.ordemFirme(t, pai, "6")
	sugestao := testutil.UniqueCode()
	testutil.Exec(t, a.pool, `INSERT INTO mrp_pegging(enterprise_id,plan_code,item_code,demand_kind,demand_code,need_date,supply_kind,supply_code,quantity)
		VALUES($1,0,$2,'PEDIDO_VENDA',$3,$4::date,'SUGESTAO',$5,6)`, a.enterpriseID, pai, linhas[0], time.Now(), sugestao)
	t.Cleanup(func() { testutil.Exec(t, a.pool, `DELETE FROM mrp_pegging WHERE enterprise_id = $1`, a.enterpriseID) })

	repoDemanda := independent_demand.NewIndependentDemandRepositorySQLC(sqlc.New(a.pool)).ComDB(a.pool)
	autor := uuid.MustParse(a.userID)
	for range 2 { // firmar de novo não duplica
		if err := repoDemanda.VincularSugestao(a.ctx, sugestao, planejadaCode, autor); err != nil {
			t.Fatalf("vinculando: %v", err)
		}
	}
	var vinculos int
	if err := a.pool.QueryRow(a.ctx, `SELECT COUNT(*) FROM sales_order_planned_orders WHERE planned_order_id = $1 AND sales_order_item_code = $2 AND enterprise_id = $3`,
		planejada, linhas[0], a.enterpriseID).Scan(&vinculos); err != nil {
		t.Fatal(err)
	}
	if vinculos != 1 {
		t.Fatalf("vínculos = %d, esperado 1", vinculos)
	}

	// O pegging é refeito a cada cálculo: o relatório não pode depender dele.
	testutil.Exec(t, a.pool, `DELETE FROM mrp_pegging WHERE enterprise_id = $1`, a.enterpriseID)
	of, _ := a.ordemDeFabricacao(t, pai, &planejada, "IN_PROGRESS", "0")
	testutil.Exec(t, a.pool, `INSERT INTO production_appointments(production_order_id,appointment_date,start_time,end_time,created_by)
		VALUES($1,$2::date,'08:00'::time,'10:00'::time,$3::uuid)`, of, time.Now(), a.userID)
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM production_appointments WHERE production_order_id = $1`, of)
	})

	horas, err := a.repo.HorasDeBeneficiamento(a.ctx, domrepo.FiltroHoras{SalesOrderCode: &pedido})
	if err != nil || len(horas) != 1 {
		t.Fatalf("horas pelo pedido: %d linha(s), err=%v", len(horas), err)
	}
	if h := horas[0]; h.CustomerCode == nil || *h.CustomerCode != cliente || h.CustomerName != "USIMAC TESTE" {
		t.Fatalf("cliente não chegou pelo vínculo: %+v", h)
	}
}

// TestRemessaEncerradaSaiDoInventario: quem encerrou com saldo declarou que aquele
// material não volta. Ele não pode seguir no inventário, sem ação possível, até
// virar prazo vencido; a remessa continua no histórico.
func TestRemessaEncerradaSaiDoInventario(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "8.5")
	comSaldo := func() int {
		t.Helper()
		l, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{SomenteComSaldo: true})
		if err != nil {
			t.Fatalf("listando: %v", err)
		}
		return len(l)
	}
	saldos, err := a.repo.SaldoPorItem(a.ctx, domrepo.FiltroSaldo{})
	if err != nil || len(saldos) != 1 || comSaldo() != 1 {
		t.Fatalf("aberta: %d saldo(s), err=%v", len(saldos), err)
	}
	if err := a.repo.Encerrar(a.ctx, r.ID, "cliente dispensou o retorno", a.userID); err != nil {
		t.Fatalf("encerrando: %v", err)
	}
	if saldos, err = a.repo.SaldoPorItem(a.ctx, domrepo.FiltroSaldo{}); err != nil || len(saldos) != 0 {
		t.Fatalf("encerrada segue no inventário: %d saldo(s), err=%v", len(saldos), err)
	}
	if n := comSaldo(); n != 0 {
		t.Fatalf("encerrada segue na fila com saldo: %d", n)
	}
	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID: r.Itens[0].ID, MovementType: entity.MovimentoSobra, Quantity: "1", IdempotencyKey: uuid.NewString(),
	}, a.userID); err == nil {
		t.Fatal("remessa encerrada aceitou baixa de saldo")
	} else if _, ok := errorsuc.AsConflict(err); !ok {
		t.Fatalf("baixa em remessa encerrada: esperado conflito, veio %v", err)
	}
	if todas, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{Status: []entity.StatusRemessa{entity.StatusEncerrada}}); err != nil || len(todas) != 1 {
		t.Fatalf("encerrada sumiu do histórico: %d, err=%v", len(todas), err)
	}
}

// TestRelatorioDeMovimentosTrazONumeroDaOF: a fábrica conhece a OF pelo número.
func TestRelatorioDeMovimentosTrazONumeroDaOF(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	of, numero := a.ordemDeFabricacao(t, a.item(t, "PECA"), nil, "IN_PROGRESS", "0")
	if _, err := a.repo.RegistrarMovimento(a.ctx, domrepo.NovoMovimento{
		RemittanceItemID:  r.Itens[0].ID,
		MovementType:      entity.MovimentoSobra,
		Quantity:          "1",
		ProductionOrderID: &of,
		IdempotencyKey:    uuid.NewString(),
	}, a.userID); err != nil {
		t.Fatalf("lançando sobra: %v", err)
	}
	l, err := a.repo.RelatorioDeMovimentos(a.ctx, domrepo.FiltroMovimentos{RemittanceID: &r.ID, Tipos: []string{string(entity.MovimentoSobra)}})
	if err != nil || len(l) != 1 {
		t.Fatalf("relatório: %d linha(s), err=%v", len(l), err)
	}
	if got := l[0].UnitValue.String(); got != "178.283333" {
		t.Fatalf("valor do movimento avulso = %s, esperado o da nota do cliente", got)
	}
	if l[0].ProductionOrderNumber == nil || *l[0].ProductionOrderNumber != numero {
		t.Fatalf("número da OF = %v, esperado %d", l[0].ProductionOrderNumber, numero)
	}
}

// TestRemessaTrazANotaPendente: a remessa sem saldo não está "encerrada" enquanto a
// nota não for autorizada; rascunho e rejeitada aparecem, autorizada e cancelada não.
func TestRemessaTrazANotaPendente(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	var nota int64
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO fiscal_exits(numero_nf,serie,data_emissao,cfop,natureza_operacao,status,created_by,enterprise_id,customer_material_remittance_id,source_type)
		VALUES(77,'1',$1::date,'5124','TESTE','REJECTED',$2::uuid,$3,$4,'BENEFICIAMENTO') RETURNING id`,
		time.Now(), a.userID, a.enterpriseID, r.ID).Scan(&nota); err != nil {
		t.Fatalf("criando a nota: %v", err)
	}
	t.Cleanup(func() { testutil.Exec(t, a.pool, `DELETE FROM fiscal_exits WHERE id = $1`, nota) })
	testutil.Exec(t, a.pool, `UPDATE fiscal_exits SET motivo_rejeicao = '508 - CST' WHERE id = $1`, nota)

	p := a.recarregar(t, r.ID).NotaPendente
	if p == nil || p.ID != nota || p.Numero != 77 || p.Status != "REJECTED" || p.MotivoRejeicao == nil {
		t.Fatalf("nota pendente lida errada: %+v", p)
	}
	lista, err := a.repo.Listar(a.ctx, domrepo.FiltroRemessa{})
	if err != nil || len(lista) != 1 || lista[0].NotaPendente == nil {
		t.Fatalf("lista sem a nota pendente: %d, err=%v", len(lista), err)
	}
	for _, status := range []string{"AUTHORIZED", "CANCELLED"} {
		testutil.Exec(t, a.pool, `UPDATE fiscal_exits SET status = $1 WHERE id = $2`, status, nota)
		if got := a.recarregar(t, r.ID).NotaPendente; got != nil {
			t.Fatalf("nota %s ainda pendente: %+v", status, got)
		}
	}
}

// TestPedidoDaOFAvulsaChegaAsHoras: a OF aberta à mão ganha cliente pelo vínculo;
// OF de item comum e de outra empresa são recusadas.
func TestPedidoDaOFAvulsaChegaAsHoras(t *testing.T) {
	a := montar(t)
	b := montar(t)
	pai := a.item(t, "PECA USINADA")
	a.marcarDoCliente(t, pai, a.item(t, "BRUTO"))
	cliente := testutil.UniqueCode() % 1_000_000_000
	testutil.Exec(t, a.pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id)
		VALUES($1,'CLIENTE AVULSO','12345678000195',$2::uuid,$3)`, cliente, a.userID, a.enterpriseID)
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM customers WHERE code = $1 AND enterprise_id = $2`, cliente, a.enterpriseID)
	})
	pedido, _ := a.pedido(t, a.codigoDaEmpresa(t), cliente, "P", linhaDeTeste{pai, "3", "1", "0"})
	of, _ := a.ordemDeFabricacao(t, pai, nil, "IN_PROGRESS", "0")
	testutil.Exec(t, a.pool, `INSERT INTO production_appointments(production_order_id,appointment_date,start_time,end_time,created_by)
		VALUES($1,$2::date,'08:00'::time,'09:00'::time,$3::uuid)`, of, time.Now(), a.userID)
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM production_appointments WHERE production_order_id = $1`, of)
	})

	pedidos := repocm.NewPedidos(a.pool)
	if err := pedidos.VincularPedidoDaOF(a.ctx, of, &pedido); err != nil {
		t.Fatalf("vinculando: %v", err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `UPDATE production_orders SET sales_order_code = NULL WHERE id = $1`, of)
	})
	horas, err := a.repo.HorasDeBeneficiamento(a.ctx, domrepo.FiltroHoras{CustomerCode: &cliente})
	if err != nil || len(horas) != 1 || horas[0].SalesOrderCode == nil || *horas[0].SalesOrderCode != pedido {
		t.Fatalf("horas pelo vínculo da OF: %+v err=%v", horas, err)
	}

	comum, _ := a.ordemDeFabricacao(t, a.item(t, "PECA PROPRIA"), nil, "IN_PROGRESS", "0")
	if err := pedidos.VincularPedidoDaOF(a.ctx, comum, &pedido); !errorsuc.EhNaoEncontrado(err) {
		t.Fatalf("OF de item comum: esperado não encontrado, veio %v", err)
	}
	if err := repocm.NewPedidos(b.pool).VincularPedidoDaOF(b.ctx, of, nil); !errorsuc.EhNaoEncontrado(err) {
		t.Fatalf("OF de outra empresa: esperado não encontrado, veio %v", err)
	}
}

// TestCompraFirmeEhOQueFaltaChegar: na ordem firme de compra, o recebido já é
// estoque e o cancelado não vem mais; só o resto é suprimento.
func TestCompraFirmeEhOQueFaltaChegar(t *testing.T) {
	a := montar(t)
	fornecedor := testutil.UniqueCode() % 1_000_000_000
	testutil.Exec(t, a.pool, `INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id) VALUES($1,'FORNECEDOR',$2,$3::uuid,$4)`,
		fornecedor, "12345678000195", a.userID, a.enterpriseID)
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM suppliers WHERE code = $1 AND enterprise_id = $2`, fornecedor, a.enterpriseID)
	})
	adapter := planned_order.NewPlannedOrderSupplyAdapter(sqlc.New(a.pool))
	item := a.item(t, "COMPRADO")
	suprimento := func() string {
		t.Helper()
		m, err := adapter.ListFirmSupplyForItems(a.ctx, []int64{item})
		if err != nil {
			t.Fatalf("lendo suprimento: %v", err)
		}
		total := decimal.Zero
		for _, s := range m[item] {
			total = total.Add(s.Quantity)
		}
		return total.String()
	}
	id, codigo := a.ordemFirme(t, item, "10")
	testutil.Exec(t, a.pool, `UPDATE planned_orders SET order_type = 'PURCHASE' WHERE id = $1`, id)
	if got := suprimento(); got != "10" {
		t.Fatalf("compra firme sem pedido = %s, esperado 10", got)
	}

	var pedido, linha int64
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO purchase_orders(order_number,enterprise_code,supplier_code,created_by) VALUES($1,$2,$3,$4::uuid) RETURNING code`,
		testutil.UniqueCode(), a.codigoDaEmpresa(t), fornecedor, a.userID).Scan(&pedido); err != nil {
		t.Fatalf("criando o pedido de compra: %v", err)
	}
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM purchase_order_items WHERE purchase_order_code = $1`, pedido)
		testutil.Exec(t, a.pool, `DELETE FROM purchase_orders WHERE code = $1`, pedido)
	})
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO purchase_order_items(purchase_order_code,item_code,purchase_uom,unit_price,planned_order_code,received_qty,cancelled_qty)
		VALUES($1,$2,'UN',1,$3,4,1) RETURNING code`, pedido, item, codigo).Scan(&linha); err != nil {
		t.Fatalf("criando a linha: %v", err)
	}
	if got := suprimento(); got != "5" {
		t.Fatalf("10 - 4 recebidos - 1 cancelado = %s, esperado 5", got)
	}
	testutil.Exec(t, a.pool, `UPDATE purchase_order_items SET received_qty = 9 WHERE code = $1`, linha)
	if got := suprimento(); got != "0" {
		t.Fatalf("compra toda chegou e ainda é suprimento: %s", got)
	}
}

// TestRemessaTrazONomeDoCliente: o relatório da remessa mostra o nome; o código do
// cliente pode repetir em outra empresa e o nome tem de ser o da empresa da remessa.
func TestRemessaTrazONomeDoCliente(t *testing.T) {
	a := montar(t)
	b := montar(t)
	r := a.remessa(t, "1")
	if r2 := a.recarregar(t, r.ID); r2.CustomerName != "" {
		t.Fatalf("cliente não cadastrado veio com nome %q", r2.CustomerName)
	}
	testutil.Exec(t, a.pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES
		($1,'CLIENTE CERTO','12345678000195',$2::uuid,$3),($1,'CLIENTE DE OUTRA EMPRESA','12345678000195',$2::uuid,$4)`,
		r.CustomerCode, a.userID, a.enterpriseID, b.enterpriseID)
	t.Cleanup(func() { testutil.Exec(t, a.pool, `DELETE FROM customers WHERE code = $1`, r.CustomerCode) })
	if got := a.recarregar(t, r.ID).CustomerName; got != "CLIENTE CERTO" {
		t.Fatalf("nome do cliente = %q", got)
	}
}
