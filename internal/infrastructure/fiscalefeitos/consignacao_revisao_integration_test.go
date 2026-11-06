//go:build integration

package fiscalefeitos_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	fiscalRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal"
	stockRepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
)

// Bloco H com as duas direções da consignação: a leitura do participante e o rateio
// do saldo não podem derrubar a geração nem quando o estoque é menor que a remessa.
func TestBlocoHComConsignacaoNasDuasDirecoes(t *testing.T) {
	c := novaNotaAutorizada(t)
	consig := novoAlmoxConsignacao(t, c)
	e := c.efeitosComTipos(t)
	repo := stockRepo.NewStockRepositorySQLC(c.pool)
	remessa := c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA")
	cliA, cliB := int64(771), int64(772)
	exige(t, c.aplicar(t, e, c.notaDoTipo(t, remessa, &cliA, "6")))
	exige(t, c.aplicar(t, e, c.notaDoTipo(t, remessa, &cliB, "4")))
	testutil.Exec(t, c.pool, `UPDATE fiscal_exits SET cnpj_destinatario='11222333000181', razao_social_destinatario='Cliente' WHERE enterprise_id=$1`, c.empresa)

	fornecedor := testutil.UniqueCode() % 1_000_000
	testutil.Exec(t, c.pool, `INSERT INTO suppliers(code,name,person_type,document_type,document_number,created_by,enterprise_id,register_date) VALUES($1,'Forn','JURIDICA','CNPJ','11222333000181',$2,$3,CURRENT_DATE)`, fornecedor, c.ator, c.empresa)
	t.Cleanup(func() {
		testutil.Exec(t, c.pool, `DELETE FROM suppliers WHERE code=$1 AND enterprise_id=$2`, fornecedor, c.empresa)
	})
	_, err := repo.ReceberConsignacao(c.ctx, stockentity.Consignacao{Parceiro: fornecedor, ItemCode: c.item, Quantidade: dq("8"), Valor: dq("30"), Emissao: mesmoDia(), CreatedBy: c.ator})
	exige(t, err)

	hoje := time.Now().AddDate(0, 0, 2)
	itens, _, err := novoFiscalPG(c).InventarioEFD(c.ctx, hoje)
	exige(t, err)
	if len(itens) < 3 {
		t.Fatalf("%+v", itens)
	}

	// Estoque menor que o consignado: a primeira remessa leva tudo e a segunda acha zero.
	_, err = repo.CreateMovement(c.ctx, &stockentity.StockMovement{ItemCode: c.item, WarehouseID: consig, MovementType: stockentity.MovementTypeOut, Quantity: dq("7"), CreatedBy: c.ator})
	exige(t, err)
	if _, _, err := novoFiscalPG(c).InventarioEFD(c.ctx, hoje); err != nil {
		t.Fatal(err)
	}
}

// Entrada cancelada e refeita: desfazer de novo devolve só o que saiu da última vez.
func TestTransferenciaEntreCNPJsCancelarARecebidaDuasVezes(t *testing.T) {
	i := duasFiliais(t, "11222333")
	o := i.origem
	n := i.notaDeTransferencia(t, "11222333000262")
	exige(t, o.aplicar(t, o.efeitosComTipos(t), n))
	receber := func(entrada int64) {
		emTx(t, i.destino, func(tx pgx.Tx) error {
			return stockRepo.ReceberEntreCNPJsTx(i.destino.ctx, tx, i.destino.empresa, i.chave, entrada, i.destino.ator)
		})
	}
	desfazer := func(entrada int64) {
		emTx(t, i.destino, func(tx pgx.Tx) error {
			return stockRepo.DesfazerRecebimentoEntreCNPJsTx(i.destino.ctx, tx, i.destino.empresa, entrada, i.destino.ator)
		})
	}
	receber(555)
	desfazer(555)
	igualQ(t, "primeiro cancelamento", saldo(t, o, i.transit), "4")
	receber(556)
	igualQ(t, "segunda entrada", saldo(t, o, i.transit), "0")
	desfazer(556)
	igualQ(t, "segundo cancelamento devolve só 4", saldo(t, o, i.transit), "4")
}

