//go:build integration

// Testes do pedido de beneficiamento e dos relatórios (migração 000453).
//
// O pedido de venda é do COMERCIAL (enterprise_code) e o razão de terceiros usa
// enterprise_id. Na empresa de teste os dois números são DIFERENTES de propósito:
// é o que denuncia a junção pela convenção errada, que passaria na empresa 1.
//
//	TEST_DATABASE_URL=... go test -tags=integration -run 'Pedido|Relatorio|Horas' ./internal/infrastructure/repository/customer_material/
package customer_material_test

import (
	"strings"
	"testing"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	repo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer_material"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	"github.com/shopspring/decimal"
)

// Mesmo texto de customer_material_uc.MotivoSemPedido; repetido para o teste do
// repositório não depender da camada de aplicação.
const motivoSemPedido = "material recebido sem pedido de beneficiamento cadastrado"

func (a *ambiente) codigoDaEmpresa(t *testing.T) int64 {
	t.Helper()
	var code int64
	if err := a.pool.QueryRow(a.ctx, `SELECT code FROM enterprise WHERE id = $1`, a.enterpriseID).Scan(&code); err != nil {
		t.Fatalf("lendo o código da empresa: %v", err)
	}
	if code == a.enterpriseID {
		t.Fatalf("empresa de teste com code = id (%d): o teste não distinguiria as convenções", code)
	}
	return code
}

// item cria um item da empresa e devolve o código.
func (a *ambiente) item(t *testing.T, nome string) int64 {
	t.Helper()
	code := testutil.UniqueCode()
	testutil.Exec(t, a.pool, `INSERT INTO items(code,business_code,warehouse_code,created_by,enterprise_id,name)
		VALUES($1,($1::bigint)::text,$1,$2::uuid,$3,$4)`, code, a.userID, a.enterpriseID, nome)
	t.Cleanup(func() { testutil.Exec(t, a.pool, `DELETE FROM items WHERE code = $1`, code) })
	return code
}

type linhaDeTeste struct {
	item     int64
	qtd      string
	preco    string
	atendida string
}

// pedido grava um pedido de venda na empresa INFORMADA (enterprise_code) e
// devolve o código do pedido e o de cada linha.
func (a *ambiente) pedido(t *testing.T, empresaCode, cliente int64, status string, linhas ...linhaDeTeste) (int64, []int64) {
	t.Helper()
	var code int64
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by)
		VALUES($1,$2,$3,$4,$5::uuid) RETURNING code`,
		testutil.UniqueCode()%1_000_000, empresaCode, status, cliente, a.userID).Scan(&code); err != nil {
		t.Fatalf("criando pedido: %v", err)
	}
	codigos := make([]int64, 0, len(linhas))
	for i, l := range linhas {
		var lc int64
		if err := a.pool.QueryRow(a.ctx, `INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,attended_qty,sales_uom)
			VALUES($1,$2,$3,$4::numeric,$5::numeric,$6::numeric,'HR') RETURNING code`,
			code, i+1, l.item, l.qtd, l.preco, l.atendida).Scan(&lc); err != nil {
			t.Fatalf("criando linha do pedido: %v", err)
		}
		codigos = append(codigos, lc)
	}
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM sales_order_items WHERE sales_order_code = $1`, code)
		testutil.Exec(t, a.pool, `DELETE FROM sales_orders WHERE code = $1`, code)
	})
	return code, codigos
}

// TestPedidoDeBeneficiamentoLeSaldoPorLinha: o saldo de cada linha é o pedido
// menos o atendido; as linhas vêm na sequência.
func TestPedidoDeBeneficiamentoLeSaldoPorLinha(t *testing.T) {
	a := montar(t)
	servico := a.item(t, "SERVICO DE USINAGEM")
	solda := a.item(t, "SERVICO DE SOLDA")
	code, linhas := a.pedido(t, a.codigoDaEmpresa(t), 100, "P",
		linhaDeTeste{servico, "10", "23", "4"},
		linhaDeTeste{solda, "5", "40", "0"})

	p, err := repo.NewPedidos(a.pool).PedidoDeBeneficiamento(a.ctx, code, 0)
	if err != nil {
		t.Fatalf("lendo o pedido: %v", err)
	}
	if p.CustomerCode != 100 || !p.Ativo || p.Status != "P" {
		t.Fatalf("cabeçalho lido errado: %+v", p)
	}
	if len(p.Linhas) != 2 || p.Linhas[0].Code != linhas[0] || p.Linhas[1].Code != linhas[1] {
		t.Fatalf("linhas fora de ordem ou faltando: %+v", p.Linhas)
	}
	if got := p.Linhas[0].Saldo().String(); got != "6" {
		t.Fatalf("saldo da linha 1 = %s, esperado 6 (10 pedidos − 4 atendidos)", got)
	}
	if got := p.Linhas[1].PrecoUnitario.String(); got != "40" {
		t.Fatalf("preço da linha 2 = %s", got)
	}
	if p.Linhas[0].Unidade != "HR" || p.Linhas[0].Descricao != "SERVICO DE USINAGEM" {
		t.Fatalf("unidade/descrição da linha 1 = %q/%q", p.Linhas[0].Unidade, p.Linhas[0].Descricao)
	}
}

// TestPedidoDeBeneficiamentoUsaEnterpriseCode: o pedido de outra empresa é "não
// encontrado", e o filtro é pelo CÓDIGO da empresa. Um pedido gravado com o id
// da empresa no lugar do código também não aparece — é a convenção errada.
func TestPedidoDeBeneficiamentoUsaEnterpriseCode(t *testing.T) {
	a := montar(t)
	b := montar(t)
	servico := a.item(t, "SERVICO")

	daOutra, _ := a.pedido(t, b.codigoDaEmpresa(t), 100, "P", linhaDeTeste{servico, "1", "1", "0"})
	if _, err := repo.NewPedidos(a.pool).PedidoDeBeneficiamento(a.ctx, daOutra, 0); !errorsuc.EhNaoEncontrado(err) {
		t.Fatalf("pedido de outra empresa: esperado não encontrado, veio %v", err)
	}
	if _, err := repo.NewPedidos(b.pool).PedidoDeBeneficiamento(b.ctx, daOutra, 0); err != nil {
		t.Fatalf("a dona do pedido não o encontra: %v", err)
	}

	pedidoComID, _ := a.pedido(t, a.enterpriseID, 100, "P", linhaDeTeste{servico, "1", "1", "0"})
	if _, err := repo.NewPedidos(a.pool).PedidoDeBeneficiamento(a.ctx, pedidoComID, 0); !errorsuc.EhNaoEncontrado(err) {
		t.Fatalf("pedido gravado com enterprise_id no lugar do código foi aceito: %v", err)
	}
}

// TestVincularPedidoLiberaOBloqueioNaMesmaTransacao: vincular o pedido grava o
// código e, quando pedido, tira o bloqueio — com autor e motivo na trilha.
func TestVincularPedidoLiberaOBloqueioNaMesmaTransacao(t *testing.T) {
	a := montar(t)
	motivo := motivoSemPedido
	emissao := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	r, err := a.repo.Registrar(a.ctx, &entity.Remessa{
		CustomerCode: 100, NFeNumber: testutil.UniqueCode(), NFeSeries: "1", CFOP: "5901",
		IssueDate: emissao, ReceivedAt: emissao, FiscalReturnDeadline: emissao.AddDate(0, 0, 30),
		Status: entity.StatusAberta, CreatedBy: a.userID, Blocked: true, BlockReason: &motivo,
		Itens: []*entity.ItemRemessa{{
			LineNumber: 1, CustomerItemCode: "X1", Description: "PECA", NCM: "76169900", UOM: "PC",
			QtyInvoiced: decimal.NewFromInt(2), QtyReceived: decimal.NewFromInt(2), UnitValue: decimal.NewFromInt(10),
		}},
	})
	if err != nil {
		t.Fatalf("registrando remessa bloqueada: %v", err)
	}
	if !r.Blocked {
		t.Fatal("a remessa deveria nascer bloqueada")
	}

	if err := a.repo.VincularPedido(a.ctx, r.ID, 4242, true, a.userID); err != nil {
		t.Fatalf("vinculando: %v", err)
	}
	depois := a.recarregar(t, r.ID)
	if depois.SalesOrderCode == nil || *depois.SalesOrderCode != 4242 {
		t.Fatalf("pedido gravado = %v, esperado 4242", depois.SalesOrderCode)
	}
	if depois.Blocked || depois.BlockReason != nil {
		t.Fatalf("bloqueio continuou: blocked=%v motivo=%v", depois.Blocked, depois.BlockReason)
	}

	var evento *entity.EventoDeAuditoria
	for _, e := range a.trilha(t, r.ID) {
		if e.EntityType == "customer_material_remittances" && e.Acao == "UPDATE" && contem(e.Alterados, "sales_order_code") {
			evento = e
		}
	}
	if evento == nil {
		t.Fatal("o vínculo do pedido não entrou na trilha")
	}
	if evento.AtorID == nil || *evento.AtorID != a.userID {
		t.Fatalf("autor do vínculo = %v", evento.AtorID)
	}
	if evento.Motivo == nil || !strings.Contains(*evento.Motivo, "pedido de beneficiamento") {
		t.Fatalf("motivo do vínculo = %v", evento.Motivo)
	}
	if !contem(evento.Alterados, "blocked") {
		t.Fatalf("liberação não aparece nos campos alterados: %v", evento.Alterados)
	}
}