// A baixa do trânsito é mudança de estabelecimento, não consumo do item.
func TestTransferenciaEntreCNPJsNaoContaComoConsumo(t *testing.T) {
	i := duasFiliais(t, "11222333")
	o := i.origem
	exige(t, o.aplicar(t, o.efeitosComTipos(t), i.notaDeTransferencia(t, "11222333000262")))
	emTx(t, i.destino, func(tx pgx.Tx) error {
		return stockRepo.ReceberEntreCNPJsTx(i.destino.ctx, tx, i.destino.empresa, i.chave, 555, i.destino.ator)
	})
	repo := stockRepo.NewStockRepositorySQLC(o.pool)
	analise, err := repo.AnaliseDeEstoque(o.ctx, 3, 0)
	exige(t, err)
	for _, l := range analise {
		if l.ItemCode == o.item && l.ConsumoQtd.IsPositive() {
			t.Errorf("a transferência virou consumo: %+v", l)
		}
	}
}

// Remessa de pedido: a reserva acompanha a mercadoria, senão sobra reserva sem saldo.
func TestRemessaDoPedidoConsomeAReserva(t *testing.T) {
	c := novaNotaAutorizada(t)
	novoAlmoxConsignacao(t, c)
	repo := stockRepo.NewStockRepositorySQLC(c.pool)
	pedido := int64(4242)
	_, err := repo.CreateReservation(c.ctx, &stockentity.StockReservation{ItemCode: c.item, WarehouseID: c.almox, Quantity: dq("6"),
		ReferenceType: stockentity.ReferenceTypeSalesOrder, ReferenceCode: pedido, CreatedBy: c.ator})
	exige(t, err)
	t.Cleanup(func() { testutil.Exec(t, c.pool, `DELETE FROM stock_reservations WHERE enterprise_id=$1`, c.empresa) })
	cliente := int64(778)
	n := c.notaDoTipo(t, c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA"), &cliente, "6")
	testutil.Exec(t, c.pool, `UPDATE fiscal_exits SET sales_order_code=$2 WHERE id=$1`, n, pedido)
	exige(t, c.aplicar(t, c.efeitosComTipos(t), n))
	var reservado string
	exige(t, c.pool.QueryRow(context.Background(), `SELECT COALESCE(SUM(reserved_qty),0)::text FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3`, c.empresa, c.item, c.almox).Scan(&reservado))
	igualQ(t, "reservado da origem", dq(reservado), "0")
	var ativa string
	exige(t, c.pool.QueryRow(context.Background(), `SELECT COALESCE(SUM(quantity),0)::text FROM stock_reservations WHERE enterprise_id=$1 AND status='ACTIVE'`, c.empresa).Scan(&ativa))
	igualQ(t, "reserva ativa depois da remessa", dq(ativa), "0")
}

// Um cliente recebeu um lote, outro cliente outro lote; a nota de venda sem lote
// não pode levar o lote do vizinho.
func TestVendaConsignadaSaiDoLoteDoCliente(t *testing.T) {
	c := novaNotaAutorizada(t)
	consig := novoAlmoxConsignacao(t, c)
	e := c.efeitosComTipos(t)
	repo := stockRepo.NewStockRepositorySQLC(c.pool)
	for _, l := range []string{"L1", "L2"} {
		lote := l
		_, err := repo.CreateMovement(c.ctx, &stockentity.StockMovement{ItemCode: c.item, WarehouseID: c.almox, MovementType: stockentity.MovementTypeIn, Quantity: dq("5"), UnitPrice: dq("20"), Lot: &lote, CreatedBy: c.ator})
		exige(t, err)
	}
	remessa := c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA")
	venda := c.tipoDeNota(t, "VENDA_CONSIGNACAO", "ATUALIZA")
	cliA, cliB := int64(781), int64(782)
	nA := c.notaDoTipo(t, remessa, &cliA, "3")
	testutil.Exec(t, c.pool, `UPDATE fiscal_exit_items SET lot='L1' WHERE fiscal_exit_id=$1`, nA)
	exige(t, c.aplicar(t, e, nA))
	nB := c.notaDoTipo(t, remessa, &cliB, "3")
	testutil.Exec(t, c.pool, `UPDATE fiscal_exit_items SET lot='L2' WHERE fiscal_exit_id=$1`, nB)
	exige(t, c.aplicar(t, e, nB))

	exige(t, c.aplicar(t, e, c.notaDoTipo(t, venda, &cliB, "2")))
	var l1, l2 string
	exige(t, c.pool.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(quantity) FILTER (WHERE lot='L1'),0)::text, COALESCE(SUM(quantity) FILTER (WHERE lot='L2'),0)::text
		   FROM stock_lot_balances WHERE enterprise_id=$1 AND warehouse_id=$2 AND item_code=$3`, c.empresa, consig, c.item).Scan(&l1, &l2))
	igualQ(t, "lote do outro cliente", dq(l1), "3")
	igualQ(t, "lote deste cliente", dq(l2), "1")
}

// A mensagem de quem não tem saldo fala a língua do usuário.
func TestVendaConsignadaSemSaldoSemCodigoInterno(t *testing.T) {
	c := novaNotaAutorizada(t)
	testutil.Exec(t, c.pool, `UPDATE items SET business_code='PECA-77' WHERE code=$1 AND enterprise_id=$2`, c.item, c.empresa)
	err := stockRepo.NewStockRepositorySQLC(c.pool).RegistrarVendaConsignada(c.ctx, stockentity.VendaConsignada{Cliente: 1, ItemCode: c.item, Quantidade: dq("1.5"), Emissao: mesmoDia(), Ator: c.ator})
	if err == nil || !strings.Contains(err.Error(), "PECA-77") {
		t.Fatalf("%v", err)
	}
}

// Nota de transferência à irmã sem almoxarifado de trânsito: o defeito aparece antes
// da SEFAZ, não como pendência de nota já autorizada.
func TestTransferenciaEntreCNPJsConfereOTransitoAntesDaSefaz(t *testing.T) {
	i := duasFiliais(t, "11222333")
	o := i.origem
	testutil.Exec(t, o.pool, `UPDATE warehouse SET location='INTERNO' WHERE id=$1`, i.transit)
	n := i.notaDeTransferencia(t, "11222333000262")
	repo := fiscalRepo.NewFiscalRepositoryPG(o.pool)
	ex, err := repo.GetExitByID(o.ctx, n)
	exige(t, err)
	itens, err := repo.GetExitItems(o.ctx, n)
	exige(t, err)
	if err := o.efeitosComTipos(t).PreConferirDepositos(o.ctx, ex, itens); err == nil || !strings.Contains(err.Error(), "Trânsito") {
		t.Fatalf("passou da conferência: %v", err)
	}
}

// Devolver para o próprio almoxarifado de consignação baixaria o cliente sem mover nada.
func TestDevolucaoNaoVoltaParaOAlmoxDeConsignacao(t *testing.T) {
	c := novaNotaAutorizada(t)
	consig := novoAlmoxConsignacao(t, c)
	e := c.efeitosComTipos(t)
	repo := stockRepo.NewStockRepositorySQLC(c.pool)
	cliente := int64(790)
	exige(t, c.aplicar(t, e, c.notaDoTipo(t, c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA"), &cliente, "2")))
	lista, err := repo.ListarConsignacoes(c.ctx, "SAIDA", true)
	exige(t, err)
	if _, err := repo.DevolverConsignacao(c.ctx, stockentity.BaixaConsignacao{ID: lista[0].ID, Quantidade: dq("1"), Almox: consig, Ator: c.ator}); err == nil {
		t.Fatal("devolveu para o próprio almoxarifado de consignação")
	}
	s, err := repo.SaldoConsignado(c.ctx, cliente, c.item, "", nil)
	exige(t, err)
	igualQ(t, "saldo do cliente intacto", s, "2")
}

// Duas empresas com o mesmo CNPJ: a nota não escolhe uma delas no escuro.
func TestEmpresaIrmaRecusaCNPJRepetido(t *testing.T) {
	i := duasFiliais(t, "11222333")
	terceira := novaNotaAutorizada(t)
	testutil.Exec(t, terceira.pool, `INSERT INTO fiscal_configs(enterprise_id,cnpj_empresa,razao_social,updated_by) VALUES($1,'11222333000262','Copia',$2)`, terceira.empresa, terceira.ator)
	t.Cleanup(func() {
		testutil.Exec(t, terceira.pool, `DELETE FROM fiscal_configs WHERE enterprise_id=$1`, terceira.empresa)
	})
	if _, err := stockRepo.NewStockRepositorySQLC(i.origem.pool).EmpresaIrma(i.origem.ctx, "11222333000262"); err == nil {
		t.Fatal("escolheu uma das empresas com o CNPJ repetido")
	}
}

// Inspeção e rejeição travam a saída: a venda consignada ficaria presa.
func TestAlmoxDeConsignacaoNaoPodeSerDeInspecao(t *testing.T) {
	c := novaNotaAutorizada(t)
	almox := testutil.NovoAlmoxarifado(t, c.pool, c.empresa)
	testutil.Exec(t, c.pool, `UPDATE warehouse SET disposition=FALSE, location='INSPECAO' WHERE id=$1`, almox)
	if err := stockRepo.NewStockRepositorySQLC(c.pool).ConferirAlmoxDeConsignacao(c.ctx, almox); err == nil {
		t.Fatal("aceitou o almoxarifado de inspeção")
	}
}

// A venda sai do almoxarifado do parâmetro: trocá-lo com remessa aberta prenderia a nota.
func TestAlmoxDeConsignacaoNaoTrocaComRemessaAberta(t *testing.T) {
	c := novaNotaAutorizada(t)
	consig := novoAlmoxConsignacao(t, c)
	outro := testutil.NovoAlmoxarifado(t, c.pool, c.empresa)
	testutil.Exec(t, c.pool, `UPDATE warehouse SET disposition=FALSE WHERE id=$1`, outro)
	repo := stockRepo.NewStockRepositorySQLC(c.pool)
	cliente := int64(791)
	exige(t, c.aplicar(t, c.efeitosComTipos(t), c.notaDoTipo(t, c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA"), &cliente, "2")))
	p := stockentity.ParametrosEstoque{ExpiredLotAction: stockentity.LoteVencidoBloqueia, ConsignmentWarehouseID: &outro}
	if err := repo.SalvarParametrosDeEstoque(c.ctx, p, c.ator); err == nil {
		t.Fatal("trocou o almoxarifado com remessa aberta")
	}
	p.ConsignmentWarehouseID = nil
	if err := repo.SalvarParametrosDeEstoque(c.ctx, p, c.ator); err == nil {
		t.Fatal("limpou o almoxarifado com remessa aberta")
	}
	p.ConsignmentWarehouseID = &consig
	exige(t, repo.SalvarParametrosDeEstoque(c.ctx, p, c.ator))
}

// Pedido já separado em onda: a remessa fecha a separação, e o cancelamento a devolve.
func TestRemessaDePedidoSeparadoFechaAOnda(t *testing.T) {
	c := novaNotaAutorizada(t)
	novoAlmoxConsignacao(t, c)
	pedido := int64(4343)
	var onda, linha int64
	exige(t, c.pool.QueryRow(context.Background(), `INSERT INTO stock_picking_waves(enterprise_id,code,warehouse_id,status,rule,created_by) VALUES($1,$2,$3,'SEPARADA','FIFO',$4) RETURNING id`,
		c.empresa, testutil.UniqueCode()%1_000_000, c.almox, c.ator).Scan(&onda))
	exige(t, c.pool.QueryRow(context.Background(), `INSERT INTO stock_picking_wave_lines(enterprise_id,wave_id,item_code,mask,lot,address,quantity,picked_qty,reference_type,reference_code)
		VALUES($1,$2,$3,'','','',2,2,$4,$5) RETURNING id`, c.empresa, onda, c.item, stockentity.ReferenceTypeSalesOrder, pedido).Scan(&linha))
	t.Cleanup(func() {
		testutil.Exec(t, c.pool, `DELETE FROM stock_picking_wave_lines WHERE wave_id=$1`, onda)
		testutil.Exec(t, c.pool, `DELETE FROM stock_picking_waves WHERE id=$1`, onda)
	})
	cliente := int64(792)
	n := c.notaDoTipo(t, c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA"), &cliente, "2")
	testutil.Exec(t, c.pool, `UPDATE fiscal_exits SET sales_order_code=$2 WHERE id=$1`, n, pedido)
	exige(t, c.aplicar(t, c.efeitosComTipos(t), n))
	faturado := func() string {
		var q string
		exige(t, c.pool.QueryRow(context.Background(), `SELECT invoiced_qty::text FROM stock_picking_wave_lines WHERE id=$1`, linha).Scan(&q))
		return q
	}
	igualQ(t, "onda fechada pela remessa", dq(faturado()), "2")
	_, err := contabil(c).EstornarEstoqueDaSaida(c.ctx, n, c.ator)
	exige(t, err)
	igualQ(t, "cancelamento devolve a separação", dq(faturado()), "0")
}

// Cancelada a remessa, o material volta ao endereço de onde saiu.
func TestCancelarRemessaVoltaAoEndereco(t *testing.T) {
	c := novaNotaAutorizada(t)
	novoAlmoxConsignacao(t, c)
	repo := stockRepo.NewStockRepositorySQLC(c.pool)
	testutil.Exec(t, c.pool, `INSERT INTO manufacturing_warehouse_addresses(enterprise_id,warehouse_id,address,is_active,zone,is_blocked,pick_sequence) VALUES($1,$2,'A1',TRUE,'Z',FALSE,1)`, c.empresa, c.almox)
	t.Cleanup(func() {
		testutil.Exec(t, c.pool, `DELETE FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1`, c.empresa)
	})
	endereco := "A1"
	_, err := repo.CreateMovement(c.ctx, &stockentity.StockMovement{ItemCode: c.item, WarehouseID: c.almox, MovementType: stockentity.MovementTypeIn, Quantity: dq("4"), UnitPrice: dq("20"), Address: &endereco, CreatedBy: c.ator})
	exige(t, err)
	cliente := int64(793)
	n := c.notaDoTipo(t, c.tipoDeNota(t, "REMESSA_CONSIGNACAO", "TRANSFERENCIA_EXTERNA"), &cliente, "3")
	testutil.Exec(t, c.pool, `UPDATE fiscal_exit_items SET address='A1' WHERE fiscal_exit_id=$1`, n)
	exige(t, c.aplicar(t, c.efeitosComTipos(t), n))
	_, err = contabil(c).EstornarEstoqueDaSaida(c.ctx, n, c.ator)
	exige(t, err)
	var q string
	exige(t, c.pool.QueryRow(context.Background(), `SELECT COALESCE(SUM(quantity),0)::text FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND warehouse_id=$3 AND address='A1'`, c.empresa, c.item, c.almox).Scan(&q))
	igualQ(t, "saldo no endereço", dq(q), "4")
}

// Duas linhas do mesmo item numa transferência de pedido separado: a segunda não leva
// de novo a separação que a primeira levou.
func TestTransferenciaDeDuasLinhasNaoRepeteASeparacao(t *testing.T) {
	i := duasFiliais(t, "11222333")
	o := i.origem
	pedido := int64(4444)
	var onda, linha int64
	exige(t, o.pool.QueryRow(context.Background(), `INSERT INTO stock_picking_waves(enterprise_id,code,warehouse_id,status,rule,created_by) VALUES($1,$2,$3,'SEPARADA','FIFO',$4) RETURNING id`,
		o.empresa, testutil.UniqueCode()%1_000_000, o.almox, o.ator).Scan(&onda))
	exige(t, o.pool.QueryRow(context.Background(), `INSERT INTO stock_picking_wave_lines(enterprise_id,wave_id,item_code,mask,lot,address,quantity,picked_qty,reference_type,reference_code)
		VALUES($1,$2,$3,'','','',3,3,$4,$5) RETURNING id`, o.empresa, onda, o.item, stockentity.ReferenceTypeSalesOrder, pedido).Scan(&linha))
	t.Cleanup(func() {
		testutil.Exec(t, o.pool, `DELETE FROM stock_picking_wave_lines WHERE wave_id=$1`, onda)
		testutil.Exec(t, o.pool, `DELETE FROM stock_picking_waves WHERE id=$1`, onda)
	})
	n := i.notaDeTransferencia(t, "11222333000262")
	_, err := o.pool.Exec(context.Background(), `UPDATE fiscal_exits SET sales_order_code=$2 WHERE id=$1`, n, pedido)
	exige(t, err)
	_, err = o.pool.Exec(context.Background(), `INSERT INTO fiscal_exit_items(fiscal_exit_id,item_code,cfop,quantity,unit_price,total_price) VALUES($1,$2,'5917',2,20,40)`, n, o.item)
	exige(t, err)
	exige(t, o.aplicar(t, o.efeitosComTipos(t), n))
	var q string
	exige(t, o.pool.QueryRow(context.Background(), `SELECT invoiced_qty::text FROM stock_picking_wave_lines WHERE id=$1`, linha).Scan(&q))
	igualQ(t, "separação levada", dq(q), "3")
	igualQ(t, "trânsito com as duas linhas", saldo(t, o, i.transit), "6")
}

type leitorDoInventario interface {
	InventarioEFD(ctx context.Context, data time.Time) ([]sped.ItemInventario, string, error)
}

func novoFiscalPG(c cenario) leitorDoInventario {
	return any(fiscalRepo.NewFiscalRepositoryPG(c.pool)).(leitorDoInventario)
}