// TestVincularPedidoSemLiberarMantemOBloqueio: bloqueio por outro motivo segue.
func TestVincularPedidoSemLiberarMantemOBloqueio(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "3")
	if err := a.repo.Bloquear(a.ctx, r.ID, "divergência de quantidade", a.userID); err != nil {
		t.Fatalf("bloqueando: %v", err)
	}
	if err := a.repo.VincularPedido(a.ctx, r.ID, 77, false, a.userID); err != nil {
		t.Fatalf("vinculando: %v", err)
	}
	depois := a.recarregar(t, r.ID)
	if !depois.Blocked || depois.BlockReason == nil || *depois.BlockReason != "divergência de quantidade" {
		t.Fatalf("bloqueio por divergência foi mexido: blocked=%v motivo=%v", depois.Blocked, depois.BlockReason)
	}
}

// TestVincularPedidoDeOutraEmpresaEhNaoEncontrado: a remessa de outra empresa
// não muda.
func TestVincularPedidoDeOutraEmpresaEhNaoEncontrado(t *testing.T) {
	a := montar(t)
	b := montar(t)
	r := a.remessa(t, "1")
	if err := b.repo.VincularPedido(b.ctx, r.ID, 1, false, b.userID); !errorsuc.EhNaoEncontrado(err) {
		t.Fatalf("vínculo cruzando empresas: esperado não encontrado, veio %v", err)
	}
	if depois := a.recarregar(t, r.ID); depois.SalesOrderCode != nil {
		t.Fatalf("a remessa da outra empresa foi alterada: %v", *depois.SalesOrderCode)
	}
}

// TestRelatorioDeMovimentosFiltraEOcultaEstornados: a movimentação traz a nota
// do cliente e a linha; filtra por tipo; o estornado só aparece quando pedido.
func TestRelatorioDeMovimentosFiltraEOcultaEstornados(t *testing.T) {
	a := montar(t)
	r := a.remessa(t, "6")
	nota := int64(920_001 + testutilUnico())
	a.notaBaixada(t, r, nota, "2")

	retornos, err := a.repo.RelatorioDeMovimentos(a.ctx, domrepo.FiltroMovimentos{
		RemittanceID: &r.ID, Tipos: []string{string(entity.MovimentoRetorno)},
	})
	if err != nil {
		t.Fatalf("relatório: %v", err)
	}
	if len(retornos) != 1 {
		t.Fatalf("retornos = %d, esperado 1", len(retornos))
	}
	l := retornos[0]
	if l.RemittanceID != r.ID || l.NFeNumber != r.NFeNumber || l.CustomerItemCode != "20042554" || l.NCM != "76169900" {
		t.Fatalf("linha do relatório sem a origem: %+v", l)
	}
	if l.Quantity.String() != "2" || l.MovementType != entity.MovimentoRetorno {
		t.Fatalf("movimento lido errado: tipo=%s qtd=%s", l.MovementType, l.Quantity)
	}

	// A entrada não é linha do razão, mas o relatório de entrada e saída a mostra:
	// a quantidade conferida, na data do recebimento.
	entradas, err := a.repo.RelatorioDeMovimentos(a.ctx, domrepo.FiltroMovimentos{
		RemittanceID: &r.ID, Tipos: []string{string(entity.MovimentoRecebimento)},
	})
	if err != nil || len(entradas) != 1 {
		t.Fatalf("entradas = %d, esperado 1 (err=%v)", len(entradas), err)
	}
	if e := entradas[0]; e.Quantity.String() != "6" || e.MovementType != entity.MovimentoRecebimento ||
		e.RemittanceItemID != r.Itens[0].ID || e.CreatedAt.IsZero() {
		t.Fatalf("entrada lida errada: tipo=%s qtd=%s linha=%d", e.MovementType, e.Quantity, e.RemittanceItemID)
	}
	todos, err := a.repo.RelatorioDeMovimentos(a.ctx, domrepo.FiltroMovimentos{RemittanceID: &r.ID})
	if err != nil || len(todos) != 2 {
		t.Fatalf("entrada + saída = %d linha(s), esperado 2 (err=%v)", len(todos), err)
	}

	ajustes, err := a.repo.RelatorioDeMovimentos(a.ctx, domrepo.FiltroMovimentos{
		RemittanceID: &r.ID, Tipos: []string{string(entity.MovimentoAjuste)},
	})
	if err != nil || len(ajustes) != 0 {
		t.Fatalf("filtro por tipo: %d linha(s), err=%v", len(ajustes), err)
	}

	if _, err := a.repo.EstornarMovimentosDaNota(a.ctx, nota, a.userID, motivoEstorno); err != nil {
		t.Fatalf("estornando: %v", err)
	}
	semEstorno, err := a.repo.RelatorioDeMovimentos(a.ctx, domrepo.FiltroMovimentos{
		RemittanceID: &r.ID, Tipos: []string{string(entity.MovimentoRetorno)},
	})
	if err != nil || len(semEstorno) != 0 {
		t.Fatalf("estornado apareceu sem pedir: %d linha(s), err=%v", len(semEstorno), err)
	}
	comEstorno, err := a.repo.RelatorioDeMovimentos(a.ctx, domrepo.FiltroMovimentos{
		RemittanceID: &r.ID, Tipos: []string{string(entity.MovimentoRetorno)}, IncluirEstornados: true,
	})
	if err != nil || len(comEstorno) != 1 || !comEstorno[0].Estornado() {
		t.Fatalf("estornado pedido explicitamente: %d linha(s), err=%v", len(comEstorno), err)
	}

	b := montar(t)
	daOutra, err := b.repo.RelatorioDeMovimentos(b.ctx, domrepo.FiltroMovimentos{RemittanceID: &r.ID, IncluirEstornados: true})
	if err != nil || len(daOutra) != 0 {
		t.Fatalf("movimentação vazou para outra empresa: %d linha(s), err=%v", len(daOutra), err)
	}
}

// TestHorasDeBeneficiamentoLevaAoClientePelaDemanda: a OF de item com
// componente do cliente aparece com as horas apontadas, e o cliente chega pela
// cadeia OF → ordem planejada → DEMANDA do pedido → pedido. A ordem planejada
// aponta a demanda, não o pedido: ligar direto daria cliente errado ou nenhum.
func TestHorasDeBeneficiamentoLevaAoClientePelaDemanda(t *testing.T) {
	a := montar(t)
	empresaCode := a.codigoDaEmpresa(t)
	pai := a.item(t, "PECA USINADA")
	materialDoCliente := a.item(t, "BRUTO DO CLIENTE")
	semMaterial := a.item(t, "PECA PROPRIA")
	testutil.Exec(t, a.pool, `INSERT INTO item_structures(parent_code,child_code,quantity,created_by,customer_supplied)
		VALUES($1,$2,1,$3::uuid,TRUE)`, pai, materialDoCliente, a.userID)
	t.Cleanup(func() { testutil.Exec(t, a.pool, `DELETE FROM item_structures WHERE parent_code = $1`, pai) })

	cliente := testutil.UniqueCode() % 1_000_000_000
	testutil.Exec(t, a.pool, `INSERT INTO customers(code,name,document_number,created_by,enterprise_id)
		VALUES($1,'USIMAC TESTE','12345678000195',$2::uuid,$3)`, cliente, a.userID, a.enterpriseID)
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM customers WHERE code = $1 AND enterprise_id = $2`, cliente, a.enterpriseID)
	})

	pedido, _ := a.pedido(t, empresaCode, cliente, "P", linhaDeTeste{pai, "10", "23", "0"})
	var demanda, planejada int64
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO sales_order_demands(sales_order_code,item_code,quantity,delivery_date,enterprise_id)
		VALUES($1,$2,10,$3::date,$4) RETURNING code`, pedido, pai, time.Now(), a.enterpriseID).Scan(&demanda); err != nil {
		t.Fatalf("criando demanda: %v", err)
	}
	if demanda == pedido {
		t.Logf("aviso: código da demanda coincidiu com o do pedido (%d) — a junção errada passaria", demanda)
	}
	if err := a.pool.QueryRow(a.ctx, `INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,sales_order_code)
		VALUES($1,$2,10,'PRODUCTION','SALES_ORDER',$3::date,$4::uuid,1,$5,$6) RETURNING id`,
		testutil.UniqueCode()%1_000_000, pai, time.Now(), a.userID, a.enterpriseID, demanda).Scan(&planejada); err != nil {
		t.Fatalf("criando ordem planejada: %v", err)
	}

	of := func(item int64, planejadaID *int64) int64 {
		var id int64
		if err := a.pool.QueryRow(a.ctx, `INSERT INTO production_orders(order_number,item_code,planned_qty,status,created_by,enterprise_id,planned_order_id)
			VALUES($1,$2,10,'IN_PROGRESS',$3::uuid,$4,$5) RETURNING id`,
			testutil.UniqueCode()%1_000_000, item, a.userID, a.enterpriseID, planejadaID).Scan(&id); err != nil {
			t.Fatalf("criando OF: %v", err)
		}
		return id
	}
	ofBenef := of(pai, &planejada)
	ofPropria := of(semMaterial, nil)
	ofSemApontamento := of(pai, nil)
	hoje := time.Now()
	for _, apt := range []struct {
		of          int64
		inicio, fim string
	}{
		{ofBenef, "08:00", "11:30"},
		{ofBenef, "23:00", "01:00"}, // vira a meia-noite: 2 h, não −22 h
		{ofPropria, "08:00", "17:00"},
	} {
		testutil.Exec(t, a.pool, `INSERT INTO production_appointments(production_order_id,appointment_date,start_time,end_time,created_by)
			VALUES($1,$2::date,$3::time,$4::time,$5::uuid)`, apt.of, hoje, apt.inicio, apt.fim, a.userID)
	}
	t.Cleanup(func() {
		testutil.Exec(t, a.pool, `DELETE FROM production_appointments WHERE production_order_id = ANY($1)`, []int64{ofBenef, ofPropria, ofSemApontamento})
		testutil.Exec(t, a.pool, `DELETE FROM production_orders WHERE id = ANY($1)`, []int64{ofBenef, ofPropria, ofSemApontamento})
		testutil.Exec(t, a.pool, `DELETE FROM planned_orders WHERE id = $1`, planejada)
		testutil.Exec(t, a.pool, `DELETE FROM sales_order_demands WHERE code = $1`, demanda)
	})

	linhas, err := a.repo.HorasDeBeneficiamento(a.ctx, domrepo.FiltroHoras{})
	if err != nil {
		t.Fatalf("relatório de horas: %v", err)
	}
	if len(linhas) != 1 {
		t.Fatalf("OFs no relatório = %d, esperado 1 (só a de beneficiamento com apontamento)", len(linhas))
	}
	h := linhas[0]
	if h.ProductionOrderID != ofBenef {
		t.Fatalf("OF errada no relatório: %d", h.ProductionOrderID)
	}
	if got := h.HorasApontadas.Round(2).String(); got != "5.5" {
		t.Fatalf("horas apontadas = %s, esperado 5.5", got)
	}
	if h.Apontamentos != 2 {
		t.Fatalf("apontamentos = %d, esperado 2", h.Apontamentos)
	}
	if h.CustomerCode == nil || *h.CustomerCode != cliente || h.CustomerName != "USIMAC TESTE" {
		t.Fatalf("cliente não chegou pela demanda: code=%v nome=%q", h.CustomerCode, h.CustomerName)
	}
	if h.SalesOrderCode == nil || *h.SalesOrderCode != pedido {
		t.Fatalf("pedido = %v, esperado %d", h.SalesOrderCode, pedido)
	}

	porCliente, err := a.repo.HorasDeBeneficiamento(a.ctx, domrepo.FiltroHoras{CustomerCode: &cliente})
	if err != nil || len(porCliente) != 1 {
		t.Fatalf("filtro por cliente: %d linha(s), err=%v", len(porCliente), err)
	}
	outro := cliente + 1
	if nenhum, err := a.repo.HorasDeBeneficiamento(a.ctx, domrepo.FiltroHoras{CustomerCode: &outro}); err != nil || len(nenhum) != 0 {
		t.Fatalf("filtro por outro cliente: %d linha(s), err=%v", len(nenhum), err)
	}
	amanha := hoje.AddDate(0, 0, 2)
	if nenhum, err := a.repo.HorasDeBeneficiamento(a.ctx, domrepo.FiltroHoras{De: &amanha}); err != nil || len(nenhum) != 0 {
		t.Fatalf("filtro de período: %d linha(s), err=%v", len(nenhum), err)
	}

	b := montar(t)
	if daOutra, err := b.repo.HorasDeBeneficiamento(b.ctx, domrepo.FiltroHoras{}); err != nil || len(daOutra) != 0 {
		t.Fatalf("horas vazaram para outra empresa: %d linha(s), err=%v", len(daOutra), err)
	}
}

// TestRemessaTrazONumeroDoPedido: a tela mostra o número que o cliente conhece,
// não o código interno.
func TestRemessaTrazONumeroDoPedido(t *testing.T) {
	a := montar(t)
	b := montar(t)
	servico := a.item(t, "SERVICO")
	emissao := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	r, err := a.repo.Registrar(a.ctx, &entity.Remessa{
		CustomerCode: 100, NFeNumber: testutil.UniqueCode(), NFeSeries: "1", CFOP: "5901",
		IssueDate: emissao, ReceivedAt: emissao, FiscalReturnDeadline: emissao.AddDate(0, 0, 30),
		Status: entity.StatusAberta, CreatedBy: a.userID,
		Itens: []*entity.ItemRemessa{{
			LineNumber: 1, CustomerItemCode: "X1", Description: "PECA", NCM: "76169900", UOM: "PC",
			QtyInvoiced: decimal.NewFromInt(2), QtyReceived: decimal.NewFromInt(2), UnitValue: decimal.NewFromInt(10),
		}},
	})
	if err != nil {
		t.Fatalf("registrando remessa: %v", err)
	}

	proprio, _ := a.pedido(t, a.codigoDaEmpresa(t), 100, "P", linhaDeTeste{servico, "1", "1", "0"})
	var numero int64
	if err := a.pool.QueryRow(a.ctx, `SELECT order_number FROM sales_orders WHERE code = $1`, proprio).Scan(&numero); err != nil {
		t.Fatalf("lendo o número: %v", err)
	}
	if numero == proprio {
		if _, err := a.pool.Exec(a.ctx, `UPDATE sales_orders SET order_number = order_number + 900000 WHERE code = $1`, proprio); err != nil {
			t.Fatalf("separando número e código: %v", err)
		}
		numero += 900000
	}
	if err := a.repo.VincularPedido(a.ctx, r.ID, proprio, false, a.userID); err != nil {
		t.Fatalf("vinculando: %v", err)
	}
	if got := a.recarregar(t, r.ID).SalesOrderNumber; got == nil || *got != numero {
		t.Fatalf("número do pedido = %v, esperado %d", got, numero)
	}

	// O vínculo com pedido de outra empresa não chega a existir: o banco recusa.
	daOutra, _ := a.pedido(t, b.codigoDaEmpresa(t), 100, "P", linhaDeTeste{servico, "1", "1", "0"})
	if err := a.repo.VincularPedido(a.ctx, r.ID, daOutra, false, a.userID); err == nil {
		t.Fatal("vínculo com pedido de outra empresa foi aceito")
	}
}
